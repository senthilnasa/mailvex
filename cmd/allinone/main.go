package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/senthilnasa/mailvex/api"
	"github.com/senthilnasa/mailvex/core"
	"github.com/senthilnasa/mailvex/core/audit"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/dns"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/storage"
	"github.com/senthilnasa/mailvex/smtp"
	"github.com/senthilnasa/mailvex/worker"
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
	dnsEngine := dns.NewEngine(dns.NewSystemResolver())
	apiSrv.SetDNSEngine(dnsEngine)

	ingestEngine := core.NewIngestionEngine(repo, store)
	apiSrv.SetIngestEngine(ingestEngine, store)

	// SMTP Ingestion Edge
	if smtpAddr == "" {
		smtpAddr = ":2525"
	}
	platformHost := os.Getenv("MVX_PLATFORM_HOST")
	if platformHost == "" {
		platformHost = "mailer.krea.edu.in"
	}
	bounceDomain := os.Getenv("MVX_BOUNCE_DOMAIN")

	smtpBackend := smtp.NewBackend(repo, ingestEngine)
	smtpServer := smtp.NewServer(smtpAddr, platformHost, smtpBackend)

	// Outbound Delivery Worker
	router := worker.NewRouter(repo, "homelab-node")
	resolver := worker.NewDNSMXResolver(10 * time.Minute)
	rules, _ := repo.ListBounceRules(context.Background())
	classifier := worker.NewClassifier(rules)

	workerCfg := worker.WorkerConfig{
		NodeID:        "homelab-node",
		PlatformHost:  platformHost,
		BounceDomain:  bounceDomain,
		DefaultPort:   25,
		ShutdownGrace: 10 * time.Second,
	}
	deliveryWorker := worker.NewDeliveryWorker(repo, store, router, resolver, classifier, workerCfg)
	deliveryWorker.SetEncryption(enc)

	// Pre-seed default platform administrator credentials for immediate access
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@example.com"
	}
	adminEmail = strings.ToLower(strings.TrimSpace(adminEmail))
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "Password123!"
	}
	orgID := "org_default"
	_ = repo.CreateOrganization(context.Background(), &domain.Organization{
		ID:        orgID,
		Name:      "Default Organization",
		Slug:      "default-org",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})
	tenantID := "tenant_default"
	_ = repo.CreateTenant(context.Background(), &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Production",
		Slug:           "production",
		Environment:    "live",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	})
	if dek, err := enc.GenerateDEK(); err == nil {
		if encDEK, err := enc.EncryptDEK(dek); err == nil {
			_ = repo.SaveTenantKey(context.Background(), &domain.TenantKey{
				ID:           crypto.NewULID(crypto.PrefixAPIKey),
				TenantID:     tenantID,
				KeyVersion:   1,
				EncryptedDEK: encDEK,
				KEKVersion:   "v1",
				CreatedAt:    time.Now().UTC(),
			})
		}
	}
	adminHash, _ := crypto.HashPassword(adminPass)
	adminUser := &domain.User{
		ID:              "user_platform_admin",
		OrganizationID:  orgID,
		Email:           adminEmail,
		PasswordHash:    adminHash,
		Status:          "active",
		IsPlatformAdmin: true,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	_ = repo.CreateUser(context.Background(), adminUser)
	_ = repo.AddTenantMembership(context.Background(), &domain.TenantMembership{
		ID:        "mem_platform_admin",
		TenantID:  tenantID,
		UserID:    adminUser.ID,
		RoleID:    "role_tenant_owner",
		CreatedAt: time.Now().UTC(),
	})
	_ = repo.SetSetting(context.Background(), "setup_completed", "true")

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
