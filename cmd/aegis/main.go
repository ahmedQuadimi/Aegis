package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "github.com/ahmedQuadimi/Aegis/internal/adapters/config"
	aegisHttp "github.com/ahmedQuadimi/Aegis/internal/adapters/http"
	"github.com/ahmedQuadimi/Aegis/internal/middleware"
)

func main() {
	middleware.SetupLoggerMiddleware()
	cfg, err := config.Load("config.yaml")
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	srv := aegisHttp.NewServer(cfg)
	go func() {
		if err := srv.Start(); err != nil {
			panic(fmt.Sprintf("Failed to start Aegis: %v", err))
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\nAegis is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Aegis forced to shutdown: %v\n", err)
	}
	fmt.Println("Aegis is no longer with us")
}
