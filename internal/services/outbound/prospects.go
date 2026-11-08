package outbound

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
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

// ProspectQuery picks prospects: those on one list (empty: any), with one
// status there, matching a search of email and name, or only those free to
// enroll. Limit and Offset page through them, newest first.
type ProspectQuery struct {
	ListID     string
	Status     string
	Search     string
	Enrollable bool
	Limit      int
	Offset     int
}

// ListRef names one outreach list.
type ListRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ProspectListStatus is a person's status on one outreach list.
type ProspectListStatus struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// CampaignRef names one outreach campaign.
type CampaignRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ProspectInfo is one person on the brand's outreach lists. Enrollable says
// whether a campaign can take them now: a prospect (on the list the query
// names), in no outreach campaign, and not suppressed.
type ProspectInfo struct {
	ID               string               `json:"id"`
	Email            string               `json:"email"`
	Name             string               `json:"name"`
	EmailSource      string               `json:"email_source"`
	ValidationStatus string               `json:"validation_status"`
	CreatedAt        string               `json:"created_at"`
	Lists            []ProspectListStatus `json:"lists"`
	Campaigns        []CampaignRef        `json:"campaigns"`
	Suppressed       bool                 `json:"suppressed"`
	Enrollable       bool                 `json:"enrollable"`
}

// ProspectPage is one page of prospects, how many match in all, and the
// brand's outreach lists to filter by.
type ProspectPage struct {
	Prospects []ProspectInfo `json:"prospects"`
	Total     int64          `json:"total"`
	Lists     []ListRef      `json:"lists"`
}

// ListProspects reads the brand's prospects, one row per person.
func (s *Service) ListProspects(ctx context.Context, org db.Organization, q ProspectQuery) (ProspectPage, error) {
	page := ProspectPage{Prospects: []ProspectInfo{}, Lists: []ListRef{}}
	var listID int64
	if q.ListID != "" {
		id, err := strconv.ParseInt(q.ListID, 10, 64)
		if err != nil {
			return page, refuse("No such outreach list.")
		}
		listID = id
	}
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 50
	}
	lists, err := s.db.ListOutreachLists(ctx, org.ID)
	if err != nil {
		return page, err
	}
	for _, l := range lists {
		page.Lists = append(page.Lists, ListRef{ID: strconv.FormatInt(l.ID, 10), Name: l.Name})
	}
	rows, err := s.db.ListProspects(ctx, db.ListProspectsParams{
		OrgID: org.ID, ListID: listID, Status: strings.TrimSpace(q.Status), Search: strings.TrimSpace(q.Search),
		Enrollable: boolInt(q.Enrollable), LimitCount: int64(q.Limit), OffsetCount: int64(max(q.Offset, 0)),
	})
	if err != nil || len(rows) == 0 {
		return page, err
	}
	ids := make([]string, 0, len(rows))
	byID := map[string]int{}
	for i, r := range rows {
		ids = append(ids, r.ID)
		byID[r.ID] = i
		page.Total = r.Total
		page.Prospects = append(page.Prospects, ProspectInfo{
			ID: r.ID, Email: r.Email, Name: r.Name, EmailSource: r.Source.String, ValidationStatus: r.ValidationStatus.String,
			CreatedAt: r.CreatedAt.String, Lists: []ProspectListStatus{}, Campaigns: []CampaignRef{},
			Suppressed: r.Suppressed == 1, Enrollable: r.Prospect == 1 && r.InCampaign == 0 && r.Suppressed == 0,
		})
	}
	subs, err := s.db.ListProspectSubscriptions(ctx, db.ListProspectSubscriptionsParams{OrgID: org.ID, ContactIds: ids})
	if err != nil {
		return page, err
	}
	for _, sub := range subs {
		p := &page.Prospects[byID[sub.ContactID]]
		p.Lists = append(p.Lists, ProspectListStatus{ID: strconv.FormatInt(sub.ListID, 10), Name: sub.ListName, Status: sub.Status.String})
	}
	nullIDs := make([]sql.NullString, 0, len(ids))
	for _, id := range ids {
		nullIDs = append(nullIDs, sql.NullString{String: id, Valid: true})
	}
	camps, err := s.db.ListActiveOutreachCampaignsFor(ctx, db.ListActiveOutreachCampaignsForParams{OrgID: sql.NullString{String: org.ID, Valid: true}, ContactIds: nullIDs})
	if err != nil {
		return page, err
	}
	for _, c := range camps {
		p := &page.Prospects[byID[c.ContactID.String]]
		p.Campaigns = append(p.Campaigns, CampaignRef{ID: c.ID, Name: c.Name})
	}
	return page, nil
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
