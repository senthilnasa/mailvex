# Mailvex

<p align="center">
  <strong>Carrier-Grade, Multi-Tenant, Self-Hosted Email Infrastructure Engine</strong>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#architecture--profiles">Profiles</a> •
  <a href="DOCKER_DEPLOYMENT.md">Docker Guide</a> •
  <a href="#quickstart-development">Dev Server</a> •
  <a href="#quickstart-production">Prod Server</a> •
  <a href="#api--ai-agent-integration">API &amp; Agent Docs</a> •
  <a href="#cli--doctor">CLI &amp; Doctor</a> •
  <a href="#go-sdk">Go SDK</a> •
  <a href="#security--compliance">Security</a>
</p>

---

## Overview

**Mailvex** is a modern, self-hosted alternative to commercial email service providers (AWS SES, Mailgun, SendGrid, Postmark). Built in **Go 1.26** with a **React 18 + Tailwind CSS** operator console, it provides carrier-grade SMTP edge ingestion, high-throughput outbound delivery, cryptographic envelope encryption, multi-tenant boundary isolation, and deliverability observability.

Whether deployed as an ultra-low-footprint homelab mail hub (< 200MB RAM) or as a high-availability multi-node cluster, Mailvex gives you total ownership of your email deliverability pipeline.

---

## Key Features

- **Multi-Tenant Isolation**: Strict cross-tenant boundary isolation with per-tenant data encryption keys (DEK) wrapped by a master key encryption key (KEK via local AES-256-GCM, HashiCorp Vault Transit, or AWS KMS).
- **Dual Ingestion Edges**:
  - **REST API (`/v1`)**: Idempotent email dispatch, batch sending, tracking configuration, and template interpolation.
  - **SMTP Submission**: Port 25, 587, and 2525 with SASL authentication, TLS, trusted network IP filtering, and IoT device registry.
- **Outbound Delivery Engine**:
  - Direct MX resolution with concurrency control, backoff retry queues, and smart-host relay fallback.
  - Automatic DKIM signing (RSA 2048-bit with PKCS#8 key rotation) and header hygiene.
- **Reputation & Deliverability Center**:
  - Bidirectional FCrDNS (Forward-Confirmed Reverse DNS) validation.
  - Real-time DNSBL monitoring across major blocklists (Spamhaus, Barracuda, SpamCop, UCEPROTECT).
  - 30-day automated IP warmup progression with per-provider throttle caps.
  - RFC 8058 One-Click List-Unsubscribe, DSN hard/soft bounce classification, and ARF complaint processing.
- **Enterprise SSO & Impersonation**:
  - OIDC and SAML 2.0 Identity Provider integration with JIT provisioning.
  - Platform Administrator cross-tenant impersonation with privacy masking and mandatory audit logging.
- **Observability**: Prometheus metrics mounted at `GET /metrics`, Grafana dashboard templates, and structured JSON audit logging.

---

## Architecture & Profiles

Mailvex supports three execution profiles via `MVX_PROFILE`:

| Profile | Target Environment | Footprint | Architecture |
|---|---|---|---|
| `homelab` | Home labs, IoT hubs, Edge nodes, single-board computers | < 200MB RAM | Unified all-in-one process supervisor with embedded queuing & SMTP edge. |
| `standard` | Small-to-medium businesses, single cloud instances | ~500MB RAM | Independent API and Delivery Worker containers with MariaDB 11.4 & Redis 7.2. |
| `cluster` | Enterprise high-volume deliverability clusters | Dynamic / Multi-Node | Horizontal API tier, distributed worker queue, Redis Cluster, read-replica DB. |

---

## Quickstart: Development

Run the frontend dev server and the backend locally with hot-reloading.

### Prerequisites
- **Go 1.26+**
- **Node.js 20+** and `npm`
- **Docker Desktop** (for local MariaDB and Redis dependencies)

### 1. Start Backing Services
```bash
docker compose -f deploy/compose/docker-compose.homelab.yml up -d mariadb redis
```

### 2. Start the Local Go Backend
```bash
# In the root repository directory:
export PORT=8080
export SMTP_PORT=2525
go run ./cmd/allinone
```
*The API will start at `http://localhost:8080` and the SMTP edge on port `2525`.*

### 3. Start the Frontend Console (Vite)
```bash
cd console
npm install
npm run dev
```
*The Console will be accessible at **`http://localhost:3000`** with live hot module replacement (HMR).*

---

## Quickstart: Production

Run the full production stack using Docker Compose. For the complete step-by-step setup, reverse proxy guides, and troubleshooting, see the [**Docker Deployment Guide**](DOCKER_DEPLOYMENT.md).

### Homelab Profile (All-In-One Supervisor)
```bash
# Start Mailvex Homelab with MariaDB & Redis
docker compose -f deploy/compose/docker-compose.homelab.yml up -d
```

### Standard Production Stack
```bash
# Start dedicated API, Worker, MariaDB & Redis services
docker compose -f deploy/compose/docker-compose.standard.yml up -d
```

### High Availability Cluster
```bash
# Start distributed multi-node cluster with HAProxy load balancing
docker compose -f deploy/compose/docker-compose.cluster.yml up -d
```

---

## Administrator Access & Default Credentials

When Mailvex boots for the first time in the Homelab profile, it automatically pre-seeds the Platform Administrator credentials:

* **Web Console**: `http://localhost:3000` (or `http://localhost:8080` in production container)
* **Email**: `admin@example.com`
* **Password**: `Password123!`
* **Primary Tenant**: `Production` (`production`)

### Authenticate via cURL
```bash
curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"Password123!"}'
```

### Reset Platform Admin Credentials
If you wish to change or recover the administrator credentials at any time:
```bash
curl -s -X POST http://localhost:8080/v1/auth/reset-admin \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"YourNewStrongPassword123!"}'
```

---

## System Health & Observability Endpoints

| Endpoint | Method | Purpose |
|---|---|---|
| `/health` | `GET` | Public liveness probe (returns status 200). |
| `/ready` | `GET` | Readiness probe (verifies database pool and cache connectivity). |
| `/metrics` | `GET` | Prometheus telemetry (queue depth, delivery rates, latencies). |
| `/v1/setup` | `GET` | Checks if platform initial setup has been executed. |

---

## API & AI Agent Integration

Mailvex features a carrier-grade REST API designed for high-throughput software pipelines and autonomous parallel AI agents.

* 📖 **Full API & AI Agent Specification**: [`API.md`](./API.md) (or [`docs/api.md`](./docs/api.md))
* ⚙️ **OpenAPI 3.1 Specification**: [`sdk/openapi.yaml`](./sdk/openapi.yaml)
* 🤖 **AI Agent Features**:
  * Dual-mode keys (`mvx_live_...` for real MX routing, `mvx_test_...` for development sink)
  * Distributed locking with 24-hour `Idempotency-Key` caching
  * Pre-built tool definitions (JSON Schema / MCP) for parallel agent swarms
  * Real-time delivery webhooks with HMAC-SHA256 anti-replay signatures

---

## CLI & System Doctor

Mailvex includes an administrative command-line utility:

### System Diagnostics
Run environmental audits covering kernel limits, network ports, DNS resolution, and database health:
```bash
go run ./cmd/mailvex doctor
```

### Cryptographic Backups
Create and restore tamper-evident backups signed with HMAC-SHA256:
```bash
# Create encrypted backup
go run ./cmd/mailvex backup create --output=mailvex-backup.tar.gz

# Restore backup
go run ./cmd/mailvex backup restore --input=mailvex-backup.tar.gz
```

---

## Go SDK

Mailvex provides an official, zero-dependency Go client library:

```go
package main

import (
	"context"
	"fmt"
	"github.com/senthilnasa/mailvex/sdk/go/mailvex"
)

func main() {
	client := mailvex.NewClient("http://localhost:8080", "mvx_live_your_api_key_here")

	resp, err := client.Send(context.Background(), &mailvex.SendRequest{
		From:    "notifications@yourdomain.com",
		To:      []string{"user@example.com"},
		Subject: "Welcome to Mailvex",
		Text:    "Your email infrastructure is now fully self-hosted.",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Message queued successfully: %s (Status: %s)\n", resp.MessageID, resp.Status)
}
```

---

## Testing & Quality Assurance

Run the automated test suite across all packages:

```bash
# Run unit & integration tests
go test -count=1 ./...

# Run static analysis
go vet ./...

# Build the frontend console
cd console && npm run build
```

---

## License

Mailvex is released under the **Apache 2.0 License**. See [LICENSE](LICENSE) for details.
