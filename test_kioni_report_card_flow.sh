#!/usr/bin/env bash
# Same flow as before, but uses an existing token directly instead of
# logging in — pass it as TOKEN, don't hardcode it in this file.
#
# Usage:
#   TOKEN="eyJhbGci..." ./test_kioni_report_card_flow.sh

set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080/api/v1}"
INSTITUTION_SLUG="upendo"

TOKEN="${TOKEN:?Set TOKEN=<your jwt> before running}"

CLASS_ID="ee000001-0000-0000-0000-000000000007"   # Form 3 East
TERM_ID="cc000001-0000-0000-0000-000000000003"     # Term 3
KIONI_ID="ac000001-0000-0000-0000-000000000021"

AUTH_HEADER="Authorization: Bearer $TOKEN"
SLUG_HEADER="X-Institution-Slug: $INSTITUTION_SLUG"

echo "── 1. Generating report cards for Form 3 East, Term 3 ──"
curl -s -X POST "$API_BASE/academics/report-cards/generate-class" \
  -H "$AUTH_HEADER" -H "$SLUG_HEADER" -H "Content-Type: application/json" \
  -d "{\"class_id\":\"$CLASS_ID\",\"term_id\":\"$TERM_ID\"}" | tee /tmp/generate_result.json
echo ""

echo ""
echo "── 2. Publishing Kioni's report card (idempotent — safe if already published) ──"
curl -s -X POST "$API_BASE/academics/report-cards/publish" \
  -H "$AUTH_HEADER" -H "$SLUG_HEADER" -H "Content-Type: application/json" \
  -d "{\"student_id\":\"$KIONI_ID\",\"term_id\":\"$TERM_ID\"}"
echo ""

echo ""
echo "── 3. Confirming status directly in the DB ──"
psql "${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/institution_os}" -c \
  "SELECT student_id, aggregate_marks, class_position, status, published_at
   FROM report_cards
   WHERE student_id = '$KIONI_ID' AND term_id = '$TERM_ID';" 2>/dev/null \
  || echo "(psql not available/reachable here — skip, check step 1/2 output instead)"

echo ""
echo "── 4. To see the actual parent-facing view ──"
echo "Log in as Sarah Waweru (parent.kioni@gmail.com) in the frontend and load:"
echo "  /parent/results?child=$KIONI_ID"
echo "Or get her token and run:"
echo "  curl -H \"Authorization: Bearer <sarah_token>\" -H \"$SLUG_HEADER\" \\"
echo "    $API_BASE/parent/children/$KIONI_ID/report-card"