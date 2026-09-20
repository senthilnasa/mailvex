package api

import (
	"log/slog"
	"net/http"

	"github.com/mailvex/mailvex/api/handlers"
	"github.com/mailvex/mailvex/api/middleware"
	"github.com/mailvex/mailvex/core/audit"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/repository"
)

// Server configures HTTP routes and middleware for Mailvex API
type Server struct {
	mux        *http.ServeMux
	handler    http.Handler
	repo       repository.Repository
	audit      *audit.Service
	encryption *crypto.EnvelopeEncryption
	logger     *slog.Logger
}

func NewServer(repo repository.Repository, auditService *audit.Service, enc *crypto.EnvelopeEncryption, logger *slog.Logger) *Server {
	s := &Server{
		mux:        http.NewServeMux(),
		repo:       repo,
		audit:      auditService,
		encryption: enc,
		logger:     logger,
	}

	s.setupRoutes()
	return s
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
