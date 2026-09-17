package mpesa

// Wire-format tests for the Daraja client, run against httptest servers
// instead of Safaricom. They pin the exact HTTP contract:
//
//   - OAuth: GET /oauth/v1/generate?grant_type=client_credentials with a
//     Basic Authorization header built as base64(key:secret)
//   - STK push: POST /mpesa/stkpush/v1/processrequest with Bearer auth,
//     JSON body field names Daraja actually expects (PascalCase), and the
//     Password = base64(shortcode + passkey + timestamp) construction
//   - error handling: 4xx/5xx from either endpoint must fail loudly —
//     auth used to return an empty token on a 401, which sent the STK
//     push out with an empty Bearer and failed confusingly later
//   - stub mode: no HTTP at all when a channel has no consumer key
//
// White-box on purpose: Client.baseURL is package-private, so tests
// construct &Client{baseURL: server.URL} directly rather than forcing a
// config knob into production code.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"institution-os/pkg/config"
)

const (
	testKey     = "test-consumer-key"
	testSecret  = "test-consumer-secret"
	testShort   = "174379"
	testPasskey = "bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada05671900"
	testToken   = "daraja-access-token-123"
)

// newTestClient builds a client pointed at an httptest server running h.
func newTestClient(t *testing.T, h http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Client{
		cfg:        &config.Config{},
		httpClient: srv.Client(),
		baseURL:    srv.URL,
	}, srv
}

// stubOAuth mounts a correct OAuth endpoint on the given mux and counts hits.
func stubOAuth(t *testing.T, mux *http.ServeMux, hits *int) {
	mux.HandleFunc("/oauth/v1/generate", func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			*hits++
		}
		if r.Method != http.MethodGet {
			t.Errorf("OAuth: want GET, got %s", r.Method)
		}
		if r.URL.RawQuery != "grant_type=client_credentials" {
			t.Errorf("OAuth: want grant_type=client_credentials, got %q", r.URL.RawQuery)
		}
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(testKey+":"+testSecret))
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Errorf("OAuth: Authorization = %q, want %q", got, wantAuth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"` + testToken + `","expires_in":"3599"}`))
	})
}

// ─── GetAccessToken ───────────────────────────────────────────────────

func TestGetAccessToken_WireFormat(t *testing.T) {
	mux := http.NewServeMux()
	stubOAuth(t, mux, nil)
	client, _ := newTestClient(t, mux)

	token, err := client.GetAccessToken(context.Background(), testKey, testSecret)
	require.NoError(t, err)
	assert.Equal(t, testToken, token)
}

func TestGetAccessToken_ErrorStatus_FailsLoudly(t *testing.T) {
	// Regression pin: a 400 used to decode its JSON error body into
	// tokenResponse and return "" with no error.
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/v1/generate", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"Invalid authentication credentials"}`))
	})
	client, _ := newTestClient(t, mux)

	token, err := client.GetAccessToken(context.Background(), "bad", "creds")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "invalid_client")
	assert.Empty(t, token)
}

func TestGetAccessToken_EmptyToken_IsRejected(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/v1/generate", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	client, _ := newTestClient(t, mux)

	_, err := client.GetAccessToken(context.Background(), testKey, testSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty access token")
}

// ─── InitiateSTKPush: live credentials ────────────────────────────────

func TestInitiateSTKPush_WireFormat(t *testing.T) {
	mux := http.NewServeMux()
	oauthHits := 0
	stkHits := 0
	stubOAuth(t, mux, &oauthHits)

	mux.HandleFunc("/mpesa/stkpush/v1/processrequest", func(w http.ResponseWriter, r *http.Request) {
		stkHits++
		if r.Method != http.MethodPost {
			t.Errorf("STK: want POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Errorf("STK: Authorization = %q, want Bearer token from the OAuth call", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("STK: Content-Type = %q", got)
		}

		var req STKPushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("STK: decode body: %v", err)
		}

		// Field names and values Daraja's contract actually requires.
		assert.Equal(t, testShort, req.BusinessShortCode, "BusinessShortCode")
		assert.Equal(t, testShort, req.PartyB, "PartyB = the till/paybill receiving the money")
		assert.Equal(t, "254722100201", req.PartyA, "PartyA")
		assert.Equal(t, "254722100201", req.PhoneNumber, "PhoneNumber")
		assert.Equal(t, int64(2500), req.Amount)
		assert.Equal(t, "CustomerPayBillOnline", req.TransactionType)
		assert.Equal(t, "https://school.example/callback", req.CallBackURL)
		assert.Equal(t, "Amina Wanjiru", req.AccountReference)
		assert.Equal(t, "School fee payment", req.TransactionDesc)

		// Timestamp must be Daraja's YYYYMMDDHHmmss format (a local
		// wall-clock string with no zone — the Daraja contract), and the
		// Password must be base64(shortcode + passkey + that same
		// timestamp). We recompute from the received timestamp rather
		// than freezing one.
		ts, err := time.ParseInLocation("20060102150405", req.Timestamp, time.Local)
		if err != nil {
			t.Fatalf("STK: Timestamp %q is not in Daraja YYYYMMDDHHmmss format: %v", req.Timestamp, err)
		}
		assert.WithinDuration(t, time.Now(), ts, 2*time.Minute, "Timestamp should be ~now")
		wantPassword := base64.StdEncoding.EncodeToString([]byte(testShort + testPasskey + req.Timestamp))
		assert.Equal(t, wantPassword, req.Password, "Password = base64(shortcode+passkey+timestamp)")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"MerchantRequestID": "29115-34620561-1",
			"CheckoutRequestID": "ws_CO_191220191020363925",
			"ResponseCode": "0",
			"ResponseDescription": "Success. Request accepted for processing",
			"CustomerMessage": "Success. Request accepted for processing"
		}`))
	})
	client, _ := newTestClient(t, mux)

	resp, err := client.InitiateSTKPush(context.Background(),
		ChannelCredentials{ConsumerKey: testKey, ConsumerSecret: testSecret, Shortcode: testShort, Passkey: testPasskey},
		"https://school.example/callback", "254722100201", "Amina Wanjiru", 2500)
	require.NoError(t, err)

	assert.Equal(t, "29115-34620561-1", resp.MerchantRequestID)
	assert.Equal(t, "ws_CO_191220191020363925", resp.CheckoutRequestID)
	assert.Equal(t, "0", resp.ResponseCode)
	assert.Equal(t, "Success. Request accepted for processing", resp.CustomerMessage)
	assert.Equal(t, 1, oauthHits, "exactly one OAuth call per push")
	assert.Equal(t, 1, stkHits)
}

func TestInitiateSTKPush_ErrorStatus_FailsWithBody(t *testing.T) {
	mux := http.NewServeMux()
	stubOAuth(t, mux, nil)
	mux.HandleFunc("/mpesa/stkpush/v1/processrequest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errorMessage":"Unable to lock subscriber, a transaction is already in process"}`))
	})
	client, _ := newTestClient(t, mux)

	_, err := client.InitiateSTKPush(context.Background(),
		ChannelCredentials{ConsumerKey: testKey, ConsumerSecret: testSecret, Shortcode: testShort, Passkey: testPasskey},
		"https://school.example/callback", "254722100201", "Amina", 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "Unable to lock subscriber")
}

func TestInitiateSTKPush_OAuthFailure_NeverReachesSTK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/v1/generate", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	})
	stkCalled := false
	mux.HandleFunc("/mpesa/stkpush/v1/processrequest", func(w http.ResponseWriter, r *http.Request) {
		stkCalled = true
		_, _ = w.Write([]byte(`{}`))
	})
	client, _ := newTestClient(t, mux)

	_, err := client.InitiateSTKPush(context.Background(),
		ChannelCredentials{ConsumerKey: testKey, ConsumerSecret: testSecret, Shortcode: testShort, Passkey: testPasskey},
		"https://school.example/callback", "254722100201", "Amina", 100)
	require.Error(t, err)
	assert.False(t, stkCalled, "a failed auth must abort before the STK request")
}

// ─── InitiateSTKPush: stub mode (no live credentials) ─────────────────

func TestInitiateSTKPush_StubMode_MakesNoHTTPCalls(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("stub mode must not make HTTP calls, got %s %s", r.Method, r.URL.Path)
	})
	client, _ := newTestClient(t, mux)

	resp, err := client.InitiateSTKPush(context.Background(),
		ChannelCredentials{Shortcode: testShort}, // no ConsumerKey → stub
		"https://school.example/callback", "254722100201", "Amina", 2500)
	require.NoError(t, err)
	assert.Equal(t, "0", resp.ResponseCode)
	assert.NotEmpty(t, resp.CheckoutRequestID)
	assert.NotEmpty(t, resp.MerchantRequestID)

	// Stub IDs must be unique across calls (crypto-rand, P0-3) — the old
	// unixnano-based ones were predictable.
	resp2, err := client.InitiateSTKPush(context.Background(),
		ChannelCredentials{Shortcode: testShort},
		"https://school.example/callback", "254722100201", "Amina", 2500)
	require.NoError(t, err)
	assert.NotEqual(t, resp.CheckoutRequestID, resp2.CheckoutRequestID)
	assert.NotEqual(t, resp.MerchantRequestID, resp2.MerchantRequestID)
}

// ─── environment → base URL selection ─────────────────────────────────

func TestNewClient_EnvironmentSelection(t *testing.T) {
	assert.Equal(t, sandboxBaseURL, NewClient(&config.Config{MpesaEnvironment: "sandbox"}).baseURL)
	assert.Equal(t, productionBaseURL, NewClient(&config.Config{MpesaEnvironment: "production"}).baseURL)
	assert.Equal(t, sandboxBaseURL, NewClient(&config.Config{}).baseURL, "unset environment defaults to sandbox, never production")
}
