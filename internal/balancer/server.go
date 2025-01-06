package balancer

import (
	"net/http"

	"github.com/arifimran5/http_loadbalancer/internal/proxy"
)

// Server represents a backend server.
type Server struct {
	Proxy  *proxy.Proxy
	Health bool
}

// NewServer creates a new Server instance for the given host.
func NewServer(host string) (*Server, error) {
	proxyInstance, err := proxy.NewProxy(host)
	if err != nil {
		return nil, err
	}
	return &Server{
		Proxy:  proxyInstance,
		Health: true,
	}, nil
}

// CheckHealth checks the health of the server.
func (s *Server) CheckHealth() bool {
	resp, err := http.Head(s.Proxy.Host)
	if err != nil || resp.StatusCode != http.StatusOK {
		s.Health = false
		return s.Health
	}
	s.Health = true
	return s.Health
}
