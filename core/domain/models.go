package domain

import (
	"time"
)

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Tenant struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Environment    string    `json:"environment"` // "live" or "test"
	Status         string    `json:"status"`      // "active" or "suspended"
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TenantKey struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	KeyVersion   int       `json:"key_version"`
	EncryptedDEK string    `json:"-"` // never return encrypted DEK in json
	KEKVersion   string    `json:"kek_version"`
	CreatedAt    time.Time `json:"created_at"`
}

type User struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organization_id"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"-"`
	Status          string    `json:"status"`
	IsPlatformAdmin bool      `json:"is_platform_admin"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Role struct {
	ID             string    `json:"id"`
	OrganizationID *string   `json:"organization_id,omitempty"`
	Name           string    `json:"name"`
	Scope          string    `json:"scope"` // platform, organization, tenant
	Description    string    `json:"description"`
	IsSystem       bool      `json:"is_system"`
	CreatedAt      time.Time `json:"created_at"`
}

type Permission struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type TenantMembership struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	RoleID    string    `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditLog struct {
	ID                 string    `json:"id"`
	TenantID           *string   `json:"tenant_id,omitempty"`
	OrgID              *string   `json:"org_id,omitempty"`
	ActorUserID        *string   `json:"actor_user_id,omitempty"`
	ActorAPIKeyID      *string   `json:"actor_api_key_id,omitempty"`
	ImpersonatorUserID *string   `json:"impersonator_user_id,omitempty"`
	Action             string    `json:"action"`
	ResourceType       string    `json:"resource_type"`
	ResourceID         string    `json:"resource_id"`
	BeforeState        string    `json:"before_state,omitempty"`
	AfterState         string    `json:"after_state,omitempty"`
	Reason             *string   `json:"reason,omitempty"`
	IPAddress          *string   `json:"ip_address,omitempty"`
	UserAgent          *string   `json:"user_agent,omitempty"`
	OccurredAt         time.Time `json:"occurred_at"`
}

type SystemSetting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Domain struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Name           string     `json:"name"`
	Type           string     `json:"type"` // "public" or "local"
	Status         string     `json:"status"` // "pending", "verifying", "verified", "degraded", "suspended", "disabled"
	OwnershipToken string     `json:"ownership_token"`
	DegradedAt     *time.Time `json:"degraded_at,omitempty"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	LastCheckedAt  *time.Time `json:"last_checked_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type DomainDelegation struct {
	ID                  string    `json:"id"`
	ParentDomainID      string    `json:"parent_domain_id"`
	DelegatedToTenantID string    `json:"delegated_to_tenant_id"`
	CreatedAt           time.Time `json:"created_at"`
}

type DKIMKey struct {
	ID                  string     `json:"id"`
	DomainID            string     `json:"domain_id"`
	TenantID            string     `json:"tenant_id"`
	Selector            string     `json:"selector"`
	Algorithm           string     `json:"algorithm"` // rsa2048
	EncryptedPrivateKey string     `json:"-"` // sensitive
	PublicKey           string     `json:"public_key"`
	Status              string     `json:"status"` // pending, active, retiring, retired
	CreatedAt           time.Time  `json:"created_at"`
	ActivatedAt         *time.Time `json:"activated_at,omitempty"`
	RetiredAt           *time.Time `json:"retired_at,omitempty"`
}

type DNSRecord struct {
	ID            string     `json:"id"`
	DomainID      string     `json:"domain_id"`
	RecordType    string     `json:"record_type"` // TXT, CNAME, MX
	Name          string     `json:"name"`
	ExpectedValue string     `json:"expected_value"`
	ObservedValue *string    `json:"observed_value,omitempty"`
	Status        string     `json:"status"` // pending, pass, warning, fail
	ErrorMessage  *string    `json:"error_message,omitempty"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type MessageStream struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // "transactional" or "broadcast"
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

