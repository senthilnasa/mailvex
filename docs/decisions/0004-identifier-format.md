# ADR 0004: Primary Key & Public Identifier Format

## Status
Accepted

## Context
Mailvex needs unique identifiers for entities across a multi-tenant distributed system. Requirements:
- Millisecond-sortable for cursor-based pagination
- High-entropy to prevent enumeration attacks
- Human-readable prefixes to immediately identify entity types in logs and API payloads
- Efficient database indexing

## Decision
1. **Type**: Universally Unique Lexicographically Sortable Identifier (**ULID**).
2. **Database Storage**: `CHAR(26)` storing the standard Crockford base32 ULID representation (or `VARCHAR(36)` including type prefix where natural).
3. **Public API Representation**: Strict typed prefixes:
   - `org_` : Organization
   - `tenant_` : Tenant
   - `user_` : User
   - `dom_` : Domain
   - `msg_` : Message
   - `rcpt_` : Message Recipient
   - `att_` : Delivery Attempt
   - `evt_` : Message Event
   - `key_` : API Key
   - `smtp_` : SMTP Credential
   - `wh_` : Webhook
   - `whd_` : Webhook Delivery
   - `ip_` : IP Address
   - `pool_` : IP Pool
   - `stream_` : Message Stream
   - `sup_` : Suppression
   - `req_` : Request ID
4. Sequential auto-increment integers are strictly forbidden as public identifiers.

## Consequences
- Uniform helper library `core/crypto/ulid.go` generates, prefixes, and parses all IDs.
- Cursors encode `(created_at, id)` cleanly.
