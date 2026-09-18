// Package migrate applies backend/migrations SQL in version order,
// recording each version in schema_migrations — the in-process
// equivalent of scripts/migrate.sh, so a container image can migrate
// its own database (`./server migrate up`) without psql on the host.
//
// Compatibility contract with scripts/migrate.sh (both tools operate on
// the same table and must interoperate):
//   - table shape: schema_migrations(version BIGINT PK, dirty BOOLEAN,
//     applied_at TIMESTAMPTZ) — created if absent, applied_at added if
//     missing (DBs initialized by golang-migrate carry a minimal shape);
//   - versions are the 6-digit numeric prefixes of NNNNNN_name.up.sql;
//   - a version counts as applied when recorded with dirty=false;
//   - each version runs inside a transaction that also inserts its
//     schema_migrations row, so a failed migration leaves no partial DDL
//     and no recorded version.
//
// Not every statement is transactional in Postgres (CREATE INDEX
// CONCURRENTLY, e.g.) — none are used in these migrations today; if one
// is ever added it must run outside this runner or the runner must grow
// a non-transactional escape hatch.
package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Runner applies migrations from a filesystem directory.
type Runner struct {
	pool *pgxpool.Pool
	dir  fs.FS
}

// New creates a Runner reading NNNNNN_*.up.sql from dir.
func New(pool *pgxpool.Pool, dir string) (*Runner, error) {
	if pool == nil {
		return nil, fmt.Errorf("migrate: nil pool")
	}
	return &Runner{pool: pool, dir: os.DirFS(dir)}, nil
}

// Version returns the highest non-dirty applied version, or 0.
func (r *Runner) Version(ctx context.Context) (int64, error) {
	if err := r.ensureTable(ctx); err != nil {
		return 0, err
	}
	var v int64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(version),0) FROM schema_migrations WHERE dirty=false`).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("read applied version: %w", err)
	}
	return v, nil
}

// Up applies every pending NNNNNN_*.up.sql in version order.
func (r *Runner) Up(ctx context.Context) (applied []int64, err error) {
	if err := r.ensureTable(ctx); err != nil {
		return nil, err
	}
	current, err := r.Version(ctx)
	if err != nil {
		return nil, err
	}

	type mig struct {
		version int64
		name    string
		sql     string
	}
	var migs []mig
	entries, err := fs.ReadDir(r.dir, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		v, err := versionOf(name)
		if err != nil {
			return nil, err
		}
		if v <= current {
			continue
		}
		sqlBytes, err := fs.ReadFile(r.dir, name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		migs = append(migs, mig{version: v, name: name, sql: string(sqlBytes)})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })

	for i, m := range migs {
		// Refuse gaps: applying 000018 when 000017 is missing means the
		// image and the repo have diverged — stop instead of guessing.
		if i > 0 && m.version != migs[i-1].version+1 {
			return applied, fmt.Errorf("migration gap: %d -> %d", migs[i-1].version, m.version)
		}
		if err := r.applyOne(ctx, m.version, m.name, m.sql); err != nil {
			return applied, err
		}
		applied = append(applied, m.version)
	}
	return applied, nil
}

// applyOne runs one migration plus its bookkeeping row in one transaction.
func (r *Runner) applyOne(ctx context.Context, version int64, name, script string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, script); err != nil {
		return fmt.Errorf("apply %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO schema_migrations (version, dirty) VALUES ($1, false)
		ON CONFLICT (version) DO UPDATE SET dirty=false, applied_at=now()`, version); err != nil {
		return fmt.Errorf("record %s: %w", name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit %s: %w", name, err)
	}
	return nil
}

// ensureTable creates/repairs schema_migrations to the shared shape.
func (r *Runner) ensureTable(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT PRIMARY KEY,
			dirty      BOOLEAN NOT NULL DEFAULT false,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	if _, err := r.pool.Exec(ctx,
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ NOT NULL DEFAULT now()`); err != nil {
		return fmt.Errorf("repair schema_migrations: %w", err)
	}
	return nil
}

// versionOf extracts the numeric prefix of NNNNNN_name.up.sql.
func versionOf(name string) (int64, error) {
	base := strings.TrimSuffix(name, ".up.sql")
	parts := strings.SplitN(base, "_", 2)
	v, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("migrations: bad filename %q (want NNNNNN_name.up.sql)", name)
	}
	return v, nil
}
