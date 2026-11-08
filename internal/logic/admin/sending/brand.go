package sending

import (
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/types"
)

// The sending routes manage the pathways of the brand named by X-Org-Id,
// through the email service (the same one the MCP tools use). Members read;
// changing a pathway or a mailbox's limit needs the brand's admin.

func limitsOut(l email.Limits) types.SendingLimits {
	return types.SendingLimits{
		DailyLimit: l.DailyLimit, WarmupStart: l.WarmupStart, WarmupStep: l.WarmupStep,
		MinGapSeconds: l.MinGapSeconds, MaxGapSeconds: l.MaxGapSeconds,
		PauseBounceRate: l.PauseBounceRate, PauseMinBounces: l.PauseMinBounces,
		PauseSpamBlocks: l.PauseSpamBlocks, PauseComplaints: l.PauseComplaints,
	}
}

func limitsIn(l *types.SendingLimits) *email.Limits {
	if l == nil {
		return nil
	}
	return &email.Limits{
		DailyLimit: l.DailyLimit, WarmupStart: l.WarmupStart, WarmupStep: l.WarmupStep,
		MinGapSeconds: l.MinGapSeconds, MaxGapSeconds: l.MaxGapSeconds,
		PauseBounceRate: l.PauseBounceRate, PauseMinBounces: l.PauseMinBounces,
		PauseSpamBlocks: l.PauseSpamBlocks, PauseComplaints: l.PauseComplaints,
	}
}

func pathwayOut(p email.Pathway) types.SendingPathway {
	return types.SendingPathway{
		Id: p.ID, Name: p.Name, Provider: p.Provider, Kinds: p.Kinds,
		AWSRegion: p.AWSRegion, AWSKeysSet: p.AWSKeysSet,
		SMTPHost: p.SMTPHost, SMTPPort: p.SMTPPort, SMTPUsername: p.SMTPUsername, SMTPPasswordSet: p.SMTPPasswordSet,
		Limits: limitsOut(p.Limits), Domains: domainsOut(p.Domains), Mailboxes: int64(p.Mailboxes),
		Warning: p.Warning, CreatedAt: p.CreatedAt,
	}
}

func domainsOut(ds []email.PathwayDomain) []types.SendingPathwayDomain {
	out := make([]types.SendingPathwayDomain, 0, len(ds))
	for _, d := range ds {
		out = append(out, types.SendingPathwayDomain{Id: d.ID, Domain: d.Domain, Purpose: d.Purpose})
	}
	return out
}
