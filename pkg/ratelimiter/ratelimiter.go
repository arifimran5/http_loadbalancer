package ratelimiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// visitor holds the rate limiter for each client and their last access time.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter implements per-IP rate limiting using token bucket algorithm.
type RateLimiter struct {
	mu         sync.RWMutex
	visitors   map[string]*visitor
	rps        int
	burstLimit int
	stopClean  chan struct{}
}

// NewRateLimiter creates a new RateLimiter with the given requests per second and burst limit.
func NewRateLimiter(rps, burstLimit int) *RateLimiter {
	rl := &RateLimiter{
		visitors:   make(map[string]*visitor),
		rps:        rps,
		burstLimit: burstLimit,
		stopClean:  make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// Allow checks if the request from the given IP should be allowed.
func (r *RateLimiter) Allow(ip string) bool {
	limiter := r.getVisitor(ip)
	return limiter.Allow()
}

// getVisitor returns the rate limiter for the given IP, creating one if needed.
func (r *RateLimiter) getVisitor(ip string) *rate.Limiter {
	// Try read lock first for existing visitors (fast path)
	r.mu.RLock()
	v, exists := r.visitors[ip]
	if exists {
		v.lastSeen = time.Now()
		r.mu.RUnlock()
		return v.limiter
	}
	r.mu.RUnlock()

	// Need to create new visitor
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if v, exists = r.visitors[ip]; exists {
		v.lastSeen = time.Now()
		return v.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(r.rps), r.burstLimit)
	r.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

// cleanupLoop periodically removes stale visitor entries.
func (r *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanup()
		case <-r.stopClean:
			return
		}
	}
}

// cleanup removes visitors that haven't been seen for more than 3 minutes.
func (r *RateLimiter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	threshold := time.Now().Add(-3 * time.Minute)
	for ip, v := range r.visitors {
		if v.lastSeen.Before(threshold) {
			delete(r.visitors, ip)
		}
	}
}

// Stop halts the cleanup goroutine.
func (r *RateLimiter) Stop() {
	close(r.stopClean)
}

// VisitorCount returns the current number of tracked visitors.
func (r *RateLimiter) VisitorCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.visitors)
}
