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
		New: func() any {
			return make([]byte, engine.DefaultBufferSize)
		},
	}
	dispatcher := engine.NewDispatcher()
	for _, route := range cfg.Routes {
		targets := make([]*lb.Backend, len(route.Backends))
		for i, backend := range route.Backends {
			targets[i] = &lb.Backend{Addr: backend.Addr, Config: backend.HealthCheck, Alive: true}
			go lb.RunHealthCheck(targets[i])
		}

		balancer := lb.RoundRobin{Backends: targets}
		backendMap := make(map[string]*lb.Backend)
		for _, b := range targets {
			backendMap[b.Addr] = b
		}
		proxyHandler := engine.NewEngine(&balancer, pool, route, backendMap)
		var finalHandler http.Handler = proxyHandler
		slog.Info("Configuring route",
			"host", route.Host,
			"rate_limit", route.RateLimit,
			"burst", route.Burst,
		)

		if route.CacheTTL > 0 {
			slog.Info("Caching enabled",
				"ttl", route.CacheTTL,
			)
			cachePerRoute := middleware.NewCacheMiddleware(cache.NewMemoryCache(), route.CacheTTL)
			finalHandler = cachePerRoute(finalHandler)
		}

		if route.RateLimit > 0 {
			slog.Info("Rate limiting enabled",
				"rps", route.RateLimit,
				"burst", route.Burst,
			)
			limiter := middleware.NewIPRateLimiter(rate.Limit(route.RateLimit), route.Burst, route)
			finalHandler = limiter.RateLimitMiddleware(finalHandler)
		}
		dispatcher.AddRoute(route.Host, finalHandler)
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

func (s *Server) Start() error {
	tlsConfig := &tls.Config{
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

	s.server.TLSConfig = tlsConfig

	slog.Info("The Aegis are heading out",
		"port", s.config.Listener.Port,
		"protocol", s.config.Listener.Protocol,
	)

	if s.config.Listener.Protocol == "https" {
		if s.config.Listener.TLSCert == "" || s.config.Listener.TLSKey == "" {
			return fmt.Errorf("https enabled but cert/key paths are missing")
		}
		if err := s.server.ListenAndServeTLS(s.config.Listener.TLSCert, s.config.Listener.TLSKey); err != nil && err != http.ErrServerClosed {
			return err
		}
	} else {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Aegis is shutting down...")
	return s.server.Shutdown(ctx)
}
