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

func setupPhase4TestEnvironment(t *testing.T) (*Server, repository.Repository, string, string, string) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	kek := bytes.Repeat([]byte{0x04}, 32)
	enc, _ := crypto.NewEnvelopeEncryption(kek)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	server := NewServer(repo, auditSvc, enc, logger)

	ctx := context.Background()

	// 1. Create org and tenant
	orgID := crypto.NewULID(crypto.PrefixOrg)
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Test Org",
		Slug:      "test-org",
		CreatedAt: time.Now().UTC(),
	})

	tenantID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Test Tenant",
		Slug:           "test-tenant",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	// 2. Create User and Membership with full tenant role
	userID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(ctx, &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		Email:          "admin@example.com",
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

	// 3. Create verified domain
	domID := crypto.NewULID(crypto.PrefixDomain)
	_ = repo.CreateDomain(ctx, &domain.Domain{
		ID:        domID,
		TenantID:  tenantID,
		Name:      "example.com",
		Status:    "verified",
		CreatedAt: time.Now().UTC(),
	})

	return server, repo, orgID, tenantID, userID
}

func TestAPIKeyAuthenticationAndManagement(t *testing.T) {
	server, _, _, tenantID, userID := setupPhase4TestEnvironment(t)

	// 1. Create API key using user session
	createPayload := `{"name":"Backend Service","environment":"live","scopes":["send.email","messages.read"]}`
	req := httptest.NewRequest("POST", "/v1/api-keys", strings.NewReader(createPayload))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var keyResp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &keyResp)
	token, _ := keyResp["token"].(string)
	keyID, _ := keyResp["id"].(string)

	if !strings.HasPrefix(token, "mvx_live_") {
		t.Fatalf("expected token starting with mvx_live_, got %s", token)
	}

	// 2. Use API key Bearer token to list messages (messages.read scope)
	reqList := httptest.NewRequest("GET", "/v1/messages", nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	server.Handler().ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with API key bearer, got %d: %s", wList.Code, wList.Body.String())
	}

	// 3. Revoke key
	reqRevoke := httptest.NewRequest("DELETE", "/v1/api-keys/"+keyID, nil)
	reqRevoke.Header.Set("X-User-ID", userID)
	reqRevoke.Header.Set("X-Tenant-ID", tenantID)
	wRevoke := httptest.NewRecorder()
	server.Handler().ServeHTTP(wRevoke, reqRevoke)

	if wRevoke.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on revoke, got %d", wRevoke.Code)
	}

	// 4. Verify revoked key is now rejected
	reqListAfter := httptest.NewRequest("GET", "/v1/messages", nil)
	reqListAfter.Header.Set("Authorization", "Bearer "+token)
	wListAfter := httptest.NewRecorder()
	server.Handler().ServeHTTP(wListAfter, reqListAfter)

	if wListAfter.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized with revoked key, got %d", wListAfter.Code)
	}
}

func TestSendEmailWithAPIKeyAndDevelopmentSink(t *testing.T) {
	server, _, _, tenantID, userID := setupPhase4TestEnvironment(t)

	// Create test environment API key (mvx_test_...)
	createPayload := `{"name":"Dev Key","environment":"test","scopes":["send.email","messages.read"]}`
	req := httptest.NewRequest("POST", "/v1/api-keys", strings.NewReader(createPayload))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	var keyResp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &keyResp)
	testToken, _ := keyResp["token"].(string)

	// Send email with test key
	sendBody := `{
		"from": "Billing <noreply@example.com>",
		"to": ["alice@external.com"],
		"subject": "Invoice #101",
		"html": "<h1>Invoice Ready</h1>",
		"text": "Invoice Ready"
	}`

	reqSend := httptest.NewRequest("POST", "/v1/email", strings.NewReader(sendBody))
	reqSend.Header.Set("Authorization", "Bearer "+testToken)
	reqSend.Header.Set("Content-Type", "application/json")
	wSend := httptest.NewRecorder()
	server.Handler().ServeHTTP(wSend, reqSend)

	if wSend.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", wSend.Code, wSend.Body.String())
	}

	var sendResp map[string]interface{}
	_ = json.Unmarshal(wSend.Body.Bytes(), &sendResp)
	msgID, _ := sendResp["id"].(string)
	env, _ := sendResp["environment"].(string)

	if env != "development" {
		t.Fatalf("expected environment 'development', got '%s'", env)
	}

	// Verify message landed in development mailbox sink
	reqDev := httptest.NewRequest("GET", "/v1/messages/development", nil)
	reqDev.Header.Set("Authorization", "Bearer "+testToken)
	wDev := httptest.NewRecorder()
	server.Handler().ServeHTTP(wDev, reqDev)

	if wDev.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from dev mailbox, got %d", wDev.Code)
	}

	var devResp map[string]interface{}
	_ = json.Unmarshal(wDev.Body.Bytes(), &devResp)
	dataList, _ := devResp["data"].([]interface{})
	if len(dataList) == 0 {
		t.Fatalf("expected at least 1 message in dev mailbox sink, got 0")
	}

	// Verify message ID matches
	found := false
	for _, itm := range dataList {
		if m, ok := itm.(map[string]interface{}); ok && m["id"] == msgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected msg %s in dev mailbox sink", msgID)
	}
}

func TestIdempotencySemantics(t *testing.T) {
	server, _, _, tenantID, userID := setupPhase4TestEnvironment(t)

	sendBody := `{
		"from": "Orders <noreply@example.com>",
		"to": ["bob@example.com"],
		"subject": "Order Confirmation",
		"text": "Your order has shipped."
	}`

	idemKey := "order-checkout-uuid-999"

	// 1. Initial Request
	req1 := httptest.NewRequest("POST", "/v1/email", strings.NewReader(sendBody))
	req1.Header.Set("X-User-ID", userID)
	req1.Header.Set("X-Tenant-ID", tenantID)
	req1.Header.Set("Idempotency-Key", idemKey)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	server.Handler().ServeHTTP(w1, req1)

	if w1.Code != http.StatusAccepted {
		t.Fatalf("expected 202 on first attempt, got %d: %s", w1.Code, w1.Body.String())
	}

	var resp1 map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	msgID1 := resp1["id"].(string)

	// 2. Exact same request with same Idempotency-Key -> original response
	req2 := httptest.NewRequest("POST", "/v1/email", strings.NewReader(sendBody))
	req2.Header.Set("X-User-ID", userID)
	req2.Header.Set("X-Tenant-ID", tenantID)
	req2.Header.Set("Idempotency-Key", idemKey)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	server.Handler().ServeHTTP(w2, req2)

	if w2.Code != http.StatusAccepted {
		t.Fatalf("expected 202 on idempotent repeat, got %d", w2.Code)
	}

	var resp2 map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	msgID2 := resp2["id"].(string)

	if msgID1 != msgID2 {
		t.Fatalf("expected same message ID %s on idempotent replay, got %s", msgID1, msgID2)
	}

	// 3. Different request body with SAME Idempotency-Key -> 409 Conflict
	diffBody := `{
		"from": "Orders <noreply@example.com>",
		"to": ["bob@example.com"],
		"subject": "DIFFERENT SUBJECT",
		"text": "Modified body."
	}`
	req3 := httptest.NewRequest("POST", "/v1/email", strings.NewReader(diffBody))
	req3.Header.Set("X-User-ID", userID)
	req3.Header.Set("X-Tenant-ID", tenantID)
	req3.Header.Set("Idempotency-Key", idemKey)
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	server.Handler().ServeHTTP(w3, req3)

	if w3.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on payload mismatch, got %d: %s", w3.Code, w3.Body.String())
	}
	if !strings.Contains(w3.Body.String(), "idempotency_key_reused") {
		t.Fatalf("expected idempotency_key_reused error, got: %s", w3.Body.String())
	}
}

func TestHeaderValidationAndCRLFFiltering(t *testing.T) {
	server, _, _, tenantID, userID := setupPhase4TestEnvironment(t)

	cases := []struct {
		name         string
		payload      string
		expectedCode int
		expectedErr  string
	}{
		{
			name: "CRLF injection in From header",
			payload: `{
				"from": "attacker@example.com\r\nBcc: evil@hacker.com",
				"to": ["user@example.com"],
				"subject": "Test",
				"text": "Hello"
			}`,
			expectedCode: http.StatusBadRequest,
			expectedErr:  "header_injection",
		},
		{
			name: "CRLF injection in Subject header",
			payload: `{
				"from": "noreply@example.com",
				"to": ["user@example.com"],
				"subject": "Invoice\r\nX-Injected: Bad",
				"text": "Hello"
			}`,
			expectedCode: http.StatusBadRequest,
			expectedErr:  "header_injection",
		},
		{
			name: "Prohibited custom header",
			payload: `{
				"from": "noreply@example.com",
				"to": ["user@example.com"],
				"subject": "Test",
				"text": "Hello",
				"headers": {"Return-Path": "<attacker@evil.com>"}
			}`,
			expectedCode: http.StatusBadRequest,
			expectedErr:  "header_not_allowed",
		},
		{
			name: "Custom header not matching allowlist",
			payload: `{
				"from": "noreply@example.com",
				"to": ["user@example.com"],
				"subject": "Test",
				"text": "Hello",
				"headers": {"Arbitrary": "value"}
			}`,
			expectedCode: http.StatusBadRequest,
			expectedErr:  "header_not_allowed",
		},
		{
			name: "Valid custom headers X-* and List-Id",
			payload: `{
				"from": "noreply@example.com",
				"to": ["user@example.com"],
				"subject": "Test",
				"text": "Hello",
				"headers": {
					"X-Campaign": "Summer-Sale",
					"List-Id": "<newsletter.example.com>"
				}
			}`,
			expectedCode: http.StatusAccepted,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/email", strings.NewReader(tc.payload))
			req.Header.Set("X-User-ID", userID)
			req.Header.Set("X-Tenant-ID", tenantID)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Fatalf("expected status %d, got %d: %s", tc.expectedCode, w.Code, w.Body.String())
			}
			if tc.expectedErr != "" && !strings.Contains(w.Body.String(), tc.expectedErr) {
				t.Fatalf("expected error '%s' in body, got: %s", tc.expectedErr, w.Body.String())
			}
		})
	}
}

func TestRecipientLimitsAndDomainVerification(t *testing.T) {
	server, _, _, tenantID, userID := setupPhase4TestEnvironment(t)

	// 1. More than 50 recipients -> 400 recipient_limit_exceeded
	var over50 []string
	for i := 0; i < 51; i++ {
		over50 = append(over50, "user"+strings.Repeat("a", i)+"@example.com")
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"from":    "noreply@example.com",
		"to":      over50,
		"subject": "Over limit",
		"text":    "Hello",
	})
	req := httptest.NewRequest("POST", "/v1/email", bytes.NewReader(payload))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "recipient_limit_exceeded") {
		t.Fatalf("expected 400 recipient_limit_exceeded, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Unverified domain -> 400 domain_not_verified
	unverifiedPayload := `{
		"from": "noreply@unverified-domain.com",
		"to": ["user@example.com"],
		"subject": "Spoof",
		"text": "Hello"
	}`
	reqUnverified := httptest.NewRequest("POST", "/v1/email", strings.NewReader(unverifiedPayload))
	reqUnverified.Header.Set("X-User-ID", userID)
	reqUnverified.Header.Set("X-Tenant-ID", tenantID)
	wUnverified := httptest.NewRecorder()
	server.Handler().ServeHTTP(wUnverified, reqUnverified)

	if wUnverified.Code != http.StatusBadRequest || !strings.Contains(wUnverified.Body.String(), "domain_not_verified") {
		t.Fatalf("expected 400 domain_not_verified, got %d: %s", wUnverified.Code, wUnverified.Body.String())
	}
}

func TestTemplateRenderingInEmailSend(t *testing.T) {
	server, _, _, tenantID, userID := setupPhase4TestEnvironment(t)

	// 1. Create published template
	tmplPayload := `{
		"name": "Welcome Email",
		"subject": "Welcome to {{company}}, {{name}}!",
		"html": "<h1>Welcome {{name}}</h1><p>Plan: {{plan}}</p>",
		"text": "Welcome {{name}}! Plan: {{plan}}",
		"publish_now": true
	}`
	reqTmpl := httptest.NewRequest("POST", "/v1/templates", strings.NewReader(tmplPayload))
	reqTmpl.Header.Set("X-User-ID", userID)
	reqTmpl.Header.Set("X-Tenant-ID", tenantID)
	wTmpl := httptest.NewRecorder()
	server.Handler().ServeHTTP(wTmpl, reqTmpl)

	if wTmpl.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for template, got %d: %s", wTmpl.Code, wTmpl.Body.String())
	}

	var tmplResp map[string]interface{}
	_ = json.Unmarshal(wTmpl.Body.Bytes(), &tmplResp)
	tmplObj := tmplResp["template"].(map[string]interface{})
	tmplID := tmplObj["id"].(string)

	// 2. Send email using template_id
	sendPayload := map[string]interface{}{
		"from":        "noreply@example.com",
		"to":          []string{"newuser@example.com"},
		"template_id": tmplID,
		"template_variables": map[string]interface{}{
			"company": "Mailvex Cloud",
			"name":    "Charlie & Bob <VIP>",
			"plan":    "Enterprise",
		},
	}
	sendBytes, _ := json.Marshal(sendPayload)
	reqSend := httptest.NewRequest("POST", "/v1/email", bytes.NewReader(sendBytes))
	reqSend.Header.Set("X-User-ID", userID)
	reqSend.Header.Set("X-Tenant-ID", tenantID)
	wSend := httptest.NewRecorder()
	server.Handler().ServeHTTP(wSend, reqSend)

	if wSend.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted on template send, got %d: %s", wSend.Code, wSend.Body.String())
	}

	var sendResp map[string]interface{}
	_ = json.Unmarshal(wSend.Body.Bytes(), &sendResp)
	msgID := sendResp["id"].(string)

	// 3. Inspect message details and stored body
	reqMsg := httptest.NewRequest("GET", "/v1/messages/"+msgID, nil)
	reqMsg.Header.Set("X-User-ID", userID)
	reqMsg.Header.Set("X-Tenant-ID", tenantID)
	wMsg := httptest.NewRecorder()
	server.Handler().ServeHTTP(wMsg, reqMsg)

	if wMsg.Code != http.StatusOK {
		t.Fatalf("expected 200 OK reading message, got %d", wMsg.Code)
	}

	var msgData map[string]interface{}
	_ = json.Unmarshal(wMsg.Body.Bytes(), &msgData)
	msgObj := msgData["message"].(map[string]interface{})
	subject := msgObj["subject"].(string)

	expectedSubject := "Welcome to Mailvex Cloud, Charlie & Bob <VIP>!"
	if subject != expectedSubject {
		t.Fatalf("expected rendered subject '%s', got '%s'", expectedSubject, subject)
	}

	// 4. Verify body retrieval
	reqBody := httptest.NewRequest("GET", "/v1/messages/"+msgID+"/body", nil)
	reqBody.Header.Set("X-User-ID", userID)
	reqBody.Header.Set("X-Tenant-ID", tenantID)
	wBody := httptest.NewRecorder()
	server.Handler().ServeHTTP(wBody, reqBody)

	if wBody.Code != http.StatusOK {
		t.Fatalf("expected 200 OK reading message body, got %d: %s", wBody.Code, wBody.Body.String())
	}
}

func TestBatchSendingEndpoint(t *testing.T) {
	server, _, _, tenantID, userID := setupPhase4TestEnvironment(t)

	batchPayload := `{
		"from": "noreply@example.com",
		"subject": "Notice to all",
		"recipients": [
			{"to": "user1@example.com", "metadata": {"account_id": "acc_1"}},
			{"to": "user2@example.com", "metadata": {"account_id": "acc_2"}},
			{"to": "user3@example.com", "metadata": {"account_id": "acc_3"}}
		]
	}`

	req := httptest.NewRequest("POST", "/v1/email/batch", strings.NewReader(batchPayload))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for batch send, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 3 {
		t.Fatalf("expected 3 results, got %d", len(data))
	}

	for _, item := range data {
		itm := item.(map[string]interface{})
		if itm["status"] != "queued" {
			t.Fatalf("expected item status queued, got %v", itm)
		}
		if itm["id"] == "" {
			t.Fatalf("expected valid message ID for batch item")
		}
	}
}

func TestMessageReadAndCursorPagination(t *testing.T) {
	server, _, _, tenantID, userID := setupPhase4TestEnvironment(t)

	// Send 5 messages sequentially
	for i := 1; i <= 5; i++ {
		payload := fmt.Sprintf(`{
			"from": "noreply@example.com",
			"to": ["user%d@example.com"],
			"subject": "Msg %d",
			"text": "Body %d"
		}`, i, i, i)
		req := httptest.NewRequest("POST", "/v1/email", strings.NewReader(payload))
		req.Header.Set("X-User-ID", userID)
		req.Header.Set("X-Tenant-ID", tenantID)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusAccepted {
			t.Fatalf("failed to send test msg %d: %s", i, w.Body.String())
		}
		time.Sleep(2 * time.Millisecond) // Ensure unique timestamps
	}

	// 1. Fetch page 1 (limit=2)
	reqPage1 := httptest.NewRequest("GET", "/v1/messages?limit=2", nil)
	reqPage1.Header.Set("X-User-ID", userID)
	reqPage1.Header.Set("X-Tenant-ID", tenantID)
	wPage1 := httptest.NewRecorder()
	server.Handler().ServeHTTP(wPage1, reqPage1)

	var page1Resp map[string]interface{}
	_ = json.Unmarshal(wPage1.Body.Bytes(), &page1Resp)
	items1 := page1Resp["data"].([]interface{})
	nextCursor, _ := page1Resp["next_cursor"].(string)
	hasMore, _ := page1Resp["has_more"].(bool)

	if len(items1) != 2 {
		t.Fatalf("expected 2 items in page 1, got %d", len(items1))
	}
	if !hasMore || nextCursor == "" {
		t.Fatalf("expected has_more=true and valid next_cursor")
	}

	// 2. Fetch page 2 using cursor
	reqPage2 := httptest.NewRequest("GET", "/v1/messages?limit=2&cursor="+nextCursor, nil)
	reqPage2.Header.Set("X-User-ID", userID)
	reqPage2.Header.Set("X-Tenant-ID", tenantID)
	wPage2 := httptest.NewRecorder()
	server.Handler().ServeHTTP(wPage2, reqPage2)

	var page2Resp map[string]interface{}
	_ = json.Unmarshal(wPage2.Body.Bytes(), &page2Resp)
	items2 := page2Resp["data"].([]interface{})

	if len(items2) != 2 {
		t.Fatalf("expected 2 items in page 2, got %d", len(items2))
	}

	// Verify items on page 1 and page 2 are distinct
	id1 := items1[0].(map[string]interface{})["id"].(string)
	id2 := items2[0].(map[string]interface{})["id"].(string)
	if id1 == id2 {
		t.Fatalf("expected pagination cursor to advance, page1 item and page2 item matched: %s", id1)
	}

	// 3. Test filter by recipient
	reqFilter := httptest.NewRequest("GET", "/v1/messages?recipient=user3@example.com", nil)
	reqFilter.Header.Set("X-User-ID", userID)
	reqFilter.Header.Set("X-Tenant-ID", tenantID)
	wFilter := httptest.NewRecorder()
	server.Handler().ServeHTTP(wFilter, reqFilter)

	var filterResp map[string]interface{}
	_ = json.Unmarshal(wFilter.Body.Bytes(), &filterResp)
	filterData := filterResp["data"].([]interface{})
	if len(filterData) != 1 {
		t.Fatalf("expected 1 message matching recipient filter, got %d", len(filterData))
	}
}
