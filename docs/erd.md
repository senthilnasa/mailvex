# Mailvex — Database ERD & Schema Design

## 1. Overview & Architectural Constraints

Mailvex targets **MariaDB 11+** (compatible with MySQL 8+). High-volume operational tables are **RANGE partitioned by day/week on `created_at`** to allow instant partition drops for data retention purges without disk fragmentation.

Because MariaDB / MySQL partitioned tables cannot have foreign keys pointing to or from them:
- **Hot partitioned tables** use **Application-Enforced Referential Integrity (App-RI)** managed within `core/repository`.
- **Low-volume configuration tables** use native Foreign Keys (`ON DELETE CASCADE` or `RESTRICT`).

---

## 2. Entity-Relationship Diagram (Mermaid)

```mermaid
erDiagram
    ORGANIZATIONS ||--o{ TENANTS : "contains"
    ORGANIZATIONS ||--o{ USERS : "owns"
    ORGANIZATIONS ||--o{ ROLES : "defines custom"
    
    TENANTS ||--o{ TENANT_KEYS : "has DEKs"
    TENANTS ||--o{ TENANT_MEMBERSHIPS : "grants access"
    USERS ||--o{ TENANT_MEMBERSHIPS : "belongs to"
    ROLES ||--o{ ROLE_PERMISSIONS : "bundles"
    PERMISSIONS ||--o{ ROLE_PERMISSIONS : "granted via"
    
    TENANTS ||--o{ DOMAINS : "owns"
    DOMAINS ||--o{ DKIM_KEYS : "has"
    DOMAINS ||--o{ DOMAIN_DNS_RECORDS : "declares"
    
    TENANTS ||--o{ API_KEYS : "issues"
    TENANTS ||--o{ SMTP_CREDENTIALS : "issues"
    TENANTS ||--o{ MESSAGE_STREAMS : "configures"
    TENANTS ||--o{ DEVICES : "registers LAN"
    
    TENANTS ||--o{ MESSAGES : "sends"
    MESSAGES ||--o{ MESSAGE_RECIPIENTS : "targets"
    MESSAGE_RECIPIENTS ||--o{ DELIVERY_ATTEMPTS : "records"
    MESSAGES ||--o{ MESSAGE_EVENTS : "emits"
    
    TENANTS ||--o{ SUPPRESSIONS : "maintains"
    TENANTS ||--o{ WEBHOOKS : "registers"
    WEBHOOKS ||--o{ WEBHOOK_DELIVERIES : "dispatches"
    
    ORGANIZATIONS {
        string id PK "org_..."
        string name
        string slug UK
        string status
        datetime created_at
    }
    
    TENANTS {
        string id PK "tenant_..."
        string organization_id FK
        string name
        string slug UK
        string environment "live | test"
        string status "active | suspended"
        datetime created_at
    }
    
    TENANT_KEYS {
        string id PK "key_..."
        string tenant_id FK
        int key_version
        string encrypted_dek
        string kek_version
        datetime created_at
    }
    
    USERS {
        string id PK "user_..."
        string organization_id FK
        string email UK
        string password_hash
        string status
        boolean is_platform_admin
        datetime created_at
    }
    
    TENANT_MEMBERSHIPS {
        string id PK "mem_..."
        string tenant_id FK
        string user_id FK
        string role_id FK
        datetime created_at
    }
    
    DOMAINS {
        string id PK "dom_..."
        string tenant_id FK
        string name UK
        string type "public | local"
        string status "pending | verifying | verified | degraded | suspended"
        string ownership_token
        datetime verified_at
        datetime created_at
    }
    
    MESSAGES {
        string id PK "msg_..."
        string tenant_id
        string stream_id
        string domain_id
        string from_address
        string subject
        string status
        string environment
        int size_bytes
        datetime created_at
    }
    
    MESSAGE_RECIPIENTS {
        string id PK "rcpt_..."
        string message_id
        string tenant_id
        string address
        string address_domain
        string mx_group
        string status
        int attempts
        datetime next_attempt_at
        datetime created_at
    }
    
    DELIVERY_ATTEMPTS {
        string id PK "att_..."
        string recipient_id
        string tenant_id
        int attempt_no
        string remote_mx
        string smtp_code
        string outcome
        datetime started_at
        datetime created_at
    }
    
    MESSAGE_EVENTS {
        string id PK "evt_..."
        string tenant_id
        string message_id
        string recipient_id
        string type
        json data
        datetime occurred_at
        datetime created_at
    }
    
    SUPPRESSIONS {
        string id PK "sup_..."
        string tenant_id
        string stream_id
        string address
        string address_domain
        string type "hard_bounce | complaint | unsubscribe | manual"
        string reason
        datetime created_at
    }
    
    WEBHOOKS {
        string id PK "wh_..."
        string tenant_id
        string name
        string url
        string encrypted_secret
        boolean active
        datetime created_at
    }
    
    WEBHOOK_DELIVERIES {
        string id PK "whd_..."
        string webhook_id
        string tenant_id
        string event_id
        int status_code
        int attempt_no
        datetime created_at
    }
```

---

## 3. Partitioning Strategy

Hot operational tables:
1. `messages`
2. `message_recipients`
3. `message_events`
4. `delivery_attempts`
5. `tracking_events`
6. `webhook_deliveries`
7. `api_request_logs`
8. `inbound_messages`

Partition definition:
```sql
PARTITION BY RANGE (UNIX_TIMESTAMP(created_at)) (
    PARTITION p_2026_09_20 VALUES LESS THAN (UNIX_TIMESTAMP('2026-09-21 00:00:00')),
    PARTITION p_2026_09_21 VALUES LESS THAN (UNIX_TIMESTAMP('2026-09-22 00:00:00')),
    ...
);
```
The `mailvex-scheduler` service runs daily to create partitions 7 days ahead and drop partitions older than the configured retention policy.

---

## 4. Application-Enforced Referential Integrity (App-RI)

To guarantee database consistency across partitioned tables where foreign keys are unsupported:
1. **Transactional Insertion**: `messages`, `message_recipients`, `message_events`, and `outbox` records are inserted within a single database transaction.
2. **Tenant Scoping Guard**: Every query and insert explicitly includes `tenant_id` and verifies parent record existence before child mutation.
3. **Automated Consistency Tests**: CI test suites specifically test for orphan prevention and transaction rollback.
