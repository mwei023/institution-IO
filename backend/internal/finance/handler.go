package finance

import (
	"encoding/json"
	"errors"
	"net/http"

	"institution-os/internal/auth"
	"institution-os/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Handler handles finance HTTP endpoints.
type Handler struct {
	svc *Service
	// callbackSecret gates the Daraja callback (pilot P0-3). Empty = warn-only.
	callbackSecret string
}

// NewHandler creates a new finance handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// NewHandlerWithCallbackSecret wires the shared-secret gate for the callback.
func NewHandlerWithCallbackSecret(svc *Service, secret string) *Handler {
	return &Handler{svc: svc, callbackSecret: secret}
}

// GetFeeAccount handles GET /api/v1/finance/students/{id}/account
// Shared by staff and the parent portal (GET /parent/children/{id}/account).
// Pilot P0-4: callers with role=parent must prove linkage, else 403.
func (h *Handler) GetFeeAccount(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid student ID")
		return
	}
	if claims != nil && claims.Role == "parent" {
		parentID, perr := uuid.Parse(claims.UserID)
		if perr != nil {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user")
			return
		}
		if verr := h.svc.VerifyParentChild(r.Context(), institutionID, parentID, studentID); verr != nil {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Not linked to student")
			return
		}
	}
	acct, err := h.svc.GetStudentFeeAccount(r.Context(), institutionID, studentID)
	if err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			response.NotFound(w, "STUDENT")
			return
		}
		log.Error().Err(err).Msg("get fee account")
		response.InternalError(w)
		return
	}

	response.Ok(w, acct)
}

// GetPaymentHistory handles GET /api/v1/finance/students/{id}/payments
func (h *Handler) GetPaymentHistory(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid student ID")
		return
	}

	payments, err := h.svc.GetPaymentHistory(r.Context(), institutionID, studentID)
	if err != nil {
		log.Error().Err(err).Msg("get payment history")
		response.InternalError(w)
		return
	}

	response.Ok(w, payments)
}

// InitiateSTKPush handles POST /api/v1/finance/pay/stk-push
// InitiateSTKPush handles POST /api/v1/finance/pay/stk-push
func (h *Handler) InitiateSTKPush(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())

	initiatedBy, err := uuid.Parse(claims.UserID)
	if err != nil {
		response.InternalError(w)
		return
	}

	var req STKPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	resp, err := h.svc.InitiateSTKPush(r.Context(), institutionID, initiatedBy, &req)
	if err != nil {
		log.Error().Err(err).Msg("initiate stk push")
		response.Error(w, http.StatusBadRequest, "STK_PUSH_FAILED", err.Error())
		return
	}

	response.Ok(w, resp)
}


// Called by Safaricom Daraja — no JWT auth. Attribution comes from OUR
// stk-request record keyed by CheckoutRequestID (never from the payload),
// unknown checkout IDs are logged + accepted without effect (idempotent),
// and a shared secret (X-Callback-Secret) gates spoofed callbacks when
// MPESA_CALLBACK_SECRET is set. Pilot P0-3: set the secret in production;
// empty secret logs a loud warning and accepts (sandbox onboarding only).
// MpesaCallback handles POST /api/v1/finance/mpesa/callback
func (h *Handler) MpesaCallback(w http.ResponseWriter, r *http.Request) {
	if h.callbackSecret != "" {
		if r.Header.Get("X-Callback-Secret") != h.callbackSecret {
			log.Warn().Str("remote", r.RemoteAddr).Msg("mpesa callback rejected — bad secret")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	} else {
		log.Warn().Msg("mpesa callback accepted without secret — set MPESA_CALLBACK_SECRET before pilot")
	}
	var rawPayload json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&rawPayload); err != nil {
		log.Error().Err(err).Msg("mpesa callback decode")
		w.WriteHeader(http.StatusOK)
		return
	}

	var callback DarajaCallbackBody
	if err := json.Unmarshal(rawPayload, &callback); err != nil {
		log.Error().Err(err).Msg("mpesa callback unmarshal")
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.svc.ProcessDarajaCallback(r.Context(), rawPayload, &callback); err != nil {
		log.Error().Err(err).Msg("process mpesa callback")
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Accepted"}`))
}

// RecordManualPayment handles POST /api/v1/finance/pay/manual
func (h *Handler) RecordManualPayment(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())

	receiverID, err := uuid.Parse(claims.UserID)
	if err != nil {
		response.InternalError(w)
		return
	}

	var req ManualPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	payment, err := h.svc.RecordManualPayment(r.Context(), institutionID, receiverID, &req)
	if err != nil {
		log.Error().Err(err).Msg("record manual payment")
		response.Error(w, http.StatusBadRequest, "PAYMENT_FAILED", err.Error())
		return
	}

	response.Created(w, payment)
}

// GetFeeCollectionSummary handles GET /api/v1/finance/summary
func (h *Handler) GetFeeCollectionSummary(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	var termID *uuid.UUID
	if termIDStr := r.URL.Query().Get("term_id"); termIDStr != "" {
		id, err := uuid.Parse(termIDStr)
		if err == nil {
			termID = &id
		}
	}

	summary, err := h.svc.GetFeeCollectionSummary(r.Context(), institutionID, termID)
	if err != nil {
		log.Error().Err(err).Msg("get fee summary")
		response.InternalError(w)
		return
	}

	response.Ok(w, summary)
}

// GetOutstandingBalances handles GET /api/v1/finance/outstanding
func (h *Handler) GetOutstandingBalances(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	balances, err := h.svc.GetOutstandingBalances(r.Context(), institutionID)
	if err != nil {
		log.Error().Err(err).Msg("get outstanding balances")
		response.InternalError(w)
		return
	}

	response.Ok(w, balances)
}

func (h *Handler) CreateFeeStructure(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req CreateFeeStructureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	fs, err := h.svc.CreateFeeStructure(r.Context(), institutionID, &req)
	if err != nil {
		log.Error().Err(err).Msg("create fee structure")
		response.Error(w, http.StatusBadRequest, "FEE_STRUCTURE_ERROR", err.Error())
		return
	}
	response.Created(w, fs)
}
 
// ListFeeStructures handles GET /api/v1/finance/fee-structures?academic_year_id=
func (h *Handler) ListFeeStructures(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var academicYearID *uuid.UUID
	if s := r.URL.Query().Get("academic_year_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			academicYearID = &id
		}
	}
	structures, err := h.svc.ListFeeStructures(r.Context(), institutionID, academicYearID)
	if err != nil {
		log.Error().Err(err).Msg("list fee structures")
		response.InternalError(w)
		return
	}
	response.Ok(w, structures)
}
 
// AddFeeComponent handles POST /api/v1/finance/fee-structures/{id}/components
func (h *Handler) AddFeeComponent(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	feeStructureID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid fee structure ID")
		return
	}
	var req AddFeeComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	fc, err := h.svc.AddFeeComponent(r.Context(), institutionID, feeStructureID, &req)
	
	if err != nil {
		if errors.Is(err, ErrFeeStructureNotFound) {
			response.NotFound(w, "FEE_STRUCTURE")
			return
		}
		log.Error().Err(err).Msg("add fee component")
		response.Error(w, http.StatusBadRequest, "FEE_COMPONENT_ERROR", err.Error())
		return
	}
	response.Created(w, fc)
}
 
// GenerateInvoicesForClass handles POST /api/v1/finance/invoices/generate-class
func (h *Handler) GenerateInvoicesForClass(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req GenerateInvoicesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	result, err := h.svc.GenerateInvoicesForClass(r.Context(), institutionID, &req)
	if err != nil {
		log.Error().Err(err).Msg("generate invoices for class")
		response.Error(w, http.StatusBadRequest, "INVOICE_GENERATION_ERROR", err.Error())
		return
	}
	response.Ok(w, result)
}
 

// AllocatePayment handles POST /api/v1/finance/payments/{id}/allocate
func (h *Handler) AllocatePayment(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	paymentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid payment ID")
		return
	}
	var req AllocatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	pa, err := h.svc.AllocateExistingPayment(r.Context(), institutionID, paymentID, req.InvoiceID, req.Amount)
	if err != nil {
		log.Error().Err(err).Msg("allocate payment")
		response.Error(w, http.StatusBadRequest, "ALLOCATION_ERROR", err.Error())
		return
	}
	response.Created(w, pa)
}
 
// DeallocatePayment handles POST /api/v1/finance/allocations/{id}/deallocate
func (h *Handler) DeallocatePayment(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	reversedBy, err := uuid.Parse(claims.UserID)
	if err != nil {
		response.InternalError(w)
		return
	}
	allocationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid allocation ID")
		return
	}
	var req DeallocateRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // reason is optional, ignore decode failure on empty body
 
	pa, err := h.svc.DeallocatePayment(r.Context(), institutionID, allocationID, reversedBy, req.Reason)
	if err != nil {
		log.Error().Err(err).Msg("deallocate payment")
		response.Error(w, http.StatusBadRequest, "DEALLOCATION_ERROR", err.Error())
		return
	}
	response.Ok(w, pa)
}