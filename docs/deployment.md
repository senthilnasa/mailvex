# Mailvex — Production Docker Deployment Guide

> **Carrier-Grade, Multi-Tenant, Self-Hosted Email Infrastructure Engine**  
> Complete step-by-step instructions for deploying Mailvex across Homelab, Standard, and Enterprise High-Availability Cluster architectures.

---

## Table of Contents

1. [Architecture & Deployment Profiles](#1-architecture--deployment-profiles)
2. [Prerequisites & Port Requirements](#2-prerequisites--port-requirements)
3. [Environment Configuration (`.env`)](#3-environment-configuration-env)
4. [Deployment Profiles](#4-deployment-profiles)
   - [Profile A: Homelab All-in-One (Minimal Footprint)](#profile-a-homelab-all-in-one-minimal-footprint)
   - [Profile B: Standard Production (Decoupled API & Workers)](#profile-b-standard-production-decoupled-api--workers)
   - [Profile C: Enterprise High-Availability Cluster](#profile-c-enterprise-high-availability-cluster)
5. [First-Time Access & Setup Wizard](#5-first-time-access--setup-wizard)
6. [Domain Verification & DKIM Configuration](#6-domain-verification--dkim-configuration)
7. [Production Hardening & Reverse Proxy (SSL/TLS)](#7-production-hardening--reverse-proxy-ssltls)
   - [Caddy (Automatic HTTPS)](#caddy-automatic-https)
   - [Nginx Configuration](#nginx-configuration)
8. [Smart-Host Relay Configuration (ISP Port 25 Bypass)](#8-smart-host-relay-configuration-isp-port-25-bypass)
9. [Operational Diagnostics & Maintenance](#9-operational-diagnostics--maintenance)
   - [Mailvex Doctor CLI](#mailvex-doctor-cli)
   - [Cryptographic Backup & Restore](#cryptographic-backup--restore)
   - [Viewing Service Logs](#viewing-service-logs)
10. [Troubleshooting & FAQ](#10-troubleshooting--faq)

---

## 1. Architecture & Deployment Profiles

Mailvex provides three deployment profiles via `MVX_PROFILE`:

| Profile | Target Environment | Memory Footprint | Architecture | Compose File |
|---|---|---|---|---|
| **`homelab`** | Homelabs, Edge nodes, IoT hubs, Single-board computers | **< 200MB RAM** | Single unified process (API + Console + Inbound SMTP + Outbound Worker) with MariaDB 11.4 & Redis 7.2 | [`deploy/compose/docker-compose.homelab.yml`](../deploy/compose/docker-compose.homelab.yml) |
| **`standard`** | Startups, SMBs, Single Cloud Instances (VPS) | **~500MB RAM** | Independent API and Outbound Worker containers with MariaDB 11.4 & Redis 7.2 | [`deploy/compose/docker-compose.standard.yml`](../deploy/compose/docker-compose.standard.yml) |
| **`cluster`** | High-volume senders, Multi-node enterprise | **Dynamic Scale** | Horizontal API tier behind HAProxy, distributed Redis queue workers, Prometheus & Grafana | [`deploy/compose/docker-compose.cluster.yml`](../deploy/compose/docker-compose.cluster.yml) |

---

## 2. Prerequisites & Port Requirements

### System Requirements
* **Docker Engine:** Version 24.0 or newer
* **Docker Compose:** Version 2.20 or newer
* **OS:** Linux (Ubuntu, Debian, AlmaLinux, Rocky), macOS, or Windows with WSL2

### Network Port Allocation

| Port | Protocol | Service | Required Access | Purpose |
|---|---|---|---|---|
| **`8080`** | TCP / HTTP | Mailvex Web Console & REST API | Internal / Reverse Proxy | Admin operator dashboard, API dispatch |
| **`25`** | TCP / SMTP | Public MX Edge | Public Internet (Inbound & Outbound) | Direct mail delivery and receiving |
| **`587`** | TCP / SMTP | Client Submission | Authenticated Users / Apps | Submission edge with mandatory STARTTLS |
| **`2525`** | TCP / SMTP | Alternate Submission | Local LAN / IoT devices | Unencrypted / internal submission fallback |
| **`3000`** | TCP / HTTP | Grafana Dashboards | Admins *(Cluster Profile only)* | Observability & delivery telemetry |
| **`9090`** | TCP / HTTP | Prometheus Engine | Internal *(Cluster Profile only)* | Metrics collection |

---

## 3. Environment Configuration (`.env`)

Before starting the containers, create your environment file in the repository root:

```bash
cp .env.example .env
```

### Generating the Cryptographic Master KEK (Key Encryption Key)
Mailvex employs envelope encryption to secure per-tenant data encryption keys (DEK), stored credentials, and DKIM private keys. Generate a cryptographically strong 32-byte master key:

```bash
# On Linux / macOS:
openssl rand -base64 32

# On Windows (PowerShell):
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Minimum 0 -Maximum 256 }))
```

Copy the generated key and set it in your `.env`:

```ini
# Operational Profile: homelab | standard | cluster
MVX_PROFILE=homelab

# App & HTTP Server
MVX_APP_ENV=production
MVX_APP_PORT=8080
MVX_APP_URL=https://mail.yourdomain.com

# Master Encryption Key (32-byte Base64)
MVX_MASTER_KEY=bWFpbHZleC1ob21lbGFiLTMyYnl0ZS1tYXN0ZXJrZWs=
MVX_KEK_PROVIDER=env

# Database Configuration (MariaDB 11.4+)
MVX_DB_HOST=mariadb
MVX_DB_PORT=3306
MVX_DB_USER=mailvex
MVX_DB_PASSWORD=your_secure_db_password
MVX_DB_NAME=mailvex

# Redis Configuration (Redis 7.2+)
MVX_REDIS_ADDR=redis:6379
MVX_REDIS_PASSWORD=

# Platform FQDN & Reverse DNS hostname
MVX_PLATFORM_HOST=mail.yourdomain.com
MVX_SMTP_HOSTNAME=mail.yourdomain.com
```

---

## 4. Deployment Profiles

### Profile A: Homelab All-in-One (Minimal Footprint)

The Homelab profile consolidates the Web Console, REST API, SMTP Edge, and Delivery Worker into a single multi-arch container running alongside lightweight MariaDB and Redis services.

```bash
# 1. Clone the repository
git clone https://github.com/senthilnasa/mailvex.git
cd mailvex

# 2. Build and launch the stack
docker compose -f deploy/compose/docker-compose.homelab.yml up -d --build

# 3. Verify container health
docker compose -f deploy/compose/docker-compose.homelab.yml ps
```

Expected output:
```text
NAME                IMAGE                            STATUS                   PORTS
mailvex-homelab     mailvex/mailvex-allinone:latest  Up (healthy)             0.0.0.0:25->25/tcp, 0.0.0.0:2525->2525/tcp, 0.0.0.0:8080->8080/tcp
mailvex-mariadb     mariadb:11.4                     Up (healthy)             3306/tcp
mailvex-redis       redis:7.2-alpine                 Up (healthy)             6379/tcp
```

---

### Profile B: Standard Production (Decoupled API & Workers)

For production workloads, the API and Delivery Worker run in separate containers, enabling independent scaling and isolated resource limits.

```bash
# 1. Build and launch the standard production stack
docker compose -f deploy/compose/docker-compose.standard.yml up -d --build

# 2. Verify containers
docker compose -f deploy/compose/docker-compose.standard.yml ps
```

Services started:
* `mailvex-api`: High-concurrency REST endpoints and Webhook receiver.
* `mailvex-worker`: Outbound MX delivery, backoff retries, and DKIM signing engine.
* `mailvex-standard-db`: Persistent MariaDB instance.
* `mailvex-standard-redis`: Stream queue for outbox messages.

---

### Profile C: Enterprise High-Availability Cluster

For high-volume senders, the cluster profile provisions a horizontal API tier behind an HAProxy load balancer with Prometheus and Grafana monitoring:

```bash
docker compose -f deploy/compose/docker-compose.cluster.yml up -d
```

* **HAProxy Dashboard:** `http://your-server-ip:8404`
* **Grafana Dashboards:** `http://your-server-ip:3000` (Pre-configured with deliverability, latency, and queue depth metrics)
* **API Entrypoint:** `http://your-server-ip:80`

---

## 5. First-Time Access & Setup Wizard

Once your containers are running:

1. Open your browser and navigate to:  
   **`http://localhost:8080`** (or your server IP / domain).
2. **Default Administrator Credentials:**
   * **Email:** `admin@example.com`
   * **Password:** `Password123!`
3. **Change Administrator Password:**
   Navigate to **Settings & Organization** in the console, or execute an immediate CLI reset:
   ```bash
   curl -X POST http://localhost:8080/v1/auth/reset-admin \
     -H "Content-Type: application/json" \
     -d '{"email":"admin@yourdomain.com","password":"YourNewStrongPassword123!"}'
   ```

---

## 6. Domain Verification & DKIM Configuration

To send emails with carrier-grade deliverability, register and verify your sending domain:

1. In the Web Console, go to **Domains** &rarr; **Add Domain** (e.g. `mail.yourdomain.com`).
2. Mailvex automatically generates an **RSA 2048-bit DKIM Key Pair** and DNS verification tokens.
3. Add the following records to your public DNS provider (Cloudflare, Route53, Namecheap, etc.):

| Record Type | Host / Name | Target / Content | Purpose |
|---|---|---|---|
| **TXT** | `_mailvex-verify.mail.yourdomain.com` | `mailvex-verify=<token_from_console>` | Domain Ownership Verification |
| **TXT** | `mvx._domainkey.mail.yourdomain.com` | `v=DKIM1; k=rsa; p=<public_key>` | Cryptographic DKIM Signature |
| **TXT** | `mail.yourdomain.com` | `v=spf1 ip4:<YOUR_SERVER_PUBLIC_IP> ~all` | Sender Policy Framework (SPF) |
| **TXT** | `_dmarc.mail.yourdomain.com` | `v=DMARC1; p=reject; rua=mailto:dmarc@yourdomain.com` | DMARC Enforcement Policy |
| **CNAME** | `bounce.mail.yourdomain.com` | `mail.yourdomain.com` | Return-Path / Bounce Handling |

4. In the console, click **Verify Domain** (or call `POST /v1/domains/{id}/verify`). Once confirmed, status transitions to `Verified (Active)`.

---

## 7. Production Hardening & Reverse Proxy (SSL/TLS)

Do not expose port 8080 unencrypted to the public internet. Use a reverse proxy to terminate SSL/TLS.

### Caddy (Automatic HTTPS)

Create a `Caddyfile`:

```caddy
mail.yourdomain.com {
    reverse_proxy localhost:8080
}
```

Start Caddy:
```bash
caddy run
```
*Caddy automatically obtains and renews Let's Encrypt / ZeroSSL certificates.*

---

### Nginx Configuration

Create `/etc/nginx/sites-available/mailvex.conf`:

```nginx
server {
    listen 80;
    server_name mail.yourdomain.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name mail.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/mail.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mail.yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 8. Smart-Host Relay Configuration (ISP Port 25 Bypass)

Many residential and cloud providers (e.g. AWS EC2, DigitalOcean, Hetzner, residential ISPs) block outbound TCP port 25 to prevent spam. 

Mailvex has native **Smart-Host Relay** support to route outbound messages via port `587` through Google Workspace, Amazon SES, SendGrid, Mailgun, or Brevo:

1. Open the Mailvex Console &rarr; **Relays & Smart Hosts** &rarr; **Add Relay**.
2. Fill in your provider details:
   * **Host:** `smtp-relay.gmail.com` or `email-smtp.us-east-1.amazonaws.com`
   * **Port:** `587`
   * **TLS Mode:** `STARTTLS`
   * **Username:** `your-smtp-username`
   * **Password:** `your-smtp-password`
3. Toggle **Active**. Mailvex will immediately route outbound email via the smart host with zero code changes.

---

## 9. Operational Diagnostics & Maintenance

### Mailvex Doctor CLI
Run diagnostic audits on network connectivity, database latency, and DNS resolution:

```bash
# In Homelab container:
docker compose -f deploy/compose/docker-compose.homelab.yml exec mailvex mailvex doctor

# In Standard container:
docker compose -f deploy/compose/docker-compose.standard.yml exec api mailvex doctor
```

Sample audit output:
```text
[✓] Kernel File Descriptors: 1048576 (Optimal)
[✓] MariaDB Database Connectivity: Latency 1.2ms
[✓] Redis Stream Queue: Connected
[✓] Master KEK Verification: Valid AES-256-GCM
[✓] Outbound Port 25 Check: Reachable
[✓] Reverse DNS (PTR): Verified mail.yourdomain.com
```

---

### Cryptographic Backup & Restore

Mailvex includes backup utilities signed with HMAC-SHA256:

```bash
# Create encrypted backup archive:
docker compose -f deploy/compose/docker-compose.homelab.yml exec mailvex \
  mailvex backup create --output=/var/lib/mailvex/backup.tar.gz

# Restore from backup archive:
docker compose -f deploy/compose/docker-compose.homelab.yml exec mailvex \
  mailvex backup restore --input=/var/lib/mailvex/backup.tar.gz
```

---

### Viewing Service Logs

```bash
# Follow all container logs:
docker compose -f deploy/compose/docker-compose.homelab.yml logs -f

# Follow delivery worker logs specifically:
docker compose -f deploy/compose/docker-compose.standard.yml logs -f worker
```

---

## 10. Troubleshooting & FAQ

### Q: Port 8080 is already in use by another application.
**A:** Edit your compose file or `.env` and change the host port mapping:
```yaml
ports:
  - "8090:8080" # Maps local port 8090 to Mailvex port 8080
```

### Q: My emails are deferred with "connection failed: dial tcp ...:25: i/o timeout".
**A:** Your cloud provider or ISP has blocked outbound port 25. Configure a **Smart-Host Relay** on port 587 (Section 8) to bypass the ISP block.

### Q: How do I test email sending without external deliverability?
**A:** Generate an API Key with `environment: test` (prefix `mvx_test_...`). Messages sent with test keys are captured in the in-memory development sink at `GET /v1/messages/development` without generating external network traffic.

### Q: Where are health check probes for Kubernetes or load balancers?
* **Liveness:** `GET /health` (Returns HTTP 200)
* **Readiness:** `GET /ready` (Verifies database connection pool and Redis cache)
* **Metrics:** `GET /metrics` (Prometheus-compatible format)
