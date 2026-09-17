// Command backfill-channels encrypts M-Pesa channel secrets at rest.
//
// Migration 000017 ships the payment_channels table with CHECK constraints
// that reject new plaintext secrets. Existing rows — and databases where
// the table pre-existed outside this repo — may still hold legacy
// plaintext passkey/consumer_key/consumer_secret values. This command
// rewrites those in place under ENCRYPTION_KEY (AES-256-GCM, "enc:v1:"
// prefix), then VALIDATEs the constraints.
//
// Usage (from backend/):
//
//	go run ./cmd/backfill-channels --dry-run    # report only, no writes
//	go run ./cmd/backfill-channels              # encrypt plaintext rows, validate
//	go run ./cmd/backfill-channels --re-encrypt # rewrite ALL secret values under current key
//
// Safety:
//   - all writes happen in a single transaction: every row rewrites or none do
//   - --dry-run makes no writes at all
//   - column names in updates come from a fixed internal map, never input
//   - exits non-zero when the table is missing (migration 000017 not applied)
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/google/uuid"

	"institution-os/pkg/config"
	"institution-os/pkg/crypto"
)

// secretCols are the columns backfill encrypts. Fixed on purpose: they are
// interpolated into UPDATE statements, so never derive them from input.
var secretCols = []string{"passkey", "consumer_key", "consumer_secret"}

func main() {
	dryRun := flag.Bool("dry-run", false, "report what would change without writing")
	reEncrypt := flag.Bool("re-encrypt", false, "rewrite ALL secret values (including already-encrypted) under the current key")
	validate := flag.Bool("validate", true, "VALIDATE the encryption CHECK constraints after rewriting")
	flag.Parse()

	cfg, err := config.Load(".env")
	if err != nil {
		fatal("config: %v", err)
	}
	key, err := crypto.ParseKey(cfg.EncryptionKey)
	if err != nil {
		fatal("%v", err)
	}

	conn, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		fatal("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	conn.SetMaxOpenConns(2)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := run(ctx, conn, key, *dryRun, *reEncrypt, *validate); err != nil {
		fatal("%v", err)
	}
}

type channel struct {
	id   uuid.UUID
	inst uuid.UUID
	name string
	// current values as stored (nil = SQL NULL)
	cur map[string]*string
	// plan holds column -> new value, only for columns to write.
	// nil pointer = write SQL NULL (normalizes legacy empty strings,
	// which are neither NULL nor ciphertext and would fail the CHECKs).
	plan map[string]*string
}

func run(ctx context.Context, conn *sql.DB, key []byte, dryRun, reEncrypt, validate bool) error {
	if err := ensureTable(ctx, conn); err != nil {
		return err
	}

	channels, plaintextCount, encryptedCount, err := loadChannels(ctx, conn)
	if err != nil {
		return err
	}

	for _, c := range channels {
		c.plan = map[string]*string{}
		for _, col := range secretCols {
			cur := c.cur[col]
			if cur == nil {
				continue // already NULL — nothing stored
			}
			if *cur == "" {
				nilVal := (*string)(nil)
				c.plan[col] = nilVal // normalize empty string to NULL
				continue
			}
			var plain string
			if crypto.IsEncrypted(*cur) {
				if !reEncrypt {
					continue // keep the existing ciphertext under the current key
				}
				p, err := crypto.Decrypt(key, *cur)
				if err != nil {
					return fmt.Errorf("decrypt channel %s (%s) %s under current key: %w", c.id, c.name, col, err)
				}
				plain = string(p)
			} else {
				plain = *cur
			}
			enc, err := crypto.Encrypt(key, []byte(plain))
			if err != nil {
				return fmt.Errorf("encrypt channel %s (%s) %s: %w", c.id, c.name, col, err)
			}
			c.plan[col] = &enc
		}
	}

	if dryRun {
		report(channels, plaintextCount, encryptedCount, true)
		return nil
	}

	if len(withPlan(channels)) > 0 {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin: %w", err)
		}
		defer func() { _ = tx.Rollback() }()

		// One UPDATE per row setting ALL secret columns together: NOT VALID
		// constraints still evaluate on every row write, so a partial update
		// that fixes one column while another still holds plaintext or ""
		// would be rejected as an intermediate state.
		for _, c := range withPlan(channels) {
			args := make([]any, 0, len(secretCols)+1)
			sets := make([]string, 0, len(secretCols)+1)
			for i, col := range secretCols {
				if newVal, ok := c.plan[col]; ok {
					args = append(args, newVal)
				} else {
					args = append(args, c.cur[col]) // unchanged: NULL or valid ciphertext
				}
				sets = append(sets, fmt.Sprintf("%s = $%d", col, i+1))
			}
			args = append(args, c.id)
			q := fmt.Sprintf("UPDATE payment_channels SET %s, updated_at = NOW() WHERE id = $%d",
				strings.Join(sets, ", "), len(secretCols)+1)
			if _, err := tx.ExecContext(ctx, q, args...); err != nil {
				return fmt.Errorf("update channel %s: %w", c.id, err)
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}

	if validate {
		for _, con := range []string{
			"chk_channels_passkey_encrypted",
			"chk_channels_consumer_key_encrypted",
			"chk_channels_consumer_secret_encrypted",
		} {
			q := fmt.Sprintf("ALTER TABLE payment_channels VALIDATE CONSTRAINT %s", con)
			if _, err := conn.ExecContext(ctx, q); err != nil {
				return fmt.Errorf("validate %s: %w (rows still plaintext — backfill them, or the constraint rejects their format)", con, err)
			}
		}
	}

	report(channels, plaintextCount, encryptedCount, false)
	fmt.Println("backfill complete")
	return nil
}

// loadChannels reads every active channel with per-row stable scan targets.
func loadChannels(ctx context.Context, conn *sql.DB) ([]*channel, int, int, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT id, institution_id, name, passkey, consumer_key, consumer_secret
		FROM payment_channels
		WHERE deleted_at IS NULL
		ORDER BY created_at`)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("query channels: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var channels []*channel
	var encrypted, plaintext int
	for rows.Next() {
		c := &channel{cur: map[string]*string{}}
		var passkey, ckey, csecret *string
		if err := rows.Scan(&c.id, &c.inst, &c.name, &passkey, &ckey, &csecret); err != nil {
			return nil, 0, 0, fmt.Errorf("scan channel: %w", err)
		}
		c.cur["passkey"] = passkey
		c.cur["consumer_key"] = ckey
		c.cur["consumer_secret"] = csecret
		for _, col := range secretCols {
			v := c.cur[col]
			if v == nil || *v == "" {
				continue
			}
			if crypto.IsEncrypted(*v) {
				encrypted++
			} else {
				plaintext++
			}
		}
		channels = append(channels, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("iterate channels: %w", err)
	}
	return channels, plaintext, encrypted, nil
}

func withPlan(channels []*channel) []*channel {
	var out []*channel
	for _, c := range channels {
		if len(c.plan) > 0 {
			out = append(out, c)
		}
	}
	return out
}

// ensureTable fails with an actionable message when payment_channels
// doesn't exist (migration 000017 not applied).
func ensureTable(ctx context.Context, conn *sql.DB) error {
	var exists bool
	err := conn.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM information_schema.tables WHERE table_name = 'payment_channels')`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check table: %w", err)
	}
	if !exists {
		return fmt.Errorf("payment_channels table missing — apply migration 000017 first (scripts/migrate.sh up)")
	}
	return nil
}

func report(channels []*channel, plaintext, alreadyEnc int, dryRun bool) {
	mode := "APPLY"
	if dryRun {
		mode = "DRY-RUN"
	}
	fmt.Printf("== channel secret backfill (%s) ==\n", mode)
	fmt.Printf("plaintext secrets to encrypt : %d\n", plaintext)
	fmt.Printf("already encrypted (kept)     : %d\n", alreadyEnc)
	pending := withPlan(channels)
	fmt.Printf("rows to write                : %d\n", len(pending))
	for _, c := range pending {
		cols := make([]string, 0, len(c.plan))
		for _, col := range secretCols {
			if _, ok := c.plan[col]; ok {
				cols = append(cols, col)
			}
		}
		sort.Strings(cols)
		fmt.Printf("  - %s (%s): %s\n", c.name, c.id, strings.Join(cols, ", "))
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "backfill-channels: "+format+"\n", args...)
	os.Exit(1)
}
