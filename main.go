package main

import (
	"log"
	"net/http"

	"github.com/arifimran5/http_loadbalancer/loadbalancer"
)

func main() {
	lb := loadbalancer.NewLoadBalancer()
	lb.AddServer("http://localhost:8080")
	lb.AddServer("http://localhost:8081")
	lb.AddServer("http://localhost:8082")
	lb.AddServer("http://localhost:8083")
	lb.StartHealthCheck()

	http.HandleFunc("/", lb.ForwardRequest)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
