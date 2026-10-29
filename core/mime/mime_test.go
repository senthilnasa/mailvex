package mime

import (
	"strings"
	"testing"
)

func TestMIMENormalizationAndParsing(t *testing.T) {
	// Raw IoT email with bare LF, missing Date, missing Message-ID, missing MIME-Version
	rawIoT := []byte("Subject: Security Camera Alert\nTo: admin@home.lan\n\nMotion detected in driveway.")

	normalized := NormalizeDeviceMIME(rawIoT, "camera@home.lan", "home.lan")

	normStr := string(normalized)

	// Check CRLF
	if !strings.Contains(normStr, "\r\n") {
		t.Fatalf("expected CRLF line endings in normalized MIME")
	}

	// Check injected headers
	if !strings.Contains(normStr, "From: camera@home.lan") {
		t.Fatalf("expected injected From header")
	}
	if !strings.Contains(normStr, "Date:") {
		t.Fatalf("expected injected Date header")
	}
	if !strings.Contains(normStr, "Message-ID:") {
		t.Fatalf("expected injected Message-ID header")
	}
	if !strings.Contains(normStr, "MIME-Version: 1.0") {
		t.Fatalf("expected injected MIME-Version header")
	}

	// Test Parsing
	parsed, err := Parse(normalized)
	if err != nil {
		t.Fatalf("failed to parse normalized MIME: %v", err)
	}

	if parsed.Subject != "Security Camera Alert" {
		t.Fatalf("expected subject 'Security Camera Alert', got '%s'", parsed.Subject)
	}
	if len(parsed.To) != 1 || parsed.To[0] != "admin@home.lan" {
		t.Fatalf("expected recipient admin@home.lan, got %+v", parsed.To)
	}
	if parsed.From != "camera@home.lan" {
		t.Fatalf("expected from camera@home.lan, got '%s'", parsed.From)
	}
	if !strings.Contains(parsed.BodyText, "Motion detected in driveway.") {
		t.Fatalf("expected body text to contain alert message")
	}
}

func TestBuildMessageWithCcAndUTF8(t *testing.T) {
	msg := BuildMessageWithCc("sender@example.com", "to@example.com", "cc@example.com", "Hello World! 🚀", "Hello éü", "<b>Hello éü</b>")
	msgStr := string(msg)

	if !strings.Contains(msgStr, "Cc: cc@example.com\r\n") {
		t.Fatalf("expected Cc header in message, got:\n%s", msgStr)
	}
	if !strings.Contains(msgStr, "Content-Transfer-Encoding: 8bit") {
		t.Fatalf("expected 8bit encoding for UTF-8 compatibility, got:\n%s", msgStr)
	}
	if !strings.Contains(msgStr, "Subject: Hello World! 🚀") {
		t.Fatalf("expected emoji in subject, got:\n%s", msgStr)
	}
}
