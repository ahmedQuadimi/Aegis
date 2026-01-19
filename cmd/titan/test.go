package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"github.com/ahmedQuadimi/Titan/internal/adapters/config"
)

func mainTest() {
	cfg := config.Load()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Titan proxy is available"))
	})
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: mux,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		fmt.Printf("The Titans are heading out on port: %d\n", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic("The server is now Closed")
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("The titan has officially ShutDown")
}

