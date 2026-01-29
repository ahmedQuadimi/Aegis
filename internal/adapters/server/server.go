package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ahmedQuadimi/Aegis/internal/adapters/config"
	"github.com/ahmedQuadimi/Aegis/internal/cache"
	"github.com/ahmedQuadimi/Aegis/internal/engine"
	"github.com/ahmedQuadimi/Aegis/internal/lb"
	"github.com/ahmedQuadimi/Aegis/internal/middleware"
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

	for _, routeCfg := range cfg.Routes {
		handler := buildRouteHandler(routeCfg, pool)
		dispatcher.AddRoute(routeCfg.Host, handler)
	}
	finalHandler := middleware.LoggerMiddleware(dispatcher)

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

func buildRouteHandler(cfg config.RouteConfig, pool *sync.Pool) http.Handler {
	targets := make([]*lb.Backend, len(cfg.Backends))
	backendMap := make(map[string]*lb.Backend)

	for i, backend := range cfg.Backends {
		b := &lb.Backend{
			Addr:   backend.Addr,
			Config: backend.HealthCheck,
			Alive:  true,
		}
		targets[i] = b
		backendMap[b.Addr] = b
		go lb.RunHealthCheck(b)
	}

	balancer := lb.RoundRobin{Backends: targets}

	proxyEngine := engine.NewEngine(&balancer, pool, cfg, backendMap)

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
		limiter := middleware.NewIPRateLimiter(rate.Limit(cfg.RateLimit), cfg.Burst, cfg)
		handler = limiter.RateLimitMiddleware(handler)
	}

	return handler
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
