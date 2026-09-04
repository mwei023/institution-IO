package finance

import (
	"time"

	"github.com/google/uuid"
)

// FeeInvoice is a student fee invoice for a term.
type FeeInvoice struct {
	ID           uuid.UUID    `json:"id"`
	InstitutionID uuid.UUID   `json:"institution_id"`
	StudentID    uuid.UUID    `json:"student_id"`
	TermID       uuid.UUID    `json:"term_id"`
	InvoiceNumber string      `json:"invoice_number"`
	TotalAmount  float64      `json:"total_amount"`
	PaidAmount   float64      `json:"paid_amount"`
	Balance      float64      `json:"balance"`
	Status       string       `json:"status"` // unpaid|partial|paid|waived|cancelled
	DueDate      *time.Time   `json:"due_date,omitempty"`
	Lines        []*InvoiceLine `json:"lines,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}

// InvoiceLine is a single line item on an invoice.
type InvoiceLine struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
}

// FeePayment is a payment receipt.
type FeePayment struct {
	ID              uuid.UUID  `json:"id"`
	InstitutionID   uuid.UUID  `json:"institution_id"`
	StudentID       uuid.UUID  `json:"student_id"`
	InvoiceID       *uuid.UUID `json:"invoice_id,omitempty"`
	ReceiptNumber   string     `json:"receipt_number"`
	Amount          float64    `json:"amount"`
	PaymentMethod   string     `json:"payment_method"` // mpesa|cash|bank_transfer|cheque|waiver
	MpesaReference  *string    `json:"mpesa_reference,omitempty"`
	BankReference   *string    `json:"bank_reference,omitempty"`
	ReceivedAt      time.Time  `json:"received_at"`
	PDFUrl          *string    `json:"pdf_url,omitempty"`
	IsRecordOnly bool `json:"is_record_only"`
	Allocations     []*PaymentAllocation `json:"allocations,omitempty"`
	UnappliedAmount float64              `json:"unapplied_amount"`
}

// StudentFeeAccount is the summary of a student's fee account.
type StudentFeeAccount struct {
	StudentID     uuid.UUID    `json:"student_id"`
	StudentName   string       `json:"student_name"`
	TotalBilled   float64      `json:"total_billed"`
	TotalPaid     float64      `json:"total_paid"`
	Balance       float64      `json:"balance"`
	LastPayment   *time.Time   `json:"last_payment,omitempty"`
	Invoices      []*FeeInvoice `json:"invoices,omitempty"`
}

// STKPushRequest is the body for initiating an M-Pesa STK push.
type STKPushRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	Amount    float64   `json:"amount" validate:"required,min=1"`
	Phone     string    `json:"phone" validate:"required"`
}

// STKPushResponse is the response after initiating STK push.
type STKPushResponse struct {
	CheckoutRequestID string `json:"checkout_request_id"`
	MerchantRequestID string `json:"merchant_request_id"`
	Message           string `json:"message"`
}

// DarajaCallbackBody mirrors the M-Pesa Daraja STK callback payload.
type DarajaCallbackBody struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        int    `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  struct {
				Item []struct {
					Name  string      `json:"Name"`
					Value interface{} `json:"Value,omitempty"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

// ManualPaymentRequest is the body to record a non-M-Pesa payment.
type ManualPaymentRequest struct {
	StudentID     uuid.UUID  `json:"student_id" validate:"required"`
	InvoiceID     *uuid.UUID `json:"invoice_id,omitempty"`
	Amount        float64    `json:"amount" validate:"required,min=1"`
	PaymentMethod string     `json:"payment_method" validate:"required,oneof=cash bank_transfer cheque waiver"`
	BankReference string     `json:"bank_reference"`
	ChequeNumber  string     `json:"cheque_number"`
	Notes         string     `json:"notes"`
	IsRecordOnly  bool       `json:"is_record_only"`
}

// FeeStructure is the fee configuration for a class/year.
type FeeStructure struct {
	ID             uuid.UUID        `json:"id"`
	InstitutionID  uuid.UUID        `json:"institution_id"`
	AcademicYearID uuid.UUID        `json:"academic_year_id"`
	ClassID        *uuid.UUID       `json:"class_id,omitempty"`
	Name           string           `json:"name"`
	IsActive       bool             `json:"is_active"`
	Components     []*FeeComponent  `json:"components,omitempty"`
}

// FeeComponent is a single fee line in a structure.
type FeeComponent struct {
	ID              uuid.UUID  `json:"id"`
	FeeStructureID  uuid.UUID  `json:"fee_structure_id"`
	Name            string     `json:"name"`
	Amount          float64    `json:"amount"`
	TermNumber      *int       `json:"term_number,omitempty"`
	IsMandatory     bool       `json:"is_mandatory"`
	DueDate         *time.Time `json:"due_date,omitempty"`
}


// MpesaTransaction is a raw Daraja API log entry.
type MpesaTransaction struct {
	ID                   uuid.UUID  `json:"id"`
	CheckoutRequestID    string     `json:"checkout_request_id"`
	MpesaReceiptNumber   *string    `json:"mpesa_receipt_number,omitempty"`
	Amount               float64    `json:"amount"`
	PhoneNumber          string     `json:"phone_number"`
	AccountReference     string     `json:"account_reference"`
	ResultCode           int        `json:"result_code"`
	ResultDescription    string     `json:"result_description"`
	Processed            bool       `json:"processed"`
	CreatedAt            time.Time  `json:"created_at"`
	StkRequestID         *uuid.UUID `json:"stk_request_id,omitempty"`
	ChannelID            *uuid.UUID `json:"channel_id,omitempty"`
}

type CreateFeeStructureRequest struct {
	AcademicYearID uuid.UUID  `json:"academic_year_id" validate:"required"`
	ClassID        *uuid.UUID `json:"class_id,omitempty"`
	Name           string     `json:"name" validate:"required"`
}
 
// AddFeeComponentRequest adds a single line item to an existing fee
// structure — e.g. "Tuition", "Boarding", "Transport" — each with its
// own amount and which term it applies to.
type AddFeeComponentRequest struct {
	Name        string     `json:"name" validate:"required"`
	Amount      float64    `json:"amount" validate:"required,gt=0"`
	TermNumber  *int       `json:"term_number,omitempty"`
	IsMandatory bool       `json:"is_mandatory"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}
 
// GenerateInvoicesRequest generates one invoice per active student in a
// class, from a fee structure's components, for a specific term.
type GenerateInvoicesRequest struct {
	FeeStructureID uuid.UUID `json:"fee_structure_id" validate:"required"`
	ClassID        uuid.UUID `json:"class_id" validate:"required"`
	TermID         uuid.UUID `json:"term_id" validate:"required"`
	TermNumber     int       `json:"term_number" validate:"required"` // used to filter which components apply
}
 
// GenerateInvoicesResult mirrors the partial-success pattern already
// established for report card generation (academics.BulkReportCardResult,
// cbc.BulkReportCardResult) — every student is processed independently;
// one student already having an invoice for this term is skipped and
// recorded, not treated as a batch failure.
type GenerateInvoicesResult struct {
	ClassID         uuid.UUID     `json:"class_id"`
	TermID          uuid.UUID     `json:"term_id"`
	TotalStudents   int           `json:"total_students"`
	InvoicesCreated int           `json:"invoices_created"`
	Skipped         []InvoiceSkip `json:"skipped"`
}
 
type InvoiceSkip struct {
	StudentID uuid.UUID `json:"student_id"`
	Reason    string    `json:"reason"`
}


// PaymentAllocation is the amount of one payment applied to one invoice.
// Previously only ever built via raw SQL inside AllocatePayment /
// CreatePaymentAllocation — never modeled as a Go type, since nothing
// needed to read one back until manual allocate/deallocate existed.
type PaymentAllocation struct {
	ID              uuid.UUID  `json:"id"`
	InstitutionID   uuid.UUID  `json:"institution_id"`
	PaymentID       uuid.UUID  `json:"payment_id"`
	InvoiceID       uuid.UUID  `json:"invoice_id"`
	AmountApplied   float64    `json:"amount_applied"`
	IsReversed      bool       `json:"is_reversed"`
	ReversedAt      *time.Time `json:"reversed_at,omitempty"`
	ReversalReason  *string    `json:"reversal_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// AllocatePaymentRequest applies an existing payment's unapplied credit to
// a specific invoice. Amount is optional — nil allocates the full
// remaining unapplied balance of the payment; a value allocates a partial
// amount, leaving the rest as credit for a later allocation.
type AllocatePaymentRequest struct {
	InvoiceID uuid.UUID `json:"invoice_id" validate:"required"`
	Amount    *float64  `json:"amount,omitempty"`
}

// DeallocateRequest reverses one allocation. Reason is optional but
// recommended — stored on the allocation row itself for the audit trail.
type DeallocateRequest struct {
	Reason string `json:"reason"`
}