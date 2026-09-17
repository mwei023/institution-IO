package finance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"institution-os/internal/finance/mpesa"
	"institution-os/pkg/events"
	"github.com/rs/zerolog/log"
)

// Service implements all finance domain business logic.
type Service struct {
	repo        Store
	mpesaClient MpesaClient
	publisher   events.Publisher
	callbackURL string
}

// MpesaClient is the subset of *mpesa.Client this service depends on,
// expressed as an interface so tests can stub Daraja without network.
type MpesaClient interface {
	InitiateSTKPush(ctx context.Context, creds mpesa.ChannelCredentials, callbackURL, phone, accountRef string, amount int64) (*mpesa.STKPushResponse, error)
}

// Store is every repository operation the finance service performs.
// Consumer-side interface (same pattern as cbc.Store): production wiring
// passes *Repository, tests pass a mock — the service was previously
// untestable against a concrete *Repository.
type Store interface {
	IsChildOfParent(ctx context.Context, institutionID, parentUserID, studentID uuid.UUID) (bool, error)
	GetStudentFeeAccount(ctx context.Context, institutionID, studentID uuid.UUID) (*StudentFeeAccount, error)
	ListInvoicesForStudent(ctx context.Context, institutionID, studentID uuid.UUID) ([]*FeeInvoice, error)
	ListPaymentsForStudent(ctx context.Context, institutionID, studentID uuid.UUID) ([]*FeePayment, error)
	GetInvoiceByID(ctx context.Context, institutionID, invoiceID uuid.UUID) (*FeeInvoice, error)

	GetDefaultChannel(ctx context.Context, institutionID uuid.UUID) (*PaymentChannel, error)
	CreateStkRequest(ctx context.Context, req *StkRequest) error
	GetStkRequestByCheckoutID(ctx context.Context, checkoutRequestID string) (*StkRequest, error)
	MarkStkRequestStatus(ctx context.Context, id uuid.UUID, status string) error
	StoreMpesaTransaction(ctx context.Context, institutionID uuid.UUID, tx *MpesaTransaction, rawPayload []byte) error
	IsMpesaReceiptProcessed(ctx context.Context, institutionID uuid.UUID, receiptNumber string) (bool, error)
	MarkMpesaTransactionProcessed(ctx context.Context, institutionID uuid.UUID, receiptNumber string, paymentID uuid.UUID) error

	GetNextReceiptNumber(ctx context.Context, institutionID uuid.UUID) (string, error)
	CreatePayment(ctx context.Context, p *FeePayment) error
	AllocatePayment(ctx context.Context, institutionID, studentID, paymentID uuid.UUID, amount float64) ([]uuid.UUID, error)
	CreatePaymentAllocation(ctx context.Context, institutionID, paymentID, invoiceID uuid.UUID, amount float64) error
	UpdateInvoicePaidAmount(ctx context.Context, institutionID, invoiceID uuid.UUID) error
	GetUnappliedAmount(ctx context.Context, institutionID, paymentID uuid.UUID) (float64, error)
	AllocateExistingPayment(ctx context.Context, institutionID, paymentID, invoiceID uuid.UUID, amount float64) (*PaymentAllocation, error)
	DeallocatePayment(ctx context.Context, institutionID, allocationID, reversedBy uuid.UUID, reason string) (*PaymentAllocation, error)

	GetFeeCollectionSummary(ctx context.Context, institutionID uuid.UUID, termID *uuid.UUID) (map[string]interface{}, error)
	ListStudentsWithOutstandingBalance(ctx context.Context, institutionID uuid.UUID) ([]*StudentBalance, error)

	CreateFeeStructure(ctx context.Context, institutionID uuid.UUID, req *CreateFeeStructureRequest) (*FeeStructure, error)
	ListFeeStructures(ctx context.Context, institutionID uuid.UUID, academicYearID *uuid.UUID) ([]*FeeStructure, error)
	AddFeeComponent(ctx context.Context, institutionID, feeStructureID uuid.UUID, req *AddFeeComponentRequest) (*FeeComponent, error)
	GetFeeComponentsForStructure(ctx context.Context, institutionID, feeStructureID uuid.UUID, termNumber int) ([]*FeeComponent, error)
	getActiveStudentIDsForClass(ctx context.Context, institutionID, classID uuid.UUID) ([]studentForInvoicing, error)
	InvoiceExistsForStudentTerm(ctx context.Context, institutionID, studentID, termID uuid.UUID) (bool, error)
	GetNextInvoiceNumber(ctx context.Context, institutionID uuid.UUID) (string, error)
	CreateInvoiceWithLines(ctx context.Context, institutionID, studentID, termID uuid.UUID, invoiceNumber string, components []*FeeComponent) (*FeeInvoice, error)
}

// NewService creates a new finance service.
func NewService(repo Store, mpesaClient MpesaClient, publisher events.Publisher, callbackURL string) *Service {
	return &Service{repo: repo, mpesaClient: mpesaClient, publisher: publisher, callbackURL: callbackURL}
}

// VerifyParentChild enforces parent→child ownership (pilot P0-4).
func (s *Service) VerifyParentChild(ctx context.Context, institutionID, parentUserID, studentID uuid.UUID) error {
	linked, err := s.repo.IsChildOfParent(ctx, institutionID, parentUserID, studentID)
	if err != nil {
		return err
	}
	if !linked {
		return fmt.Errorf("not linked to student")
	}
	return nil
}

// GetStudentFeeAccount retrieves the fee summary for a student.
func (s *Service) GetStudentFeeAccount(ctx context.Context, institutionID, studentID uuid.UUID) (*StudentFeeAccount, error) {
	acct, err := s.repo.GetStudentFeeAccount(ctx, institutionID, studentID)
	if err != nil {
		return nil, err
	}

	invoices, err := s.repo.ListInvoicesForStudent(ctx, institutionID, studentID)
	if err != nil {
		return nil, err
	}
	acct.Invoices = invoices

	return acct, nil
}

// InitiateSTKPush starts an M-Pesa STK push payment for a student.
func (s *Service) InitiateSTKPush(ctx context.Context, institutionID uuid.UUID, initiatedBy uuid.UUID, req *STKPushRequest) (*STKPushResponse, error) {
	// Verify student exists
	acct, err := s.repo.GetStudentFeeAccount(ctx, institutionID, req.StudentID)
	if err != nil {
		return nil, fmt.Errorf("student not found: %w", err)
	}

	amountInt := int64(req.Amount)
	if amountInt <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	// Resolve which till/paybill this institution pushes through.
	// This — not anything in the callback payload — is the source of truth
	// for institution attribution going forward.
	channel, err := s.repo.GetDefaultChannel(ctx, institutionID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			return nil, fmt.Errorf("institution has no active M-Pesa channel configured")
		}
		return nil, fmt.Errorf("resolve payment channel: %w", err)
	}

	creds := mpesa.ChannelCredentials{
		Shortcode: derefOr(channel.Shortcode, ""),
		Passkey:   derefOr(channel.Passkey, ""),
	}
	if channel.ConsumerKey != nil {
		creds.ConsumerKey = *channel.ConsumerKey
	}
	if channel.ConsumerSecret != nil {
		creds.ConsumerSecret = *channel.ConsumerSecret
	}

	phone := formatPhone(req.Phone)
	if len(phone) != 12 {
		return nil, fmt.Errorf("invalid phone number — want 254XXXXXXXXX")
	}
	accountRef := acct.StudentName

	stkResp, err := s.mpesaClient.InitiateSTKPush(ctx, creds, s.callbackURL, phone, accountRef, amountInt)
	if err != nil {
		return nil, fmt.Errorf("initiate STK push: %w", err)
	}

	// Record the request BEFORE returning to the caller — this is what lets
	// the callback resolve institution_id/channel_id later without trusting
	// anything Daraja sends us.
	stkReq := &StkRequest{
		ID:                 uuid.New(),
		InstitutionID:      institutionID,
		ChannelID:          channel.ID,
		StudentID:          req.StudentID,
		InitiatedBy:        initiatedBy,
		CheckoutRequestID:  stkResp.CheckoutRequestID,
		MerchantRequestID:  stkResp.MerchantRequestID,
		PhoneNumber:        phone,
		Amount:             req.Amount,
	}
	if err := s.repo.CreateStkRequest(ctx, stkReq); err != nil {
		// The push already went out to the customer's phone — we can't undo that.
		// But if we fail to record it, the callback will have nothing to resolve
		// against and the payment will silently fail to attribute. This must be loud.
		log.Error().Err(err).
			Str("checkout_request_id", stkResp.CheckoutRequestID).
			Msg("CRITICAL: failed to record stk request — incoming callback will be unattributable")
		return nil, fmt.Errorf("record stk request: %w", err)
	}

	return &STKPushResponse{
		CheckoutRequestID: stkResp.CheckoutRequestID,
		MerchantRequestID: stkResp.MerchantRequestID,
		Message:           stkResp.CustomerMessage,
	}, nil
}

// ProcessDarajaCallback handles an M-Pesa payment callback.
// institutionID is NOT a trusted input — it's resolved inside this function
// from mpesa_stk_requests, keyed by CheckoutRequestID. This is the fix for
// the pre-launch blocker: attribution now comes from a record we created
// ourselves at push time, never from parsing anything Daraja sends back.
func (s *Service) ProcessDarajaCallback(ctx context.Context, payload []byte, callback *DarajaCallbackBody) error {
	stk := callback.Body.StkCallback

	if stk.CheckoutRequestID == "" {
		return fmt.Errorf("callback missing CheckoutRequestID — cannot attribute")
	}

	stkReq, err := s.repo.GetStkRequestByCheckoutID(ctx, stk.CheckoutRequestID)
	if err != nil {
		if errors.Is(err, ErrStkRequestNotFound) {
			// A callback for a push we never initiated (or a stale/spoofed one).
			// Log loudly but return nil so Daraja doesn't retry.
			log.Warn().
				Str("checkout_request_id", stk.CheckoutRequestID).
				Msg("mpesa callback for unknown checkout_request_id — possible spoof or stale request")
			return nil
		}
		return fmt.Errorf("resolve stk request: %w", err)
	}
	institutionID := stkReq.InstitutionID

		tx := &MpesaTransaction{
		ID:                uuid.New(),
		CheckoutRequestID: stk.CheckoutRequestID,
		ResultCode:        stk.ResultCode,
		ResultDescription: stk.ResultDesc,
		StkRequestID:      &stkReq.ID,
		ChannelID:         &stkReq.ChannelID,
	}
	for _, item := range stk.CallbackMetadata.Item {
		switch item.Name {
		case "Amount":
			if v, ok := item.Value.(float64); ok {
				tx.Amount = v
			}
		case "MpesaReceiptNumber":
			if v, ok := item.Value.(string); ok {
				tx.MpesaReceiptNumber = &v
			}
		case "PhoneNumber":
			if v, ok := item.Value.(float64); ok {
				tx.PhoneNumber = fmt.Sprintf("%.0f", v)
			}
		case "AccountReference":
			if v, ok := item.Value.(string); ok {
				tx.AccountReference = v
			}
		}
	}

	if err := s.repo.StoreMpesaTransaction(ctx, institutionID, tx, payload); err != nil {
		log.Error().Err(err).Msg("store mpesa transaction")
	}

	if stk.ResultCode != 0 {
		_ = s.repo.MarkStkRequestStatus(ctx, stkReq.ID, "failed")
		log.Info().
			Int("result_code", stk.ResultCode).
			Str("result_desc", stk.ResultDesc).
			Msg("mpesa payment not successful")
		return nil
	}

	if tx.MpesaReceiptNumber == nil {
		return fmt.Errorf("missing MpesaReceiptNumber in successful callback")
	}

	processed, err := s.repo.IsMpesaReceiptProcessed(ctx, institutionID, *tx.MpesaReceiptNumber)
	if err != nil {
		return err
	}
	if processed {
		log.Info().Str("receipt", *tx.MpesaReceiptNumber).Msg("mpesa transaction already processed — skipping")
		return nil
	}

	// Student comes from OUR record of who this push was for — not from
	// parsing AccountReference. (Note: the old code looked up AccountReference
	// via GetStudentByAdmissionNumber, but AccountReference was actually the
	// student's full name, not an admission number — that lookup would never
	// have matched. This replaces it correctly.)
	studentID := stkReq.StudentID

	receiptNumber, err := s.repo.GetNextReceiptNumber(ctx, institutionID)
	if err != nil {
		return err
	}

	payment := &FeePayment{
		ID:             uuid.New(),
		InstitutionID:  institutionID,
		StudentID:      studentID,
		ReceiptNumber:  receiptNumber,
		Amount:         tx.Amount,
		PaymentMethod:  "mpesa",
		MpesaReference: tx.MpesaReceiptNumber,
		ReceivedAt:     time.Now(),
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return fmt.Errorf("create payment: %w", err)
	}

	touchedInvoices, err := s.repo.AllocatePayment(ctx, institutionID, studentID, payment.ID, tx.Amount)
	if err != nil {
		log.Error().Err(err).Msg("allocate mpesa payment")
	}
	for _, invID := range touchedInvoices {
		if err := s.repo.UpdateInvoicePaidAmount(ctx, institutionID, invID); err != nil {
			log.Error().Err(err).Str("invoice_id", invID.String()).Msg("update invoice after mpesa allocation")
		}
	}

	if err := s.repo.MarkMpesaTransactionProcessed(ctx, institutionID, *tx.MpesaReceiptNumber, payment.ID); err != nil {
		log.Error().Err(err).Msg("mark mpesa processed")
	}
	_ = s.repo.MarkStkRequestStatus(ctx, stkReq.ID, "completed")

	evt := events.New(events.EventPaymentReceived, institutionID.String())
	evt.EntityType = "fee_payment"
	evt.EntityID = payment.ID.String()
	evt.Payload = map[string]interface{}{
		"student_id":      studentID.String(),
		"amount":          tx.Amount,
		"receipt_number":  receiptNumber,
		"mpesa_reference": tx.MpesaReceiptNumber,
	}
	_ = s.publisher.Publish(evt)

	log.Info().
		Str("receipt", receiptNumber).
		Str("mpesa_ref", *tx.MpesaReceiptNumber).
		Float64("amount", tx.Amount).
		Msg("mpesa payment processed successfully")

	return nil
}

// RecordManualPayment records a cash/cheque/bank transfer payment.
func (s *Service) RecordManualPayment(ctx context.Context, institutionID, receiverID uuid.UUID, req *ManualPaymentRequest) (*FeePayment, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}
	switch req.PaymentMethod {
	case "cash", "bank_transfer", "cheque", "waiver":
	default:
		return nil, fmt.Errorf("invalid payment method")
	}
	// Verify student belongs to this institution.
	if _, err := s.repo.GetStudentFeeAccount(ctx, institutionID, req.StudentID); err != nil {
		return nil, fmt.Errorf("student not found: %w", err)
	}
	// Verify invoice (if given) belongs to this student + institution.
	if req.InvoiceID != nil {
		inv, err := s.repo.GetInvoiceByID(ctx, institutionID, *req.InvoiceID)
		if err != nil {
			return nil, fmt.Errorf("invoice not found: %w", err)
		}
		if inv.StudentID != req.StudentID {
			return nil, fmt.Errorf("invoice does not belong to student")
		}
	}
	receiptNumber, err := s.repo.GetNextReceiptNumber(ctx, institutionID)
	if err != nil {
		return nil, fmt.Errorf("generate receipt number: %w", err)
	}

	payment := &FeePayment{
		ID:            uuid.New(),
		InstitutionID: institutionID,
		StudentID:     req.StudentID,
		InvoiceID:     req.InvoiceID,
		ReceiptNumber: receiptNumber,
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
		ReceivedAt:    time.Now(),
	}

	if req.BankReference != "" {
		payment.BankReference = &req.BankReference
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	// Update invoice if linked
	if req.InvoiceID != nil {
		if err := s.repo.CreatePaymentAllocation(ctx, institutionID, payment.ID, *req.InvoiceID, payment.Amount); err != nil {
			log.Error().Err(err).Msg("create payment allocation for manual payment")
		}

		if err := s.repo.UpdateInvoicePaidAmount(ctx, institutionID, *req.InvoiceID); err != nil {
			log.Error().Err(err).Msg("update invoice paid amount")
		}
	}

	evt := events.New(events.EventPaymentReceived, institutionID.String())
	evt.EntityType = "fee_payment"
	evt.EntityID = payment.ID.String()
	_ = s.publisher.Publish(evt)

	return payment, nil
}

// GetFeeCollectionSummary returns aggregated fee collection stats.
func (s *Service) GetFeeCollectionSummary(ctx context.Context, institutionID uuid.UUID, termID *uuid.UUID) (map[string]interface{}, error) {
	return s.repo.GetFeeCollectionSummary(ctx, institutionID, termID)
}

// GetPaymentHistory returns all payments for a student.
func (s *Service) GetPaymentHistory(ctx context.Context, institutionID, studentID uuid.UUID) ([]*FeePayment, error) {
	return s.repo.ListPaymentsForStudent(ctx, institutionID, studentID)
}

// formatPhone normalises a phone number to 254XXXXXXXXX format.
// Pilot P0-3: strips spaces, dashes, parens; rejects non-digits.
func formatPhone(phone string) string {
	if len(phone) == 0 {
		return phone
	}
	clean := make([]byte, 0, len(phone))
	for i := 0; i < len(phone); i++ {
		c := phone[i]
		if c >= '0' && c <= '9' {
			clean = append(clean, c)
		} else if c == '+' && len(clean) == 0 {
			continue // leading +
		}
		// drop spaces, dashes, parens
	}
	phone = string(clean)
	if len(phone) == 10 && phone[0] == '0' {
		phone = "254" + phone[1:]
	}
	return phone
}

// derefOr returns *s if non-nil, otherwise fallback.
func derefOr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

// GetOutstandingBalances returns every student with an unpaid balance.
func (s *Service) GetOutstandingBalances(ctx context.Context, institutionID uuid.UUID) ([]*StudentBalance, error) {
	return s.repo.ListStudentsWithOutstandingBalance(ctx, institutionID)
}

// --- Payment Channels ---

// PaymentChannel represents a till/paybill/bank account an institution receives money through.
type PaymentChannel struct {
	ID                uuid.UUID
	InstitutionID     uuid.UUID
	ChannelType       string // mpesa_paybill | mpesa_till | bank_account
	Name              string
	Shortcode         *string
	Passkey           *string
	ConsumerKey       *string
	ConsumerSecret    *string
	BankName          *string
	BankAccountNumber *string
	BankAccountName   *string
	IsDefault         bool
	IsActive          bool
}

var ErrChannelNotFound = errors.New("payment channel not found")

// GetDefaultChannel returns the institution's default active M-Pesa channel.
// This is what InitiateSTKPush uses when the caller doesn't specify a channel explicitly.
func (r *Repository) GetDefaultChannel(ctx context.Context, institutionID uuid.UUID) (*PaymentChannel, error) {
	var c PaymentChannel
	err := r.db.QueryRow(ctx, `
		SELECT id, institution_id, channel_type, name, shortcode, passkey,
		       consumer_key, consumer_secret, is_default, is_active
		FROM payment_channels
		WHERE institution_id = $1
		  AND is_default = true
		  AND is_active = true
		  AND deleted_at IS NULL
		  AND channel_type IN ('mpesa_paybill','mpesa_till')`,
		institutionID,
	).Scan(
		&c.ID, &c.InstitutionID, &c.ChannelType, &c.Name, &c.Shortcode, &c.Passkey,
		&c.ConsumerKey, &c.ConsumerSecret, &c.IsDefault, &c.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChannelNotFound
		}
		return nil, fmt.Errorf("get default channel: %w", err)
	}
	return &c, nil
}

// GetChannelByShortcode resolves a channel by its Daraja BusinessShortCode.
// Not currently called by the STK flow (checkout_request_id is authoritative there) —
// kept for the future manual/bank-reconciliation path.
func (r *Repository) GetChannelByShortcode(ctx context.Context, shortcode string) (*PaymentChannel, error) {
	var c PaymentChannel
	err := r.db.QueryRow(ctx, `
		SELECT id, institution_id, channel_type, name, shortcode, passkey,
		       consumer_key, consumer_secret, is_default, is_active
		FROM payment_channels
		WHERE shortcode = $1 AND deleted_at IS NULL`,
		shortcode,
	).Scan(
		&c.ID, &c.InstitutionID, &c.ChannelType, &c.Name, &c.Shortcode, &c.Passkey,
		&c.ConsumerKey, &c.ConsumerSecret, &c.IsDefault, &c.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChannelNotFound
		}
		return nil, fmt.Errorf("get channel by shortcode: %w", err)
	}
	return &c, nil
}

// --- STK Requests ---

// StkRequest tracks an outbound STK push, keyed by Daraja's checkout_request_id.
// This is the record that lets the callback resolve institution_id without
// trusting anything in the callback payload itself.
type StkRequest struct {
	ID                uuid.UUID
	InstitutionID     uuid.UUID
	ChannelID         uuid.UUID
	StudentID         uuid.UUID
	InitiatedBy       uuid.UUID
	CheckoutRequestID string
	MerchantRequestID string
	PhoneNumber       string
	Amount            float64
	Status            string
}

var ErrStkRequestNotFound = errors.New("stk request not found")

// CreateStkRequest records an outbound STK push before it's sent to Daraja,
// so the callback has something authoritative to resolve against.
func (r *Repository) CreateStkRequest(ctx context.Context, req *StkRequest) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO mpesa_stk_requests (
			id, institution_id, channel_id, student_id, initiated_by,
			checkout_request_id, merchant_request_id, phone_number, amount, status, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'pending',NOW())`,
		req.ID, req.InstitutionID, req.ChannelID, req.StudentID, req.InitiatedBy,
		req.CheckoutRequestID, req.MerchantRequestID, req.PhoneNumber, req.Amount,
	)
	return err
}

// GetStkRequestByCheckoutID is the core lookup the callback handler uses to
// resolve which institution and channel a Daraja callback actually belongs to.
func (r *Repository) GetStkRequestByCheckoutID(ctx context.Context, checkoutRequestID string) (*StkRequest, error) {
	var req StkRequest
	err := r.db.QueryRow(ctx, `
		SELECT id, institution_id, channel_id, student_id, initiated_by,
		       checkout_request_id, merchant_request_id, phone_number, amount, status
		FROM mpesa_stk_requests
		WHERE checkout_request_id = $1`,
		checkoutRequestID,
	).Scan(
		&req.ID, &req.InstitutionID, &req.ChannelID, &req.StudentID, &req.InitiatedBy,
		&req.CheckoutRequestID, &req.MerchantRequestID, &req.PhoneNumber, &req.Amount, &req.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStkRequestNotFound
		}
		return nil, fmt.Errorf("get stk request: %w", err)
	}
	return &req, nil
}

// MarkStkRequestStatus updates an mpesa_stk_requests row once the callback resolves it.
func (r *Repository) MarkStkRequestStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE mpesa_stk_requests SET status = $1 WHERE id = $2`,
		status, id,
	)
	return err
}

func (s *Service) CreateFeeStructure(ctx context.Context, institutionID uuid.UUID, req *CreateFeeStructureRequest) (*FeeStructure, error) {
	return s.repo.CreateFeeStructure(ctx, institutionID, req)
}
 
func (s *Service) ListFeeStructures(ctx context.Context, institutionID uuid.UUID, academicYearID *uuid.UUID) ([]*FeeStructure, error) {
	return s.repo.ListFeeStructures(ctx, institutionID, academicYearID)
}
 
func (s *Service) AddFeeComponent(ctx context.Context, institutionID, feeStructureID uuid.UUID, req *AddFeeComponentRequest) (*FeeComponent, error) {
	return s.repo.AddFeeComponent(ctx, institutionID, feeStructureID, req)
}
 
// GenerateInvoicesForClass creates one invoice per active student in a
// class, built from a fee structure's components for the given term.
// Same partial-success pattern as report card generation: a student who
// already has an invoice for this term is skipped and recorded, not
// treated as a batch failure — running this twice on the same class
// (e.g. after new admissions) is safe and won't double-bill anyone.
func (s *Service) GenerateInvoicesForClass(ctx context.Context, institutionID uuid.UUID, req *GenerateInvoicesRequest) (*GenerateInvoicesResult, error) {
	components, err := s.repo.GetFeeComponentsForStructure(ctx, institutionID, req.FeeStructureID, req.TermNumber)
	if err != nil {
		return nil, fmt.Errorf("get fee components: %w", err)
	}
	if len(components) == 0 {
		return nil, fmt.Errorf("fee structure has no components for term %d — add components before generating invoices", req.TermNumber)
	}
 
	students, err := s.repo.getActiveStudentIDsForClass(ctx, institutionID, req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("get class roster: %w", err)
	}
 
	result := &GenerateInvoicesResult{
		ClassID:       req.ClassID,
		TermID:        req.TermID,
		TotalStudents: len(students),
		Skipped:       []InvoiceSkip{},
	}
 
	for _, student := range students {
		exists, err := s.repo.InvoiceExistsForStudentTerm(ctx, institutionID, student.id, req.TermID)
		if err != nil {
			log.Warn().Err(err).Str("student_id", student.id.String()).Msg("skipping invoice generation — could not check existing invoice")
			result.Skipped = append(result.Skipped, InvoiceSkip{StudentID: student.id, Reason: err.Error()})
			continue
		}
		if exists {
			result.Skipped = append(result.Skipped, InvoiceSkip{StudentID: student.id, Reason: "invoice already exists for this term"})
			continue
		}
 
		invoiceNumber, err := s.repo.GetNextInvoiceNumber(ctx, institutionID)
		if err != nil {
			result.Skipped = append(result.Skipped, InvoiceSkip{StudentID: student.id, Reason: fmt.Sprintf("could not generate invoice number: %v", err)})
			continue
		}
 
		if _, err := s.repo.CreateInvoiceWithLines(ctx, institutionID, student.id, req.TermID, invoiceNumber, components); err != nil {
			log.Warn().Err(err).Str("student_id", student.id.String()).Msg("skipping invoice — create failed")
			result.Skipped = append(result.Skipped, InvoiceSkip{StudentID: student.id, Reason: err.Error()})
			continue
		}
		result.InvoicesCreated++
	}
 
	return result, nil
}
 
var ErrAllocationExceedsUnapplied = errors.New("allocation amount exceeds the payment's unapplied balance")


// AllocateExistingPayment applies an existing payment's unapplied credit
// to an invoice. amount nil = allocate the full remaining unapplied
// balance; a value allocates a partial amount, leaving the rest as
// credit available for a later allocation.
func (s *Service) AllocateExistingPayment(ctx context.Context, institutionID, paymentID, invoiceID uuid.UUID, amount *float64) (*PaymentAllocation, error) {
	unapplied, err := s.repo.GetUnappliedAmount(ctx, institutionID, paymentID)
	if err != nil {
		return nil, err
	}
 
	amt := unapplied
	if amount != nil {
		amt = *amount
	}
	if amt <= 0 {
		return nil, fmt.Errorf("no unapplied amount available to allocate")
	}
	if amt > unapplied {
		return nil, ErrAllocationExceedsUnapplied
	}
 
	pa, err := s.repo.AllocateExistingPayment(ctx, institutionID, paymentID, invoiceID, amt)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateInvoicePaidAmount(ctx, institutionID, invoiceID); err != nil {
		log.Error().Err(err).Str("invoice_id", invoiceID.String()).Msg("update invoice after manual allocation")
	}
	return pa, nil
}
 
// DeallocatePayment reverses one allocation and recalculates the
// affected invoice's paid_amount/status. The money returns to the
// payment's unapplied credit — it isn't removed from the account, only
// unlinked from that particular invoice.
func (s *Service) DeallocatePayment(ctx context.Context, institutionID, allocationID, reversedBy uuid.UUID, reason string) (*PaymentAllocation, error) {
	pa, err := s.repo.DeallocatePayment(ctx, institutionID, allocationID, reversedBy, reason)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateInvoicePaidAmount(ctx, institutionID, pa.InvoiceID); err != nil {
		log.Error().Err(err).Str("invoice_id", pa.InvoiceID.String()).Msg("update invoice after deallocation")
	}
	return pa, nil
}
