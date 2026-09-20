# ADR 0005: Table Partitioning and Referential Integrity

## Status
Accepted

## Context
High-volume mail platforms accumulate millions of operational events, delivery attempts, and messages. Deleting old records using `DELETE FROM messages WHERE created_at < ...` causes catastrophic table locking, transaction log bloat, and severe disk fragmentation in MariaDB / MySQL.

Table partitioning by `RANGE (UNIX_TIMESTAMP(created_at))` solves retention purges through instantaneous `ALTER TABLE ... DROP PARTITION`. However, MariaDB/MySQL engine constraints prohibit foreign keys referencing or originating from partitioned tables.

## Decision
1. **High-Volume Tables Partitioned**:
   - `messages`, `message_recipients`, `message_events`, `delivery_attempts`, `tracking_events`, `webhook_deliveries`, `api_request_logs`, `inbound_messages`.
2. **Application-Level Referential Integrity (App-RI)**:
   - For partitioned tables, referential integrity is strictly verified in the `core/repository` layer within database transactions.
   - Core repository constructors require tenant context and enforce verification of parent entity existence before child creation.
3. **Configuration Tables Retain Native Foreign Keys**:
   - `organizations`, `tenants`, `users`, `domains`, `api_keys`, `smtp_credentials`, `message_streams` use native InnoDB foreign keys with cascade constraints.

## Consequences
- Retention management operates at near-zero I/O overhead.
- Automated tests in CI verify application-level transaction integrity and orphan avoidance.
