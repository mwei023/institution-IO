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
}

// NewHandler creates a new finance handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetFeeAccount handles GET /api/v1/finance/students/{id}/account
func (h *Handler) GetFeeAccount(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid student ID")
		return
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


// This is called by Safaricom Daraja — no auth required, validated via IP + payload.
// MpesaCallback handles POST /api/v1/finance/mpesa/callback
func (h *Handler) MpesaCallback(w http.ResponseWriter, r *http.Request) {
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