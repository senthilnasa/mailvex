package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

// BackupManifest describes the contents and cryptographic checksum of a backup archive
type BackupManifest struct {
	Version      string            `json:"version"`
	CreatedAt    time.Time         `json:"created_at"`
	TenantID     string            `json:"tenant_id,omitempty"`
	RecordCounts map[string]int    `json:"record_counts"`
	FileHashes   map[string]string `json:"file_hashes"`
	HMAC         string            `json:"hmac"`
}

// CreateBackup exports repository data into an HMAC-signed tar.gz archive
func CreateBackup(ctx context.Context, repo repository.Repository, outputPath string, tenantID string, hmacSecret []byte) (*BackupManifest, error) {
	if len(hmacSecret) == 0 {
		hmacSecret = []byte("mailvex-default-backup-hmac-secret-v1")
	}

	manifest := &BackupManifest{
		Version:      "1.0.0",
		CreatedAt:    time.Now().UTC(),
		TenantID:     tenantID,
		RecordCounts: make(map[string]int),
		FileHashes:   make(map[string]string),
	}

	// Gather data
	orgs, err := repo.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	var tenants []*domain.Tenant
	for _, o := range orgs {
		ts, _ := repo.ListTenantsByOrg(ctx, o.ID)
		tenants = append(tenants, ts...)
	}
	manifest.RecordCounts["tenants"] = len(tenants)

	var domains []*domain.Domain
	var suppressions []*domain.Suppression
	for _, t := range tenants {
		if tenantID != "" && t.ID != tenantID {
			continue
		}
		doms, _ := repo.ListDomains(ctx, t.ID)
		domains = append(domains, doms...)
		supps, _ := repo.ListSuppressions(ctx, t.ID, 10000)
		suppressions = append(suppressions, supps...)
	}
	manifest.RecordCounts["domains"] = len(domains)
	manifest.RecordCounts["suppressions"] = len(suppressions)

	// Serialize JSON payloads
	files := make(map[string][]byte)
	orgsJSON, _ := json.MarshalIndent(orgs, "", "  ")
	files["organizations.json"] = orgsJSON

	tenantsJSON, _ := json.MarshalIndent(tenants, "", "  ")
	files["tenants.json"] = tenantsJSON

	domainsJSON, _ := json.MarshalIndent(domains, "", "  ")
	files["domains.json"] = domainsJSON

	suppsJSON, _ := json.MarshalIndent(suppressions, "", "  ")
	files["suppressions.json"] = suppsJSON

	// Compute hashes
	fnames := make([]string, 0, len(files))
	for fname := range files {
		fnames = append(fnames, fname)
	}
	sort.Strings(fnames)

	h := hmac.New(sha256.New, hmacSecret)
	for _, fname := range fnames {
		data := files[fname]
		sum := sha256.Sum256(data)
		hashHex := hex.EncodeToString(sum[:])
		manifest.FileHashes[fname] = hashHex
		h.Write([]byte(fname + ":" + hashHex))
	}
	manifest.HMAC = hex.EncodeToString(h.Sum(nil))

	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
	files["manifest.json"] = manifestJSON

	// Ensure parent directory exists
	if dir := filepath.Dir(outputPath); dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file %s: %w", outputPath, err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for fname, data := range files {
		hdr := &tar.Header{
			Name:    fname,
			Mode:    0600,
			Size:    int64(len(data)),
			ModTime: manifest.CreatedAt,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(data); err != nil {
			return nil, err
		}
	}

	return manifest, nil
}

// RestoreBackup verifies and unpacks a backup archive into the repository
func RestoreBackup(ctx context.Context, repo repository.Repository, inputPath string, hmacSecret []byte) (*BackupManifest, error) {
	if len(hmacSecret) == 0 {
		hmacSecret = []byte("mailvex-default-backup-hmac-secret-v1")
	}

	inFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup archive %s: %w", inputPath, err)
	}
	defer inFile.Close()

	gr, err := gzip.NewReader(inFile)
	if err != nil {
		return nil, fmt.Errorf("invalid gzip archive: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	files := make(map[string][]byte)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read error: %w", err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry %s: %w", hdr.Name, err)
		}
		files[hdr.Name] = data
	}

	manifestBytes, exists := files["manifest.json"]
	if !exists {
		return nil, errors.New("corrupted archive: missing manifest.json")
	}

	var manifest BackupManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest.json: %w", err)
	}

	// Verify HMAC signature
	manifestFnames := make([]string, 0, len(manifest.FileHashes))
	for fname := range manifest.FileHashes {
		manifestFnames = append(manifestFnames, fname)
	}
	sort.Strings(manifestFnames)

	h := hmac.New(sha256.New, hmacSecret)
	for _, fname := range manifestFnames {
		expectedHash := manifest.FileHashes[fname]
		content, hasFile := files[fname]
		if !hasFile {
			return nil, fmt.Errorf("archive missing declared file: %s", fname)
		}
		actualSum := sha256.Sum256(content)
		actualHash := hex.EncodeToString(actualSum[:])
		if actualHash != expectedHash {
			return nil, fmt.Errorf("integrity violation in %s: hash mismatch", fname)
		}
		h.Write([]byte(fname + ":" + expectedHash))
	}

	expectedHMAC := hex.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(manifest.HMAC), []byte(expectedHMAC)) {
		return nil, errors.New("cryptographic HMAC signature verification failed: archive may have been tampered with")
	}

	// Restore organizations
	if orgsBytes, ok := files["organizations.json"]; ok {
		var orgs []*domain.Organization
		if err := json.Unmarshal(orgsBytes, &orgs); err == nil {
			for _, o := range orgs {
				_ = repo.CreateOrganization(ctx, o)
			}
		}
	}

	// Restore tenants
	if tenantsBytes, ok := files["tenants.json"]; ok {
		var tenants []*domain.Tenant
		if err := json.Unmarshal(tenantsBytes, &tenants); err == nil {
			for _, t := range tenants {
				// Re-create or update
				_ = repo.CreateTenant(ctx, t)
			}
		}
	}

	// Restore domains
	if domainsBytes, ok := files["domains.json"]; ok {
		var domains []*domain.Domain
		if err := json.Unmarshal(domainsBytes, &domains); err == nil {
			for _, d := range domains {
				_ = repo.CreateDomain(ctx, d)
			}
		}
	}

	// Restore suppressions
	if suppsBytes, ok := files["suppressions.json"]; ok {
		var supps []*domain.Suppression
		if err := json.Unmarshal(suppsBytes, &supps); err == nil {
			for _, s := range supps {
				_ = repo.AddSuppression(ctx, s)
			}
		}
	}

	return &manifest, nil
}
