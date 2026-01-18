package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// TransportConfig holds timeout and connection pool settings for upstream connections.
type TransportConfig struct {
	DialTimeout           time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	IdleConnTimeout       time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	MaxConnsPerHost       int
}

// DefaultTransportConfig returns production-safe default transport settings.
func DefaultTransportConfig() TransportConfig {
	return TransportConfig{
		DialTimeout:           5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       100,
	}
}

// Proxy represents a reverse proxy to a backend server with hardened transport.
type Proxy struct {
	ReverseProxy *httputil.ReverseProxy
	Host         string
	transport    *http.Transport
}

// NewProxy creates a new Proxy instance with default transport configuration.
func NewProxy(host string) (*Proxy, error) {
	return NewProxyWithConfig(host, DefaultTransportConfig())
}

// NewProxyWithConfig creates a new Proxy instance with custom transport configuration.
func NewProxyWithConfig(host string, config TransportConfig) (*Proxy, error) {
	serverURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   config.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		IdleConnTimeout:       config.IdleConnTimeout,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		ForceAttemptHTTP2:     true,
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(serverURL)
	reverseProxy.Transport = transport

	// Custom error handler for upstream failures
	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if err != nil {
			// Determine appropriate error code based on error type
			if isTimeout(err) {
				http.Error(w, "Gateway Timeout", http.StatusGatewayTimeout)
			} else {
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
			}
		}
	}

	return &Proxy{
		ReverseProxy: reverseProxy,
		Host:         host,
		transport:    transport,
	}, nil
}

// ServeHTTP serves HTTP requests by forwarding them to the backend server.
func (p *Proxy) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	p.ReverseProxy.ServeHTTP(res, req)
}

// Close closes idle connections in the transport.
func (p *Proxy) Close() {
	if p.transport != nil {
		p.transport.CloseIdleConnections()
	}
}

// isTimeout checks if the error is a timeout error.
func isTimeout(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}
