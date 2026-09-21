package api

import (
	"log/slog"
	"net/http"

	"github.com/mailvex/mailvex/api/handlers"
	"github.com/mailvex/mailvex/api/middleware"
	"github.com/mailvex/mailvex/core"
	"github.com/mailvex/mailvex/core/audit"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/dns"
	"github.com/mailvex/mailvex/core/repository"
	"github.com/mailvex/mailvex/core/storage"
)

// Server configures HTTP routes and middleware for Mailvex API
type Server struct {
	mux          *http.ServeMux
	handler      http.Handler
	repo         repository.Repository
	audit        *audit.Service
	encryption   *crypto.EnvelopeEncryption
	dnsEngine    *dns.Engine
	storage      storage.Storage
	ingestEngine *core.IngestionEngine
	logger       *slog.Logger
}

func NewServer(repo repository.Repository, auditService *audit.Service, enc *crypto.EnvelopeEncryption, logger *slog.Logger) *Server {
	store := storage.NewMemoryStorage()
	s := &Server{
		mux:          http.NewServeMux(),
		repo:         repo,
		audit:        auditService,
		encryption:   enc,
		dnsEngine:    dns.NewEngine(dns.NewSystemResolver()),
		storage:      store,
		ingestEngine: core.NewIngestionEngine(repo, store),
		logger:       logger,
	}

	s.setupRoutes()
	return s
}

func (s *Server) SetDNSEngine(engine *dns.Engine) {
	s.dnsEngine = engine
}

func (s *Server) SetIngestEngine(engine *core.IngestionEngine, store storage.Storage) {
	s.ingestEngine = engine
	s.storage = store
	s.setupRoutes()
}

func (s *Server) IngestEngine() *core.IngestionEngine {
	return s.ingestEngine
}

func (s *Server) setupRoutes() {
	apiHandler := handlers.NewAPI(s.repo, s.audit, s.encryption)
	apiHandler.SetIngestEngine(s.ingestEngine, s.storage)

	// Public Health Probes
	s.mux.HandleFunc("GET /health", apiHandler.HealthHandler)
	s.mux.HandleFunc("GET /ready", apiHandler.ReadyHandler)

	// Setup Wizard (Pre-auth)
	s.mux.HandleFunc("GET /v1/setup", apiHandler.SetupStatusHandler)
	s.mux.HandleFunc("POST /v1/setup", apiHandler.SetupExecuteHandler)

	// Auth Handlers
	s.mux.HandleFunc("POST /v1/auth/login", apiHandler.LoginHandler)

	// Authenticated Handlers
	authMiddleware := middleware.RequireAuth(s.repo)

	s.mux.Handle("GET /v1/auth/me", authMiddleware(http.HandlerFunc(apiHandler.MeHandler)))
	s.mux.Handle("GET /v1/tenants", authMiddleware(http.HandlerFunc(apiHandler.ListTenantsHandler)))
	s.mux.Handle("POST /v1/tenants", authMiddleware(
		middleware.RequirePermission(s.repo, "tenants.write")(http.HandlerFunc(apiHandler.CreateTenantHandler)),
	))
	s.mux.Handle("GET /v1/audit-logs", authMiddleware(http.HandlerFunc(apiHandler.ListAuditLogsHandler)))

	// Phase 2: Domains & DNS Handlers
	s.mux.Handle("GET /v1/domains", authMiddleware(
		middleware.RequirePermission(s.repo, "domains.read")(http.HandlerFunc(apiHandler.ListDomainsHandler)),
	))
	s.mux.Handle("POST /v1/domains", authMiddleware(
		middleware.RequirePermission(s.repo, "domains.write")(http.HandlerFunc(apiHandler.CreateDomainHandler)),
	))
	s.mux.Handle("GET /v1/domains/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "domains.read")(http.HandlerFunc(apiHandler.GetDomainHandler)),
	))
	s.mux.Handle("POST /v1/domains/{id}/verify", authMiddleware(
		middleware.RequirePermission(s.repo, "domains.verify")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiHandler.VerifyDomainHandler(w, r, s.dnsEngine)
		})),
	))
	s.mux.Handle("POST /v1/domains/{id}/dkim/rotate", authMiddleware(
		middleware.RequirePermission(s.repo, "domains.dkim.rotate")(http.HandlerFunc(apiHandler.RotateDKIMHandler)),
	))

	// Phase 2: Message Streams Handlers
	s.mux.Handle("GET /v1/streams", authMiddleware(
		middleware.RequirePermission(s.repo, "streams.read")(http.HandlerFunc(apiHandler.ListStreamsHandler)),
	))
	s.mux.Handle("POST /v1/streams", authMiddleware(
		middleware.RequirePermission(s.repo, "streams.write")(http.HandlerFunc(apiHandler.CreateStreamHandler)),
	))

	// Phase 3: SMTP Credentials Handlers
	s.mux.Handle("GET /v1/smtp-credentials", authMiddleware(
		middleware.RequirePermission(s.repo, "smtp_credentials.read")(http.HandlerFunc(apiHandler.ListSMTPCredentialsHandler)),
	))
	s.mux.Handle("POST /v1/smtp-credentials", authMiddleware(
		middleware.RequirePermission(s.repo, "smtp_credentials.create")(http.HandlerFunc(apiHandler.CreateSMTPCredentialHandler)),
	))

	// Phase 3: Devices & Trusted Networks
	s.mux.Handle("GET /v1/devices", authMiddleware(http.HandlerFunc(apiHandler.ListDevicesHandler)))
	s.mux.Handle("POST /v1/devices", authMiddleware(http.HandlerFunc(apiHandler.CreateDeviceHandler)))
	s.mux.Handle("POST /v1/trusted-networks", authMiddleware(http.HandlerFunc(apiHandler.AddTrustedNetworkHandler)))

	// Phase 3: Message Logs & Development Sink
	s.mux.Handle("GET /v1/messages", authMiddleware(
		middleware.RequirePermission(s.repo, "messages.read")(http.HandlerFunc(apiHandler.ListMessagesHandler)),
	))
	s.mux.Handle("GET /v1/messages/development", authMiddleware(
		middleware.RequirePermission(s.repo, "messages.read")(http.HandlerFunc(apiHandler.ListDevMessagesHandler)),
	))

	// Phase 4: API Sending & Batch
	s.mux.Handle("POST /v1/email", authMiddleware(
		middleware.RequirePermission(s.repo, "send.email")(http.HandlerFunc(apiHandler.SendEmailHandler)),
	))
	s.mux.Handle("POST /v1/email/batch", authMiddleware(
		middleware.RequirePermission(s.repo, "send.email")(http.HandlerFunc(apiHandler.SendEmailBatchHandler)),
	))

	// Phase 4: Message Inspection, Body, Retry, and Cancel
	s.mux.Handle("GET /v1/messages/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "messages.read")(http.HandlerFunc(apiHandler.GetMessageHandler)),
	))
	s.mux.Handle("GET /v1/messages/{id}/recipients", authMiddleware(
		middleware.RequirePermission(s.repo, "messages.read")(http.HandlerFunc(apiHandler.GetMessageRecipientsHandler)),
	))
	s.mux.Handle("GET /v1/messages/{id}/events", authMiddleware(
		middleware.RequirePermission(s.repo, "messages.read")(http.HandlerFunc(apiHandler.GetMessageEventsHandler)),
	))
	s.mux.Handle("GET /v1/messages/{id}/body", authMiddleware(
		middleware.RequirePermission(s.repo, "messages.body.read")(http.HandlerFunc(apiHandler.GetMessageBodyHandler)),
	))
	s.mux.Handle("POST /v1/messages/{id}/retry", authMiddleware(
		middleware.RequirePermission(s.repo, "messages.retry")(http.HandlerFunc(apiHandler.RetryMessageHandler)),
	))
	s.mux.Handle("POST /v1/messages/{id}/cancel", authMiddleware(
		middleware.RequirePermission(s.repo, "messages.cancel")(http.HandlerFunc(apiHandler.CancelMessageHandler)),
	))

	// Phase 4: API Keys Management
	s.mux.Handle("GET /v1/api-keys", authMiddleware(
		middleware.RequirePermission(s.repo, "api_keys.read")(http.HandlerFunc(apiHandler.ListAPIKeysHandler)),
	))
	s.mux.Handle("POST /v1/api-keys", authMiddleware(
		middleware.RequirePermission(s.repo, "api_keys.create")(http.HandlerFunc(apiHandler.CreateAPIKeyHandler)),
	))
	s.mux.Handle("DELETE /v1/api-keys/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "api_keys.revoke")(http.HandlerFunc(apiHandler.RevokeAPIKeyHandler)),
	))

	// Phase 4: Templates
	s.mux.Handle("GET /v1/templates", authMiddleware(
		middleware.RequirePermission(s.repo, "templates.read")(http.HandlerFunc(apiHandler.ListTemplatesHandler)),
	))
	s.mux.Handle("POST /v1/templates", authMiddleware(
		middleware.RequirePermission(s.repo, "templates.write")(http.HandlerFunc(apiHandler.CreateTemplateHandler)),
	))
	s.mux.Handle("GET /v1/templates/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "templates.read")(http.HandlerFunc(apiHandler.GetTemplateHandler)),
	))
	s.mux.Handle("POST /v1/templates/{id}/render", authMiddleware(
		middleware.RequirePermission(s.repo, "templates.read")(http.HandlerFunc(apiHandler.RenderTemplateHandler)),
	))

	// Phase 5: Suppressions Management
	s.mux.Handle("GET /v1/suppressions", authMiddleware(
		middleware.RequirePermission(s.repo, "suppressions.read")(http.HandlerFunc(apiHandler.ListSuppressionsHandler)),
	))
	s.mux.Handle("POST /v1/suppressions", authMiddleware(
		middleware.RequirePermission(s.repo, "suppressions.write")(http.HandlerFunc(apiHandler.CreateSuppressionHandler)),
	))
	s.mux.Handle("DELETE /v1/suppressions/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "suppressions.write")(http.HandlerFunc(apiHandler.DeleteSuppressionHandler)),
	))

	// Phase 5: RFC 8058 One-Click & Hosted Unsubscribe Endpoints (§17.3)
	unsubHandler := handlers.NewUnsubscribeHandler(s.repo, []byte("mailvex-unsubscribe-token-secret"))
	s.mux.HandleFunc("POST /u/{token}", unsubHandler.HandleOneClick)
	s.mux.HandleFunc("GET /u/{token}", unsubHandler.HandleHostedPage)

	// Phase 6: Abuse Reports Management & Intake
	s.mux.Handle("GET /v1/abuse-reports", authMiddleware(
		middleware.RequirePermission(s.repo, "abuse.read")(http.HandlerFunc(apiHandler.ListAbuseReportsHandler)),
	))
	s.mux.Handle("GET /v1/abuse-reports/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "abuse.read")(http.HandlerFunc(apiHandler.GetAbuseReportHandler)),
	))
	s.mux.Handle("POST /v1/abuse-reports/{id}/resolve", authMiddleware(
		middleware.RequirePermission(s.repo, "abuse.write")(http.HandlerFunc(apiHandler.ResolveAbuseReportHandler)),
	))
	s.mux.Handle("POST /v1/abuse-reports/{id}/escalate", authMiddleware(
		middleware.RequirePermission(s.repo, "abuse.write")(http.HandlerFunc(apiHandler.EscalateAbuseReportHandler)),
	))

	// Phase 6: Inbound Reports Webhook Ingestion & Provider Snapshots
	s.mux.HandleFunc("POST /v1/reports/dsn", apiHandler.IngestDSNReportHandler)
	s.mux.HandleFunc("POST /v1/reports/arf", apiHandler.IngestARFReportHandler)
	s.mux.Handle("GET /v1/reports/fbl", authMiddleware(
		middleware.RequirePermission(s.repo, "reports.read")(http.HandlerFunc(apiHandler.ListFBLReportsHandler)),
	))
	s.mux.Handle("GET /v1/reports/postmaster", authMiddleware(
		middleware.RequirePermission(s.repo, "reports.read")(http.HandlerFunc(apiHandler.ListPostmasterSnapshotsHandler)),
	))
	s.mux.Handle("POST /v1/reports/postmaster", authMiddleware(
		middleware.RequirePermission(s.repo, "reports.read")(http.HandlerFunc(apiHandler.SavePostmasterSnapshotHandler)),
	))

	// Phase 7: Analytics & Reporting Endpoints (§19.2, §19.3)
	s.mux.Handle("GET /v1/analytics/overview", authMiddleware(
		middleware.RequirePermission(s.repo, "analytics.read")(http.HandlerFunc(apiHandler.GetAnalyticsOverviewHandler)),
	))
	s.mux.Handle("GET /v1/analytics/time-series", authMiddleware(
		middleware.RequirePermission(s.repo, "analytics.read")(http.HandlerFunc(apiHandler.GetAnalyticsTimeSeriesHandler)),
	))
	s.mux.Handle("GET /v1/analytics/providers", authMiddleware(
		middleware.RequirePermission(s.repo, "analytics.read")(http.HandlerFunc(apiHandler.GetAnalyticsProvidersHandler)),
	))
	s.mux.Handle("GET /v1/analytics/latency", authMiddleware(
		middleware.RequirePermission(s.repo, "analytics.read")(http.HandlerFunc(apiHandler.GetAnalyticsLatencyHandler)),
	))
	s.mux.Handle("POST /v1/analytics/export", authMiddleware(
		middleware.RequirePermission(s.repo, "analytics.export")(http.HandlerFunc(apiHandler.CreateAnalyticsExportHandler)),
	))
	s.mux.Handle("GET /v1/analytics/exports", authMiddleware(
		middleware.RequirePermission(s.repo, "analytics.read")(http.HandlerFunc(apiHandler.ListAnalyticsExportsHandler)),
	))
	s.mux.Handle("GET /v1/analytics/exports/{id}/download", authMiddleware(
		middleware.RequirePermission(s.repo, "analytics.read")(http.HandlerFunc(apiHandler.DownloadAnalyticsExportHandler)),
	))

	// Phase 8: Webhook Delivery Engine (§20)
	s.mux.Handle("POST /v1/webhooks", authMiddleware(
		middleware.RequirePermission(s.repo, "webhooks.write")(http.HandlerFunc(apiHandler.CreateWebhookEndpointHandler)),
	))
	s.mux.Handle("GET /v1/webhooks", authMiddleware(
		middleware.RequirePermission(s.repo, "webhooks.read")(http.HandlerFunc(apiHandler.ListWebhookEndpointsHandler)),
	))
	s.mux.Handle("GET /v1/webhooks/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "webhooks.read")(http.HandlerFunc(apiHandler.GetWebhookEndpointHandler)),
	))
	s.mux.Handle("PATCH /v1/webhooks/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "webhooks.write")(http.HandlerFunc(apiHandler.UpdateWebhookEndpointHandler)),
	))
	s.mux.Handle("DELETE /v1/webhooks/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "webhooks.write")(http.HandlerFunc(apiHandler.DeleteWebhookEndpointHandler)),
	))
	s.mux.Handle("POST /v1/webhooks/{id}/rotate-secret", authMiddleware(
		middleware.RequirePermission(s.repo, "webhooks.secret.rotate")(http.HandlerFunc(apiHandler.RotateWebhookSecretHandler)),
	))
	s.mux.Handle("POST /v1/webhooks/{id}/test", authMiddleware(
		middleware.RequirePermission(s.repo, "webhooks.write")(http.HandlerFunc(apiHandler.TestWebhookEndpointHandler)),
	))
	s.mux.Handle("GET /v1/webhooks/{id}/deliveries", authMiddleware(
		middleware.RequirePermission(s.repo, "webhooks.read")(http.HandlerFunc(apiHandler.ListWebhookDeliveriesHandler)),
	))

	// Phase 9: Suppression Management, Privacy & Retention Engine (§20, §23)
	s.mux.Handle("POST /v1/suppressions/import", authMiddleware(
		middleware.RequirePermission(s.repo, "suppressions.import")(http.HandlerFunc(apiHandler.ImportSuppressionsHandler)),
	))
	s.mux.Handle("GET /v1/suppressions/export", authMiddleware(
		middleware.RequirePermission(s.repo, "suppressions.read")(http.HandlerFunc(apiHandler.ExportSuppressionsHandler)),
	))
	s.mux.Handle("POST /v1/privacy/erasure", authMiddleware(
		middleware.RequirePermission(s.repo, "privacy.erase")(http.HandlerFunc(apiHandler.PrivacyErasureHandler)),
	))
	s.mux.Handle("POST /v1/privacy/export", authMiddleware(
		middleware.RequirePermission(s.repo, "privacy.export")(http.HandlerFunc(apiHandler.PrivacyExportHandler)),
	))
	s.mux.Handle("GET /v1/privacy/requests", authMiddleware(
		middleware.RequirePermission(s.repo, "privacy.read")(http.HandlerFunc(apiHandler.ListPrivacyRequestsHandler)),
	))
	s.mux.Handle("GET /v1/retention", authMiddleware(
		middleware.RequirePermission(s.repo, "retention.read")(http.HandlerFunc(apiHandler.GetRetentionPolicyHandler)),
	))
	s.mux.Handle("PUT /v1/retention", authMiddleware(
		middleware.RequirePermission(s.repo, "retention.write")(http.HandlerFunc(apiHandler.UpdateRetentionPolicyHandler)),
	))
	s.mux.Handle("POST /v1/retention/purge", authMiddleware(
		middleware.RequirePermission(s.repo, "retention.write")(http.HandlerFunc(apiHandler.ExecuteRetentionPurgeHandler)),
	))

	// Apply global middleware: RequestID -> SecurityHeaders -> Logger
	var h http.Handler = s.mux
	h = middleware.SecurityHeaders(h)
	h = middleware.Logger(s.logger)(h)
	h = middleware.RequestID(h)

	s.handler = h
}

func (s *Server) Handler() http.Handler {
	return s.handler
}
