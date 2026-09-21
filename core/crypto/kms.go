package crypto

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// AWSKMSConfig contains settings for connecting to AWS Key Management Service (KMS)
type AWSKMSConfig struct {
	KeyID      string        // KMS Key ID, ARN, or Alias (e.g. "arn:aws:kms:us-east-1:123456789012:key/...")
	Region     string        // AWS Region (e.g. "us-east-1")
	Endpoint   string        // Optional custom endpoint for LocalStack or VPC endpoint
	AccessKey  string        // Optional static credentials (defaults to IAM role if empty)
	SecretKey  string        // Optional static credentials
	Timeout    time.Duration // Request timeout
}

// AWSKMSKEKProvider implements KEKProvider via AWS KMS Encrypt / Decrypt API
type AWSKMSKEKProvider struct {
	cfg    AWSKMSConfig
	client *http.Client
}

// NewAWSKMSKEKProvider creates an AWS KMS KEK provider
func NewAWSKMSKEKProvider(cfg AWSKMSConfig, customClient *http.Client) (*AWSKMSKEKProvider, error) {
	if cfg.KeyID == "" {
		return nil, errors.New("aws kms key id is required")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	client := customClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &AWSKMSKEKProvider{cfg: cfg, client: client}, nil
}

func (k *AWSKMSKEKProvider) KeyID() string {
	return fmt.Sprintf("awskms:%s", k.cfg.KeyID)
}

func (k *AWSKMSKEKProvider) WrapDEK(ctx context.Context, dek []byte) (string, error) {
	plaintextB64 := base64.StdEncoding.EncodeToString(dek)
	payload, err := json.Marshal(map[string]interface{}{
		"KeyId":     k.cfg.KeyID,
		"Plaintext": plaintextB64,
	})
	if err != nil {
		return "", err
	}

	endpoint := k.cfg.Endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://kms.%s.amazonaws.com", k.cfg.Region)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "TrentService.Encrypt")

	resp, err := k.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("aws kms encrypt request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("aws kms encrypt failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	var kmsResp struct {
		CiphertextBlob string `json:"CiphertextBlob"`
		KeyId          string `json:"KeyId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&kmsResp); err != nil {
		return "", fmt.Errorf("failed to decode aws kms encrypt response: %w", err)
	}
	if kmsResp.CiphertextBlob == "" {
		return "", errors.New("aws kms returned empty CiphertextBlob")
	}

	return kmsResp.CiphertextBlob, nil
}

func (k *AWSKMSKEKProvider) UnwrapDEK(ctx context.Context, wrappedB64 string) ([]byte, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"KeyId":          k.cfg.KeyID,
		"CiphertextBlob": wrappedB64,
	})
	if err != nil {
		return nil, err
	}

	endpoint := k.cfg.Endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://kms.%s.amazonaws.com", k.cfg.Region)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "TrentService.Decrypt")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aws kms decrypt request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("aws kms decrypt failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	var kmsResp struct {
		Plaintext string `json:"Plaintext"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&kmsResp); err != nil {
		return nil, fmt.Errorf("failed to decode aws kms decrypt response: %w", err)
	}
	if kmsResp.Plaintext == "" {
		return nil, errors.New("aws kms returned empty Plaintext")
	}

	rawDEK, err := base64.StdEncoding.DecodeString(kmsResp.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode plaintext from aws kms: %w", err)
	}
	return rawDEK, nil
}

// AWSKMSEnvProvider returns an AWS KMS KEK provider if AWS_KMS_KEY_ID is configured
func AWSKMSEnvProvider() (KEKProvider, error) {
	keyID := strings.TrimSpace(osGetenv("AWS_KMS_KEY_ID", "MVX_AWS_KMS_KEY_ID"))
	if keyID == "" {
		return nil, errors.New("AWS_KMS_KEY_ID environment variable is not configured")
	}
	region := osGetenv("AWS_REGION", "AWS_DEFAULT_REGION")
	if region == "" {
		region = "us-east-1"
	}
	endpoint := osGetenv("AWS_KMS_ENDPOINT", "MVX_AWS_KMS_ENDPOINT")

	return NewAWSKMSKEKProvider(AWSKMSConfig{
		KeyID:    keyID,
		Region:   region,
		Endpoint: endpoint,
	}, nil)
}

func osGetenv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(osGetenvVal(k)); v != "" {
			return v
		}
	}
	return ""
}

func osGetenvVal(k string) string {
	// Wrapper to allow mocking in tests
	return osGetenvFunc(k)
}

var osGetenvFunc = func(key string) string {
	return os.Getenv(key)
}
