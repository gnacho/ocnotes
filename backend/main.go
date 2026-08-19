// Package main is the entry point for the Notes daemon.
//
// Daemon Go estático que expone la API REST compatible con Nextcloud Notes v1.x.
// Sirve las rutas bajo /index.php/apps/notes/api/v1/ y se despliega como servicio
// systemd independiente detrás de Nginx Proxy Manager. La extensión Vue montada en
// OpenCloud consume esa misma base HTTP con Bearer (sesión web).
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/api"
	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/auth"
	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/config"
	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/crypto"
	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/store"
)

func main() {
	cfg := config.Load()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("ocnotes starting addr=%s dataDir=%s", cfg.Addr, cfg.DataDir)

	db, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	secretKey, err := crypto.EnsureSecretKey(cfg.DataDir, "notessecret")
	if err != nil {
		log.Fatalf("ensure secret key: %v", err)
	}

	validator := auth.NewOpenCloudValidator(cfg.GraphURL, secretKey)
	storeInstance := store.New(db, validator)

	handler := api.NewServer(api.Base, storeInstance, validator)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler.Router(),
		ReadHeaderTimeout: 10 * 1024, // anti slowloris
	}

	idle := make(chan os.Signal, 1)
	signal.Notify(idle, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-idle
		log.Println("shutting down...")
		ctx, cancel := signal.Context(idle, syscall.SIGTERM)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	log.Println("stopped")
}
