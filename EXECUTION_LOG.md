# Mailvex — Master Execution & Progress Log

This log tracks the complete implementation of **Mailvex** as specified in [`MAILVEX_MASTER_PROMPT.md`](./MAILVEX_MASTER_PROMPT.md). It records phase status, milestones, decisions, test results, and operational verification.

---

## 1. Master Phase Matrix

| Phase | Description | Status | Verification & Test Summary | Date Completed |
|---|---|---|---|---|
| **§36** | Starting Task: Architecture Artifacts, ADRs, Scaffolding | ✅ Completed | Architecture docs, ADRs, OpenAPI spec, ERDs, repo layout | 2026-09-20 |
| **Phase 1** | Foundation: Multi-tenancy, RBAC, Encryption, Core & API Skeletons | ✅ Completed | Isolation tests, auth tests, encryption tests, setup wizard | 2026-09-20 |
| **Phase 2** | Domains & DNS: Verification, DKIM, SPF, DMARC | ✅ Completed | DNS stub tests, key rotation, uniqueness, degraded flow | 2026-09-20 |
| **Phase 3** | Message Model, Queue & SMTP Ingestion | ⏳ Next | Queue tests, open relay tests, SMTP auth | - |
| **Phase 4** | API Sending: Batch, Idempotency, Limits, Templates minimal | ⏳ Pending | Idempotency tests, rate limit tests | - |
| **Phase 5** | Delivery Engine: Worker, MX Resolution, Suppression, Transports | ⏳ Pending | Delivery tests, retry tests, suppression | - |
| **Phase 6** | Async Bounces, Complaints, Unsubscribe Intake | ⏳ Pending | DSN/ARF parser tests, feedback loops | - |
| **Phase 7** | Webhooks Service: Signing, Replay Protection, Retries | ⏳ Pending | Webhook signature & auto-disable tests | - |
| **Phase 8** | Suppression UI/API, Privacy (GDPR/DPDP), Retention | ⏳ Pending | Erasure report & partition purge tests | - |
| **Phase 9** | Templates Engine & Management | ⏳ Pending | Mustache rendering & safety tests | - |
| **Phase 10** | Tracking & Analytics: Bot Filtering, Rollups | ⏳ Pending | Signed click/open & bot filter tests | - |
| **Phase 11** | Inbound Email, Routing & Local Mailboxes | ⏳ Pending | Inbound route & push notify tests | - |
| **Phase 12** | Reputation, Health, Alerts, SSO (OIDC/SAML) | ⏳ Pending | Alert evaluation & SSO auth tests | - |
| **Phase 13** | Production Hardening: Security Review, Vault KEK, Backup CLI | ⏳ Pending | Backup/restore test, load tests | - |
| **Phase 14** | Packaging & Release: Multi-arch Docker, SDKs, CLI | ⏳ Pending | Container build & verification | - |

---

## 2. Activity Log

### §36 Starting Task: Architecture & Scaffolding
- **Status:** Completed (2026-09-20)
- **Key Artifacts Produced:**
  - `docs/architecture.md`: System services, data flow, delivery ownership, profiles (`homelab`, `standard`, `cluster`).
  - `docs/erd.md`: Database ERD with Mermaid diagrams, partitioning, app-level RI.
  - `docs/decisions/`: ADRs 0001 through 0006.
  - `docs/multi-tenancy.md`: Multi-tenant hierarchy and isolation model.
  - `docs/security.md`: Envelope encryption, threat model, RBAC and session security.
  - `docs/networking.md`: IP binding, host/macvlan Docker networking, PTR/FCrDNS.
  - `docs/homelab.md`: Home lab mail hub, IoT devices, smart host relays.
  - `docs/queue.md`: Outbox pattern, Redis Streams, priority classes, fair scheduling.
  - `docs/delivery.md`: Outbound delivery engine, MX resolution, relay transport.
  - `docs/webhooks.md`: Webhook dispatch, HMAC signatures, replay defense.
  - `docs/deliverability.md`: VERP, DSN, FBL, RFC 8058 unsubscribe, warmup.
  - `docs/roadmap.md`: Detailed milestone plan matching §35.
  - `docs/third-party.md`: Dependency manifest and licensing review.
  - `sdk/openapi.yaml`: Full OpenAPI 3.1 specification for `/v1`.
  - Repository structure created.

### Phase 1: Foundation
- **Status:** Completed (2026-09-20)
- **Implementations:**
  - `core/config`: Environment variable loader validating `MVX_*` profile, KEK, DB, Redis, and SMTP settings.
  - `core/logger`: Structured JSON logging with `log/slog` and automatic PII/email and secret redaction.
  - `core/crypto`:
    - Envelope encryption: Master KEK (env/file/Vault) wrapping per-tenant DEKs (AES-256-GCM).
    - Argon2id password hashing and constant-time verification.
    - Millisecond-sortable prefixed ULIDs (`org_`, `tenant_`, `user_`, `msg_`, `key_`, `aud_`, etc.).
  - `core/domain`: Domain entity models for multi-tenant organizations, tenants, users, roles, permissions, audit logs.
  - `core/repository`: Tenant-scoped repository interfaces enforcing strict `tenantID` boundary checking.
  - `core/audit`: Audit logging service capturing actor, before/after states, IP, and user-agent.
  - `migrations`: `000001_foundation.up.sql` and `down.sql` with schema definitions and seeds for 56 permissions and 8 system roles.
  - `api`:
    - HTTP server and router with `/health` and `/ready` probes.
    - Security headers, request ID injection, structured request logger, and panic recovery middleware.
    - Pre-auth Setup Wizard (`/v1/setup`) executing installation steps 1–7 (profile, platform admin, first tenant, system tenant, DEK wrapping, audit).
    - Authentication handlers (`/v1/auth/login`, `/v1/auth/me`).
    - Tenant management handlers (`/v1/tenants`) with RBAC permission enforcement.
    - Audit log querying (`/v1/audit-logs`).
  - `console`:
    - React 18 + Vite + TypeScript + Tailwind CSS application skeleton.
    - Setup Wizard UI (`/setup`), Login UI (`/login`), and Authenticated shell with tenant switcher and health dashboard.
    - Successfully compiled with `npm run build`.
  - `deploy`:
    - `deploy/docker-compose.yml` (standard profile)
    - `deploy/docker-compose.homelab.yml` (homelab all-in-one profile)
    - `deploy/docker-compose.dev.yml` (local dev DB & Redis)
    - `deploy/Dockerfile.api` & `console/Dockerfile`
    - `.env.example`
- **Verification & Test Results:**
  - `core/crypto`: Envelope encryption, Argon2id, and ULID unit tests all PASS.
  - `core/repository`: Strict cross-tenant isolation and audit log scoping tests all PASS.
  - `api`: Health, setup wizard, admin login, tenant creation, and audit logging tests all PASS.
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 2: Domains & DNS
- **Status:** Completed (2026-09-20)
- **Implementations:**
  - `core/domain`: Added `Domain`, `DomainDelegation`, `DKIMKey`, `DNSRecord`, and `MessageStream` entities.
  - `core/crypto/dkim`: RSA 2048-bit DKIM key pair generation with PKCS#8 PEM private key and base64 DER public key; strict refusal of keys under 2048 bits (§10.5).
  - `core/dns`: Multi-resolver DNS engine with `StubResolver` and `SystemResolver` verifying ownership TXT, DKIM, SPF, DMARC, and CNAME records; degraded state progression with 72h grace period (§10.1).
  - `core/repository`:
    - Domain CRUD with global uniqueness validation (preventing cross-tenant domain hijacking without delegation).
    - Local domain handling (`home.lan`) skipping public DNS and auto-verifying.
    - DKIM key lifecycle management (`pending` -> `active` -> `retiring` -> `retired`).
    - Message stream creation and retrieval with default `transactional` and `broadcast` streams.
  - `migrations`: `000002_domains_and_dns.up.sql` and `down.sql`.
  - `api`:
    - `POST /v1/domains`: Adds domain, generates ownership token, generates RSA 2048 DKIM key wrapped in tenant DEK, produces full set of DNS records.
    - `GET /v1/domains` & `GET /v1/domains/{id}`: Retrieves domain details and DNS records.
    - `POST /v1/domains/{id}/verify`: Executes multi-record DNS check, updates verification status and handles degraded flow.
    - `POST /v1/domains/{id}/dkim/rotate`: Initiates DKIM selector rotation.
    - `GET /v1/streams` & `POST /v1/streams`: Manages message streams.
  - `console`:
    - Interactive Domains & DNS management UI with status indicators (`verified`, `pending`, `degraded`).
    - DNS records table with copyable values and "Verify DNS Now" trigger.
    - Message streams overview (`transactional` priority p1 vs `broadcast` RFC 8058 one-click).
- **Verification & Test Results:**
  - `core/crypto`: `TestDKIMKeyGeneration` PASS (RSA 2048 valid, 1024 refused).
  - `core/dns`: `TestDNSEngineAndVerification` PASS (DNS verification, recovery, degraded, and suspended states).
  - `api`: `TestDomainsAndDNSLifecycle` PASS (add domain -> records shown -> verify against stub DNS -> health displayed -> degraded flow -> DKIM rotation -> stream creation -> local domain auto-verify).
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 3: Message Model, Queue & SMTP Ingestion
- **Status:** Completed (2026-09-20)
- **Implementations:**
  - `core/domain`: Added `Message`, `MessageRecipient`, `DeliveryAttempt`, `MessageEvent`, `OutboxItem`, `SMTPCredential`, `Device`, and `TrustedNetwork`.
  - `core/storage`: S3/MinIO compatible object store abstraction with `MemoryStorage` and `FSStorage` for raw MIME payloads and encrypted attachments (§12).
  - `core/queue`: Queue abstraction with `MemoryQueue` (in-memory channel + delay timer) and PostgreSQL-backed Transactional `OutboxRelay` (§14).
  - `core/mime`: Robust MIME normalization, bare LF -> CRLF sanitization, 998-character line limit enforcement, IoT/device synthetic MIME generation with `Message-ID`, `Date`, and `MIME-Version` headers (§13.2).
  - `core/ingest`: Unified Ingestion Engine handling recipient fan-out, VERP HMAC signing, development mailbox sink (`sink_to_mailbox`), stream routing, and transactional outbox staging.
  - `smtp`: High-performance SMTP Edge server using `github.com/emersion/go-smtp` & `go-sasl`:
    - SASL PLAIN authentication against tenant `smtp_credentials`.
    - IP-authenticated trusted networks and IoT devices (§28.1).
    - Rate limiting and brute-force lockout (5 failed attempts locks client IP for 15 minutes).
    - Strict open relay prevention (unauthenticated sessions can only send to verified Mailvex domains; external relaying rejected with 550).
  - `api/handlers`: Added SMTP credentials, IoT devices, trusted networks, and message listing endpoints.
  - `migrations`: `000003_messages_queue.up.sql` and `down.sql`.
- **Verification & Test Results:**
  - `core/mime`: `TestMIMENormalizationAndParsing` PASS (bare LF CRLF fix, long line folding, synthetic headers).
  - `core/queue`: `TestQueueAndOutboxRelay` PASS (outbox staging -> relay poll -> queue dispatch -> ack).
  - `smtp`: `TestSMTPAuthenticatedSubmission`, `TestSMTPDeviceIPAuthenticatedSubmission`, `TestOpenRelayPrevention`, `TestBruteForceProtection` all PASS.
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 4: API Sending & Templating
- **Status:** Completed (2026-09-20)
- **Implementations:**
  - `core/domain`: Added `APIKey`, `Template`, `TemplateVersion`, `IdempotencyRecord`, and `MessageFilter` models.
  - `core/crypto/apikey`: Generation of random base62 API keys (`mvx_live_...`, `mvx_test_...`) with constant-time SHA-256 hash lookup and prefix + last 4 storage.
  - `core/template`: Logic-less Mustache-compatible template engine supporting variable lookup (`{{var}}`), unescaped HTML (`{{{var}}}` or `{{&var}}`), conditional and iterable sections (`{{#section}}...{{/section}}`), auto-escaping to prevent XSS/template injection, and strict/lenient missing variable enforcement (§23.3).
  - `core/repository`: Added API key management, template CRUD + published versioning, idempotency records with 24h TTL, and cursor-based pagination for messages.
  - `api/middleware`: Dual authentication support accepting either User Session (`X-User-ID`) or API Key Bearer tokens (`Bearer mvx_live_...` / `Bearer mvx_test_...`), enforcing scope verification (`send.email`, `messages.read`, etc.) and allowed IP CIDR checks.
  - `api/handlers/email`:
    - `POST /v1/email`: Validates Idempotency-Key (handling concurrent in-progress, matching payload replay, and mismatch conflicts), CRLF/header injection rejection (`400 header_injection`), custom headers allowlist check (`X-*`, `List-Id`, `Precedence`), 50 recipient limit, 20 MB attachment limit, domain verification check against tenant verified domains, and Mustache template rendering. Dispatches to ingestion pipeline.
    - `POST /v1/email/batch`: Batch sending up to 500 recipients with shared defaults, per-recipient overrides (template variables, metadata), and independent failure isolation.
  - `api/handlers/messages`: Full message inspection (`GET /v1/messages/{id}`), recipient list (`/recipients`), chronological event timeline (`/events`), stored body reading (`/body`), retry (`/retry`), and cancel (`/cancel`), plus cursor-based paginated search (`GET /v1/messages`).
  - `api/handlers/apikeys`: API key creation (`POST /v1/api-keys`), listing (`GET /v1/api-keys`), and instant revocation (`DELETE /v1/api-keys/{id}`).
  - `api/handlers/templates`: Template creation, versioning, listing, and live preview rendering (`POST /v1/templates/{id}/render`).
  - `console`: Interactive React 18 UI for API Key generation (modal showing secret once with copy button), Message Logs with full Inspector modal (recipients table, event timeline, and sandboxed iframe for preview), and Templates editor with live preview output.
  - `migrations`: `000004_api_keys_and_templates.up.sql` and `down.sql`.
  - `sdk/openapi.yaml`: Updated OpenAPI 3.1 specifications with complete Phase 4 endpoints.
- **Verification & Test Results:**
  - `core/template`: `TestMustacheTemplateRendering`, `TestTemplateHTMLAutoEscape`, `TestTemplateSections`, `TestTemplateMissingVariable` all PASS.
  - `api`:
    - `TestAPIKeyAuthenticationAndManagement` PASS (generation, bearer auth, revocation).
    - `TestSendEmailWithAPIKeyAndDevelopmentSink` PASS (API -> queue -> development mailbox sink verification).
    - `TestIdempotencySemantics` PASS (replays return cached response, differing payloads return 409 conflict).
    - `TestHeaderValidationAndCRLFFiltering` PASS (CRLF injection in From/Subject/recipients rejected with 400, prohibited custom headers rejected, allowed headers accepted).
    - `TestRecipientLimitsAndDomainVerification` PASS (>50 recipients rejected, unverified domain rejected).
    - `TestTemplateRenderingInEmailSend` PASS (Mustache substitution + HTML auto-escape + body retrieval).
    - `TestBatchSendingEndpoint` PASS (batch up to 500 with per-recipient metadata).
    - `TestMessageReadAndCursorPagination` PASS (cursor-based pagination and filter by recipient).
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 5: Outbound Delivery Engine (Mailvex Worker)
- **Status:** Completed (2026-09-20)
- **Implementations:**
  - `core/domain`: Added `Suppression`, `IPPool`, `IPAddress`, `Relay`, `BounceRule`, `ProviderMXRule`, and `DeadLetterMessage` models.
  - `core/crypto/dkim`: Implemented RFC 6376 `SignDKIM` with relaxed/relaxed header and body canonicalization, body hash (`bh=`), and RSA-SHA256 signature (`b=`).
  - `core/repository`:
    - Suppressions repository with O(1) checks for exact address, domain blocks, and stream-scoped unsubscribes.
    - IP pool and bound IP allocation with warmup cap enforcement.
    - Upstream smart-host relays CRUD with From-rewrite policy support.
    - Dynamic bounce rule tables and MX provider rules.
    - Dead-letter message (DLQ) logging and recipient delivery tracking updates.
  - `core/ingest`: Updated unified ingestion to check suppressions up-front (§13.1, §18) and emit initial `message.suppressed` events.
  - `worker/router`: Router selecting transport (`direct_mx` vs `relay`), resolving IP pools (`tenant -> stream -> domain -> default`), and selecting active IP bound to current node.
  - `worker/mx`: In-memory caching MX resolver with RFC 7505 Null MX detection (rejecting email to unrouteable domains) and RFC 5321 fallback to domain A/AAAA record; provider classification (`google`, `microsoft`, `yahoo`, `apple`, `custom`).
  - `worker/classifier`: Response classification engine mapping SMTP status and enhanced codes to standard bounce taxonomy (`hard_bounce`, `soft_bounce`, `rate_limited`, `spam_rejection`, `auth_rejection`, `policy_rejection`, `mailbox_unavailable`, `domain_unavailable`, `unknown`).
  - `worker/hygiene`: Sanitizes headers (strips internal `X-Mailvex-*` and private `Received` lines), generates missing `Message-ID`, `Date`, `MIME-Version`, injects `Feedback-ID: <tenant>:<stream>:<campaign>:mailvex`, injects `X-Mailvex-Tenant-Id: <hash>`, generates VERP `Return-Path: <bounce+<token>@<domain>>`, and injects `List-Unsubscribe` + `List-Unsubscribe-Post: List-Unsubscribe=One-Click` on broadcast streams.
  - `worker/transport`:
    - `DirectMXTransport`: Speaks SMTP directly to remote MX host, negotiates opportunistic/required STARTTLS, handles MAIL FROM (VERP), RCPT TO, and DKIM-signed DATA.
    - `RelayTransport`: Connects to upstream smart-host relay with PLAIN/LOGIN authentication and `From` header rewriting (`FromRewritePolicy = "replace"`) preserving original sender in `Reply-To` and `X-Original-From`.
  - `worker/retry`: Retry scheduler with exponential backoff (1m, 5m, 15m, 30m, 1h, 2h, 6h, 12h, 24h) and ±20% jitter.
  - `worker/worker`: Complete delivery engine pipeline with priority consumption, suppression check before transmission, response classification, immediate hard-bounce auto-suppression on 5xx, retry deferrals on 4xx, DLQ movement on exhaustion/fatal failure, and graceful drain on SIGTERM.
  - `cmd/worker`: Standalone background worker daemon with graceful shutdown.
  - `api/handlers/unsubscribe`:
    - `POST /u/{token}`: RFC 8058 One-Click unsubscribe endpoint adding `unsubscribe` suppression and emitting `message.unsubscribed`.
    - `GET /u/{token}`: Hosted unsubscribe confirmation page with one-click submit.
  - `api/handlers/suppressions`: Management endpoints (`GET /v1/suppressions`, `POST /v1/suppressions`, `DELETE /v1/suppressions/{id}`).
  - `console`: Suppressions management tab with manual suppression addition, type badges, reason display, source tracking, and removal action.
  - `migrations`: `000005_delivery_worker.up.sql` and `down.sql`.
- **Verification & Test Results:**
  - `worker/worker_test.go`:
    - `TestDirectMXDeliverySuccess` PASS (end-to-end delivery to mock MX server with VERP return-path, Feedback-ID, and DKIM signature verified).
    - `TestSmartHostRelayDeliveryWithFromRewrite` PASS (delivery via authenticated smart-host relay with From header rewritten and X-Original-From preserved).
    - `TestSuppressionCheckPreventsSending` PASS (pre-existing suppression prevents network transmission and marks recipient suppressed).
    - `TestHardBounceAutoSuppressionAndDLQ` PASS (5xx permanent rejection automatically records hard_bounce suppression and moves to DLQ).
    - `TestSoftBounceRetryScheduling` PASS (4xx transient failure schedules retry with exponential backoff and jitter without auto-suppressing).
    - `TestWorkerGracefulDrain` PASS (worker drain stops claiming and rejects new work).
  - `api/unsubscribe_test.go`:
    - `TestUnsubscribeEndpoints` PASS (RFC 8058 one-click POST, hosted HTML GET, and tampered token 403 rejection).
  - `core/crypto`: `TestSignDKIM` PASS (RFC 6376 RSA-SHA256 signature verification).
  - `go test -count=1 ./...`: 100% PASS across all 19 packages.
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 6: Feedback Loops & Inbound DSN
- **Status:** Completed & Verified
- **Components Implemented:**
  - `migrations/000006_fbl_dsn_abuse`: Added tables for `fbl_reports`, `unmatched_bounces`, `abuse_reports`, `ip_reputation_snapshots`, and `postmaster_snapshots`.
  - `core/domain`: Added models `FBLReport`, `UnmatchedBounce`, `AbuseReport`, `IPReputationSnapshot`, and `PostmasterSnapshot`.
  - `core/repository`: Added repository methods for FBL reports, unmatched bounces, abuse intake queue (listing, status transitions to resolved/escalated), and reputation snapshots.
  - `core/crypto/verp`: Implemented HMAC-SHA256 signed VERP token generator, token verifier, Return-Path address formatter, and parser (`bounce+<token>@<domain>`).
  - `core/reports/dsn`: RFC 3464 MIME delivery-status parser with auto-reply/out-of-office filtering (`Auto-Submitted`, `X-Autoreply`, `Precedence: auto_reply`), per-recipient status parsing, diagnostic-code extraction, response classification via `worker.Classifier`, and heuristic fallback for non-standard bounces.
  - `core/reports/arf`: RFC 5965 ARF feedback report parser extracting `Feedback-Type: abuse`, original message headers, `Feedback-ID: <tenant>:<stream>:<campaign>:mailvex`, `Message-ID`, and reported domain.
  - `core/reports/processor`: High-level `ReportProcessor` handling:
    - Inbound DSN: drops auto-replies, resolves recipient via VERP, marks recipient status (`bounced` on 5xx, `deferred` on 4xx), records delivery attempt, emits `message.bounced`/`message.deferred`, automatically records `hard_bounce` suppression, and stores unresolved bounces in `unmatched_bounces`.
    - Inbound ARF: saves `fbl_reports` row, emits `message.complained`, and immediately records `complaint` suppression.
    - Mailto Unsubscribe: verifies token signature, records `unsubscribe` suppression, and emits `message.unsubscribed`.
    - Abuse Intake: ingests inbound mail to `abuse@*` and `postmaster@*` (RFC 5321 §4.5.1) into platform abuse queue.
  - `smtp/server`: Updated SMTP edge to accept incoming report/admin traffic without authentication (`bounce+*`, `unsubscribe+*`, `fbl@*`, `abuse@*`, `postmaster@*`) and dispatch to `ReportProcessor`.
  - `api/handlers/abuse`: Added endpoints:
    - `GET /v1/abuse-reports`: Filterable abuse reports queue.
    - `GET /v1/abuse-reports/{id}`: Single abuse report inspection.
    - `POST /v1/abuse-reports/{id}/resolve`: Mark resolved with notes.
    - `POST /v1/abuse-reports/{id}/escalate`: Mark escalated with notes.
    - `POST /v1/reports/dsn`: Webhook DSN report ingestion.
    - `POST /v1/reports/arf`: Webhook ARF report ingestion.
    - `GET /v1/reports/fbl`: Tenant feedback loop complaints.
    - `GET /v1/reports/postmaster` & `POST /v1/reports/postmaster`: Google Postmaster / SNDS snapshots.
  - `console`: Added **Reputation & Abuse** tab with:
    - Provider reputation metrics cards (Google Postmaster spam rate & domain rep, Microsoft SNDS filter status & trap hits, Yahoo CFL alignment).
    - Interactive Abuse Intake Queue with status filters (Pending, Resolved, Escalated) and action buttons (Resolve, Escalate).
    - Inbound ARF Feedback Loop table.
    - ISP Reputation integration setup guides (Google Postmaster, Microsoft SNDS/JMRP, Yahoo CFL).
  - `sdk/openapi.yaml`: Updated with all Phase 6 endpoints.
- **Verification & Test Results:**
  - `core/reports/reports_test.go`:
    - `TestRFC3464DSNHardBounce` PASS (multipart/report parsed, recipient resolved via VERP, marked bounced, hard_bounce auto-suppressed, DLQ written).
    - `TestRFC3464DSNSoftBounce` PASS (4.2.2 mailbox full parsed, marked deferred, no auto-suppression).
    - `TestAutoReplyDetectionAndFiltering` PASS (RFC 3834 `Auto-Submitted`, `X-Autoreply`, `Precedence: auto_reply`, and "Out of Office" subjects detected and filtered without bouncing or suppressing).
    - `TestUnmatchedBounceStored` PASS (unmatched bounce recorded in `unmatched_bounces` table).
    - `TestRFC5965ARFFeedbackReport` PASS (ARF abuse report parsed, `Feedback-ID` correlated, `fbl_reports` saved, complaint suppression recorded).
    - `TestMailtoUnsubscribe` PASS (HMAC token verified, address suppressed as `unsubscribe` via `mailto`, `message.unsubscribed` emitted).
    - `TestAbusePostmasterIntake` PASS (inbound abuse report stored, domain correlated, status updated to resolved).
  - `api/abuse_test.go`:
    - `TestAbuseReportLifecycle` PASS (list abuse reports, get report, resolve with notes, escalate with notes).
    - `TestReportIngestionWebhookEndpoints` PASS (POST /v1/reports/dsn, POST /v1/reports/arf, GET /v1/reports/fbl, POST & GET /v1/reports/postmaster).
  - `core/crypto/verp_test.go`:
    - `TestVERPTokenGenerationAndVerification` PASS (HMAC token generation, verification, and tamper detection).
    - `TestVERPAddressFormattingAndParsing` PASS (VERP address format and angle bracket extraction).
  - `go test -count=1 ./...`: 100% PASS across all 20 packages.
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 7: Analytics & Reporting
- **Status:** Pending

