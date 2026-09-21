package reports

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
	"github.com/mailvex/mailvex/worker"
)

type ProcessDSNResult struct {
	IsAutoReply   bool              `json:"is_auto_reply"`
	IsUnmatched   bool              `json:"is_unmatched"`
	RecipientID   string            `json:"recipient_id,omitempty"`
	MessageID     string            `json:"message_id,omitempty"`
	TenantID      string            `json:"tenant_id,omitempty"`
	Status        string            `json:"status"`
	Action        string            `json:"action"`
	Classification string           `json:"classification"`
	Suppressed    bool              `json:"suppressed"`
}

type ProcessARFResult struct {
	ReportID    string `json:"report_id"`
	TenantID    string `json:"tenant_id"`
	MessageID   string `json:"message_id,omitempty"`
	RecipientID string `json:"recipient_id,omitempty"`
	Address     string `json:"address"`
	Suppressed  bool   `json:"suppressed"`
}

type ReportProcessor struct {
	repo        repository.Repository
	classifier  *worker.Classifier
	verpSecret  []byte
	unsubSecret []byte
}

func NewReportProcessor(repo repository.Repository, verpSecret, unsubSecret []byte) *ReportProcessor {
	if len(verpSecret) == 0 {
		verpSecret = []byte("mailvex-default-verp-secret-key-32b")
	}
	if len(unsubSecret) == 0 {
		unsubSecret = []byte("mailvex-unsubscribe-token-secret")
	}
	return &ReportProcessor{
		repo:        repo,
		classifier:  worker.NewClassifier(nil),
		verpSecret:  verpSecret,
		unsubSecret: unsubSecret,
	}
}

// ProcessInboundDSN processes an inbound bounce DSN email (§18.1)
func (p *ReportProcessor) ProcessInboundDSN(ctx context.Context, rawMIME []byte, envelopeRcpt string) (*ProcessDSNResult, error) {
	parsed, err := ParseDSN(rawMIME, p.verpSecret)
	if err != nil {
		// Log unmatched bounce with parse error
		_ = p.repo.SaveUnmatchedBounce(ctx, &domain.UnmatchedBounce{
			ID:         crypto.NewULID("ub_"),
			RawMIMERef: nil,
			ParseError: ptr(err.Error()),
			CreatedAt:  time.Now().UTC(),
		})
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	// 1. If auto-reply / out-of-office, drop without marking as bounce
	if parsed.IsAutoReply {
		return &ProcessDSNResult{
			IsAutoReply: true,
		}, nil
	}

	// 2. Resolve recipient: check parsed.RecipientID, or envelope recipient if VERP formatted
	rcptID := parsed.RecipientID
	if rcptID == "" && envelopeRcpt != "" {
		if id, err := crypto.ParseVERPAddress(envelopeRcpt, p.verpSecret); err == nil && id != "" {
			rcptID = id
		}
	}

	var rcpt *domain.MessageRecipient
	var msg *domain.Message

	if rcptID != "" {
		rcpt, msg, _ = p.repo.GetRecipientByTokenOrID(ctx, rcptID)
	}

	// If still not resolved, try finding recipient by parsed address
	if rcpt == nil && parsed.RecipientAddr != "" {
		rcpt, msg, _ = p.repo.GetRecipientByTokenOrID(ctx, parsed.RecipientAddr)
	}

	// 3. If recipient cannot be identified, store in unmatched_bounces (§18.1)
	if rcpt == nil {
		msgHeader, _ := mail.ReadMessage(strings.NewReader(string(rawMIME)))
		var sender, subject string
		if msgHeader != nil {
			sender = msgHeader.Header.Get("From")
			subject = msgHeader.Header.Get("Subject")
		}
		_ = p.repo.SaveUnmatchedBounce(ctx, &domain.UnmatchedBounce{
			ID:         crypto.NewULID("ub_"),
			Sender:     &sender,
			Recipient:  &parsed.RecipientAddr,
			Subject:    &subject,
			ParseError: ptr("recipient could not be correlated from VERP or headers"),
			CreatedAt:  time.Now().UTC(),
		})
		return &ProcessDSNResult{
			IsUnmatched:    true,
			Status:         parsed.Status,
			Action:         parsed.Action,
			Classification: parsed.Classification,
		}, nil
	}

	now := time.Now().UTC()
	var suppressed bool

	// 4. Update recipient status and record delivery attempt
	if parsed.IsPermanent {
		// Hard bounce or permanent policy rejection
		_ = p.repo.UpdateRecipientDelivery(ctx, rcpt.ID, "bounced", rcpt.Attempts+1, &parsed.Status, &parsed.DiagnosticCode, nil, &now, nil, nil)
		
		// Record delivery attempt
		_ = p.repo.RecordDeliveryAttempt(ctx, &domain.DeliveryAttempt{
			ID:           crypto.NewULID("att_"),
			RecipientID:  rcpt.ID,
			TenantID:     rcpt.TenantID,
			AttemptNo:    rcpt.Attempts + 1,
			SMTPCode:     &parsed.Status,
			EnhancedCode: &parsed.Status,
			SMTPResponse: &parsed.DiagnosticCode,
			Outcome:      "bounced",
			StartedAt:    now,
			CreatedAt:    now,
		})

		// Emit message.bounced event (§13.2)
		_ = p.repo.AddMessageEvent(ctx, &domain.MessageEvent{
			ID:          crypto.NewULID(crypto.PrefixEvent),
			TenantID:    rcpt.TenantID,
			MessageID:   rcpt.MessageID,
			RecipientID: &rcpt.ID,
			Type:        "message.bounced",
			Data:        fmt.Sprintf(`{"address":"%s","status":"%s","action":"%s","classification":"%s","diag":"%s"}`, rcpt.Address, parsed.Status, parsed.Action, parsed.Classification, escapeJSON(parsed.DiagnosticCode)),
			OccurredAt:  now,
			CreatedAt:   now,
		})

		// Auto-suppress address (§15 step 3, §18.1)
		suppReason := fmt.Sprintf("Asynchronous DSN hard bounce: %s (%s)", parsed.Status, parsed.Classification)
		supp := &domain.Suppression{
			ID:            crypto.NewULID("supp_"),
			TenantID:      rcpt.TenantID,
			StreamID:      nil,
			Address:       rcpt.Address,
			AddressDomain: rcpt.AddressDomain,
			Type:          "hard_bounce",
			Reason:        &suppReason,
			Source:        "dsn",
			CreatedAt:     now,
		}
		if msg != nil {
			supp.StreamID = &msg.StreamID
		}
		_ = p.repo.AddSuppression(ctx, supp)
		suppressed = true

		// Record in DLQ
		_ = p.repo.RecordDeadLetter(ctx, &domain.DeadLetterMessage{
			ID:               crypto.NewULID("dlq_"),
			TenantID:         rcpt.TenantID,
			MessageID:        rcpt.MessageID,
			RecipientID:      rcpt.ID,
			Destination:      rcpt.Address,
			Reason:           "dsn_hard_bounce",
			LastSMTPCode:     &parsed.Status,
			LastEnhancedCode: &parsed.Status,
			LastResponse:     &parsed.DiagnosticCode,
			Attempts:         rcpt.Attempts + 1,
			CreatedAt:        now,
		})

		var streamIDPtr, domainIDPtr *string
		campaignTag := "default"
		if msg != nil {
			streamIDPtr = &msg.StreamID
			domainIDPtr = &msg.DomainID
			if len(msg.Tags) > 0 {
				campaignTag = msg.Tags[0]
			}
		}
		_ = p.repo.RecordAnalyticsEvent(ctx, rcpt.TenantID, "hard_bounce", streamIDPtr, domainIDPtr, rcpt.AddressDomain, campaignTag, 0, now)
	} else {
		// Soft bounce / deferred
		_ = p.repo.UpdateRecipientDelivery(ctx, rcpt.ID, "deferred", rcpt.Attempts+1, &parsed.Status, &parsed.DiagnosticCode, nil, nil, nil, nil)
		_ = p.repo.AddMessageEvent(ctx, &domain.MessageEvent{
			ID:          crypto.NewULID(crypto.PrefixEvent),
			TenantID:    rcpt.TenantID,
			MessageID:   rcpt.MessageID,
			RecipientID: &rcpt.ID,
			Type:        "message.deferred",
			Data:        fmt.Sprintf(`{"address":"%s","status":"%s","action":"%s","classification":"%s"}`, rcpt.Address, parsed.Status, parsed.Action, parsed.Classification),
			OccurredAt:  now,
			CreatedAt:   now,
		})

		var streamIDPtr, domainIDPtr *string
		campaignTag := "default"
		if msg != nil {
			streamIDPtr = &msg.StreamID
			domainIDPtr = &msg.DomainID
			if len(msg.Tags) > 0 {
				campaignTag = msg.Tags[0]
			}
		}
		_ = p.repo.RecordAnalyticsEvent(ctx, rcpt.TenantID, "soft_bounce", streamIDPtr, domainIDPtr, rcpt.AddressDomain, campaignTag, 0, now)
	}

	tenantID := rcpt.TenantID
	msgID := rcpt.MessageID
	if msg != nil {
		tenantID = msg.TenantID
		msgID = msg.ID
	}

	return &ProcessDSNResult{
		RecipientID:    rcpt.ID,
		MessageID:      msgID,
		TenantID:       tenantID,
		Status:         parsed.Status,
		Action:         parsed.Action,
		Classification: parsed.Classification,
		Suppressed:     suppressed,
	}, nil
}

// ProcessInboundARF processes an inbound RFC 5965 ARF feedback report (§18.1)
func (p *ReportProcessor) ProcessInboundARF(ctx context.Context, rawMIME []byte) (*ProcessARFResult, error) {
	parsed, err := ParseARF(rawMIME, p.verpSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ARF: %w", err)
	}

	// Correlate recipient & tenant
	var rcpt *domain.MessageRecipient
	var msg *domain.Message

	if parsed.RecipientID != "" {
		rcpt, msg, _ = p.repo.GetRecipientByTokenOrID(ctx, parsed.RecipientID)
	}
	if rcpt == nil && parsed.RecipientAddr != "" {
		rcpt, msg, _ = p.repo.GetRecipientByTokenOrID(ctx, parsed.RecipientAddr)
	}
	if rcpt == nil && parsed.MessageID != "" {
		rcpt, msg, _ = p.repo.GetRecipientByTokenOrID(ctx, parsed.MessageID)
	}

	tenantID := parsed.TenantID
	var recipientID, messageID, recipientAddr, addressDomain string

	if rcpt != nil {
		recipientID = rcpt.ID
		recipientAddr = rcpt.Address
		addressDomain = rcpt.AddressDomain
		tenantID = rcpt.TenantID
		messageID = rcpt.MessageID
	} else {
		recipientAddr = parsed.RecipientAddr
		parts := strings.Split(recipientAddr, "@")
		if len(parts) == 2 {
			addressDomain = parts[1]
		}
		messageID = parsed.MessageID
	}

	if tenantID == "" {
		tenantID = "unknown_tenant"
	}

	now := time.Now().UTC()
	reportID := crypto.NewULID("fbl_")

	// 1. Save FBL report row
	var origHeadersStr string
	if len(parsed.OriginalHeaders) > 0 {
		var b strings.Builder
		for k, v := range parsed.OriginalHeaders {
			b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
		origHeadersStr = b.String()
	}

	fbl := &domain.FBLReport{
		ID:              reportID,
		TenantID:        tenantID,
		FeedbackType:    parsed.FeedbackType,
		UserAgent:       &parsed.UserAgent,
		ReportedDomain:  &parsed.ReportedDomain,
		OriginalHeaders: &origHeadersStr,
		ArrivalDate:     parsed.ArrivalDate,
		CreatedAt:       now,
	}
	if messageID != "" {
		fbl.MessageID = &messageID
	}
	if recipientID != "" {
		fbl.RecipientID = &recipientID
	}
	_ = p.repo.SaveFBLReport(ctx, fbl)

	// 2. Emit message.complained event (§13.2)
	if messageID != "" {
		var rcptIDPtr *string
		if recipientID != "" {
			rcptIDPtr = &recipientID
		}
		_ = p.repo.AddMessageEvent(ctx, &domain.MessageEvent{
			ID:          crypto.NewULID(crypto.PrefixEvent),
			TenantID:    tenantID,
			MessageID:   messageID,
			RecipientID: rcptIDPtr,
			Type:        "message.complained",
			Data:        fmt.Sprintf(`{"feedback_type":"%s","address":"%s","reported_domain":"%s"}`, parsed.FeedbackType, recipientAddr, parsed.ReportedDomain),
			OccurredAt:  now,
			CreatedAt:   now,
		})
	}

	// 3. Immediately add address to suppressions table (type: complaint, source: fbl)
	if recipientAddr != "" {
		suppReason := fmt.Sprintf("ARF FBL complaint: feedback_type=%s", parsed.FeedbackType)
		var streamIDPtr *string
		if msg != nil {
			streamIDPtr = &msg.StreamID
		}
		_ = p.repo.AddSuppression(ctx, &domain.Suppression{
			ID:            crypto.NewULID("supp_"),
			TenantID:      tenantID,
			StreamID:      streamIDPtr,
			Address:       recipientAddr,
			AddressDomain: addressDomain,
			Type:          "complaint",
			Reason:        &suppReason,
			Source:        "fbl",
			CreatedAt:     now,
		})

		var domainIDPtr *string
		campaignTag := "default"
		if parsed.CampaignTag != "" {
			campaignTag = parsed.CampaignTag
		} else if msg != nil {
			if len(msg.Tags) > 0 {
				campaignTag = msg.Tags[0]
			}
			domainIDPtr = &msg.DomainID
		}
		_ = p.repo.RecordAnalyticsEvent(ctx, tenantID, "message.complained", streamIDPtr, domainIDPtr, addressDomain, campaignTag, 0, now)
	}

	return &ProcessARFResult{
		ReportID:    reportID,
		TenantID:    tenantID,
		MessageID:   messageID,
		RecipientID: recipientID,
		Address:     recipientAddr,
		Suppressed:  recipientAddr != "",
	}, nil
}

// ProcessMailtoUnsubscribe processes an inbound mailto unsubscribe email (§18.1)
func (p *ReportProcessor) ProcessMailtoUnsubscribe(ctx context.Context, recipientEnvelope string, rawMIME []byte) error {
	// Recipient envelope is e.g. unsubscribe+<token>@<domain>
	cleaned := strings.TrimSpace(recipientEnvelope)
	cleaned = strings.TrimPrefix(cleaned, "<")
	cleaned = strings.TrimSuffix(cleaned, ">")

	plusIdx := strings.Index(cleaned, "+")
	atIdx := strings.LastIndex(cleaned, "@")
	if plusIdx == -1 || atIdx == -1 || plusIdx > atIdx {
		return fmt.Errorf("invalid mailto unsubscribe recipient format: %s", recipientEnvelope)
	}

	token := cleaned[plusIdx+1 : atIdx]
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return fmt.Errorf("malformed unsubscribe token: %s", token)
	}

	rcptID, sig := parts[0], parts[1]

	rcpt, err := p.repo.GetRecipient(ctx, rcptID)
	if err != nil {
		return fmt.Errorf("recipient not found: %w", err)
	}

	// Validate HMAC signature
	mac := hmac.New(sha256.New, p.unsubSecret)
	mac.Write([]byte(fmt.Sprintf("%s:%s:%s", rcpt.TenantID, rcpt.ID, rcpt.Address)))
	expectedSig := hex.EncodeToString(mac.Sum(nil))[:24]

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return fmt.Errorf("unauthorized unsubscribe token")
	}

	msg, _ := p.repo.GetMessageByID(ctx, rcpt.TenantID, rcpt.MessageID)
	var streamIDPtr *string
	if msg != nil {
		streamIDPtr = &msg.StreamID
	}

	now := time.Now().UTC()
	reason := "Unsubscribed via mailto"
	supp := &domain.Suppression{
		ID:            crypto.NewULID("supp_"),
		TenantID:      rcpt.TenantID,
		StreamID:      streamIDPtr,
		Address:       rcpt.Address,
		AddressDomain: rcpt.AddressDomain,
		Type:          "unsubscribe",
		Reason:        &reason,
		Source:        "mailto",
		CreatedAt:     now,
	}

	if err := p.repo.AddSuppression(ctx, supp); err != nil {
		return fmt.Errorf("failed to record unsubscribe suppression: %w", err)
	}

	// Emit message.unsubscribed event
	_ = p.repo.AddMessageEvent(ctx, &domain.MessageEvent{
		ID:          crypto.NewULID(crypto.PrefixEvent),
		TenantID:    rcpt.TenantID,
		MessageID:   rcpt.MessageID,
		RecipientID: &rcpt.ID,
		Type:        "message.unsubscribed",
		Data:        fmt.Sprintf(`{"address":"%s","source":"mailto"}`, rcpt.Address),
		OccurredAt:  now,
		CreatedAt:   now,
	})

	var domainIDPtr *string
	campaignTag := "default"
	if msg != nil {
		domainIDPtr = &msg.DomainID
		if len(msg.Tags) > 0 {
			campaignTag = msg.Tags[0]
		}
	}
	_ = p.repo.RecordAnalyticsEvent(ctx, rcpt.TenantID, "message.unsubscribed", streamIDPtr, domainIDPtr, rcpt.AddressDomain, campaignTag, 0, now)

	return nil
}

// ProcessAbuseIntake processes inbound mail to abuse@ or postmaster@ (§18.1, RFC 5321 §4.5.1)
func (p *ReportProcessor) ProcessAbuseIntake(ctx context.Context, sender, recipient, subject string, rawMIME []byte) (*domain.AbuseReport, error) {
	now := time.Now().UTC()
	id := crypto.NewULID("abuse_")

	// Attempt to correlate domain
	var domainIDPtr *string
	var tenantIDPtr *string
	parts := strings.Split(recipient, "@")
	if len(parts) == 2 {
		domName := strings.ToLower(parts[1])
		// Check domains
		if dom, err := p.repo.GetDomainByName(ctx, domName); err == nil && dom != nil {
			domainIDPtr = &dom.ID
			tenantIDPtr = &dom.TenantID
		}
	}

	report := &domain.AbuseReport{
		ID:        id,
		TenantID:  tenantIDPtr,
		DomainID:  domainIDPtr,
		Sender:    sender,
		Recipient: recipient,
		Subject:   &subject,
		Status:    "pending",
		CreatedAt: now,
	}

	if err := p.repo.SaveAbuseReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to save abuse report: %w", err)
	}

	return report, nil
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", ` `)
	s = strings.ReplaceAll(s, "\r", ` `)
	return s
}

func ptr(s string) *string {
	return &s
}
