package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arifimran5/http_loadbalancer/internal/balancer"
	"github.com/arifimran5/http_loadbalancer/internal/config"
)

func main() {
	conf, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
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
	if conf.LoadBalancer.MaxInFlight > 0 {
		lb.SetMaxInFlight(conf.LoadBalancer.MaxInFlight)
	}

	// Add servers from config
	for _, server := range conf.LoadBalancer.Servers {
		lb.AddServer(server.Host, time.Duration(server.HealthyTimeThreshold)*time.Millisecond)
	}

	// Start health checks
	healthChecker := balancer.StartHealthCheck(lb)

	// Configure HTTP server with timeouts
	port := fmt.Sprintf(":%d", conf.LoadBalancer.Port)
	server := &http.Server{
		Addr:              port,
		Handler:           http.HandlerFunc(lb.ForwardRequest),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// Channel to listen for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("Switchblade load balancer started on %s", port)
		log.Printf("Algorithm: %s, Backends: %d", conf.LoadBalancer.Algorithm, len(conf.LoadBalancer.Servers))
		if conf.LoadBalancer.MaxInFlight > 0 {
			log.Printf("Max In-Flight Requests: %d", conf.LoadBalancer.MaxInFlight)
		}
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-shutdown
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop health checks first
	healthChecker.Stop()

	// Gracefully shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	// Cleanup load balancer resources
	lb.Shutdown()

	log.Println("Switchblade shutdown complete")
}
