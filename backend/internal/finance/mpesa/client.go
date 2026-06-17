package mpesa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"institution-os/pkg/config"
	"github.com/rs/zerolog/log"
)

const (
	sandboxBaseURL    = "https://sandbox.safaricom.co.ke"
	productionBaseURL = "https://api.safaricom.co.ke"
)

// Client is the M-Pesa Daraja API client.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Daraja client.
func NewClient(cfg *config.Config) *Client {
	baseURL := sandboxBaseURL
	if cfg.MpesaEnvironment == "production" {
		baseURL = productionBaseURL
	}
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}
}

// tokenResponse is the OAuth token response.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

// GetAccessToken fetches a Daraja OAuth access token.
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
	creds := base64.StdEncoding.EncodeToString(
		[]byte(c.cfg.MpesaConsumerKey + ":" + c.cfg.MpesaConsumerSecret),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/oauth/v1/generate?grant_type=client_credentials", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+creds)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("daraja auth: %w", err)
	}
	defer resp.Body.Close()

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("daraja auth decode: %w", err)
	}

	return tr.AccessToken, nil
}

// STKPushRequest is the Daraja STK Push API request body.
type STKPushRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            int64  `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

// STKPushResponse is the Daraja STK Push API response.
type STKPushResponse struct {
	MerchantRequestID string `json:"MerchantRequestID"`
	CheckoutRequestID string `json:"CheckoutRequestID"`
	ResponseCode      string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage   string `json:"CustomerMessage"`
}

// InitiateSTKPush sends an STK push request to the Daraja API.
// In sandbox/stub mode with empty credentials, this logs and returns a fake response.
func (c *Client) InitiateSTKPush(ctx context.Context, phone, accountRef string, amount int64) (*STKPushResponse, error) {
	// If no credentials configured, return a stub response
	if c.cfg.MpesaConsumerKey == "" {
		log.Info().
			Str("phone", phone).
			Str("account_ref", accountRef).
			Int64("amount", amount).
			Msg("MPESA_STUB: would send STK push")
		return &STKPushResponse{
			MerchantRequestID: "stub-merchant-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			CheckoutRequestID: "stub-checkout-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			ResponseCode:      "0",
			CustomerMessage:   "Success. Request accepted for processing",
		}, nil
	}

	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format("20060102150405")
	password := base64.StdEncoding.EncodeToString(
		[]byte(c.cfg.MpesaShortcode + c.cfg.MpesaPasskey + timestamp),
	)

	payload := STKPushRequest{
		BusinessShortCode: c.cfg.MpesaShortcode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            amount,
		PartyA:            phone,
		PartyB:            c.cfg.MpesaShortcode,
		PhoneNumber:       phone,
		CallBackURL:       c.cfg.MpesaCallbackURL,
		AccountReference:  accountRef,
		TransactionDesc:   "School fee payment",
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/mpesa/stkpush/v1/processrequest",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stk push request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daraja STK push failed: %s — %s", resp.Status, string(respBody))
	}

	var stkResp STKPushResponse
	if err := json.Unmarshal(respBody, &stkResp); err != nil {
		return nil, fmt.Errorf("decode STK push response: %w", err)
	}

	return &stkResp, nil
}
