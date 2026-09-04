package finance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"institution-os/pkg/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvoiceNotFound  = errors.New("invoice not found")
	ErrPaymentNotFound  = errors.New("payment not found")
	ErrStudentNotFound   = errors.New("student not found")
	ErrDuplicateMpesaTx = errors.New("M-Pesa transaction already processed")
	ErrFeeStructureNotFound = errors.New("fee structure not found")
	ErrAllocationNotFound = errors.New("allocation not found or already reversed")
)

// Repository handles database access for the finance domain.
type Repository struct {
	db *db.Pool
}

// NewRepository creates a new finance repository.
func NewRepository(db *db.Pool) *Repository {
	return &Repository{db: db}
}


// GetStudentFeeAccount retrieves the fee summary for a student.
//
// Uses pre-aggregated subqueries (not direct joins) for the same reason as
// ListStudentsWithOutstandingBalance: avoids row fan-out when a student has
// more than one fee_payments row.
//    unapplied_credit — real payments (not record-only, not reversed)
//    minus whatever's already been allocated to an invoice — in
//    addition to the existing invoice-level paid_amount subtraction.
//
//    unapplied_credit is computed per-payment first (a correlated
//    subquery against payment_allocations), THEN summed per student —
//    not a direct join — for the same fan-out reason documented on
//    the original query: a payment with multiple partial allocations
//    would otherwise multiply SUM(fp.amount). ────────────────────
func (r *Repository) GetStudentFeeAccount(ctx context.Context, institutionID, studentID uuid.UUID) (*StudentFeeAccount, error) {
	query := `
		SELECT
			s.id,
			s.first_name || ' ' || s.last_name AS student_name,
			COALESCE(inv.total_billed, 0) AS total_billed,
			COALESCE(inv.total_paid, 0) AS total_paid,
			COALESCE(inv.total_billed, 0) - COALESCE(inv.total_paid, 0) - COALESCE(credit.unapplied_credit, 0) AS balance,
			pay.last_payment_at
		FROM students s
		LEFT JOIN (
			SELECT student_id,
			       SUM(total_amount) AS total_billed,
			       SUM(paid_amount) AS total_paid
			FROM fee_invoices
			WHERE institution_id = $1 AND deleted_at IS NULL
			GROUP BY student_id
		) inv ON inv.student_id = s.id
		LEFT JOIN (
			SELECT student_id, MAX(received_at) AS last_payment_at
			FROM fee_payments
			WHERE institution_id = $1 AND deleted_at IS NULL
			GROUP BY student_id
		) pay ON pay.student_id = s.id
		LEFT JOIN (
			SELECT student_id, SUM(unapplied) AS unapplied_credit
			FROM (
				SELECT fp.id, fp.student_id,
				       fp.amount - COALESCE(
				           (SELECT SUM(pa.amount_applied) FROM payment_allocations pa
				            WHERE pa.payment_id = fp.id AND pa.is_reversed = false),
				           0
				       ) AS unapplied
				FROM fee_payments fp
				WHERE fp.institution_id = $1
				  AND fp.deleted_at IS NULL
				  AND fp.is_reversed = false
				  AND fp.is_record_only = false
			) per_payment
			GROUP BY student_id
		) credit ON credit.student_id = s.id
		WHERE s.institution_id = $1 AND s.id = $2 AND s.deleted_at IS NULL`
 
	var acct StudentFeeAccount
	err := r.db.QueryRow(ctx, query, institutionID, studentID).Scan(
		&acct.StudentID, &acct.StudentName,
		&acct.TotalBilled, &acct.TotalPaid, &acct.Balance,
		&acct.LastPayment,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, fmt.Errorf("get fee account: %w", err)
	}
	return &acct, nil
}

// ListInvoicesForStudent returns all invoices for a student.
func (r *Repository) ListInvoicesForStudent(ctx context.Context, institutionID, studentID uuid.UUID) ([]*FeeInvoice, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, student_id, term_id, invoice_number,
		       total_amount, paid_amount, balance, status, due_date, created_at
		FROM fee_invoices
		WHERE institution_id = $1 AND student_id = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC`,
		institutionID, studentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*FeeInvoice
	for rows.Next() {
		var inv FeeInvoice
		if err := rows.Scan(
			&inv.ID, &inv.InstitutionID, &inv.StudentID, &inv.TermID, &inv.InvoiceNumber,
			&inv.TotalAmount, &inv.PaidAmount, &inv.Balance, &inv.Status, &inv.DueDate, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		invoices = append(invoices, &inv)
	}
	return invoices, rows.Err()
}

// ListPaymentsForStudent returns all payments for a student.
func (r *Repository) ListPaymentsForStudent(ctx context.Context, institutionID, studentID uuid.UUID) ([]*FeePayment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, student_id, invoice_id, receipt_number,
		       amount, payment_method, mpesa_reference, bank_reference, received_at, pdf_url, is_record_only
		FROM fee_payments
		WHERE institution_id = $1 AND student_id = $2 AND deleted_at IS NULL
		ORDER BY received_at DESC`,
		institutionID, studentID,
	)
	if err != nil {
		return nil, err
	}
 
	payments := []*FeePayment{}
	for rows.Next() {
		var p FeePayment
		if err := rows.Scan(
			&p.ID, &p.InstitutionID, &p.StudentID, &p.InvoiceID, &p.ReceiptNumber,
			&p.Amount, &p.PaymentMethod, &p.MpesaReference, &p.BankReference, &p.ReceivedAt, &p.PDFUrl,
			&p.IsRecordOnly,
		); err != nil {
			rows.Close()
			return nil, err
		}
		payments = append(payments, &p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(payments) == 0 {
		return payments, nil
	}
 
	paymentIDs := make([]uuid.UUID, len(payments))
	byID := make(map[uuid.UUID]*FeePayment, len(payments))
	for i, p := range payments {
		paymentIDs[i] = p.ID
		p.Allocations = []*PaymentAllocation{}
		p.UnappliedAmount = p.Amount // starts as the full amount, reduced below
		byID[p.ID] = p
	}
 
	allocRows, err := r.db.Query(ctx, `
		SELECT id, institution_id, payment_id, invoice_id, amount_applied,
		       is_reversed, reversed_at, reversal_reason, created_at
		FROM payment_allocations
		WHERE institution_id = $1 AND payment_id = ANY($2)
		ORDER BY created_at`,
		institutionID, paymentIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("list payment allocations: %w", err)
	}
	defer allocRows.Close()
 
	for allocRows.Next() {
		var pa PaymentAllocation
		if err := allocRows.Scan(
			&pa.ID, &pa.InstitutionID, &pa.PaymentID, &pa.InvoiceID, &pa.AmountApplied,
			&pa.IsReversed, &pa.ReversedAt, &pa.ReversalReason, &pa.CreatedAt,
		); err != nil {
			return nil, err
		}
		p, ok := byID[pa.PaymentID]
		if !ok {
			continue
		}
		p.Allocations = append(p.Allocations, &pa)
		if !pa.IsReversed {
			p.UnappliedAmount -= pa.AmountApplied
		}
	}
	if err := allocRows.Err(); err != nil {
		return nil, err
	}
 
	return payments, nil
}

// CreatePayment inserts a fee payment record.
func (r *Repository) CreatePayment(ctx context.Context, p *FeePayment) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO fee_payments (
			id, institution_id, student_id, invoice_id, receipt_number,
			amount, payment_method, mpesa_reference, bank_reference,
			is_record_only, received_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())`,
		p.ID, p.InstitutionID, p.StudentID, p.InvoiceID, p.ReceiptNumber,
		p.Amount, p.PaymentMethod, p.MpesaReference, p.BankReference,
		p.IsRecordOnly, p.ReceivedAt,
	)
	return err
}

// UpdateInvoicePaidAmount recalculates and updates the paid amount on an invoice.
func (r *Repository) UpdateInvoicePaidAmount(ctx context.Context, institutionID, invoiceID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		WITH new_paid AS (
			SELECT COALESCE(SUM(pa.amount_applied), 0) AS amount
			FROM payment_allocations pa
			JOIN fee_payments fp ON fp.id = pa.payment_id
			WHERE pa.invoice_id = $2
			  AND pa.institution_id = $1
			  AND pa.is_reversed = false
			  AND fp.deleted_at IS NULL
			  AND fp.is_reversed = false
		)
		UPDATE fee_invoices fi
		SET paid_amount = new_paid.amount,
		    status = CASE
		        WHEN new_paid.amount >= fi.total_amount THEN 'paid'
		        WHEN new_paid.amount > 0 THEN 'partial'
		        ELSE 'unpaid'
		    END,
		    updated_at = NOW()
		FROM new_paid
		WHERE fi.institution_id = $1 AND fi.id = $2`,
		institutionID, invoiceID,
	)
	log.Debug().
		Str("invoice_id", invoiceID.String()).
		Int64("rows_affected", tag.RowsAffected()).
		Msg("invoice paid_amount recalculated")
	return err
}

// GetNextReceiptNumber generates the next sequential receipt number.
func (r *Repository) GetNextReceiptNumber(ctx context.Context, institutionID uuid.UUID) (string, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM fee_payments WHERE institution_id = $1`,
		institutionID,
	).Scan(&count)
	if err != nil {
		return "", err
	}
	year := time.Now().Year()
	return fmt.Sprintf("RCP/%d/%06d", year, count+1), nil
}

// StoreMpesaTransaction saves a raw Daraja callback.
// StoreMpesaTransaction saves a raw Daraja callback.
func (r *Repository) StoreMpesaTransaction(ctx context.Context, institutionID uuid.UUID, tx *MpesaTransaction, rawPayload []byte) error {
	var receiptNum interface{} = tx.MpesaReceiptNumber
	_, err := r.db.Exec(ctx, `
		INSERT INTO mpesa_transactions (
			id, institution_id, checkout_request_id, mpesa_receipt_number,
			amount, phone_number, account_reference,
			result_code, result_description, raw_payload, processed,
			stk_request_id, channel_id, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,false,$11,$12,NOW())
		ON CONFLICT (institution_id, mpesa_receipt_number) DO NOTHING`,
		tx.ID, institutionID, tx.CheckoutRequestID, receiptNum,
		tx.Amount, tx.PhoneNumber, tx.AccountReference,
		tx.ResultCode, tx.ResultDescription, rawPayload,
		tx.StkRequestID, tx.ChannelID,
	)
	return err
}

// GetStudentByAdmissionNumber finds a student by admission number (for M-Pesa account reference).
func (r *Repository) GetStudentByAdmissionNumber(ctx context.Context, institutionID uuid.UUID, admissionNumber string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT id FROM students WHERE institution_id = $1 AND admission_number = $2 AND deleted_at IS NULL`,
		institutionID, admissionNumber,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("student not found")
		}
		return nil, err
	}
	return &id, nil
}

// IsMpesaReceiptProcessed checks if a receipt number was already processed.
func (r *Repository) IsMpesaReceiptProcessed(ctx context.Context, institutionID uuid.UUID, receiptNumber string) (bool, error) {
	var processed bool
	err := r.db.QueryRow(ctx,
		`SELECT processed FROM mpesa_transactions WHERE institution_id = $1 AND mpesa_receipt_number = $2`,
		institutionID, receiptNumber,
	).Scan(&processed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return processed, nil
}

// MarkMpesaTransactionProcessed marks a Daraja transaction as processed.
func (r *Repository) MarkMpesaTransactionProcessed(ctx context.Context, institutionID uuid.UUID, receiptNumber string, paymentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE mpesa_transactions
		SET processed = true, payment_id = $1
		WHERE institution_id = $2 AND mpesa_receipt_number = $3`,
		paymentID, institutionID, receiptNumber,
	)
	return err
}

// GetFeeCollectionSummary returns aggregate fee collection stats.
func (r *Repository) GetFeeCollectionSummary(ctx context.Context, institutionID uuid.UUID, termID *uuid.UUID) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(DISTINCT fi.student_id) AS total_students,
			COALESCE(SUM(fi.total_amount), 0) AS total_billed,
			COALESCE(SUM(fi.paid_amount), 0) AS total_collected,
			COALESCE(SUM(fi.total_amount - fi.paid_amount), 0) AS total_outstanding,
			COUNT(CASE WHEN fi.status = 'paid' THEN 1 END) AS fully_paid_count,
			COUNT(CASE WHEN fi.status = 'partial' THEN 1 END) AS partial_count,
			COUNT(CASE WHEN fi.status = 'unpaid' THEN 1 END) AS unpaid_count
		FROM fee_invoices fi
		WHERE fi.institution_id = $1
		  AND fi.deleted_at IS NULL
		  AND ($2::uuid IS NULL OR fi.term_id = $2)`

	row := r.db.QueryRow(ctx, query, institutionID, termID)

	var summary struct {
		TotalStudents    int     `json:"total_students"`
		TotalBilled      float64 `json:"total_billed"`
		TotalCollected   float64 `json:"total_collected"`
		TotalOutstanding float64 `json:"total_outstanding"`
		FullyPaidCount   int     `json:"fully_paid_count"`
		PartialCount     int     `json:"partial_count"`
		UnpaidCount      int     `json:"unpaid_count"`
	}

	if err := row.Scan(
		&summary.TotalStudents, &summary.TotalBilled, &summary.TotalCollected,
		&summary.TotalOutstanding, &summary.FullyPaidCount, &summary.PartialCount, &summary.UnpaidCount,
	); err != nil {
		return nil, err
	}

	b, _ := json.Marshal(summary)
	var result map[string]interface{}
	_ = json.Unmarshal(b, &result)
	return result, nil
}

// StudentBalance represents one row in the outstanding-fees list.
type StudentBalance struct {
	StudentID     uuid.UUID  `json:"student_id"`
	StudentName   string     `json:"student_name"`
	AdmissionNo   string     `json:"admission_number"`
	ClassName     *string    `json:"class_name"`
	TotalBilled   float64    `json:"total_billed"`
	TotalPaid     float64    `json:"total_paid"`
	Balance       float64    `json:"balance"`
	LastPaymentAt *time.Time `json:"last_payment_at"`
}

// ListStudentsWithOutstandingBalance returns every student in the institution
// who currently owes money, ordered by largest balance first.
//
// Uses pre-aggregated subqueries (not direct joins) to avoid a fan-out bug:
// joining fee_invoices and fee_payments directly on student_id multiplies
// rows when a student has more than one payment, inflating SUM() totals.
func (r *Repository) ListStudentsWithOutstandingBalance(ctx context.Context, institutionID uuid.UUID) ([]*StudentBalance, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			s.id,
			s.first_name || ' ' || s.last_name AS student_name,
			s.admission_number,
			c.name AS class_name,
			inv.total_billed,
			inv.total_paid,
			inv.total_billed - inv.total_paid - COALESCE(credit.unapplied_credit, 0) AS balance,
			pay.last_payment_at
		FROM students s
		LEFT JOIN classes c ON c.id = s.current_class_id
		JOIN (
			SELECT student_id,
			       SUM(total_amount) AS total_billed,
			       SUM(paid_amount) AS total_paid
			FROM fee_invoices
			WHERE institution_id = $1 AND deleted_at IS NULL
			GROUP BY student_id
		) inv ON inv.student_id = s.id
		LEFT JOIN (
			SELECT student_id, MAX(received_at) AS last_payment_at
			FROM fee_payments
			WHERE institution_id = $1 AND deleted_at IS NULL
			GROUP BY student_id
		) pay ON pay.student_id = s.id
		LEFT JOIN (
			SELECT student_id, SUM(unapplied) AS unapplied_credit
			FROM (
				SELECT fp.id, fp.student_id,
				       fp.amount - COALESCE(
				           (SELECT SUM(pa.amount_applied) FROM payment_allocations pa
				            WHERE pa.payment_id = fp.id AND pa.is_reversed = false),
				           0
				       ) AS unapplied
				FROM fee_payments fp
				WHERE fp.institution_id = $1
				  AND fp.deleted_at IS NULL
				  AND fp.is_reversed = false
				  AND fp.is_record_only = false
			) per_payment
			GROUP BY student_id
		) credit ON credit.student_id = s.id
		WHERE s.institution_id = $1
		  AND s.deleted_at IS NULL
		  AND (inv.total_billed - inv.total_paid - COALESCE(credit.unapplied_credit, 0)) > 0
		ORDER BY balance DESC`,
		institutionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list outstanding balances: %w", err)
	}
	defer rows.Close()
 
	var result []*StudentBalance
	for rows.Next() {
		var b StudentBalance
		if err := rows.Scan(
			&b.StudentID, &b.StudentName, &b.AdmissionNo, &b.ClassName,
			&b.TotalBilled, &b.TotalPaid, &b.Balance, &b.LastPaymentAt,
		); err != nil {
			return nil, err
		}
		result = append(result, &b)
	}
	return result, rows.Err()
}

// AllocatePayment applies a payment amount across a student's outstanding
// invoices, oldest first, and returns the invoice IDs that received money.
// This is intentionally isolated so a configurable allocation policy can
// replace the ordering logic later without touching callers.
// AllocatePayment applies a payment across a student's outstanding invoices,
// oldest first, recording exactly how much was applied to each invoice in
// payment_allocations. Returns the invoice IDs that received money.
func (r *Repository) AllocatePayment(ctx context.Context, institutionID, studentID, paymentID uuid.UUID, amount float64) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, balance
		FROM fee_invoices
		WHERE institution_id = $1 AND student_id = $2
		  AND deleted_at IS NULL
		  AND status IN ('unpaid', 'partial')
		ORDER BY due_date ASC NULLS LAST, created_at ASC`,
		institutionID, studentID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch outstanding invoices: %w", err)
	}
	defer rows.Close()

	type invoiceBalance struct {
		id      uuid.UUID
		balance float64
	}
	var invoices []invoiceBalance
	for rows.Next() {
		var ib invoiceBalance
		if err := rows.Scan(&ib.id, &ib.balance); err != nil {
			return nil, err
		}
		invoices = append(invoices, ib)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	remaining := amount
	var touched []uuid.UUID
	for _, inv := range invoices {
		if remaining <= 0 {
			break
		}
		applied := inv.balance
		if remaining < applied {
			applied = remaining
		}
		_, err := r.db.Exec(ctx, `
			INSERT INTO payment_allocations (id, institution_id, payment_id, invoice_id, amount_applied, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())`,
			uuid.New(), institutionID, paymentID, inv.id, applied,
		)
		if err != nil {
			return touched, fmt.Errorf("insert payment allocation for invoice %s: %w", inv.id, err)
		}
		touched = append(touched, inv.id)
		remaining -= applied
	}
	return touched, nil
}

func (r *Repository) CreatePaymentAllocation(ctx context.Context, institutionID, paymentID, invoiceID uuid.UUID, amount float64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO payment_allocations (id, institution_id, payment_id, invoice_id, amount_applied, created_at)
		VALUES ($1,$2,$3,$4,$5,NOW())`,
		uuid.New(), institutionID, paymentID, invoiceID, amount,
	)
	return err
}

func (r *Repository) CreateFeeStructure(ctx context.Context, institutionID uuid.UUID, req *CreateFeeStructureRequest) (*FeeStructure, error) {
	fs := &FeeStructure{
		ID:             uuid.New(),
		InstitutionID:  institutionID,
		AcademicYearID: req.AcademicYearID,
		ClassID:        req.ClassID,
		Name:           req.Name,
		IsActive:       true,
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO fee_structures (id, institution_id, academic_year_id, class_id, name, is_active, created_at)
		VALUES ($1,$2,$3,$4,$5,true,NOW())`,
		fs.ID, fs.InstitutionID, fs.AcademicYearID, fs.ClassID, fs.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("create fee structure: %w", err)
	}
	return fs, nil
}
 
// ListFeeStructures returns active fee structures for an institution,
// optionally filtered to one academic year.
// ListFeeStructures returns active fee structures for an institution,
// optionally filtered to one academic year — each with its components
// attached.
//
// Fetches components in a single second query across every returned
// structure ID (an IN/ANY, not a query-per-structure loop) and groups
// them in Go. The original version only ever selected the base
// fee_structures columns, so Components was always nil regardless of
// what AddFeeComponent had actually inserted — AddFeeComponent itself
// was working correctly the whole time.
//
// NOTE: assumes fee_components has no deleted_at column, matching what
// AddFeeComponent's INSERT list shows (id, institution_id,
// fee_structure_id, name, amount, term_number, is_mandatory, due_date,
// created_at — no deleted_at). If fee_components does have a soft-delete
// column, add `AND deleted_at IS NULL` to the second query below.
func (r *Repository) ListFeeStructures(ctx context.Context, institutionID uuid.UUID, academicYearID *uuid.UUID) ([]*FeeStructure, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, academic_year_id, class_id, name, is_active
		FROM fee_structures
		WHERE institution_id = $1
		  AND deleted_at IS NULL
		  AND ($2::uuid IS NULL OR academic_year_id = $2)
		ORDER BY name`,
		institutionID, academicYearID,
	)
	if err != nil {
		return nil, err
	}

	structures := []*FeeStructure{}
	for rows.Next() {
		var fs FeeStructure
		if err := rows.Scan(&fs.ID, &fs.InstitutionID, &fs.AcademicYearID, &fs.ClassID, &fs.Name, &fs.IsActive); err != nil {
			rows.Close()
			return nil, err
		}
		structures = append(structures, &fs)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(structures) == 0 {
		return structures, nil
	}

	structureIDs := make([]uuid.UUID, len(structures))
	byID := make(map[uuid.UUID]*FeeStructure, len(structures))
	for i, fs := range structures {
		structureIDs[i] = fs.ID
		fs.Components = []*FeeComponent{} // default to empty, not nil, once we know the structure exists
		byID[fs.ID] = fs
	}

	compRows, err := r.db.Query(ctx, `
		SELECT id, fee_structure_id, name, amount, term_number, is_mandatory, due_date
		FROM fee_components
		WHERE institution_id = $1
		  AND fee_structure_id = ANY($2)
		ORDER BY name`,
		institutionID, structureIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("list fee components: %w", err)
	}
	defer compRows.Close()

	for compRows.Next() {
		var fc FeeComponent
		if err := compRows.Scan(&fc.ID, &fc.FeeStructureID, &fc.Name, &fc.Amount, &fc.TermNumber, &fc.IsMandatory, &fc.DueDate); err != nil {
			return nil, err
		}
		if fs, ok := byID[fc.FeeStructureID]; ok {
			fs.Components = append(fs.Components, &fc)
		}
	}
	if err := compRows.Err(); err != nil {
		return nil, err
	}

	return structures, nil
}
 
// AddFeeComponent adds a line item to an existing fee structure. Confirms
// the structure exists and belongs to this institution first — without
// this check, a component could be silently attached to another
// institution's structure if the ID were guessed or mistyped.
func (r *Repository) AddFeeComponent(ctx context.Context, institutionID, feeStructureID uuid.UUID, req *AddFeeComponentRequest) (*FeeComponent, error) {
	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM fee_structures WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL)`,
		feeStructureID, institutionID,
	).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check fee structure: %w", err)
	}
	if !exists {
		return nil, ErrFeeStructureNotFound
	}
 
	fc := &FeeComponent{
		ID:             uuid.New(),
		FeeStructureID: feeStructureID,
		Name:           req.Name,
		Amount:         req.Amount,
		TermNumber:     req.TermNumber,
		IsMandatory:    req.IsMandatory,
		DueDate:        req.DueDate,
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO fee_components (id, institution_id, fee_structure_id, name, amount, term_number, is_mandatory, due_date, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
		fc.ID, institutionID, fc.FeeStructureID, fc.Name, fc.Amount, fc.TermNumber, fc.IsMandatory, fc.DueDate,
	)
	if err != nil {
		return nil, fmt.Errorf("add fee component: %w", err)
	}
	return fc, nil
}
 
// GetFeeComponentsForStructure returns every component for a structure
// that applies to a given term number. TermNumber IS NULL on a component
// means it applies to every term (e.g. an annual boarding levy charged
// once but attributed to term 1) — matched here explicitly rather than
// left to the caller to remember.
func (r *Repository) GetFeeComponentsForStructure(ctx context.Context, institutionID, feeStructureID uuid.UUID, termNumber int) ([]*FeeComponent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, fee_structure_id, name, amount, term_number, is_mandatory, due_date
		FROM fee_components
		WHERE institution_id = $1
		  AND fee_structure_id = $2
		  AND (term_number IS NULL OR term_number = $3)
		ORDER BY name`,
		institutionID, feeStructureID, termNumber,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
 
	components := []*FeeComponent{}
	for rows.Next() {
		var fc FeeComponent
		if err := rows.Scan(&fc.ID, &fc.FeeStructureID, &fc.Name, &fc.Amount, &fc.TermNumber, &fc.IsMandatory, &fc.DueDate); err != nil {
			return nil, err
		}
		components = append(components, &fc)
	}
	return components, rows.Err()
}
 
// studentForInvoicing is a narrow projection — just enough to generate
// an invoice, not a full student record.
type studentForInvoicing struct {
	id uuid.UUID
}
 
// getActiveStudentIDsForClass is finance's own copy of the same query
// academics.GetActiveStudentIDsForClass already runs — duplicated
// deliberately rather than importing academics.Repository, matching the
// existing precedent of cbc.Service depending directly on
// *academics.Repository being the ONE cross-package repository
// dependency in the codebase; finance doing the same would be a second,
// avoidable one for a three-line query.
func (r *Repository) getActiveStudentIDsForClass(ctx context.Context, institutionID, classID uuid.UUID) ([]studentForInvoicing, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id FROM students
		WHERE institution_id = $1 AND current_class_id = $2
		  AND status = 'active' AND deleted_at IS NULL`,
		institutionID, classID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
 
	var students []studentForInvoicing
	for rows.Next() {
		var s studentForInvoicing
		if err := rows.Scan(&s.id); err != nil {
			return nil, err
		}
		students = append(students, s)
	}
	return students, rows.Err()
}
 
// InvoiceExistsForStudentTerm checks whether a student already has any
// invoice for this term — used to skip (not duplicate-bill) a student on
// a second run of invoice generation, e.g. if new students join a class
// after the first batch ran.
func (r *Repository) InvoiceExistsForStudentTerm(ctx context.Context, institutionID, studentID, termID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM fee_invoices WHERE institution_id = $1 AND student_id = $2 AND term_id = $3 AND deleted_at IS NULL)`,
		institutionID, studentID, termID,
	).Scan(&exists)
	return exists, err
}
 
// GetNextInvoiceNumber generates a sequential invoice number, same
// pattern as GetNextReceiptNumber.
func (r *Repository) GetNextInvoiceNumber(ctx context.Context, institutionID uuid.UUID) (string, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM fee_invoices WHERE institution_id = $1`,
		institutionID,
	).Scan(&count)
	if err != nil {
		return "", err
	}
	year := time.Now().Year()
	return fmt.Sprintf("INV/%d/%06d", year, count+1), nil
}
 
// CreateInvoiceWithLines creates one invoice plus its line items, in a
// transaction — an invoice with zero lines, or lines whose sum doesn't
// match the invoice total, would be a real accounting-data-integrity
// bug, so both inserts succeed together or neither does.
func (r *Repository) CreateInvoiceWithLines(ctx context.Context, institutionID, studentID, termID uuid.UUID, invoiceNumber string, components []*FeeComponent) (*FeeInvoice, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
 
	var total float64
	for _, c := range components {
		total += c.Amount
	}
 
	invoiceID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO fee_invoices (id, institution_id, student_id, term_id, invoice_number, total_amount, paid_amount, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,0,'unpaid',NOW())`,
		invoiceID, institutionID, studentID, termID, invoiceNumber, total,
	)
	if err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}
 
	for _, c := range components {
		_, err = tx.Exec(ctx, `
			INSERT INTO fee_invoice_lines (id, institution_id, invoice_id, component_id, description, amount, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,NOW())`,
			uuid.New(), institutionID, invoiceID, c.ID, c.Name, c.Amount,
		)
		if err != nil {
			return nil, fmt.Errorf("create invoice line %s: %w", c.Name, err)
		}
	}
 
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
 
	return &FeeInvoice{
		ID: invoiceID, InstitutionID: institutionID, StudentID: studentID, TermID: termID,
		InvoiceNumber: invoiceNumber, TotalAmount: total, Status: "unpaid",
	}, nil
}
 
func (r *Repository) GetUnappliedAmount(ctx context.Context, institutionID, paymentID uuid.UUID) (float64, error) {
	var amount, applied float64
	err := r.db.QueryRow(ctx, `
		SELECT fp.amount,
		       COALESCE((SELECT SUM(pa.amount_applied) FROM payment_allocations pa
		                 WHERE pa.payment_id = fp.id AND pa.is_reversed = false), 0)
		FROM fee_payments fp
		WHERE fp.id = $1 AND fp.institution_id = $2 AND fp.deleted_at IS NULL`,
		paymentID, institutionID,
	).Scan(&amount, &applied)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrPaymentNotFound
		}
		return 0, err
	}
	return amount - applied, nil
}
 
// AllocateExistingPayment applies (some or all of) an already-recorded
// payment's unapplied credit to a specific invoice, chosen explicitly by
// whoever is allocating it — unlike AllocatePayment (M-Pesa's oldest-
// invoice-first auto-allocation across possibly many invoices), this
// always targets exactly one invoice.
func (r *Repository) AllocateExistingPayment(ctx context.Context, institutionID, paymentID, invoiceID uuid.UUID, amount float64) (*PaymentAllocation, error) {
	pa := &PaymentAllocation{
		ID:            uuid.New(),
		InstitutionID: institutionID,
		PaymentID:     paymentID,
		InvoiceID:     invoiceID,
		AmountApplied: amount,
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO payment_allocations (id, institution_id, payment_id, invoice_id, amount_applied, is_reversed, created_at)
		VALUES ($1,$2,$3,$4,$5,false,NOW())`,
		pa.ID, pa.InstitutionID, pa.PaymentID, pa.InvoiceID, pa.AmountApplied,
	)
	if err != nil {
		return nil, fmt.Errorf("allocate payment: %w", err)
	}
	return pa, nil
}
 
// DeallocatePayment reverses one allocation — marks it reversed rather
// than deleting it, so the audit trail survives and the same payment can
// later be reallocated (as a fresh row) without losing the history of
// the first attempt.
func (r *Repository) DeallocatePayment(ctx context.Context, institutionID, allocationID, reversedBy uuid.UUID, reason string) (*PaymentAllocation, error) {
	var pa PaymentAllocation
	var reasonArg interface{}
	if reason != "" {
		reasonArg = reason
	}
	err := r.db.QueryRow(ctx, `
		UPDATE payment_allocations
		SET is_reversed = true, reversed_at = NOW(), reversed_by = $3, reversal_reason = $4
		WHERE id = $1 AND institution_id = $2 AND is_reversed = false
		RETURNING id, institution_id, payment_id, invoice_id, amount_applied, is_reversed, reversed_at, created_at`,
		allocationID, institutionID, reversedBy, reasonArg,
	).Scan(&pa.ID, &pa.InstitutionID, &pa.PaymentID, &pa.InvoiceID, &pa.AmountApplied, &pa.IsReversed, &pa.ReversedAt, &pa.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAllocationNotFound
		}
		return nil, fmt.Errorf("deallocate payment: %w", err)
	}
	return &pa, nil
}