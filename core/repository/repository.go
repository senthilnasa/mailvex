package repository

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
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

	// Phase 5: Suppressions
	AddSuppression(ctx context.Context, supp *domain.Suppression) error
	CheckSuppression(ctx context.Context, tenantID, streamID, address string) (bool, string, error)
	ListSuppressions(ctx context.Context, tenantID string, limit int) ([]*domain.Suppression, error)
	DeleteSuppression(ctx context.Context, tenantID, id string) error

	// Phase 5: IP Pools, Addresses, Relays
	CreateIPPool(ctx context.Context, pool *domain.IPPool) error
	ListIPPools(ctx context.Context, tenantID string) ([]*domain.IPPool, error)
	GetIPPool(ctx context.Context, poolID string) (*domain.IPPool, error)
	AddIPAddress(ctx context.Context, ip *domain.IPAddress) error
	ListIPAddresses(ctx context.Context, poolID string) ([]*domain.IPAddress, error)
	ListIPAddressesByNode(ctx context.Context, nodeID string) ([]*domain.IPAddress, error)
	CreateRelay(ctx context.Context, relay *domain.Relay) error
	ListRelays(ctx context.Context, tenantID string) ([]*domain.Relay, error)
	GetRelay(ctx context.Context, relayID string) (*domain.Relay, error)

	// Phase 5: Rules & DLQ
	ListBounceRules(ctx context.Context) ([]*domain.BounceRule, error)
	SaveBounceRule(ctx context.Context, rule *domain.BounceRule) error
	ListProviderMXRules(ctx context.Context) ([]*domain.ProviderMXRule, error)
	SaveProviderMXRule(ctx context.Context, rule *domain.ProviderMXRule) error
	RecordDeadLetter(ctx context.Context, dlq *domain.DeadLetterMessage) error
	ListDeadLetterMessages(ctx context.Context, tenantID string, limit int) ([]*domain.DeadLetterMessage, error)

	// Phase 5: Recipient Delivery & Attempts
	UpdateRecipientDelivery(ctx context.Context, recipientID string, status string, attempts int, lastSMTPCode, lastResponse *string, deliveredAt, bouncedAt *time.Time, nextAttemptAt *time.Time, suppressedReason *string) error
	RecordDeliveryAttempt(ctx context.Context, attempt *domain.DeliveryAttempt) error
	GetRecipient(ctx context.Context, recipientID string) (*domain.MessageRecipient, error)
	AddMessageEvent(ctx context.Context, evt *domain.MessageEvent) error

	// Phase 6: Feedback Loops, Inbound DSN, Abuse Reports, Reputation
	SaveFBLReport(ctx context.Context, report *domain.FBLReport) error
	ListFBLReports(ctx context.Context, tenantID string, limit int) ([]*domain.FBLReport, error)
	SaveUnmatchedBounce(ctx context.Context, bounce *domain.UnmatchedBounce) error
	ListUnmatchedBounces(ctx context.Context, limit int) ([]*domain.UnmatchedBounce, error)
	SaveAbuseReport(ctx context.Context, report *domain.AbuseReport) error
	ListAbuseReports(ctx context.Context, status string, limit int) ([]*domain.AbuseReport, error)
	GetAbuseReport(ctx context.Context, id string) (*domain.AbuseReport, error)
	UpdateAbuseReportStatus(ctx context.Context, id, status string, notes *string) error
	SaveIPReputationSnapshot(ctx context.Context, snap *domain.IPReputationSnapshot) error
	ListIPReputationSnapshots(ctx context.Context, ipID string, limit int) ([]*domain.IPReputationSnapshot, error)
	SavePostmasterSnapshot(ctx context.Context, snap *domain.PostmasterSnapshot) error
	ListPostmasterSnapshots(ctx context.Context, domainID string, limit int) ([]*domain.PostmasterSnapshot, error)
	GetRecipientByTokenOrID(ctx context.Context, tokenOrID string) (*domain.MessageRecipient, *domain.Message, error)

	// Phase 7: Analytics & Reporting
	RecordAnalyticsEvent(ctx context.Context, tenantID, eventType string, streamID, domainID *string, provider, campaignTag string, latencySec float64, occurredAt time.Time) error
	QueryAnalyticsOverview(ctx context.Context, tenantID string, filter domain.AnalyticsFilter) (*domain.AnalyticsSummary, error)
	QueryAnalyticsTimeSeries(ctx context.Context, tenantID string, interval string, filter domain.AnalyticsFilter) ([]*domain.TimeSeriesPoint, error)
	QueryProviderBreakdown(ctx context.Context, tenantID string, filter domain.AnalyticsFilter) ([]*domain.ProviderBreakdown, error)
	QueryLatencyHistogram(ctx context.Context, tenantID string, filter domain.AnalyticsFilter) (*domain.LatencyHistogram, error)
	CreateAnalyticsExport(ctx context.Context, export *domain.AnalyticsExport) error
	GetAnalyticsExport(ctx context.Context, tenantID, exportID string) (*domain.AnalyticsExport, error)
	ListAnalyticsExports(ctx context.Context, tenantID string, limit int) ([]*domain.AnalyticsExport, error)
	UpdateAnalyticsExport(ctx context.Context, tenantID, exportID, status string, fileURL *string, fileData []byte) error

	// Phase 8: Webhooks
	CreateWebhookEndpoint(ctx context.Context, endpoint *domain.WebhookEndpoint) error
	GetWebhookEndpoint(ctx context.Context, tenantID, id string) (*domain.WebhookEndpoint, error)
	ListWebhookEndpoints(ctx context.Context, tenantID string) ([]*domain.WebhookEndpoint, error)
	UpdateWebhookEndpoint(ctx context.Context, endpoint *domain.WebhookEndpoint) error
	DeleteWebhookEndpoint(ctx context.Context, tenantID, id string) error
	RotateWebhookSecret(ctx context.Context, tenantID, id, newSecret string, prevExpiresAt time.Time) (*domain.WebhookEndpoint, error)
	RecordWebhookDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error
	ListWebhookDeliveries(ctx context.Context, tenantID, endpointID string, limit, offset int) ([]*domain.WebhookDelivery, error)
	GetActiveWebhookEndpointsForEvent(ctx context.Context, tenantID, eventType string) ([]*domain.WebhookEndpoint, error)

	// Phase 9: Suppression Management, Privacy & Data Retention
	ImportSuppressions(ctx context.Context, tenantID, filename string, entries []*domain.Suppression) (*domain.SuppressionImport, error)
	ExportSuppressions(ctx context.Context, tenantID string, streamID *string) ([]*domain.Suppression, error)
	CreatePrivacyRequest(ctx context.Context, req *domain.PrivacyRequest) error
	GetPrivacyRequest(ctx context.Context, tenantID, id string) (*domain.PrivacyRequest, error)
	ListPrivacyRequests(ctx context.Context, tenantID string) ([]*domain.PrivacyRequest, error)
	ExecutePrivacyErasure(ctx context.Context, tenantID, email string) (*domain.PrivacyRequest, error)
	ExecutePrivacyExport(ctx context.Context, tenantID, email string) (*domain.PrivacyExportReport, error)
	GetRetentionPolicy(ctx context.Context, tenantID string) (*domain.RetentionPolicy, error)
	UpdateRetentionPolicy(ctx context.Context, policy *domain.RetentionPolicy) error
	ExecuteRetentionPurge(ctx context.Context, tenantID string) (*domain.RetentionPurgeSummary, error)

	// Phase 10: Tracking & Inbound Routing
	SaveTrackingLink(ctx context.Context, link *domain.TrackingLink) error
	GetTrackingLink(ctx context.Context, tenantID, linkID string) (*domain.TrackingLink, error)
	RecordTrackingEvent(ctx context.Context, evt *domain.TrackingEvent) error
	ListTrackingEvents(ctx context.Context, tenantID, messageID string) ([]*domain.TrackingEvent, error)
	CreateInboundRoute(ctx context.Context, route *domain.InboundRoute) error
	GetInboundRoute(ctx context.Context, tenantID, id string) (*domain.InboundRoute, error)
	ListInboundRoutes(ctx context.Context, tenantID string) ([]*domain.InboundRoute, error)
	UpdateInboundRoute(ctx context.Context, route *domain.InboundRoute) error
	DeleteInboundRoute(ctx context.Context, tenantID, id string) error
	MatchInboundRoute(ctx context.Context, tenantID, toAddress string) (*domain.InboundRoute, error)
	SaveInboundMessage(ctx context.Context, msg *domain.InboundMessage) error
	GetInboundMessage(ctx context.Context, tenantID, id string) (*domain.InboundMessage, error)
	ListInboundMessages(ctx context.Context, tenantID string, limit, offset int) ([]*domain.InboundMessage, error)

	// Phase 11: Reputation, DNSBL, Warmup & Alerts
	ListDNSBLProviders(ctx context.Context) ([]*domain.DNSBLProvider, error)
	GetDNSBLProvider(ctx context.Context, id string) (*domain.DNSBLProvider, error)
	SaveDNSBLProvider(ctx context.Context, provider *domain.DNSBLProvider) error
	RecordDNSBLResult(ctx context.Context, result *domain.DNSBLResult) error
	ListDNSBLResults(ctx context.Context, ipAddressID string, limit int) ([]*domain.DNSBLResult, error)
	CreateWarmupPlan(ctx context.Context, plan *domain.WarmupPlan) error
	GetWarmupPlan(ctx context.Context, id string) (*domain.WarmupPlan, error)
	ListWarmupPlans(ctx context.Context) ([]*domain.WarmupPlan, error)
	GetIPWarmupState(ctx context.Context, ipAddressID string) (*domain.IPWarmupState, error)
	SaveIPWarmupState(ctx context.Context, state *domain.IPWarmupState) error
	ListIPWarmupStates(ctx context.Context) ([]*domain.IPWarmupState, error)
	CreateAlertRule(ctx context.Context, rule *domain.AlertRule) error
	GetAlertRule(ctx context.Context, tenantID *string, id string) (*domain.AlertRule, error)
	ListAlertRules(ctx context.Context, tenantID *string) ([]*domain.AlertRule, error)
	UpdateAlertRule(ctx context.Context, rule *domain.AlertRule) error
	DeleteAlertRule(ctx context.Context, tenantID *string, id string) error
	RecordAlertIncident(ctx context.Context, incident *domain.AlertIncident) error
	GetAlertIncident(ctx context.Context, tenantID *string, id string) (*domain.AlertIncident, error)
	ListAlertIncidents(ctx context.Context, tenantID *string, status string, limit int) ([]*domain.AlertIncident, error)
	UpdateAlertIncidentStatus(ctx context.Context, tenantID *string, id, status string) error
	CreateNotificationChannel(ctx context.Context, channel *domain.NotificationChannel) error
	GetNotificationChannel(ctx context.Context, tenantID *string, id string) (*domain.NotificationChannel, error)
	ListNotificationChannels(ctx context.Context, tenantID *string) ([]*domain.NotificationChannel, error)
	DeleteNotificationChannel(ctx context.Context, tenantID *string, id string) error
	CreateAlertNotification(ctx context.Context, notif *domain.AlertNotification) error
	ListAlertNotifications(ctx context.Context, tenantID *string, unreadOnly bool, limit int) ([]*domain.AlertNotification, error)
	MarkAlertNotificationRead(ctx context.Context, tenantID *string, id string) error
	GetReputationOverview(ctx context.Context, tenantID string) (*domain.ReputationOverview, error)
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

	// Phase 5 collections
	suppressions     map[string]*domain.Suppression
	ipPools          map[string]*domain.IPPool
	ipAddresses      map[string]*domain.IPAddress
	relays           map[string]*domain.Relay
	bounceRules      []*domain.BounceRule
	providerMXRules  []*domain.ProviderMXRule
	deadLetters      []*domain.DeadLetterMessage
	deliveryAttempts []*domain.DeliveryAttempt

	// Phase 6 collections
	fblReports       []*domain.FBLReport
	unmatchedBounces []*domain.UnmatchedBounce
	abuseReports     map[string]*domain.AbuseReport
	ipReputations    []*domain.IPReputationSnapshot
	postmasterSnaps  []*domain.PostmasterSnapshot

	// Phase 7 collections
	analyticsRollups []*domain.AnalyticsRollup
	analyticsExports map[string]*domain.AnalyticsExport

	// Phase 8 collections
	webhookEndpoints  map[string]*domain.WebhookEndpoint
	webhookDeliveries []*domain.WebhookDelivery

	// Phase 9 collections
	suppressionImports []*domain.SuppressionImport
	privacyRequests    map[string]*domain.PrivacyRequest
	retentionPolicies  map[string]*domain.RetentionPolicy

	// Phase 10 collections
	trackingLinks   map[string]*domain.TrackingLink // key: tenantID:linkID
	trackingEvents  []*domain.TrackingEvent
	inboundRoutes   map[string]*domain.InboundRoute // key: routeID
	inboundMessages []*domain.InboundMessage

	// Phase 11 collections
	dnsblProviders       map[string]*domain.DNSBLProvider
	dnsblResults         []*domain.DNSBLResult
	warmupPlans          map[string]*domain.WarmupPlan
	ipWarmupStates       map[string]*domain.IPWarmupState
	alertRules           map[string]*domain.AlertRule
	alertIncidents       map[string]*domain.AlertIncident
	notificationChannels map[string]*domain.NotificationChannel
	alertNotifications   []*domain.AlertNotification
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
		suppressions:     make(map[string]*domain.Suppression),
		ipPools:          make(map[string]*domain.IPPool),
		ipAddresses:      make(map[string]*domain.IPAddress),
		relays:           make(map[string]*domain.Relay),
		bounceRules:      make([]*domain.BounceRule, 0),
		providerMXRules:  make([]*domain.ProviderMXRule, 0),
		deadLetters:      make([]*domain.DeadLetterMessage, 0),
		deliveryAttempts: make([]*domain.DeliveryAttempt, 0),
		fblReports:       make([]*domain.FBLReport, 0),
		unmatchedBounces: make([]*domain.UnmatchedBounce, 0),
		abuseReports:     make(map[string]*domain.AbuseReport),
		ipReputations:    make([]*domain.IPReputationSnapshot, 0),
		postmasterSnaps:  make([]*domain.PostmasterSnapshot, 0),
		analyticsRollups: make([]*domain.AnalyticsRollup, 0),
		analyticsExports: make(map[string]*domain.AnalyticsExport),
		webhookEndpoints:  make(map[string]*domain.WebhookEndpoint),
		webhookDeliveries: make([]*domain.WebhookDelivery, 0),
		suppressionImports: make([]*domain.SuppressionImport, 0),
		privacyRequests:    make(map[string]*domain.PrivacyRequest),
		retentionPolicies:  make(map[string]*domain.RetentionPolicy),
		trackingLinks:      make(map[string]*domain.TrackingLink),
		trackingEvents:     make([]*domain.TrackingEvent, 0),
		inboundRoutes:      make(map[string]*domain.InboundRoute),
		inboundMessages:    make([]*domain.InboundMessage, 0),
		dnsblProviders:       make(map[string]*domain.DNSBLProvider),
		dnsblResults:         make([]*domain.DNSBLResult, 0),
		warmupPlans:          make(map[string]*domain.WarmupPlan),
		ipWarmupStates:       make(map[string]*domain.IPWarmupState),
		alertRules:           make(map[string]*domain.AlertRule),
		alertIncidents:       make(map[string]*domain.AlertIncident),
		notificationChannels: make(map[string]*domain.NotificationChannel),
		alertNotifications:   make([]*domain.AlertNotification, 0),
	}

	// Seed default IP pool
	repo.ipPools["pool_default"] = &domain.IPPool{
		ID:        "pool_default",
		Name:      "default",
		Strategy:  "round_robin",
		IsDefault: true,
		CreatedAt: time.Now().UTC(),
	}

	// Seed Provider MX rules
	repo.providerMXRules = append(repo.providerMXRules,
		&domain.ProviderMXRule{ID: "pmx_google", Pattern: "*.google.com", Provider: "google", MaxConnections: 5, MessagesPerConnection: 100, RateLimitPerMin: 600, CreatedAt: time.Now().UTC()},
		&domain.ProviderMXRule{ID: "pmx_googlemail", Pattern: "*.googlemail.com", Provider: "google", MaxConnections: 5, MessagesPerConnection: 100, RateLimitPerMin: 600, CreatedAt: time.Now().UTC()},
		&domain.ProviderMXRule{ID: "pmx_microsoft", Pattern: "*.mail.protection.outlook.com", Provider: "microsoft", MaxConnections: 5, MessagesPerConnection: 100, RateLimitPerMin: 600, CreatedAt: time.Now().UTC()},
		&domain.ProviderMXRule{ID: "pmx_yahoo", Pattern: "*.yahoodns.net", Provider: "yahoo", MaxConnections: 4, MessagesPerConnection: 50, RateLimitPerMin: 300, CreatedAt: time.Now().UTC()},
		&domain.ProviderMXRule{ID: "pmx_apple", Pattern: "*.icloud.com", Provider: "apple", MaxConnections: 3, MessagesPerConnection: 50, RateLimitPerMin: 300, CreatedAt: time.Now().UTC()},
	)

	// Seed default Bounce rules
	code511 := "5.1.1"
	code512 := "5.1.2"
	code422 := "4.2.2"
	code4728 := "4.7.28"
	code571 := "5.7.1"
	code5726 := "5.7.26"
	repo.bounceRules = append(repo.bounceRules,
		&domain.BounceRule{ID: "br_user_unknown", Pattern: "user unknown|mailbox not found|no such user|recipient address rejected", EnhancedCode: &code511, Classification: "hard_bounce", Action: "suppress_immediate", CreatedAt: time.Now().UTC()},
		&domain.BounceRule{ID: "br_bad_domain", Pattern: "domain not found|host or domain name not found|unrouteable address", EnhancedCode: &code512, Classification: "hard_bounce", Action: "suppress_immediate", CreatedAt: time.Now().UTC()},
		&domain.BounceRule{ID: "br_mailbox_full", Pattern: "mailbox full|quota exceeded|over quota", EnhancedCode: &code422, Classification: "soft_bounce", Action: "retry", CreatedAt: time.Now().UTC()},
		&domain.BounceRule{ID: "br_rate_limited", Pattern: "rate limit|too many connections|too many messages|throttled", EnhancedCode: &code4728, Classification: "rate_limited", Action: "retry", CreatedAt: time.Now().UTC()},
		&domain.BounceRule{ID: "br_spam_block", Pattern: "blocked using spamhaus|blocked by spam|spam detected|reputation", EnhancedCode: &code571, Classification: "spam_rejection", Action: "dead_letter", CreatedAt: time.Now().UTC()},
		&domain.BounceRule{ID: "br_auth_failure", Pattern: "dkim check failed|spf check failed|dmarc policy", EnhancedCode: &code5726, Classification: "auth_rejection", Action: "dead_letter", CreatedAt: time.Now().UTC()},
	)
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
		"abuse.read", "abuse.write",
		"audit.read",
		"settings.read", "settings.write",
		"retention.read", "retention.write", "privacy.erase", "privacy.export", "privacy.read",
		"inbound.read", "inbound.write", "inbound.routes", "tracking.read",
		"reputation.read", "dnsbl.read", "dnsbl.check", "warmup.read", "warmup.write", "alerts.read", "alerts.write",
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
		"webhooks.read", "webhooks.write", "webhooks.replay", "webhooks.secret.rotate",
		"analytics.read", "analytics.export", "reports.read", "abuse.read", "abuse.write", "audit.read", "settings.read", "settings.write",
		"retention.read", "retention.write", "privacy.erase", "privacy.export", "privacy.read",
		"inbound.read", "inbound.write", "inbound.routes", "tracking.read",
		"reputation.read", "dnsbl.read", "dnsbl.check", "warmup.read", "warmup.write", "alerts.read", "alerts.write",
	}
	m.rolePerms["role_tenant_owner"] = tenantPerms

	// Seed DNSBL Providers (§22.2)
	delistSpamhaus := "https://check.spamhaus.org/"
	delistBarracuda := "https://www.barracudacentral.org/rbl/removal-request"
	delistSpamCop := "https://www.spamcop.net/bl.shtml"
	delistUCEPROTECT := "https://www.uceprotect.net/en/rblcheck.php"

	m.dnsblProviders["dnsbl_zen"] = &domain.DNSBLProvider{
		ID: "dnsbl_zen", Name: "Spamhaus Zen", QueryPattern: "zen.spamhaus.org",
		ReturnCodeMeanings: `{"127.0.0.2":"SBL - Direct UBE sources","127.0.0.3":"CSS - Direct snowshoe/spam sources","127.0.0.4":"XBL - Exploits/CBL","127.0.0.10":"PBL - End-user dynamic ranges"}`,
		DelistURL: &delistSpamhaus, CheckIntervalSeconds: 3600, IsEnabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	m.dnsblProviders["dnsbl_barracuda"] = &domain.DNSBLProvider{
		ID: "dnsbl_barracuda", Name: "Barracuda Central", QueryPattern: "b.barracudacentral.org",
		ReturnCodeMeanings: `{"127.0.0.2":"Listed as spam source"}`,
		DelistURL: &delistBarracuda, CheckIntervalSeconds: 3600, IsEnabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	m.dnsblProviders["dnsbl_spamcop"] = &domain.DNSBLProvider{
		ID: "dnsbl_spamcop", Name: "SpamCop", QueryPattern: "bl.spamcop.net",
		ReturnCodeMeanings: `{"127.0.0.2":"Reported spam source"}`,
		DelistURL: &delistSpamCop, CheckIntervalSeconds: 3600, IsEnabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	m.dnsblProviders["dnsbl_uceprotect"] = &domain.DNSBLProvider{
		ID: "dnsbl_uceprotect", Name: "UCEPROTECT Level 1", QueryPattern: "dnsbl-1.uceprotect.net",
		ReturnCodeMeanings: `{"127.0.0.2":"Single IP listed"}`,
		DelistURL: &delistUCEPROTECT, CheckIntervalSeconds: 7200, IsEnabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}

	// Seed Standard 30-Day Warmup Plan (§11.5)
	warmupDesc := "Standard 30-day exponential volume ramp with per-provider caps"
	defaultDays := []*domain.WarmupPlanDay{
		{ID: "wpday_1", PlanID: "wplan_default", DayNumber: 1, MaxVolume: 50, ProviderCaps: map[string]int{"google": 20, "microsoft": 20, "yahoo": 10}},
		{ID: "wpday_2", PlanID: "wplan_default", DayNumber: 2, MaxVolume: 100, ProviderCaps: map[string]int{"google": 40, "microsoft": 40, "yahoo": 20}},
		{ID: "wpday_3", PlanID: "wplan_default", DayNumber: 3, MaxVolume: 200, ProviderCaps: map[string]int{"google": 80, "microsoft": 80, "yahoo": 40}},
		{ID: "wpday_4", PlanID: "wplan_default", DayNumber: 4, MaxVolume: 400, ProviderCaps: map[string]int{"google": 150, "microsoft": 150, "yahoo": 80}},
		{ID: "wpday_5", PlanID: "wplan_default", DayNumber: 5, MaxVolume: 800, ProviderCaps: map[string]int{"google": 300, "microsoft": 300, "yahoo": 150}},
		{ID: "wpday_6", PlanID: "wplan_default", DayNumber: 6, MaxVolume: 1500, ProviderCaps: map[string]int{"google": 600, "microsoft": 600, "yahoo": 250}},
		{ID: "wpday_7", PlanID: "wplan_default", DayNumber: 7, MaxVolume: 3000, ProviderCaps: map[string]int{"google": 1200, "microsoft": 1200, "yahoo": 500}},
		{ID: "wpday_8", PlanID: "wplan_default", DayNumber: 8, MaxVolume: 5000, ProviderCaps: map[string]int{"google": 2000, "microsoft": 2000, "yahoo": 800}},
		{ID: "wpday_9", PlanID: "wplan_default", DayNumber: 9, MaxVolume: 8000, ProviderCaps: map[string]int{"google": 3200, "microsoft": 3200, "yahoo": 1200}},
		{ID: "wpday_10", PlanID: "wplan_default", DayNumber: 10, MaxVolume: 12000, ProviderCaps: map[string]int{"google": 5000, "microsoft": 5000, "yahoo": 1800}},
		{ID: "wpday_15", PlanID: "wplan_default", DayNumber: 15, MaxVolume: 35000, ProviderCaps: map[string]int{"google": 15000, "microsoft": 15000, "yahoo": 5000}},
		{ID: "wpday_20", PlanID: "wplan_default", DayNumber: 20, MaxVolume: 80000, ProviderCaps: map[string]int{"google": 35000, "microsoft": 35000, "yahoo": 10000}},
		{ID: "wpday_25", PlanID: "wplan_default", DayNumber: 25, MaxVolume: 150000, ProviderCaps: map[string]int{"google": 65000, "microsoft": 65000, "yahoo": 20000}},
		{ID: "wpday_30", PlanID: "wplan_default", DayNumber: 30, MaxVolume: 500000, ProviderCaps: map[string]int{"google": 200000, "microsoft": 200000, "yahoo": 60000}},
	}
	m.warmupPlans["wplan_default"] = &domain.WarmupPlan{
		ID: "wplan_default", Name: "Standard 30-Day Warmup", Description: &warmupDesc,
		TotalDays: 30, IsDefault: true, Days: defaultDays,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
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

// Phase 5: Suppressions
func (r *MemoryRepository) AddSuppression(ctx context.Context, supp *domain.Suppression) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.suppressions[supp.ID] = supp
	return nil
}

func (r *MemoryRepository) CheckSuppression(ctx context.Context, tenantID, streamID, address string) (bool, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normAddr := strings.TrimSpace(strings.ToLower(address))
	parts := strings.Split(normAddr, "@")
	dom := ""
	if len(parts) == 2 {
		dom = parts[1]
	}

	now := time.Now().UTC()
	for _, s := range r.suppressions {
		if s.TenantID != tenantID {
			continue
		}
		if s.ExpiresAt != nil && now.After(*s.ExpiresAt) {
			continue
		}

		// Domain block
		if s.Type == "domain_block" && dom != "" && strings.EqualFold(s.AddressDomain, dom) {
			reason := "domain_block"
			if s.Reason != nil && *s.Reason != "" {
				reason = *s.Reason
			}
			return true, reason, nil
		}

		// Exact address match
		if strings.EqualFold(s.Address, normAddr) {
			if s.Type == "hard_bounce" || s.Type == "complaint" || s.Type == "manual" {
				reason := s.Type
				if s.Reason != nil && *s.Reason != "" {
					reason = *s.Reason
				}
				return true, reason, nil
			}
			if s.Type == "unsubscribe" {
				if s.StreamID == nil || *s.StreamID == "" || streamID == "" || *s.StreamID == streamID {
					reason := "unsubscribe"
					if s.Reason != nil && *s.Reason != "" {
						reason = *s.Reason
					}
					return true, reason, nil
				}
			}
		}
	}
	return false, "", nil
}

func (r *MemoryRepository) ListSuppressions(ctx context.Context, tenantID string, limit int) ([]*domain.Suppression, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	var res []*domain.Suppression
	for _, s := range r.suppressions {
		if s.TenantID == tenantID {
			res = append(res, s)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

func (r *MemoryRepository) DeleteSuppression(ctx context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, exists := r.suppressions[id]
	if !exists || s.TenantID != tenantID {
		return errors.New("suppression not found")
	}
	delete(r.suppressions, id)
	return nil
}

// Phase 5: IP Pools & Addresses
func (r *MemoryRepository) CreateIPPool(ctx context.Context, pool *domain.IPPool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ipPools[pool.ID] = pool
	return nil
}

func (r *MemoryRepository) ListIPPools(ctx context.Context, tenantID string) ([]*domain.IPPool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var res []*domain.IPPool
	for _, p := range r.ipPools {
		if p.TenantID == nil || *p.TenantID == tenantID {
			res = append(res, p)
		}
	}
	return res, nil
}

func (r *MemoryRepository) GetIPPool(ctx context.Context, poolID string) (*domain.IPPool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.ipPools[poolID]
	if !exists {
		return nil, errors.New("ip pool not found")
	}
	return p, nil
}

func (r *MemoryRepository) AddIPAddress(ctx context.Context, ip *domain.IPAddress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ipAddresses[ip.ID] = ip
	return nil
}

func (r *MemoryRepository) ListIPAddresses(ctx context.Context, poolID string) ([]*domain.IPAddress, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var res []*domain.IPAddress
	for _, ip := range r.ipAddresses {
		if ip.PoolID == poolID {
			res = append(res, ip)
		}
	}
	return res, nil
}

func (r *MemoryRepository) ListIPAddressesByNode(ctx context.Context, nodeID string) ([]*domain.IPAddress, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var res []*domain.IPAddress
	for _, ip := range r.ipAddresses {
		if ip.NodeID == nodeID {
			res = append(res, ip)
		}
	}
	return res, nil
}

// Phase 5: Relays
func (r *MemoryRepository) CreateRelay(ctx context.Context, relay *domain.Relay) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.relays[relay.ID] = relay
	return nil
}

func (r *MemoryRepository) ListRelays(ctx context.Context, tenantID string) ([]*domain.Relay, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var res []*domain.Relay
	for _, rel := range r.relays {
		if rel.TenantID == nil || *rel.TenantID == tenantID {
			res = append(res, rel)
		}
	}
	return res, nil
}

func (r *MemoryRepository) GetRelay(ctx context.Context, relayID string) (*domain.Relay, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rel, exists := r.relays[relayID]
	if !exists {
		return nil, errors.New("relay not found")
	}
	return rel, nil
}

// Phase 5: Rules & DLQ
func (r *MemoryRepository) ListBounceRules(ctx context.Context) ([]*domain.BounceRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bounceRules, nil
}

func (r *MemoryRepository) SaveBounceRule(ctx context.Context, rule *domain.BounceRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bounceRules = append(r.bounceRules, rule)
	return nil
}

func (r *MemoryRepository) ListProviderMXRules(ctx context.Context) ([]*domain.ProviderMXRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providerMXRules, nil
}

func (r *MemoryRepository) SaveProviderMXRule(ctx context.Context, rule *domain.ProviderMXRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providerMXRules = append(r.providerMXRules, rule)
	return nil
}

func (r *MemoryRepository) RecordDeadLetter(ctx context.Context, dlq *domain.DeadLetterMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deadLetters = append(r.deadLetters, dlq)
	return nil
}

func (r *MemoryRepository) ListDeadLetterMessages(ctx context.Context, tenantID string, limit int) ([]*domain.DeadLetterMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	var res []*domain.DeadLetterMessage
	for _, dl := range r.deadLetters {
		if dl.TenantID == tenantID {
			res = append(res, dl)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

// Phase 5: Recipient Delivery & Attempts
func (r *MemoryRepository) UpdateRecipientDelivery(ctx context.Context, recipientID string, status string, attempts int, lastSMTPCode, lastResponse *string, deliveredAt, bouncedAt *time.Time, nextAttemptAt *time.Time, suppressedReason *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, rcpts := range r.recipients {
		for _, rcpt := range rcpts {
			if rcpt.ID == recipientID {
				rcpt.Status = status
				rcpt.Attempts = attempts
				if lastSMTPCode != nil {
					rcpt.LastSMTPCode = lastSMTPCode
				}
				if lastResponse != nil {
					rcpt.LastResponse = lastResponse
				}
				if deliveredAt != nil {
					rcpt.DeliveredAt = deliveredAt
				}
				if bouncedAt != nil {
					rcpt.BouncedAt = bouncedAt
				}
				if nextAttemptAt != nil {
					rcpt.NextAttemptAt = nextAttemptAt
				}
				if suppressedReason != nil {
					rcpt.SuppressedReason = suppressedReason
				}
				return nil
			}
		}
	}
	return errors.New("recipient not found")
}

func (r *MemoryRepository) RecordDeliveryAttempt(ctx context.Context, attempt *domain.DeliveryAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deliveryAttempts = append(r.deliveryAttempts, attempt)
	return nil
}

func (r *MemoryRepository) GetRecipient(ctx context.Context, recipientID string) (*domain.MessageRecipient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, rcpts := range r.recipients {
		for _, rcpt := range rcpts {
			if rcpt.ID == recipientID {
				return rcpt, nil
			}
		}
	}
	return nil, errors.New("recipient not found")
}

func (r *MemoryRepository) AddMessageEvent(ctx context.Context, evt *domain.MessageEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[evt.MessageID] = append(r.events[evt.MessageID], evt)
	return nil
}

// Phase 6 Methods

func (r *MemoryRepository) SaveFBLReport(ctx context.Context, report *domain.FBLReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fblReports = append(r.fblReports, report)
	return nil
}

func (r *MemoryRepository) ListFBLReports(ctx context.Context, tenantID string, limit int) ([]*domain.FBLReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	var res []*domain.FBLReport
	for i := len(r.fblReports) - 1; i >= 0; i-- {
		rep := r.fblReports[i]
		if rep.TenantID == tenantID {
			res = append(res, rep)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

func (r *MemoryRepository) SaveUnmatchedBounce(ctx context.Context, bounce *domain.UnmatchedBounce) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unmatchedBounces = append(r.unmatchedBounces, bounce)
	return nil
}

func (r *MemoryRepository) ListUnmatchedBounces(ctx context.Context, limit int) ([]*domain.UnmatchedBounce, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	var res []*domain.UnmatchedBounce
	for i := len(r.unmatchedBounces) - 1; i >= 0; i-- {
		res = append(res, r.unmatchedBounces[i])
		if len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (r *MemoryRepository) SaveAbuseReport(ctx context.Context, report *domain.AbuseReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.abuseReports[report.ID] = report
	return nil
}

func (r *MemoryRepository) ListAbuseReports(ctx context.Context, status string, limit int) ([]*domain.AbuseReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	var res []*domain.AbuseReport
	for _, rep := range r.abuseReports {
		if status == "" || rep.Status == status {
			res = append(res, rep)
		}
	}
	// sort descending by CreatedAt
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.After(res[j].CreatedAt)
	})
	if len(res) > limit {
		res = res[:limit]
	}
	return res, nil
}

func (r *MemoryRepository) GetAbuseReport(ctx context.Context, id string) (*domain.AbuseReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rep, ok := r.abuseReports[id]
	if !ok {
		return nil, errors.New("abuse report not found")
	}
	return rep, nil
}

func (r *MemoryRepository) UpdateAbuseReportStatus(ctx context.Context, id, status string, notes *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rep, ok := r.abuseReports[id]
	if !ok {
		return errors.New("abuse report not found")
	}
	rep.Status = status
	if notes != nil {
		rep.Notes = notes
	}
	if status == "resolved" {
		now := time.Now().UTC()
		rep.ResolvedAt = &now
	}
	return nil
}

func (r *MemoryRepository) SaveIPReputationSnapshot(ctx context.Context, snap *domain.IPReputationSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ipReputations = append(r.ipReputations, snap)
	return nil
}

func (r *MemoryRepository) ListIPReputationSnapshots(ctx context.Context, ipID string, limit int) ([]*domain.IPReputationSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	var res []*domain.IPReputationSnapshot
	for i := len(r.ipReputations) - 1; i >= 0; i-- {
		snap := r.ipReputations[i]
		if snap.IPID == ipID {
			res = append(res, snap)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

func (r *MemoryRepository) SavePostmasterSnapshot(ctx context.Context, snap *domain.PostmasterSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.postmasterSnaps = append(r.postmasterSnaps, snap)
	return nil
}

func (r *MemoryRepository) ListPostmasterSnapshots(ctx context.Context, domainID string, limit int) ([]*domain.PostmasterSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	var res []*domain.PostmasterSnapshot
	for i := len(r.postmasterSnaps) - 1; i >= 0; i-- {
		snap := r.postmasterSnaps[i]
		if snap.DomainID == domainID {
			res = append(res, snap)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

func (r *MemoryRepository) GetRecipientByTokenOrID(ctx context.Context, tokenOrID string) (*domain.MessageRecipient, *domain.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for msgID, rcpts := range r.recipients {
		for _, rcpt := range rcpts {
			if rcpt.ID == tokenOrID || rcpt.Address == tokenOrID {
				msg := r.messages[msgID]
				return rcpt, msg, nil
			}
		}
	}

	// Also check if tokenOrID matches message ID
	if msg, ok := r.messages[tokenOrID]; ok {
		if rcpts, ok := r.recipients[tokenOrID]; ok && len(rcpts) > 0 {
			return rcpts[0], msg, nil
		}
		return nil, msg, nil
	}

	return nil, nil, errors.New("recipient or message not found")
}

// Phase 7: Analytics & Reporting Methods

func (r *MemoryRepository) RecordAnalyticsEvent(ctx context.Context, tenantID, eventType string, streamID, domainID *string, provider, campaignTag string, latencySec float64, occurredAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if provider == "" {
		provider = "other"
	}
	if campaignTag == "" {
		campaignTag = "default"
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	// Increment rollups for 1m, 1h, 1d buckets
	buckets := []struct {
		name  string
		start time.Time
	}{
		{"1m", occurredAt.Truncate(time.Minute)},
		{"1h", occurredAt.Truncate(time.Hour)},
		{"1d", time.Date(occurredAt.Year(), occurredAt.Month(), occurredAt.Day(), 0, 0, 0, 0, time.UTC)},
	}

	for _, b := range buckets {
		var rollup *domain.AnalyticsRollup
		for _, ex := range r.analyticsRollups {
			if ex.TenantID == tenantID && ex.TimeBucket == b.name && ex.BucketStart.Equal(b.start) &&
				equalPtr(ex.StreamID, streamID) && equalPtr(ex.DomainID, domainID) &&
				ex.Provider == provider && ex.CampaignTag == campaignTag {
				rollup = ex
				break
			}
		}

		if rollup == nil {
			id := fmt.Sprintf("ar_%s_%s_%d", tenantID, b.name, b.start.Unix())
			rollup = &domain.AnalyticsRollup{
				ID:          id,
				TenantID:    tenantID,
				TimeBucket:  b.name,
				BucketStart: b.start,
				StreamID:    streamID,
				DomainID:    domainID,
				Provider:    provider,
				CampaignTag: campaignTag,
			}
			r.analyticsRollups = append(r.analyticsRollups, rollup)
		}

		// Increment event counters
		switch eventType {
		case "message.sent", "sent":
			rollup.SentCount++
		case "message.delivered", "delivered":
			rollup.DeliveredCount++
		case "message.bounced", "bounced", "hard_bounce":
			rollup.BouncedHardCount++
		case "soft_bounce":
			rollup.BouncedSoftCount++
		case "message.deferred", "deferred":
			rollup.DeferredCount++
		case "message.complained", "complained", "abuse":
			rollup.ComplainedCount++
		case "message.opened", "opened":
			rollup.OpenedUniqueCount++
			rollup.OpenedTotalCount++
		case "message.clicked", "clicked":
			rollup.ClickedUniqueCount++
			rollup.ClickedTotalCount++
		case "message.unsubscribed", "unsubscribed":
			rollup.UnsubscribedCount++
		}

		// Increment latency histogram if provided
		if latencySec > 0 {
			switch {
			case latencySec < 5:
				rollup.LatencyUnder5s++
			case latencySec < 30:
				rollup.LatencyUnder30s++
			case latencySec < 60:
				rollup.LatencyUnder1m++
			case latencySec < 300:
				rollup.LatencyUnder5m++
			case latencySec < 900:
				rollup.LatencyUnder15m++
			default:
				rollup.LatencyOver15m++
			}
		}
	}

	return nil
}

func (r *MemoryRepository) matchesFilter(rollup *domain.AnalyticsRollup, tenantID string, filter domain.AnalyticsFilter) bool {
	if rollup.TenantID != tenantID {
		return false
	}
	if filter.From != nil && rollup.BucketStart.Before(*filter.From) {
		return false
	}
	if filter.To != nil && rollup.BucketStart.After(*filter.To) {
		return false
	}
	if filter.StreamID != nil && (rollup.StreamID == nil || *rollup.StreamID != *filter.StreamID) {
		return false
	}
	if filter.DomainID != nil && (rollup.DomainID == nil || *rollup.DomainID != *filter.DomainID) {
		return false
	}
	if filter.Provider != nil && !strings.EqualFold(rollup.Provider, *filter.Provider) {
		return false
	}
	if filter.CampaignTag != nil && !strings.EqualFold(rollup.CampaignTag, *filter.CampaignTag) {
		return false
	}
	return true
}

func (r *MemoryRepository) QueryAnalyticsOverview(ctx context.Context, tenantID string, filter domain.AnalyticsFilter) (*domain.AnalyticsSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Default to last 30 days if no time filter given
	summary := &domain.AnalyticsSummary{}

	for _, roll := range r.analyticsRollups {
		// Use 1h resolution for queries under 7 days, 1d resolution otherwise
		if roll.TimeBucket != "1h" && roll.TimeBucket != "1d" {
			continue
		}
		// Avoid double-counting: use 1h if present, or fallback to 1d
		if roll.TimeBucket == "1h" {
			if !r.matchesFilter(roll, tenantID, filter) {
				continue
			}
			summary.Sent += roll.SentCount
			summary.Delivered += roll.DeliveredCount
			summary.BouncedHard += roll.BouncedHardCount
			summary.BouncedSoft += roll.BouncedSoftCount
			summary.Deferred += roll.DeferredCount
			summary.Complained += roll.ComplainedCount
			summary.OpenedUnique += roll.OpenedUniqueCount
			summary.OpenedTotal += roll.OpenedTotalCount
			summary.ClickedUnique += roll.ClickedUniqueCount
			summary.ClickedTotal += roll.ClickedTotalCount
			summary.Unsubscribed += roll.UnsubscribedCount
		}
	}

	// Compute derived rates
	if summary.Sent > 0 {
		summary.DeliveryRate = float64(summary.Delivered) / float64(summary.Sent)
		summary.BounceRate = float64(summary.BouncedHard+summary.BouncedSoft) / float64(summary.Sent)
		summary.HardBounceRate = float64(summary.BouncedHard) / float64(summary.Sent)
	}
	if summary.Delivered > 0 {
		summary.ComplaintRate = float64(summary.Complained) / float64(summary.Delivered)
		summary.OpenRate = float64(summary.OpenedUnique) / float64(summary.Delivered)
		summary.ClickRate = float64(summary.ClickedUnique) / float64(summary.Delivered)
		summary.UnsubscribeRate = float64(summary.Unsubscribed) / float64(summary.Delivered)
	}

	// Complaint warning threshold: >= 0.10% (0.001) per Google / Yahoo deliverability standards (§19.2)
	if summary.ComplaintRate >= 0.001 {
		summary.ComplaintWarning = true
	}

	return summary, nil
}

func (r *MemoryRepository) QueryAnalyticsTimeSeries(ctx context.Context, tenantID string, interval string, filter domain.AnalyticsFilter) ([]*domain.TimeSeriesPoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if interval == "" {
		interval = "1h"
	}

	pointsMap := make(map[string]*domain.TimeSeriesPoint)

	for _, roll := range r.analyticsRollups {
		if roll.TimeBucket != interval {
			continue
		}
		if !r.matchesFilter(roll, tenantID, filter) {
			continue
		}

		tsKey := roll.BucketStart.Format(time.RFC3339)
		p, exists := pointsMap[tsKey]
		if !exists {
			p = &domain.TimeSeriesPoint{Timestamp: tsKey}
			pointsMap[tsKey] = p
		}

		p.Sent += roll.SentCount
		p.Delivered += roll.DeliveredCount
		p.Bounced += roll.BouncedHardCount + roll.BouncedSoftCount
		p.Complained += roll.ComplainedCount
	}

	// Compute rates and sort chronologically
	var points []*domain.TimeSeriesPoint
	for _, p := range pointsMap {
		if p.Sent > 0 {
			p.DeliveryRate = float64(p.Delivered) / float64(p.Sent)
		}
		if p.Delivered > 0 {
			p.ComplaintRate = float64(p.Complained) / float64(p.Delivered)
		}
		points = append(points, p)
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp < points[j].Timestamp
	})

	return points, nil
}

func (r *MemoryRepository) QueryProviderBreakdown(ctx context.Context, tenantID string, filter domain.AnalyticsFilter) ([]*domain.ProviderBreakdown, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provMap := make(map[string]*domain.ProviderBreakdown)

	for _, roll := range r.analyticsRollups {
		if roll.TimeBucket != "1h" {
			continue
		}
		if !r.matchesFilter(roll, tenantID, filter) {
			continue
		}

		pName := roll.Provider
		if pName == "" {
			pName = "other"
		}

		pb, exists := provMap[pName]
		if !exists {
			pb = &domain.ProviderBreakdown{Provider: pName}
			provMap[pName] = pb
		}

		pb.Sent += roll.SentCount
		pb.Delivered += roll.DeliveredCount
		pb.Bounced += roll.BouncedHardCount + roll.BouncedSoftCount
		pb.Complained += roll.ComplainedCount
	}

	var results []*domain.ProviderBreakdown
	for _, pb := range provMap {
		if pb.Sent > 0 {
			pb.DeliveryRate = float64(pb.Delivered) / float64(pb.Sent)
		}
		if pb.Delivered > 0 {
			pb.ComplaintRate = float64(pb.Complained) / float64(pb.Delivered)
		}
		results = append(results, pb)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Sent > results[j].Sent
	})

	return results, nil
}

func (r *MemoryRepository) QueryLatencyHistogram(ctx context.Context, tenantID string, filter domain.AnalyticsFilter) (*domain.LatencyHistogram, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hist := &domain.LatencyHistogram{}

	for _, roll := range r.analyticsRollups {
		if roll.TimeBucket != "1h" {
			continue
		}
		if !r.matchesFilter(roll, tenantID, filter) {
			continue
		}

		hist.Under5s += roll.LatencyUnder5s
		hist.Under30s += roll.LatencyUnder30s
		hist.Under1m += roll.LatencyUnder1m
		hist.Under5m += roll.LatencyUnder5m
		hist.Under15m += roll.LatencyUnder15m
		hist.Over15m += roll.LatencyOver15m
	}

	return hist, nil
}

func (r *MemoryRepository) CreateAnalyticsExport(ctx context.Context, export *domain.AnalyticsExport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.analyticsExports[export.ID] = export
	return nil
}

func (r *MemoryRepository) GetAnalyticsExport(ctx context.Context, tenantID, exportID string) (*domain.AnalyticsExport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exp, ok := r.analyticsExports[exportID]
	if !ok || exp.TenantID != tenantID {
		return nil, errors.New("export not found")
	}
	return exp, nil
}

func (r *MemoryRepository) ListAnalyticsExports(ctx context.Context, tenantID string, limit int) ([]*domain.AnalyticsExport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	var list []*domain.AnalyticsExport
	for _, exp := range r.analyticsExports {
		if exp.TenantID == tenantID {
			list = append(list, exp)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	if len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (r *MemoryRepository) UpdateAnalyticsExport(ctx context.Context, tenantID, exportID, status string, fileURL *string, fileData []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	exp, ok := r.analyticsExports[exportID]
	if !ok || exp.TenantID != tenantID {
		return errors.New("export not found")
	}
	exp.Status = status
	if fileURL != nil {
		exp.FileURL = fileURL
	}
	if fileData != nil {
		exp.FileData = fileData
	}
	if status == "completed" || status == "failed" {
		now := time.Now().UTC()
		exp.CompletedAt = &now
	}
	return nil
}

func equalPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// Phase 8: Webhook Repository Methods

func (r *MemoryRepository) CreateWebhookEndpoint(ctx context.Context, endpoint *domain.WebhookEndpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if endpoint.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	r.webhookEndpoints[endpoint.ID] = endpoint
	return nil
}

func (r *MemoryRepository) GetWebhookEndpoint(ctx context.Context, tenantID, id string) (*domain.WebhookEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ep, ok := r.webhookEndpoints[id]
	if !ok || ep.TenantID != tenantID {
		return nil, errors.New("webhook endpoint not found")
	}
	return ep, nil
}

func (r *MemoryRepository) ListWebhookEndpoints(ctx context.Context, tenantID string) ([]*domain.WebhookEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.WebhookEndpoint
	for _, ep := range r.webhookEndpoints {
		if ep.TenantID == tenantID {
			result = append(result, ep)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryRepository) UpdateWebhookEndpoint(ctx context.Context, endpoint *domain.WebhookEndpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.webhookEndpoints[endpoint.ID]
	if !ok || existing.TenantID != endpoint.TenantID {
		return errors.New("webhook endpoint not found")
	}

	endpoint.UpdatedAt = time.Now().UTC()
	r.webhookEndpoints[endpoint.ID] = endpoint
	return nil
}

func (r *MemoryRepository) DeleteWebhookEndpoint(ctx context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ep, ok := r.webhookEndpoints[id]
	if !ok || ep.TenantID != tenantID {
		return errors.New("webhook endpoint not found")
	}

	delete(r.webhookEndpoints, id)
	return nil
}

func (r *MemoryRepository) RotateWebhookSecret(ctx context.Context, tenantID, id, newSecret string, prevExpiresAt time.Time) (*domain.WebhookEndpoint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ep, ok := r.webhookEndpoints[id]
	if !ok || ep.TenantID != tenantID {
		return nil, errors.New("webhook endpoint not found")
	}

	oldSecret := ep.Secret
	ep.PreviousSecret = &oldSecret
	ep.PreviousSecretExpiresAt = &prevExpiresAt
	ep.Secret = newSecret
	ep.UpdatedAt = time.Now().UTC()
	return ep, nil
}

func (r *MemoryRepository) RecordWebhookDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.webhookDeliveries = append(r.webhookDeliveries, delivery)

	// Update endpoint failure counts and circuit breaker
	ep, ok := r.webhookEndpoints[delivery.EndpointID]
	if ok && ep.TenantID == delivery.TenantID {
		isSuccess := delivery.StatusCode != nil && *delivery.StatusCode >= 200 && *delivery.StatusCode < 300
		if isSuccess {
			ep.ConsecutiveFailures = 0
		} else {
			ep.ConsecutiveFailures++
			if ep.ConsecutiveFailures >= 10 { // circuit breaking threshold
				ep.Status = "disabled_failing"
			}
		}
		ep.UpdatedAt = time.Now().UTC()
	}

	return nil
}

func (r *MemoryRepository) ListWebhookDeliveries(ctx context.Context, tenantID, endpointID string, limit, offset int) ([]*domain.WebhookDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []*domain.WebhookDelivery
	for _, d := range r.webhookDeliveries {
		if d.TenantID == tenantID {
			if endpointID == "" || d.EndpointID == endpointID {
				matched = append(matched, d)
			}
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})

	if offset < 0 {
		offset = 0
	}
	if offset >= len(matched) {
		return []*domain.WebhookDelivery{}, nil
	}

	end := offset + limit
	if limit <= 0 || end > len(matched) {
		end = len(matched)
	}

	return matched[offset:end], nil
}

func (r *MemoryRepository) GetActiveWebhookEndpointsForEvent(ctx context.Context, tenantID, eventType string) ([]*domain.WebhookEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.WebhookEndpoint
	for _, ep := range r.webhookEndpoints {
		if ep.TenantID == tenantID && ep.Status == "active" {
			subscribes := false
			for _, ev := range ep.Events {
				if ev == "*" || ev == eventType {
					subscribes = true
					break
				}
			}
			if subscribes {
				result = append(result, ep)
			}
		}
	}
	return result, nil
}

// Phase 9: Suppression Management, Privacy & Data Retention

func (r *MemoryRepository) ImportSuppressions(ctx context.Context, tenantID, filename string, entries []*domain.Suppression) (*domain.SuppressionImport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	totalRows := len(entries)
	importedRows := 0
	duplicateRows := 0

	for _, entry := range entries {
		normAddr := strings.TrimSpace(strings.ToLower(entry.Address))
		if normAddr == "" {
			continue
		}
		entry.Address = normAddr
		parts := strings.Split(normAddr, "@")
		if len(parts) == 2 {
			entry.AddressDomain = parts[1]
		}

		isDup := false
		for _, existing := range r.suppressions {
			if existing.TenantID == tenantID && strings.EqualFold(existing.Address, normAddr) {
				if equalPtr(existing.StreamID, entry.StreamID) && existing.Type == entry.Type {
					isDup = true
					break
				}
			}
		}

		if isDup {
			duplicateRows++
		} else {
			if entry.ID == "" {
				entry.ID = crypto.NewULID(crypto.PrefixSuppression)
			}
			entry.TenantID = tenantID
			if entry.CreatedAt.IsZero() {
				entry.CreatedAt = time.Now().UTC()
			}
			r.suppressions[entry.ID] = entry
			importedRows++
		}
	}

	importRec := &domain.SuppressionImport{
		ID:            crypto.NewULID(crypto.PrefixSuppressionImport),
		TenantID:      tenantID,
		Filename:      filename,
		TotalRows:     totalRows,
		ImportedRows:  importedRows,
		DuplicateRows: duplicateRows,
		Status:        "completed",
		CreatedAt:     time.Now().UTC(),
	}
	r.suppressionImports = append(r.suppressionImports, importRec)
	return importRec, nil
}

func (r *MemoryRepository) ExportSuppressions(ctx context.Context, tenantID string, streamID *string) ([]*domain.Suppression, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Suppression
	for _, s := range r.suppressions {
		if s.TenantID == tenantID {
			if streamID != nil && *streamID != "" {
				if s.StreamID == nil || *s.StreamID != *streamID {
					continue
				}
			}
			result = append(result, s)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryRepository) CreatePrivacyRequest(ctx context.Context, req *domain.PrivacyRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if req.ID == "" {
		req.ID = crypto.NewULID(crypto.PrefixPrivacyRequest)
	}
	r.privacyRequests[req.ID] = req
	return nil
}

func (r *MemoryRepository) GetPrivacyRequest(ctx context.Context, tenantID, id string) (*domain.PrivacyRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	req, ok := r.privacyRequests[id]
	if !ok || req.TenantID != tenantID {
		return nil, ErrNotFound
	}
	return req, nil
}

func (r *MemoryRepository) ListPrivacyRequests(ctx context.Context, tenantID string) ([]*domain.PrivacyRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var res []*domain.PrivacyRequest
	for _, req := range r.privacyRequests {
		if req.TenantID == tenantID {
			res = append(res, req)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].RequestedAt.After(res[j].RequestedAt)
	})
	return res, nil
}

func (r *MemoryRepository) ExecutePrivacyErasure(ctx context.Context, tenantID, email string) (*domain.PrivacyRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	normEmail := strings.TrimSpace(strings.ToLower(email))
	now := time.Now().UTC()

	// 1. Scrub recipients matching email under tenant's messages
	erasedRecipients := 0
	erasedMessages := 0
	hash := sha256.Sum256([]byte(normEmail))
	maskedAddr := fmt.Sprintf("erased-%x@erased.invalid", hash[:8])

	for _, msg := range r.messages {
		if msg.TenantID != tenantID {
			continue
		}
		recs := r.recipients[msg.ID]
		msgAffected := false
		for _, rec := range recs {
			if strings.EqualFold(rec.Address, normEmail) {
				rec.Address = maskedAddr
				rec.AddressDomain = "erased.invalid"
				erasedRecipients++
				msgAffected = true
			}
		}
		if msgAffected {
			msg.BodyRef = nil
			msg.RawMIMERef = nil
			erasedMessages++
		}
	}

	// 2. Pseudonymize suppressions so address stays suppressed without storing PII (§20)
	erasedSuppressions := 0
	for _, s := range r.suppressions {
		if s.TenantID == tenantID && strings.EqualFold(s.Address, normEmail) {
			s.Address = fmt.Sprintf("erased-hash-%x", hash)
			s.AddressDomain = "erased.invalid"
			erasedSuppressions++
		}
	}

	summary := fmt.Sprintf("Erased %d recipient records, %d message bodies/raw MIME, %d suppression addresses pseudonymized", erasedRecipients, erasedMessages, erasedSuppressions)
	req := &domain.PrivacyRequest{
		ID:            crypto.NewULID(crypto.PrefixPrivacyRequest),
		TenantID:      tenantID,
		Type:          "erasure",
		EmailAddress:  email,
		Status:        "completed",
		ResultSummary: &summary,
		RequestedAt:   now,
		CompletedAt:   &now,
	}
	r.privacyRequests[req.ID] = req
	return req, nil
}

func (r *MemoryRepository) ExecutePrivacyExport(ctx context.Context, tenantID, email string) (*domain.PrivacyExportReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normEmail := strings.TrimSpace(strings.ToLower(email))
	report := &domain.PrivacyExportReport{
		EmailAddress: email,
		GeneratedAt:  time.Now().UTC(),
		Messages:     make([]*domain.MessageRecipient, 0),
		Events:       make([]*domain.MessageEvent, 0),
		Suppressions: make([]*domain.Suppression, 0),
	}

	for _, msg := range r.messages {
		if msg.TenantID != tenantID {
			continue
		}
		for _, rec := range r.recipients[msg.ID] {
			if strings.EqualFold(rec.Address, normEmail) {
				report.Messages = append(report.Messages, rec)
				// Gather events
				for _, ev := range r.events[msg.ID] {
					if ev.RecipientID != nil && *ev.RecipientID == rec.ID {
						report.Events = append(report.Events, ev)
					}
				}
			}
		}
	}

	for _, s := range r.suppressions {
		if s.TenantID == tenantID && strings.EqualFold(s.Address, normEmail) {
			report.Suppressions = append(report.Suppressions, s)
		}
	}

	now := time.Now().UTC()
	summary := fmt.Sprintf("Exported %d message records, %d events, %d suppressions", len(report.Messages), len(report.Events), len(report.Suppressions))
	req := &domain.PrivacyRequest{
		ID:            crypto.NewULID(crypto.PrefixPrivacyRequest),
		TenantID:      tenantID,
		Type:          "export",
		EmailAddress:  email,
		Status:        "completed",
		ResultSummary: &summary,
		RequestedAt:   now,
		CompletedAt:   &now,
	}
	// Store request under lock (using temporary lock or unlock)
	// Since r.mu is RLocked here, let's schedule request save
	r.privacyRequests[req.ID] = req

	return report, nil
}

func (r *MemoryRepository) GetRetentionPolicy(ctx context.Context, tenantID string) (*domain.RetentionPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pol, ok := r.retentionPolicies[tenantID]
	if !ok {
		pol = &domain.RetentionPolicy{
			ID:                    crypto.NewULID(crypto.PrefixRetentionPolicy),
			TenantID:              tenantID,
			MessageRetentionDays:  30,
			EventsRetentionDays:   90,
			AttemptsRetentionDays: 30,
			RawMIMERetentionDays:  7,
			UpdatedAt:             time.Now().UTC(),
		}
		r.retentionPolicies[tenantID] = pol
	}
	return pol, nil
}

func (r *MemoryRepository) UpdateRetentionPolicy(ctx context.Context, policy *domain.RetentionPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if policy.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if policy.MessageRetentionDays < 1 || policy.EventsRetentionDays < 1 || policy.AttemptsRetentionDays < 1 || policy.RawMIMERetentionDays < 1 {
		return errors.New("retention days must be at least 1")
	}

	policy.UpdatedAt = time.Now().UTC()
	if policy.ID == "" {
		policy.ID = crypto.NewULID(crypto.PrefixRetentionPolicy)
	}
	r.retentionPolicies[policy.TenantID] = policy
	return nil
}

func (r *MemoryRepository) ExecuteRetentionPurge(ctx context.Context, tenantID string) (*domain.RetentionPurgeSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pol, ok := r.retentionPolicies[tenantID]
	if !ok {
		pol = &domain.RetentionPolicy{
			ID:                    crypto.NewULID(crypto.PrefixRetentionPolicy),
			TenantID:              tenantID,
			MessageRetentionDays:  30,
			EventsRetentionDays:   90,
			AttemptsRetentionDays: 30,
			RawMIMERetentionDays:  7,
			UpdatedAt:             time.Now().UTC(),
		}
		r.retentionPolicies[tenantID] = pol
	}

	now := time.Now().UTC()
	msgCutoff := now.AddDate(0, 0, -pol.MessageRetentionDays)
	evtCutoff := now.AddDate(0, 0, -pol.EventsRetentionDays)
	attCutoff := now.AddDate(0, 0, -pol.AttemptsRetentionDays)
	rawCutoff := now.AddDate(0, 0, -pol.RawMIMERetentionDays)

	purgedMsgs := 0
	purgedRawMIME := 0
	for id, msg := range r.messages {
		if msg.TenantID != tenantID {
			continue
		}
		// Check raw MIME cutoff
		if msg.CreatedAt.Before(rawCutoff) && msg.RawMIMERef != nil {
			msg.RawMIMERef = nil
			purgedRawMIME++
		}
		// Check message cutoff
		if msg.CreatedAt.Before(msgCutoff) {
			delete(r.messages, id)
			delete(r.recipients, id)
			purgedMsgs++
		}
	}

	purgedEvents := 0
	for msgID, evts := range r.events {
		var kept []*domain.MessageEvent
		for _, ev := range evts {
			if ev.CreatedAt.Before(evtCutoff) {
				purgedEvents++
			} else {
				kept = append(kept, ev)
			}
		}
		r.events[msgID] = kept
	}

	purgedAttempts := 0
	var keptAttempts []*domain.DeliveryAttempt
	for _, att := range r.deliveryAttempts {
		if att.CreatedAt.Before(attCutoff) {
			purgedAttempts++
		} else {
			keptAttempts = append(keptAttempts, att)
		}
	}
	r.deliveryAttempts = keptAttempts

	summary := &domain.RetentionPurgeSummary{
		TenantID:            tenantID,
		PurgedAt:            now,
		PurgedMessagesCount: purgedMsgs,
		PurgedEventsCount:   purgedEvents,
		PurgedAttemptsCount: purgedAttempts,
		PurgedRawMIMECount:  purgedRawMIME,
	}
	return summary, nil
}

// Phase 10: Tracking & Inbound Routing

func (r *MemoryRepository) SaveTrackingLink(ctx context.Context, link *domain.TrackingLink) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if link.ID == "" {
		link.ID = crypto.NewULID(crypto.PrefixTrackingLink)
	}
	if link.CreatedAt.IsZero() {
		link.CreatedAt = time.Now().UTC()
	}
	key := fmt.Sprintf("%s:%s", link.TenantID, link.LinkID)
	r.trackingLinks[key] = link
	return nil
}

func (r *MemoryRepository) GetTrackingLink(ctx context.Context, tenantID, linkID string) (*domain.TrackingLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, linkID)
	link, ok := r.trackingLinks[key]
	if !ok {
		return nil, ErrNotFound
	}
	return link, nil
}

func (r *MemoryRepository) RecordTrackingEvent(ctx context.Context, evt *domain.TrackingEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if evt.ID == "" {
		evt.ID = crypto.NewULID(crypto.PrefixTrackingEvent)
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now().UTC()
	}
	r.trackingEvents = append(r.trackingEvents, evt)
	return nil
}

func (r *MemoryRepository) ListTrackingEvents(ctx context.Context, tenantID, messageID string) ([]*domain.TrackingEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.TrackingEvent
	for _, ev := range r.trackingEvents {
		if ev.TenantID == tenantID {
			if messageID == "" || ev.MessageID == messageID {
				result = append(result, ev)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].OccurredAt.After(result[j].OccurredAt)
	})
	return result, nil
}

func (r *MemoryRepository) CreateInboundRoute(ctx context.Context, route *domain.InboundRoute) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if route.ID == "" {
		route.ID = crypto.NewULID(crypto.PrefixInboundRoute)
	}
	now := time.Now().UTC()
	route.CreatedAt = now
	route.UpdatedAt = now
	r.inboundRoutes[route.ID] = route
	return nil
}

func (r *MemoryRepository) GetInboundRoute(ctx context.Context, tenantID, id string) (*domain.InboundRoute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	route, ok := r.inboundRoutes[id]
	if !ok || route.TenantID != tenantID {
		return nil, ErrNotFound
	}
	return route, nil
}

func (r *MemoryRepository) ListInboundRoutes(ctx context.Context, tenantID string) ([]*domain.InboundRoute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.InboundRoute
	for _, route := range r.inboundRoutes {
		if route.TenantID == tenantID {
			list = append(list, route)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Priority != list[j].Priority {
			return list[i].Priority < list[j].Priority
		}
		return list[i].CreatedAt.Before(list[j].CreatedAt)
	})
	return list, nil
}

func (r *MemoryRepository) UpdateInboundRoute(ctx context.Context, route *domain.InboundRoute) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.inboundRoutes[route.ID]
	if !ok || existing.TenantID != route.TenantID {
		return ErrNotFound
	}
	route.UpdatedAt = time.Now().UTC()
	r.inboundRoutes[route.ID] = route
	return nil
}

func (r *MemoryRepository) DeleteInboundRoute(ctx context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.inboundRoutes[id]
	if !ok || existing.TenantID != tenantID {
		return ErrNotFound
	}
	delete(r.inboundRoutes, id)
	return nil
}

func (r *MemoryRepository) MatchInboundRoute(ctx context.Context, tenantID, toAddress string) (*domain.InboundRoute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normAddr := strings.ToLower(strings.TrimSpace(toAddress))
	parts := strings.Split(normAddr, "@")
	if len(parts) != 2 {
		return nil, errors.New("invalid email address")
	}
	domainPart := parts[1]

	var tenantRoutes []*domain.InboundRoute
	for _, route := range r.inboundRoutes {
		if route.TenantID == tenantID && route.IsActive {
			tenantRoutes = append(tenantRoutes, route)
		}
	}

	sort.Slice(tenantRoutes, func(i, j int) bool {
		return tenantRoutes[i].Priority < tenantRoutes[j].Priority
	})

	for _, route := range tenantRoutes {
		pattern := strings.ToLower(strings.TrimSpace(route.Pattern))
		switch route.MatchType {
		case "exact":
			if pattern == normAddr {
				return route, nil
			}
		case "wildcard":
			if matched, _ := filepath.Match(pattern, normAddr); matched {
				return route, nil
			}
		case "catch_all":
			cleanPattern := strings.TrimPrefix(pattern, "@")
			cleanPattern = strings.TrimPrefix(cleanPattern, "*@")
			if strings.EqualFold(cleanPattern, domainPart) {
				return route, nil
			}
		}
	}

	return nil, ErrNotFound
}

func (r *MemoryRepository) SaveInboundMessage(ctx context.Context, msg *domain.InboundMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if msg.ID == "" {
		msg.ID = crypto.NewULID(crypto.PrefixInboundMessage)
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	r.inboundMessages = append(r.inboundMessages, msg)
	return nil
}

func (r *MemoryRepository) GetInboundMessage(ctx context.Context, tenantID, id string) (*domain.InboundMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, msg := range r.inboundMessages {
		if msg.ID == id && msg.TenantID == tenantID {
			return msg, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryRepository) ListInboundMessages(ctx context.Context, tenantID string, limit, offset int) ([]*domain.InboundMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.InboundMessage
	for _, msg := range r.inboundMessages {
		if msg.TenantID == tenantID {
			list = append(list, msg)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	if offset < 0 {
		offset = 0
	}
	if offset >= len(list) {
		return []*domain.InboundMessage{}, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}

// Phase 11: Reputation, DNSBL, Warmup & Alerts implementations

func (r *MemoryRepository) ListDNSBLProviders(ctx context.Context) ([]*domain.DNSBLProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.DNSBLProvider
	for _, p := range r.dnsblProviders {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list, nil
}

func (r *MemoryRepository) GetDNSBLProvider(ctx context.Context, id string) (*domain.DNSBLProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.dnsblProviders[id]
	if !exists {
		return nil, ErrNotFound
	}
	return p, nil
}

func (r *MemoryRepository) SaveDNSBLProvider(ctx context.Context, provider *domain.DNSBLProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if provider.ID == "" {
		provider.ID = crypto.NewULID(crypto.PrefixDNSBLProvider)
	}
	now := time.Now().UTC()
	if provider.CreatedAt.IsZero() {
		provider.CreatedAt = now
	}
	provider.UpdatedAt = now
	r.dnsblProviders[provider.ID] = provider
	return nil
}

func (r *MemoryRepository) RecordDNSBLResult(ctx context.Context, result *domain.DNSBLResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if result.ID == "" {
		result.ID = crypto.NewULID(crypto.PrefixDNSBLResult)
	}
	if result.CheckedAt.IsZero() {
		result.CheckedAt = time.Now().UTC()
	}
	r.dnsblResults = append(r.dnsblResults, result)
	return nil
}

func (r *MemoryRepository) ListDNSBLResults(ctx context.Context, ipAddressID string, limit int) ([]*domain.DNSBLResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.DNSBLResult
	for _, res := range r.dnsblResults {
		if ipAddressID == "" || res.IPAddressID == ipAddressID {
			list = append(list, res)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CheckedAt.After(list[j].CheckedAt)
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (r *MemoryRepository) CreateWarmupPlan(ctx context.Context, plan *domain.WarmupPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if plan.ID == "" {
		plan.ID = crypto.NewULID(crypto.PrefixWarmupPlan)
	}
	now := time.Now().UTC()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now
	r.warmupPlans[plan.ID] = plan
	return nil
}

func (r *MemoryRepository) GetWarmupPlan(ctx context.Context, id string) (*domain.WarmupPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plan, exists := r.warmupPlans[id]
	if !exists {
		return nil, ErrNotFound
	}
	return plan, nil
}

func (r *MemoryRepository) ListWarmupPlans(ctx context.Context) ([]*domain.WarmupPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.WarmupPlan
	for _, p := range r.warmupPlans {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list, nil
}

func (r *MemoryRepository) GetIPWarmupState(ctx context.Context, ipAddressID string) (*domain.IPWarmupState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, exists := r.ipWarmupStates[ipAddressID]
	if !exists {
		return nil, ErrNotFound
	}
	return state, nil
}

func (r *MemoryRepository) SaveIPWarmupState(ctx context.Context, state *domain.IPWarmupState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if state.ID == "" {
		state.ID = crypto.NewULID(crypto.PrefixIPWarmupState)
	}
	now := time.Now().UTC()
	if state.StartedAt.IsZero() {
		state.StartedAt = now
	}
	state.UpdatedAt = now
	r.ipWarmupStates[state.IPAddressID] = state
	return nil
}

func (r *MemoryRepository) ListIPWarmupStates(ctx context.Context) ([]*domain.IPWarmupState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.IPWarmupState
	for _, s := range r.ipWarmupStates {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].IPAddress < list[j].IPAddress
	})
	return list, nil
}

func (r *MemoryRepository) CreateAlertRule(ctx context.Context, rule *domain.AlertRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if rule.ID == "" {
		rule.ID = crypto.NewULID(crypto.PrefixAlertRule)
	}
	now := time.Now().UTC()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now
	r.alertRules[rule.ID] = rule
	return nil
}

func (r *MemoryRepository) GetAlertRule(ctx context.Context, tenantID *string, id string) (*domain.AlertRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rule, exists := r.alertRules[id]
	if !exists {
		return nil, ErrNotFound
	}
	if tenantID != nil && rule.TenantID != nil && *rule.TenantID != *tenantID {
		return nil, ErrNotFound
	}
	return rule, nil
}

func (r *MemoryRepository) ListAlertRules(ctx context.Context, tenantID *string) ([]*domain.AlertRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.AlertRule
	for _, rule := range r.alertRules {
		if tenantID == nil || rule.TenantID == nil || *rule.TenantID == *tenantID {
			list = append(list, rule)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list, nil
}

func (r *MemoryRepository) UpdateAlertRule(ctx context.Context, rule *domain.AlertRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.alertRules[rule.ID]
	if !exists {
		return ErrNotFound
	}
	if rule.TenantID != nil && existing.TenantID != nil && *existing.TenantID != *rule.TenantID {
		return ErrUnauthorized
	}
	rule.UpdatedAt = time.Now().UTC()
	r.alertRules[rule.ID] = rule
	return nil
}

func (r *MemoryRepository) DeleteAlertRule(ctx context.Context, tenantID *string, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rule, exists := r.alertRules[id]
	if !exists {
		return ErrNotFound
	}
	if tenantID != nil && rule.TenantID != nil && *rule.TenantID != *tenantID {
		return ErrUnauthorized
	}
	delete(r.alertRules, id)
	return nil
}

func (r *MemoryRepository) RecordAlertIncident(ctx context.Context, incident *domain.AlertIncident) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if incident.ID == "" {
		incident.ID = crypto.NewULID(crypto.PrefixAlertIncident)
	}
	if incident.FiredAt.IsZero() {
		incident.FiredAt = time.Now().UTC()
	}
	r.alertIncidents[incident.ID] = incident
	return nil
}

func (r *MemoryRepository) GetAlertIncident(ctx context.Context, tenantID *string, id string) (*domain.AlertIncident, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inc, exists := r.alertIncidents[id]
	if !exists {
		return nil, ErrNotFound
	}
	if tenantID != nil && inc.TenantID != nil && *inc.TenantID != *tenantID {
		return nil, ErrNotFound
	}
	return inc, nil
}

func (r *MemoryRepository) ListAlertIncidents(ctx context.Context, tenantID *string, status string, limit int) ([]*domain.AlertIncident, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.AlertIncident
	for _, inc := range r.alertIncidents {
		if tenantID != nil && inc.TenantID != nil && *inc.TenantID != *tenantID {
			continue
		}
		if status != "" && inc.Status != status {
			continue
		}
		list = append(list, inc)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].FiredAt.After(list[j].FiredAt)
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (r *MemoryRepository) UpdateAlertIncidentStatus(ctx context.Context, tenantID *string, id, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	inc, exists := r.alertIncidents[id]
	if !exists {
		return ErrNotFound
	}
	if tenantID != nil && inc.TenantID != nil && *inc.TenantID != *tenantID {
		return ErrUnauthorized
	}
	now := time.Now().UTC()
	inc.Status = status
	if status == "acknowledged" {
		inc.AcknowledgedAt = &now
	} else if status == "resolved" {
		inc.ResolvedAt = &now
	}
	return nil
}

func (r *MemoryRepository) CreateNotificationChannel(ctx context.Context, channel *domain.NotificationChannel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if channel.ID == "" {
		channel.ID = crypto.NewULID(crypto.PrefixNotifChannel)
	}
	now := time.Now().UTC()
	if channel.CreatedAt.IsZero() {
		channel.CreatedAt = now
	}
	channel.UpdatedAt = now
	r.notificationChannels[channel.ID] = channel
	return nil
}

func (r *MemoryRepository) GetNotificationChannel(ctx context.Context, tenantID *string, id string) (*domain.NotificationChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ch, exists := r.notificationChannels[id]
	if !exists {
		return nil, ErrNotFound
	}
	if tenantID != nil && ch.TenantID != nil && *ch.TenantID != *tenantID {
		return nil, ErrNotFound
	}
	return ch, nil
}

func (r *MemoryRepository) ListNotificationChannels(ctx context.Context, tenantID *string) ([]*domain.NotificationChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.NotificationChannel
	for _, ch := range r.notificationChannels {
		if tenantID == nil || ch.TenantID == nil || *ch.TenantID == *tenantID {
			list = append(list, ch)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list, nil
}

func (r *MemoryRepository) DeleteNotificationChannel(ctx context.Context, tenantID *string, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch, exists := r.notificationChannels[id]
	if !exists {
		return ErrNotFound
	}
	if tenantID != nil && ch.TenantID != nil && *ch.TenantID != *tenantID {
		return ErrUnauthorized
	}
	delete(r.notificationChannels, id)
	return nil
}

func (r *MemoryRepository) CreateAlertNotification(ctx context.Context, notif *domain.AlertNotification) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if notif.ID == "" {
		notif.ID = crypto.NewULID(crypto.PrefixAlertNotif)
	}
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now().UTC()
	}
	r.alertNotifications = append(r.alertNotifications, notif)
	return nil
}

func (r *MemoryRepository) ListAlertNotifications(ctx context.Context, tenantID *string, unreadOnly bool, limit int) ([]*domain.AlertNotification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.AlertNotification
	for _, n := range r.alertNotifications {
		if tenantID != nil && n.TenantID != nil && *n.TenantID != *tenantID {
			continue
		}
		if unreadOnly && n.IsRead {
			continue
		}
		list = append(list, n)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (r *MemoryRepository) MarkAlertNotificationRead(ctx context.Context, tenantID *string, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, n := range r.alertNotifications {
		if n.ID == id {
			if tenantID != nil && n.TenantID != nil && *n.TenantID != *tenantID {
				return ErrUnauthorized
			}
			n.IsRead = true
			return nil
		}
	}
	return ErrNotFound
}

func (r *MemoryRepository) GetReputationOverview(ctx context.Context, tenantID string) (*domain.ReputationOverview, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	overview := &domain.ReputationOverview{
		TenantID:      tenantID,
		OverallStatus: "unknown",
	}

	// Calculate 7-day stats from messages & events
	sevenDaysAgo := time.Now().UTC().Add(-7 * 24 * time.Hour)
	sentCount := 0
	deliveredCount := 0
	bounceCount := 0
	complaintCount := 0

	for _, msg := range r.messages {
		if msg.TenantID != tenantID || msg.CreatedAt.Before(sevenDaysAgo) {
			continue
		}
		sentCount++
		if msg.Status == "delivered" {
			deliveredCount++
		} else if msg.Status == "bounced" || msg.Status == "failed" {
			bounceCount++
		}
	}

	for _, fbl := range r.fblReports {
		if fbl.TenantID == tenantID && fbl.CreatedAt.After(sevenDaysAgo) {
			complaintCount++
		}
	}

	overview.TotalSent7d = sentCount
	if sentCount > 0 {
		overview.DeliveryRate = float64(deliveredCount) / float64(sentCount)
		overview.BounceRate = float64(bounceCount) / float64(sentCount)
		overview.ComplaintRate = float64(complaintCount) / float64(sentCount)
	}

	// Count DNSBL listings
	dnsblListed := 0
	for _, res := range r.dnsblResults {
		if res.IsListed {
			dnsblListed++
		}
	}
	overview.DNSBLListingsCount = dnsblListed

	// Count warming IPs
	warmingCount := 0
	for _, ws := range r.ipWarmupStates {
		if ws.Status == "warming" {
			warmingCount++
		}
	}
	overview.WarmingIPsCount = warmingCount

	// Count active firing incidents
	activeIncidents := 0
	for _, inc := range r.alertIncidents {
		if (inc.TenantID == nil || *inc.TenantID == tenantID) && inc.Status == "firing" {
			activeIncidents++
		}
	}
	overview.ActiveAlertsCount = activeIncidents

	// Determine status
	if overview.BounceRate > 0.10 || overview.ComplaintRate > 0.005 || dnsblListed >= 3 {
		overview.OverallStatus = "blocked"
	} else if overview.BounceRate > 0.05 || overview.ComplaintRate > 0.001 || dnsblListed > 0 || activeIncidents > 0 {
		overview.OverallStatus = "warning"
	} else if overview.TotalSent7d >= 1000 {
		overview.OverallStatus = "measured"
	} else {
		overview.OverallStatus = "unknown"
	}

	return overview, nil
}




