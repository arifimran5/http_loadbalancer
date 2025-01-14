package balancer

import (
	"hash/fnv"
	"math"
	"net"
	"net/http"
	"sync"
)

type LoadBalancingStrategy interface {
	GetNextServer(servers []*Server, req *http.Request) *Server
}

type RoundRobin struct {
	iterator int
	mu       *sync.Mutex
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{mu: &sync.Mutex{}}
}

func (rr *RoundRobin) GetNextServer(servers []*Server, req *http.Request) *Server {
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

func (lc *LeastConnections) GetNextServer(servers []*Server, req *http.Request) *Server {
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

type IPHash struct{}

func (ih *IPHash) GetNextServer(servers []*Server, req *http.Request) *Server {
	if len(servers) == 0 {
		return nil
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return nil
	}
	hash := fnv.New32a()
	hash.Write([]byte(ip))

	index := hash.Sum32() % uint32(len(servers))
	return servers[index]
}
