[![GitHub Workflow Status (branch)](https://img.shields.io/github/actions/workflow/status/golang-migrate/migrate/ci.yaml?branch=master)](https://github.com/golang-migrate/migrate/actions/workflows/ci.yaml?query=branch%3Amaster)
[![GoDoc](https://pkg.go.dev/badge/github.com/golang-migrate/migrate)](https://pkg.go.dev/github.com/golang-migrate/migrate/v4)
[![Coverage Status](https://img.shields.io/coveralls/github/golang-migrate/migrate/master.svg)](https://coveralls.io/github/golang-migrate/migrate?branch=master)
[![packagecloud.io](https://img.shields.io/badge/deb-packagecloud.io-844fec.svg)](https://packagecloud.io/golang-migrate/migrate?filter=debs)
[![Docker Pulls](https://img.shields.io/docker/pulls/migrate/migrate.svg)](https://hub.docker.com/r/migrate/migrate/)
![Supported Go Versions](https://img.shields.io/badge/Go-1.21%2C%201.22-lightgrey.svg)
[![GitHub Release](https://img.shields.io/github/release/golang-migrate/migrate.svg)](https://github.com/golang-migrate/migrate/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/golang-migrate/migrate/v4)](https://goreportcard.com/report/github.com/golang-migrate/migrate/v4)


# Institution OS

A multi-tenant school management platform for Kenyan primary and secondary
institutions, covering both the 8-4-4 and CBE (Competency-Based Education)
curricula. Built as a modular monolith: Go backend, Next.js frontend,
PostgreSQL.

**Status: pre-pilot.** Core student, academics, and finance workflows are
implemented and have been manually verified against real seeded data
(payments, grading, attendance, timetabling, authorization). This is **not**
yet production-hardened — see [Known Gaps](#known-gaps) before deploying
against real student data or real money.

## Stack

- **Backend:** Go, chi router, pgx (PostgreSQL driver), JWT (RS256) auth
- **Frontend:** Next.js (App Router), TypeScript, Tailwind
- **Database:** PostgreSQL, `golang-migrate`-style versioned migrations
- **Payments:** M-Pesa Daraja (STK Push), sandbox-tested only

## Architecture

Modular monolith. Each domain (`academics`, `cbc`, `finance`, `student`,
`auth`, `communication`, ...) is its own Go package under
`backend/internal/`, following handler → service → repository layering.
Multi-tenancy is enforced by `institution_id` scoping on every table and
every query — see `docs/database_schema.md` and `docs/technical_architecture.md`
for the full design.

8-4-4 and CBE academics are deliberately **separate** packages (`academics`
and `cbc`) with parallel but non-shared tables, not a single polymorphic
schema — see the migration comments in `000009_create_cbc_academic_layer`
for the reasoning.

## Getting started

```bash
# Backend
cd backend
cp .env.example .env   # fill in DB connection, JWT keys, etc.
go run cmd/server/main.go

# Frontend
cd frontend
npm install
npm run dev
```

Database migrations live in `backend/migrations/`, applied via
`golang-migrate` conventions (`<version>_<name>.up.sql` / `.down.sql`).

## Known gaps

This list is maintained by hand, not auto-generated — see project session
notes for the fuller, current version. Headline items as of this pass:

- **Test coverage is minimal.** Manual verification (curl + direct DB
  checks) has caught and fixed real bugs in payments, attendance, and
  grading authorization, but there is no automated regression suite
  protecting any of it yet.
- **HR, Boarding, Transport, and Clinic modules are schema-only** — the
  database tables exist; no handler/service/repository code or routes
  exist for any of them yet.
- **SMS and Email are architecturally wired but not connected to a real
  provider.** `ConsoleSMSProvider`/`ConsoleEmailProvider` log to console;
  swapping in a real provider (Africa's Talking, SendGrid, etc.) is a
  constructor-level change, not a rewrite, but hasn't been done.
- **M-Pesa integration is sandbox-only.** Real Daraja credentials are not
  yet in place, and `payment_channels.consumer_secret`/`passkey` are
  currently stored as plaintext — must be encrypted before any real
  credentials are entered.
- **`schema_migrations` does not reflect true database state** — needs a
  manual reconciliation pass before this is deployed anywhere new.
- **No `subject_enrollments` concept** — attendance and grading currently
  apply to every student in a class roster, not just students enrolled in
  that specific subject. Matters most for CBE senior-school electives.

See `docs/business_requirements_specification.md`,
`docs/technical_architecture.md`, and `docs/security_policy.md` for full
design documentation.
