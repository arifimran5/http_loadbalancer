package balancer

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/arifimran5/http_loadbalancer/internal/config"
	"github.com/arifimran5/http_loadbalancer/pkg/ratelimiter"
)

// LoadBalancer manages backend servers and routes incoming requests.
type LoadBalancer struct {
	strategy    LoadBalancingStrategy
	servers     []*Server
	serversMu   sync.RWMutex // Protects servers slice for safe concurrent access
	config      *config.Config
	rateLimiter *ratelimiter.RateLimiter

	// Concurrency control
	maxInFlight   int64
	inFlightCount int64
	inFlightMu    sync.Mutex
	inFlightCond  *sync.Cond
}

// NewLoadBalancer creates a new LoadBalancer with the given strategy and configuration.
func NewLoadBalancer(strategy LoadBalancingStrategy, conf *config.Config) *LoadBalancer {
	lb := &LoadBalancer{
		servers:     []*Server{},
		strategy:    strategy,
		config:      conf,
		maxInFlight: 10000, // Default max concurrent requests
	}
	lb.inFlightCond = sync.NewCond(&lb.inFlightMu)

	if conf.LoadBalancer.RateLimiting.Enabled {
		lb.rateLimiter = ratelimiter.NewRateLimiter(
			conf.LoadBalancer.RateLimiting.RequestsPerSecond,
			conf.LoadBalancer.RateLimiting.BurstLimit,
		)
	}
	return lb
}

// SetMaxInFlight sets the maximum number of concurrent in-flight requests.
func (lb *LoadBalancer) SetMaxInFlight(max int64) {
	lb.inFlightMu.Lock()
	lb.maxInFlight = max
	lb.inFlightMu.Unlock()
}

// AddServer adds a new backend server to the load balancer.
func (lb *LoadBalancer) AddServer(host string, respTime time.Duration) {
	newServer, err := NewServer(host, respTime)
	if err != nil {
		log.Printf("Error adding server %s: %v", host, err)
		return
	}
	lb.serversMu.Lock()
	lb.servers = append(lb.servers, newServer)
	lb.serversMu.Unlock()
}

// GetServers returns a copy of the current server list (thread-safe).
func (lb *LoadBalancer) GetServers() []*Server {
	lb.serversMu.RLock()
	defer lb.serversMu.RUnlock()
	servers := make([]*Server, len(lb.servers))
	copy(servers, lb.servers)
	return servers
}

// ForwardRequest handles incoming requests and forwards them to a backend server.
func (lb *LoadBalancer) ForwardRequest(res http.ResponseWriter, req *http.Request) {
	// Extract client IP safely
	clientIP := ExtractClientIP(req)

	// Check blacklist first (fail fast)
	if lb.isBlacklisted(clientIP) {
		http.Error(res, "Forbidden", http.StatusForbidden)
		return
	}

	// Check whitelist
	if !lb.isWhitelisted(clientIP) {
		http.Error(res, "Forbidden", http.StatusForbidden)
		return
	}

	// Apply rate limiting
	if lb.rateLimiter != nil && !lb.rateLimiter.Allow(clientIP) {
		http.Error(res, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	// Backpressure: wait if at max capacity (with timeout)
	if !lb.acquireInFlight() {
		http.Error(res, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	defer lb.releaseInFlight()

	// Get servers snapshot and select next server
	servers := lb.GetServers()
	server := lb.strategy.GetNextServer(servers, req)
	if server == nil {
		http.Error(res, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	// Track active connections on the selected server
	server.IncrementConnections()
	defer server.DecrementConnections()

	// Forward the request
	server.Proxy.ServeHTTP(res, req)
}

// acquireInFlight attempts to acquire an in-flight slot with backpressure.
// Returns false if unable to acquire within timeout.
func (lb *LoadBalancer) acquireInFlight() bool {
	lb.inFlightMu.Lock()
	defer lb.inFlightMu.Unlock()

	// Simple check without blocking for now
	// For production, could add wait with timeout using lb.inFlightCond
	if lb.inFlightCount >= lb.maxInFlight {
		log.Printf("Too many in-flight requests: %d\n", lb.inFlightCount)
		return false
	}
	lb.inFlightCount++
	return true
}

// releaseInFlight releases an in-flight slot.
func (lb *LoadBalancer) releaseInFlight() {
	lb.inFlightMu.Lock()
	lb.inFlightCount--
	lb.inFlightCond.Signal() // Wake up any waiting goroutines
	lb.inFlightMu.Unlock()
}

// isBlacklisted checks if the given IP is in the blacklist.
func (lb *LoadBalancer) isBlacklisted(ip string) bool {
	for _, blockedIP := range lb.config.LoadBalancer.IPBlacklist {
		if ip == blockedIP {
			return true
		}
	}
	return false
}

// isWhitelisted checks if the given IP is in the whitelist.
// If no whitelist is defined, all IPs are allowed.
func (lb *LoadBalancer) isWhitelisted(ip string) bool {
	if len(lb.config.LoadBalancer.IPWhitelist) == 0 {
		return true
	}
	for _, allowedIP := range lb.config.LoadBalancer.IPWhitelist {
		if ip == allowedIP {
			return true
		}
	}
	return false
}

// Shutdown gracefully shuts down the load balancer.
func (lb *LoadBalancer) Shutdown() {
	lb.serversMu.Lock()
	defer lb.serversMu.Unlock()

	for _, server := range lb.servers {
		server.Proxy.Close()
	}
}
