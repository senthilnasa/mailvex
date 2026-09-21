package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mailvex/mailvex/api"
	"github.com/mailvex/mailvex/core"
	"github.com/mailvex/mailvex/core/audit"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/dns"
	"github.com/mailvex/mailvex/core/repository"
	"github.com/mailvex/mailvex/core/storage"
	"github.com/mailvex/mailvex/smtp"
	"github.com/mailvex/mailvex/worker"
)

// AllInOneSupervisor coordinates all Mailvex services in a single process for homelab and edge deployments (§28)
type AllInOneSupervisor struct {
	repo           repository.Repository
	store          storage.Storage
	apiServer      *api.Server
	httpSrv        *http.Server
	smtpServer     *smtp.Server
	deliveryWorker *worker.DeliveryWorker
	logger         *slog.Logger
	stopWkChan     chan struct{}
}

func NewAllInOneSupervisor(repo repository.Repository, store storage.Storage, logger *slog.Logger, smtpAddr string) *AllInOneSupervisor {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	kekBytes := []byte("mailvex-homelab-32byte-masterkek")
	enc, _ := crypto.NewEnvelopeEncryption(kekBytes)
	auditSvc := audit.NewService(repo)

	apiSrv := api.NewServer(repo, auditSvc, enc, logger)
	dnsEngine := dns.NewEngine(nil)
	apiSrv.SetDNSEngine(dnsEngine)

	ingestEngine := core.NewIngestionEngine(repo, store)
	apiSrv.SetIngestEngine(ingestEngine, store)

	// SMTP Ingestion Edge
	if smtpAddr == "" {
		smtpAddr = ":2525"
	}
	smtpBackend := smtp.NewBackend(repo, ingestEngine)
	smtpServer := smtp.NewServer(smtpAddr, "mailvex.local", smtpBackend)

	// Outbound Delivery Worker
	router := worker.NewRouter(repo, "homelab-node")
	resolver := worker.NewDNSMXResolver(10 * time.Minute)
	rules, _ := repo.ListBounceRules(context.Background())
	classifier := worker.NewClassifier(rules)

	workerCfg := worker.WorkerConfig{
		NodeID:        "homelab-node",
		PlatformHost:  "mailvex.local",
		BounceDomain:  "bounce.mailvex.local",
		DefaultPort:   25,
		ShutdownGrace: 10 * time.Second,
	}
	deliveryWorker := worker.NewDeliveryWorker(repo, store, router, resolver, classifier, workerCfg)

	return &AllInOneSupervisor{
		repo:           repo,
		store:          store,
		apiServer:      apiSrv,
		smtpServer:     smtpServer,
		deliveryWorker: deliveryWorker,
		logger:         logger,
		stopWkChan:     make(chan struct{}),
	}
}

func (s *AllInOneSupervisor) Start(ctx context.Context, apiAddr string) error {
	if apiAddr == "" {
		apiAddr = ":8080"
	}

	s.logger.Info("Starting Mailvex All-in-One Process Supervisor", "api_addr", apiAddr, "profile", "homelab")

	// 1. Start HTTP API
	s.httpSrv = &http.Server{
		Addr:    apiAddr,
		Handler: s.apiServer.Handler(),
	}
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("API Server failed", "error", err)
		}
	}()

	// 2. Start SMTP Inbound Edge
	go func() {
		if err := s.smtpServer.ListenAndServe(); err != nil {
			s.logger.Error("SMTP Edge listener closed", "error", err)
		}
	}()

	// 3. Start Outbound Delivery Worker Loop
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopWkChan:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.deliveryWorker.ProcessOutboxBatch(ctx, 50)
			}
		}
	}()

	s.logger.Info("All Mailvex microservices initialized inside unified supervisor ✅",
		"http_api", apiAddr,
		"status", "running",
	)

	return nil
}

func (s *AllInOneSupervisor) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Mailvex All-in-One Supervisor gracefully...")

	close(s.stopWkChan)

	if s.smtpServer != nil {
		_ = s.smtpServer.Close()
	}
	if s.httpSrv != nil {
		_ = s.httpSrv.Shutdown(ctx)
	}

	s.logger.Info("Mailvex All-in-One Supervisor stopped cleanly.")
	return nil
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	repo := repository.NewMemoryRepository()
	store := storage.NewMemoryStorage()

	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "2525"
	}

	supervisor := NewAllInOneSupervisor(repo, store, logger, ":"+smtpPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	if err := supervisor.Start(ctx, ":"+apiPort); err != nil {
		logger.Error("Failed to start supervisor", "error", err)
		os.Exit(1)
	}

	// Trap termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	sig := <-sigChan
	logger.Info("Received termination signal", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := supervisor.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", "error", err)
	}
	fmt.Println("Mailvex shutdown complete.")
}
