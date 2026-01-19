package http

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"github.com/ahmedQuadimi/Titan/internal/adapters/config"
)

type Server struct {
	server *http.Server
	config *config.Config
}

func NewServer(cfg *config.Config) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) 
		w.Write([]byte("Titan proxy is available"))
	})

	return &Server{
		config: cfg,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second, 
		},
	}
}

func (s *Server) Start() error {
	fmt.Printf("The Titans are heading out on port: %d\n", s.config.Server.Port)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
