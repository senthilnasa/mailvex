package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
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

func setupPrivacyTestEnvironment(t *testing.T) (*Server, repository.Repository, string, string) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	kek := bytes.Repeat([]byte{0x05}, 32)
	enc, _ := crypto.NewEnvelopeEncryption(kek)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	server := NewServer(repo, auditSvc, enc, logger)

	ctx := context.Background()

	orgID := crypto.NewULID(crypto.PrefixOrg)
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Privacy Org",
		Slug:      "privacy-org",
		CreatedAt: time.Now().UTC(),
	})

	tenantID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Privacy Tenant",
		Slug:           "privacy-tenant",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	userID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(ctx, &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		Email:          "admin@privacy.local",
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

	return server, repo, tenantID, userID
}

func TestSuppressionImportAndExportEndpoints(t *testing.T) {
	server, _, tenantID, userID := setupPrivacyTestEnvironment(t)

	// 1. Bulk CSV Import via Multipart Form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "suppressions.csv")
	if err != nil {
		t.Fatalf("failed to create multipart file: %v", err)
	}

	csvContent := "email,type,reason\nuser1@example.com,hard_bounce,Mailbox not found\nuser2@example.com,complaint,Marked spam\nuser1@example.com,hard_bounce,Duplicate entry\n"
	_, _ = part.Write([]byte(csvContent))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/suppressions/import", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("import failed with status %d: %s", w.Code, w.Body.String())
	}

	var importResult domain.SuppressionImport
	if err := json.NewDecoder(w.Body).Decode(&importResult); err != nil {
		t.Fatalf("failed to decode import result: %v", err)
	}

	if importResult.TotalRows != 3 {
		t.Errorf("expected 3 total rows, got %d", importResult.TotalRows)
	}
	if importResult.ImportedRows != 2 {
		t.Errorf("expected 2 imported rows, got %d", importResult.ImportedRows)
	}
	if importResult.DuplicateRows != 1 {
		t.Errorf("expected 1 duplicate row, got %d", importResult.DuplicateRows)
	}

	// 2. Export Suppressions as CSV
	exportReq := httptest.NewRequest(http.MethodGet, "/v1/suppressions/export?format=csv", nil)
	exportReq.Header.Set("X-Tenant-ID", tenantID)
	exportReq.Header.Set("X-User-ID", userID)

	wExp := httptest.NewRecorder()
	server.Handler().ServeHTTP(wExp, exportReq)

	if wExp.Code != http.StatusOK {
		t.Fatalf("export failed with status %d: %s", wExp.Code, wExp.Body.String())
	}

	if wExp.Header().Get("Content-Type") != "text/csv" {
		t.Errorf("expected Content-Type text/csv, got %s", wExp.Header().Get("Content-Type"))
	}

	csvReader := csv.NewReader(wExp.Body)
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse exported CSV: %v", err)
	}

	// 1 header row + 2 data rows
	if len(records) != 3 {
		t.Errorf("expected 3 CSV rows, got %d", len(records))
	}
	if records[0][0] != "id" || records[0][1] != "address" {
		t.Errorf("unexpected CSV header: %v", records[0])
	}
}

func TestPrivacyErasureAndExportEndpoints(t *testing.T) {
	server, repo, tenantID, userID := setupPrivacyTestEnvironment(t)
	ctx := context.Background()

	email := "forget.me@example.com"
	msgID := "msg_test_gdpr"
	bodyRef := "blob://body_gdpr"
	rawMIME := "blob://mime_gdpr"

	msg := &domain.Message{
		ID:         msgID,
		TenantID:   tenantID,
		StreamID:   "stream_tx",
		BodyRef:    &bodyRef,
		RawMIMERef: &rawMIME,
		CreatedAt:  time.Now().UTC(),
	}
	rcptID := "rcpt_test_gdpr"
	rcpt := &domain.MessageRecipient{
		ID:            rcptID,
		MessageID:     msgID,
		TenantID:      tenantID,
		Address:       email,
		AddressDomain: "example.com",
		Status:        "delivered",
		CreatedAt:     time.Now().UTC(),
	}
	_ = repo.SaveMessage(ctx, msg, []*domain.MessageRecipient{rcpt}, nil, nil)
	_ = repo.AddSuppression(ctx, &domain.Suppression{
		ID:            "sup_test_gdpr",
		TenantID:      tenantID,
		Address:       email,
		AddressDomain: "example.com",
		Type:          "unsubscribe",
		CreatedAt:     time.Now().UTC(),
	})

	// 1. Test POST /v1/privacy/export
	expPayload, _ := json.Marshal(map[string]string{"email": email})
	reqExp := httptest.NewRequest(http.MethodPost, "/v1/privacy/export", bytes.NewReader(expPayload))
	reqExp.Header.Set("Content-Type", "application/json")
	reqExp.Header.Set("X-Tenant-ID", tenantID)
	reqExp.Header.Set("X-User-ID", userID)

	wExp := httptest.NewRecorder()
	server.Handler().ServeHTTP(wExp, reqExp)

	if wExp.Code != http.StatusOK {
		t.Fatalf("privacy export failed: status %d, body %s", wExp.Code, wExp.Body.String())
	}

	var exportReport domain.PrivacyExportReport
	_ = json.NewDecoder(wExp.Body).Decode(&exportReport)
	if exportReport.EmailAddress != email {
		t.Errorf("expected email %s, got %s", email, exportReport.EmailAddress)
	}
	if len(exportReport.Messages) != 1 || len(exportReport.Suppressions) != 1 {
		t.Errorf("expected 1 message and 1 suppression in export report, got %d and %d", len(exportReport.Messages), len(exportReport.Suppressions))
	}

	// 2. Test POST /v1/privacy/erasure (GDPR right-to-be-forgotten)
	erasePayload, _ := json.Marshal(map[string]string{"email": email})
	reqErase := httptest.NewRequest(http.MethodPost, "/v1/privacy/erasure", bytes.NewReader(erasePayload))
	reqErase.Header.Set("Content-Type", "application/json")
	reqErase.Header.Set("X-Tenant-ID", tenantID)
	reqErase.Header.Set("X-User-ID", userID)

	wErase := httptest.NewRecorder()
	server.Handler().ServeHTTP(wErase, reqErase)

	if wErase.Code != http.StatusOK {
		t.Fatalf("privacy erasure failed: status %d, body %s", wErase.Code, wErase.Body.String())
	}

	var eraseReq domain.PrivacyRequest
	_ = json.NewDecoder(wErase.Body).Decode(&eraseReq)
	if eraseReq.Status != "completed" {
		t.Errorf("expected status completed, got %s", eraseReq.Status)
	}

	// Verify cleartext email is gone from repo
	storedMsg, recs, _, _ := repo.GetMessage(ctx, tenantID, msgID)
	if storedMsg.BodyRef != nil || storedMsg.RawMIMERef != nil {
		t.Errorf("message bodies and raw MIME were not wiped")
	}
	if len(recs) != 1 || strings.Contains(recs[0].Address, email) {
		t.Errorf("recipient address was not anonymized: %v", recs)
	}

	// 3. Test GET /v1/privacy/requests
	reqList := httptest.NewRequest(http.MethodGet, "/v1/privacy/requests", nil)
	reqList.Header.Set("X-Tenant-ID", tenantID)
	reqList.Header.Set("X-User-ID", userID)

	wList := httptest.NewRecorder()
	server.Handler().ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("privacy requests list failed: status %d, body %s", wList.Code, wList.Body.String())
	}

	var reqsResp struct {
		Data []*domain.PrivacyRequest `json:"data"`
	}
	_ = json.NewDecoder(wList.Body).Decode(&reqsResp)
	if len(reqsResp.Data) < 2 {
		t.Errorf("expected at least 2 logged privacy requests, got %d", len(reqsResp.Data))
	}
}

func TestRetentionPolicyAndPurgeEndpoints(t *testing.T) {
	server, _, tenantID, userID := setupPrivacyTestEnvironment(t)

	// 1. GET /v1/retention
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/retention", nil)
	reqGet.Header.Set("X-Tenant-ID", tenantID)
	reqGet.Header.Set("X-User-ID", userID)

	wGet := httptest.NewRecorder()
	server.Handler().ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("get retention failed: status %d, body %s", wGet.Code, wGet.Body.String())
	}

	var pol domain.RetentionPolicy
	_ = json.NewDecoder(wGet.Body).Decode(&pol)
	if pol.MessageRetentionDays != 30 || pol.EventsRetentionDays != 90 {
		t.Errorf("unexpected default retention policy: %+v", pol)
	}

	// 2. PUT /v1/retention
	pol.MessageRetentionDays = 14
	pol.EventsRetentionDays = 45
	pol.AttemptsRetentionDays = 14
	pol.RawMIMERetentionDays = 3

	putBytes, _ := json.Marshal(pol)
	reqPut := httptest.NewRequest(http.MethodPut, "/v1/retention", bytes.NewReader(putBytes))
	reqPut.Header.Set("Content-Type", "application/json")
	reqPut.Header.Set("X-Tenant-ID", tenantID)
	reqPut.Header.Set("X-User-ID", userID)

	wPut := httptest.NewRecorder()
	server.Handler().ServeHTTP(wPut, reqPut)

	if wPut.Code != http.StatusOK {
		t.Fatalf("update retention failed: status %d, body %s", wPut.Code, wPut.Body.String())
	}

	var updatedPol domain.RetentionPolicy
	_ = json.NewDecoder(wPut.Body).Decode(&updatedPol)
	if updatedPol.MessageRetentionDays != 14 || updatedPol.RawMIMERetentionDays != 3 {
		t.Errorf("retention policy was not updated correctly: %+v", updatedPol)
	}

	// 3. POST /v1/retention/purge
	reqPurge := httptest.NewRequest(http.MethodPost, "/v1/retention/purge", nil)
	reqPurge.Header.Set("X-Tenant-ID", tenantID)
	reqPurge.Header.Set("X-User-ID", userID)

	wPurge := httptest.NewRecorder()
	server.Handler().ServeHTTP(wPurge, reqPurge)

	if wPurge.Code != http.StatusOK {
		t.Fatalf("execute retention purge failed: status %d, body %s", wPurge.Code, wPurge.Body.String())
	}

	var purgeSummary domain.RetentionPurgeSummary
	_ = json.NewDecoder(wPurge.Body).Decode(&purgeSummary)
	if purgeSummary.TenantID != tenantID {
		t.Errorf("expected tenantID %s, got %s", tenantID, purgeSummary.TenantID)
	}
}
