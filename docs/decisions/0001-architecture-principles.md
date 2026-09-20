# ADR 0001: Core Architecture Principles

## Status
Accepted

## Context
Mailvex must serve diverse deployment contexts from single-board computers (home labs) to multi-node carrier-grade platforms. To maintain maintainability, security, and avoid fragmented business logic, foundational architectural boundaries must be established.

## Decisions

1. **Clean / Hexagonal Architecture**:
   - `core/domain`: Pure entities, interfaces, business rules. Zero external dependencies.
   - `core/application`: Use cases, orchestration, transactional boundaries.
   - `adapters`: Database (MariaDB), Cache/Queue (Redis), HTTP (REST API), SMTP edge, Storage (FS/S3).
2. **Strict Delivery Ownership**:
   - Mailvex Worker is the single owner of outbound mail delivery and retry scheduling.
   - The SMTP edge is an ingestion and reception adapter only; its outbound queue is completely disabled.
3. **Database as Source of Truth with Redis Delivery Hints**:
   - API and SMTP ingestion write messages, recipients, and transactional outbox entries in a single MariaDB transaction.
   - Redis Streams conveys delivery hints to Workers. If Redis is flushed or unavailable, the system recovers directly from MariaDB.
4. **Tenant Isolation Enforced at Kernel Layer**:
   - Every tenant-scoped repository requires `tenantID` as an explicit parameter and prepends `WHERE tenant_id = ?`. Isolation is never left to API route handlers alone.

## Consequences
- Single runtime for delivery logic prevents race conditions and divergent retry behaviors.
- Monorepo keeps Go data structures synchronized across API, Worker, SMTP edge, and Webhooks.
