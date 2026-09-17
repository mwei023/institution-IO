#!/usr/bin/env bash
# =============================================================
# Institution OS — Database Restore Script
# Usage: ./scripts/restore.sh <backup_filename>
# Example: ./scripts/restore.sh institution_os_daily_20241015_020000.sql.gz.enc
# =============================================================
set -euo pipefail

BACKUP_FILE="${1:-}"
TMP_DIR="/tmp/ios_restore"

# ── Validate arguments ────────────────────────────────────────
if [[ -z "${BACKUP_FILE}" ]]; then
  echo "Usage: $0 <backup_filename>"
  echo ""
  echo "To list available backups:"
  echo "  aws s3 ls s3://\${BACKUP_BUCKET}/daily/"
  echo "  aws s3 ls s3://\${BACKUP_BUCKET}/weekly/"
  exit 1
fi

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${BACKUP_ENCRYPTION_KEY:?BACKUP_ENCRYPTION_KEY is required}"
: "${BACKUP_BUCKET:?BACKUP_BUCKET is required}"

STORAGE_CMD="aws s3"
if [[ -n "${MINIO_ENDPOINT:-}" ]]; then
  STORAGE_CMD="aws s3 --endpoint-url ${MINIO_ENDPOINT}"
fi

# ── Determine backup type from filename ──────────────────────
if [[ "${BACKUP_FILE}" =~ _daily_ ]]; then
  BACKUP_TYPE="daily"
elif [[ "${BACKUP_FILE}" =~ _weekly_ ]]; then
  BACKUP_TYPE="weekly"
elif [[ "${BACKUP_FILE}" =~ _monthly_ ]]; then
  BACKUP_TYPE="monthly"
else
  BACKUP_TYPE="daily"
fi

mkdir -p "${TMP_DIR}"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "=== Institution OS Restore ==="
echo "Backup file: ${BACKUP_FILE}"
echo "Started:     $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

# ── Safety confirmation ───────────────────────────────────────
echo "⚠️  WARNING: This will OVERWRITE the current database."
echo "   Target: ${DATABASE_URL%%@*}@****"
echo ""
read -rp "Type 'RESTORE' to confirm: " CONFIRM
if [[ "${CONFIRM}" != "RESTORE" ]]; then
  echo "Restore cancelled."
  exit 0
fi

# ── Step 1: Download backup ───────────────────────────────────
echo ""
echo "[1/4] Downloading backup from S3..."
${STORAGE_CMD} cp \
  "s3://${BACKUP_BUCKET}/${BACKUP_TYPE}/${BACKUP_FILE}" \
  "${TMP_DIR}/${BACKUP_FILE}"
echo "      Downloaded: $(du -sh "${TMP_DIR}/${BACKUP_FILE}" | cut -f1)"

# ── Step 2: Decrypt ───────────────────────────────────────────
echo "[2/4] Decrypting backup..."
DECRYPTED_FILE="${BACKUP_FILE%.enc}"
openssl enc -d -aes-256-cbc -pbkdf2 -iter 100000 \
  -in  "${TMP_DIR}/${BACKUP_FILE}" \
  -out "${TMP_DIR}/${DECRYPTED_FILE}" \
  -pass "env:BACKUP_ENCRYPTION_KEY"
echo "      Decrypted: ${DECRYPTED_FILE}"

# ── Step 3: Restore database ──────────────────────────────────
echo "[3/4] Restoring database..."

# Extract database name from URL
DB_NAME="${DATABASE_URL##*/}"
DB_NAME="${DB_NAME%%\?*}"

# Drop and recreate database (clean slate)
echo "      Dropping existing connections..."
psql "${DATABASE_URL}" -c "
  SELECT pg_terminate_backend(pid)
  FROM pg_stat_activity
  WHERE datname = '${DB_NAME}'
    AND pid <> pg_backend_pid();" || true

echo "      Restoring from dump..."
gunzip -c "${TMP_DIR}/${DECRYPTED_FILE}" | pg_restore \
  --dbname="${DATABASE_URL}" \
  --clean \
  --if-exists \
  --no-owner \
  --no-privileges \
  --verbose \
  2>&1 | tail -20

echo "      Restore complete."

# ── Step 4: Verify ───────────────────────────────────────────
echo "[4/4] Verifying restore..."

STUDENT_COUNT="$(psql "${DATABASE_URL}" -t -c "SELECT COUNT(*) FROM students WHERE deleted_at IS NULL;" 2>/dev/null | xargs)"
USER_COUNT="$(psql "${DATABASE_URL}" -t -c "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;" 2>/dev/null | xargs)"
PAYMENT_COUNT="$(psql "${DATABASE_URL}" -t -c "SELECT COUNT(*) FROM fee_payments WHERE deleted_at IS NULL;" 2>/dev/null | xargs)"
LAST_AUDIT="$(psql "${DATABASE_URL}" -t -c "SELECT MAX(created_at) FROM audit_logs;" 2>/dev/null | xargs)"

echo ""
echo "=== Verification Results ==="
echo "  Students:     ${STUDENT_COUNT}"
echo "  Users:        ${USER_COUNT}"
echo "  Payments:     ${PAYMENT_COUNT}"
echo "  Last audit:   ${LAST_AUDIT}"
echo ""
echo "=== Restore Complete ==="
echo "Backup:    ${BACKUP_FILE}"
echo "Finished:  $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""
echo "NEXT STEPS:"
echo "  1. Restart the backend service"
echo "  2. Run 'migrate up' to apply any pending schema migrations"
echo "  3. Verify application health at /health/ready"
