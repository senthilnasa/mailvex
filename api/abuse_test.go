package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/audit"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

func setupAbuseTestEnvironment(t *testing.T) (*Server, repository.Repository, string, string, string) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	kek := bytes.Repeat([]byte{0x04}, 32)
	enc, _ := crypto.NewEnvelopeEncryption(kek)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	server := NewServer(repo, auditSvc, enc, logger)

	ctx := context.Background()

	orgID := crypto.NewULID(crypto.PrefixOrg)
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Abuse Test Org",
		Slug:      "abuse-test-org",
		CreatedAt: time.Now().UTC(),
	})

	tenantID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Abuse Test Tenant",
		Slug:           "abuse-test-tenant",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	userID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(ctx, &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		Email:          "admin@abuse-test.local",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	_ = repo.AddTenantMembership(ctx, &domain.TenantMembership{
		ID:        crypto.NewULID(crypto.PrefixMembership),
		TenantID:  tenantID,
		UserID:    userID,
		RoleID:    "role_tenant_owner",
		CreatedAt: time.Now().UTC(),
	})

	domID := crypto.NewULID(crypto.PrefixDomain)
	_ = repo.CreateDomain(ctx, &domain.Domain{
		ID:        domID,
		TenantID:  tenantID,
		Name:      "abuse-test.local",
		Status:    "verified",
		CreatedAt: time.Now().UTC(),
	})

	return server, repo, tenantID, userID, domID
}

func TestAbuseReportLifecycle(t *testing.T) {
	server, repo, tenantID, userID, domID := setupAbuseTestEnvironment(t)
	ctx := context.Background()

	// Seed an abuse report
	reportID := "abuse_test_01"
	subj := "Spam report"
	_ = repo.SaveAbuseReport(ctx, &domain.AbuseReport{
		ID:        reportID,
		TenantID:  &tenantID,
		DomainID:  &domID,
		Sender:    "spammer@external.test",
		Recipient: "abuse@abuse-test.local",
		Subject:   &subj,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	})

	// 1. GET /v1/abuse-reports
	reqList := httptest.NewRequest(http.MethodGet, "/v1/abuse-reports?status=pending", nil)
	reqList.Header.Set("X-User-ID", userID)
	reqList.Header.Set("X-Tenant-ID", tenantID)
	wList := httptest.NewRecorder()
	server.Handler().ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /v1/abuse-reports, got %d: %s", wList.Code, wList.Body.String())
	}
	var listResp map[string][]domain.AbuseReport
	_ = json.Unmarshal(wList.Body.Bytes(), &listResp)
	if len(listResp["data"]) == 0 {
		t.Fatalf("expected at least 1 abuse report in list")
	}

	// 2. GET /v1/abuse-reports/{id}
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/abuse-reports/"+reportID, nil)
	reqGet.Header.Set("X-User-ID", userID)
	reqGet.Header.Set("X-Tenant-ID", tenantID)
	wGet := httptest.NewRecorder()
	server.Handler().ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /v1/abuse-reports/{id}, got %d", wGet.Code)
	}

	// 3. POST /v1/abuse-reports/{id}/resolve
	resolveBody := `{"notes":"Verified legitimate traffic, customer whitelisted"}`
	reqResolve := httptest.NewRequest(http.MethodPost, "/v1/abuse-reports/"+reportID+"/resolve", strings.NewReader(resolveBody))
	reqResolve.Header.Set("X-User-ID", userID)
	reqResolve.Header.Set("X-Tenant-ID", tenantID)
	reqResolve.Header.Set("Content-Type", "application/json")
	wResolve := httptest.NewRecorder()
	server.Handler().ServeHTTP(wResolve, reqResolve)

	if wResolve.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on resolve, got %d: %s", wResolve.Code, wResolve.Body.String())
	}

	var resolvedReport domain.AbuseReport
	_ = json.Unmarshal(wResolve.Body.Bytes(), &resolvedReport)
	if resolvedReport.Status != "resolved" {
		t.Fatalf("expected status 'resolved', got '%s'", resolvedReport.Status)
	}
	if resolvedReport.ResolvedAt == nil {
		t.Fatalf("expected ResolvedAt timestamp to be set")
	}

	// 4. POST /v1/abuse-reports/{id}/escalate
	escalateBody := `{"notes":"Malicious volume detected, credentials revoked"}`
	reqEscalate := httptest.NewRequest(http.MethodPost, "/v1/abuse-reports/"+reportID+"/escalate", strings.NewReader(escalateBody))
	reqEscalate.Header.Set("X-User-ID", userID)
	reqEscalate.Header.Set("X-Tenant-ID", tenantID)
	reqEscalate.Header.Set("Content-Type", "application/json")
	wEscalate := httptest.NewRecorder()
	server.Handler().ServeHTTP(wEscalate, reqEscalate)

	if wEscalate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on escalate, got %d", wEscalate.Code)
	}
}

func TestReportIngestionWebhookEndpoints(t *testing.T) {
	server, repo, tenantID, userID, domID := setupAbuseTestEnvironment(t)
	ctx := context.Background()

	// Seed message & recipient
	msgID := "msg_p6_api"
	rcptID := "rcpt_p6_api"
	_ = repo.SaveMessage(ctx,
		&domain.Message{
			ID:        msgID,
			TenantID:  tenantID,
			StreamID:  "str_tx",
			Status:    "processing",
			CreatedAt: time.Now().UTC(),
		},
		[]*domain.MessageRecipient{
			{
				ID:            rcptID,
				TenantID:      tenantID,
				MessageID:     msgID,
				Address:       "bounced-api-user@example.test",
				AddressDomain: "example.test",
				Status:        "submitted",
				CreatedAt:     time.Now().UTC(),
			},
		},
		nil, nil,
	)

	// 1. Ingest DSN bounce via POST /v1/reports/dsn
	dsnMIME := `From: mailer-daemon@remote.mta
To: bounce+rcpt_p6_api@bounce.local
Subject: Delivery Status Notification (Failure)
Content-Type: multipart/report; report-type=delivery-status; boundary="b-dsn"

--b-dsn
Content-Type: text/plain

Failed delivery.

--b-dsn
Content-Type: message/delivery-status

Action: failed
Status: 5.1.1
Diagnostic-Code: smtp; 550 5.1.1 User unknown

--b-dsn
Content-Type: message/rfc822

Return-Path: <bounce+rcpt_p6_api@bounce.local>
To: bounced-api-user@example.test
Message-ID: <msg_p6_api@example.test>

Test body
--b-dsn--`

	dsnPayload, _ := json.Marshal(map[string]string{
		"raw_mime":           dsnMIME,
		"envelope_recipient": "bounce+rcpt_p6_api@bounce.local",
	})
	reqDSN := httptest.NewRequest(http.MethodPost, "/v1/reports/dsn", bytes.NewReader(dsnPayload))
	reqDSN.Header.Set("Content-Type", "application/json")
	wDSN := httptest.NewRecorder()
	server.Handler().ServeHTTP(wDSN, reqDSN)

	if wDSN.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on POST /v1/reports/dsn, got %d: %s", wDSN.Code, wDSN.Body.String())
	}

	// 2. Ingest ARF complaint via POST /v1/reports/arf
	arfMIME := fmt.Sprintf(`From: fbl@provider.test
To: fbl@mailvex.local
Subject: Spam report
Content-Type: multipart/report; report-type=feedback-report; boundary="b-arf"

--b-arf
Content-Type: text/plain

Spam reported.

--b-arf
Content-Type: message/feedback-report

Feedback-Type: abuse
User-Agent: MailProvider-FBL/1.0
Reported-Domain: example.test

--b-arf
Content-Type: message/rfc822

Feedback-ID: %s:str_tx:promo:mailvex
Message-ID: <%s@example.test>
To: bounced-api-user@example.test

Hello spam!
--b-arf--`, tenantID, msgID)

	arfPayload, _ := json.Marshal(map[string]string{
		"raw_mime": arfMIME,
	})
	reqARF := httptest.NewRequest(http.MethodPost, "/v1/reports/arf", bytes.NewReader(arfPayload))
	reqARF.Header.Set("Content-Type", "application/json")
	wARF := httptest.NewRecorder()
	server.Handler().ServeHTTP(wARF, reqARF)

	if wARF.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on POST /v1/reports/arf, got %d: %s", wARF.Code, wARF.Body.String())
	}

	// 3. GET /v1/reports/fbl
	reqFBL := httptest.NewRequest(http.MethodGet, "/v1/reports/fbl", nil)
	reqFBL.Header.Set("X-User-ID", userID)
	reqFBL.Header.Set("X-Tenant-ID", tenantID)
	wFBL := httptest.NewRecorder()
	server.Handler().ServeHTTP(wFBL, reqFBL)

	if wFBL.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /v1/reports/fbl, got %d: %s", wFBL.Code, wFBL.Body.String())
	}

	// 4. POST & GET /v1/reports/postmaster
	pmPayload := fmt.Sprintf(`{"domain_id":"%s","spam_rate":0.001,"domain_reputation":"high","dkim_success_rate":0.99}`, domID)
	reqPMCreate := httptest.NewRequest(http.MethodPost, "/v1/reports/postmaster", strings.NewReader(pmPayload))
	reqPMCreate.Header.Set("X-User-ID", userID)
	reqPMCreate.Header.Set("X-Tenant-ID", tenantID)
	reqPMCreate.Header.Set("Content-Type", "application/json")
	wPMCreate := httptest.NewRecorder()
	server.Handler().ServeHTTP(wPMCreate, reqPMCreate)

	if wPMCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on postmaster snapshot, got %d: %s", wPMCreate.Code, wPMCreate.Body.String())
	}

	reqPMList := httptest.NewRequest(http.MethodGet, "/v1/reports/postmaster?domain_id="+domID, nil)
	reqPMList.Header.Set("X-User-ID", userID)
	reqPMList.Header.Set("X-Tenant-ID", tenantID)
	wPMList := httptest.NewRecorder()
	server.Handler().ServeHTTP(wPMList, reqPMList)

	if wPMList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /v1/reports/postmaster, got %d: %s", wPMList.Code, wPMList.Body.String())
	}
}
