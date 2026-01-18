# Phase 0: Safety & Correctness

This document describes the safety improvements made in Phase 0 to make Switchblade production-safe.

## Summary of Changes

Phase 0 focuses on eliminating correctness and reliability issues before adding features. Without these fixes, upstream failures could cascade and stall the load balancer under load.

---

## 1. Concurrency Fixes

### Problem
- `Server.Health` was read/written concurrently without synchronization (data race)
- `activeConnections` could go negative on panics or double-decrements
- `lb.servers` slice had no protection for concurrent access

### Solution

**Atomic operations for server state** ([server.go](../internal/balancer/server.go)):
```go
type Server struct {
    health            atomic.Bool   // Thread-safe health status
    activeConnections atomic.Int64  // Thread-safe connection counter
}
```

**Safe connection decrement** prevents negative values:
```go
func (s *Server) DecrementConnections() {
    for {
        current := s.activeConnections.Load()
        if current <= 0 {
            return // Don't decrement below zero
        }
        if s.activeConnections.CompareAndSwap(current, current-1) {
            return
        }
    }
}
```

**RWMutex for server list** ([balancer.go](../internal/balancer/balancer.go)):
```go
type LoadBalancer struct {
    servers   []*Server
    serversMu sync.RWMutex // Protects servers slice
}
```

---

## 2. Algorithm Bug Fixes

### Problem
RoundRobin's `GetNextServer()` had a logic bug:
- It checked if *any* server was healthy
- But returned `servers[nextIndex]` which could be unhealthy

### Solution

**Fixed RoundRobin** ([strategy.go](../internal/balancer/strategy.go)):
```go
func (rr *RoundRobin) GetNextServer(servers []*Server, req *http.Request) *Server {
    n := uint64(len(servers))
    start := rr.counter.Add(1) - 1
    
    // Try each server starting from the current position
    for i := uint64(0); i < n; i++ {
        idx := (start + i) % n
        if servers[idx].IsHealthy() {
            return servers[idx]
        }
    }
    return nil // No healthy servers
}
```

**IPHash fallback** now tries other servers if hashed server is unhealthy.

---

## 3. Upstream Timeout Hardening

### Problem
No explicit timeouts for upstream connections. Default transports can hang on failures, causing tail-latency and resource leaks.

### Solution

**Custom transport with timeouts** ([proxy.go](../internal/proxy/proxy.go)):
```go
transport := &http.Transport{
    DialContext: (&net.Dialer{
        Timeout:   5 * time.Second,
        KeepAlive: 30 * time.Second,
    }).DialContext,
    TLSHandshakeTimeout:   5 * time.Second,
    ResponseHeaderTimeout: 30 * time.Second,
    IdleConnTimeout:       90 * time.Second,
    MaxIdleConns:          100,
    MaxIdleConnsPerHost:   10,
    MaxConnsPerHost:       100,
}
```

**Custom error handler** returns appropriate status codes:
- `504 Gateway Timeout` for timeout errors
- `502 Bad Gateway` for other upstream failures

---

## 4. HTTP Server Hardening

### Problem
No server-level timeouts; malicious or slow clients could exhaust resources.

### Solution

**Server with timeouts** ([main.go](../cmd/loadbalancer/main.go)):
```go
server := &http.Server{
    Addr:              port,
    ReadHeaderTimeout: 10 * time.Second,
    ReadTimeout:       30 * time.Second,
    WriteTimeout:      60 * time.Second,
    IdleTimeout:       120 * time.Second,
    MaxHeaderBytes:    1 << 20, // 1 MB
}
```

---

## 5. Graceful Shutdown

### Problem
No graceful shutdown; abrupt termination drops in-flight requests.

### Solution

**SIGTERM/SIGINT handling** ([main.go](../cmd/loadbalancer/main.go)):
```go
shutdown := make(chan os.Signal, 1)
signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

// Wait for signal
sig := <-shutdown
log.Printf("Received signal %v, initiating graceful shutdown...", sig)

// Graceful shutdown with 30s timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

healthChecker.Stop()           // Stop health checks
server.Shutdown(ctx)           // Drain connections
lb.Shutdown()                  // Cleanup resources
```

---

## 6. Client IP Extraction

### Problem
Using `req.RemoteAddr` directly doesn't work behind proxies/load balancers. Rate limiting and ACLs would apply to the proxy IP instead of the real client.

### Solution

**Safe X-Forwarded-For parsing** ([clientip.go](../internal/balancer/clientip.go)):
- Only trusts XFF headers from known private/internal networks
- Supports both `X-Forwarded-For` and standard `Forwarded` headers
- Returns the leftmost non-proxy IP

```go
func ExtractClientIP(req *http.Request) string {
    remoteIP := parseRemoteAddr(req.RemoteAddr)
    
    // Only trust XFF if connection is from a trusted proxy
    if !isTrustedProxy(remoteIP) {
        return remoteIP
    }
    
    // Parse X-Forwarded-For and return first non-proxy IP
    return parseXForwardedFor(req.Header.Get("X-Forwarded-For"))
}
```

---

## 7. Backpressure / Max In-Flight Requests

### Problem
No limit on concurrent requests; under heavy load, the system could be overwhelmed.

### Solution

**Simple semaphore** ([balancer.go](../internal/balancer/balancer.go)):
```go
type LoadBalancer struct {
    maxInFlight   int64  // Default: 10000
    inFlightCount int64
}

func (lb *LoadBalancer) ForwardRequest(...) {
    if !lb.acquireInFlight() {
        http.Error(res, "Service Unavailable", http.StatusServiceUnavailable)
        return
    }
    defer lb.releaseInFlight()
    // ... handle request
}
```

---

## 8. Config Validation

### Problem
Invalid config could cause runtime panics or undefined behavior.

### Solution

**Validation on load** ([config.go](../internal/config/config.go)):
- Validates port range (1-65535)
- Validates algorithm name
- Checks for duplicate server hosts
- Validates URL format
- Validates rate limiting parameters

```go
func (c *Config) Validate() error {
    // Port validation
    // Algorithm validation
    // Server validation (no duplicates, valid URLs)
    // Rate limiting validation
}
```

---

## Testing the Changes

```bash
# Build
go build ./...

# Run
go run cmd/loadbalancer/main.go

# Test graceful shutdown
kill -SIGTERM <pid>
```

---

## What's Next: Phase 1

Phase 1 will add operational features:
- Prometheus metrics
- Structured JSON logging
- Admin endpoints (`/healthz`, `/readyz`, `/metrics`)
- Hot config reload
- Improved health checks with thresholds
