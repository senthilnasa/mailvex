package scanner

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMockScanner(t *testing.T) {
	ctx := context.Background()
	scanner := NewMockScanner(ActionReject)

	// Clean payload
	cleanRes, err := scanner.Scan(ctx, []byte("Hello, this is a clean transactional email."))
	if err != nil {
		t.Fatalf("unexpected error scanning clean payload: %v", err)
	}
	if !cleanRes.IsClean {
		t.Errorf("expected clean, got unclean: %s", cleanRes.Details)
	}

	// Malware payload (EICAR)
	eicar := "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
	malwareRes, err := scanner.Scan(ctx, []byte("Attached file:\n"+eicar))
	if err != nil {
		t.Fatalf("unexpected error scanning malware: %v", err)
	}
	if malwareRes.IsClean {
		t.Error("expected malware to be detected, but scanner marked clean")
	}
	if malwareRes.ThreatName != "Eicar-Test-Signature" {
		t.Errorf("expected threat name Eicar-Test-Signature, got %s", malwareRes.ThreatName)
	}
	if malwareRes.ActionTaken != ActionReject {
		t.Errorf("expected ActionReject, got %s", malwareRes.ActionTaken)
	}

	// Spam payload
	spamRes, err := scanner.Scan(ctx, []byte("Check this out: SPAM_TEST_TRIGGER"))
	if err != nil {
		t.Fatalf("unexpected error scanning spam: %v", err)
	}
	if spamRes.IsClean {
		t.Error("expected spam to be detected")
	}
	if spamRes.HeadersToAdd["X-Spam-Status"] != "Yes" {
		t.Errorf("expected X-Spam-Status: Yes, got %s", spamRes.HeadersToAdd["X-Spam-Status"])
	}
}

func TestClamAVScannerWithMockTCP(t *testing.T) {
	ctx := context.Background()

	// Mock ClamAV TCP Server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock ClamAV listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				// Read command "zINSTREAM\x00"
				cmdBuf := make([]byte, 10)
				n, _ := c.Read(cmdBuf)
				if n < 8 || !strings.Contains(string(cmdBuf[:n]), "INSTREAM") {
					c.Write([]byte("ERROR\n"))
					return
				}

				// Read chunks
				var totalContent strings.Builder
				for {
					var sizeBuf [4]byte
					if _, err := io.ReadFull(c, sizeBuf[:]); err != nil {
						break
					}
					chunkSize := binary.BigEndian.Uint32(sizeBuf[:])
					if chunkSize == 0 {
						break // Terminated
					}
					chunk := make([]byte, chunkSize)
					io.ReadFull(c, chunk)
					totalContent.Write(chunk)
				}

				// Determine result
				if strings.Contains(totalContent.String(), "VIRUS_PAYLOAD") {
					c.Write([]byte("stream: Win.Test.CustomVirus FOUND\x00"))
				} else {
					c.Write([]byte("stream: OK\x00"))
				}
			}(conn)
		}
	}()

	client := NewClamAVScanner(listener.Addr().String(), 2*time.Second, ActionReject)

	// Clean test
	resClean, err := client.Scan(ctx, []byte("Normal friendly message"))
	if err != nil {
		t.Fatalf("ClamAV clean scan failed: %v", err)
	}
	if !resClean.IsClean {
		t.Errorf("expected clean, got: %s", resClean.Details)
	}

	// Virus test
	resInfected, err := client.Scan(ctx, []byte("Danger: VIRUS_PAYLOAD inside"))
	if err != nil {
		t.Fatalf("ClamAV virus scan failed: %v", err)
	}
	if resInfected.IsClean {
		t.Error("expected virus detection, but got clean")
	}
	if !strings.Contains(resInfected.ThreatName, "CustomVirus") {
		t.Errorf("expected CustomVirus threat name, got %s", resInfected.ThreatName)
	}
}

func TestRspamdScannerWithMockHTTP(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checkv2" {
			http.NotFound(w, r)
			return
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		body := string(bodyBytes)

		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(body, "SPAM_CONTENT") {
			fmt.Fprintln(w, `{"is_spam": true, "score": 18.5, "action": "reject", "required_score": 15.0}`)
		} else {
			fmt.Fprintln(w, `{"is_spam": false, "score": 1.2, "action": "no action", "required_score": 15.0}`)
		}
	}))
	defer server.Close()

	rspamd := NewRspamdScanner(server.URL, 2*time.Second, ActionFlag, 15.0, server.Client())

	// Test Clean
	cleanRes, err := rspamd.Scan(ctx, []byte("From: alice@example.com\nSubject: Invoice\n\nHere is your invoice."))
	if err != nil {
		t.Fatalf("Rspamd scan failed: %v", err)
	}
	if !cleanRes.IsClean {
		t.Errorf("expected clean, got: %s", cleanRes.Details)
	}
	if cleanRes.HeadersToAdd["X-Spam-Status"] != "No" {
		t.Errorf("expected X-Spam-Status No, got %s", cleanRes.HeadersToAdd["X-Spam-Status"])
	}

	// Test Spam
	spamRes, err := rspamd.Scan(ctx, []byte("From: spammer@example.com\nSubject: WIN MONEY\n\nSPAM_CONTENT buy now!"))
	if err != nil {
		t.Fatalf("Rspamd spam scan failed: %v", err)
	}
	if spamRes.IsClean {
		t.Error("expected spam detected, got clean")
	}
	if spamRes.HeadersToAdd["X-Spam-Status"] != "Yes" {
		t.Errorf("expected X-Spam-Status Yes, got %s", spamRes.HeadersToAdd["X-Spam-Status"])
	}
	if spamRes.SpamScore != 18.5 {
		t.Errorf("expected spam score 18.5, got %f", spamRes.SpamScore)
	}
}

func TestPipelineAndEvaluatePolicy(t *testing.T) {
	ctx := context.Background()

	scannerClean := NewMockScanner(ActionReject)
	scannerFlag := NewMockScanner(ActionFlag)

	pipeline := NewPipeline(scannerClean, scannerFlag)

	// Clean payload
	res, err := pipeline.ScanContent(ctx, []byte("Harmless email"))
	if err != nil {
		t.Fatalf("pipeline scan failed: %v", err)
	}
	if !res.IsClean {
		t.Error("expected clean pipeline result")
	}
	if err := EvaluatePolicy(res); err != nil {
		t.Errorf("clean policy evaluate failed: %v", err)
	}

	// Payload triggering rejection
	eicar := "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
	resRejected, err := pipeline.ScanContent(ctx, []byte(eicar))
	if err != nil {
		t.Fatalf("pipeline scan failed: %v", err)
	}
	if resRejected.IsClean {
		t.Error("expected rejected")
	}
	if err := EvaluatePolicy(resRejected); err == nil {
		t.Error("expected EvaluatePolicy to return error on ActionReject")
	}

	// Payload with ActionFlag
	flaggedRes := &ScanResult{
		IsClean:     false,
		ActionTaken: ActionFlag,
		Details:     "Flagged as probable spam",
	}
	if err := EvaluatePolicy(flaggedRes); err != nil {
		t.Errorf("expected ActionFlag to be permitted through EvaluatePolicy, got %v", err)
	}
}
