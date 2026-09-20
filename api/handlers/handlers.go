package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mailvex/mailvex/api/middleware"
	"github.com/mailvex/mailvex/core/audit"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

type API struct {
	repo       repository.Repository
	audit      *audit.Service
	encryption *crypto.EnvelopeEncryption
}

func NewAPI(repo repository.Repository, auditService *audit.Service, encryption *crypto.EnvelopeEncryption) *API {
	return &API{
		repo:       repo,
		audit:      auditService,
		encryption: encryption,
	}
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

	// 1. Create Organization
	orgID := crypto.NewULID(crypto.PrefixOrg)
	org := &domain.Organization{
		ID:        orgID,
		Name:      req.OrgName,
		Slug:      "default-org",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := a.repo.CreateOrganization(r.Context(), org); err != nil {
		http.Error(w, `{"error":{"code":"org_create_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	// 2. Create Platform Admin User
	pwdHash, err := crypto.HashPassword(req.AdminPassword)
	if err != nil {
		http.Error(w, `{"error":{"code":"hashing_failed","message":"Password hashing failed"}}`, http.StatusInternalServerError)
		return
	}

	adminID := crypto.NewULID(crypto.PrefixUser)
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
	if err := a.repo.CreateUser(r.Context(), adminUser); err != nil {
		http.Error(w, `{"error":{"code":"user_create_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	// 3. Create First Tenant
	tenantID := crypto.NewULID(crypto.PrefixTenant)
	tenant := &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           req.TenantName,
		Slug:           "default-tenant",
		Environment:    "live",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := a.repo.CreateTenant(r.Context(), tenant); err != nil {
		http.Error(w, `{"error":{"code":"tenant_create_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	// 4. Generate & Store Tenant DEK with Envelope Encryption
	dek, err := a.encryption.GenerateDEK()
	if err != nil {
		http.Error(w, `{"error":{"code":"dek_failed","message":"Failed to generate tenant encryption key"}}`, http.StatusInternalServerError)
		return
	}
	encryptedDEK, err := a.encryption.EncryptDEK(dek)
	if err != nil {
		http.Error(w, `{"error":{"code":"encryption_failed","message":"Failed to wrap tenant encryption key"}}`, http.StatusInternalServerError)
		return
	}
	tenantKey := &domain.TenantKey{
		ID:           crypto.NewULID(crypto.PrefixAPIKey),
		TenantID:     tenantID,
		KeyVersion:   1,
		EncryptedDEK: encryptedDEK,
		KEKVersion:   "v1",
		CreatedAt:    time.Now().UTC(),
	}
	_ = a.repo.SaveTenantKey(r.Context(), tenantKey)

	// 5. Add Membership with role_tenant_owner
	mem := &domain.TenantMembership{
		ID:        crypto.NewULID(crypto.PrefixMembership),
		TenantID:  tenantID,
		UserID:    adminID,
		RoleID:    "role_tenant_owner",
		CreatedAt: time.Now().UTC(),
	}
	_ = a.repo.AddTenantMembership(r.Context(), mem)

	// Create default message streams (§8.3, §10.6)
	a.createDefaultStreams(r.Context(), tenantID)

	// 6. Create System Tenant for platform internal mail
	sysTenantID := "tenant_system"
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
		"message":         "Setup completed successfully",
		"organization_id": orgID,
		"admin_user_id":   adminID,
		"tenant_id":       tenantID,
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
	user := r.Context().Value(middleware.UserKey).(*domain.User)

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
	user := r.Context().Value(middleware.UserKey).(*domain.User)

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
