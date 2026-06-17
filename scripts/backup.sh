#!/usr/bin/env bash
# =============================================================
# Institution OS — Database Backup Script
# Usage: ./scripts/backup.sh [daily|weekly|monthly]
# =============================================================
set -euo pipefail

BACKUP_TYPE="${1:-daily}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_FILE="institution_os_${BACKUP_TYPE}_${TIMESTAMP}.sql.gz"
ENCRYPTED_FILE="${BACKUP_FILE}.enc"
TMP_DIR="/tmp/ios_backup"

# ── Validate required environment variables ──────────────────
: "${DATABASE_URL:?DATABASE_URL is required}"
: "${BACKUP_ENCRYPTION_KEY:?BACKUP_ENCRYPTION_KEY is required}"
: "${BACKUP_BUCKET:?BACKUP_BUCKET is required}"

# ── AWS CLI / MinIO client configuration ─────────────────────
# Set AWS_ENDPOINT_URL for MinIO, or leave unset for AWS S3
STORAGE_CMD="aws s3"
if [[ -n "${MINIO_ENDPOINT:-}" ]]; then
  STORAGE_CMD="aws s3 --endpoint-url ${MINIO_ENDPOINT}"
fi

# ── Setup ─────────────────────────────────────────────────────
mkdir -p "${TMP_DIR}"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "=== Institution OS Backup ==="
echo "Type:      ${BACKUP_TYPE}"
echo "Timestamp: ${TIMESTAMP}"
echo "Started:   $(date -u +%Y-%m-%dT%H:%M:%SZ)"

# ── Step 1: Dump PostgreSQL ───────────────────────────────────
echo ""
echo "[1/4] Dumping PostgreSQL database..."
pg_dump \
  --format=custom \
  --compress=9 \
  --verbose \
  "${DATABASE_URL}" \
  | gzip > "${TMP_DIR}/${BACKUP_FILE}"

UNENCRYPTED_SIZE="$(du -sh "${TMP_DIR}/${BACKUP_FILE}" | cut -f1)"
echo "      Dump size: ${UNENCRYPTED_SIZE}"

# ── Step 2: Encrypt backup ────────────────────────────────────
echo "[2/4] Encrypting backup..."
openssl enc -aes-256-cbc -pbkdf2 -iter 100000 \
  -in  "${TMP_DIR}/${BACKUP_FILE}" \
  -out "${TMP_DIR}/${ENCRYPTED_FILE}" \
  -pass "env:BACKUP_ENCRYPTION_KEY"

rm "${TMP_DIR}/${BACKUP_FILE}"
echo "      Encrypted: ${ENCRYPTED_FILE}"

# ── Step 3: Upload to object storage ─────────────────────────
echo "[3/4] Uploading to ${BACKUP_BUCKET}/${BACKUP_TYPE}/..."
${STORAGE_CMD} cp \
  "${TMP_DIR}/${ENCRYPTED_FILE}" \
  "s3://${BACKUP_BUCKET}/${BACKUP_TYPE}/${ENCRYPTED_FILE}"

echo "      Upload complete."

# ── Step 4: Apply retention policy ───────────────────────────
echo "[4/4] Applying retention policy..."

case "${BACKUP_TYPE}" in
  daily)
    RETAIN_DAYS=30
    CUTOFF="$(date -d "-${RETAIN_DAYS} days" +%Y%m%d 2>/dev/null || date -v-${RETAIN_DAYS}d +%Y%m%d)"
    ;;
  weekly)
    RETAIN_DAYS=84  # 12 weeks
    CUTOFF="$(date -d "-${RETAIN_DAYS} days" +%Y%m%d 2>/dev/null || date -v-${RETAIN_DAYS}d +%Y%m%d)"
    ;;
  monthly)
    RETAIN_DAYS=730  # 24 months
    CUTOFF="$(date -d "-${RETAIN_DAYS} days" +%Y%m%d 2>/dev/null || date -v-${RETAIN_DAYS}d +%Y%m%d)"
    ;;
  *)
    echo "Unknown backup type: ${BACKUP_TYPE}"
    exit 1
    ;;
esac

echo "      Removing backups older than ${RETAIN_DAYS} days (before ${CUTOFF})..."
${STORAGE_CMD} ls "s3://${BACKUP_BUCKET}/${BACKUP_TYPE}/" \
  | awk '{print $4}' \
  | grep "institution_os_${BACKUP_TYPE}" \
  | while read -r file; do
      file_date="${file##*_}"
      file_date="${file_date%%_*}"
      # Compare dates: remove files older than cutoff
      if [[ "${file_date}" < "${CUTOFF}" ]]; then
        echo "        Deleting old backup: ${file}"
        ${STORAGE_CMD} rm "s3://${BACKUP_BUCKET}/${BACKUP_TYPE}/${file}" || true
      fi
    done

# ── Done ──────────────────────────────────────────────────────
echo ""
echo "=== Backup Complete ==="
echo "File:      ${ENCRYPTED_FILE}"
echo "Location:  s3://${BACKUP_BUCKET}/${BACKUP_TYPE}/${ENCRYPTED_FILE}"
echo "Finished:  $(date -u +%Y-%m-%dT%H:%M:%SZ)"
