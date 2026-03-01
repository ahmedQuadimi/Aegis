package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ahmedQuadimi/Aegis/internal/adapters/config"
	"github.com/ahmedQuadimi/Aegis/internal/cache"
	"github.com/ahmedQuadimi/Aegis/internal/engine"
	"github.com/ahmedQuadimi/Aegis/internal/lb"
	"github.com/ahmedQuadimi/Aegis/internal/middleware"
	redis_adapter "github.com/ahmedQuadimi/Aegis/internal/redis"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
)

type Server struct {
	server *http.Server
	config *config.Config
}

func NewServer(cfg *config.Config) *Server {
	pool := &sync.Pool{
		New: func() any { return make([]byte, engine.DefaultBufferSize) },
	}
	dispatcher := engine.NewDispatcher()
	mux := http.NewServeMux()

	var rdb *redis_adapter.Client
	if cfg.Redis.Enabled {
		var err error
		rdb, err = redis_adapter.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			slog.Error("Failed to connect to Redis", "error", err)
		} else {
			slog.Info("Connected to Redis")
		}
	}

	if cfg.Observability.MetricsEnabled {
		slog.Info("Observability: ENABLED (Route /metrics is active)")
		mux.Handle("/metrics", promhttp.Handler())
	} else {
		slog.Info("Observability: DISABLED (Lightweight mode)")
	}

	for _, routeCfg := range cfg.Routes {
		handler := buildRouteHandler(cfg, routeCfg, pool, rdb)
		dispatcher.AddRoute(routeCfg.Host, handler)
	}
	mux.Handle("/", dispatcher)

	var finalHandler http.Handler = mux
	if cfg.Observability.MetricsEnabled {
		finalHandler = middleware.MetricsMiddleware(finalHandler)
	}
	finalHandler = middleware.LoggerMiddleware(finalHandler)

	return &Server{
		config: cfg,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Listener.Port),
			Handler:      finalHandler,
			ReadTimeout:  time.Duration(cfg.Defaults.TimeoutRead) * time.Millisecond,
			WriteTimeout: time.Duration(cfg.Defaults.TimeoutWrite) * time.Millisecond,
			IdleTimeout:  time.Duration(cfg.Defaults.TimeoutIdle) * time.Millisecond,
		},
	}
}

func buildRouteHandler(gcfg *config.Config, cfg config.RouteConfig, pool *sync.Pool, rdb *redis_adapter.Client) http.Handler {
	targets := make([]*lb.Backend, len(cfg.Backends))
	backendMap := make(map[string]*lb.Backend)

	for i, backend := range cfg.Backends {
		parsedURL, err := url.Parse(backend.Addr)
		if err != nil {
			panic(fmt.Sprintf("Invalid backend URL %s: %v", backend.Addr, err))
		}
		b := &lb.Backend{
			Addr:   backend.Addr,
			Config: backend.HealthCheck,
			Alive:  true,
			URL:    parsedURL,
		}
		targets[i] = b
		backendMap[b.Addr] = b
		go lb.RunHealthCheck(b)
	}

	balancer := getRightBalancer(&cfg, targets)

	proxyEngine := engine.NewEngine(balancer, pool, cfg, backendMap)

	var handler http.Handler = proxyEngine

	slog.Info("Configuring route",
		"host", cfg.Host,
		"rate_limit", cfg.RateLimit,
		"cache_ttl", cfg.CacheTTL,
	)

	if cfg.CacheTTL > 0 {
		storage := cache.NewMemoryCache()
		handler = middleware.NewCacheMiddleware(storage, cfg.CacheTTL)(handler)
	}

	if cfg.RateLimit > 0 {
		if gcfg.Redis.Enabled && rdb != nil {
			slog.Info("Using REDIS Rate Limiter", "host", cfg.Host)
			handler = middleware.RedisRateLimiter(handler, rdb, cfg)
		} else {
			slog.Info("Using MEMORY Rate Limiter", "host", cfg.Host)
			limiter := middleware.NewIPRateLimiter(rate.Limit(cfg.RateLimit), cfg.Burst, cfg)
			handler = limiter.RateLimitMiddleware(handler)
		}
	}

	return handler
}

func getRightBalancer(cfg *config.RouteConfig, backends []*lb.Backend) lb.Balancer {
	slog.Info("Loading Balancer Strategy", "strategy", cfg.BalancerStrategy)
	switch cfg.BalancerStrategy {
	case lb.BalancerRoundRobin:
		return &lb.RoundRobin{Backends: backends}
	case lb.BalancerLeastConnections:
		return &lb.LeastConnections{Backends: backends}
	}
	panic("Unknown balancer strategy: " + cfg.BalancerStrategy)
}

func (s *Server) Start() error {
	slog.Info("The Aegis are heading out",
		"port", s.config.Listener.Port,
		"protocol", s.config.Listener.Protocol,
	)

	if s.config.Listener.Protocol == "https" {
		return s.listenAndServeTLS()
	}
	return s.server.ListenAndServe()
}

func (s *Server) listenAndServeTLS() error {
	if s.config.Listener.TLSCert == "" || s.config.Listener.TLSKey == "" {
		return fmt.Errorf("https enabled but cert/key paths are missing")
	}

	s.server.TLSConfig = &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.CurveP256, tls.X25519},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	return s.server.ListenAndServeTLS(s.config.Listener.TLSCert, s.config.Listener.TLSKey)
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Aegis is shutting down...")
	return s.server.Shutdown(ctx)
}
