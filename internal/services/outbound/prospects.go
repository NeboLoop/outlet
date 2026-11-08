package outbound

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/google/uuid"
)

// ProspectInput is one creator (or any person) to reach. EmailSource says
// where the address was published for business: it is required, because
// outbound only writes to addresses people put out to be contacted.
type ProspectInput struct {
	Email        string            `json:"email"`
	FirstName    string            `json:"first_name,omitempty"`
	Name         string            `json:"name,omitempty"`
	Timezone     string            `json:"timezone,omitempty"`
	Platform     string            `json:"platform,omitempty"` // youtube, instagram, tiktok, ...
	ChannelURL   string            `json:"channel_url,omitempty"`
	AudienceSize int64             `json:"audience_size,omitempty"`
	Niche        string            `json:"niche,omitempty"`
	RecentVideo  string            `json:"recent_video,omitempty"`
	PersonalLine string            `json:"personal_line,omitempty"`
	EmailSource  string            `json:"email_source"`
	Fields       map[string]string `json:"fields,omitempty"`
}

// ProspectResult is the outcome for one prospect: accepted, held or refused.
type ProspectResult struct {
	Email  string `json:"email"`
	ID     string `json:"id,omitempty"`
	Status string `json:"status"` // added, held, duplicate, suppressed, refused
	Reason string `json:"reason,omitempty"`
}

// AddProspects imports a batch: it removes duplicates, checks suppression and
// verifies each address. Addresses that fail verification are kept as held,
// never sent.
func (s *Service) AddProspects(ctx context.Context, org db.Organization, inputs []ProspectInput) ([]ProspectResult, error) {
	if len(inputs) > maxBatch {
		return nil, refuse("at most %d prospects per call", maxBatch)
	}
	out := make([]ProspectResult, 0, len(inputs))
	seen := map[string]bool{}
	for _, in := range inputs {
		email := normEmail(in.Email)
		res := ProspectResult{Email: email}
		switch {
		case !strings.Contains(email, "@"):
			res.Status, res.Reason = "refused", "not an email address"
		case strings.TrimSpace(in.EmailSource) == "":
			res.Status, res.Reason = "refused", "email_source is required: where the address was published for business (channel About page, bio, website)"
		case seen[email]:
			res.Status, res.Reason = "duplicate", "listed twice in this batch"
		}
		seen[email] = true
		if res.Status != "" {
			out = append(out, res)
			continue
		}
		if s.suppressed(ctx, org.ID, email) {
			res.Status, res.Reason = "suppressed", "on the brand's suppression list, bounced or complained"
			out = append(out, res)
			continue
		}
		status, verify := "new", "ok"
		if ok, reason := s.verify(ctx, email); !ok {
			status, verify = "held", reason
		}
		fields, _ := json.Marshal(in.Fields)
		if in.Fields == nil {
			fields = []byte("{}")
		}
		p, err := s.db.CreateOutboundProspect(ctx, db.CreateOutboundProspectParams{
			ID: uuid.NewString(), OrgID: org.ID, Email: email,
			FirstName: strings.TrimSpace(in.FirstName), Name: strings.TrimSpace(in.Name), Timezone: in.Timezone,
			Platform: strings.ToLower(in.Platform), ChannelUrl: in.ChannelURL, AudienceSize: in.AudienceSize,
			Niche: in.Niche, RecentVideo: in.RecentVideo, PersonalLine: in.PersonalLine,
			EmailSource: strings.TrimSpace(in.EmailSource), CustomFields: string(fields),
			Status: status, VerifyResult: verify,
		})
		if err != nil {
			// ON CONFLICT DO NOTHING returns no row: already a prospect.
			if existing, gerr := s.db.GetOutboundProspectByEmail(ctx, db.GetOutboundProspectByEmailParams{OrgID: org.ID, Email: email}); gerr == nil {
				res.ID, res.Status, res.Reason = existing.ID, "duplicate", "already a prospect ("+existing.Status+")"
			} else {
				res.Status, res.Reason = "refused", err.Error()
			}
			out = append(out, res)
			continue
		}
		res.ID = p.ID
		if status == "held" {
			res.Status, res.Reason = "held", "failed verification: "+verify
		} else {
			res.Status = "added"
		}
		out = append(out, res)
	}
	return out, nil
}

// suppressed checks every list that stops a send: the brand's suppression
// list, and every bounce and complaint on the server.
func (s *Service) suppressed(ctx context.Context, orgID, email string) bool {
	if n, err := s.db.IsEmailSuppressed(ctx, db.IsEmailSuppressedParams{OrgID: orgID, Email: email}); err != nil || n > 0 {
		return true
	}
	if n, err := s.db.IsEmailBlocked(ctx, db.IsEmailBlockedParams{Email: email, Email2: email}); err != nil || n > 0 {
		return true
	}
	return false
}

// suppress adds an address to the brand's suppression list, which
// transactional and marketing mail check too.
func (s *Service) suppress(ctx context.Context, orgID, email, reason string) {
	_, _ = s.db.AddToSuppressionList(ctx, db.AddToSuppressionListParams{
		OrgID: orgID, Email: normEmail(email),
		Reason: sql.NullString{String: reason, Valid: true}, Source: sql.NullString{String: "outbound", Valid: true},
	})
}

func customFields(p db.OutboundProspect) map[string]string {
	m := map[string]string{}
	_ = json.Unmarshal([]byte(p.CustomFields), &m)
	return m
}
