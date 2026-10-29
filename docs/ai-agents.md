# Mailvex — AI Agent & Autonomous System Protocol

> **Specification Version:** 1.0.0  
> **Target Audience:** Autonomous Agents, LLM Swarms, MCP Clients, Tool-Calling Pipelines, and Backend Engineers  
> **Integration Surface:** REST API (`/v1`), Model Context Protocol (MCP), and Function Calling Schemas  

---

## 1. Overview & Architectural Philosophy

Mailvex was engineered from inception to be seamlessly manipulated by **autonomous AI agents** (Cursor, Claude Code, GitHub Copilot, Devin, LangChain, AutoGen, CrewAI, OpenAI Assistants).

Traditional email service providers (SES, SendGrid) present complex hurdles for AI agents:
1. High risk of accidental email spam or duplicate sends caused by agent retries or hallucinated loops.
2. Inflexible rate limiting that causes unhandled network crashes in parallel agent swarms.
3. Lack of unified development sandboxes where agents can verify delivered messages without polluting production domains.

Mailvex eliminates these pain points through:
- **Dual-Mode API Keys:** Seamlessly switch between `mvx_test_...` (development sink, zero external MX routing) and `mvx_live_...` (production direct-to-MX delivery).
- **Mandatory Distributed Idempotency:** The `Idempotency-Key` header ensures that agent retries or network replays never produce duplicate emails.
- **Structured Error Responses:** Every error payload includes a machine-readable `code`, detailed explanation, and remediation guidance.
- **Native Tool Schemas:** Pre-built JSON Schemas compatible with OpenAI, Anthropic Claude, Google Gemini, and the Model Context Protocol (MCP).

---

## 2. Agent Key Modes & Development Sandbox

| Key Prefix | Execution Mode | Behavior | Safe for Agent Testing? |
|---|---|---|---|
| **`mvx_test_...`** | **Development Sink** | Full MIME parsing, DKIM signing, and validation occur. Message is stored in the development sink table (`/v1/messages/development`) but **never connects to external port 25 or MX servers**. | ✅ **100% Safe** (Recommended for agent prompt evaluation) |
| **`mvx_live_...`** | **Production Delivery** | Resolves target MX servers via DNS, performs TLS handshake, and delivers directly to recipient mailboxes. | ⚠️ **Live Traffic** (Ensure recipient addresses are verified) |

### Retrieving Sent Messages from the Development Sink
When testing agent workflows with `mvx_test_...`, the agent can inspect the generated email, headers, and rendered HTML body:
```http
GET /v1/messages/development?limit=10
Authorization: Bearer mvx_test_your_key_here
```

---

## 3. Concurrency, Swarms & Idempotency Guarantees

When deploying swarms of parallel AI agents:
1. **Always Supply an Idempotency Key:**
   Pass `Idempotency-Key: <unique-uuid-or-hash>` (or `X-Idempotency-Key`) in every `POST /v1/send` call.
2. **Deterministic Hashing:**
   If your agent generates notifications dynamically, compute an SHA-256 hash of the `(recipient + subject + timestamp_bucket)` as the idempotency key:
   ```python
   import hashlib
   idempotency_key = hashlib.sha256(f"{to_email}:{subject}:{date_hour}".encode()).hexdigest()
   ```
3. **Cache Duration:**
   Mailvex retains idempotency locks in Redis for **24 hours**. If an agent experiences a timeout and retries the exact same request, Mailvex immediately returns the cached `200 OK` response with the original `message_id` without re-queuing the email.

---

## 4. Machine-Readable Function Calling / Tool Schemas

Use these JSON Schemas to configure tools in OpenAI Assistants, Anthropic Tools, LangChain, or MCP servers.

### 4.1. `mailvex_send_email`
Dispatches a single transactional or promotional email.

```json
{
  "name": "mailvex_send_email",
  "description": "Send an email message via Mailvex carrier-grade infrastructure with direct MX delivery and DKIM signing.",
  "parameters": {
    "type": "object",
    "properties": {
      "from": {
        "type": "string",
        "description": "Sender email address (e.g. 'notifications@yourdomain.com' or 'Billing <billing@yourdomain.com>')."
      },
      "to": {
        "type": "array",
        "items": { "type": "string" },
        "description": "List of recipient email addresses."
      },
      "subject": {
        "type": "string",
        "description": "Email subject line."
      },
      "text": {
        "type": "string",
        "description": "Plain-text fallback body."
      },
      "html": {
        "type": "string",
        "description": "HTML rendered email body."
      },
      "cc": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Optional list of carbon copy (Cc) recipient addresses."
      },
      "bcc": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Optional list of blind carbon copy (Bcc) recipient addresses."
      },
      "reply_to": {
        "type": "string",
        "description": "Optional Reply-To email address."
      },
      "idempotency_key": {
        "type": "string",
        "description": "Unique UUID or token to prevent duplicate dispatch on agent retry."
      }
    },
    "required": ["from", "to", "subject"]
  }
}
```

### 4.2. `mailvex_get_message`
Queries the real-time delivery state of a previously dispatched email.

```json
{
  "name": "mailvex_get_message",
  "description": "Retrieve delivery status, attempt counts, and target MX server response codes for a message.",
  "parameters": {
    "type": "object",
    "properties": {
      "message_id": {
        "type": "string",
        "description": "The unique UUID of the message returned by mailvex_send_email."
      }
    },
    "required": ["message_id"]
  }
}
```

### 4.3. `mailvex_verify_domain`
Audits DNS configuration (DKIM, SPF, DMARC, MX) for a custom sending domain.

```json
{
  "name": "mailvex_verify_domain",
  "description": "Trigger a real-time DNS audit to verify DKIM RSA-2048 keys, SPF TXT records, and DMARC alignment for a sending domain.",
  "parameters": {
    "type": "object",
    "properties": {
      "domain_id": {
        "type": "string",
        "description": "The unique UUID of the sending domain in Mailvex."
      }
    },
    "required": ["domain_id"]
  }
}
```

---

## 5. Model Context Protocol (MCP) Configuration

To expose Mailvex tools directly to Cursor, Claude Desktop, or Claude Code via the Model Context Protocol, add the following entry to your `mcp.json` or `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mailvex": {
      "command": "npx",
      "args": ["-y", "@mailvex/mcp-server"],
      "env": {
        "MAILVEX_API_URL": "http://localhost:8080",
        "MAILVEX_API_KEY": "mvx_live_your_api_key_here"
      }
    }
  }
}
```

---

## 6. Agent System Prompt & Operational Directives

When configuring an autonomous agent tasked with sending or monitoring emails, inject the following context into the system prompt:

```text
You are an autonomous operations agent integrated with the Mailvex email infrastructure platform.
Your objective is to compose, send, and verify transactional and operational email communications.

Operational Directives:
1. Always generate a unique `idempotency_key` (UUID v4) for every `mailvex_send_email` call to guard against network timeout re-transmissions.
2. Prior to sending emails to a new domain, verify that the domain has passed DKIM and SPF checks using `mailvex_verify_domain`.
3. If an email fails with a 422 Unprocessable Entity indicating a suppression or bounce, do not retry dispatching to that address.
4. When drafting HTML content, always include a matching `text` alternative for multipart MIME compliance and maximum deliverability scores.
5. In development or test environments, use `mvx_test_...` API keys to inspect output in the message sink without sending external traffic.
```

---

## 7. Multi-Agent Code Implementations

### Python (OpenAI Function Calling Example)
```python
import os
import uuid
import httpx
from openai import OpenAI

client = OpenAI(api_key=os.environ["OPENAI_API_KEY"])
MAILVEX_API_KEY = os.environ.get("MAILVEX_API_KEY", "mvx_test_sandbox")
MAILVEX_URL = os.environ.get("MAILVEX_API_URL", "http://localhost:8080")

def execute_mailvex_send(from_addr, to_addrs, subject, html_body, text_body=None):
    headers = {
        "Authorization": f"Bearer {MAILVEX_API_KEY}",
        "Content-Type": "application/json",
        "Idempotency-Key": str(uuid.uuid4())
    }
    payload = {
        "from": from_addr,
        "to": to_addrs,
        "subject": subject,
        "html": html_body,
        "text": text_body or subject
    }
    with httpx.Client(base_url=MAILVEX_URL) as http:
        resp = http.post("/v1/send", json=payload, headers=headers)
        resp.raise_for_status()
        return resp.json()

# Agent tool execution loop handles the returned tool call seamlessly
```

### TypeScript / Vercel AI SDK Example
```typescript
import { tool } from 'ai';
import { z } from 'zod';

export const sendEmailTool = tool({
  description: 'Send an email via Mailvex delivery engine',
  parameters: z.object({
    from: z.string().email(),
    to: z.array(z.string().email()),
    subject: z.string(),
    html: z.string(),
    text: z.string().optional(),
  }),
  execute: async ({ from, to, subject, html, text }) => {
    const res = await fetch('http://localhost:8080/v1/send', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${process.env.MAILVEX_API_KEY}`,
        'Content-Type': 'application/json',
        'Idempotency-Key': crypto.randomUUID(),
      },
      body: JSON.stringify({ from, to, subject, html, text }),
    });
    if (!res.ok) throw new Error(`Mailvex error: ${await res.text()}`);
    return await res.json();
  },
});
```

---

## 8. Summary Table of Agent Integration URLs

| Task | HTTP Method | Endpoint | Documentation |
|---|---|---|---|
| Single Email | `POST` | `/v1/send` | [docs/api.md#41-send-single-email](api.md#41-send-single-email) |
| Batch Email | `POST` | `/v1/send/batch` | [docs/api.md#42-send-batch-email](api.md#42-send-batch-email) |
| Message Status | `GET` | `/v1/messages/{id}` | [docs/api.md#52-message-details-events--raw-body](api.md#52-message-details-events--raw-body) |
| Development Sink | `GET` | `/v1/messages/development` | [docs/api.md#54-development-message-sink](api.md#54-development-message-sink) |
| Verify Domain DNS | `POST` | `/v1/domains/{id}/verify` | [docs/api.md#62-verify-dns--ownership](api.md#62-verify-dns--ownership) |
| Delivery Webhooks | `POST` | `/v1/webhooks` | [docs/api.md#9-webhooks-engine--signature-verification](api.md#9-webhooks-engine--signature-verification) |
