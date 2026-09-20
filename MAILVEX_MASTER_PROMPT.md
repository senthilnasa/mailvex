# Mailvex — Multi-Tenant, Self-Hosted Email Infrastructure Platform

## Master Implementation Prompt for an AI Coding Agent

**Project name:** Mailvex
**Repository:** `mailvex/mailvex` (monorepo)
**Primary goal:** Build a secure, multi-tenant, self-hosted email infrastructure platform providing REST API + SMTP submission + webhooks + domain authentication + full deliverability tooling (bounces, complaints, unsubscribe, warmup) + delivery observability + IP management + inbound email, with a modern administrator console and Docker-first deployment.

Mailvex provides functionality similar to AWS SES, Mailgun, SendGrid and Postmark, but is designed primarily for self-hosting. It must be suitable for internal institutional use (many applications inside one organization), multiple independent tenants, public open-source deployment, single-server installations and large multi-node installations. Do not design it around any single customer. The product must remain generic and configurable.

---

## 0. HOW TO USE THIS PROMPT

This file is the **single source of truth** for the project. It is deliberately large. Do not attempt to implement it in one pass.

1. On the first run, produce the architecture artifacts listed in §36 (Starting Task) and write them into `docs/`. The architecture document `docs/architecture.md` becomes the shared reference for every later phase.
2. Then execute the phases in §35 **strictly in order**. Each phase should be run as its own session/prompt that says: *"Read `MAILVEX_MASTER_PROMPT.md` and `docs/architecture.md`, then implement Phase N."*
3. Never begin a phase until the acceptance criteria of the previous phase pass.
4. Anything listed under **§2 Non-goals for v1** must not consume implementation effort. Leave interfaces so it can be added later.
5. Follow the development rules in §37 for every change.

Whenever a section says **DECISION**, the decision is final and must not be re-litigated during implementation. Whenever a section says **EVALUATE**, produce a short written comparison in `docs/decisions/` (ADR format) and then implement the chosen option.

---

## 1. ROLE

You are a Principal Software Architect, Senior Go Engineer, Senior React/TypeScript Engineer, SMTP/MTA and Deliverability Engineer, Database Architect, Security Engineer and DevOps Engineer.

You are responsible for designing and implementing a production-ready, open-source, self-hosted email infrastructure platform called **Mailvex**.

The platform must support:

- Multi-tenancy with strict isolation
- Multiple organizations and tenants
- Multiple sending domains per tenant
- SMTP submission and REST API sending
- Message streams (transactional vs broadcast)
- Multiple public IPs, IP pools, dedicated and shared IPs, IP warmup
- DKIM, SPF, DMARC, MTA-STS, DANE, TLS-RPT
- Bounce handling (synchronous and asynchronous DSN)
- Complaint feedback loops (ARF, Yahoo FBL, Microsoft JMRP/SNDS, Google Postmaster Tools)
- Unsubscribe handling (List-Unsubscribe, RFC 8058 one-click, hosted page)
- Suppression management
- Templates
- Webhooks
- Message tracking with bot filtering
- Delivery logs and a premium message inspector
- Inbound email with filtering and authentication checks
- Full management API, OpenAPI, SDKs and CLI
- Full administrative console
- Docker deployment, single-server and cluster mode

---

## 2. SCOPE CONTROL

### 2.1 In scope for v1

Everything in this document that is not listed below.

### 2.2 Non-goals for v1 (DECISION)

Do **not** build these in v1. Leave clean interfaces or stubs where indicated.

| Item | v1 position |
|---|---|
| Smart reputation-aware routing algorithm | Interface only (`Router` interface, `fixed`/`round_robin`/`least_used`/`weighted` strategies implemented) |
| Ed25519 DKIM | Schema field `algorithm` present; only RSA implemented |
| Google Cloud Storage backend | Storage abstraction only; implement Local FS and S3-compatible (covers R2/MinIO) |
| SDKs in four languages | Ship Go SDK + CLI (`mailvex`) generated/hand-written from OpenAPI. Node/PHP/Python SDKs are post-v1; provide OpenAPI so they can be generated |
| Terraform provider | Post-v1; management API must make it possible |
| Kubernetes/Helm chart | Optional stretch in Phase 14; Docker Compose is the primary deployment |
| Full-text search backend (OpenSearch/Meilisearch) | Adapter interface only; v1 uses MariaDB FULLTEXT |
| Billing/payment integration | Usage metering and quota enforcement only; no payment provider |
| Visual drag-and-drop template builder | Code editor + preview only |
| WYSIWYG anything | No |

### 2.3 Scope discipline

If the agent finds itself building something not in this document, stop and add it to `docs/backlog.md` instead.

---

## 3. SERVICE ARCHITECTURE

Mailvex is a monorepo producing seven deployable/consumable units plus shared infrastructure.

```text
mailvex/
├── core/        Mailvex Core      — shared Go kernel: domain model, repositories, queue,
│                                    outbox, crypto, scheduler/housekeeping jobs
├── api/         Mailvex API       — Go HTTP service: public /v1 API, management API,
│                                    console BFF, tracking/unsubscribe endpoints
├── smtp/        Mailvex SMTP      — SMTP edge: authenticated submission (587/465/25)
│                                    and inbound MX receiver; hands messages to Core
├── console/     Mailvex Console   — React/TypeScript/Vite/Tailwind administrator UI
├── worker/      Mailvex Worker    — Go outbound delivery engine: queue consumer,
│                                    routing, MX resolution, SMTP client, retries, DSN/FBL
│                                    ingestion, DNS checks, analytics rollups
├── webhooks/    Mailvex Webhooks  — Go webhook dispatcher: signing, delivery, retries,
│                                    replay, auto-disable
├── sdk/         Mailvex SDK       — Go SDK, `mailvex` CLI, OpenAPI spec, SDK generation config
├── deploy/      Docker Compose files, reverse proxy config, optional Helm
├── docs/        Documentation and ADRs
└── migrations/  Versioned SQL migrations
```

### 3.1 Service responsibilities (DECISION)

**Mailvex Core** — A Go module imported by API, SMTP adapter, Worker and Webhooks. Contains: entities, repository interfaces and MariaDB implementations, queue abstraction (Redis Streams implementation), transactional outbox, envelope encryption, ID generation, tenant-scoping middleware for repositories, template engine, MIME builder/parser, suppression checks, rate limiter, event emitter. Also compiles to the `mailvex-scheduler` binary that runs periodic jobs (DNS re-verification, outbox relay, retention purge, partition management, rollups, DNSBL checks, certificate renewal, warmup ramp advancement). No business logic may live outside Core or Worker.

**Mailvex API** — HTTP only. Never delivers mail. Validates, authenticates, rate-limits, writes message + outbox row in one transaction, returns. Also serves: management API, console backend, tracking pixel/redirects, hosted unsubscribe page, RFC 8058 one-click endpoint, DMARC/TLS-RPT/ARF inbound processing endpoints (for reports arriving via HTTP), health/metrics.

**Mailvex SMTP** — The SMTP edge. See §4. It authenticates submissions, enforces domain/sender authorization, and hands the message to Core's ingestion API. It also acts as the inbound MX receiver for inbound routes, the bounce/return-path domain (async DSN intake) and FBL/postmaster/abuse mailboxes. **It does not perform outbound delivery and its outbound queue must be disabled.**

**Mailvex Worker** — The single owner of outbound delivery: queue consumption, tenant fairness scheduling, suppression check, routing, IP selection and source-IP binding, MX resolution and MX-based grouping, per-destination concurrency, MTA-STS/DANE/TLS policy, DKIM signing, header hygiene, SMTP client, response classification, retry scheduling, dead-lettering, event emission. Also hosts background consumers: DSN parser, ARF/FBL parser, DMARC/TLS-RPT report parser, DNS check jobs, analytics rollups.

**Mailvex Webhooks** — Consumes `webhook.delivery` stream, signs and delivers payloads, records attempts, applies retry policy, exposes replay, auto-disables failing endpoints, sends test events.

**Mailvex Console** — The UI. Talks only to Mailvex API. No business logic.

**Mailvex SDK** — `openapi.yaml` (source of truth), Go client, `mailvex` CLI, examples.

### 3.2 Delivery ownership (DECISION)

There is exactly **one** queue, **one** retry system and **one** MX/SMTP-client implementation, and it lives in Mailvex Worker. The SMTP edge is an ingestion and reception adapter only. This resolves the conflict between "use the MTA for outbound" and "workers speak SMTP themselves": workers speak SMTP; the edge does not relay.

### 3.3 Data flow

```text
 SMTP client ──► Mailvex SMTP ─┐
                               ├─► Core.Ingest() ─► DB (message + outbox) ─► Redis Streams ─► Mailvex Worker ─► Internet MX
 API client  ──► Mailvex API ──┘                                                     │
                                                                                     ├─► message_events ─► Mailvex Webhooks ─► tenant endpoints
 Internet MX ──► Mailvex SMTP (inbound / bounce domain / FBL) ─► Core.IngestInbound() ─► Worker parsers ─► events/suppressions
```

---

## 4. SMTP EDGE (Mailvex SMTP)

Do NOT implement the SMTP protocol from scratch.

**EVALUATE** (write ADR `docs/decisions/0002-smtp-edge.md`) the following for the edge role only (submission + inbound, no outbound):

| Option | Notes |
|---|---|
| Haraka | Modular, plugin hooks, AUTH, TLS. Adds a Node.js runtime to a Go stack. Outbound must be disabled. |
| KumoMTA | Rust, high-performance, Lua policy. Strong outbound (which we don't use at the edge); evaluate its receiving/policy story. |
| Postfix | Extremely mature. Integrate via policy daemons/milter and SASL against Mailvex credentials. Hardest to make tenant-aware. |
| Go library (e.g. `emersion/go-smtp`) | Mature protocol library, single runtime with the rest of the stack, full control over AUTH/tenant mapping. Not an MTA, but the edge doesn't need MTA features. |

Selection criteria, in order: security track record, ease of tenant-aware AUTH and sender authorization, single-runtime preference, TLS/STARTTLS/SMTPS support, inbound MIME robustness, operational simplicity in Docker.

Whatever is chosen, it must sit behind a stable internal contract:

```text
SubmissionAdapter
  Authenticate(username, password, remoteIP) -> Credential | error
  AuthorizeSender(credential, mailFrom, rcptTo[]) -> allow | deny(reason)
  Ingest(credential, rawMIME, envelope) -> messageID | error

InboundAdapter
  RouteRecipient(rcptTo) -> InboundRoute | BounceDomain | FBLMailbox | reject
  IngestInbound(route, rawMIME, envelope, authResults) -> inboundID
```

Requirements for the edge:

- Ports 25 (inbound MX and legacy submission), 465 (SMTPS), 587 (STARTTLS submission). Prefer authenticated submission on 587.
- SMTP AUTH PLAIN/LOGIN over TLS only. Never accept AUTH on plaintext.
- **Never permit open relay.** Unauthenticated sessions may only deliver to inbound routes/bounce/FBL/postmaster addresses on Mailvex-hosted domains.
- SMTP AUTH brute-force protection: per-IP and per-username failure counters, exponential backoff, temporary block, audit event, alert. Same standard as dashboard login (§7).
- Per-tenant and per-credential concurrency and message-rate limits enforced at the edge.
- Message size limit (default 25 MB, configurable per tenant), recipient count limit, header count/size limits.
- Reject sessions that violate authorization with clear enhanced status codes (5.7.1 etc.).
- Inbound: SPF, DKIM, DMARC verification with results attached as `Authentication-Results`; ARC sealing when Mailvex forwards mail; spam/virus hook (rspamd/ClamAV, optional, off by default); size limits; greylisting optional.
- SMTP-submitted messages must enter the **same internal pipeline** as API messages (§13). No separate business logic.

---

## 5. TECHNOLOGY STACK

### Backend

- Go 1.24+
- Clean/Hexagonal architecture: `domain` → `application` → `adapters` (http, smtp, mysql, redis, s3, fs)
- Context propagation everywhere
- Structured JSON logging (`log/slog`)
- OpenTelemetry instrumentation (traces + metrics), Prometheus exposition
- Libraries chosen by the agent must be mature, actively maintained, permissively licensed and recorded in `docs/third-party.md`. Do not add frameworks that hide control flow.

### Frontend (Mailvex Console)

- React 18+, TypeScript strict, Vite, Tailwind CSS
- Accessible headless component base (e.g. Radix primitives) with a Mailvex design system on top (§28)
- TanStack Query for data, TanStack Table for tables, React Router
- Desktop-first, responsive, dark/light mode, keyboard navigable
- Design references: AWS console density, Stripe dashboard clarity, Cloudflare structure, Postmark-style observability. Do not copy proprietary designs.

### Data

- MariaDB 11+ (MySQL 8 compatible where cheap; MariaDB is primary target)
- Redis 7+ (queues, rate limiting, caches)
- Object storage: local FS or S3-compatible

### Deployment

- Docker, Docker Compose (single-server), external DB/Redis for cluster mode
- Reverse proxy: Caddy preferred (built-in ACME) or Traefik. EVALUATE in ADR 0003.

---

## 6. TENANCY AND ORGANIZATION MODEL (DECISION)

The hierarchy is defined **once**, here, and used everywhere.

```text
Platform
└── Organization            (legal/customer entity; owns users and billing/quota plans)
    └── Tenant              (isolation boundary; owns all sending resources)
        ├── Domains
        ├── API Keys
        ├── SMTP Credentials
        ├── Message Streams
        ├── Templates
        ├── IP Pool assignments
        ├── Webhooks
        ├── Messages / Events
        ├── Suppressions
        ├── Inbound Routes
        └── Settings
```

- **Organization** = who the customer is. Contains users, SSO configuration, plan/quotas, invoices-in-future. An organization has one or more tenants (e.g. `production`, `staging`, or one per department).
- **Tenant** = the isolation and scoping unit. **`tenant_id` is the column that scopes every tenant-owned query.** Nothing sending-related attaches to an organization directly.
- **User** belongs to exactly one organization (SSO or local) and has zero or more `tenant_memberships`, each with roles. Platform admins are users with a platform-level role and no tenant membership requirement.
- The Console switcher is labelled **"Tenant"** and lists the tenants the user is a member of, grouped by organization when the user belongs to more than one organization (platform admins only).
- Switching tenant refreshes all tenant-scoped state; a stale `tenant_id` must never survive a switch (clear query caches, re-read from URL/context).

### 6.1 Isolation rules

A tenant must NEVER read or use another tenant's messages, domains, API keys, SMTP credentials, webhooks, analytics, templates, suppressions, inbound mail or audit logs.

- Every repository method for tenant-owned tables takes `tenantID` as a required parameter and appends `WHERE tenant_id = ?`. Provide a `TenantScopedRepo` wrapper and a linter/test that fails if any query on a tenant-owned table lacks the predicate.
- Isolation is enforced in Core (service/repository), never only in Console or API handlers.
- Automated cross-tenant tests are mandatory from Phase 1 and run in CI on every change.

### 6.2 Identifiers

Internal primary keys: ULID (26-char, stored as `BINARY(16)` or `CHAR(26)`; choose one in ADR 0004 and keep it). Public identifiers are prefixed:

```text
org_xxxxx  tenant_xxxxx  user_xxxxx  dom_xxxxx  msg_xxxxx  rcpt_xxxxx  evt_xxxxx
key_xxxxx  smtp_xxxxx  wh_xxxxx  whd_xxxxx  ip_xxxxx  pool_xxxxx  tpl_xxxxx
sup_xxxxx  route_xxxxx  in_xxxxx  stream_xxxxx  node_xxxxx  req_xxxxx
```

Never expose sequential integers publicly.

### 6.3 Plans, quotas and usage metering

Optional, off by default (single-institution installs). Per organization or tenant:

```text
plan_id, messages_per_day, messages_per_month, max_domains, max_api_keys, max_webhooks,
max_recipients_per_message, max_message_size, dedicated_ip_allowed, streams_allowed
```

Record `usage_counters` (hourly) for messages accepted/delivered/inbound/storage bytes, exportable as CSV/JSON for chargeback. Quota exceeded → `429 quota_exceeded` with `Retry-After`.

---

## 7. AUTHENTICATION

### 7.1 Console authentication

- Email + password (Argon2id), password policy (min length 12, breached-password check optional via local list, no composition rules beyond length + deny common), password history (last 5).
- Session cookies: `HttpOnly`, `Secure`, `SameSite=Lax`, server-side session store in Redis with revocation; optional short-lived JWT + refresh only if needed for SDK/console parity — prefer sessions.
- Login rate limiting (per IP, per account), progressive delays, account lock with self-service unlock via email, audit on every event.
- MFA: TOTP (RFC 6238) and WebAuthn/passkeys. Recovery codes (10, single-use, hashed). Platform admins may be forced to MFA by policy. Store in `mfa_factors`.
- Email verification on signup/invite; password reset via single-use, 30-minute tokens (`password_resets`).
- User invitations (`invitations`): invite by email to an organization with tenant roles; expiring token; audit.
- Session listing and revocation in user settings ("sign out other sessions").

### 7.2 SSO

Institutions expect SSO. Implement per organization:

- OIDC (works with Google Workspace, Microsoft Entra ID, Keycloak, Authentik). Required in v1.
- SAML 2.0 SP (Entra, Okta, Shibboleth). Required in v1 via a mature library.
- LDAP/AD bind. Optional adapter; implement if time allows in Phase 12.
- Domain-based auto-join: users with `@university.edu` join organization X with a default role.
- Just-in-time provisioning, attribute → role mapping, SSO-only enforcement per organization (disables password login).

### 7.3 API authentication

API keys (§8). Never store secrets in plaintext: store `sha256(secret)` for lookup and Argon2id hash for verification is unnecessary — a random 256-bit secret hashed with SHA-256 is sufficient; document why. Secrets that must be recovered (SMTP passwords are **not** recoverable — store hash only; DKIM private keys and webhook secrets **are**) use envelope encryption (§23.1).

### 7.4 Platform admin impersonation ("View as tenant")

- Platform Owner/Admin can enter a tenant read-only or read-write with an explicit reason.
- Every impersonated request is audited with `actor`, `impersonated_tenant`, `reason`, and shown with a persistent banner in the Console.
- Support role sees message metadata but body content is masked unless a tenant grants time-boxed access.

### 7.5 Bootstrap system mail (DECISION)

Mailvex must send its own mail (password resets, invitations, alerts) before any customer domain is verified. Solve this with a **system tenant** (`tenant_system`) created by the installer, owning the platform hostname's domain (e.g. `mail.mailvex.example`) with DKIM/SPF/DMARC instructions shown in the wizard. Until verified, system mail is queued and the Console shows a blocking "System mail domain not verified" warning. Alternative: configure an external SMTP relay for system mail in `.env` (`SYSTEM_MAIL_MODE=relay`). Both paths must exist.

---

## 8. API KEYS, SMTP CREDENTIALS AND MESSAGE STREAMS

### 8.1 API keys

```text
mvx_live_xxxxxxxxxxxxxxxxxxxxxxxx
mvx_test_xxxxxxxxxxxxxxxxxxxxxxxx
```

Restrictions: tenant (always), stream, allowed sending domains, allowed source IP CIDRs, permission scopes (`send.email`, `messages.read`, `suppressions.write`, ...), rate limit, expiry. Show the secret once. Store prefix + last 4 for display. Revocation is immediate (cache invalidation via Redis pub/sub).

**Test keys vs Development mode (DECISION):** They are the same mechanism. A tenant has an environment flag per key: `mvx_test_` keys always operate in Development mode (no external delivery, everything else identical) regardless of tenant setting. `mvx_live_` keys deliver externally unless the **tenant** is switched to Development mode, in which case they also do not deliver and the API response includes `"environment": "development"`. There is no third state.

### 8.2 SMTP credentials

Username `smtp_xxxxx`, generated password shown once, stored as Argon2id hash. Restrictions identical to API keys (domains, source CIDRs, stream, rate). Support 25/465/587; recommend 587 STARTTLS. Rotate/revoke/regenerate with audit.

### 8.3 Message streams (DECISION, Postmark-style)

Each tenant has streams; two are created by default:

| Stream | Purpose | Defaults |
|---|---|---|
| `transactional` | Receipts, password resets, alerts | No List-Unsubscribe by default, open/click tracking off, highest queue priority, `Precedence` not set |
| `broadcast` | Newsletters, announcements | List-Unsubscribe + one-click mandatory, unsubscribe suppression enforced, tracking configurable, lower queue priority, per-stream sending rules |

Streams can be added (e.g. `marketing`, `alerts`). Each stream has: default IP pool, priority class, tracking settings, unsubscribe policy, suppression scope (shared with tenant or stream-local), retention override, webhook filter. A message specifies `stream` (default `transactional`). Suppression lists for `unsubscribe` type are per stream; `hard_bounce` and `complaint` are per tenant.

---

## 9. RBAC

Permission-based access control. Roles are bundles of permissions; **`role_permissions` links them** (this table is mandatory). Roles can be platform-level or tenant-level; custom roles allowed per organization.

Built-in roles:

| Role | Scope | Summary |
|---|---|---|
| Platform Owner | platform | Everything, including other platform admins |
| Platform Admin | platform | Everything except managing Platform Owners |
| Platform Support | platform | Read-only across tenants, bodies masked, impersonation read-only |
| Organization Owner | organization | Manage org, SSO, users, create tenants |
| Tenant Owner | tenant | Everything within tenant |
| Tenant Admin | tenant | Domains, keys, credentials, webhooks, templates, users, sending settings |
| Developer | tenant | Keys, credentials, templates, webhooks, message logs |
| Analyst | tenant | Read-only messages, statistics, reports, domains, IP health |
| Support | tenant | Read messages and logs; secrets and bodies masked |

Permission catalogue (seed via migration):

```text
tenants.read tenants.write tenants.suspend
org.read org.write org.sso.write
users.read users.write users.invite roles.read roles.write
domains.read domains.write domains.verify domains.dkim.rotate domains.delete
streams.read streams.write
api_keys.read api_keys.create api_keys.revoke
smtp_credentials.read smtp_credentials.create smtp_credentials.revoke
ip.read ip.write pools.read pools.write nodes.read nodes.write nodes.drain
messages.read messages.body.read messages.retry messages.export messages.cancel
suppressions.read suppressions.write suppressions.import
templates.read templates.write templates.publish
webhooks.read webhooks.write webhooks.replay webhooks.secret.rotate
inbound.read inbound.write
analytics.read reports.read
audit.read
settings.read settings.write
retention.write privacy.erase privacy.export
impersonate.read impersonate.write
system.read system.write
```

All elevated/platform actions are audited. Privilege escalation tests are part of the security test suite.

---

## 10. DOMAIN MANAGEMENT

### 10.1 Lifecycle

```text
pending → verifying → verified → (degraded) → suspended | disabled
```

`degraded` is a new state: previously verified, DNS currently failing, inside grace period (default 72 h, configurable). Sending continues with a warning. After grace period → `suspended` (sending rejected with `domain_not_verified`), notification sent at each transition.

### 10.2 Ownership verification and conflicts (DECISION)

- Adding a domain generates a per-tenant ownership token: `TXT _mailvex-verify.<domain>` = `mailvex-verify=<random>`. Ownership is proven only when this TXT exists. SPF/DKIM/DMARC checks are separate health checks.
- **Global uniqueness:** a verified domain may be owned by only one tenant platform-wide. A tenant may not add a domain that is a subdomain of a domain verified by a different tenant unless the parent tenant explicitly delegates it (`domain_delegations`). Platform admins can override with audit.
- A second tenant adding an already-verified domain sees `domain_claimed_by_another_tenant` (does not reveal which tenant).
- Periodic re-verification (scheduler, default every 6 h; every 15 min while `verifying` or `degraded`).

### 10.3 DNS records generated per domain

- Ownership TXT (above)
- SPF (`v=spf1 include:<platform spf host> -all` recommended; show merge instructions if an SPF record already exists)
- DKIM `selector._domainkey` TXT (or CNAME to Mailvex-hosted key when `DKIM_HOSTED_MODE=cname`)
- DMARC `_dmarc` TXT with recommended progression `p=none` → `quarantine` → `reject`, including `rua=` pointing to Mailvex's DMARC report address
- Return-path / bounce domain: `CNAME bounce.<domain> → <platform bounce host>` (or MX record if tenant hosts it) — used for VERP (§17.2)
- Tracking domain: `CNAME track.<domain> → <platform tracking host>` (optional)
- MTA-STS (for inbound domains): `_mta-sts` TXT + hosted policy; TLS-RPT `_smtp._tls` TXT with `rua=` to Mailvex

### 10.4 DNS verification engine

Checks: ownership TXT, SPF, DKIM (per selector), DMARC, MX, A/AAAA, CNAME (tracking, bounce), PTR (for IPs), MTA-STS, TLS-RPT, TLS certificate for tracking domain, propagation across ≥2 resolvers (system + 1.1.1.1/8.8.8.8 configurable; never hard-code only one).

Results: `PASS | WARNING | FAIL | PENDING | UNKNOWN` with structured explanation:

```text
DKIM verification failed.
Expected:   s1._domainkey.example.com TXT "v=DKIM1; k=rsa; p=MIIB..."
Detected:   No TXT record found (resolvers: 1.1.1.1, 8.8.8.8)
Action:     Create the TXT record shown below, wait for propagation, then click Verify.
```

### 10.5 DKIM

- RSA 2048 default (1024 refused, 4096 allowed). `algorithm` column present for future Ed25519.
- Multiple selectors per domain; states `pending | active | retiring | retired`. Rotation flow: create new selector → verify DNS → activate → keep old selector signing-capable for 7 days → retire.
- Private keys encrypted with envelope encryption (§23.1), never returned by any API, never logged. Export only via platform-admin backup tool with re-encryption.
- Signing occurs in Mailvex Worker immediately before SMTP transmission (after header hygiene). Sign headers: `From, To, Cc, Subject, Date, Message-ID, Reply-To, MIME-Version, Content-Type, List-Unsubscribe, List-Unsubscribe-Post, Feedback-ID`; oversign `From`.
- Optional hosted-CNAME mode so Mailvex controls rotation without tenant DNS edits.

### 10.6 SPF

Validation, syntax inspection, 10-lookup limit warning, `include` detection, `+all` warning, alignment display, merge guidance. Never modify user DNS automatically.

### 10.7 DMARC

Discovery, parsing, policy display, alignment status, recommendations. Aggregate (RUA) report ingestion via inbound mailbox `dmarc-reports@<platform>` and via HTTP upload; parse XML (including gzip/zip), store in `dmarc_reports` + `dmarc_report_records`, dashboard per domain (sources, pass/fail, alignment). Forensic (RUF) reports accepted but body content masked by default.

### 10.8 TLS-RPT

Ingest RFC 8460 JSON reports at `tlsrpt@<platform>` (and HTTPS endpoint); store `tlsrpt_reports`; surface failures per domain alongside DMARC.

### 10.9 Custom tracking and bounce domains — certificates

Per-tenant tracking domains need on-demand TLS certificates. Use the reverse proxy's on-demand ACME (Caddy `on_demand_tls` with an `ask` endpoint in Mailvex API that returns 200 only for verified tracking domains) or Traefik's equivalent. Certificates for the platform hostnames (console, API, SMTP EHLO hostnames) are also ACME-managed; SMTP daemon reloads certificates on renewal without dropping sessions. Expiry monitored (§22).

---

## 11. IP MANAGEMENT

### 11.1 IP addresses

Support 10–20+ public IPv4 (and IPv6) addresses.

```text
ip_addresses: id, address, family, node_id, ehlo_hostname, ptr_expected, ptr_observed,
              fcrdns_ok, provider, location, status, pool memberships, dedicated_tenant_id (nullable),
              reputation_state, warmup_plan_id, warmup_day, last_checked_at
status: available | warming | active | paused | suspended | retired
```

Checks: PTR, FCrDNS (A of PTR must include the IP), DNSBL (pluggable, §22.2), SMTP connectivity from the IP, TLS handshake, `EHLO` hostname == PTR.

### 11.2 Source IP binding and networking (DECISION)

- Every IP is bound to exactly one **SMTP node** (`smtp_nodes.id`) — a Mailvex Worker host. Workers only select IPs bound to their own `node_id`.
- Each IP has its own `ehlo_hostname` that **must** match its PTR; the Worker uses it in `EHLO` and binds the local address on the outbound socket (`net.Dialer.LocalAddr`).
- Docker networking: the default bridge cannot present multiple public source IPs. Worker containers run with `network_mode: host` (recommended) or a `macvlan` network with all public IPs attached; document both in `docs/networking.md`, and provide a `deploy/docker-compose.multi-ip.yml` example. The installer must detect bridge mode and warn that only one source IP will work.
- Health checker verifies each IP is actually locally bound (`ip addr`) on its node and marks it `unbound` otherwise.

### 11.3 IP pools

Default pools: `default, transactional, critical, bulk, low_volume`. Membership many-to-many with weights. Strategies: `fixed | round_robin | least_used | weighted` behind a `Router` interface (smart routing is a non-goal).

Routing: `tenant policy → stream default pool → domain override → pool → strategy → IP on this node`. Pools may be shared or dedicated; a dedicated pool contains IPs with `dedicated_tenant_id` set and is usable only by that tenant.

### 11.4 Shared-pool protection

A tenant whose rolling bounce rate or complaint rate exceeds thresholds (default bounce > 5 %, complaint > 0.1 %, configurable per pool) is automatically moved from shared pools to a `quarantine` pool (or paused if none) with alert + audit, until an admin releases it. This protects other tenants' reputation.

### 11.5 IP warmup

`warmup_plans` define daily volume ramps and per-provider caps:

```text
day 1:  50   (gmail 20, microsoft 20, yahoo 10, other unlimited-within-total)
day 2:  100
day 3:  200
...     doubling to plateau, configurable, default 30-day plan
```

An IP in `warming` state may only send up to its daily allotment; overflow is routed to other `active` IPs in the pool or deferred (`message.deferred` reason `warmup_cap`). Scheduler advances the day only if the previous day's bounce/complaint rates were acceptable, else holds. Console shows warmup progress per IP.

---

## 12. SENDING API (Mailvex API)

### 12.1 Send

```http
POST /v1/email
Authorization: Bearer mvx_live_...
Idempotency-Key: <client unique>
```

```json
{
  "from": "Billing <noreply@example.com>",
  "to": ["user@example.com"],
  "cc": [], "bcc": [],
  "reply_to": "support@example.com",
  "subject": "Hello",
  "text": "Hello",
  "html": "<h1>Hello</h1>",
  "attachments": [{"filename": "a.pdf", "content_type": "application/pdf", "content": "<base64>", "content_id": null}],
  "headers": {"X-Campaign": "sept"},
  "tags": ["receipt"],
  "metadata": {"order_id": "123"},
  "template_id": null,
  "template_variables": {},
  "stream": "transactional",
  "track_opens": null, "track_clicks": null,
  "scheduled_at": null,
  "unsubscribe_url": null
}
```

Response `202`:

```json
{"id": "msg_xxxxx", "status": "queued", "environment": "live", "request_id": "req_xxxxx"}
```

### 12.2 Batch send

```http
POST /v1/email/batch
```

Up to 500 messages per call; shared fields at top level, per-recipient overrides (`template_variables`, `metadata`, `to`). Each item returns its own `id` or `error`. Whole batch is one idempotency unit. Enforced per-recipient variable substitution; one recipient's failure never fails the others.

### 12.3 Read endpoints

```http
GET  /v1/messages?limit=&cursor=&status=&recipient=&from=&stream=&tag=&since=&until=
GET  /v1/messages/{id}
GET  /v1/messages/{id}/events
GET  /v1/messages/{id}/recipients
GET  /v1/messages/{id}/body            (requires messages.body.read; honors retention)
POST /v1/messages/{id}/retry
POST /v1/messages/{id}/cancel          (only if still queued/scheduled)
```

### 12.4 Limits (defaults, all configurable per tenant/plan)

| Limit | Default |
|---|---|
| Max message size (encoded) | 25 MB |
| Max attachment total | 20 MB |
| Recipients per message (to+cc+bcc) | 50 |
| Batch items | 500 |
| Custom headers | 20, allowlisted names, `X-*` only plus `List-Id`, `Precedence`; **never** `From`, `To`, `Received`, `Return-Path`, `DKIM-*`, `Authentication-Results`, `Message-ID` (unless `allow_custom_message_id`) |
| Metadata | 20 keys, 1 KB total |
| Tags | 10 |
| Inline images | via `attachments[].content_id` → `cid:` references |

- Reject CR/LF anywhere in header names/values (`400 header_injection`).
- Support IDN domains and SMTPUTF8 addresses (RFC 6531); downgrade or reject per destination capability with a clear event reason.
- Validate `from` domain is verified and permitted for the key/credential; `reply_to` need not be.

### 12.5 Idempotency (DECISION)

- Key scoped to `(tenant_id, api_key_id, idempotency_key)`. TTL 24 h.
- Same key + same request body hash → return the original response (200/202 with original `id`).
- Same key + **different** body → `409 idempotency_key_reused` with the original `request_id`.
- Key in flight (first request still processing) → `409 idempotency_in_progress`.

### 12.6 Pagination convention

Cursor-based everywhere: `?limit=50&cursor=<opaque>`; response `{"data": [...], "next_cursor": "...", "has_more": true}`. Cursor encodes `(created_at, id)`; never offset pagination on large tables.

### 12.7 Management API

Everything the Console can do must be possible via `/v1/` with the same RBAC:

```text
/v1/tenants  /v1/organizations  /v1/users  /v1/invitations  /v1/roles
/v1/domains  /v1/domains/{id}/verify  /v1/domains/{id}/dkim  /v1/domains/{id}/dns
/v1/streams  /v1/api-keys  /v1/smtp-credentials
/v1/ip-addresses  /v1/ip-pools  /v1/smtp-nodes  /v1/warmup-plans
/v1/webhooks  /v1/webhooks/{id}/deliveries  /v1/webhooks/{id}/test  /v1/webhooks/{id}/rotate-secret
/v1/templates  /v1/templates/{id}/versions  /v1/templates/{id}/publish  /v1/templates/{id}/render
/v1/suppressions  (+ import/export)
/v1/inbound/routes  /v1/inbound/messages
/v1/analytics/*  /v1/reports/dmarc  /v1/reports/tlsrpt  /v1/reports/fbl
/v1/audit-logs  /v1/settings  /v1/retention  /v1/privacy/erasure  /v1/privacy/export
/v1/system/health  /v1/system/workers  /v1/system/queues  /v1/system/dlq
```

This enables the CLI now and a Terraform provider later.

### 12.8 Errors

```json
{"error": {"code": "domain_not_verified", "message": "The sending domain has not been verified.", "request_id": "req_xxxxx", "details": {}}}
```

Stable machine-readable codes documented in `docs/api/errors.md`. HTTP status mapping fixed. Rate limit headers `X-RateLimit-Limit/Remaining/Reset` and `Retry-After` on 429.

### 12.9 Versioning

`/v1/` only. Never break `/v1`. Additive changes allowed. Semantic versioning for releases; `openapi.yaml` is the contract, validated in CI against the implementation (request/response schema tests).

---

## 13. MESSAGE PIPELINE AND MODEL

### 13.1 Unified ingestion

API and SMTP both call `core.Ingest(ctx, IngestRequest)` which: authenticates context, validates, resolves stream, applies template, builds MIME (API) or parses and normalizes MIME (SMTP), checks suppressions per recipient (§18), applies quotas/rate limits, persists message + recipients + first event + **outbox row** in one DB transaction, returns. No delivery logic here.

### 13.2 Model

```text
messages:            id, tenant_id, stream_id, domain_id, source (api|smtp|system|inbound_reply),
                     api_key_id, smtp_credential_id, template_version_id, from_address, from_name,
                     reply_to, subject, message_id_header, status, priority, environment,
                     size_bytes, body_ref (storage key), raw_mime_ref, tags, metadata,
                     scheduled_at, created_at, queued_at, sent_at, completed_at, expires_at
message_recipients:  id, message_id, tenant_id, address, address_domain, mx_group, type (to|cc|bcc),
                     status, attempts, next_attempt_at, last_smtp_code, last_response, delivered_at,
                     bounced_at, suppressed_reason, verp_token
delivery_attempts:   id, recipient_id, tenant_id, attempt_no, ip_id, node_id, remote_mx, remote_ip,
                     tls_version, tls_cipher, tls_policy_applied, smtp_code, enhanced_code, smtp_response,
                     duration_ms, started_at, outcome (delivered|deferred|bounced|failed)
message_events:      id, tenant_id, message_id, recipient_id, type, occurred_at, data (JSON) — append-only
```

Message status is derived from recipient statuses and must never hide per-recipient state: `queued | sending | partially_delivered | delivered | failed | cancelled`.

Events:

```text
message.accepted message.queued message.scheduled message.sending message.sent message.delivered
message.deferred message.bounced message.failed message.complained message.suppressed
message.unsubscribed message.opened message.clicked message.cancelled message.expired
inbound.received inbound.rejected
```

### 13.3 Body storage

Configurable per tenant: `metadata_only | metadata_body | metadata_body_raw`. Bodies and attachments live in object storage (`storage` abstraction: local FS, S3-compatible; GCS interface only). Encrypt at rest with tenant data key (§23.1). Retention per class (§26.3).

---

## 14. QUEUE, OUTBOX AND SCHEDULING (Mailvex Core)

### 14.1 Queue abstraction

Interface `Queue` with `Publish`, `Consume(group)`, `Ack`, `Nack`, `Delay`, `Depth`. v1 implementation: **Redis Streams** with consumer groups. NATS JetStream adapter is a documented extension point (not implemented).

Streams:

```text
mvx:delivery:{priority}   (p0 critical, p1 transactional, p2 broadcast, p3 bulk)
mvx:retry                 (sorted-set scheduler feeds back into delivery streams)
mvx:webhook
mvx:inbound
mvx:reports               (DSN, ARF, DMARC, TLS-RPT parsing)
mvx:analytics
mvx:dns
mvx:housekeeping
```

### 14.2 Transactional outbox (DECISION)

The API/SMTP never writes to Redis directly. Within the same MariaDB transaction as the message insert, insert into `outbox (id, tenant_id, stream, payload, available_at, claimed_by, claimed_at, published_at)`. A relay in `mailvex-scheduler` (and as a fallback in Worker) polls `outbox WHERE published_at IS NULL AND available_at <= NOW()` in batches, publishes to Redis, marks published. Consumers are idempotent on `message_recipient.id + attempt_no`, so a relay crash after publish but before mark yields a duplicate publish that is safely ignored. **The database is the source of truth; Redis is a delivery hint.** A reconciler job re-enqueues recipients stuck in `queued` with no stream entry after N minutes.

### 14.3 Delayed delivery (DECISION)

Redis Streams has no native delay. Implement delays (retries, `scheduled_at`, warmup deferral) with a Redis **sorted set** `mvx:delayed` scored by `available_at` epoch; a scheduler loop (`ZRANGEBYSCORE ... LIMIT`, atomic move via Lua script) publishes due items to the correct delivery stream. Also mirrored in `message_recipients.next_attempt_at` so the reconciler can rebuild the sorted set after Redis loss.

### 14.4 Tenant fairness and priority (DECISION)

- Four priority classes map to streams (§14.1). Workers consume p0 → p3 with weighted round-robin (e.g. 8:4:2:1) so lower priorities are never starved but transactional mail never waits behind bulk.
- Within a priority class, **per-tenant fair scheduling**: the outbox relay and delayed scheduler publish in round-robin across tenants with pending work (bounded batch per tenant per tick), so a 500k broadcast from Tenant A cannot monopolize the stream ahead of Tenant B's password reset.
- Per-tenant in-flight concurrency limit (default 50, configurable) enforced with Redis counters.

### 14.5 Queue observability

Depth per stream/priority/tenant, oldest-item age, consumer lag, DLQ size, all exported as metrics and shown in Console → System → Queues.

---

## 15. DELIVERY WORKER (Mailvex Worker)

Each worker process runs N delivery goroutines and:

1. Claims a recipient job from a delivery stream.
2. Loads message + recipient; validates state (skip if cancelled/expired/suppressed since enqueue).
3. Re-checks suppression (hard_bounce, complaint, unsubscribe for the stream, manual, domain_block) — **this check ships in Phase 5, not later.**
4. Applies tenant/stream/domain policy → IP pool → `Router` → source IP **bound to this node**; respects warmup caps.
5. Resolves destination MX (with caching, NULL MX handling RFC 7505, fallback to A/AAAA, implicit MX rules).
6. Groups by **MX host** (§21.2) and acquires per-destination connection slot (default 5 concurrent connections per MX host per IP, 100 msgs per connection, configurable, provider presets for Gmail/Microsoft/Yahoo).
7. Determines TLS policy: `MTA-STS` policy fetch/cache (RFC 8461), `DANE` TLSA lookup (RFC 7672, requires DNSSEC-validating resolver; degrade gracefully), per-destination-domain override (`opportunistic | required | none`). Records `tls_policy_applied`.
8. **Header hygiene:** strip internal `Received`, `X-Mailvex-*` internal headers and private IPs; ensure `Message-ID` (generate if missing, using the sending domain), `Date`, `MIME-Version`; add `Feedback-ID: <tenant>:<stream>:<campaign>:mailvex`; add `X-Mailvex-Tenant-Id` hashed identifier for abuse tracing; add `List-Unsubscribe` + `List-Unsubscribe-Post: List-Unsubscribe=One-Click` per stream policy (§17.3); set `Return-Path` to VERP address (§17.2).
9. Applies tracking rewrites if enabled (§19).
10. DKIM-signs with the domain's active selector.
11. Connects from the bound IP with its `ehlo_hostname`, negotiates TLS per policy, transmits.
12. Records `delivery_attempts` row with full SMTP response (§13.2).
13. Classifies response (§16) → updates recipient, emits event, schedules retry via delayed set, or dead-letters.
14. Releases connection slot; reuses connection for next recipient to the same MX host when possible.

Never block API requests on delivery. Workers must be horizontally scalable and stateless apart from their node identity.

### 15.1 Retry

Default schedule (configurable per tenant/stream): 1m, 5m, 15m, 30m, 1h, 2h, 6h, 12h, 24h; max age 72 h. Jitter ±20 %. Then `failed` + `message.failed` + DLQ entry. Never retry 5xx permanent failures. Per-destination backoff when an MX returns rate-limit codes (e.g. Gmail 421/450 4.7.28) applies to the whole MX group, not just one recipient.

### 15.2 Dead letter queue

`dead_letter_messages`: recipient, reason, last response, attempts, destination, IP, node, timestamp. Console: inspect, retry, discard, suppress recipient. Bulk actions with confirmation and audit.

### 15.3 Graceful drain and shutdown (DECISION)

On `SIGTERM`, Worker stops claiming, finishes in-flight SMTP sessions (bounded by `SHUTDOWN_GRACE`, default 60 s), nacks unclaimed jobs, sets its node `draining`, exits 0. Console "Drain" action sets a flag the worker polls. API on `SIGTERM` stops accepting, finishes in-flight requests, exits. Zero-downtime upgrade guide in `docs/upgrades.md`: roll API behind proxy, drain workers one node at a time, run migrations that are backward compatible (expand → migrate → contract).

---

## 16. BOUNCE CLASSIFICATION

Classify every SMTP response and every DSN:

```text
hard_bounce           (5.1.1 user unknown, 5.1.2 bad domain, 5.1.10 ...)
soft_bounce           (4.2.2 mailbox full, 4.4.x transient)
policy_rejection      (5.7.1 not authorized, content policy)
spam_rejection        (5.7.x with spam wording, provider block messages)
authentication_rejection (DKIM/SPF/DMARC failure text)
rate_limited          (4.7.28, 421 too many, throttling text)
mailbox_unavailable   (4.2.1)
domain_unavailable    (MX resolution failure, connection refused all MX)
unknown
```

Use a **pattern table** (`bounce_rules`: pattern, enhanced code prefix, provider, classification, action) seeded with common provider responses and editable by platform admins, not a hard-coded switch. Suppression actions follow configured policy: hard_bounce → suppress immediately; soft_bounce → suppress after N (default 3) distinct messages within 7 days; rate_limited → never suppress; spam_rejection → alert + count toward shared-pool protection.

---

## 17. FEEDBACK LOOPS, ASYNC BOUNCES AND UNSUBSCRIBE

### 17.1 Complaint feedback loops (DECISION — required for v1)

Without these, complaint rate always reads zero.

- **ARF ingestion:** Mailvex SMTP accepts mail at `fbl@<platform bounce host>` and per-tenant `abuse@`/`postmaster@` on hosted domains; Worker parses ARF (RFC 5965) → identifies original message via `Feedback-ID`, `Message-ID`, or VERP token → emits `message.complained`, adds `complaint` suppression, stores `fbl_reports`.
- **Yahoo CFL:** documentation and a per-tenant/platform DKIM-domain registration checklist in the Console with the `fbl@` address to use.
- **Microsoft JMRP/SNDS:** registration checklist for each IP; JMRP reports arrive as ARF to the registered address → same parser. SNDS data: optional CSV import (manual or scheduled download with the tenant-supplied key) into `ip_reputation_snapshots`.
- **Google Postmaster Tools:** Gmail sends no ARF. Integrate the Postmaster Tools API (OAuth client credentials configured by platform admin) to pull daily spam rate, domain/IP reputation, authentication rates per verified domain into `postmaster_snapshots`; show in Domain Health and IP Reputation. Optional: off until credentials configured.
- Complaint rate = complaints / delivered over rolling 24h/7d, per tenant, domain, IP, stream. Feeds alerts and shared-pool protection.

### 17.2 Asynchronous bounces (DSN) and VERP (DECISION)

Many bounces arrive later as DSN emails to the return-path.

- Every outbound recipient gets a **VERP return-path**: `bounce+<verp_token>@<bounce domain>` where `verp_token` is an HMAC-signed compact token encoding `recipient_id`. Bounce domain is per platform by default (`bounce.<platform host>`) or per tenant via CNAME/MX (§10.3).
- Mailvex SMTP accepts mail for `bounce+*@<bounce domains>` (and rejects everything else on those domains), stores raw MIME, publishes to `mvx:reports`.
- Worker parses DSN (RFC 3464 `multipart/report; report-type=delivery-status`), extracts `Status`, `Diagnostic-Code`, `Final-Recipient`, `Action`; maps to recipient via token; classifies (§16); emits `message.bounced`/`message.deferred`; applies suppression policy. Non-standard bounces (no DSN structure) go through a heuristic parser using the pattern table; unparseable ones are stored as `unmatched_bounces` for admin review.
- Auto-replies/out-of-office are detected (`Auto-Submitted`, `X-Autoreply`, `Precedence: auto_reply`) and ignored.

### 17.3 Unsubscribe (DECISION — required for v1)

Gmail and Yahoo require it for bulk senders.

- For `broadcast`-type streams (and any stream with `unsubscribe_policy = required`), Worker adds `List-Unsubscribe: <https://<tracking or platform host>/u/<signed token>>, <mailto:unsubscribe+<token>@<bounce domain>>` and `List-Unsubscribe-Post: List-Unsubscribe=One-Click`.
- **RFC 8058 one-click endpoint:** `POST /u/<token>` with body `List-Unsubscribe=One-Click` → immediately adds `unsubscribe` suppression for `(tenant, stream, address)`, emits `message.unsubscribed`, webhook fired, returns 200 without any UI. Token is HMAC-signed, encodes `recipient_id`, expires never (or configurable) but is single-purpose.
- **Hosted unsubscribe page:** `GET /u/<token>` shows a minimal, tenant-brandable (logo, colors, text) confirmation page with one button; language auto-detected; no tracking. Optional "manage preferences" hook: tenant may specify a redirect URL instead of the hosted page.
- **mailto unsubscribe:** Mailvex SMTP accepts `unsubscribe+<token>@<bounce domain>` and processes it like one-click.
- Tenant may supply `unsubscribe_url` per message to replace the hosted URL (still signed through Mailvex redirect so the event is recorded, unless `unsubscribe_tracking=false`).
- Transactional stream: headers off by default; tenant can enable.
- Bulk-sender compliance checklist in the Console (SPF+DKIM aligned, DMARC published, one-click present, complaint rate < 0.3 %, PTR valid) with pass/fail per domain.

### 17.4 postmaster@ and abuse@

For every Mailvex-hosted inbound-capable domain and the platform domain, `postmaster@` and `abuse@` are always accepted (RFC 5321 §4.5.1). Mail to them is stored in a platform **Abuse Intake** queue with an admin workflow: view, link to tenant/message via headers, mark resolved, escalate (suspend credential/domain/tenant), reply. Notifications to platform admins.

---

## 18. SUPPRESSION SYSTEM

Types: `hard_bounce | complaint | unsubscribe | manual | domain_block | role_account (optional)`.

```text
suppressions: id, tenant_id, stream_id (nullable; unsubscribe is per stream), address (normalized, lowercase),
              address_domain, type, reason, source (smtp|dsn|fbl|one_click|api|import|admin), source_ref,
              created_at, expires_at, created_by
```

- Check is on `(tenant_id, address)` and `(tenant_id, address_domain)` for `domain_block`, plus `(tenant_id, stream_id, address)` for unsubscribe. Indexed for O(1) lookups; hot cache in Redis with invalidation.
- Suppressed recipients get `message.suppressed` with reason; the API response still returns `202` with per-recipient status visible on read.
- Console: searchable, filterable, bulk delete, CSV import/export with validation report, "why is this address suppressed" view showing the originating event.
- API `GET/POST/DELETE /v1/suppressions`, `POST /v1/suppressions/import`, `GET /v1/suppressions/export`.
- Never auto-suppress on temporary failures.

---

## 19. TRACKING

Off by default per stream; tenant enables opens/clicks separately.

- **Opens:** 1×1 pixel `GET /o/<signed token>`; **Clicks:** `GET /c/<signed token>` → 302 to target.
- **Signed URLs (DECISION):** token = base64url(`recipient_id` + `link_id` + HMAC-SHA256 with tenant tracking key). Unsigned or tampered → 404. Redirect target is read from `tracking_links` by `link_id`, never from the URL — this eliminates open-redirect abuse.
- **Bot and prefetch filtering (DECISION):** classify each hit before counting: Apple Mail Privacy Protection (Apple proxy IP ranges / UA), Gmail image proxy (`GoogleImageProxy`), Outlook/Microsoft safe-links scanners, known security scanner UAs and datacenter IP ranges (maintained list in `tracking_bot_rules`, updatable), hits < 2 s after delivery, multiple hits from the same IP across many recipients. Store every hit with `is_bot`, `bot_reason`; analytics count only `is_bot = false` by default with a toggle to show raw. Document that open rates are estimates.
- Store `message_id, recipient_id, occurred_at, user_agent, ip (truncated /24 or /48 by default — privacy setting), target_url, is_bot`.
- Custom tracking domains per tenant with on-demand certificates (§10.9). Link rewriting skips `mailto:`, `tel:`, anchors, and URLs marked `data-mvx-notrack`.

---

## 20. WEBHOOKS (Mailvex Webhooks)

Configuration: URL (https required, SSRF-protected — resolve and reject private/link-local/loopback ranges, re-validate on redirect, max 1 redirect), event filter, stream filter, secret, active, retry policy, timeout (default 10 s), custom headers (allowlisted).

Headers on every delivery:

```text
X-Mailvex-Event, X-Mailvex-Delivery-Id, X-Mailvex-Event-Id, X-Mailvex-Timestamp,
X-Mailvex-Signature: t=<unix>,v1=<hex hmac-sha256(secret, t + "." + body)>
```

- **Replay protection:** receivers should reject `|now - t| > 300 s`; documented with verification snippets in Go/Node/Python/PHP in `docs/webhooks.md`.
- **Secret rotation with overlap:** `POST /v1/webhooks/{id}/rotate-secret` creates a new secret; for 24 h deliveries carry `v1=` signatures for both old and new (`v1=...,v1=...`); old secret then expires.
- **At-least-once delivery, documented.** `X-Mailvex-Event-Id` (= `evt_xxxxx`) is the dedupe key. Payload `{ "id": "evt_...", "type": "message.delivered", "occurred_at": ..., "data": {...}, "tenant_id": ..., "api_version": "v1" }`.
- Retry schedule: 30s, 2m, 10m, 30m, 1h, 3h, 6h, 12h, 24h (configurable). **Auto-disable** after N consecutive failed deliveries (default 50 over ≥ 24 h) → status `disabled_failing`, notification, banner in Console; re-enable resumes from the oldest undelivered event within retention.
- Delivery log per webhook: attempts with request headers/body (secret redacted), response status/headers/body (truncated 16 KB), duration. Replay single or bulk, retry now.
- **"Send test event"** button and API: delivers a synthetic `webhook.test` event with sample data.
- Batching (optional per webhook): deliver up to 100 events per request as an array when enabled.
- Retention configurable.

---

## 21. INBOUND EMAIL

Separate module using the SMTP edge's inbound adapter.

```text
Internet → MX → Mailvex SMTP (inbound) → auth checks (SPF/DKIM/DMARC/ARC) → spam/virus hook
        → tenant/domain/route match → store raw MIME → parse → inbound_messages → webhook / event
```

- Routes per tenant: exact address, wildcard (`support-*@`), catch-all per domain, with actions `webhook | store_only | forward (with ARC seal) | discard`.
- Size limit (default 25 MB), attachment extraction to storage, HTML/text extraction, header capture, `Authentication-Results` stored as structured JSON and shown in the inspector.
- Spam/virus scanning hook (rspamd via milter/HTTP, ClamAV via clamd) optional; results attached; action thresholds (`tag | quarantine | reject`).
- Rejected mail (no route, over size, virus) produces `inbound.rejected` event with reason and SMTP 5xx to sender; never accept-then-drop silently except spam quarantine.
- Inbound webhook payload as specified, plus `spf`, `dkim`, `dmarc`, `spam_score`, `raw_mime_url` (signed, expiring).
- Raw MIME optionally preserved per retention.

### 21.1 Provider classification (DECISION)

Classify destination **by MX host**, not recipient domain: `*.google.com` / `aspmx.l.google.com` → `google` (covers Gmail and Workspace custom domains), `*.mail.protection.outlook.com` → `microsoft`, `*.yahoodns.net` → `yahoo`, `*.icloud.com`, `*.mail.ru`, `*.qq.com`, etc.; `mx_group` stored on the recipient. Fallback `custom`. Rule table `provider_mx_rules` editable by platform admins.

### 21.2 MX-based grouping

Connection concurrency, rate limits, backoff and analytics all key on `mx_group` (§15 step 6). Provider presets seeded for major providers; never claim provider-specific causes without SMTP evidence.

---

## 22. REPUTATION, DNSBL, ALERTS AND NOTIFICATIONS

### 22.1 IP and domain reputation

Track per IP/domain/tenant/stream: bounce rate, complaint rate, delivery rate, deferred rate, DNSBL status, PTR/FCrDNS, volume, Google Postmaster reputation (if configured), SNDS colour (if imported). States: `measured | unknown | warning | blocked`. Never label "good" without measured data over a minimum volume (default 1,000 delivered / 7 days).

### 22.2 DNSBL monitor

Pluggable `dnsbl_providers (name, query_pattern, return_code_meanings, enabled, interval)`, seeded with several public lists; results in `dnsbl_results`. Alert on new listing; show delist guidance links.

### 22.3 Alerts (`alert_rules`)

Threshold-based, scoped platform/tenant: bounce rate > X %, complaint rate > X %, queue depth > X, oldest queued > X min, webhook failures > X, IP on DNSBL, certificate expires in X days, disk > X %, SMTP node/worker unavailable, DB/Redis unhealthy, domain degraded/suspended, DKIM failure, warmup held, quota at 80/100 %. Evaluated by the scheduler; deduplicated with cooldown; resolved notifications sent.

### 22.4 Notification channels (DECISION)

`notification_channels`: `email` (via system tenant), `slack` (incoming webhook), `generic_webhook` (signed like tenant webhooks), `pagerduty` (Events API v2), `microsoft_teams`. Per rule → one or more channels. In-app notification center in Console with read/unread.

---

## 23. SECURITY

### 23.1 Master key management and envelope encryption (DECISION)

- A **Key Encryption Key (KEK)** is never stored in the database. Providers: `env` (`MVX_MASTER_KEY`, 32 bytes base64 — default for single-server), `file`, `vault` (HashiCorp Vault Transit), `aws_kms`, `gcp_kms`, `azure_keyvault`. v1 implements `env`, `file`, `vault`; cloud KMS adapters behind the same interface (implement AWS KMS if time allows in Phase 13).
- Per-tenant **Data Encryption Keys (DEK)** (AES-256-GCM) generated at tenant creation, stored encrypted by the KEK in `tenant_keys` with `key_version`. All secrets (DKIM private keys, webhook secrets, SSO client secrets, Postmaster credentials, stored message bodies at rest) are encrypted with the tenant DEK. Ciphertext records include `kek_version` and `dek_version`.
- Rotation: `mailvex keys rotate-kek` re-wraps all DEKs; `mailvex keys rotate-dek --tenant` re-encrypts that tenant's secrets in batches. Documented in `docs/security.md`. Startup fails loudly if the KEK is missing/wrong.

### 23.2 Outbound content scanning

Volume controls are not enough; a leaked credential sending low-volume phishing must be caught. Optional per platform/tenant: rspamd (HTTP `/checkv2`) and/or ClamAV hook at ingestion. Actions by score: `allow | tag | hold_for_review | reject`. Held messages appear in a platform Review Queue. Off by default; recommended on for shared installs.

### 23.3 Application security checklist (all mandatory)

- HTTPS only, HSTS, secure cookies, CSRF tokens for cookie-authenticated routes, strict CORS allowlist, security headers (CSP, X-Content-Type-Options, Referrer-Policy, Permissions-Policy).
- Input validation on every boundary; parameterized SQL only; output encoding.
- **CRLF/header injection** rejected in all header fields and SMTP envelope inputs; custom header allowlist (§12.4).
- **SSRF:** webhooks, template remote assets (disallowed), inbound forward targets, DMARC `rua` — all resolve-and-check against private ranges, with DNS rebinding protection (connect to the validated IP).
- **XSS in the message inspector (DECISION):** rendered HTML is shown only inside a sandboxed `<iframe sandbox="" srcdoc=...>` served from a separate origin/sub-path with `Content-Security-Policy: default-src 'none'; img-src data:; style-src 'unsafe-inline'`, remote content blocked by default with a per-view "load remote images" opt-in. Never inject stored HTML into the Console DOM.
- **Template injection (DECISION):** logic-less engine (Mustache-compatible via a mature Go library), HTML auto-escaping in HTML bodies, no triple-mustache unless variable is declared `html_safe` in the template schema; missing variables → configurable per template: `error` (message rejected with `template_variable_missing`) or `empty`. Variable names validated; max depth/size limits.
- **Click tracking / unsubscribe / VERP tokens** all HMAC-signed (§19, §17).
- **SMTP AUTH brute-force** protection (§4). Dashboard/API brute-force protection (§7).
- Webhook replay protection, rotation, auto-disable (§20).
- Impersonation audited, Support role body masking (§7.4).
- Secrets never logged; log scrubber tests.
- Dependency scanning (govulncheck, npm audit), SBOM (CycloneDX) in release pipeline, `SECURITY.md` with disclosure policy.
- Open-relay tests in CI (Phase 3 onwards).

### 23.4 Privacy (GDPR / DPDP-style)

- **Per-recipient erasure:** `POST /v1/privacy/erasure {address}` → removes/pseudonymizes the address in messages, recipients, events, tracking, suppressions (keeps a salted hash in suppressions so the address stays suppressed without storing it), deletes bodies addressed to them; produces an erasure report; audited. Also platform-wide erasure for platform admins.
- **Data export:** `POST /v1/privacy/export {address}` → JSON/CSV bundle of all data about a recipient within the tenant.
- **PII minimization in logs:** log `recipient_id` not addresses; IPs truncated in tracking by default; configurable log redaction of email addresses (`LOG_REDACT_EMAIL=true` default).
- Retention defaults are short (§26.3). Never retain forever by default.
- Document data map in `docs/privacy.md`.

---

## 24. RATE LIMITING, QUOTAS AND ABUSE CONTROLS

Rate limits at: global, organization, tenant, stream, domain, API key, SMTP credential, source IP, recipient domain / MX group. Units: requests/min, messages/min/hour/day, recipients/day, connections. Token bucket in Redis with local fallback; headers on API responses.

Abuse controls: daily quotas per tenant/domain/IP; **new-tenant restrictions** (first 72 h: lower quotas, no bulk stream, no shared `critical` pool); suspicious-volume alerts (z-score vs 7-day baseline); automated tenant/domain/credential suspension with reason; shared-pool protection (§11.4); content scanning (§23.2); abuse intake (§17.4).

The platform must prioritize preventing open relay and credential abuse over convenience.

---

## 25. SEARCH

- Global tenant-scoped search: message ID, recipient, sender, subject, domain, status, API key, SMTP credential, IP, event type, date range, tag, metadata key/value.
- **DECISION:** exact/prefix lookups use B-tree indexes on normalized columns (`address`, `address_domain`, `from_address`, `message_id_header`). Subject and free text use MariaDB **FULLTEXT** indexes on `messages.subject` and an `message_search` denormalized table (`message_id, tenant_id, recipients_text, subject, tags_text`). Never `LIKE '%x%'` on large tables. A `SearchBackend` interface allows an OpenSearch/Meilisearch adapter post-v1.
- All search is bounded by tenant and by default time window (30 days) unless narrowed.

---

## 26. LOGGING, AUDIT, RETENTION AND PARTITIONING

### 26.1 Logging

Structured JSON with `timestamp, level, service, node_id, request_id, trace_id, tenant_id, message_id, recipient_id, event, error`. Never log API secrets, SMTP passwords, DKIM private keys, authorization headers, session tokens, full recipient addresses (redacted by default), message bodies.

### 26.2 Audit

Immutable `audit_logs (id, tenant_id nullable, org_id nullable, actor_user_id, actor_api_key_id, impersonator_user_id, action, resource_type, resource_id, before JSON (secrets redacted), after JSON, reason, ip, user_agent, occurred_at)`. Events: login/logout/MFA, key/credential lifecycle, domain lifecycle, DKIM rotation, webhook changes, user/role/permission changes, retries, suppression changes, settings changes (with previous/new value), impersonation, erasure, exports, DLQ actions, pool moves. Exportable; platform actions never bypass audit.

### 26.3 Retention

Per platform default, per tenant override where allowed:

```text
message metadata   30 days     message body        7 days     raw MIME           3 days
inbound raw        7 days      tracking events    30 days     webhook deliveries 14 days
audit logs        365 days     analytics rollups  730 days    api_request_logs    7 days
dsn/fbl raw        7 days      dmarc/tlsrpt raw   30 days
```

Purge job runs hourly; bodies deleted from storage; metadata rows removed via partition drops.

### 26.4 Time-based partitioning (DECISION)

`messages`, `message_recipients`, `message_events`, `delivery_attempts`, `tracking_events`, `webhook_deliveries`, `api_request_logs`, `inbound_messages` are **RANGE-partitioned by day (or week) on `created_at`**, managed by the scheduler (create 7 days ahead, drop past retention). MariaDB/MySQL partitioned tables **cannot have foreign keys**; therefore these hot tables use **application-enforced referential integrity** (Core repository layer validates parent existence in the same transaction; tests cover orphans), while low-volume configuration tables keep real foreign keys. This trade-off is documented in ADR 0005 and is not negotiable later.

### 26.5 Time zones and i18n

All storage in UTC (`DATETIME(6)`, no `TIMESTAMP` ambiguity); API returns RFC 3339 UTC; Console renders in user-selected time zone (default browser) with UTC toggle. Console strings externalized (i18n-ready, English only in v1).

---

## 27. DATABASE

### 27.1 Tables (at minimum)

```text
-- identity & tenancy
organizations, tenants, tenant_keys, tenant_memberships, users, roles, permissions, role_permissions,
user_roles (deprecated in favor of tenant_memberships.role_ids — pick one, ADR 0006), sessions,
password_resets, email_verifications, invitations, mfa_factors, recovery_codes, sso_providers,
plans, tenant_quotas, usage_counters

-- domains & auth
domains, domain_delegations, domain_dns_records, domain_verifications, dkim_keys, tracking_domains,
bounce_domains, dmarc_reports, dmarc_report_records, tlsrpt_reports

-- infrastructure
smtp_nodes, workers, ip_addresses, ip_pools, ip_pool_members, ip_routing_rules, warmup_plans,
warmup_plan_days, ip_warmup_state, dnsbl_providers, dnsbl_results, ip_reputation_snapshots,
postmaster_snapshots, provider_mx_rules, destination_policies (tls, concurrency), certificates

-- credentials
api_keys, smtp_credentials, auth_failures

-- sending
message_streams, messages, message_recipients, message_events, message_headers, delivery_attempts,
attachments, message_search, outbox, dead_letter_messages, idempotency_keys, bounce_rules,
unmatched_bounces, fbl_reports, held_messages (content scanning)

-- tracking
tracking_links, tracking_events, tracking_bot_rules

-- webhooks
webhooks, webhook_secrets, webhook_events, webhook_deliveries

-- templates
templates, template_versions

-- suppression
suppressions

-- inbound
inbound_routes, inbound_messages, inbound_attachments, abuse_intake

-- analytics
analytics_hourly (tenant, stream, domain, ip, mx_group, template, api_key, smtp_credential dims),
analytics_daily

-- platform
alert_rules, alert_incidents, notification_channels, notifications, audit_logs, api_request_logs,
retention_policies, privacy_requests, system_settings, setting_changes, schema_migrations
```

### 27.2 Analytics rollups (DECISION)

Dashboards never aggregate raw `message_events`. Worker consumes `mvx:analytics` and increments `analytics_hourly` counters (upsert); scheduler folds hourly → daily. Dimensions: tenant, stream, domain, ip, mx_group, template, api_key, smtp_credential. Metrics: accepted, sent, delivered, deferred, bounced (by class), failed, complained, unsubscribed, suppressed, opened, clicked (bot-filtered and raw). Custom-range queries read rollups; "last hour" may read events with a bounded window.

### 27.3 Indexing

Index on actual query patterns; at least `(tenant_id, created_at)`, `(tenant_id, status, created_at)`, `(tenant_id, address_domain)`, `(tenant_id, address)`, `(message_id)`, `(recipient_id, attempt_no)`, `(tenant_id, message_id_header)`, FULLTEXT on subject/search table. Review with `EXPLAIN` in Phase 13 load tests.

### 27.4 Migrations

Versioned SQL migrations (golang-migrate or equivalent), forward + rollback where safe, seeded roles/permissions/bounce rules/provider rules/DNSBL providers/warmup default plan/streams for new tenants, verification step (`mailvex migrate verify`) comparing schema to expected checksum. Never require manual production SQL edits. Expand → migrate → contract pattern for zero-downtime.

---

## 28. DEPLOYMENT PROFILES — INCLUDING HOME LAB / LOCAL SERVER (DECISION)

Mailvex must run equally well as a **home-lab mail hub on a local server** (Raspberry Pi / mini PC / NAS) that collects mail from IoT and LAN devices, and as a multi-node institutional platform. A single `MVX_PROFILE` setting selects sane defaults; every feature stays available in every profile.

| Profile | Target | Defaults |
|---|---|---|
| `homelab` | One box on a LAN, residential internet, IoT/device mail, 1–3 tenants | All-in-one container + MariaDB + Redis; relay transport on; trusted-LAN submission; device registry; low-memory tuning; local domains allowed |
| `standard` | One server with public IPs, small org | Separate containers per service; direct MX delivery; public DNS verification required |
| `cluster` | Multi-node | External DB/Redis, many workers/SMTP nodes, host networking for multi-IP |

### 28.1 Home-lab requirements

**Footprint.** Publish `linux/amd64` and `linux/arm64` images. `homelab` profile must idle under 512 MB RAM total (all containers) and start on a Raspberry Pi 4/5 with a 4 GB card. Provide `mailvex-allinone` image running API + Worker + Webhooks + Scheduler + SMTP edge in one process supervisor for this profile; the same code, only the packaging differs. MariaDB tuned with a low-memory config (`deploy/homelab/my.cnf`).

**Device ingestion (the primary home-lab use case).** Cameras, printers, NAS, UPS, routers, Home Assistant, ESP32/Arduino sketches and similar devices send mail over SMTP with limited capabilities. Support them without ever becoming an open relay:

- **Device registry** (`devices` table, Console → Devices): name, type, expected source IP/CIDR, SMTP credential, allowed From addresses, last seen, message count, last error. Creating a device generates its SMTP credential and a copy-paste settings card (host, port, user, password, TLS mode).
- **Trusted LAN networks** (`trusted_networks`, per tenant, platform-approved CIDRs such as `192.168.0.0/16`, `10.0.0.0/8`, `fd00::/8`): within them the SMTP edge may, per explicit per-device settings, (a) accept `AUTH PLAIN/LOGIN` **without TLS**, (b) accept **IP-authenticated submission** for devices that cannot authenticate at all — the source IP must match the device's registered IP exactly and the From address must be in its allowed list. Both options show a persistent warning in the Console and are impossible from outside the trusted CIDRs. Default: off.
- **Legacy TLS opt-in**: allow TLS 1.0/1.1 and older cipher suites **only** for sessions from trusted networks when a device is flagged `legacy_tls`. Never on the public interface.
- Port 25 unauthenticated submission from trusted networks maps to the IP-authenticated device path above; everything else on port 25 is inbound MX only.
- Message normalization for poor device MIME: missing `Date`/`Message-ID`/`MIME-Version`, bare LF line endings, 8-bit bodies without declaration, empty From (`<>` → rewrite to device's allowed From), oversized inline images (optional downscale off by default).

**Relay transport (smart host).** Residential connections usually block outbound port 25 and have poor reputation, so direct MX delivery is often impossible. Implement a second delivery transport in Mailvex Worker alongside direct MX:

```text
transports: direct_mx | relay
relays (per tenant or pool): name, host, port, tls_mode, auth (encrypted), from_rewrite_policy,
                             daily_cap, provider preset (gmail_workspace | ses_smtp | mailgun | postmark |
                             sendgrid | brevo | generic_smtp), health
```

Routing rules choose transport per stream/domain/destination (`default → relay` in `homelab`). Relay-specific handling: provider caps (e.g. Gmail 500/2000 per day), `From` rewriting when the upstream requires a verified sender (original sender preserved in `Reply-To` and `X-Original-From`), relay response classification, relay health checks, fallback relay. Bounces from the relay arrive as DSNs to the configured return-path and go through the normal DSN pipeline (§17.2).

**Local domains.** A domain may be type `local` (e.g. `home.lan`, `iot.local`). It skips public DNS verification (ownership implied by platform admin) and is marked `LOCAL — not deliverable on the Internet`. Mail from local domains can only go to local mailboxes, or out via a relay with From-rewrite.

**Local mailboxes.** Inbound routes may target a `mailbox` action: a built-in tenant mailbox viewable in the Console (read, search, attachments, mark read, delete) with optional forward-via-relay to a personal address. Useful for `alerts@home.lan`. IMAP/POP3 access is a **non-goal for v1** (interface hook only).

**Mail → push routing.** IoT mail is mostly alerts. Add `mail_routing_rules` (per tenant): match on From/To/Subject/body regex/device → actions `deliver | drop | forward_to | webhook | notify(channel)` using the notification channels of §22.4 plus **`ntfy`**, **`telegram`**, **`home_assistant` (webhook/REST)**, **`gotify`**, **`pushover`** channel types. Rules can also **deduplicate/coalesce** repeated identical alerts within a window (e.g. camera motion every 30 s → one digest per 10 min).

**Networking realities.** Document in `docs/homelab.md`: dynamic IPs and DDNS, why inbound MX on residential IPs is unreliable (recommend relay + local mailboxes or a cheap VPS/tunnel for inbound), CGNAT, ISP port-25 blocks, running behind Tailscale/WireGuard, mDNS/`.local` caveats, exposing only 587 on the LAN, using a private CA / self-signed certificate for LAN TLS (Console offers to generate one and shows how to trust it on devices), and using the reverse proxy only when a public hostname exists.

**Ops for one person.** Single `docker compose up -d`; wizard offers "Home lab" path (creates one tenant, one local domain, a relay, a first device, sends a test); `mailvex backup create` produces one encrypted tarball (DB dump + keys + config + storage); `mailvex backup restore`; Watchtower-compatible images with a Console "update available" indicator; health page in plain language.

### 28.2 Docker

```text
deploy/
  docker-compose.yml            (standard profile, one server)
  docker-compose.homelab.yml    (all-in-one, arm64/amd64, low memory)
  docker-compose.dev.yml
  docker-compose.multi-ip.yml   (host networking / macvlan example)
  homelab/my.cnf
  caddy/Caddyfile  (or traefik/)
  helm/                         (optional, Phase 14)
```

Services (standard): `console, api, worker, scheduler, smtp, webhooks, mariadb, redis, proxy`. Never force everything into one container outside the `homelab` profile. Images: distroless/alpine, non-root, read-only FS where possible, health checks defined.

### 28.3 Cluster mode

Load balancer → N API; N SMTP edge nodes; N Worker nodes each with bound IPs; shared external MariaDB (Galera or primary/replica with automatic failover documented) and Redis (Sentinel or Cluster). No critical state in local memory. Guidance in `docs/ha.md`.

### 28.4 Configuration

Environment variables with `.env.example`, grouped: `MVX_APP_ MVX_DB_ MVX_REDIS_ MVX_SMTP_ MVX_QUEUE_ MVX_STORAGE_ MVX_TLS_ MVX_AUTH_ MVX_WEBHOOK_ MVX_METRICS_ MVX_LOG_ MVX_KMS_ MVX_RELAY_ MVX_PROFILE`. Validated at startup with clear errors. Never commit secrets.

### 28.5 Backup and restore

`mailvex backup create|restore|verify` covering MariaDB dump, tenant keys (re-wrapped under a backup passphrase), configuration, object storage. Never store plaintext credentials in backups. Restore procedure tested in CI on a fixture.

---

## 29. CONSOLE (Mailvex Console)

### 29.1 Navigation

```text
Dashboard
Sending        Messages · Queue · Dead Letters · Suppressions · Templates · Streams
Devices        (home-lab & LAN devices; hidden in cluster profile unless enabled)
Infrastructure Domains · IP Addresses · IP Pools · Warmup · SMTP Nodes · Relays · Certificates
Developers     API Keys · SMTP Credentials · Webhooks · API Logs · Docs
Deliverability Complaints (FBL) · DMARC Reports · TLS-RPT · Compliance Checklist · Reputation
Analytics      Delivery · Domains · IPs · Providers · Templates
Inbound        Routes · Messages · Mailboxes · Abuse Intake
Administration Organization · Tenants · Users · Roles · Invitations · SSO · Audit Logs · Privacy · Settings
System         Health · Workers · Queues · Jobs · Alerts · Notification Channels · Configuration · Updates
```

### 29.2 Dashboard home

Messages today, delivered, bounced, deferred, complaint rate, delivery rate, unsubscribe rate; 24h volume, 7-day delivery, bounce trend, provider distribution charts; system cards (SMTP nodes, workers, queue depth/age, DB, Redis, storage, certificates); active alerts; Development-mode banner when applicable.

### 29.3 Key screens

- **Messages list** with saved filters; **Message inspector** tabs: Overview · Timeline · Recipients · Headers · Authentication (SPF/DKIM/DMARC/TLS/PTR/MTA-STS/DANE result of the sending side) · SMTP (IP, EHLO, MX, TLS, response, timing, attempt count, transport direct/relay) · Tracking (bot-flagged) · Webhooks · Raw Message · Attachments · Body preview (sandboxed, §23.3).
- **Domains**: list (status, SPF/DKIM/DMARC, pool, messages, health) and detail (Overview · DNS · DKIM · DMARC & TLS-RPT · Sending · Tracking/Bounce domains · Compliance · Statistics · Activity) with copy-to-clipboard DNS records and provider-specific DNS guides.
- **IPs**: list and detail (Health · DNS · Reputation · Warmup · Usage · Events); Pools with drag-assign; Nodes with Enable/Disable/Drain/View Logs (never arbitrary shell).
- **Webhooks** detail with delivery log, replay, test event, secret rotation.
- **Suppressions** with import wizard and "why suppressed".
- **Templates** code editor (HTML/text), variables panel with schema, live preview with sample data, versions, diff, test send, publish.
- **Devices** (home lab) with settings card and last-seen.
- **Queue/DLQ** with per-tenant/stream breakdown and safe bulk actions.
- **System → Health** in plain language with links to fixes.

### 29.4 UX rules

Sidebar, breadcrumbs, global search (⌘K), notification center, user menu, tenant switcher, status badges, tables with filters/pagination/column chooser/CSV export, detail drawers/pages, confirmation dialogs for destructive actions (type-to-confirm for delete domain/tenant), empty/loading/error states, skeletons, toasts, keyboard shortcuts, WCAG 2.1 AA. Every error explains **what happened, why, what to do**.

### 29.5 Design system

Reusable components only: Button, Input, Select, Combobox, Modal, Drawer, Table, Badge, StatusIndicator, Card, Tabs, Timeline, Chart, EmptyState, ErrorState, CodeBlock, DNSRecord, MetricCard, ConfirmDialog, CopyField, KeyValue, JsonViewer, SandboxedHtml, DateRangePicker, TimeZoneToggle. Storybook (or equivalent) for every component. No duplicated components across screens.

### 29.6 First screens to build

`/login /setup /dashboard /tenants /domains /domains/:id /devices /ip-addresses /ip-pools /relays /messages /messages/:id /queue /api-keys /smtp-credentials /webhooks /webhooks/:id /templates /suppressions /analytics /users /roles /audit-logs /system/health /settings`

---

## 30. INSTALLATION WIZARD

First launch at `/setup` (disabled once complete; re-enable only via CLI):

```text
1 Choose profile (Home lab / Standard / Cluster)
2 Create platform administrator (+ MFA prompt)
3 Hostname & TLS (ACME / private CA / self-signed / bring your own)
4 Database & Redis check
5 Master key (generate & display once / Vault)
6 System mail (system tenant domain or external relay)
7 First tenant
8 Home lab: add relay + local domain + first device      Standard/Cluster: first IP + EHLO/PTR check + add domain + verify DNS
9 Create API key
10 Send test email and watch it in the inspector
11 Complete → checklist of remaining recommendations
```

---

## 31. SDK AND CLI (Mailvex SDK)

- `openapi.yaml` is generated from code annotations or hand-maintained but **validated in CI** against live handlers.
- Go SDK (`sdk/go`): typed client for the full `/v1` surface, retries with backoff, idempotency helper, webhook signature verification helper.
- CLI `mailvex`: `login`, `send`, `messages list|get|events`, `domains add|verify|dns`, `keys`, `smtp-creds`, `webhooks ... test`, `suppressions import|export`, `devices add`, `relays add|test`, `backup create|restore|verify`, `keys rotate-kek|rotate-dek`, `migrate up|down|verify`, `doctor` (checks DNS, ports, IP binding, relay reachability, PTR), `setup reset`. Output human or `--json`.
- Docs examples for Node/PHP/Python via plain HTTP and via OpenAPI-generated clients (generation config committed; SDK packages themselves are post-v1).
- Developer docs section in Console: Quick Start, API, SMTP setup (including device presets: Home Assistant, Synology, Unifi, Hikvision/Reolink, Proxmox, TrueNAS, HP/Brother printers, APC UPS), Webhooks, Errors, Rate limits, Auth.

---

## 32. OBSERVABILITY AND HEALTH

- `GET /health` (liveness), `GET /ready` (DB, Redis, storage, SMTP edge, DNS resolver, disk, queue, webhook worker, relay reachability where configured), `GET /metrics` (Prometheus).
- OpenTelemetry traces across API → outbox → worker → SMTP with `trace_id` in logs.
- Metrics: `mvx_emails_{accepted,queued,sent,delivered,bounced,deferred,failed,complained,unsubscribed,suppressed}_total{tenant,stream,mx_group,transport}`, `mvx_queue_depth{stream,priority}`, `mvx_queue_oldest_age_seconds`, `mvx_outbox_lag_seconds`, `mvx_smtp_connections{direction}`, `mvx_smtp_errors_total`, `mvx_smtp_auth_failures_total`, `mvx_delivery_duration_seconds`, `mvx_webhook_{success,failure,retry}_total`, `mvx_api_requests_total`, `mvx_api_latency_seconds`, `mvx_relay_health`, `mvx_warmup_cap_deferrals_total`, `mvx_dnsbl_listed`, `mvx_cert_expiry_seconds`.
- Ship a Grafana dashboard JSON and example alert rules in `deploy/observability/`.

---

## 33. TESTING

**Unit:** domain verification, DKIM signing/rotation, SPF/DMARC/MTA-STS/TLSA parsing, DSN/ARF parsing, bounce classifier + rule table, VERP/tracking/unsubscribe token signing, template engine escaping and missing-variable behaviour, header hygiene and injection rejection, idempotency semantics, rate limiter, retry scheduler + jitter, fairness scheduler, suppression rules, provider MX classification, bot filtering, tenant isolation, RBAC/permission resolution, envelope encryption, relay From-rewrite.

**Integration:** MariaDB (partitioned tables, app-level RI), Redis Streams + delayed set + outbox relay, SMTP edge AUTH/authorization/open-relay, outbound SMTP against a test MX (e.g. a containerized receiving server), relay transport against a test smart host, webhooks against a test receiver, storage backends, DNS against a stub resolver, ACME `ask` endpoint.

**End-to-end:** create org/tenant/user → SSO login (mock OIDC) → add domain → verify (stub DNS) → create key → send API + SMTP + device IP-auth → queue → deliver (direct and relay) → DSN bounce → FBL complaint → one-click unsubscribe → suppression → webhook received & verified → analytics rollup → Console reflects all. Playwright for Console.

**Security:** cross-tenant access (every endpoint, automated), broken authorization/privilege escalation, open relay (all three ports, with/without auth, trusted vs untrusted CIDR, legacy TLS), credential reuse across tenants, webhook forgery and replay, API secret leakage in logs/responses, SSRF (webhook URL, forward target, rebinding), SQL injection, XSS in inspector, CSRF, header/CRLF injection, template injection, rate-limit bypass, brute-force lockout on dashboard and SMTP AUTH, impersonation audit presence, erasure completeness.

**Fixtures:** `example.test`, `mail.test`, `tenant-a.test`, `tenant-b.test`, `home.lan` (local). Never real external recipients.

**Failure scenarios:** DB down, Redis down (queue rebuild from DB), SMTP edge down, DNS down, remote MX down/4xx/5xx/rate-limit, relay down/auth failure/cap reached, webhook endpoint down, worker crash mid-session, node crash with bound IPs, network interruption, disk full, certificate expiry, KEK missing, invalid DKIM, domain degraded→suspended, API key revoked mid-request, tenant suspended mid-batch, outbox relay crash after publish, duplicate delivery job. System must fail safely and never double-send beyond at-least-once semantics.

**Performance:** load tests (API, ingestion, queue, delivery to a sink MX, webhooks, DB queries) recording throughput, latency, error rate, queue delay, worker utilization, DB utilization. No throughput claims without measurements. Verify `homelab` footprint target on arm64.

---

## 34. DOCUMENTATION AND PROJECT HYGIENE

```text
README.md  SECURITY.md  CONTRIBUTING.md  CODE_OF_CONDUCT.md  LICENSE  CHANGELOG.md
docs/architecture.md  installation.md  homelab.md  networking.md  configuration.md  smtp.md  devices.md
     relays.md  api.md  webhooks.md  domains.md  dkim.md  spf.md  dmarc.md  mta-sts-dane.md
     bounces-complaints.md  unsubscribe.md  ip-pools.md  warmup.md  multi-tenancy.md  inbound-mail.md
     security.md  privacy.md  backups.md  upgrades.md  ha.md  observability.md  troubleshooting.md
     development.md  third-party.md  api/errors.md  decisions/ (ADRs)
```

CI: lint (`golangci-lint`, eslint, prettier), `go vet`, `govulncheck`, `npm audit`, unit + integration + security tests, OpenAPI validation, migration verify, multi-arch image build, SBOM (CycloneDX), signed releases (cosign), semantic-version tags, release notes from CHANGELOG. Pre-commit hooks. Issue/PR templates.

**License:** permissive (Apache-2.0 preferred, review dependency compatibility first; document in `docs/third-party.md`).

**Code quality rules:** no huge functions, no duplicated business logic, no hidden global state, no hard-coded credentials/domains/IPs, no vendor assumptions, strong validation and error handling, context propagation, structured logging, clear interfaces, tests on critical paths. Prefer **simple, secure, observable, testable, replaceable**.

---

## 35. IMPLEMENTATION ORDER (DECISION — corrected dependencies)

Do not attempt the whole application in one step. Each phase is a separate agent session. Acceptance criteria must pass (tests green in CI) before the next phase.

**Phase 1 — Foundation.** Monorepo, Docker (dev + homelab + standard compose), MariaDB, Redis, Mailvex Core skeleton, Mailvex API skeleton, Console skeleton, migrations + seeds, envelope encryption (env/file KEK), organizations/tenants/tenant_keys/users/memberships/roles/permissions/role_permissions, password auth + sessions + MFA scaffolding + invitations + password reset, RBAC middleware, tenant-scoped repository layer + isolation tests, logging, config validation, health endpoints, setup wizard (steps 1–7), system tenant + system mail relay mode, audit log.
*Accept:* `docker compose up` works on amd64 and arm64; admin logs in; org/tenant created; cross-tenant tests pass; audit entries recorded.

**Phase 2 — Domains & DNS.** Domains, ownership TXT, global uniqueness + delegations, local domains, DNS engine (multi-resolver), SPF/DKIM/DMARC generation and checks, DKIM key lifecycle, degraded/grace/re-verification scheduler, tracking/bounce domain records, domain health UI, streams (default two per tenant).
*Accept:* add domain → records shown → verify against stub DNS → health displayed → degraded flow tested.

**Phase 3 — Message model, queue and SMTP ingestion.** Message/recipient/event/attempt model (partitioned), storage abstraction (FS, S3), outbox + Redis Streams + delayed set + reconciler, `core.Ingest`, MIME builder/parser + normalization, **development mailbox** (intercept sink) and Development mode, SMTP edge (chosen engine) with AUTH, TLS, tenant mapping, domain/sender authorization, trusted networks + IP-authenticated devices + device registry, brute-force protection, open-relay test suite, SMTP credentials UI.
*Accept:* SMTP client (and a simulated IoT device with IP auth) → Mailvex → queue → development mailbox; open relay tests pass on 25/465/587.

**Phase 4 — API sending.** API keys, `POST /v1/email`, batch, idempotency semantics, limits/header allowlist/CRLF, templates minimal render path, read endpoints, pagination, error format, OpenAPI + CI validation, API logs.
*Accept:* API → queue → development mailbox; idempotency and limit tests pass.

**Phase 5 — Delivery.** Mailvex Worker: fairness/priority consumption, **suppression check (required here)**, hard-bounce suppression on 5xx, router + pools + IP binding + EHLO/PTR, MX resolution + MX grouping + provider rules + concurrency, TLS policy (opportunistic/required; MTA-STS/DANE), header hygiene, VERP return-path, **List-Unsubscribe + one-click endpoint + hosted page**, DKIM signing, SMTP client, response classification + rule table, retries, DLQ, **relay transport**, graceful drain, events, delivery metrics.
*Accept:* end-to-end delivery to a test MX and via a test relay; bounce classification, retry, DLQ, drain tests pass; suppression prevents send.

**Phase 6 — Async bounces, complaints, unsubscribe intake.** Bounce domain intake, DSN parser, unmatched bounces, ARF/FBL parser, JMRP/Yahoo checklists, Google Postmaster integration, mailto unsubscribe, postmaster/abuse intake, complaint-rate computation, shared-pool protection.
*Accept:* synthetic DSN and ARF messages produce correct events/suppressions; complaint rate non-zero in tests.

**Phase 7 — Webhooks.** Mailvex Webhooks service: signing with timestamp, SSRF guard, retries, auto-disable, rotation with overlap, replay, test event, delivery log UI, inbound-style batching option.
*Accept:* forged/replayed payloads rejected in receiver test; auto-disable and re-enable tested.

**Phase 8 — Suppression UI/API & privacy.** Full suppression management, import/export, erasure + export requests, retention policies + purge + partition manager.
*Accept:* erasure report complete; partition drop verified.

**Phase 9 — Templates.** Editor, schema, versions, preview, test send, publish, injection tests.

**Phase 10 — Tracking & analytics.** Signed open/click, bot filtering, custom tracking domains + on-demand ACME `ask`, rollups (hourly/daily), dashboards, provider analytics by MX group, FULLTEXT search.

**Phase 11 — Inbound, mailboxes & routing.** Inbound routes, auth checks, ARC, spam/virus hook, local mailboxes, forward, mail→push routing rules with ntfy/Telegram/Home Assistant/Gotify/Pushover/Slack channels, dedup/coalesce.

**Phase 12 — Reputation, health, alerts, SSO.** DNSBL, PTR/FCrDNS, TLS checks, warmup plans and enforcement, IP/domain reputation, alert rules, notification channels, compliance checklist, OIDC + SAML (+ LDAP optional), impersonation.

**Phase 13 — Production hardening.** Security review against §23 and §91-style checklist, content scanning hook, Vault KEK + key rotation tooling, load tests + index review, HA guidance, backup/restore CLI + tests, upgrade process, observability bundle, full documentation, CLI completion, Go SDK.

**Phase 14 — Packaging & stretch.** Multi-arch release pipeline, signed images, SBOM, homelab footprint verification on arm64, optional Helm chart, AWS KMS adapter, IMAP hook interface (no implementation).

---

## 36. STARTING TASK

On the first session produce, in `docs/`:

1. `architecture.md` — services, data flow, delivery ownership, profiles
2. Repository structure (create it)
3. Database ERD (`docs/erd.md` + Mermaid) including partitioning and app-level RI notes
4. Multi-tenant and organization data model
5. Authentication, SSO and RBAC model
6. `openapi.yaml` skeleton with all `/v1` resources
7. Docker and profile architecture (homelab / standard / cluster), networking notes
8. SMTP edge integration architecture + ADR 0002 (engine choice) + trusted-network/device design
9. Queue, outbox, delayed-delivery and fairness architecture
10. Delivery engine architecture (direct MX + relay transports)
11. Webhook architecture
12. Security model (envelope encryption, token signing, threat model)
13. Deliverability architecture (VERP, DSN, FBL, unsubscribe, warmup)
14. Development roadmap mirroring §35

Then begin **Phase 1**.

Do not skip multi-tenancy. Do not postpone tenant isolation. Do not build mock-only architecture that must be rewritten. Do not build a second retry system. Do not let the SMTP edge deliver outbound mail.

---

## 37. DEVELOPMENT RULES FOR THE AI AGENT

Do not merely generate example code. Implement the project.

For every milestone:

1. Read this file and `docs/architecture.md`; inspect the repository.
2. Write or update an ADR for any **EVALUATE** item touched.
3. Create/modify files; keep changes cohesive per commit with conventional commit messages.
4. Run formatter, linters and static analysis.
5. Run unit tests; run integration tests where possible (Docker services available in CI).
6. Run the security test suite (isolation, open relay, injection) — it is never optional after Phase 3.
7. Fix failures before proceeding.
8. Update migrations (forward + rollback), seeds and `openapi.yaml`.
9. Update documentation and `CHANGELOG.md`.
10. Report completed functionality honestly: list what exists, what is tested, what is stubbed. Never claim completion without existing implementation and passing tests.

Architectural preference order: **correctness → security → isolation → observability → reliability → performance**. Prefer simple, secure, observable, testable, replaceable solutions over clever ones. When this document is silent, choose the option that is easiest to run on a single small server and to scale out later, write down the choice in `docs/decisions/`, and continue.
