// Credit to Alex Edwards for this article - https://www.alexedwards.net/blog/how-to-rate-limit-http-requests
package ratelimiter

import (
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// A custom visitor struct which holds the rate limiter for each
// visitor and the last time that the visitor was seen.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu         sync.Mutex
	visitors   map[string]*visitor
	rps        int
	burstLimit int
}

func NewRateLimiter(rps, burstLimit int) *RateLimiter {
	rl := &RateLimiter{
		mu:         sync.Mutex{},
		visitors:   make(map[string]*visitor),
		rps:        rps,
		burstLimit: burstLimit,
	}
	go cleanupVisitors(rl)
	return rl
}

func (r *RateLimiter) getVisitor(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, exists := r.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(r.rps), r.burstLimit)
		// Include the current time when creating a new visitor.
		r.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	return v.limiter
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func cleanupVisitors(rl *RateLimiter) {
	for {
		time.Sleep(time.Minute)
		log.Println("cleaning up")
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				log.Println("cleaned:", ip)
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (r *RateLimiter) Allow(ip string) bool {
	limiter := r.getVisitor(ip)
	return limiter.Allow()
}
