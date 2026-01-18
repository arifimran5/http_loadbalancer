package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
)

// Config represents the complete load balancer configuration.
type Config struct {
	LoadBalancer LoadBalancerConfig `json:"load_balancer"`
}

// LoadBalancerConfig contains all load balancer settings.
type LoadBalancerConfig struct {
	Port         int                `json:"port"`
	Algorithm    string             `json:"algorithm"`
	Servers      []ServerConfig     `json:"servers"`
	IPWhitelist  []string           `json:"ip_whitelist"`
	IPBlacklist  []string           `json:"ip_blacklist"`
	RateLimiting RateLimitingConfig `json:"rate_limiting"`
	MaxInFlight  int64              `json:"max_in_flight"`
}

// ServerConfig represents a backend server configuration.
type ServerConfig struct {
	Host                 string `json:"host"`
	HealthyTimeThreshold int    `json:"healthy_time_threshold"` // milliseconds
}

// RateLimitingConfig holds rate limiting settings.
type RateLimitingConfig struct {
	Enabled           bool `json:"enabled"`
	RequestsPerSecond int  `json:"requests_per_second"`
	BurstLimit        int  `json:"burst_limit"`
}

// LoadConfig reads and parses the configuration from a JSON file.
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(byteValue, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.LoadBalancer.Port <= 0 || c.LoadBalancer.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.LoadBalancer.Port)
	}

	validAlgorithms := map[string]bool{
		"roundrobin": true,
		"least_conn": true,
		"iphash":     true,
	}
	if !validAlgorithms[c.LoadBalancer.Algorithm] {
		return fmt.Errorf("invalid algorithm: %s (valid: roundrobin, least_conn, iphash)", c.LoadBalancer.Algorithm)
	}

	if len(c.LoadBalancer.Servers) == 0 {
		return fmt.Errorf("no servers configured")
	}

	seen := make(map[string]bool)
	for i, server := range c.LoadBalancer.Servers {
		if server.Host == "" {
			return fmt.Errorf("server %d: host is required", i)
		}
		if _, err := url.Parse(server.Host); err != nil {
			return fmt.Errorf("server %d: invalid host URL: %w", i, err)
		}
		if seen[server.Host] {
			return fmt.Errorf("server %d: duplicate host: %s", i, server.Host)
		}
		seen[server.Host] = true

		if server.HealthyTimeThreshold <= 0 {
			return fmt.Errorf("server %d: healthy_time_threshold must be positive", i)
		}
	}

	if c.LoadBalancer.RateLimiting.Enabled {
		if c.LoadBalancer.RateLimiting.RequestsPerSecond <= 0 {
			return fmt.Errorf("rate_limiting: requests_per_second must be positive")
		}
		if c.LoadBalancer.RateLimiting.BurstLimit <= 0 {
			return fmt.Errorf("rate_limiting: burst_limit must be positive")
		}
	}

	if c.LoadBalancer.MaxInFlight <= 0 {
		return fmt.Errorf("max_in_flight must be positive")
	}

	return nil
}
