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
	mainRoute := cfg.Routes[0]
	targets := make([]string, len(mainRoute.Backends))
	for i, backend := range mainRoute.Backends {
		targets[i] = backend.Addr
	}
	balancer := lb.RoundRobin{Servers: targets}
	proxyHandler := engine.NewEngine(&balancer, pool)

	return &Server{
		config: cfg,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Listener.Port),
			Handler:      proxyHandler,
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
