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

type FBLReport struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	MessageID       *string    `json:"message_id,omitempty"`
	RecipientID     *string    `json:"recipient_id,omitempty"`
	FeedbackType    string     `json:"feedback_type"`
	UserAgent       *string    `json:"user_agent,omitempty"`
	ReportedDomain  *string    `json:"reported_domain,omitempty"`
	OriginalHeaders *string    `json:"original_headers,omitempty"`
	ArrivalDate     *time.Time `json:"arrival_date,omitempty"`
	RawMIMERef      *string    `json:"raw_mime_ref,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type UnmatchedBounce struct {
	ID         string    `json:"id"`
	Sender     *string   `json:"sender,omitempty"`
	Recipient  *string   `json:"recipient,omitempty"`
	Subject    *string   `json:"subject,omitempty"`
	RawMIMERef *string   `json:"raw_mime_ref,omitempty"`
	ParseError *string   `json:"parse_error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type AbuseReport struct {
	ID         string     `json:"id"`
	TenantID   *string    `json:"tenant_id,omitempty"`
	DomainID   *string    `json:"domain_id,omitempty"`
	MessageID  *string    `json:"message_id,omitempty"`
	Sender     string     `json:"sender"`
	Recipient  string     `json:"recipient"`
	Subject    *string    `json:"subject,omitempty"`
	Status     string     `json:"status"` // pending, resolved, escalated
	Notes      *string    `json:"notes,omitempty"`
	RawMIMERef *string    `json:"raw_mime_ref,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type IPReputationSnapshot struct {
	ID         string    `json:"id"`
	IPID       string    `json:"ip_id"`
	Source     string    `json:"source"` // snds, postmaster, dnsbl
	SampleDate string    `json:"sample_date"`
	Score      float64   `json:"score"`
	SampleData string    `json:"sample_data,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type PostmasterSnapshot struct {
	ID               string    `json:"id"`
	DomainID         string    `json:"domain_id"`
	SampleDate       string    `json:"sample_date"`
	SpamRate         float64   `json:"spam_rate"`
	DomainReputation string    `json:"domain_reputation"` // bad, low, medium, high
	IPReputation     string    `json:"ip_reputation"`
	DKIMSuccessRate  float64   `json:"dkim_success_rate"`
	SPFSuccessRate   float64   `json:"spf_success_rate"`
	DMARCSuccessRate float64   `json:"dmarc_success_rate"`
	CreatedAt        time.Time `json:"created_at"`
}

type AnalyticsRollup struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	TimeBucket         string    `json:"time_bucket"` // 1m, 1h, 1d, 1mo
	BucketStart        time.Time `json:"bucket_start"`
	StreamID           *string   `json:"stream_id,omitempty"`
	DomainID           *string   `json:"domain_id,omitempty"`
	Provider           string    `json:"provider"` // google, microsoft, yahoo, apple, other
	CampaignTag        string    `json:"campaign_tag"`
	SentCount          int64     `json:"sent_count"`
	DeliveredCount     int64     `json:"delivered_count"`
	BouncedHardCount   int64     `json:"bounced_hard_count"`
	BouncedSoftCount   int64     `json:"bounced_soft_count"`
	DeferredCount      int64     `json:"deferred_count"`
	ComplainedCount    int64     `json:"complained_count"`
	OpenedUniqueCount  int64     `json:"opened_unique_count"`
	OpenedTotalCount   int64     `json:"opened_total_count"`
	ClickedUniqueCount int64     `json:"clicked_unique_count"`
	ClickedTotalCount  int64     `json:"clicked_total_count"`
	UnsubscribedCount  int64     `json:"unsubscribed_count"`
	LatencyUnder5s     int64     `json:"latency_under_5s"`
	LatencyUnder30s    int64     `json:"latency_under_30s"`
	LatencyUnder1m     int64     `json:"latency_under_1m"`
	LatencyUnder5m     int64     `json:"latency_under_5m"`
	LatencyUnder15m    int64     `json:"latency_under_15m"`
	LatencyOver15m     int64     `json:"latency_over_15m"`
}

type AnalyticsFilter struct {
	From        *time.Time `json:"from,omitempty"`
	To          *time.Time `json:"to,omitempty"`
	StreamID    *string    `json:"stream_id,omitempty"`
	DomainID    *string    `json:"domain_id,omitempty"`
	Provider    *string    `json:"provider,omitempty"`
	CampaignTag *string    `json:"campaign_tag,omitempty"`
}

type AnalyticsSummary struct {
	Sent             int64   `json:"sent"`
	Delivered        int64   `json:"delivered"`
	BouncedHard      int64   `json:"bounced_hard"`
	BouncedSoft      int64   `json:"bounced_soft"`
	Deferred         int64   `json:"deferred"`
	Complained       int64   `json:"complained"`
	OpenedUnique     int64   `json:"opened_unique"`
	OpenedTotal      int64   `json:"opened_total"`
	ClickedUnique    int64   `json:"clicked_unique"`
	ClickedTotal     int64   `json:"clicked_total"`
	Unsubscribed     int64   `json:"unsubscribed"`
	DeliveryRate     float64 `json:"delivery_rate"`     // delivered / sent
	BounceRate       float64 `json:"bounce_rate"`       // (bounced_hard + bounced_soft) / sent
	HardBounceRate   float64 `json:"hard_bounce_rate"`  // bounced_hard / sent
	ComplaintRate    float64 `json:"complaint_rate"`    // complained / delivered
	OpenRate         float64 `json:"open_rate"`         // opened_unique / delivered
	ClickRate        float64 `json:"click_rate"`        // clicked_unique / delivered
	UnsubscribeRate  float64 `json:"unsubscribe_rate"`  // unsubscribed / delivered
	ComplaintWarning bool    `json:"complaint_warning"` // true if complaint_rate >= 0.001 (0.10%)
}

type TimeSeriesPoint struct {
	Timestamp     string  `json:"timestamp"` // ISO 8601
	Sent          int64   `json:"sent"`
	Delivered     int64   `json:"delivered"`
	Bounced       int64   `json:"bounced"`
	Complained    int64   `json:"complained"`
	DeliveryRate  float64 `json:"delivery_rate"`
	ComplaintRate float64 `json:"complaint_rate"`
}

type ProviderBreakdown struct {
	Provider      string  `json:"provider"`
	Sent          int64   `json:"sent"`
	Delivered     int64   `json:"delivered"`
	Bounced       int64   `json:"bounced"`
	Complained    int64   `json:"complained"`
	DeliveryRate  float64 `json:"delivery_rate"`
	ComplaintRate float64 `json:"complaint_rate"`
}

type LatencyHistogram struct {
	Under5s  int64 `json:"under_5s"`
	Under30s int64 `json:"under_30s"`
	Under1m  int64 `json:"under_1m"`
	Under5m  int64 `json:"under_5m"`
	Under15m int64 `json:"under_15m"`
	Over15m  int64 `json:"over_15m"`
}

type AnalyticsExport struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Status         string     `json:"status"` // pending, completed, failed
	Format         string     `json:"format"` // csv
	FilterCriteria string     `json:"filter_criteria,omitempty"`
	FileURL        *string    `json:"file_url,omitempty"`
	FileData       []byte     `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type WebhookEndpoint struct {
	ID                      string     `json:"id"`
	TenantID                string     `json:"tenant_id"`
	URL                     string     `json:"url"`
	Description             *string    `json:"description,omitempty"`
	Events                  []string   `json:"events"` // e.g. ["*"], ["message.delivered", "message.bounced"]
	Secret                  string     `json:"secret,omitempty"`
	PreviousSecret          *string    `json:"previous_secret,omitempty"`
	PreviousSecretExpiresAt *time.Time `json:"previous_secret_expires_at,omitempty"`
	Status                  string     `json:"status"` // active, disabled_failing, disabled_manual
	ConsecutiveFailures     int        `json:"consecutive_failures"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type WebhookDelivery struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	EndpointID   string    `json:"endpoint_id"`
	EventID      string    `json:"event_id"`
	EventType    string    `json:"event_type"`
	Payload      string    `json:"payload"`
	StatusCode   *int      `json:"status_code,omitempty"`
	ResponseBody *string   `json:"response_body,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	Attempt      int       `json:"attempt"`
	DurationMS   int       `json:"duration_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

type SuppressionImport struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Filename      string    `json:"filename"`
	TotalRows     int       `json:"total_rows"`
	ImportedRows  int       `json:"imported_rows"`
	DuplicateRows int       `json:"duplicate_rows"`
	Status        string    `json:"status"` // completed, failed
	CreatedAt     time.Time `json:"created_at"`
}

type PrivacyRequest struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	Type          string     `json:"type"` // erasure, export
	EmailAddress  string     `json:"email_address"`
	Status        string     `json:"status"` // completed, failed
	ResultSummary *string    `json:"result_summary,omitempty"`
	RequestedAt   time.Time  `json:"requested_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type RetentionPolicy struct {
	ID                    string    `json:"id"`
	TenantID              string    `json:"tenant_id"`
	MessageRetentionDays  int       `json:"message_retention_days"`  // default 30
	EventsRetentionDays   int       `json:"events_retention_days"`   // default 90
	AttemptsRetentionDays int       `json:"attempts_retention_days"` // default 30
	RawMIMERetentionDays  int       `json:"raw_mime_retention_days"` // default 7
	UpdatedAt             time.Time `json:"updated_at"`
}

type PrivacyExportReport struct {
	EmailAddress string              `json:"email_address"`
	GeneratedAt  time.Time           `json:"generated_at"`
	Messages     []*MessageRecipient `json:"messages"`
	Events       []*MessageEvent     `json:"events"`
	Suppressions []*Suppression      `json:"suppressions"`
}

type RetentionPurgeSummary struct {
	TenantID            string    `json:"tenant_id"`
	PurgedAt            time.Time `json:"purged_at"`
	PurgedMessagesCount int       `json:"purged_messages_count"`
	PurgedEventsCount   int       `json:"purged_events_count"`
	PurgedAttemptsCount int       `json:"purged_attempts_count"`
	PurgedRawMIMECount  int       `json:"purged_raw_mime_count"`
}

type TrackingLink struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	MessageID string    `json:"message_id"`
	LinkID    string    `json:"link_id"`
	TargetURL string    `json:"target_url"`
	CreatedAt time.Time `json:"created_at"`
}

type TrackingEvent struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	MessageID   string    `json:"message_id"`
	RecipientID string    `json:"recipient_id"`
	Type        string    `json:"type"` // open, click
	TargetURL   *string   `json:"target_url,omitempty"`
	IP          string    `json:"ip"`
	UserAgent   *string   `json:"user_agent,omitempty"`
	IsBot       bool      `json:"is_bot"`
	BotReason   *string   `json:"bot_reason,omitempty"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type InboundRoute struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Priority     int       `json:"priority"`
	MatchType    string    `json:"match_type"` // exact, wildcard, catch_all
	Pattern      string    `json:"pattern"`
	Action       string    `json:"action"` // webhook, store_only, forward, discard
	ActionTarget *string   `json:"action_target,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type InboundMessage struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	RouteID     *string   `json:"route_id,omitempty"`
	MessageID   *string   `json:"message_id,omitempty"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	Subject     *string   `json:"subject,omitempty"`
	SizeBytes   int       `json:"size_bytes"`
	RawMIMERef  *string   `json:"raw_mime_ref,omitempty"`
	SPFStatus   string    `json:"spf_status"`
	DKIMStatus  string    `json:"dkim_status"`
	DMARCStatus string    `json:"dmarc_status"`
	SpamScore   float64   `json:"spam_score"`
	Status      string    `json:"status"` // processed, failed, quarantined
	CreatedAt   time.Time `json:"created_at"`
}

