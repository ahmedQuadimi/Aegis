package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ahmedQuadimi/Aegis/internal/adapters/config"
	"github.com/ahmedQuadimi/Aegis/internal/engine"
	"github.com/ahmedQuadimi/Aegis/internal/lb"
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
	dispatcher := engine.NewDiscpatcher()
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
		dispatcher.AddRoute(route.Host, proxyHandler)
	}

	return &Server{
		config: cfg,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Listener.Port),
			Handler:      dispatcher,
			ReadTimeout:  time.Duration(cfg.Defaults.TimeoutRead) * time.Millisecond,
			WriteTimeout: time.Duration(cfg.Defaults.TimeoutWrite) * time.Millisecond,
			IdleTimeout:  time.Duration(cfg.Defaults.TimeoutIdle) * time.Millisecond,
		},
	}
}

func (s *Server) Start() error {
	fmt.Printf("The Aegis are heading out on port: %d\n", s.config.Listener.Port)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
