package balancer

import (
	"fmt"
	"log"
	"net/http"
)

type LoadBalancer struct {
	servers  []*Server
	iterator int32
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		iterator: 0,
	}
}

func (lb *LoadBalancer) AddServer(host string) {
	newServer, err := NewServer(host)
	if err != nil {
		log.Printf("Error adding server %s: %v", host, err)
		return
	}
	lb.servers = append(lb.servers, newServer)
}

func (lb *LoadBalancer) ForwardRequest(res http.ResponseWriter, req *http.Request) {
	server, err := lb.getHealthyServer()
	if err != nil {
		fmt.Println("Error getting healthy server:", err)
		http.Error(res, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	server.Proxy.ServeHTTP(res, req)
}

func (lb *LoadBalancer) getHealthyServer() (*Server, error) {
	for i := 0; i < len(lb.servers); i++ {
		nextIndex := (lb.iterator + 1) % int32(len(lb.servers))
		server := lb.servers[nextIndex]
		lb.iterator = nextIndex
		if server.Health {
			return server, nil
		}
	}
	return nil, fmt.Errorf("no healthy servers found")
}
