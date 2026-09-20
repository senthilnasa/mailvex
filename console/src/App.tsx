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

export default function App() {
  const [setupCompleted, setSetupCompleted] = useState<boolean | null>(null)
  const [user, setUser] = useState<User | null>(null)
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [activeTenant, setActiveTenant] = useState<Tenant | null>(null)

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
      // Auto-login
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
            Infrastructure
          </div>
          <a
            href="#dashboard"
            className="flex items-center px-3 py-2 text-sm font-medium rounded-lg bg-indigo-600/10 text-indigo-400 border border-indigo-500/20"
          >
            Dashboard
          </a>
          <a
            href="#domains"
            className="flex items-center px-3 py-2 text-sm font-medium rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-colors"
          >
            Domains & DNS
          </a>
          <a
            href="#messages"
            className="flex items-center px-3 py-2 text-sm font-medium rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-colors"
          >
            Messages & Queue
          </a>
          <a
            href="#keys"
            className="flex items-center px-3 py-2 text-sm font-medium rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-colors"
          >
            API Keys & SMTP
          </a>
          <a
            href="#devices"
            className="flex items-center px-3 py-2 text-sm font-medium rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-colors"
          >
            LAN Devices
          </a>
          <a
            href="#audit"
            className="flex items-center px-3 py-2 text-sm font-medium rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-colors"
          >
            Audit Logs
          </a>
        </nav>

        {/* Workspace Body */}
        <main className="flex-1 overflow-y-auto p-8">
          <div className="max-w-5xl mx-auto">
            <div className="flex items-center justify-between pb-6 border-b border-slate-800">
              <div>
                <h1 className="text-2xl font-bold text-white">
                  {activeTenant?.name || 'Active Tenant'}
                </h1>
                <p className="text-sm text-slate-400 mt-1">
                  Tenant ID: <code className="text-slate-300">{activeTenant?.id}</code>
                </p>
              </div>
              <div className="flex items-center space-x-2">
                <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
                <span className="text-sm text-emerald-400 font-medium">Healthy</span>
              </div>
            </div>

            {/* Quick Metrics */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mt-6">
              <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                <div className="text-xs text-slate-400 uppercase font-semibold">Messages Today</div>
                <div className="text-2xl font-bold text-white mt-2">0</div>
                <div className="text-xs text-slate-500 mt-1">Ready for sending</div>
              </div>
              <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                <div className="text-xs text-slate-400 uppercase font-semibold">Queue Depth</div>
                <div className="text-2xl font-bold text-white mt-2">0</div>
                <div className="text-xs text-slate-500 mt-1">Workers idle</div>
              </div>
              <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                <div className="text-xs text-slate-400 uppercase font-semibold">Bounce Rate</div>
                <div className="text-2xl font-bold text-emerald-400 mt-2">0.0%</div>
                <div className="text-xs text-slate-500 mt-1">Optimal</div>
              </div>
              <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl">
                <div className="text-xs text-slate-400 uppercase font-semibold">Verified Domains</div>
                <div className="text-2xl font-bold text-white mt-2">0</div>
                <div className="text-xs text-slate-500 mt-1">Add domain in Phase 2</div>
              </div>
            </div>

            {/* Status Panel */}
            <div className="mt-8 bg-slate-900 border border-slate-800 rounded-xl p-6">
              <h2 className="text-lg font-semibold text-white">Phase 1 Foundation Operational</h2>
              <p className="text-sm text-slate-400 mt-2">
                Multi-tenancy kernel, envelope encryption (AES-256-GCM), Argon2id authentication,
                prefixed ULIDs, and tenant-scoped repository isolation are active and validated.
              </p>
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}
