package repository

import (
	"context"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
)

func TestCrossTenantIsolation(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	// 1. Create Organization
	orgID := crypto.NewULID(crypto.PrefixOrg)
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Acme Corp",
		Slug:      "acme",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// 2. Create Tenant A and Tenant B
	tenantA_ID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantA_ID,
		OrganizationID: orgID,
		Name:           "Tenant Alpha",
		Slug:           "alpha",
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	tenantB_ID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantB_ID,
		OrganizationID: orgID,
		Name:           "Tenant Beta",
		Slug:           "beta",
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	// 3. Create User 1 (member of Tenant A only) and User 2 (member of Tenant B only)
	user1_ID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(ctx, &domain.User{
		ID:              user1_ID,
		OrganizationID:  orgID,
		Email:           "user1@alpha.com",
		Status:          "active",
		IsPlatformAdmin: false,
		CreatedAt:       time.Now(),
	})
	_ = repo.AddTenantMembership(ctx, &domain.TenantMembership{
		ID:        crypto.NewULID(crypto.PrefixMembership),
		TenantID:  tenantA_ID,
		UserID:    user1_ID,
		RoleID:    "role_tenant_owner",
		CreatedAt: time.Now(),
	})

	user2_ID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(ctx, &domain.User{
		ID:              user2_ID,
		OrganizationID:  orgID,
		Email:           "user2@beta.com",
		Status:          "active",
		IsPlatformAdmin: false,
		CreatedAt:       time.Now(),
	})
	_ = repo.AddTenantMembership(ctx, &domain.TenantMembership{
		ID:        crypto.NewULID(crypto.PrefixMembership),
		TenantID:  tenantB_ID,
		UserID:    user2_ID,
		RoleID:    "role_tenant_owner",
		CreatedAt: time.Now(),
	})

	// 4. Verify User 1 cannot see or belong to Tenant B
	user1Tenants, err := repo.ListTenantsForUser(ctx, user1_ID)
	if err != nil {
		t.Fatalf("failed to list tenants for user 1: %v", err)
	}
	if len(user1Tenants) != 1 || user1Tenants[0].ID != tenantA_ID {
		t.Fatalf("expected user 1 to see only Tenant Alpha, saw %d tenants", len(user1Tenants))
	}

	// 5. Verify User 1 cannot get membership for Tenant B
	_, err = repo.GetTenantMembership(ctx, tenantB_ID, user1_ID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound when checking user 1 in Tenant Beta, got %v", err)
	}

	// 6. Test Platform Admin can see both tenants
	adminID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(ctx, &domain.User{
		ID:              adminID,
		OrganizationID:  orgID,
		Email:           "admin@platform.com",
		Status:          "active",
		IsPlatformAdmin: true,
		CreatedAt:       time.Now(),
	})
	adminTenants, err := repo.ListTenantsForUser(ctx, adminID)
	if err != nil {
		t.Fatalf("failed to list tenants for platform admin: %v", err)
	}
	if len(adminTenants) < 2 {
		t.Fatalf("expected platform admin to see all tenants, saw %d", len(adminTenants))
	}

	// 7. Verify Audit Log isolation: tenant-scoped query returns only that tenant's logs
	_ = repo.RecordAuditLog(ctx, &domain.AuditLog{
		ID:           crypto.NewULID(crypto.PrefixAudit),
		TenantID:     &tenantA_ID,
		Action:       "domain.created",
		ResourceType: "domain",
		ResourceID:   "dom_1",
		OccurredAt:   time.Now(),
	})
	_ = repo.RecordAuditLog(ctx, &domain.AuditLog{
		ID:           crypto.NewULID(crypto.PrefixAudit),
		TenantID:     &tenantB_ID,
		Action:       "domain.created",
		ResourceType: "domain",
		ResourceID:   "dom_2",
		OccurredAt:   time.Now(),
	})

	logsA, err := repo.ListAuditLogs(ctx, &tenantA_ID, 50)
	if err != nil {
		t.Fatalf("failed to list logs for tenant A: %v", err)
	}
	if len(logsA) != 1 || *logsA[0].TenantID != tenantA_ID {
		t.Fatalf("tenant A query returned mismatched audit logs: %+v", logsA)
	}
}
