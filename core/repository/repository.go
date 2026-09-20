package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mailvex/mailvex/core/domain"
)

var (
	ErrNotFound       = errors.New("resource not found")
	ErrAlreadyExists  = errors.New("resource already exists")
	ErrCrossTenant    = errors.New("cross-tenant access prohibited")
	ErrUnauthorized   = errors.New("unauthorized action")
	ErrInvalidInput   = errors.New("invalid input")
)

// Repository defines the full data access layer
type Repository interface {
	// Organization & Platform operations
	CreateOrganization(ctx context.Context, org *domain.Organization) error
	GetOrganization(ctx context.Context, id string) (*domain.Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*domain.Organization, error)
	ListOrganizations(ctx context.Context) ([]*domain.Organization, error)

	// User operations
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, id string) (*domain.User, error)

	// Tenant operations
	CreateTenant(ctx context.Context, tenant *domain.Tenant) error
	GetTenant(ctx context.Context, id string) (*domain.Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	ListTenantsByOrg(ctx context.Context, orgID string) ([]*domain.Tenant, error)
	ListTenantsForUser(ctx context.Context, userID string) ([]*domain.Tenant, error)

	// Tenant Keys (Envelope Encryption DEKs)
	SaveTenantKey(ctx context.Context, key *domain.TenantKey) error
	GetActiveTenantKey(ctx context.Context, tenantID string) (*domain.TenantKey, error)

	// RBAC & Memberships
	AddTenantMembership(ctx context.Context, membership *domain.TenantMembership) error
	GetTenantMembership(ctx context.Context, tenantID, userID string) (*domain.TenantMembership, error)
	GetRole(ctx context.Context, roleID string) (*domain.Role, error)
	GetRolePermissions(ctx context.Context, roleID string) ([]string, error)
	AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error

	// Audit Logs
	RecordAuditLog(ctx context.Context, log *domain.AuditLog) error
	ListAuditLogs(ctx context.Context, tenantID *string, limit int) ([]*domain.AuditLog, error)

	// System Settings
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
}

// MemoryRepository is an in-memory thread-safe implementation of Repository with strict tenant isolation
type MemoryRepository struct {
	mu            sync.RWMutex
	organizations map[string]*domain.Organization
	tenants       map[string]*domain.Tenant
	tenantKeys    map[string][]*domain.TenantKey
	users         map[string]*domain.User
	roles         map[string]*domain.Role
	permissions   map[string]*domain.Permission
	rolePerms     map[string][]string // roleID -> permissionIDs
	memberships   map[string]*domain.TenantMembership // key: tenantID:userID
	auditLogs     []*domain.AuditLog
	settings      map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	repo := &MemoryRepository{
		organizations: make(map[string]*domain.Organization),
		tenants:       make(map[string]*domain.Tenant),
		tenantKeys:    make(map[string][]*domain.TenantKey),
		users:         make(map[string]*domain.User),
		roles:         make(map[string]*domain.Role),
		permissions:   make(map[string]*domain.Permission),
		rolePerms:     make(map[string][]string),
		memberships:   make(map[string]*domain.TenantMembership),
		auditLogs:     make([]*domain.AuditLog, 0),
		settings:      make(map[string]string),
	}
	repo.seedDefaults()
	return repo
}

func (m *MemoryRepository) seedDefaults() {
	// Standard permissions
	perms := []string{
		"tenants.read", "tenants.write", "tenants.suspend",
		"org.read", "org.write", "org.sso.write",
		"users.read", "users.write", "users.invite",
		"roles.read", "roles.write",
		"domains.read", "domains.write", "domains.verify", "domains.dkim.rotate", "domains.delete",
		"streams.read", "streams.write",
		"api_keys.read", "api_keys.create", "api_keys.revoke",
		"smtp_credentials.read", "smtp_credentials.create", "smtp_credentials.revoke",
		"ip.read", "ip.write", "pools.read", "pools.write",
		"messages.read", "messages.body.read", "messages.retry", "messages.export", "messages.cancel",
		"suppressions.read", "suppressions.write", "suppressions.import",
		"templates.read", "templates.write", "templates.publish",
		"webhooks.read", "webhooks.write", "webhooks.replay", "webhooks.secret.rotate",
		"inbound.read", "inbound.write",
		"analytics.read", "reports.read",
		"audit.read",
		"settings.read", "settings.write",
		"retention.write", "privacy.erase", "privacy.export",
		"impersonate.read", "impersonate.write",
		"system.read", "system.write",
	}

	for _, p := range perms {
		m.permissions[p] = &domain.Permission{ID: p, Name: p, CreatedAt: time.Now()}
	}

	// Roles
	m.roles["role_platform_owner"] = &domain.Role{
		ID: "role_platform_owner", Name: "Platform Owner", Scope: "platform", IsSystem: true, CreatedAt: time.Now(),
	}
	m.rolePerms["role_platform_owner"] = perms // all permissions

	m.roles["role_tenant_owner"] = &domain.Role{
		ID: "role_tenant_owner", Name: "Tenant Owner", Scope: "tenant", IsSystem: true, CreatedAt: time.Now(),
	}
	tenantPerms := []string{
		"tenants.read", "tenants.write", "users.read", "users.write", "users.invite",
		"domains.read", "domains.write", "domains.verify", "domains.dkim.rotate",
		"streams.read", "streams.write", "api_keys.read", "api_keys.create", "api_keys.revoke",
		"smtp_credentials.read", "smtp_credentials.create", "smtp_credentials.revoke",
		"messages.read", "messages.body.read", "messages.retry", "messages.export",
		"suppressions.read", "suppressions.write", "suppressions.import",
		"templates.read", "templates.write", "templates.publish",
		"webhooks.read", "webhooks.write", "webhooks.replay",
		"analytics.read", "audit.read", "settings.read", "settings.write",
	}
	m.rolePerms["role_tenant_owner"] = tenantPerms
}

func (m *MemoryRepository) CreateOrganization(ctx context.Context, org *domain.Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.organizations {
		if existing.Slug == org.Slug {
			return ErrAlreadyExists
		}
	}
	m.organizations[org.ID] = org
	return nil
}

func (m *MemoryRepository) GetOrganization(ctx context.Context, id string) (*domain.Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	org, exists := m.organizations[id]
	if !exists {
		return nil, ErrNotFound
	}
	return org, nil
}

func (m *MemoryRepository) GetOrganizationBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, org := range m.organizations {
		if org.Slug == slug {
			return org, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryRepository) ListOrganizations(ctx context.Context) ([]*domain.Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*domain.Organization, 0, len(m.organizations))
	for _, org := range m.organizations {
		list = append(list, org)
	}
	return list, nil
}

func (m *MemoryRepository) CreateUser(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.users {
		if existing.Email == user.Email {
			return ErrAlreadyExists
		}
	}
	m.users[user.ID] = user
	return nil
}

func (m *MemoryRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists {
		return nil, ErrNotFound
	}
	return user, nil
}

func (m *MemoryRepository) CreateTenant(ctx context.Context, tenant *domain.Tenant) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.organizations[tenant.OrganizationID]; !exists {
		return fmt.Errorf("parent organization %s does not exist", tenant.OrganizationID)
	}

	for _, existing := range m.tenants {
		if existing.Slug == tenant.Slug {
			return ErrAlreadyExists
		}
	}
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MemoryRepository) GetTenant(ctx context.Context, id string) (*domain.Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, exists := m.tenants[id]
	if !exists {
		return nil, ErrNotFound
	}
	return tenant, nil
}

func (m *MemoryRepository) GetTenantBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tenant := range m.tenants {
		if tenant.Slug == slug {
			return tenant, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryRepository) ListTenantsByOrg(ctx context.Context, orgID string) ([]*domain.Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*domain.Tenant
	for _, tenant := range m.tenants {
		if tenant.OrganizationID == orgID {
			list = append(list, tenant)
		}
	}
	return list, nil
}

func (m *MemoryRepository) ListTenantsForUser(ctx context.Context, userID string) ([]*domain.Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil, ErrNotFound
	}

	if user.IsPlatformAdmin {
		list := make([]*domain.Tenant, 0, len(m.tenants))
		for _, t := range m.tenants {
			list = append(list, t)
		}
		return list, nil
	}

	var list []*domain.Tenant
	for _, mem := range m.memberships {
		if mem.UserID == userID {
			if tenant, ok := m.tenants[mem.TenantID]; ok {
				list = append(list, tenant)
			}
		}
	}
	return list, nil
}

func (m *MemoryRepository) SaveTenantKey(ctx context.Context, key *domain.TenantKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tenants[key.TenantID]; !exists {
		return ErrNotFound
	}
	m.tenantKeys[key.TenantID] = append(m.tenantKeys[key.TenantID], key)
	return nil
}

func (m *MemoryRepository) GetActiveTenantKey(ctx context.Context, tenantID string) (*domain.TenantKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys, exists := m.tenantKeys[tenantID]
	if !exists || len(keys) == 0 {
		return nil, ErrNotFound
	}
	return keys[len(keys)-1], nil
}

func (m *MemoryRepository) AddTenantMembership(ctx context.Context, membership *domain.TenantMembership) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tenants[membership.TenantID]; !ok {
		return fmt.Errorf("tenant %s does not exist", membership.TenantID)
	}
	if _, ok := m.users[membership.UserID]; !ok {
		return fmt.Errorf("user %s does not exist", membership.UserID)
	}
	if _, ok := m.roles[membership.RoleID]; !ok {
		return fmt.Errorf("role %s does not exist", membership.RoleID)
	}

	key := membership.TenantID + ":" + membership.UserID
	m.memberships[key] = membership
	return nil
}

func (m *MemoryRepository) GetTenantMembership(ctx context.Context, tenantID, userID string) (*domain.TenantMembership, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := tenantID + ":" + userID
	mem, exists := m.memberships[key]
	if !exists {
		return nil, ErrNotFound
	}
	return mem, nil
}

func (m *MemoryRepository) GetRole(ctx context.Context, roleID string) (*domain.Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, exists := m.roles[roleID]
	if !exists {
		return nil, ErrNotFound
	}
	return role, nil
}

func (m *MemoryRepository) GetRolePermissions(ctx context.Context, roleID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	perms, exists := m.rolePerms[roleID]
	if !exists {
		return []string{}, nil
	}
	return perms, nil
}

func (m *MemoryRepository) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.roles[roleID]; !ok {
		return fmt.Errorf("role %s does not exist", roleID)
	}
	if _, ok := m.permissions[permissionID]; !ok {
		return fmt.Errorf("permission %s does not exist", permissionID)
	}

	m.rolePerms[roleID] = append(m.rolePerms[roleID], permissionID)
	return nil
}

func (m *MemoryRepository) RecordAuditLog(ctx context.Context, log *domain.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.auditLogs = append(m.auditLogs, log)
	return nil
}

func (m *MemoryRepository) ListAuditLogs(ctx context.Context, tenantID *string, limit int) ([]*domain.AuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.AuditLog
	for i := len(m.auditLogs) - 1; i >= 0; i-- {
		log := m.auditLogs[i]
		if tenantID != nil {
			if log.TenantID == nil || *log.TenantID != *tenantID {
				continue
			}
		}
		result = append(result, log)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *MemoryRepository) GetSetting(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, exists := m.settings[key]
	if !exists {
		return "", ErrNotFound
	}
	return val, nil
}

func (m *MemoryRepository) SetSetting(ctx context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.settings[key] = value
	return nil
}
