package balancer

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/arifimran5/http_loadbalancer/internal/config"
	"github.com/arifimran5/http_loadbalancer/pkg/ratelimiter"
)

type LoadBalancer struct {
	strategy    LoadBalancingStrategy
	servers     []*Server
	config      *config.Config
	rateLimiter *ratelimiter.RateLimiter
}

func NewLoadBalancer(strategy LoadBalancingStrategy, conf *config.Config) *LoadBalancer {
	var rl *ratelimiter.RateLimiter
	if conf.LoadBalancer.RateLimiting.Enabled {
		rl = ratelimiter.NewRateLimiter(conf.LoadBalancer.RateLimiting.RequestsPerSecond,
			conf.LoadBalancer.RateLimiting.BurstLimit)
	}
	return &LoadBalancer{
		strategy:    strategy,
		config:      conf,
		rateLimiter: rl,
	}
}

func (lb *LoadBalancer) AddServer(host string, respTime time.Duration) {
	newServer, err := NewServer(host, respTime)
	if err != nil {
		log.Printf("Error adding server %s: %v", host, err)
		return
	}
	lb.servers = append(lb.servers, newServer)
}

func (lb *LoadBalancer) ForwardRequest(res http.ResponseWriter, req *http.Request) {
	server := lb.strategy.GetNextServer(lb.servers, req)
	if server == nil {
		http.Error(res, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if lb.isBlacklisted(ip) {
		http.Error(res, "Forbidden", http.StatusForbidden)
		return
	}
	if !lb.isWhitelisted(ip) {
		http.Error(res, "Forbidden", http.StatusForbidden)
		return
	}

	if lb.rateLimiter != nil && !lb.rateLimiter.Allow(ip) {
		http.Error(res, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	server.IncrementConnections()       // Increment active connection count before forwarding
	defer server.DecrementConnections() // Decrement after response is sent

	server.Proxy.ServeHTTP(res, req)
}

func (lb *LoadBalancer) isBlacklisted(ip string) bool {
	for _, blockedIP := range lb.config.LoadBalancer.IPBlacklist {
		if ip == blockedIP {
			return true
		}
	}
	return false
}

func (lb *LoadBalancer) isWhitelisted(ip string) bool {
	if len(lb.config.LoadBalancer.IPWhitelist) == 0 {
		return true // If no whitelist is defined, allow all
	}
	for _, allowedIP := range lb.config.LoadBalancer.IPWhitelist {
		if ip == allowedIP {
			return true
		}
	}
	return false
}
