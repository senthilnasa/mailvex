# Mailvex — Master Execution & Progress Log

This log tracks the complete implementation of **Mailvex** as specified in [`MAILVEX_MASTER_PROMPT.md`](./MAILVEX_MASTER_PROMPT.md). It records phase status, milestones, decisions, test results, and operational verification.

---

## 1. Master Phase Matrix

| Phase | Description | Status | Verification & Test Summary | Date Completed |
|---|---|---|---|---|
| **§36** | Starting Task: Architecture Artifacts, ADRs, Scaffolding | ✅ Completed | Architecture docs, ADRs, OpenAPI spec, ERDs, repo layout | 2026-09-20 |
| **Phase 1** | Foundation: Multi-tenancy, RBAC, Encryption, Core & API Skeletons | ✅ Completed | Isolation tests, auth tests, encryption tests, setup wizard | 2026-09-20 |
| **Phase 2** | Domains & DNS: Verification, DKIM, SPF, DMARC | ✅ Completed | DNS stub tests, key rotation, uniqueness, degraded flow | 2026-09-20 |
| **Phase 3** | Message Model, Queue & SMTP Ingestion | ✅ Completed | Outbox queue tests, open relay security tests, MIME parser | 2026-09-20 |
| **Phase 4** | API Sending: Batch, Idempotency, Limits, Templates minimal | ✅ Completed | Idempotency semantics, API keys, rate limits, templates | 2026-09-20 |
| **Phase 5** | Delivery Engine: Worker, MX Resolution, Suppression, Transports | ✅ Completed | Direct MX delivery, smart-host relay, retry scheduler, DLQ | 2026-09-21 |
| **Phase 6** | Async Bounces, Complaints, Unsubscribe Intake | ✅ Completed | DSN hard/soft bounce, ARF FBL complaints, VERP, abuse queue | 2026-09-21 |
| **Phase 7** | Analytics & Reporting: Rollups, Histograms, ISP Breakdown, CSV | ✅ Completed | Pre-aggregated rollups, latency histogram, provider breakdown, CSV exporter | 2026-09-21 |
| **Phase 8** | Webhooks Service: Signing, Replay Protection, Retries | ✅ Completed | HMAC signing, 24h secret rotation overlap, replay defense, circuit breaker | 2026-09-21 |
| **Phase 9** | Suppression Import/Export, GDPR/DPDP Article 17/15, Rolling Data Retention | ✅ Completed | Import/export CSV, Erasure report & partition purge tests | 2026-09-21 |
| **Phase 10** | Open & Click Tracking, Bot Filtering, Inbound Email Routing | ✅ Completed | Bot filtering, HMAC tracking redirect, inbound MX routing | 2026-09-21 |
| **Phase 11** | Reputation, DNSBL, Warmup & Alert Center | ✅ Completed | DNSBL reverse queries, ramp schedules, multi-channel dispatch | 2026-09-21 |
| **Phase 12** | SSO (OIDC/SAML), Impersonation, Support Privacy, Deliverability Compliance | ✅ Completed | OIDC/SAML SP metadata, JIT provisioning, support body masking, FCrDNS/TLS audit | 2026-09-21 |
| **Phase 13** | Production Hardening: Security Review, Vault KEK, Backup CLI | ⏳ Next | Security audit (§23), Vault KEK adapter, backup/restore CLI | - |
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
- **Status:** Completed & Verified (2026-09-21)
- **Components Implemented:**
  - `migrations/000007_analytics`:
    - `analytics_rollups`: Multi-dimensional pre-aggregated counter table partitioned by time bucket (`1h`, `1d`) with dimensions (`tenant_id`, `stream_id`, `domain_id`, `provider`, `campaign_tag`), latency histogram distribution columns (`latency_lt_5s`, `latency_lt_30s`, `latency_lt_1m`, `latency_lt_5m`, `latency_lt_15m`, `latency_gt_15m`), and sum/count for percentiles.
    - `analytics_exports`: Persistent job records for on-demand and asynchronous CSV exports.
  - `core/domain`: Added `AnalyticsRollup`, `AnalyticsFilter`, `AnalyticsSummary`, `TimeSeriesPoint`, `ProviderBreakdown`, `LatencyHistogram`, and `AnalyticsExport`.
  - `core/analytics`:
    - Destination Provider Classifier (`ClassifyProvider`) categorizing destination domains into `google`, `microsoft`, `yahoo`, `apple`, `fastmail`, and `other`.
    - `AnalyticsEngine` with high-level aggregations (`QueryOverview`, `QueryTimeSeries`, `QueryProviderBreakdown`, `QueryLatencyHistogram`, `GenerateCSVExport`).
    - Reputation warning rule: sets `ComplaintWarning = true` when complaint rate reaches or exceeds 0.10% (Google / Yahoo spam rate threshold).
  - `core/repository`: Tenant-scoped analytics recording methods (`RecordAnalyticsEvent`, `QueryAnalyticsSummary`, `QueryTimeSeriesPoints`, `QueryProviderBreakdown`, `QueryLatencyHistogram`, `SaveAnalyticsExport`, `GetAnalyticsExport`, `ListAnalyticsExports`).
  - Event recording hooks:
    - Ingestion: `message.sent` recorded in `core/ingest.go`.
    - Delivery: `message.delivered` (with latency duration), `hard_bounce`, and `soft_bounce` recorded in `worker/worker.go`.
    - Reports: DSN hard/soft bounces, ARF complaints, and mailto unsubscribes recorded in `core/reports/processor.go`.
  - `api/handlers/analytics`:
    - `GET /v1/analytics/overview`: Aggregated KPI summary with delivery, bounce, complaint, unsubscribe rates and Google/Yahoo threshold warning flag.
    - `GET /v1/analytics/time-series`: Dynamic hourly/daily trend buckets.
    - `GET /v1/analytics/providers`: Major ISP delivery & complaint performance breakdown.
    - `GET /v1/analytics/latency`: Latency distribution histogram across 6 latency buckets.
    - `POST /v1/analytics/export`: Create on-demand CSV export job.
    - `GET /v1/analytics/exports`: List tenant exports.
    - `GET /v1/analytics/exports/{id}/download`: Stream CSV attachment directly to client.
  - `api/server`: Mounted all Phase 7 routes with RBAC middlewares (`analytics.read`, `analytics.export`).
  - `console`: Added **Analytics & Reports** tab:
    - Time range selector (`24 Hours`, `7 Days`, `30 Days`).
    - Prominent Google & Yahoo Deliverability Alert banner when complaint rate exceeds 0.10%.
    - KPI cards (Total Sent, Delivery Rate %, Bounce Rate %, Complaint Rate %).
    - Latency percentile metrics (Median p50, Tail p95) and Unsubscribe metrics.
    - Delivery Latency distribution histogram bar chart.
    - ISP Provider Breakdown table with sortable metrics per mailbox provider.
    - On-demand CSV Export button with automatic browser download.
  - `sdk/openapi.yaml`: Documented all Phase 7 endpoints.
- **Verification & Test Results:**
  - `core/analytics/analytics_test.go`:
    - `TestAnalyticsEngineRollupsAndRates` PASS (pre-aggregated rollups, KPI calculations, delivery rate, bounce rate, complaint rate warning evaluation, CSV generation).
    - `TestClassifyProvider` PASS (Google, Microsoft, Yahoo, Apple, Fastmail, unknown domain classification).
  - `api/analytics_test.go`:
    - `TestAnalyticsHTTPEndpoints` PASS (RBAC protection, overview endpoint, time-series endpoint, provider breakdown endpoint, latency endpoint, export generation, and CSV download).
  - `go test -count=1 ./...`: 100% PASS across all 21 packages.
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 8: Webhook Delivery Engine
- **Status:** Completed & Verified (2026-09-21)
- **Components Implemented:**
  - `migrations/000008_webhooks`:
    - `webhook_endpoints`: Endpoints with destination URL, description, subscribed events JSON, active HMAC signing secret, previous secret & 24h expiration timestamp for zero-downtime secret rollover, status (`active`, `disabled_failing`, `disabled_manual`), and consecutive failure counter.
    - `webhook_deliveries`: Full historical delivery log with attempt ID, event ID, event type, serialized payload, HTTP status code, response body preview, error message, attempt number, and duration (ms).
  - `core/domain`: Added models `WebhookEndpoint` and `WebhookDelivery`.
  - `core/webhooks`:
    - `Dispatcher`: Dispatches events to subscribed endpoints with timeout and security headers:
      - `X-Mailvex-Delivery-Id`: Unique delivery ID (`whd_...`)
      - `X-Mailvex-Event-Id`: Unique event ID (`evt_...`)
      - `X-Mailvex-Timestamp`: Unix epoch timestamp `t`
      - `X-Mailvex-Signature`: `t=<unix>,v1=<sig1>[,v1=<sig2>]`
    - HMAC-SHA256 Signature computation over string `${t}.${payload}`.
    - 24-Hour Secret Rotation Overlap (§1.2): Generates dual signatures (`v1=<new>,v1=<old>`) during the 24-hour rollover window.
    - Replay Defense (`VerifySignature`): Strictly verifies signatures and rejects deliveries where `|now - t| > 300` seconds (§1.1).
    - Automatic Circuit Breaker: Tracks consecutive failure counts. After 10 consecutive failures, automatically transitions endpoint status to `disabled_failing` to prevent cascading outbound exhaustion (§1.3).
    - Test Ping (`DispatchPing`): Immediately sends `webhook.test` event ping.
  - `core/repository`: Added tenant-scoped webhook operations (`CreateWebhookEndpoint`, `GetWebhookEndpoint`, `ListWebhookEndpoints`, `UpdateWebhookEndpoint`, `DeleteWebhookEndpoint`, `RotateWebhookSecret`, `RecordWebhookDelivery`, `ListWebhookDeliveries`, `GetActiveWebhookEndpointsForEvent`).
  - `api/handlers/webhooks`:
    - `POST /v1/webhooks`: Register webhook endpoint with validation and secret generation.
    - `GET /v1/webhooks`: List tenant webhook endpoints.
    - `GET /v1/webhooks/{id}`: Get endpoint details.
    - `PATCH /v1/webhooks/{id}`: Update endpoint URL, events, description, status.
    - `DELETE /v1/webhooks/{id}`: Delete endpoint.
    - `POST /v1/webhooks/{id}/rotate-secret`: Rotate secret with 24-hour overlap.
    - `POST /v1/webhooks/{id}/test`: Trigger test event ping.
    - `GET /v1/webhooks/{id}/deliveries`: Paginated delivery attempts history.
  - `api/server`: Mounted all Phase 8 routes with RBAC middleware (`webhooks.read`, `webhooks.write`, `webhooks.secret.rotate`).
  - `console`: Added **Webhooks & Signing** tab:
    - Endpoint registration form with destination URL, description, and event subscriptions.
    - Registered endpoints table with status badges (`Active`, `Circuit Broken (10+ Failures)`, `Disabled`).
    - Secret rotation button with zero-downtime 24-hour dual-signature alert banner.
    - Test Ping trigger button.
    - Delivery history drawer/modal showing HTTP status, duration, error messages, and timestamps.
    - Inline Webhook Security documentation and signature verification guide.
  - `sdk/openapi.yaml`: Documented all Phase 8 endpoints.
- **Verification & Test Results:**
  - `core/webhooks/webhooks_test.go`:
    - `TestWebhookSignatureGenerationAndVerification` PASS (HMAC-SHA256 valid signature, payload tampering rejection, invalid secret rejection).
    - `TestWebhookSecretRotation24HourOverlap` PASS (zero-downtime dual signatures verified by receivers using either old or new secret; old secret signature omitted after 24h expiration).
    - `TestWebhookReplayAttackToleranceRejection` PASS (timestamp > 300s rejected with `ErrReplayAttack`).
    - `TestWebhookCircuitBreakerAutoDisable` PASS (endpoint automatically disabled after 10 consecutive failures and stopped receiving dispatches).
    - `TestWebhookTestPing` PASS (ping dispatches `webhook.test` event, signed header verified, delivery logged).
  - `api/webhooks_test.go`:
    - `TestWebhookHTTPEndpoints` PASS (full REST API lifecycle: create, list, get, update, secret rotation with 24h overlap, test ping dispatch, delivery history listing, delete).
  - `go test -count=1 ./...`: 100% PASS across all 22 Go packages.
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 9: Suppression Management, Privacy & Retention Engine
- **Status:** Completed & Verified (2026-09-21)
- **Components Implemented:**
  - `migrations/000009_suppression_privacy_retention`:
    - `suppression_imports`: Tracking bulk suppression imports with filename, row counters (total, imported, duplicates), and status.
    - `privacy_requests`: GDPR/DPDP Article 15/17 compliance records (erasure, export, status, result summary, requested/completed timestamps).
    - `retention_policies`: Per-tenant rolling data retention configuration (message retention days, events retention days, attempts retention days, raw MIME retention days).
  - `core/domain`: Added models `SuppressionImport`, `PrivacyRequest`, `RetentionPolicy`, `PrivacyExportReport`, `RetentionPurgeSummary`.
  - `core/crypto`: Added ULID prefixes `PrefixSuppressionImport` (`supimp_`), `PrefixPrivacyRequest` (`priv_`), `PrefixRetentionPolicy` (`retpol_`).
  - `core/privacy`:
    - `PrivacyService`: High-level service for handling GDPR Article 17 (Right-to-be-Forgotten erasure) and Article 15 (Data Portability export).
    - `RequestErasure`: Irreversibly pseudonymizes recipient addresses, wipes message bodies (`BodyRef`) and MIME artifacts (`RawMIMERef`), and replaces suppression addresses with salted one-way SHA-256 hashes (`erased-hash-...`) so the tenant cannot identify the cleartext email while preventing accidental re-sending (§20).
    - `RequestExport`: Aggregates all recipient delivery histories, engagement events, and active suppressions for a subject email.
  - `core/retention`:
    - `RetentionService`: Manages per-tenant retention policies and executes automated data purges.
    - `ExecutePurge`: Drops message and recipient rows past message retention cutoff, trims events past events retention cutoff, drops delivery attempts past attempts cutoff, and clears raw MIME references older than raw MIME cutoff.
  - `core/repository`: Added repository methods `ImportSuppressions`, `ExportSuppressions`, `CreatePrivacyRequest`, `GetPrivacyRequest`, `ListPrivacyRequests`, `ExecutePrivacyErasure`, `ExecutePrivacyExport`, `GetRetentionPolicy`, `UpdateRetentionPolicy`, `ExecuteRetentionPurge`.
  - `api/handlers`:
    - `POST /v1/suppressions/import`: Multipart CSV file upload or JSON payload bulk import with duplicate detection.
    - `GET /v1/suppressions/export`: CSV or JSON export with optional stream filter.
    - `POST /v1/privacy/erasure`: GDPR right-to-be-forgotten erasure endpoint.
    - `POST /v1/privacy/export`: GDPR data portability export endpoint.
    - `GET /v1/privacy/requests`: List logged privacy compliance requests.
    - `GET /v1/retention`: Retrieve tenant retention policy (with 30d/90d/30d/7d defaults).
    - `PUT /v1/retention`: Update retention policy.
    - `POST /v1/retention/purge`: Trigger immediate tenant data purge.
  - `api/server`: Mounted all Phase 9 routes with strict RBAC permission middleware (`suppressions.import`, `suppressions.read`, `privacy.erase`, `privacy.export`, `privacy.read`, `retention.read`, `retention.write`).
  - `console`:
    - **Suppressions Tab**: Added "Import CSV" modal with live preview/counters and "Export CSV" instant download.
    - **Privacy & Retention Tab**:
      - GDPR Article 17 Right-to-be-Forgotten erasure form.
      - GDPR Article 15 Data Portability export form with interactive JSON preview.
      - Data Retention policy form (Message, Events, Attempts, Raw MIME days).
      - Immediate purge execution button with live summary counter.
      - Privacy Requests Audit Log table.
  - `sdk/openapi.yaml`: Documented all Phase 9 endpoints.
- **Verification & Test Results:**
  - `core/privacy/privacy_test.go`:
    - `TestPrivacyExportAndErasure` PASS (verified export compiles messages/events/suppressions, erasure pseudonymizes recipient addresses, wipes message bodies and raw MIME blobs, and converts suppression entries to one-way hashes).
  - `core/retention/retention_test.go`:
    - `TestRetentionPolicyAndPurge` PASS (verified default policies, policy update validation, and automated purge removing stale messages/events past cutoff while keeping fresh records intact).
  - `api/privacy_retention_test.go`:
    - `TestSuppressionImportAndExportEndpoints` PASS (multipart CSV upload, duplicate tracking, CSV download streaming).
    - `TestPrivacyErasureAndExportEndpoints` PASS (GDPR erasure and export HTTP endpoints, audit logging).
    - `TestRetentionPolicyAndPurgeEndpoints` PASS (GET/PUT retention policies, immediate purge execution).
  - `go test -count=1 ./...`: 100% PASS across all 24 Go packages.
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 10: Open/Click Tracking Engine & Inbound Mail Routing
- **Status:** Completed & Verified (2026-09-21)
- **Implementations:**
  - `migrations/000010_tracking_and_inbound`: Added tables `tracking_links`, `tracking_events`, `inbound_routes`, and `inbound_messages`.
  - `core/domain`: Added models `TrackingLink`, `TrackingEvent`, `InboundRoute`, and `InboundMessage`.
  - `core/crypto/ulid`: Added prefixes `tl_` (`PrefixTrackingLink`), `te_` (`PrefixTrackingEvent`), `inr_` (`PrefixInboundRoute`), and `inm_` (`PrefixInboundMessage`).
  - `core/tracking`:
    - `TrackingEngine`: RFC 2046 1x1 transparent GIF generation (`Cache-Control: no-store, no-cache, must-revalidate, max-age=0`), signed HMAC tokens for open tracking and click tracking with expiration check.
    - Anti-Open Redirect Defense (§19): Target URL strictly resolved from `tracking_links` repository table by `link_id`, NEVER accepted directly from query or URL path parameters. Tampered or expired tokens return HTTP 404.
    - Bot Prefetch Filtering (§19): Identifies Apple Mail Privacy Protection (`Apple-Mail-Privacy-Protection`), GoogleImageProxy (`GoogleImageProxy`), Microsoft SafeLinks, common vulnerability scanner user-agents, and automated hits occurring within < 2 seconds of delivery. Flagged with `is_bot = true` and `bot_reason` recorded in `tracking_events`.
    - IP Pseudonymization (§20): Privacy-first IP masking (/24 for IPv4, /48 for IPv6) before logging.
    - HTML Processing: Link rewriting (`href="..."` replaced with signed tracking redirect) and open pixel injection before closing `</body>` tag or appended to HTML.
  - `core/inbound`:
    - `InboundRouter`: Evaluates inbound routes sorted by `priority ASC` (lowest number first).
    - Precedence Engine (§21): Matches recipient address against `exact` match, `wildcard` pattern (e.g. `support-*@help.com`), and `catch_all` domain rule (e.g. `@help.com`).
    - Skipping inactive routes (`is_active = false`), storing inbound messages with SPF, DKIM, and DMARC verification verdicts.
  - `core/repository`: Added repository methods and thread-safe implementations for tracking links/events and inbound routes/messages.
  - `api/handlers`:
    - `GET /o/{token}`: Transparent 1x1 GIF open tracking pixel.
    - `GET /c/{token}`: Click tracking link redirect with safe repository URL lookup.
    - `GET /v1/inbound/routes` & `POST /v1/inbound/routes`: Inbound routing rule CRUD.
    - `GET /v1/inbound/routes/{id}`, `PATCH /v1/inbound/routes/{id}`, `DELETE /v1/inbound/routes/{id}`: Route management.
    - `POST /v1/inbound/routes/test`: Precedence engine route simulation and match testing.
    - `GET /v1/inbound/messages` & `GET /v1/inbound/messages/{id}`: Inbound email log retrieval.
  - `api/server`: Mounted all Phase 10 endpoints with RBAC middleware (`tracking.read`, `inbound.routes`, `inbound.read`, `inbound.write`).
  - `console`:
    - **Inbound Routing Tab**:
      - Precedence Engine Route Simulator & Match Tester (`POST /v1/inbound/routes/test`).
      - Inbound Route creation form with Priority, Match Type (`exact`, `wildcard`, `catch_all`), Pattern, Action (`webhook`, `store_only`, `forward`, `discard`), and Target.
      - Inbound Routing Rules management table with live active/disabled toggle and deletion.
      - Inbound Mail Logs table with SPF/DKIM/DMARC authentication verification badges.
  - `sdk/openapi.yaml`: Documented all Phase 10 tracking and inbound endpoints.
- **Verification & Test Results:**
  - `core/tracking/tracking_test.go`:
    - `TestOpenPixelAndTokenValidation` PASS (HMAC token verification, 1x1 transparent GIF response).
    - `TestClickTrackingSafeRedirect` PASS (safe repository URL lookup, tampered token 404 rejection).
    - `TestBotAndPrefetchFiltering` PASS (Apple MPP, GoogleImageProxy, instant hits < 2s flagged as bot).
    - `TestHTMLInjectionAndRewriting` PASS (link rewriting and pixel injection).
  - `core/inbound/inbound_test.go`:
    - `TestInboundRoutePrecedence` PASS (exact > wildcard > catch_all precedence, priority order, inactive route skipping).
    - `TestInboundMessageStorage` PASS (inbound message storage with SPF/DKIM/DMARC verdicts).
  - `api/tracking_inbound_test.go`:
    - `TestTrackingEndpoints` PASS (open pixel, click redirect, anti-open redirect defense).
    - `TestInboundRoutingEndpoints` PASS (route CRUD, match simulation, inbound message listing).
  - `go test -count=1 ./...`: 100% PASS across all 24 Go packages.
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 11: Reputation, DNSBL, Warmup Engine, Alert Rules & Multi-Channel Notifications
- **Status:** Completed & Verified
- **Components Implemented:**
  - `migrations/000011_reputation_dnsbl_warmup_alerts.up.sql` & `down.sql`:
    - Tables: `dnsbl_providers`, `dnsbl_results`, `warmup_plans`, `warmup_plan_days`, `ip_warmup_states`, `alert_rules`, `alert_incidents`, `notification_channels`, `alert_rule_channels`, `alert_notifications`.
  - `core/crypto/ulid.go`:
    - Prefixes: `PrefixDNSBLProvider` (`dnsbl_`), `PrefixDNSBLResult` (`dres_`), `PrefixWarmupPlan` (`wplan_`), `PrefixWarmupPlanDay` (`wpday_`), `PrefixIPWarmupState` (`wstate_`), `PrefixAlertRule` (`arule_`), `PrefixAlertIncident` (`inc_`), `PrefixNotifChannel` (`nchan_`), `PrefixAlertNotif` (`anot_`).
  - `core/domain/models.go`:
    - Domain models: `DNSBLProvider`, `DNSBLResult`, `WarmupPlan`, `WarmupPlanDay`, `IPWarmupState`, `AlertRule`, `AlertIncident`, `NotificationChannel`, `AlertNotification`, `ReputationOverview`.
  - `core/dnsbl`:
    - DNSBL lookup engine with IPv4 (reverse octet) and IPv6 (reverse nibble 32-label) address construction.
    - Return code interpretation: `127.0.0.2` (direct spammer), `127.0.0.3` (dialup/open relay), `127.0.0.4` (open proxy), `127.0.0.10-11` (PBL/end-user IP).
    - Provider delist URL generator and NXDOMAIN clean status verification.
  - `core/warmup`:
    - Staged warmup ramp engine (§11.5) with automated volume doubling schedule (Day 1: 50/day up to Day 30: 1,000,000/day).
    - Per-provider daily limits (Google, Microsoft, Yahoo/AOL).
    - Automated hold enforcement: freezes IP ramp if rolling bounce rate > 5% or complaint rate > 0.1%. Manual advance, force advance, hold, and resume controls.
  - `core/alerts`:
    - Multi-channel notification dispatcher (§22.3) supporting Slack incoming webhooks, Microsoft Teams connectors, PagerDuty Events API v2, signed generic HTTP webhooks (HMAC-SHA256), and in-app notifications.
    - Threshold evaluation engine for bounce rates, complaint rates, DNSBL blocklistings, queue depths, and delivery rate degradation.
    - Cooldown deduplication preventing notification storms and automated incident auto-resolution when metrics recover.
  - `core/repository`:
    - Thread-safe repository methods with pre-seeded providers (Spamhaus ZEN, Barracuda BRBL, SpamCop, UCEPROTECT-1) and default 30-day warmup ramp plan.
    - RBAC permissions seeded: `reputation.read`, `dnsbl.read`, `dnsbl.check`, `warmup.read`, `warmup.write`, `alerts.read`, `alerts.write`.
  - `api/handlers`:
    - `GET /v1/reputation/overview`: Health status summary (`blocked`, `warning`, `measured`, `unknown`).
    - `GET /v1/dnsbl/providers`, `POST /v1/dnsbl/check`, `GET /v1/dnsbl/results`: DNSBL inspection.
    - `GET /v1/warmup/plans`, `POST /v1/warmup/plans`, `GET /v1/warmup/plans/{id}`: Warmup schedules.
    - `GET /v1/warmup/ips`, `POST /v1/warmup/ips`, `POST /v1/warmup/ips/{id}/advance`, `POST /v1/warmup/ips/{id}/hold`, `POST /v1/warmup/ips/{id}/resume`: Warmup lifecycle.
    - `GET /v1/alerts/rules`, `POST /v1/alerts/rules`, `DELETE /v1/alerts/rules/{id}`, `POST /v1/alerts/evaluate`: Rules and on-demand trigger.
    - `GET /v1/alerts/incidents`, `POST /v1/alerts/incidents/{id}/acknowledge`, `POST /v1/alerts/incidents/{id}/resolve`: Incident triage.
    - `GET /v1/alerts/channels`, `POST /v1/alerts/channels`, `DELETE /v1/alerts/channels/{id}`, `POST /v1/alerts/channels/{id}/test`: Multi-channel dispatch.
    - `GET /v1/alerts/notifications`, `POST /v1/alerts/notifications/{id}/read`: In-app notification center.
  - `api/server`: Mounted all Phase 11 endpoints with RBAC middleware.
  - `console`: Added **Alerts, DNSBL & Warmup** tab featuring:
    - Reputation overview cards and health badge indicators.
    - On-demand DNSBL blocklist monitor with status indicators and delist portal links.
    - IP warmup ramp visual progress bar, provider caps, and advance/hold controls.
    - Alert Rules form with cooldown duration, active rules table, and Incidents Log with acknowledge/resolve actions.
    - Multi-channel notification management with test dispatch.
    - In-app notification feed with read/unread tracking.
  - `sdk/openapi.yaml`: Complete OpenAPI 3.0 path definitions for all Phase 11 endpoints.
- **Verification & Test Results:**
  - `core/dnsbl/dnsbl_test.go`:
    - `TestReverseIPFormat` PASS (IPv4 reverse octet and IPv6 reverse nibble formatting).
    - `TestDNSBLResolution` PASS (DNSBL lookup, return code interpretation, NXDOMAIN clean detection).
  - `core/warmup/warmup_test.go`:
    - `TestIPWarmupLifecycleAndLimits` PASS (enrollment, daily caps, provider caps, automated hold on bounce > 5%, force advance, manual hold/resume, completion).
  - `core/alerts/alerts_test.go`:
    - `TestMultiChannelAlertDispatch` PASS (Slack, Teams, PagerDuty Events v2, HMAC signed generic webhook, in-app notifications).
    - `TestAlertRuleEvaluationAndResolution` PASS (threshold firing, cooldown suppression, auto-resolution).
  - `api/reputation_alerts_test.go`:
    - `TestReputationAndDNSBLEndpoints` PASS (overview stats, DNSBL check and result listing).
    - `TestWarmupEndpoints` PASS (plans, IP enrollment, advance, hold, resume).
    - `TestAlertRulesAndChannelsEndpoints` PASS (rule CRUD, evaluate, channel test, notification read).
  - `go test -count=1 ./...`: 100% PASS across all 27 Go packages.
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---

### Phase 12: SSO (OIDC/SAML), Platform Admin Impersonation, Support Body Masking, FCrDNS/TLS Checks & Deliverability Compliance (§7.2, §7.4, §22)
- **Status:** Completed (2026-09-21)
- **Key Modules & Artifacts Implemented:**
  - `migrations/000012_sso_impersonation_compliance.up.sql` & `down.sql`:
    - `sso_providers`: Org & tenant level identity provider configurations, OIDC/SAML metadata URLs, allowed auto-join domains array, default JIT role, and strict SSO enforcement flag.
    - `impersonation_sessions`: Audit trail of platform admin tenant impersonations with mandatory reason justification, actor identity, access type, support role privacy masking status, and start/end timestamps.
    - `compliance_checks`: Automated 100-point deliverability scorecard records and 7-pillar breakdown logs.
  - `core/domain/models.go` & `core/crypto/ulid.go`:
    - Models `SSOProvider`, `ImpersonationSession`, `ComplianceCheck`, `FCrDNSResult`, `TLSCheckResult`.
    - ULID prefixes: `sso_`, `imps_`, `comp_`.
  - `core/auth/`:
    - `sso.go`: OpenID Connect (OIDC 1.0) authorization URL generator, token callback handler, SAML 2.0 Service Provider EntityDescriptor XML generator, AuthnRequest generator, SAML Response assertion parser, Just-in-Time (JIT) provisioning, and domain auto-join validation.
    - `impersonation.go`: Platform admin impersonation session manager, mandatory reason validation, immutable audit log recorder, and Support role message body privacy masking engine (`MaskMessageBody`).
    - Unit tests: `sso_test.go`, `impersonation_test.go` (100% PASS).
  - `core/health/`:
    - `compliance.go`: Bidirectional FCrDNS verification engine (IP -> PTR query -> A/AAAA query matching original IP), remote MTA STARTTLS & cipher suite probe, and automated 100-point deliverability compliance checklist evaluator (SPF, DKIM, DMARC, FCrDNS, TLS, DNSBL, RFC 8058 Unsubscribe).
    - Unit tests: `compliance_test.go` (100% PASS).
  - `core/repository/`:
    - Repository methods for SSO provider CRUD, domain auto-join resolution, impersonation session management with audit logging, and compliance check records.
    - Seeded standard tenant roles: `admin`, `developer`, `analyst`, `support`, `viewer` (with `support` intentionally lacking `messages.body.read` permission for §7.4 masking).
  - `api/middleware/middleware.go`:
    - Dynamic impersonation context injection via `X-Impersonate-Tenant-ID` and `X-Impersonate-Reason` headers.
    - Context keys: `IsImpersonatingKey`, `ImpersonatedTenantIDKey`, `ImpersonatorUserIDKey`, `IsSupportRoleKey` (when `X-Support-Mode: true` or user role is `support`). Response header `X-Impersonating: true`.
  - `api/handlers/messages.go`:
    - Updated `GetMessageBodyHandler` to return `[MASKED - SUPPORT ROLE ACCESS]` when `IsSupportRoleKey` is active.
  - `api/handlers/sso_impersonation.go`:
    - Handlers for SSO provider CRUD, SAML SP metadata (`/v1/auth/sso/saml/metadata`), SAML ACS (`/v1/auth/sso/saml/acs`), OIDC callback (`/v1/auth/sso/oidc/callback`), Impersonation start/exit/sessions, Compliance audit (`/v1/compliance/check`), FCrDNS (`/v1/health/fcrdns`), and TLS (`/v1/health/tls`).
  - `api/server.go`: Wired Phase 12 engines and mounted all endpoints with RBAC & platform admin authentication.
  - `console`: Added **SSO & Compliance** tab and impersonation UX:
    - Impersonation session modal and sticky banner (`🔒 IMPERSONATING TENANT [name]`).
    - Message Inspector body preview shows privacy notice when viewed under support role or support impersonation.
    - Single Sign-On (OIDC/SAML) provider registration form and active providers table.
    - SP Metadata XML download link.
    - Impersonation sessions audit trail table.
    - Deliverability Compliance Checklist score gauge (0-100) and 7-pillar breakdown cards (SPF, DKIM, DMARC, FCrDNS, TLS, DNSBL, Unsubscribe).
    - Live FCrDNS bidirectional verification probe tool.
  - `sdk/openapi.yaml`: Complete OpenAPI 3.0 path definitions for all Phase 12 endpoints.
- **Verification & Test Results:**
  - `core/auth/sso_test.go`:
    - `TestSSOOIDCAuthorizationURL` PASS.
    - `TestSSOSAMLMetadataGeneration` PASS (XML structure, EntityDescriptor, AssertionConsumerService binding).
    - `TestSSOSAMLAuthnRequest` PASS.
    - `TestSSOProvisionJITUserWithDomainAutoJoin` PASS.
  - `core/auth/impersonation_test.go`:
    - `TestPlatformAdminImpersonationLifecycle` PASS (start session with mandatory reason, audit trail, end session).
    - `TestSupportRoleMessageBodyMasking` PASS (privacy redaction for support role).
  - `core/health/compliance_test.go`:
    - `TestFCrDNSBidirectionalVerification` PASS (PTR -> A match, mismatch detection).
    - `TestDeliverabilityComplianceChecklist` PASS (100-point scoring algorithm, pillar evaluations).
    - `TestTLSVerification` PASS.
  - `api/sso_impersonation_test.go`:
    - `TestSSOEndpoints` PASS (create provider, list, get, delete, SAML metadata XML, OIDC callback).
    - `TestPlatformAdminImpersonationEndpoints` PASS (start impersonation, audit listing, body masking in message inspector, exit).
    - `TestComplianceAndHealthEndpoints` PASS (compliance checklist check, FCrDNS check, TLS check).
  - `go test -count=1 ./...`: 100% PASS across all 28 Go packages.
  - `go vet ./...`: 0 warnings, clean.
  - `console`: Production build succeeded (`npm run build`).

---







