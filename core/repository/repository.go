package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
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

	// Domains & DNS (Phase 2)
	CreateDomain(ctx context.Context, dom *domain.Domain) error
	GetDomain(ctx context.Context, tenantID, domainID string) (*domain.Domain, error)
	GetDomainByName(ctx context.Context, name string) (*domain.Domain, error)
	ListDomains(ctx context.Context, tenantID string) ([]*domain.Domain, error)
	UpdateDomain(ctx context.Context, dom *domain.Domain) error
	DeleteDomain(ctx context.Context, tenantID, domainID string) error

	// DKIM Keys (Phase 2)
	SaveDKIMKey(ctx context.Context, key *domain.DKIMKey) error
	GetActiveDKIMKey(ctx context.Context, domainID string) (*domain.DKIMKey, error)
	ListDKIMKeys(ctx context.Context, domainID string) ([]*domain.DKIMKey, error)
	UpdateDKIMKeyStatus(ctx context.Context, keyID, status string) error

	// DNS Records (Phase 2)
	SaveDNSRecords(ctx context.Context, domainID string, records []*domain.DNSRecord) error
	GetDNSRecords(ctx context.Context, domainID string) ([]*domain.DNSRecord, error)

	// Message Streams (Phase 2)
	CreateStream(ctx context.Context, stream *domain.MessageStream) error
	ListStreams(ctx context.Context, tenantID string) ([]*domain.MessageStream, error)
	GetStream(ctx context.Context, tenantID, streamID string) (*domain.MessageStream, error)

	// Delegations (Phase 2)
	AddDomainDelegation(ctx context.Context, delegation *domain.DomainDelegation) error
	IsDomainDelegated(ctx context.Context, parentDomainID, targetTenantID string) (bool, error)

	// Outbox & Queue (Phase 3)
	SaveOutboxItem(ctx context.Context, item *domain.OutboxItem) error
	ListPendingOutbox(ctx context.Context, limit int) ([]*domain.OutboxItem, error)
	MarkOutboxPublished(ctx context.Context, id string) error

	// Messages & Recipients (Phase 3)
	SaveMessage(ctx context.Context, msg *domain.Message, recipients []*domain.MessageRecipient, events []*domain.MessageEvent, outbox *domain.OutboxItem) error
	GetMessage(ctx context.Context, tenantID, messageID string) (*domain.Message, []*domain.MessageRecipient, []*domain.MessageEvent, error)
	ListMessages(ctx context.Context, tenantID string, limit int) ([]*domain.Message, error)
	ListDevelopmentMessages(ctx context.Context, tenantID string) ([]*domain.Message, error)

	// SMTP Credentials (Phase 3)
	CreateSMTPCredential(ctx context.Context, cred *domain.SMTPCredential) error
	GetSMTPCredentialByUsername(ctx context.Context, username string) (*domain.SMTPCredential, error)
	ListSMTPCredentials(ctx context.Context, tenantID string) ([]*domain.SMTPCredential, error)

	// LAN Devices (Phase 3)
	CreateDevice(ctx context.Context, dev *domain.Device) error
	GetDeviceByIP(ctx context.Context, tenantID, ip string) (*domain.Device, error)
	ListDevices(ctx context.Context, tenantID string) ([]*domain.Device, error)

	// Trusted Networks (Phase 3)
	AddTrustedNetwork(ctx context.Context, net *domain.TrustedNetwork) error
	ListTrustedNetworks(ctx context.Context, tenantID string) ([]*domain.TrustedNetwork, error)
	IsIPInTrustedNetwork(ctx context.Context, tenantID, ip string) (bool, error)

	// Phase 4: API Keys
	CreateAPIKey(ctx context.Context, key *domain.APIKey) error
	GetAPIKeyByHash(ctx context.Context, hash string) (*domain.APIKey, error)
	ListAPIKeys(ctx context.Context, tenantID string) ([]*domain.APIKey, error)
	RevokeAPIKey(ctx context.Context, tenantID, id string) error

	// Phase 4: Templates
	CreateTemplate(ctx context.Context, t *domain.Template, v *domain.TemplateVersion) error
	GetTemplate(ctx context.Context, tenantID, id string) (*domain.Template, error)
	GetTemplateVersion(ctx context.Context, tenantID, templateID string, version int) (*domain.TemplateVersion, error)
	GetPublishedTemplateVersion(ctx context.Context, tenantID, templateID string) (*domain.TemplateVersion, error)
	ListTemplates(ctx context.Context, tenantID string) ([]*domain.Template, error)

	// Phase 4: Idempotency
	GetIdempotencyRecord(ctx context.Context, tenantID, key string) (*domain.IdempotencyRecord, error)
	SetIdempotencyRecord(ctx context.Context, rec *domain.IdempotencyRecord) error

	// Phase 4: Messages read, details & pagination
	GetMessageByID(ctx context.Context, tenantID, messageID string) (*domain.Message, error)
	GetMessageRecipients(ctx context.Context, tenantID, messageID string) ([]*domain.MessageRecipient, error)
	GetMessageEvents(ctx context.Context, tenantID, messageID string) ([]*domain.MessageEvent, error)
	ListMessagesPaginated(ctx context.Context, tenantID string, filter domain.MessageFilter) ([]*domain.Message, string, bool, error)
	UpdateMessageStatus(ctx context.Context, tenantID, messageID, status string) error
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

	// Phase 2 collections
	domains     map[string]*domain.Domain // id -> domain
	delegations []*domain.DomainDelegation
	dkimKeys    map[string][]*domain.DKIMKey // domainID -> keys
	dnsRecords  map[string][]*domain.DNSRecord // domainID -> records
	streams     map[string][]*domain.MessageStream // tenantID -> streams

	// Phase 3 collections
	outboxItems map[string]*domain.OutboxItem
	messages    map[string]*domain.Message
	recipients  map[string][]*domain.MessageRecipient // messageID -> recipients
	events      map[string][]*domain.MessageEvent     // messageID -> events
	smtpCreds   map[string]*domain.SMTPCredential     // username -> cred
	devices     map[string][]*domain.Device           // tenantID -> devices
	trustedNets map[string][]*domain.TrustedNetwork   // tenantID -> nets

	// Phase 4 collections
	apiKeys          map[string]*domain.APIKey
	apiKeyHashes     map[string]*domain.APIKey
	templates        map[string]*domain.Template
	templateVersions map[string][]*domain.TemplateVersion
	idempotency      map[string]*domain.IdempotencyRecord
}

func NewMemoryRepository() *MemoryRepository {
	repo := &MemoryRepository{
		organizations:    make(map[string]*domain.Organization),
		tenants:          make(map[string]*domain.Tenant),
		tenantKeys:       make(map[string][]*domain.TenantKey),
		users:            make(map[string]*domain.User),
		roles:            make(map[string]*domain.Role),
		permissions:      make(map[string]*domain.Permission),
		rolePerms:        make(map[string][]string),
		memberships:      make(map[string]*domain.TenantMembership),
		auditLogs:        make([]*domain.AuditLog, 0),
		settings:         make(map[string]string),
		domains:          make(map[string]*domain.Domain),
		delegations:      make([]*domain.DomainDelegation, 0),
		dkimKeys:         make(map[string][]*domain.DKIMKey),
		dnsRecords:       make(map[string][]*domain.DNSRecord),
		streams:          make(map[string][]*domain.MessageStream),
		outboxItems:      make(map[string]*domain.OutboxItem),
		messages:         make(map[string]*domain.Message),
		recipients:       make(map[string][]*domain.MessageRecipient),
		events:           make(map[string][]*domain.MessageEvent),
		smtpCreds:        make(map[string]*domain.SMTPCredential),
		devices:          make(map[string][]*domain.Device),
		trustedNets:      make(map[string][]*domain.TrustedNetwork),
		apiKeys:          make(map[string]*domain.APIKey),
		apiKeyHashes:     make(map[string]*domain.APIKey),
		templates:        make(map[string]*domain.Template),
		templateVersions: make(map[string][]*domain.TemplateVersion),
		idempotency:      make(map[string]*domain.IdempotencyRecord),
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
		"send.email", "messages.read", "messages.body.read", "messages.retry", "messages.export", "messages.cancel",
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
		"send.email", "messages.read", "messages.body.read", "messages.retry", "messages.export", "messages.cancel",
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

// Phase 2: Domain Management

func (m *MemoryRepository) CreateDomain(ctx context.Context, dom *domain.Domain) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify tenant exists
	if _, ok := m.tenants[dom.TenantID]; !ok {
		return fmt.Errorf("tenant %s does not exist", dom.TenantID)
	}

	// Global uniqueness check (§10.2):
	// A verified domain may be owned by only one tenant platform-wide
	for _, existing := range m.domains {
		if existing.Name == dom.Name {
			if existing.TenantID == dom.TenantID {
				return ErrAlreadyExists
			}
			if existing.Status == "verified" || existing.Status == "degraded" {
				// Check delegation
				delegated := false
				for _, del := range m.delegations {
					if del.ParentDomainID == existing.ID && del.DelegatedToTenantID == dom.TenantID {
						delegated = true
						break
					}
				}
				if !delegated {
					return errors.New("domain_claimed_by_another_tenant")
				}
			}
		}
	}

	// Local domain handling (§28.1): Skips public DNS verification, ownership implied
	if dom.Type == "local" {
		dom.Status = "verified"
		now := time.Now().UTC()
		dom.VerifiedAt = &now
	}

	m.domains[dom.ID] = dom
	return nil
}

func (m *MemoryRepository) GetDomain(ctx context.Context, tenantID, domainID string) (*domain.Domain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dom, exists := m.domains[domainID]
	if !exists {
		return nil, ErrNotFound
	}

	// Strict tenant isolation guard
	if dom.TenantID != tenantID {
		return nil, ErrCrossTenant
	}

	return dom, nil
}

func (m *MemoryRepository) GetDomainByName(ctx context.Context, name string) (*domain.Domain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, dom := range m.domains {
		if dom.Name == name {
			return dom, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryRepository) ListDomains(ctx context.Context, tenantID string) ([]*domain.Domain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.Domain
	for _, dom := range m.domains {
		if dom.TenantID == tenantID {
			result = append(result, dom)
		}
	}
	return result, nil
}

func (m *MemoryRepository) UpdateDomain(ctx context.Context, dom *domain.Domain) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.domains[dom.ID]
	if !exists {
		return ErrNotFound
	}
	if existing.TenantID != dom.TenantID {
		return ErrCrossTenant
	}

	dom.UpdatedAt = time.Now().UTC()
	m.domains[dom.ID] = dom
	return nil
}

func (m *MemoryRepository) DeleteDomain(ctx context.Context, tenantID, domainID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dom, exists := m.domains[domainID]
	if !exists {
		return ErrNotFound
	}
	if dom.TenantID != tenantID {
		return ErrCrossTenant
	}

	delete(m.domains, domainID)
	delete(m.dkimKeys, domainID)
	delete(m.dnsRecords, domainID)
	return nil
}

// Phase 2: DKIM Keys

func (m *MemoryRepository) SaveDKIMKey(ctx context.Context, key *domain.DKIMKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dkimKeys[key.DomainID] = append(m.dkimKeys[key.DomainID], key)
	return nil
}

func (m *MemoryRepository) GetActiveDKIMKey(ctx context.Context, domainID string) (*domain.DKIMKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys, exists := m.dkimKeys[domainID]
	if !exists || len(keys) == 0 {
		return nil, ErrNotFound
	}

	for i := len(keys) - 1; i >= 0; i-- {
		if keys[i].Status == "active" {
			return keys[i], nil
		}
	}

	// If none is active yet, return the latest pending
	return keys[len(keys)-1], nil
}

func (m *MemoryRepository) ListDKIMKeys(ctx context.Context, domainID string) ([]*domain.DKIMKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys, exists := m.dkimKeys[domainID]
	if !exists {
		return []*domain.DKIMKey{}, nil
	}
	return keys, nil
}

func (m *MemoryRepository) UpdateDKIMKeyStatus(ctx context.Context, keyID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, keys := range m.dkimKeys {
		for _, k := range keys {
			if k.ID == keyID {
				k.Status = status
				now := time.Now().UTC()
				if status == "active" {
					k.ActivatedAt = &now
				} else if status == "retired" {
					k.RetiredAt = &now
				}
				return nil
			}
		}
	}
	return ErrNotFound
}

// Phase 2: DNS Records

func (m *MemoryRepository) SaveDNSRecords(ctx context.Context, domainID string, records []*domain.DNSRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dnsRecords[domainID] = records
	return nil
}

func (m *MemoryRepository) GetDNSRecords(ctx context.Context, domainID string) ([]*domain.DNSRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	records, exists := m.dnsRecords[domainID]
	if !exists {
		return []*domain.DNSRecord{}, nil
	}
	return records, nil
}

// Phase 2: Message Streams

func (m *MemoryRepository) CreateStream(ctx context.Context, stream *domain.MessageStream) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingList := m.streams[stream.TenantID]
	for _, s := range existingList {
		if s.Name == stream.Name {
			return ErrAlreadyExists
		}
	}

	m.streams[stream.TenantID] = append(m.streams[stream.TenantID], stream)
	return nil
}

func (m *MemoryRepository) ListStreams(ctx context.Context, tenantID string) ([]*domain.MessageStream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	streams, exists := m.streams[tenantID]
	if !exists {
		return []*domain.MessageStream{}, nil
	}
	return streams, nil
}

func (m *MemoryRepository) GetStream(ctx context.Context, tenantID, streamID string) (*domain.MessageStream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	streams, exists := m.streams[tenantID]
	if !exists {
		return nil, ErrNotFound
	}

	for _, s := range streams {
		if s.ID == streamID {
			return s, nil
		}
	}
	return nil, ErrNotFound
}

// Phase 2: Delegations

func (m *MemoryRepository) AddDomainDelegation(ctx context.Context, delegation *domain.DomainDelegation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.delegations = append(m.delegations, delegation)
	return nil
}

func (m *MemoryRepository) IsDomainDelegated(ctx context.Context, parentDomainID, targetTenantID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, del := range m.delegations {
		if del.ParentDomainID == parentDomainID && del.DelegatedToTenantID == targetTenantID {
			return true, nil
		}
	}
	return false, nil
}

// Phase 3: Outbox & Queue

func (m *MemoryRepository) SaveOutboxItem(ctx context.Context, item *domain.OutboxItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.outboxItems[item.ID] = item
	return nil
}

func (m *MemoryRepository) ListPendingOutbox(ctx context.Context, limit int) ([]*domain.OutboxItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now().UTC()
	var pending []*domain.OutboxItem
	for _, item := range m.outboxItems {
		if item.PublishedAt == nil && !item.AvailableAt.After(now) {
			pending = append(pending, item)
			if limit > 0 && len(pending) >= limit {
				break
			}
		}
	}
	return pending, nil
}

func (m *MemoryRepository) MarkOutboxPublished(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, exists := m.outboxItems[id]
	if !exists {
		return ErrNotFound
	}
	now := time.Now().UTC()
	item.PublishedAt = &now
	return nil
}

// Phase 3: Messages & Recipients

func (m *MemoryRepository) SaveMessage(ctx context.Context, msg *domain.Message, recipients []*domain.MessageRecipient, events []*domain.MessageEvent, outbox *domain.OutboxItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages[msg.ID] = msg
	m.recipients[msg.ID] = recipients
	m.events[msg.ID] = events
	if outbox != nil {
		m.outboxItems[outbox.ID] = outbox
	}
	return nil
}

func (m *MemoryRepository) GetMessage(ctx context.Context, tenantID, messageID string) (*domain.Message, []*domain.MessageRecipient, []*domain.MessageEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	msg, exists := m.messages[messageID]
	if !exists {
		return nil, nil, nil, ErrNotFound
	}
	if msg.TenantID != tenantID {
		return nil, nil, nil, ErrCrossTenant
	}

	rcpts := m.recipients[messageID]
	evts := m.events[messageID]
	return msg, rcpts, evts, nil
}

func (m *MemoryRepository) ListMessages(ctx context.Context, tenantID string, limit int) ([]*domain.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.Message
	for _, msg := range m.messages {
		if msg.TenantID == tenantID {
			result = append(result, msg)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *MemoryRepository) ListDevelopmentMessages(ctx context.Context, tenantID string) ([]*domain.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.Message
	for _, msg := range m.messages {
		if msg.TenantID == tenantID && (msg.Environment == "development" || msg.Environment == "test") {
			result = append(result, msg)
		}
	}
	return result, nil
}

// Phase 3: SMTP Credentials

func (m *MemoryRepository) CreateSMTPCredential(ctx context.Context, cred *domain.SMTPCredential) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.smtpCreds[cred.Username]; exists {
		return ErrAlreadyExists
	}
	m.smtpCreds[cred.Username] = cred
	return nil
}

func (m *MemoryRepository) GetSMTPCredentialByUsername(ctx context.Context, username string) (*domain.SMTPCredential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cred, exists := m.smtpCreds[username]
	if !exists || cred.RevokedAt != nil {
		return nil, ErrNotFound
	}
	return cred, nil
}

func (m *MemoryRepository) ListSMTPCredentials(ctx context.Context, tenantID string) ([]*domain.SMTPCredential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.SMTPCredential
	for _, cred := range m.smtpCreds {
		if cred.TenantID == tenantID && cred.RevokedAt == nil {
			result = append(result, cred)
		}
	}
	return result, nil
}

// Phase 3: LAN Devices

func (m *MemoryRepository) CreateDevice(ctx context.Context, dev *domain.Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.devices[dev.TenantID] = append(m.devices[dev.TenantID], dev)
	return nil
}

func (m *MemoryRepository) GetDeviceByIP(ctx context.Context, tenantID, ip string) (*domain.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devs := m.devices[tenantID]
	for _, d := range devs {
		if d.ExpectedIP == ip {
			return d, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryRepository) ListDevices(ctx context.Context, tenantID string) ([]*domain.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devs := m.devices[tenantID]
	if devs == nil {
		return []*domain.Device{}, nil
	}
	return devs, nil
}

// Phase 3: Trusted Networks

func (m *MemoryRepository) AddTrustedNetwork(ctx context.Context, net *domain.TrustedNetwork) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.trustedNets[net.TenantID] = append(m.trustedNets[net.TenantID], net)
	return nil
}

func (m *MemoryRepository) ListTrustedNetworks(ctx context.Context, tenantID string) ([]*domain.TrustedNetwork, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nets := m.trustedNets[tenantID]
	if nets == nil {
		return []*domain.TrustedNetwork{}, nil
	}
	return nets, nil
}

func (m *MemoryRepository) IsIPInTrustedNetwork(ctx context.Context, tenantID, clientIP string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false, nil
	}

	nets := m.trustedNets[tenantID]
	for _, n := range nets {
		_, ipNet, err := net.ParseCIDR(n.CIDR)
		if err == nil && ipNet.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}

// Phase 4: API Keys

func (m *MemoryRepository) CreateAPIKey(ctx context.Context, key *domain.APIKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.apiKeyHashes[key.KeyHash]; exists {
		return ErrAlreadyExists
	}

	m.apiKeys[key.ID] = key
	m.apiKeyHashes[key.KeyHash] = key
	return nil
}

func (m *MemoryRepository) GetAPIKeyByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, exists := m.apiKeyHashes[hash]
	if !exists {
		return nil, ErrNotFound
	}
	if key.RevokedAt != nil {
		return nil, errors.New("api key has been revoked")
	}
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, errors.New("api key has expired")
	}
	return key, nil
}

func (m *MemoryRepository) ListAPIKeys(ctx context.Context, tenantID string) ([]*domain.APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.APIKey
	for _, k := range m.apiKeys {
		if k.TenantID == tenantID && k.RevokedAt == nil {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *MemoryRepository) RevokeAPIKey(ctx context.Context, tenantID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, exists := m.apiKeys[id]
	if !exists {
		return ErrNotFound
	}
	if key.TenantID != tenantID {
		return ErrCrossTenant
	}

	now := time.Now()
	key.RevokedAt = &now
	return nil
}

// Phase 4: Templates

func (m *MemoryRepository) CreateTemplate(ctx context.Context, t *domain.Template, v *domain.TemplateVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.templates[t.ID] = t
	m.templateVersions[t.ID] = append(m.templateVersions[t.ID], v)
	return nil
}

func (m *MemoryRepository) GetTemplate(ctx context.Context, tenantID, id string) (*domain.Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tmpl, exists := m.templates[id]
	if !exists {
		return nil, ErrNotFound
	}
	if tmpl.TenantID != tenantID {
		return nil, ErrCrossTenant
	}
	return tmpl, nil
}

func (m *MemoryRepository) GetTemplateVersion(ctx context.Context, tenantID, templateID string, version int) (*domain.TemplateVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vers, exists := m.templateVersions[templateID]
	if !exists {
		return nil, ErrNotFound
	}
	for _, v := range vers {
		if v.TenantID != tenantID {
			return nil, ErrCrossTenant
		}
		if v.Version == version {
			return v, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryRepository) GetPublishedTemplateVersion(ctx context.Context, tenantID, templateID string) (*domain.TemplateVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vers, exists := m.templateVersions[templateID]
	if !exists {
		return nil, ErrNotFound
	}
	var highest *domain.TemplateVersion
	for _, v := range vers {
		if v.TenantID != tenantID {
			return nil, ErrCrossTenant
		}
		if v.IsPublished {
			if highest == nil || v.Version > highest.Version {
				highest = v
			}
		}
	}
	if highest == nil {
		return nil, ErrNotFound
	}
	return highest, nil
}

func (m *MemoryRepository) ListTemplates(ctx context.Context, tenantID string) ([]*domain.Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.Template
	for _, t := range m.templates {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	return result, nil
}

// Phase 4: Idempotency

func (m *MemoryRepository) GetIdempotencyRecord(ctx context.Context, tenantID, key string) (*domain.IdempotencyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	recID := fmt.Sprintf("%s:%s", tenantID, key)
	rec, exists := m.idempotency[recID]
	if !exists {
		return nil, ErrNotFound
	}
	if time.Now().After(rec.ExpiresAt) {
		return nil, ErrNotFound
	}
	return rec, nil
}

func (m *MemoryRepository) SetIdempotencyRecord(ctx context.Context, rec *domain.IdempotencyRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.idempotency[rec.ID] = rec
	return nil
}

// Phase 4: Messages read, details & pagination

func (m *MemoryRepository) GetMessageByID(ctx context.Context, tenantID, messageID string) (*domain.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	msg, exists := m.messages[messageID]
	if !exists {
		return nil, ErrNotFound
	}
	if msg.TenantID != tenantID {
		return nil, ErrCrossTenant
	}
	return msg, nil
}

func (m *MemoryRepository) GetMessageRecipients(ctx context.Context, tenantID, messageID string) ([]*domain.MessageRecipient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	msg, exists := m.messages[messageID]
	if !exists {
		return nil, ErrNotFound
	}
	if msg.TenantID != tenantID {
		return nil, ErrCrossTenant
	}

	rcpts := m.recipients[messageID]
	if rcpts == nil {
		return []*domain.MessageRecipient{}, nil
	}
	return rcpts, nil
}

func (m *MemoryRepository) GetMessageEvents(ctx context.Context, tenantID, messageID string) ([]*domain.MessageEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	msg, exists := m.messages[messageID]
	if !exists {
		return nil, ErrNotFound
	}
	if msg.TenantID != tenantID {
		return nil, ErrCrossTenant
	}

	evts := m.events[messageID]
	if evts == nil {
		return []*domain.MessageEvent{}, nil
	}
	return evts, nil
}

func (m *MemoryRepository) UpdateMessageStatus(ctx context.Context, tenantID, messageID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg, exists := m.messages[messageID]
	if !exists {
		return ErrNotFound
	}
	if msg.TenantID != tenantID {
		return ErrCrossTenant
	}

	msg.Status = status
	return nil
}

func (m *MemoryRepository) ListMessagesPaginated(ctx context.Context, tenantID string, filter domain.MessageFilter) ([]*domain.Message, string, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matched []*domain.Message
	for _, msg := range m.messages {
		if msg.TenantID != tenantID {
			continue
		}

		if filter.Status != "" && msg.Status != filter.Status {
			continue
		}
		if filter.From != "" && !strings.Contains(strings.ToLower(msg.FromAddress), strings.ToLower(filter.From)) {
			continue
		}
		if filter.Stream != "" && msg.StreamID != filter.Stream {
			continue
		}
		if filter.Since != nil && msg.CreatedAt.Before(*filter.Since) {
			continue
		}
		if filter.Until != nil && msg.CreatedAt.After(*filter.Until) {
			continue
		}
		if filter.Recipient != "" {
			hasRcpt := false
			for _, r := range m.recipients[msg.ID] {
				if strings.Contains(strings.ToLower(r.Address), strings.ToLower(filter.Recipient)) {
					hasRcpt = true
					break
				}
			}
			if !hasRcpt {
				continue
			}
		}

		matched = append(matched, msg)
	}

	// Sort descending by CreatedAt, then by ID
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].CreatedAt.Equal(matched[j].CreatedAt) {
			return matched[i].ID > matched[j].ID
		}
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})

	// Cursor decoding: base64(unixNano:id)
	startIndex := 0
	if filter.Cursor != "" {
		if decoded, err := base64.URLEncoding.DecodeString(filter.Cursor); err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				curNano, _ := strconv.ParseInt(parts[0], 10, 64)
				curID := parts[1]

				for idx, msg := range matched {
					if msg.CreatedAt.UnixNano() < curNano || (msg.CreatedAt.UnixNano() == curNano && msg.ID < curID) {
						startIndex = idx
						break
					}
					// If reached end without finding strictly older
					if idx == len(matched)-1 {
						startIndex = len(matched)
					}
				}
			}
		}
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	endIndex := startIndex + limit
	hasMore := false
	if endIndex < len(matched) {
		hasMore = true
	} else {
		endIndex = len(matched)
	}

	var paged []*domain.Message
	if startIndex < len(matched) {
		paged = matched[startIndex:endIndex]
	}

	var nextCursor string
	if hasMore && len(paged) > 0 {
		last := paged[len(paged)-1]
		raw := fmt.Sprintf("%d:%s", last.CreatedAt.UnixNano(), last.ID)
		nextCursor = base64.URLEncoding.EncodeToString([]byte(raw))
	}

	return paged, nextCursor, hasMore, nil
}

