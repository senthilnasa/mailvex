package mailvex

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the default Mailvex API endpoint
const DefaultBaseURL = "http://localhost:8080"

// Client is the official Go SDK client for Mailvex
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int
}

// Option configures a Client
type Option func(*Client)

func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithMaxRetries(retries int) Option {
	return func(c *Client) {
		c.maxRetries = retries
	}
}

// NewClient creates a new Mailvex Go SDK client
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		maxRetries: 3,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Models
type EmailRecipient struct {
	Email string            `json:"email"`
	Name  string            `json:"name,omitempty"`
	Vars  map[string]interface{} `json:"vars,omitempty"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	ContentB64  string `json:"content_base64"`
}

type SendEmailRequest struct {
	From           string            `json:"from"`
	To             []string          `json:"to"`
	Subject        string            `json:"subject"`
	HTML           string            `json:"html,omitempty"`
	Text           string            `json:"text,omitempty"`
	TemplateID     string            `json:"template_id,omitempty"`
	TemplateVars   map[string]interface{} `json:"template_vars,omitempty"`
	StreamID       string            `json:"stream_id,omitempty"`
	ReplyTo        string            `json:"reply_to,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Attachments    []Attachment      `json:"attachments,omitempty"`
	IdempotencyKey string            `json:"-"`
}

type SendEmailResponse struct {
	MessageID string   `json:"message_id"`
	Status    string   `json:"status"`
	Accepted  []string `json:"accepted"`
	Rejected  []string `json:"rejected,omitempty"`
}

type BatchRecipient struct {
	To   string                 `json:"to"`
	Vars map[string]interface{} `json:"vars,omitempty"`
}

type SendBatchRequest struct {
	From           string            `json:"from"`
	Subject        string            `json:"subject"`
	HTML           string            `json:"html,omitempty"`
	Text           string            `json:"text,omitempty"`
	TemplateID     string            `json:"template_id,omitempty"`
	Recipients     []BatchRecipient  `json:"recipients"`
	StreamID       string            `json:"stream_id,omitempty"`
	IdempotencyKey string            `json:"-"`
}

type SendBatchResponse struct {
	BatchID string   `json:"batch_id"`
	Total   int      `json:"total"`
	MessageIDs []string `json:"message_ids"`
}

type Message struct {
	ID          string    `json:"id"`
	FromAddress string    `json:"from_address"`
	Subject     string    `json:"subject"`
	Status      string    `json:"status"`
	StreamID    string    `json:"stream_id"`
	Environment string    `json:"environment"`
	CreatedAt   time.Time `json:"created_at"`
}

type MessageEvent struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	Type      string    `json:"type"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

type Domain struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	OwnershipToken string    `json:"ownership_token"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateDomainRequest struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"` // "public" or "internal"
}

type DomainVerificationResult struct {
	DomainID  string `json:"domain_id"`
	Status    string `json:"status"`
	Verified  bool   `json:"verified"`
}

type Suppression struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Type      string    `json:"type"`
	Reason    string    `json:"reason,omitempty"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

type AddSuppressionRequest struct {
	Address string `json:"address"`
	Type    string `json:"type,omitempty"` // "bounce", "complaint", "manual", "unsubscribe"
	Reason  string `json:"reason,omitempty"`
}

// API Methods

func (c *Client) Send(ctx context.Context, req *SendEmailRequest) (*SendEmailResponse, error) {
	headers := make(map[string]string)
	if req.IdempotencyKey != "" {
		headers["X-Idempotency-Key"] = req.IdempotencyKey
	}
	var resp SendEmailResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/send", req, &resp, headers); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SendBatch(ctx context.Context, req *SendBatchRequest) (*SendBatchResponse, error) {
	headers := make(map[string]string)
	if req.IdempotencyKey != "" {
		headers["X-Idempotency-Key"] = req.IdempotencyKey
	}
	var resp SendBatchResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/send/batch", req, &resp, headers); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	var resp Message
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/messages/%s", messageID), nil, &resp, nil); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetMessageEvents(ctx context.Context, messageID string) ([]*MessageEvent, error) {
	var resp struct {
		Data []*MessageEvent `json:"data"`
	}
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/messages/%s/events", messageID), nil, &resp, nil); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) CreateDomain(ctx context.Context, req *CreateDomainRequest) (*Domain, error) {
	var resp Domain
	if err := c.doRequest(ctx, http.MethodPost, "/v1/domains", req, &resp, nil); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) VerifyDomain(ctx context.Context, domainID string) (*DomainVerificationResult, error) {
	var resp DomainVerificationResult
	if err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/domains/%s/verify", domainID), nil, &resp, nil); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListSuppressions(ctx context.Context, limit int) ([]*Suppression, error) {
	if limit <= 0 {
		limit = 100
	}
	var resp struct {
		Data []*Suppression `json:"data"`
	}
	url := fmt.Sprintf("/v1/suppressions?limit=%d", limit)
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &resp, nil); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) AddSuppression(ctx context.Context, req *AddSuppressionRequest) (*Suppression, error) {
	var resp Suppression
	if err := c.doRequest(ctx, http.MethodPost, "/v1/suppressions", req, &resp, nil); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) DeleteSuppression(ctx context.Context, suppressionID string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/v1/suppressions/%s", suppressionID), nil, nil, nil)
}

// Webhook Verification Helper (§18)
func VerifyWebhookSignature(payload []byte, signatureHeader string, secret string) bool {
	// Header format: t=<timestamp>,v1=<signature>
	var timestamp, expectedSig string
	parts := strings.Split(signatureHeader, ",")
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 {
			if kv[0] == "t" {
				timestamp = kv[1]
			} else if kv[0] == "v1" {
				expectedSig = kv[1]
			}
		}
	}
	if timestamp == "" || expectedSig == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	computed := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(computed), []byte(expectedSig))
}

// HTTP Helper with retry and exponential backoff
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, out interface{}, extraHeaders map[string]string) error {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	url := c.baseURL + path
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			backoff := time.Duration(1<<attempt)*100*time.Millisecond + time.Duration(rand.Intn(50))*time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mailvex-Go-SDK/1.0.0")
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Check status code
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server returned retryable status %d", resp.StatusCode)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			errBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(errBytes))
		}

		if out != nil {
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}
		}

		return nil
	}

	if lastErr != nil {
		return fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
	}
	return errors.New("request failed")
}
