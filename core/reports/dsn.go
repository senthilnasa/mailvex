package reports

import (
	"bufio"
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"regexp"
	"strings"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/worker"
)

var (
	rxEnhancedCode = regexp.MustCompile(`\b([245]\.[0-9]{1,3}\.[0-9]{1,3})\b`)
	rxSMTPCode     = regexp.MustCompile(`\b([45][0-9]{2})\b`)
	rxEmail        = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
)

type ParsedDSN struct {
	IsAutoReply      bool              `json:"is_auto_reply"`
	RecipientID      string            `json:"recipient_id,omitempty"`
	RecipientAddr    string            `json:"recipient_addr,omitempty"`
	Action           string            `json:"action,omitempty"` // failed, delayed, delivered, relayed, expanded
	Status           string            `json:"status,omitempty"` // e.g. 5.1.1, 4.2.2
	DiagnosticCode   string            `json:"diagnostic_code,omitempty"`
	ReportingMTA     string            `json:"reporting_mta,omitempty"`
	OriginalHeaders  map[string]string `json:"original_headers,omitempty"`
	Classification   string            `json:"classification"` // hard_bounce, soft_bounce, etc.
	IsPermanent      bool              `json:"is_permanent"`
	RawMIME          []byte            `json:"-"`
}

// IsAutoReply checks whether an email is an automatic out-of-office or vacation response
func IsAutoReply(hdr mail.Header) bool {
	// RFC 3834 Auto-Submitted
	autoSubmitted := strings.ToLower(hdr.Get("Auto-Submitted"))
	if autoSubmitted != "" && autoSubmitted != "no" {
		return true
	}

	// Microsoft & generic headers
	if strings.EqualFold(hdr.Get("X-Autoreply"), "yes") ||
		strings.EqualFold(hdr.Get("X-Autorespond"), "yes") ||
		strings.EqualFold(hdr.Get("X-Auto-Response-Suppress"), "all") ||
		strings.EqualFold(hdr.Get("X-Auto-Response-Suppress"), "oof") {
		return true
	}

	// Precedence: auto_reply
	prec := strings.ToLower(hdr.Get("Precedence"))
	if prec == "auto_reply" {
		return true
	}

	// Subject keyword heuristics
	subj := strings.ToLower(hdr.Get("Subject"))
	autoKeywords := []string{
		"out of office",
		"automatic reply",
		"auto-reply",
		"autoreply",
		"auto response",
		"vacation notice",
		"away from the office",
	}
	for _, kw := range autoKeywords {
		if strings.Contains(subj, kw) {
			return true
		}
	}

	return false
}

// ParseDSN parses raw RFC 3464 MIME message or heuristic non-standard bounce
func ParseDSN(rawMIME []byte, verpSecret []byte) (*ParsedDSN, error) {
	reader := bytes.NewReader(rawMIME)
	msg, err := mail.ReadMessage(reader)
	if err != nil {
		return nil, err
	}

	// 1. Check if it's an auto-reply
	if IsAutoReply(msg.Header) {
		return &ParsedDSN{
			IsAutoReply: true,
			RawMIME:     rawMIME,
		}, nil
	}

	res := &ParsedDSN{
		OriginalHeaders: make(map[string]string),
		RawMIME:         rawMIME,
	}

	contentType := msg.Header.Get("Content-Type")
	mediaType, params, _ := mime.ParseMediaType(contentType)

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary != "" {
			mr := multipart.NewReader(msg.Body, boundary)
			partIdx := 0
			for {
				part, err := mr.NextPart()
				if err != nil {
					break
				}
				partIdx++
				partMediaType, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
				partBytes, _ := io.ReadAll(part)

				switch {
				case partMediaType == "message/delivery-status" || partIdx == 2:
					parseDeliveryStatusPart(partBytes, res)
				case partMediaType == "message/rfc822" || partMediaType == "text/rfc822-headers" || partIdx == 3:
					parseOriginalHeadersPart(partBytes, res, verpSecret)
				}
			}
		}
	}

	// If machine status fields weren't found from multipart, fall back to heuristic body inspection
	if res.Status == "" && res.Action == "" {
		parseDSNHeuristics(rawMIME, res, verpSecret)
	}

	// Classify response
	classifier := worker.NewClassifier(nil)
	class := classifier.Classify("", res.Status, res.DiagnosticCode)
	res.Classification = class.Classification

	// Determine if permanent bounce
	if strings.HasPrefix(res.Status, "5.") || strings.EqualFold(res.Action, "failed") || class.Action == "suppress_immediate" {
		res.IsPermanent = true
		if res.Action == "" {
			res.Action = "failed"
		}
	} else if strings.HasPrefix(res.Status, "4.") || strings.EqualFold(res.Action, "delayed") {
		res.IsPermanent = false
		if res.Action == "" {
			res.Action = "delayed"
		}
	} else {
		// default to permanent if 5xx or failed action
		res.IsPermanent = strings.EqualFold(res.Action, "failed")
	}

	return res, nil
}

func parseDeliveryStatusPart(data []byte, res *ParsedDSN) {
	tp := textproto.NewReader(bufio.NewReader(bytes.NewReader(data)))
	// Read blocks (per-message fields then per-recipient fields)
	for {
		hdr, err := tp.ReadMIMEHeader()
		if err != nil {
			break
		}
		if action := hdr.Get("Action"); action != "" {
			res.Action = strings.ToLower(action)
		}
		if status := hdr.Get("Status"); status != "" {
			res.Status = status
		}
		if diag := hdr.Get("Diagnostic-Code"); diag != "" {
			res.DiagnosticCode = diag
		}
		if rMTA := hdr.Get("Reporting-MTA"); rMTA != "" {
			res.ReportingMTA = rMTA
		}
		if rcpt := hdr.Get("Final-Recipient"); rcpt != "" {
			parts := strings.SplitN(rcpt, ";", 2)
			if len(parts) == 2 {
				res.RecipientAddr = strings.TrimSpace(parts[1])
			} else {
				res.RecipientAddr = strings.TrimSpace(rcpt)
			}
		}
	}
}

func parseOriginalHeadersPart(data []byte, res *ParsedDSN, verpSecret []byte) {
	tp := textproto.NewReader(bufio.NewReader(bytes.NewReader(data)))
	hdr, err := tp.ReadMIMEHeader()
	if err != nil {
		return
	}

	for k, vv := range hdr {
		if len(vv) > 0 {
			res.OriginalHeaders[k] = vv[0]
		}
	}

	// Try extracting VERP from Return-Path
	if rp := hdr.Get("Return-Path"); rp != "" {
		if rcptID, err := crypto.ParseVERPAddress(rp, verpSecret); err == nil && rcptID != "" {
			res.RecipientID = rcptID
		}
	}

	// If no recipient address yet, check To
	if res.RecipientAddr == "" {
		if to := hdr.Get("To"); to != "" {
			if matches := rxEmail.FindString(to); matches != "" {
				res.RecipientAddr = matches
			}
		}
	}
}

func parseDSNHeuristics(raw []byte, res *ParsedDSN, verpSecret []byte) {
	content := string(raw)

	// Search for enhanced status code e.g. 5.1.1
	if match := rxEnhancedCode.FindString(content); match != "" {
		res.Status = match
	} else if match := rxSMTPCode.FindString(content); match != "" {
		res.Status = match
	}

	// Search for Return-Path: <bounce+...>
	rpIdx := strings.Index(content, "Return-Path:")
	if rpIdx != -1 {
		endIdx := strings.Index(content[rpIdx:], "\n")
		if endIdx != -1 {
			line := content[rpIdx : rpIdx+endIdx]
			if rcptID, err := crypto.ParseVERPAddress(line, verpSecret); err == nil && rcptID != "" {
				res.RecipientID = rcptID
			}
		}
	}

	// Search for email address if RecipientAddr is empty
	if res.RecipientAddr == "" {
		emails := rxEmail.FindAllString(content, -1)
		for _, e := range emails {
			if !strings.HasPrefix(e, "bounce+") && !strings.Contains(e, "postmaster") && !strings.Contains(e, "mailer-daemon") {
				res.RecipientAddr = e
				break
			}
		}
	}

	// Search for common diagnostic lines
	diagMarkers := []string{"Diagnostic-Code:", "Remote-MTA:", "Diagnostic:"}
	for _, m := range diagMarkers {
		if idx := strings.Index(content, m); idx != -1 {
			end := strings.Index(content[idx:], "\n")
			if end != -1 {
				res.DiagnosticCode = strings.TrimSpace(content[idx+len(m) : idx+end])
				break
			}
		}
	}
}
