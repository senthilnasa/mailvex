package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

var (
	ErrReplayAttack   = errors.New("webhook timestamp outside tolerance window")
	ErrInvalidHeader  = errors.New("malformed webhook signature header")
	ErrSignatureMismatch = errors.New("webhook signature mismatch")
)

type WebhookPayload struct {
	ID         string    `json:"id"`
	Event      string    `json:"event"`
	TenantID   string    `json:"tenant_id"`
	OccurredAt time.Time `json:"occurred_at"`
	Data       any       `json:"data"`
}

type Dispatcher struct {
	repo       repository.Repository
	httpClient *http.Client
	tolerance  time.Duration
}

func NewDispatcher(repo repository.Repository, client *http.Client) *Dispatcher {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return &Dispatcher{
		repo:       repo,
		httpClient: client,
		tolerance:  300 * time.Second, // 5 minutes tolerance (§1.1)
	}
}

// GenerateSecret generates a secure cryptographic webhook secret
func GenerateSecret() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "whsec_" + hex.EncodeToString(b)
}

// ComputeSignature calculates HMAC-SHA256 signature over "<timestamp>.<payload>" (§1.1)
func ComputeSignature(secret string, timestamp int64, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// FormatSignatureHeader builds "t=<unix>,v1=<sig1>[,v1=<sig2>]" with 24h secret rotation overlap (§1.2)
func FormatSignatureHeader(timestamp int64, primarySecret string, prevSecret *string, prevExpiresAt *time.Time, payload []byte) string {
	sig1 := ComputeSignature(primarySecret, timestamp, payload)
	header := fmt.Sprintf("t=%d,v1=%s", timestamp, sig1)

	// If rotating secret with 24-hour overlap window, include previous secret signature (§1.2)
	if prevSecret != nil && *prevSecret != "" && prevExpiresAt != nil {
		if time.Now().UTC().Before(*prevExpiresAt) {
			sig2 := ComputeSignature(*prevSecret, timestamp, payload)
			header += fmt.Sprintf(",v1=%s", sig2)
		}
	}
	return header
}

// VerifySignature validates signature header with timestamp tolerance replay defense (§1.1)
func VerifySignature(payload []byte, header string, secrets []string, tolerance time.Duration) (bool, error) {
	if tolerance <= 0 {
		tolerance = 300 * time.Second
	}

	parts := strings.Split(header, ",")
	var timestamp int64
	var signatures []string

	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := kv[0], kv[1]
		if k == "t" {
			t, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return false, ErrInvalidHeader
			}
			timestamp = t
		} else if k == "v1" {
			signatures = append(signatures, v)
		}
	}

	if timestamp == 0 || len(signatures) == 0 {
		return false, ErrInvalidHeader
	}

	// Verify timestamp freshness (|now - t| <= tolerance)
	nowUnix := time.Now().UTC().Unix()
	if math.Abs(float64(nowUnix-timestamp)) > tolerance.Seconds() {
		return false, ErrReplayAttack
	}

	// Verify matching signature for any provided valid secret
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		expectedSig := ComputeSignature(secret, timestamp, payload)
		for _, sig := range signatures {
			if hmac.Equal([]byte(sig), []byte(expectedSig)) {
				return true, nil
			}
		}
	}

	return false, ErrSignatureMismatch
}

// DispatchEvent delivers an event to all subscribed webhook endpoints for the tenant
func (d *Dispatcher) DispatchEvent(ctx context.Context, tenantID, eventID, eventType string, data any) error {
	endpoints, err := d.repo.GetActiveWebhookEndpointsForEvent(ctx, tenantID, eventType)
	if err != nil {
		return err
	}
	if len(endpoints) == 0 {
		return nil
	}

	payloadObj := WebhookPayload{
		ID:         eventID,
		Event:      eventType,
		TenantID:   tenantID,
		OccurredAt: time.Now().UTC(),
		Data:       data,
	}

	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	for _, ep := range endpoints {
		_, _ = d.deliverToEndpoint(ctx, ep, eventID, eventType, payloadBytes)
	}

	return nil
}

// DispatchPing sends a test ping event to a specific endpoint
func (d *Dispatcher) DispatchPing(ctx context.Context, tenantID, endpointID string) (*domain.WebhookDelivery, error) {
	ep, err := d.repo.GetWebhookEndpoint(ctx, tenantID, endpointID)
	if err != nil {
		return nil, err
	}

	eventID := crypto.NewULID(crypto.PrefixEvent)
	payloadObj := WebhookPayload{
		ID:         eventID,
		Event:      "webhook.test",
		TenantID:   tenantID,
		OccurredAt: time.Now().UTC(),
		Data: map[string]any{
			"message": "Mailvex webhook test event ping",
			"endpoint_id": endpointID,
		},
	}

	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return nil, err
	}

	return d.deliverToEndpoint(ctx, ep, eventID, "webhook.test", payloadBytes)
}

func (d *Dispatcher) deliverToEndpoint(ctx context.Context, ep *domain.WebhookEndpoint, eventID, eventType string, payloadBytes []byte) (*domain.WebhookDelivery, error) {
	deliveryID := crypto.NewULID("whd_")
	now := time.Now().UTC()
	timestamp := now.Unix()

	sigHeader := FormatSignatureHeader(timestamp, ep.Secret, ep.PreviousSecret, ep.PreviousSecretExpiresAt, payloadBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		errMsg := err.Error()
		del := &domain.WebhookDelivery{
			ID:           deliveryID,
			TenantID:     ep.TenantID,
			EndpointID:   ep.ID,
			EventID:      eventID,
			EventType:    eventType,
			Payload:      string(payloadBytes),
			ErrorMessage: &errMsg,
			Attempt:      1,
			DurationMS:   0,
			CreatedAt:    now,
		}
		_ = d.repo.RecordWebhookDelivery(ctx, del)
		return del, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mailvex-Webhook/1.0")
	req.Header.Set("X-Mailvex-Delivery-Id", deliveryID)
	req.Header.Set("X-Mailvex-Event-Id", eventID)
	req.Header.Set("X-Mailvex-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Mailvex-Signature", sigHeader)

	start := time.Now()
	resp, err := d.httpClient.Do(req)
	duration := int(time.Since(start).Milliseconds())

	del := &domain.WebhookDelivery{
		ID:         deliveryID,
		TenantID:   ep.TenantID,
		EndpointID: ep.ID,
		EventID:    eventID,
		EventType:  eventType,
		Payload:    string(payloadBytes),
		Attempt:    1,
		DurationMS: duration,
		CreatedAt:  now,
	}

	if err != nil {
		errMsg := err.Error()
		del.ErrorMessage = &errMsg
	} else {
		defer resp.Body.Close()
		del.StatusCode = &resp.StatusCode
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		bodyStr := string(bodyBytes)
		del.ResponseBody = &bodyStr
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errStr := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
			del.ErrorMessage = &errStr
		}
	}

	_ = d.repo.RecordWebhookDelivery(ctx, del)
	return del, nil
}
