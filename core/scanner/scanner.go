package scanner

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// ScanAction specifies the enforcement action to take if content is deemed malicious/spam
type ScanAction string

const (
	ActionReject      ScanAction = "reject"      // Reject ingestion or delivery immediately
	ActionQuarantine  ScanAction = "quarantine"  // Hold in quarantine queue for administrative review
	ActionFlag        ScanAction = "flag"        // Append X-Spam-Status / X-Virus-Status headers
	ActionPassthrough ScanAction = "passthrough" // Log detection but allow transmission
)

// ScanType specifies whether this is an antivirus or antispam scan
type ScanType string

const (
	ScanTypeAntivirus ScanType = "antivirus"
	ScanTypeAntispam  ScanType = "antispam"
)

// ScanResult contains the outcome of a content inspection
type ScanResult struct {
	IsClean      bool              `json:"is_clean"`
	ScanType     ScanType          `json:"scan_type"`
	Engine       string            `json:"engine"`
	ThreatName   string            `json:"threat_name,omitempty"`
	SpamScore    float64           `json:"spam_score,omitempty"`
	ActionTaken  ScanAction        `json:"action_taken"`
	HeadersToAdd map[string]string `json:"headers_to_add,omitempty"`
	Details      string            `json:"details,omitempty"`
	Duration     time.Duration     `json:"duration"`
}

// ContentScanner defines the interface for content inspection engines
type ContentScanner interface {
	Scan(ctx context.Context, payload []byte) (*ScanResult, error)
	Name() string
}

// ScannerConfig provides settings for the content scanning engine
type ScannerConfig struct {
	Enabled        bool          `json:"enabled"`
	Engine         string        `json:"engine"` // "clamav", "rspamd", "mock"
	Address        string        `json:"address"` // e.g. "127.0.0.1:3310" or "http://127.0.0.1:11333"
	Timeout        time.Duration `json:"timeout"`
	PolicyAction   ScanAction    `json:"policy_action"` // reject, quarantine, flag, passthrough
	MaxScanSize    int64         `json:"max_scan_size"` // Max bytes to scan (e.g. 26214400 for 25MB)
	SpamThreshold  float64       `json:"spam_threshold"` // Rspamd spam score threshold (e.g. 15.0)
}

// ClamAVScanner connects to a ClamAV daemon over TCP using the INSTREAM protocol
type ClamAVScanner struct {
	addr         string
	timeout      time.Duration
	policyAction ScanAction
}

// NewClamAVScanner creates a ClamAV INSTREAM client
func NewClamAVScanner(addr string, timeout time.Duration, policyAction ScanAction) *ClamAVScanner {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	if policyAction == "" {
		policyAction = ActionReject
	}
	return &ClamAVScanner{
		addr:         addr,
		timeout:      timeout,
		policyAction: policyAction,
	}
}

func (c *ClamAVScanner) Name() string {
	return "clamav"
}

func (c *ClamAVScanner) Scan(ctx context.Context, payload []byte) (*ScanResult, error) {
	start := time.Now()

	d := net.Dialer{Timeout: c.timeout}
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClamAV daemon at %s: %w", c.addr, err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}

	// 1. Send INSTREAM command
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return nil, fmt.Errorf("failed to send INSTREAM command: %w", err)
	}

	// 2. Stream chunks of up to 64KB
	chunkSize := 64 * 1024
	buf := payload
	for len(buf) > 0 {
		n := len(buf)
		if n > chunkSize {
			n = chunkSize
		}
		chunk := buf[:n]
		buf = buf[n:]

		// Write 4-byte chunk size in network byte order (big-endian)
		var sizeBuf [4]byte
		binary.BigEndian.PutUint32(sizeBuf[:], uint32(len(chunk)))
		if _, err := conn.Write(sizeBuf[:]); err != nil {
			return nil, fmt.Errorf("failed to write chunk size: %w", err)
		}
		if _, err := conn.Write(chunk); err != nil {
			return nil, fmt.Errorf("failed to write chunk data: %w", err)
		}
	}

	// Terminate stream with 4 zero bytes
	var zeroBuf [4]byte
	if _, err := conn.Write(zeroBuf[:]); err != nil {
		return nil, fmt.Errorf("failed to terminate stream: %w", err)
	}

	// 3. Read response: e.g. "stream: OK" or "stream: Eicar-Signature FOUND"
	respBytes, err := io.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read ClamAV response: %w", err)
	}

	respStr := strings.TrimSpace(string(respBytes))
	duration := time.Since(start)

	if strings.Contains(respStr, "OK") {
		return &ScanResult{
			IsClean:     true,
			ScanType:    ScanTypeAntivirus,
			Engine:      "clamav",
			ActionTaken: ActionPassthrough,
			Details:     "Clean (ClamAV OK)",
			Duration:    duration,
		}, nil
	}

	if strings.Contains(respStr, "FOUND") {
		// Parse threat name: "stream: Eicar-Signature FOUND"
		threat := "Virus/Malware"
		parts := strings.Split(respStr, ":")
		if len(parts) >= 2 {
			foundParts := strings.Fields(parts[1])
			if len(foundParts) >= 1 {
				threat = foundParts[0]
			}
		}

		headers := map[string]string{
			"X-Virus-Status": fmt.Sprintf("Infected (%s)", threat),
		}

		return &ScanResult{
			IsClean:      false,
			ScanType:     ScanTypeAntivirus,
			Engine:       "clamav",
			ThreatName:   threat,
			ActionTaken:  c.policyAction,
			HeadersToAdd: headers,
			Details:      fmt.Sprintf("Malware detected by ClamAV: %s", threat),
			Duration:     duration,
		}, nil
	}

	return nil, fmt.Errorf("unexpected ClamAV response: %s", respStr)
}

// RspamdScanner connects to Rspamd HTTP API (/checkv2)
type RspamdScanner struct {
	baseURL       string
	timeout       time.Duration
	policyAction  ScanAction
	spamThreshold float64
	client        *http.Client
}

// NewRspamdScanner creates an Rspamd client
func NewRspamdScanner(baseURL string, timeout time.Duration, policyAction ScanAction, spamThreshold float64, customClient *http.Client) *RspamdScanner {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	if policyAction == "" {
		policyAction = ActionFlag
	}
	if spamThreshold <= 0 {
		spamThreshold = 15.0
	}
	client := customClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &RspamdScanner{
		baseURL:       strings.TrimRight(baseURL, "/"),
		timeout:       timeout,
		policyAction:  policyAction,
		spamThreshold: spamThreshold,
		client:        client,
	}
}

func (r *RspamdScanner) Name() string {
	return "rspamd"
}

func (r *RspamdScanner) Scan(ctx context.Context, payload []byte) (*ScanResult, error) {
	start := time.Now()

	url := fmt.Sprintf("%s/checkv2", r.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "message/rfc822")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rspamd check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rspamd returned status %d: %s", resp.StatusCode, string(body))
	}

	var rspamdResp struct {
		Action    string  `json:"action"`
		Score     float64 `json:"score"`
		Required  float64 `json:"required_score"`
		IsSpam    bool    `json:"is_spam"`
		Messages  []string `json:"messages,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rspamdResp); err != nil {
		return nil, fmt.Errorf("failed to decode rspamd response: %w", err)
	}

	duration := time.Since(start)
	isClean := !rspamdResp.IsSpam && rspamdResp.Score < r.spamThreshold && rspamdResp.Action != "reject"

	headers := map[string]string{
		"X-Spam-Score":  fmt.Sprintf("%.2f", rspamdResp.Score),
		"X-Spam-Action": rspamdResp.Action,
	}

	if isClean {
		headers["X-Spam-Status"] = "No"
		return &ScanResult{
			IsClean:      true,
			ScanType:     ScanTypeAntispam,
			Engine:       "rspamd",
			SpamScore:    rspamdResp.Score,
			ActionTaken:  ActionPassthrough,
			HeadersToAdd: headers,
			Details:      fmt.Sprintf("Clean (Score: %.2f)", rspamdResp.Score),
			Duration:     duration,
		}, nil
	}

	headers["X-Spam-Status"] = "Yes"
	return &ScanResult{
		IsClean:      false,
		ScanType:     ScanTypeAntispam,
		Engine:       "rspamd",
		ThreatName:   "Spam",
		SpamScore:    rspamdResp.Score,
		ActionTaken:  r.policyAction,
		HeadersToAdd: headers,
		Details:      fmt.Sprintf("Spam detected by Rspamd: score %.2f exceeds threshold", rspamdResp.Score),
		Duration:     duration,
	}, nil
}

// MockScanner is a deterministic scanner for unit testing and local development
type MockScanner struct {
	name         string
	policyAction ScanAction
	malwareMatch string
	spamMatch    string
}

// NewMockScanner creates a mock content scanner
func NewMockScanner(policyAction ScanAction) *MockScanner {
	if policyAction == "" {
		policyAction = ActionReject
	}
	return &MockScanner{
		name:         "mock",
		policyAction: policyAction,
		malwareMatch: "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*",
		spamMatch:    "SPAM_TEST_TRIGGER",
	}
}

func (m *MockScanner) Name() string {
	return m.name
}

func (m *MockScanner) Scan(ctx context.Context, payload []byte) (*ScanResult, error) {
	start := time.Now()
	payloadStr := string(payload)

	if strings.Contains(payloadStr, m.malwareMatch) {
		return &ScanResult{
			IsClean:      false,
			ScanType:     ScanTypeAntivirus,
			Engine:       "mock-antivirus",
			ThreatName:   "Eicar-Test-Signature",
			ActionTaken:  m.policyAction,
			HeadersToAdd: map[string]string{"X-Virus-Status": "Infected (Eicar-Test-Signature)"},
			Details:      "EICAR test malware signature matched",
			Duration:     time.Since(start),
		}, nil
	}

	if strings.Contains(payloadStr, m.spamMatch) {
		return &ScanResult{
			IsClean:      false,
			ScanType:     ScanTypeAntispam,
			Engine:       "mock-antispam",
			ThreatName:   "Spam-Pattern",
			SpamScore:    99.9,
			ActionTaken:  m.policyAction,
			HeadersToAdd: map[string]string{"X-Spam-Status": "Yes", "X-Spam-Score": "99.9"},
			Details:      "Spam test trigger matched",
			Duration:     time.Since(start),
		}, nil
	}

	return &ScanResult{
		IsClean:      true,
		ScanType:     ScanTypeAntivirus,
		Engine:       "mock",
		ActionTaken:  ActionPassthrough,
		Details:      "Clean (No threat patterns detected)",
		Duration:     time.Since(start),
	}, nil
}

// Pipeline executes multiple scanners sequentially
type Pipeline struct {
	scanners []ContentScanner
}

// NewPipeline creates a chain of content scanners
func NewPipeline(scanners ...ContentScanner) *Pipeline {
	return &Pipeline{scanners: scanners}
}

// ScanContent runs all scanners; if any scanner flags malicious content, returns the offending result
func (p *Pipeline) ScanContent(ctx context.Context, payload []byte) (*ScanResult, error) {
	for _, s := range p.scanners {
		res, err := s.Scan(ctx, payload)
		if err != nil {
			return nil, fmt.Errorf("scanner %s failed: %w", s.Name(), err)
		}
		if !res.IsClean {
			return res, nil
		}
	}

	return &ScanResult{
		IsClean:     true,
		Engine:      "pipeline",
		ActionTaken: ActionPassthrough,
		Details:     "All scanners passed clean",
	}, nil
}

// EvaluatePolicy checks if the scan result permits outbound delivery or inbound acceptance
func EvaluatePolicy(res *ScanResult) error {
	if res.IsClean {
		return nil
	}
	switch res.ActionTaken {
	case ActionReject:
		return fmt.Errorf("content rejected by %s: %s", res.Engine, res.Details)
	case ActionQuarantine:
		return errors.New("content quarantined due to security policy")
	case ActionFlag, ActionPassthrough:
		return nil // Permitted with headers or warning
	default:
		return fmt.Errorf("content rejected by policy: %s", res.Details)
	}
}
