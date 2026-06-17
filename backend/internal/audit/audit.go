package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"institution-os/internal/auth"
	"institution-os/pkg/db"
	"github.com/rs/zerolog/log"
)

// Log is an immutable audit log entry.
type Log struct {
	ID            uuid.UUID   `json:"id"`
	InstitutionID *uuid.UUID  `json:"institution_id,omitempty"`
	UserID        *string     `json:"user_id,omitempty"`
	Action        string      `json:"action"`
	TableName     *string     `json:"table_name,omitempty"`
	RecordID      *uuid.UUID  `json:"record_id,omitempty"`
	OldValues     interface{} `json:"old_values,omitempty"`
	NewValues     interface{} `json:"new_values,omitempty"`
	IPAddress     string      `json:"ip_address,omitempty"`
	UserAgent     string      `json:"user_agent,omitempty"`
	RequestID     string      `json:"request_id,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
}

// Service handles writing to the immutable audit log.
type Service struct {
	db *db.Pool
}

// NewService creates a new audit service.
func NewService(db *db.Pool) *Service {
	return &Service{db: db}
}

// Write inserts an audit log entry. This is fire-and-forget — errors are logged but not returned.
func (s *Service) Write(ctx context.Context, entry *Log) {
	var oldJSON, newJSON []byte
	if entry.OldValues != nil {
		oldJSON, _ = json.Marshal(entry.OldValues)
	}
	if entry.NewValues != nil {
		newJSON, _ = json.Marshal(entry.NewValues)
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO audit_logs (
			id, institution_id, user_id, action, table_name, record_id,
			old_values, new_values, ip_address, user_agent, request_id, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())`,
		uuid.New(),
		entry.InstitutionID,
		entry.UserID,
		entry.Action,
		entry.TableName,
		entry.RecordID,
		oldJSON,
		newJSON,
		entry.IPAddress,
		entry.UserAgent,
		entry.RequestID,
	)
	if err != nil {
		// Audit write failure is serious but should not break the request
		log.Error().Err(err).Str("action", entry.Action).Msg("audit log write failed")
	}
}

// requestIDKey is the context key for request IDs.
type requestIDKey struct{}

// RequestIDMiddleware injects a unique request ID into every request context.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext extracts the request ID from context.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// HTTPMiddleware logs every authenticated API request to the audit log.
func HTTPMiddleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only audit state-changing methods
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			claims := auth.ClaimsFromContext(r.Context())
			if claims == nil {
				next.ServeHTTP(w, r)
				return
			}

			iid, _ := uuid.Parse(claims.InstitutionID)
			userID := claims.UserID

			entry := &Log{
				Action:    r.Method + " " + r.URL.Path,
				UserID:    &userID,
				IPAddress: r.RemoteAddr,
				UserAgent: r.UserAgent(),
				RequestID: RequestIDFromContext(r.Context()),
			}
			if iid != uuid.Nil {
				entry.InstitutionID = &iid
			}

			// Write audit log asynchronously
			go svc.Write(context.Background(), entry)

			next.ServeHTTP(w, r)
		})
	}
}

// LogAction is a helper to log a specific domain action.
func (s *Service) LogAction(ctx context.Context, institutionID uuid.UUID, userID, action, tableName string, recordID *uuid.UUID, oldValues, newValues interface{}) {
	iid := institutionID
	entry := &Log{
		InstitutionID: &iid,
		UserID:        &userID,
		Action:        action,
		TableName:     &tableName,
		RecordID:      recordID,
		OldValues:     oldValues,
		NewValues:     newValues,
		IPAddress:     "",
		RequestID:     RequestIDFromContext(ctx),
	}
	go s.Write(context.Background(), entry)
}
