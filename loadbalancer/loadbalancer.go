package loadbalancer

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-co-op/gocron"
)

type Server struct {
	Proxy  *httputil.ReverseProxy
	Host   string
	Health bool
}

func NewServer(host string) *Server {
	serverUrl, err := url.Parse(host)
	if err != nil {
		panic(err)
	}
	return &Server{
		Proxy:  httputil.NewSingleHostReverseProxy(serverUrl),
		Host:   host,
		Health: true,
	}
}

func (s *Server) CheckHealth() bool {
	resp, err := http.Head(s.Host)
	if err != nil {
		s.Health = false
		return s.Health
	}
	if resp.StatusCode != http.StatusOK {
		s.Health = false
		return s.Health
	}
	s.Health = true
	return s.Health
}

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
	newServer := NewServer(host)
	lb.servers = append(lb.servers, newServer)
}

func (lb *LoadBalancer) ForwardRequest(res http.ResponseWriter, req *http.Request) {
	server, err := lb.getHealthyServer()

	if err != nil {
		fmt.Println("Error getting healthy server:", err)
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

func (lb *LoadBalancer) StartHealthCheck() {
	sch := gocron.NewScheduler(time.Local)
	for _, host := range lb.servers {
		_, err := sch.Every(2).Seconds().Do(func(s *Server) {
			healthy := s.CheckHealth()
			if healthy {
				log.Printf("'%s' is healthy!", s.Host)
			} else {
				log.Printf("'%s' is not healthy", s.Host)
			}
		}, host)
		if err != nil {
			log.Fatalln(err)
		}
	}
	sch.StartAsync()
}
