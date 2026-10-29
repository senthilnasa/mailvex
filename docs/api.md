# Mailvex API & AI Agent Integration Specification

> **Version:** 1.0.0  
> **Status:** Production / Carrier-Grade  
> **Target Audience:** Parallel AI Agents, Autonomous Systems, and Software Engineers  
> **Base URLs:**
> - Local Dev / Homelab: `http://localhost:8080` (API) & `localhost:2525` (SMTP)
> - Docker / Staging: `http://mailvex-api:8080`
> - Production: `https://api.mailvex.yourdomain.com`

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [AI Agent & Parallel System Integration Protocol](#2-ai-agent--parallel-system-integration-protocol)
   - [Agent System Prompt & Instructions](#21-agent-system-prompt--context-injection)
   - [Parallel Execution & Concurrency Guarantees](#22-parallel-execution--concurrency-guarantees)
   - [AI Agent Tool Definitions (JSON Schema / MCP)](#23-ai-agent-tool-definitions-mcp--json-schema)
3. [Authentication, Headers & Tenancy](#3-authentication-headers--tenancy)
   - [API Key Authentication](#31-api-key-authentication)
   - [User Session & Console Authentication](#32-user-session--console-authentication)
   - [Multi-Tenancy & Platform Impersonation](#33-multi-tenancy--platform-impersonation)
   - [Standard Request & Response Headers](#34-standard-request--response-headers)
4. [Email Dispatch API](#4-email-dispatch-api)
   - [Send Single Email (`POST /v1/email` or `POST /v1/send`)](#41-send-single-email)
   - [Send Batch Email (`POST /v1/email/batch` or `POST /v1/send/batch`)](#42-send-batch-email)
   - [Idempotency & Replay Semantics](#43-idempotency--replay-semantics)
5. [Message Tracking, Inspection & Sink](#5-message-tracking-inspection--sink)
   - [List Messages (`GET /v1/messages`)](#51-list-messages)
   - [Message Details, Events & Raw Body](#52-message-details-events--raw-body)
   - [Retry & Cancel Message](#53-retry--cancel-message)
   - [Development Message Sink (`GET /v1/messages/development`)](#54-development-message-sink)
6. [Domains & DKIM Lifecycle](#6-domains--dkim-lifecycle)
   - [Add Domain (`POST /v1/domains`)](#61-add-domain)
   - [Verify DNS & Ownership (`POST /v1/domains/{id}/verify`)](#62-verify-dns--ownership)
   - [Rotate DKIM Key (`POST /v1/domains/{id}/dkim/rotate`)](#63-rotate-dkim-key)
7. [Message Streams & Templates](#7-message-streams--templates)
8. [Suppressions, Bounces & Unsubscribes](#8-suppressions-bounces--unsubscribes)
9. [Webhooks Engine & Signature Verification](#9-webhooks-engine--signature-verification)
10. [Inbound Email Routing](#10-inbound-email-routing)
11. [Reputation, DNSBL & Warmup](#11-reputation-dnsbl--warmup)
12. [SMTP Edge Submission Protocol](#12-smtp-edge-submission-protocol)
13. [Multi-Language Client Integration](#13-multi-language-client-integration)
    - [Go Official SDK](#131-go-official-sdk)
    - [Python (HTTPX / Requests)](#132-python-httpx--requests)
    - [Node.js / TypeScript](#133-nodejs--typescript)
    - [cURL](#134-curl)
    - [PHP & Laravel Official SDK](#135-php--laravel-official-sdk)
14. [Canonical Error Responses](#14-canonical-error-responses)

---

## 1. System Overview

**Mailvex** is a high-throughput, multi-tenant email infrastructure engine. It exposes dual ingestion edges:
- **REST API (`/v1`)**: RESTful JSON endpoints for programmatic dispatch, domain configuration, deliverability telemetry, and webhook subscriptions.
- **SMTP Submission**: RFC 5321 edge supporting ports `25`, `587`, and `2525` with SASL authentication and STARTTLS.

### Identifier Prefix Conventions
All system entities use millisecond-sortable ULIDs (26 uppercase Crockford Base32 characters) with distinct prefixes:

| Entity | Prefix | Example |
|---|---|---|
| Organization | `org_` | `org_01HZX876543210ABCDEFGHJKMN` |
| Tenant | `tenant_` | `tenant_01HZX876543210ABCDEFGHJK` |
| User | `usr_` | `usr_01HZX876543210ABCDEFGHJKMN` |
| API Key | `key_` | `key_01HZX876543210ABCDEFGHJKMN` |
| Domain | `dom_` | `dom_01HZX876543210ABCDEFGHJKMN` |
| Message Stream | `stm_` | `stm_01HZX876543210ABCDEFGHJKMN` |
| Message | `msg_` | `msg_01HZX876543210ABCDEFGHJKMN` |
| Request ID | `req_` | `req_01HZX876543210ABCDEFGHJKMN` |
| Webhook | `whk_` | `whk_01HZX876543210ABCDEFGHJKMN` |

---

## 2. AI Agent & Parallel System Integration Protocol

When running as an autonomous AI agent (or a swarm of parallel agents), you can execute automated workflows such as:
1. Sending transactional alerts or reports based on parallel trigger events.
2. Checking message delivery status, hard/soft bounces, and spam complaints.
3. Managing suppressions to protect domain reputation.
4. Auditing domain DNS compliance (DKIM, SPF, DMARC, FCrDNS).

### 2.1 Agent System Prompt & Context Injection
Copy and inject the following system instructions into your parallel agent’s prompt context:

```markdown
You are an autonomous AI Agent integrated with Mailvex Email Infrastructure.
When interacting with the Mailvex API:
1. BASE URL: Default to "http://localhost:8080" or the value of $MAILVEX_API_URL.
2. AUTHENTICATION: Always pass "Authorization: Bearer <API_KEY>" in request headers.
   - Use test keys (prefixed `mvx_test_`) for sandboxed dry-runs (messages sent to dev sink).
   - Use live keys (prefixed `mvx_live_`) for actual delivery.
3. CONCURRENCY & IDEMPOTENCY:
   - When executing parallel operations, ALWAYS supply a unique "Idempotency-Key" header (UUIDv4 or ULID).
   - If Mailvex responds with HTTP 409 and code "idempotency_in_progress", wait 500ms and retry.
   - If Mailvex responds with HTTP 429 (rate limited), respect the "RateLimit-Reset" header with exponential backoff and jitter.
4. RECIPIENT INTEGRITY:
   - Never send to more than 50 recipients in a single `POST /v1/email` call; use `POST /v1/email/batch` for up to 500 recipients.
   - Custom headers MUST start with `X-`, `List-Id`, or `Precedence`. Never inject CRLF characters.
5. SINK AUDITING:
   - In test mode, query `GET /v1/messages/development` to inspect captured test emails without sending external traffic.
```

### 2.2 Parallel Execution & Concurrency Guarantees
- **Atomic Idempotency Locking**: When parallel agents trigger requests with the same `Idempotency-Key`, Mailvex acquires an atomic lock. The first request processes while concurrent duplicates receive `409 Conflict (idempotency_in_progress)` or the cached final response once completed.
- **Tenant Isolation**: All operations are strictly partitioned by `tenant_id`. Parallel agents operating across different tenants are physically isolated cryptographically and logically.
- **Development Sink**: Parallel test runs using `mvx_test_...` keys capture emails in memory at `GET /v1/messages/development`, allowing agents to assert delivery assertions in automated CI/CD pipelines without polluting live queues or external mailboxes.

---

### 2.3 AI Agent Tool Definitions (MCP / JSON Schema)

Below are machine-readable tool schemas for function calling (OpenAI, Anthropic Claude, Gemini, or Model Context Protocol / MCP):

#### Tool 1: `mailvex_send_email`
```json
{
  "name": "mailvex_send_email",
  "description": "Send a transactional or broadcast email via Mailvex REST API.",
  "parameters": {
    "type": "object",
    "required": ["from", "to", "subject"],
    "properties": {
      "from": {
        "type": "string",
        "description": "Sender email address (must match a verified domain, e.g., 'alerts@example.com')"
      },
      "to": {
        "type": "array",
        "items": { "type": "string" },
        "description": "List of recipient email addresses (max 50)"
      },
      "subject": {
        "type": "string",
        "description": "Email subject line"
      },
      "text": {
        "type": "string",
        "description": "Plain text body content"
      },
      "html": {
        "type": "string",
        "description": "HTML body content"
      },
      "template_id": {
        "type": "string",
        "description": "Optional template ID (e.g., 'tmpl_01HZX8765...')"
      },
      "template_variables": {
        "type": "object",
        "description": "Key-value variables for template interpolation"
      },
      "stream": {
        "type": "string",
        "enum": ["transactional", "broadcast"],
        "description": "Message stream (defaults to 'transactional')"
      },
      "idempotency_key": {
        "type": "string",
        "description": "Unique key to ensure idempotent delivery in parallel agent environments"
      }
    }
  }
}
```

#### Tool 2: `mailvex_get_message_status`
```json
{
  "name": "mailvex_get_message_status",
  "description": "Inspect message delivery state, timeline events, and bounce details.",
  "parameters": {
    "type": "object",
    "required": ["message_id"],
    "properties": {
      "message_id": {
        "type": "string",
        "description": "ULID message ID (e.g. 'msg_01HZX...')"
      }
    }
  }
}
```

#### Tool 3: `mailvex_list_messages`
```json
{
  "name": "mailvex_list_messages",
  "description": "Query recent messages with optional status or recipient filters.",
  "parameters": {
    "type": "object",
    "properties": {
      "status": {
        "type": "string",
        "enum": ["queued", "delivering", "delivered", "bounced", "failed", "cancelled"]
      },
      "recipient": { "type": "string" },
      "limit": { "type": "integer", "default": 20 }
    }
  }
}
```

#### Tool 4: `mailvex_check_reputation`
```json
{
  "name": "mailvex_check_reputation",
  "description": "Retrieve tenant reputation score, bounce rates, complaint metrics, and DNSBL status.",
  "parameters": {
    "type": "object",
    "properties": {}
  }
}
```

---

## 3. Authentication, Headers & Tenancy

### 3.1 API Key Authentication
API keys provide programmatic access for parallel agents and background services:
- **Header**: `Authorization: Bearer <API_KEY>`
- **Live Key Prefix**: `mvx_live_<32_random_hex>` (Sends live outbound messages)
- **Test Key Prefix**: `mvx_test_<32_random_hex>` (Diverts messages to in-memory development sink)

Example:
```http
Authorization: Bearer mvx_live_a89f30b2c1409f921df048bbef8001a1
```

### 3.2 User Session & Console Authentication
For management scripts acting as administrative console users:
- **Login**: `POST /v1/auth/login`
  ```json
  {
    "email": "admin@example.com",
    "password": "Password123!"
  }
  ```
  Returns `200 OK` with user details and sets session cookie or provides `X-User-ID` header.
- **Me Endpoint**: `GET /v1/auth/me`

### 3.3 Multi-Tenancy & Platform Impersonation
- **Explicit Tenant Context**: Pass `X-Tenant-ID: <tenant_id>` to scope requests to a specific tenant workspace.
- **Platform Admin Impersonation**: Platform administrators can inspect or resolve issues on behalf of another tenant:
  - `X-Impersonate-Tenant-ID: <target_tenant_id>`
  - `X-Impersonate-Reason: "Diagnosing bounce alert incident #412"`
  - `X-Support-Mode: true` (Enforces PII email body masking in message inspector)

### 3.4 Standard Request & Response Headers

| Header | Description | Direction |
|---|---|---|
| `Authorization` | `Bearer mvx_live_...` or `Bearer mvx_test_...` | Request |
| `Idempotency-Key` or `X-Idempotency-Key` | Unique request token (cached for 24 hours) | Request |
| `X-Request-ID` | Tracing token (auto-generated if omitted) | Request & Response |
| `X-Tenant-ID` | Target tenant ULID | Request & Response |
| `RateLimit-Limit` | Maximum requests permitted per minute | Response |
| `RateLimit-Remaining` | Remaining request quota in current window | Response |
| `RateLimit-Reset` | Unix timestamp when the quota resets | Response |

---

## 4. Email Dispatch API

### 4.1 Send Single Email
**Endpoint:** `POST /v1/email` (Alias: `POST /v1/send`)  
**Scope Required:** `send.email`

#### Request Headers
```http
POST /v1/email HTTP/1.1
Host: localhost:8080
Authorization: Bearer mvx_live_...
Content-Type: application/json
Idempotency-Key: 9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d
```

#### Request Payload
```json
{
  "from": "Security Alert <alerts@mycompany.com>",
  "to": ["alice@example.com"],
  "cc": ["auditor@example.com"],
  "bcc": ["archive@mycompany.com"],
  "reply_to": "support@mycompany.com",
  "subject": "Action Required: Security Key Rotation",
  "text": "Hello Alice,\nYour API key key_01HZX... is scheduled for rotation.",
  "html": "<html><body><h2>Security Notice</h2><p>Your API key <code>key_01HZX...</code> is scheduled for rotation.</p></body></html>",
  "stream": "transactional",
  "tags": ["security", "key-rotation"],
  "metadata": {
    "user_id": "usr_99812",
    "agent_job_id": "job_parallel_042"
  },
  "track_opens": true,
  "track_clicks": true,
  "attachments": [
    {
      "filename": "security-guidelines.pdf",
      "content_type": "application/pdf",
      "content": "JVBERi0xLjQKJcOkw7zDtsOfCDCm..." 
    }
  ],
  "headers": {
    "X-Agent-Origin": "autonomous-worker-4",
    "List-Id": "<alerts.mycompany.com>"
  }
}
```

> [!NOTE]
> `attachments[].content` must be Base64-encoded. Total attachment size across all attachments must not exceed 20MB.

#### Response (`202 Accepted`)
```json
{
  "id": "msg_01HZX876543210ABCDEFGHJKMN",
  "status": "queued",
  "environment": "live",
  "request_id": "req_01HZX876543210ABCDEFGHJKMN"
}
```

---

### 4.2 Send Batch Email
**Endpoint:** `POST /v1/email/batch` (Alias: `POST /v1/send/batch`)  
**Scope Required:** `send.email`  
**Limit:** Up to 500 recipients per batch call.

#### Request Payload
```json
{
  "from": "Weekly Digest <news@mycompany.com>",
  "subject": "Your Weekly Performance Digest",
  "stream": "broadcast",
  "template_id": "tmpl_01HZX876543210ABCDEFGHJKMN",
  "recipients": [
    {
      "to": "alice@example.com",
      "template_variables": { "name": "Alice", "score": "98%" },
      "metadata": { "account_tier": "enterprise" }
    },
    {
      "to": "bob@example.com",
      "template_variables": { "name": "Bob", "score": "91%" },
      "metadata": { "account_tier": "standard" }
    }
  ]
}
```

#### Response (`200 OK`)
```json
{
  "batch_id": "batch_01HZX876543210ABCDEFGHJK",
  "total": 2,
  "accepted": 2,
  "failed": 0,
  "message_ids": [
    "msg_01HZX876543210ABCDEF001",
    "msg_01HZX876543210ABCDEF002"
  ]
}
```

---

### 4.3 Idempotency & Replay Semantics
When sending emails under parallel execution, pass the `Idempotency-Key` (or `X-Idempotency-Key`) header:
1. **First Request**: Mailvex hashes the payload with SHA-256 and records the key as `in_progress`.
2. **Concurrent Re-entrant Request**: If another parallel worker sends the exact same key while the first is running, Mailvex returns:
   ```json
   HTTP/1.1 409 Conflict
   {"error":{"code":"idempotency_in_progress","message":"A request with this idempotency key is currently processing"}}
   ```
3. **Payload Mismatch with Same Key**: If an agent reuses a key with different parameters, Mailvex aborts:
   ```json
   HTTP/1.1 409 Conflict
   {"error":{"code":"idempotency_key_reused","message":"Idempotency key reused with differing request payload"}}
   ```
4. **Completed Replay**: If the operation already succeeded, Mailvex returns the exact cached HTTP status code and response body without creating a duplicate outbound message.

---

## 5. Message Tracking, Inspection & Sink

### 5.1 List Messages
**Endpoint:** `GET /v1/messages`  
**Query Parameters:**
- `status`: Filter by status (`queued`, `delivering`, `delivered`, `bounced`, `failed`, `cancelled`).
- `recipient`: Partial/exact match on recipient email.
- `from`: Filter by sender address.
- `stream`: Filter by message stream (e.g. `transactional`, `broadcast`).
- `tag`: Filter by tag.
- `since`: ISO 8601 / RFC 3339 start timestamp (`2026-10-01T00:00:00Z`).
- `until`: ISO 8601 / RFC 3339 end timestamp.
- `limit`: Number of records (1-100, default 50).
- `cursor`: ULID cursor for next page.

#### Example Request
```http
GET /v1/messages?status=bounced&limit=10 HTTP/1.1
Authorization: Bearer mvx_live_...
```

#### Response (`200 OK`)
```json
{
  "data": [
    {
      "id": "msg_01HZX876543210ABCDEFGHJKMN",
      "tenant_id": "tenant_01HZX876543210ABCDEFGHJK",
      "stream_id": "stm_01HZX876543210ABCDEFGHJKMN",
      "from_address": "alerts@mycompany.com",
      "subject": "Action Required: Security Key Rotation",
      "status": "bounced",
      "environment": "live",
      "created_at": "2026-10-28T09:12:00Z",
      "updated_at": "2026-10-28T09:12:05Z"
    }
  ],
  "next_cursor": "msg_01HZX876543210ABCDEFGHJKMN",
  "has_more": false
}
```

---

### 5.2 Message Details, Events & Raw Body

#### Get Single Message
```http
GET /v1/messages/msg_01HZX876543210ABCDEFGHJKMN HTTP/1.1
Authorization: Bearer mvx_live_...
```
Returns message metadata and recipient delivery states.

#### Get Chronological Delivery Events
```http
GET /v1/messages/msg_01HZX876543210ABCDEFGHJKMN/events HTTP/1.1
Authorization: Bearer mvx_live_...
```
Response:
```json
{
  "data": [
    {
      "id": "evt_01HZX876543210ABCDEFGHJK01",
      "message_id": "msg_01HZX876543210ABCDEFGHJKMN",
      "type": "queued",
      "details": "Message accepted via REST API into transactional queue",
      "created_at": "2026-10-28T09:12:00Z"
    },
    {
      "id": "evt_01HZX876543210ABCDEFGHJK02",
      "message_id": "msg_01HZX876543210ABCDEFGHJKMN",
      "type": "delivering",
      "details": "Connecting to MX mx1.example.com [203.0.113.25:25]",
      "created_at": "2026-10-28T09:12:02Z"
    },
    {
      "id": "evt_01HZX876543210ABCDEFGHJK03",
      "message_id": "msg_01HZX876543210ABCDEFGHJKMN",
      "type": "bounced",
      "details": "550 5.1.1 User unknown",
      "created_at": "2026-10-28T09:12:05Z"
    }
  ]
}
```

#### Get Message Raw Body
```http
GET /v1/messages/msg_01HZX876543210ABCDEFGHJKMN/body HTTP/1.1
Authorization: Bearer mvx_live_...
```
*(Requires `messages.body.read` permission. If accessed under Platform Support mode, personal identifiable information and email contents are masked)*.

---

### 5.3 Retry & Cancel Message
- **Retry Failed Message:** `POST /v1/messages/{id}/retry`  
  Re-queues a failed or bounced message for outbound delivery.
- **Cancel Pending Message:** `POST /v1/messages/{id}/cancel`  
  Cancels a message currently in `queued` state before worker pickup.

---

### 5.4 Development Message Sink
**Endpoint:** `GET /v1/messages/development`  
When using a test API key (`mvx_test_...`), outbound messages do not hit external MX servers. Instead, they are routed to the in-memory sink for agent assertion.

```http
GET /v1/messages/development?limit=20 HTTP/1.1
Authorization: Bearer mvx_test_...
```
Returns list of captured emails including rendered HTML and plain text.

---

## 6. Domains & DKIM Lifecycle

### 6.1 Add Domain
**Endpoint:** `POST /v1/domains`  
**Payload:**
```json
{
  "name": "mail.mycompany.com",
  "type": "public"
}
```
*(Use `"type": "local"` for internal homelab domains like `home.lan` which auto-verify without public DNS).*

**Response:**
```json
{
  "id": "dom_01HZX876543210ABCDEFGHJKMN",
  "name": "mail.mycompany.com",
  "type": "public",
  "status": "pending",
  "ownership_token": "a1b2c3d4e5f6789012345678abcdef01",
  "dns_records": [
    {
      "type": "TXT",
      "name": "_mailvex.mail.mycompany.com",
      "value": "mailvex-verify=a1b2c3d4e5f6789012345678abcdef01",
      "purpose": "ownership"
    },
    {
      "type": "TXT",
      "name": "mvx1._domainkey.mail.mycompany.com",
      "value": "v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A...",
      "purpose": "dkim"
    },
    {
      "type": "TXT",
      "name": "mail.mycompany.com",
      "value": "v=spf1 include:_spf.mailvex.com ~all",
      "purpose": "spf"
    },
    {
      "type": "TXT",
      "name": "_dmarc.mail.mycompany.com",
      "value": "v=DMARC1; p=reject; rua=mailto:dmarc-rua@mail.mycompany.com",
      "purpose": "dmarc"
    }
  ]
}
```

### 6.2 Verify DNS & Ownership
**Endpoint:** `POST /v1/domains/{id}/verify`  
Queries DNS nameservers to confirm ownership TXT record, DKIM public key, and SPF alignment. Returns `verified: true` and updates status to `active`.

### 6.3 Rotate DKIM Key
**Endpoint:** `POST /v1/domains/{id}/dkim/rotate`  
Generates a new RSA 2048-bit key pair with next selector (e.g. `mvx2`), transitioning old key to `retiring` with zero downtime.

---

## 7. Message Streams & Templates

### Message Streams
- `GET /v1/streams` — List streams
- `POST /v1/streams` — Create stream
  ```json
  {
    "name": "marketing-campaigns",
    "type": "broadcast",
    "description": "Promotional product launches"
  }
  ```

### Templates
Templates use Mustache/Go template syntax (`{{name}}`):
- `POST /v1/templates` — Create template:
  ```json
  {
    "name": "Password Reset",
    "subject": "Reset your password, {{user_name}}",
    "html": "<p>Click <a href='{{reset_url}}'>here</a> to reset your password. Link expires in {{minutes}} minutes.</p>",
    "text": "Reset your password here: {{reset_url}}",
    "variables": ["user_name", "reset_url", "minutes"],
    "publish_now": true
  }
  ```
- `POST /v1/templates/{id}/render` — Preview rendered output with variables.

---

## 8. Suppressions, Bounces & Unsubscribes

Mailvex protects domain reputation by actively preventing delivery to suppressed addresses.

### Endpoints
- `GET /v1/suppressions?limit=50` — List active suppressions.
- `POST /v1/suppressions` — Manually add suppression:
  ```json
  {
    "address": "unhappy-customer@example.com",
    "type": "unsubscribe",
    "reason": "Requested removal via customer support"
  }
  ```
- `DELETE /v1/suppressions/{id}` — Remove suppression.
- `POST /v1/suppressions/import` — Bulk CSV upload.
- `GET /v1/suppressions/export` — Download CSV export.

### RFC 8058 One-Click List-Unsubscribe
Mailvex embeds RFC 8058 headers automatically:
```http
List-Unsubscribe: <https://api.mailvex.com/u/tok_01HZX87654...>, <mailto:unsub@mailvex.com?subject=tok_01HZX87654...>
List-Unsubscribe-Post: List-Unsubscribe=One-Click
```
- Receiver click invokes `POST /u/{token}` without user friction.
- Web browser view invokes `GET /u/{token}` to render branded confirmation UI.

---

## 9. Webhooks Engine & Signature Verification

Subscribe to real-time events (`message.delivered`, `message.bounced`, `message.complained`, `message.opened`, `message.clicked`, `domain.verified`, `alert.incident_created`).

### Create Webhook Endpoint
**Endpoint:** `POST /v1/webhooks`  
```json
{
  "url": "https://agent-gateway.example.com/webhooks/mailvex",
  "description": "AI Agent Dispatch Feedback Loop",
  "events": ["message.bounced", "message.complained", "message.delivered"]
}
```
**Response:** Includes a signing secret prefixed with `whsec_` (e.g. `whsec_d41d8cd98f00b204e9800998ecf8427e`). Store this secret securely.

### Signature Verification Algorithm
Incoming webhook deliveries include the header:
```http
X-Mailvex-Signature: t=1730107200,v1=3c6f8b9e4a2...
```
To verify the payload in your agent backend:
1. Parse `t` (Unix timestamp) and `v1` (HMAC signature) from `X-Mailvex-Signature`.
2. Check timestamp drift: `|current_time - t| <= 300 seconds` (5 minute anti-replay tolerance).
3. Compute expected signature:
   $$\text{signature} = \text{HMAC-SHA256}(\text{secret}, \text{t} + "." + \text{raw\_body})$$
4. Compare hex signatures using constant-time comparison.

---

## 10. Inbound Email Routing

Mailvex can receive inbound emails via SMTP port 25 and route parsed messages directly to your agent's webhook or storage bucket.

### Endpoints
- `GET /v1/inbound/routes` — List routes.
- `POST /v1/inbound/routes` — Create inbound routing rule:
  ```json
  {
    "priority": 10,
    "match_type": "wildcard",
    "pattern": "*@support.mycompany.com",
    "action": "webhook",
    "action_target": "https://ai-support-agent.mycompany.com/inbound-tickets"
  }
  ```
- `POST /v1/inbound/routes/test` — Test email address matching without dispatch.
- `GET /v1/inbound/messages` — Query parsed inbound messages.

---

## 11. Reputation, DNSBL & Warmup

### 11.1 Reputation Overview
**Endpoint:** `GET /v1/reputation/overview`  
Returns comprehensive deliverability metrics:
- Overall reputation score (0 - 100)
- Bounce rate & complaint rate (relative to industry thresholds: < 2% bounce, < 0.1% complaint)
- Active DNSBL listings across Spamhaus, Barracuda, SpamCop, and UCEPROTECT

### 11.2 DNSBL On-Demand Check
**Endpoint:** `POST /v1/dnsbl/check`  
```json
{
  "ip": "198.51.100.42"
}
```

### 11.3 IP Warmup Progression
Automate new IP warmup schedules across 30 days:
- `GET /v1/warmup/plans` — View warmup plans
- `POST /v1/warmup/ips` — Enroll IP into warmup
- `POST /v1/warmup/ips/{id}/advance` — Advance to next day's volume tier
- `POST /v1/warmup/ips/{id}/hold` — Freeze warmup on elevated bounce rates

---

## 12. SMTP Edge Submission Protocol

In addition to REST, Mailvex accepts standard RFC 5321 SMTP submission.

### Connection Parameters
- **Host:** `localhost` (dev) or your server IP / domain
- **Ports:**
  - `587`: Submission with mandatory `STARTTLS`
  - `465`: SMTPS with implicit TLS
  - `2525`: Alternative unencrypted/dev port
- **Authentication:** `AUTH PLAIN` or `AUTH LOGIN`
  - **Username:** Your SMTP Credential username
  - **Password:** Your SMTP Credential password
- **Trusted Networks:** IP addresses registered under `POST /v1/trusted-networks` can submit relay email without SASL credentials (ideal for legacy printers, IoT devices, and scanners).

---

## 13. Multi-Language Client Integration

### 13.1 Go Official SDK

Install or import the official Mailvex Go client:
```bash
go get github.com/senthilnasa/mailvex/sdk/go/mailvex
```

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/senthilnasa/mailvex/sdk/go/mailvex"
)

func main() {
	client := mailvex.NewClient("mvx_live_your_api_key_here",
		mailvex.WithBaseURL("http://localhost:8080"),
		mailvex.WithMaxRetries(3),
	)

	resp, err := client.Send(context.Background(), &mailvex.SendEmailRequest{
		From:           "alerts@mycompany.com",
		To:             []string{"ops@mycompany.com"},
		Subject:        "Cluster Alert: Automated Recovery Triggered",
		Text:           "Service postgres-primary re-attached successfully.",
		IdempotencyKey: "agent-run-uuid-001",
	})
	if err != nil {
		log.Fatalf("Send failed: %v", err)
	}

	fmt.Printf("Queued message ID: %s (Status: %s)\n", resp.MessageID, resp.Status)
}
```

---

### 13.2 Python (HTTPX / Requests)

```python
import httpx
import uuid

MAILVEX_URL = "http://localhost:8080"
API_KEY = "mvx_live_your_api_key_here"

def send_alert(recipient: str, subject: str, message: str) -> dict:
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json",
        "Idempotency-Key": str(uuid.uuid4())
    }
    payload = {
        "from": "ai-agent@mycompany.com",
        "to": [recipient],
        "subject": subject,
        "text": message,
        "stream": "transactional"
    }

    with httpx.Client(base_url=MAILVEX_URL, timeout=10.0) as client:
        response = client.post("/v1/email", headers=headers, json=payload)
        response.raise_for_status()
        return response.json()

if __name__ == "__main__":
    result = send_alert("developer@example.com", "Pipeline Finished", "All unit tests passed.")
    print("Email sent successfully:", result)
```

---

### 13.3 Node.js / TypeScript

```typescript
import axios from 'axios';
import { randomUUID } from 'crypto';

interface SendEmailParams {
  from: string;
  to: string[];
  subject: string;
  html?: string;
  text?: string;
}

const mailvex = axios.create({
  baseURL: process.env.MAILVEX_URL || 'http://localhost:8080',
  headers: {
    'Authorization': `Bearer ${process.env.MAILVEX_API_KEY}`,
    'Content-Type': 'application/json'
  }
});

async function sendEmail(params: SendEmailParams) {
  const response = await mailvex.post('/v1/email', params, {
    headers: {
      'Idempotency-Key': randomUUID()
    }
  });
  return response.data;
}

// Example usage
sendEmail({
  from: 'system@mycompany.com',
  to: ['team@example.com'],
  subject: 'Autonomous Agent Status Report',
  text: 'All sync workers are running at optimal capacity.'
}).then(console.log).catch(console.error);
```

---

### 13.4 cURL

```bash
# 1. Send an email with test key (captured in dev sink)
curl -X POST http://localhost:8080/v1/email \
  -H "Authorization: Bearer mvx_test_sample_key" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-req-001" \
  -d '{
    "from": "test@mycompany.com",
    "to": ["user@example.com"],
    "subject": "Hello from Parallel Agent",
    "text": "This is a sandboxed test message."
  }'

# 2. Inspect captured test email in dev sink
curl -s http://localhost:8080/v1/messages/development \
  -H "Authorization: Bearer mvx_test_sample_key" | jq .
```

---

### 13.5 PHP & Laravel Official SDK

Install via Composer:
```bash
composer require mailvex/mailvex-php
```

#### Standalone PHP:
```php
use Mailvex\Mailvex;
use Mailvex\Models\EmailMessage;
use Mailvex\Models\BatchMessage;

$client = Mailvex::client('mvx_live_your_api_key_here');

// 1. Send with Attachment & Multiple Recipients
$email = EmailMessage::create()
    ->from('alerts@mycompany.com')
    ->to(['alice@example.com', 'bob@example.com'])
    ->subject('Monthly Report')
    ->html('<p>Attached is your monthly report.</p>')
    ->attachFromPath('/path/to/report.pdf');

$client->send($email);

// 2. Send Batch Email in One Call (up to 500 recipients)
$batch = BatchMessage::create()
    ->from('news@mycompany.com')
    ->subject('Weekly Newsletter')
    ->template('tmpl_newsletter')
    ->addRecipient('alice@example.com', ['name' => 'Alice'])
    ->addRecipient('bob@example.com', ['name' => 'Bob']);

$client->sendBatch($batch);

// 3. Webhook Signature Verification
$isValid = Mailvex::verifyWebhook($rawPayload, $signatureHeader, $webhookSecret);
```

#### Laravel Integration:
Configure in `config/mail.php` under `'mailers'` as `'mailvex' => ['transport' => 'mailvex']` and set `.env`:
```env
MAIL_MAILER=mailvex
MAILVEX_API_KEY=mvx_live_...
MAILVEX_WEBHOOK_SECRET=whsec_...
```
Send natively via `Mail::to()->send(new InvoiceMail())` or register webhook routes with `Route::mailvexWebhooks()`.

---

## 14. Canonical Error Responses

All error responses return a standardized JSON envelope with a structured error code:

```json
{
  "error": {
    "code": "error_code_identifier",
    "message": "Human readable explanation of the failure"
  }
}
```

### Common HTTP Status Codes & Error Identifiers

| Status Code | Error Code (`code`) | Description & Recommended Agent Action |
|---|---|---|
| `400 Bad Request` | `invalid_json` | Malformed JSON in request payload. Verify syntax. |
| `400 Bad Request` | `header_injection` | CRLF characters (`\r` or `\n`) detected in headers or addresses. Strip invalid characters. |
| `400 Bad Request` | `recipient_limit_exceeded` | Exceeded 50 recipients in a single email request. Switch to `POST /v1/email/batch`. |
| `401 Unauthorized` | `unauthorized` | Missing, malformed, or revoked API key / session token. |
| `403 Forbidden` | `forbidden` | API key lacks required scope (e.g. key missing `send.email`). |
| `403 Forbidden` | `cross_tenant_forbidden` | Attempted access to a resource belonging to a different tenant. |
| `404 Not Found` | `message_not_found` | Specified ULID does not exist or was purged by retention policy. |
| `409 Conflict` | `idempotency_in_progress` | A request with the same `Idempotency-Key` is currently executing. Wait 500ms and retry. |
| `409 Conflict` | `idempotency_key_reused` | Same `Idempotency-Key` was submitted with a differing request body. Generate a fresh key. |
| `409 Conflict` | `domain_claimed_by_another_tenant` | Domain is already verified by another tenant without delegation. |
| `429 Too Many Requests` | `rate_limit_exceeded` | Rate limit threshold reached. Sleep for duration indicated in `RateLimit-Reset`. |
| `500 Internal Error` | `internal_error` | Server encountered an error. Safe to retry with exponential backoff. |
| `503 Service Unavailable` | `unhealthy` | Database pool or Redis queue unavailable. Check `/ready` probe. |
