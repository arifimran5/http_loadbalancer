package balancer

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/arifimran5/http_loadbalancer/internal/proxy"
)

// Server represents a backend server with thread-safe health and connection tracking.
type Server struct {
	Proxy                 *proxy.Proxy
	health                atomic.Bool
	activeConnections     atomic.Int64
	ResponseTimeThreshold time.Duration
}

// NewServer creates a new Server instance for the given host.
func NewServer(host string, respTime time.Duration) (*Server, error) {
	proxyInstance, err := proxy.NewProxy(host)
	if err != nil {
		return nil, err
	}
	s := &Server{
		Proxy:                 proxyInstance,
		ResponseTimeThreshold: respTime,
	}
	s.health.Store(true) // Initially healthy
	return s, nil
}

// IsHealthy returns the current health status (thread-safe).
func (s *Server) IsHealthy() bool {
	return s.health.Load()
}

// SetHealth updates the health status (thread-safe).
func (s *Server) SetHealth(healthy bool) {
	s.health.Store(healthy)
}

// GetActiveConnections returns the current active connection count (thread-safe).
func (s *Server) GetActiveConnections() int64 {
	return s.activeConnections.Load()
}

// IncrementConnections atomically increments the active connection count.
func (s *Server) IncrementConnections() {
	s.activeConnections.Add(1)
}

// DecrementConnections atomically decrements the active connection count.
// Guards against negative values.
func (s *Server) DecrementConnections() {
	// Use CompareAndSwap loop to prevent going negative
	for {
		current := s.activeConnections.Load()
		if current <= 0 {
			return // Don't decrement below zero
		}
		if s.activeConnections.CompareAndSwap(current, current-1) {
			return
		}
	}
}

// CheckHealth checks the health of the server with proper timeout.
func (s *Server) CheckHealth() bool {
	client := &http.Client{
		Timeout: s.ResponseTimeThreshold,
	}

	startTime := time.Now()
	resp, err := client.Head(s.Proxy.Host)
	responseTime := time.Since(startTime)

	if err != nil {
		s.SetHealth(false)
		return false
	}
	defer resp.Body.Close()

	// Allow 2xx and 3xx status codes as healthy
	healthy := resp.StatusCode >= 200 && resp.StatusCode < 400 && responseTime <= s.ResponseTimeThreshold
	s.SetHealth(healthy)
	return healthy
}
