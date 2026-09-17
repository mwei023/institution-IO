package finance

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"institution-os/internal/finance/mpesa"
	"institution-os/pkg/events"
)

// ─── Mocks ────────────────────────────────────────────────────────────

type MockStore struct {
	mock.Mock
}

func (m *MockStore) IsChildOfParent(ctx context.Context, institutionID, parentUserID, studentID uuid.UUID) (bool, error) {
	args := m.Called(ctx, institutionID, parentUserID, studentID)
	return args.Bool(0), args.Error(1)
}
func (m *MockStore) GetStudentFeeAccount(ctx context.Context, institutionID, studentID uuid.UUID) (*StudentFeeAccount, error) {
	args := m.Called(ctx, institutionID, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StudentFeeAccount), args.Error(1)
}
func (m *MockStore) ListInvoicesForStudent(ctx context.Context, institutionID, studentID uuid.UUID) ([]*FeeInvoice, error) {
	args := m.Called(ctx, institutionID, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FeeInvoice), args.Error(1)
}
func (m *MockStore) ListPaymentsForStudent(ctx context.Context, institutionID, studentID uuid.UUID) ([]*FeePayment, error) {
	args := m.Called(ctx, institutionID, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FeePayment), args.Error(1)
}
func (m *MockStore) GetInvoiceByID(ctx context.Context, institutionID, invoiceID uuid.UUID) (*FeeInvoice, error) {
	args := m.Called(ctx, institutionID, invoiceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeeInvoice), args.Error(1)
}
func (m *MockStore) GetDefaultChannel(ctx context.Context, institutionID uuid.UUID) (*PaymentChannel, error) {
	args := m.Called(ctx, institutionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentChannel), args.Error(1)
}
func (m *MockStore) CreateStkRequest(ctx context.Context, req *StkRequest) error {
	return m.Called(ctx, req).Error(0)
}
func (m *MockStore) GetStkRequestByCheckoutID(ctx context.Context, checkoutRequestID string) (*StkRequest, error) {
	args := m.Called(ctx, checkoutRequestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StkRequest), args.Error(1)
}
func (m *MockStore) MarkStkRequestStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *MockStore) StoreMpesaTransaction(ctx context.Context, institutionID uuid.UUID, tx *MpesaTransaction, rawPayload []byte) error {
	return m.Called(ctx, institutionID, tx, rawPayload).Error(0)
}
func (m *MockStore) IsMpesaReceiptProcessed(ctx context.Context, institutionID uuid.UUID, receiptNumber string) (bool, error) {
	args := m.Called(ctx, institutionID, receiptNumber)
	return args.Bool(0), args.Error(1)
}
func (m *MockStore) MarkMpesaTransactionProcessed(ctx context.Context, institutionID uuid.UUID, receiptNumber string, paymentID uuid.UUID) error {
	return m.Called(ctx, institutionID, receiptNumber, paymentID).Error(0)
}
func (m *MockStore) GetNextReceiptNumber(ctx context.Context, institutionID uuid.UUID) (string, error) {
	args := m.Called(ctx, institutionID)
	return args.String(0), args.Error(1)
}
func (m *MockStore) CreatePayment(ctx context.Context, p *FeePayment) error {
	return m.Called(ctx, p).Error(0)
}
func (m *MockStore) AllocatePayment(ctx context.Context, institutionID, studentID, paymentID uuid.UUID, amount float64) ([]uuid.UUID, error) {
	args := m.Called(ctx, institutionID, studentID, paymentID, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}
func (m *MockStore) CreatePaymentAllocation(ctx context.Context, institutionID, paymentID, invoiceID uuid.UUID, amount float64) error {
	return m.Called(ctx, institutionID, paymentID, invoiceID, amount).Error(0)
}
func (m *MockStore) UpdateInvoicePaidAmount(ctx context.Context, institutionID, invoiceID uuid.UUID) error {
	return m.Called(ctx, institutionID, invoiceID).Error(0)
}
func (m *MockStore) GetUnappliedAmount(ctx context.Context, institutionID, paymentID uuid.UUID) (float64, error) {
	args := m.Called(ctx, institutionID, paymentID)
	return args.Get(0).(float64), args.Error(1)
}
func (m *MockStore) AllocateExistingPayment(ctx context.Context, institutionID, paymentID, invoiceID uuid.UUID, amount float64) (*PaymentAllocation, error) {
	args := m.Called(ctx, institutionID, paymentID, invoiceID, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentAllocation), args.Error(1)
}
func (m *MockStore) DeallocatePayment(ctx context.Context, institutionID, allocationID, reversedBy uuid.UUID, reason string) (*PaymentAllocation, error) {
	args := m.Called(ctx, institutionID, allocationID, reversedBy, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentAllocation), args.Error(1)
}
func (m *MockStore) GetFeeCollectionSummary(ctx context.Context, institutionID uuid.UUID, termID *uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, institutionID, termID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}
func (m *MockStore) ListStudentsWithOutstandingBalance(ctx context.Context, institutionID uuid.UUID) ([]*StudentBalance, error) {
	args := m.Called(ctx, institutionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*StudentBalance), args.Error(1)
}
func (m *MockStore) CreateFeeStructure(ctx context.Context, institutionID uuid.UUID, req *CreateFeeStructureRequest) (*FeeStructure, error) {
	args := m.Called(ctx, institutionID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeeStructure), args.Error(1)
}
func (m *MockStore) ListFeeStructures(ctx context.Context, institutionID uuid.UUID, academicYearID *uuid.UUID) ([]*FeeStructure, error) {
	args := m.Called(ctx, institutionID, academicYearID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FeeStructure), args.Error(1)
}
func (m *MockStore) AddFeeComponent(ctx context.Context, institutionID, feeStructureID uuid.UUID, req *AddFeeComponentRequest) (*FeeComponent, error) {
	args := m.Called(ctx, institutionID, feeStructureID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeeComponent), args.Error(1)
}
func (m *MockStore) GetFeeComponentsForStructure(ctx context.Context, institutionID, feeStructureID uuid.UUID, termNumber int) ([]*FeeComponent, error) {
	args := m.Called(ctx, institutionID, feeStructureID, termNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FeeComponent), args.Error(1)
}
func (m *MockStore) getActiveStudentIDsForClass(ctx context.Context, institutionID, classID uuid.UUID) ([]studentForInvoicing, error) {
	args := m.Called(ctx, institutionID, classID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]studentForInvoicing), args.Error(1)
}
func (m *MockStore) InvoiceExistsForStudentTerm(ctx context.Context, institutionID, studentID, termID uuid.UUID) (bool, error) {
	args := m.Called(ctx, institutionID, studentID, termID)
	return args.Bool(0), args.Error(1)
}
func (m *MockStore) GetNextInvoiceNumber(ctx context.Context, institutionID uuid.UUID) (string, error) {
	args := m.Called(ctx, institutionID)
	return args.String(0), args.Error(1)
}
func (m *MockStore) CreateInvoiceWithLines(ctx context.Context, institutionID, studentID, termID uuid.UUID, invoiceNumber string, components []*FeeComponent) (*FeeInvoice, error) {
	args := m.Called(ctx, institutionID, studentID, termID, invoiceNumber, components)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeeInvoice), args.Error(1)
}

// RecordingPublisher captures published events so tests can assert on them.
type RecordingPublisher struct {
	Events []events.Event
}

func (r *RecordingPublisher) Publish(e events.Event) error {
	r.Events = append(r.Events, e)
	return nil
}

type MockMpesaClient struct {
	mock.Mock
}

func (m *MockMpesaClient) InitiateSTKPush(ctx context.Context, creds mpesa.ChannelCredentials, callbackURL, phone, accountRef string, amount int64) (*mpesa.STKPushResponse, error) {
	args := m.Called(ctx, creds, callbackURL, phone, accountRef, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mpesa.STKPushResponse), args.Error(1)
}

// helper: a successful Daraja callback payload for a given checkout ID + receipt
func darajaCallback(t *testing.T, checkoutID, receipt string, amount float64, resultCode int) ([]byte, *DarajaCallbackBody) {
	t.Helper()
	raw := []byte(`{"Body":{"stkCallback":{"MerchantRequestID":"mr-1","CheckoutRequestID":"` + checkoutID +
		`","ResultCode":` + jsonInt(resultCode) + `,"ResultDesc":"ok","CallbackMetadata":{"Item":[` +
		`{"Name":"Amount","Value":` + jsonFloat(amount) + `},` +
		`{"Name":"MpesaReceiptNumber","Value":"` + receipt + `"},` +
		`{"Name":"PhoneNumber","Value":254722100201},` +
		`{"Name":"AccountReference","Value":"Amina Wanjiru"}` +
		`]}}}}`)
	var cb DarajaCallbackBody
	require.NoError(t, json.Unmarshal(raw, &cb))
	return raw, &cb
}

func jsonInt(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}
func jsonFloat(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func newService(repo *MockStore, client MpesaClient, pub events.Publisher) *Service {
	return NewService(repo, client, pub, "https://example.com/callback")
}

// ─── RecordManualPayment ──────────────────────────────────────────────

func TestRecordManualPayment_Success(t *testing.T) {
	inst := uuid.New()
	student := uuid.New()
	invoice := uuid.New()
	repo := new(MockStore)
	pub := &RecordingPublisher{}

	repo.On("GetStudentFeeAccount", mock.Anything, inst, student).
		Return(&StudentFeeAccount{StudentID: student}, nil)
	repo.On("GetInvoiceByID", mock.Anything, inst, invoice).
		Return(&FeeInvoice{ID: invoice, StudentID: student}, nil)
	repo.On("GetNextReceiptNumber", mock.Anything, inst).Return("R-000001", nil)
	repo.On("CreatePayment", mock.Anything, mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			p := args.Get(1).(*FeePayment)
			assert.Equal(t, "cash", p.PaymentMethod)
			assert.Equal(t, 500.0, p.Amount)
			assert.Equal(t, "R-000001", p.ReceiptNumber)
			assert.NotNil(t, p.InvoiceID)
		})
	repo.On("CreatePaymentAllocation", mock.Anything, inst, mock.Anything, invoice, 500.0).Return(nil)
	repo.On("UpdateInvoicePaidAmount", mock.Anything, inst, invoice).Return(nil)

	svc := newService(repo, nil, pub)
	payment, err := svc.RecordManualPayment(context.Background(), inst, uuid.New(), &ManualPaymentRequest{
		StudentID:     student,
		InvoiceID:     &invoice,
		Amount:        500,
		PaymentMethod: "cash",
	})

	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Len(t, pub.Events, 1, "payment-received event must be published")
	repo.AssertExpectations(t)
}

func TestRecordManualPayment_RejectsInvalidAmountAndMethod(t *testing.T) {
	inst := uuid.New()
	repo := new(MockStore)
	svc := newService(repo, nil, &events.NoopPublisher{})

	_, err := svc.RecordManualPayment(context.Background(), inst, uuid.New(), &ManualPaymentRequest{Amount: 0, PaymentMethod: "cash"})
	assert.ErrorContains(t, err, "amount must be greater than 0")

	_, err = svc.RecordManualPayment(context.Background(), inst, uuid.New(), &ManualPaymentRequest{Amount: 100, PaymentMethod: "bitcoin"})
	assert.ErrorContains(t, err, "invalid payment method")

	repo.AssertNotCalled(t, "CreatePayment", mock.Anything)
}

func TestRecordManualPayment_RejectsInvoiceOfAnotherStudent(t *testing.T) {
	inst := uuid.New()
	student := uuid.New()
	otherStudent := uuid.New()
	invoice := uuid.New()
	repo := new(MockStore)

	repo.On("GetStudentFeeAccount", mock.Anything, inst, student).
		Return(&StudentFeeAccount{StudentID: student}, nil)
	repo.On("GetInvoiceByID", mock.Anything, inst, invoice).
		Return(&FeeInvoice{ID: invoice, StudentID: otherStudent}, nil)

	svc := newService(repo, nil, &events.NoopPublisher{})
	_, err := svc.RecordManualPayment(context.Background(), inst, uuid.New(), &ManualPaymentRequest{
		StudentID:     student,
		InvoiceID:     &invoice,
		Amount:        100,
		PaymentMethod: "cash",
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "invoice does not belong to student")
	repo.AssertNotCalled(t, "CreatePayment", mock.Anything)
}

// ─── ProcessDarajaCallback ────────────────────────────────────────────

func TestProcessDarajaCallback_Success_CreatesPayment(t *testing.T) {
	inst := uuid.New()
	student := uuid.New()
	channel := uuid.New()
	stkReqID := uuid.New()
	repo := new(MockStore)
	pub := &RecordingPublisher{}

	raw, cb := darajaCallback(t, "ws_CO_123", "SBX12AB34CD", 2500, 0)

	repo.On("GetStkRequestByCheckoutID", mock.Anything, "ws_CO_123").
		Return(&StkRequest{
			ID: stkReqID, InstitutionID: inst, ChannelID: channel,
			StudentID: student, CheckoutRequestID: "ws_CO_123",
		}, nil)
	repo.On("StoreMpesaTransaction", mock.Anything, inst, mock.Anything, raw).Return(nil)
	repo.On("IsMpesaReceiptProcessed", mock.Anything, inst, "SBX12AB34CD").Return(false, nil)
	repo.On("GetNextReceiptNumber", mock.Anything, inst).Return("R-000042", nil)
	repo.On("CreatePayment", mock.Anything, mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			p := args.Get(1).(*FeePayment)
			// Attribution must come from OUR stk-request record, never the payload.
			assert.Equal(t, inst, p.InstitutionID)
			assert.Equal(t, student, p.StudentID)
			assert.Equal(t, "mpesa", p.PaymentMethod)
			assert.Equal(t, 2500.0, p.Amount)
			require.NotNil(t, p.MpesaReference)
			assert.Equal(t, "SBX12AB34CD", *p.MpesaReference)
		})
	repo.On("AllocatePayment", mock.Anything, inst, student, mock.Anything, 2500.0).
		Return([]uuid.UUID{}, nil)
	repo.On("MarkMpesaTransactionProcessed", mock.Anything, inst, "SBX12AB34CD", mock.Anything).Return(nil)
	repo.On("MarkStkRequestStatus", mock.Anything, stkReqID, "completed").Return(nil)

	svc := newService(repo, nil, pub)
	require.NoError(t, svc.ProcessDarajaCallback(context.Background(), raw, cb))
	assert.Len(t, pub.Events, 1)
	repo.AssertExpectations(t)
}

func TestProcessDarajaCallback_UnknownCheckoutID_IsAcceptedWithoutEffect(t *testing.T) {
	// A spoofed or stale callback must never create a payment, and must
	// return nil so Daraja does not retry it forever.
	repo := new(MockStore)
	raw, cb := darajaCallback(t, "ws_CO_SPOOFED", "SBX00000000", 9999, 0)

	repo.On("GetStkRequestByCheckoutID", mock.Anything, "ws_CO_SPOOFED").
		Return(nil, ErrStkRequestNotFound)

	svc := newService(repo, nil, &events.NoopPublisher{})
	require.NoError(t, svc.ProcessDarajaCallback(context.Background(), raw, cb))
	repo.AssertNotCalled(t, "CreatePayment", mock.Anything)
	repo.AssertNotCalled(t, "GetNextReceiptNumber", mock.Anything, mock.Anything)
}

func TestProcessDarajaCallback_FailedResult_MarksRequestFailedOnly(t *testing.T) {
	// ResultCode != 0 = the user cancelled/timeout: store the raw
	// transaction for audit, mark the push failed, never create a payment.
	inst := uuid.New()
	stkReqID := uuid.New()
	repo := new(MockStore)

	raw, cb := darajaCallback(t, "ws_CO_CANCELLED", "", 0, 1032)

	repo.On("GetStkRequestByCheckoutID", mock.Anything, "ws_CO_CANCELLED").
		Return(&StkRequest{ID: stkReqID, InstitutionID: inst, StudentID: uuid.New()}, nil)
	repo.On("StoreMpesaTransaction", mock.Anything, inst, mock.Anything, raw).Return(nil)
	repo.On("MarkStkRequestStatus", mock.Anything, stkReqID, "failed").Return(nil)

	svc := newService(repo, nil, &events.NoopPublisher{})
	require.NoError(t, svc.ProcessDarajaCallback(context.Background(), raw, cb))
	repo.AssertNotCalled(t, "CreatePayment", mock.Anything)
	repo.AssertNotCalled(t, "IsMpesaReceiptProcessed", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessDarajaCallback_DuplicateReceipt_IsIdempotent(t *testing.T) {
	// Daraja retries callbacks. A receipt already processed must be
	// skipped — this is what prevents double-crediting a student's account.
	inst := uuid.New()
	repo := new(MockStore)

	raw, cb := darajaCallback(t, "ws_CO_RETRY", "SBXDUP00001", 2500, 0)

	repo.On("GetStkRequestByCheckoutID", mock.Anything, "ws_CO_RETRY").
		Return(&StkRequest{ID: uuid.New(), InstitutionID: inst, StudentID: uuid.New()}, nil)
	repo.On("StoreMpesaTransaction", mock.Anything, inst, mock.Anything, raw).Return(nil)
	repo.On("IsMpesaReceiptProcessed", mock.Anything, inst, "SBXDUP00001").Return(true, nil)

	svc := newService(repo, nil, &events.NoopPublisher{})
	require.NoError(t, svc.ProcessDarajaCallback(context.Background(), raw, cb))
	repo.AssertNotCalled(t, "CreatePayment", mock.Anything)
	repo.AssertNotCalled(t, "GetNextReceiptNumber", mock.Anything, mock.Anything)
}

func TestProcessDarajaCallback_MissingCheckoutID_IsRejected(t *testing.T) {
	repo := new(MockStore)
	cb := &DarajaCallbackBody{} // no CheckoutRequestID at all

	svc := newService(repo, nil, &events.NoopPublisher{})
	err := svc.ProcessDarajaCallback(context.Background(), []byte("{}"), cb)
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing CheckoutRequestID")
}

// ─── AllocateExistingPayment ──────────────────────────────────────────

func TestAllocateExistingPayment_RejectsMoreThanUnapplied(t *testing.T) {
	inst := uuid.New()
	payment := uuid.New()
	invoice := uuid.New()
	repo := new(MockStore)

	repo.On("GetUnappliedAmount", mock.Anything, inst, payment).Return(300.0, nil)

	svc := newService(repo, nil, &events.NoopPublisher{})

	_, err := svc.AllocateExistingPayment(context.Background(), inst, payment, invoice, floatPtr(500))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAllocationExceedsUnapplied)

	// Zero/negative also rejected — nothing to allocate.
	_, err = svc.AllocateExistingPayment(context.Background(), inst, payment, invoice, floatPtr(0))
	require.Error(t, err)
	assert.ErrorContains(t, err, "no unapplied amount available")
}

func TestAllocateExistingPayment_NilAmount_AllocatesFullBalance(t *testing.T) {
	inst := uuid.New()
	payment := uuid.New()
	invoice := uuid.New()
	repo := new(MockStore)

	repo.On("GetUnappliedAmount", mock.Anything, inst, payment).Return(300.0, nil)
	repo.On("AllocateExistingPayment", mock.Anything, inst, payment, invoice, 300.0).
		Return(&PaymentAllocation{ID: uuid.New(), InvoiceID: invoice, AmountApplied: 300}, nil)
	repo.On("UpdateInvoicePaidAmount", mock.Anything, inst, invoice).Return(nil)

	svc := newService(repo, nil, &events.NoopPublisher{})
	pa, err := svc.AllocateExistingPayment(context.Background(), inst, payment, invoice, nil)

	require.NoError(t, err)
	assert.Equal(t, 300.0, pa.AmountApplied)
	repo.AssertExpectations(t)
}

// ─── GenerateInvoicesForClass ─────────────────────────────────────────

func TestGenerateInvoicesForClass_SkipsExistingAndCountsCreated(t *testing.T) {
	inst := uuid.New()
	class := uuid.New()
	structure := uuid.New()
	term := uuid.New()
	s1, s2 := uuid.New(), uuid.New()
	repo := new(MockStore)

	comps := []*FeeComponent{{ID: uuid.New(), Name: "Tuition", Amount: 12000}}
	repo.On("GetFeeComponentsForStructure", mock.Anything, inst, structure, 1).Return(comps, nil)
	repo.On("getActiveStudentIDsForClass", mock.Anything, inst, class).
		Return([]studentForInvoicing{{id: s1}, {id: s2}}, nil)

	// s1 already invoiced this term → skipped; s2 gets a new invoice.
	repo.On("InvoiceExistsForStudentTerm", mock.Anything, inst, s1, term).Return(true, nil)
	repo.On("InvoiceExistsForStudentTerm", mock.Anything, inst, s2, term).Return(false, nil)
	repo.On("GetNextInvoiceNumber", mock.Anything, inst).Return("INV-000008", nil)
	repo.On("CreateInvoiceWithLines", mock.Anything, inst, s2, term, "INV-000008", comps).
		Return(&FeeInvoice{}, nil)

	svc := newService(repo, nil, &events.NoopPublisher{})
	result, err := svc.GenerateInvoicesForClass(context.Background(), inst, &GenerateInvoicesRequest{
		FeeStructureID: structure, ClassID: class, TermID: term, TermNumber: 1,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalStudents)
	assert.Equal(t, 1, result.InvoicesCreated)
	require.Len(t, result.Skipped, 1)
	assert.Equal(t, s1, result.Skipped[0].StudentID)
	assert.Contains(t, result.Skipped[0].Reason, "already exists")
	repo.AssertExpectations(t)
}

func TestGenerateInvoicesForClass_RejectsStructureWithoutComponents(t *testing.T) {
	// Guard against generating zero-amount invoices — which would silently
	// under-bill every student in the class.
	inst := uuid.New()
	repo := new(MockStore)

	repo.On("GetFeeComponentsForStructure", mock.Anything, inst, mock.Anything, 1).
		Return([]*FeeComponent{}, nil)

	svc := newService(repo, nil, &events.NoopPublisher{})
	_, err := svc.GenerateInvoicesForClass(context.Background(), inst, &GenerateInvoicesRequest{
		FeeStructureID: uuid.New(), ClassID: uuid.New(), TermID: uuid.New(), TermNumber: 1,
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "no components for term 1")
}

// ─── InitiateSTKPush ──────────────────────────────────────────────────

func TestInitiateSTKPush_HappyPath_RecordsRequestBeforeReturning(t *testing.T) {
	inst := uuid.New()
	student := uuid.New()
	channel := uuid.New()
	repo := new(MockStore)
	client := new(MockMpesaClient)

	creds := mpesa.ChannelCredentials{Shortcode: "174379", Passkey: "pass"}

	repo.On("GetStudentFeeAccount", mock.Anything, inst, student).
		Return(&StudentFeeAccount{StudentID: student, StudentName: "Amina Wanjiru"}, nil)
	repo.On("GetDefaultChannel", mock.Anything, inst).
		Return(&PaymentChannel{ID: channel, Shortcode: strPtr("174379"), Passkey: strPtr("pass")}, nil)
	client.On("InitiateSTKPush", mock.Anything, creds, "https://example.com/callback", "254722100201", "Amina Wanjiru", int64(2500)).
		Return(&mpesa.STKPushResponse{CheckoutRequestID: "ws_CO_NEW", CustomerMessage: "Check your phone"}, nil)
	repo.On("CreateStkRequest", mock.Anything, mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			req := args.Get(1).(*StkRequest)
			// The record created here is what makes the callback attributable.
			assert.Equal(t, inst, req.InstitutionID)
			assert.Equal(t, channel, req.ChannelID)
			assert.Equal(t, student, req.StudentID)
			assert.Equal(t, "ws_CO_NEW", req.CheckoutRequestID)
		})

	svc := newService(repo, client, &events.NoopPublisher{})
	resp, err := svc.InitiateSTKPush(context.Background(), inst, uuid.New(), &STKPushRequest{
		StudentID: student, Amount: 2500, Phone: "0722 100 201",
	})

	require.NoError(t, err)
	assert.Equal(t, "ws_CO_NEW", resp.CheckoutRequestID)
	repo.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestInitiateSTKPush_RejectsBadPhoneAndMissingChannel(t *testing.T) {
	inst := uuid.New()
	student := uuid.New()
	repo := new(MockStore)
	client := new(MockMpesaClient)

	repo.On("GetStudentFeeAccount", mock.Anything, inst, student).
		Return(&StudentFeeAccount{StudentID: student}, nil)
	repo.On("GetDefaultChannel", mock.Anything, inst).Return(nil, ErrChannelNotFound)

	svc := newService(repo, client, &events.NoopPublisher{})

	// No channel configured — must fail before any push goes out.
	_, err := svc.InitiateSTKPush(context.Background(), inst, uuid.New(), &STKPushRequest{
		StudentID: student, Amount: 100, Phone: "+254722100201",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "no active M-Pesa channel")
	client.AssertNotCalled(t, "InitiateSTKPush", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	// Channel exists but the phone is garbage — still nothing sent.
	repo2 := new(MockStore)
	repo2.On("GetStudentFeeAccount", mock.Anything, inst, student).
		Return(&StudentFeeAccount{StudentID: student}, nil)
	repo2.On("GetDefaultChannel", mock.Anything, inst).
		Return(&PaymentChannel{ID: uuid.New(), Shortcode: strPtr("174379"), Passkey: strPtr("pass")}, nil)
	svc2 := newService(repo2, client, &events.NoopPublisher{})

	_, err = svc2.InitiateSTKPush(context.Background(), inst, uuid.New(), &STKPushRequest{
		StudentID: student, Amount: 100, Phone: "12345-ABC",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid phone number")
	client.AssertNotCalled(t, "InitiateSTKPush", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ─── formatPhone (unit) ───────────────────────────────────────────────

func TestFormatPhone(t *testing.T) {
	cases := map[string]string{
		"+254722100201":  "254722100201",
		"0722 100 201":   "254722100201",
		"0722-100-201":   "254722100201",
		"(0722) 100201":  "254722100201",
		"254722100201":   "254722100201",
		"722100201":      "722100201", // 9 digits, no leading 0 — left as-is (rejected by length check)
		"12345-ABC":      "12345",
		"":               "",
	}
	for in, want := range cases {
		assert.Equal(t, want, formatPhone(in), "formatPhone(%q)", in)
	}
}

// ─── misc ─────────────────────────────────────────────────────────────

func TestVerifyParentChild(t *testing.T) {
	inst, parent, student := uuid.New(), uuid.New(), uuid.New()
	repo := new(MockStore)

	repo.On("IsChildOfParent", mock.Anything, inst, parent, student).Return(true, nil).Once()
	repo.On("IsChildOfParent", mock.Anything, inst, parent, student).Return(false, nil).Once()

	svc := newService(repo, nil, &events.NoopPublisher{})
	require.NoError(t, svc.VerifyParentChild(context.Background(), inst, parent, student))

	err := svc.VerifyParentChild(context.Background(), inst, parent, student)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not linked to student")
}

func floatPtr(f float64) *float64 { return &f }
func strPtr(s string) *string    { return &s }
