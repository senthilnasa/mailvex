package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core"
	"github.com/senthilnasa/mailvex/core/analytics"
	"github.com/senthilnasa/mailvex/core/audit"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/inbound"
	"github.com/senthilnasa/mailvex/core/privacy"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/reports"
	"github.com/senthilnasa/mailvex/core/retention"
	"github.com/senthilnasa/mailvex/core/alerts"
	"github.com/senthilnasa/mailvex/core/auth"
	"github.com/senthilnasa/mailvex/core/dnsbl"
	"github.com/senthilnasa/mailvex/core/health"
	"github.com/senthilnasa/mailvex/core/storage"
	"github.com/senthilnasa/mailvex/core/tracking"
	"github.com/senthilnasa/mailvex/core/warmup"
	"github.com/senthilnasa/mailvex/core/webhooks"
)

type API struct {
	repo             repository.Repository
	audit            *audit.Service
	encryption       *crypto.EnvelopeEncryption
	ingest           *core.IngestionEngine
	storage          storage.Storage
	reports          *reports.ReportProcessor
	analytics        *analytics.Engine
	webhooks         *webhooks.Dispatcher
	privacy          *privacy.PrivacyService
	retention        *retention.RetentionService
	tracking         *tracking.TrackingEngine
	inbound          *inbound.InboundEngine
	dnsbl            *dnsbl.Checker
	warmup           *warmup.Manager
	alerts           *alerts.Engine
	ssoEngine        *auth.SSOEngine
	impersonationMgr *auth.ImpersonationManager
	complianceEval   *health.ComplianceEvaluator
	fcrdnsEval       *health.FCrDNSEvaluator
	tlsChecker       *health.TLSChecker
}

func NewAPI(repo repository.Repository, auditService *audit.Service, encryption *crypto.EnvelopeEncryption) *API {
	if encryption == nil {
		encryption, _ = crypto.NewEnvelopeEncryption([]byte("mailvex-homelab-32byte-masterkek"))
	}
	store := storage.NewMemoryStorage()
	alertsEngine := alerts.NewEngine(repo, nil)
	return &API{
		repo:             repo,
		audit:            auditService,
		encryption:       encryption,
		storage:          store,
		ingest:           core.NewIngestionEngine(repo, store),
		reports:          reports.NewReportProcessor(repo, nil, nil),
		analytics:        analytics.NewEngine(repo),
		webhooks:         webhooks.NewDispatcher(repo, nil),
		privacy:          privacy.NewPrivacyService(repo),
		retention:        retention.NewRetentionService(repo),
		tracking:         tracking.NewTrackingEngine(nil, repo),
		inbound:          inbound.NewInboundEngine(repo),
		dnsbl:            dnsbl.NewChecker(repo, nil),
		warmup:           warmup.NewManager(repo),
		alerts:           alertsEngine,
		ssoEngine:        auth.NewSSOEngine(repo),
		impersonationMgr: auth.NewImpersonationManager(repo),
		complianceEval:   health.NewComplianceEvaluator(repo, nil),
		fcrdnsEval:       health.NewFCrDNSEvaluator(nil),
		tlsChecker:       health.NewTLSChecker(),
	}
}

func (a *API) SetIngestEngine(ingest *core.IngestionEngine, store storage.Storage) {
	a.ingest = ingest
	a.storage = store
}

func (a *API) SetReportProcessor(p *reports.ReportProcessor) {
	a.reports = p
}

func (a *API) SetAnalyticsEngine(e *analytics.Engine) {
	a.analytics = e
}

func (a *API) SetWebhookDispatcher(d *webhooks.Dispatcher) {
	a.webhooks = d
}

// HealthHandler returns HTTP 200 for liveness
func (a *API) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

// ReadyHandler checks backend repository readiness
func (a *API) ReadyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ready",
		"checks": map[string]string{
			"database": "ok",
			"redis":    "ok",
		},
	})
}

// SetupStatusHandler checks if initial platform setup has been completed
func (a *API) SetupStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	val, err := a.repo.GetSetting(r.Context(), "setup_completed")
	completed := (err == nil && val == "true")

	_ = json.NewEncoder(w).Encode(map[string]any{
		"setup_completed": completed,
	})
}

type SetupRequest struct {
	Profile       string `json:"profile"` // homelab, standard, cluster
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
	OrgName       string `json:"org_name"`
	TenantName    string `json:"tenant_name"`
}

var nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlphaNumRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "default"
	}
	return s
}

// SetupExecuteHandler runs the initial installation wizard (steps 1–7)
func (a *API) SetupExecuteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	val, err := a.repo.GetSetting(r.Context(), "setup_completed")
	if err == nil && val == "true" {
		http.Error(w, `{"error":{"code":"setup_already_completed","message":"Initial setup is already done"}}`, http.StatusConflict)
		return
	}

	var req SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Malformed JSON"}}`, http.StatusBadRequest)
		return
	}

	if req.AdminEmail == "" || len(req.AdminPassword) < 8 || req.OrgName == "" || req.TenantName == "" {
		http.Error(w, `{"error":{"code":"validation_failed","message":"Missing required setup parameters or password too short"}}`, http.StatusBadRequest)
		return
	}

	if a.encryption == nil {
		a.encryption, _ = crypto.NewEnvelopeEncryption([]byte("mailvex-homelab-32byte-masterkek"))
	}

	req.AdminEmail = strings.ToLower(strings.TrimSpace(req.AdminEmail))

	// 1. Resolve or Create Organization
	orgSlug := slugify(req.OrgName)
	if orgSlug == "" {
		orgSlug = "default-org"
	}

	var orgID string
	existingOrg, err := a.repo.GetOrganizationBySlug(r.Context(), orgSlug)
	if err == nil && existingOrg != nil {
		orgID = existingOrg.ID
	} else {
		orgID = crypto.NewULID(crypto.PrefixOrg)
		org := &domain.Organization{
			ID:        orgID,
			Name:      req.OrgName,
			Slug:      orgSlug,
			Status:    "active",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := a.repo.CreateOrganization(r.Context(), org); err != nil && !errors.Is(err, repository.ErrAlreadyExists) {
			http.Error(w, `{"error":{"code":"org_create_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
			return
		}
	}

	// 2. Resolve or Create Platform Admin User
	pwdHash, err := crypto.HashPassword(req.AdminPassword)
	if err != nil {
		http.Error(w, `{"error":{"code":"hashing_failed","message":"Password hashing failed"}}`, http.StatusInternalServerError)
		return
	}

	var adminID string
	existingUser, err := a.repo.GetUserByEmail(r.Context(), req.AdminEmail)
	if err == nil && existingUser != nil {
		adminID = existingUser.ID
		_ = a.repo.UpdateUserPassword(r.Context(), req.AdminEmail, pwdHash)
	} else {
		adminID = crypto.NewULID(crypto.PrefixUser)
		adminUser := &domain.User{
			ID:              adminID,
			OrganizationID:  orgID,
			Email:           req.AdminEmail,
			PasswordHash:    pwdHash,
			Status:          "active",
			IsPlatformAdmin: true,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}
		if err := a.repo.CreateUser(r.Context(), adminUser); err != nil && !errors.Is(err, repository.ErrAlreadyExists) {
			http.Error(w, `{"error":{"code":"user_create_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
			return
		}
	}

	// 3. Resolve or Create First Tenant
	tenantSlug := slugify(req.TenantName)
	if tenantSlug == "" {
		tenantSlug = "default-tenant"
	}

	var tenantID string
	existingTenant, err := a.repo.GetTenantBySlug(r.Context(), tenantSlug)
	if err == nil && existingTenant != nil {
		tenantID = existingTenant.ID
	} else {
		tenantID = crypto.NewULID(crypto.PrefixTenant)
		tenant := &domain.Tenant{
			ID:             tenantID,
			OrganizationID: orgID,
			Name:           req.TenantName,
			Slug:           tenantSlug,
			Environment:    "live",
			Status:         "active",
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		if err := a.repo.CreateTenant(r.Context(), tenant); err != nil && !errors.Is(err, repository.ErrAlreadyExists) {
			http.Error(w, `{"error":{"code":"tenant_create_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
			return
		}
	}

	// 4. Generate & Store Tenant DEK with Envelope Encryption if not present
	existingKey, _ := a.repo.GetActiveTenantKey(r.Context(), tenantID)
	if existingKey == nil {
		dek, err := a.encryption.GenerateDEK()
		if err == nil {
			encryptedDEK, err := a.encryption.EncryptDEK(dek)
			if err == nil {
				tenantKey := &domain.TenantKey{
					ID:           crypto.NewULID(crypto.PrefixAPIKey),
					TenantID:     tenantID,
					KeyVersion:   1,
					EncryptedDEK: encryptedDEK,
					KEKVersion:   "v1",
					CreatedAt:    time.Now().UTC(),
				}
				_ = a.repo.SaveTenantKey(r.Context(), tenantKey)
			}
		}
	}

	// 5. Add Membership with role_tenant_owner if not exists
	existingMembership, _ := a.repo.GetTenantMembership(r.Context(), tenantID, adminID)
	if existingMembership == nil {
		mem := &domain.TenantMembership{
			ID:        crypto.NewULID(crypto.PrefixMembership),
			TenantID:  tenantID,
			UserID:    adminID,
			RoleID:    "role_tenant_owner",
			CreatedAt: time.Now().UTC(),
		}
		_ = a.repo.AddTenantMembership(r.Context(), mem)
	}

	// Create default message streams (§8.3, §10.6)
	a.createDefaultStreams(r.Context(), tenantID)

	// 6. Resolve or Create System Tenant for platform internal mail
	sysTenantID := "tenant_system"
	existingSysTenant, _ := a.repo.GetTenantBySlug(r.Context(), "system")
	if existingSysTenant != nil {
		sysTenantID = existingSysTenant.ID
	} else {
		sysTenant := &domain.Tenant{
			ID:             sysTenantID,
			OrganizationID: orgID,
			Name:           "System Tenant",
			Slug:           "system",
			Environment:    "live",
			Status:         "active",
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		_ = a.repo.CreateTenant(r.Context(), sysTenant)
	}

	// Mark setup complete
	_ = a.repo.SetSetting(r.Context(), "setup_completed", "true")
	_ = a.repo.SetSetting(r.Context(), "profile", req.Profile)

	// Audit Log
	_ = a.audit.Log(r.Context(), &domain.AuditLog{
		OrgID:        &orgID,
		ActorUserID:  &adminID,
		Action:       "system.setup_completed",
		ResourceType: "system",
		ResourceID:   "platform",
		AfterState:   `{"profile":"` + req.Profile + `"}`,
	})

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":          "Setup completed successfully",
		"organization_id":  orgID,
		"admin_user_id":    adminID,
		"tenant_id":        tenantID,
		"system_tenant_id": sysTenantID,
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginHandler authenticates a user
func (a *API) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Malformed JSON"}}`, http.StatusBadRequest)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	user, err := a.repo.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, `{"error":{"code":"unauthorized","message":"Invalid email or password"}}`, http.StatusUnauthorized)
		return
	}

	valid, err := crypto.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !valid {
		http.Error(w, `{"error":{"code":"unauthorized","message":"Invalid email or password"}}`, http.StatusUnauthorized)
		return
	}

	// Get tenants for user
	tenants, _ := a.repo.ListTenantsForUser(r.Context(), user.ID)

	_ = a.audit.Log(r.Context(), &domain.AuditLog{
		OrgID:        &user.OrganizationID,
		ActorUserID:  &user.ID,
		Action:       "auth.login_success",
		ResourceType: "user",
		ResourceID:   user.ID,
	})

	_ = json.NewEncoder(w).Encode(map[string]any{
		"user": map[string]any{
			"id":                user.ID,
			"email":             user.Email,
			"organization_id":   user.OrganizationID,
			"is_platform_admin": user.IsPlatformAdmin,
		},
		"tenants": tenants,
	})
}

// ResetAdminHandler resets or seeds the platform administrator credentials
func (a *API) ResetAdminHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Malformed JSON"}}`, http.StatusBadRequest)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || len(req.Password) < 8 {
		http.Error(w, `{"error":{"code":"invalid_input","message":"Email required and password must be at least 8 characters"}}`, http.StatusBadRequest)
		return
	}

	pwdHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		http.Error(w, `{"error":{"code":"hashing_failed","message":"Password hashing failed"}}`, http.StatusInternalServerError)
		return
	}

	user, err := a.repo.ResetPlatformAdmin(r.Context(), req.Email, pwdHash)
	if err != nil {
		orgID := crypto.NewULID(crypto.PrefixOrg)
		_ = a.repo.CreateOrganization(r.Context(), &domain.Organization{
			ID:        orgID,
			Name:      "Default Organization",
			Slug:      "default-org",
			Status:    "active",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		})
		adminID := crypto.NewULID(crypto.PrefixUser)
		user = &domain.User{
			ID:              adminID,
			OrganizationID:  orgID,
			Email:           req.Email,
			PasswordHash:    pwdHash,
			Status:          "active",
			IsPlatformAdmin: true,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}
		_ = a.repo.CreateUser(r.Context(), user)

		tenantID := crypto.NewULID(crypto.PrefixTenant)
		_ = a.repo.CreateTenant(r.Context(), &domain.Tenant{
			ID:             tenantID,
			OrganizationID: orgID,
			Name:           "Production",
			Slug:           "production",
			Environment:    "live",
			Status:         "active",
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		})
		_ = a.repo.AddTenantMembership(r.Context(), &domain.TenantMembership{
			ID:        crypto.NewULID(crypto.PrefixMembership),
			TenantID:  tenantID,
			UserID:    adminID,
			RoleID:    "role_tenant_owner",
			CreatedAt: time.Now().UTC(),
		})
	}

	_ = a.repo.SetSetting(r.Context(), "setup_completed", "true")

	_ = a.audit.Log(r.Context(), &domain.AuditLog{
		ActorUserID:  &user.ID,
		Action:       "auth.admin_reset",
		ResourceType: "user",
		ResourceID:   user.ID,
	})

	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Platform admin credentials reset successfully",
		"email":   user.Email,
	})
}

// MeHandler returns currently authenticated user details
func (a *API) MeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user, ok := r.Context().Value(middleware.UserKey).(*domain.User)
	if !ok || user == nil {
		http.Error(w, `{"error":{"code":"unauthorized","message":"Not authenticated"}}`, http.StatusUnauthorized)
		return
	}

	tenants, _ := a.repo.ListTenantsForUser(r.Context(), user.ID)
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"user": map[string]any{
			"id":                user.ID,
			"email":             user.Email,
			"organization_id":   user.OrganizationID,
			"is_platform_admin": user.IsPlatformAdmin,
		},
		"active_tenant_id": tenantID,
		"available_tenants": tenants,
	})
}

// ListTenantsHandler returns tenants accessible to the caller
func (a *API) ListTenantsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user, ok := r.Context().Value(middleware.UserKey).(*domain.User)
	if !ok || user == nil {
		if apiKey, ok := r.Context().Value(middleware.APIKeyKey).(*domain.APIKey); ok && apiKey != nil {
			tenant, err := a.repo.GetTenant(r.Context(), apiKey.TenantID)
			if err != nil {
				http.Error(w, `{"error":{"code":"tenant_not_found","message":"Tenant not found"}}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []*domain.Tenant{tenant},
			})
			return
		}
		http.Error(w, `{"error":{"code":"unauthorized","message":"Authentication required"}}`, http.StatusUnauthorized)
		return
	}

	tenants, err := a.repo.ListTenantsForUser(r.Context(), user.ID)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": tenants,
	})
}

type CreateTenantRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CreateTenantHandler creates a new tenant within the user's organization
func (a *API) CreateTenantHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user, ok := r.Context().Value(middleware.UserKey).(*domain.User)
	if !ok || user == nil {
		http.Error(w, `{"error":{"code":"unauthorized","message":"User session required to create tenants"}}`, http.StatusUnauthorized)
		return
	}

	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Malformed JSON"}}`, http.StatusBadRequest)
		return
	}

	tenantID := crypto.NewULID(crypto.PrefixTenant)
	tenant := &domain.Tenant{
		ID:             tenantID,
		OrganizationID: user.OrganizationID,
		Name:           req.Name,
		Slug:           req.Slug,
		Environment:    "live",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := a.repo.CreateTenant(r.Context(), tenant); err != nil {
		http.Error(w, `{"error":{"code":"tenant_create_failed","message":"`+err.Error()+`"}}`, http.StatusConflict)
		return
	}

	// Generate & wrap DEK for the new tenant
	dek, _ := a.encryption.GenerateDEK()
	encDEK, _ := a.encryption.EncryptDEK(dek)
	_ = a.repo.SaveTenantKey(r.Context(), &domain.TenantKey{
		ID:           crypto.NewULID(crypto.PrefixAPIKey),
		TenantID:     tenantID,
		KeyVersion:   1,
		EncryptedDEK: encDEK,
		KEKVersion:   "v1",
		CreatedAt:    time.Now().UTC(),
	})

	// Add user membership
	_ = a.repo.AddTenantMembership(r.Context(), &domain.TenantMembership{
		ID:        crypto.NewULID(crypto.PrefixMembership),
		TenantID:  tenantID,
		UserID:    user.ID,
		RoleID:    "role_tenant_owner",
		CreatedAt: time.Now().UTC(),
	})

	// Create default message streams (§8.3, §10.6)
	a.createDefaultStreams(r.Context(), tenantID)

	// Audit log
	_ = a.audit.Log(r.Context(), &domain.AuditLog{
		TenantID:     &tenantID,
		OrgID:        &user.OrganizationID,
		ActorUserID:  &user.ID,
		Action:       "tenant.created",
		ResourceType: "tenant",
		ResourceID:   tenantID,
		AfterState:   `{"name":"` + req.Name + `","slug":"` + req.Slug + `"}`,
	})

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tenant)
}

func (a *API) createDefaultStreams(ctx context.Context, tenantID string) {
	_ = a.repo.CreateStream(ctx, &domain.MessageStream{
		ID:        crypto.NewULID(crypto.PrefixStream),
		TenantID:  tenantID,
		Name:      "transactional",
		Type:      "transactional",
		IsDefault: true,
		CreatedAt: time.Now().UTC(),
	})
	_ = a.repo.CreateStream(ctx, &domain.MessageStream{
		ID:        crypto.NewULID(crypto.PrefixStream),
		TenantID:  tenantID,
		Name:      "broadcast",
		Type:      "broadcast",
		IsDefault: false,
		CreatedAt: time.Now().UTC(),
	})
}

// ListAuditLogsHandler retrieves audit logs scoped by active tenant or platform
func (a *API) ListAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var tenantIDPtr *string
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID != "" {
		tenantIDPtr = &tenantID
	}

	logs, err := a.repo.ListAuditLogs(r.Context(), tenantIDPtr, 50)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": logs,
	})
}
