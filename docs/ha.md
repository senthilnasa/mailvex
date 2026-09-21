# Mailvex — High Availability & Disaster Recovery Architecture

This document provides operational specifications for running Mailvex in a high-availability, fault-tolerant production environment across multi-node clusters (§28, §34).

---

## 1. High Availability Architecture Overview

Mailvex separates stateless components (API Gateway, Inbound SMTP, Webhook Dispatcher) from stateful data stores (MariaDB, Redis, Object Storage).

```mermaid
flowchart TD
    subgraph TrafficManagement [Edge Traffic Management]
        DNS[GeoDNS / Anycast BGP]
        LB[HAProxy / Cloud Load Balancers]
    end

    subgraph APILayer [Stateless API & SMTP Edge]
        API1[Mailvex API Node 1]
        API2[Mailvex API Node 2]
        SMTP1[Inbound SMTP Edge 1]
        SMTP2[Inbound SMTP Edge 2]
    end

    subgraph WorkerPool [Outbound Delivery Workers]
        W1[Delivery Worker Node 1 (IP Pool A)]
        W2[Delivery Worker Node 2 (IP Pool B)]
        W3[Delivery Worker Node 3 (IP Pool C)]
    end

    subgraph DataPlane [Stateful Resilient Cluster]
        MDB_P[(MariaDB Galera / Primary)]
        MDB_R[(MariaDB Replica / DR)]
        REDIS[Redis Sentinel / Cluster]
        S3[(S3 Compatible Object Storage MinIO/Ceph)]
    end

    DNS --> LB
    LB --> API1
    LB --> API2
    LB --> SMTP1
    LB --> SMTP2

    API1 --> MDB_P
    API2 --> MDB_P
    API1 --> REDIS
    API2 --> REDIS
    API1 --> S3
    API2 --> S3

    W1 --> REDIS
    W2 --> REDIS
    W3 --> REDIS
    W1 --> MDB_P
    W2 --> MDB_P
    W3 --> MDB_P

    MDB_P -. Sync Replication .-> MDB_R
```

---

## 2. Component Redundancy

### 2.1 API Gateway & Webhook Nodes
- **State**: Completely stateless.
- **Redundancy**: Minimum 2 instances behind Layer 7 reverse proxy (HAProxy, NGINX, Envoy, AWS ALB).
- **Session Management**: JWT and API Key authentication with DB verification or distributed Redis caching. No sticky sessions required.
- **Health Checks**:
  - Liveness probe: `GET /health` (HTTP 200)
  - Readiness probe: `GET /ready` (Verifies DB & Redis latency)
  - Prometheus Metrics: `GET /metrics`

### 2.2 SMTP Ingestion Edge
- **Ports**: 25 (Inbound MX), 465 (Implicit TLS Submission), 587 (STARTTLS Submission).
- **Redundancy**: Multi-node SMTP edge mapped to equal-priority MX DNS records (`IN MX 10 mx1.mailvex.com`, `IN MX 10 mx2.mailvex.com`).
- **Load Balancing**: If single public IP, load balance via HAProxy with `send-proxy-v2` to preserve client source IP for SPF and rate-limiting.

### 2.3 Outbound Delivery Worker Pool
- **Queue Engine**: Transactional Outbox pattern backed by MariaDB partitions and distributed Redis locks.
- **Worker Fairness**: Workers acquire delivery leases using Redis atomic tokens (`SETNX` with TTL). If a worker crashes mid-transmission, the lease expires and another worker retries.
- **IP Address Binding**: Each worker node or container binds to dedicated outbound IPs via `MVX_OUTBOUND_IPS` with Linux `IP_FREEBIND` capability.

### 2.4 Database (MariaDB / MySQL 8)
- **Primary Setup**: MariaDB Galera Cluster (3 nodes) for synchronous multi-master replication, or MySQL Group Replication.
- **Read Scaling**: Read-heavy reporting queries (analytics rollups, logs inspector) route to read replicas via database connection pool routing.
- **Partitioning**: Messages and Delivery Attempts partitioned monthly (`PARTITION BY RANGE (UNIX_TIMESTAMP(created_at))`).

### 2.5 Redis
- **Topology**: Redis Sentinel (3 sentinels, 1 master, 2 replicas) or Redis Cluster.
- **Usage**: Distributed rate limiting tokens, outbox delivery leases, warmup IP counters.

---

## 3. Disaster Recovery (DR) Playbook

### 3.1 Automated Backups
Run `mailvex backup create` via scheduled Cron or Kubernetes CronJob every 6 hours:
```bash
mailvex backup create --output /backups/mailvex-$(date +%s).tar.gz
```
Archives are cryptographically signed with HMAC-SHA256 (`manifest.json`) and pushed to secondary off-site cold storage.

### 3.2 Point-in-Time Restoration
To restore a replacement instance or disaster recovery cluster:
```bash
mailvex backup restore --input /backups/mailvex-latest.tar.gz
```
The command verifies cryptographic signature integrity before writing state into the repository.
