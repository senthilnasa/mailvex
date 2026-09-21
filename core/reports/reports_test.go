package reports

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

func setupTestEnvironment() (repository.Repository, *ReportProcessor, string, string, string) {
	repo := repository.NewMemoryRepository()
	verpSecret := []byte("test-verp-secret-key-32b-long!!")
	unsubSecret := []byte("test-unsub-secret-key-32b-long!")
	processor := NewReportProcessor(repo, verpSecret, unsubSecret)

	orgID := "org_test_p6"
	tenantID := "tenant_test_p6"
	msgID := "msg_test_p6"
	rcptID := "rcpt_test_p6"

	_ = repo.CreateOrganization(context.Background(), &domain.Organization{
		ID:        orgID,
		Name:      "Test Org",
		CreatedAt: time.Now().UTC(),
	})

	_ = repo.CreateTenant(context.Background(), &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Test Tenant",
		CreatedAt:      time.Now().UTC(),
	})

	_ = repo.SaveMessage(context.Background(),
		&domain.Message{
			ID:        msgID,
			TenantID:  tenantID,
			StreamID:  "stream_broadcast",
			Status:    "processing",
			CreatedAt: time.Now().UTC(),
		},
		[]*domain.MessageRecipient{
			{
				ID:            rcptID,
				TenantID:      tenantID,
				MessageID:     msgID,
				Address:       "bounce-victim@recipient.test",
				AddressDomain: "recipient.test",
				Status:        "submitted",
				CreatedAt:     time.Now().UTC(),
			},
		},
		nil,
		nil,
	)

	return repo, processor, tenantID, msgID, rcptID
}

func TestRFC3464DSNHardBounce(t *testing.T) {
	repo, processor, tenantID, msgID, rcptID := setupTestEnvironment()
	ctx := context.Background()

	verpToken := crypto.GenerateVERPToken(rcptID, processor.verpSecret)
	bounceAddr := fmt.Sprintf("bounce+%s@bounce.mailvex.local", verpToken)

	dsnMIME := fmt.Sprintf(`From: MAILER-DAEMON@mta.remote.test
To: <%s>
Subject: Delivery Status Notification (Failure)
MIME-Version: 1.0
Content-Type: multipart/report; report-type=delivery-status; boundary="boundary-dsn-1"

--boundary-dsn-1
Content-Type: text/plain; charset=utf-8

Your message could not be delivered to bounce-victim@recipient.test.

--boundary-dsn-1
Content-Type: message/delivery-status

Reporting-MTA: dns; mta.remote.test
Final-Recipient: rfc822; bounce-victim@recipient.test
Action: failed
Status: 5.1.1
Diagnostic-Code: smtp; 550 5.1.1 User unknown

--boundary-dsn-1
Content-Type: message/rfc822

Return-Path: <%s>
From: sender@domain.test
To: bounce-victim@recipient.test
Subject: Test Newsletter
Message-ID: <%s@domain.test>

Hello world!
--boundary-dsn-1--
`, bounceAddr, bounceAddr, msgID)

	res, err := processor.ProcessInboundDSN(ctx, []byte(dsnMIME), bounceAddr)
	if err != nil {
		t.Fatalf("unexpected error processing DSN: %v", err)
	}

	if res.IsAutoReply {
		t.Fatalf("expected real bounce, got auto-reply")
	}
	if res.IsUnmatched {
		t.Fatalf("expected matched bounce, got unmatched")
	}
	if res.RecipientID != rcptID {
		t.Fatalf("expected recipient %s, got %s", rcptID, res.RecipientID)
	}
	if !res.Suppressed {
		t.Fatalf("expected hard bounce to be automatically suppressed")
	}

	// Verify recipient status in repository
	rcpt, err := repo.GetRecipient(ctx, rcptID)
	if err != nil || rcpt.Status != "bounced" {
		t.Fatalf("expected recipient status 'bounced', got '%s'", rcpt.Status)
	}

	// Verify suppression entry
	suppressed, reason, err := repo.CheckSuppression(ctx, tenantID, "stream_broadcast", "bounce-victim@recipient.test")
	if err != nil || !suppressed {
		t.Fatalf("expected recipient to be suppressed, err: %v", err)
	}
	if !strings.Contains(reason, "5.1.1") {
		t.Fatalf("expected suppression reason to contain 5.1.1, got: %s", reason)
	}

	// Verify DLQ entry
	dlqs, err := repo.ListDeadLetterMessages(ctx, tenantID, 10)
	if err != nil || len(dlqs) == 0 {
		t.Fatalf("expected DLQ entry for hard bounce")
	}
	if dlqs[0].RecipientID != rcptID {
		t.Fatalf("expected DLQ recipient %s, got %s", rcptID, dlqs[0].RecipientID)
	}
}

func TestRFC3464DSNSoftBounce(t *testing.T) {
	repo, processor, tenantID, msgID, rcptID := setupTestEnvironment()
	ctx := context.Background()

	verpToken := crypto.GenerateVERPToken(rcptID, processor.verpSecret)
	bounceAddr := fmt.Sprintf("bounce+%s@bounce.mailvex.local", verpToken)

	dsnMIME := fmt.Sprintf(`From: MAILER-DAEMON@mta.remote.test
To: <%s>
Subject: Delivery Status Notification (Delay)
MIME-Version: 1.0
Content-Type: multipart/report; report-type=delivery-status; boundary="boundary-dsn-2"

--boundary-dsn-2
Content-Type: text/plain

Delivery delayed due to mailbox full.

--boundary-dsn-2
Content-Type: message/delivery-status

Final-Recipient: rfc822; bounce-victim@recipient.test
Action: delayed
Status: 4.2.2
Diagnostic-Code: smtp; 452 4.2.2 Mailbox full

--boundary-dsn-2
Content-Type: message/rfc822

Return-Path: <%s>
Subject: Test Newsletter
Message-ID: <%s@domain.test>

Hello!
--boundary-dsn-2--
`, bounceAddr, bounceAddr, msgID)

	res, err := processor.ProcessInboundDSN(ctx, []byte(dsnMIME), bounceAddr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Suppressed {
		t.Fatalf("soft bounce (4.2.2) should NOT be suppressed")
	}

	rcpt, _ := repo.GetRecipient(ctx, rcptID)
	if rcpt.Status != "deferred" {
		t.Fatalf("expected recipient status 'deferred', got '%s'", rcpt.Status)
	}

	suppressed, _, _ := repo.CheckSuppression(ctx, tenantID, "stream_broadcast", "bounce-victim@recipient.test")
	if suppressed {
		t.Fatalf("address must not be suppressed on 4.2.2 soft bounce")
	}
}

func TestAutoReplyDetectionAndFiltering(t *testing.T) {
	repo, processor, tenantID, _, rcptID := setupTestEnvironment()
	ctx := context.Background()

	tests := []struct {
		name string
		mime string
	}{
		{
			name: "RFC 3834 Auto-Submitted header",
			mime: "From: oof@recipient.test\r\nTo: bounce@mailvex.local\r\nAuto-Submitted: auto-replied\r\nSubject: Away\r\n\r\nI am away on holiday.",
		},
		{
			name: "Microsoft X-Autoreply header",
			mime: "From: oof@recipient.test\r\nTo: bounce@mailvex.local\r\nX-Autoreply: yes\r\nSubject: Re: Newsletter\r\n\r\nOut of office notice.",
		},
		{
			name: "Precedence auto_reply",
			mime: "From: oof@recipient.test\r\nTo: bounce@mailvex.local\r\nPrecedence: auto_reply\r\nSubject: Notice\r\n\r\nAutomatic reply.",
		},
		{
			name: "Subject keyword Out of Office",
			mime: "From: oof@recipient.test\r\nTo: bounce@mailvex.local\r\nSubject: Automatic reply: Out of Office until Monday\r\n\r\nI will review your message when I return.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := processor.ProcessInboundDSN(ctx, []byte(tc.mime), "bounce@mailvex.local")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !res.IsAutoReply {
				t.Fatalf("expected IsAutoReply to be true for %s", tc.name)
			}
			if res.Suppressed {
				t.Fatalf("auto-reply must never trigger suppression")
			}
			// Verify recipient untouched
			rcpt, _ := repo.GetRecipient(ctx, rcptID)
			if rcpt.Status == "bounced" {
				t.Fatalf("auto-reply must not mark recipient bounced")
			}
			supp, _, _ := repo.CheckSuppression(ctx, tenantID, "", "bounce-victim@recipient.test")
			if supp {
				t.Fatalf("auto-reply must not add suppression")
			}
		})
	}
}

func TestUnmatchedBounceStored(t *testing.T) {
	repo, processor, _, _, _ := setupTestEnvironment()
	ctx := context.Background()

	rawMIME := `From: MAILER-DAEMON@unknown.net
To: mystery-bounce@domain.test
Subject: Undeliverable: Unknown Subject
Date: Mon, 21 Sep 2026 06:00:00 +0000

Failed delivery to totally-unknown-person@nowhere.test
550 5.1.1 Unknown user
`

	res, err := processor.ProcessInboundDSN(ctx, []byte(rawMIME), "mystery-bounce@domain.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsUnmatched {
		t.Fatalf("expected IsUnmatched to be true")
	}

	unmatched, err := repo.ListUnmatchedBounces(ctx, 10)
	if err != nil || len(unmatched) == 0 {
		t.Fatalf("expected entry in unmatched_bounces table")
	}
}

func TestRFC5965ARFFeedbackReport(t *testing.T) {
	repo, processor, tenantID, msgID, rcptID := setupTestEnvironment()
	ctx := context.Background()

	arfMIME := fmt.Sprintf(`From: fbl@isp.example
To: fbl@mailvex.local
Subject: Staff report: spam
MIME-Version: 1.0
Content-Type: multipart/report; report-type=feedback-report; boundary="boundary-arf-1"

--boundary-arf-1
Content-Type: text/plain; charset=utf-8

This is an email abuse report for an email message received from your platform.

--boundary-arf-1
Content-Type: message/feedback-report

Feedback-Type: abuse
User-Agent: ISP-Feedback-Engine/2.0
Reported-Domain: recipient.test
Original-Rcpt-To: bounce-victim@recipient.test
Arrival-Date: Mon, 21 Sep 2026 06:10:00 +0000

--boundary-arf-1
Content-Type: message/rfc822

From: sender@domain.test
To: bounce-victim@recipient.test
Subject: Exclusive Offers
Message-ID: <%s@domain.test>
Feedback-ID: %s:broadcast:promo2026:mailvex

Buy now!
--boundary-arf-1--
`, msgID, tenantID)

	res, err := processor.ProcessInboundARF(ctx, []byte(arfMIME))
	if err != nil {
		t.Fatalf("unexpected error processing ARF: %v", err)
	}

	if res.TenantID != tenantID {
		t.Fatalf("expected tenant %s, got %s", tenantID, res.TenantID)
	}
	if !res.Suppressed {
		t.Fatalf("expected recipient to be suppressed after spam complaint")
	}

	// Verify suppression entry
	suppressed, reason, err := repo.CheckSuppression(ctx, tenantID, "stream_broadcast", "bounce-victim@recipient.test")
	if err != nil || !suppressed {
		t.Fatalf("expected recipient address to be suppressed, err: %v", err)
	}
	if !strings.Contains(reason, "complaint") {
		t.Fatalf("expected suppression reason to reference complaint, got: %s", reason)
	}

	// Verify FBL report entry
	fbls, err := repo.ListFBLReports(ctx, tenantID, 10)
	if err != nil || len(fbls) == 0 {
		t.Fatalf("expected FBL report entry in repository")
	}
	if fbls[0].FeedbackType != "abuse" {
		t.Fatalf("expected feedback_type 'abuse', got '%s'", fbls[0].FeedbackType)
	}
	if fbls[0].RecipientID != nil && *fbls[0].RecipientID != rcptID {
		t.Fatalf("expected correlated recipient %s, got %s", rcptID, *fbls[0].RecipientID)
	}
}

func TestMailtoUnsubscribe(t *testing.T) {
	repo, processor, tenantID, _, rcptID := setupTestEnvironment()
	ctx := context.Background()

	rcpt, _ := repo.GetRecipient(ctx, rcptID)

	mac := hmac.New(sha256.New, processor.unsubSecret)
	mac.Write([]byte(fmt.Sprintf("%s:%s:%s", tenantID, rcptID, rcpt.Address)))
	sig := hex.EncodeToString(mac.Sum(nil))[:24]
	token := fmt.Sprintf("%s.%s", rcptID, sig)

	envelope := fmt.Sprintf("unsubscribe+%s@bounce.mailvex.local", token)

	err := processor.ProcessMailtoUnsubscribe(ctx, envelope, []byte("Subject: Unsubscribe\r\n\r\nPlease unsubscribe me."))
	if err != nil {
		t.Fatalf("failed to process mailto unsubscribe: %v", err)
	}

	// Verify suppression
	suppressed, reason, err := repo.CheckSuppression(ctx, tenantID, "stream_broadcast", rcpt.Address)
	if err != nil || !suppressed {
		t.Fatalf("expected recipient address to be suppressed via mailto")
	}
	if !strings.Contains(strings.ToLower(reason), "unsubscribe") {
		t.Fatalf("expected unsubscribe reason, got: %s", reason)
	}
}

func TestAbusePostmasterIntake(t *testing.T) {
	repo, processor, tenantID, _, _ := setupTestEnvironment()
	ctx := context.Background()

	// Register domain so domain correlation works
	_ = repo.CreateDomain(ctx, &domain.Domain{
		ID:        "dom_test_p6",
		TenantID:  tenantID,
		Name:      "company.test",
		Status:    "verified",
		CreatedAt: time.Now().UTC(),
	})

	report, err := processor.ProcessAbuseIntake(ctx, "victim@external.test", "abuse@company.test", "Phishing report", []byte("Raw abuse email body"))
	if err != nil {
		t.Fatalf("failed to process abuse intake: %v", err)
	}

	if report.Status != "pending" {
		t.Fatalf("expected initial status 'pending', got '%s'", report.Status)
	}
	if report.DomainID == nil || *report.DomainID != "dom_test_p6" {
		t.Fatalf("expected domain correlation to dom_test_p6")
	}
	if report.TenantID == nil || *report.TenantID != tenantID {
		t.Fatalf("expected tenant correlation to %s", tenantID)
	}

	// Verify list abuse reports
	list, err := repo.ListAbuseReports(ctx, "pending", 10)
	if err != nil || len(list) == 0 {
		t.Fatalf("expected pending report in list")
	}

	// Update status to resolved
	notes := "Investigated and cleared."
	err = repo.UpdateAbuseReportStatus(ctx, report.ID, "resolved", &notes)
	if err != nil {
		t.Fatalf("failed to resolve report: %v", err)
	}

	updated, err := repo.GetAbuseReport(ctx, report.ID)
	if err != nil || updated.Status != "resolved" || updated.ResolvedAt == nil {
		t.Fatalf("expected report resolved with timestamp")
	}
}
