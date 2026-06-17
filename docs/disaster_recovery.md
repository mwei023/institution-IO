# Disaster Recovery Playbook
## Institution OS
**RTO**: ≤ 4 hours | **RPO**: ≤ 1 hour

---

## 1. Backup Schedule

| Frequency | Type | Retention | Storage |
|-----------|------|-----------|---------|
| Hourly | WAL archive (incremental) | 7 days | Object storage |
| Daily 02:00 EAT | Full pg_dump | 30 days | Object storage |
| Weekly Sunday 03:00 | Full pg_dump | 12 weeks | Object storage |
| Monthly 1st 04:00 | Full pg_dump | 24 months | Object storage |

Backups encrypted with AES-256 before upload. Encryption key stored separately from backup files.

---

## 2. Backup Script

```bash
# scripts/backup.sh
#!/bin/bash
set -euo pipefail

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="institution_os_${TIMESTAMP}.sql.gz"
ENCRYPTED_FILE="${BACKUP_FILE}.enc"

# Dump and compress
pg_dump "$DATABASE_URL" | gzip > "/tmp/${BACKUP_FILE}"

# Encrypt
openssl enc -aes-256-cbc -pbkdf2 -in "/tmp/${BACKUP_FILE}" \
  -out "/tmp/${ENCRYPTED_FILE}" -pass "env:BACKUP_ENCRYPTION_KEY"

# Upload to object storage
aws s3 cp "/tmp/${ENCRYPTED_FILE}" \
  "s3://${BACKUP_BUCKET}/daily/${ENCRYPTED_FILE}"

# Cleanup
rm "/tmp/${BACKUP_FILE}" "/tmp/${ENCRYPTED_FILE}"

echo "Backup complete: ${ENCRYPTED_FILE}"
```

---

## 3. Restore Procedure

### Step 1: Assess
- Identify failure type: DB corruption / server loss / accidental deletion
- Check latest backup timestamp: `aws s3 ls s3://${BACKUP_BUCKET}/daily/ | tail -5`

### Step 2: Provision (if server lost)
```bash
# Spin up new VPS, install dependencies
apt-get install -y postgresql-15 redis-server docker.io

# Or restore from Docker Compose:
cd /opt/institution-os
docker-compose up -d postgres redis
```

### Step 3: Restore Database
```bash
# scripts/restore.sh
#!/bin/bash
BACKUP_FILE=$1  # e.g. institution_os_20240115_020000.sql.gz.enc

# Download
aws s3 cp "s3://${BACKUP_BUCKET}/daily/${BACKUP_FILE}" /tmp/

# Decrypt
openssl enc -d -aes-256-cbc -pbkdf2 \
  -in "/tmp/${BACKUP_FILE}" \
  -out "/tmp/restore.sql.gz" \
  -pass "env:BACKUP_ENCRYPTION_KEY"

# Restore
gunzip -c /tmp/restore.sql.gz | psql "$DATABASE_URL"

echo "Restore complete"
```

### Step 4: Apply WAL (Point-in-Time Recovery)
```bash
# Apply WAL archives from last full backup up to desired timestamp
pg_restore_command="aws s3 cp s3://${BACKUP_BUCKET}/wal/%f %p"
recovery_target_time="2024-01-15 09:30:00+03"
# Configure recovery.conf and restart PostgreSQL
```

### Step 5: Verify
```bash
# Check row counts match expectations
psql $DATABASE_URL -c "SELECT COUNT(*) FROM students WHERE institution_id = '11111111-...';"
psql $DATABASE_URL -c "SELECT COUNT(*) FROM fee_payments WHERE institution_id = '11111111-...';"
psql $DATABASE_URL -c "SELECT MAX(created_at) FROM audit_logs;"
```

### Step 6: Restart Application
```bash
cd /opt/institution-os
docker-compose up -d backend frontend
# Run migrations to ensure schema is current
docker-compose exec backend ./server migrate up
```

### Step 7: Notify
- Notify Super Admin via phone/email: estimated restore time, data loss window (if any)
- Update status page

---

## 4. Monthly Restore Test (Automated)

```yaml
# .github/workflows/restore-test.yml
name: Monthly DR Test
on:
  schedule:
    - cron: '0 2 1 * *'  # 1st of month at 02:00 UTC
jobs:
  restore-test:
    runs-on: ubuntu-latest
    steps:
      - name: Download latest backup
        run: aws s3 cp s3://$BACKUP_BUCKET/daily/$(aws s3 ls ... | tail -1 | awk '{print $4}') /tmp/backup.enc
      - name: Restore to test DB
        run: |
          # decrypt, restore to staging DB
          # run row count checks
          # send Slack notification with result
```

---

## 5. Emergency Contacts

| Role | Name | Phone | Email |
|------|------|-------|-------|
| Lead Engineer | — | — | engineering@institution-os.com |
| DevOps | — | — | ops@institution-os.com |
| On-call (PagerDuty) | — | — | — |
