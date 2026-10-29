package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/mime"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/storage"
)

var (
	ErrDomainNotVerified = errors.New("domain_not_verified")
	ErrRelayingDenied    = errors.New("relaying_denied")
	ErrHeaderInjection   = errors.New("header_injection")
)

type IngestRequest struct {
	TenantID         string
	Source           string // "api", "smtp", "system"
	APIKeyID         *string
	SMTPCredentialID *string
	FromAddress      string
	FromName         string
	To               []string
	Cc               []string
	Bcc              []string
	ReplyTo          *string
	Subject          string
	BodyText         string
	BodyHTML         string
	RawMIME          []byte
	StreamName       string
	Tags             []string
	Metadata         map[string]string
	Environment      string // "live", "development", "test"
}

type IngestResult struct {
	MessageID   string `json:"id"`
	Status      string `json:"status"` // "queued"
	Environment string `json:"environment"` // "live", "development"
	RecipientIDs []string `json:"recipient_ids"`
}

type IngestionEngine struct {
	repo    repository.Repository
	storage storage.Storage
}

func NewIngestionEngine(repo repository.Repository, store storage.Storage) *IngestionEngine {
	return &IngestionEngine{
		repo:    repo,
		storage: store,
	}
}

// Ingest performs unified ingestion across API and SMTP (§13.1)
func (eng *IngestionEngine) Ingest(ctx context.Context, req *IngestRequest) (*IngestResult, error) {
	// 1. Validate inputs and guard against CRLF/header injection (§23.3)
	if strings.ContainsAny(req.FromAddress, "\r\n") || strings.ContainsAny(req.Subject, "\r\n") {
		return nil, ErrHeaderInjection
	}
	for _, to := range req.To {
		if strings.ContainsAny(to, "\r\n") {
			return nil, ErrHeaderInjection
		}
	}

	if len(req.To) == 0 {
		return nil, errors.New("no recipients specified")
	}

	// 2. Extract from domain and verify ownership
	parts := strings.Split(req.FromAddress, "@")
	if len(parts) != 2 {
		return nil, errors.New("invalid from address format")
	}
	fromDomainName := strings.ToLower(parts[1])

	fromDom, err := eng.repo.GetDomainByName(ctx, fromDomainName)
	if err != nil {
		return nil, fmt.Errorf("%w: domain %s not registered", ErrDomainNotVerified, fromDomainName)
	}

	if fromDom.TenantID != req.TenantID {
		return nil, fmt.Errorf("%w: domain %s belongs to another tenant", ErrDomainNotVerified, fromDomainName)
	}

	if fromDom.Status != "verified" && fromDom.Status != "degraded" {
		return nil, fmt.Errorf("%w: domain %s is in %s status", ErrDomainNotVerified, fromDomainName, fromDom.Status)
	}

	// 3. Resolve tenant environment
	tenant, err := eng.repo.GetTenant(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}

	env := req.Environment
	if env == "" {
		env = tenant.Environment
	}
	if env == "test" {
		env = "development"
	}
	if env == "" {
		env = "live"
	}

	// 4. Resolve stream
	streamName := req.StreamName
	if streamName == "" {
		streamName = "transactional"
	}

	streams, _ := eng.repo.ListStreams(ctx, req.TenantID)
	var streamID string
	for _, s := range streams {
		if s.Name == streamName {
			streamID = s.ID
			break
		}
	}
	if streamID == "" {
		streamID = crypto.NewULID(crypto.PrefixStream)
	}

	// 5. Generate message and IDs
	msgID := crypto.NewULID(crypto.PrefixMessage)
	headerMsgID := fmt.Sprintf("<%s@%s>", crypto.NewULID(""), fromDomainName)

	priority := 1
	if streamName == "transactional" {
		priority = 1
	} else if streamName == "broadcast" {
		priority = 2
	}

	msg := &domain.Message{
		ID:               msgID,
		TenantID:         req.TenantID,
		StreamID:         streamID,
		DomainID:         fromDom.ID,
		Source:           req.Source,
		APIKeyID:         req.APIKeyID,
		SMTPCredentialID: req.SMTPCredentialID,
		FromAddress:      req.FromAddress,
		FromName:         req.FromName,
		ReplyTo:          req.ReplyTo,
		Subject:          req.Subject,
		MessageIDHeader:  headerMsgID,
		Status:           "queued",
		Priority:         priority,
		Environment:      env,
		SizeBytes:        len(req.RawMIME) + len(req.BodyText) + len(req.BodyHTML),
		Tags:             req.Tags,
		Metadata:         req.Metadata,
		QueuedAt:         time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
	}

	// 6. Recipients and VERP tokens (§17.2)
	var recipients []*domain.MessageRecipient
	var recipientIDs []string

	allAddrs := append([]string{}, req.To...)
	for _, addr := range allAddrs {
		addr = strings.TrimSpace(strings.ToLower(addr))
		rcptParts := strings.Split(addr, "@")
		rcptDomain := ""
		if len(rcptParts) == 2 {
			rcptDomain = rcptParts[1]
		}

		rcptID := crypto.NewULID(crypto.PrefixRecipient)
		recipientIDs = append(recipientIDs, rcptID)

		// VERP token: HMAC-SHA256(secret, rcptID)
		mac := hmac.New(sha256.New, []byte("mailvex-verp-secret-key"))
		mac.Write([]byte(rcptID))
		verpToken := hex.EncodeToString(mac.Sum(nil))[:16]

		status := "queued"
		var suppressedReason *string
		isSuppressed, reason, _ := eng.repo.CheckSuppression(ctx, req.TenantID, streamID, addr)
		if isSuppressed {
			status = "suppressed"
			suppressedReason = &reason
		}

		rcpt := &domain.MessageRecipient{
			ID:               rcptID,
			MessageID:        msgID,
			TenantID:         req.TenantID,
			Address:          addr,
			AddressDomain:    rcptDomain,
			MXGroup:          "custom",
			Type:             "to",
			Status:           status,
			SuppressedReason: suppressedReason,
			VERPToken:        &verpToken,
			CreatedAt:        time.Now().UTC(),
		}
		recipients = append(recipients, rcpt)
	}

	// 7. Initial message events
	var events []*domain.MessageEvent
	events = append(events, &domain.MessageEvent{
		ID:         crypto.NewULID(crypto.PrefixEvent),
		TenantID:   req.TenantID,
		MessageID:  msgID,
		Type:       "message.accepted",
		Data:       `{"stream":"` + streamName + `","recipients_count":` + fmt.Sprintf("%d", len(recipients)) + `}`,
		OccurredAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
	})

	for _, rcpt := range recipients {
		if rcpt.Status == "suppressed" {
			rcptIDCopy := rcpt.ID
			reasonStr := "unknown"
			if rcpt.SuppressedReason != nil {
				reasonStr = *rcpt.SuppressedReason
			}
			events = append(events, &domain.MessageEvent{
				ID:          crypto.NewULID(crypto.PrefixEvent),
				TenantID:    req.TenantID,
				MessageID:   msgID,
				RecipientID: &rcptIDCopy,
				Type:        "message.suppressed",
				Data:        `{"reason":"` + reasonStr + `","address":"` + rcpt.Address + `"}`,
				OccurredAt:  time.Now().UTC(),
				CreatedAt:   time.Now().UTC(),
			})
		}
	}

	// 8. Outbox item
	outboxStream := fmt.Sprintf("mvx:delivery:p%d", priority)
	taskPayload, _ := json.Marshal(map[string]any{
		"message_id": msgID,
		"tenant_id":  req.TenantID,
		"stream_id":  streamID,
		"priority":   priority,
	})

	outboxItem := &domain.OutboxItem{
		ID:          crypto.NewULID("out_"),
		TenantID:    req.TenantID,
		Stream:      outboxStream,
		Payload:     string(taskPayload),
		AvailableAt: time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	}

	// 9. Store body in Storage abstraction
	if len(req.RawMIME) > 0 {
		bodyKey := fmt.Sprintf("%s/%s/raw.eml", req.TenantID, msgID)
		_ = eng.storage.Put(ctx, bodyKey, req.RawMIME)
		msg.RawMIMERef = &bodyKey
	} else if req.BodyHTML != "" || req.BodyText != "" {
		constructedMIME := mime.BuildMessage(req.FromAddress, strings.Join(req.To, ", "), req.Subject, req.BodyText, req.BodyHTML)
		rawKey := fmt.Sprintf("%s/%s/raw.eml", req.TenantID, msgID)
		_ = eng.storage.Put(ctx, rawKey, constructedMIME)
		msg.RawMIMERef = &rawKey

		bodyKey := fmt.Sprintf("%s/%s/body.txt", req.TenantID, msgID)
		_ = eng.storage.Put(ctx, bodyKey, []byte(req.BodyText+"\n"+req.BodyHTML))
		msg.BodyRef = &bodyKey
	}

	// 10. Atomic database transaction
	if err := eng.repo.SaveMessage(ctx, msg, recipients, events, outboxItem); err != nil {
		return nil, fmt.Errorf("failed to commit message transaction: %w", err)
	}

	campaignTag := "default"
	if len(msg.Tags) > 0 {
		campaignTag = msg.Tags[0]
	}
	now := time.Now().UTC()
	for _, rcpt := range recipients {
		if rcpt.Status != "suppressed" {
			_ = eng.repo.RecordAnalyticsEvent(ctx, req.TenantID, "message.sent", &streamID, &fromDom.ID, rcpt.AddressDomain, campaignTag, 0, now)
		}
	}

	return &IngestResult{
		MessageID:    msgID,
		Status:       "queued",
		Environment:  env,
		RecipientIDs: recipientIDs,
	}, nil
}

// IngestRawMIME ingests an SMTP submission session after normalizing device MIME
func (eng *IngestionEngine) IngestRawMIME(ctx context.Context, tenantID, from string, to []string, rawMIME []byte, credID *string) (*IngestResult, error) {
	parts := strings.Split(from, "@")
	domainName := "localhost"
	if len(parts) == 2 {
		domainName = parts[1]
	}

	normalized := mime.NormalizeDeviceMIME(rawMIME, from, domainName)
	parsed, err := mime.Parse(normalized)
	if err != nil {
		return nil, fmt.Errorf("invalid MIME message: %w", err)
	}

	if parsed.From == "" {
		parsed.From = from
	}
	if len(to) > 0 {
		parsed.To = to
	}

	req := &IngestRequest{
		TenantID:         tenantID,
		Source:           "smtp",
		SMTPCredentialID: credID,
		FromAddress:      parsed.From,
		FromName:         parsed.FromName,
		To:               parsed.To,
		Cc:               parsed.Cc,
		Subject:          parsed.Subject,
		BodyText:         parsed.BodyText,
		RawMIME:          normalized,
		StreamName:       "transactional",
	}

	return eng.Ingest(ctx, req)
}
