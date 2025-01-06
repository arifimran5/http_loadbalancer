package main

import (
	"log"
	"net/http"

	"github.com/arifimran5/http_loadbalancer/internal/balancer"
)

func main() {
	lb := balancer.NewLoadBalancer()
	lb.AddServer("http://localhost:8080")
	lb.AddServer("http://localhost:8081")
	lb.AddServer("http://localhost:8082")
	lb.AddServer("http://localhost:8083")

	balancer.StartHealthCheck(lb)

	http.HandleFunc("/", lb.ForwardRequest)
	log.Println("Load balancer started at :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
