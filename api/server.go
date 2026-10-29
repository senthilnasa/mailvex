package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/senthilnasa/mailvex/api/handlers"
	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core"
	"github.com/senthilnasa/mailvex/core/audit"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/dns"
	"github.com/senthilnasa/mailvex/core/metrics"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/storage"
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
	if enc == nil {
		enc, _ = crypto.NewEnvelopeEncryption([]byte("mailvex-homelab-32byte-masterkek"))
	}
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
	s.mux = http.NewServeMux()
	apiHandler := handlers.NewAPI(s.repo, s.audit, s.encryption)
	apiHandler.SetIngestEngine(s.ingestEngine, s.storage)

	// Public Health & Observability Probes
	s.mux.HandleFunc("GET /health", apiHandler.HealthHandler)
	s.mux.HandleFunc("GET /ready", apiHandler.ReadyHandler)
	s.mux.Handle("GET /metrics", metrics.Handler())

	// Setup Wizard (Pre-auth)
	s.mux.HandleFunc("GET /v1/setup", apiHandler.SetupStatusHandler)
	s.mux.HandleFunc("POST /v1/setup", apiHandler.SetupExecuteHandler)

	// Auth Handlers
	s.mux.HandleFunc("POST /v1/auth/login", apiHandler.LoginHandler)
	s.mux.HandleFunc("POST /v1/auth/reset-admin", apiHandler.ResetAdminHandler)

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
	s.mux.Handle("POST /v1/send", authMiddleware(
		middleware.RequirePermission(s.repo, "send.email")(http.HandlerFunc(apiHandler.SendEmailHandler)),
	))
	s.mux.Handle("POST /v1/send/batch", authMiddleware(
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

	// Phase 10: Open & Click Tracking Endpoints (§19)
	s.mux.HandleFunc("GET /o/{token}", apiHandler.HandleOpenPixel)
	s.mux.HandleFunc("GET /c/{token}", apiHandler.HandleClickRedirect)

	// Phase 10: Inbound Email Routing & Message Management (§21)
	s.mux.Handle("GET /v1/inbound/routes", authMiddleware(
		middleware.RequirePermission(s.repo, "inbound.read")(http.HandlerFunc(apiHandler.ListInboundRoutesHandler)),
	))
	s.mux.Handle("POST /v1/inbound/routes", authMiddleware(
		middleware.RequirePermission(s.repo, "inbound.write")(http.HandlerFunc(apiHandler.CreateInboundRouteHandler)),
	))
	s.mux.Handle("GET /v1/inbound/routes/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "inbound.read")(http.HandlerFunc(apiHandler.GetInboundRouteHandler)),
	))
	s.mux.Handle("PATCH /v1/inbound/routes/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "inbound.write")(http.HandlerFunc(apiHandler.UpdateInboundRouteHandler)),
	))
	s.mux.Handle("DELETE /v1/inbound/routes/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "inbound.write")(http.HandlerFunc(apiHandler.DeleteInboundRouteHandler)),
	))
	s.mux.Handle("POST /v1/inbound/routes/test", authMiddleware(
		middleware.RequirePermission(s.repo, "inbound.read")(http.HandlerFunc(apiHandler.TestInboundRouteMatchHandler)),
	))
	s.mux.Handle("GET /v1/inbound/messages", authMiddleware(
		middleware.RequirePermission(s.repo, "inbound.read")(http.HandlerFunc(apiHandler.ListInboundMessagesHandler)),
	))
	s.mux.Handle("GET /v1/inbound/messages/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "inbound.read")(http.HandlerFunc(apiHandler.GetInboundMessageHandler)),
	))

	// Phase 11: Reputation, DNSBL, Warmup & Alerts (§22, §11.5)
	s.mux.Handle("GET /v1/reputation/overview", authMiddleware(
		middleware.RequirePermission(s.repo, "reputation.read")(http.HandlerFunc(apiHandler.GetReputationOverviewHandler)),
	))
	s.mux.Handle("GET /v1/dnsbl/providers", authMiddleware(
		middleware.RequirePermission(s.repo, "dnsbl.read")(http.HandlerFunc(apiHandler.ListDNSBLProvidersHandler)),
	))
	s.mux.Handle("POST /v1/dnsbl/check", authMiddleware(
		middleware.RequirePermission(s.repo, "dnsbl.check")(http.HandlerFunc(apiHandler.CheckDNSBLHandler)),
	))
	s.mux.Handle("GET /v1/dnsbl/results", authMiddleware(
		middleware.RequirePermission(s.repo, "dnsbl.read")(http.HandlerFunc(apiHandler.ListDNSBLResultsHandler)),
	))
	s.mux.Handle("GET /v1/warmup/plans", authMiddleware(
		middleware.RequirePermission(s.repo, "warmup.read")(http.HandlerFunc(apiHandler.ListWarmupPlansHandler)),
	))
	s.mux.Handle("GET /v1/warmup/plans/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "warmup.read")(http.HandlerFunc(apiHandler.GetWarmupPlanHandler)),
	))
	s.mux.Handle("GET /v1/warmup/ips", authMiddleware(
		middleware.RequirePermission(s.repo, "warmup.read")(http.HandlerFunc(apiHandler.ListIPWarmupStatesHandler)),
	))
	s.mux.Handle("POST /v1/warmup/ips", authMiddleware(
		middleware.RequirePermission(s.repo, "warmup.write")(http.HandlerFunc(apiHandler.EnrollIPWarmupHandler)),
	))
	s.mux.Handle("POST /v1/warmup/ips/{id}/advance", authMiddleware(
		middleware.RequirePermission(s.repo, "warmup.write")(http.HandlerFunc(apiHandler.AdvanceIPWarmupHandler)),
	))
	s.mux.Handle("POST /v1/warmup/ips/{id}/hold", authMiddleware(
		middleware.RequirePermission(s.repo, "warmup.write")(http.HandlerFunc(apiHandler.HoldIPWarmupHandler)),
	))
	s.mux.Handle("POST /v1/warmup/ips/{id}/resume", authMiddleware(
		middleware.RequirePermission(s.repo, "warmup.write")(http.HandlerFunc(apiHandler.ResumeIPWarmupHandler)),
	))
	s.mux.Handle("GET /v1/alerts/rules", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.read")(http.HandlerFunc(apiHandler.ListAlertRulesHandler)),
	))
	s.mux.Handle("POST /v1/alerts/rules", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.write")(http.HandlerFunc(apiHandler.CreateAlertRuleHandler)),
	))
	s.mux.Handle("GET /v1/alerts/rules/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.read")(http.HandlerFunc(apiHandler.GetAlertRuleHandler)),
	))
	s.mux.Handle("DELETE /v1/alerts/rules/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.write")(http.HandlerFunc(apiHandler.DeleteAlertRuleHandler)),
	))
	s.mux.Handle("POST /v1/alerts/evaluate", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.write")(http.HandlerFunc(apiHandler.EvaluateAlertsHandler)),
	))
	s.mux.Handle("GET /v1/alerts/incidents", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.read")(http.HandlerFunc(apiHandler.ListAlertIncidentsHandler)),
	))
	s.mux.Handle("POST /v1/alerts/incidents/{id}/acknowledge", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.write")(http.HandlerFunc(apiHandler.AcknowledgeAlertIncidentHandler)),
	))
	s.mux.Handle("POST /v1/alerts/incidents/{id}/resolve", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.write")(http.HandlerFunc(apiHandler.ResolveAlertIncidentHandler)),
	))
	s.mux.Handle("GET /v1/alerts/channels", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.read")(http.HandlerFunc(apiHandler.ListNotificationChannelsHandler)),
	))
	s.mux.Handle("POST /v1/alerts/channels", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.write")(http.HandlerFunc(apiHandler.CreateNotificationChannelHandler)),
	))
	s.mux.Handle("DELETE /v1/alerts/channels/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.write")(http.HandlerFunc(apiHandler.DeleteNotificationChannelHandler)),
	))
	s.mux.Handle("POST /v1/alerts/channels/{id}/test", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.write")(http.HandlerFunc(apiHandler.TestNotificationChannelHandler)),
	))
	s.mux.Handle("GET /v1/alerts/notifications", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.read")(http.HandlerFunc(apiHandler.ListAlertNotificationsHandler)),
	))
	s.mux.Handle("POST /v1/alerts/notifications/{id}/read", authMiddleware(
		middleware.RequirePermission(s.repo, "alerts.write")(http.HandlerFunc(apiHandler.MarkAlertNotificationReadHandler)),
	))

	// Phase 12: SSO, Impersonation & Deliverability Compliance (§7.2, §7.4, §22)
	s.mux.Handle("GET /v1/auth/sso/providers", authMiddleware(
		middleware.RequirePermission(s.repo, "org.read")(http.HandlerFunc(apiHandler.ListSSOProvidersHandler)),
	))
	s.mux.Handle("POST /v1/auth/sso/providers", authMiddleware(
		middleware.RequirePermission(s.repo, "org.sso.write")(http.HandlerFunc(apiHandler.CreateSSOProviderHandler)),
	))
	s.mux.Handle("GET /v1/auth/sso/providers/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "org.read")(http.HandlerFunc(apiHandler.GetSSOProviderHandler)),
	))
	s.mux.Handle("DELETE /v1/auth/sso/providers/{id}", authMiddleware(
		middleware.RequirePermission(s.repo, "org.sso.write")(http.HandlerFunc(apiHandler.DeleteSSOProviderHandler)),
	))

	// Public SSO Callback & Metadata Endpoints (No bearer auth required)
	s.mux.HandleFunc("GET /v1/auth/sso/saml/metadata", apiHandler.GetSAMLMetadataHandler)
	s.mux.HandleFunc("POST /v1/auth/sso/saml/acs", apiHandler.PostSAMLACSHandler)
	s.mux.HandleFunc("POST /v1/auth/sso/oidc/callback", apiHandler.PostOIDCCallbackHandler)

	// Platform Admin Tenant Impersonation (§7.4)
	s.mux.Handle("POST /v1/admin/impersonate", authMiddleware(
		middleware.RequirePermission(s.repo, "impersonate.write")(http.HandlerFunc(apiHandler.StartImpersonationHandler)),
	))
	s.mux.Handle("POST /v1/admin/impersonate/exit", authMiddleware(
		middleware.RequirePermission(s.repo, "impersonate.write")(http.HandlerFunc(apiHandler.ExitImpersonationHandler)),
	))
	s.mux.Handle("GET /v1/admin/impersonate/sessions", authMiddleware(
		middleware.RequirePermission(s.repo, "impersonate.read")(http.HandlerFunc(apiHandler.ListImpersonationSessionsHandler)),
	))

	// Deliverability Compliance & Health Checks
	s.mux.Handle("GET /v1/compliance/check", authMiddleware(
		middleware.RequirePermission(s.repo, "domains.read")(http.HandlerFunc(apiHandler.GetComplianceCheckHandler)),
	))
	s.mux.Handle("POST /v1/health/fcrdns", authMiddleware(
		middleware.RequirePermission(s.repo, "domains.read")(http.HandlerFunc(apiHandler.PostFCrDNSCheckHandler)),
	))
	s.mux.Handle("POST /v1/health/tls", authMiddleware(
		middleware.RequirePermission(s.repo, "domains.read")(http.HandlerFunc(apiHandler.PostTLSCheckHandler)),
	))

	// Static SPA / Console Handler for GET / and frontend assets
	s.setupStaticRoutes()

	// Apply global middleware: RequestID -> Logger -> Recoverer -> SecurityHeaders
	var h http.Handler = s.mux
	h = middleware.SecurityHeaders(h)
	h = middleware.Recoverer(s.logger)(h)
	h = middleware.Logger(s.logger)(h)
	h = middleware.RequestID(h)

	s.handler = h
}

func (s *Server) setupStaticRoutes() {
	staticDir := os.Getenv("MVX_STATIC_DIR")
	if staticDir == "" {
		for _, dir := range []string{"/var/www/mailvex", "./console/dist", "../console/dist"} {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				staticDir = dir
				break
			}
		}
	}

	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(staticDir, filepath.Clean(r.URL.Path))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				fs.ServeHTTP(w, r)
				return
			}
			indexPath := filepath.Join(staticDir, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				http.ServeFile(w, r, indexPath)
				return
			}
			fs.ServeHTTP(w, r)
		})
		return
	}

	// Fallback landing page when console assets are not mounted
	s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if strings.Contains(r.Header.Get("Accept"), "application/json") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"name":        "Mailvex Email Infrastructure API",
				"status":      "healthy",
				"version":     "1.0.0",
				"health_url":  "/health",
				"setup_url":   "/v1/setup",
				"metrics_url": "/metrics",
			})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mailvex API</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background: #0b0f19; color: #f1f5f9; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; }
        .card { background: rgba(30, 41, 59, 0.7); border: 1px solid rgba(255, 255, 255, 0.1); border-radius: 1rem; padding: 2.5rem; max-width: 500px; width: 90%; backdrop-filter: blur(12px); box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5); text-align: center; }
        .badge { display: inline-block; background: #059669; color: #ecfdf5; font-size: 0.75rem; font-weight: 700; text-transform: uppercase; padding: 0.25rem 0.75rem; border-radius: 9999px; margin-bottom: 1rem; }
        h1 { margin: 0 0 0.5rem 0; font-size: 1.75rem; font-weight: 800; letter-spacing: -0.025em; }
        p { color: #94a3b8; font-size: 0.95rem; line-height: 1.5; margin-bottom: 1.75rem; }
        .links { display: flex; flex-direction: column; gap: 0.75rem; }
        .btn { display: block; text-decoration: none; padding: 0.75rem 1rem; border-radius: 0.5rem; font-size: 0.875rem; font-weight: 600; transition: all 0.15s ease; }
        .btn-primary { background: #3b82f6; color: white; }
        .btn-primary:hover { background: #2563eb; }
        .btn-secondary { background: rgba(255, 255, 255, 0.05); color: #cbd5e1; border: 1px solid rgba(255, 255, 255, 0.1); }
        .btn-secondary:hover { background: rgba(255, 255, 255, 0.1); }
    </style>
</head>
<body>
    <div class="card">
        <span class="badge">● Online &amp; Healthy</span>
        <h1>Mailvex API Gateway</h1>
        <p>Your self-hosted email infrastructure platform is up and operational.</p>
        <div class="links">
            <a href="/v1/setup" class="btn btn-primary">Initial Setup Wizard (JSON)</a>
            <a href="/health" class="btn btn-secondary">System Health Endpoint</a>
            <a href="/metrics" class="btn btn-secondary">Prometheus Telemetry Metrics</a>
        </div>
    </div>
</body>
</html>`)
	})
}

func (s *Server) Handler() http.Handler {
	return s.handler
}
