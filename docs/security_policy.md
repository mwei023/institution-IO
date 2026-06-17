# Security Policy
## Institution OS
**Version**: 1.0.0

---

## 1. Authentication & Session Security

### 1.1 Password Policy
- Minimum 8 characters; at least one uppercase, lowercase, digit
- Stored as bcrypt (cost 12) — never reversible
- Account locked after 5 consecutive failures (30-minute lockout)
- Password history: cannot reuse last 5 passwords
- Forced change on first login

### 1.2 JWT Tokens
- Access token: RS256 signed, 15-minute TTL
- Refresh token: cryptographically random 32 bytes, stored in Redis + DB
- Refresh token stored in httpOnly, Secure, SameSite=Strict cookie
- Logout: blacklist access token JTI in Redis until expiry; delete refresh token
- Maximum 3 concurrent sessions per user (configurable)

### 1.3 MFA
- Phase 1: TOTP (RFC 6238) — Google Authenticator compatible
- Secret stored AES-256-GCM encrypted in database
- Backup codes: 8 single-use codes generated on MFA setup
- Admins can enforce MFA per role

---

## 2. Authorization (RBAC)

- Every API route declares required permission: `{resource}:{action}`
- Permission checked from JWT claims on every request
- Scope constraints enforced in service layer (e.g., teacher can only read their own students)
- Super Admin cannot be deleted; must always be at least 1 per institution

---

## 3. Data Encryption

| Data | Method |
|------|--------|
| Passwords | bcrypt cost 12 |
| National IDs, bank accounts | AES-256-GCM, key from env |
| M-Pesa API keys | AES-256-GCM |
| JWT signing | RS256 RSA-2048 |
| TLS | 1.2+ enforced at Nginx |
| Data at rest | Volume encryption (cloud) |

---

## 4. Transport Security

```nginx
# Nginx TLS config
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:...;
ssl_prefer_server_ciphers off;
add_header Strict-Transport-Security "max-age=63072000" always;
add_header X-Frame-Options DENY always;
add_header X-Content-Type-Options nosniff always;
add_header Referrer-Policy no-referrer-when-downgrade always;
```

---

## 5. Input Validation & Injection Prevention

- All SQL queries use parameterized statements (pgx named params)
- No string concatenation in SQL ever
- Input validated at handler layer using `go-playground/validator`
- File uploads: MIME type checked; max size 10MB; stored outside web root
- HTML sanitized before storage (rich text fields)

---

## 6. Rate Limiting

| Context | Limit |
|---------|-------|
| Unauthenticated endpoints | 20 req/min per IP |
| Login endpoint | 10 req/min per IP |
| Authenticated endpoints | 1,000 req/min per user |
| M-Pesa callback | 500 req/min (Daraja IPs only) |

---

## 7. Audit Trail

- Every INSERT, UPDATE, DELETE, VIEW (sensitive), LOGIN logged to `audit_logs`
- Audit table is append-only (PostgreSQL rules prevent UPDATE/DELETE)
- Logs retained minimum 7 years
- Searchable by admin; exportable for external audits

---

## 8. RBAC Permission Matrix (Abridged)

| Permission | SysAdmin | Super Admin | Dept Admin | Teacher | Student | Parent | Accountant |
|-----------|----------|-------------|------------|---------|---------|--------|------------|
| students:read | ✗ | ✓ | ✓(dept) | ✓(own) | ✓(self) | ✓(child) | ✗ |
| students:write | ✗ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ |
| fees:read | ✗ | ✓ | ✓(dept) | ✗ | ✓(self) | ✓(child) | ✓ |
| fees:write | ✗ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ |
| ledger:write | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✓ |
| payroll:approve | ✗ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ |
| audit_logs:read | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ |
| settings:write | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ |

---

## 9. Data Retention & GDPR/DPA

- Kenya Data Protection Act, 2019 compliance required
- Student records: retained 7 years post-graduation (legal requirement)
- Staff records: retained 7 years post-exit (statutory)
- Financial records: retained 10 years (Tax Procedures Act)
- Audit logs: retained 7 years
- Soft delete only — `deleted_at` set; data never physically removed within retention period
- Data subject rights: access, correction, portability via admin workflow
- Data breach: report to ODPC within 72 hours

---

## 10. Backup & Disaster Recovery

- Hourly WAL archive to object storage
- Daily full dump (retained 30 days)
- Weekly full dump (retained 12 weeks)
- Monthly full dump (retained 24 months)
- RTO: ≤ 4 hours | RPO: ≤ 1 hour
- Monthly restore test to staging (automated CI job)
- See `disaster_recovery.md` for playbook

---

## 11. Incident Response

1. **Detect**: monitoring alert / user report
2. **Contain**: revoke affected tokens, disable compromised accounts
3. **Assess**: audit logs review — scope and impact
4. **Notify**: institution Super Admin within 1 hour; ODPC within 72 hours if personal data affected
5. **Remediate**: patch, rotate secrets, re-deploy
6. **Review**: post-incident report within 5 business days
