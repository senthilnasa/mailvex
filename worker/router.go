package worker

import (
	"context"
	"sync"

	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

type RouteResult struct {
	Transport    string // "direct_mx" or "relay"
	IPID         *string
	SourceIP     string
	EHLOHostname string
	Relay        *domain.Relay
}

type Router struct {
	repo   repository.Repository
	nodeID string
	mu     sync.Mutex
	rrIdx  int
}

func NewRouter(repo repository.Repository, nodeID string) *Router {
	if nodeID == "" {
		nodeID = "node-1"
	}
	return &Router{
		repo:   repo,
		nodeID: nodeID,
	}
}

// SelectRoute chooses transport, IP pool, and source IP bound to this node (§11.3, §15)
func (r *Router) SelectRoute(ctx context.Context, tenantID, streamID, domainID string) (*RouteResult, error) {
	// 1. Check if a relay is configured for this tenant or platform (§8.4)
	relays, err := r.repo.ListRelays(ctx, tenantID)
	if err == nil && len(relays) > 0 {
		for _, rel := range relays {
			if rel.IsActive {
				return &RouteResult{
					Transport:    "relay",
					EHLOHostname: "mailvex.relay",
					Relay:        rel,
				}, nil
			}
		}
	}

	// 2. Select IP Pool (default pool fallback)
	pools, err := r.repo.ListIPPools(ctx, tenantID)
	var poolID string
	if err == nil && len(pools) > 0 {
		for _, p := range pools {
			if p.IsDefault {
				poolID = p.ID
				break
			}
		}
		if poolID == "" {
			poolID = pools[0].ID
		}
	} else {
		poolID = "pool_default"
	}

	// 3. Find active IP bound to this node in the selected pool
	ips, err := r.repo.ListIPAddresses(ctx, poolID)
	var nodeIPs []*domain.IPAddress
	if err == nil {
		for _, ip := range ips {
			if ip.NodeID == r.nodeID && (ip.Status == "active" || ip.Status == "warming") {
				// Warmup cap check (§11.5)
				if ip.Status == "warming" && ip.MaxDailyVolume > 0 && ip.SentToday >= ip.MaxDailyVolume {
					continue // Cap reached for today
				}
				nodeIPs = append(nodeIPs, ip)
			}
		}
	}

	// If no bound IPs in the specific pool, fallback to any active IP on this node
	if len(nodeIPs) == 0 {
		allNodeIPs, _ := r.repo.ListIPAddressesByNode(ctx, r.nodeID)
		for _, ip := range allNodeIPs {
			if ip.Status == "active" || ip.Status == "warming" {
				nodeIPs = append(nodeIPs, ip)
			}
		}
	}

	if len(nodeIPs) > 0 {
		r.mu.Lock()
		selected := nodeIPs[r.rrIdx%len(nodeIPs)]
		r.rrIdx++
		r.mu.Unlock()

		return &RouteResult{
			Transport:    "direct_mx",
			IPID:         &selected.ID,
			SourceIP:     selected.IP,
			EHLOHostname: selected.EHLOHostname,
		}, nil
	}

	// Default fallback when running on local machine / dev environment
	return &RouteResult{
		Transport:    "direct_mx",
		SourceIP:     "127.0.0.1",
		EHLOHostname: "mailvex.local",
	}, nil
}
