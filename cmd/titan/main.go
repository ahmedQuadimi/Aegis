package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "github.com/ahmedQuadimi/Titan/internal/adapters/config"
	titanHttp "github.com/ahmedQuadimi/Titan/internal/adapters/http"
)

func main() {
	cfg := config.Load()
	srv := titanHttp.NewServer(cfg)
	go func() {
		if err := srv.Start(); err != nil {
			panic(fmt.Sprintf("Failed to start Titan: %v", err))
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit 
	fmt.Println("\nTitan is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Titan forced to shutdown: %v\n", err)
	}
	fmt.Println("Titan is no longer with us")
}
