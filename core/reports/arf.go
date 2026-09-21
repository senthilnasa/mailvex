package reports

import (
	"bufio"
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
)

type ParsedARF struct {
	FeedbackType    string            `json:"feedback_type"` // abuse, opt-out, not-spam, etc.
	UserAgent       string            `json:"user_agent,omitempty"`
	ReportedDomain  string            `json:"reported_domain,omitempty"`
	ArrivalDate     *time.Time        `json:"arrival_date,omitempty"`
	OriginalHeaders map[string]string `json:"original_headers,omitempty"`
	RecipientID     string            `json:"recipient_id,omitempty"`
	RecipientAddr   string            `json:"recipient_addr,omitempty"`
	TenantID        string            `json:"tenant_id,omitempty"`
	MessageID       string            `json:"message_id,omitempty"`
	StreamName      string            `json:"stream_name,omitempty"`
	CampaignTag     string            `json:"campaign_tag,omitempty"`
	RawMIME         []byte            `json:"-"`
}

// ParseARF parses RFC 5965 Abuse Reporting Format (ARF) feedback reports
func ParseARF(rawMIME []byte, verpSecret []byte) (*ParsedARF, error) {
	reader := bytes.NewReader(rawMIME)
	msg, err := mail.ReadMessage(reader)
	if err != nil {
		return nil, err
	}

	res := &ParsedARF{
		FeedbackType:    "abuse",
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
				case partMediaType == "message/feedback-report" || partIdx == 2:
					parseFeedbackReportPart(partBytes, res)
				case partMediaType == "message/rfc822" || partMediaType == "text/rfc822-headers" || partIdx == 3:
					parseARFOriginalHeaders(partBytes, res, verpSecret)
				}
			}
		}
	} else {
		// Fallback for single-part or non-standard reports: parse as headers
		parseFeedbackReportPart(rawMIME, res)
	}

	// Correlate recipient, tenant, and campaign from Feedback-ID if present:
	// Feedback-ID: <tenant>:<stream>:<campaign>:mailvex
	if fID, ok := res.OriginalHeaders["Feedback-Id"]; ok || res.OriginalHeaders["Feedback-ID"] != "" {
		if fID == "" {
			fID = res.OriginalHeaders["Feedback-ID"]
		}
		parts := strings.Split(fID, ":")
		if len(parts) >= 4 && parts[len(parts)-1] == "mailvex" {
			if res.TenantID == "" {
				res.TenantID = parts[0]
			}
			if res.StreamName == "" {
				res.StreamName = parts[1]
			}
			if res.CampaignTag == "" {
				res.CampaignTag = parts[2]
			}
		}
	}

	// Extract Message-ID
	if mID, ok := res.OriginalHeaders["Message-Id"]; ok || res.OriginalHeaders["Message-ID"] != "" {
		if mID == "" {
			mID = res.OriginalHeaders["Message-ID"]
		}
		res.MessageID = strings.Trim(mID, "<> ")
	}

	return res, nil
}

func parseFeedbackReportPart(data []byte, res *ParsedARF) {
	tp := textproto.NewReader(bufio.NewReader(bytes.NewReader(data)))
	hdr, err := tp.ReadMIMEHeader()
	if err != nil {
		return
	}

	if ft := hdr.Get("Feedback-Type"); ft != "" {
		res.FeedbackType = strings.ToLower(ft)
	}
	if ua := hdr.Get("User-Agent"); ua != "" {
		res.UserAgent = ua
	}
	if dom := hdr.Get("Reported-Domain"); dom != "" {
		res.ReportedDomain = dom
	}
	if rcpt := hdr.Get("Original-Rcpt-To"); rcpt != "" {
		if email := rxEmail.FindString(rcpt); email != "" {
			res.RecipientAddr = email
		}
	}
	if arr := hdr.Get("Arrival-Date"); arr != "" {
		if t, err := mail.ParseDate(arr); err == nil {
			res.ArrivalDate = &t
		}
	}
}

func parseARFOriginalHeaders(data []byte, res *ParsedARF, verpSecret []byte) {
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
			if email := rxEmail.FindString(to); email != "" {
				res.RecipientAddr = email
			}
		}
	}
}
