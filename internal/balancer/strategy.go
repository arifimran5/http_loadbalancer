package balancer

import (
	"hash/fnv"
	"net/http"
	"sync/atomic"
)

// LoadBalancingStrategy defines the interface for load balancing algorithms.
type LoadBalancingStrategy interface {
	GetNextServer(servers []*Server, req *http.Request) *Server
}

// RoundRobin distributes requests evenly across healthy servers.
type RoundRobin struct {
	counter atomic.Uint64
}

// NewRoundRobin creates a new RoundRobin strategy.
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

// GetNextServer returns the next healthy server using round-robin selection.
// It skips unhealthy servers and wraps around the server list.
func (rr *RoundRobin) GetNextServer(servers []*Server, req *http.Request) *Server {
	n := uint64(len(servers))
	if n == 0 {
		return nil
	}

	// Get current counter and increment atomically
	start := rr.counter.Add(1) - 1

	// Try each server starting from the current position
	for i := uint64(0); i < n; i++ {
		idx := (start + i) % n
		server := servers[idx]
		if server.IsHealthy() {
			return server
		}
	}
	return nil // No healthy servers available
}

// LeastConnections routes requests to the server with fewest active connections.
type LeastConnections struct{}

// NewLeastConnections creates a new LeastConnections strategy.
func NewLeastConnections() *LeastConnections {
	return &LeastConnections{}
}

// GetNextServer returns the healthy server with the fewest active connections.
func (lc *LeastConnections) GetNextServer(servers []*Server, req *http.Request) *Server {
	var leastLoadedServer *Server
	var minConn int64 = -1

	for _, server := range servers {
		if !server.IsHealthy() {
			continue
		}
		conn := server.GetActiveConnections()
		if minConn == -1 || conn < minConn {
			minConn = conn
			leastLoadedServer = server
		}
	}
	return leastLoadedServer
}

// IPHash routes requests from the same client IP to the same backend server.
type IPHash struct{}

// GetNextServer returns a server based on the hash of the client's IP address.
// Falls back to finding any healthy server if the hashed server is unhealthy.
func (ih *IPHash) GetNextServer(servers []*Server, req *http.Request) *Server {
	n := len(servers)
	if n == 0 {
		return nil
	}

	ip := ExtractClientIP(req)
	if ip == "" {
		return nil
	}

	hash := fnv.New32a()
	hash.Write([]byte(ip))
	startIdx := int(hash.Sum32() % uint32(n))

	// Try the hashed server first, then fall back to others
	for i := 0; i < n; i++ {
		idx := (startIdx + i) % n
		if servers[idx].IsHealthy() {
			return servers[idx]
		}
	}
	return nil
}
