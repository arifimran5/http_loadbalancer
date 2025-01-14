package config

import (
	"encoding/json"
	"io"
	"os"
)

type Config struct {
	LoadBalancer struct {
		Port         int                `json:"port"`
		Algorithm    string             `json:"algorithm"`
		Servers      []ServerConfig     `json:"servers"`
		IPWhitelist  []string           `json:"ip_whitelist"`
		IPBlacklist  []string           `json:"ip_blacklist"`
		RateLimiting RateLimitingConfig `json:"rate_limiting"`
	} `json:"load_balancer"`
}

type ServerConfig struct {
	Host                   string `json:"host"`
	Healthy_Time_Threshold int    `json:"healthy_time_threshold"`
}

type RateLimitingConfig struct {
	Enabled           bool `json:"enabled"`
	RequestsPerSecond int  `json:"requests_per_second"`
	BurstLimit        int  `json:"burst_limit"`
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	var config Config
	if err := json.Unmarshal(byteValue, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
