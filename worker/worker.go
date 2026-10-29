package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/mime"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/storage"
)

type WorkerConfig struct {
	NodeID          string
	PlatformHost    string
	BounceDomain    string
	DefaultPort     int // default 25, can be overridden for tests
	ShutdownGrace   time.Duration
}

type DeliveryWorker struct {
	repo            repository.Repository
	storage         storage.Storage
	router          *Router
	resolver        MXResolver
	classifier      *Classifier
	retry           *RetryScheduler
	directTransport Transport
	relayTransport  Transport
	encryption      *crypto.EnvelopeEncryption
	cfg             WorkerConfig

	inFlightWg      sync.WaitGroup
	draining        atomic.Bool
	stopCh          chan struct{}
}

func NewDeliveryWorker(
	repo repository.Repository,
	store storage.Storage,
	router *Router,
	resolver MXResolver,
	classifier *Classifier,
	cfg WorkerConfig,
) *DeliveryWorker {
	if cfg.ShutdownGrace <= 0 {
		cfg.ShutdownGrace = 30 * time.Second
	}
	if cfg.NodeID == "" {
		cfg.NodeID = "node-1"
	}
	if cfg.PlatformHost == "" {
		cfg.PlatformHost = "mailvex.local"
	}

	return &DeliveryWorker{
		repo:            repo,
		storage:         store,
		router:          router,
		resolver:        resolver,
		classifier:      classifier,
		retry:           NewRetryScheduler(),
		directTransport: NewDirectMXTransport(),
		relayTransport:  NewRelayTransport(),
		cfg:             cfg,
		stopCh:          make(chan struct{}),
	}
}

// SetTransports allows injecting mock transports for testing
func (w *DeliveryWorker) SetTransports(direct, relay Transport) {
	if direct != nil {
		w.directTransport = direct
	}
	if relay != nil {
		w.relayTransport = relay
	}
}

func (w *DeliveryWorker) SetEncryption(enc *crypto.EnvelopeEncryption) {
	w.encryption = enc
}

// IsDraining returns true if the worker is draining for shutdown
func (w *DeliveryWorker) IsDraining() bool {
	return w.draining.Load()
}

// Drain initiates graceful shutdown, stops accepting new work, and waits for in-flight jobs (§15.3)
func (w *DeliveryWorker) Drain(gracePeriod time.Duration) bool {
	w.draining.Store(true)
	close(w.stopCh)

	if gracePeriod <= 0 {
		gracePeriod = w.cfg.ShutdownGrace
	}

	done := make(chan struct{})
	go func() {
		w.inFlightWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(gracePeriod):
		return false // Timed out
	}
}

// ProcessRecipient executes the complete delivery pipeline for a single recipient (§15)
func (w *DeliveryWorker) ProcessRecipient(ctx context.Context, recipientID string) error {
	if w.draining.Load() {
		return errors.New("worker is draining")
	}

	w.inFlightWg.Add(1)
	defer w.inFlightWg.Done()

	// 1. Load recipient
	rcpt, err := w.repo.GetRecipient(ctx, recipientID)
	if err != nil {
		return fmt.Errorf("recipient not found: %w", err)
	}

	// 2. Validate state: skip if already terminal or currently deferred to future
	if rcpt.Status == "delivered" || rcpt.Status == "bounced" || rcpt.Status == "suppressed" || rcpt.Status == "failed" {
		return nil
	}
	if rcpt.NextAttemptAt != nil && time.Now().Before(*rcpt.NextAttemptAt) {
		return nil // Not yet due for retry
	}

	// 3. Load parent message
	msg, err := w.repo.GetMessageByID(ctx, rcpt.TenantID, rcpt.MessageID)
	if err != nil {
		return fmt.Errorf("message not found: %w", err)
	}
	if msg.Status == "cancelled" || (msg.ExpiresAt != nil && time.Now().After(*msg.ExpiresAt)) {
		return nil
	}

	// 4. Re-check suppression (§15 step 3: "this check ships in Phase 5, not later")
	isSuppressed, suppReason, err := w.repo.CheckSuppression(ctx, rcpt.TenantID, msg.StreamID, rcpt.Address)
	if err == nil && isSuppressed {
		_ = w.repo.UpdateRecipientDelivery(ctx, rcpt.ID, "suppressed", rcpt.Attempts, nil, nil, nil, nil, nil, &suppReason)
		_ = w.repo.AddMessageEvent(ctx, &domain.MessageEvent{
			ID:          crypto.NewULID(crypto.PrefixEvent),
			TenantID:    rcpt.TenantID,
			MessageID:   msg.ID,
			RecipientID: &rcpt.ID,
			Type:        "message.suppressed",
			Data:        fmt.Sprintf(`{"reason":"%s","address":"%s"}`, suppReason, rcpt.Address),
			OccurredAt:  time.Now().UTC(),
			CreatedAt:   time.Now().UTC(),
		})
		return nil
	}

	// 5. Select route (direct MX vs relay, source IP, EHLO hostname)
	route, err := w.router.SelectRoute(ctx, rcpt.TenantID, msg.StreamID, msg.DomainID)
	if err != nil {
		return fmt.Errorf("route selection failed: %w", err)
	}

	// 6. Record message.sending event
	_ = w.repo.AddMessageEvent(ctx, &domain.MessageEvent{
		ID:          crypto.NewULID(crypto.PrefixEvent),
		TenantID:    rcpt.TenantID,
		MessageID:   msg.ID,
		RecipientID: &rcpt.ID,
		Type:        "message.sending",
		Data:        fmt.Sprintf(`{"transport":"%s","attempt":%d}`, route.Transport, rcpt.Attempts+1),
		OccurredAt:  time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	})

	// 7. Load raw MIME or construct from body
	var rawMIME []byte
	if msg.RawMIMERef != nil {
		rawMIME, _ = w.storage.Get(ctx, *msg.RawMIMERef)
	}
	if len(rawMIME) == 0 && msg.BodyRef != nil {
		bodyData, _ := w.storage.Get(ctx, *msg.BodyRef)
		content := string(bodyData)
		if strings.Contains(content, "<html") || strings.Contains(content, "<div") || strings.Contains(content, "<body") {
			rawMIME = mime.BuildMessage(msg.FromAddress, rcpt.Address, msg.Subject, "", content)
		} else {
			rawMIME = mime.BuildMessage(msg.FromAddress, rcpt.Address, msg.Subject, content, "")
		}
	}
	if len(rawMIME) == 0 {
		rawMIME = mime.BuildMessage(msg.FromAddress, rcpt.Address, msg.Subject, "Hello", "")
	}

	// 8. Resolve domain name for DKIM & headers
	domainName := "mailvex.local"
	fromParts := strings.Split(msg.FromAddress, "@")
	if len(fromParts) == 2 {
		domainName = fromParts[1]
	}

	verpToken := rcpt.ID
	if rcpt.VERPToken != nil && *rcpt.VERPToken != "" {
		verpToken = *rcpt.VERPToken
	}

	// 9. Apply Header Hygiene, VERP, List-Unsubscribe (§15 step 8, §17.2, §17.3)
	stream, _ := w.repo.GetStream(ctx, rcpt.TenantID, msg.StreamID)
	streamName := "transactional"
	if stream != nil {
		streamName = stream.Name
	}

	bounceHost := domainName
	if w.cfg.BounceDomain != "" && !strings.HasSuffix(w.cfg.BounceDomain, ".local") {
		if strings.HasSuffix(w.cfg.BounceDomain, "."+domainName) || w.cfg.BounceDomain == domainName {
			bounceHost = w.cfg.BounceDomain
		}
	}
	returnPath := fmt.Sprintf("bounce+%s@%s", verpToken, bounceHost)

	ehloHost := w.cfg.PlatformHost
	if route.EHLOHostname != "" {
		ehloHost = route.EHLOHostname
	}

	hygieneParams := HygieneParams{
		TenantID:      rcpt.TenantID,
		StreamName:    streamName,
		SendingDomain: domainName,
		PlatformHost:  ehloHost,
		BounceDomain:  bounceHost,
		RecipientID:   rcpt.ID,
		RecipientAddr: rcpt.Address,
		VERPToken:     verpToken,
		IsBroadcast:   (streamName == "broadcast"),
		CampaignTag:   "default",
	}
	if len(msg.Tags) > 0 {
		hygieneParams.CampaignTag = msg.Tags[0]
	}

	sanitizedMIME, err := SanitizeAndDecorateMIME(rawMIME, hygieneParams)
	if err != nil {
		return fmt.Errorf("header hygiene failed: %w", err)
	}

	// 10. DKIM Signing (§15 step 10)
	finalMIME := sanitizedMIME
	dkimKey, err := w.repo.GetActiveDKIMKey(ctx, msg.DomainID)
	if err == nil && dkimKey != nil && dkimKey.EncryptedPrivateKey != "" {
		privKeyPEM := dkimKey.EncryptedPrivateKey
		if !strings.HasPrefix(strings.TrimSpace(privKeyPEM), "-----BEGIN") && w.encryption != nil {
			if tKey, tErr := w.repo.GetActiveTenantKey(ctx, msg.TenantID); tErr == nil && tKey != nil {
				if dek, dErr := w.encryption.DecryptDEK(tKey.EncryptedDEK); dErr == nil {
					if dec, sErr := crypto.DecryptSecret(dek, privKeyPEM); sErr == nil {
						privKeyPEM = string(dec)
					}
				}
			}
		}
		signedMIME, err := crypto.SignDKIM(sanitizedMIME, domainName, dkimKey.Selector, privKeyPEM)
		if err == nil {
			finalMIME = signedMIME
		}
	}

	// 11. Prepare Delivery Request
	delReq := &DeliveryRequest{
		RecipientID:   rcpt.ID,
		RecipientAddr: rcpt.Address,
		ReturnPath:    returnPath,
		MIMEData:      finalMIME,
		EHLOHostname:  ehloHost,
		SourceIP:      route.SourceIP,
		TLSPolicy:     "opportunistic",
	}

	var delResp *DeliveryResponse

	if route.Transport == "relay" {
		delReq.Relay = route.Relay
		if route.Relay.AuthPassEncrypted != nil {
			delReq.RelayAuthPass = *route.Relay.AuthPassEncrypted // in production decrypted via KEK
		}
		delResp, err = w.relayTransport.Deliver(ctx, delReq)
	} else {
		// Direct MX resolution
		mxRecords, err := w.resolver.Resolve(ctx, rcpt.AddressDomain)
		if err != nil {
			// RFC 7505 Null MX or permanent DNS resolution failure -> hard bounce
			delResp = &DeliveryResponse{
				Success:      false,
				SMTPCode:     "550",
				EnhancedCode: "5.1.2",
				ResponseText: fmt.Sprintf("MX resolution failed: %v", err),
				RemoteMX:     rcpt.AddressDomain,
			}
		} else {
			// Pick highest preference MX
			primaryMX := mxRecords[0].Host
			delReq.RemoteMX = primaryMX
			delReq.RemotePort = w.cfg.DefaultPort
			delResp, err = w.directTransport.Deliver(ctx, delReq)
		}
	}

	if err != nil {
		delResp = &DeliveryResponse{
			Success:      false,
			SMTPCode:     "451",
			EnhancedCode: "4.4.0",
			ResponseText: err.Error(),
		}
	}

	// 12. Record delivery attempt in delivery_attempts table
	attemptNo := rcpt.Attempts + 1
	outcome := "delivered"
	if !delResp.Success {
		outcome = "deferred"
		if strings.HasPrefix(delResp.SMTPCode, "5") {
			outcome = "bounced"
		}
	}

	_ = w.repo.RecordDeliveryAttempt(ctx, &domain.DeliveryAttempt{
		ID:               crypto.NewULID("att_"),
		RecipientID:      rcpt.ID,
		TenantID:         rcpt.TenantID,
		AttemptNo:        attemptNo,
		IPID:             route.IPID,
		NodeID:           &w.cfg.NodeID,
		RemoteMX:         &delResp.RemoteMX,
		RemoteIP:         &delResp.RemoteIP,
		TLSVersion:       &delResp.TLSVersion,
		TLSCipher:        &delResp.TLSCipher,
		TLSPolicyApplied: &delResp.TLSPolicyApplied,
		SMTPCode:         &delResp.SMTPCode,
		EnhancedCode:     &delResp.EnhancedCode,
		SMTPResponse:     &delResp.ResponseText,
		DurationMS:       int(delResp.Duration.Milliseconds()),
		StartedAt:        time.Now().UTC().Add(-delResp.Duration),
		Outcome:          outcome,
		CreatedAt:        time.Now().UTC(),
	})

	now := time.Now().UTC()

	// 13. Process Delivery Outcome
	if delResp.Success {
		// 2xx Success: Delivered
		_ = w.repo.UpdateRecipientDelivery(ctx, rcpt.ID, "delivered", attemptNo, &delResp.SMTPCode, &delResp.ResponseText, &now, nil, nil, nil)
		_ = w.repo.AddMessageEvent(ctx, &domain.MessageEvent{
			ID:          crypto.NewULID(crypto.PrefixEvent),
			TenantID:    rcpt.TenantID,
			MessageID:   msg.ID,
			RecipientID: &rcpt.ID,
			Type:        "message.delivered",
			Data:        fmt.Sprintf(`{"smtp_code":"%s","response":"%s"}`, delResp.SMTPCode, delResp.ResponseText),
			OccurredAt:  now,
			CreatedAt:   now,
		})
		campaignTag := hygieneParams.CampaignTag
		_ = w.repo.RecordAnalyticsEvent(ctx, rcpt.TenantID, "message.delivered", &msg.StreamID, &msg.DomainID, rcpt.AddressDomain, campaignTag, delResp.Duration.Seconds(), now)
		_ = w.updateMessageSummary(ctx, msg.TenantID, msg.ID)
		return nil
	}

	// Delivery Failed: Classify bounce
	classification := w.classifier.Classify(delResp.SMTPCode, delResp.EnhancedCode, delResp.ResponseText)
	campaignTag := hygieneParams.CampaignTag

	if classification.Classification == "hard_bounce" || classification.Action == "suppress_immediate" {
		// 5xx Hard Bounce: suppress immediately & dead letter (§15 step 13, §16, §18)
		_ = w.repo.AddSuppression(ctx, &domain.Suppression{
			ID:            crypto.NewULID("supp_"),
			TenantID:      rcpt.TenantID,
			Address:       rcpt.Address,
			AddressDomain: rcpt.AddressDomain,
			Type:          "hard_bounce",
			Reason:        &delResp.ResponseText,
			Source:        "smtp",
			CreatedAt:     now,
		})

		_ = w.repo.UpdateRecipientDelivery(ctx, rcpt.ID, "bounced", attemptNo, &delResp.SMTPCode, &delResp.ResponseText, nil, &now, nil, nil)
		_ = w.repo.AddMessageEvent(ctx, &domain.MessageEvent{
			ID:          crypto.NewULID(crypto.PrefixEvent),
			TenantID:    rcpt.TenantID,
			MessageID:   msg.ID,
			RecipientID: &rcpt.ID,
			Type:        "message.bounced",
			Data:        fmt.Sprintf(`{"classification":"%s","reason":"%s"}`, classification.Classification, delResp.ResponseText),
			OccurredAt:  now,
			CreatedAt:   now,
		})
		_ = w.repo.RecordAnalyticsEvent(ctx, rcpt.TenantID, "hard_bounce", &msg.StreamID, &msg.DomainID, rcpt.AddressDomain, campaignTag, 0, now)

		// Record to DLQ (§15.2)
		_ = w.repo.RecordDeadLetter(ctx, &domain.DeadLetterMessage{
			ID:               crypto.NewULID("dlq_"),
			TenantID:         rcpt.TenantID,
			MessageID:        msg.ID,
			RecipientID:      rcpt.ID,
			Destination:      rcpt.Address,
			Reason:           classification.Classification,
			LastSMTPCode:     &delResp.SMTPCode,
			LastEnhancedCode: &delResp.EnhancedCode,
			LastResponse:     &delResp.ResponseText,
			Attempts:         attemptNo,
			SourceIP:         &route.SourceIP,
			NodeID:           &w.cfg.NodeID,
			CreatedAt:        now,
		})

	} else if classification.Classification == "soft_bounce" || classification.Action == "retry" {
		// 4xx Soft Bounce / Rate Limited: schedule retry with backoff + jitter (§15.1)
		nextDelay, canRetry := w.retry.CalculateNextAttempt(attemptNo)
		if canRetry {
			nextAttempt := now.Add(nextDelay)
			_ = w.repo.UpdateRecipientDelivery(ctx, rcpt.ID, "deferred", attemptNo, &delResp.SMTPCode, &delResp.ResponseText, nil, nil, &nextAttempt, nil)
			_ = w.repo.AddMessageEvent(ctx, &domain.MessageEvent{
				ID:          crypto.NewULID(crypto.PrefixEvent),
				TenantID:    rcpt.TenantID,
				MessageID:   msg.ID,
				RecipientID: &rcpt.ID,
				Type:        "message.deferred",
				Data:        fmt.Sprintf(`{"attempt":%d,"next_attempt_in_seconds":%d,"reason":"%s"}`, attemptNo, int(nextDelay.Seconds()), delResp.ResponseText),
				OccurredAt:  now,
				CreatedAt:   now,
			})
			_ = w.repo.RecordAnalyticsEvent(ctx, rcpt.TenantID, "soft_bounce", &msg.StreamID, &msg.DomainID, rcpt.AddressDomain, campaignTag, 0, now)
		} else {
			// Retries exhausted -> failed and move to DLQ
			_ = w.repo.UpdateRecipientDelivery(ctx, rcpt.ID, "failed", attemptNo, &delResp.SMTPCode, &delResp.ResponseText, nil, nil, nil, nil)
			_ = w.repo.AddMessageEvent(ctx, &domain.MessageEvent{
				ID:          crypto.NewULID(crypto.PrefixEvent),
				TenantID:    rcpt.TenantID,
				MessageID:   msg.ID,
				RecipientID: &rcpt.ID,
				Type:        "message.failed",
				Data:        fmt.Sprintf(`{"attempts":%d,"reason":"max retries exceeded"}`, attemptNo),
				OccurredAt:  now,
				CreatedAt:   now,
			})
			_ = w.repo.RecordDeadLetter(ctx, &domain.DeadLetterMessage{
				ID:               crypto.NewULID("dlq_"),
				TenantID:         rcpt.TenantID,
				MessageID:        msg.ID,
				RecipientID:      rcpt.ID,
				Destination:      rcpt.Address,
				Reason:           "max_retries_exceeded",
				LastSMTPCode:     &delResp.SMTPCode,
				LastEnhancedCode: &delResp.EnhancedCode,
				LastResponse:     &delResp.ResponseText,
				Attempts:         attemptNo,
				SourceIP:         &route.SourceIP,
				NodeID:           &w.cfg.NodeID,
				CreatedAt:        now,
			})
		}
	} else {
		// Other fatal policy/auth rejection -> fail immediately and DLQ
		_ = w.repo.UpdateRecipientDelivery(ctx, rcpt.ID, "failed", attemptNo, &delResp.SMTPCode, &delResp.ResponseText, nil, nil, nil, nil)
		_ = w.repo.AddMessageEvent(ctx, &domain.MessageEvent{
			ID:          crypto.NewULID(crypto.PrefixEvent),
			TenantID:    rcpt.TenantID,
			MessageID:   msg.ID,
			RecipientID: &rcpt.ID,
			Type:        "message.failed",
			Data:        fmt.Sprintf(`{"reason":"%s"}`, delResp.ResponseText),
			OccurredAt:  now,
			CreatedAt:   now,
		})
		_ = w.repo.RecordDeadLetter(ctx, &domain.DeadLetterMessage{
			ID:               crypto.NewULID("dlq_"),
			TenantID:         rcpt.TenantID,
			MessageID:        msg.ID,
			RecipientID:      rcpt.ID,
			Destination:      rcpt.Address,
			Reason:           classification.Classification,
			LastSMTPCode:     &delResp.SMTPCode,
			LastEnhancedCode: &delResp.EnhancedCode,
			LastResponse:     &delResp.ResponseText,
			Attempts:         attemptNo,
			SourceIP:         &route.SourceIP,
			NodeID:           &w.cfg.NodeID,
			CreatedAt:        now,
		})
	}

	_ = w.updateMessageSummary(ctx, msg.TenantID, msg.ID)
	return nil
}

// updateMessageSummary recalculates message level status from recipient statuses (§13.2)
func (w *DeliveryWorker) updateMessageSummary(ctx context.Context, tenantID, messageID string) error {
	rcpts, err := w.repo.GetMessageRecipients(ctx, tenantID, messageID)
	if err != nil || len(rcpts) == 0 {
		return err
	}

	allDelivered := true
	hasDelivered := false
	allTerminal := true

	for _, r := range rcpts {
		if r.Status != "delivered" {
			allDelivered = false
		} else {
			hasDelivered = true
		}
		if r.Status == "queued" || r.Status == "deferred" || r.Status == "sending" {
			allTerminal = false
		}
	}

	newStatus := "sending"
	if allDelivered {
		newStatus = "delivered"
	} else if allTerminal && !hasDelivered {
		newStatus = "failed"
	} else if hasDelivered && !allDelivered {
		newStatus = "partially_delivered"
	}

	return w.repo.UpdateMessageStatus(ctx, tenantID, messageID, newStatus)
}

// ProcessOutboxBatch processes pending items from the outbox relay (§14.2)
func (w *DeliveryWorker) ProcessOutboxBatch(ctx context.Context, limit int) (int, error) {
	items, err := w.repo.ListPendingOutbox(ctx, limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, item := range items {
		var payload struct {
			MessageID string `json:"message_id"`
			TenantID  string `json:"tenant_id"`
		}
		if err := json.Unmarshal([]byte(item.Payload), &payload); err == nil {
			rcpts, err := w.repo.GetMessageRecipients(ctx, payload.TenantID, payload.MessageID)
			if err == nil {
				hasActive := false
				for _, r := range rcpts {
					if r.Status == "queued" || r.Status == "deferred" {
						hasActive = true
						_ = w.ProcessRecipient(ctx, r.ID)
					}
				}
				if !hasActive {
					_ = w.updateMessageSummary(ctx, payload.TenantID, payload.MessageID)
				}
			}
		}
		_ = w.repo.MarkOutboxPublished(ctx, item.ID)
		processed++
	}

	return processed, nil
}
