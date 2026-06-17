# Technical Architecture
## Institution OS
**Version**: 1.0.0  
**Status**: Approved

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [C4 Model](#2-c4-model)
3. [Technology Stack](#3-technology-stack)
4. [Backend Architecture](#4-backend-architecture)
5. [Frontend Architecture](#5-frontend-architecture)
6. [Database Architecture](#6-database-architecture)
7. [Caching & Queuing](#7-caching--queuing)
8. [Security Architecture](#8-security-architecture)
9. [Integration Architecture](#9-integration-architecture)
10. [Deployment Architecture](#10-deployment-architecture)
11. [Observability](#11-observability)
12. [Data Flow Diagrams](#12-data-flow-diagrams)

---

## 1. Architecture Overview

Institution OS is built as a **Modular Monolith** in Phase 1. This means:

- **One deployable unit**: a single Go binary that contains all domain logic
- **Clear internal boundaries**: each domain (`student`, `finance`, `academics`, etc.) is a separate Go package with no direct cross-package dependencies — communication happens via interfaces and domain events
- **One database**: PostgreSQL with logical separation via table naming conventions and `institution_id` on every row
- **Explicit contracts**: domain packages expose only service interfaces; internal implementation details are unexported

This design gives us the development simplicity of a monolith with the conceptual clarity needed to extract microservices in Phase 3 if required.

```
┌─────────────────────────────────────────────────────────────────┐
│                        Institution OS                            │
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐  │
│  │  Next.js     │    │   Go Backend │    │   PostgreSQL     │  │
│  │  Frontend    │◄──►│   (Modular   │◄──►│   (Primary DB)   │  │
│  │  (PWA)       │    │   Monolith)  │    │                  │  │
│  └──────────────┘    └──────┬───────┘    └──────────────────┘  │
│                             │                                    │
│                    ┌────────▼────────┐                          │
│                    │     Redis       │                          │
│                    │  (Cache/Queue/  │                          │
│                    │   Sessions)     │                          │
│                    └─────────────────┘                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. C4 Model

### 2.1 Level 1 — System Context

```
[Parent]──uses──►[Institution OS]◄──uses──[Teacher]
                      │
                      │ calls API
                      ▼
              [M-Pesa Daraja API]
              [SMS Provider]
              [Email Provider]
```

**External Systems:**
- M-Pesa Daraja (payment processing)
- Africa's Talking (SMS)
- SendGrid (email)
- WhatsApp Business API (Phase 2)
- KRA (PAYE filing — Phase 2)
- KNEC (exam registration — Phase 2)

### 2.2 Level 2 — Container Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│  Institution OS Platform                                          │
│                                                                   │
│  ┌───────────────┐   HTTPS    ┌───────────────────────────────┐  │
│  │               │◄──────────►│                               │  │
│  │  Next.js 14   │            │   Go Backend API              │  │
│  │  App Router   │            │   (chi router, port 8080)     │  │
│  │  Port 3000    │            │                               │  │
│  │               │            │  ┌─────────┐ ┌─────────────┐ │  │
│  └───────────────┘            │  │  Auth   │ │  Student    │ │  │
│                               │  ├─────────┤ ├─────────────┤ │  │
│                               │  │Finance  │ │  Academics  │ │  │
│                               │  ├─────────┤ ├─────────────┤ │  │
│                               │  │  HR     │ │  Comms      │ │  │
│                               │  ├─────────┤ ├─────────────┤ │  │
│                               │  │ Audit   │ │  Workflow   │ │  │
│                               │  └─────────┘ └─────────────┘ │  │
│                               └────────────┬──────────────────┘  │
│                                            │                      │
│           ┌────────────────────────────────▼──────────────────┐  │
│           │                                                    │  │
│  ┌────────▼──────┐    ┌───────────────┐    ┌───────────────┐ │  │
│  │  PostgreSQL   │    │     Redis     │    │  File Storage │ │  │
│  │  Port 5432    │    │   Port 6379   │    │  (MinIO/S3)   │ │  │
│  └───────────────┘    └───────────────┘    └───────────────┘ │  │
│                                                               │  │
└───────────────────────────────────────────────────────────────┘
```

### 2.3 Level 3 — Component Diagram (Go Backend)

```
cmd/server/
  └── main.go                     ← HTTP server, dependency injection, middleware chain

internal/
  ├── auth/
  │     ├── handler.go            ← HTTP handlers (login, logout, refresh, reset-password)
  │     ├── service.go            ← Business logic (validate, issue tokens, MFA)
  │     ├── repository.go         ← DB queries for users, refresh tokens
  │     ├── middleware.go         ← JWT validation, RBAC enforcement
  │     └── model.go              ← User, Role, Permission structs
  │
  ├── student/
  │     ├── handler.go
  │     ├── service.go
  │     ├── repository.go
  │     └── model.go
  │
  ├── academics/
  │     ├── handler.go
  │     ├── service.go
  │     ├── repository.go
  │     └── model.go
  │
  ├── finance/
  │     ├── handler.go
  │     ├── service.go
  │     ├── repository.go
  │     ├── mpesa/                ← Daraja API client
  │     └── model.go
  │
  ├── communication/
  │     ├── handler.go
  │     ├── service.go
  │     ├── repository.go
  │     └── model.go
  │
  ├── notification/
  │     ├── service.go            ← Event subscriber, channel dispatch
  │     ├── sms_stub.go
  │     ├── email_stub.go
  │     └── push_stub.go
  │
  ├── audit/
  │     ├── middleware.go         ← HTTP request audit logging
  │     ├── service.go            ← Domain event audit logging
  │     └── repository.go
  │
  ├── workflow/
  │     ├── engine.go             ← State machine
  │     ├── handler.go
  │     └── model.go
  │
  └── digital_locker/
        ├── handler.go
        ├── service.go
        └── storage.go            ← S3/MinIO client

pkg/
  ├── db/           ← PostgreSQL connection pool (pgx)
  ├── redis/        ← Redis client wrapper
  ├── logger/       ← Structured logging (zerolog)
  ├── config/       ← Environment variable loading
  ├── validator/    ← Input validation helpers
  ├── pagination/   ← Cursor/offset pagination helpers
  └── events/       ← Domain event types and publisher interface
```

---

## 3. Technology Stack

### 3.1 Backend

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| Language | Go 1.22 | High concurrency, low memory, fast binary, simple deployment |
| HTTP Router | chi v5 | Lightweight, idiomatic, great middleware support |
| Database Driver | pgx v5 | High-performance PostgreSQL driver for Go |
| Migrations | golang-migrate | SQL-based, reversible, supports PostgreSQL |
| Redis Client | go-redis v9 | Full-featured Redis client |
| JWT | golang-jwt/jwt v5 | Standard JWT library |
| Password Hash | bcrypt (built-in) | Standard secure hashing |
| Validation | go-playground/validator | Struct tag validation |
| Logging | rs/zerolog | Structured JSON logging, zero allocation |
| Config | joho/godotenv | .env file loading |
| Testing | testify | Assertions, mocking |
| OTP/TOTP | pquerna/otp | TOTP generation and validation |
| File Upload | Built-in multipart | Standard Go |
| Object Storage | minio-go | S3-compatible client (MinIO on-prem or AWS S3) |

### 3.2 Frontend

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| Framework | Next.js 14 (App Router) | Server components, streaming, layout system |
| Language | TypeScript | Type safety, better DX |
| Styling | Tailwind CSS v3 | Utility-first, consistent design system |
| Components | shadcn/ui | Accessible, unstyled base + Radix UI primitives |
| Animation | Framer Motion v11 | Page transitions, modal entrances (used sparingly) |
| Charts | Recharts | React-native chart library, simple API |
| State | Zustand | Lightweight global state |
| Forms | React Hook Form + Zod | Performant forms with schema validation |
| HTTP Client | Axios | Interceptors for auth token refresh |
| Table | TanStack Table v8 | Headless, powerful data tables |
| Date | date-fns | Lightweight date utility |

### 3.3 Infrastructure

| Component | Technology |
|-----------|-----------|
| Containerisation | Docker + Docker Compose |
| Orchestration (Phase 2) | Kubernetes |
| CI/CD | GitHub Actions |
| Reverse Proxy | Nginx |
| SSL | Let's Encrypt (certbot) |
| Monitoring | Prometheus + Grafana |
| Logging | Loki + Promtail |
| Object Storage | MinIO (self-hosted) or AWS S3 |

---

## 4. Backend Architecture

### 4.1 Request Lifecycle

```
Client Request
      │
      ▼
[Nginx] ─── TLS termination, rate limiting, static file serving
      │
      ▼
[Go chi Router]
      │
      ├── Global Middleware Stack:
      │     ├── RequestID (inject X-Request-ID)
      │     ├── RealIP (extract client IP behind proxy)
      │     ├── Logger (log method, path, status, duration)
      │     ├── Recoverer (catch panics, return 500)
      │     ├── CORS (configured whitelist of origins)
      │     ├── RateLimit (Redis token bucket per IP)
      │     └── Compress (gzip responses > 1KB)
      │
      ├── /api/v1/auth/* ─── No auth required
      │
      └── /api/v1/* ─── Protected routes:
            ├── JWTMiddleware (parse + validate JWT)
            ├── InstitutionMiddleware (inject institution_id from JWT claim)
            ├── RBACMiddleware (check permission for this route)
            └── AuditMiddleware (log request to audit_logs after handler)
                    │
                    ▼
              Domain Handler
                    │
                    ▼
              Domain Service (business logic)
                    │
                    ▼
              Domain Repository (SQL queries via pgx)
                    │
                    ▼
              PostgreSQL
```

### 4.2 Domain Event System

Domain services publish events that the notification and audit services consume:

```go
// pkg/events/events.go
type Event struct {
    ID          string
    Type        EventType
    InstitutionID string
    ActorID     string
    EntityType  string
    EntityID    string
    Payload     interface{}
    OccurredAt  time.Time
}

type Publisher interface {
    Publish(ctx context.Context, event Event) error
}

// Events are published to Redis Streams
// Notification service consumes and dispatches
// Audit service consumes and writes immutable log
```

### 4.3 Error Handling

All API errors follow a consistent format:

```json
{
  "error": {
    "code": "STUDENT_NOT_FOUND",
    "message": "No student found with the given ID",
    "details": null,
    "request_id": "req_01HX..."
  }
}
```

**HTTP Status Code Map:**

| Situation | Status |
|-----------|--------|
| Resource not found | 404 |
| Validation error | 422 (with field errors array) |
| Unauthorized (no/invalid token) | 401 |
| Forbidden (valid token, no permission) | 403 |
| Conflict (duplicate unique field) | 409 |
| Server error | 500 |
| Service unavailable (circuit breaker open) | 503 |

### 4.4 Pagination

All list endpoints use cursor-based pagination:

```json
{
  "data": [...],
  "pagination": {
    "total": 850,
    "page": 1,
    "page_size": 50,
    "has_next": true,
    "next_cursor": "eyJpZCI6IjEyMyJ9"
  }
}
```

---

## 5. Frontend Architecture

### 5.1 App Router Structure

```
frontend/app/
  ├── (auth)/
  │     ├── login/page.tsx
  │     └── forgot-password/page.tsx
  │
  ├── (dashboard)/
  │     ├── layout.tsx           ← Protected layout with sidebar + topbar
  │     │
  │     ├── principal/
  │     │     └── page.tsx       ← Principal dashboard
  │     │
  │     ├── teacher/
  │     │     ├── page.tsx       ← Teacher dashboard
  │     │     ├── attendance/page.tsx
  │     │     ├── grades/page.tsx
  │     │     └── assignments/page.tsx
  │     │
  │     ├── parent/
  │     │     ├── page.tsx       ← Parent dashboard
  │     │     ├── fees/page.tsx
  │     │     └── messages/page.tsx
  │     │
  │     ├── student/
  │     │     └── page.tsx       ← Student dashboard
  │     │
  │     ├── students/            ← Admin: student management
  │     │     ├── page.tsx       ← Student list
  │     │     ├── [id]/page.tsx  ← Student profile
  │     │     └── new/page.tsx   ← Admission form
  │     │
  │     ├── finance/             ← Finance module
  │     ├── staff/               ← HR module
  │     ├── academics/           ← Academics module
  │     ├── communications/      ← Comms module
  │     └── settings/            ← School settings
  │
  └── portal/                    ← Public portal (unauthenticated)
        └── [school_slug]/page.tsx
```

### 5.2 State Management

```
Zustand Stores:
├── authStore       — current user, permissions, logout
├── institutionStore — institution settings, term dates
└── notificationStore — unread count, notification list

Server State (React Query / Next.js fetch):
└── All API data fetched with SWR-style caching via Next.js fetch()
```

### 5.3 Component Architecture

```
components/
  ├── ui/                 ← shadcn/ui primitives (button, input, card, etc.)
  ├── layout/
  │     ├── Sidebar.tsx
  │     ├── Topbar.tsx
  │     └── PageHeader.tsx
  ├── data-table/         ← Reusable TanStack Table wrapper
  ├── forms/              ← Reusable form components
  ├── charts/             ← Recharts wrappers
  ├── student/            ← Student-specific components
  ├── finance/            ← Finance-specific components
  └── shared/             ← Badges, avatars, status indicators
```

### 5.4 Animation Guidelines (Framer Motion)

```typescript
// Page transition — subtle fade + slight upward drift
const pageVariants = {
  initial: { opacity: 0, y: 8 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -8 }
}
const pageTransition = { duration: 0.2, ease: "easeOut" }

// Modal entrance — scale from center
const modalVariants = {
  initial: { opacity: 0, scale: 0.97 },
  animate: { opacity: 1, scale: 1 },
  exit: { opacity: 0, scale: 0.97 }
}

// PROHIBITED:
// - Bouncing (type: "spring" with high bounce)
// - Spinning charts or counters
// - Staggered list animations on data tables
// - Continuous ambient animations
```

---

## 6. Database Architecture

### 6.1 Design Principles

1. **`institution_id` on every table** — enables multi-tenancy and future sharding
2. **UUID primary keys** — globally unique, no information leakage from sequential IDs
3. **Composite indexes** — all foreign keys indexed as `(institution_id, fk_column)`
4. **Soft deletes** — `deleted_at TIMESTAMPTZ` — no hard deletes in the application layer
5. **Audit columns** — `created_at`, `updated_at`, `created_by`, `updated_by` on every table
6. **JSONB for flexible attributes** — medical details, custom fee components, workflow state
7. **No stored procedures** — all business logic in Go; SQL for data access only
8. **Parameterized queries only** — no string concatenation in SQL

### 6.2 Multi-Tenancy Strategy

```sql
-- Phase 1: Single institution, institution_id is constant
-- Phase 2: Row-level security via PostgreSQL RLS
-- Phase 3: Schema-per-tenant or database-per-tenant for large institutions

-- Every query includes institution_id in WHERE clause:
SELECT * FROM students
WHERE institution_id = $1 AND id = $2 AND deleted_at IS NULL;

-- Composite indexes for performance:
CREATE INDEX idx_students_institution_id ON students(institution_id, id);
CREATE INDEX idx_students_class_id ON students(institution_id, class_id);
```

### 6.3 Connection Pool Configuration

```go
// pkg/db/db.go
config := pgxpool.ParseConfig(connStr)
config.MaxConns = 25          // Max connections
config.MinConns = 5           // Keep warm
config.MaxConnLifetime = 1 * time.Hour
config.MaxConnIdleTime = 30 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute
```

### 6.4 Migration Strategy

**Tool**: `golang-migrate/migrate`

```bash
# Directory: backend/migrations/
# Format: {version}_{description}.up.sql / .down.sql

backend/migrations/
  ├── 000001_create_institutions.up.sql
  ├── 000001_create_institutions.down.sql
  ├── 000002_create_users.up.sql
  ├── 000002_create_users.down.sql
  ├── 000003_create_roles_permissions.up.sql
  ...

# Run migrations:
migrate -path ./migrations -database $DATABASE_URL up
```

---

## 7. Caching & Queuing

### 7.1 Redis Usage Map

| Data | Key Pattern | TTL | Purpose |
|------|------------|-----|---------|
| JWT refresh tokens | `rt:{institution_id}:{user_id}:{token_id}` | 7 days | Session management |
| Blacklisted JWTs | `bj:{jti}` | Until JWT expiry | Logout invalidation |
| Rate limit counters | `rl:{ip}` | 1 minute | Request rate limiting |
| Session data | `sess:{session_id}` | 15 minutes | Server-side session |
| Permission cache | `perm:{user_id}` | 5 minutes | Avoid DB query per request |
| Notification queue | `queue:notifications` | — | Redis Streams |
| Audit queue | `queue:audit` | — | Redis Streams |
| School settings | `settings:{institution_id}` | 1 hour | Avoid repeated DB read |
| Fee balance | `feebal:{student_id}` | 5 minutes | Dashboard perf |

### 7.2 Redis Streams (Message Queue)

```
Stream: institution_os:events

Producer: any domain service (on significant state change)
Consumers:
  - notification_consumer (group: notifications)
  - audit_consumer (group: audit)

Event Schema:
{
  "id": "01HX...",
  "type": "student.absent",
  "institution_id": "...",
  "actor_id": "...",
  "payload": {...},
  "occurred_at": "2024-01-15T08:30:00Z"
}
```

---

## 8. Security Architecture

### 8.1 Authentication Flow

```
1. POST /api/v1/auth/login
   Body: { email, password }

2. Server: verify bcrypt(password, stored_hash)
   If mismatch: increment login_attempts; return 401
   If attempts >= 5: lock account for 30 min; return 423

3. If MFA enabled:
   Return: { mfa_required: true, mfa_token: "<short-lived token>" }
   Client: prompt TOTP code
   POST /api/v1/auth/mfa/verify { mfa_token, totp_code }

4. On success:
   access_token = JWT (RS256, 15min, claims: user_id, institution_id, role, permissions[])
   refresh_token = crypto/rand 32 bytes (base64url), stored in Redis
   Response: { access_token }
   Cookie: Set-Cookie: refresh_token=...; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth/refresh

5. Token refresh:
   POST /api/v1/auth/refresh (no body; cookie sent automatically)
   Server: validate refresh_token from Redis; issue new access_token
```

### 8.2 RBAC Enforcement

```go
// Middleware checks permission from JWT claims
func RequirePermission(perm string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := ClaimsFromContext(r.Context())
            if !claims.HasPermission(perm) {
                respondForbidden(w, r)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Route registration
r.With(RequirePermission("students:write")).Post("/students", handler.CreateStudent)
r.With(RequirePermission("students:read")).Get("/students/{id}", handler.GetStudent)
```

### 8.3 Data Encryption

| Data Type | Encryption |
|-----------|-----------|
| Passwords | bcrypt (cost 12) — NOT reversible |
| National IDs | AES-256-GCM, key from environment |
| Bank account numbers | AES-256-GCM |
| M-Pesa API credentials | AES-256-GCM |
| JWT signing | RS256 (RSA 2048-bit) |
| TLS | 1.2+ (Nginx handles) |
| Data at rest | Volume-level encryption (cloud provider) |

### 8.4 Content Security Policy

```nginx
add_header Content-Security-Policy "
  default-src 'self';
  script-src 'self' 'unsafe-inline';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data: blob:;
  font-src 'self';
  connect-src 'self' https://api.safaricom.co.ke;
  frame-ancestors 'none';
" always;
```

---

## 9. Integration Architecture

### 9.1 M-Pesa Daraja Integration

```
STK Push Flow:
──────────────
Frontend → POST /api/v1/finance/fees/pay
  │  { student_id, amount, phone }
  ▼
Backend → Daraja /mpesa/stkpush/v1/processrequest
  │  { BusinessShortCode, Password, Timestamp, Amount, PartyA=phone,
  │    AccountReference=admission_number, CallBackURL }
  ▼
M-Pesa → sends STK prompt to parent's phone
  │
  │  (parent enters PIN)
  ▼
Daraja → POST {CallBackURL} /api/v1/finance/mpesa/callback
  │  { Body.stkCallback.ResultCode, MpesaReceiptNumber, Amount, ... }
  ▼
Backend:
  - Validate callback IP (Daraja IPs whitelist)
  - Verify HMAC signature
  - Idempotency check (MpesaReceiptNumber already processed?)
  - Post receipt to student fee account
  - Publish payment.received event
  - Notify parent via push + SMS
```

### 9.2 SMS Integration (Africa's Talking Stub)

```go
// Phase 1: Log to console, return success
// Phase 2: Real API call

type SMSProvider interface {
    Send(ctx context.Context, to, message string) error
}

type ConsoleSMSStub struct{}
func (s *ConsoleSMSStub) Send(ctx context.Context, to, message string) error {
    log.Info().Str("to", to).Str("message", message).Msg("SMS_STUB: would send SMS")
    return nil
}
```

### 9.3 File Storage (MinIO / S3)

```go
// All file operations go through StorageService interface
type StorageService interface {
    Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error
    PresignedDownloadURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
    Delete(ctx context.Context, bucket, key string) error
}

// Files stored at: {institution_id}/{entity_type}/{entity_id}/{filename}
// e.g.: 11111111-1111.../students/abc123.../birth_certificate.pdf
```

---

## 10. Deployment Architecture

### 10.1 Development (Docker Compose)

```yaml
# All services on local Docker network
services:
  postgres:    port 5432
  redis:       port 6379
  minio:       port 9000 (API), 9001 (console)
  backend:     port 8080 (hot reload via Air)
  frontend:    port 3000 (Next.js dev server)
```

### 10.2 Production (Single VPS — Phase 1)

```
Internet
    │
    ▼
[Nginx on VPS]
    │  TLS: Let's Encrypt
    ├── /api/* ──────────► [Go Backend :8080] ──► [PostgreSQL :5432]
    │                              │              [Redis :6379]
    └── /* ─────────────► [Next.js :3000]
                          (or Next.js export to static files)

Deployment:
- GitHub Actions builds Docker images
- Pushes to GitHub Container Registry (ghcr.io)
- SSH to VPS, pull latest images, docker-compose up -d
```

### 10.3 Kubernetes (Future — Manifests Provided)

```
Namespace: institution-os
├── Deployment: backend (3 replicas, HPA: 2–10)
├── Deployment: frontend (2 replicas)
├── StatefulSet: postgres (1 replica + standby)
├── StatefulSet: redis (1 replica)
├── Service: backend (ClusterIP)
├── Service: frontend (ClusterIP)
├── Ingress: nginx-ingress (TLS via cert-manager)
└── PersistentVolumeClaim: postgres-data, redis-data, minio-data
```

---

## 11. Observability

### 11.1 Logging

All application logs are structured JSON via zerolog:

```json
{
  "level": "info",
  "time": "2024-01-15T08:30:00Z",
  "service": "institution-os",
  "request_id": "req_01HX...",
  "institution_id": "11111111-...",
  "user_id": "abc123...",
  "method": "POST",
  "path": "/api/v1/students",
  "status": 201,
  "duration_ms": 45
}
```

Log levels:
- `debug`: development only (off in production)
- `info`: request logs, significant state changes
- `warn`: recoverable errors, rate limit hits
- `error`: unexpected errors (always include stack trace)

### 11.2 Metrics (Prometheus)

**Exposed at** `/metrics` (Prometheus scrape endpoint):

| Metric | Type | Description |
|--------|------|-------------|
| `http_requests_total` | Counter | Total HTTP requests by method, path, status |
| `http_request_duration_seconds` | Histogram | Request latency by endpoint |
| `db_query_duration_seconds` | Histogram | PostgreSQL query latency |
| `redis_operations_total` | Counter | Redis ops by command |
| `active_sessions` | Gauge | Current active user sessions |
| `notification_sent_total` | Counter | Notifications by channel and status |
| `mpesa_transactions_total` | Counter | M-Pesa transactions by result |

### 11.3 Health Checks

```
GET /health         — liveness probe (returns 200 if binary is running)
GET /health/ready   — readiness probe (checks DB + Redis connectivity)

Response:
{
  "status": "ok",
  "version": "1.0.0",
  "database": "ok",
  "redis": "ok",
  "uptime_seconds": 86400
}
```

---

## 12. Data Flow Diagrams

### 12.1 Student Admission Flow

```
Admissions Officer
      │ fills admission form
      ▼
POST /api/v1/admissions
      │
      ├── Validate input (required fields, phone format, date ranges)
      ├── Check duplicate (same name + DOB in same institution)
      ├── Create admission record (status: submitted)
      ├── Upload documents to Digital Locker
      ├── Publish: admission.created event
      └── Notify: Principal (in-app notification)

Principal Dashboard shows pending admission
      │ clicks Review → Approve
      ▼
POST /api/v1/admissions/{id}/approve
      │
      ├── Update admission status → approved
      ├── Generate admission fee invoice
      ├── Publish: admission.approved event
      └── Notify: Parent (SMS + email: "Application approved, please pay admission fee")

Parent pays admission fee via M-Pesa
      │
      ▼ (M-Pesa callback)
POST /api/v1/finance/mpesa/callback
      │
      ├── Record payment
      ├── Update admission status → awaiting_enrollment
      └── Notify: Admissions officer

Admissions Officer completes enrollment
POST /api/v1/admissions/{id}/enroll
      │
      ├── Create student record
      ├── Create parent user accounts
      ├── Assign to class
      ├── Create fee account
      ├── Create Digital Locker folder
      ├── Publish: student.enrolled event
      └── Notify: Parent (welcome SMS with portal login link)
```

### 12.2 Fee Payment Flow

```
Parent opens Parent Portal → Fees tab
GET /api/v1/parent/children/{child_id}/fee-account
  Response: { balance: 22500, due_date: "2024-02-15", invoice_items: [...] }

Parent clicks "Pay Fees"
POST /api/v1/finance/fees/pay/stk-push
  Body: { student_id, amount: 22500, phone: "+254712345678" }
  Response: { checkout_request_id: "ws_CO_...", message: "Check your phone" }

M-Pesa sends STK push → Parent enters PIN

Daraja sends callback
POST /api/v1/finance/mpesa/callback (Daraja server → Institution OS backend)
  Body: { MpesaReceiptNumber, Amount, PhoneNumber, ... }

Backend:
  1. Verify callback authenticity
  2. Find student by account reference (admission number)
  3. Create receipt record
  4. Post to general ledger (Debit Bank, Credit Fee Income)
  5. Update student fee account balance
  6. Invalidate fee balance cache in Redis
  7. Generate PDF receipt
  8. Send SMS: "Received KES 22,500 for Amara Odhiambo. Ref: QKA..."
  9. Send push notification to parent portal
  10. Update parent portal in real-time (WebSocket or long-poll)
```
