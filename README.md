# Mailvex

<p align="center">
  <strong>Carrier-Grade, Multi-Tenant, Self-Hosted Email Infrastructure & Delivery Engine</strong><br>
  <em>The modern, open-source alternative to SendGrid, Mailgun, AWS SES, and Postmark.</em>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/senthilnasa/mailvex/sdk/go/mailvex"><img src="https://pkg.go.dev/badge/github.com/senthilnasa/mailvex/sdk/go/mailvex.svg" alt="Go Reference"></a>
  <a href="https://github.com/senthilnasa/mailvex/releases"><img src="https://img.shields.io/github/v/release/senthilnasa/mailvex?color=emerald&label=Release" alt="Latest Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="docs/deployment.md"><img src="https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white" alt="Docker Ready"></a>
  <a href="docs/deliverability.md"><img src="https://img.shields.io/badge/Mail--Tester-10%2F10-brightgreen" alt="Mail-Tester Score"></a>
  <a href="llms.txt"><img src="https://img.shields.io/badge/AI%20Agent-Native%20(llms.txt)-8A2BE2" alt="AI Agent Native"></a>
  <a href="https://goreportcard.com/report/github.com/senthilnasa/mailvex"><img src="https://goreportcard.com/badge/github.com/senthilnasa/mailvex" alt="Go Report Card"></a>
</p>

<p align="center">
  <a href="#-key-features">Key Features</a> •
  <a href="#-feature-comparison">Comparison</a> •
  <a href="#-frosted-glass-operator-console">Console UI</a> •
  <a href="#-quickstart-production-docker">Docker Quickstart</a> •
  <a href="#-ai-agent--llm-native-integration">AI Agents (llms.txt)</a> •
  <a href="#-official-client-sdks">SDKs (Go, PHP, Laravel)</a> •
  <a href="#-deliverability--reputation-engine">Deliverability</a> •
  <a href="#-documentation-hub">Documentation</a> •
  <a href="#-license">License</a>
</p>

---

## ⚡ Overview

**Mailvex** is a high-throughput, self-hosted transactional email infrastructure platform written in **Go 1.24** with an ultra-responsive, frosted-glass dark mode **React 19** operator console. 

Engineered for startups, homelab enthusiasts, autonomous AI agent swarms, and enterprise clusters, Mailvex eliminates recurring per-email cloud bills by giving you complete ownership over your email delivery pipeline. It includes dual ingestion edges (REST API & SMTP Submission), direct-to-MX delivery over Port 25, automatic RSA-2048 DKIM signing, RFC 8058 one-click list unsubscribes, automatic bounce classification, and cryptographic tenant envelope encryption.

> 🤖 **AI-Agent Ready:** Mailvex publishes a root [`llms.txt`](llms.txt) and [`docs/ai-agents.md`](docs/ai-agents.md) specification with pre-built JSON function schemas (OpenAI, Anthropic, Gemini, MCP) and dual-mode keys (`mvx_test_...` development sink vs `mvx_live_...` direct MX delivery).

---

## 🚀 Key Features

- **Carrier-Grade Outbound Engine**:
  - Direct MX DNS resolution with connection pooling, TLS opportunistic encryption, and RFC 5321/5322 compliance.
  - Automatic DKIM signing (RSA 2048-bit with PKCS#8 key rotation) and header hygiene.
  - Smart-host relay fallback for home ISPs or cloud providers blocking outbound Port 25.
- **Dual Ingestion Edges**:
  - **High-Throughput REST API (`/v1`)**: Idempotent email dispatch, batch sending (up to 500 recipients per call), tracking webhooks, and template interpolation.
  - **SMTP Edge**: Ports 25 (direct inbound MX), 587 (authenticated client submission), and 2525 with SASL PLAIN/LOGIN and TLS.
- **Deliverability & Reputation Center**:
  - Bidirectional FCrDNS (Forward-Confirmed Reverse DNS) validation.
  - Real-time DNSBL monitoring across Spamhaus, Barracuda, SpamCop, and UCEPROTECT.
  - Automated 30-day IP warmup curves with per-provider throttling (Gmail, Yahoo, Microsoft).
  - RFC 8058 One-Click List-Unsubscribe, DSN hard/soft bounce classification, and ARF complaint processing.
- **Cryptographic Multi-Tenancy**:
  - Strict tenant boundary isolation with per-tenant data encryption keys (DEK) wrapped by a master key (KEK via AES-256-GCM, HashiCorp Vault, or AWS KMS).
  - Platform administrator impersonation safeguards with mandatory audit trails.
- **Frosted-Glass Operator Console**:
  - Built with React 19, Tailwind CSS v4, Lucide icons, and real-time WebSocket delivery telemetry.
  - Instant CSV & JSON audit report exports for messages, domains, and suppressions.

---

## 📊 Feature Comparison

| Capability | Mailvex (Self-Hosted) | SendGrid | Mailgun | AWS SES | Postmark |
|---|---|---|---|---|---|
| **Monthly Email Fees** | **\$0 (Unlimited)** | \$89.95/mo (100k) | \$75/mo (100k) | \$10/mo (100k) | \$115/mo (100k) |
| **Data Privacy & Ownership** | **100% Self-Hosted** | Cloud / Third-Party | Cloud / Third-Party | Cloud / Third-Party | Cloud / Third-Party |
| **Direct-to-MX Delivery** | ✅ Built-in | ❌ Proprietary MTA | ❌ Proprietary MTA | ❌ Proprietary MTA | ❌ Proprietary MTA |
| **Homelab Profile (<200MB RAM)** | ✅ Built-in | ❌ N/A | ❌ N/A | ❌ N/A | ❌ N/A |
| **Dual-Mode AI Keys (`mvx_test`)**| ✅ Built-in | ❌ Partial sandbox | ❌ Sandbox domain | ❌ Simulator | ❌ Sandbox token |
| **AI Agent Native (`llms.txt`, MCP)**| ✅ Built-in | ❌ No | ❌ No | ❌ No | ❌ No |
| **Automatic DKIM (RSA-2048)** | ✅ Built-in | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes |
| **1-Click Unsubscribe (RFC 8058)**| ✅ Built-in | ✅ Yes | ✅ Yes | ❌ Manual | ✅ Yes |
| **License** | **MIT (Open Source)** | Proprietary | Proprietary | Proprietary | Proprietary |

---

## 🖥 Frosted-Glass Operator Console

Mailvex includes an ultra-modern, responsive operator dashboard located in [`console/`](console/):

- **Live Analytics Hub:** Real-time throughput graphs, delivery success percentages, bounce rate gauges, and spam complaint monitors.
- **Outbox Inspector:** Search, filter, and inspect email payloads, raw MIME data, delivery transit hops, and target MX SMTP response codes.
- **Deliverability Doctor:** Real-time health auditing of your server's IP, reverse DNS (PTR), port 25 reachability, and DNSBL blocklist status.
- **Domain Provisioning:** Add sending domains with automated RSA-2048 DKIM key generation, SPF record validators, and DMARC alignment checkers.
- **One-Click Data Exports:** Export outbox logs, delivery events, and suppression lists directly to CSV, JSON, or formatted print views.

---

## 🐳 Quickstart: Production (Docker)

Deploy Mailvex in minutes using Docker Compose. For comprehensive setup instructions, reverse proxy configs, and troubleshooting, read the [**Production Docker Deployment Guide**](docs/deployment.md).

### 1. Homelab Profile (All-in-One — < 200MB RAM)
Perfect for homelabs, VPS nodes, or single-board computers (combines API, Console, Worker, SMTP, MariaDB & Redis):
```bash
docker compose -f deploy/compose/docker-compose.homelab.yml up -d
```

### 2. Standard Production Profile (Decoupled API & Workers)
Recommended for business production servers with independent API and worker scaling:
```bash
docker compose -f deploy/compose/docker-compose.standard.yml up -d
```

### 3. Enterprise High-Availability Cluster
Multi-node enterprise setup with HAProxy load balancing, distributed workers, and Prometheus metrics:
```bash
docker compose -f deploy/compose/docker-compose.cluster.yml up -d
```

### Default Operator Credentials
When Mailvex initializes for the first time, it provisions the default administrator account:
- **Console URL:** `http://localhost:3000` (or `http://localhost:8080` in all-in-one container)
- **Email:** `admin@example.com`
- **Password:** `Password123!`
- **Primary Tenant:** `Production`

---

## 💻 Quickstart: Local Development

### Prerequisites
- **Go 1.24+**
- **Node.js 20+** and `npm`
- **Docker** (for local MariaDB and Redis backing services)

```bash
# 1. Start backing services (MariaDB 11.4 & Redis 7.2)
docker compose -f deploy/compose/docker-compose.homelab.yml up -d mariadb redis

# 2. Run the Go All-in-One server (API on :8080, SMTP on :2525)
go run ./cmd/allinone

# 3. In a separate terminal, launch the Console UI dev server (Vite on :3000)
cd console
npm install
npm run dev
```

---

## 🤖 AI Agent & LLM Native Integration

Mailvex is architected from the ground up for autonomous AI agent pipelines (Cursor, Claude Code, GitHub Copilot, Devin, LangChain, AutoGen, CrewAI):

- 📄 **`llms.txt`**: Standard machine-readable project index located at repository root ([`llms.txt`](llms.txt)).
- 📘 **AI Agent Protocol**: Comprehensive function calling schemas, system prompts, and MCP server instructions located in [`docs/ai-agents.md`](docs/ai-agents.md).
- 🔑 **Dual-Mode API Keys**:
  - `mvx_test_...`: Safe sandbox mode. Generates full MIME, executes DKIM signing, and dumps into the development sink (`GET /v1/messages/development`) without sending external network traffic.
  - `mvx_live_...`: Production mode. Routes messages directly to external MX servers over Port 25.
- 🔒 **Distributed 24-Hour Idempotency**:
  Pass `Idempotency-Key: <uuid>` on every `POST /v1/send`. Mailvex acquires distributed Redis locks to ensure agent retries or hallucinated loops never send duplicate emails.

---

## 📦 Official Client SDKs

### 1. Go SDK (`sdk/go`)
Published on `pkg.go.dev`, zero external dependencies:
```bash
go get github.com/senthilnasa/mailvex/sdk/go/mailvex
```

```go
package main

import (
	"context"
	"fmt"
	"github.com/senthilnasa/mailvex/sdk/go/mailvex"
)

func main() {
	client := mailvex.NewClient("http://localhost:8080", "mvx_live_your_api_key")

	resp, err := client.Send(context.Background(), &mailvex.SendRequest{
		From:    "hello@yourdomain.com",
		To:      []string{"recipient@example.com"},
		Subject: "Hello from Go",
		Html:    "<p>Sent via Mailvex Go SDK</p>",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Queued Message ID: %s (Status: %s)\n", resp.MessageID, resp.Status)
}
```

### 2. PHP & Laravel Official SDK (`sdk/php`)
Composer package with fluent message builders and a native Laravel Mail driver:
```bash
composer require mailvex/mailvex-php
```

```php
use Mailvex\Mailvex;
use Mailvex\Models\EmailMessage;

$client = Mailvex::client('mvx_live_your_api_key');

$client->send(
    EmailMessage::create()
        ->from('billing@yourdomain.com')
        ->to('client@example.com')
        ->subject('Monthly Invoice')
        ->html('<p>Please find attached invoice.</p>')
        ->attachFromPath('/invoices/inv-042.pdf')
);
```
*(In Laravel: configure `MAIL_MAILER=mailvex` in `.env` to send via `Mail::to()->send()` automatically).*

---

## 🛡 Deliverability & Reputation Engine

Mailvex achieves industry-leading deliverability through strict protocol compliance:

- **Mail-Tester Score:** **10/10 Perfect Score** verified across DKIM, SPF, SpamAssassin (-0.1 score), and MX validity.
- **MailGenius Score:** **98/100 Delivery Grade** with zero blacklistings and passing reverse DNS.
- **DKIM Signing:** RSA 2048-bit with PKCS#8 keys, auto-generated DNS TXT records, and cryptographic key rotation.
- **RFC 8058 One-Click Unsubscribe:** Automatic insertion of `List-Unsubscribe` and `List-Unsubscribe-Post: List-Unsubscribe=One-Click` headers.
- **DSN Bounce Engine:** Automated parsing of Delivery Status Notifications (RFC 3464) into categorized suppressions (`hard_bounce`, `soft_bounce`, `spam_complaint`).

---

## 🛠 Operational CLI & System Doctor

Mailvex includes an administrative command-line utility for system audits and disaster recovery:

```bash
# Run system doctor diagnostics (ports, DNS, kernel limits, database)
go run ./cmd/mailvex doctor

# Create an encrypted, tamper-evident HMAC-SHA256 backup
go run ./cmd/mailvex backup create --output=mailvex-backup.tar.gz

# Restore from an encrypted backup
go run ./cmd/mailvex backup restore --input=mailvex-backup.tar.gz
```

---

## 📚 Documentation Hub

Explore the exhaustive technical guides in the [`docs/`](docs/) directory:

| Document | Description |
|---|---|
| [**Docs Navigation Hub**](docs/README.md) | Complete documentation index and sitemap |
| [**REST API Specification**](docs/api.md) | Canonical `/v1` endpoint definitions, payload schemas, and errors |
| [**AI Agent Protocol**](docs/ai-agents.md) | Autonomous agent integration, MCP servers, and tool calling schemas |
| [**Production Docker Guide**](docs/deployment.md) | Homelab, Standard, and Enterprise HA cluster deployment walkthrough |
| [**System Architecture**](docs/architecture.md) | Ingestion supervisors, queue topology, and outbound MTA mechanics |
| [**Deliverability & Warmup**](docs/deliverability.md) | DKIM, SPF, DMARC, FCrDNS, and 30-day automated IP warmup curves |
| [**Webhooks & Signatures**](docs/webhooks.md) | Real-time event notifications with HMAC-SHA256 anti-replay signatures |
| [**Security & Cryptography**](docs/security.md) | Envelope encryption, tenant DEK/KEK architecture, and RBAC rules |
| [**Database Schema & ERD**](docs/erd.md) | Entity relationship diagrams and partitioned MariaDB table layouts |

---

## 🧪 Testing & Verification

Run the full automated test suite across all 37 Go packages:

```bash
# Execute unit and integration tests
go test -count=1 ./...

# Run Go static analysis
go vet ./...

# Build the frontend console for production
npm --prefix console run build
```

---

## 📄 License

Mailvex is open-source software released under the [**MIT License**](LICENSE).  
Copyright © 2026 senthilnasa. Free for personal, commercial, and enterprise deployment.
