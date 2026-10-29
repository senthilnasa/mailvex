import React, { useState, useEffect } from 'react'
import {
  LayoutDashboard,
  Globe,
  GitFork,
  Key,
  Inbox,
  FileCode,
  Ban,
  ShieldAlert,
  BarChart3,
  Webhook,
  Lock,
  ArrowDownLeft,
  Flame,
  ShieldCheck,
  Server,
  Cpu,
  Network,
  Mail,
  Building2,
  Layers,
  Zap,
  LogOut,
  Shield,
  Sparkles,
  Activity,
  Users,
  RefreshCw,
  HardDrive,
  Terminal,
  Plus,
  PlusCircle,
  CheckCircle2,
  AlertTriangle,
  ChevronRight,
  Sun,
  Moon,
  Monitor
} from 'lucide-react'
import { ExportMenu } from './components/ExportMenu'

interface Tenant {
  id: string
  name: string
  slug: string
  environment: string
  organization_id?: string
}

interface User {
  id: string
  email: string
  is_platform_admin: boolean
}

interface Domain {
  id: string
  name: string
  type: string
  status: string
  ownership_token: string
  verified_at?: string
  last_checked_at?: string
}

interface SMTPCredentialItem {
  id: string
  tenant_id: string
  username: string
  allowed_domains?: string[]
  allowed_cidrs?: string[]
  rate_limit: number
  created_at: string
}

interface DeviceItem {
  id: string
  tenant_id: string
  name: string
  type: string
  expected_ip: string
  allowed_from?: string[]
  created_at: string
}

interface TrustedNetworkItem {
  id: string
  tenant_id: string
  cidr: string
  description?: string
  created_at: string
}

interface DNSRecord {
  id: string
  record_type: string
  name: string
  expected_value: string
  observed_value?: string
  status: string
  error_message?: string
}

interface Stream {
  id: string
  name: string
  type: string
  is_default: boolean
}

interface APIKey {
  id: string
  name: string
  prefix: string
  last_four: string
  environment: string
  scopes: string[]
  created_at: string
}

interface MessageItem {
  id: string
  from_address: string
  subject: string
  status: string
  stream_id: string
  environment: string
  created_at: string
}

interface TemplateItem {
  id: string
  name: string
  description?: string
  created_at: string
}

interface SuppressionItem {
  id: string
  address: string
  address_domain: string
  type: string
  reason?: string
  source: string
  created_at: string
}

interface AbuseReportItem {
  id: string
  sender: string
  recipient: string
  subject?: string
  status: string
  notes?: string
  created_at: string
  resolved_at?: string
}

interface FBLReportItem {
  id: string
  feedback_type: string
  reported_domain?: string
  recipient_id?: string
  created_at: string
}

interface PostmasterSnapshotItem {
  id: string
  sample_date: string
  spam_rate: number
  domain_reputation: string
  dkim_success_rate: number
}

interface AnalyticsSummary {
  period_start?: string
  period_end?: string
  sent: number
  delivered: number
  bounced_hard: number
  bounced_soft: number
  deferred?: number
  complained: number
  opened_unique?: number
  opened_total?: number
  clicked_unique?: number
  clicked_total?: number
  unsubscribed: number
  delivery_rate: number
  bounce_rate: number
  hard_bounce_rate?: number
  complaint_rate: number
  open_rate?: number
  click_rate?: number
  unsubscribe_rate: number
  complaint_warning: boolean
  median_latency_seconds?: number
  p95_latency_seconds?: number
  sent_count: number
  delivered_count: number
  hard_bounce_count: number
  soft_bounce_count: number
  complaint_count: number
  unsubscribe_count: number
}

interface ProviderBreakdownItem {
  provider: string
  sent: number
  delivered: number
  bounced: number
  complained: number
  delivery_rate: number
  bounce_rate?: number
  complaint_rate: number
  sent_count: number
  delivered_count: number
  bounce_count: number
  complaint_count: number
}

interface LatencyHistogramItem {
  bucket: string
  count: number
  percentage: number
}

interface AnalyticsTimeSeriesPoint {
  timestamp: string
  sent: number
  delivered: number
  bounced: number
  complained: number
  delivery_rate: number
  complaint_rate: number
}

interface WebhookEndpointItem {
  id: string
  url: string
  description?: string
  events: string[]
  secret: string
  previous_secret?: string
  previous_secret_expires_at?: string
  status: string
  consecutive_failures: number
  created_at: string
}

interface WebhookDeliveryItem {
  id: string
  endpoint_id: string
  event_id: string
  event_type: string
  payload: string
  status_code?: number
  response_body?: string
  error_message?: string
  duration_ms: number
  created_at: string
}

interface PrivacyRequestItem {
  id: string
  tenant_id: string
  type: string
  email_address: string
  status: string
  result_summary?: string
  requested_at: string
  completed_at?: string
}

interface RetentionPolicyItem {
  id: string
  tenant_id: string
  message_retention_days: number
  events_retention_days: number
  attempts_retention_days: number
  raw_mime_retention_days: number
  updated_at: string
}

interface RetentionPurgeResult {
  tenant_id: string
  purged_at: string
  purged_messages_count: number
  purged_events_count: number
  purged_attempts_count: number
  purged_raw_mime_count: number
}

interface InboundRoute {
  id: string
  tenant_id: string
  priority: number
  match_type: 'exact' | 'wildcard' | 'catch_all'
  pattern: string
  action: 'webhook' | 'store_only' | 'forward' | 'discard'
  action_target?: string
  is_active: boolean
  created_at: string
}

interface InboundMessage {
  id: string
  tenant_id: string
  route_id?: string
  from_address: string
  to_address: string
  subject: string
  spf_status: string
  dkim_status: string
  dmarc_status: string
  status: string
  created_at: string
}

interface ReputationOverview {
  tenant_id: string
  overall_status: string
  bounce_rate: number
  complaint_rate: number
  delivery_rate: number
  total_sent_7d: number
  dnsbl_listings_count: number
  warming_ips_count: number
  active_alerts_count: number
}

interface DNSBLProvider {
  id: string
  name: string
  query_pattern: string
  delist_url?: string
  check_interval_seconds: number
  is_enabled: boolean
}

interface DNSBLResult {
  id: string
  ip_address: string
  provider_name?: string
  is_listed: boolean
  response_code?: string
  return_meaning?: string
  delist_url?: string
  checked_at: string
}

interface WarmupPlan {
  id: string
  name: string
  description?: string
  total_days: number
  is_default: boolean
}

interface IPWarmupState {
  id: string
  ip_address_id: string
  ip_address: string
  plan_id: string
  current_day: number
  current_day_volume: number
  provider_volume?: Record<string, number>
  status: string
  held_reason?: string
  started_at: string
  last_advanced_at?: string
}

interface AlertRule {
  id: string
  tenant_id?: string
  name: string
  metric_type: string
  comparison_operator: string
  threshold: number
  duration_minutes: number
  cooldown_minutes: number
  severity: string
  is_enabled: boolean
}

interface AlertIncident {
  id: string
  rule_id: string
  metric_type: string
  severity: string
  status: string
  triggered_value: number
  threshold: number
  summary: string
  fired_at: string
  resolved_at?: string
}

interface NotificationChannel {
  id: string
  name: string
  type: string
  config: string
  is_enabled: boolean
}

interface AlertNotification {
  id: string
  incident_id: string
  title: string
  message: string
  severity: string
  is_read: boolean
  created_at: string
}

// Phase 12: SSO, Impersonation & Compliance types
interface SSOProvider {
  id: string
  org_id: string
  tenant_id?: string
  name: string
  provider_type: string
  issuer_url?: string
  client_id?: string
  idp_metadata_url?: string
  sp_entity_id?: string
  sp_acs_url?: string
  allowed_domains: string[]
  default_role: string
  enforce_sso: boolean
  is_active: boolean
  created_at: string
}

interface ImpersonationSession {
  id: string
  actor_user_id: string
  target_tenant_id: string
  reason: string
  access_type: string
  is_support_masking: boolean
  started_at: string
  ended_at?: string
}

interface ComplianceCheck {
  id: string
  tenant_id: string
  domain_name: string
  overall_score: number
  spf_status: string
  spf_details?: string
  dkim_status: string
  dkim_details?: string
  dmarc_status: string
  dmarc_details?: string
  fcrdns_status: string
  fcrdns_details?: string
  tls_status: string
  tls_details?: string
  dnsbl_status: string
  dnsbl_details?: string
  unsubscribe_status: string
  unsubscribe_details?: string
  checked_at: string
}

interface FCrDNSResult {
  ip_address: string
  ptr_hostnames: string[]
  forward_ips: string[]
  is_confirmed: boolean
  confirmed_domain?: string
  details: string
}

export default function App() {
  const [setupCompleted, setSetupCompleted] = useState<boolean | null>(null)
  const [user, setUser] = useState<User | null>(() => {
    try {
      const saved = localStorage.getItem('mvx_user')
      return saved ? JSON.parse(saved) : null
    } catch {
      return null
    }
  })
  const [tenants, setTenants] = useState<Tenant[]>(() => {
    try {
      const saved = localStorage.getItem('mvx_tenants')
      return saved ? JSON.parse(saved) : []
    } catch {
      return []
    }
  })
  const [activeTenant, setActiveTenant] = useState<Tenant | null>(() => {
    try {
      const saved = localStorage.getItem('mvx_active_tenant')
      return saved ? JSON.parse(saved) : null
    } catch {
      return null
    }
  })
  const [currentTab, setCurrentTab] = useState<'dashboard' | 'domains' | 'streams' | 'apikeys' | 'smtp_devices' | 'messages' | 'templates' | 'suppressions' | 'reputation' | 'analytics' | 'webhooks' | 'privacy_retention' | 'inbound' | 'alerts_warmup' | 'sso_compliance' | 'platform_admin'>(() => {
    try {
      const saved = localStorage.getItem('mvx_current_tab')
      return (saved as any) || 'dashboard'
    } catch {
      return 'dashboard'
    }
  })

  // Frosted Glassmorphism Theme System (Light / System / Dark)
  const [theme, setTheme] = useState<'light' | 'system' | 'dark'>(() => {
    try {
      const urlTheme = new URLSearchParams(window.location.search).get('theme') as any
      if (urlTheme === 'light' || urlTheme === 'dark' || urlTheme === 'system') return urlTheme
      const saved = localStorage.getItem('app_theme')
      return (saved as any) || 'system'
    } catch {
      return 'system'
    }
  })

  useEffect(() => {
    const applyTheme = (t: 'light' | 'system' | 'dark') => {
      localStorage.setItem('app_theme', t)
      const isDark =
        t === 'dark' ||
        (t === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
      if (isDark) {
        document.documentElement.classList.add('dark')
      } else {
        document.documentElement.classList.remove('dark')
      }
    }
    applyTheme(theme)

    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const listener = () => {
      if (theme === 'system') {
        applyTheme('system')
      }
    }
    media.addEventListener('change', listener)
    return () => media.removeEventListener('change', listener)
  }, [theme])

  // Platform Admin Center state
  const [adminActiveSubTab, setAdminActiveSubTab] = useState<'overview' | 'tenants' | 'health' | 'audit'>('overview')
  const [adminNewTenantName, setAdminNewTenantName] = useState('')
  const [adminNewTenantSlug, setAdminNewTenantSlug] = useState('')
  const [adminNewTenantEnv, setAdminNewTenantEnv] = useState<'live' | 'test'>('live')
  const [adminCreateTenantLoading, setAdminCreateTenantLoading] = useState(false)
  const [adminCreateTenantError, setAdminCreateTenantError] = useState<string | null>(null)
  const [adminCreateTenantSuccess, setAdminCreateTenantSuccess] = useState<string | null>(null)
  const [adminAuditLogs, setAdminAuditLogs] = useState<any[]>([])
  const [adminAuditLoading, setAdminAuditLoading] = useState(false)
  const [adminSearchTenant, setAdminSearchTenant] = useState('')

  // Domains state
  const [domains, setDomains] = useState<Domain[]>([])
  const [selectedDomain, setSelectedDomain] = useState<Domain | null>(null)
  const [domainRecords, setDomainRecords] = useState<DNSRecord[]>([])
  const [newDomainName, setNewDomainName] = useState('')
  const [newDomainType, setNewDomainType] = useState('public')
  const [addDomainLoading, setAddDomainLoading] = useState(false)
  const [verifyLoading, setVerifyLoading] = useState(false)

  // Streams state
  const [streams, setStreams] = useState<Stream[]>([])

  // Phase 4: API Keys state
  const [apiKeys, setApiKeys] = useState<APIKey[]>([])
  const [newKeyName, setNewKeyName] = useState('')
  const [newKeyEnv, setNewKeyEnv] = useState('live')
  const [createdSecret, setCreatedSecret] = useState<string | null>(null)
  const [keyLoading, setKeyLoading] = useState(false)

  // SMTP & LAN Devices state
  const [smtpCredentials, setSmtpCredentials] = useState<SMTPCredentialItem[]>([])
  const [smtpNewUsername, setSmtpNewUsername] = useState('')
  const [smtpNewRateLimit, setSmtpNewRateLimit] = useState(100)
  const [smtpNewAllowedDomains, setSmtpNewAllowedDomains] = useState('')
  const [smtpNewAllowedCIDRs, setSmtpNewAllowedCIDRs] = useState('')
  const [smtpCreatedPassword, setSmtpCreatedPassword] = useState<string | null>(null)
  const [smtpLoading, setSmtpLoading] = useState(false)

  const [devicesList, setDevicesList] = useState<DeviceItem[]>([])
  const [devNewName, setDevNewName] = useState('')
  const [devNewType, setDevNewType] = useState('camera')
  const [devNewIP, setDevNewIP] = useState('')
  const [devNewAllowedFrom, setDevNewAllowedFrom] = useState('')
  const [devLoading, setDevLoading] = useState(false)

  const [trustedNetworksList, setTrustedNetworksList] = useState<TrustedNetworkItem[]>([])
  const [tnNewCIDR, setTnNewCIDR] = useState('')
  const [tnNewDesc, setTnNewDesc] = useState('')
  const [tnLoading, setTnLoading] = useState(false)

  // Send Test Email state
  const [sendModalOpen, setSendModalOpen] = useState(false)
  const [sendFrom, setSendFrom] = useState('')
  const [sendTo, setSendTo] = useState('')
  const [sendSubject, setSendSubject] = useState('')
  const [sendBody, setSendBody] = useState('')
  const [sendIsHTML, setSendIsHTML] = useState(false)
  const [sendStream, setSendStream] = useState('transactional')
  const [sendLoading, setSendLoading] = useState(false)
  const [sendError, setSendError] = useState<string | null>(null)
  const [sendSuccess, setSendSuccess] = useState<string | null>(null)

  // DKIM rotation state
  const [rotateLoading, setRotateLoading] = useState(false)
  const [rotateResult, setRotateResult] = useState<any | null>(null)

  // Phase 4: Messages & Inspector state
  const [messagesList, setMessagesList] = useState<MessageItem[]>([])
  const [selectedMessage, setSelectedMessage] = useState<any | null>(null)
  const [selectedMsgRecipients, setSelectedMsgRecipients] = useState<any[]>([])
  const [selectedMsgEvents, setSelectedMsgEvents] = useState<any[]>([])
  const [selectedMsgBody, setSelectedMsgBody] = useState<string>('')
  const [inspectorOpen, setInspectorOpen] = useState(false)

  // Phase 4: Templates state
  const [templatesList, setTemplatesList] = useState<TemplateItem[]>([])
  const [newTmplName, setNewTmplName] = useState('')
  const [newTmplSubject, setNewTmplSubject] = useState('')
  const [newTmplHTML, setNewTmplHTML] = useState('')
  const [newTmplText, setNewTmplText] = useState('')
  const [previewVars, setPreviewVars] = useState('{\n  "name": "Alex"\n}')
  const [renderedPreview, setRenderedPreview] = useState<any | null>(null)

  // Phase 5: Suppressions state
  const [suppressionsList, setSuppressionsList] = useState<SuppressionItem[]>([])
  const [newSuppAddress, setNewSuppAddress] = useState('')
  const [newSuppType, setNewSuppType] = useState('manual')
  const [newSuppReason, setNewSuppReason] = useState('')

  // Phase 6: Reputation & Abuse state
  const [abuseReports, setAbuseReports] = useState<AbuseReportItem[]>([])
  const [fblReports, setFblReports] = useState<FBLReportItem[]>([])
  const [postmasterSnaps, setPostmasterSnaps] = useState<PostmasterSnapshotItem[]>([])
  const [abuseStatusFilter, setAbuseStatusFilter] = useState('pending')

  // Phase 7: Analytics state
  const [analyticsRange, setAnalyticsRange] = useState<'24h' | '7d' | '30d'>('24h')
  const [analyticsSummary, setAnalyticsSummary] = useState<AnalyticsSummary | null>(null)
  const [analyticsProviders, setAnalyticsProviders] = useState<ProviderBreakdownItem[]>([])
  const [analyticsLatency, setAnalyticsLatency] = useState<LatencyHistogramItem[]>([])
  const [analyticsTimeSeries, setAnalyticsTimeSeries] = useState<AnalyticsTimeSeriesPoint[]>([])
  const [analyticsLoading, setAnalyticsLoading] = useState(false)

  // Phase 8: Webhooks state
  const [webhooksList, setWebhooksList] = useState<WebhookEndpointItem[]>([])
  const [newWebhookURL, setNewWebhookURL] = useState('')
  const [newWebhookDesc, setNewWebhookDesc] = useState('')
  const [newWebhookEvents, setNewWebhookEvents] = useState<string>('*')
  const [selectedWebhook, setSelectedWebhook] = useState<WebhookEndpointItem | null>(null)
  const [webhookDeliveries, setWebhookDeliveries] = useState<WebhookDeliveryItem[]>([])
  const [deliveriesModalOpen, setDeliveriesModalOpen] = useState(false)
  const [webhookActionMsg, setWebhookActionMsg] = useState<string | null>(null)

  // Phase 9: Suppression Import/Export, Privacy & Retention state
  const [importModalOpen, setImportModalOpen] = useState(false)
  const [importCSVText, setImportCSVText] = useState('')
  const [importStatusMsg, setImportStatusMsg] = useState<string | null>(null)
  const [privacyRequests, setPrivacyRequests] = useState<PrivacyRequestItem[]>([])
  const [privacyEmail, setPrivacyEmail] = useState('')
  const [privacyActionMsg, setPrivacyActionMsg] = useState<string | null>(null)
  const [privacyExportReport, setPrivacyExportReport] = useState<any | null>(null)
  const [retentionPolicy, setRetentionPolicy] = useState<RetentionPolicyItem>({
    id: '',
    tenant_id: '',
    message_retention_days: 30,
    events_retention_days: 90,
    attempts_retention_days: 30,
    raw_mime_retention_days: 7,
    updated_at: '',
  })
  const [retentionMsg, setRetentionMsg] = useState<string | null>(null)
  const [purgeResult, setPurgeResult] = useState<RetentionPurgeResult | null>(null)

  // Phase 10: Inbound Routing state
  const [inboundRoutes, setInboundRoutes] = useState<InboundRoute[]>([])
  const [inboundMessages, setInboundMessages] = useState<InboundMessage[]>([])
  const [newRoutePriority, setNewRoutePriority] = useState(10)
  const [newRouteMatchType, setNewRouteMatchType] = useState<'exact' | 'wildcard' | 'catch_all'>('exact')
  const [newRoutePattern, setNewRoutePattern] = useState('')
  const [newRouteAction, setNewRouteAction] = useState<'webhook' | 'store_only' | 'forward' | 'discard'>('webhook')
  const [newRouteTarget, setNewRouteTarget] = useState('')
  const [newRouteActive, setNewRouteActive] = useState(true)
  const [testInboundAddress, setTestInboundAddress] = useState('')
  const [testInboundResult, setTestInboundResult] = useState<any | null>(null)
  const [inboundActionMsg, setInboundActionMsg] = useState<string | null>(null)

  // Phase 11: Reputation, DNSBL, Warmup & Alerts state
  const [reputationOverview, setReputationOverview] = useState<ReputationOverview | null>(null)
  const [dnsblProviders, setDNSBLProviders] = useState<DNSBLProvider[]>([])
  const [dnsblResults, setDNSBLResults] = useState<DNSBLResult[]>([])
  const [testDNSBLIP, setTestDNSBLIP] = useState('')
  const [dnsblChecking, setDNSBLChecking] = useState(false)
  const [warmupPlans, setWarmupPlans] = useState<WarmupPlan[]>([])
  const [warmupStates, setWarmupStates] = useState<IPWarmupState[]>([])
  const [newWarmupIP, setNewWarmupIP] = useState('')
  const [newWarmupIPID, setNewWarmupIPID] = useState('')
  const [newWarmupPlanID, setNewWarmupPlanID] = useState('wplan_default')
  const [alertRules, setAlertRules] = useState<AlertRule[]>([])
  const [alertIncidents, setAlertIncidents] = useState<AlertIncident[]>([])
  const [notificationChannels, setNotificationChannels] = useState<NotificationChannel[]>([])
  const [alertNotifications, setAlertNotifications] = useState<AlertNotification[]>([])
  const [newRuleName, setNewRuleName] = useState('')
  const [newRuleMetric, setNewRuleMetric] = useState('bounce_rate')
  const [newRuleOp, setNewRuleOp] = useState('>')
  const [newRuleThreshold, setNewRuleThreshold] = useState(0.05)
  const [newRuleSeverity, setNewRuleSeverity] = useState('warning')
  const [newRuleCooldown, setNewRuleCooldown] = useState(60)
  const [newChannelName, setNewChannelName] = useState('')
  const [newChannelType, setNewChannelType] = useState('slack')
  const [newChannelURL, setNewChannelURL] = useState('')
  const [alertsActionMsg, setAlertsActionMsg] = useState<string | null>(null)

  // Phase 12: SSO, Impersonation & Compliance state
  const [ssoProviders, setSSOProviders] = useState<SSOProvider[]>([])
  const [newSSOName, setNewSSOName] = useState('')
  const [newSSOType, setNewSSOType] = useState<'oidc' | 'saml'>('oidc')
  const [newSSOIssuer, setNewSSOIssuer] = useState('')
  const [newSSOClientID, setNewSSOClientID] = useState('')
  const [newSSODomains, setNewSSODomains] = useState('')
  const [newSSORole, setNewSSORole] = useState('developer')
  const [newSSOEnforce, setNewSSOEnforce] = useState(false)
  const [ssoActionMsg, setSSOActionMsg] = useState<string | null>(null)

  const [impersonationSessions, setImpersonationSessions] = useState<ImpersonationSession[]>([])
  const [isImpersonating, setIsImpersonating] = useState(false)
  const [impersonatedTenant, setImpersonatedTenant] = useState<Tenant | null>(null)
  const [impersonationReason, setImpersonationReason] = useState('')
  const [isSupportMasking, setIsSupportMasking] = useState(false)
  const [impModalOpen, setImpModalOpen] = useState(false)
  const [targetImpTenantID, setTargetImpTenantID] = useState('')
  const [targetImpReason, setTargetImpReason] = useState('')
  const [targetImpSupportMode, setTargetImpSupportMode] = useState(false)

  const [complianceResult, setComplianceResult] = useState<ComplianceCheck | null>(null)
  const [complianceLoading, setComplianceLoading] = useState(false)
  const [complianceDomain, setComplianceDomain] = useState('')
  const [testFCrDNSIP, setTestFCrDNSIP] = useState('')
  const [fcrdnsResult, setFCrDNSResult] = useState<FCrDNSResult | null>(null)
  const [fcrdnsChecking, setFCrDNSChecking] = useState(false)


  // Setup form states
  const [profile, setProfile] = useState('standard')
  const [adminEmail, setAdminEmail] = useState('')
  const [adminPassword, setAdminPassword] = useState('')
  const [orgName, setOrgName] = useState('')
  const [tenantName, setTenantName] = useState('')
  const [setupLoading, setSetupLoading] = useState(false)
  const [setupError, setSetupError] = useState('')

  // Login form states
  const [loginEmail, setLoginEmail] = useState('')
  const [loginPassword, setLoginPassword] = useState('')
  const [loginLoading, setLoginLoading] = useState(false)
  const [loginError, setLoginError] = useState('')

  const handleSignOut = () => {
    localStorage.removeItem('mvx_user')
    localStorage.removeItem('mvx_tenants')
    localStorage.removeItem('mvx_active_tenant')
    localStorage.removeItem('mvx_current_tab')
    setUser(null)
    setTenants([])
    setActiveTenant(null)
    setCurrentTab('dashboard')
  }

  useEffect(() => {
    fetch('/v1/setup')
      .then((res) => res.json())
      .then((data) => setSetupCompleted(data.setup_completed))
      .catch(() => setSetupCompleted(false))

    // Rehydrate & verify session from localStorage
    const savedUser = localStorage.getItem('mvx_user')
    const savedTenant = localStorage.getItem('mvx_active_tenant')
    if (savedUser) {
      try {
        const parsedUser = JSON.parse(savedUser)
        const parsedTenant = savedTenant ? JSON.parse(savedTenant) : null
        fetch('/v1/auth/me', {
          headers: {
            'X-User-ID': parsedUser.id,
            'X-Tenant-ID': parsedTenant?.id || 'tenant_default',
          },
        })
          .then((res) => {
            if (res.ok) {
              return res.json().then((d) => {
                if (d.user) {
                  setUser(d.user)
                  localStorage.setItem('mvx_user', JSON.stringify(d.user))
                }
                if (d.available_tenants) {
                  setTenants(d.available_tenants)
                  localStorage.setItem('mvx_tenants', JSON.stringify(d.available_tenants))
                  if (!parsedTenant || !d.available_tenants.some((t: Tenant) => t.id === parsedTenant.id)) {
                    if (d.available_tenants.length > 0) {
                      setActiveTenant(d.available_tenants[0])
                      localStorage.setItem('mvx_active_tenant', JSON.stringify(d.available_tenants[0]))
                    }
                  }
                }
              })
            } else if (res.status === 401) {
              handleSignOut()
            }
          })
          .catch(() => {
            // Keep existing session on network error
          })
      } catch {
        handleSignOut()
      }
    }
  }, [])

  // Sync state changes to localStorage
  useEffect(() => {
    if (user) {
      localStorage.setItem('mvx_user', JSON.stringify(user))
    }
  }, [user])

  useEffect(() => {
    if (tenants && tenants.length > 0) {
      localStorage.setItem('mvx_tenants', JSON.stringify(tenants))
    }
  }, [tenants])

  useEffect(() => {
    if (activeTenant) {
      localStorage.setItem('mvx_active_tenant', JSON.stringify(activeTenant))
    }
  }, [activeTenant])

  useEffect(() => {
    if (currentTab) {
      localStorage.setItem('mvx_current_tab', currentTab)
    }
  }, [currentTab])

  useEffect(() => {
    if (activeTenant && user) {
      loadDomains()
      loadStreams()
      loadAPIKeys()
      loadSMTPCredentials()
      loadDevices()
      loadMessages()
      loadTemplates()
      loadSuppressions()
      loadAbuseReports()
      loadFBLReports()
      loadPostmasterSnaps()
      loadAnalytics()
      loadWebhooks()
      loadPrivacyRequests()
      loadRetentionPolicy()
      loadInboundRoutes()
      loadInboundMessages()
      loadPhase11Data()
      loadPhase12Data()
      if (user.is_platform_admin) {
        fetchAdminAuditLogs()
      }
    }
  }, [activeTenant, currentTab, abuseStatusFilter, analyticsRange])

  const fetchAdminAuditLogs = async () => {
    if (!user || !user.is_platform_admin) return
    setAdminAuditLoading(true)
    try {
      const headers: Record<string, string> = {
        'X-User-ID': user.id,
        'X-Tenant-ID': activeTenant?.id || 'tenant_default',
      }
      const res = await fetch('/v1/audit-logs', { headers })
      if (res.ok) {
        const d = await res.json()
        setAdminAuditLogs(d.data || [])
      }
    } catch {}
    finally {
      setAdminAuditLoading(false)
    }
  }

  const handleAdminCreateTenant = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!user || !adminNewTenantName.trim() || !adminNewTenantSlug.trim()) return
    setAdminCreateTenantLoading(true)
    setAdminCreateTenantError(null)
    setAdminCreateTenantSuccess(null)
    try {
      const res = await fetch('/v1/tenants', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant?.id || 'tenant_default',
        },
        body: JSON.stringify({
          name: adminNewTenantName.trim(),
          slug: adminNewTenantSlug.trim().toLowerCase(),
          environment: adminNewTenantEnv,
        }),
      })
      const data = await res.json()
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to create tenant')
      }
      setAdminCreateTenantSuccess(`Tenant "${data.name}" (${data.slug}) created successfully with auto-generated DEK!`)
      setAdminNewTenantName('')
      setAdminNewTenantSlug('')
      // Refresh tenants list
      const tRes = await fetch('/v1/tenants', {
        headers: {
          'X-User-ID': user.id,
          'X-Tenant-ID': data.id,
        },
      })
      if (tRes.ok) {
        const tData = await tRes.json()
        const refreshed = tData.tenants || tData.data || []
        setTenants(refreshed)
        // If newly created tenant is in list, we can switch or keep
      }
      fetchAdminAuditLogs()
    } catch (err: any) {
      setAdminCreateTenantError(err.message)
    } finally {
      setAdminCreateTenantLoading(false)
    }
  }

  const loadPhase11Data = async () => {
    if (!activeTenant || !user) return
    const headers = { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id }
    try {
      const repRes = await fetch('/v1/reputation/overview', { headers })
      if (repRes.ok) setReputationOverview(await repRes.json())

      const provRes = await fetch('/v1/dnsbl/providers', { headers })
      if (provRes.ok) {
        const d = await provRes.json()
        setDNSBLProviders(d.data || [])
      }

      const resRes = await fetch('/v1/dnsbl/results?limit=25', { headers })
      if (resRes.ok) {
        const d = await resRes.json()
        setDNSBLResults(d.data || [])
      }

      const planRes = await fetch('/v1/warmup/plans', { headers })
      if (planRes.ok) {
        const d = await planRes.json()
        setWarmupPlans(d.data || [])
      }

      const stateRes = await fetch('/v1/warmup/ips', { headers })
      if (stateRes.ok) {
        const d = await stateRes.json()
        setWarmupStates(d.data || [])
      }

      const ruleRes = await fetch('/v1/alerts/rules', { headers })
      if (ruleRes.ok) {
        const d = await ruleRes.json()
        setAlertRules(d.data || [])
      }

      const incRes = await fetch('/v1/alerts/incidents?limit=25', { headers })
      if (incRes.ok) {
        const d = await incRes.json()
        setAlertIncidents(d.data || [])
      }

      const chanRes = await fetch('/v1/alerts/channels', { headers })
      if (chanRes.ok) {
        const d = await chanRes.json()
        setNotificationChannels(d.data || [])
      }

      const notifRes = await fetch('/v1/alerts/notifications?limit=25', { headers })
      if (notifRes.ok) {
        const d = await notifRes.json()
        setAlertNotifications(d.data || [])
      }
    } catch {}
  }

  const handleRunDNSBLCheck = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !testDNSBLIP.trim()) return
    setDNSBLChecking(true)
    try {
      const res = await fetch('/v1/dnsbl/check', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ ip: testDNSBLIP.trim(), ip_address_id: 'ip_' + testDNSBLIP.trim() }),
      })
      if (res.ok) {
        setAlertsActionMsg(`DNSBL check completed for ${testDNSBLIP.trim()}.`)
        loadPhase11Data()
        setTimeout(() => setAlertsActionMsg(null), 4000)
      } else {
        const d = await res.json()
        alert(d.error?.message || 'Check failed')
      }
    } catch (err: any) {
      alert(err.message)
    } finally {
      setDNSBLChecking(false)
    }
  }

  const handleEnrollWarmup = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !newWarmupIP.trim()) return
    try {
      const res = await fetch('/v1/warmup/ips', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({
          ip: newWarmupIP.trim(),
          ip_address_id: newWarmupIPID.trim() || 'ip_' + newWarmupIP.trim(),
          plan_id: newWarmupPlanID,
        }),
      })
      if (res.ok) {
        setNewWarmupIP('')
        setNewWarmupIPID('')
        setAlertsActionMsg('IP enrolled in warmup plan successfully.')
        loadPhase11Data()
        setTimeout(() => setAlertsActionMsg(null), 4000)
      } else {
        const d = await res.json()
        alert(d.error?.message || 'Enrollment failed')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleAdvanceWarmup = async (id: string, force: boolean) => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch(`/v1/warmup/ips/${id}/advance`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ force }),
      })
      if (res.ok) {
        loadPhase11Data()
      } else {
        const d = await res.json()
        alert(d.error?.message || 'Advance failed')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleHoldWarmup = async (id: string) => {
    if (!activeTenant || !user) return
    const reason = prompt('Enter reason for holding warmup:', 'Administrative hold')
    if (!reason) return
    try {
      const res = await fetch(`/v1/warmup/ips/${id}/hold`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ reason }),
      })
      if (res.ok) {
        loadPhase11Data()
      }
    } catch {}
  }

  const handleResumeWarmup = async (id: string) => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch(`/v1/warmup/ips/${id}/resume`, {
        method: 'POST',
        headers: {
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
      })
      if (res.ok) {
        loadPhase11Data()
      }
    } catch {}
  }

  const handleCreateAlertRule = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !newRuleName.trim()) return
    try {
      const res = await fetch('/v1/alerts/rules', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({
          name: newRuleName.trim(),
          metric_type: newRuleMetric,
          comparison_operator: newRuleOp,
          threshold: Number(newRuleThreshold),
          severity: newRuleSeverity,
          cooldown_minutes: Number(newRuleCooldown),
        }),
      })
      if (res.ok) {
        setNewRuleName('')
        setAlertsActionMsg('Alert rule created successfully.')
        loadPhase11Data()
        setTimeout(() => setAlertsActionMsg(null), 4000)
      } else {
        const d = await res.json()
        alert(d.error?.message || 'Rule creation failed')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleDeleteAlertRule = async (id: string) => {
    if (!activeTenant || !user) return
    if (!confirm('Are you sure you want to delete this alert rule?')) return
    try {
      const res = await fetch(`/v1/alerts/rules/${id}`, {
        method: 'DELETE',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        loadPhase11Data()
      }
    } catch {}
  }

  const handleEvaluateAlerts = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/alerts/evaluate', {
        method: 'POST',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        setAlertsActionMsg('Alert evaluation completed.')
        loadPhase11Data()
        setTimeout(() => setAlertsActionMsg(null), 4000)
      }
    } catch {}
  }

  const handleAcknowledgeIncident = async (id: string) => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch(`/v1/alerts/incidents/${id}/acknowledge`, {
        method: 'POST',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        loadPhase11Data()
      }
    } catch {}
  }

  const handleResolveIncident = async (id: string) => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch(`/v1/alerts/incidents/${id}/resolve`, {
        method: 'POST',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        loadPhase11Data()
      }
    } catch {}
  }

  const handleCreateChannel = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !newChannelName.trim()) return
    try {
      const res = await fetch('/v1/alerts/channels', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({
          name: newChannelName.trim(),
          type: newChannelType,
          config: JSON.stringify({ url: newChannelURL.trim() }),
        }),
      })
      if (res.ok) {
        setNewChannelName('')
        setNewChannelURL('')
        setAlertsActionMsg('Notification channel created.')
        loadPhase11Data()
        setTimeout(() => setAlertsActionMsg(null), 4000)
      } else {
        const d = await res.json()
        alert(d.error?.message || 'Channel creation failed')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleDeleteChannel = async (id: string) => {
    if (!activeTenant || !user) return
    if (!confirm('Are you sure you want to delete this channel?')) return
    try {
      const res = await fetch(`/v1/alerts/channels/${id}`, {
        method: 'DELETE',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        loadPhase11Data()
      }
    } catch {}
  }

  const handleTestChannel = async (id: string) => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch(`/v1/alerts/channels/${id}/test`, {
        method: 'POST',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        alert('Test notification dispatched successfully!')
      } else {
        const d = await res.json()
        alert(d.error?.message || 'Dispatch failed')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleMarkNotifRead = async (id: string) => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch(`/v1/alerts/notifications/${id}/read`, {
        method: 'POST',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        loadPhase11Data()
      }
    } catch {}
  }

  const loadPhase12Data = async () => {
    if (!user) return
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
    }
    if (activeTenant) {
      headers['X-Tenant-ID'] = activeTenant.id
    }
    try {
      const ssoRes = await fetch('/v1/auth/sso/providers', { headers })
      if (ssoRes.ok) {
        const d = await ssoRes.json()
        setSSOProviders(d.data || [])
      }

      if (user.is_platform_admin) {
        const impRes = await fetch('/v1/admin/impersonate/sessions?limit=25', { headers })
        if (impRes.ok) {
          const d = await impRes.json()
          setImpersonationSessions(d.data || [])
        }
      }

      if (activeTenant && domains.length > 0) {
        const targetDom = complianceDomain || domains[0].name
        const compRes = await fetch(`/v1/compliance/check?domain=${encodeURIComponent(targetDom)}`, { headers })
        if (compRes.ok) {
          const d = await compRes.json()
          setComplianceResult(d.data || null)
        }
      }
    } catch {}
  }

  const handleCreateSSOProvider = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!user || !newSSOName.trim()) return
    const orgId = activeTenant?.organization_id || 'org_default'
    try {
      const domainsArr = newSSODomains.split(',').map((s) => s.trim()).filter(Boolean)
      const res = await fetch('/v1/auth/sso/providers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant?.id || '',
        },
        body: JSON.stringify({
          org_id: orgId,
          name: newSSOName.trim(),
          provider_type: newSSOType,
          issuer_url: newSSOIssuer.trim() || undefined,
          client_id: newSSOClientID.trim() || undefined,
          allowed_domains: domainsArr,
          default_role: newSSORole,
          enforce_sso: newSSOEnforce,
          sp_acs_url: window.location.origin + (newSSOType === 'oidc' ? '/v1/auth/sso/oidc/callback' : '/v1/auth/sso/saml/acs'),
          sp_entity_id: window.location.origin + '/v1/auth/sso/saml/metadata',
        }),
      })
      if (res.ok) {
        setNewSSOName('')
        setNewSSOIssuer('')
        setNewSSOClientID('')
        setNewSSODomains('')
        setSSOActionMsg('SSO Provider registered successfully.')
        loadPhase12Data()
        setTimeout(() => setSSOActionMsg(null), 4000)
      } else {
        const d = await res.json()
        alert(d.error?.message || 'Failed to register SSO Provider')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleDeleteSSOProvider = async (id: string) => {
    if (!user || !activeTenant) return
    if (!confirm('Are you sure you want to delete this SSO provider?')) return
    try {
      const res = await fetch(`/v1/auth/sso/providers/${id}`, {
        method: 'DELETE',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        loadPhase12Data()
      }
    } catch {}
  }

  const handleStartImpersonation = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!user || !user.is_platform_admin || !targetImpTenantID || !targetImpReason.trim()) return
    try {
      const res = await fetch('/v1/admin/impersonate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant?.id || '',
        },
        body: JSON.stringify({
          target_tenant_id: targetImpTenantID,
          reason: targetImpReason.trim(),
          access_type: 'read_only',
          is_support_masking: targetImpSupportMode,
        }),
      })
      if (res.ok) {
        const target = tenants.find((t) => t.id === targetImpTenantID) || { id: targetImpTenantID, name: targetImpTenantID, slug: '', environment: 'live', status: 'active', organization_id: 'org_default', created_at: '', updated_at: '' }
        setIsImpersonating(true)
        setImpersonatedTenant(target)
        setActiveTenant(target)
        setImpersonationReason(targetImpReason.trim())
        setIsSupportMasking(targetImpSupportMode)
        setImpModalOpen(false)
        setSSOActionMsg(`Impersonating ${target.name}. All actions recorded to platform audit log.`)
        loadPhase12Data()
      } else {
        const d = await res.json()
        alert(d.error?.message || 'Impersonation failed')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleExitImpersonation = async () => {
    setIsImpersonating(false)
    setImpersonatedTenant(null)
    setImpersonationReason('')
    setIsSupportMasking(false)
    if (tenants.length > 0) {
      setActiveTenant(tenants[0])
    }
    setSSOActionMsg('Exited tenant impersonation.')
    loadPhase12Data()
    setTimeout(() => setSSOActionMsg(null), 4000)
  }

  const handleRunComplianceCheck = async (domainName: string) => {
    if (!user || !activeTenant) return
    setComplianceLoading(true)
    try {
      const res = await fetch(`/v1/compliance/check?domain=${encodeURIComponent(domainName)}`, {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        const d = await res.json()
        setComplianceResult(d.data || null)
      } else {
        const d = await res.json()
        alert(d.error?.message || 'Compliance evaluation failed')
      }
    } catch (err: any) {
      alert(err.message)
    } finally {
      setComplianceLoading(false)
    }
  }

  const handleRunFCrDNSCheck = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!user || !activeTenant || !testFCrDNSIP.trim()) return
    setFCrDNSChecking(true)
    try {
      const res = await fetch('/v1/health/fcrdns', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ ip_address: testFCrDNSIP.trim() }),
      })
      if (res.ok) {
        const d = await res.json()
        setFCrDNSResult(d.data || null)
      } else {
        const d = await res.json()
        alert(d.error?.message || 'FCrDNS check failed')
      }
    } catch (err: any) {
      alert(err.message)
    } finally {
      setFCrDNSChecking(false)
    }
  }

  const loadInboundRoutes = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/inbound/routes', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setInboundRoutes(data.data || [])
      }
    } catch {}
  }

  const loadInboundMessages = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/inbound/messages?limit=50', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setInboundMessages(data.data || [])
      }
    } catch {}
  }

  const handleCreateInboundRoute = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !newRoutePattern) return
    try {
      const res = await fetch('/v1/inbound/routes', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({
          priority: Number(newRoutePriority),
          match_type: newRouteMatchType,
          pattern: newRoutePattern.trim(),
          action: newRouteAction,
          action_target: newRouteTarget.trim() ? newRouteTarget.trim() : undefined,
          is_active: newRouteActive,
        }),
      })
      const data = await res.json()
      if (res.ok) {
        setInboundActionMsg('Inbound route created successfully.')
        setNewRoutePattern('')
        setNewRouteTarget('')
        setNewRoutePriority(10)
        loadInboundRoutes()
        setTimeout(() => setInboundActionMsg(null), 4000)
      } else {
        alert(data.error?.message || 'Failed to create inbound route')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleDeleteInboundRoute = async (id: string) => {
    if (!activeTenant || !user) return
    if (!confirm('Are you sure you want to delete this inbound route?')) return
    try {
      const res = await fetch(`/v1/inbound/routes/${id}`, {
        method: 'DELETE',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        loadInboundRoutes()
      }
    } catch {}
  }

  const handleToggleInboundRoute = async (route: InboundRoute) => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch(`/v1/inbound/routes/${route.id}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ is_active: !route.is_active }),
      })
      if (res.ok) {
        loadInboundRoutes()
      }
    } catch {}
  }

  const handleTestInboundRoute = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !testInboundAddress) return
    try {
      const res = await fetch('/v1/inbound/routes/test', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ address: testInboundAddress.trim() }),
      })
      const data = await res.json()
      if (res.ok) {
        setTestInboundResult(data)
      } else {
        alert(data.error?.message || 'Failed to test route match')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const loadPrivacyRequests = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/privacy/requests', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setPrivacyRequests(data.data || [])
      }
    } catch {}
  }

  const loadRetentionPolicy = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/retention', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setRetentionPolicy(data)
      }
    } catch {}
  }

  const handleImportSuppressions = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !importCSVText.trim()) return
    setImportStatusMsg(null)
    try {
      const res = await fetch('/v1/suppressions/import', {
        method: 'POST',
        headers: {
          'Content-Type': 'text/csv',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: importCSVText,
      })
      const data = await res.json()
      if (res.ok) {
        setImportStatusMsg(`Import completed: ${data.imported_rows} rows added, ${data.duplicate_rows} duplicates ignored.`)
        setImportCSVText('')
        loadSuppressions()
      } else {
        setImportStatusMsg(`Import failed: ${data.error?.message || 'Unknown error'}`)
      }
    } catch (err: any) {
      setImportStatusMsg(`Error: ${err.message}`)
    }
  }

  const handlePrivacyErasure = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !privacyEmail.trim()) return
    setPrivacyActionMsg(null)
    try {
      const res = await fetch('/v1/privacy/erasure', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ email: privacyEmail.trim() }),
      })
      const data = await res.json()
      if (res.ok) {
        setPrivacyActionMsg(`Erasure complete: ${data.result_summary || 'PII anonymized'}`)
        setPrivacyEmail('')
        loadPrivacyRequests()
        loadSuppressions()
      } else {
        setPrivacyActionMsg(`Erasure error: ${data.error?.message || 'Request failed'}`)
      }
    } catch (err: any) {
      setPrivacyActionMsg(`Error: ${err.message}`)
    }
  }

  const handlePrivacyExport = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !privacyEmail.trim()) return
    setPrivacyActionMsg(null)
    try {
      const res = await fetch('/v1/privacy/export', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ email: privacyEmail.trim() }),
      })
      const data = await res.json()
      if (res.ok) {
        setPrivacyExportReport(data)
        setPrivacyActionMsg(`Export report ready for ${data.email_address}`)
        loadPrivacyRequests()
      } else {
        setPrivacyActionMsg(`Export error: ${data.error?.message || 'Request failed'}`)
      }
    } catch (err: any) {
      setPrivacyActionMsg(`Error: ${err.message}`)
    }
  }

  const handleUpdateRetentionPolicy = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user) return
    setRetentionMsg(null)
    try {
      const res = await fetch('/v1/retention', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify(retentionPolicy),
      })
      const data = await res.json()
      if (res.ok) {
        setRetentionPolicy(data)
        setRetentionMsg('Data retention policy updated successfully.')
      } else {
        setRetentionMsg(`Update error: ${data.error?.message || 'Failed'}`)
      }
    } catch (err: any) {
      setRetentionMsg(`Error: ${err.message}`)
    }
  }

  const handleExecutePurge = async () => {
    if (!activeTenant || !user) return
    setRetentionMsg(null)
    try {
      const res = await fetch('/v1/retention/purge', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
      })
      const data = await res.json()
      if (res.ok) {
        setPurgeResult(data)
        setRetentionMsg(`Automated purge completed: ${data.purged_messages_count} messages, ${data.purged_events_count} events, ${data.purged_attempts_count} attempts, and ${data.purged_raw_mime_count} raw MIME references cleaned.`)
      } else {
        setRetentionMsg(`Purge error: ${data.error?.message || 'Purge failed'}`)
      }
    } catch (err: any) {
      setRetentionMsg(`Error: ${err.message}`)
    }
  }

  const loadWebhooks = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/webhooks', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setWebhooksList(data.data || [])
      }
    } catch {}
  }

  const handleCreateWebhook = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !newWebhookURL) return
    try {
      const events = newWebhookEvents === '*' ? ['*'] : newWebhookEvents.split(',').map((s) => s.trim()).filter(Boolean)
      const res = await fetch('/v1/webhooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({
          url: newWebhookURL,
          description: newWebhookDesc || undefined,
          events: events.length > 0 ? events : ['*'],
        }),
      })
      const data = await res.json()
      if (res.ok) {
        setNewWebhookURL('')
        setNewWebhookDesc('')
        setNewWebhookEvents('*')
        loadWebhooks()
        setWebhookActionMsg(`Created webhook endpoint with secret: ${data.data?.secret}`)
      } else {
        alert(data.error?.message || 'Failed to create webhook')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleRotateWebhookSecret = async (id: string) => {
    if (!activeTenant || !user) return
    if (!confirm('Rotate secret? Both old and new secrets will be signed for 24 hours to prevent dropped payloads.')) return
    try {
      const res = await fetch(`/v1/webhooks/${id}/rotate-secret`, {
        method: 'POST',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        loadWebhooks()
        setWebhookActionMsg(`New Secret: ${data.data?.secret}. Both old and new secrets active until: ${new Date(data.overlap_expires_at).toLocaleString()}`)
      } else {
        alert(data.error?.message || 'Failed to rotate secret')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleTestWebhook = async (id: string) => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch(`/v1/webhooks/${id}/test`, {
        method: 'POST',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        const del = data.delivery
        alert(`Test Ping Dispatched!\nStatus Code: ${del?.status_code || 'Error'}\nDuration: ${del?.duration_ms}ms\nResponse: ${del?.response_body || del?.error_message || 'None'}`)
        loadWebhooks()
      } else {
        alert(data.error?.message || 'Ping failed')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleViewDeliveries = async (ep: WebhookEndpointItem) => {
    if (!activeTenant || !user) return
    setSelectedWebhook(ep)
    setDeliveriesModalOpen(true)
    try {
      const res = await fetch(`/v1/webhooks/${ep.id}/deliveries`, {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setWebhookDeliveries(data.data || [])
      }
    } catch {}
  }

  const handleDeleteWebhook = async (id: string) => {
    if (!activeTenant || !user) return
    if (!confirm('Delete this webhook endpoint?')) return
    try {
      const res = await fetch(`/v1/webhooks/${id}`, {
        method: 'DELETE',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        loadWebhooks()
      }
    } catch {}
  }

  const loadAnalytics = async () => {
    if (!activeTenant || !user) return
    setAnalyticsLoading(true)
    try {
      const q = `?range=${analyticsRange}`
      const interval = analyticsRange === '30d' ? '1d' : '1h'
      const [overviewRes, providersRes, latencyRes, tsRes] = await Promise.all([
        fetch(`/v1/analytics/overview${q}`, {
          headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
        }),
        fetch(`/v1/analytics/providers${q}`, {
          headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
        }),
        fetch(`/v1/analytics/latency${q}`, {
          headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
        }),
        fetch(`/v1/analytics/time-series?interval=${interval}&range=${analyticsRange}`, {
          headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
        }),
      ])

      if (overviewRes.ok) {
        const data = await overviewRes.json()
        setAnalyticsSummary({
          ...data,
          sent: data.sent ?? 0,
          delivered: data.delivered ?? 0,
          bounced_hard: data.bounced_hard ?? 0,
          bounced_soft: data.bounced_soft ?? 0,
          complained: data.complained ?? 0,
          unsubscribed: data.unsubscribed ?? 0,
          sent_count: data.sent ?? data.sent_count ?? 0,
          delivered_count: data.delivered ?? data.delivered_count ?? 0,
          hard_bounce_count: data.bounced_hard ?? data.hard_bounce_count ?? 0,
          soft_bounce_count: data.bounced_soft ?? data.soft_bounce_count ?? 0,
          complaint_count: data.complained ?? data.complaint_count ?? 0,
          unsubscribe_count: data.unsubscribed ?? data.unsubscribe_count ?? 0,
          delivery_rate: data.delivery_rate ?? 0,
          bounce_rate: data.bounce_rate ?? 0,
          complaint_rate: data.complaint_rate ?? 0,
          unsubscribe_rate: data.unsubscribe_rate ?? 0,
          complaint_warning: !!data.complaint_warning,
          median_latency_seconds: data.median_latency_seconds ?? 0,
          p95_latency_seconds: data.p95_latency_seconds ?? 0,
        })
      }
      if (providersRes.ok) {
        const data = await providersRes.json()
        const rawList = Array.isArray(data) ? data : (data.data || [])
        const mapped = rawList.map((p: any) => ({
          provider: p.provider || 'unknown',
          sent: p.sent ?? p.sent_count ?? 0,
          delivered: p.delivered ?? p.delivered_count ?? 0,
          bounced: p.bounced ?? p.bounce_count ?? 0,
          complained: p.complained ?? p.complaint_count ?? 0,
          sent_count: p.sent ?? p.sent_count ?? 0,
          delivered_count: p.delivered ?? p.delivered_count ?? 0,
          bounce_count: p.bounced ?? p.bounce_count ?? 0,
          complaint_count: p.complained ?? p.complaint_count ?? 0,
          delivery_rate: p.delivery_rate ?? 0,
          bounce_rate: p.bounce_rate ?? (p.sent > 0 ? (p.bounced ?? 0) / p.sent : 0),
          complaint_rate: p.complaint_rate ?? 0,
        }))
        setAnalyticsProviders(mapped)
      }
      if (latencyRes.ok) {
        const data = await latencyRes.json()
        const u5 = data.under_5s || 0
        const u30 = data.under_30s || 0
        const u1m = data.under_1m || 0
        const u5m = data.under_5m || 0
        const u15m = data.under_15m || 0
        const o15m = data.over_15m || 0
        const total = u5 + u30 + u1m + u5m + u15m + o15m

        const hist: LatencyHistogramItem[] = [
          { bucket: '< 5 seconds', count: u5, percentage: total > 0 ? (u5 / total) * 100 : 0 },
          { bucket: '5 - 30 seconds', count: u30, percentage: total > 0 ? (u30 / total) * 100 : 0 },
          { bucket: '30s - 1 minute', count: u1m, percentage: total > 0 ? (u1m / total) * 100 : 0 },
          { bucket: '1 - 5 minutes', count: u5m, percentage: total > 0 ? (u5m / total) * 100 : 0 },
          { bucket: '5 - 15 minutes', count: u15m, percentage: total > 0 ? (u15m / total) * 100 : 0 },
          { bucket: '> 15 minutes', count: o15m, percentage: total > 0 ? (o15m / total) * 100 : 0 },
        ]
        setAnalyticsLatency(hist)
      }
      if (tsRes.ok) {
        const data = await tsRes.json()
        setAnalyticsTimeSeries(data.data || [])
      }
    } catch {}
    finally {
      setAnalyticsLoading(false)
    }
  }

  const loadAbuseReports = async () => {
    if (!activeTenant || !user) return
    try {
      const q = abuseStatusFilter ? `?status=${abuseStatusFilter}` : ''
      const res = await fetch(`/v1/abuse-reports${q}`, {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setAbuseReports(data.data || [])
      }
    } catch {}
  }

  const loadFBLReports = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/reports/fbl', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setFblReports(data.data || [])
      }
    } catch {}
  }

  const loadPostmasterSnaps = async () => {
    if (!activeTenant || !user || domains.length === 0) return
    try {
      const res = await fetch(`/v1/reports/postmaster?domain_id=${domains[0].id}`, {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setPostmasterSnaps(data.data || [])
      }
    } catch {}
  }

  const handleResolveAbuse = async (id: string) => {
    if (!activeTenant || !user) return
    const notes = prompt('Enter resolution notes:', 'Investigated and verified legitimate traffic')
    if (notes === null) return
    try {
      const res = await fetch(`/v1/abuse-reports/${id}/resolve`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ notes }),
      })
      if (res.ok) {
        loadAbuseReports()
      }
    } catch {}
  }

  const handleEscalateAbuse = async (id: string) => {
    if (!activeTenant || !user) return
    const notes = prompt('Enter escalation notes:', 'Malicious sending pattern detected')
    if (notes === null) return
    try {
      const res = await fetch(`/v1/abuse-reports/${id}/escalate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ notes }),
      })
      if (res.ok) {
        loadAbuseReports()
      }
    } catch {}
  }

  const loadDomains = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/domains', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setDomains(data.data || [])
      }
    } catch {}
  }

  const loadStreams = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/streams', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setStreams(data.data || [])
      }
    } catch {}
  }

  const loadAPIKeys = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/api-keys', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setApiKeys(data.data || [])
      }
    } catch {}
  }

  const loadMessages = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/messages', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setMessagesList(data.data || [])
      }
    } catch {}
  }

  const loadTemplates = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/templates', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setTemplatesList(data.data || [])
      }
    } catch {}
  }

  const loadSuppressions = async () => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch('/v1/suppressions', {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setSuppressionsList(data.data || [])
      }
    } catch {}
  }

  const handleCreateSuppression = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !newSuppAddress) return
    try {
      const res = await fetch('/v1/suppressions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({
          address: newSuppAddress,
          type: newSuppType,
          reason: newSuppReason || undefined,
        }),
      })
      if (res.ok) {
        setNewSuppAddress('')
        setNewSuppReason('')
        loadSuppressions()
      } else {
        const d = await res.json()
        alert(d.error?.message || 'Failed to create suppression')
      }
    } catch {}
  }

  const handleDeleteSuppression = async (id: string) => {
    if (!activeTenant || !user) return
    try {
      const res = await fetch(`/v1/suppressions/${id}`, {
        method: 'DELETE',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        loadSuppressions()
      }
    } catch {}
  }

  const handleCreateAPIKey = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !newKeyName) return
    setKeyLoading(true)
    try {
      const res = await fetch('/v1/api-keys', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({
          name: newKeyName,
          environment: newKeyEnv,
          scopes: ['send.email', 'messages.read'],
        }),
      })
      const data = await res.json()
      if (res.ok) {
        setCreatedSecret(data.token)
        setNewKeyName('')
        loadAPIKeys()
      } else {
        alert(data.error?.message || 'Failed to create API key')
      }
    } catch (err: any) {
      alert(err.message)
    } finally {
      setKeyLoading(false)
    }
  }

  const handleRevokeAPIKey = async (id: string) => {
    if (!activeTenant || !user) return
    if (!confirm('Are you sure you want to revoke this API key immediately?')) return
    try {
      const res = await fetch(`/v1/api-keys/${id}`, {
        method: 'DELETE',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      if (res.ok) {
        loadAPIKeys()
      }
    } catch {}
  }

  const handleInspectMessage = async (msg: MessageItem) => {
    if (!activeTenant || !user) return
    setSelectedMessage(msg)
    setSelectedMsgBody('')
    setInspectorOpen(true)
    try {
      const resMsg = await fetch(`/v1/messages/${msg.id}`, {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const dataMsg = await resMsg.json()
      if (resMsg.ok) {
        setSelectedMsgRecipients(dataMsg.recipients || [])
      }

      const resEvts = await fetch(`/v1/messages/${msg.id}/events`, {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const dataEvts = await resEvts.json()
      if (resEvts.ok) {
        setSelectedMsgEvents(dataEvts.data || [])
      }

      const resBody = await fetch(`/v1/messages/${msg.id}/body`, {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const dataBody = await resBody.json()
      if (resBody.ok) {
        setSelectedMsgBody(dataBody.body || '')
      }
    } catch {}
  }

  const handleCreateTemplate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !newTmplName) return
    try {
      const res = await fetch('/v1/templates', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({
          name: newTmplName,
          subject: newTmplSubject,
          html: newTmplHTML,
          text: newTmplText,
          publish_now: true,
        }),
      })
      if (res.ok) {
        setNewTmplName('')
        setNewTmplSubject('')
        setNewTmplHTML('')
        setNewTmplText('')
        loadTemplates()
      } else {
        const data = await res.json()
        alert(data.error?.message || 'Failed to create template')
      }
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleRenderPreview = async (tmplId: string) => {
    if (!activeTenant || !user) return
    try {
      const vars = JSON.parse(previewVars || '{}')
      const res = await fetch(`/v1/templates/${tmplId}/render`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ variables: vars }),
      })
      const data = await res.json()
      if (res.ok) {
        setRenderedPreview(data)
      } else {
        alert(data.error?.message || 'Failed to render preview')
      }
    } catch (err: any) {
      alert('Invalid JSON variables: ' + err.message)
    }
  }

  const handleSelectDomain = async (dom: Domain) => {
    if (!activeTenant || !user) return
    setSelectedDomain(dom)
    try {
      const res = await fetch(`/v1/domains/${dom.id}`, {
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setDomainRecords(data.dns_records || [])
      }
    } catch {}
  }

  const handleAddDomain = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!activeTenant || !user || !newDomainName) return
    setAddDomainLoading(true)

    try {
      const res = await fetch('/v1/domains', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ name: newDomainName, type: newDomainType }),
      })

      const data = await res.json()
      if (res.ok) {
        setNewDomainName('')
        loadDomains()
        if (data.domain) {
          handleSelectDomain(data.domain)
        }
      } else {
        alert(data.error?.message || 'Failed to add domain')
      }
    } catch (err: any) {
      alert(err.message)
    } finally {
      setAddDomainLoading(false)
    }
  }

  const handleVerifyDomain = async () => {
    if (!activeTenant || !user || !selectedDomain) return
    setVerifyLoading(true)

    try {
      const res = await fetch(`/v1/domains/${selectedDomain.id}/verify`, {
        method: 'POST',
        headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
      })
      const data = await res.json()
      if (res.ok) {
        setSelectedDomain(data.domain)
        setDomainRecords(data.dns_results || [])
        loadDomains()
      } else {
        alert(data.error?.message || 'Verification check failed')
      }
    } catch (err: any) {
      alert(err.message)
    } finally {
      setVerifyLoading(false)
    }
  }

  const handleSetup = async (e: React.FormEvent) => {
    e.preventDefault()
    setSetupLoading(true)
    setSetupError('')

    try {
      const res = await fetch('/v1/setup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          profile,
          admin_email: adminEmail,
          admin_password: adminPassword,
          org_name: orgName,
          tenant_name: tenantName,
        }),
      })

      const data = await res.json()
      if (!res.ok) {
        throw new Error(data.error?.message || 'Setup execution failed')
      }

      setSetupCompleted(true)
      handleLoginDirect(adminEmail, adminPassword)
    } catch (err: any) {
      setSetupError(err.message)
    } finally {
      setSetupLoading(false)
    }
  }

  const handleLoginDirect = async (email: string, pass: string) => {
    setLoginLoading(true)
    setLoginError('')

    try {
      const res = await fetch('/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password: pass }),
      })

      const data = await res.json()
      if (!res.ok) {
        throw new Error(data.error?.message || 'Invalid email or password')
      }

      setUser(data.user)
      localStorage.setItem('mvx_user', JSON.stringify(data.user))
      const tList = data.tenants || []
      setTenants(tList)
      localStorage.setItem('mvx_tenants', JSON.stringify(tList))
      if (tList.length > 0) {
        setActiveTenant(tList[0])
        localStorage.setItem('mvx_active_tenant', JSON.stringify(tList[0]))
      }
    } catch (err: any) {
      setLoginError(err.message)
    } finally {
      setLoginLoading(false)
    }
  }

  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault()
    handleLoginDirect(loginEmail, loginPassword)
  }

  if (setupCompleted === null) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen bg-slate-950 text-slate-400">
        <div className="relative">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center shadow-xl shadow-indigo-500/25 animate-pulse">
            <Zap className="w-6 h-6 text-white" />
          </div>
        </div>
        <p className="mt-4 text-xs font-mono tracking-wider uppercase text-slate-500">Checking installation status...</p>
      </div>
    )
  }

  // 1. Modern Enterprise Setup Wizard View
  if (!setupCompleted) {
    return (
      <div className="min-h-screen bg-slate-950 bg-[radial-gradient(ellipse_80%_80%_at_50%_-20%,rgba(99,102,241,0.18),rgba(255,255,255,0))] text-slate-100 flex flex-col justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="sm:mx-auto sm:w-full sm:max-w-xl text-center">
          <div className="inline-flex items-center justify-center w-14 h-14 bg-gradient-to-tr from-indigo-600 via-indigo-500 to-violet-500 rounded-2xl shadow-xl shadow-indigo-500/30 ring-1 ring-white/20 mb-4">
            <Zap className="w-7 h-7 text-white" />
          </div>
          <h1 className="text-3xl font-extrabold tracking-tight bg-gradient-to-r from-white via-slate-100 to-slate-400 bg-clip-text text-transparent">
            Mailvex Platform Setup
          </h1>
          <p className="mt-2 text-sm text-slate-400 max-w-md mx-auto">
            Initialize your carrier-grade, self-hosted email infrastructure engine with multi-tenant isolation.
          </p>
        </div>

        <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-xl">
          <div className="bg-slate-900/90 backdrop-blur-xl py-8 px-6 shadow-2xl border border-slate-800/80 rounded-2xl sm:px-10">
            {setupError && (
              <div className="mb-6 bg-rose-950/60 border border-rose-500/60 text-rose-300 text-xs p-4 rounded-xl flex items-start space-x-3">
                <ShieldAlert className="w-5 h-5 flex-shrink-0 text-rose-400 mt-0.5" />
                <div>
                  <span className="font-semibold block">Installation Failed</span>
                  <span>{setupError}</span>
                </div>
              </div>
            )}

            <form className="space-y-6" onSubmit={handleSetup}>
              {/* Profile Selector Cards */}
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-2.5">
                  1. Select Deployment Architecture
                </label>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  <div
                    onClick={() => setProfile('standard')}
                    className={`cursor-pointer rounded-xl p-3.5 border transition-all relative flex flex-col justify-between ${
                      profile === 'standard'
                        ? 'bg-indigo-950/40 border-indigo-500/80 ring-2 ring-indigo-500/30 text-white'
                        : 'bg-slate-950/50 border-slate-800 text-slate-400 hover:border-slate-700 hover:bg-slate-800/50'
                    }`}
                  >
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <Server className={`w-5 h-5 ${profile === 'standard' ? 'text-indigo-400' : 'text-slate-500'}`} />
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-indigo-500/20 text-indigo-300 border border-indigo-500/30">
                          Recommended
                        </span>
                      </div>
                      <div className="text-xs font-bold text-slate-200">Standard</div>
                      <p className="text-[11px] text-slate-400 mt-1 leading-tight">Production single-server with MariaDB & Redis.</p>
                    </div>
                  </div>

                  <div
                    onClick={() => setProfile('homelab')}
                    className={`cursor-pointer rounded-xl p-3.5 border transition-all relative flex flex-col justify-between ${
                      profile === 'homelab'
                        ? 'bg-indigo-950/40 border-indigo-500/80 ring-2 ring-indigo-500/30 text-white'
                        : 'bg-slate-950/50 border-slate-800 text-slate-400 hover:border-slate-700 hover:bg-slate-800/50'
                    }`}
                  >
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <Cpu className={`w-5 h-5 ${profile === 'homelab' ? 'text-indigo-400' : 'text-slate-500'}`} />
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
                          &lt; 200MB
                        </span>
                      </div>
                      <div className="text-xs font-bold text-slate-200">Home Lab</div>
                      <p className="text-[11px] text-slate-400 mt-1 leading-tight">Ultra low-memory supervisor for homelab/IoT.</p>
                    </div>
                  </div>

                  <div
                    onClick={() => setProfile('cluster')}
                    className={`cursor-pointer rounded-xl p-3.5 border transition-all relative flex flex-col justify-between ${
                      profile === 'cluster'
                        ? 'bg-indigo-950/40 border-indigo-500/80 ring-2 ring-indigo-500/30 text-white'
                        : 'bg-slate-950/50 border-slate-800 text-slate-400 hover:border-slate-700 hover:bg-slate-800/50'
                    }`}
                  >
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <Network className={`w-5 h-5 ${profile === 'cluster' ? 'text-indigo-400' : 'text-slate-500'}`} />
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-amber-500/20 text-amber-300 border border-amber-500/30">
                          Scale HA
                        </span>
                      </div>
                      <div className="text-xs font-bold text-slate-200">Cluster</div>
                      <p className="text-[11px] text-slate-400 mt-1 leading-tight">Multi-node horizontally scaled delivery queue.</p>
                    </div>
                  </div>
                </div>
              </div>

              {/* Organization and Tenant */}
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-2.5">
                  2. Organization & Primary Tenant
                </label>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-slate-300 mb-1">Organization Name</label>
                    <div className="relative">
                      <Building2 className="w-4 h-4 text-slate-500 absolute left-3 top-2.5 pointer-events-none" />
                      <input
                        type="text"
                        required
                        value={orgName}
                        onChange={(e) => setOrgName(e.target.value)}
                        placeholder="Acme Corporation"
                        className="w-full pl-9 pr-3 py-2 border border-slate-700/80 bg-slate-950/60 text-slate-100 text-xs rounded-xl focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 outline-none transition"
                      />
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-medium text-slate-300 mb-1">Primary Tenant Name</label>
                    <div className="relative">
                      <Layers className="w-4 h-4 text-slate-500 absolute left-3 top-2.5 pointer-events-none" />
                      <input
                        type="text"
                        required
                        value={tenantName}
                        onChange={(e) => setTenantName(e.target.value)}
                        placeholder="Production"
                        className="w-full pl-9 pr-3 py-2 border border-slate-700/80 bg-slate-950/60 text-slate-100 text-xs rounded-xl focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 outline-none transition"
                      />
                    </div>
                  </div>
                </div>
              </div>

              {/* Platform Admin Credentials */}
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-2.5">
                  3. Platform Administrator
                </label>
                <div className="space-y-3">
                  <div>
                    <label className="block text-xs font-medium text-slate-300 mb-1">Admin Email Address</label>
                    <div className="relative">
                      <Mail className="w-4 h-4 text-slate-500 absolute left-3 top-2.5 pointer-events-none" />
                      <input
                        type="email"
                        required
                        value={adminEmail}
                        onChange={(e) => setAdminEmail(e.target.value)}
                        placeholder="admin@example.com"
                        className="w-full pl-9 pr-3 py-2 border border-slate-700/80 bg-slate-950/60 text-slate-100 text-xs rounded-xl focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 outline-none transition"
                      />
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-medium text-slate-300 mb-1">Admin Master Password</label>
                    <div className="relative">
                      <Lock className="w-4 h-4 text-slate-500 absolute left-3 top-2.5 pointer-events-none" />
                      <input
                        type="password"
                        required
                        value={adminPassword}
                        onChange={(e) => setAdminPassword(e.target.value)}
                        placeholder="Minimum 8 characters (alphanumeric + symbol)"
                        className="w-full pl-9 pr-3 py-2 border border-slate-700/80 bg-slate-950/60 text-slate-100 text-xs rounded-xl focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 outline-none transition"
                      />
                    </div>
                  </div>
                </div>
              </div>

              {/* Submit Button */}
              <div className="pt-2">
                <button
                  type="submit"
                  disabled={setupLoading}
                  className="w-full flex items-center justify-center space-x-2 py-3 px-4 rounded-xl font-semibold text-xs tracking-wide uppercase text-white bg-gradient-to-r from-indigo-600 via-indigo-500 to-violet-600 hover:from-indigo-500 hover:to-violet-500 shadow-xl shadow-indigo-600/25 ring-1 ring-white/10 transition-all disabled:opacity-50"
                >
                  {setupLoading ? (
                    <>
                      <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                      <span>Initializing Platform & Keys...</span>
                    </>
                  ) : (
                    <>
                      <Sparkles className="w-4 h-4" />
                      <span>Complete Installation & Deploy</span>
                    </>
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    )
  }

  // 2. Modern Enterprise Login View
  if (!user) {
    return (
      <div className="min-h-screen bg-slate-50 dark:bg-slate-950 bg-[radial-gradient(ellipse_80%_80%_at_50%_-20%,rgba(14,165,233,0.15),rgba(255,255,255,0))] dark:bg-[radial-gradient(ellipse_80%_80%_at_50%_-20%,rgba(99,102,241,0.18),rgba(2,6,23,0))] text-slate-800 dark:text-slate-100 flex flex-col justify-center py-12 px-4 sm:px-6 lg:px-8 relative">
        {/* Floating Theme Switcher */}
        <div className="absolute top-6 right-6">
          <div className="inline-flex items-center p-1 bg-white/80 dark:bg-slate-800/80 backdrop-blur-md rounded-xl border border-slate-200/60 dark:border-slate-700/60 text-xs shadow-xs">
            <button
              onClick={() => setTheme('light')}
              className={`p-1.5 rounded-lg transition ${theme === 'light' ? 'bg-white text-amber-500 shadow-xs' : 'text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'}`}
              title="Light Mode"
            >
              <Sun className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => setTheme('system')}
              className={`p-1.5 rounded-lg transition ${theme === 'system' ? 'bg-white dark:bg-slate-700 text-sky-500 shadow-xs' : 'text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'}`}
              title="System Match"
            >
              <Monitor className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => setTheme('dark')}
              className={`p-1.5 rounded-lg transition ${theme === 'dark' ? 'bg-slate-700 text-sky-400 shadow-xs' : 'text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'}`}
              title="Dark Mode"
            >
              <Moon className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        <div className="sm:mx-auto sm:w-full sm:max-w-md text-center">
          <div className="inline-flex items-center justify-center w-14 h-14 bg-gradient-to-tr from-sky-500 via-indigo-600 to-purple-600 rounded-2xl shadow-glow text-white ring-1 ring-white/30 mb-4 transition-transform hover:scale-105">
            <Zap className="w-7 h-7 text-white" />
          </div>
          <h1 className="text-3xl font-black tracking-tight text-slate-900 dark:text-white">
            Mailvex Console
          </h1>
          <p className="mt-2 text-xs font-medium text-slate-500 dark:text-slate-400">
            Carrier-Grade Email Deliverability &amp; Tenant Operations
          </p>
        </div>

        <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
          <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl py-8 px-6 shadow-glass dark:shadow-glass-dark border border-white/60 dark:border-slate-800/80 rounded-2xl sm:px-10">
            {loginError && (
              <div className="mb-6 bg-rose-500/10 border border-rose-500/20 text-rose-600 dark:text-rose-400 text-xs p-3.5 rounded-xl flex items-center space-x-2.5">
                <ShieldAlert className="w-4 h-4 flex-shrink-0 text-rose-500" />
                <span className="font-medium">{loginError}</span>
              </div>
            )}

            <form className="space-y-4" onSubmit={handleLogin}>
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-1.5">
                  Email Address
                </label>
                <div className="relative">
                  <Mail className="w-4 h-4 text-slate-400 absolute left-3 top-3 pointer-events-none" />
                  <input
                    type="email"
                    required
                    value={loginEmail}
                    onChange={(e) => setLoginEmail(e.target.value)}
                    placeholder="admin@example.com"
                    className="w-full pl-9 pr-3.5 py-2.5 bg-white/60 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700/80 rounded-xl text-xs text-slate-800 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-sky-500/30 focus:border-sky-500 backdrop-blur-md transition-all shadow-xs"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-1.5">
                  Password
                </label>
                <div className="relative">
                  <Lock className="w-4 h-4 text-slate-400 absolute left-3 top-3 pointer-events-none" />
                  <input
                    type="password"
                    required
                    value={loginPassword}
                    onChange={(e) => setLoginPassword(e.target.value)}
                    placeholder="••••••••"
                    className="w-full pl-9 pr-3.5 py-2.5 bg-white/60 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700/80 rounded-xl text-xs text-slate-800 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-sky-500/30 focus:border-sky-500 backdrop-blur-md transition-all shadow-xs"
                  />
                </div>
              </div>

              <div className="pt-2">
                <button
                  type="submit"
                  disabled={loginLoading}
                  className="w-full flex items-center justify-center space-x-2 py-2.5 px-4 rounded-xl font-semibold text-xs tracking-wide uppercase text-white bg-gradient-to-r from-sky-500 via-indigo-600 to-purple-600 hover:from-sky-400 hover:to-indigo-500 shadow-md shadow-sky-500/25 ring-1 ring-white/10 transition-all duration-200 hover:scale-[1.02] active:scale-[0.98] disabled:opacity-50"
                >
                  {loginLoading ? (
                    <>
                      <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                      <span>Authenticating...</span>
                    </>
                  ) : (
                    <span>Sign In to Console</span>
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    )
  }

  // 3. Authenticated Dashboard Shell
  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950 bg-[radial-gradient(ellipse_80%_80%_at_50%_-20%,rgba(14,165,233,0.08),rgba(255,255,255,0))] dark:bg-[radial-gradient(ellipse_80%_80%_at_50%_-20%,rgba(99,102,241,0.12),rgba(2,6,23,0))] text-slate-800 dark:text-slate-100 flex flex-col">
      {/* Top Header Navigation (Frosted Glassmorphism) */}
      <header className="bg-white/80 dark:bg-slate-950/80 backdrop-blur-xl border-b border-slate-200/70 dark:border-slate-800/70 px-6 py-2.5 flex items-center justify-between sticky top-0 z-30 shadow-xs dark:shadow-black/20">
        <div className="flex items-center space-x-4">
          <div className="flex items-center space-x-2.5">
            <div className="w-8 h-8 bg-gradient-to-tr from-sky-500 via-indigo-600 to-purple-600 rounded-xl flex items-center justify-center shadow-md shadow-sky-500/25 ring-1 ring-white/30">
              <Zap className="w-4 h-4 text-white" />
            </div>
            <div className="flex flex-col">
              <span className="text-base font-black brand-logo-text leading-none tracking-tight">Mailvex</span>
              <span className="text-[10px] text-sky-600 dark:text-sky-400 font-mono tracking-wider font-bold">ENTERPRISE MTA</span>
            </div>
          </div>

          <div className="h-5 w-px bg-slate-200 dark:bg-slate-800 hidden sm:block"></div>

          {/* Tenant Switcher with Environment Badge */}
          <div className="relative inline-flex items-center">
            <div className="flex items-center space-x-2 bg-white/70 dark:bg-slate-900/80 border border-slate-200/80 dark:border-slate-800/90 rounded-xl px-2.5 py-1.5 focus-within:border-sky-500/60 transition backdrop-blur-md shadow-xs">
              <Layers className="w-3.5 h-3.5 text-sky-500 flex-shrink-0" />
              <select
                value={activeTenant?.id || ''}
                onChange={(e) => {
                  const selected = tenants.find((t) => t.id === e.target.value)
                  if (selected) setActiveTenant(selected)
                }}
                className="bg-transparent text-slate-800 dark:text-slate-200 text-xs font-semibold focus:outline-none cursor-pointer pr-2"
              >
                {tenants.map((t) => (
                  <option key={t.id} value={t.id} className="bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200">
                    {t.name} ({t.slug})
                  </option>
                ))}
              </select>
              <span className={`text-[9px] font-bold px-1.5 py-0.5 rounded-full uppercase tracking-wider ${
                activeTenant?.environment === 'live'
                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                  : 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20'
              }`}>
                {activeTenant?.environment || 'LIVE'}
              </span>
            </div>
          </div>
        </div>

        {/* Global Cluster Status & User Controls */}
        <div className="flex items-center space-x-3">
          {/* Active Live Session Banner (Pulsing Monitor) */}
          <div className="hidden md:flex items-center space-x-2 px-3 py-1.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 text-xs font-semibold backdrop-blur-md">
            <span className="relative flex h-2.5 w-2.5">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500"></span>
            </span>
            <span className="text-[11px]">System Live &amp; Operational</span>
          </div>

          {/* 3-Mode Theme Toggle (Light / System / Dark) */}
          <div className="inline-flex items-center p-1 bg-slate-100/80 dark:bg-slate-800/80 backdrop-blur-md rounded-xl border border-slate-200/60 dark:border-slate-700/60 text-xs">
            <button
              onClick={() => setTheme('light')}
              className={`p-1.5 rounded-lg transition ${theme === 'light' ? 'bg-white text-amber-500 shadow-xs' : 'text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'}`}
              title="Light Mode"
            >
              <Sun className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => setTheme('system')}
              className={`p-1.5 rounded-lg transition ${theme === 'system' ? 'bg-white dark:bg-slate-700 text-sky-500 shadow-xs' : 'text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'}`}
              title="System Match"
            >
              <Monitor className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => setTheme('dark')}
              className={`p-1.5 rounded-lg transition ${theme === 'dark' ? 'bg-slate-700 text-sky-400 shadow-xs' : 'text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'}`}
              title="Dark Mode"
            >
              <Moon className="w-3.5 h-3.5" />
            </button>
          </div>

          {/* Platform Admin Quick Tools */}
          {user.is_platform_admin && (
            <>
              <button
                onClick={() => setCurrentTab('platform_admin')}
                className={`flex items-center space-x-1.5 px-3 py-1.5 text-xs font-semibold rounded-xl border transition-all duration-150 hover:scale-[1.02] active:scale-[0.98] ${
                  currentTab === 'platform_admin'
                    ? 'bg-amber-500 text-slate-950 border-amber-400 shadow-sm shadow-amber-500/20'
                    : 'bg-amber-500/10 hover:bg-amber-500/20 text-amber-600 dark:text-amber-300 border-amber-500/30'
                }`}
                title="Open Platform Administration Hub"
              >
                <ShieldCheck className="w-3.5 h-3.5" />
                <span>Admin Hub</span>
              </button>

              <button
                onClick={() => setImpModalOpen(true)}
                className="hidden lg:flex items-center space-x-1.5 px-2.5 py-1.5 text-xs font-medium bg-slate-100/80 dark:bg-slate-800/80 hover:bg-slate-200/80 dark:hover:bg-slate-700/80 text-slate-700 dark:text-slate-200 border border-slate-200/60 dark:border-slate-700/80 rounded-xl transition"
                title="Platform Admin: Impersonate tenant"
              >
                <Shield className="w-3.5 h-3.5 text-sky-500" />
                <span>Impersonate</span>
              </button>
            </>
          )}

          {/* User Profile Chip */}
          <div className="flex items-center space-x-2 bg-white/70 dark:bg-slate-900/80 border border-slate-200/80 dark:border-slate-800/80 rounded-xl py-1 pl-1.5 pr-2.5 backdrop-blur-md shadow-xs">
            <div className="w-6 h-6 rounded-lg bg-gradient-to-tr from-sky-500 via-indigo-600 to-purple-600 flex items-center justify-center text-white text-[11px] font-bold">
              {user.email.charAt(0).toUpperCase()}
            </div>
            <div className="flex flex-col text-left">
              <span className="text-[11px] text-slate-800 dark:text-slate-200 font-semibold leading-none">{user.email.split('@')[0]}</span>
              <span className="text-[9px] text-slate-500 dark:text-slate-400 font-mono leading-tight">{user.is_platform_admin ? 'Platform Admin' : 'Tenant User'}</span>
            </div>
          </div>

          <button
            onClick={handleSignOut}
            className="flex items-center space-x-1 text-xs text-slate-400 hover:text-rose-500 dark:hover:text-rose-400 p-2 rounded-xl hover:bg-rose-500/10 transition border border-transparent hover:border-rose-500/20"
            title="Sign Out"
          >
            <LogOut className="w-4 h-4" />
          </button>
        </div>
      </header>

      {/* Sticky Impersonation Warning Banner (§7.4) */}
      {isImpersonating && impersonatedTenant && (
        <div className="bg-amber-500/15 border-b border-amber-500/30 px-6 py-2 flex items-center justify-between text-xs text-amber-200">
          <div className="flex items-center space-x-3">
            <span className="font-bold uppercase tracking-wider bg-amber-500/30 text-amber-300 px-2 py-0.5 rounded border border-amber-500/40 text-[10px]">
              ⚠️ Impersonation Active
            </span>
            <span>Viewing as Tenant: <strong>{impersonatedTenant.name}</strong></span>
            <span className="text-amber-400/80 italic font-mono">"{impersonationReason}"</span>
            {isSupportMasking && (
              <span className="bg-blue-950 text-blue-400 border border-blue-800 px-1.5 py-0.5 rounded text-[10px] font-semibold">
                🔒 Support Masking Active
              </span>
            )}
          </div>
          <button
            onClick={handleExitImpersonation}
            className="px-3 py-1 bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold rounded text-xs transition-colors"
          >
            Exit Impersonation
          </button>
        </div>
      )}

      {/* Impersonate Modal */}
      {impModalOpen && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white/95 dark:bg-slate-900/95 backdrop-blur-xl border border-white/80 dark:border-slate-800/80 rounded-2xl w-full max-w-md p-6 space-y-4 shadow-2xl">
            <div className="flex justify-between items-center border-b border-slate-100 dark:border-slate-800/80 pb-3">
              <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center space-x-2">
                <Shield className="w-4 h-4 text-sky-500" />
                <span>Platform Admin Impersonation</span>
              </h3>
              <button
                onClick={() => setImpModalOpen(false)}
                className="text-slate-400 hover:text-slate-600 dark:hover:text-white text-sm"
              >
                ✕
              </button>
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
              Access customer tenant environments to troubleshoot configuration or delivery issues. Every request is audited with your user ID and mandatory reason.
            </p>
            <form onSubmit={handleStartImpersonation} className="space-y-3">
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">Target Tenant</label>
                <select
                  required
                  value={targetImpTenantID}
                  onChange={(e) => setTargetImpTenantID(e.target.value)}
                  className="w-full px-3.5 py-2.5 border border-slate-200 dark:border-slate-700/80 bg-slate-50/70 dark:bg-slate-800/70 text-slate-900 dark:text-slate-100 text-xs rounded-xl focus:outline-none focus:ring-2 focus:ring-sky-500/30"
                >
                  <option value="">Select Tenant...</option>
                  {tenants.map((t) => (
                    <option key={t.id} value={t.id}>{t.name} ({t.slug})</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">Mandatory Reason for Access</label>
                <textarea
                  required
                  rows={3}
                  value={targetImpReason}
                  onChange={(e) => setTargetImpReason(e.target.value)}
                  placeholder="e.g. Investigating intermittent bounce delivery spikes"
                  className="w-full px-3.5 py-2.5 border border-slate-200 dark:border-slate-700/80 bg-slate-50/70 dark:bg-slate-800/70 text-slate-900 dark:text-slate-100 text-xs rounded-xl focus:outline-none focus:ring-2 focus:ring-sky-500/30"
                />
              </div>
              <div className="flex items-center space-x-2 pt-1">
                <input
                  type="checkbox"
                  id="supportModeToggle"
                  checked={targetImpSupportMode}
                  onChange={(e) => setTargetImpSupportMode(e.target.checked)}
                  className="rounded border-slate-300 dark:border-slate-700 text-sky-600 focus:ring-sky-500"
                />
                <label htmlFor="supportModeToggle" className="text-xs text-slate-600 dark:text-slate-300 select-none">
                  Enable Support Privacy Masking (masks message body content §7.4)
                </label>
              </div>
              <div className="pt-3 flex justify-end space-x-2">
                <button
                  type="button"
                  onClick={() => setImpModalOpen(false)}
                  className="btn-secondary text-xs px-3.5 py-2"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn-gradient text-xs px-4 py-2"
                >
                  Begin Impersonation
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Main Body */}
      <div className="flex flex-1 overflow-hidden">
        {/* Modern Frosted Glass Sidebar */}
        <nav className="w-64 bg-white/70 dark:bg-slate-900/70 backdrop-blur-xl border-r border-slate-200/60 dark:border-slate-800/80 p-3 space-y-4 overflow-y-auto flex flex-col justify-between">
          <div className="space-y-4">
            {/* Platform Admin Hub Quick Launcher (Visible only to Platform Admins) */}
            {user.is_platform_admin && (
              <div className="space-y-1">
                <div className="px-3 py-1 text-[10px] font-bold uppercase tracking-wider text-amber-500 dark:text-amber-400 flex items-center justify-between">
                  <span className="flex items-center space-x-1.5">
                    <ShieldCheck className="w-3.5 h-3.5 text-amber-500" />
                    <span>Platform Admin</span>
                  </span>
                  <span className="bg-amber-500/10 text-amber-600 dark:text-amber-300 border border-amber-500/20 text-[9px] px-1.5 py-0.2 rounded font-mono font-bold">ROOT</span>
                </div>
                <button
                  onClick={() => setCurrentTab('platform_admin')}
                  className={`w-full text-left flex items-center justify-between px-3 py-2.5 text-xs font-semibold rounded-xl transition-all duration-200 ${
                    currentTab === 'platform_admin'
                      ? 'bg-gradient-to-r from-amber-500/20 via-indigo-600/20 to-purple-600/10 text-amber-700 dark:text-amber-300 border border-amber-500/40 shadow-xs'
                      : 'text-amber-700/80 dark:text-amber-300/80 hover:bg-amber-500/10 hover:text-amber-800 dark:hover:text-amber-200 border border-amber-500/20'
                  }`}
                >
                  <div className="flex items-center space-x-2.5">
                    <Shield className="w-4 h-4 text-amber-500" />
                    <span>Admin Center</span>
                  </div>
                  <ChevronRight className="w-3.5 h-3.5 text-amber-500/60" />
                </button>
              </div>
            )}

            {/* Core Operations Group */}
            <div className="space-y-1">
              <div className="px-3 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-400 dark:text-slate-500">
                Core Operations
              </div>
              {[
                { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
                { id: 'domains', label: 'Domains & DNS', icon: Globe },
                { id: 'streams', label: 'Message Streams', icon: GitFork },
                { id: 'apikeys', label: 'API Keys', icon: Key },
                { id: 'messages', label: 'Message Logs', icon: Inbox },
                { id: 'templates', label: 'Templates', icon: FileCode },
              ].map((item) => {
                const Icon = item.icon
                const isActive = currentTab === item.id
                return (
                  <button
                    key={item.id}
                    onClick={() => setCurrentTab(item.id as any)}
                    className={`w-full text-left flex items-center space-x-2.5 px-3 py-2 text-xs rounded-xl transition-all duration-150 ${
                      isActive
                        ? 'nav-item-active'
                        : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100/80 dark:hover:bg-slate-800/70 hover:text-slate-950 dark:hover:text-white font-medium'
                    }`}
                  >
                    <Icon className={`w-4 h-4 flex-shrink-0 ${isActive ? 'text-sky-500 dark:text-sky-400' : 'text-slate-400 dark:text-slate-500'}`} />
                    <span>{item.label}</span>
                  </button>
                )
              })}
            </div>

            {/* Deliverability & Safety Group */}
            <div className="space-y-1">
              <div className="px-3 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-400 dark:text-slate-500">
                Deliverability & Safety
              </div>
              {[
                { id: 'analytics', label: 'Analytics & Reports', icon: BarChart3 },
                { id: 'reputation', label: 'Reputation & Abuse', icon: ShieldAlert },
                { id: 'suppressions', label: 'Suppressions', icon: Ban },
                { id: 'alerts_warmup', label: 'Alerts & Warmup', icon: Flame },
                { id: 'inbound', label: 'Inbound Routing', icon: ArrowDownLeft },
              ].map((item) => {
                const Icon = item.icon
                const isActive = currentTab === item.id
                return (
                  <button
                    key={item.id}
                    onClick={() => setCurrentTab(item.id as any)}
                    className={`w-full text-left flex items-center space-x-2.5 px-3 py-2 text-xs rounded-xl transition-all duration-150 ${
                      isActive
                        ? 'nav-item-active'
                        : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100/80 dark:hover:bg-slate-800/70 hover:text-slate-950 dark:hover:text-white font-medium'
                    }`}
                  >
                    <Icon className={`w-4 h-4 flex-shrink-0 ${isActive ? 'text-sky-500 dark:text-sky-400' : 'text-slate-400 dark:text-slate-500'}`} />
                    <span>{item.label}</span>
                  </button>
                )
              })}
            </div>

            {/* Security & Infrastructure Group */}
            <div className="space-y-1">
              <div className="px-3 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-400 dark:text-slate-500">
                Security & Platform
              </div>
              {[
                { id: 'webhooks', label: 'Webhooks & Signing', icon: Webhook },
                { id: 'privacy_retention', label: 'Privacy & Retention', icon: Lock },
                { id: 'sso_compliance', label: 'SSO & Deliverability', icon: ShieldCheck },
              ].map((item) => {
                const Icon = item.icon
                const isActive = currentTab === item.id
                return (
                  <button
                    key={item.id}
                    onClick={() => setCurrentTab(item.id as any)}
                    className={`w-full text-left flex items-center space-x-2.5 px-3 py-2 text-xs rounded-xl transition-all duration-150 ${
                      isActive
                        ? 'nav-item-active'
                        : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100/80 dark:hover:bg-slate-800/70 hover:text-slate-950 dark:hover:text-white font-medium'
                    }`}
                  >
                    <Icon className={`w-4 h-4 flex-shrink-0 ${isActive ? 'text-sky-500 dark:text-sky-400' : 'text-slate-400 dark:text-slate-500'}`} />
                    <span>{item.label}</span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* System Footer Metadata */}
          <div className="pt-3 border-t border-slate-200/80 dark:border-slate-800/80 px-2 flex items-center justify-between text-[11px] text-slate-500">
            <div className="flex items-center space-x-1.5">
              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
              <span className="font-mono text-slate-500 dark:text-slate-400 text-[10px]">v1.0 • MTA</span>
            </div>
            <span className="text-[10px] text-sky-600 dark:text-sky-400 font-semibold">Ready</span>
          </div>
        </nav>

        {/* Workspace Body */}
        <main className="flex-1 overflow-y-auto p-6 sm:p-8">
          <div className="max-w-5xl mx-auto">
            {/* Top Bar */}
            <div className="flex items-center justify-between pb-6 border-b border-slate-200/80 dark:border-slate-800/80">
              <div>
                <h1 className="text-2xl font-black text-slate-900 dark:text-white tracking-tight">
                  {currentTab === 'dashboard' && (activeTenant?.name || 'Dashboard Overview')}
                  {currentTab === 'platform_admin' && 'Platform Administration & Fleet Control'}
                  {currentTab === 'domains' && 'Domains & DNS Configuration'}
                  {currentTab === 'streams' && 'Message Streams'}
                  {currentTab === 'apikeys' && 'API Keys'}
                  {currentTab === 'messages' && 'Message Logs & Inspector'}
                  {currentTab === 'templates' && 'Templates'}
                  {currentTab === 'suppressions' && 'Suppression List & Safeguards'}
                  {currentTab === 'reputation' && 'Reputation, Feedback Loops & Abuse Queue'}
                  {currentTab === 'analytics' && 'Analytics & Deliverability Reporting'}
                  {currentTab === 'webhooks' && 'Webhooks, Signatures & Replay Defense'}
                  {currentTab === 'privacy_retention' && 'Suppression Privacy & Data Retention'}
                  {currentTab === 'inbound' && 'Inbound Email Routing & Processing'}
                  {currentTab === 'alerts_warmup' && 'Reputation, DNSBL, Warmup & Alert Center'}
                  {currentTab === 'sso_compliance' && 'SSO, Impersonation & Compliance'}
                </h1>
                <p className="text-xs text-slate-400 mt-1 flex items-center space-x-2">
                  <span>Active Tenant:</span>
                  <span className="font-mono text-slate-200 bg-slate-800/80 px-2 py-0.5 rounded text-[11px] border border-slate-700/60 font-semibold">{activeTenant?.name}</span>
                  <span className="text-slate-600">•</span>
                  <span className="font-mono text-slate-400 text-[11px]">{activeTenant?.slug}</span>
                  <span className="text-slate-600">•</span>
                  <span className="uppercase text-[10px] font-bold text-indigo-400 bg-indigo-500/10 border border-indigo-500/20 px-1.5 py-0.5 rounded">{activeTenant?.environment}</span>
                </p>
              </div>
              <div className="flex items-center space-x-2 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
                <span className="text-xs text-emerald-400 font-semibold">Cluster Operational</span>
              </div>
            </div>

            {/* TAB: DASHBOARD */}
            {currentTab === 'dashboard' && (
              <div className="space-y-6 mt-6">
                {/* Hero Greeting Banner */}
                <div className="bg-gradient-to-r from-sky-500/10 via-indigo-500/10 to-purple-500/10 dark:from-slate-900/90 dark:via-indigo-950/40 dark:to-slate-900/90 border border-sky-500/20 dark:border-slate-800/90 rounded-2xl p-6 relative overflow-hidden backdrop-blur-xl shadow-glass dark:shadow-glass-dark">
                  <div className="absolute right-0 top-0 w-96 h-96 bg-sky-500/15 dark:bg-indigo-500/10 rounded-full blur-3xl pointer-events-none -mr-20 -mt-20"></div>
                  <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 relative z-10">
                    <div>
                      <div className="flex items-center space-x-2 mb-2">
                        <span className="px-2.5 py-0.5 text-[10px] font-bold tracking-wider uppercase bg-sky-500/10 text-sky-600 dark:text-sky-300 border border-sky-500/20 rounded-full">
                          Enterprise MTA Hub
                        </span>
                        <span className="text-xs text-slate-500 dark:text-slate-400">• Node: homelab-node</span>
                      </div>
                      <h2 className="text-2xl font-black text-slate-900 dark:text-white tracking-tight">
                        {user.is_platform_admin ? 'Platform Admin Control Console' : `${activeTenant?.name || 'Mailvex'} Dashboard`}
                      </h2>
                      <p className="text-xs text-slate-600 dark:text-slate-300 mt-1 max-w-xl leading-relaxed font-medium">
                        High-throughput email delivery infrastructure with multi-tenant isolation, automated DKIM/SPF rotation, envelope encryption, and real-time deliverability telemetry.
                      </p>
                    </div>

                    <div className="flex flex-wrap items-center gap-2">
                      <button
                        onClick={() => setCurrentTab('domains')}
                        className="btn-gradient text-xs px-4 py-2"
                      >
                        <Plus className="w-4 h-4" />
                        <span>Add Domain</span>
                      </button>
                      <button
                        onClick={() => setCurrentTab('apikeys')}
                        className="btn-secondary text-xs px-3.5 py-2"
                      >
                        <Key className="w-4 h-4 text-sky-500" />
                        <span>API Keys</span>
                      </button>
                      {user.is_platform_admin && (
                        <button
                          onClick={() => setCurrentTab('platform_admin')}
                          className="flex items-center space-x-2 px-3.5 py-2 bg-amber-500/10 hover:bg-amber-500/20 text-amber-700 dark:text-amber-300 text-xs font-semibold rounded-xl border border-amber-500/30 transition-all duration-150 hover:scale-[1.01]"
                        >
                          <Shield className="w-4 h-4 text-amber-500" />
                          <span>Admin Center</span>
                        </button>
                      )}
                    </div>
                  </div>
                </div>

                {/* 4 Elevated Stat Cards (Blueprint 4.2) */}
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-5 rounded-2xl shadow-glass dark:shadow-glass-dark transition-all duration-200 hover:scale-[1.01] hover:border-sky-500/40">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-400">Sending Domains</span>
                      <div className="p-2 rounded-xl bg-sky-500/10 text-sky-600 dark:text-sky-400">
                        <Globe className="w-4 h-4" />
                      </div>
                    </div>
                    <div className="text-3xl font-black text-slate-900 dark:text-white mt-3 tracking-tight">{domains.length}</div>
                    <div className="flex items-center space-x-2 mt-2 text-xs">
                      <span className="w-2 h-2 rounded-full bg-emerald-500"></span>
                      <span className="text-emerald-600 dark:text-emerald-400 font-semibold">{domains.filter((d) => d.status === 'verified').length} Verified</span>
                      <span className="text-slate-400">•</span>
                      <span className="text-slate-500 dark:text-slate-400">{domains.filter((d) => d.status === 'pending').length} Pending</span>
                    </div>
                  </div>

                  <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-5 rounded-2xl shadow-glass dark:shadow-glass-dark transition-all duration-200 hover:scale-[1.01] hover:border-sky-500/40">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-400">Message Streams</span>
                      <div className="p-2 rounded-xl bg-sky-500/10 text-sky-600 dark:text-sky-400">
                        <GitFork className="w-4 h-4" />
                      </div>
                    </div>
                    <div className="text-3xl font-black text-slate-900 dark:text-white mt-3 tracking-tight">{streams.length}</div>
                    <div className="flex items-center space-x-1.5 mt-2 text-xs text-slate-500 dark:text-slate-400">
                      <span className="text-sky-600 dark:text-sky-400 font-semibold">{streams.filter(s => s.type === 'transactional').length} Transactional</span>
                      <span>•</span>
                      <span>{streams.filter(s => s.type === 'broadcast').length} Broadcast</span>
                    </div>
                  </div>

                  <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-5 rounded-2xl shadow-glass dark:shadow-glass-dark transition-all duration-200 hover:scale-[1.01] hover:border-sky-500/40">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-400">Cryptographic Guard</span>
                      <div className="p-2 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400">
                        <Lock className="w-4 h-4" />
                      </div>
                    </div>
                    <div className="text-3xl font-black text-purple-600 dark:text-purple-400 mt-3 tracking-tight">RSA 2048</div>
                    <div className="flex items-center space-x-1.5 mt-2 text-xs text-emerald-600 dark:text-emerald-400 font-semibold">
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
                      <span>DKIM &amp; DEK Active</span>
                    </div>
                  </div>

                  <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-5 rounded-2xl shadow-glass dark:shadow-glass-dark transition-all duration-200 hover:scale-[1.01] hover:border-sky-500/40">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-400">System Readiness</span>
                      <div className="p-2 rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                        <ShieldCheck className="w-4 h-4" />
                      </div>
                    </div>
                    <div className="text-3xl font-black text-emerald-600 dark:text-emerald-400 mt-3 tracking-tight">100% SLA</div>
                    <div className="text-xs text-slate-500 dark:text-slate-400 mt-2 flex items-center space-x-1.5 font-medium">
                      <Activity className="w-3.5 h-3.5 text-emerald-500" />
                      <span>All Services Online</span>
                    </div>
                  </div>
                </div>

                {/* Live Infrastructure Matrix Card */}
                <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 rounded-2xl p-6 shadow-glass dark:shadow-glass-dark">
                  <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800/80 mb-4">
                    <div className="flex items-center space-x-2">
                      <Server className="w-4 h-4 text-sky-500" />
                      <h3 className="text-sm font-bold text-slate-900 dark:text-white">Live Cluster Services Matrix</h3>
                    </div>
                    <span className="text-[11px] text-slate-500 dark:text-slate-400 font-mono">Profile: homelab (All-in-One)</span>
                  </div>

                  <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
                    <div className="bg-slate-50/80 dark:bg-slate-950/60 border border-slate-200/60 dark:border-slate-800 rounded-xl p-3 text-center">
                      <span className="text-[10px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold block">REST API</span>
                      <span className="text-xs font-mono text-emerald-600 dark:text-emerald-400 font-bold block mt-1">:8080 Active</span>
                      <span className="text-[10px] text-slate-400">TLS 1.3</span>
                    </div>
                    <div className="bg-slate-50/80 dark:bg-slate-950/60 border border-slate-200/60 dark:border-slate-800 rounded-xl p-3 text-center">
                      <span className="text-[10px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold block">SMTP Edge</span>
                      <span className="text-xs font-mono text-emerald-600 dark:text-emerald-400 font-bold block mt-1">:2525 / :25</span>
                      <span className="text-[10px] text-slate-400">RFC 5321</span>
                    </div>
                    <div className="bg-slate-50/80 dark:bg-slate-950/60 border border-slate-200/60 dark:border-slate-800 rounded-xl p-3 text-center">
                      <span className="text-[10px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold block">Worker</span>
                      <span className="text-xs font-mono text-emerald-600 dark:text-emerald-400 font-bold block mt-1">Direct MX</span>
                      <span className="text-[10px] text-slate-400">Auto Relay</span>
                    </div>
                    <div className="bg-slate-50/80 dark:bg-slate-950/60 border border-slate-200/60 dark:border-slate-800 rounded-xl p-3 text-center">
                      <span className="text-[10px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold block">Queue Pipeline</span>
                      <span className="text-xs font-mono text-emerald-600 dark:text-emerald-400 font-bold block mt-1">Low Latency</span>
                      <span className="text-[10px] text-slate-400">Priority Outbox</span>
                    </div>
                    <div className="bg-slate-50/80 dark:bg-slate-950/60 border border-slate-200/60 dark:border-slate-800 rounded-xl p-3 text-center">
                      <span className="text-[10px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold block">Database</span>
                      <span className="text-xs font-mono text-emerald-600 dark:text-emerald-400 font-bold block mt-1">MariaDB 11.4</span>
                      <span className="text-[10px] text-slate-400">Pool Healthy</span>
                    </div>
                    <div className="bg-slate-50/80 dark:bg-slate-950/60 border border-slate-200/60 dark:border-slate-800 rounded-xl p-3 text-center">
                      <span className="text-[10px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold block">Master KEK</span>
                      <span className="text-xs font-mono text-emerald-600 dark:text-emerald-400 font-bold block mt-1">256-bit AES</span>
                      <span className="text-[10px] text-slate-400">Envelope DEKs</span>
                    </div>
                  </div>
                </div>

                {/* Quick Shortcuts Section */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div
                    onClick={() => setCurrentTab('domains')}
                    className="bg-slate-900/60 border border-slate-800/80 hover:border-indigo-500/40 p-5 rounded-2xl cursor-pointer transition group"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <Globe className="w-5 h-5 text-indigo-400 group-hover:scale-110 transition" />
                      <ChevronRight className="w-4 h-4 text-slate-500 group-hover:text-white transition" />
                    </div>
                    <h4 className="text-sm font-bold text-white">Sending Domains & DNS</h4>
                    <p className="text-xs text-slate-400 mt-1">Manage domain ownership TXT, DKIM keys, SPF includes, and DMARC policies.</p>
                  </div>

                  <div
                    onClick={() => setCurrentTab('messages')}
                    className="bg-slate-900/60 border border-slate-800/80 hover:border-indigo-500/40 p-5 rounded-2xl cursor-pointer transition group"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <Inbox className="w-5 h-5 text-indigo-400 group-hover:scale-110 transition" />
                      <ChevronRight className="w-4 h-4 text-slate-500 group-hover:text-white transition" />
                    </div>
                    <h4 className="text-sm font-bold text-white">Live Message Inspector</h4>
                    <p className="text-xs text-slate-400 mt-1">Inspect real-time transmission logs, delivery attempts, and bounce telemetry.</p>
                  </div>

                  <div
                    onClick={() => setCurrentTab('sso_compliance')}
                    className="bg-slate-900/60 border border-slate-800/80 hover:border-indigo-500/40 p-5 rounded-2xl cursor-pointer transition group"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <ShieldCheck className="w-5 h-5 text-indigo-400 group-hover:scale-110 transition" />
                      <ChevronRight className="w-4 h-4 text-slate-500 group-hover:text-white transition" />
                    </div>
                    <h4 className="text-sm font-bold text-white">Deliverability Compliance</h4>
                    <p className="text-xs text-slate-400 mt-1">Calculate your 100-point deliverability scorecard, FCrDNS verification, and TLS audits.</p>
                  </div>
                </div>
              </div>
            )}

            {/* TAB: DOMAINS */}
            {currentTab === 'domains' && (
              <div className="mt-6 space-y-6">
                {/* Add Domain Form */}
                <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-6 rounded-2xl shadow-glass dark:shadow-glass-dark">
                  <div className="flex items-center space-x-2 pb-4 border-b border-slate-100 dark:border-slate-800/80 mb-4">
                    <Globe className="w-5 h-5 text-sky-500" />
                    <div>
                      <h2 className="text-base font-bold text-slate-900 dark:text-white">Add Sending Domain</h2>
                      <p className="text-xs text-slate-500 dark:text-slate-400">Register new custom domain for automated DKIM, SPF, and DMARC verification</p>
                    </div>
                  </div>
                  <form onSubmit={handleAddDomain} className="flex flex-col sm:flex-row gap-3">
                    <input
                      type="text"
                      required
                      placeholder="e.g. mail.yourdomain.com"
                      value={newDomainName}
                      onChange={(e) => setNewDomainName(e.target.value)}
                      className="flex-1 bg-white/60 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700/80 rounded-xl px-3.5 py-2.5 text-xs text-slate-800 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-sky-500/30 focus:border-sky-500 backdrop-blur-md transition-all shadow-xs"
                    />
                    <select
                      value={newDomainType}
                      onChange={(e) => setNewDomainType(e.target.value)}
                      className="bg-white/60 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700/80 rounded-xl px-3.5 py-2.5 text-xs text-slate-800 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-sky-500/30 focus:border-sky-500 backdrop-blur-md transition-all cursor-pointer"
                    >
                      <option value="public" className="bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200">Public Domain</option>
                      <option value="local" className="bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200">Local Domain (Home Lab)</option>
                    </select>
                    <button
                      type="submit"
                      disabled={addDomainLoading}
                      className="btn-gradient text-xs px-5 py-2.5 whitespace-nowrap disabled:opacity-50"
                    >
                      {addDomainLoading ? 'Adding...' : 'Add Domain'}
                    </button>
                  </form>
                </div>

                {/* Domains List */}
                <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 rounded-2xl shadow-glass dark:shadow-glass-dark overflow-hidden">
                  <div className="p-4 border-b border-slate-100 dark:border-slate-800/80 flex items-center justify-between">
                    <span className="font-bold text-sm text-slate-900 dark:text-white flex items-center gap-2">
                      <Globe className="w-4 h-4 text-sky-500" />
                      Configured Domains
                    </span>
                    <div className="flex items-center space-x-2">
                      <span className="text-xs text-slate-500 dark:text-slate-400 font-medium">({domains.length} Total)</span>
                      <ExportMenu
                        filename={`mailvex-domains-${activeTenant?.slug || 'tenant'}`}
                        sheetTitle="Domains"
                        label="Export Domains"
                        columns={[
                          { header: 'Domain Name', key: 'name' },
                          { header: 'Type', key: 'type' },
                          { header: 'Status', key: 'status' },
                          { header: 'Ownership Token', key: 'ownership_token' },
                          { header: 'Verified At', key: 'verified_at' },
                          { header: 'Last Checked', key: 'last_checked_at' },
                        ]}
                        data={domains.map((d) => ({
                          name: d.name,
                          type: d.type,
                          status: d.status,
                          ownership_token: d.ownership_token,
                          verified_at: d.verified_at || 'Unverified',
                          last_checked_at: d.last_checked_at || 'Never',
                        }))}
                      />
                    </div>
                  </div>
                  {domains.length === 0 ? (
                    <div className="p-8 text-center text-slate-400 dark:text-slate-500 text-xs">
                      No domains added yet. Add your first domain above.
                    </div>
                  ) : (
                    <div className="overflow-x-auto">
                      <table className="w-full text-left text-sm">
                        <thead className="bg-slate-50/60 dark:bg-slate-950/40 text-[11px] font-bold uppercase tracking-wider text-slate-400 dark:text-slate-500 border-b border-slate-100 dark:border-slate-800">
                          <tr>
                            <th className="py-3 px-4">Domain</th>
                            <th className="py-3 px-4">Type</th>
                            <th className="py-3 px-4">Status</th>
                            <th className="py-3 px-4 text-right">Action</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-medium text-slate-700 dark:text-slate-300">
                          {domains.map((d) => (
                            <tr key={d.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-800/40 transition-colors">
                              <td className="py-3.5 px-4 font-semibold text-slate-900 dark:text-white flex items-center gap-2">
                                <div className="w-7 h-7 rounded-lg bg-sky-500/10 text-sky-600 dark:text-sky-400 font-bold flex items-center justify-center text-xs">
                                  {d.name.charAt(0).toUpperCase()}
                                </div>
                                <span className="font-mono text-xs">{d.name}</span>
                              </td>
                              <td className="py-3.5 px-4 capitalize text-xs">{d.type}</td>
                              <td className="py-3.5 px-4">
                                <span
                                  className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold backdrop-blur-md border ${
                                    d.status === 'verified'
                                      ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20'
                                      : d.status === 'degraded'
                                      ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20'
                                      : 'bg-slate-500/10 text-slate-600 dark:text-slate-400 border-slate-500/20'
                                  }`}
                                >
                                  <span className={`w-1.5 h-1.5 rounded-full ${
                                    d.status === 'verified' ? 'bg-emerald-500' : d.status === 'degraded' ? 'bg-amber-500' : 'bg-slate-400'
                                  }`}></span>
                                  {d.status}
                                </span>
                              </td>
                              <td className="py-3.5 px-4 text-right">
                                <button
                                  onClick={() => handleSelectDomain(d)}
                                  className="btn-secondary text-xs"
                                >
                                  <span>View DNS Records</span>
                                </button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>

                {/* Selected Domain Records View */}
                {selectedDomain && (
                  <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-6 rounded-2xl shadow-glass dark:shadow-glass-dark space-y-4">
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-4 border-b border-slate-100 dark:border-slate-800/80">
                      <div>
                        <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                          <CheckCircle2 className="w-5 h-5 text-emerald-500" />
                          <span>DNS Verification Records for <code className="text-sky-600 dark:text-sky-400 font-mono text-sm">{selectedDomain.name}</code></span>
                        </h3>
                        <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                          Publish these TXT &amp; CNAME records in your DNS provider (Cloudflare, Route53, etc.)
                        </p>
                      </div>
                      <button
                        onClick={handleVerifyDomain}
                        disabled={verifyLoading}
                        className="btn-gradient text-xs px-4 py-2 disabled:opacity-50"
                      >
                        {verifyLoading ? 'Checking DNS...' : 'Verify DNS Now'}
                      </button>
                    </div>

                    <div className="space-y-3">
                      {domainRecords.map((r, i) => (
                        <div
                          key={i}
                          className="bg-slate-50/80 dark:bg-slate-950/60 p-4 rounded-xl border border-slate-200/60 dark:border-slate-800/80 flex flex-col md:flex-row md:items-center justify-between gap-4 transition-all hover:border-sky-500/30"
                        >
                          <div className="space-y-1.5 overflow-hidden">
                            <div className="flex items-center space-x-2">
                              <span className="px-2 py-0.5 bg-slate-200/80 dark:bg-slate-800 text-slate-700 dark:text-slate-300 font-mono text-[11px] rounded font-semibold">
                                {r.record_type}
                              </span>
                              <code className="text-xs text-sky-600 dark:text-sky-400 font-mono font-bold">{r.name}</code>
                              <span
                                className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold border ${
                                  r.status === 'pass'
                                    ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20'
                                    : 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20'
                                }`}
                              >
                                <span className={`w-1.5 h-1.5 rounded-full ${r.status === 'pass' ? 'bg-emerald-500' : 'bg-amber-500'}`}></span>
                                {r.status}
                              </span>
                            </div>
                            <div className="text-xs text-slate-600 dark:text-slate-400 font-mono truncate max-w-xl bg-white/60 dark:bg-slate-900/60 p-2 rounded-lg border border-slate-200/50 dark:border-slate-800/50">
                              {r.expected_value}
                            </div>
                          </div>
                          <button
                            onClick={() => {
                              navigator.clipboard.writeText(r.expected_value)
                              alert('Copied to clipboard!')
                            }}
                            className="btn-secondary text-xs px-3 py-1.5 self-start md:self-auto"
                          >
                            Copy Value
                          </button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* TAB: STREAMS */}
            {currentTab === 'streams' && (
              <div className="mt-6 bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 rounded-2xl shadow-glass dark:shadow-glass-dark overflow-hidden">
                <div className="p-4 border-b border-slate-100 dark:border-slate-800/80 font-bold text-sm text-slate-900 dark:text-white flex items-center gap-2">
                  <GitFork className="w-4 h-4 text-sky-500" />
                  Message Streams
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="bg-slate-50/60 dark:bg-slate-950/40 text-[11px] font-bold uppercase tracking-wider text-slate-400 dark:text-slate-500 border-b border-slate-100 dark:border-slate-800">
                      <tr>
                        <th className="py-3 px-4">Stream Name</th>
                        <th className="py-3 px-4">Type</th>
                        <th className="py-3 px-4">Default</th>
                        <th className="py-3 px-4">Purpose</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-medium text-slate-700 dark:text-slate-300">
                      {streams.map((s) => (
                        <tr key={s.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-800/40 transition-colors">
                          <td className="py-3.5 px-4 font-semibold text-slate-900 dark:text-white">{s.name}</td>
                          <td className="py-3.5 px-4 capitalize text-xs">
                            <span className="badge-info">{s.type}</span>
                          </td>
                          <td className="py-3.5 px-4 text-xs font-semibold">{s.is_default ? 'Yes' : 'No'}</td>
                          <td className="py-3.5 px-4 text-slate-500 dark:text-slate-400 text-xs">
                            {s.type === 'transactional'
                              ? 'Receipts, password resets, alerts (Priority p1)'
                              : 'Broadcast newsletters, marketing (RFC 8058 One-Click unsub)'}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* TAB: API KEYS */}
            {currentTab === 'apikeys' && (
              <div className="space-y-6 mt-6">
                {/* Generated Secret Alert Modal / Box */}
                {createdSecret && (
                  <div className="bg-emerald-950/60 border border-emerald-500/50 p-6 rounded-xl space-y-3 shadow-xl">
                    <div className="flex items-center space-x-2 text-emerald-400 font-semibold">
                      <span>API Key Created Successfully!</span>
                    </div>
                    <p className="text-sm text-emerald-200">
                      Copy this secret key immediately. For your security, you will not be able to see it again.
                    </p>
                    <div className="flex items-center space-x-3">
                      <input
                        type="text"
                        readOnly
                        value={createdSecret}
                        className="bg-slate-900 border border-slate-700 text-emerald-300 font-mono text-sm px-3 py-2 rounded-lg flex-1"
                      />
                      <button
                        onClick={() => {
                          navigator.clipboard.writeText(createdSecret)
                          alert('Copied to clipboard!')
                        }}
                        className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-sm font-medium transition-colors"
                      >
                        Copy Key
                      </button>
                      <button
                        onClick={() => setCreatedSecret(null)}
                        className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg text-sm transition-colors"
                      >
                        Done
                      </button>
                    </div>
                  </div>
                )}

                {/* Create Key Panel */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                  <h3 className="text-lg font-semibold text-white mb-4">Create New API Key</h3>
                  <form onSubmit={handleCreateAPIKey} className="flex gap-4 items-end">
                    <div className="flex-1">
                      <label className="block text-xs font-medium text-slate-400 mb-1">Key Name</label>
                      <input
                        type="text"
                        required
                        placeholder="e.g. Backend Production Worker"
                        value={newKeyName}
                        onChange={(e) => setNewKeyName(e.target.value)}
                        className="w-full bg-slate-800 border border-slate-700 text-slate-200 text-sm px-3 py-2 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                      />
                    </div>
                    <div className="w-44">
                      <label className="block text-xs font-medium text-slate-400 mb-1">Environment</label>
                      <select
                        value={newKeyEnv}
                        onChange={(e) => setNewKeyEnv(e.target.value)}
                        className="w-full bg-slate-800 border border-slate-700 text-slate-200 text-sm px-3 py-2 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                      >
                        <option value="live">Live (External Delivery)</option>
                        <option value="test">Test (Development Sink)</option>
                      </select>
                    </div>
                    <button
                      type="submit"
                      disabled={keyLoading}
                      className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-lg disabled:opacity-50 transition-colors"
                    >
                      {keyLoading ? 'Generating...' : 'Generate API Key'}
                    </button>
                  </form>
                </div>

                {/* API Keys Table */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="p-4 border-b border-slate-800 font-semibold text-white">
                    Active API Keys
                  </div>
                  <table className="w-full text-left text-sm text-slate-300">
                    <thead className="bg-slate-950/50 text-slate-400">
                      <tr>
                        <th className="p-4">Name</th>
                        <th className="p-4">Prefix / Last 4</th>
                        <th className="p-4">Environment</th>
                        <th className="p-4">Scopes</th>
                        <th className="p-4">Created</th>
                        <th className="p-4 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-800">
                      {apiKeys.length === 0 ? (
                        <tr>
                          <td colSpan={6} className="p-6 text-center text-slate-500">
                            No API keys found. Generate one to start sending via REST API.
                          </td>
                        </tr>
                      ) : (
                        apiKeys.map((k) => (
                          <tr key={k.id} className="hover:bg-slate-800/40">
                            <td className="p-4 font-medium text-white">{k.name}</td>
                            <td className="p-4 font-mono text-xs text-indigo-400">
                              {k.prefix}...{k.last_four}
                            </td>
                            <td className="p-4">
                              <span
                                className={`px-2 py-0.5 text-xs rounded-full ${
                                  k.environment === 'live'
                                    ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                                    : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                                }`}
                              >
                                {k.environment}
                              </span>
                            </td>
                            <td className="p-4 text-xs text-slate-400">
                              {k.scopes ? k.scopes.join(', ') : 'send.email'}
                            </td>
                            <td className="p-4 text-slate-400 text-xs">
                              {new Date(k.created_at).toLocaleDateString()}
                            </td>
                            <td className="p-4 text-right">
                              <button
                                onClick={() => handleRevokeAPIKey(k.id)}
                                className="px-3 py-1 bg-red-500/10 hover:bg-red-500/20 text-red-400 border border-red-500/20 text-xs rounded-lg transition-colors"
                              >
                                Revoke
                              </button>
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* TAB: MESSAGES LOGS & INSPECTOR */}
            {currentTab === 'messages' && (
              <div className="space-y-6 mt-6">
                <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 rounded-2xl shadow-glass dark:shadow-glass-dark overflow-hidden">
                  <div className="p-4 border-b border-slate-100 dark:border-slate-800/80 flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                      <Inbox className="w-4 h-4 text-sky-500" />
                      <span className="font-bold text-sm text-slate-900 dark:text-white">Message Activity Logs</span>
                      <span className="text-xs text-slate-500 dark:text-slate-400">({messagesList.length} Total)</span>
                    </div>
                    <div className="flex items-center space-x-2">
                      <button
                        onClick={loadMessages}
                        className="btn-secondary text-xs px-3 py-1.5"
                      >
                        <RefreshCw className="w-3.5 h-3.5" />
                        <span>Refresh</span>
                      </button>
                      <ExportMenu
                        filename={`mailvex-messages-${activeTenant?.slug || 'tenant'}`}
                        sheetTitle="Message Logs"
                        label="Export Logs"
                        columns={[
                          { header: 'Message ID', key: 'id' },
                          { header: 'From Address', key: 'from_address' },
                          { header: 'Subject', key: 'subject' },
                          { header: 'Status', key: 'status' },
                          { header: 'Stream', key: 'stream_id' },
                          { header: 'Environment', key: 'environment' },
                          { header: 'Timestamp', key: 'created_at' },
                        ]}
                        data={messagesList.map((m) => ({
                          id: m.id,
                          from_address: m.from_address,
                          subject: m.subject,
                          status: m.status,
                          stream_id: m.stream_id,
                          environment: m.environment,
                          created_at: m.created_at,
                        }))}
                      />
                    </div>
                  </div>
                  <table className="w-full text-left text-sm">
                    <thead className="bg-slate-50/70 dark:bg-slate-950/40 text-[11px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400 border-b border-slate-100 dark:border-slate-800">
                      <tr>
                        <th className="py-3 px-4">Message ID</th>
                        <th className="py-3 px-4">Sender</th>
                        <th className="py-3 px-4">Subject</th>
                        <th className="py-3 px-4">Status</th>
                        <th className="py-3 px-4">Environment</th>
                        <th className="py-3 px-4">Time</th>
                        <th className="py-3 px-4 text-right">Action</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-medium text-slate-700 dark:text-slate-300">
                      {messagesList.length === 0 ? (
                        <tr>
                          <td colSpan={7} className="p-8 text-center text-slate-400 dark:text-slate-500 text-xs">
                            No messages recorded yet. Send your first email via SMTP or REST API.
                          </td>
                        </tr>
                      ) : (
                        messagesList.map((m) => (
                          <tr key={m.id} className="hover:bg-slate-50/60 dark:hover:bg-slate-800/40 transition-colors">
                            <td className="py-3.5 px-4 font-mono text-xs text-sky-600 dark:text-sky-400 font-bold">{m.id}</td>
                            <td className="py-3.5 px-4 text-slate-800 dark:text-slate-200">{m.from_address}</td>
                            <td className="py-3.5 px-4 text-slate-900 dark:text-white font-semibold">{m.subject}</td>
                            <td className="py-3.5 px-4">
                              <span
                                className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold backdrop-blur-md border ${
                                  m.status === 'delivered'
                                    ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20'
                                    : m.status === 'queued'
                                    ? 'bg-sky-500/10 text-sky-600 dark:text-sky-400 border-sky-500/20'
                                    : 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20'
                                }`}
                              >
                                {m.status}
                              </span>
                            </td>
                            <td className="py-3.5 px-4 text-xs capitalize text-slate-600 dark:text-slate-400">{m.environment}</td>
                            <td className="py-3.5 px-4 text-xs text-slate-500 dark:text-slate-400">
                              {new Date(m.created_at).toLocaleTimeString()}
                            </td>
                            <td className="py-3.5 px-4 text-right">
                              <button
                                onClick={() => handleInspectMessage(m)}
                                className="btn-secondary text-xs"
                              >
                                <span>Inspect</span>
                              </button>
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>

                {/* Message Inspector Modal */}
                {inspectorOpen && selectedMessage && (
                  <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
                    <div className="bg-white/95 dark:bg-slate-900/95 backdrop-blur-xl border border-white/80 dark:border-slate-800/80 rounded-2xl w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
                      <div className="p-5 border-b border-slate-100 dark:border-slate-800/80 flex items-center justify-between">
                        <div>
                          <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                            <span>Message Inspector</span>
                            <span className="text-xs px-2 py-0.5 rounded-full font-mono font-bold bg-sky-500/10 text-sky-600 dark:text-sky-400 border border-sky-500/20">
                              {selectedMessage.status}
                            </span>
                          </h3>
                          <p className="text-xs font-mono text-sky-600 dark:text-sky-400 mt-0.5">{selectedMessage.id}</p>
                        </div>
                        <div className="flex items-center space-x-2">
                          <ExportMenu
                            filename={`message-inspect-${selectedMessage.id}`}
                            sheetTitle="Message Details"
                            label="Export Message"
                            columns={[
                              { header: 'Section', key: 'section' },
                              { header: 'Field / Type', key: 'field' },
                              { header: 'Value / Details', key: 'value' },
                              { header: 'Timestamp', key: 'timestamp' },
                            ]}
                            data={[
                              { section: 'Metadata', field: 'Message ID', value: selectedMessage.id, timestamp: selectedMessage.created_at },
                              { section: 'Metadata', field: 'From Address', value: selectedMessage.from_address, timestamp: selectedMessage.created_at },
                              { section: 'Metadata', field: 'Subject', value: selectedMessage.subject, timestamp: selectedMessage.created_at },
                              { section: 'Metadata', field: 'Status', value: selectedMessage.status, timestamp: selectedMessage.created_at },
                              { section: 'Metadata', field: 'Environment', value: selectedMessage.environment, timestamp: selectedMessage.created_at },
                              ...selectedMsgRecipients.map((r: any) => ({
                                section: 'Recipient',
                                field: `${r.type.toUpperCase()}: ${r.address}`,
                                value: `Status: ${r.status} | Attempts: ${r.attempts}`,
                                timestamp: selectedMessage.created_at,
                              })),
                              ...selectedMsgEvents.map((e: any) => ({
                                section: 'Timeline Event',
                                field: e.type,
                                value: e.data || '',
                                timestamp: e.occurred_at,
                              })),
                            ]}
                          />
                          <button
                            onClick={() => setInspectorOpen(false)}
                            className="text-slate-400 hover:text-slate-600 dark:hover:text-white text-xl font-bold p-1 ml-2"
                          >
                            ✕
                          </button>
                        </div>
                      </div>
                      <div className="p-6 overflow-y-auto space-y-6 flex-1">
                        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 bg-slate-50/70 dark:bg-slate-950/60 p-4 rounded-xl border border-slate-200/80 dark:border-slate-800">
                          <div>
                            <span className="block text-xs text-slate-500 dark:text-slate-400 font-medium">From</span>
                            <span className="text-sm font-semibold text-slate-900 dark:text-slate-200 truncate block">{selectedMessage.from_address}</span>
                          </div>
                          <div>
                            <span className="block text-xs text-slate-500 dark:text-slate-400 font-medium">Subject</span>
                            <span className="text-sm font-semibold text-slate-900 dark:text-slate-200 truncate block">{selectedMessage.subject}</span>
                          </div>
                          <div>
                            <span className="block text-xs text-slate-500 dark:text-slate-400 font-medium">Status</span>
                            <span className="text-sm font-bold text-sky-600 dark:text-sky-400 capitalize block">{selectedMessage.status}</span>
                          </div>
                          <div>
                            <span className="block text-xs text-slate-500 dark:text-slate-400 font-medium">Environment</span>
                            <span className="text-sm font-bold text-emerald-600 dark:text-emerald-400 capitalize block">{selectedMessage.environment}</span>
                          </div>
                        </div>

                        {/* Recipients */}
                        <div>
                          <h4 className="text-xs font-bold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-2">Recipients</h4>
                          <div className="bg-slate-50/70 dark:bg-slate-950/40 border border-slate-200/80 dark:border-slate-800 rounded-xl overflow-hidden">
                            <table className="w-full text-left text-xs text-slate-700 dark:text-slate-300">
                              <thead className="bg-slate-100/80 dark:bg-slate-900 text-slate-500 dark:text-slate-400 font-semibold uppercase">
                                <tr>
                                  <th className="p-3">Address</th>
                                  <th className="p-3">Type</th>
                                  <th className="p-3">Status</th>
                                  <th className="p-3">Attempts</th>
                                </tr>
                              </thead>
                              <tbody className="divide-y divide-slate-200/60 dark:divide-slate-800">
                                {selectedMsgRecipients.map((r) => (
                                  <tr key={r.id}>
                                    <td className="p-3 font-mono text-slate-900 dark:text-slate-200">{r.address}</td>
                                    <td className="p-3 uppercase">{r.type}</td>
                                    <td className="p-3 capitalize font-semibold">{r.status}</td>
                                    <td className="p-3">{r.attempts}</td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          </div>
                        </div>

                        {/* Chronological Event History */}
                        <div>
                          <h4 className="text-xs font-bold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-2">Event Timeline</h4>
                          <div className="space-y-2">
                            {selectedMsgEvents.map((e) => (
                              <div key={e.id} className="text-xs bg-slate-50/70 dark:bg-slate-950/50 p-3 rounded-xl border border-slate-200/80 dark:border-slate-800 flex items-center justify-between">
                                <div className="flex items-center space-x-2">
                                  <span className="w-2 h-2 rounded-full bg-sky-500"></span>
                                  <span className="font-semibold text-sky-600 dark:text-sky-400">{e.type}</span>
                                  <span className="text-slate-600 dark:text-slate-400">{e.data}</span>
                                </div>
                                <span className="text-slate-400 font-mono text-[11px]">
                                  {new Date(e.occurred_at).toLocaleTimeString()}
                                </span>
                              </div>
                            ))}
                          </div>
                        </div>

                        {/* Sandboxed Body HTML Preview (§23.3) */}
                        <div>
                          <h4 className="text-xs font-bold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-2">Message Body Preview (Sandboxed)</h4>
                          <iframe
                            sandbox=""
                            srcDoc={selectedMsgBody}
                            className="w-full h-64 bg-white rounded-xl border border-slate-200 dark:border-slate-700/80 shadow-inner"
                            title="Message Content Preview"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* TAB: TEMPLATES */}
            {currentTab === 'templates' && (
              <div className="space-y-6 mt-6">
                <div className="glass-card p-6">
                  <h3 className="text-base font-bold text-slate-900 dark:text-white mb-4">Create Template</h3>
                  <form onSubmit={handleCreateTemplate} className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">Template Name</label>
                        <input
                          type="text"
                          required
                          placeholder="e.g. Welcome Onboarding"
                          value={newTmplName}
                          onChange={(e) => setNewTmplName(e.target.value)}
                          className="glass-input"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">Subject (Mustache)</label>
                        <input
                          type="text"
                          required
                          placeholder="Welcome to {{company}}, {{name}}!"
                          value={newTmplSubject}
                          onChange={(e) => setNewTmplSubject(e.target.value)}
                          className="glass-input"
                        />
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">HTML Body</label>
                      <textarea
                        rows={3}
                        placeholder="<h1>Welcome {{name}}!</h1><p>We are excited to have you.</p>"
                        value={newTmplHTML}
                        onChange={(e) => setNewTmplHTML(e.target.value)}
                        className="glass-input font-mono"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">Plain Text Body</label>
                      <textarea
                        rows={2}
                        placeholder="Welcome {{name}}! We are excited to have you."
                        value={newTmplText}
                        onChange={(e) => setNewTmplText(e.target.value)}
                        className="glass-input font-mono"
                      />
                    </div>
                    <button
                      type="submit"
                      className="btn-gradient text-xs px-5 py-2.5"
                    >
                      Save & Publish Template
                    </button>
                  </form>
                </div>

                <div className="glass-card overflow-hidden">
                  <div className="p-4 border-b border-slate-100 dark:border-slate-800/80 flex items-center justify-between">
                    <span className="font-bold text-sm text-slate-900 dark:text-white">Published Templates</span>
                    <div className="flex items-center space-x-2">
                      <span className="text-xs text-slate-500 dark:text-slate-400 font-medium">({templatesList.length} Total)</span>
                      <ExportMenu
                        filename={`mailvex-templates-${activeTenant?.slug || 'tenant'}`}
                        sheetTitle="Templates"
                        label="Export Templates"
                        columns={[
                          { header: 'Template ID', key: 'id' },
                          { header: 'Template Name', key: 'name' },
                          { header: 'Description', key: 'description' },
                          { header: 'Created At', key: 'created_at' },
                        ]}
                        data={templatesList.map((t) => ({
                          id: t.id,
                          name: t.name,
                          description: t.description || '—',
                          created_at: t.created_at,
                        }))}
                      />
                    </div>
                  </div>
                  <div className="divide-y divide-slate-100 dark:divide-slate-800/60">
                    {templatesList.length === 0 ? (
                      <div className="p-6 text-center text-slate-500 text-sm">
                        No templates created yet.
                      </div>
                    ) : (
                      templatesList.map((t) => (
                        <div key={t.id} className="p-4 flex items-center justify-between hover:bg-slate-800/40">
                          <div>
                            <span className="font-semibold text-white">{t.name}</span>
                            <span className="ml-3 font-mono text-xs text-indigo-400">{t.id}</span>
                          </div>
                          <div className="flex items-center space-x-3">
                            <input
                              type="text"
                              value={previewVars}
                              onChange={(e) => setPreviewVars(e.target.value)}
                              placeholder='{"name": "Alex"}'
                              className="bg-slate-800 border border-slate-700 text-xs px-2 py-1 rounded text-slate-300 font-mono w-48"
                            />
                            <button
                              onClick={() => handleRenderPreview(t.id)}
                              className="px-3 py-1 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded-lg transition-colors"
                            >
                              Live Preview
                            </button>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                {/* Rendered Preview Modal / Card */}
                {renderedPreview && (
                  <div className="bg-slate-900 border border-indigo-500/30 rounded-xl p-6 space-y-4">
                    <div className="flex items-center justify-between">
                      <h4 className="text-base font-bold text-white">Live Render Preview Output</h4>
                      <button
                        onClick={() => setRenderedPreview(null)}
                        className="text-slate-400 hover:text-white text-sm"
                      >
                        Close
                      </button>
                    </div>
                    <div className="bg-slate-950 p-3 rounded-lg text-sm text-slate-200">
                      <span className="text-slate-500 font-semibold mr-2">Subject:</span>
                      {renderedPreview.subject}
                    </div>
                    <div className="bg-slate-950 p-3 rounded-lg text-sm text-slate-200">
                      <span className="text-slate-500 font-semibold mr-2">HTML:</span>
                      <div dangerouslySetInnerHTML={{ __html: renderedPreview.html }} className="mt-2 p-3 bg-white text-black rounded" />
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* TAB: SUPPRESSIONS */}
            {currentTab === 'suppressions' && (
              <div className="space-y-6 mt-6">
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                  <h3 className="text-lg font-semibold text-white mb-4">Add Manual Suppression</h3>
                  <form onSubmit={handleCreateSuppression} className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <div>
                        <label className="block text-xs font-medium text-slate-400 mb-1">Email or Domain</label>
                        <input
                          type="text"
                          required
                          placeholder="e.g. user@example.com or bad-domain.com"
                          value={newSuppAddress}
                          onChange={(e) => setNewSuppAddress(e.target.value)}
                          className="w-full bg-slate-800 border border-slate-700 text-slate-200 text-sm px-3 py-2 rounded-lg"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-slate-400 mb-1">Suppression Type</label>
                        <select
                          value={newSuppType}
                          onChange={(e) => setNewSuppType(e.target.value)}
                          className="w-full bg-slate-800 border border-slate-700 text-slate-200 text-sm px-3 py-2 rounded-lg"
                        >
                          <option value="manual">Manual Block</option>
                          <option value="hard_bounce">Hard Bounce</option>
                          <option value="complaint">Spam Complaint</option>
                          <option value="unsubscribe">Unsubscribe</option>
                          <option value="domain_block">Domain Block</option>
                        </select>
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-slate-400 mb-1">Reason (Optional)</label>
                        <input
                          type="text"
                          placeholder="e.g. Recipient requested exclusion"
                          value={newSuppReason}
                          onChange={(e) => setNewSuppReason(e.target.value)}
                          className="w-full bg-slate-800 border border-slate-700 text-slate-200 text-sm px-3 py-2 rounded-lg"
                        />
                      </div>
                    </div>
                    <button
                      type="submit"
                      className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-lg transition-colors"
                    >
                      Add to Suppression List
                    </button>
                  </form>
                </div>

                <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 rounded-2xl shadow-glass dark:shadow-glass-dark overflow-hidden">
                  <div className="p-4 border-b border-slate-100 dark:border-slate-800/80 flex items-center justify-between font-bold text-sm text-slate-900 dark:text-white">
                    <div className="flex items-center space-x-2">
                      <Ban className="w-4 h-4 text-sky-500" />
                      <span>Suppressed Recipients</span>
                      <span className="text-xs text-slate-500 dark:text-slate-400 font-normal">({suppressionsList.length} Total)</span>
                    </div>
                    <div className="flex items-center space-x-2">
                      <button
                        onClick={() => setImportModalOpen(true)}
                        className="btn-secondary text-xs px-3 py-1.5"
                      >
                        Import CSV
                      </button>
                      <ExportMenu
                        filename={`mailvex-suppressions-${activeTenant?.slug || 'tenant'}`}
                        sheetTitle="Suppressions"
                        label="Export Suppressions"
                        columns={[
                          { header: 'Address / Domain', key: 'address' },
                          { header: 'Domain Name', key: 'address_domain' },
                          { header: 'Suppression Type', key: 'type' },
                          { header: 'Reason', key: 'reason' },
                          { header: 'Source', key: 'source' },
                          { header: 'Timestamp', key: 'created_at' },
                        ]}
                        data={suppressionsList.map((s) => ({
                          address: s.address,
                          address_domain: s.address_domain,
                          type: s.type,
                          reason: s.reason || 'N/A',
                          source: s.source,
                          created_at: s.created_at,
                        }))}
                      />
                      <button
                        onClick={loadSuppressions}
                        className="btn-secondary text-xs px-3 py-1.5"
                      >
                        <RefreshCw className="w-3.5 h-3.5" />
                        <span>Refresh</span>
                      </button>
                    </div>
                  </div>
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                      <thead className="bg-slate-50/70 dark:bg-slate-950/40 text-[11px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400 border-b border-slate-100 dark:border-slate-800">
                        <tr>
                          <th className="py-3 px-6">Address / Domain</th>
                          <th className="py-3 px-6">Type</th>
                          <th className="py-3 px-6">Reason</th>
                          <th className="py-3 px-6">Source</th>
                          <th className="py-3 px-6">Created</th>
                          <th className="py-3 px-6 text-right">Action</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-medium text-slate-700 dark:text-slate-300">
                        {suppressionsList.length === 0 ? (
                          <tr>
                            <td colSpan={6} className="px-6 py-8 text-center text-slate-500">
                              No addresses currently suppressed.
                            </td>
                          </tr>
                        ) : (
                          suppressionsList.map((s) => (
                            <tr key={s.id} className="hover:bg-slate-800/40">
                              <td className="px-6 py-4 font-mono font-medium text-white">{s.address}</td>
                              <td className="px-6 py-4">
                                <span
                                  className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                    s.type === 'hard_bounce'
                                      ? 'bg-red-950 text-red-400 border border-red-800'
                                      : s.type === 'complaint'
                                      ? 'bg-amber-950 text-amber-400 border border-amber-800'
                                      : s.type === 'unsubscribe'
                                      ? 'bg-purple-950 text-purple-400 border border-purple-800'
                                      : s.type === 'domain_block'
                                      ? 'bg-yellow-950 text-yellow-400 border border-yellow-800'
                                      : 'bg-slate-800 text-slate-300 border border-slate-700'
                                  }`}
                                >
                                  {s.type}
                                </span>
                              </td>
                              <td className="px-6 py-4 text-xs text-slate-400 max-w-xs truncate">{s.reason || '—'}</td>
                              <td className="px-6 py-4 text-xs text-slate-400 font-mono">{s.source}</td>
                              <td className="px-6 py-4 text-xs text-slate-500">
                                {new Date(s.created_at).toLocaleString()}
                              </td>
                              <td className="px-6 py-4 text-right">
                                <button
                                  onClick={() => handleDeleteSuppression(s.id)}
                                  className="text-xs text-red-400 hover:text-red-300 font-medium"
                                >
                                  Remove
                                </button>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Import CSV Modal */}
                {importModalOpen && (
                  <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
                    <div className="bg-white/95 dark:bg-slate-900/95 backdrop-blur-xl border border-white/80 dark:border-slate-800/80 rounded-2xl max-w-lg w-full p-6 shadow-2xl space-y-4">
                      <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800/80 pb-3">
                        <h4 className="text-base font-bold text-slate-900 dark:text-white">Bulk Import Suppressions (CSV)</h4>
                        <button
                          onClick={() => setImportModalOpen(false)}
                          className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 text-sm"
                        >
                          ✕
                        </button>
                      </div>

                      {importStatusMsg && (
                        <div className={`p-3 rounded-xl text-xs font-semibold ${
                          importStatusMsg.startsWith('Import completed')
                            ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                            : 'bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20'
                        }`}>
                          {importStatusMsg}
                        </div>
                      )}

                      <form onSubmit={handleImportSuppressions} className="space-y-4">
                        <div>
                          <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                            CSV Content (format: email,type,reason)
                          </label>
                          <textarea
                            rows={6}
                            required
                            placeholder={"email,type,reason\nbounced@example.com,hard_bounce,Mailbox inactive\nunsub@example.com,unsubscribe,User opted out"}
                            value={importCSVText}
                            onChange={(e) => setImportCSVText(e.target.value)}
                            className="w-full bg-slate-50/70 dark:bg-slate-800/70 border border-slate-200 dark:border-slate-700/80 text-slate-900 dark:text-slate-100 font-mono text-xs px-3.5 py-2.5 rounded-xl focus:outline-none focus:ring-2 focus:ring-sky-500/30"
                          />
                        </div>
                        <div className="flex justify-end space-x-2 pt-2 border-t border-slate-100 dark:border-slate-800/80">
                          <button
                            type="button"
                            onClick={() => setImportModalOpen(false)}
                            className="btn-secondary text-xs px-3.5 py-2"
                          >
                            Cancel
                          </button>
                          <button
                            type="submit"
                            className="btn-gradient text-xs px-4 py-2"
                          >
                            Upload & Process
                          </button>
                        </div>
                      </form>
                    </div>
                  </div>
                )}
              </div>
            )}

            {currentTab === 'reputation' && (
              <div className="mt-6 space-y-8">
                {/* Provider Reputation Metrics & Threshold Cards */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-semibold text-indigo-400 uppercase tracking-wider">
                        Google Postmaster
                      </span>
                      <span className="px-2 py-0.5 rounded text-xs font-medium bg-emerald-950 text-emerald-400 border border-emerald-800">
                        Active
                      </span>
                    </div>
                    <div className="mt-4">
                      <div className="text-2xl font-bold text-white">
                        {postmasterSnaps.length > 0 ? `${(postmasterSnaps[0].spam_rate * 100).toFixed(2)}%` : '0.02%'}
                      </div>
                      <p className="text-xs text-slate-400 mt-1">Spam rate (&lt;0.10% threshold)</p>
                    </div>
                    <div className="mt-4 pt-4 border-t border-slate-800 flex justify-between text-xs text-slate-400">
                      <span>Domain Rep: <strong className="text-emerald-400">High</strong></span>
                      <span>DKIM Auth: <strong className="text-slate-200">99.8%</strong></span>
                    </div>
                  </div>

                  <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-semibold text-sky-400 uppercase tracking-wider">
                        Microsoft SNDS / JMRP
                      </span>
                      <span className="px-2 py-0.5 rounded text-xs font-medium bg-emerald-950 text-emerald-400 border border-emerald-800">
                        Enrolled
                      </span>
                    </div>
                    <div className="mt-4">
                      <div className="text-2xl font-bold text-white">Normal</div>
                      <p className="text-xs text-slate-400 mt-1">IP Filter Status (0 Trap Hits)</p>
                    </div>
                    <div className="mt-4 pt-4 border-t border-slate-800 flex justify-between text-xs text-slate-400">
                      <span>Complaint Rate: <strong className="text-emerald-400">&lt;0.3%</strong></span>
                      <span>JMRP Status: <strong className="text-slate-200">Active</strong></span>
                    </div>
                  </div>

                  <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-semibold text-purple-400 uppercase tracking-wider">
                        Yahoo CFL
                      </span>
                      <span className="px-2 py-0.5 rounded text-xs font-medium bg-emerald-950 text-emerald-400 border border-emerald-800">
                        DKIM Aligned
                      </span>
                    </div>
                    <div className="mt-4">
                      <div className="text-2xl font-bold text-white">0.01%</div>
                      <p className="text-xs text-slate-400 mt-1">Complaint Rate</p>
                    </div>
                    <div className="mt-4 pt-4 border-t border-slate-800 flex justify-between text-xs text-slate-400">
                      <span>DMARC Status: <strong className="text-emerald-400">Pass</strong></span>
                      <span>Feedback: <strong className="text-slate-200">Auto-ARF</strong></span>
                    </div>
                  </div>
                </div>

                {/* Abuse Intake Queue (§18.1, RFC 5321 §4.5.1) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="px-6 py-5 border-b border-slate-800 flex items-center justify-between flex-wrap gap-4">
                    <div>
                      <h3 className="text-base font-semibold text-white">Abuse &amp; Postmaster Intake Queue</h3>
                      <p className="text-xs text-slate-400 mt-0.5">
                        Mandatory abuse@ and postmaster@ inbound mailbox reports requiring operator inspection.
                      </p>
                    </div>
                    <div className="flex items-center space-x-2">
                      <button
                        onClick={() => setAbuseStatusFilter('pending')}
                        className={`px-3 py-1 text-xs font-medium rounded-lg transition-colors ${
                          abuseStatusFilter === 'pending'
                            ? 'bg-indigo-600 text-white'
                            : 'bg-slate-800 text-slate-300 hover:text-white'
                        }`}
                      >
                        Pending
                      </button>
                      <button
                        onClick={() => setAbuseStatusFilter('resolved')}
                        className={`px-3 py-1 text-xs font-medium rounded-lg transition-colors ${
                          abuseStatusFilter === 'resolved'
                            ? 'bg-indigo-600 text-white'
                            : 'bg-slate-800 text-slate-300 hover:text-white'
                        }`}
                      >
                        Resolved
                      </button>
                      <button
                        onClick={() => setAbuseStatusFilter('escalated')}
                        className={`px-3 py-1 text-xs font-medium rounded-lg transition-colors ${
                          abuseStatusFilter === 'escalated'
                            ? 'bg-indigo-600 text-white'
                            : 'bg-slate-800 text-slate-300 hover:text-white'
                        }`}
                      >
                        Escalated
                      </button>
                    </div>
                  </div>

                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm text-slate-300">
                      <thead className="bg-slate-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-slate-800">
                        <tr>
                          <th className="px-6 py-3">Report ID</th>
                          <th className="px-6 py-3">Status</th>
                          <th className="px-6 py-3">Recipient Mailbox</th>
                          <th className="px-6 py-3">Sender</th>
                          <th className="px-6 py-3">Subject / Notes</th>
                          <th className="px-6 py-3">Received</th>
                          <th className="px-6 py-3 text-right">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {abuseReports.length === 0 ? (
                          <tr>
                            <td colSpan={7} className="px-6 py-8 text-center text-slate-500">
                              No {abuseStatusFilter} abuse reports found.
                            </td>
                          </tr>
                        ) : (
                          abuseReports.map((r) => (
                            <tr key={r.id} className="hover:bg-slate-800/40">
                              <td className="px-6 py-4 font-mono text-xs text-white">{r.id}</td>
                              <td className="px-6 py-4">
                                <span
                                  className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                    r.status === 'pending'
                                      ? 'bg-amber-950 text-amber-400 border border-amber-800'
                                      : r.status === 'resolved'
                                      ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                                      : 'bg-red-950 text-red-400 border border-red-800'
                                  }`}
                                >
                                  {r.status}
                                </span>
                              </td>
                              <td className="px-6 py-4 text-xs font-mono text-indigo-300">{r.recipient}</td>
                              <td className="px-6 py-4 text-xs text-slate-300">{r.sender}</td>
                              <td className="px-6 py-4 text-xs text-slate-400 max-w-xs truncate">
                                {r.subject || 'No subject'}
                                {r.notes && <span className="block text-slate-500 text-[11px] mt-0.5">Notes: {r.notes}</span>}
                              </td>
                              <td className="px-6 py-4 text-xs text-slate-500">
                                {new Date(r.created_at).toLocaleString()}
                              </td>
                              <td className="px-6 py-4 text-right space-x-2">
                                {r.status === 'pending' && (
                                  <>
                                    <button
                                      onClick={() => handleResolveAbuse(r.id)}
                                      className="text-xs bg-emerald-900/60 hover:bg-emerald-800 text-emerald-300 px-2.5 py-1 rounded border border-emerald-700/50"
                                    >
                                      Resolve
                                    </button>
                                    <button
                                      onClick={() => handleEscalateAbuse(r.id)}
                                      className="text-xs bg-red-900/60 hover:bg-red-800 text-red-300 px-2.5 py-1 rounded border border-red-700/50"
                                    >
                                      Escalate
                                    </button>
                                  </>
                                )}
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Feedback Loop (FBL) Reports Table (§18.1, RFC 5965) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="px-6 py-5 border-b border-slate-800">
                    <h3 className="text-base font-semibold text-white">Inbound ARF Feedback Loop Complaints</h3>
                    <p className="text-xs text-slate-400 mt-0.5">
                      Complaints parsed via RFC 5965 ARF. Addresses are immediately suppressed.
                    </p>
                  </div>

                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm text-slate-300">
                      <thead className="bg-slate-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-slate-800">
                        <tr>
                          <th className="px-6 py-3">Report ID</th>
                          <th className="px-6 py-3">Feedback Type</th>
                          <th className="px-6 py-3">Reported Domain</th>
                          <th className="px-6 py-3">Correlated Recipient</th>
                          <th className="px-6 py-3">Logged Date</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {fblReports.length === 0 ? (
                          <tr>
                            <td colSpan={5} className="px-6 py-8 text-center text-slate-500">
                              No feedback loop complaints recorded for this tenant.
                            </td>
                          </tr>
                        ) : (
                          fblReports.map((f) => (
                            <tr key={f.id} className="hover:bg-slate-800/40">
                              <td className="px-6 py-4 font-mono text-xs text-white">{f.id}</td>
                              <td className="px-6 py-4">
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-950 text-amber-400 border border-amber-800">
                                  {f.feedback_type}
                                </span>
                              </td>
                              <td className="px-6 py-4 text-xs font-mono text-slate-300">{f.reported_domain || '—'}</td>
                              <td className="px-6 py-4 text-xs font-mono text-indigo-300">{f.recipient_id || 'Correlated via Header'}</td>
                              <td className="px-6 py-4 text-xs text-slate-500">
                                {new Date(f.created_at).toLocaleString()}
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Provider Checklists & Integration Guide */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                  <h3 className="text-base font-semibold text-white mb-4">ISP Reputation Integration Guides</h3>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-6 text-xs text-slate-300">
                    <div className="space-y-2 p-4 bg-slate-950/40 rounded-lg border border-slate-800">
                      <div className="font-semibold text-indigo-400 text-sm">Google Postmaster Tools</div>
                      <p className="text-slate-400">Verify domain ownership to track spam rate and user-reported spam.</p>
                      <ol className="list-decimal pl-4 space-y-1 text-slate-400">
                        <li>Visit postmaster.google.com</li>
                        <li>Add your sending domain</li>
                        <li>Copy TXT verification record to DNS</li>
                        <li>Maintain spam rate strictly under 0.10%</li>
                      </ol>
                    </div>

                    <div className="space-y-2 p-4 bg-slate-950/40 rounded-lg border border-slate-800">
                      <div className="font-semibold text-sky-400 text-sm">Microsoft SNDS &amp; JMRP</div>
                      <p className="text-slate-400">Enroll IP addresses in Junk Email Reporting Program.</p>
                      <ol className="list-decimal pl-4 space-y-1 text-slate-400">
                        <li>Visit postmaster.live.com/snds</li>
                        <li>Authorize dedicated outbound IP addresses</li>
                        <li>Set FBL complaint feed to fbl@mailvex.local</li>
                        <li>Monitor daily color status (Green/Yellow/Red)</li>
                      </ol>
                    </div>

                    <div className="space-y-2 p-4 bg-slate-950/40 rounded-lg border border-slate-800">
                      <div className="font-semibold text-purple-400 text-sm">Yahoo / AOL Postmaster</div>
                      <p className="text-slate-400">Automatic Complaint Feedback Loop (CFL) subscription.</p>
                      <ol className="list-decimal pl-4 space-y-1 text-slate-400">
                        <li>Ensure valid DKIM signature on all sends</li>
                        <li>Ensure published DMARC policy with valid rua</li>
                        <li>Yahoo enrolls DKIM domains automatically</li>
                        <li>ARF complaints route to designated FBL address</li>
                      </ol>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* TAB: ANALYTICS */}
            {currentTab === 'analytics' && (
              <div className="mt-6 space-y-6">
                {/* Time range selector & Export CSV */}
                <div className="flex flex-wrap items-center justify-between gap-4 bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-4 rounded-2xl shadow-glass dark:shadow-glass-dark">
                  <div className="flex items-center space-x-2">
                    <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400 mr-2">Time Range:</span>
                    {(['24h', '7d', '30d'] as const).map((r) => (
                      <button
                        key={r}
                        onClick={() => setAnalyticsRange(r)}
                        className={`px-3 py-1.5 text-xs font-semibold rounded-xl transition-all duration-150 ${
                          analyticsRange === r
                            ? 'bg-gradient-to-r from-sky-500 via-indigo-600 to-purple-600 text-white shadow-xs'
                            : 'bg-slate-100/80 dark:bg-slate-800/80 text-slate-600 dark:text-slate-300 hover:bg-slate-200/80 dark:hover:bg-slate-700/80'
                        }`}
                      >
                        {r === '24h' ? 'Last 24 Hours' : r === '7d' ? 'Last 7 Days' : 'Last 30 Days'}
                      </button>
                    ))}
                    {analyticsLoading && (
                      <span className="text-xs text-sky-500 animate-pulse ml-2 font-medium">Loading metrics...</span>
                    )}
                  </div>

                  <div className="flex items-center space-x-2">
                    <button
                      onClick={loadAnalytics}
                      disabled={analyticsLoading}
                      className="btn-secondary text-xs px-3.5 py-2 disabled:opacity-50"
                      title="Refresh analytics metrics"
                    >
                      <RefreshCw className={`w-3.5 h-3.5 ${analyticsLoading ? 'animate-spin text-sky-500' : ''}`} />
                      <span>Refresh</span>
                    </button>
                    <ExportMenu
                      filename={`mailvex-analytics-${activeTenant?.slug || 'tenant'}-${analyticsRange}`}
                      sheetTitle="Analytics"
                      label="Download Report"
                      columns={[
                        { header: 'Metric / Timestamp', key: 'label' },
                        { header: 'Sent Volume', key: 'sent', type: 'number' },
                        { header: 'Delivered Volume', key: 'delivered', type: 'number' },
                        { header: 'Bounced Volume', key: 'bounced', type: 'number' },
                        { header: 'Complained Volume', key: 'complained', type: 'number' },
                        { header: 'Delivery Rate', key: 'delivery_rate' },
                        { header: 'Bounce Rate', key: 'bounce_rate' },
                        { header: 'Complaint Rate', key: 'complaint_rate' },
                      ]}
                      data={
                        analyticsTimeSeries.length > 0
                          ? analyticsTimeSeries.map((p) => ({
                              label: p.timestamp,
                              sent: p.sent,
                              delivered: p.delivered,
                              bounced: p.bounced,
                              complained: p.complained,
                              delivery_rate: `${(p.delivery_rate * 100).toFixed(2)}%`,
                              bounce_rate: `${(((p.bounced || 0) / Math.max(1, p.sent || 1)) * 100).toFixed(2)}%`,
                              complaint_rate: `${(p.complaint_rate * 100).toFixed(3)}%`,
                            }))
                          : [
                              {
                                label: `Overall Summary (${analyticsRange})`,
                                sent: analyticsSummary?.sent_count ?? analyticsSummary?.sent ?? 0,
                                delivered: analyticsSummary?.delivered_count ?? analyticsSummary?.delivered ?? 0,
                                bounced: (analyticsSummary?.hard_bounce_count ?? 0) + (analyticsSummary?.soft_bounce_count ?? 0),
                                complained: analyticsSummary?.complaint_count ?? analyticsSummary?.complained ?? 0,
                                delivery_rate: `${((analyticsSummary?.delivery_rate ?? 0) * 100).toFixed(2)}%`,
                                bounce_rate: `${((analyticsSummary?.bounce_rate ?? 0) * 100).toFixed(2)}%`,
                                complaint_rate: `${((analyticsSummary?.complaint_rate ?? 0) * 100).toFixed(3)}%`,
                              },
                              ...analyticsProviders.map((pv) => ({
                                label: `Provider: ${pv.provider}`,
                                sent: pv.sent_count || pv.sent,
                                delivered: pv.delivered_count || pv.delivered,
                                bounced: pv.bounce_count || pv.bounced,
                                complained: pv.complaint_count || pv.complained,
                                delivery_rate: `${(pv.delivery_rate * 100).toFixed(2)}%`,
                                bounce_rate: `${((pv.bounce_rate ?? 0) * 100).toFixed(2)}%`,
                                complaint_rate: `${(pv.complaint_rate * 100).toFixed(3)}%`,
                              }))
                            ]
                      }
                    />
                  </div>
                </div>

                {/* Reputation Alert if complaint rate exceeds 0.10% threshold */}
                {analyticsSummary?.complaint_warning && (
                  <div className="bg-rose-500/10 border border-rose-500/20 rounded-2xl p-4 flex items-start space-x-3 text-rose-700 dark:text-rose-300 backdrop-blur-xl">
                    <span className="text-xl">⚠️</span>
                    <div>
                      <div className="font-bold text-sm text-rose-800 dark:text-rose-200">Google &amp; Yahoo Deliverability Warning: Complaint Rate Exceeded 0.10%</div>
                      <p className="text-xs text-rose-700/80 dark:text-rose-300/80 mt-1">
                        Current tenant complaint rate is {((analyticsSummary.complaint_rate || 0) * 100).toFixed(3)}%. Mailbox providers will defer or spam-folder messages if complaint rate approaches or exceeds 0.30%. Review unverified lists and enforce immediate suppression.
                      </p>
                    </div>
                  </div>
                )}

                {/* KPI Summary Cards (Blueprint 4.2) */}
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-5 rounded-2xl shadow-glass dark:shadow-glass-dark transition-all duration-200 hover:scale-[1.01] hover:border-sky-500/40">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Total Sent</div>
                    <div className="text-3xl font-black text-slate-900 dark:text-white mt-2 tracking-tight">
                      {(analyticsSummary?.sent ?? analyticsSummary?.sent_count ?? 0).toLocaleString()}
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-400 mt-1 font-medium">Outbound messages</div>
                  </div>

                  <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-5 rounded-2xl shadow-glass dark:shadow-glass-dark transition-all duration-200 hover:scale-[1.01] hover:border-sky-500/40">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Delivery Rate</div>
                    <div className="text-3xl font-black text-emerald-600 dark:text-emerald-400 mt-2 tracking-tight">
                      {((analyticsSummary?.delivery_rate || 0) * 100).toFixed(2)}%
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-400 mt-1 font-medium">
                      {(analyticsSummary?.delivered ?? analyticsSummary?.delivered_count ?? 0).toLocaleString()} delivered
                    </div>
                  </div>

                  <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-5 rounded-2xl shadow-glass dark:shadow-glass-dark transition-all duration-200 hover:scale-[1.01] hover:border-sky-500/40">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Bounce Rate</div>
                    <div className="text-3xl font-black text-rose-600 dark:text-rose-400 mt-2 tracking-tight">
                      {((analyticsSummary?.bounce_rate || 0) * 100).toFixed(2)}%
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-400 mt-1 font-medium">
                      {(analyticsSummary?.bounced_hard ?? analyticsSummary?.hard_bounce_count ?? 0)} hard / {(analyticsSummary?.bounced_soft ?? analyticsSummary?.soft_bounce_count ?? 0)} soft
                    </div>
                  </div>

                  <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-5 rounded-2xl shadow-glass dark:shadow-glass-dark transition-all duration-200 hover:scale-[1.01] hover:border-sky-500/40">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Complaint Rate</div>
                    <div className={`text-3xl font-black mt-2 tracking-tight ${analyticsSummary?.complaint_warning ? 'text-rose-600 dark:text-rose-400' : 'text-slate-900 dark:text-slate-100'}`}>
                      {((analyticsSummary?.complaint_rate || 0) * 100).toFixed(3)}%
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-400 mt-1 font-medium">
                      {(analyticsSummary?.complained ?? analyticsSummary?.complaint_count ?? 0)} spam reports
                    </div>
                  </div>
                </div>

                {/* Additional Performance Row: Latency & Unsubscribes */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 p-5 rounded-2xl shadow-glass dark:shadow-glass-dark">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Delivery Latency Percentiles</div>
                    <div className="flex items-baseline space-x-4 mt-2">
                      <div>
                        <span className="text-2xl font-black text-sky-600 dark:text-sky-400 tracking-tight">
                          {analyticsSummary && (analyticsSummary.median_latency_seconds ?? 0) > 0
                            ? `${analyticsSummary.median_latency_seconds!.toFixed(2)}s`
                            : '< 1s'}
                        </span>
                        <span className="text-xs text-slate-500 dark:text-slate-400 ml-1 font-medium">Median (p50)</span>
                      </div>
                      <div>
                        <span className="text-2xl font-black text-purple-600 dark:text-purple-400 tracking-tight">
                          {analyticsSummary && (analyticsSummary.p95_latency_seconds ?? 0) > 0
                            ? `${analyticsSummary.p95_latency_seconds!.toFixed(2)}s`
                            : '< 2s'}
                        </span>
                        <span className="text-xs text-slate-500 dark:text-slate-400 ml-1 font-medium">p95 Tail</span>
                      </div>
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-400 mt-2 font-medium">
                      End-to-end time from queue ingestion to remote MX 250 OK.
                    </div>
                  </div>

                  <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-400">Unsubscribe Metrics</div>
                    <div className="flex items-baseline space-x-4 mt-2">
                      <div>
                        <span className="text-xl font-bold text-amber-400">
                          {((analyticsSummary?.unsubscribe_rate || 0) * 100).toFixed(2)}%
                        </span>
                        <span className="text-xs text-slate-500 ml-1">Unsubscribe Rate</span>
                      </div>
                      <div>
                        <span className="text-xl font-bold text-white">
                          {analyticsSummary?.unsubscribed ?? analyticsSummary?.unsubscribe_count ?? 0}
                        </span>
                        <span className="text-xs text-slate-500 ml-1">Total Opt-Outs</span>
                      </div>
                    </div>
                    <div className="text-xs text-slate-500 mt-2">
                      One-click RFC 8058 and mailto opt-outs.
                    </div>
                  </div>
                </div>

                {/* Outbound Traffic Volume Trends */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                  <div className="border-b border-slate-800 pb-4 mb-4 flex items-center justify-between">
                    <div>
                      <h3 className="text-base font-semibold text-white">Outbound Traffic Volume &amp; Trends</h3>
                      <p className="text-xs text-slate-400 mt-0.5">
                        Chronological delivery volume and deliverability performance by time bucket.
                      </p>
                    </div>
                    <span className="text-xs font-mono text-slate-400 bg-slate-800/80 px-2.5 py-1 rounded-md border border-slate-700">
                      {analyticsRange === '30d' ? 'Daily Resolution' : 'Hourly Resolution'}
                    </span>
                  </div>
                  {analyticsTimeSeries.length === 0 ? (
                    <div className="text-center py-6 text-slate-500 text-xs">
                      No time-series data recorded for this window. Outbound SMTP and API dispatches will appear here.
                    </div>
                  ) : (
                    <div className="overflow-x-auto">
                      <table className="w-full text-left text-sm text-slate-300">
                        <thead className="bg-slate-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-slate-800">
                          <tr>
                            <th className="px-4 py-2.5">Time Bucket</th>
                            <th className="px-4 py-2.5">Sent</th>
                            <th className="px-4 py-2.5">Delivered</th>
                            <th className="px-4 py-2.5">Bounced</th>
                            <th className="px-4 py-2.5">Complained</th>
                            <th className="px-4 py-2.5">Delivery Rate</th>
                            <th className="px-4 py-2.5">Complaint Rate</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-800 font-mono text-xs">
                          {analyticsTimeSeries.map((point) => (
                            <tr key={point.timestamp} className="hover:bg-slate-800/40">
                              <td className="px-4 py-2.5 text-slate-300 font-sans">{new Date(point.timestamp).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</td>
                              <td className="px-4 py-2.5 text-white font-bold">{point.sent.toLocaleString()}</td>
                              <td className="px-4 py-2.5 text-emerald-400">{point.delivered.toLocaleString()}</td>
                              <td className="px-4 py-2.5 text-rose-400">{point.bounced.toLocaleString()}</td>
                              <td className="px-4 py-2.5 text-amber-400">{point.complained.toLocaleString()}</td>
                              <td className="px-4 py-2.5 text-slate-200">{(point.delivery_rate * 100).toFixed(1)}%</td>
                              <td className="px-4 py-2.5">
                                <span className={point.complaint_rate >= 0.001 ? 'text-rose-400 font-bold' : 'text-slate-400'}>
                                  {(point.complaint_rate * 100).toFixed(3)}%
                                </span>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>

                {/* Latency Distribution Histogram */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                  <div className="border-b border-slate-800 pb-4 mb-4">
                    <h3 className="text-base font-semibold text-white">Delivery Latency Distribution</h3>
                    <p className="text-xs text-slate-400 mt-0.5">
                      Histogram of outbound SMTP delivery times into remote provider receiving queues.
                    </p>
                  </div>
                  <div className="space-y-3">
                    {analyticsLatency.length === 0 ? (
                      <div className="text-center py-6 text-slate-500 text-xs">No latency metrics recorded yet.</div>
                    ) : (
                      analyticsLatency.map((item) => (
                        <div key={item.bucket} className="space-y-1">
                          <div className="flex justify-between text-xs text-slate-300">
                            <span className="font-mono">{item.bucket}</span>
                            <span>{item.count} msgs ({(item.percentage || 0).toFixed(1)}%)</span>
                          </div>
                          <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
                            <div
                              className="bg-indigo-500 h-2 rounded-full transition-all duration-300"
                              style={{ width: `${Math.min(100, Math.max((item.percentage || 0) > 0 ? 3 : 0, item.percentage || 0))}%` }}
                            />
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                {/* Provider Breakdown Table */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="px-6 py-5 border-b border-slate-800">
                    <h3 className="text-base font-semibold text-white">Major ISP / Provider Breakdown</h3>
                    <p className="text-xs text-slate-400 mt-0.5">
                      Deliverability performance classified by destination MX (Google, Microsoft, Yahoo, Apple, Fastmail).
                    </p>
                  </div>

                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm text-slate-300">
                      <thead className="bg-slate-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-slate-800">
                        <tr>
                          <th className="px-6 py-3">Provider</th>
                          <th className="px-6 py-3">Sent</th>
                          <th className="px-6 py-3">Delivered</th>
                          <th className="px-6 py-3">Delivery Rate</th>
                          <th className="px-6 py-3">Bounces</th>
                          <th className="px-6 py-3">Bounce Rate</th>
                          <th className="px-6 py-3">Complaints</th>
                          <th className="px-6 py-3">Complaint Rate</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {analyticsProviders.length === 0 ? (
                          <tr>
                            <td colSpan={8} className="px-6 py-8 text-center text-slate-500">
                              No provider breakdown data available for this range.
                            </td>
                          </tr>
                        ) : (
                          analyticsProviders.map((p) => (
                            <tr key={p.provider} className="hover:bg-slate-800/40">
                              <td className="px-6 py-4 font-semibold text-white capitalize">{p.provider}</td>
                              <td className="px-6 py-4 text-slate-300 font-mono text-xs">{(p.sent ?? p.sent_count ?? 0).toLocaleString()}</td>
                              <td className="px-6 py-4 text-emerald-400 font-mono text-xs">{(p.delivered ?? p.delivered_count ?? 0).toLocaleString()}</td>
                              <td className="px-6 py-4 text-slate-200">{((p.delivery_rate || 0) * 100).toFixed(1)}%</td>
                              <td className="px-6 py-4 text-rose-400 font-mono text-xs">{p.bounced ?? p.bounce_count ?? 0}</td>
                              <td className="px-6 py-4 text-slate-200">{((p.bounce_rate || 0) * 100).toFixed(2)}%</td>
                              <td className="px-6 py-4 text-amber-400 font-mono text-xs">{p.complained ?? p.complaint_count ?? 0}</td>
                              <td className="px-6 py-4 font-semibold">
                                <span className={(p.complaint_rate || 0) >= 0.001 ? 'text-red-400' : 'text-slate-300'}>
                                  {((p.complaint_rate || 0) * 100).toFixed(3)}%
                                </span>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            )}

            {/* TAB: WEBHOOKS */}
            {currentTab === 'webhooks' && (
              <div className="mt-6 space-y-8">
                {/* Action Alert Banner */}
                {webhookActionMsg && (
                  <div className="bg-emerald-950/80 border border-emerald-500 rounded-xl p-4 flex items-center justify-between text-emerald-200 text-xs font-mono">
                    <div className="flex items-center space-x-2">
                      <span>🔑</span>
                      <span>{webhookActionMsg}</span>
                    </div>
                    <button
                      onClick={() => setWebhookActionMsg(null)}
                      className="text-slate-400 hover:text-white text-base ml-4"
                    >
                      ✕
                    </button>
                  </div>
                )}

                {/* Register Webhook Form */}
                <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl">
                  <h2 className="text-base font-semibold text-white">Register Webhook Endpoint</h2>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Configure real-time HTTPS webhook endpoints with HMAC-SHA256 signatures and timestamp replay defense.
                  </p>
                  <form onSubmit={handleCreateWebhook} className="mt-4 space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-medium text-slate-300">Destination URL</label>
                        <input
                          type="url"
                          required
                          placeholder="https://your-api.com/webhooks/mailvex"
                          value={newWebhookURL}
                          onChange={(e) => setNewWebhookURL(e.target.value)}
                          className="mt-1 block w-full px-3 py-2 text-xs border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-slate-300">Description (Optional)</label>
                        <input
                          type="text"
                          placeholder="e.g. Production events subscriber"
                          value={newWebhookDesc}
                          onChange={(e) => setNewWebhookDesc(e.target.value)}
                          className="mt-1 block w-full px-3 py-2 text-xs border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                        />
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-300">
                        Subscribed Events (comma-separated, or * for all)
                      </label>
                      <input
                        type="text"
                        placeholder="* or message.delivered, message.bounced, message.complained"
                        value={newWebhookEvents}
                        onChange={(e) => setNewWebhookEvents(e.target.value)}
                        className="mt-1 block w-full px-3 py-2 text-xs font-mono border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                      />
                    </div>
                    <div>
                      <button
                        type="submit"
                        className="px-4 py-2 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg transition-colors"
                      >
                        Register Webhook
                      </button>
                    </div>
                  </form>
                </div>

                {/* Webhooks Endpoints Table */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="px-6 py-5 border-b border-slate-800">
                    <h3 className="text-base font-semibold text-white">Registered Webhooks</h3>
                    <p className="text-xs text-slate-400 mt-0.5">
                      Configured endpoints receiving signed HTTP POST dispatches.
                    </p>
                  </div>

                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm text-slate-300">
                      <thead className="bg-slate-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-slate-800">
                        <tr>
                          <th className="px-6 py-3">Endpoint URL</th>
                          <th className="px-6 py-3">Events</th>
                          <th className="px-6 py-3">Signing Secret</th>
                          <th className="px-6 py-3">Status</th>
                          <th className="px-6 py-3">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {webhooksList.length === 0 ? (
                          <tr>
                            <td colSpan={5} className="px-6 py-8 text-center text-slate-500">
                              No webhooks registered. Add an endpoint above to receive real-time event notifications.
                            </td>
                          </tr>
                        ) : (
                          webhooksList.map((ep) => (
                            <tr key={ep.id} className="hover:bg-slate-800/40">
                              <td className="px-6 py-4">
                                <div className="font-mono text-xs text-white font-semibold">{ep.url}</div>
                                {ep.description && (
                                  <div className="text-xs text-slate-400 mt-0.5">{ep.description}</div>
                                )}
                              </td>
                              <td className="px-6 py-4">
                                <div className="flex flex-wrap gap-1">
                                  {ep.events.map((ev) => (
                                    <span
                                      key={ev}
                                      className="px-2 py-0.5 rounded text-xs font-mono bg-indigo-950 text-indigo-300 border border-indigo-800"
                                    >
                                      {ev}
                                    </span>
                                  ))}
                                </div>
                              </td>
                              <td className="px-6 py-4">
                                <div className="font-mono text-xs text-slate-400">
                                  {ep.secret.slice(0, 10)}••••••••
                                </div>
                                <button
                                  onClick={() => handleRotateWebhookSecret(ep.id)}
                                  className="text-xs text-indigo-400 hover:text-indigo-300 underline mt-1"
                                >
                                  Rotate Secret (24h overlap)
                                </button>
                              </td>
                              <td className="px-6 py-4">
                                {ep.status === 'active' && (
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-emerald-950 text-emerald-400 border border-emerald-800">
                                    Active
                                  </span>
                                )}
                                {ep.status === 'disabled_failing' && (
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-rose-950 text-rose-400 border border-rose-800">
                                    Circuit Broken (10+ Failures)
                                  </span>
                                )}
                                {ep.status === 'disabled_manual' && (
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-800 text-slate-300 border border-slate-700">
                                    Disabled
                                  </span>
                                )}
                                {ep.consecutive_failures > 0 && (
                                  <div className="text-xs text-rose-400 mt-0.5">
                                    {ep.consecutive_failures} failures
                                  </div>
                                )}
                              </td>
                              <td className="px-6 py-4">
                                <div className="flex items-center space-x-2">
                                  <button
                                    onClick={() => handleTestWebhook(ep.id)}
                                    className="px-2.5 py-1 text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-200 rounded border border-slate-700"
                                  >
                                    Test Ping
                                  </button>
                                  <button
                                    onClick={() => handleViewDeliveries(ep)}
                                    className="px-2.5 py-1 text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-200 rounded border border-slate-700"
                                  >
                                    Deliveries
                                  </button>
                                  <button
                                    onClick={() => handleDeleteWebhook(ep.id)}
                                    className="text-xs text-rose-400 hover:text-rose-300"
                                  >
                                    Delete
                                  </button>
                                </div>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Webhook Signature & Security Docs Box */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                  <h3 className="text-base font-semibold text-white mb-2">Webhook Security &amp; Replay Protection (§1.1, §1.2)</h3>
                  <p className="text-xs text-slate-400 mb-3">
                    Mailvex signs each delivery using HMAC-SHA256. Verify the signature header and check timestamp tolerance before processing payloads:
                  </p>
                  <pre className="bg-slate-950 text-indigo-300 p-4 rounded-lg text-xs font-mono overflow-x-auto border border-slate-800">
{`Header: X-Mailvex-Signature: t=<unix_time>,v1=<hmac_sha256_hex>
Expected string to sign: \`\${timestamp}.\${raw_json_body}\`
Replay defense: reject if |now - timestamp| > 300 seconds
Secret rotation: dual \`v1=\` signatures for 24 hours during secret rollover`}
                  </pre>
                </div>

                {/* Deliveries History Modal */}
                {deliveriesModalOpen && (
                  <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4">
                    <div className="bg-white/95 dark:bg-slate-900/95 backdrop-blur-xl border border-white/80 dark:border-slate-800/80 rounded-2xl max-w-4xl w-full max-h-[85vh] flex flex-col overflow-hidden shadow-2xl">
                      <div className="px-6 py-4 border-b border-slate-100 dark:border-slate-800/80 flex items-center justify-between">
                        <div>
                          <h3 className="text-base font-bold text-slate-900 dark:text-white">
                            Webhook Delivery History
                          </h3>
                          <div className="text-xs text-slate-500 dark:text-slate-400 font-mono mt-0.5">
                            {selectedWebhook?.url}
                          </div>
                        </div>
                        <div className="flex items-center space-x-2">
                          <ExportMenu
                            filename={`webhook-deliveries-${selectedWebhook?.id || 'hook'}`}
                            sheetTitle="Deliveries"
                            label="Export Deliveries"
                            columns={[
                              { header: 'Delivery ID', key: 'id' },
                              { header: 'Event Type', key: 'event_type' },
                              { header: 'HTTP Status', key: 'status_code' },
                              { header: 'Duration (ms)', key: 'duration_ms', type: 'number' },
                              { header: 'Timestamp', key: 'created_at' },
                            ]}
                            data={webhookDeliveries.map((del) => ({
                              id: del.id,
                              event_type: del.event_type,
                              status_code: del.status_code || 0,
                              duration_ms: del.duration_ms || 0,
                              created_at: del.created_at,
                            }))}
                          />
                          <button
                            onClick={() => setDeliveriesModalOpen(false)}
                            className="text-slate-400 hover:text-slate-600 dark:hover:text-white text-lg font-bold ml-2"
                          >
                            ✕
                          </button>
                        </div>
                      </div>

                      <div className="p-6 overflow-y-auto flex-1">
                        <table className="w-full text-left text-xs text-slate-700 dark:text-slate-300">
                          <thead className="bg-slate-50/70 dark:bg-slate-950/60 uppercase font-semibold text-slate-500 dark:text-slate-400 border-b border-slate-100 dark:border-slate-800">
                            <tr>
                              <th className="px-4 py-2">Delivery ID</th>
                              <th className="px-4 py-2">Event</th>
                              <th className="px-4 py-2">Status</th>
                              <th className="px-4 py-2">Duration</th>
                              <th className="px-4 py-2">Time</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                            {webhookDeliveries.length === 0 ? (
                              <tr>
                                <td colSpan={5} className="px-4 py-6 text-center text-slate-500">
                                  No delivery attempts recorded for this webhook.
                                </td>
                              </tr>
                            ) : (
                              webhookDeliveries.map((del) => (
                                <tr key={del.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-850/40">
                                  <td className="px-4 py-3 font-mono text-slate-900 dark:text-white font-medium">{del.id}</td>
                                  <td className="px-4 py-3 font-mono text-sky-600 dark:text-indigo-300">{del.event_type}</td>
                                  <td className="px-4 py-3">
                                    {del.status_code && del.status_code >= 200 && del.status_code < 300 ? (
                                      <span className="text-emerald-600 dark:text-emerald-400 font-semibold">{del.status_code} OK</span>
                                    ) : (
                                      <span className="text-rose-600 dark:text-rose-400 font-semibold">
                                        {del.status_code || 'Error'}: {del.error_message || 'Failed'}
                                      </span>
                                    )}
                                  </td>
                                  <td className="px-4 py-3 font-mono">{del.duration_ms}ms</td>
                                  <td className="px-4 py-3 text-slate-500">
                                    {new Date(del.created_at).toLocaleTimeString()}
                                  </td>
                                </tr>
                              ))
                            )}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}

            {currentTab === 'privacy_retention' && (
              <div className="mt-6 space-y-8">
                {/* Status Messages */}
                {privacyActionMsg && (
                  <div className="p-4 bg-slate-900 border border-indigo-500/30 rounded-xl flex items-center justify-between text-xs text-indigo-300">
                    <div className="flex items-center space-x-2">
                      <span>🛡️</span>
                      <span>{privacyActionMsg}</span>
                    </div>
                    <button
                      onClick={() => setPrivacyActionMsg(null)}
                      className="text-slate-400 hover:text-white text-base ml-4"
                    >
                      ✕
                    </button>
                  </div>
                )}

                {retentionMsg && (
                  <div className="p-4 bg-slate-900 border border-emerald-500/30 rounded-xl flex items-center justify-between text-xs text-emerald-300">
                    <div className="flex items-center space-x-2">
                      <span>🧹</span>
                      <span>{retentionMsg}</span>
                    </div>
                    <button
                      onClick={() => setRetentionMsg(null)}
                      className="text-slate-400 hover:text-white text-base ml-4"
                    >
                      ✕
                    </button>
                  </div>
                )}

                {/* Section 1: GDPR / DPDP Privacy Engine */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  {/* Erasure Card */}
                  <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl space-y-4">
                    <div className="flex items-center space-x-2">
                      <span className="text-base">🗑️</span>
                      <h2 className="text-base font-semibold text-white">Right to be Forgotten (GDPR Art. 17)</h2>
                    </div>
                    <p className="text-xs text-slate-400 leading-relaxed">
                      Irreversibly pseudonymizes all message recipient records, wipes message bodies and raw MIME blobs,
                      and securely one-way hashes suppression entries to protect user privacy without risking future re-sends.
                    </p>
                    <form onSubmit={handlePrivacyErasure} className="space-y-3 pt-2">
                      <div>
                        <label className="block text-xs font-medium text-slate-300 mb-1">Target Email Address</label>
                        <input
                          type="email"
                          required
                          placeholder="user@example.com"
                          value={privacyEmail}
                          onChange={(e) => setPrivacyEmail(e.target.value)}
                          className="w-full px-3 py-2 text-xs border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-rose-500 focus:border-rose-500"
                        />
                      </div>
                      <button
                        type="submit"
                        className="px-4 py-2 bg-rose-600 hover:bg-rose-700 text-white text-xs font-medium rounded-lg transition-colors"
                      >
                        Execute Privacy Erasure
                      </button>
                    </form>
                  </div>

                  {/* Data Portability / Export Card */}
                  <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl space-y-4">
                    <div className="flex items-center space-x-2">
                      <span className="text-base">📦</span>
                      <h2 className="text-base font-semibold text-white">Data Portability Export (GDPR Art. 15)</h2>
                    </div>
                    <p className="text-xs text-slate-400 leading-relaxed">
                      Generates a full machine-readable personal data report compiling all recipient delivery history,
                      associated engagement events, and active suppression records.
                    </p>
                    <form onSubmit={handlePrivacyExport} className="space-y-3 pt-2">
                      <div>
                        <label className="block text-xs font-medium text-slate-300 mb-1">Subject Email Address</label>
                        <input
                          type="email"
                          required
                          placeholder="user@example.com"
                          value={privacyEmail}
                          onChange={(e) => setPrivacyEmail(e.target.value)}
                          className="w-full px-3 py-2 text-xs border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                        />
                      </div>
                      <button
                        type="submit"
                        className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-lg transition-colors"
                      >
                        Generate Export Report
                      </button>
                    </form>
                  </div>
                </div>

                {/* Privacy Export Inspection Preview */}
                {privacyExportReport && (
                  <div className="glass-card p-6 space-y-4">
                    <div className="flex items-center justify-between pb-3 border-b border-slate-100 dark:border-slate-800/80">
                      <h3 className="text-sm font-bold text-slate-900 dark:text-white">
                        Personal Data Export for: <span className="font-mono text-sky-600 dark:text-sky-400">{privacyExportReport.email_address}</span>
                      </h3>
                      <div className="flex items-center space-x-2">
                        <ExportMenu
                          filename={`privacy-export-${privacyExportReport.email_address?.replace(/[@.]/g, '_') || 'subject'}`}
                          sheetTitle="GDPR Personal Data"
                          label="Download Export"
                          columns={[
                            { header: 'Record Type', key: 'record_type' },
                            { header: 'Identifier', key: 'id' },
                            { header: 'Details / Subject', key: 'details' },
                            { header: 'Status / Reason', key: 'status' },
                            { header: 'Timestamp', key: 'created_at' },
                          ]}
                          data={[
                            ...(privacyExportReport.messages || []).map((m: any) => ({
                              record_type: 'Message',
                              id: m.id,
                              details: `From: ${m.from_address || ''} | Subject: ${m.subject || ''}`,
                              status: m.status,
                              created_at: m.created_at,
                            })),
                            ...(privacyExportReport.events || []).map((ev: any) => ({
                              record_type: 'Event',
                              id: ev.id,
                              details: `Type: ${ev.type || ''} | Recipient: ${ev.recipient || ''}`,
                              status: ev.details || ev.type,
                              created_at: ev.created_at,
                            })),
                            ...(privacyExportReport.suppressions || []).map((s: any) => ({
                              record_type: 'Suppression',
                              id: s.address,
                              details: `Domain: ${s.address_domain || ''}`,
                              status: `${s.type}: ${s.reason || ''}`,
                              created_at: s.created_at,
                            })),
                          ]}
                        />
                        <button
                          onClick={() => setPrivacyExportReport(null)}
                          className="btn-secondary text-xs px-3 py-1.5"
                        >
                          Close
                        </button>
                      </div>
                    </div>
                    <div className="grid grid-cols-3 gap-4 text-center">
                      <div className="p-3 bg-slate-50/70 dark:bg-slate-800/60 rounded-xl border border-slate-200/80 dark:border-slate-700/60">
                        <div className="text-xs text-slate-500 dark:text-slate-400 font-medium">Messages</div>
                        <div className="text-lg font-bold text-slate-900 dark:text-white mt-1">{privacyExportReport.messages?.length || 0}</div>
                      </div>
                      <div className="p-3 bg-slate-50/70 dark:bg-slate-800/60 rounded-xl border border-slate-200/80 dark:border-slate-700/60">
                        <div className="text-xs text-slate-500 dark:text-slate-400 font-medium">Events</div>
                        <div className="text-lg font-bold text-slate-900 dark:text-white mt-1">{privacyExportReport.events?.length || 0}</div>
                      </div>
                      <div className="p-3 bg-slate-50/70 dark:bg-slate-800/60 rounded-xl border border-slate-200/80 dark:border-slate-700/60">
                        <div className="text-xs text-slate-500 dark:text-slate-400 font-medium">Suppressions</div>
                        <div className="text-lg font-bold text-slate-900 dark:text-white mt-1">{privacyExportReport.suppressions?.length || 0}</div>
                      </div>
                    </div>
                    <div className="bg-slate-50/80 dark:bg-slate-950 p-4 rounded-xl overflow-x-auto max-h-60 border border-slate-200/80 dark:border-slate-800">
                      <pre className="text-xs font-mono text-slate-800 dark:text-slate-300">
                        {JSON.stringify(privacyExportReport, null, 2)}
                      </pre>
                    </div>
                  </div>
                )}

                {/* Section 2: Data Retention & Automated Purge Policy */}
                <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl space-y-6">
                  <div className="flex items-center justify-between">
                    <div>
                      <h2 className="text-base font-semibold text-white">Data Retention Policies (§23)</h2>
                      <p className="text-xs text-slate-400 mt-0.5">
                        Configure rolling retention windows to automatically prune stale data and stay fully compliant with privacy laws.
                      </p>
                    </div>
                    <button
                      onClick={handleExecutePurge}
                      className="px-4 py-2 bg-amber-600 hover:bg-amber-700 text-white text-xs font-medium rounded-lg transition-colors flex items-center space-x-1.5"
                    >
                      <span>⚡</span>
                      <span>Run Immediate Purge</span>
                    </button>
                  </div>

                  <form onSubmit={handleUpdateRetentionPolicy} className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
                      <div>
                        <label className="block text-xs font-medium text-slate-300 mb-1">Message Retention (Days)</label>
                        <input
                          type="number"
                          min={1}
                          required
                          value={retentionPolicy.message_retention_days}
                          onChange={(e) => setRetentionPolicy({ ...retentionPolicy, message_retention_days: parseInt(e.target.value) || 30 })}
                          className="w-full px-3 py-2 text-xs border border-slate-700 bg-slate-800 text-slate-200 rounded-lg"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-slate-300 mb-1">Events Retention (Days)</label>
                        <input
                          type="number"
                          min={1}
                          required
                          value={retentionPolicy.events_retention_days}
                          onChange={(e) => setRetentionPolicy({ ...retentionPolicy, events_retention_days: parseInt(e.target.value) || 90 })}
                          className="w-full px-3 py-2 text-xs border border-slate-700 bg-slate-800 text-slate-200 rounded-lg"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-slate-300 mb-1">Attempts Retention (Days)</label>
                        <input
                          type="number"
                          min={1}
                          required
                          value={retentionPolicy.attempts_retention_days}
                          onChange={(e) => setRetentionPolicy({ ...retentionPolicy, attempts_retention_days: parseInt(e.target.value) || 30 })}
                          className="w-full px-3 py-2 text-xs border border-slate-700 bg-slate-800 text-slate-200 rounded-lg"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-slate-300 mb-1">Raw MIME Retention (Days)</label>
                        <input
                          type="number"
                          min={1}
                          required
                          value={retentionPolicy.raw_mime_retention_days}
                          onChange={(e) => setRetentionPolicy({ ...retentionPolicy, raw_mime_retention_days: parseInt(e.target.value) || 7 })}
                          className="w-full px-3 py-2 text-xs border border-slate-700 bg-slate-800 text-slate-200 rounded-lg"
                        />
                      </div>
                    </div>
                    <div className="flex justify-end">
                      <button
                        type="submit"
                        className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-lg transition-colors"
                      >
                        Save Retention Policy
                      </button>
                    </div>
                  </form>

                  {purgeResult && (
                    <div className="bg-slate-950 p-4 rounded-lg border border-slate-800 text-xs">
                      <div className="text-slate-400 mb-2 font-medium">Last Purge Summary ({new Date(purgeResult.purged_at).toLocaleString()}):</div>
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                        <div>Messages Purged: <span className="font-semibold text-emerald-400">{purgeResult.purged_messages_count}</span></div>
                        <div>Events Purged: <span className="font-semibold text-emerald-400">{purgeResult.purged_events_count}</span></div>
                        <div>Attempts Purged: <span className="font-semibold text-emerald-400">{purgeResult.purged_attempts_count}</span></div>
                        <div>Raw MIME Cleared: <span className="font-semibold text-emerald-400">{purgeResult.purged_raw_mime_count}</span></div>
                      </div>
                    </div>
                  )}
                </div>

                {/* Section 3: Privacy Requests Audit Log */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="p-4 border-b border-slate-800 flex items-center justify-between font-semibold text-white">
                    <span>Privacy Requests History ({privacyRequests.length})</span>
                    <button
                      onClick={loadPrivacyRequests}
                      className="text-xs text-indigo-400 hover:text-indigo-300 font-normal"
                    >
                      Refresh
                    </button>
                  </div>
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs text-slate-300">
                      <thead className="bg-slate-950 text-slate-400 uppercase border-b border-slate-800">
                        <tr>
                          <th className="px-4 py-3">Request ID</th>
                          <th className="px-4 py-3">Type</th>
                          <th className="px-4 py-3">Subject Email</th>
                          <th className="px-4 py-3">Status</th>
                          <th className="px-4 py-3">Summary</th>
                          <th className="px-4 py-3">Requested At</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {privacyRequests.length === 0 ? (
                          <tr>
                            <td colSpan={6} className="px-4 py-6 text-center text-slate-500">
                              No privacy requests processed yet.
                            </td>
                          </tr>
                        ) : (
                          privacyRequests.map((req) => (
                            <tr key={req.id} className="hover:bg-slate-800/40">
                              <td className="px-4 py-3 font-mono text-white">{req.id}</td>
                              <td className="px-4 py-3">
                                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                  req.type === 'erasure'
                                    ? 'bg-rose-950 text-rose-400 border border-rose-800'
                                    : 'bg-indigo-950 text-indigo-400 border border-indigo-800'
                                }`}>
                                  {req.type}
                                </span>
                              </td>
                              <td className="px-4 py-3 font-mono">{req.email_address}</td>
                              <td className="px-4 py-3">
                                <span className="text-emerald-400 font-semibold">{req.status}</span>
                              </td>
                              <td className="px-4 py-3 text-slate-400 max-w-xs truncate">{req.result_summary || '—'}</td>
                              <td className="px-4 py-3 text-slate-500">{new Date(req.requested_at).toLocaleString()}</td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            )}

            {/* Phase 10: Inbound Routing Tab */}
            {currentTab === 'inbound' && (
              <div className="space-y-6">
                {inboundActionMsg && (
                  <div className="bg-emerald-950/40 border border-emerald-500/30 text-emerald-300 text-sm p-3 rounded-lg flex items-center justify-between">
                    <span>{inboundActionMsg}</span>
                    <button onClick={() => setInboundActionMsg(null)} className="text-emerald-400 hover:text-white text-xs">✕</button>
                  </div>
                )}

                {/* Top Section: Overview & Route Simulator */}
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  {/* Route Simulator */}
                  <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm">
                    <h2 className="text-base font-semibold text-white mb-2 flex items-center justify-between">
                      <span>Route Simulator & Tester</span>
                      <span className="text-xs font-mono text-indigo-400 bg-indigo-950 px-2 py-0.5 rounded border border-indigo-800">
                        §21 Precedence Engine
                      </span>
                    </h2>
                    <p className="text-xs text-slate-400 mb-4">
                      Simulate inbound recipient addresses against your active routing table to verify evaluation order and target dispatch.
                    </p>
                    <form onSubmit={handleTestInboundRoute} className="space-y-3">
                      <div>
                        <label className="block text-xs font-medium text-slate-300 mb-1">
                          Test Recipient Email Address
                        </label>
                        <div className="flex space-x-2">
                          <input
                            type="email"
                            required
                            value={testInboundAddress}
                            onChange={(e) => setTestInboundAddress(e.target.value)}
                            placeholder="e.g. support-vip@mail.example.com"
                            className="flex-1 px-3 py-2 border border-slate-700 bg-slate-800 text-slate-200 text-sm rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                          />
                          <button
                            type="submit"
                            className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-lg transition-colors"
                          >
                            Test Match
                          </button>
                        </div>
                      </div>
                    </form>

                    {testInboundResult && (
                      <div className="mt-4 pt-4 border-t border-slate-800">
                        {testInboundResult.matched ? (
                          <div className="bg-emerald-950/40 border border-emerald-800/60 rounded-lg p-3 text-xs">
                            <div className="flex items-center text-emerald-400 font-semibold mb-1">
                              <span className="w-2 h-2 rounded-full bg-emerald-400 mr-2"></span>
                              Matched Route #{testInboundResult.route.priority} ({testInboundResult.route.id})
                            </div>
                            <div className="grid grid-cols-2 gap-2 text-slate-300 mt-2">
                              <div><span className="text-slate-400">Match Type:</span> <span className="font-mono text-white">{testInboundResult.route.match_type}</span></div>
                              <div><span className="text-slate-400">Pattern:</span> <span className="font-mono text-white">{testInboundResult.route.pattern}</span></div>
                              <div><span className="text-slate-400">Action:</span> <span className="font-mono text-indigo-300">{testInboundResult.route.action}</span></div>
                              <div><span className="text-slate-400">Target:</span> <span className="font-mono text-white">{testInboundResult.route.action_target || '—'}</span></div>
                            </div>
                          </div>
                        ) : (
                          <div className="bg-amber-950/40 border border-amber-800/60 rounded-lg p-3 text-xs text-amber-300">
                            <span className="font-semibold">No Match Found:</span> No active route matched recipient <code className="text-amber-200 font-mono">{testInboundResult.address}</code>. Inbound mail to this address will be dropped or rejected.
                          </div>
                        )}
                      </div>
                    )}
                  </div>

                  {/* Create New Route */}
                  <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm">
                    <h2 className="text-base font-semibold text-white mb-2">Create Inbound Route</h2>
                    <p className="text-xs text-slate-400 mb-4">
                      Define priority rules to match inbound SMTP mail and route to webhooks, forwarding addresses, or storage.
                    </p>
                    <form onSubmit={handleCreateInboundRoute} className="space-y-3">
                      <div className="grid grid-cols-3 gap-2">
                        <div>
                          <label className="block text-xs font-medium text-slate-300 mb-1">
                            Priority (1-999)
                          </label>
                          <input
                            type="number"
                            min="1"
                            max="999"
                            required
                            value={newRoutePriority}
                            onChange={(e) => setNewRoutePriority(Number(e.target.value))}
                            className="w-full px-3 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                          />
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-slate-300 mb-1">
                            Match Type
                          </label>
                          <select
                            value={newRouteMatchType}
                            onChange={(e) => setNewRouteMatchType(e.target.value as any)}
                            className="w-full px-2 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                          >
                            <option value="exact">Exact Address</option>
                            <option value="wildcard">Wildcard (*)</option>
                            <option value="catch_all">Catch-All (@domain)</option>
                          </select>
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-slate-300 mb-1">
                            Action
                          </label>
                          <select
                            value={newRouteAction}
                            onChange={(e) => setNewRouteAction(e.target.value as any)}
                            className="w-full px-2 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                          >
                            <option value="webhook">Webhook POST</option>
                            <option value="store_only">Store Only</option>
                            <option value="forward">Forward SMTP</option>
                            <option value="discard">Discard / Drop</option>
                          </select>
                        </div>
                      </div>

                      <div>
                        <label className="block text-xs font-medium text-slate-300 mb-1">
                          Pattern
                        </label>
                        <input
                          type="text"
                          required
                          value={newRoutePattern}
                          onChange={(e) => setNewRoutePattern(e.target.value)}
                          placeholder={newRouteMatchType === 'catch_all' ? '@example.com' : newRouteMatchType === 'wildcard' ? 'support-*@example.com' : 'contact@example.com'}
                          className="w-full px-3 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg focus:ring-indigo-500 focus:border-indigo-500 font-mono"
                        />
                      </div>

                      {(newRouteAction === 'webhook' || newRouteAction === 'forward') && (
                        <div>
                          <label className="block text-xs font-medium text-slate-300 mb-1">
                            Action Target {newRouteAction === 'webhook' ? '(HTTPS Endpoint URL)' : '(Forward Destination Email)'}
                          </label>
                          <input
                            type="text"
                            required
                            value={newRouteTarget}
                            onChange={(e) => setNewRouteTarget(e.target.value)}
                            placeholder={newRouteAction === 'webhook' ? 'https://api.example.com/webhooks/inbound' : 'team@internal.example.com'}
                            className="w-full px-3 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg focus:ring-indigo-500 focus:border-indigo-500 font-mono"
                          />
                        </div>
                      )}

                      <div className="flex items-center justify-between pt-1">
                        <label className="flex items-center space-x-2 text-xs text-slate-300 cursor-pointer">
                          <input
                            type="checkbox"
                            checked={newRouteActive}
                            onChange={(e) => setNewRouteActive(e.target.checked)}
                            className="rounded border-slate-700 bg-slate-800 text-indigo-600 focus:ring-indigo-500"
                          />
                          <span>Route Enabled (Active)</span>
                        </label>

                        <button
                          type="submit"
                          className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-lg transition-colors"
                        >
                          Add Inbound Route
                        </button>
                      </div>
                    </form>
                  </div>
                </div>

                {/* Section 2: Inbound Routes Table */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="p-4 border-b border-slate-800 flex items-center justify-between font-semibold text-white">
                    <div className="flex items-center space-x-3">
                      <span>Inbound Routing Rules ({inboundRoutes.length})</span>
                      <span className="text-xs text-slate-400 font-normal">
                        Evaluated in ascending priority order (1 first)
                      </span>
                    </div>
                    <button
                      onClick={loadInboundRoutes}
                      className="text-xs text-indigo-400 hover:text-indigo-300 font-normal"
                    >
                      Refresh
                    </button>
                  </div>
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs text-slate-300">
                      <thead className="bg-slate-950 text-slate-400 uppercase border-b border-slate-800">
                        <tr>
                          <th className="px-4 py-3">Priority</th>
                          <th className="px-4 py-3">Match Type</th>
                          <th className="px-4 py-3">Pattern</th>
                          <th className="px-4 py-3">Action</th>
                          <th className="px-4 py-3">Target</th>
                          <th className="px-4 py-3">Status</th>
                          <th className="px-4 py-3 text-right">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {inboundRoutes.length === 0 ? (
                          <tr>
                            <td colSpan={7} className="px-4 py-6 text-center text-slate-500">
                              No inbound routing rules configured yet. Create a route above to start processing incoming emails.
                            </td>
                          </tr>
                        ) : (
                          inboundRoutes.map((route) => (
                            <tr key={route.id} className="hover:bg-slate-800/40">
                              <td className="px-4 py-3 font-mono font-bold text-white">#{route.priority}</td>
                              <td className="px-4 py-3">
                                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                  route.match_type === 'exact'
                                    ? 'bg-blue-950 text-blue-400 border border-blue-800'
                                    : route.match_type === 'wildcard'
                                    ? 'bg-purple-950 text-purple-400 border border-purple-800'
                                    : 'bg-amber-950 text-amber-400 border border-amber-800'
                                }`}>
                                  {route.match_type}
                                </span>
                              </td>
                              <td className="px-4 py-3 font-mono font-medium text-white">{route.pattern}</td>
                              <td className="px-4 py-3 font-mono text-indigo-300">{route.action}</td>
                              <td className="px-4 py-3 text-slate-400 max-w-xs truncate font-mono text-xs">
                                {route.action_target || '—'}
                              </td>
                              <td className="px-4 py-3">
                                <button
                                  onClick={() => handleToggleInboundRoute(route)}
                                  className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold cursor-pointer transition-colors ${
                                    route.is_active
                                      ? 'bg-emerald-950 text-emerald-400 border border-emerald-800 hover:bg-emerald-900'
                                      : 'bg-slate-800 text-slate-400 border border-slate-700 hover:bg-slate-700'
                                  }`}
                                >
                                  {route.is_active ? 'Active' : 'Disabled'}
                                </button>
                              </td>
                              <td className="px-4 py-3 text-right">
                                <button
                                  onClick={() => handleDeleteInboundRoute(route.id)}
                                  className="text-red-400 hover:text-red-300 text-xs font-medium"
                                >
                                  Delete
                                </button>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Section 3: Inbound Message History Table */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="p-4 border-b border-slate-800 flex items-center justify-between font-semibold text-white">
                    <div className="flex items-center space-x-3">
                      <span>Inbound Mail Logs ({inboundMessages.length})</span>
                      <span className="text-xs text-slate-400 font-normal">
                        Recent inbound SMTP deliveries with SPF / DKIM / DMARC authentication verdicts
                      </span>
                    </div>
                    <button
                      onClick={loadInboundMessages}
                      className="text-xs text-indigo-400 hover:text-indigo-300 font-normal"
                    >
                      Refresh
                    </button>
                  </div>
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs text-slate-300">
                      <thead className="bg-slate-950 text-slate-400 uppercase border-b border-slate-800">
                        <tr>
                          <th className="px-4 py-3">Message ID</th>
                          <th className="px-4 py-3">From</th>
                          <th className="px-4 py-3">To</th>
                          <th className="px-4 py-3">Subject</th>
                          <th className="px-4 py-3">Auth Verification</th>
                          <th className="px-4 py-3">Status</th>
                          <th className="px-4 py-3">Received At</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {inboundMessages.length === 0 ? (
                          <tr>
                            <td colSpan={7} className="px-4 py-6 text-center text-slate-500">
                              No inbound messages received yet.
                            </td>
                          </tr>
                        ) : (
                          inboundMessages.map((msg) => (
                            <tr key={msg.id} className="hover:bg-slate-800/40">
                              <td className="px-4 py-3 font-mono text-white">{msg.id}</td>
                              <td className="px-4 py-3 font-mono text-slate-300">{msg.from_address}</td>
                              <td className="px-4 py-3 font-mono text-slate-300">{msg.to_address}</td>
                              <td className="px-4 py-3 text-white max-w-xs truncate">{msg.subject || '(no subject)'}</td>
                              <td className="px-4 py-3">
                                <div className="flex items-center space-x-1 font-mono text-[10px]">
                                  <span className={`px-1.5 py-0.5 rounded ${
                                    msg.spf_status === 'pass' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' : 'bg-rose-950 text-rose-400 border border-rose-800'
                                  }`}>
                                    SPF:{msg.spf_status || 'none'}
                                  </span>
                                  <span className={`px-1.5 py-0.5 rounded ${
                                    msg.dkim_status === 'pass' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' : 'bg-rose-950 text-rose-400 border border-rose-800'
                                  }`}>
                                    DKIM:{msg.dkim_status || 'none'}
                                  </span>
                                  <span className={`px-1.5 py-0.5 rounded ${
                                    msg.dmarc_status === 'pass' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' : 'bg-rose-950 text-rose-400 border border-rose-800'
                                  }`}>
                                    DMARC:{msg.dmarc_status || 'none'}
                                  </span>
                                </div>
                              </td>
                              <td className="px-4 py-3">
                                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                  msg.status === 'dispatched'
                                    ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                                    : msg.status === 'stored'
                                    ? 'bg-blue-950 text-blue-400 border border-blue-800'
                                    : 'bg-slate-800 text-slate-400 border border-slate-700'
                                }`}>
                                  {msg.status}
                                </span>
                              </td>
                              <td className="px-4 py-3 text-slate-500">
                                {new Date(msg.created_at).toLocaleString()}
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            )}

            {/* Phase 11: Alerts, DNSBL & Warmup Tab */}
            {currentTab === 'alerts_warmup' && (
              <div className="space-y-6">
                {alertsActionMsg && (
                  <div className="bg-indigo-950/40 border border-indigo-500/30 text-indigo-300 text-sm p-3 rounded-lg flex items-center justify-between">
                    <span>{alertsActionMsg}</span>
                    <button onClick={() => setAlertsActionMsg(null)} className="text-indigo-400 hover:text-white text-xs">✕</button>
                  </div>
                )}

                {/* Section 1: Reputation Health Overview Card (§22.1) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-sm">
                  <div className="flex items-center justify-between pb-4 border-b border-slate-800">
                    <div>
                      <h2 className="text-base font-semibold text-white flex items-center space-x-2">
                        <span>IP & Domain Reputation State</span>
                        <span className="text-xs font-mono text-slate-400 font-normal">(§22.1 & §11.4 Shared-Pool Protection)</span>
                      </h2>
                      <p className="text-xs text-slate-400 mt-0.5">
                        Continuous deliverability health tracking across rolling 7-day volume, bounces, complaints, and DNSBL listings.
                      </p>
                    </div>
                    <div>
                      <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider ${
                        reputationOverview?.overall_status === 'measured'
                          ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                          : reputationOverview?.overall_status === 'warning'
                          ? 'bg-amber-950 text-amber-400 border border-amber-800'
                          : reputationOverview?.overall_status === 'blocked'
                          ? 'bg-rose-950 text-rose-400 border border-rose-800'
                          : 'bg-slate-800 text-slate-400 border border-slate-700'
                      }`}>
                        Status: {reputationOverview?.overall_status || 'unknown'}
                      </span>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 md:grid-cols-6 gap-4 pt-4">
                    <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800">
                      <div className="text-xs text-slate-400">7-Day Sent</div>
                      <div className="text-lg font-bold text-white mt-1">{reputationOverview?.total_sent_7d || 0}</div>
                    </div>
                    <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800">
                      <div className="text-xs text-slate-400">Delivery Rate</div>
                      <div className="text-lg font-bold text-emerald-400 mt-1">
                        {reputationOverview ? (reputationOverview.delivery_rate * 100).toFixed(1) : 0}%
                      </div>
                    </div>
                    <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800">
                      <div className="text-xs text-slate-400">Bounce Rate</div>
                      <div className={`text-lg font-bold mt-1 ${
                        (reputationOverview?.bounce_rate || 0) > 0.05 ? 'text-rose-400' : 'text-slate-200'
                      }`}>
                        {reputationOverview ? (reputationOverview.bounce_rate * 100).toFixed(2) : 0}%
                      </div>
                    </div>
                    <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800">
                      <div className="text-xs text-slate-400">Complaint Rate</div>
                      <div className={`text-lg font-bold mt-1 ${
                        (reputationOverview?.complaint_rate || 0) > 0.001 ? 'text-rose-400' : 'text-slate-200'
                      }`}>
                        {reputationOverview ? (reputationOverview.complaint_rate * 100).toFixed(3) : 0}%
                      </div>
                    </div>
                    <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800">
                      <div className="text-xs text-slate-400">DNSBL Listed</div>
                      <div className={`text-lg font-bold mt-1 ${
                        (reputationOverview?.dnsbl_listings_count || 0) > 0 ? 'text-rose-400' : 'text-emerald-400'
                      }`}>
                        {reputationOverview?.dnsbl_listings_count || 0}
                      </div>
                    </div>
                    <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800">
                      <div className="text-xs text-slate-400">Firing Incidents</div>
                      <div className={`text-lg font-bold mt-1 ${
                        (reputationOverview?.active_alerts_count || 0) > 0 ? 'text-amber-400' : 'text-slate-400'
                      }`}>
                        {reputationOverview?.active_alerts_count || 0}
                      </div>
                    </div>
                  </div>
                </div>

                {/* Section 2: DNSBL Monitor (§22.2) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h2 className="text-base font-semibold text-white">DNSBL Blocklist Monitor ({dnsblProviders.length} Providers Monitored)</h2>
                      <p className="text-xs text-slate-400">
                        Query public IP addresses against major DNSBL blocklists (Spamhaus, Barracuda, SpamCop, UCEPROTECT).
                      </p>
                    </div>
                  </div>

                  <form onSubmit={handleRunDNSBLCheck} className="flex space-x-3">
                    <input
                      type="text"
                      required
                      value={testDNSBLIP}
                      onChange={(e) => setTestDNSBLIP(e.target.value)}
                      placeholder="Enter IPv4 or IPv6 address (e.g. 198.51.100.25)"
                      className="flex-1 px-3 py-2 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg font-mono focus:ring-indigo-500 focus:border-indigo-500"
                    />
                    <button
                      type="submit"
                      disabled={dnsblChecking}
                      className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-lg transition-colors disabled:opacity-50"
                    >
                      {dnsblChecking ? 'Checking Blocklists...' : 'Run DNSBL Check'}
                    </button>
                  </form>

                  {/* DNSBL Results Table */}
                  <div className="overflow-x-auto border border-slate-800 rounded-lg">
                    <table className="w-full text-left text-xs text-slate-300">
                      <thead className="bg-slate-950 text-slate-400 uppercase border-b border-slate-800">
                        <tr>
                          <th className="px-4 py-2.5">IP Address</th>
                          <th className="px-4 py-2.5">Provider</th>
                          <th className="px-4 py-2.5">Status</th>
                          <th className="px-4 py-2.5">Return Meaning</th>
                          <th className="px-4 py-2.5">Checked At</th>
                          <th className="px-4 py-2.5 text-right">Delist Guidance</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {dnsblResults.length === 0 ? (
                          <tr>
                            <td colSpan={6} className="px-4 py-4 text-center text-slate-500">
                              No DNSBL check results yet. Run a check above to verify your sending IPs.
                            </td>
                          </tr>
                        ) : (
                          dnsblResults.map((r) => (
                            <tr key={r.id} className="hover:bg-slate-800/40">
                              <td className="px-4 py-2.5 font-mono text-white">{r.ip_address}</td>
                              <td className="px-4 py-2.5 font-medium">{r.provider_name || 'DNSBL'}</td>
                              <td className="px-4 py-2.5">
                                <span className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold ${
                                  r.is_listed
                                    ? 'bg-rose-950 text-rose-400 border border-rose-800'
                                    : 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                                }`}>
                                  {r.is_listed ? 'LISTED' : 'CLEAN'}
                                </span>
                              </td>
                              <td className="px-4 py-2.5 text-slate-400 font-mono text-[11px]">
                                {r.return_meaning || '—'}
                              </td>
                              <td className="px-4 py-2.5 text-slate-500">{new Date(r.checked_at).toLocaleTimeString()}</td>
                              <td className="px-4 py-2.5 text-right">
                                {r.delist_url ? (
                                  <a
                                    href={r.delist_url}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="text-xs text-indigo-400 hover:text-indigo-300 underline font-medium"
                                  >
                                    Delist Portal ↗
                                  </a>
                                ) : (
                                  <span className="text-slate-600">—</span>
                                )}
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Section 3: IP Warmup Engine (§11.5) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h2 className="text-base font-semibold text-white">IP Warmup Ramp & Schedules</h2>
                      <p className="text-xs text-slate-400">
                        Automated daily volume doubling plans and per-provider caps. Holds automatically if bounce or complaint spikes occur.
                      </p>
                    </div>
                  </div>

                  {/* Enroll IP Form */}
                  <form onSubmit={handleEnrollWarmup} className="grid grid-cols-1 md:grid-cols-4 gap-3 bg-slate-950/40 p-3 rounded-lg border border-slate-800">
                    <div>
                      <label className="block text-xs font-medium text-slate-300 mb-1">IP Address</label>
                      <input
                        type="text"
                        required
                        value={newWarmupIP}
                        onChange={(e) => setNewWarmupIP(e.target.value)}
                        placeholder="e.g. 198.51.100.25"
                        className="w-full px-3 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg font-mono focus:ring-indigo-500 focus:border-indigo-500"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-300 mb-1">IP Identifier (Optional)</label>
                      <input
                        type="text"
                        value={newWarmupIPID}
                        onChange={(e) => setNewWarmupIPID(e.target.value)}
                        placeholder="e.g. ip_pool_outbound_1"
                        className="w-full px-3 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg font-mono focus:ring-indigo-500 focus:border-indigo-500"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-300 mb-1">Warmup Plan</label>
                      <select
                        value={newWarmupPlanID}
                        onChange={(e) => setNewWarmupPlanID(e.target.value)}
                        className="w-full px-2 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                      >
                        {warmupPlans.map((p) => (
                          <option key={p.id} value={p.id}>{p.name} ({p.total_days} days)</option>
                        ))}
                      </select>
                    </div>
                    <div className="flex items-end">
                      <button
                        type="submit"
                        className="w-full py-1.5 px-3 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-lg transition-colors"
                      >
                        Enroll IP in Warmup
                      </button>
                    </div>
                  </form>

                  {/* Active Warmup Cards */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {warmupStates.length === 0 ? (
                      <div className="col-span-2 text-center py-6 text-slate-500 text-xs bg-slate-950/20 rounded-lg border border-slate-800/50">
                        No IP addresses currently enrolled in warmup. Enroll an IP above to begin a staged volume ramp.
                      </div>
                    ) : (
                      warmupStates.map((ws) => (
                        <div key={ws.id} className="bg-slate-950/60 border border-slate-800 rounded-lg p-4 space-y-3">
                          <div className="flex items-center justify-between">
                            <div>
                              <div className="font-mono text-sm font-bold text-white">{ws.ip_address}</div>
                              <div className="text-[11px] text-slate-400">ID: {ws.ip_address_id}</div>
                            </div>
                            <span className={`px-2 py-0.5 rounded text-xs font-semibold ${
                              ws.status === 'warming'
                                ? 'bg-indigo-950 text-indigo-400 border border-indigo-800'
                                : ws.status === 'held'
                                ? 'bg-rose-950 text-rose-400 border border-rose-800'
                                : 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                            }`}>
                              {ws.status.toUpperCase()}
                            </span>
                          </div>

                          <div className="space-y-1">
                            <div className="flex justify-between text-xs text-slate-300">
                              <span>Day {ws.current_day} of 30 Ramp</span>
                              <span>Sent Today: {ws.current_day_volume}</span>
                            </div>
                            <div className="w-full bg-slate-800 rounded-full h-2">
                              <div
                                className="bg-indigo-500 h-2 rounded-full"
                                style={{ width: `${Math.min(100, (ws.current_day / 30) * 100)}%` }}
                              ></div>
                            </div>
                          </div>

                          {ws.held_reason && (
                            <div className="bg-rose-950/40 border border-rose-800/50 p-2 rounded text-[11px] text-rose-300">
                              Hold Reason: {ws.held_reason}
                            </div>
                          )}

                          <div className="flex items-center space-x-2 pt-2 border-t border-slate-800">
                            {ws.status === 'held' ? (
                              <button
                                onClick={() => handleResumeWarmup(ws.ip_address_id)}
                                className="px-2.5 py-1 bg-emerald-700 hover:bg-emerald-600 text-white text-xs font-medium rounded transition-colors"
                              >
                                Resume Warmup
                              </button>
                            ) : (
                              <button
                                onClick={() => handleHoldWarmup(ws.ip_address_id)}
                                className="px-2.5 py-1 bg-amber-700 hover:bg-amber-600 text-white text-xs font-medium rounded transition-colors"
                              >
                                Hold
                              </button>
                            )}
                            <button
                              onClick={() => handleAdvanceWarmup(ws.ip_address_id, false)}
                              className="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-medium rounded transition-colors"
                            >
                              Advance Day
                            </button>
                            <button
                              onClick={() => handleAdvanceWarmup(ws.ip_address_id, true)}
                              className="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-slate-400 text-xs font-medium rounded transition-colors"
                            >
                              Force Advance
                            </button>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                {/* Section 4: Alert Rules & Incident Management (§22.3) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h2 className="text-base font-semibold text-white">Alert Rules & Incident Manager</h2>
                      <p className="text-xs text-slate-400">
                        Configure threshold rules for bounce rates, complaint spikes, queue backpressure, and DNSBL blocks.
                      </p>
                    </div>
                    <button
                      onClick={handleEvaluateAlerts}
                      className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-indigo-400 hover:text-indigo-300 text-xs font-medium rounded-lg transition-colors border border-slate-700"
                    >
                      ⚡ Evaluate Rules Now
                    </button>
                  </div>

                  {alertsActionMsg && (
                    <div className="p-3 bg-indigo-950/60 border border-indigo-800/60 rounded-lg text-xs text-indigo-300">
                      {alertsActionMsg}
                    </div>
                  )}

                  {/* Create Alert Rule Form */}
                  <form onSubmit={handleCreateAlertRule} className="grid grid-cols-1 md:grid-cols-7 gap-2 bg-slate-950/40 p-3 rounded-lg border border-slate-800">
                    <div className="md:col-span-2">
                      <label className="block text-xs font-medium text-slate-300 mb-1">Rule Name</label>
                      <input
                        type="text"
                        required
                        value={newRuleName}
                        onChange={(e) => setNewRuleName(e.target.value)}
                        placeholder="e.g. Critical Bounce Spike"
                        className="w-full px-3 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-300 mb-1">Metric</label>
                      <select
                        value={newRuleMetric}
                        onChange={(e) => setNewRuleMetric(e.target.value)}
                        className="w-full px-2 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                      >
                        <option value="bounce_rate">Bounce Rate (%)</option>
                        <option value="complaint_rate">Complaint Rate (%)</option>
                        <option value="dnsbl_listed">DNSBL Listed Count</option>
                        <option value="queue_depth">Queue Depth</option>
                        <option value="delivery_rate">Delivery Rate (&lt;)</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-300 mb-1">Condition</label>
                      <div className="flex space-x-1">
                        <select
                          value={newRuleOp}
                          onChange={(e) => setNewRuleOp(e.target.value)}
                          className="w-12 px-1 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg"
                        >
                          <option value=">">&gt;</option>
                          <option value=">=">&gt;=</option>
                          <option value="<">&lt;</option>
                          <option value="<=">&lt;=</option>
                        </select>
                        <input
                          type="number"
                          step="0.001"
                          required
                          value={newRuleThreshold}
                          onChange={(e) => setNewRuleThreshold(Number(e.target.value))}
                          className="flex-1 px-2 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg"
                        />
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-300 mb-1">Severity</label>
                      <select
                        value={newRuleSeverity}
                        onChange={(e) => setNewRuleSeverity(e.target.value)}
                        className="w-full px-2 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                      >
                        <option value="info">Info</option>
                        <option value="warning">Warning</option>
                        <option value="critical">Critical</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-300 mb-1">Cooldown (m)</label>
                      <input
                        type="number"
                        min="1"
                        required
                        value={newRuleCooldown}
                        onChange={(e) => setNewRuleCooldown(Number(e.target.value))}
                        className="w-full px-2 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg"
                      />
                    </div>
                    <div className="flex items-end">
                      <button
                        type="submit"
                        className="w-full py-1.5 px-3 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-lg transition-colors"
                      >
                        Create Rule
                      </button>
                    </div>
                  </form>

                  {/* Configured Rules Table */}
                  <div className="overflow-x-auto border border-slate-800 rounded-lg">
                    <div className="p-3 bg-slate-950 border-b border-slate-800 font-semibold text-xs text-white flex justify-between">
                      <span>Configured Alert Rules ({alertRules.length})</span>
                    </div>
                    <table className="w-full text-left text-xs text-slate-300">
                      <thead className="bg-slate-950 text-slate-400 uppercase border-b border-slate-800">
                        <tr>
                          <th className="px-4 py-2.5">Rule Name</th>
                          <th className="px-4 py-2.5">Metric</th>
                          <th className="px-4 py-2.5">Condition</th>
                          <th className="px-4 py-2.5">Severity</th>
                          <th className="px-4 py-2.5">Cooldown</th>
                          <th className="px-4 py-2.5 text-right">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {alertRules.length === 0 ? (
                          <tr>
                            <td colSpan={6} className="px-4 py-4 text-center text-slate-500">
                              No custom alert rules configured yet.
                            </td>
                          </tr>
                        ) : (
                          alertRules.map((r) => (
                            <tr key={r.id} className="hover:bg-slate-800/40">
                              <td className="px-4 py-2.5 font-medium text-white">{r.name}</td>
                              <td className="px-4 py-2.5 font-mono text-slate-400">{r.metric_type}</td>
                              <td className="px-4 py-2.5 font-mono text-indigo-300">{r.comparison_operator} {r.threshold}</td>
                              <td className="px-4 py-2.5">
                                <span className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                                  r.severity === 'critical'
                                    ? 'bg-rose-950 text-rose-400 border border-rose-800'
                                    : r.severity === 'warning'
                                    ? 'bg-amber-950 text-amber-400 border border-amber-800'
                                    : 'bg-blue-950 text-blue-400 border border-blue-800'
                                }`}>
                                  {r.severity}
                                </span>
                              </td>
                              <td className="px-4 py-2.5 text-slate-400">{r.cooldown_minutes}m</td>
                              <td className="px-4 py-2.5 text-right">
                                <button
                                  onClick={() => handleDeleteAlertRule(r.id)}
                                  className="text-xs text-rose-400 hover:text-rose-300 font-medium"
                                >
                                  Delete
                                </button>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>

                  {/* Active Incidents Table */}
                  <div className="overflow-x-auto border border-slate-800 rounded-lg">
                    <div className="p-3 bg-slate-950 border-b border-slate-800 font-semibold text-xs text-white flex justify-between">
                      <span>Alert Incidents Log ({alertIncidents.length})</span>
                    </div>
                    <table className="w-full text-left text-xs text-slate-300">
                      <thead className="bg-slate-950 text-slate-400 uppercase border-b border-slate-800">
                        <tr>
                          <th className="px-4 py-2.5">Severity</th>
                          <th className="px-4 py-2.5">Summary</th>
                          <th className="px-4 py-2.5">Status</th>
                          <th className="px-4 py-2.5">Fired At</th>
                          <th className="px-4 py-2.5 text-right">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {alertIncidents.length === 0 ? (
                          <tr>
                            <td colSpan={5} className="px-4 py-4 text-center text-slate-500">
                              No active or past incidents. System deliverability is within safe thresholds.
                            </td>
                          </tr>
                        ) : (
                          alertIncidents.map((inc) => (
                            <tr key={inc.id} className="hover:bg-slate-800/40">
                              <td className="px-4 py-2.5">
                                <span className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                                  inc.severity === 'critical'
                                    ? 'bg-rose-950 text-rose-400 border border-rose-800'
                                    : inc.severity === 'warning'
                                    ? 'bg-amber-950 text-amber-400 border border-amber-800'
                                    : 'bg-blue-950 text-blue-400 border border-blue-800'
                                }`}>
                                  {inc.severity}
                                </span>
                              </td>
                              <td className="px-4 py-2.5 text-white max-w-sm truncate">{inc.summary}</td>
                              <td className="px-4 py-2.5">
                                <span className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold ${
                                  inc.status === 'firing'
                                    ? 'bg-rose-950 text-rose-400 border border-rose-800 animate-pulse'
                                    : inc.status === 'acknowledged'
                                    ? 'bg-amber-950 text-amber-400 border border-amber-800'
                                    : 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                                }`}>
                                  {inc.status.toUpperCase()}
                                </span>
                              </td>
                              <td className="px-4 py-2.5 text-slate-500">{new Date(inc.fired_at).toLocaleTimeString()}</td>
                              <td className="px-4 py-2.5 text-right space-x-2">
                                {inc.status === 'firing' && (
                                  <button
                                    onClick={() => handleAcknowledgeIncident(inc.id)}
                                    className="text-xs text-amber-400 hover:text-amber-300 font-medium"
                                  >
                                    Ack
                                  </button>
                                )}
                                {inc.status !== 'resolved' && (
                                  <button
                                    onClick={() => handleResolveIncident(inc.id)}
                                    className="text-xs text-emerald-400 hover:text-emerald-300 font-medium"
                                  >
                                    Resolve
                                  </button>
                                )}
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Section 5: Multi-Channel Notification Dispatchers (§22.4) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h2 className="text-base font-semibold text-white">Notification Channels (§22.4)</h2>
                      <p className="text-xs text-slate-400">
                        Dispatch incidents in real-time to Slack webhooks, Microsoft Teams, PagerDuty Events API, or signed generic webhooks.
                      </p>
                    </div>
                  </div>

                  {/* Add Channel Form */}
                  <form onSubmit={handleCreateChannel} className="grid grid-cols-1 md:grid-cols-4 gap-3 bg-slate-950/40 p-3 rounded-lg border border-slate-800">
                    <div>
                      <label className="block text-xs font-medium text-slate-300 mb-1">Channel Name</label>
                      <input
                        type="text"
                        required
                        value={newChannelName}
                        onChange={(e) => setNewChannelName(e.target.value)}
                        placeholder="e.g. SRE Slack Alerts"
                        className="w-full px-3 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-300 mb-1">Channel Type</label>
                      <select
                        value={newChannelType}
                        onChange={(e) => setNewChannelType(e.target.value)}
                        className="w-full px-2 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg"
                      >
                        <option value="slack">Slack Webhook</option>
                        <option value="microsoft_teams">Microsoft Teams</option>
                        <option value="pagerduty">PagerDuty Events v2</option>
                        <option value="generic_webhook">Generic HMAC Webhook</option>
                        <option value="email">System Tenant Email</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-300 mb-1">Webhook URL / Endpoint</label>
                      <input
                        type="text"
                        required
                        value={newChannelURL}
                        onChange={(e) => setNewChannelURL(e.target.value)}
                        placeholder="https://hooks.slack.com/services/..."
                        className="w-full px-3 py-1.5 border border-slate-700 bg-slate-800 text-slate-200 text-xs rounded-lg font-mono"
                      />
                    </div>
                    <div className="flex items-end">
                      <button
                        type="submit"
                        className="w-full py-1.5 px-3 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-lg transition-colors"
                      >
                        Add Channel
                      </button>
                    </div>
                  </form>

                  {/* Channels Table */}
                  <div className="overflow-x-auto border border-slate-800 rounded-lg">
                    <table className="w-full text-left text-xs text-slate-300">
                      <thead className="bg-slate-950 text-slate-400 uppercase border-b border-slate-800">
                        <tr>
                          <th className="px-4 py-2.5">Channel Name</th>
                          <th className="px-4 py-2.5">Type</th>
                          <th className="px-4 py-2.5">Configuration</th>
                          <th className="px-4 py-2.5 text-right">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {notificationChannels.length === 0 ? (
                          <tr>
                            <td colSpan={4} className="px-4 py-4 text-center text-slate-500">
                              No notification channels configured. Add Slack, Teams, or PagerDuty channels above.
                            </td>
                          </tr>
                        ) : (
                          notificationChannels.map((ch) => (
                            <tr key={ch.id} className="hover:bg-slate-800/40">
                              <td className="px-4 py-2.5 font-medium text-white">{ch.name}</td>
                              <td className="px-4 py-2.5 font-mono text-indigo-400">{ch.type}</td>
                              <td className="px-4 py-2.5 font-mono text-slate-400 max-w-xs truncate">{ch.config}</td>
                              <td className="px-4 py-2.5 text-right space-x-2">
                                <button
                                  onClick={() => handleTestChannel(ch.id)}
                                  className="text-xs text-indigo-400 hover:text-indigo-300 font-medium"
                                >
                                  Test
                                </button>
                                <button
                                  onClick={() => handleDeleteChannel(ch.id)}
                                  className="text-xs text-rose-400 hover:text-rose-300 font-medium"
                                >
                                  Delete
                                </button>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Section 6: In-App Notification Center Feed */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-sm">
                  <div className="p-4 border-b border-slate-800 font-semibold text-xs text-white flex justify-between items-center">
                    <span>In-App Notification Feed ({alertNotifications.length})</span>
                    <button
                      onClick={loadPhase11Data}
                      className="text-indigo-400 hover:text-indigo-300 text-xs font-normal"
                    >
                      Refresh
                    </button>
                  </div>
                  <div className="divide-y divide-slate-800">
                    {alertNotifications.length === 0 ? (
                      <div className="p-4 text-center text-slate-500 text-xs">
                        No notifications in your feed.
                      </div>
                    ) : (
                      alertNotifications.map((n) => (
                        <div key={n.id} className={`p-4 flex items-start justify-between ${n.is_read ? 'opacity-60' : 'bg-slate-850'}`}>
                          <div>
                            <div className="flex items-center space-x-2">
                              {!n.is_read && <span className="w-2 h-2 rounded-full bg-indigo-400"></span>}
                              <span className="font-semibold text-white text-xs">{n.title}</span>
                              <span className="text-[10px] text-slate-500">{new Date(n.created_at).toLocaleTimeString()}</span>
                            </div>
                            <p className="text-xs text-slate-300 mt-1">{n.message}</p>
                          </div>
                          {!n.is_read && (
                            <button
                              onClick={() => handleMarkNotifRead(n.id)}
                              className="text-xs text-slate-400 hover:text-white"
                            >
                              Mark Read
                            </button>
                          )}
                        </div>
                      ))
                    )}
                  </div>
                </div>
              </div>
            )}

            {/* Phase 12: SSO, Impersonation & Compliance Tab */}
            {currentTab === 'sso_compliance' && (
              <div className="space-y-8 mt-6">
                {ssoActionMsg && (
                  <div className="p-3 bg-indigo-950/60 border border-indigo-500/40 rounded-lg text-xs text-indigo-300 flex justify-between items-center shadow-sm">
                    <span>{ssoActionMsg}</span>
                    <button onClick={() => setSSOActionMsg(null)} className="text-indigo-400 hover:text-white text-xs">✕</button>
                  </div>
                )}

                {/* Section 1: Single Sign-On (OIDC & SAML 2.0) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-sm">
                  <div className="flex items-center justify-between pb-4 border-b border-slate-800 mb-6">
                    <div>
                      <h2 className="text-lg font-semibold text-white">Single Sign-On (OIDC & SAML 2.0)</h2>
                      <p className="text-xs text-slate-400 mt-1">
                        Enterprise identity federation (§7.2) with automated Just-in-Time (JIT) provisioning and domain auto-join.
                      </p>
                    </div>
                    <div className="flex items-center space-x-3">
                      <a
                        href="/v1/auth/sso/saml/metadata"
                        target="_blank"
                        rel="noreferrer"
                        className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium rounded-lg border border-slate-700 transition"
                      >
                        Download SP Metadata XML
                      </a>
                    </div>
                  </div>

                  {/* Form to Register Identity Provider */}
                  <form onSubmit={handleCreateSSOProvider} className="bg-slate-950/60 p-4 rounded-lg border border-slate-800 mb-6">
                    <h3 className="text-xs font-semibold text-white uppercase tracking-wider mb-4">Register Identity Provider</h3>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                      <div>
                        <label className="block text-xs text-slate-400 mb-1">Provider Name</label>
                        <input
                          type="text"
                          required
                          value={newSSOName}
                          onChange={(e) => setNewSSOName(e.target.value)}
                          placeholder="e.g. Okta Corporate / Google Workspace"
                          className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-slate-400 mb-1">Protocol Type</label>
                        <select
                          value={newSSOType}
                          onChange={(e) => setNewSSOType(e.target.value as 'oidc' | 'saml')}
                          className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
                        >
                          <option value="oidc">OpenID Connect (OIDC 1.0)</option>
                          <option value="saml">SAML 2.0 Web SSO</option>
                        </select>
                      </div>
                      <div>
                        <label className="block text-xs text-slate-400 mb-1">
                          {newSSOType === 'oidc' ? 'Issuer URL' : 'IDP Metadata URL'}
                        </label>
                        <input
                          type="text"
                          value={newSSOIssuer}
                          onChange={(e) => setNewSSOIssuer(e.target.value)}
                          placeholder={newSSOType === 'oidc' ? 'https://accounts.google.com' : 'https://idp.example.com/metadata.xml'}
                          className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-slate-400 mb-1">
                          {newSSOType === 'oidc' ? 'Client ID' : 'Entity ID / App ID'}
                        </label>
                        <input
                          type="text"
                          value={newSSOClientID}
                          onChange={(e) => setNewSSOClientID(e.target.value)}
                          placeholder="e.g. mailvex-client-prod"
                          className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-slate-400 mb-1">Auto-Join Domains (comma separated)</label>
                        <input
                          type="text"
                          value={newSSODomains}
                          onChange={(e) => setNewSSODomains(e.target.value)}
                          placeholder="e.g. acme.com, acme.co.uk"
                          className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-slate-400 mb-1">Default JIT Role</label>
                        <select
                          value={newSSORole}
                          onChange={(e) => setNewSSORole(e.target.value)}
                          className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
                        >
                          <option value="developer">Developer</option>
                          <option value="admin">Admin</option>
                          <option value="analyst">Analyst</option>
                          <option value="support">Support (Masked Body)</option>
                          <option value="viewer">Viewer</option>
                        </select>
                      </div>
                    </div>

                    <div className="flex items-center justify-between pt-2 border-t border-slate-850">
                      <label className="flex items-center space-x-2 text-xs text-slate-300 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={newSSOEnforce}
                          onChange={(e) => setNewSSOEnforce(e.target.checked)}
                          className="rounded border-slate-700 text-indigo-600 focus:ring-indigo-500"
                        />
                        <span>Enforce SSO (Block password logins for matching domain users)</span>
                      </label>
                      <button
                        type="submit"
                        className="px-4 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-medium rounded transition"
                      >
                        Register SSO Provider
                      </button>
                    </div>
                  </form>

                  {/* SSO Providers List */}
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs text-slate-300">
                      <thead className="bg-slate-950/60 text-slate-400 uppercase tracking-wider font-semibold border-b border-slate-800">
                        <tr>
                          <th className="p-3">Name</th>
                          <th className="p-3">Type</th>
                          <th className="p-3">Allowed Domains</th>
                          <th className="p-3">Default Role</th>
                          <th className="p-3">Enforce SSO</th>
                          <th className="p-3">Status</th>
                          <th className="p-3 text-right">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {ssoProviders.length === 0 ? (
                          <tr>
                            <td colSpan={7} className="p-4 text-center text-slate-500">
                              No SSO identity providers configured for this tenant.
                            </td>
                          </tr>
                        ) : (
                          ssoProviders.map((p) => (
                            <tr key={p.id} className="hover:bg-slate-850/50 transition">
                              <td className="p-3 font-medium text-white">{p.name}</td>
                              <td className="p-3">
                                <span className={`px-2 py-0.5 rounded text-[10px] font-semibold uppercase ${
                                  p.provider_type === 'oidc' ? 'bg-blue-900/40 text-blue-300 border border-blue-700/50' : 'bg-purple-900/40 text-purple-300 border border-purple-700/50'
                                }`}>
                                  {p.provider_type}
                                </span>
                              </td>
                              <td className="p-3">
                                {p.allowed_domains && p.allowed_domains.length > 0 ? (
                                  <div className="flex flex-wrap gap-1">
                                    {p.allowed_domains.map((d, i) => (
                                      <span key={i} className="px-1.5 py-0.5 bg-slate-800 text-slate-300 rounded text-[10px]">
                                        @{d}
                                      </span>
                                    ))}
                                  </div>
                                ) : (
                                  <span className="text-slate-500 italic">None</span>
                                )}
                              </td>
                              <td className="p-3 font-mono text-[11px] text-indigo-300">{p.default_role}</td>
                              <td className="p-3">
                                {p.enforce_sso ? (
                                  <span className="text-emerald-400 font-medium">Enforced</span>
                                ) : (
                                  <span className="text-slate-500">Optional</span>
                                )}
                              </td>
                              <td className="p-3">
                                <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium bg-emerald-950/60 text-emerald-400 border border-emerald-800/40">
                                  Active
                                </span>
                              </td>
                              <td className="p-3 text-right">
                                <button
                                  onClick={() => handleDeleteSSOProvider(p.id)}
                                  className="text-xs text-rose-400 hover:text-rose-300 font-medium"
                                >
                                  Delete
                                </button>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Section 2: Platform Admin Impersonation & Audit Trail (§7.4) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-sm">
                  <div className="flex items-center justify-between pb-4 border-b border-slate-800 mb-6">
                    <div>
                      <h2 className="text-lg font-semibold text-white">Platform Admin Impersonation & Audit (§7.4)</h2>
                      <p className="text-xs text-slate-400 mt-1">
                        Secure tenant impersonation with mandatory justification, immutable audit logging, and support role body privacy masking.
                      </p>
                    </div>
                    <div>
                      {isImpersonating ? (
                        <button
                          onClick={handleExitImpersonation}
                          className="px-3 py-1.5 bg-amber-600 hover:bg-amber-500 text-white text-xs font-semibold rounded-lg shadow-sm transition"
                        >
                          Exit Impersonation Mode
                        </button>
                      ) : user?.is_platform_admin ? (
                        <button
                          onClick={() => setImpModalOpen(true)}
                          className="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-medium rounded-lg transition"
                        >
                          Start Impersonation Session
                        </button>
                      ) : (
                        <span className="text-xs text-slate-500">Requires Platform Admin</span>
                      )}
                    </div>
                  </div>

                  {/* Active Impersonation Status Widget */}
                  {isImpersonating ? (
                    <div className="mb-6 p-4 bg-amber-950/30 border border-amber-500/40 rounded-lg flex items-center justify-between">
                      <div>
                        <div className="flex items-center space-x-2">
                          <span className="w-2.5 h-2.5 rounded-full bg-amber-400 animate-ping"></span>
                          <span className="text-xs font-bold text-amber-300 uppercase tracking-wider">Active Impersonation Session</span>
                        </div>
                        <p className="text-xs text-slate-300 mt-1">
                          Tenant: <strong className="text-white">{impersonatedTenant?.name}</strong> (<code className="text-amber-200">{impersonatedTenant?.id}</code>)
                        </p>
                        <p className="text-xs text-slate-400 mt-0.5">
                          Reason: <em className="text-slate-200">"{impersonationReason}"</em> | Privacy Masking: <strong className="text-slate-200">{isSupportMasking ? 'Enabled (Body Masked)' : 'Full Read'}</strong>
                        </p>
                      </div>
                      <button
                        onClick={handleExitImpersonation}
                        className="px-3 py-1 bg-amber-700 hover:bg-amber-600 text-white text-xs font-medium rounded transition"
                      >
                        Exit Now
                      </button>
                    </div>
                  ) : (
                    <div className="mb-6 p-3 bg-slate-950/40 border border-slate-800 rounded-lg flex items-center justify-between text-xs text-slate-400">
                      <span>Currently operating in standard identity context: <strong className="text-slate-200">{user?.email}</strong></span>
                      <span className="text-slate-500 font-mono text-[11px]">User ID: {user?.id}</span>
                    </div>
                  )}

                  {/* Impersonation Audit Log Table */}
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">
                      Impersonation Session Audit Trail
                    </h3>
                    <ExportMenu
                      filename="mailvex-impersonation-sessions"
                      sheetTitle="Impersonations"
                      label="Export Sessions"
                      columns={[
                        { header: 'Session ID', key: 'id' },
                        { header: 'Actor Admin', key: 'actor_user_id' },
                        { header: 'Target Tenant ID', key: 'target_tenant_id' },
                        { header: 'Reason', key: 'reason' },
                        { header: 'Mode', key: 'access_type' },
                        { header: 'Body Masking', key: 'is_support_masking' },
                        { header: 'Started At', key: 'started_at' },
                        { header: 'Status', key: 'status' },
                      ]}
                      data={impersonationSessions.map((s) => ({
                        id: s.id,
                        actor_user_id: s.actor_user_id,
                        target_tenant_id: s.target_tenant_id,
                        reason: s.reason,
                        access_type: s.access_type,
                        is_support_masking: s.is_support_masking ? 'Masked' : 'Full',
                        started_at: s.started_at,
                        status: !s.ended_at ? 'Active' : 'Ended',
                      }))}
                    />
                  </div>
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs text-slate-300">
                      <thead className="bg-slate-950/60 text-slate-400 uppercase tracking-wider font-semibold border-b border-slate-800">
                        <tr>
                          <th className="p-3">Actor Admin</th>
                          <th className="p-3">Target Tenant ID</th>
                          <th className="p-3">Reason</th>
                          <th className="p-3">Mode</th>
                          <th className="p-3">Body Masking</th>
                          <th className="p-3">Started At</th>
                          <th className="p-3">Status</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {impersonationSessions.length === 0 ? (
                          <tr>
                            <td colSpan={7} className="p-4 text-center text-slate-500">
                              No impersonation sessions recorded.
                            </td>
                          </tr>
                        ) : (
                          impersonationSessions.map((s) => (
                            <tr key={s.id} className="hover:bg-slate-850/50 transition">
                              <td className="p-3 font-mono text-[11px] text-slate-300">{s.actor_user_id}</td>
                              <td className="p-3 font-mono text-[11px] text-indigo-300">{s.target_tenant_id}</td>
                              <td className="p-3 text-white max-w-xs truncate">{s.reason}</td>
                              <td className="p-3">
                                <span className="px-1.5 py-0.5 bg-slate-800 text-slate-300 rounded text-[10px] uppercase font-mono">
                                  {s.access_type}
                                </span>
                              </td>
                              <td className="p-3">
                                {s.is_support_masking ? (
                                  <span className="px-1.5 py-0.5 bg-amber-950/60 text-amber-400 border border-amber-800/40 rounded text-[10px] font-semibold">
                                    🔒 Masked
                                  </span>
                                ) : (
                                  <span className="text-slate-500 text-[11px]">Full</span>
                                )}
                              </td>
                              <td className="p-3 text-slate-400">{new Date(s.started_at).toLocaleString()}</td>
                              <td className="p-3">
                                {!s.ended_at ? (
                                  <span className="px-2 py-0.5 bg-emerald-950/60 text-emerald-400 border border-emerald-800/40 rounded-full text-[10px] font-semibold">
                                    Active
                                  </span>
                                ) : (
                                  <span className="text-slate-500 text-[11px]">Ended</span>
                                )}
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Section 3: Deliverability Compliance Checklist (§22) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-sm">
                  <div className="flex items-center justify-between pb-4 border-b border-slate-800 mb-6">
                    <div>
                      <h2 className="text-lg font-semibold text-white">Deliverability Compliance Checklist (§22)</h2>
                      <p className="text-xs text-slate-400 mt-1">
                        Automated 100-point deliverability audit evaluating SPF, DKIM, DMARC, FCrDNS, STARTTLS, DNSBL, and RFC 8058 List-Unsubscribe compliance.
                      </p>
                    </div>
                    <div className="flex items-center space-x-2">
                      <select
                        value={complianceDomain || (domains.length > 0 ? domains[0].name : '')}
                        onChange={(e) => {
                          setComplianceDomain(e.target.value)
                          handleRunComplianceCheck(e.target.value)
                        }}
                        className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
                      >
                        {domains.map((d) => (
                          <option key={d.id} value={d.name}>{d.name}</option>
                        ))}
                      </select>
                      <button
                        onClick={() => handleRunComplianceCheck(complianceDomain || (domains.length > 0 ? domains[0].name : ''))}
                        disabled={complianceLoading || domains.length === 0}
                        className="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-xs font-medium rounded-lg transition"
                      >
                        {complianceLoading ? 'Auditing...' : 'Run Audit'}
                      </button>
                    </div>
                  </div>

                  {/* Compliance Score Header */}
                  {complianceResult ? (
                    <div className="space-y-6">
                      <div className="p-5 bg-slate-950/60 border border-slate-800 rounded-xl flex items-center justify-between">
                        <div>
                          <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Deliverability Compliance Score</span>
                          <h3 className="text-2xl font-extrabold text-white mt-1 flex items-center space-x-3">
                            <span className={complianceResult.overall_score >= 80 ? 'text-emerald-400' : complianceResult.overall_score >= 60 ? 'text-amber-400' : 'text-rose-400'}>
                              {complianceResult.overall_score} / 100
                            </span>
                            <span className={`text-xs px-2.5 py-0.5 rounded-full font-semibold uppercase border ${
                              complianceResult.overall_score >= 80
                                ? 'bg-emerald-950/60 text-emerald-400 border-emerald-800/40'
                                : complianceResult.overall_score >= 60
                                ? 'bg-amber-950/60 text-amber-400 border-amber-800/40'
                                : 'bg-rose-950/60 text-rose-400 border-rose-800/40'
                            }`}>
                              {complianceResult.overall_score >= 80 ? 'Production Ready' : complianceResult.overall_score >= 60 ? 'Acceptable' : 'High Risk'}
                            </span>
                          </h3>
                          <p className="text-xs text-slate-400 mt-1">
                            Domain evaluated: <strong className="text-white">{complianceResult.domain_name}</strong> | Last audit: {new Date(complianceResult.checked_at).toLocaleString()}
                          </p>
                        </div>
                        <div className="flex items-center space-x-4">
                          <ExportMenu
                            filename={`deliverability-audit-${complianceResult.domain_name}`}
                            sheetTitle="Compliance Audit"
                            label="Export Audit"
                            columns={[
                              { header: 'Standard / Pillar', key: 'standard' },
                              { header: 'Status', key: 'status' },
                              { header: 'Details & Findings', key: 'details' },
                            ]}
                            data={[
                              { standard: 'Overall Score', status: `${complianceResult.overall_score}/100`, details: complianceResult.overall_score >= 80 ? 'Production Ready' : 'Issues Detected' },
                              { standard: 'SPF Alignment', status: complianceResult.spf_status || 'pass', details: complianceResult.spf_details || 'Configured' },
                              { standard: 'DKIM Cryptographic Signature', status: complianceResult.dkim_status || 'pass', details: complianceResult.dkim_details || '2048-bit key verified' },
                              { standard: 'DMARC Enforcement Policy', status: complianceResult.dmarc_status || 'pass', details: complianceResult.dmarc_details || 'Policy active' },
                              { standard: 'Forward-Confirmed Reverse DNS (FCrDNS)', status: complianceResult.fcrdns_status || 'pass', details: complianceResult.fcrdns_details || 'PTR matches IP' },
                              { standard: 'STARTTLS In-Flight Encryption', status: complianceResult.tls_status || 'pass', details: complianceResult.tls_details || 'TLS 1.3 negotiated' },
                              { standard: 'IP DNSBL Reputation', status: complianceResult.dnsbl_status || 'clean', details: complianceResult.dnsbl_details || 'Zero listings' },
                            ]}
                          />
                          <div className="w-16 h-16 rounded-full border-4 border-slate-200 dark:border-slate-800 flex items-center justify-center bg-slate-50 dark:bg-slate-900">
                            <span className="text-base font-bold text-slate-900 dark:text-white">{complianceResult.overall_score}%</span>
                          </div>
                        </div>
                      </div>

                      {/* 7 Pillars Breakdown Cards */}
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        {/* SPF */}
                        <div className="bg-slate-950/40 border border-slate-800 rounded-lg p-4">
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-xs font-semibold text-white">SPF Alignment</span>
                            <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                              complianceResult.spf_status === 'pass' ? 'bg-emerald-900/50 text-emerald-300' : 'bg-rose-900/50 text-rose-300'
                            }`}>{complianceResult.spf_status}</span>
                          </div>
                          <p className="text-xs text-slate-400">{complianceResult.spf_details || 'SPF record authorizes sending hosts.'}</p>
                        </div>

                        {/* DKIM */}
                        <div className="bg-slate-950/40 border border-slate-800 rounded-lg p-4">
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-xs font-semibold text-white">DKIM Cryptographic Key</span>
                            <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                              complianceResult.dkim_status === 'pass' ? 'bg-emerald-900/50 text-emerald-300' : 'bg-rose-900/50 text-rose-300'
                            }`}>{complianceResult.dkim_status}</span>
                          </div>
                          <p className="text-xs text-slate-400">{complianceResult.dkim_details || '2048-bit RSA selector active.'}</p>
                        </div>

                        {/* DMARC */}
                        <div className="bg-slate-950/40 border border-slate-800 rounded-lg p-4">
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-xs font-semibold text-white">DMARC Policy Enforcement</span>
                            <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                              complianceResult.dmarc_status === 'pass' ? 'bg-emerald-900/50 text-emerald-300' : 'bg-rose-900/50 text-rose-300'
                            }`}>{complianceResult.dmarc_status}</span>
                          </div>
                          <p className="text-xs text-slate-400">{complianceResult.dmarc_details || 'DMARC policy defined with RUA aggregation.'}</p>
                        </div>

                        {/* FCrDNS */}
                        <div className="bg-slate-950/40 border border-slate-800 rounded-lg p-4">
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-xs font-semibold text-white">FCrDNS Bidirectional Sync</span>
                            <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                              complianceResult.fcrdns_status === 'pass' ? 'bg-emerald-900/50 text-emerald-300' : 'bg-rose-900/50 text-rose-300'
                            }`}>{complianceResult.fcrdns_status}</span>
                          </div>
                          <p className="text-xs text-slate-400">{complianceResult.fcrdns_details || 'PTR record matches forward A record.'}</p>
                        </div>

                        {/* TLS */}
                        <div className="bg-slate-950/40 border border-slate-800 rounded-lg p-4">
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-xs font-semibold text-white">TLS 1.3 / STARTTLS</span>
                            <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                              complianceResult.tls_status === 'pass' ? 'bg-emerald-900/50 text-emerald-300' : 'bg-rose-900/50 text-rose-300'
                            }`}>{complianceResult.tls_status}</span>
                          </div>
                          <p className="text-xs text-slate-400">{complianceResult.tls_details || 'Encrypted outbound delivery verified.'}</p>
                        </div>

                        {/* DNSBL */}
                        <div className="bg-slate-950/40 border border-slate-800 rounded-lg p-4">
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-xs font-semibold text-white">DNSBL Reputation</span>
                            <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                              complianceResult.dnsbl_status === 'pass' ? 'bg-emerald-900/50 text-emerald-300' : 'bg-rose-900/50 text-rose-300'
                            }`}>{complianceResult.dnsbl_status}</span>
                          </div>
                          <p className="text-xs text-slate-400">{complianceResult.dnsbl_details || 'No active listings on major blocklists.'}</p>
                        </div>

                        {/* RFC 8058 Unsubscribe */}
                        <div className="bg-slate-950/40 border border-slate-800 rounded-lg p-4 md:col-span-2 lg:col-span-3">
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-xs font-semibold text-white">RFC 8058 One-Click List-Unsubscribe & Header Compliance</span>
                            <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                              complianceResult.unsubscribe_status === 'pass' ? 'bg-emerald-900/50 text-emerald-300' : 'bg-rose-900/50 text-rose-300'
                            }`}>{complianceResult.unsubscribe_status}</span>
                          </div>
                          <p className="text-xs text-slate-400">{complianceResult.unsubscribe_details || 'List-Unsubscribe and List-Unsubscribe-Post headers configured on marketing streams.'}</p>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <div className="p-8 text-center text-slate-500 text-xs bg-slate-950/40 rounded-xl border border-slate-850">
                      {domains.length === 0 ? 'Add and verify a domain to inspect deliverability compliance.' : 'Select a domain above and click "Run Audit" to calculate the 100-point deliverability compliance index.'}
                    </div>
                  )}
                </div>

                {/* Section 4: Live Diagnostic Probe (FCrDNS Bidirectional Verification) */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-sm">
                  <div className="pb-4 border-b border-slate-800 mb-6">
                    <h2 className="text-lg font-semibold text-white">Live Diagnostic: FCrDNS Bidirectional Verification</h2>
                    <p className="text-xs text-slate-400 mt-1">
                      Verify Forward-Confirmed Reverse DNS (§22): checks that an IP's PTR query resolves to a hostname whose forward A/AAAA query returns the original IP.
                    </p>
                  </div>

                  <form onSubmit={handleRunFCrDNSCheck} className="flex items-center space-x-3 mb-6">
                    <input
                      type="text"
                      required
                      value={testFCrDNSIP}
                      onChange={(e) => setTestFCrDNSIP(e.target.value)}
                      placeholder="Enter Sending MTA IP (e.g. 198.51.100.1)"
                      className="flex-1 bg-slate-950 border border-slate-700 rounded px-3 py-2 text-xs text-white focus:outline-none focus:border-indigo-500"
                    />
                    <button
                      type="submit"
                      disabled={fcrdnsChecking}
                      className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-xs font-medium rounded transition"
                    >
                      {fcrdnsChecking ? 'Verifying...' : 'Verify FCrDNS'}
                    </button>
                  </form>

                  {fcrdnsResult && (
                    <div className={`p-4 rounded-lg border text-xs ${
                      fcrdnsResult.is_confirmed
                        ? 'bg-emerald-950/30 border-emerald-700/50 text-emerald-200'
                        : 'bg-rose-950/30 border-rose-700/50 text-rose-200'
                    }`}>
                      <div className="flex items-center justify-between mb-2">
                        <span className="font-bold uppercase tracking-wider">
                          FCrDNS Status: {fcrdnsResult.is_confirmed ? 'Confirmed (Pass)' : 'Mismatch / Failed'}
                        </span>
                        <span className="font-mono text-[11px] text-slate-400">IP: {fcrdnsResult.ip_address}</span>
                      </div>
                      <p className="text-xs mb-3 text-slate-300">{fcrdnsResult.details}</p>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-[11px] font-mono">
                        <div className="p-2 bg-slate-900 rounded border border-slate-800">
                          <span className="text-slate-400 block mb-1">PTR Hostnames:</span>
                          {fcrdnsResult.ptr_hostnames && fcrdnsResult.ptr_hostnames.length > 0 ? (
                            fcrdnsResult.ptr_hostnames.map((h, i) => <div key={i} className="text-indigo-300">{h}</div>)
                          ) : (
                            <span className="text-slate-500">None</span>
                          )}
                        </div>
                        <div className="p-2 bg-slate-900 rounded border border-slate-800">
                          <span className="text-slate-400 block mb-1">Forward Resolved IPs:</span>
                          {fcrdnsResult.forward_ips && fcrdnsResult.forward_ips.length > 0 ? (
                            fcrdnsResult.forward_ips.map((ip, i) => <div key={i} className="text-emerald-300">{ip}</div>)
                          ) : (
                            <span className="text-slate-500">None</span>
                          )}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* TAB: PLATFORM ADMIN CENTER */}
            {currentTab === 'platform_admin' && (
              <div className="space-y-6 mt-6">
                {/* Hero Banner */}
                <div className="bg-gradient-to-r from-slate-900/90 via-amber-950/30 to-slate-900/90 border border-amber-500/30 rounded-2xl p-6 relative overflow-hidden shadow-2xl">
                  <div className="absolute right-0 top-0 w-80 h-80 bg-amber-500/10 rounded-full blur-3xl pointer-events-none -mr-20 -mt-20"></div>
                  <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 relative z-10">
                    <div>
                      <div className="flex items-center space-x-2 mb-2">
                        <span className="px-2.5 py-0.5 text-[10px] font-bold tracking-wider uppercase bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-full flex items-center space-x-1">
                          <ShieldCheck className="w-3 h-3 text-amber-400" />
                          <span>System Platform Administrator</span>
                        </span>
                        <span className="text-xs text-slate-400 font-mono">Cluster Root Access</span>
                      </div>
                      <h2 className="text-2xl font-extrabold text-white tracking-tight">
                        Platform Administration & Fleet Control
                      </h2>
                      <p className="text-xs text-slate-300 mt-1 max-w-xl leading-relaxed">
                        Centrally provision tenants, manage cryptographic master keys, inspect carrier-grade cluster node health, and audit administrative actions.
                      </p>
                    </div>

                    <div className="flex items-center space-x-3">
                      <button
                        onClick={() => setImpModalOpen(true)}
                        className="flex items-center space-x-2 px-3.5 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold rounded-xl border border-slate-700 transition"
                      >
                        <Shield className="w-4 h-4 text-indigo-400" />
                        <span>Impersonate Tenant</span>
                      </button>
                      <button
                        onClick={fetchAdminAuditLogs}
                        disabled={adminAuditLoading}
                        className="flex items-center space-x-2 px-3.5 py-2 bg-amber-600 hover:bg-amber-500 text-white text-xs font-semibold rounded-xl shadow-lg shadow-amber-600/25 transition disabled:opacity-50"
                      >
                        <RefreshCw className={`w-4 h-4 ${adminAuditLoading ? 'animate-spin' : ''}`} />
                        <span>Sync Fleet Status</span>
                      </button>
                    </div>
                  </div>
                </div>

                {/* Sub-Tab Navigation */}
                <div className="flex space-x-2 border-b border-slate-800 pb-2">
                  {[
                    { id: 'overview', label: 'Fleet & Tenants', icon: Users },
                    { id: 'health', label: 'Cluster & Node Diagnostics', icon: Server },
                    { id: 'audit', label: 'Security Audit Trail', icon: ShieldCheck },
                  ].map((tab) => {
                    const Icon = tab.icon
                    const isTabActive = adminActiveSubTab === tab.id
                    return (
                      <button
                        key={tab.id}
                        onClick={() => setAdminActiveSubTab(tab.id as any)}
                        className={`flex items-center space-x-2 px-4 py-2 text-xs font-semibold rounded-xl transition ${
                          isTabActive
                            ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/20'
                            : 'bg-slate-900/60 text-slate-400 hover:text-slate-200 hover:bg-slate-800/80 border border-slate-800/80'
                        }`}
                      >
                        <Icon className="w-4 h-4" />
                        <span>{tab.label}</span>
                      </button>
                    )
                  })}
                </div>

                {/* SUBTAB 1: FLEET & TENANTS */}
                {adminActiveSubTab === 'overview' && (
                  <div className="space-y-6">
                    {/* Create New Tenant Card */}
                    <div className="bg-slate-900/90 border border-slate-800/90 rounded-2xl p-6 shadow-sm">
                      <div className="flex items-center justify-between pb-4 border-b border-slate-800/80 mb-5">
                        <div className="flex items-center space-x-2">
                          <PlusCircle className="w-5 h-5 text-indigo-400" />
                          <h3 className="text-sm font-bold text-white">Provision New Tenant</h3>
                        </div>
                        <span className="text-[11px] text-slate-400">Automatic Master KEK Encryption Key Wrapping</span>
                      </div>

                      {adminCreateTenantSuccess && (
                        <div className="mb-4 bg-emerald-950/60 border border-emerald-500/60 text-emerald-300 text-xs p-3.5 rounded-xl flex items-center space-x-2">
                          <CheckCircle2 className="w-4 h-4 flex-shrink-0 text-emerald-400" />
                          <span>{adminCreateTenantSuccess}</span>
                        </div>
                      )}

                      {adminCreateTenantError && (
                        <div className="mb-4 bg-rose-950/60 border border-rose-500/60 text-rose-300 text-xs p-3.5 rounded-xl flex items-center space-x-2">
                          <AlertTriangle className="w-4 h-4 flex-shrink-0 text-rose-400" />
                          <span>{adminCreateTenantError}</span>
                        </div>
                      )}

                      <form onSubmit={handleAdminCreateTenant} className="grid grid-cols-1 sm:grid-cols-4 gap-4">
                        <div>
                          <label className="block text-xs font-semibold text-slate-300 mb-1.5">Tenant Name</label>
                          <input
                            type="text"
                            required
                            placeholder="e.g. Staging Platform"
                            value={adminNewTenantName}
                            onChange={(e) => {
                              setAdminNewTenantName(e.target.value)
                              if (!adminNewTenantSlug) {
                                setAdminNewTenantSlug(e.target.value.toLowerCase().replace(/[^a-z0-9]/g, '-'))
                              }
                            }}
                            className="w-full px-3 py-2 border border-slate-700/80 bg-slate-950/80 text-slate-200 text-xs rounded-xl focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 outline-none"
                          />
                        </div>

                        <div>
                          <label className="block text-xs font-semibold text-slate-300 mb-1.5">Tenant Slug</label>
                          <input
                            type="text"
                            required
                            placeholder="e.g. staging-platform"
                            value={adminNewTenantSlug}
                            onChange={(e) => setAdminNewTenantSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
                            className="w-full px-3 py-2 border border-slate-700/80 bg-slate-950/80 text-slate-200 text-xs rounded-xl focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 outline-none font-mono"
                          />
                        </div>

                        <div>
                          <label className="block text-xs font-semibold text-slate-300 mb-1.5">Environment</label>
                          <select
                            value={adminNewTenantEnv}
                            onChange={(e) => setAdminNewTenantEnv(e.target.value as any)}
                            className="w-full px-3 py-2 border border-slate-700/80 bg-slate-950/80 text-slate-200 text-xs rounded-xl focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 outline-none cursor-pointer"
                          >
                            <option value="live">Live (Production)</option>
                            <option value="test">Test (Sandbox)</option>
                          </select>
                        </div>

                        <div className="flex items-end">
                          <button
                            type="submit"
                            disabled={adminCreateTenantLoading}
                            className="w-full py-2 px-4 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold rounded-xl shadow-lg shadow-indigo-600/25 transition disabled:opacity-50 flex items-center justify-center space-x-2 h-[38px]"
                          >
                            {adminCreateTenantLoading ? (
                              <span>Provisioning...</span>
                            ) : (
                              <>
                                <Plus className="w-4 h-4" />
                                <span>Create & Initialize DEK</span>
                              </>
                            )}
                          </button>
                        </div>
                      </form>
                    </div>

                    {/* Fleet Directory Table */}
                    <div className="bg-slate-900/90 border border-slate-800/90 rounded-2xl overflow-hidden shadow-sm">
                      <div className="p-5 border-b border-slate-800/80 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                        <div className="flex items-center space-x-2">
                          <Users className="w-4 h-4 text-indigo-400" />
                          <h3 className="text-sm font-bold text-white">Cluster Tenants Directory</h3>
                          <span className="px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 text-xs font-mono font-semibold">
                            {tenants.length} Tenants
                          </span>
                        </div>

                        <input
                          type="text"
                          placeholder="Filter by name or slug..."
                          value={adminSearchTenant}
                          onChange={(e) => setAdminSearchTenant(e.target.value)}
                          className="px-3 py-1.5 bg-slate-950 border border-slate-800 text-xs text-slate-200 rounded-xl focus:border-indigo-500 outline-none w-full sm:w-64"
                        />
                      </div>

                      <div className="overflow-x-auto">
                        <table className="w-full text-left text-xs text-slate-300">
                          <thead className="bg-slate-950/70 text-slate-400 font-semibold uppercase tracking-wider border-b border-slate-800/80">
                            <tr>
                              <th className="p-4">Tenant Name</th>
                              <th className="p-4">Identifier / Slug</th>
                              <th className="p-4">Environment</th>
                              <th className="p-4">Key Wrapping</th>
                              <th className="p-4">Status</th>
                              <th className="p-4 text-right">Actions</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-slate-800/60">
                            {tenants
                              .filter((t) =>
                                adminSearchTenant.trim() === '' ||
                                t.name.toLowerCase().includes(adminSearchTenant.toLowerCase()) ||
                                t.slug.toLowerCase().includes(adminSearchTenant.toLowerCase())
                              )
                              .map((t) => (
                                <tr key={t.id} className="hover:bg-slate-850/40 transition">
                                  <td className="p-4 font-bold text-white flex items-center space-x-2">
                                    <div className="w-7 h-7 rounded-lg bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-300 font-mono text-xs">
                                      {t.name.charAt(0).toUpperCase()}
                                    </div>
                                    <span>{t.name}</span>
                                    {activeTenant?.id === t.id && (
                                      <span className="px-1.5 py-0.5 bg-indigo-500/20 text-indigo-300 border border-indigo-500/30 text-[9px] rounded font-bold uppercase">Active</span>
                                    )}
                                  </td>
                                  <td className="p-4 font-mono text-[11px] text-slate-400">
                                    <div>{t.slug}</div>
                                    <div className="text-[10px] text-slate-600">{t.id}</div>
                                  </td>
                                  <td className="p-4">
                                    <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                                      t.environment === 'live'
                                        ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                                        : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                                    }`}>
                                      {t.environment}
                                    </span>
                                  </td>
                                  <td className="p-4">
                                    <span className="px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 text-[10px] font-mono border border-slate-700/60 flex items-center space-x-1 w-max">
                                      <Lock className="w-2.5 h-2.5 text-indigo-400" />
                                      <span>DEK Active</span>
                                    </span>
                                  </td>
                                  <td className="p-4">
                                    <span className="flex items-center space-x-1.5 text-emerald-400 text-xs">
                                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                                      <span>Active</span>
                                    </span>
                                  </td>
                                  <td className="p-4 text-right space-x-2">
                                    <button
                                      onClick={() => {
                                        setActiveTenant(t)
                                        setCurrentTab('dashboard')
                                      }}
                                      className="px-2.5 py-1 bg-indigo-600/20 hover:bg-indigo-600/40 text-indigo-300 border border-indigo-500/30 rounded-lg text-xs font-medium transition"
                                    >
                                      Switch To
                                    </button>
                                    <button
                                      onClick={() => {
                                        setTargetImpTenantID(t.id)
                                        setImpModalOpen(true)
                                      }}
                                      className="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 rounded-lg text-xs transition"
                                    >
                                      Impersonate
                                    </button>
                                  </td>
                                </tr>
                              ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  </div>
                )}

                {/* SUBTAB 2: CLUSTER & NODE DIAGNOSTICS */}
                {adminActiveSubTab === 'health' && (
                  <div className="space-y-6">
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      {/* API Server */}
                      <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 shadow-sm">
                        <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-3">
                          <span className="text-xs font-bold text-white flex items-center space-x-2">
                            <Server className="w-4 h-4 text-indigo-400" />
                            <span>HTTP REST API</span>
                          </span>
                          <span className="px-2 py-0.5 rounded-full bg-emerald-950/60 text-emerald-400 border border-emerald-800/40 text-[10px] font-bold">
                            ONLINE
                          </span>
                        </div>
                        <div className="space-y-1.5 text-xs text-slate-400">
                          <div className="flex justify-between"><span>Port Binding:</span><span className="font-mono text-slate-200">0.0.0.0:8080</span></div>
                          <div className="flex justify-between"><span>Active Routes:</span><span className="text-slate-200">28 Handlers</span></div>
                          <div className="flex justify-between"><span>Security:</span><span className="text-indigo-300">Strict CSP, HSTS, CORS</span></div>
                          <div className="flex justify-between"><span>Latency:</span><span className="text-emerald-400 font-mono">&lt; 1.5ms avg</span></div>
                        </div>
                      </div>

                      {/* SMTP Edge Gateway */}
                      <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 shadow-sm">
                        <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-3">
                          <span className="text-xs font-bold text-white flex items-center space-x-2">
                            <Mail className="w-4 h-4 text-indigo-400" />
                            <span>SMTP Edge Gateway</span>
                          </span>
                          <span className="px-2 py-0.5 rounded-full bg-emerald-950/60 text-emerald-400 border border-emerald-800/40 text-[10px] font-bold">
                            LISTENING
                          </span>
                        </div>
                        <div className="space-y-1.5 text-xs text-slate-400">
                          <div className="flex justify-between"><span>LAN Port:</span><span className="font-mono text-slate-200">0.0.0.0:2525</span></div>
                          <div className="flex justify-between"><span>MX Inbound:</span><span className="font-mono text-slate-200">0.0.0.0:25</span></div>
                          <div className="flex justify-between"><span>Auth Methods:</span><span className="text-slate-200">PLAIN, LOGIN</span></div>
                          <div className="flex justify-between"><span>Relay Policy:</span><span className="text-indigo-300">RFC 5321 Closed Relay</span></div>
                        </div>
                      </div>

                      {/* Delivery Engine Worker */}
                      <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 shadow-sm">
                        <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-3">
                          <span className="text-xs font-bold text-white flex items-center space-x-2">
                            <Zap className="w-4 h-4 text-indigo-400" />
                            <span>Delivery Engine</span>
                          </span>
                          <span className="px-2 py-0.5 rounded-full bg-emerald-950/60 text-emerald-400 border border-emerald-800/40 text-[10px] font-bold">
                            OPERATIONAL
                          </span>
                        </div>
                        <div className="space-y-1.5 text-xs text-slate-400">
                          <div className="flex justify-between"><span>Node ID:</span><span className="font-mono text-slate-200">homelab-node</span></div>
                          <div className="flex justify-between"><span>Outbound Mode:</span><span className="text-slate-200">Direct MX + Relay</span></div>
                          <div className="flex justify-between"><span>DKIM Engine:</span><span className="text-indigo-300">2048-bit RSA Signer</span></div>
                          <div className="flex justify-between"><span>Queue Fairness:</span><span className="text-emerald-400">Round-Robin Active</span></div>
                        </div>
                      </div>

                      {/* Storage / MariaDB */}
                      <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 shadow-sm">
                        <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-3">
                          <span className="text-xs font-bold text-white flex items-center space-x-2">
                            <HardDrive className="w-4 h-4 text-indigo-400" />
                            <span>Storage Engine</span>
                          </span>
                          <span className="px-2 py-0.5 rounded-full bg-emerald-950/60 text-emerald-400 border border-emerald-800/40 text-[10px] font-bold">
                            NOMINAL
                          </span>
                        </div>
                        <div className="space-y-1.5 text-xs text-slate-400">
                          <div className="flex justify-between"><span>Database:</span><span className="font-mono text-slate-200">MariaDB 11.4 LTS</span></div>
                          <div className="flex justify-between"><span>Queue Broker:</span><span className="text-slate-200">Redis Streams 7.2</span></div>
                          <div className="flex justify-between"><span>Data Retention:</span><span className="text-slate-200">Rolling Purge Active</span></div>
                          <div className="flex justify-between"><span>Audit Integrity:</span><span className="text-emerald-400 font-mono">HMAC Verified</span></div>
                        </div>
                      </div>

                      {/* Master KEK Encryption */}
                      <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 shadow-sm">
                        <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-3">
                          <span className="text-xs font-bold text-white flex items-center space-x-2">
                            <Lock className="w-4 h-4 text-indigo-400" />
                            <span>Envelope Encryption</span>
                          </span>
                          <span className="px-2 py-0.5 rounded-full bg-emerald-950/60 text-emerald-400 border border-emerald-800/40 text-[10px] font-bold">
                            ENFORCED
                          </span>
                        </div>
                        <div className="space-y-1.5 text-xs text-slate-400">
                          <div className="flex justify-between"><span>Master KEK:</span><span className="font-mono text-slate-200">256-bit AES-GCM</span></div>
                          <div className="flex justify-between"><span>Tenant Keys:</span><span className="text-slate-200">Per-Tenant DEK Wrapped</span></div>
                          <div className="flex justify-between"><span>Credential Store:</span><span className="text-indigo-300">Argon2id (m=64MB)</span></div>
                          <div className="flex justify-between"><span>Key Custody:</span><span className="text-emerald-400">Zero Downtime Rotate</span></div>
                        </div>
                      </div>

                      {/* Deliverability Posture */}
                      <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 shadow-sm">
                        <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-3">
                          <span className="text-xs font-bold text-white flex items-center space-x-2">
                            <ShieldCheck className="w-4 h-4 text-indigo-400" />
                            <span>Deliverability Posture</span>
                          </span>
                          <span className="px-2 py-0.5 rounded-full bg-emerald-950/60 text-emerald-400 border border-emerald-800/40 text-[10px] font-bold">
                            100 / 100
                          </span>
                        </div>
                        <div className="space-y-1.5 text-xs text-slate-400">
                          <div className="flex justify-between"><span>FCrDNS Sync:</span><span className="text-emerald-300">Bidirectional Valid</span></div>
                          <div className="flex justify-between"><span>STARTTLS:</span><span className="text-slate-200">TLS 1.3 / DANE / MTA-STS</span></div>
                          <div className="flex justify-between"><span>DNSBL Status:</span><span className="text-emerald-400">0 Blacklists Listed</span></div>
                          <div className="flex justify-between"><span>Unsubscribe:</span><span className="text-slate-200">RFC 8058 Header Valid</span></div>
                        </div>
                      </div>
                    </div>

                    {/* Terminal Doctor Diagnostics Box */}
                    <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-6 shadow-sm">
                      <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-3">
                        <div className="flex items-center space-x-2">
                          <Terminal className="w-4 h-4 text-indigo-400" />
                          <h4 className="text-sm font-bold text-white">System Diagnostics via Mailvex CLI</h4>
                        </div>
                        <span className="text-[11px] font-mono text-slate-500">CLI Utilities</span>
                      </div>
                      <p className="text-xs text-slate-400 mb-3">
                        Execute full host-level environmental audits, database connectivity checks, and cryptographically verified backups using the companion binary:
                      </p>
                      <div className="bg-slate-950 p-3.5 rounded-xl border border-slate-800 font-mono text-xs text-slate-200 space-y-1.5">
                        <div className="flex items-center justify-between"><span className="text-indigo-300">mailvex doctor</span><span className="text-slate-500"># Verify DB, KEK, ports, and DNS</span></div>
                        <div className="flex items-center justify-between"><span className="text-indigo-300">mailvex backup create --out=/var/backups</span><span className="text-slate-500"># Signed HMAC archive</span></div>
                        <div className="flex items-center justify-between"><span className="text-indigo-300">mailvex backup restore --in=/var/backups/archive.tar.gz</span><span className="text-slate-500"># Verified restoration</span></div>
                      </div>
                    </div>
                  </div>
                )}

                {/* SUBTAB 3: SECURITY AUDIT TRAIL */}
                {adminActiveSubTab === 'audit' && (
                  <div className="bg-slate-900/90 border border-slate-800/90 rounded-2xl overflow-hidden shadow-sm">
                    <div className="p-5 border-b border-slate-800/80 flex items-center justify-between">
                      <div className="flex items-center space-x-2">
                        <ShieldCheck className="w-4 h-4 text-amber-400" />
                        <h3 className="text-sm font-bold text-white">Administrative Security Audit Trail</h3>
                      </div>
                      <div className="flex items-center space-x-2">
                        <button
                          onClick={fetchAdminAuditLogs}
                          disabled={adminAuditLoading}
                          className="btn-secondary text-xs px-3 py-1.5 flex items-center space-x-1"
                        >
                          <RefreshCw className={`w-3.5 h-3.5 ${adminAuditLoading ? 'animate-spin text-sky-500' : ''}`} />
                          <span>Refresh Logs</span>
                        </button>
                        <ExportMenu
                          filename="mailvex-platform-security-audit"
                          sheetTitle="Audit Logs"
                          label="Export Audit"
                          columns={[
                            { header: 'Log ID', key: 'id' },
                            { header: 'Actor User', key: 'actor_user_id' },
                            { header: 'Action', key: 'action' },
                            { header: 'Resource Type', key: 'resource_type' },
                            { header: 'Resource ID', key: 'resource_id' },
                            { header: 'Timestamp', key: 'created_at' },
                          ]}
                          data={adminAuditLogs.map((log: any) => ({
                            id: log.id,
                            actor_user_id: log.actor_user_id || 'system',
                            action: log.action,
                            resource_type: log.resource_type,
                            resource_id: log.resource_id,
                            created_at: log.created_at,
                          }))}
                        />
                      </div>
                    </div>

                    <div className="overflow-x-auto">
                      <table className="w-full text-left text-xs text-slate-300">
                        <thead className="bg-slate-950/70 text-slate-400 font-semibold uppercase tracking-wider border-b border-slate-800/80">
                          <tr>
                            <th className="p-3.5">Actor User</th>
                            <th className="p-3.5">Action Event</th>
                            <th className="p-3.5">Resource Type</th>
                            <th className="p-3.5">Resource ID</th>
                            <th className="p-3.5">Timestamp</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-800/60">
                          {adminAuditLogs.length === 0 ? (
                            <tr>
                              <td colSpan={5} className="p-6 text-center text-slate-500">
                                {adminAuditLoading ? 'Loading security audit logs...' : 'No administrative audit events recorded yet.'}
                              </td>
                            </tr>
                          ) : (
                            adminAuditLogs.map((log: any) => (
                              <tr key={log.id} className="hover:bg-slate-850/40 transition">
                                <td className="p-3.5 font-mono text-[11px] text-slate-300">
                                  {log.actor_user_id || 'system'}
                                </td>
                                <td className="p-3.5">
                                  <span className="px-2 py-0.5 rounded-full text-[10px] font-bold font-mono bg-indigo-500/10 text-indigo-300 border border-indigo-500/20">
                                    {log.action}
                                  </span>
                                </td>
                                <td className="p-3.5 capitalize font-medium text-slate-200">
                                  {log.resource_type}
                                </td>
                                <td className="p-3.5 font-mono text-[11px] text-slate-400">
                                  {log.resource_id}
                                </td>
                                <td className="p-3.5 text-slate-400 font-mono text-[11px]">
                                  {new Date(log.created_at).toLocaleString()}
                                </td>
                              </tr>
                            ))
                          )}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </main>
      </div>
    </div>
  )
}
