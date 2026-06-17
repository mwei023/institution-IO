package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"institution-os/internal/finance/mpesa"
	"institution-os/pkg/events"
	"github.com/rs/zerolog/log"
)

// Service implements all finance domain business logic.
type Service struct {
	repo        *Repository
	mpesaClient *mpesa.Client
	publisher   events.Publisher
}

// NewService creates a new finance service.
func NewService(repo *Repository, mpesaClient *mpesa.Client, publisher events.Publisher) *Service {
	return &Service{repo: repo, mpesaClient: mpesaClient, publisher: publisher}
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
func (s *Service) InitiateSTKPush(ctx context.Context, institutionID uuid.UUID, req *STKPushRequest) (*STKPushResponse, error) {
	// Verify student exists
	studentID := req.StudentID
	acct, err := s.repo.GetStudentFeeAccount(ctx, institutionID, studentID)
	if err != nil {
		return nil, fmt.Errorf("student not found: %w", err)
	}

	// Format phone for Daraja (254XXXXXXXXX)
	phone := formatPhone(req.Phone)

	// Use student's admission number as account reference (accountant can identify payment)
	// For now use student ID as reference
	accountRef := acct.StudentName

	amountInt := int64(req.Amount)
	if amountInt <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	stkResp, err := s.mpesaClient.InitiateSTKPush(ctx, phone, accountRef, amountInt)
	if err != nil {
		return nil, fmt.Errorf("initiate STK push: %w", err)
	}

	return &STKPushResponse{
		CheckoutRequestID: stkResp.CheckoutRequestID,
		MerchantRequestID: stkResp.MerchantRequestID,
		Message:           stkResp.CustomerMessage,
	}, nil
}

// ProcessDarajaCallback handles an M-Pesa payment callback.
func (s *Service) ProcessDarajaCallback(ctx context.Context, institutionID uuid.UUID, payload []byte, callback *DarajaCallbackBody) error {
	stk := callback.Body.StkCallback

	// Build transaction record
	tx := &MpesaTransaction{
		ID:                uuid.New(),
		CheckoutRequestID: stk.CheckoutRequestID,
		ResultCode:        stk.ResultCode,
		ResultDescription: stk.ResultDesc,
	}

	// Extract metadata fields (Amount, PhoneNumber, MpesaReceiptNumber, etc.)
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

	// Store raw Daraja event regardless of result
	if err := s.repo.StoreMpesaTransaction(ctx, institutionID, tx, payload); err != nil {
		log.Error().Err(err).Msg("store mpesa transaction")
		// Don't fail — just log
	}

	// Only process successful payments
	if stk.ResultCode != 0 {
		log.Info().
			Int("result_code", stk.ResultCode).
			Str("result_desc", stk.ResultDesc).
			Msg("mpesa payment not successful")
		return nil
	}

	if tx.MpesaReceiptNumber == nil {
		return fmt.Errorf("missing MpesaReceiptNumber in successful callback")
	}

	// Idempotency check
	processed, err := s.repo.IsMpesaReceiptProcessed(ctx, institutionID, *tx.MpesaReceiptNumber)
	if err != nil {
		return err
	}
	if processed {
		log.Info().Str("receipt", *tx.MpesaReceiptNumber).Msg("mpesa transaction already processed — skipping")
		return nil
	}

	// Look up student by account reference (admission number)
	studentID, err := s.repo.GetStudentByAdmissionNumber(ctx, institutionID, tx.AccountReference)
	if err != nil {
		log.Error().Err(err).Str("account_ref", tx.AccountReference).Msg("student not found for mpesa payment")
		return nil // Don't fail callback — log and investigate manually
	}

	// Record payment
	receiptNumber, err := s.repo.GetNextReceiptNumber(ctx, institutionID)
	if err != nil {
		return err
	}

	payment := &FeePayment{
		ID:             uuid.New(),
		InstitutionID:  institutionID,
		StudentID:      *studentID,
		ReceiptNumber:  receiptNumber,
		Amount:         tx.Amount,
		PaymentMethod:  "mpesa",
		MpesaReference: tx.MpesaReceiptNumber,
		ReceivedAt:     time.Now(),
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return fmt.Errorf("create payment: %w", err)
	}

	// Mark transaction as processed
	if err := s.repo.MarkMpesaTransactionProcessed(ctx, institutionID, *tx.MpesaReceiptNumber, payment.ID); err != nil {
		log.Error().Err(err).Msg("mark mpesa processed")
	}

	// Publish payment event
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
func formatPhone(phone string) string {
	if len(phone) == 0 {
		return phone
	}
	// Strip leading +
	if phone[0] == '+' {
		phone = phone[1:]
	}
	// Convert 07XXXXXXXX → 2547XXXXXXXX
	if len(phone) == 10 && phone[0] == '0' {
		phone = "254" + phone[1:]
	}
	return phone
}
