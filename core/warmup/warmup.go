package warmup

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

type Manager struct {
	mu   sync.RWMutex
	repo repository.Repository
}

func NewManager(repo repository.Repository) *Manager {
	return &Manager{
		repo: repo,
	}
}

// EnrollIP begins warmup for an IP address on a specific plan
func (m *Manager) EnrollIP(ctx context.Context, ipAddressID, ipStr, planID string) (*domain.IPWarmupState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	plan, err := m.repo.GetWarmupPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("warmup plan not found: %w", err)
	}

	state := &domain.IPWarmupState{
		ID:               crypto.NewULID(crypto.PrefixIPWarmupState),
		IPAddressID:      ipAddressID,
		IPAddress:        ipStr,
		PlanID:           plan.ID,
		CurrentDay:       1,
		CurrentDayVolume: 0,
		ProviderVolume:   make(map[string]int),
		Status:           "warming",
		StartedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	if err := m.repo.SaveIPWarmupState(ctx, state); err != nil {
		return nil, err
	}

	return state, nil
}

// CanSend checks whether an IP has available volume for the current day and destination provider
func (m *Manager) CanSend(ctx context.Context, ipAddressID, provider string) (bool, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, err := m.repo.GetIPWarmupState(ctx, ipAddressID)
	if err != nil {
		// Not in warmup, unrestricted
		return true, "", nil
	}

	if state.Status == "completed" {
		return true, "", nil
	}

	if state.Status == "held" {
		reason := "IP warmup held"
		if state.HeldReason != nil && *state.HeldReason != "" {
			reason = fmt.Sprintf("IP warmup held: %s", *state.HeldReason)
		}
		return false, reason, nil
	}

	if state.Status == "paused" {
		return false, "IP warmup is paused", nil
	}

	plan, err := m.repo.GetWarmupPlan(ctx, state.PlanID)
	if err != nil {
		return false, "warmup plan not found", err
	}

	// Find day configuration
	var dayCfg *domain.WarmupPlanDay
	for _, d := range plan.Days {
		if d.DayNumber == state.CurrentDay {
			dayCfg = d
			break
		}
	}

	// If no exact day configuration found, use the latest day or default
	if dayCfg == nil && len(plan.Days) > 0 {
		dayCfg = plan.Days[len(plan.Days)-1]
	}

	if dayCfg != nil {
		if dayCfg.MaxVolume > 0 && state.CurrentDayVolume >= dayCfg.MaxVolume {
			return false, fmt.Sprintf("Daily warmup cap reached (%d/%d)", state.CurrentDayVolume, dayCfg.MaxVolume), nil
		}

		cleanProvider := strings.ToLower(strings.TrimSpace(provider))
		if cleanProvider != "" && dayCfg.ProviderCaps != nil {
			if capVal, hasCap := dayCfg.ProviderCaps[cleanProvider]; hasCap && capVal > 0 {
				currentProvVol := 0
				if state.ProviderVolume != nil {
					currentProvVol = state.ProviderVolume[cleanProvider]
				}
				if currentProvVol >= capVal {
					return false, fmt.Sprintf("Provider warmup cap for %s reached (%d/%d)", cleanProvider, currentProvVol, capVal), nil
				}
			}
		}
	}

	return true, "", nil
}

// RecordSend increments the volume sent for today
func (m *Manager) RecordSend(ctx context.Context, ipAddressID, provider string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.repo.GetIPWarmupState(ctx, ipAddressID)
	if err != nil {
		// Not warming, no-op
		return nil
	}

	if state.Status != "warming" {
		return nil
	}

	state.CurrentDayVolume++
	cleanProvider := strings.ToLower(strings.TrimSpace(provider))
	if cleanProvider != "" {
		if state.ProviderVolume == nil {
			state.ProviderVolume = make(map[string]int)
		}
		state.ProviderVolume[cleanProvider]++
	}

	return m.repo.SaveIPWarmupState(ctx, state)
}

// AdvanceDay advances an IP to the next warmup day if reputation metrics are acceptable
func (m *Manager) AdvanceDay(ctx context.Context, ipAddressID string, bounceRate, complaintRate float64, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.repo.GetIPWarmupState(ctx, ipAddressID)
	if err != nil {
		return err
	}

	if state.Status == "completed" {
		return nil
	}

	// Check reputation constraints (§11.5: bounce > 5% or complaint > 0.1% holds warmup)
	if !force {
		if bounceRate > 0.05 || complaintRate > 0.001 {
			heldReason := fmt.Sprintf("Automated hold: bounce rate %.2f%% or complaint rate %.3f%% exceeded safe thresholds", bounceRate*100, complaintRate*100)
			state.Status = "held"
			state.HeldReason = &heldReason
			return m.repo.SaveIPWarmupState(ctx, state)
		}
	}

	plan, err := m.repo.GetWarmupPlan(ctx, state.PlanID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	state.LastAdvancedAt = &now
	state.CurrentDay++
	state.CurrentDayVolume = 0
	state.ProviderVolume = make(map[string]int)
	state.Status = "warming"
	state.HeldReason = nil

	if state.CurrentDay > plan.TotalDays {
		state.Status = "completed"
	}

	return m.repo.SaveIPWarmupState(ctx, state)
}

// HoldWarmup manually holds warmup
func (m *Manager) HoldWarmup(ctx context.Context, ipAddressID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.repo.GetIPWarmupState(ctx, ipAddressID)
	if err != nil {
		return err
	}

	state.Status = "held"
	state.HeldReason = &reason
	return m.repo.SaveIPWarmupState(ctx, state)
}

// ResumeWarmup resumes a held or paused warmup
func (m *Manager) ResumeWarmup(ctx context.Context, ipAddressID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.repo.GetIPWarmupState(ctx, ipAddressID)
	if err != nil {
		return err
	}

	state.Status = "warming"
	state.HeldReason = nil
	return m.repo.SaveIPWarmupState(ctx, state)
}
