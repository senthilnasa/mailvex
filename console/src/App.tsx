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

export default function App() {
  const [setupCompleted, setSetupCompleted] = useState<boolean | null>(null)
  const [user, setUser] = useState<User | null>(null)
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [activeTenant, setActiveTenant] = useState<Tenant | null>(null)
  const [currentTab, setCurrentTab] = useState<'dashboard' | 'domains' | 'streams'>('dashboard')

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
    }
  }, [activeTenant])

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
                </h1>
                <p className="text-sm text-slate-400 mt-1">
                  Tenant: <code className="text-slate-300">{activeTenant?.slug}</code> (
                  {activeTenant?.environment})
                </p>
              </div>
              <div className="flex items-center space-x-2">
                <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
                <span className="text-sm text-emerald-400 font-medium">Phase 2 Active</span>
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
          </div>
        </main>
      </div>
    </div>
  )
}
