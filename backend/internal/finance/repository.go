package finance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"institution-os/pkg/db"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvoiceNotFound  = errors.New("invoice not found")
	ErrPaymentNotFound  = errors.New("payment not found")
	ErrDuplicateMpesaTx = errors.New("M-Pesa transaction already processed")
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
func (r *Repository) GetStudentFeeAccount(ctx context.Context, institutionID, studentID uuid.UUID) (*StudentFeeAccount, error) {
	query := `
		SELECT
			s.id,
			s.first_name || ' ' || s.last_name AS student_name,
			COALESCE(SUM(fi.total_amount), 0) AS total_billed,
			COALESCE(SUM(fi.paid_amount), 0) AS total_paid,
			COALESCE(SUM(fi.total_amount - fi.paid_amount), 0) AS balance,
			MAX(fp.received_at) AS last_payment
		FROM students s
		LEFT JOIN fee_invoices fi ON fi.student_id = s.id AND fi.institution_id = $1 AND fi.deleted_at IS NULL
		LEFT JOIN fee_payments fp ON fp.student_id = s.id AND fp.institution_id = $1 AND fp.deleted_at IS NULL
		WHERE s.institution_id = $1 AND s.id = $2 AND s.deleted_at IS NULL
		GROUP BY s.id, s.first_name, s.last_name`

	var acct StudentFeeAccount
	err := r.db.QueryRow(ctx, query, institutionID, studentID).Scan(
		&acct.StudentID, &acct.StudentName,
		&acct.TotalBilled, &acct.TotalPaid, &acct.Balance,
		&acct.LastPayment,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("student not found")
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
		       amount, payment_method, mpesa_reference, bank_reference, received_at, pdf_url
		FROM fee_payments
		WHERE institution_id = $1 AND student_id = $2 AND deleted_at IS NULL
		ORDER BY received_at DESC`,
		institutionID, studentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []*FeePayment
	for rows.Next() {
		var p FeePayment
		if err := rows.Scan(
			&p.ID, &p.InstitutionID, &p.StudentID, &p.InvoiceID, &p.ReceiptNumber,
			&p.Amount, &p.PaymentMethod, &p.MpesaReference, &p.BankReference, &p.ReceivedAt, &p.PDFUrl,
		); err != nil {
			return nil, err
		}
		payments = append(payments, &p)
	}
	return payments, rows.Err()
}

// CreatePayment inserts a fee payment record.
func (r *Repository) CreatePayment(ctx context.Context, p *FeePayment) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO fee_payments (
			id, institution_id, student_id, invoice_id, receipt_number,
			amount, payment_method, mpesa_reference, bank_reference,
			received_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())`,
		p.ID, p.InstitutionID, p.StudentID, p.InvoiceID, p.ReceiptNumber,
		p.Amount, p.PaymentMethod, p.MpesaReference, p.BankReference, p.ReceivedAt,
	)
	return err
}

// UpdateInvoicePaidAmount recalculates and updates the paid amount on an invoice.
func (r *Repository) UpdateInvoicePaidAmount(ctx context.Context, institutionID, invoiceID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE fee_invoices fi
		SET paid_amount = COALESCE((
			SELECT SUM(fp.amount)
			FROM fee_payments fp
			WHERE fp.invoice_id = fi.id
			  AND fp.institution_id = fi.institution_id
			  AND fp.deleted_at IS NULL
			  AND fp.is_reversed = false
		), 0),
		status = CASE
			WHEN paid_amount >= total_amount THEN 'paid'
			WHEN paid_amount > 0 THEN 'partial'
			ELSE 'unpaid'
		END,
		updated_at = NOW()
		WHERE institution_id = $1 AND id = $2`,
		institutionID, invoiceID,
	)
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
func (r *Repository) StoreMpesaTransaction(ctx context.Context, institutionID uuid.UUID, tx *MpesaTransaction, rawPayload []byte) error {
	var receiptNum interface{} = tx.MpesaReceiptNumber
	_, err := r.db.Exec(ctx, `
		INSERT INTO mpesa_transactions (
			id, institution_id, checkout_request_id, mpesa_receipt_number,
			amount, phone_number, account_reference,
			result_code, result_description, raw_payload, processed, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,false,NOW())
		ON CONFLICT (institution_id, mpesa_receipt_number) DO NOTHING`,
		tx.ID, institutionID, tx.CheckoutRequestID, receiptNum,
		tx.Amount, tx.PhoneNumber, tx.AccountReference,
		tx.ResultCode, tx.ResultDescription, rawPayload,
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
			return nil, errors.New("student not found for admission number: " + admissionNumber)
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
