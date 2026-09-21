import React, { useState, useEffect } from 'react'

interface Tenant {
  id: string
  name: string
  slug: string
  environment: string
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
  period_start: string
  period_end: string
  sent_count: number
  delivered_count: number
  delivery_rate: number
  hard_bounce_count: number
  soft_bounce_count: number
  bounce_rate: number
  complaint_count: number
  complaint_rate: number
  complaint_warning: boolean
  unsubscribe_count: number
  unsubscribe_rate: number
  median_latency_seconds: number
  p95_latency_seconds: number
}

interface ProviderBreakdownItem {
  provider: string
  sent_count: number
  delivered_count: number
  bounce_count: number
  complaint_count: number
  delivery_rate: number
  bounce_rate: number
  complaint_rate: number
}

interface LatencyHistogramItem {
  bucket: string
  count: number
  percentage: number
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

export default function App() {
  const [setupCompleted, setSetupCompleted] = useState<boolean | null>(null)
  const [user, setUser] = useState<User | null>(null)
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [activeTenant, setActiveTenant] = useState<Tenant | null>(null)
  const [currentTab, setCurrentTab] = useState<'dashboard' | 'domains' | 'streams' | 'apikeys' | 'messages' | 'templates' | 'suppressions' | 'reputation' | 'analytics' | 'webhooks' | 'privacy_retention'>('dashboard')

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
  const [analyticsLoading, setAnalyticsLoading] = useState(false)
  const [exportLoading, setExportLoading] = useState(false)

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

  useEffect(() => {
    fetch('/v1/setup')
      .then((res) => res.json())
      .then((data) => setSetupCompleted(data.setup_completed))
      .catch(() => setSetupCompleted(false))
  }, [])

  useEffect(() => {
    if (activeTenant && user) {
      loadDomains()
      loadStreams()
      loadAPIKeys()
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
    }
  }, [activeTenant, currentTab, abuseStatusFilter, analyticsRange])

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

  const handleExportSuppressions = () => {
    if (!activeTenant || !user) return
    window.open(`/v1/suppressions/export?format=csv`, '_blank')
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
      const [overviewRes, providersRes, latencyRes] = await Promise.all([
        fetch(`/v1/analytics/overview${q}`, {
          headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
        }),
        fetch(`/v1/analytics/providers${q}`, {
          headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
        }),
        fetch(`/v1/analytics/latency${q}`, {
          headers: { 'X-User-ID': user.id, 'X-Tenant-ID': activeTenant.id },
        }),
      ])

      if (overviewRes.ok) {
        const data = await overviewRes.json()
        setAnalyticsSummary(data)
      }
      if (providersRes.ok) {
        const data = await providersRes.json()
        setAnalyticsProviders(data.data || [])
      }
      if (latencyRes.ok) {
        const data = await latencyRes.json()
        setAnalyticsLatency(data.data || [])
      }
    } catch {}
    finally {
      setAnalyticsLoading(false)
    }
  }

  const handleExportCSV = async () => {
    if (!activeTenant || !user) return
    setExportLoading(true)
    try {
      const res = await fetch('/v1/analytics/export', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': user.id,
          'X-Tenant-ID': activeTenant.id,
        },
        body: JSON.stringify({ range: analyticsRange }),
      })
      const data = await res.json()
      if (res.ok && data.export?.id) {
        window.open(`/v1/analytics/exports/${data.export.id}/download`, '_blank')
      } else {
        alert(data.error?.message || 'Failed to generate export')
      }
    } catch (err: any) {
      alert(err.message)
    } finally {
      setExportLoading(false)
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
      setTenants(data.tenants || [])
      if (data.tenants && data.tenants.length > 0) {
        setActiveTenant(data.tenants[0])
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
      <div className="flex items-center justify-center min-h-screen bg-slate-950 text-slate-400">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-indigo-500"></div>
      </div>
    )
  }

  // 1. Setup Wizard View
  if (!setupCompleted) {
    return (
      <div className="min-h-screen bg-slate-950 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
        <div className="sm:mx-auto sm:w-full sm:max-w-md">
          <div className="flex justify-center">
            <div className="w-12 h-12 bg-indigo-600 rounded-xl flex items-center justify-center font-bold text-white text-2xl shadow-lg shadow-indigo-500/20">
              M
            </div>
          </div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-white">
            Welcome to Mailvex
          </h2>
          <p className="mt-2 text-center text-sm text-slate-400">
            Initial Platform Installation Wizard
          </p>
        </div>

        <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
          <div className="bg-slate-900 py-8 px-6 shadow-xl border border-slate-800 sm:rounded-2xl sm:px-10">
            {setupError && (
              <div className="mb-4 bg-red-950/50 border border-red-500 text-red-300 text-sm p-3 rounded-lg">
                {setupError}
              </div>
            )}

            <form className="space-y-5" onSubmit={handleSetup}>
              <div>
                <label className="block text-sm font-medium text-slate-300">
                  Deployment Profile
                </label>
                <select
                  value={profile}
                  onChange={(e) => setProfile(e.target.value)}
                  className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-slate-700 bg-slate-800 text-slate-200 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 rounded-lg"
                >
                  <option value="standard">Standard (Production / Single Server)</option>
                  <option value="homelab">Home Lab (Low Resource / IoT Mail Hub)</option>
                  <option value="cluster">Cluster (High Availability Multi-Node)</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300">
                  Organization Name
                </label>
                <input
                  type="text"
                  required
                  value={orgName}
                  onChange={(e) => setOrgName(e.target.value)}
                  placeholder="e.g. Acme Corporation"
                  className="mt-1 block w-full px-3 py-2 border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300">
                  Primary Tenant Name
                </label>
                <input
                  type="text"
                  required
                  value={tenantName}
                  onChange={(e) => setTenantName(e.target.value)}
                  placeholder="e.g. Production"
                  className="mt-1 block w-full px-3 py-2 border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300">
                  Platform Admin Email
                </label>
                <input
                  type="email"
                  required
                  value={adminEmail}
                  onChange={(e) => setAdminEmail(e.target.value)}
                  placeholder="admin@example.com"
                  className="mt-1 block w-full px-3 py-2 border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300">
                  Admin Password
                </label>
                <input
                  type="password"
                  required
                  value={adminPassword}
                  onChange={(e) => setAdminPassword(e.target.value)}
                  placeholder="At least 8 characters"
                  className="mt-1 block w-full px-3 py-2 border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>

              <div>
                <button
                  type="submit"
                  disabled={setupLoading}
                  className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors disabled:opacity-50"
                >
                  {setupLoading ? 'Configuring System...' : 'Complete Installation'}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    )
  }

  // 2. Login View
  if (!user) {
    return (
      <div className="min-h-screen bg-slate-950 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
        <div className="sm:mx-auto sm:w-full sm:max-w-md">
          <div className="flex justify-center">
            <div className="w-12 h-12 bg-indigo-600 rounded-xl flex items-center justify-center font-bold text-white text-2xl shadow-lg shadow-indigo-500/20">
              M
            </div>
          </div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-white">
            Mailvex Console
          </h2>
          <p className="mt-2 text-center text-sm text-slate-400">
            Sign in to manage your email infrastructure
          </p>
        </div>

        <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
          <div className="bg-slate-900 py-8 px-6 shadow-xl border border-slate-800 sm:rounded-2xl sm:px-10">
            {loginError && (
              <div className="mb-4 bg-red-950/50 border border-red-500 text-red-300 text-sm p-3 rounded-lg">
                {loginError}
              </div>
            )}

            <form className="space-y-6" onSubmit={handleLogin}>
              <div>
                <label className="block text-sm font-medium text-slate-300">
                  Email address
                </label>
                <input
                  type="email"
                  required
                  value={loginEmail}
                  onChange={(e) => setLoginEmail(e.target.value)}
                  className="mt-1 block w-full px-3 py-2 border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300">
                  Password
                </label>
                <input
                  type="password"
                  required
                  value={loginPassword}
                  onChange={(e) => setLoginPassword(e.target.value)}
                  className="mt-1 block w-full px-3 py-2 border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>

              <div>
                <button
                  type="submit"
                  disabled={loginLoading}
                  className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors disabled:opacity-50"
                >
                  {loginLoading ? 'Signing in...' : 'Sign In'}
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
    <div className="min-h-screen bg-slate-950 flex flex-col">
      {/* Top Header Navigation */}
      <header className="bg-slate-900 border-b border-slate-800 px-6 py-3 flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <div className="w-8 h-8 bg-indigo-600 rounded-lg flex items-center justify-center font-bold text-white text-lg">
            M
          </div>
          <span className="text-lg font-bold text-white">Mailvex</span>

          {/* Tenant Switcher */}
          <div className="relative inline-block text-left ml-6">
            <select
              value={activeTenant?.id || ''}
              onChange={(e) => {
                const selected = tenants.find((t) => t.id === e.target.value)
                if (selected) setActiveTenant(selected)
              }}
              className="bg-slate-800 border border-slate-700 text-slate-200 text-sm rounded-lg px-3 py-1.5 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
            >
              {tenants.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} ({t.slug})
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="flex items-center space-x-4">
          <span className="text-sm text-slate-400">{user.email}</span>
          {user.is_platform_admin && (
            <span className="px-2 py-0.5 text-xs font-semibold bg-amber-500/20 text-amber-300 border border-amber-500/30 rounded-full">
              Platform Admin
            </span>
          )}
          <button
            onClick={() => setUser(null)}
            className="text-sm text-slate-400 hover:text-white transition-colors"
          >
            Sign Out
          </button>
        </div>
      </header>

      {/* Main Content Area */}
      <div className="flex-1 flex overflow-hidden">
        {/* Sidebar */}
        <nav className="w-64 bg-slate-900/60 border-r border-slate-800 p-4 space-y-1">
          <div className="px-3 py-2 text-xs font-semibold uppercase tracking-wider text-slate-400">
            Navigation
          </div>
          <button
            onClick={() => setCurrentTab('dashboard')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'dashboard'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            Dashboard
          </button>
          <button
            onClick={() => setCurrentTab('domains')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'domains'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            Domains & DNS
          </button>
          <button
            onClick={() => setCurrentTab('streams')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'streams'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            Message Streams
          </button>
          <button
            onClick={() => setCurrentTab('apikeys')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'apikeys'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            API Keys
          </button>
          <button
            onClick={() => setCurrentTab('messages')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'messages'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            Message Logs & Inspector
          </button>
          <button
            onClick={() => setCurrentTab('templates')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'templates'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            Templates
          </button>
          <button
            onClick={() => setCurrentTab('suppressions')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'suppressions'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            Suppressions
          </button>
          <button
            onClick={() => setCurrentTab('reputation')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'reputation'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            Reputation & Abuse
          </button>
          <button
            onClick={() => setCurrentTab('analytics')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'analytics'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            Analytics & Reports
          </button>
          <button
            onClick={() => setCurrentTab('webhooks')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'webhooks'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            Webhooks & Signing
          </button>
          <button
            onClick={() => setCurrentTab('privacy_retention')}
            className={`w-full text-left flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              currentTab === 'privacy_retention'
                ? 'bg-indigo-600/10 text-indigo-400 border border-indigo-500/20'
                : 'text-slate-300 hover:bg-slate-800 hover:text-white'
            }`}
          >
            Privacy & Retention
          </button>
        </nav>

        {/* Workspace Body */}
        <main className="flex-1 overflow-y-auto p-8">
          <div className="max-w-5xl mx-auto">
            {/* Top Bar */}
            <div className="flex items-center justify-between pb-6 border-b border-slate-800">
              <div>
                <h1 className="text-2xl font-bold text-white">
                  {currentTab === 'dashboard' && (activeTenant?.name || 'Dashboard')}
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
                </h1>
                <p className="text-sm text-slate-400 mt-1">
                  Tenant: <code className="text-slate-300">{activeTenant?.slug}</code> (
                  {activeTenant?.environment})
                </p>
              </div>
              <div className="flex items-center space-x-2">
                <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
                <span className="text-sm text-emerald-400 font-medium">Phase 9 Active</span>
              </div>
            </div>

            {/* TAB: DASHBOARD */}
            {currentTab === 'dashboard' && (
              <>
                <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mt-6">
                  <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                    <div className="text-xs text-slate-400 uppercase font-semibold">
                      Total Domains
                    </div>
                    <div className="text-2xl font-bold text-white mt-2">{domains.length}</div>
                    <div className="text-xs text-slate-500 mt-1">
                      {domains.filter((d) => d.status === 'verified').length} verified
                    </div>
                  </div>
                  <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                    <div className="text-xs text-slate-400 uppercase font-semibold">
                      Message Streams
                    </div>
                    <div className="text-2xl font-bold text-white mt-2">{streams.length}</div>
                    <div className="text-xs text-slate-500 mt-1">Default 2 streams</div>
                  </div>
                  <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                    <div className="text-xs text-slate-400 uppercase font-semibold">
                      Security Engine
                    </div>
                    <div className="text-2xl font-bold text-indigo-400 mt-2">RSA 2048</div>
                    <div className="text-xs text-slate-500 mt-1">DKIM active</div>
                  </div>
                  <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                    <div className="text-xs text-slate-400 uppercase font-semibold">
                      Verification
                    </div>
                    <div className="text-2xl font-bold text-emerald-400 mt-2">Active</div>
                    <div className="text-xs text-slate-500 mt-1">Multi-resolver</div>
                  </div>
                </div>

                <div className="mt-8 bg-slate-900 border border-slate-800 rounded-xl p-6">
                  <h2 className="text-lg font-semibold text-white">
                    Phase 2: Domains & DNS Active
                  </h2>
                  <p className="text-sm text-slate-400 mt-2">
                    Multi-resolver DNS verification engine, RSA 2048 DKIM key generator,
                    global domain uniqueness guard, degraded/grace period scheduler, and default
                    message streams are active and validated.
                  </p>
                </div>
              </>
            )}

            {/* TAB: DOMAINS */}
            {currentTab === 'domains' && (
              <div className="mt-6 space-y-8">
                {/* Add Domain Form */}
                <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl">
                  <h2 className="text-lg font-semibold text-white">Add Sending Domain</h2>
                  <form onSubmit={handleAddDomain} className="mt-4 flex gap-4">
                    <input
                      type="text"
                      required
                      placeholder="e.g. mail.yourdomain.com"
                      value={newDomainName}
                      onChange={(e) => setNewDomainName(e.target.value)}
                      className="flex-1 px-4 py-2 border border-slate-700 bg-slate-800 text-slate-200 rounded-lg focus:ring-indigo-500 focus:border-indigo-500"
                    />
                    <select
                      value={newDomainType}
                      onChange={(e) => setNewDomainType(e.target.value)}
                      className="px-4 py-2 border border-slate-700 bg-slate-800 text-slate-200 rounded-lg"
                    >
                      <option value="public">Public Domain</option>
                      <option value="local">Local Domain (Home Lab)</option>
                    </select>
                    <button
                      type="submit"
                      disabled={addDomainLoading}
                      className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg transition-colors disabled:opacity-50"
                    >
                      {addDomainLoading ? 'Adding...' : 'Add Domain'}
                    </button>
                  </form>
                </div>

                {/* Domains List */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="p-4 border-b border-slate-800 font-semibold text-white">
                    Configured Domains
                  </div>
                  {domains.length === 0 ? (
                    <div className="p-6 text-center text-slate-500">
                      No domains added yet. Add your first domain above.
                    </div>
                  ) : (
                    <table className="w-full text-left text-sm text-slate-300">
                      <thead className="bg-slate-950/50 text-slate-400">
                        <tr>
                          <th className="p-4">Domain</th>
                          <th className="p-4">Type</th>
                          <th className="p-4">Status</th>
                          <th className="p-4">Action</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {domains.map((d) => (
                          <tr key={d.id} className="hover:bg-slate-800/40">
                            <td className="p-4 font-medium text-white">{d.name}</td>
                            <td className="p-4 capitalize">{d.type}</td>
                            <td className="p-4">
                              <span
                                className={`px-2.5 py-1 text-xs font-semibold rounded-full border ${
                                  d.status === 'verified'
                                    ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                                    : d.status === 'degraded'
                                    ? 'bg-amber-500/10 text-amber-400 border-amber-500/20'
                                    : 'bg-slate-500/10 text-slate-400 border-slate-500/20'
                                }`}
                              >
                                {d.status}
                              </span>
                            </td>
                            <td className="p-4">
                              <button
                                onClick={() => handleSelectDomain(d)}
                                className="text-indigo-400 hover:text-indigo-300 font-medium"
                              >
                                View DNS Records
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>

                {/* Selected Domain Records View */}
                {selectedDomain && (
                  <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl space-y-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <h3 className="text-lg font-bold text-white">
                          DNS Records for {selectedDomain.name}
                        </h3>
                        <p className="text-sm text-slate-400">
                          Create these records in your DNS provider to authenticate this domain.
                        </p>
                      </div>
                      <button
                        onClick={handleVerifyDomain}
                        disabled={verifyLoading}
                        className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white font-medium rounded-lg transition-colors disabled:opacity-50"
                      >
                        {verifyLoading ? 'Checking DNS...' : 'Verify DNS Now'}
                      </button>
                    </div>

                    <div className="space-y-3">
                      {domainRecords.map((r, i) => (
                        <div
                          key={i}
                          className="bg-slate-950 p-4 rounded-lg border border-slate-800 flex flex-col md:flex-row md:items-center justify-between gap-4"
                        >
                          <div className="space-y-1 overflow-hidden">
                            <div className="flex items-center space-x-2">
                              <span className="px-2 py-0.5 bg-slate-800 text-slate-300 font-mono text-xs rounded">
                                {r.record_type}
                              </span>
                              <code className="text-sm text-indigo-300 font-mono">{r.name}</code>
                              <span
                                className={`text-xs px-2 py-0.5 rounded ${
                                  r.status === 'pass'
                                    ? 'bg-emerald-500/10 text-emerald-400'
                                    : 'bg-amber-500/10 text-amber-400'
                                }`}
                              >
                                {r.status}
                              </span>
                            </div>
                            <div className="text-xs text-slate-400 font-mono truncate max-w-xl">
                              {r.expected_value}
                            </div>
                          </div>
                          <button
                            onClick={() => {
                              navigator.clipboard.writeText(r.expected_value)
                              alert('Copied to clipboard!')
                            }}
                            className="text-xs px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded transition-colors self-start md:self-auto"
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
              <div className="mt-6 bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                <div className="p-4 border-b border-slate-800 font-semibold text-white">
                  Message Streams
                </div>
                <table className="w-full text-left text-sm text-slate-300">
                  <thead className="bg-slate-950/50 text-slate-400">
                    <tr>
                      <th className="p-4">Stream Name</th>
                      <th className="p-4">Type</th>
                      <th className="p-4">Default</th>
                      <th className="p-4">Purpose</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800">
                    {streams.map((s) => (
                      <tr key={s.id} className="hover:bg-slate-800/40">
                        <td className="p-4 font-medium text-white">{s.name}</td>
                        <td className="p-4 capitalize">{s.type}</td>
                        <td className="p-4">{s.is_default ? 'Yes' : 'No'}</td>
                        <td className="p-4 text-slate-400 text-xs">
                          {s.type === 'transactional'
                            ? 'Receipts, password resets, alerts (Priority p1)'
                            : 'Broadcast newsletters, marketing (RFC 8058 One-Click unsub)'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
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
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="p-4 border-b border-slate-800 flex items-center justify-between">
                    <span className="font-semibold text-white">Message Activity Logs</span>
                    <button
                      onClick={loadMessages}
                      className="text-xs px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg transition-colors"
                    >
                      Refresh
                    </button>
                  </div>
                  <table className="w-full text-left text-sm text-slate-300">
                    <thead className="bg-slate-950/50 text-slate-400">
                      <tr>
                        <th className="p-4">Message ID</th>
                        <th className="p-4">Sender</th>
                        <th className="p-4">Subject</th>
                        <th className="p-4">Status</th>
                        <th className="p-4">Environment</th>
                        <th className="p-4">Time</th>
                        <th className="p-4 text-right">Action</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-800">
                      {messagesList.length === 0 ? (
                        <tr>
                          <td colSpan={7} className="p-6 text-center text-slate-500">
                            No messages recorded yet.
                          </td>
                        </tr>
                      ) : (
                        messagesList.map((m) => (
                          <tr key={m.id} className="hover:bg-slate-800/40">
                            <td className="p-4 font-mono text-xs text-indigo-400">{m.id}</td>
                            <td className="p-4 text-slate-200">{m.from_address}</td>
                            <td className="p-4 text-white font-medium">{m.subject}</td>
                            <td className="p-4">
                              <span
                                className={`px-2 py-0.5 text-xs rounded-full font-medium ${
                                  m.status === 'delivered'
                                    ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                                    : m.status === 'queued'
                                    ? 'bg-blue-500/10 text-blue-400 border border-blue-500/20'
                                    : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                                }`}
                              >
                                {m.status}
                              </span>
                            </td>
                            <td className="p-4 text-xs capitalize text-slate-400">{m.environment}</td>
                            <td className="p-4 text-xs text-slate-400">
                              {new Date(m.created_at).toLocaleTimeString()}
                            </td>
                            <td className="p-4 text-right">
                              <button
                                onClick={() => handleInspectMessage(m)}
                                className="px-3 py-1 bg-indigo-600/10 hover:bg-indigo-600/20 text-indigo-400 border border-indigo-500/20 text-xs rounded-lg transition-colors"
                              >
                                Inspect
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
                  <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
                    <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
                      <div className="p-6 border-b border-slate-800 flex items-center justify-between">
                        <div>
                          <h3 className="text-lg font-bold text-white">Message Inspector</h3>
                          <p className="text-xs font-mono text-indigo-400">{selectedMessage.id}</p>
                        </div>
                        <button
                          onClick={() => setInspectorOpen(false)}
                          className="text-slate-400 hover:text-white text-xl font-bold"
                        >
                          ✕
                        </button>
                      </div>
                      <div className="p-6 overflow-y-auto space-y-6 flex-1">
                        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 bg-slate-950/60 p-4 rounded-xl border border-slate-800">
                          <div>
                            <span className="block text-xs text-slate-500">From</span>
                            <span className="text-sm font-medium text-slate-200">{selectedMessage.from_address}</span>
                          </div>
                          <div>
                            <span className="block text-xs text-slate-500">Subject</span>
                            <span className="text-sm font-medium text-slate-200">{selectedMessage.subject}</span>
                          </div>
                          <div>
                            <span className="block text-xs text-slate-500">Status</span>
                            <span className="text-sm font-medium text-indigo-400 capitalize">{selectedMessage.status}</span>
                          </div>
                          <div>
                            <span className="block text-xs text-slate-500">Environment</span>
                            <span className="text-sm font-medium text-emerald-400 capitalize">{selectedMessage.environment}</span>
                          </div>
                        </div>

                        {/* Recipients */}
                        <div>
                          <h4 className="text-sm font-semibold text-slate-300 mb-2">Recipients</h4>
                          <div className="bg-slate-950/40 border border-slate-800 rounded-lg overflow-hidden">
                            <table className="w-full text-left text-xs text-slate-300">
                              <thead className="bg-slate-900 text-slate-400">
                                <tr>
                                  <th className="p-3">Address</th>
                                  <th className="p-3">Type</th>
                                  <th className="p-3">Status</th>
                                  <th className="p-3">Attempts</th>
                                </tr>
                              </thead>
                              <tbody className="divide-y divide-slate-800">
                                {selectedMsgRecipients.map((r) => (
                                  <tr key={r.id}>
                                    <td className="p-3 font-mono">{r.address}</td>
                                    <td className="p-3 uppercase">{r.type}</td>
                                    <td className="p-3 capitalize">{r.status}</td>
                                    <td className="p-3">{r.attempts}</td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          </div>
                        </div>

                        {/* Chronological Event History */}
                        <div>
                          <h4 className="text-sm font-semibold text-slate-300 mb-2">Event Timeline</h4>
                          <div className="space-y-2">
                            {selectedMsgEvents.map((e) => (
                              <div key={e.id} className="text-xs bg-slate-950/50 p-2.5 rounded-lg border border-slate-800 flex items-center justify-between">
                                <div className="flex items-center space-x-2">
                                  <span className="w-2 h-2 rounded-full bg-indigo-500"></span>
                                  <span className="font-semibold text-indigo-300">{e.type}</span>
                                  <span className="text-slate-400">{e.data}</span>
                                </div>
                                <span className="text-slate-500 font-mono">
                                  {new Date(e.occurred_at).toLocaleTimeString()}
                                </span>
                              </div>
                            ))}
                          </div>
                        </div>

                        {/* Sandboxed Body HTML Preview (§23.3) */}
                        <div>
                          <h4 className="text-sm font-semibold text-slate-300 mb-2">Message Body Preview (Sandboxed)</h4>
                          <iframe
                            sandbox=""
                            srcDoc={selectedMsgBody}
                            className="w-full h-64 bg-white rounded-lg border border-slate-700 shadow-inner"
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
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                  <h3 className="text-lg font-semibold text-white mb-4">Create Template</h3>
                  <form onSubmit={handleCreateTemplate} className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-medium text-slate-400 mb-1">Template Name</label>
                        <input
                          type="text"
                          required
                          placeholder="e.g. Welcome Onboarding"
                          value={newTmplName}
                          onChange={(e) => setNewTmplName(e.target.value)}
                          className="w-full bg-slate-800 border border-slate-700 text-slate-200 text-sm px-3 py-2 rounded-lg"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-slate-400 mb-1">Subject (Mustache)</label>
                        <input
                          type="text"
                          required
                          placeholder="Welcome to {{company}}, {{name}}!"
                          value={newTmplSubject}
                          onChange={(e) => setNewTmplSubject(e.target.value)}
                          className="w-full bg-slate-800 border border-slate-700 text-slate-200 text-sm px-3 py-2 rounded-lg"
                        />
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-400 mb-1">HTML Body</label>
                      <textarea
                        rows={3}
                        placeholder="<h1>Welcome {{name}}!</h1><p>We are excited to have you.</p>"
                        value={newTmplHTML}
                        onChange={(e) => setNewTmplHTML(e.target.value)}
                        className="w-full bg-slate-800 border border-slate-700 text-slate-200 text-sm px-3 py-2 rounded-lg font-mono"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-400 mb-1">Plain Text Body</label>
                      <textarea
                        rows={2}
                        placeholder="Welcome {{name}}! We are excited to have you."
                        value={newTmplText}
                        onChange={(e) => setNewTmplText(e.target.value)}
                        className="w-full bg-slate-800 border border-slate-700 text-slate-200 text-sm px-3 py-2 rounded-lg font-mono"
                      />
                    </div>
                    <button
                      type="submit"
                      className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-lg transition-colors"
                    >
                      Save & Publish Template
                    </button>
                  </form>
                </div>

                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="p-4 border-b border-slate-800 font-semibold text-white">
                    Published Templates
                  </div>
                  <div className="divide-y divide-slate-800">
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

                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                  <div className="p-4 border-b border-slate-800 flex items-center justify-between font-semibold text-white">
                    <span>Suppressed Recipients ({suppressionsList.length})</span>
                    <div className="flex items-center space-x-3">
                      <button
                        onClick={() => setImportModalOpen(true)}
                        className="px-3 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-medium rounded border border-slate-700 transition-colors"
                      >
                        Import CSV
                      </button>
                      <button
                        onClick={handleExportSuppressions}
                        className="px-3 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-medium rounded border border-slate-700 transition-colors"
                      >
                        Export CSV
                      </button>
                      <button
                        onClick={loadSuppressions}
                        className="text-xs text-indigo-400 hover:text-indigo-300 font-normal"
                      >
                        Refresh
                      </button>
                    </div>
                  </div>
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm text-slate-300">
                      <thead className="bg-slate-950 text-xs text-slate-400 uppercase border-b border-slate-800">
                        <tr>
                          <th className="px-6 py-3">Address / Domain</th>
                          <th className="px-6 py-3">Type</th>
                          <th className="px-6 py-3">Reason</th>
                          <th className="px-6 py-3">Source</th>
                          <th className="px-6 py-3">Created</th>
                          <th className="px-6 py-3 text-right">Action</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800">
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
                  <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
                    <div className="bg-slate-900 border border-slate-800 rounded-xl max-w-lg w-full p-6 shadow-2xl space-y-4">
                      <div className="flex items-center justify-between border-b border-slate-800 pb-3">
                        <h4 className="text-md font-semibold text-white">Bulk Import Suppressions (CSV)</h4>
                        <button
                          onClick={() => setImportModalOpen(false)}
                          className="text-slate-400 hover:text-slate-200 text-sm"
                        >
                          ✕
                        </button>
                      </div>

                      {importStatusMsg && (
                        <div className={`p-3 rounded-lg text-xs font-medium ${
                          importStatusMsg.startsWith('Import completed')
                            ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                            : 'bg-rose-950 text-rose-400 border border-rose-800'
                        }`}>
                          {importStatusMsg}
                        </div>
                      )}

                      <form onSubmit={handleImportSuppressions} className="space-y-4">
                        <div>
                          <label className="block text-xs font-medium text-slate-400 mb-1">
                            CSV Content (format: email,type,reason)
                          </label>
                          <textarea
                            rows={6}
                            required
                            placeholder={"email,type,reason\nbounced@example.com,hard_bounce,Mailbox inactive\nunsub@example.com,unsubscribe,User opted out"}
                            value={importCSVText}
                            onChange={(e) => setImportCSVText(e.target.value)}
                            className="w-full bg-slate-800 border border-slate-700 text-slate-200 font-mono text-xs px-3 py-2 rounded-lg"
                          />
                        </div>
                        <div className="flex justify-end space-x-3 pt-2 border-t border-slate-800">
                          <button
                            type="button"
                            onClick={() => setImportModalOpen(false)}
                            className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm rounded-lg"
                          >
                            Cancel
                          </button>
                          <button
                            type="submit"
                            className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-lg"
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
              <div className="mt-6 space-y-8">
                {/* Time range selector & Export CSV */}
                <div className="flex flex-wrap items-center justify-between gap-4 bg-slate-900 border border-slate-800 p-4 rounded-xl">
                  <div className="flex items-center space-x-2">
                    <span className="text-xs font-semibold uppercase text-slate-400 mr-2">Time Range:</span>
                    {(['24h', '7d', '30d'] as const).map((r) => (
                      <button
                        key={r}
                        onClick={() => setAnalyticsRange(r)}
                        className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-colors ${
                          analyticsRange === r
                            ? 'bg-indigo-600 text-white shadow'
                            : 'bg-slate-800 text-slate-300 hover:bg-slate-700'
                        }`}
                      >
                        {r === '24h' ? 'Last 24 Hours' : r === '7d' ? 'Last 7 Days' : 'Last 30 Days'}
                      </button>
                    ))}
                    {analyticsLoading && (
                      <span className="text-xs text-indigo-400 animate-pulse ml-2">Loading metrics...</span>
                    )}
                  </div>

                  <div>
                    <button
                      onClick={handleExportCSV}
                      disabled={exportLoading}
                      className="px-4 py-2 text-xs font-semibold bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg transition-colors flex items-center space-x-2 disabled:opacity-50"
                    >
                      <span>{exportLoading ? 'Generating...' : '📥 Export CSV Report'}</span>
                    </button>
                  </div>
                </div>

                {/* Reputation Alert if complaint rate exceeds 0.10% threshold */}
                {analyticsSummary?.complaint_warning && (
                  <div className="bg-red-950/70 border border-red-500/80 rounded-xl p-4 flex items-start space-x-3 text-red-200">
                    <span className="text-xl">⚠️</span>
                    <div>
                      <div className="font-semibold text-sm text-red-300">Google &amp; Yahoo Deliverability Warning: Complaint Rate Exceeded 0.10%</div>
                      <p className="text-xs text-red-300/80 mt-1">
                        Current tenant complaint rate is {(analyticsSummary.complaint_rate * 100).toFixed(3)}%. Mailbox providers will defer or spam-folder messages if complaint rate approaches or exceeds 0.30%. Review unverified lists and enforce immediate suppression.
                      </p>
                    </div>
                  </div>
                )}

                {/* KPI Summary Cards */}
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-400">Total Sent</div>
                    <div className="text-2xl font-bold text-white mt-2">
                      {analyticsSummary?.sent_count?.toLocaleString() || 0}
                    </div>
                    <div className="text-xs text-slate-500 mt-1">Outbound messages</div>
                  </div>

                  <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-400">Delivery Rate</div>
                    <div className="text-2xl font-bold text-emerald-400 mt-2">
                      {((analyticsSummary?.delivery_rate || 0) * 100).toFixed(2)}%
                    </div>
                    <div className="text-xs text-slate-500 mt-1">
                      {analyticsSummary?.delivered_count?.toLocaleString() || 0} delivered
                    </div>
                  </div>

                  <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-400">Bounce Rate</div>
                    <div className="text-2xl font-bold text-rose-400 mt-2">
                      {((analyticsSummary?.bounce_rate || 0) * 100).toFixed(2)}%
                    </div>
                    <div className="text-xs text-slate-500 mt-1">
                      {analyticsSummary?.hard_bounce_count || 0} hard / {analyticsSummary?.soft_bounce_count || 0} soft
                    </div>
                  </div>

                  <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-400">Complaint Rate</div>
                    <div className={`text-2xl font-bold mt-2 ${analyticsSummary?.complaint_warning ? 'text-red-400' : 'text-slate-200'}`}>
                      {((analyticsSummary?.complaint_rate || 0) * 100).toFixed(3)}%
                    </div>
                    <div className="text-xs text-slate-500 mt-1">
                      {analyticsSummary?.complaint_count || 0} spam reports
                    </div>
                  </div>
                </div>

                {/* Additional Performance Row: Latency & Unsubscribes */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                    <div className="text-xs font-semibold uppercase tracking-wider text-slate-400">Delivery Latency Percentiles</div>
                    <div className="flex items-baseline space-x-4 mt-2">
                      <div>
                        <span className="text-xl font-bold text-indigo-400">
                          {analyticsSummary ? `${analyticsSummary.median_latency_seconds.toFixed(2)}s` : '—'}
                        </span>
                        <span className="text-xs text-slate-500 ml-1">Median (p50)</span>
                      </div>
                      <div>
                        <span className="text-xl font-bold text-sky-400">
                          {analyticsSummary ? `${analyticsSummary.p95_latency_seconds.toFixed(2)}s` : '—'}
                        </span>
                        <span className="text-xs text-slate-500 ml-1">p95 Tail</span>
                      </div>
                    </div>
                    <div className="text-xs text-slate-500 mt-2">
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
                          {analyticsSummary?.unsubscribe_count || 0}
                        </span>
                        <span className="text-xs text-slate-500 ml-1">Total Opt-Outs</span>
                      </div>
                    </div>
                    <div className="text-xs text-slate-500 mt-2">
                      One-click RFC 8058 and mailto opt-outs.
                    </div>
                  </div>
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
                            <span>{item.count} msgs ({item.percentage.toFixed(1)}%)</span>
                          </div>
                          <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
                            <div
                              className="bg-indigo-500 h-2 rounded-full transition-all duration-300"
                              style={{ width: `${Math.min(100, Math.max(item.percentage > 0 ? 3 : 0, item.percentage))}%` }}
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
                              <td className="px-6 py-4 text-slate-300 font-mono text-xs">{p.sent_count.toLocaleString()}</td>
                              <td className="px-6 py-4 text-emerald-400 font-mono text-xs">{p.delivered_count.toLocaleString()}</td>
                              <td className="px-6 py-4 text-slate-200">{(p.delivery_rate * 100).toFixed(1)}%</td>
                              <td className="px-6 py-4 text-rose-400 font-mono text-xs">{p.bounce_count}</td>
                              <td className="px-6 py-4 text-slate-200">{(p.bounce_rate * 100).toFixed(2)}%</td>
                              <td className="px-6 py-4 text-amber-400 font-mono text-xs">{p.complaint_count}</td>
                              <td className="px-6 py-4 font-semibold">
                                <span className={p.complaint_rate >= 0.001 ? 'text-red-400' : 'text-slate-300'}>
                                  {(p.complaint_rate * 100).toFixed(3)}%
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
                  <div className="fixed inset-0 z-50 bg-black/70 flex items-center justify-center p-4">
                    <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-4xl w-full max-h-[85vh] flex flex-col overflow-hidden shadow-2xl">
                      <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
                        <div>
                          <h3 className="text-base font-semibold text-white">
                            Webhook Delivery History
                          </h3>
                          <div className="text-xs text-slate-400 font-mono mt-0.5">
                            {selectedWebhook?.url}
                          </div>
                        </div>
                        <button
                          onClick={() => setDeliveriesModalOpen(false)}
                          className="text-slate-400 hover:text-white text-lg font-bold"
                        >
                          ✕
                        </button>
                      </div>

                      <div className="p-6 overflow-y-auto flex-1">
                        <table className="w-full text-left text-xs text-slate-300">
                          <thead className="bg-slate-950/60 uppercase font-semibold text-slate-400 border-b border-slate-800">
                            <tr>
                              <th className="px-4 py-2">Delivery ID</th>
                              <th className="px-4 py-2">Event</th>
                              <th className="px-4 py-2">Status</th>
                              <th className="px-4 py-2">Duration</th>
                              <th className="px-4 py-2">Time</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-slate-800">
                            {webhookDeliveries.length === 0 ? (
                              <tr>
                                <td colSpan={5} className="px-4 py-6 text-center text-slate-500">
                                  No delivery attempts recorded for this webhook.
                                </td>
                              </tr>
                            ) : (
                              webhookDeliveries.map((del) => (
                                <tr key={del.id} className="hover:bg-slate-800/40">
                                  <td className="px-4 py-3 font-mono text-white">{del.id}</td>
                                  <td className="px-4 py-3 font-mono text-indigo-300">{del.event_type}</td>
                                  <td className="px-4 py-3">
                                    {del.status_code && del.status_code >= 200 && del.status_code < 300 ? (
                                      <span className="text-emerald-400 font-semibold">{del.status_code} OK</span>
                                    ) : (
                                      <span className="text-rose-400 font-semibold">
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
                  <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl space-y-4">
                    <div className="flex items-center justify-between">
                      <h3 className="text-sm font-semibold text-white">
                        Personal Data Export for: <span className="font-mono text-indigo-400">{privacyExportReport.email_address}</span>
                      </h3>
                      <button
                        onClick={() => setPrivacyExportReport(null)}
                        className="text-slate-400 hover:text-white text-xs"
                      >
                        Close
                      </button>
                    </div>
                    <div className="grid grid-cols-3 gap-4 text-center">
                      <div className="p-3 bg-slate-800/60 rounded-lg">
                        <div className="text-xs text-slate-400">Messages</div>
                        <div className="text-lg font-bold text-white mt-1">{privacyExportReport.messages?.length || 0}</div>
                      </div>
                      <div className="p-3 bg-slate-800/60 rounded-lg">
                        <div className="text-xs text-slate-400">Events</div>
                        <div className="text-lg font-bold text-white mt-1">{privacyExportReport.events?.length || 0}</div>
                      </div>
                      <div className="p-3 bg-slate-800/60 rounded-lg">
                        <div className="text-xs text-slate-400">Suppressions</div>
                        <div className="text-lg font-bold text-white mt-1">{privacyExportReport.suppressions?.length || 0}</div>
                      </div>
                    </div>
                    <div className="bg-slate-950 p-4 rounded-lg overflow-x-auto max-h-60">
                      <pre className="text-xs font-mono text-slate-300">
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
          </div>
        </main>
      </div>
    </div>
  )
}
