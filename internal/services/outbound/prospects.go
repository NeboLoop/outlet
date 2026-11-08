package outbound

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/google/uuid"
)

// Prospects are Outlet contacts on one of the brand's outreach lists, with
// status 'prospect': they never signed up, so campaigns and sign-up mail
// (which go to 'active' subscribers) never reach them. Their details are the
// list's custom fields, which merge tags read like any other. Finding them
// happens outside Outlet; this is how they come in.

// ProspectInput is one person to reach. EmailSource says where the address
// was published for business; it is required, because outreach only writes
// to addresses people put out to be contacted.
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

// ProspectResult is the outcome for one person.
type ProspectResult struct {
	Email  string `json:"email"`
	ID     string `json:"id,omitempty"` // the contact
	Status string `json:"status"`       // added, held, duplicate, suppressed, refused
	Reason string `json:"reason,omitempty"`
}

// DefaultOutreachList is the list prospects join when no list is named.
const DefaultOutreachList = "Outreach"

var fieldKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,40}$`)

// AddProspects imports a batch into an outreach list (made if missing):
// it removes duplicates, checks suppression and verifies each address.
// Addresses that fail verification are kept as held, and never sent to.
func (s *Service) AddProspects(ctx context.Context, org db.Organization, listName string, inputs []ProspectInput) ([]ProspectResult, error) {
	if len(inputs) > maxBatch {
		return nil, refuse("at most %d prospects per call", maxBatch)
	}
	list, err := s.outreachList(ctx, org, listName)
	if err != nil {
		return nil, err
	}
	out := make([]ProspectResult, 0, len(inputs))
	seen := map[string]bool{}
	for _, in := range inputs {
		res := s.addProspect(ctx, org, list, in, seen)
		out = append(out, res)
	}
	return out, nil
}

func (s *Service) addProspect(ctx context.Context, org db.Organization, list db.EmailList, in ProspectInput, seen map[string]bool) ProspectResult {
	address := normEmail(in.Email)
	res := ProspectResult{Email: address}
	switch {
	case !strings.Contains(address, "@"):
		res.Status, res.Reason = "refused", "not an email address"
		return res
	case strings.TrimSpace(in.EmailSource) == "":
		res.Status, res.Reason = "refused", "email_source is required: where the address was published for business (channel About page, bio, website)"
		return res
	case seen[address]:
		res.Status, res.Reason = "duplicate", "listed twice in this batch"
		return res
	}
	seen[address] = true
	if blocked, err := s.db.IsSendBlocked(ctx, db.IsSendBlockedParams{Email: address, OrgID: org.ID, Domain: domainOf(address)}); err != nil || blocked == 1 {
		res.Status, res.Reason = "suppressed", "on the brand's suppression list, or bounced or complained before"
		return res
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.TrimSpace(in.FirstName)
	}
	contact, err := s.db.GetContactByOrgAndEmail(ctx, db.GetContactByOrgAndEmailParams{OrgID: sql.NullString{String: org.ID, Valid: true}, Email: address})
	if errors.Is(err, sql.ErrNoRows) {
		contact, err = s.db.CreateContact(ctx, db.CreateContactParams{
			ID: uuid.NewString(), OrgID: sql.NullString{String: org.ID, Valid: true}, Name: name, Email: address,
			Source: sql.NullString{String: strings.TrimSpace(in.EmailSource), Valid: true},
		})
	}
	if err != nil {
		res.Status, res.Reason = "refused", err.Error()
		return res
	}
	res.ID = contact.ID

	status := "prospect"
	ok, reason := s.verify(ctx, org.ID, address)
	if !ok {
		status = "held"
	}
	sub, err := s.db.CreateListSubscriber(ctx, db.CreateListSubscriberParams{ID: uuid.NewString(), ListID: list.ID, ContactID: contact.ID, Status: sql.NullString{String: status, Valid: true}})
	if errors.Is(err, sql.ErrNoRows) {
		res.Status, res.Reason = "duplicate", "already on "+list.Name
		return res
	}
	if err != nil {
		res.Status, res.Reason = "refused", err.Error()
		return res
	}
	fields := map[string]string{
		"platform": strings.ToLower(in.Platform), "channel_url": in.ChannelURL, "niche": in.Niche,
		"recent_video": in.RecentVideo, "personal_line": in.PersonalLine, "timezone": in.Timezone,
	}
	if in.AudienceSize > 0 {
		fields["audience_size"] = fmt.Sprint(in.AudienceSize)
	}
	for k, v := range in.Fields {
		if fieldKeyRe.MatchString(k) {
			fields[k] = v
		}
	}
	for k, v := range fields {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if err := s.setField(ctx, list.ID, sub.ID, k, v); err != nil {
			res.Status, res.Reason = "refused", err.Error()
			return res
		}
	}
	if status == "held" {
		res.Status, res.Reason = "held", "failed verification: "+reason
	} else {
		res.Status = "added"
	}
	return res
}

// outreachList finds the brand's outreach list by name, or makes it.
func (s *Service) outreachList(ctx context.Context, org db.Organization, name string) (db.EmailList, error) {
	if name = strings.TrimSpace(name); name == "" {
		name = DefaultOutreachList
	}
	lists, err := s.db.ListOutreachLists(ctx, org.ID)
	if err != nil {
		return db.EmailList{}, err
	}
	for _, l := range lists {
		if strings.EqualFold(l.Name, name) {
			return l, nil
		}
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	list, err := s.db.CreateEmailList(ctx, db.CreateEmailListParams{
		PublicID: hex.EncodeToString(b), OrgID: org.ID, Name: name, Slug: slugify(name) + "-" + hex.EncodeToString(b[:3]),
		Description: sql.NullString{String: "Outreach prospects. They never signed up: campaigns don't reach them.", Valid: true},
		DoubleOptin: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		return list, err
	}
	if err := s.db.SetListKind(ctx, db.SetListKindParams{Kind: "outreach", ID: list.ID, OrgID: org.ID}); err != nil {
		return list, err
	}
	list.Kind = "outreach"
	return list, nil
}

// setField writes one custom field value for a subscriber, adding the field
// to the list when it is new.
func (s *Service) setField(ctx context.Context, listID int64, subscriberID, key, value string) error {
	f, err := s.db.GetCustomFieldByKey(ctx, db.GetCustomFieldByKeyParams{ListID: listID, FieldKey: key})
	if errors.Is(err, sql.ErrNoRows) {
		f, err = s.db.CreateCustomField(ctx, db.CreateCustomFieldParams{
			ID: uuid.NewString(), ListID: listID, Name: strings.ReplaceAll(key, "_", " "), FieldKey: key, FieldType: "text",
		})
	}
	if err != nil {
		return err
	}
	_, err = s.db.UpsertCustomFieldValue(ctx, db.UpsertCustomFieldValueParams{
		ID: uuid.NewString(), SubscriberID: subscriberID, FieldID: f.ID, Value: sql.NullString{String: value, Valid: true},
	})
	return err
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// suppress records an opt-out on the brand's suppression list, which every
// kind of mail checks.
func (s *Service) suppress(ctx context.Context, orgID, address, reason string) {
	_, _ = s.db.AddToSuppressionList(ctx, db.AddToSuppressionListParams{
		OrgID: orgID, Email: normEmail(address),
		Reason: sql.NullString{String: reason, Valid: true}, Source: sql.NullString{String: "outbound", Valid: true},
	})
}

// suppressed is the same check every send makes.
func (s *Service) suppressed(ctx context.Context, orgID, address string) bool {
	blocked, err := s.db.IsSendBlocked(ctx, db.IsSendBlockedParams{Email: normEmail(address), OrgID: orgID, Domain: domainOf(address)})
	return err != nil || blocked == 1
}
