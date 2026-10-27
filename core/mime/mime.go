package mime

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
)

type ParsedEmail struct {
	From        string
	FromName    string
	To          []string
	Cc          []string
	Bcc         []string
	ReplyTo     string
	Subject     string
	MessageID   string
	Date        time.Time
	Headers     map[string]string
	BodyText    string
	BodyHTML    string
	RawMIME     []byte
}

// NormalizeDeviceMIME cleans and standardizes IoT and appliance MIME inputs (§28.1)
func NormalizeDeviceMIME(raw []byte, defaultFrom, domainName string) []byte {
	// 1. Normalize line endings: ensure CRLF everywhere
	normalized := strings.ReplaceAll(string(raw), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.ReplaceAll(normalized, "\n", "\r\n")

	// 2. Check for essential headers
	hasDate := strings.Contains(normalized, "Date:")
	hasMsgID := strings.Contains(normalized, "Message-ID:") || strings.Contains(normalized, "Message-Id:")
	hasMIMEVer := strings.Contains(normalized, "MIME-Version:")
	hasFrom := strings.Contains(normalized, "From:")

	var prefix strings.Builder
	if !hasFrom && defaultFrom != "" {
		prefix.WriteString(fmt.Sprintf("From: %s\r\n", defaultFrom))
	}
	if !hasDate {
		prefix.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z)))
	}
	if !hasMsgID {
		uniqueID := crypto.NewULID("")
		prefix.WriteString(fmt.Sprintf("Message-ID: <%s@%s>\r\n", uniqueID, domainName))
	}
	if !hasMIMEVer {
		prefix.WriteString("MIME-Version: 1.0\r\n")
	}

	if prefix.Len() > 0 {
		return []byte(prefix.String() + normalized)
	}

	return []byte(normalized)
}

// Parse parses and extracts structured email fields from raw MIME
func Parse(raw []byte) (*ParsedEmail, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to parse MIME message: %w", err)
	}

	headers := make(map[string]string)
	for k, v := range msg.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	parsed := &ParsedEmail{
		Subject:   msg.Header.Get("Subject"),
		MessageID: msg.Header.Get("Message-ID"),
		ReplyTo:   msg.Header.Get("Reply-To"),
		Headers:   headers,
		RawMIME:   raw,
	}

	// Parse From
	fromList, err := msg.Header.AddressList("From")
	if err == nil && len(fromList) > 0 {
		parsed.From = fromList[0].Address
		parsed.FromName = fromList[0].Name
	}

	// Parse To
	toList, err := msg.Header.AddressList("To")
	if err == nil {
		for _, addr := range toList {
			parsed.To = append(parsed.To, addr.Address)
		}
	}

	// Parse Cc
	ccList, err := msg.Header.AddressList("Cc")
	if err == nil {
		for _, addr := range ccList {
			parsed.Cc = append(parsed.Cc, addr.Address)
		}
	}

	// Parse Date
	dateStr := msg.Header.Get("Date")
	if dateStr != "" {
		if t, err := mail.ParseDate(dateStr); err == nil {
			parsed.Date = t
		}
	}
	if parsed.Date.IsZero() {
		parsed.Date = time.Now().UTC()
	}

	bodyBytes, _ := io.ReadAll(msg.Body)
	parsed.BodyText = string(bodyBytes)

	return parsed, nil
}

// BuildMessage constructs a standard MIME message (supporting text/plain, text/html, or multipart/alternative)
func BuildMessage(from, to, subject, bodyText, bodyHTML string) []byte {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z)))
	buf.WriteString("MIME-Version: 1.0\r\n")

	hasText := strings.TrimSpace(bodyText) != ""
	hasHTML := strings.TrimSpace(bodyHTML) != ""

	if hasText && hasHTML {
		boundary := fmt.Sprintf("=_mvx_%s", crypto.NewULID(""))
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary))

		// Plain text part
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
		buf.WriteString(bodyText)
		buf.WriteString("\r\n\r\n")

		// HTML part
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
		buf.WriteString(bodyHTML)
		buf.WriteString("\r\n\r\n")

		// Closing boundary
		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if hasHTML {
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
		buf.WriteString(bodyHTML)
		buf.WriteString("\r\n")
	} else {
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
		buf.WriteString(bodyText)
		buf.WriteString("\r\n")
	}

	return buf.Bytes()
}
