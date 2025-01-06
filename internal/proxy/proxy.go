package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Proxy represents a reverse proxy to a backend server.
type Proxy struct {
	ReverseProxy *httputil.ReverseProxy
	Host         string
}

// NewProxy creates a new Proxy instance for the given host.
func NewProxy(host string) (*Proxy, error) {
	serverURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	return &Proxy{
		ReverseProxy: httputil.NewSingleHostReverseProxy(serverURL),
		Host:         host,
	}, nil
}

// ServeHTTP serves HTTP requests by forwarding them to the backend server.
func (p *Proxy) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	p.ReverseProxy.ServeHTTP(res, req)
}
