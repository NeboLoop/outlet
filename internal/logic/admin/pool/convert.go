package pool

import (
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/types"
)

func domainInfo(d db.PoolDomain) types.PoolDomainInfo {
	return types.PoolDomainInfo{
		Id:           d.ID,
		Domain:       d.Domain,
		Status:       d.Status,
		PausedReason: d.PausedReason.String,
		CreatedAt:    d.CreatedAt,
	}
}

// mailboxInfo never carries the mailbox password.
func mailboxInfo(mb db.PoolMailbox) types.PoolMailboxInfo {
	return types.PoolMailboxInfo{
		Id:               mb.ID,
		DomainId:         mb.DomainID,
		Address:          mb.Address,
		SmtpHost:         mb.SmtpHost,
		SmtpPort:         int(mb.SmtpPort),
		SmtpUsername:     mb.SmtpUsername,
		WarmStartedAt:    mb.WarmStartedAt,
		Status:           mb.Status,
		PausedReason:     mb.PausedReason.String,
		PauseKind:        mb.PausedKind.String,
		TenantOrgId:      mb.TenantOrgID.String,
		Tenant:           mb.Tenant.String,
		TenantAssignedAt: mb.TenantAssignedAt.String,
		TenantReleasedAt: mb.TenantReleasedAt.String,
		CreatedAt:        mb.CreatedAt,
	}
}
