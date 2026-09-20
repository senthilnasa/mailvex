# Mailvex — Multi-Tenancy Architecture

## 1. Hierarchy & Scoping Boundaries

```text
Platform
└── Organization            (Account owner / Legal entity / Quota & Billing)
    └── Tenant              (Strict isolation boundary for all mail infrastructure)
        ├── Domains
        ├── API Keys & SMTP Credentials
        ├── Message Streams (transactional, broadcast)
        ├── Messages, Recipients & Events
        ├── Suppressions
        ├── Webhooks
        ├── Inbound Routes & Mailboxes
        └── LAN Devices
```

### 1.1 Tenant Isolation Rules
- **Repository Scoping**: Every database query on a tenant-owned resource MUST include `tenant_id = ?` in its `WHERE` clause.
- **Tenant Context**: In the API and Core services, the active `tenant_id` is passed via `context.Context` and enforced by `core/repository`.
- **Zero Cross-Tenant Leakage**: Attempting to read or mutate another tenant's entity (e.g. referencing a domain or API key belonging to another tenant) returns HTTP 404 or an authorization failure. Automated cross-tenant tests run continuously in CI to verify this.

### 1.2 System Tenant
- During installation, a dedicated `tenant_system` is automatically created.
- It owns platform host domains and handles internal transactional emails (user invitations, password resets, system alerts) via direct MX or smart-host relay (`SYSTEM_MAIL_MODE=relay`).
