# Mailvex — Technical Documentation Hub

Welcome to the official technical documentation for **Mailvex**, the carrier-grade, multi-tenant, self-hosted email infrastructure and delivery engine.

---

## 🧭 Documentation Sitemap

### 1. 🚀 Getting Started & Deployment
- [**Production Docker Deployment Guide**](deployment.md) — Comprehensive guide for deploying Mailvex across Homelab, Standard, and High-Availability Cluster architectures. Includes reverse proxy (Caddy/Nginx), SSL/TLS, and smart-host configurations.
- [**Homelab & Edge Deployment**](homelab.md) — Guide for running Mailvex in ultra-low-memory environments (< 200MB RAM) such as Raspberry Pi, Proxmox LXC, and homelab servers.
- [**High-Availability (HA) Clustering**](ha.md) — Multi-node enterprise topology with horizontal API scaling, HAProxy load balancing, Redis clustering, and read-replica databases.
- [**Zero-Downtime Upgrade Guide**](upgrade.md) — Step-by-step procedures for database schema migrations, blue-green deployments, and rolling binary updates.

---

### 2. 📡 API, SDKs & Agent Integration
- [**REST API Specification**](api.md) — Full canonical specification for the Mailvex HTTP API (`/v1`), covering dual-mode API keys, message dispatching, batch sends, custom domains, templates, and error codes.
- [**AI Agent & Autonomous System Protocol**](ai-agents.md) — Guide for integrating AI agents (Cursor, Claude Code, Copilot, LangChain) with Mailvex, including machine-readable function calling schemas, MCP server configs, and idempotency guarantees.
- [**Webhooks Engine & Signatures**](webhooks.md) — Real-time event notifications (`delivered`, `bounced`, `opened`, `clicked`, `complained`) with HMAC-SHA256 signature verification.
- [**Go Client SDK (`sdk/go`)**](../sdk/go/README.md) — Official, thread-safe, zero-dependency Go client library published at `pkg.go.dev/github.com/senthilnasa/mailvex/sdk/go/mailvex`.
- [**PHP & Laravel SDK (`sdk/php`)**](../sdk/php/README.md) — Official PHP library and native Laravel Mail Transport driver (`mailvex/mailvex-php`).
- [**OpenAPI 3.1 Specification**](../sdk/openapi.yaml) — Formal machine-readable OpenAPI contract.

---

### 3. 🏗 Core Architecture & Engineering
- [**System Architecture**](architecture.md) — Deep dive into the Mailvex supervisor, dual ingestion edges (REST API & SMTP), and asynchronous worker pipelines.
- [**Delivery Engine & Direct MX**](delivery.md) — Mechanics of direct MX DNS resolution, TLS opportunistic handshakes, connection pooling, and smart-host fallback.
- [**Redis Queue Topology**](queue.md) — Worker dispatch architecture, prioritized queues, retry exponential backoff schedules, and dead-letter handling.
- [**Database Schema & ERD**](erd.md) — Full entity-relationship diagrams and table schemas across MariaDB partitions.
- [**Networking & Ports**](networking.md) — Port allocations (25, 587, 2525, 8080, 9090), firewall rules, and NAT traversal.

---

### 4. 🛡 Deliverability, Reputation & Security
- [**Deliverability & Reputation Center**](deliverability.md) — DKIM RSA-2048 signing, SPF validation, DMARC enforcement, FCrDNS validation, automated 30-day IP warmup, and real-time DNSBL monitoring.
- [**Security Architecture & Cryptography**](security.md) — Envelope encryption (DEK/KEK), AES-256-GCM, TLS cipher suites, and RBAC privilege separation.
- [**Security Audit & Hardening**](security_audit.md) — Enterprise security checklist, vulnerability disclosures, and production hardening recommendations.
- [**Multi-Tenancy & Boundary Isolation**](multi-tenancy.md) — Strict logical separation of workspaces, cryptographic tenant keys, and platform administrator impersonation safeguards.
- [**Third-Party Relays & Smart-Hosts**](third-party.md) — Outbound relay integration with SendGrid, Mailgun, Amazon SES, or custom SMTP relays for networks where Port 25 is blocked by ISPs.

---

### 5. 📜 Architecture Decision Records (ADRs)
Mailvex maintains formal Architectural Decision Records in the [`decisions/`](decisions/) directory:
- [**ADR 0001: Architecture Principles**](decisions/0001-architecture-principles.md) — Core values of simplicity, zero external runtime baggage, and cryptographic privacy.
- [**ADR 0002: SMTP Edge Strategy**](decisions/0002-smtp-edge.md) — Custom Go SMTP edge vs Postfix/Exim.
- [**ADR 0003: Reverse Proxy Architecture**](decisions/0003-reverse-proxy.md) — Edge TLS termination strategy.
- [**ADR 0004: Identifier Format**](decisions/0004-identifier-format.md) — UUIDv7 time-ordered identifiers for message tracing.
- [**ADR 0005: Partitioning & Referential Integrity**](decisions/0005-partitioning-and-referential-integrity.md) — High-volume message table partitioning.
- [**ADR 0006: Role Assignment Model**](decisions/0006-role-assignment-model.md) — Multi-tenant RBAC permissions.

---

### 6. 🔬 Internal Specifications
Internal developer prompts and milestone execution records are archived in the [`internals/`](internals/) directory:
- [**Master Development Prompt**](internals/MASTER_PROMPT.md)
- [**Milestone Execution Log**](internals/EXECUTION_LOG.md)

---

## 🤝 Contributing to Documentation
If you find a typo or wish to expand an existing guide, please open a pull request or submit an issue on GitHub at [github.com/senthilnasa/mailvex](https://github.com/senthilnasa/mailvex).
