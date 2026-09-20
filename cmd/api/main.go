package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mailvex/mailvex/api"
	"github.com/mailvex/mailvex/core/audit"
	"github.com/mailvex/mailvex/core/config"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/logger"
	"github.com/mailvex/mailvex/core/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	appLogger := logger.New(cfg.LogLevel, cfg.LogRedactEmail)
	appLogger.Info("starting mailvex api",
		"profile", cfg.Profile,
		"env", cfg.AppEnv,
		"port", cfg.AppPort,
	)

	// Initialize Envelope Encryption with master KEK
	enc, err := crypto.NewEnvelopeEncryption(cfg.MasterKey)
	if err != nil {
		appLogger.Error("failed to initialize envelope encryption", "error", err)
		os.Exit(1)
	}

	// Initialize repository (In-memory storage for Phase 1 / local dev)
	repo := repository.NewMemoryRepository()
	auditService := audit.NewService(repo)

	server := api.NewServer(repo, auditService, enc, appLogger)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.AppPort),
		Handler:      server.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	appLogger.Info("mailvex api listening", "addr", httpServer.Addr)

	<-stop
	appLogger.Info("shutting down mailvex api gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		appLogger.Error("error during server shutdown", "error", err)
	}
	appLogger.Info("mailvex api stopped")
}
