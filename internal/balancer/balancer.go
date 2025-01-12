package balancer

import (
	"log"
	"net/http"
	"time"
)

type LoadBalancer struct {
	strategy LoadBalancingStrategy
	servers  []*Server
}

func NewLoadBalancer(strategy LoadBalancingStrategy) *LoadBalancer {
	return &LoadBalancer{
		strategy: strategy,
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

	server.IncrementConnections()       // Increment active connection count before forwarding
	defer server.DecrementConnections() // Decrement after response is sent

	server.Proxy.ServeHTTP(res, req)
}
