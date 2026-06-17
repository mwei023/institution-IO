# Testing Strategy
## Institution OS
**Version**: 1.0.0

---

## 1. Testing Philosophy

Institution OS handles financial transactions, student academic records, and sensitive personal data. Our testing strategy reflects this responsibility:

- **Trust nothing** — every business rule has a test
- **Test at the right level** — most coverage from unit tests; integration tests for database behaviour; e2e for critical user journeys only
- **Fast feedback** — unit tests run in < 30 seconds; full CI suite in < 5 minutes
- **Regression prevention** — any bug fix must include a test that would have caught it

---

## 2. Test Pyramid

```
        ┌─────────────────┐
        │   E2E Tests     │  ~10 tests  (Playwright)
        │  (slow, costly) │
        ├─────────────────┤
        │  Integration    │  ~80 tests  (real DB/Redis)
        │     Tests       │
        ├─────────────────┤
        │   Unit Tests    │  ~300 tests (mock dependencies)
        │   (fast, many)  │
        └─────────────────┘
```

---

## 3. Backend Testing (Go)

### 3.1 Unit Tests

**Location**: `backend/tests/` and co-located `*_test.go` files

**What to test:**
- Service layer business logic (all domain rules)
- Permission checking functions
- Utility functions (formatPhone, pagination, etc.)
- Error path coverage (not found, invalid input, locked account)

**Tooling:**
- `testing` (stdlib)
- `github.com/stretchr/testify` — assertions, mock, require
- No database required (all dependencies mocked)

**Coverage target**: 80%+ on all `internal/*/service.go` files

**Example patterns:**
```go
// Test naming: TestFunctionName_Scenario
func TestLogin_InvalidCredentials(t *testing.T) { ... }
func TestCreateAdmission_InvalidDateOfBirth(t *testing.T) { ... }
func TestSubmitAttendance_LockedSession(t *testing.T) { ... }
```

**Run:**
```bash
cd backend
go test ./tests/... -v -race -cover
```

### 3.2 Integration Tests

**Location**: `backend/tests/integration/`

**What to test:**
- Repository layer SQL queries against a real PostgreSQL test database
- Migration correctness (all tables exist, constraints enforced)
- Multi-step database transactions
- M-Pesa callback idempotency
- Concurrent write safety

**Tooling:**
- Real PostgreSQL (via Docker Compose test profile)
- `testcontainers-go` for isolated DB per test (optional, Phase 2)
- Test fixtures loaded before each test suite

**Setup:**
```bash
# Start test database
docker-compose -f infrastructure/docker-compose.yml up -d postgres

# Run migrations
migrate -path ./backend/migrations -database "$TEST_DATABASE_URL" up

# Load fixtures
psql "$TEST_DATABASE_URL" -f scripts/seed_test_data.sql

# Run integration tests
cd backend
go test ./tests/integration/... -tags integration -v
```

**Test database:** Separate `institution_os_test` database; never runs against production

**Critical integration test cases:**
```
TestRepository_CreateAdmission_UniqueApplicationNumber
TestRepository_MpesaCallback_Idempotency
TestRepository_AttendanceSession_ConcurrentWrites
TestRepository_FeePayment_UpdatesInvoiceBalance
TestRepository_AuditLog_CannotBeModified
TestMigrations_AllTablesExist
TestMigrations_InstitutionIDOnEveryTable
```

### 3.3 HTTP Handler Tests

**What to test:**
- Correct HTTP status codes
- Request body validation (missing fields, wrong types)
- Authorization enforcement (403 when missing permission)
- Response shape matches API spec

**Tooling:**
- `net/http/httptest` (stdlib)
- `github.com/go-chi/chi/v5` test router

**Example:**
```go
func TestHandlerCreateAdmission_MissingFirstName_Returns422(t *testing.T) {
    // Setup test router with mocked service
    // POST with missing first_name
    // Assert status 422 and error code VALIDATION_ERROR
}
```

---

## 4. Frontend Testing (TypeScript)

### 4.1 Unit Tests

**Location**: `frontend/__tests__/`

**What to test:**
- Utility functions (`formatKES`, `formatDate`, `computeAge`, `getInitials`)
- Store logic (auth state persistence, hasPermission, hasRole)
- API client interceptor logic

**Tooling:**
- `vitest` or `jest`
- No DOM required (pure function tests)

**Example:**
```typescript
describe('formatKES', () => {
  it('formats whole numbers without decimal', () => {
    expect(formatKES(22500)).toBe('KES 22,500')
  })
  it('handles zero', () => {
    expect(formatKES(0)).toBe('KES 0')
  })
})
```

### 4.2 Component Tests

**Location**: `frontend/__tests__/components/`

**What to test:**
- StatCard renders correct value and label
- StatusBadge maps status string to correct colour class
- Login form shows error on invalid credentials
- EmptyState renders with optional action button

**Tooling:**
- `@testing-library/react`
- `vitest`
- `msw` (Mock Service Worker) for API mocking

**Run:**
```bash
cd frontend
npm test
```

### 4.3 E2E Tests (Playwright)

**Location**: `frontend/e2e/`

**Critical user journeys to cover:**
```
e2e/auth.spec.ts
  - Login with valid credentials redirects to correct dashboard
  - Login with invalid credentials shows error
  - Logout clears session and redirects to login

e2e/admissions.spec.ts
  - Principal can create a new admission application
  - Principal can approve an admission
  - Accountant can see fee invoice after approval

e2e/fee-payment.spec.ts
  - Parent sees correct fee balance
  - STK push request is sent on payment initiation
  - Payment receipt appears in payment history after M-Pesa callback

e2e/attendance.spec.ts
  - Teacher opens attendance sheet for their class
  - Teacher submits attendance with present/absent mix
  - Absent student appears in class dashboard
```

**Run:**
```bash
cd frontend
npx playwright test
```

**CI:** E2E tests run against a Docker Compose stack with seeded test data. Only on `main` branch pushes (not PRs) to avoid slow CI.

---

## 5. Load Testing

**Tool**: `k6`

**Target scenarios:**
```javascript
// 500 concurrent users — typical school day load
// Scenarios:
// - 60% Teachers: attendance + grade queries
// - 25% Parents: dashboard + fee balance
// - 10% Admins: student list + reports
// - 5%  Students: timetable

export const options = {
  vus: 500,
  duration: '5m',
  thresholds: {
    http_req_duration: ['p95<200'],  // 95th percentile < 200ms
    http_req_failed:   ['rate<0.01'], // < 1% errors
  },
}
```

**Run during CI on staging** (weekly, not every commit):
```bash
k6 run infrastructure/load-tests/school-day.js
```

---

## 6. Security Testing

### 6.1 Automated (every CI run)
- `golangci-lint` with `gosec` linter — detect SQL injection, hardcoded credentials
- `npm audit` — frontend dependency vulnerability scan
- `trivy` — Docker image vulnerability scan

### 6.2 Manual (before each major release)
- SQL injection attempts on all parameterised queries
- JWT tampering (change role claim, expired token)
- RBAC bypass attempts (access other institution's data via API)
- File upload security (malicious filenames, oversized files, wrong MIME type)
- Rate limiting verification (login brute force protection)

---

## 7. CI Test Execution Order

```yaml
# GitHub Actions CI pipeline:
1. Backend lint (golangci-lint)      — ~30s
2. Backend unit tests                — ~45s
3. Backend build                     — ~60s
4. Frontend lint + type-check        — ~30s
5. Frontend unit tests               — ~20s
6. Frontend build                    — ~90s
7. Docker image build                — ~120s
8. Integration tests (DB required)   — ~90s   [main/develop only]
9. E2E tests (full stack)            — ~180s  [main only]

Total (PR):  ~5 minutes
Total (main): ~10 minutes
```

---

## 8. Test Data Management

### Seed Data
- `scripts/seed_test_data.sql` — deterministic, realistic Kenyan school data
- Always resets to known state before integration test suites
- Contains: 1 institution, 15 users, 20 students, 3 parents, fee invoices, payments

### Test Fixtures (Go)
```go
// backend/tests/fixtures/
func NewTestInstitution() uuid.UUID { return uuid.MustParse("11111111-...") }
func NewTestUser(role string) *auth.User { ... }
func NewTestStudent() *student.Student { ... }
```

### Sensitive Data in Tests
- All test passwords: `Admin1234!` (documented, never a real credential)
- Test M-Pesa references: `TESTSTK_...` prefix (sandbox environment only)
- No real phone numbers, real national IDs, or real bank details in any test file

---

## 9. Coverage Requirements

| Package | Minimum Coverage | Current |
|---------|-----------------|---------|
| `internal/auth` | 85% | ✅ Tested |
| `internal/student` | 80% | ✅ Tested |
| `internal/finance` | 80% | ✅ Tested |
| `internal/academics` | 80% | ✅ Tested |
| `internal/communication` | 70% | Pending |
| `internal/digital_locker` | 70% | Pending |
| `pkg/` | 75% | Pending |
| Frontend utils | 90% | Pending |
| Frontend components | 60% | Pending |

Generate coverage report:
```bash
cd backend
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```
