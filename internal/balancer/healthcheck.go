package balancer

import (
	"context"
	"log"
	"time"

	"github.com/go-co-op/gocron"
)

// HealthChecker manages periodic health checks for backend servers.
type HealthChecker struct {
	scheduler *gocron.Scheduler
	lb        *LoadBalancer
	interval  time.Duration
	cancel    context.CancelFunc
}

// NewHealthChecker creates a new HealthChecker for the given load balancer.
func NewHealthChecker(lb *LoadBalancer, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		lb:       lb,
		interval: interval,
	}
}

// Start begins periodic health checks for all servers.
func (hc *HealthChecker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	hc.cancel = cancel

	hc.scheduler = gocron.NewScheduler(time.Local)
	hc.scheduler.WithDistributedLocker(nil) // Disable distributed locking

	servers := hc.lb.GetServers()
	for _, server := range servers {
		s := server // Capture for closure
		_, err := hc.scheduler.Every(int(hc.interval.Seconds())).Seconds().Do(func() {
			select {
			case <-ctx.Done():
				return
			default:
				healthy := s.CheckHealth()
				if !healthy {
					log.Printf("[HealthCheck] '%s' is unhealthy", s.Proxy.Host)
				}
			}
		})
		if err != nil {
			log.Printf("[HealthCheck] Error scheduling health check for %s: %v", s.Proxy.Host, err)
		}
	}
	hc.scheduler.StartAsync()
	log.Printf("[HealthCheck] Started health checks (interval: %v)", hc.interval)
}

// Stop gracefully stops the health checker.
func (hc *HealthChecker) Stop() {
	if hc.cancel != nil {
		hc.cancel()
	}
	if hc.scheduler != nil {
		hc.scheduler.Stop()
	}
	log.Println("[HealthCheck] Stopped health checks")
}

// StartHealthCheck starts health checks with default 2-second interval.
// Deprecated: Use NewHealthChecker and Start() for more control.
func StartHealthCheck(lb *LoadBalancer) *HealthChecker {
	hc := NewHealthChecker(lb, 2*time.Second)
	hc.Start()
	return hc
}
