#!/usr/bin/env bash
# Institution OS — pilot Go/No-Go gate (P0-5).
# Fails loudly on any pilot blocker. Run before EVERY pilot deploy.
# Usage: ./scripts/pilot_gate.sh
set -uo pipefail
PASS=0; FAIL=0
ok()   { echo "  PASS: $1"; PASS=$((PASS+1)); }
bad()  { echo "  FAIL: $1"; FAIL=$((FAIL+1)); }

echo "=== Institution OS pilot gate ==="

echo "[1] No tracked secrets"
if git ls-files | grep -Eq "login\.json|cookies\.txt|comp_.*\.json|entries_.*\.json|fee_structure\.json|generate_.*\.json|regen_result\.json"; then
  bad "credential/seed JSON still tracked in git"
else
  ok "no credential JSON tracked"
fi
if git ls-files | grep -Eq "\.pem$|^backend/\.env$"; then
  bad "keys/.env tracked in git"
else
  ok "no keys/.env tracked"
fi

echo "[2] Migrations canonical + fresh-DB clean"
if DATABASE_URL="${DATABASE_URL:-}" ./scripts/migrate.sh validate >/dev/null 2>&1; then
  ok "migrate.sh validate"
else
  bad "migrate.sh validate (run with DATABASE_URL set for full check)"
fi

echo "[3] Backend builds + vets clean"
if (cd backend && go build ./... >/dev/null 2>&1); then ok "go build"; else bad "go build"; fi
if (cd backend && go vet ./... >/dev/null 2>&1); then ok "go vet"; else bad "go vet"; fi

echo "[4] Pilot env guards (warn-only unless PROD_GATE=1)"
warn_or_fail() {
  local msg="$1"; shift
  if [[ "${PROD_GATE:-0}" == "1" ]]; then bad "$msg"; else echo "  WARN: $msg"; fi
}
[[ -n "${MPESA_CALLBACK_SECRET:-}" ]] || warn_or_fail "MPESA_CALLBACK_SECRET unset — callback accepts without secret"
[[ "${ENVIRONMENT:-development}" != "development" ]] || warn_or_fail "ENVIRONMENT=development — cookies not Secure, sslmode likely disable"
if [[ -n "${DATABASE_URL:-}" ]] && [[ "$DATABASE_URL" == *"sslmode=disable"* ]]; then
  warn_or_fail "DATABASE_URL uses sslmode=disable — require TLS in pilot"
fi

echo "[5] DB reachable + version sane"
if [[ -n "${DATABASE_URL:-}" ]]; then
  VER="$(psql "$DATABASE_URL" -t -A -c "SELECT COALESCE(MAX(version),-1) FROM schema_migrations;" 2>/dev/null || echo ERR)"
  if [[ "$VER" == "ERR" ]]; then bad "DB unreachable"
  elif [[ "$VER" -lt 16 ]]; then bad "schema_migrations=$VER, want >=16"
  else ok "schema_migrations=$VER"; fi
  RCPT_DUPES="$(psql "$DATABASE_URL" -t -A -c "SELECT count(*) FROM (SELECT institution_id, receipt_number FROM fee_payments GROUP BY 1,2 HAVING count(*)>1) d;" 2>/dev/null || echo ERR)"
  if [[ "$RCPT_DUPES" == "0" ]]; then ok "no duplicate receipts"; else bad "duplicate receipts: $RCPT_DUPES"; fi
else
  warn_or_fail "DATABASE_URL unset — skipped live DB checks"
fi

echo ""
echo "=== result: $PASS pass, $FAIL fail ==="
[[ "$FAIL" -eq 0 ]]
