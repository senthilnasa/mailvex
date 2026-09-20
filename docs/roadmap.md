# Mailvex — Development Roadmap

This roadmap mirrors §35 of `MAILVEX_MASTER_PROMPT.md`. Each phase is executed in strict sequence with full acceptance tests.

- **Phase 1: Foundation**
  - Monorepo, Docker Compose (dev, homelab, standard), MariaDB, Redis
  - Core kernel: envelope encryption (KEK/DEK), database pool, migrations, seeder
  - Organizations, tenants, users, roles, permissions, RBAC
  - Tenant-scoped repository layer + automated cross-tenant isolation tests
  - Authentication: Argon2id, Redis sessions, audit logs
  - API HTTP server skeleton, setup wizard (steps 1–7), health & readiness probes

- **Phase 2: Domains & DNS**
  - Domains, ownership TXT, global uniqueness & delegations, local domains
  - Multi-resolver DNS verification engine (SPF, DKIM, DMARC, MX, PTR)
  - DKIM key generation & rotation (RSA 2048), domain health status, message streams

- **Phase 3: Message Model, Queue & SMTP Ingestion**
  - Messages, recipients, events, delivery attempts
  - Outbox pattern + Redis Streams + delayed scheduler
  - SMTP edge (`emersion/go-smtp`): AUTH, TLS, tenant mapping, trusted networks, LAN device registry
  - Open-relay security test suite

- **Phase 4: API Sending**
  - API keys, `POST /v1/email`, batch sending, idempotency semantics
  - Header validation, CRLF injection prevention, OpenAPI validation

- **Phase 5: Outbound Delivery Engine**
  - Mailvex Worker: priority consumption, suppression checks, IP selection
  - MX resolution, grouping, TLS policy (MTA-STS, DANE), DKIM signing, SMTP client
  - Smart-host relay transport, retry scheduler, DLQ

- **Phase 6: Async Bounces & Feedback Loops**
  - VERP token generation & parsing, DSN parser, ARF/FBL parser, Yahoo/JMRP/Postmaster

- **Phase 7: Webhooks**
  - HMAC-SHA256 signing, timestamp replay guard, secret rotation, auto-disable

- **Phase 8: Suppression & Privacy**
  - Suppression management (import/export), GDPR/DPDP erasure & export, retention purge

- **Phase 9: Templates Engine**
  - Mustache-compatible templating, HTML escaping, syntax validation, versioning

- **Phase 10: Tracking & Analytics**
  - Signed click/open tracking, bot detection & filtering, hourly/daily rollups

- **Phase 11: Inbound Email & Push Routing**
  - Inbound MX routes, auth verification, mail-to-push bridges (ntfy, Telegram, HA)

- **Phase 12: Reputation & SSO**
  - DNSBL monitoring, warmup progression, OIDC/SAML integration

- **Phase 13: Production Hardening**
  - Security review, HashiCorp Vault KEK adapter, backup/restore CLI (`mailvex backup`)

- **Phase 14: Packaging & Release**
  - Multi-arch Docker images (`amd64`/`arm64`), Go SDK, `mailvex` CLI
