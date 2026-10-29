package worker

import (
	"context"
	"io"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/core"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/storage"
)

type mockSMTPServer struct {
	listener net.Listener
	mu       sync.Mutex
	received [][]byte
	response string // custom response to DATA e.g. "250 OK" or "550 5.1.1 User unknown"
	rcptResp string
	authPass string
	fromSeen string
}

func startMockSMTPServer(t *testing.T, response string) *mockSMTPServer {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock SMTP server: %v", err)
	}

	s := &mockSMTPServer{
		listener: l,
		response: response,
	}

	go s.serve()
	return s
}

func (s *mockSMTPServer) port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *mockSMTPServer) close() {
	_ = s.listener.Close()
}

func (s *mockSMTPServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *mockSMTPServer) handleConn(conn net.Conn) {
	defer conn.Close()
	tc := textproto.NewConn(conn)

	_ = tc.Writer.PrintfLine("220 mock.smtp.server Mailvex Test")

	for {
		line, err := tc.ReadLine()
		if err != nil {
			return
		}
		upper := strings.ToUpper(line)

		if strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO") {
			_ = tc.Writer.PrintfLine("250-mock.smtp.server")
			_ = tc.Writer.PrintfLine("250 AUTH PLAIN LOGIN")
		} else if strings.HasPrefix(upper, "AUTH PLAIN") {
			s.mu.Lock()
			s.authPass = "authenticated"
			s.mu.Unlock()
			_ = tc.Writer.PrintfLine("235 2.7.0 Authentication successful")
		} else if strings.HasPrefix(upper, "MAIL FROM:") {
			s.mu.Lock()
			s.fromSeen = line
			s.mu.Unlock()
			_ = tc.Writer.PrintfLine("250 2.1.0 Sender OK")
		} else if strings.HasPrefix(upper, "RCPT TO:") {
			s.mu.Lock()
			rcptResp := s.rcptResp
			s.mu.Unlock()
			if rcptResp != "" {
				_ = tc.Writer.PrintfLine("%s", rcptResp)
			} else {
				_ = tc.Writer.PrintfLine("250 2.1.5 Recipient OK")
			}
		} else if upper == "DATA" {
			_ = tc.Writer.PrintfLine("354 Start mail input")
			data, _ := io.ReadAll(tc.DotReader())

			s.mu.Lock()
			s.received = append(s.received, data)
			resp := s.response
			s.mu.Unlock()

			if resp == "" {
				resp = "250 2.0.0 OK: message queued"
			}
			_ = tc.Writer.PrintfLine("%s", resp)
		} else if upper == "QUIT" {
			_ = tc.Writer.PrintfLine("221 2.0.0 Bye")
			return
		} else {
			_ = tc.Writer.PrintfLine("502 Command not implemented")
		}
	}
}

type staticResolver struct {
	host string
}

func (r *staticResolver) Resolve(ctx context.Context, domain string) ([]*MXRecord, error) {
	return []*MXRecord{
		{Host: r.host, Pref: 10, Provider: "custom"},
	}, nil
}

func setupTestEnvironment(t *testing.T) (repository.Repository, storage.Storage, *DeliveryWorker, *mockSMTPServer) {
	repo := repository.NewMemoryRepository()
	store := storage.NewMemoryStorage()
	ctx := context.Background()

	// Organization & Tenant
	org := &domain.Organization{ID: "org_1", Name: "Test Org", Slug: "test-org"}
	_ = repo.CreateOrganization(ctx, org)
	tenant := &domain.Tenant{ID: "t_1", OrganizationID: "org_1", Name: "Tenant 1", Slug: "tenant-1", Environment: "live"}
	_ = repo.CreateTenant(ctx, tenant)

	// Sending domain
	dom := &domain.Domain{ID: "dom_1", TenantID: "t_1", Name: "example.com", Status: "verified"}
	_ = repo.CreateDomain(ctx, dom)

	// Active DKIM key
	dkim, _ := crypto.GenerateDKIMKey(2048)
	_ = repo.SaveDKIMKey(ctx, &domain.DKIMKey{
		ID:            "dkim_1",
		DomainID:      "dom_1",
		Selector:            "default",
		EncryptedPrivateKey: dkim.PrivateKeyPEM,
		PublicKey:           dkim.PublicKeyB64,
		Status:              "active",
		CreatedAt:     time.Now().UTC(),
	})

	// Mock SMTP Server
	mockServer := startMockSMTPServer(t, "250 2.0.0 OK: delivered")

	router := NewRouter(repo, "node-1")
	resolver := &staticResolver{host: "127.0.0.1"}
	rules, _ := repo.ListBounceRules(ctx)
	classifier := NewClassifier(rules)

	workerCfg := WorkerConfig{
		NodeID:        "node-1",
		PlatformHost:  "mailvex.example.com",
		BounceDomain:  "bounce.example.com",
		DefaultPort:   mockServer.port(),
		ShutdownGrace: 5 * time.Second,
	}

	worker := NewDeliveryWorker(repo, store, router, resolver, classifier, workerCfg)

	return repo, store, worker, mockServer
}

func TestDirectMXDeliverySuccess(t *testing.T) {
	repo, store, worker, mockServer := setupTestEnvironment(t)
	defer mockServer.close()
	ctx := context.Background()

	ingestEngine := core.NewIngestionEngine(repo, store)
	res, err := ingestEngine.Ingest(ctx, &core.IngestRequest{
		TenantID:    "t_1",
		Source:      "api",
		FromAddress: "noreply@example.com",
		To:          []string{"user@destination.com"},
		Subject:     "Test Delivery",
		BodyText:    "Hello Mailvex Worker Delivery!",
		StreamName:  "transactional",
	})
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	rcptID := res.RecipientIDs[0]

	// Process recipient delivery
	err = worker.ProcessRecipient(ctx, rcptID)
	if err != nil {
		t.Fatalf("ProcessRecipient failed: %v", err)
	}

	// Verify recipient state is delivered
	rcpt, _ := repo.GetRecipient(ctx, rcptID)
	if rcpt.Status != "delivered" {
		t.Fatalf("expected status 'delivered', got: %s", rcpt.Status)
	}
	if rcpt.DeliveredAt == nil {
		t.Fatalf("expected DeliveredAt timestamp set")
	}

	// Verify parent message summary updated
	msg, _ := repo.GetMessageByID(ctx, "t_1", res.MessageID)
	if msg.Status != "delivered" {
		t.Fatalf("expected message status 'delivered', got: %s", msg.Status)
	}

	// Verify mock server received payload with DKIM signature and header hygiene
	mockServer.mu.Lock()
	defer mockServer.mu.Unlock()
	if len(mockServer.received) == 0 {
		t.Fatalf("mock SMTP server did not receive message data")
	}
	receivedData := string(mockServer.received[0])
	if !strings.Contains(receivedData, "DKIM-Signature:") {
		t.Fatalf("expected DKIM-Signature in delivered payload, got: %s", receivedData[:200])
	}
	if !strings.Contains(receivedData, "Feedback-Id:") && !strings.Contains(receivedData, "Feedback-ID:") {
		t.Fatalf("expected Feedback-ID in delivered payload, got: %s", receivedData[:300])
	}
	if !strings.Contains(receivedData, "Return-Path: <bounce+") {
		t.Fatalf("expected VERP Return-Path in delivered payload")
	}
}

func TestSmartHostRelayDeliveryWithFromRewrite(t *testing.T) {
	repo, store, worker, mockServer := setupTestEnvironment(t)
	defer mockServer.close()
	ctx := context.Background()

	// Configure smart-host relay with FromRewritePolicy = "replace" (§8.4)
	authUser := "relayuser@smart-relay.net"
	rewriteFrom := "verified-sender@smart-relay.net"
	relay := &domain.Relay{
		ID:                 "relay_1",
		TenantID:           nil, // platform-wide relay
		Name:               "Homelab External Relay",
		Host:               "127.0.0.1",
		Port:               mockServer.port(),
		TLSMode:            "none",
		AuthUser:           &authUser,
		AuthPassEncrypted:  &authUser,
		FromRewritePolicy:  "replace",
		FromRewriteAddress: &rewriteFrom,
		IsActive:           true,
		CreatedAt:          time.Now().UTC(),
	}
	_ = repo.CreateRelay(ctx, relay)

	ingestEngine := core.NewIngestionEngine(repo, store)
	res, err := ingestEngine.Ingest(ctx, &core.IngestRequest{
		TenantID:    "t_1",
		Source:      "api",
		FromAddress: "original-sender@example.com",
		To:          []string{"client@external.com"},
		Subject:     "Testing Relay Transport",
		BodyText:    "Hello via Smart Host!",
	})
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	rcptID := res.RecipientIDs[0]
	err = worker.ProcessRecipient(ctx, rcptID)
	if err != nil {
		t.Fatalf("ProcessRecipient via relay failed: %v", err)
	}

	rcpt, _ := repo.GetRecipient(ctx, rcptID)
	if rcpt.Status != "delivered" {
		t.Fatalf("expected status 'delivered', got: %s", rcpt.Status)
	}

	mockServer.mu.Lock()
	defer mockServer.mu.Unlock()
	if mockServer.authPass != "authenticated" {
		t.Fatalf("expected relay authentication to be performed")
	}
	if len(mockServer.received) == 0 {
		t.Fatalf("expected relay to receive data")
	}

	receivedStr := string(mockServer.received[0])
	if !strings.Contains(receivedStr, "From: verified-sender@smart-relay.net") {
		t.Fatalf("expected From header rewritten to verified address, got: %s", receivedStr[:300])
	}
	if !strings.Contains(receivedStr, "X-Original-From: original-sender@example.com") {
		t.Fatalf("expected original sender preserved in X-Original-From header")
	}
}

func TestSuppressionCheckPreventsSending(t *testing.T) {
	repo, store, worker, mockServer := setupTestEnvironment(t)
	defer mockServer.close()
	ctx := context.Background()

	// Pre-suppress address
	suppReason := "Known bad spam trap"
	_ = repo.AddSuppression(ctx, &domain.Suppression{
		ID:            "supp_pre",
		TenantID:      "t_1",
		Address:       "suppressed@destination.com",
		AddressDomain: "destination.com",
		Type:          "hard_bounce",
		Reason:        &suppReason,
		Source:        "admin",
		CreatedAt:     time.Now().UTC(),
	})

	ingestEngine := core.NewIngestionEngine(repo, store)
	res, err := ingestEngine.Ingest(ctx, &core.IngestRequest{
		TenantID:    "t_1",
		Source:      "api",
		FromAddress: "noreply@example.com",
		To:          []string{"suppressed@destination.com"},
		Subject:     "Will Not Be Sent",
		BodyText:    "Suppression must prevent delivery",
	})
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	rcptID := res.RecipientIDs[0]

	// Recipient should already be suppressed during ingest (§13.1, §18)
	rcpt, _ := repo.GetRecipient(ctx, rcptID)
	if rcpt.Status != "suppressed" {
		t.Fatalf("expected status 'suppressed' at ingest, got: %s", rcpt.Status)
	}

	// Worker run should immediately skip and not hit network (§15 step 3)
	err = worker.ProcessRecipient(ctx, rcptID)
	if err != nil {
		t.Fatalf("ProcessRecipient error: %v", err)
	}

	mockServer.mu.Lock()
	defer mockServer.mu.Unlock()
	if len(mockServer.received) > 0 {
		t.Fatalf("mock SMTP server received message when recipient was suppressed!")
	}
}

func TestHardBounceAutoSuppressionAndDLQ(t *testing.T) {
	repo, store, worker, mockServer := setupTestEnvironment(t)
	defer mockServer.close()
	ctx := context.Background()

	// Configure mock server to return 550 5.1.1 User unknown
	mockServer.mu.Lock()
	mockServer.response = "550 5.1.1 <nobody@destination.com>: Recipient address rejected: User unknown"
	mockServer.mu.Unlock()

	ingestEngine := core.NewIngestionEngine(repo, store)
	res, err := ingestEngine.Ingest(ctx, &core.IngestRequest{
		TenantID:    "t_1",
		Source:      "api",
		FromAddress: "noreply@example.com",
		To:          []string{"nobody@destination.com"},
		Subject:     "Testing Hard Bounce",
		BodyText:    "This recipient does not exist",
	})
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	rcptID := res.RecipientIDs[0]

	err = worker.ProcessRecipient(ctx, rcptID)
	if err != nil {
		t.Fatalf("ProcessRecipient failed: %v", err)
	}

	// Recipient must be bounced
	rcpt, _ := repo.GetRecipient(ctx, rcptID)
	if rcpt.Status != "bounced" {
		t.Fatalf("expected status 'bounced', got: %s", rcpt.Status)
	}

	// Auto-suppression must be recorded immediately (§15 step 13, §16)
	isSupp, reason, err := repo.CheckSuppression(ctx, "t_1", "", "nobody@destination.com")
	if err != nil || !isSupp {
		t.Fatalf("expected recipient to be auto-suppressed after 5xx hard bounce! isSupp=%v, err=%v", isSupp, err)
	}
	if !strings.Contains(reason, "User unknown") {
		t.Fatalf("expected suppression reason to contain SMTP response, got: %s", reason)
	}

	// Dead-letter queue entry must be recorded (§15.2)
	dlqs, _ := repo.ListDeadLetterMessages(ctx, "t_1", 10)
	if len(dlqs) == 0 {
		t.Fatalf("expected entry in dead_letter_messages")
	}
	if dlqs[0].Reason != "hard_bounce" {
		t.Fatalf("expected DLQ reason 'hard_bounce', got: %s", dlqs[0].Reason)
	}
}

func TestSoftBounceRetryScheduling(t *testing.T) {
	repo, store, worker, mockServer := setupTestEnvironment(t)
	defer mockServer.close()
	ctx := context.Background()

	// Configure mock server to return 452 4.2.2 Mailbox full
	mockServer.mu.Lock()
	mockServer.response = "452 4.2.2 Mailbox full"
	mockServer.mu.Unlock()

	ingestEngine := core.NewIngestionEngine(repo, store)
	res, err := ingestEngine.Ingest(ctx, &core.IngestRequest{
		TenantID:    "t_1",
		Source:      "api",
		FromAddress: "noreply@example.com",
		To:          []string{"full@destination.com"},
		Subject:     "Testing Soft Bounce",
		BodyText:    "Mailbox is full",
	})
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	rcptID := res.RecipientIDs[0]

	err = worker.ProcessRecipient(ctx, rcptID)
	if err != nil {
		t.Fatalf("ProcessRecipient failed: %v", err)
	}

	// Recipient must be deferred
	rcpt, _ := repo.GetRecipient(ctx, rcptID)
	if rcpt.Status != "deferred" {
		t.Fatalf("expected status 'deferred', got: %s", rcpt.Status)
	}
	if rcpt.NextAttemptAt == nil {
		t.Fatalf("expected next_attempt_at timestamp to be populated for retry")
	}
	if !rcpt.NextAttemptAt.After(time.Now()) {
		t.Fatalf("expected next_attempt_at to be in the future")
	}

	// Ensure NOT auto-suppressed for soft bounce (§18: never auto-suppress on temporary failures)
	isSupp, _, _ := repo.CheckSuppression(ctx, "t_1", "", "full@destination.com")
	if isSupp {
		t.Fatalf("temporary soft bounce must never trigger immediate suppression!")
	}
}

func TestWorkerGracefulDrain(t *testing.T) {
	_, _, worker, mockServer := setupTestEnvironment(t)
	defer mockServer.close()

	// Initiate drain
	drained := worker.Drain(1 * time.Second)
	if !drained {
		t.Fatalf("expected worker to drain successfully")
	}

	if !worker.IsDraining() {
		t.Fatalf("expected worker.IsDraining() == true")
	}

	// Submitting work to draining worker must be rejected
	err := worker.ProcessRecipient(context.Background(), "rcpt_dummy")
	if err == nil || !strings.Contains(err.Error(), "draining") {
		t.Fatalf("expected worker draining error, got: %v", err)
	}
}
