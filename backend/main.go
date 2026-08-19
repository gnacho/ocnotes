package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
	"syscall"

	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/api"
	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/auth"
	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/config"
	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/store"
)

func main() {
	cfg := config.Load()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("ocnotes starting addr=%s dataDir=%s", cfg.Addr, cfg.DataDir)

	dbStore, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer dbStore.Close()

	validator := auth.NewOpenCloudValidator(cfg.GraphURL)
	server := api.NewServer(api.Base, dbStore, validator)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Router(),
		ReadHeaderTimeout: 10 * 1024,
	}

	idle := make(chan os.Signal, 1)
	signal.Notify(idle, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-idle
		log.Println("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	log.Println("stopped")
}
