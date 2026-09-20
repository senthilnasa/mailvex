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

type Message struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenant_id"`
	StreamID          string     `json:"stream_id"`
	DomainID          string     `json:"domain_id"`
	Source            string     `json:"source"` // api, smtp, system, inbound_reply
	APIKeyID          *string    `json:"api_key_id,omitempty"`
	SMTPCredentialID  *string    `json:"smtp_credential_id,omitempty"`
	TemplateVersionID *string    `json:"template_version_id,omitempty"`
	FromAddress       string     `json:"from_address"`
	FromName          string     `json:"from_name"`
	ReplyTo           *string    `json:"reply_to,omitempty"`
	Subject           string     `json:"subject"`
	MessageIDHeader   string     `json:"message_id_header"`
	Status            string     `json:"status"` // queued, sending, partially_delivered, delivered, failed, cancelled
	Priority          int        `json:"priority"`
	Environment       string     `json:"environment"` // live, test, development
	SizeBytes         int        `json:"size_bytes"`
	BodyRef           *string    `json:"body_ref,omitempty"`
	RawMIMERef        *string    `json:"raw_mime_ref,omitempty"`
	Tags              []string   `json:"tags,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	ScheduledAt       *time.Time `json:"scheduled_at,omitempty"`
	QueuedAt          time.Time  `json:"queued_at"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type MessageRecipient struct {
	ID               string     `json:"id"`
	MessageID        string     `json:"message_id"`
	TenantID         string     `json:"tenant_id"`
	Address          string     `json:"address"`
	AddressDomain    string     `json:"address_domain"`
	MXGroup          string     `json:"mx_group"`
	Type             string     `json:"type"` // to, cc, bcc
	Status           string     `json:"status"` // queued, sending, delivered, deferred, bounced, suppressed, failed
	Attempts         int        `json:"attempts"`
	NextAttemptAt    *time.Time `json:"next_attempt_at,omitempty"`
	LastSMTPCode     *string    `json:"last_smtp_code,omitempty"`
	LastResponse     *string    `json:"last_response,omitempty"`
	DeliveredAt      *time.Time `json:"delivered_at,omitempty"`
	BouncedAt        *time.Time `json:"bounced_at,omitempty"`
	SuppressedReason *string    `json:"suppressed_reason,omitempty"`
	VERPToken        *string    `json:"verp_token,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type DeliveryAttempt struct {
	ID                string    `json:"id"`
	RecipientID       string    `json:"recipient_id"`
	TenantID          string    `json:"tenant_id"`
	AttemptNo         int       `json:"attempt_no"`
	IPID              *string   `json:"ip_id,omitempty"`
	NodeID            *string   `json:"node_id,omitempty"`
	RemoteMX          *string   `json:"remote_mx,omitempty"`
	RemoteIP          *string   `json:"remote_ip,omitempty"`
	TLSVersion        *string   `json:"tls_version,omitempty"`
	TLSCipher         *string   `json:"tls_cipher,omitempty"`
	TLSPolicyApplied  *string   `json:"tls_policy_applied,omitempty"`
	SMTPCode          *string   `json:"smtp_code,omitempty"`
	EnhancedCode      *string   `json:"enhanced_code,omitempty"`
	SMTPResponse      *string   `json:"smtp_response,omitempty"`
	DurationMS        int       `json:"duration_ms"`
	StartedAt         time.Time `json:"started_at"`
	Outcome           string    `json:"outcome"` // delivered, deferred, bounced, failed
	CreatedAt         time.Time `json:"created_at"`
}

type MessageEvent struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	MessageID   string    `json:"message_id"`
	RecipientID *string   `json:"recipient_id,omitempty"`
	Type        string    `json:"type"`
	Data        string    `json:"data,omitempty"` // JSON payload
	OccurredAt  time.Time `json:"occurred_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type OutboxItem struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Stream      string     `json:"stream"`
	Payload     string     `json:"payload"` // JSON serialized task
	AvailableAt time.Time  `json:"available_at"`
	ClaimedBy   *string    `json:"claimed_by,omitempty"`
	ClaimedAt   *time.Time `json:"claimed_at,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type SMTPCredential struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Username       string     `json:"username"`
	PasswordHash   string     `json:"-"`
	AllowedDomains []string   `json:"allowed_domains,omitempty"`
	AllowedCIDRs   []string   `json:"allowed_cidrs,omitempty"`
	StreamID       *string    `json:"stream_id,omitempty"`
	RateLimit      int        `json:"rate_limit"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

type Device struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	Name             string     `json:"name"`
	Type             string     `json:"type"` // e.g. printer, camera, nas, home_assistant
	ExpectedIP       string     `json:"expected_ip"`
	SMTPCredentialID *string    `json:"smtp_credential_id,omitempty"`
	AllowedFrom      []string   `json:"allowed_from,omitempty"`
	LastSeenAt       *time.Time `json:"last_seen_at,omitempty"`
	MessageCount     int64      `json:"message_count"`
	LastError        *string    `json:"last_error,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type TrustedNetwork struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	CIDR        string    `json:"cidr"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type APIKey struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Name           string     `json:"name"`
	Prefix         string     `json:"prefix"`
	KeyHash        string     `json:"-"`
	LastFour       string     `json:"last_four"`
	Environment    string     `json:"environment"` // live or test
	Scopes         []string   `json:"scopes"`
	AllowedDomains []string   `json:"allowed_domains,omitempty"`
	AllowedCIDRs   []string   `json:"allowed_cidrs,omitempty"`
	StreamID       *string    `json:"stream_id,omitempty"`
	RateLimit      int        `json:"rate_limit"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Template struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TemplateVersion struct {
	ID          string    `json:"id"`
	TemplateID  string    `json:"template_id"`
	TenantID    string    `json:"tenant_id"`
	Version     int       `json:"version"`
	Subject     string    `json:"subject"`
	HTML        string    `json:"html,omitempty"`
	Text        string    `json:"text,omitempty"`
	Variables   []string  `json:"variables,omitempty"`
	IsPublished bool      `json:"is_published"`
	CreatedAt   time.Time `json:"created_at"`
}

type IdempotencyRecord struct {
	ID              string            `json:"id"` // tenant_id:api_key_id:idempotency_key
	TenantID        string            `json:"tenant_id"`
	RequestHash     string            `json:"request_hash"`
	Status          string            `json:"status"` // in_progress, completed
	StatusCode      int               `json:"status_code"`
	ResponseBody    []byte            `json:"response_body"`
	ResponseHeaders map[string]string `json:"response_headers"`
	CreatedAt       time.Time         `json:"created_at"`
	ExpiresAt       time.Time         `json:"expires_at"`
}

type MessageFilter struct {
	Limit     int        `json:"limit"`
	Cursor    string     `json:"cursor"` // opaque base64
	Status    string     `json:"status"`
	Recipient string     `json:"recipient"`
	From      string     `json:"from"`
	Stream    string     `json:"stream"`
	Tag       string     `json:"tag"`
	Since     *time.Time `json:"since"`
	Until     *time.Time `json:"until"`
}

type Suppression struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	StreamID      *string    `json:"stream_id,omitempty"`
	Address       string     `json:"address"`
	AddressDomain string     `json:"address_domain"`
	Type          string     `json:"type"` // hard_bounce, complaint, unsubscribe, manual, domain_block, role_account
	Reason        *string    `json:"reason,omitempty"`
	Source        string     `json:"source"` // smtp, dsn, fbl, one_click, api, import, admin
	SourceRef     *string    `json:"source_ref,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedBy     *string    `json:"created_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type IPPool struct {
	ID        string    `json:"id"`
	TenantID  *string   `json:"tenant_id,omitempty"`
	Name      string    `json:"name"`
	Strategy  string    `json:"strategy"` // fixed, round_robin, least_used, weighted
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

type IPAddress struct {
	ID             string    `json:"id"`
	PoolID         string    `json:"pool_id"`
	IP             string    `json:"ip"`
	Subnet         *string   `json:"subnet,omitempty"`
	EHLOHostname   string    `json:"ehlo_hostname"`
	PTRRecord      *string   `json:"ptr_record,omitempty"`
	Status         string    `json:"status"` // active, warming, quarantine, disabled
	WarmupDay      int       `json:"warmup_day"`
	MaxDailyVolume int       `json:"max_daily_volume"`
	SentToday      int       `json:"sent_today"`
	NodeID         string    `json:"node_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type Relay struct {
	ID                 string    `json:"id"`
	TenantID           *string   `json:"tenant_id,omitempty"`
	Name               string    `json:"name"`
	Host               string    `json:"host"`
	Port               int       `json:"port"`
	TLSMode            string    `json:"tls_mode"` // starttls, tls, none
	AuthUser           *string   `json:"auth_user,omitempty"`
	AuthPassEncrypted  *string   `json:"auth_pass_encrypted,omitempty"`
	FromRewritePolicy  string    `json:"from_rewrite_policy"` // preserve, replace
	FromRewriteAddress *string   `json:"from_rewrite_address,omitempty"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
}

type BounceRule struct {
	ID             string    `json:"id"`
	Pattern        string    `json:"pattern"`
	EnhancedCode   *string   `json:"enhanced_code,omitempty"`
	Provider       *string   `json:"provider,omitempty"`
	Classification string    `json:"classification"`
	Action         string    `json:"action"`
	CreatedAt      time.Time `json:"created_at"`
}

type ProviderMXRule struct {
	ID                    string    `json:"id"`
	Pattern               string    `json:"pattern"`
	Provider              string    `json:"provider"`
	MaxConnections        int       `json:"max_connections"`
	MessagesPerConnection int       `json:"messages_per_connection"`
	RateLimitPerMin       int       `json:"rate_limit_per_min"`
	CreatedAt             time.Time `json:"created_at"`
}

type DeadLetterMessage struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	MessageID        string    `json:"message_id"`
	RecipientID      string    `json:"recipient_id"`
	Destination      string    `json:"destination"`
	Reason           string    `json:"reason"`
	LastSMTPCode     *string   `json:"last_smtp_code,omitempty"`
	LastEnhancedCode *string   `json:"last_enhanced_code,omitempty"`
	LastResponse     *string   `json:"last_response,omitempty"`
	Attempts         int       `json:"attempts"`
	SourceIP         *string   `json:"source_ip,omitempty"`
	NodeID           *string   `json:"node_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}



