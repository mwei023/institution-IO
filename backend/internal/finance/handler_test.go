package finance

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"institution-os/pkg/events"
)

// The callback secret gate accepts the secret from EITHER the
// X-Callback-Secret header OR a ?callback_secret= query param — the
// query path exists because stock Daraja sends no custom headers and
// operators embed the secret in the registered callback URL. These
// tests pin both acceptance paths and both rejection paths.

func postCallback(target string) *http.Response {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	NewHandlerWithCallbackSecret(nil, "s3cret-value").MpesaCallback(w, req)
	return w.Result()
}

func TestMpesaCallback_SecretViaHeader_Accepted(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/mpesa/callback", strings.NewReader(`{}`))
	req.Header.Set("X-Callback-Secret", "s3cret-value")
	w := httptest.NewRecorder()
	NewHandlerWithCallbackSecret(nil, "s3cret-value").MpesaCallback(w, req)
	require.Equal(t, http.StatusOK, w.Result().StatusCode) // 200: reached the handler ({} payload is a no-op)
}

func TestMpesaCallback_SecretViaQuery_Accepted(t *testing.T) {
	// The Daraja reality: the secret arrives embedded in the registered
	// URL, never as a header.
	resp := postCallback("/api/v1/finance/mpesa/callback?callback_secret=s3cret-value")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMpesaCallback_BadSecret_Rejected(t *testing.T) {
	resp := postCallback("/api/v1/finance/mpesa/callback?callback_secret=wrong")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMpesaCallback_MissingSecret_Rejected(t *testing.T) {
	resp := postCallback("/api/v1/finance/mpesa/callback")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMpesaCallback_NoSecretConfigured_AcceptsForSandbox(t *testing.T) {
	repo := new(MockStore)
	svc := NewService(repo, nil, &events.NoopPublisher{}, "https://cb.example")
	handler := NewHandler(svc) // no callbackSecret wired (sandbox onboarding mode)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/mpesa/callback", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	handler.MpesaCallback(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}
