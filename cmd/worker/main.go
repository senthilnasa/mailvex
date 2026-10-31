package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/senthilnasa/mailvex/core/config"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/logger"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/storage"
	"github.com/senthilnasa/mailvex/core/webhooks"
	"github.com/senthilnasa/mailvex/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	appLogger := logger.New(cfg.LogLevel, cfg.LogRedactEmail)
	appLogger.Info("Starting Mailvex Outbound Delivery Worker...",
		"profile", cfg.Profile,
		"env", cfg.AppEnv,
	)

	// Initialize Envelope Encryption with master KEK
	_, err = crypto.NewEnvelopeEncryption(cfg.MasterKey)
	if err != nil {
		appLogger.Error("failed to initialize envelope encryption", "error", err)
		os.Exit(1)
	}

	// Repositories and Storage
	repo := repository.NewMemoryRepository()
	store := storage.NewMemoryStorage()

	// Router and MX Resolver
	router := worker.NewRouter(repo, "node-1")
	resolver := worker.NewDNSMXResolver(10 * time.Minute)

	rules, _ := repo.ListBounceRules(context.Background())
	classifier := worker.NewClassifier(rules)

	workerCfg := worker.WorkerConfig{
		NodeID:        "node-1",
		PlatformHost:  cfg.SMTPHostname,
		BounceDomain:  "bounce." + cfg.SMTPHostname,
		DefaultPort:   25,
		ShutdownGrace: 30 * time.Second,
	}

	deliveryWorker := worker.NewDeliveryWorker(repo, store, router, resolver, classifier, workerCfg)
	deliveryWorker.SetWebhookDispatcher(webhooks.NewDispatcher(repo, nil))

	// Background worker processing loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !deliveryWorker.IsDraining() {
					_, _ = deliveryWorker.ProcessOutboxBatch(ctx, 50)
				}
			}
		}
	}()

	appLogger.Info("Mailvex Worker ready and consuming delivery jobs.")

	// Handle graceful drain on SIGINT / SIGTERM (§15.3)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	appLogger.Info("SIGTERM received, initiating graceful drain...")
	cancel()
	drained := deliveryWorker.Drain(30 * time.Second)
	if drained {
		appLogger.Info("Worker drained successfully. Exiting.")
	} else {
		appLogger.Warn("Worker drain timed out. Exiting.")
	}
	os.Exit(0)
}
