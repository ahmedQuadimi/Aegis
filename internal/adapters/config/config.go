package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listener ListenerConfig `yaml:"listener"`
	Defaults DefaultConfig  `yaml:"defaults"`
	Routes   []RouteConfig  `yaml:"routes"`
}

type ListenerConfig struct {
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
	TLSCert  string `yaml:"tls_cert"`
	TLSKey   string `yaml:"tls_key"`
}

type DefaultConfig struct {
	HealthCheckInterval int `yaml:"health_check_interval"`
	TimeoutRead         int `yaml:"timeout_read"`
	TimeoutWrite        int `yaml:"timeout_write"`
	TimeoutIdle         int `yaml:"timeout_idle"`
	TimeoutConnect      int `yaml:"timeout_connect"`
}

type RouteConfig struct {
	CacheTTL int             `yaml:"cache_ttl"`
	Host     string          `yaml:"host"`
	Path     string          `yaml:"path"`
	Strategy string          `yaml:"strategy"`
	Retries  int             `yaml:"retries"`
	Backends []BackendConfig `yaml:"backends"`

	RateLimitRPS int    `yaml:"rate_limit_rps"`
	RedisURL     string `yaml:"redis_url"`

	HealthCheckInterval int     `yaml:"health_check_interval"`
	TimeoutConnect      int     `yaml:"timeout_connect"`
	RetryBufferSize     int64   `yaml:"retry_buffer_size"`
	RateLimit           float64 `yaml:"rate_limit"`
	Burst               int     `yaml:"burst"`
	CleanupDuration     int     `yaml:"cleanup_duration"`
}

type BackendConfig struct {
	Addr        string            `yaml:"addr"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

type HealthCheckConfig struct {
	Method             string `yaml:"method"`
	Path               string `yaml:"path"`
	Interval           int    `yaml:"interval"`
	Timeout            int    `yaml:"timeout"`
	UnhealthyThreshold int    `yaml:"unhealthy_threshold"`
	HealthyThreshold   int    `yaml:"healthy_threshold"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
