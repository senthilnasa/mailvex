# Mailvex — System Architecture

## 1. Overview

Mailvex is a production-grade, secure, multi-tenant, self-hosted email infrastructure platform. It provides transactional and broadcast email delivery, SMTP submission, REST sending APIs, full deliverability tooling (bounces, FBL, suppressions, unsubscribe), inbound email, and an administrative console.

Mailvex is engineered to run seamlessly across three operational deployment profiles:
- **Home Lab (`homelab`)**: Resource-constrained single box (e.g. Raspberry Pi 4/5, mini-PC), IoT device mail hub, trusted LAN submission, smart-host relay delivery, local domains.
- **Standard (`standard`)**: Single virtual machine or bare-metal server with public IPv4/IPv6, direct MX delivery, Caddy reverse proxy with on-demand ACME TLS.
- **Cluster (`cluster`)**: Multi-node deployment with dedicated API, SMTP edge, and worker nodes, external MariaDB/Galera cluster, and Redis Sentinel/Cluster.

---

## 2. Monorepo Structure & Deployable Units

```text
mailvex/
├── core/        Mailvex Core      — Shared Go kernel: domain models, repository interfaces,
│                                    crypto/envelope encryption, queue abstraction, outbox,
│                                    rate limiting, scheduler/housekeeping jobs
├── api/         Mailvex API       — Go HTTP service: public /v1 API, management API,
│                                    console BFF, tracking/unsubscribe endpoints, health checks
├── smtp/        Mailvex SMTP      — SMTP edge: submission (587/465/25) and inbound MX receiver
├── console/     Mailvex Console   — React 18+ / TypeScript / Vite / Tailwind UI
├── worker/      Mailvex Worker    — Outbound delivery engine: queue consumer, fair scheduler,
│                                    MX resolution, SMTP client, retries, DSN/FBL parsers
├── webhooks/    Mailvex Webhooks  — Go webhook dispatcher: signing, delivery, retries, replay
├── sdk/         Mailvex SDK       — Go client library, CLI (`mailvex`), openapi.yaml
├── deploy/      Deployment configurations (Docker Compose profiles, Caddyfile, systemd)
├── docs/        Architecture, ADRs, guides, deliverability specifications
└── migrations/  Versioned SQL migrations for MariaDB / MySQL 8
```

---

## 3. Delivery Ownership & Data Flow

### 3.1 Strict Single-Queue & Delivery Ownership
To prevent split-brain delivery logic:
1. **Mailvex Worker** is the **exclusive owner** of outbound email delivery (direct MX and smart-host relay).
2. **Mailvex SMTP** (the SMTP edge) is strictly an **ingestion and reception adapter**. Outbound relaying is disabled at the edge.
3. Both HTTP API requests (`POST /v1/email`) and SMTP submissions (`core.Ingest`) write to the database outbox table in an atomic transaction. Redis Streams serves as the fast delivery hint.

```text
 SMTP Client ────────► Mailvex SMTP ──┐
                                     ├──► Core.Ingest() ──► MariaDB (messages + outbox)
 HTTP API Client ────► Mailvex API ───┘                             │
                                                                   ▼
                                                            Outbox Relay / Scheduler
                                                                   │
                                                                   ▼
                                                             Redis Streams
                                                                   │
                                                                   ▼
                                                            Mailvex Worker
                                                            ├── Suppression check
                                                            ├── Routing & IP selection
                                                            ├── MX Resolution / Grouping
                                                            ├── TLS & DKIM signing
                                                            └── Transmit (Direct MX or Relay)
                                                                   │
                                                                   ▼
                                                            Internet MX / Smart Host
```

---

## 4. Operational Profiles

### 4.1 Home Lab (`homelab`)
- Packaging: `mailvex-allinone` running API, Worker, Webhooks, Scheduler, and SMTP in a single process supervisor, alongside MariaDB (low-memory tuned) and Redis.
- Memory target: < 512 MB RAM idle.
- Capabilities: Device registry for IoT appliances, unauthenticated or IP-authenticated LAN submission on port 25/587 within trusted CIDRs (`192.168.0.0/16`, `10.0.0.0/8`), smart-host relay transport (Gmail Workspace, Amazon SES, SendGrid, Mailgun) with automatic From-rewriting, local domains (e.g. `home.lan`), mail-to-push notification bridges (ntfy, Telegram, Home Assistant).

### 4.2 Standard (`standard`)
- Separate containers for `api`, `worker`, `smtp`, `webhooks`, `scheduler`, `console`, `mariadb`, `redis`, and `caddy`.
- Direct outbound MX delivery with bound public IPv4/IPv6, DKIM signing, automatic TLS.

### 4.3 Cluster (`cluster`)
- N API nodes behind load balancers.
- N SMTP edge nodes with dedicated submission and inbound ports.
- N Worker nodes with `network_mode: host` to bind specific public IP pools.
- External MariaDB Galera cluster and Redis Cluster.
