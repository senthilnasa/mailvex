# Mailvex — Queue, Outbox & Scheduling Architecture

## 1. Transactional Outbox Pattern

To eliminate distributed transaction inconsistencies between the database and message broker:
1. When an email is accepted via HTTP or SMTP, the message, recipients, initial event, and an **outbox record** are committed inside a single MariaDB transaction.
2. The `mailvex-scheduler` (or in-process outbox relay) polls `outbox` records where `published_at IS NULL AND available_at <= NOW()`.
3. It pushes delivery tasks to **Redis Streams** and marks the outbox record as published.
4. If Redis crashes or drops data, the database remains the source of truth, and the reconciler seamlessly re-enqueues unpublished or stalled messages.

---

## 2. Priority Classes & Fair Scheduling

Mailvex organizes delivery across four priority streams:
- `mvx:delivery:p0` (Critical - password resets, MFA codes)
- `mvx:delivery:p1` (Transactional - receipts, order confirmations)
- `mvx:delivery:p2` (Broadcast - marketing, announcements)
- `mvx:delivery:p3` (Bulk - high-volume newsletters)

### Fair Scheduling
- Workers pull from streams with weighted ratios (e.g. `8:4:2:1`) to prevent bulk campaigns from starving transactional emails.
- Within each stream, round-robin batch scheduling across tenants ensures that no single tenant monopolizes workers.
- Delayed deliveries (scheduled emails, retry backoffs) utilize a Redis sorted set `mvx:delayed` scored by timestamp.
