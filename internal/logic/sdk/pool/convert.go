package pool

import (
	"time"

	"github.com/outlet-sh/outlet/internal/services/mailpool"
	"github.com/outlet-sh/outlet/internal/types"
)

func mailboxCapacities(c *mailpool.Capacity) []types.PoolMailboxCapacity {
	out := make([]types.PoolMailboxCapacity, 0, len(c.Mailboxes))
	for _, m := range c.Mailboxes {
		item := types.PoolMailboxCapacity{
			Id:             m.ID,
			Address:        m.Address,
			Status:         m.Status,
			PausedReason:   m.PausedReason,
			PauseKind:      m.PauseKind,
			Movable:        m.Movable,
			RemainingToday: m.RemainingToday,
		}
		if !m.NextAllowedAt.IsZero() {
			item.NextAllowedAt = m.NextAllowedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, item)
	}
	return out
}
