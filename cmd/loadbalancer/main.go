package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/arifimran5/http_loadbalancer/internal/balancer"
	"github.com/arifimran5/http_loadbalancer/internal/config"
)

func main() {
	conf, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
		return
	}

	var strategy balancer.LoadBalancingStrategy

	switch conf.LoadBalancer.Algorithm {
	case "roundrobin":
		strategy = balancer.NewRoundRobin()
	case "least_conn":
		strategy = balancer.NewLeastConnections()
	case "iphash":
		strategy = &balancer.IPHash{}
	default:
		log.Fatalf("Unknown algorithm: %s", conf.LoadBalancer.Algorithm)
	}

	lb := balancer.NewLoadBalancer(strategy, conf)
	for _, server := range conf.LoadBalancer.Servers {
		lb.AddServer(server.Host, time.Duration(server.Healthy_Time_Threshold)*time.Millisecond)
	}

	balancer.StartHealthCheck(lb)

	http.HandleFunc("/", lb.ForwardRequest)
	port := fmt.Sprintf(":%d", conf.LoadBalancer.Port)
	log.Println("Load balancer started at port ", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Error starting load balancer: %v", err)
		return
	}
}
