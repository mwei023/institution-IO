#!/usr/bin/env bash
# Institution OS — sequential migration runner (pilot P0-2).
# Applies backend/migrations/NNNN_*.up.sql in version order, one file at a
# time, recording each version in schema_migrations. Replaces manual psql.
#
# Usage:
#   DATABASE_URL=postgres://... ./scripts/migrate.sh up
#   DATABASE_URL=postgres://... ./scripts/migrate.sh validate  (no writes)
#   DATABASE_URL=postgres://... ./scripts/migrate.sh down-one  (last applied down)
set -euo pipefail
MIG_DIR="$(cd "$(dirname "$0")/../backend/migrations" && pwd)"
: "${DATABASE_URL:?DATABASE_URL is required}"

psql_exec() { psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -q -t -A -c "$1"; }

ensure_table() {
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -q -c \
    "CREATE TABLE IF NOT EXISTS schema_migrations (version BIGINT PRIMARY KEY, dirty BOOLEAN NOT NULL DEFAULT false, applied_at TIMESTAMPTZ NOT NULL DEFAULT now());" >/dev/null
  # Databases initialized by other tools (e.g. golang-migrate) may carry a
  # minimal table shape without applied_at; add it if missing instead of
  # failing mid-run.
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -q -c \
    "ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ NOT NULL DEFAULT now();" >/dev/null
}

# P0-2 guard: reject non-canonical filenames before touching the DB.
validate_names() {
  local bad=0
  for f in "$MIG_DIR"/*; do
    base="$(basename "$f")"
    if ! [[ "$base" =~ ^[0-9]{6}_[a-z0-9_]+\.(up|down)\.sql$ ]]; then
      echo "BAD FILENAME: $base (want NNNNNN_snake_case.up.sql)" >&2
      bad=1
    fi
  done
  # Gap check
  local prev=""
  for v in $(ls "$MIG_DIR"/*.up.sql | sed 's/.*\///;s/_.*//' | sort -n); do
    if [[ -n "$prev" && "$((10#$v))" -ne "$((10#$prev + 1))" ]]; then
      echo "VERSION GAP: $prev -> $v" >&2
      bad=1
    fi
    prev="$v"
  done
  return $bad
}

applied_version() { psql_exec "SELECT COALESCE(MAX(version),0) FROM schema_migrations WHERE dirty=false;" ; }

case "${1:-up}" in
  validate)
    validate_names && echo "migrations OK ($(ls "$MIG_DIR"/*.up.sql | wc -l) up files, no gaps)"
    ;;
  up)
    validate_names || { echo "fix filenames/gaps first" >&2; exit 1; }
    ensure_table
    cur="$(applied_version)"
    echo "current version: $cur"
    for up in $(ls "$MIG_DIR"/*.up.sql | sort); do
      v="$(basename "$up" | cut -d_ -f1)"
      v_num=$((10#$v))
      cur_num=$((10#$cur))
      if (( v_num > cur_num )); then
        echo "=== applying $v ==="
        psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f "$up"
        psql_exec "INSERT INTO schema_migrations(version,dirty) VALUES ($v_num,false) ON CONFLICT (version) DO UPDATE SET dirty=false, applied_at=now();"
        echo "applied $v"
      fi
    done
    echo "migrated to $(applied_version)"
    ;;
  down-one)
    ensure_table
    cur="$(applied_version)"
    down="$(ls "$MIG_DIR"/${cur}_*.down.sql 2>/dev/null | head -1 || true)"
    [[ -n "$down" ]] || { echo "no down file for $cur" >&2; exit 1; }
    echo "=== rolling back $cur ($down) ==="
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f "$down"
    psql_exec "DELETE FROM schema_migrations WHERE version=$((10#$cur));"
    ;;
  *) echo "usage: $0 [validate|up|down-one]" >&2; exit 1;;
esac
