package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

func TestBackupAndRestoreWithHMACVerification(t *testing.T) {
	ctx := context.Background()
	repo1 := repository.NewMemoryRepository()

	// Seed source repository
	org := &domain.Organization{
		ID:        "org_backup_test",
		Name:      "Backup Org",
		Slug:      "backup-org",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo1.CreateOrganization(ctx, org); err != nil {
		t.Fatalf("failed to create org: %v", err)
	}

	tenant := &domain.Tenant{
		ID:             "ten_backup_test",
		OrganizationID: org.ID,
		Name:           "Backup Corp",
		Slug:           "backup-corp",
		Environment:    "live",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := repo1.CreateTenant(ctx, tenant); err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	dom := &domain.Domain{
		ID:             "dom_backup_1",
		TenantID:       tenant.ID,
		Name:           "backupcorp.com",
		Type:           "public",
		Status:         "verified",
		OwnershipToken: "token_123",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := repo1.CreateDomain(ctx, dom); err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "mailvex-backup.tar.gz")
	hmacSecret := []byte("secret-key-32bytes-secure-mailvex!")

	// 1. Create Backup
	manifest, err := CreateBackup(ctx, repo1, archivePath, "", hmacSecret)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	if manifest.RecordCounts["tenants"] != 1 {
		t.Errorf("expected 1 tenant in manifest, got %d", manifest.RecordCounts["tenants"])
	}
	if manifest.RecordCounts["domains"] != 1 {
		t.Errorf("expected 1 domain in manifest, got %d", manifest.RecordCounts["domains"])
	}
	if manifest.HMAC == "" {
		t.Error("manifest HMAC should not be empty")
	}

	// 2. Restore Backup into clean repository
	repo2 := repository.NewMemoryRepository()
	restoredManifest, err := RestoreBackup(ctx, repo2, archivePath, hmacSecret)
	if err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}
	if restoredManifest.Version != manifest.Version {
		t.Errorf("expected restored version %s, got %s", manifest.Version, restoredManifest.Version)
	}

	// Verify restored entity
	restoredTenant, err := repo2.GetTenant(ctx, "ten_backup_test")
	if err != nil || restoredTenant == nil {
		t.Fatalf("restored tenant not found: %v", err)
	}
	if restoredTenant.Name != "Backup Corp" {
		t.Errorf("expected Backup Corp, got %s", restoredTenant.Name)
	}

	restoredDomains, err := repo2.ListDomains(ctx, "ten_backup_test")
	if err != nil || len(restoredDomains) != 1 {
		t.Fatalf("expected 1 restored domain, got %d", len(restoredDomains))
	}
	if restoredDomains[0].Name != "backupcorp.com" {
		t.Errorf("expected backupcorp.com, got %s", restoredDomains[0].Name)
	}

	// 3. Test tampering detection with incorrect secret
	wrongSecret := []byte("wrong-hmac-secret-attempt!")
	_, err = RestoreBackup(ctx, repo2, archivePath, wrongSecret)
	if err == nil {
		t.Error("expected HMAC verification failure with wrong secret, but succeeded")
	}
	if !strings.Contains(err.Error(), "cryptographic HMAC signature verification failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSystemDoctor(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemoryRepository()

	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        "org_doc",
		Name:      "Doc Org",
		Slug:      "doc-org",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             "ten_doc",
		OrganizationID: "org_doc",
		Name:           "Doc Tenant",
		Slug:           "doc-tenant",
		Environment:    "live",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	})

	os.Setenv("MVX_KEK", "12345678901234567890123456789012") // 32 bytes

	doctor := NewSystemDoctor(repo)
	results := doctor.RunAll(ctx)

	if len(results) == 0 {
		t.Fatal("expected doctor diagnostic results, got none")
	}

	table := RenderDoctorTable(results)
	if !strings.Contains(table, "MAILVEX SYSTEM DOCTOR DIAGNOSTIC REPORT") {
		t.Error("expected table header in doctor output")
	}
	if !strings.Contains(table, "Repository Connection") {
		t.Error("expected database check in doctor output")
	}
	if !strings.Contains(table, "Master KEK Configuration") {
		t.Error("expected master KEK check in doctor output")
	}
}
