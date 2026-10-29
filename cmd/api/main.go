package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/senthilnasa/mailvex/api"
	"github.com/senthilnasa/mailvex/core/audit"
	"github.com/senthilnasa/mailvex/core/config"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/logger"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/storage"
	"github.com/senthilnasa/mailvex/worker"
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

	// Initialize repository
	repo := repository.NewMemoryRepository()
	auditService := audit.NewService(repo)

	server := api.NewServer(repo, auditService, enc, appLogger)

	// Seed persistent tenant and domain state
	seedInitialState(context.Background(), repo, server.Storage(), enc)

	// Initialize and run background outbound delivery worker loop
	router := worker.NewRouter(repo, "api-node-1")
	resolver := worker.NewDNSMXResolver(10 * time.Minute)
	rules, _ := repo.ListBounceRules(context.Background())
	classifier := worker.NewClassifier(rules)

	workerCfg := worker.WorkerConfig{
		NodeID:        "api-node-1",
		PlatformHost:  cfg.SMTPHostname,
		BounceDomain:  "bounce." + cfg.SMTPHostname,
		DefaultPort:   25,
		ShutdownGrace: 10 * time.Second,
	}
	deliveryWorker := worker.NewDeliveryWorker(repo, server.Storage(), router, resolver, classifier, workerCfg)
	deliveryWorker.SetEncryption(enc)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	go func() {
		appLogger.Info("outbound delivery worker loop started inside api gateway")
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				n, err := deliveryWorker.ProcessOutboxBatch(workerCtx, 50)
				if err != nil {
					appLogger.Error("error processing outbox batch", "error", err)
				} else if n > 0 {
					appLogger.Info("outbound delivery batch processed", "dispatched", n)
				}
			}
		}
	}()

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

func seedInitialState(ctx context.Context, repo repository.Repository, store storage.Storage, enc *crypto.EnvelopeEncryption) {
	orgID := "org_system"
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Default Organization",
		Slug:      "default-org",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})

	tenantID := "tenant_system"
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Production",
		Slug:           "production",
		Environment:    "live",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	})

	if enc != nil {
		if dek, err := enc.GenerateDEK(); err == nil {
			if encDEK, err := enc.EncryptDEK(dek); err == nil {
				_ = repo.SaveTenantKey(ctx, &domain.TenantKey{
					ID:           crypto.NewULID(crypto.PrefixAPIKey),
					TenantID:     tenantID,
					KeyVersion:   1,
					EncryptedDEK: encDEK,
					KEKVersion:   "v1",
					CreatedAt:    time.Now().UTC(),
				})
			}
		}
	}

	adminHash, _ := crypto.HashPassword("Password123!")
	adminUser := &domain.User{
		ID:              "user_platform_admin",
		OrganizationID:  orgID,
		Email:           "erpadmin@krea.edu.in",
		PasswordHash:    adminHash,
		Status:          "active",
		IsPlatformAdmin: true,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	_ = repo.CreateUser(ctx, adminUser)
	_ = repo.AddTenantMembership(ctx, &domain.TenantMembership{
		ID:        "mem_platform_admin",
		TenantID:  tenantID,
		UserID:    adminUser.ID,
		RoleID:    "role_tenant_owner",
		CreatedAt: time.Now().UTC(),
	})

	domainID := "dom_01m64jadpxnbkk6rx6wt9wfmft"
	now := time.Now().UTC()
	_ = repo.CreateDomain(ctx, &domain.Domain{
		ID:            domainID,
		TenantID:      tenantID,
		Name:          "crm.krea.edu.in",
		Type:          "public",
		Status:        "verified",
		VerifiedAt:    &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	})

	const defaultDKIMPrivKey = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDGwgnEYqXI7E3O
fwy8ZrGEK2EQWlGHK5+GarRp2UFOD/v4IB8elBUKDQrNz74StTqW6LmaW23Vo18c
QocILKxIoIB4RMn9NPz/PJVCLnCB//BSOCC/24ZXaVgN8DzMZN9soVVfvUhkNJuv
urkSmqF5xAQp0OFS2NQRT1PvrRw62LwIltpyM403OtKYyF3of237vn1m/RguhS7X
dMbIXdJd/z4Jr+dhDuDhR6NX3dM04ppaP6m3fpjdMUzoFyU3lGOjwaXaaP/RfErA
Yg5ZkXr12J3H24TLx/F7QVQCEjxyjZg1zvBgEQSc8nWJzBg0qIhYGA0hkk7M34uH
73vvzUIPAgMBAAECggEAICWyPrlV5zGRRBz4imBxxdbeVC7HkP1VSABqPdSN43pi
XreTUPbSkFWbsb4wGAaH+3CY1IaFNVC/SWrZPFE9bGHJZap6oSG46e96lMG+IMis
rh+17FItYGYBf/+jVcva908A6ANNeieoJrwkwwWPqIozwi+uwh+f7UzPXqy5LCso
c/0Od2/dydkviMj/QsfsFPyJJyRaNpAtL/KL8GAOGCdfzi4AhLdnIeKYkQnFmvYF
M0Bl/4GxWIuyHNv+vZ+Y90INloTOtIr/HGMrM5MKPHmrg/CBqtskJ//Pcr49xfrH
+aR97BYG/vfTNmUxkrXrO1HmYQkp95h497NmCpU4wQKBgQD/pvw0n8VE5/7fct/b
j1xvPVZGDJtMnzjC6P0SjvQATYBCo0El3pGR/58fwUYYy3tE3dQMLnr3k77RG1C3
qkbyZ7SFc0wOZWzvlMAfU+3lw3zIO4yhQ/T+DFhjECnvJKjLeKAuFR8HhhIiLO0Y
N+rJlc9K9c+LYSMGvr65rBl8DQKBgQDHBz48ON79TbaaaxBPDpQMiUKV75k2hr1b
gYFFQKimZP0lxYswWF9EnEFIL458qgplr0q9arAkzqN+fcyGDNwQRaOEqeevUAD4
dM2bx6lukSEZa/IDhZUl+k9awtISFLDgVt4zsQ6CSCw0Gyu0QJG+V7PymFq+w/ks
GXWYYx3DiwKBgGAyISxcAgqPkSPbQciIBSwGVHQ/QiBuuRj+OlvFVqLWhrCMZXMu
cHATU3jLbOjfiB69qZB73OFTwrVEJfc2J0QHKbIdVD0GhRsRScl/90BBcq6VWbFE
tObzFj8QUfceQDVvyjYM1IQsOp6JcR1vzPTVXg7Ct7mmb1uWYoXS8DPxAoGBALip
Kb8qbQrMfe4MBBe1yPtQpy0CHz+OCEbMg6pDcbbg4FeZT6lNdSbta0yUgzl7cyYh
W9EfGcssxljoU0jUyaF3W14NXl80dZIQnWw59iQWF4XEt8k4uceA1Pu64rnP5MRh
FM0VLAcI+6pkw0n81ryqFtWxMhDpN1kJTihTXhw9AoGBANV/URf00QVcJWuJupDL
Zt/fUtqN1Sj3Gnsl+6F/z+dq9mMrt1NPrnQH2MHVXi9B6neGuogE/XERMnzT22j+
9pWeeLnju44+5d+1k0/+R/FikACjAOm14aGMkYe6RUrkD8Dh9ViJF7UIm9KI+0Re
Uj2MO4wLvHQdWEG2gcUS55xI
-----END PRIVATE KEY-----`
	const defaultDKIMPubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxsIJxGKlyOxNzn8MvGaxhCthEFpRhyufhmq0adlBTg/7+CAfHpQVCg0Kzc++ErU6lui5mltt1aNfHEKHCCysSKCAeETJ/TT8/zyVQi5wgf/wUjggv9uGV2lYDfA8zGTfbKFVX71IZDSbr7q5EpqhecQEKdDhUtjUEU9T760cOti8CJbacjONNzrSmMhd6H9t+759Zv0YLoUu13TGyF3SXf8+Ca/nYQ7g4UejV93TNOKaWj+pt36Y3TFM6BclN5Rjo8Gl2mj/0XxKwGIOWZF69didx9uEy8fxe0FUAhI8co2YNc7wYBEEnPJ1icwYNKiIWBgNIZJOzN+Lh+97781CDwIDAQAB"

	_ = repo.SaveDKIMKey(ctx, &domain.DKIMKey{
		ID:                  crypto.NewULID(crypto.PrefixAPIKey),
		DomainID:            domainID,
		TenantID:            tenantID,
		Selector:            "mvx",
		Algorithm:           "rsa2048",
		EncryptedPrivateKey: defaultDKIMPrivKey,
		PublicKey:           defaultDKIMPubKey,
		Status:              "active",
		CreatedAt:           now,
	})

	streamID := "stream_01m64jyhzg4whb3vkckwm6wsyf"
	_ = repo.CreateStream(ctx, &domain.MessageStream{
		ID:        streamID,
		TenantID:  tenantID,
		Name:      "transactional",
		Type:      "transactional",
		IsDefault: true,
		CreatedAt: now,
	})

	apiKeyID := "key_01m64jn2f9jp6besjm80sv93k1"
	token := "mvx_live_zrUDzzGQoy3wh4xJpMoOHsMD1SwAPsft"
	_ = repo.CreateAPIKey(ctx, &domain.APIKey{
		ID:          apiKeyID,
		TenantID:    tenantID,
		Name:        "Production Live Key",
		Prefix:      token[:12],
		KeyHash:     crypto.HashAPIKey(token),
		LastFour:    token[len(token)-4:],
		Scopes:      []string{"send.email", "messages.read", "domains.read"},
		Environment: "live",
		RateLimit:   100,
		CreatedAt:   now,
	})

	msgID := "msg_01m64jyhzgypqhtde94awheez0"
	rcptID := "rcpt_01m64jyhzggb549fnjww6g31kr"
	rawMIME := []byte("From: erpadmin@crm.krea.edu.in\r\nTo: erpadmin@krea.edu.in\r\nSubject: Test Email from Mailvex (crm.krea.edu.in)\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<p>Hello! This is a test email sent from <strong>Mailvex</strong> through verified domain <code>crm.krea.edu.in</code>.</p>")
	rawRef := fmt.Sprintf("%s/%s/raw.eml", tenantID, msgID)
	_ = store.Put(ctx, rawRef, rawMIME)

	msg := &domain.Message{
		ID:          msgID,
		TenantID:    tenantID,
		StreamID:    streamID,
		DomainID:    domainID,
		Source:      "api",
		APIKeyID:    &apiKeyID,
		FromAddress: "erpadmin@crm.krea.edu.in",
		Subject:     "Test Email from Mailvex (crm.krea.edu.in)",
		Status:      "queued",
		Priority:    1,
		Environment: "live",
		RawMIMERef:  &rawRef,
		CreatedAt:   now,
	}

	verpToken := "cdf35d0883bd25fc"
	rcpt := &domain.MessageRecipient{
		ID:            rcptID,
		MessageID:     msgID,
		TenantID:      tenantID,
		Address:       "erpadmin@krea.edu.in",
		AddressDomain: "krea.edu.in",
		Status:        "queued",
		Attempts:      0,
		VERPToken:     &verpToken,
		CreatedAt:     now,
	}

	payloadBytes, _ := json.Marshal(map[string]string{
		"message_id": msgID,
		"tenant_id":  tenantID,
	})
	outbox := &domain.OutboxItem{
		ID:          crypto.NewULID(crypto.PrefixAPIKey),
		TenantID:    tenantID,
		Stream:      "transactional",
		Payload:     string(payloadBytes),
		AvailableAt: now,
		CreatedAt:   now,
	}

	_ = repo.SaveMessage(ctx, msg, []*domain.MessageRecipient{rcpt}, nil, outbox)

	_ = repo.SetSetting(ctx, "setup_completed", "true")
}

