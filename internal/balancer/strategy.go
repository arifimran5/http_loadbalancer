package balancer

import (
	"math"
	"sync"
)

type LoadBalancingStrategy interface {
	GetNextServer(servers []*Server) *Server
}

type RoundRobin struct {
	iterator int
	mu       *sync.Mutex
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{mu: &sync.Mutex{}}
}

func (rr *RoundRobin) GetNextServer(servers []*Server) *Server {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	for _, server := range servers {
		if server.Health {
			nextIndex := rr.iterator % int(len(servers))
			rr.iterator++
			return servers[nextIndex]
		}
	}
	return nil
}

type LeastConnections struct{}

func NewLeastConnections() *LeastConnections {
	return &LeastConnections{}
}

func (lc *LeastConnections) GetNextServer(servers []*Server) *Server {
	var leastLoadedServer *Server
	minConn := math.MaxInt32

	for _, server := range servers {
		if server.Health && server.GetActiveConnections() < minConn {
			minConn = server.GetActiveConnections()
			leastLoadedServer = server
		}
	}
	return leastLoadedServer
}
