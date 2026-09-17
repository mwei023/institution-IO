package tests

// Integration tests for the finance repository's SQL — the allocation
// arithmetic that unit tests with mocks structurally cannot cover:
// oldest-invoice-first waterfall allocation, unapplied-credit tracking,
// deallocation restoring credit, and invoice status transitions.
//
// These tests run against a REAL PostgreSQL instance:
//   - CI already provides one (ci.yml service container, migrated before
//     `go test` runs) — they execute there on every push;
//   - locally they run against $DATABASE_URL (your dev DB);
//   - with no DATABASE_URL they SKIP, so nothing depends on one.
//
// Fixtures are fully isolated: every test creates its own institution
// (and thus its own students/invoices/payments under unique foreign
// keys), then cascades it away in a defer.

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"institution-os/internal/finance"
	"institution-os/pkg/db"
)

var (
	intPoolOnce sync.Once
	intPool     *pgxpool.Pool
	intPoolErr  error
)

func intDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("integration test: DATABASE_URL not set")
	}
	intPoolOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		intPool, intPoolErr = pgxpool.New(ctx, url)
		if intPoolErr == nil {
			intPoolErr = intPool.Ping(ctx)
		}
	})
	if intPoolErr != nil {
		t.Skipf("integration test: cannot reach database: %v", intPoolErr)
	}
	return intPool
}

// financeFixture holds one isolated institution graph for a single test.
type financeFixture struct {
	t           *testing.T
	ctx         context.Context
	pool        *pgxpool.Pool
	instID      uuid.UUID
	studentID   uuid.UUID
	reverserID  uuid.UUID // a user row for reversed_by FK references
	termRef     uuid.UUID
}

func newFinanceFixture(t *testing.T) *financeFixture {
	t.Helper()
	pool := intDB(t)
	ctx := context.Background()

	f := &financeFixture{t: t, ctx: ctx, pool: pool, instID: uuid.New()}

	// institution — unique school_slug per fixture
	_, err := pool.Exec(ctx, `
		INSERT INTO institutions (id, name, school_slug, settings)
		VALUES ($1, $2, $3, '{}')`,
		f.instID, "Integration Test School "+f.instID.String()[:8], "it-"+f.instID.String()[:13])
	require.NoError(t, err)

	// role + user (users.role_id is NOT NULL; role FK-scoped to institution)
	f.reverserID = uuid.New()
	roleID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO roles (id, institution_id, name, slug)
		VALUES ($1, $2, 'tester', $3)`,
		roleID, f.instID, "tester-"+f.instID.String()[:13])
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, institution_id, role_id, first_name, last_name, password_hash)
		VALUES ($1, $2, $3, 'Test', 'Reverser', 'x')`,
		f.reverserID, f.instID, roleID)
	require.NoError(t, err)

	// student (minimal NOT NULL set)
	f.studentID = uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO students (id, institution_id, admission_number, first_name, last_name, date_of_birth, gender, admission_date)
		VALUES ($1, $2, $3, 'Amina', 'Integration', '2012-03-14', 'F', '2026-01-05')`,
		f.studentID, f.instID, "ADM-"+f.instID.String()[:13])
	require.NoError(t, err)

	t.Cleanup(func() {
		// Children first (defensive — FKs mostly have no ON DELETE CASCADE),
		// then the root. Tests that fail mid-flight must not leak either.
		for _, q := range []string{
			`DELETE FROM payment_allocations WHERE institution_id = $1`,
			`DELETE FROM fee_payments WHERE institution_id = $1`,
			`DELETE FROM fee_invoices WHERE institution_id = $1`,
			`DELETE FROM fee_components WHERE institution_id = $1`,
			`DELETE FROM fee_structures WHERE institution_id = $1`,
			`DELETE FROM users WHERE id = $1`,
			`DELETE FROM roles WHERE institution_id = $1`,
			`DELETE FROM students WHERE institution_id = $1`,
			`DELETE FROM institutions WHERE id = $1`,
		} {
			_, _ = pool.Exec(context.Background(), q, f.instID)
		}
	})
	return f
}

// termID creates (or reuses, per fixture) a minimal academic year + term.
// fee_invoices.term_id is NOT NULL and FK-referenced.
func (f *financeFixture) termID() uuid.UUID {
	f.t.Helper()
	if f.termRef != uuid.Nil {
		return f.termRef
	}
	yearID := uuid.New()
	_, err := f.pool.Exec(f.ctx, `
		INSERT INTO academic_years (id, institution_id, name, start_date, end_date)
		VALUES ($1, $2, $3, '2026-01-01', '2026-12-31')`,
		yearID, f.instID, "AY-"+f.instID.String()[:13])
	require.NoError(f.t, err)

	f.termRef = uuid.New()
	_, err = f.pool.Exec(f.ctx, `
		INSERT INTO terms (id, institution_id, academic_year_id, name, term_number, start_date, end_date)
		VALUES ($1, $2, $3, 'T1', 1, '2026-01-01', '2026-04-30')`,
		f.termRef, f.instID, yearID)
	require.NoError(f.t, err)
	return f.termRef
}

// addInvoice inserts an unpaid invoice for the fixture student.
func (f *financeFixture) addInvoice(number string, total float64, dueDate *time.Time) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	_, err := f.pool.Exec(f.ctx, `
		INSERT INTO fee_invoices (id, institution_id, student_id, term_id, invoice_number, total_amount, due_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, f.instID, f.studentID, f.termID(), number, total, dueDate)
	require.NoError(f.t, err)
	return id
}

// addPayment records a real fee_payment row.
func (f *financeFixture) addPayment(receipt string, amount float64) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	_, err := f.pool.Exec(f.ctx, `
		INSERT INTO fee_payments (id, institution_id, student_id, receipt_number, amount, payment_method, received_at)
		VALUES ($1, $2, $3, $4, $5, 'cash', NOW())`,
		id, f.instID, f.studentID, receipt, amount)
	require.NoError(f.t, err)
	return id
}

// invoiceState reads back paid_amount/balance/status straight from the DB.
func (f *financeFixture) invoiceState(id uuid.UUID) (paid, balance float64, status string) {
	f.t.Helper()
	err := f.pool.QueryRow(f.ctx, `
		SELECT paid_amount::float8, balance::float8, status
		FROM fee_invoices WHERE id = $1`, id).
		Scan(&paid, &balance, &status)
	require.NoError(f.t, err)
	return
}

// allocationRow reads back one allocation with its reversal state.
func (f *financeFixture) allocationRow(id uuid.UUID) (amount float64, reversed bool) {
	f.t.Helper()
	err := f.pool.QueryRow(f.ctx, `
		SELECT amount_applied::float8, is_reversed
		FROM payment_allocations WHERE id = $1`, id).
		Scan(&amount, &reversed)
	require.NoError(f.t, err)
	return
}

func newRepo(f *financeFixture) *finance.Repository {
	// db.Pool embeds *pgxpool.Pool as an exported field — build one
	// directly around the shared test pool.
	return finance.NewRepository(&db.Pool{Pool: f.pool})
}

// ─── GetUnappliedAmount ───────────────────────────────────────────────

func TestIntegration_UnappliedAmount_TracksCredit(t *testing.T) {
	f := newFinanceFixture(t)
	repo := newRepo(f)
	ctx := f.ctx

	// Fresh payment: full amount unapplied.
	payID := f.addPayment("RCP/2026/IT001", 1000)
	got, err := repo.GetUnappliedAmount(ctx, f.instID, payID)
	require.NoError(t, err)
	assert.Equal(t, 1000.0, got)

	// Allocate 400 via the real SQL path.
	invoice := f.addInvoice("IT-INV-1", 2000, nil)
	_, err = repo.AllocateExistingPayment(ctx, f.instID, payID, invoice, 400)
	require.NoError(t, err)
	require.NoError(t, repo.UpdateInvoicePaidAmount(ctx, f.instID, invoice))

	got, err = repo.GetUnappliedAmount(ctx, f.instID, payID)
	require.NoError(t, err)
	assert.Equal(t, 600.0, got, "1000 paid - 400 allocated")

	// Unknown payment → sentinel error, not a raw pgx error.
	_, err = repo.GetUnappliedAmount(ctx, f.instID, uuid.New())
	assert.ErrorIs(t, err, finance.ErrPaymentNotFound)
}

// ─── AllocateExistingPayment + invoice status transitions ─────────────

func TestIntegration_AllocateExisting_InvoiceStatusTransitions(t *testing.T) {
	f := newFinanceFixture(t)
	repo := newRepo(f)
	ctx := f.ctx

	invoice := f.addInvoice("IT-INV-2", 1000, nil)
	payID := f.addPayment("RCP/2026/IT002", 400)

	// unpaid → partial after first allocation
	pa, err := repo.AllocateExistingPayment(ctx, f.instID, payID, invoice, 400)
	require.NoError(t, err)
	require.NoError(t, repo.UpdateInvoicePaidAmount(ctx, f.instID, invoice))

	amount, reversed := f.allocationRow(pa.ID)
	assert.Equal(t, 400.0, amount)
	assert.False(t, reversed)

	paid, balance, status := f.invoiceState(invoice)
	assert.Equal(t, 400.0, paid)
	assert.Equal(t, 600.0, balance)
	assert.Equal(t, "partial", status)

	// partial → paid once the invoice is fully covered
	pay2 := f.addPayment("RCP/2026/IT003", 600)
	_, err = repo.AllocateExistingPayment(ctx, f.instID, pay2, invoice, 600)
	require.NoError(t, err)
	require.NoError(t, repo.UpdateInvoicePaidAmount(ctx, f.instID, invoice))

	paid, balance, status = f.invoiceState(invoice)
	assert.Equal(t, 1000.0, paid)
	assert.Equal(t, 0.0, balance)
	assert.Equal(t, "paid", status)
}

// ─── DeallocatePayment: reversal restores credit + status ─────────────

func TestIntegration_DeallocatePayment_RestoresCredit(t *testing.T) {
	f := newFinanceFixture(t)
	repo := newRepo(f)
	ctx := f.ctx

	invoice := f.addInvoice("IT-INV-3", 1000, nil)
	payID := f.addPayment("RCP/2026/IT004", 700)

	pa, err := repo.AllocateExistingPayment(ctx, f.instID, payID, invoice, 700)
	require.NoError(t, err)
	require.NoError(t, repo.UpdateInvoicePaidAmount(ctx, f.instID, invoice))
	_, _, status := f.invoiceState(invoice)
	require.Equal(t, "partial", status, "700 of a 1000 invoice is partial, not paid")

	// Deallocate → invoice back to unpaid (recalc sees 0 live allocations),
	// and the money returns to the payment's unapplied credit.
	got, err := repo.DeallocatePayment(ctx, f.instID, pa.ID, f.reverserID, "wrong invoice")
	require.NoError(t, err)
	assert.True(t, got.IsReversed)
	require.NoError(t, repo.UpdateInvoicePaidAmount(ctx, f.instID, invoice))

	paid, balance, status := f.invoiceState(invoice)
	assert.Equal(t, 0.0, paid)
	assert.Equal(t, 1000.0, balance)
	assert.Equal(t, "unpaid", status)

	got2, err := repo.GetUnappliedAmount(ctx, f.instID, payID)
	require.NoError(t, err)
	assert.Equal(t, 700.0, got2, "credit returns after deallocation")

	// Re-deallocation of the same row is refused (idempotence guard).
	_, err = repo.DeallocatePayment(ctx, f.instID, pa.ID, f.reverserID, "again")
	assert.ErrorIs(t, err, finance.ErrAllocationNotFound)

	// A fresh allocation of the same credit still works after reversal.
	pa2, err := repo.AllocateExistingPayment(ctx, f.instID, payID, invoice, 250)
	require.NoError(t, err)
	amt, reversed := f.allocationRow(pa2.ID)
	assert.Equal(t, 250.0, amt)
	assert.False(t, reversed)
}

// ─── AllocatePayment: the M-Pesa oldest-first waterfall ───────────────

func TestIntegration_AllocatePayment_WaterfallAcrossInvoices(t *testing.T) {
	f := newFinanceFixture(t)
	repo := newRepo(f)
	ctx := f.ctx

	old := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	invOld := f.addInvoice("IT-INV-4A", 800, &old)
	invNew := f.addInvoice("IT-INV-4B", 500, &newer)
	payID := f.addPayment("RCP/2026/IT005", 1000)

	touched, err := repo.AllocatePayment(ctx, f.instID, f.studentID, payID, 1000)
	require.NoError(t, err)

	// Oldest due date first: 800 to invOld, then the remaining 200 to invNew.
	require.Len(t, touched, 2)
	assert.Equal(t, invOld, touched[0])
	assert.Equal(t, invNew, touched[1])

	require.NoError(t, repo.UpdateInvoicePaidAmount(ctx, f.instID, invOld))
	require.NoError(t, repo.UpdateInvoicePaidAmount(ctx, f.instID, invNew))

	paid1, balance1, status1 := f.invoiceState(invOld)
	assert.Equal(t, 800.0, paid1)
	assert.Equal(t, 0.0, balance1)
	assert.Equal(t, "paid", status1)

	paid2, balance2, status2 := f.invoiceState(invNew)
	assert.Equal(t, 200.0, paid2)
	assert.Equal(t, 300.0, balance2)
	assert.Equal(t, "partial", status2)

	// Second payment: 300 exactly clears the leftover.
	pay2 := f.addPayment("RCP/2026/IT006", 300)
	touched2, err := repo.AllocatePayment(ctx, f.instID, f.studentID, pay2, 300)
	require.NoError(t, err)
	require.Len(t, touched2, 1)
	assert.Equal(t, invNew, touched2[0], "only the outstanding invoice is touched")
	require.NoError(t, repo.UpdateInvoicePaidAmount(ctx, f.instID, invNew))

	_, _, status2b := f.invoiceState(invNew)
	assert.Equal(t, "paid", status2b)

	// Fully-paid invoices are excluded from future waterfalls.
	pay3 := f.addPayment("RCP/2026/IT007", 100)
	touched3, err := repo.AllocatePayment(ctx, f.instID, f.studentID, pay3, 100)
	require.NoError(t, err)
	assert.Empty(t, touched3, "no outstanding invoices remain")
}

// ─── Isolation + soft-delete semantics ────────────────────────────────

func TestIntegration_PaymentIsolation_BetweenInstitutions(t *testing.T) {
	f := newFinanceFixture(t)
	other := newFinanceFixture(t)
	repo := newRepo(other)
	ctx := f.ctx

	payID := f.addPayment("RCP/2026/IT008", 500)

	// Another institution must not see this payment's credit.
	_, err := repo.GetUnappliedAmount(ctx, other.instID, payID)
	assert.ErrorIs(t, err, finance.ErrPaymentNotFound)
}

func TestIntegration_UpdateInvoicePaidAmount_IgnoresDeletedPayments(t *testing.T) {
	f := newFinanceFixture(t)
	repo := newRepo(f)
	ctx := f.ctx

	invoice := f.addInvoice("IT-INV-9", 600, nil)
	payID := f.addPayment("RCP/2026/IT009", 600)

	_, err := repo.AllocateExistingPayment(ctx, f.instID, payID, invoice, 600)
	require.NoError(t, err)
	require.NoError(t, repo.UpdateInvoicePaidAmount(ctx, f.instID, invoice))
	_, _, status := f.invoiceState(invoice)
	require.Equal(t, "paid", status)

	// A payment soft-deleted afterwards must drop out of the recalc.
	_, err = f.pool.Exec(ctx, `UPDATE fee_payments SET deleted_at = NOW() WHERE id = $1`, payID)
	require.NoError(t, err)
	require.NoError(t, repo.UpdateInvoicePaidAmount(ctx, f.instID, invoice))

	paid, _, status := f.invoiceState(invoice)
	assert.Equal(t, 0.0, paid)
	assert.Equal(t, "unpaid", status)
}
