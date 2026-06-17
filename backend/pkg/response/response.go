package response

import (
	"encoding/json"
	"net/http"
)

// Envelope wraps successful API responses.
type Envelope struct {
	Data       interface{}  `json:"data,omitempty"`
	Pagination *Pagination  `json:"pagination,omitempty"`
	Meta       interface{}  `json:"meta,omitempty"`
}

// Pagination metadata for list responses.
type Pagination struct {
	Total    int64  `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	HasNext  bool   `json:"has_next"`
}

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information.
type ErrorDetail struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"` // validation errors
	RequestID string            `json:"request_id,omitempty"`
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Ok writes a 200 response with data wrapped in an envelope.
func Ok(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Envelope{Data: data})
}

// OkList writes a 200 response with data and pagination.
func OkList(w http.ResponseWriter, data interface{}, pagination *Pagination) {
	JSON(w, http.StatusOK, Envelope{Data: data, Pagination: pagination})
}

// Created writes a 201 response.
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, Envelope{Data: data})
}

// NoContent writes a 204 response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error writes a structured error response.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorResponse{
		Error: ErrorDetail{Code: code, Message: message},
	})
}

// ValidationError writes a 422 response with field-level errors.
func ValidationError(w http.ResponseWriter, fields map[string]string) {
	JSON(w, http.StatusUnprocessableEntity, ErrorResponse{
		Error: ErrorDetail{
			Code:    "VALIDATION_ERROR",
			Message: "One or more fields failed validation",
			Fields:  fields,
		},
	})
}

// NotFound writes a 404 response.
func NotFound(w http.ResponseWriter, resource string) {
	Error(w, http.StatusNotFound, resource+"_NOT_FOUND",
		resource+" not found")
}

// Unauthorized writes a 401 response.
func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, "UNAUTHORIZED",
		"Authentication required")
}

// Forbidden writes a 403 response.
func Forbidden(w http.ResponseWriter) {
	Error(w, http.StatusForbidden, "FORBIDDEN",
		"You do not have permission to perform this action")
}

// InternalError writes a 500 response.
func InternalError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
		"An unexpected error occurred")
}

// Conflict writes a 409 response.
func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, "CONFLICT", message)
}
