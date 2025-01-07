package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/arifimran5/http_loadbalancer/internal/balancer"
)

func main() {
	algorithm := flag.String("algorithm", "roundrobin", "Load balancing algorithm (roundrobin or leastconnections)")
	flag.Parse()

	var strategy balancer.LoadBalancingStrategy

	switch *algorithm {
	case "roundrobin":
		strategy = balancer.NewRoundRobin()
	case "leastconnections":
		strategy = balancer.NewLeastConnections()
	default:
		log.Fatalf("Unknown algorithm: %s", *algorithm)
	}

	lb := balancer.NewLoadBalancer(strategy)
	responseTimeThreshold := 500 * time.Millisecond
	lb.AddServer("http://localhost:8080", responseTimeThreshold)
	lb.AddServer("http://localhost:8081", responseTimeThreshold)
	lb.AddServer("http://localhost:8082", responseTimeThreshold)
	lb.AddServer("http://localhost:8083", responseTimeThreshold)

	balancer.StartHealthCheck(lb)

	http.HandleFunc("/", lb.ForwardRequest)
	log.Println("Load balancer started at :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
