# Mailvex — Zero-Downtime Rolling Upgrades & Migration Playbook

This document details the operational procedure for performing zero-downtime upgrades across Mailvex services (§28, §34).

---

## 1. Upgrade Principles

1. **Backward Compatibility**: Every minor release maintains full backward compatibility with preceding schema versions.
2. **Expand / Contract Database Migrations**: Database schema modifications follow the Expand / Contract pattern:
   - **Step 1 (Expand)**: Add nullable columns or new tables. Do not remove or alter existing columns.
   - **Step 2 (Deploy Code)**: Deploy new application code that writes to both old and new structures, or handles nullable new columns gracefully.
   - **Step 3 (Contract)**: After all nodes run the new version, a subsequent migration cleans up decommissioned columns.
3. **Graceful Worker Draining**: Workers trap `SIGTERM` / `SIGINT`, finish active SMTP dialogues, release queue leases, and exit within 30 seconds.

---

## 2. Step-by-Step Upgrade Procedure

### Step 1: Pre-Upgrade Health Check & Backup
Run the automated system diagnostic:
```bash
mailvex doctor
```
Ensure all critical checks show `[PASS]`. Create an immutable point-in-time backup:
```bash
mailvex backup create --output /var/backups/pre-upgrade-$(date +%s).tar.gz
```

### Step 2: Database Migration Check
Verify pending migrations before applying:
```bash
mailvex migrate status
```
Apply forward migrations:
```bash
mailvex-api --migrate-only
```
Verify schema checksum:
```bash
mailvex migrate verify
```

### Step 3: Rolling Deployment of Stateless Services
1. **API Nodes**:
   - Update Node 1 container image.
   - Wait for readiness probe (`GET /ready` returns 200).
   - Update Node 2 container image.
2. **SMTP Inbound Edge**:
   - Rolling update of SMTP instances. Traffic shifts automatically to alternate MX peers.

### Step 4: Rolling Update of Delivery Workers
1. Send `SIGTERM` to Worker Node 1.
2. Worker finishes processing in-flight email batches and stops dequeuing new tasks.
3. Replace container image and restart worker.
4. Verify Prometheus metrics (`mailvex_messages_delivered_total` incrementing).
5. Repeat for remaining worker instances.

### Step 5: Post-Upgrade Verification
Run the system doctor to confirm full cluster operational readiness:
```bash
mailvex doctor
```
Inspect Prometheus `/metrics` for zero 5xx errors.

---

## 3. Rollback Playbook

If a critical issue occurs:
1. Re-deploy previous container image tags (`v1.x.x`).
2. Because migrations follow the Expand / Contract pattern, the previous code version runs seamlessly against the expanded schema without rolling back the database.
3. If database restoration is strictly necessary:
   ```bash
   mailvex backup restore --input /var/backups/pre-upgrade-*.tar.gz
   ```
