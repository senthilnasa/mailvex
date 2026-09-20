package api

import (
	"log/slog"
	"net/http"

	"github.com/mailvex/mailvex/api/handlers"
	"github.com/mailvex/mailvex/api/middleware"
	"github.com/mailvex/mailvex/core/audit"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/dns"
	"github.com/mailvex/mailvex/core/repository"
)

// Server configures HTTP routes and middleware for Mailvex API
type Server struct {
	mux        *http.ServeMux
	handler    http.Handler
	repo       repository.Repository
	audit      *audit.Service
	encryption *crypto.EnvelopeEncryption
	dnsEngine  *dns.Engine
	logger     *slog.Logger
}

func NewServer(repo repository.Repository, auditService *audit.Service, enc *crypto.EnvelopeEncryption, logger *slog.Logger) *Server {
	s := &Server{
		mux:        http.NewServeMux(),
		repo:       repo,
		audit:      auditService,
		encryption: enc,
		dnsEngine:  dns.NewEngine(dns.NewSystemResolver()),
		logger:     logger,
	}

	s.setupRoutes()
	return s
}

func (s *Server) SetDNSEngine(engine *dns.Engine) {
	s.dnsEngine = engine
}

func (s *Server) setupRoutes() {
	apiHandler := handlers.NewAPI(s.repo, s.audit, s.encryption)

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
