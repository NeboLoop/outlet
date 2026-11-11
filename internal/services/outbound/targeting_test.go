package outbound

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/events"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// business is a prospect for business outreach.
func business(email, first, company, country string) ProspectInput {
	return ProspectInput{Email: email, FirstName: first, Company: company, Title: "Office Manager", Country: country,
		Industry: "dental", CompanySize: "12", PersonalLine: "Saw the new clinic in Provo.", EmailSource: "clinic website, contact page", Timezone: "UTC"}
}

var businessSteps = []StepInput{
	{Subject: "{Quick question|A question} for {{company}}", Body: "Hi {{first_name}},\n\n{{personal_line}} Worth a look?\n\n{{sender_first}}"},
	{DelayDays: 2, Body: "Following up, {{first_name}}."},
}

// addTo adds prospects to a segment and verifies them, returning each
// address's result after verification.
func (h *harness) addTo(list string, in ...ProspectInput) map[string]ProspectResult {
	res, err := h.svc.AddProspects(h.ctx, h.org, list, in)
	require.NoError(h.t, err)
	h.svc.VerifyProspects(h.ctx)
	l, err := h.svc.outreachList(h.ctx, h.org, list)
	require.NoError(h.t, err)
	out := map[string]ProspectResult{}
	for _, r := range res {
		if r.Status == "verifying" {
			sub, err := h.store.GetListSubscriber(h.ctx, db.GetListSubscriberParams{ListID: l.ID, ContactID: r.ID})
			require.NoError(h.t, err)
			r.Status = sub.Status.String
		}
		out[r.Email] = r
	}
	return out
}

// subscriber puts an address on one of the brand's newsletter lists: a
// customer, as far as outreach is concerned.
func (h *harness) subscriber(address string) {
	list, err := h.store.CreateEmailList(h.ctx, db.CreateEmailListParams{PublicID: uuid.NewString()[:12], OrgID: h.org.ID, Name: "Newsletter", Slug: "newsletter-" + uuid.NewString()[:6]})
	require.NoError(h.t, err)
	c, err := h.store.CreateContact(h.ctx, db.CreateContactParams{ID: uuid.NewString(), OrgID: sql.NullString{String: h.org.ID, Valid: true}, Name: "Customer", Email: address})
	require.NoError(h.t, err)
	_, err = h.store.CreateListSubscriber(h.ctx, db.CreateListSubscriberParams{ID: uuid.NewString(), ListID: list.ID, ContactID: c.ID, Status: sql.NullString{String: "active", Valid: true}})
	require.NoError(h.t, err)
}

// A segment is an outreach list with a definition: prospects added to it are
// checked against its filters, a campaign on it enrolls its prospects, and
// results break down by it.
func TestSegments(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	h.setup()

	sg, err := h.svc.SaveSegment(ctx, h.org, "", SegmentInput{Name: "Utah dental", Description: "Dental clinics in Utah, 10 to 50 staff.",
		Filters: &SegmentFilters{Countries: []string{"US", "us"}, Industries: []string{"dental"}, CompanySizes: []string{"11-50"}, TitleKeywords: []string{"manager"}}})
	require.NoError(t, err)
	assert.Equal(t, "Utah dental", sg.Name)
	assert.Equal(t, []string{"US"}, sg.Filters.Countries, "filters are cleaned of duplicates")
	_, err = h.svc.SaveSegment(ctx, h.org, "", SegmentInput{Name: "Bad", Filters: &SegmentFilters{CompanySizes: []string{"big"}}})
	assert.ErrorContains(t, err, "isn't a range")

	got := h.addTo("Utah dental", business("amy@clinic-a.com", "Amy", "Clinic A", "US"), business("bo@clinic-b.ca", "Bo", "Clinic B", "CA"))
	assert.Equal(t, "prospect", got["amy@clinic-a.com"].Status)
	assert.Equal(t, "prospect", got["bo@clinic-b.ca"].Status, "a filter mismatch is noted, never refused")
	res, _ := h.svc.AddProspects(ctx, h.org, "Utah dental", []ProspectInput{business("cy@clinic-c.ca", "Cy", "Clinic C", "CA")})
	assert.Contains(t, res[0].Reason, "country CA is not one of US")

	// Description and filters update; leaving filters out keeps them.
	sg, err = h.svc.SaveSegment(ctx, h.org, sg.ID, SegmentInput{Description: "Dental clinics in Utah."})
	require.NoError(t, err)
	assert.Equal(t, "Dental clinics in Utah.", sg.Description)
	assert.Equal(t, []string{"dental"}, sg.Filters.Industries)

	// A campaign on the segment, with no prospect ids, enrolls its prospects.
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "Dental 1", List: "Utah dental", Steps: businessSteps})
	require.NoError(t, err)
	assert.Equal(t, 2, camp.Enrolled, "the segment's verified prospects; one still verifying waits")
	require.NotEmpty(t, camp.Drafts)
	assert.Contains(t, camp.Drafts[0].Subject, "Clinic", "company fields are merge tags")
	_, err = h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "Nobody", List: "No such segment", Steps: businessSteps})
	assert.ErrorContains(t, err, "No segment")

	// Another segment takes existing prospects, with their fields.
	other, err := h.svc.SaveSegment(ctx, h.org, "", SegmentInput{Name: "Canada", Filters: &SegmentFilters{Countries: []string{"CA"}}})
	require.NoError(t, err)
	_, err = h.svc.SaveSegment(ctx, h.org, other.ID, SegmentInput{Name: "utah DENTAL"})
	assert.ErrorContains(t, err, "already called", "two segments never share a name")
	moved, err := h.svc.AddToSegment(ctx, h.org, other.ID, []string{got["bo@clinic-b.ca"].ID})
	require.NoError(t, err)
	assert.Equal(t, "prospect", moved[0].Status)
	l, err := h.svc.segment(ctx, h.org, "Canada")
	require.NoError(t, err)
	assert.Equal(t, "Clinic B", h.seq.GetCustomFieldsForContact(ctx, got["bo@clinic-b.ca"].ID, l.ID)["company"])

	// The overview breaks results down by segment.
	o, err := h.svc.Overview(ctx, h.org)
	require.NoError(t, err)
	bySeg := map[string]Segment{}
	for _, s := range o.Segments {
		bySeg[s.Name] = s
	}
	assert.EqualValues(t, 1, bySeg["Utah dental"].Results.Campaigns)
	assert.EqualValues(t, 1, bySeg["Canada"].Prospects["prospect"])
	require.Len(t, o.Campaigns, 1)
	assert.Equal(t, "Utah dental", o.Campaigns[0].Segment)
}

// Materials: kept per brand, for one segment or all; the brief hands them
// over with the profile and merge tags; a campaign pins the ones it used.
func TestMaterialsAndBrief(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	h.setup()
	_, err := h.svc.SaveSegment(ctx, h.org, "", SegmentInput{Name: "Utah dental"})
	require.NoError(t, err)
	got := h.addTo("Utah dental", business("amy@clinic-a.com", "Amy", "Clinic A", "US"))

	_, err = h.svc.SaveMaterial(ctx, h.org, "", MaterialInput{Kind: "brochure", Title: "x", Summary: "y"})
	assert.ErrorContains(t, err, "kind")
	_, err = h.svc.SaveMaterial(ctx, h.org, "", MaterialInput{Kind: "offer", Title: "x", Summary: strings.Repeat("a", 700)})
	assert.ErrorContains(t, err, "short version")
	cs, err := h.svc.SaveMaterial(ctx, h.org, "", MaterialInput{Kind: "case_study", Title: "Clinic Z doubled bookings", Summary: "Bookings up 2x in 90 days.", URL: "https://acme.com/z", Segment: "Utah dental"})
	require.NoError(t, err)
	assert.Equal(t, "Utah dental", cs.Segment)
	offer, err := h.svc.SaveMaterial(ctx, h.org, "", MaterialInput{Kind: "offer", Title: "Free audit", Summary: "A free 20-minute audit."})
	require.NoError(t, err)
	_, err = h.svc.SaveSegment(ctx, h.org, "", SegmentInput{Name: "Other"})
	require.NoError(t, err)
	_, err = h.svc.SaveMaterial(ctx, h.org, "", MaterialInput{Kind: "proof_point", Title: "Other only", Summary: "Not for dental.", Segment: "Other"})
	require.NoError(t, err)

	// A change keeps what it leaves out.
	offer, err = h.svc.SaveMaterial(ctx, h.org, offer.ID, MaterialInput{Summary: "A free 30-minute audit."})
	require.NoError(t, err)
	assert.Equal(t, "Free audit", offer.Title)

	_, err = h.svc.UpdateProfile(ctx, h.org, Profile{Website: "https://acme.com", Sells: "Booking software for clinics", Audience: "Small clinics"})
	require.NoError(t, err)
	p, err := h.svc.UpdateProfile(ctx, h.org, Profile{Proof: "300 clinics"})
	require.NoError(t, err)
	assert.Equal(t, "Booking software for clinics", p.Sells, "a profile change keeps what it leaves out")

	b, err := h.svc.Brief(ctx, h.org, "Utah dental", nil)
	require.NoError(t, err)
	assert.Equal(t, "300 clinics", b.Profile.Proof)
	assert.EqualValues(t, 1, b.Enrollable)
	assert.Contains(t, b.MergeTags, "company")
	assert.Contains(t, b.MergeTags, "first_name")
	titles := []string{}
	for _, m := range b.Materials {
		titles = append(titles, m.Title)
	}
	assert.ElementsMatch(t, []string{"Clinic Z doubled bookings", "Free audit"}, titles, "the segment's and the brand-wide ones, not another segment's")

	// Pinned materials stay with the campaign.
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "C", List: "Utah dental", Steps: businessSteps, ProspectIDs: []string{got["amy@clinic-a.com"].ID}, MaterialIDs: []string{cs.ID}})
	require.NoError(t, err)
	require.Len(t, camp.Materials, 1)
	r, err := h.svc.CampaignResults(ctx, mustCampaign(t, h, camp.CampaignID), true)
	require.NoError(t, err)
	require.Len(t, r.Materials, 1)
	assert.Equal(t, cs.ID, r.Materials[0].ID)
	_, err = h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "D", Steps: businessSteps, ProspectIDs: []string{got["amy@clinic-a.com"].ID}, MaterialIDs: []string{"nope"}})
	assert.ErrorContains(t, err, "No material")

	require.NoError(t, h.svc.DeleteMaterial(ctx, h.org, offer.ID))
	assert.Error(t, h.svc.DeleteMaterial(ctx, h.org, offer.ID))
}

// Nobody the brand already knows gets outreach: customers, their company
// domains (when asked), uploaded lists, the brand's own domains, and what an
// outside check says. Each skip says why.
func TestDoNotContact(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	h.setup()

	h.subscriber("cara@customer.com")
	_, err := h.svc.AddDoNotContact(ctx, h.org, "Open deals", []string{"dan@deal.com", "https://www.bigco.com/about", "not an address", ""})
	require.NoError(t, err)
	lists, err := h.svc.DoNotContactLists(ctx, h.org)
	require.NoError(t, err)
	require.Len(t, lists, 1)
	assert.EqualValues(t, 1, lists[0].Emails)
	assert.EqualValues(t, 1, lists[0].Domains)

	got := h.addTo("", business("cara@customer.com", "Cara", "Customer Co", "US"), business("dan@deal.com", "Dan", "Deal", "US"),
		business("eve@bigco.com", "Eve", "BigCo", "US"), business("sam2@getexample.com", "Sam", "Us", "US"),
		business("ned@neboai.com", "Ned", "Us", "US"), business("fin@customer.com", "Fin", "Customer Co", "US"),
		business("gil@gmail.com", "Gil", "Self", "US"))
	assert.Equal(t, "excluded", got["cara@customer.com"].Status)
	assert.Contains(t, got["cara@customer.com"].Reason, "(customer): already known to the brand: list Newsletter")
	assert.Contains(t, got["dan@deal.com"].Reason, `on the do-not-contact list "Open deals"`)
	assert.Contains(t, got["eve@bigco.com"].Reason, "domain bigco.com", "a website counts as its domain")
	assert.Contains(t, got["sam2@getexample.com"].Reason, "own_domain", "the brand's own outreach domain")
	assert.Contains(t, got["ned@neboai.com"].Reason, "own_domain", "a protected domain")
	assert.Equal(t, "prospect", got["fin@customer.com"].Status, "a customer's domain counts only when asked")

	// With customer_domains on, a customer's colleague is skipped at
	// enrollment; a free-mail domain never counts.
	on := true
	_, err = h.svc.UpdateDNCSettings(ctx, h.org, DNCSettingsInput{CustomerDomains: &on})
	require.NoError(t, err)
	h.refresh()
	h.subscriber("hal@gmail.com")
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "C", Steps: businessSteps, ProspectIDs: []string{got["fin@customer.com"].ID, got["gil@gmail.com"].ID}, DryRun: true})
	require.NoError(t, err)
	assert.Contains(t, camp.Skipped[got["fin@customer.com"].ID], "customer_domain")
	assert.NotContains(t, camp.Skipped, got["gil@gmail.com"].ID)

	ex, skip := h.svc.DoNotContact(ctx, h.org, "Cara@Customer.com")
	assert.True(t, skip)
	assert.Equal(t, SourceCustomer, ex.Source)
	why, err := h.svc.Explain(ctx, h.org, ExplainQuery{ProspectID: got["fin@customer.com"].ID})
	require.NoError(t, err)
	assert.Contains(t, why, "customer_domain")

	n, err := h.svc.DeleteDoNotContactList(ctx, h.org, "Open deals")
	require.NoError(t, err)
	assert.EqualValues(t, 2, n)
	_, skip = h.svc.DoNotContact(ctx, h.org, "dan@deal.com")
	assert.False(t, skip)
}

// The outside check: a signed POST answered yes or no, called before each
// email is queued. When it doesn't answer, nobody is enrolled, and a batch
// calls it once.
func TestOutsideCheck(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	h.setup()
	var calls atomic.Int32
	down := atomic.Bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		mac := hmac.New(sha256.New, []byte("shh"))
		mac.Write(body)
		if r.Header.Get("X-Outlet-Signature") != hex.EncodeToString(mac.Sum(nil)) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if down.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var req struct{ Email, Domain, Brand string }
		_ = json.Unmarshal(body, &req)
		deal := req.Email == "ida@acme.org"
		_ = json.NewEncoder(w).Encode(map[string]any{"do_not_contact": deal, "reason": "open deal in the CRM"})
	}))
	t.Cleanup(srv.Close)

	url := srv.URL + "/check"
	_, err := h.svc.UpdateDNCSettings(ctx, h.org, DNCSettingsInput{CheckURL: &url})
	assert.ErrorContains(t, err, "check_secret", "a new check needs its secret")
	st, err := h.svc.UpdateDNCSettings(ctx, h.org, DNCSettingsInput{CheckURL: &url, CheckSecret: "shh"})
	require.NoError(t, err)
	assert.True(t, st.CheckSecretSet)
	h.refresh()

	got := h.addTo("", business("ida@acme.org", "Ida", "Acme", "US"), business("jo@acme.org", "Jo", "Acme", "US"))
	assert.Equal(t, int32(0), calls.Load(), "import leaves the outside check to the queueing")
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "C", Steps: businessSteps, ProspectIDs: []string{got["ida@acme.org"].ID, got["jo@acme.org"].ID}, ApprovalMode: "first"})
	require.NoError(t, err)
	assert.Equal(t, 1, camp.Enrolled)
	assert.Equal(t, "do not contact (outside_check): open deal in the CRM", camp.Skipped[got["ida@acme.org"].ID])

	down.Store(true)
	calls.Store(0)
	got = h.addTo("", business("kim@acme.org", "Kim", "Acme", "US"), business("lu@acme.org", "Lu", "Acme", "US"))
	dry, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "D", Steps: businessSteps, ProspectIDs: []string{got["kim@acme.org"].ID, got["lu@acme.org"].ID}, DryRun: true})
	require.NoError(t, err)
	assert.Contains(t, dry.Skipped[got["kim@acme.org"].ID], "didn't answer")
	assert.Contains(t, dry.Skipped[got["lu@acme.org"].ID], "didn't answer")
	assert.Equal(t, int32(1), calls.Load(), "a check that is down is called once a batch")

	// Jo is in a campaign: while the check is down her follow-up waits, and
	// she is not stopped; once it answers, the follow-up goes.
	drafts, err := h.svc.ListDrafts(ctx, h.org, 10)
	require.NoError(t, err)
	var ids []string
	for _, d := range drafts {
		if d.CampaignID == camp.CampaignID {
			ids = append(ids, d.MessageID)
		}
	}
	require.Len(t, ids, 1)
	down.Store(false)
	_, err = h.svc.Review(ctx, h.org, "approve", ids)
	require.NoError(t, err)
	h.svc.Tick(ctx)
	require.Len(t, h.mail.sentTo("jo@acme.org"), 1)
	down.Store(true)
	h.advance(48*time.Hour + time.Minute)
	h.svc.Tick(ctx)
	h.svc.Tick(ctx)
	assert.Len(t, h.mail.sentTo("jo@acme.org"), 1, "held while the check is down")
	joID := h.contact("jo@acme.org").ID
	assert.Empty(t, h.state(joID, camp.CampaignID).StoppedReason, "a check that is down never stops anyone")
	down.Store(false)
	h.advance(20 * time.Minute)
	h.svc.Tick(ctx)
	assert.Len(t, h.mail.sentTo("jo@acme.org"), 2, "the follow-up goes once the check answers")

	off := ""
	st, err = h.svc.UpdateDNCSettings(ctx, h.org, DNCSettingsInput{CheckURL: &off})
	require.NoError(t, err)
	assert.False(t, st.CheckSecretSet)
}

// The check runs again before each follow-up: someone who became a customer
// after their first email gets no more, and results say why they stopped.
func TestDoNotContactStopsFollowUps(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	h.setup()
	got := h.addTo("", business("max@shop.com", "Max", "Shop", "US"))
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "C", Steps: businessSteps, ProspectIDs: []string{got["max@shop.com"].ID}, ApprovalMode: "first"})
	require.NoError(t, err)
	_, err = h.svc.Review(ctx, h.org, "approve", []string{camp.Drafts[0].MessageID})
	require.NoError(t, err)
	h.svc.Tick(ctx)
	require.Len(t, h.mail.sentTo("max@shop.com"), 1)

	_, err = h.svc.AddDoNotContact(ctx, h.org, "Customers", []string{"max@shop.com"})
	require.NoError(t, err)
	h.advance(48*time.Hour + time.Minute)
	h.svc.Tick(ctx)
	assert.Len(t, h.mail.sentTo("max@shop.com"), 1, "no follow-up")
	assert.Equal(t, "do not contact: do_not_contact_list", h.state(got["max@shop.com"].ID, camp.CampaignID).StoppedReason)
	r, err := h.svc.CampaignResults(ctx, mustCampaign(t, h, camp.CampaignID), false)
	require.NoError(t, err)
	assert.EqualValues(t, 1, r.StoppedByReason["do not contact: do_not_contact_list"])
}

// The pace dial: past the day's new prospects, first emails wait for the
// next day's window, the owner is told once, and follow-ups aren't counted.
func TestPaceDial(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	bus := events.NewSubject(events.WithBufferSize(16))
	h.store.SetEvents(bus)
	paused := make(chan events.OutreachPausedEvent, 8)
	events.Subscribe[events.OutreachPausedEvent](bus, events.TopicOutreachPaused, func(_ context.Context, e events.OutreachPausedEvent) error {
		paused <- e
		return nil
	})
	h.setup()

	p, err := h.svc.SetPace(ctx, h.org, "typical", nil)
	require.NoError(t, err)
	assert.EqualValues(t, 50, p.Current.DailyNewProspects)
	assert.Equal(t, "typical", p.Current.Pace)
	assert.EqualValues(t, 150, p.Current.SendsPerDay, "50 new a day, 3 steps each")
	assert.EqualValues(t, 5, p.Current.MailboxesNeeded, "at the pool's 30 a day per mailbox")
	require.Len(t, p.Presets, 4)
	_, err = h.svc.SetPace(ctx, h.org, "warp", nil)
	assert.ErrorContains(t, err, "light, typical, growth or unlimited")
	one := int64(1)
	p, err = h.svc.SetPace(ctx, h.org, "", &one)
	require.NoError(t, err)
	assert.Equal(t, "custom", p.Current.Pace)

	got := h.addTo("", business("nia@a.com", "Nia", "A", "US"), business("oz@b.com", "Oz", "B", "US"))
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "C", Steps: businessSteps, ProspectIDs: []string{got["nia@a.com"].ID, got["oz@b.com"].ID}})
	require.NoError(t, err)
	var ids []string
	for _, d := range camp.Drafts {
		ids = append(ids, d.MessageID)
	}
	_, err = h.svc.Review(ctx, h.org, "approve", ids)
	require.NoError(t, err)

	h.svc.Tick(ctx)
	assert.Equal(t, 1, h.mail.count(), "one new prospect a day")
	h.advance(10 * time.Minute)
	h.svc.Tick(ctx)
	assert.Equal(t, 1, h.mail.count(), "still one: the other waits for tomorrow")
	select {
	case e := <-paused:
		assert.Equal(t, "pace", e.Scope)
		assert.Equal(t, h.org.ID, e.OrgID)
		assert.NotEmpty(t, e.ResumesAt)
	case <-time.After(2 * time.Second):
		t.Fatal("the owner wasn't told")
	}
	select {
	case <-paused:
		t.Fatal("told twice in one day")
	case <-time.After(200 * time.Millisecond):
	}
	o, err := h.svc.Overview(ctx, h.org)
	require.NoError(t, err)
	assert.True(t, o.PaceReached)
	assert.Contains(t, strings.Join(o.Next, " "), "new prospects are reached")

	// The next business day the waiting first email goes.
	h.advance(24 * time.Hour)
	h.svc.Tick(ctx)
	assert.Equal(t, 2, h.mail.count())
}

// Hot leads are the interested replies; the leads export is one CSV row per
// person per campaign with its segment, status and classification.
func TestHotLeadsAndExport(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	h.setup()
	_, err := h.svc.SaveSegment(ctx, h.org, "", SegmentInput{Name: "Shops"})
	require.NoError(t, err)
	got := h.addTo("Shops", business("pia@shop.com", "Pia", "Pia's Shop", "US"), business("quin@shop2.com", "Quin", "Shop 2", "US"))
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "Spring", List: "Shops", Steps: businessSteps})
	require.NoError(t, err)
	var ids []string
	for _, d := range camp.Drafts {
		ids = append(ids, d.MessageID)
	}
	_, err = h.svc.Review(ctx, h.org, "approve", ids)
	require.NoError(t, err)
	h.svc.Tick(ctx)
	pia := h.mail.sentTo("pia@shop.com")
	require.Len(t, pia, 1)
	box := strings.Trim(strings.Split(pia[0].Header.Get("From"), "<")[1], ">")
	h.mail.deliver(t, box, "From: Pia <pia@shop.com>\nTo: "+box+"\nSubject: Re: hi\nMessage-ID: <r-pia@shop.com>\nIn-Reply-To: "+pia[0].Header.Get("Message-Id")+"\n\nYes, let's talk.")
	h.svc.PollReplies(ctx)
	all, err := h.svc.ListReplies(ctx, h.org, false, "", 10)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "Shops", all[0].SegmentName)
	require.NoError(t, h.svc.Classify(ctx, h.org, all[0].ID, "interested"))
	hot, err := h.svc.ListReplies(ctx, h.org, false, "interested", 10)
	require.NoError(t, err)
	assert.Len(t, hot, 1)
	none, err := h.svc.ListReplies(ctx, h.org, false, "question", 10)
	require.NoError(t, err)
	assert.Empty(t, none)
	_, err = h.svc.ListReplies(ctx, h.org, false, "hot", 10)
	assert.ErrorContains(t, err, "classification is")

	var buf strings.Builder
	rows, next, err := h.svc.WriteLeads(ctx, h.org, &buf, LeadQuery{})
	require.NoError(t, err)
	assert.Equal(t, 2, rows)
	assert.Empty(t, next)
	recs, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err)
	assert.Equal(t, LeadColumns, recs[0])
	byEmail := map[string][]string{}
	for _, r := range recs[1:] {
		byEmail[r[0]] = r
	}
	assert.Equal(t, []string{"pia@shop.com", "Pia", "Pia's Shop", "Office Manager", "Shops", "Spring", "replied", "replied", "interested"}, byEmail["pia@shop.com"][:9])
	assert.Equal(t, "in_progress", byEmail["quin@shop2.com"][6])

	buf.Reset()
	rows, _, err = h.svc.WriteLeads(ctx, h.org, &buf, LeadQuery{Classification: "interested"})
	require.NoError(t, err)
	assert.Equal(t, 1, rows, "the hot leads only")
	buf.Reset()
	rows, next, err = h.svc.WriteLeads(ctx, h.org, &buf, LeadQuery{Limit: 1})
	require.NoError(t, err)
	require.Equal(t, 1, rows)
	require.NotEmpty(t, next)
	buf.Reset()
	rows, next, err = h.svc.WriteLeads(ctx, h.org, &buf, LeadQuery{Limit: 1, After: next})
	require.NoError(t, err)
	assert.Equal(t, 1, rows)
	_ = got
}

// Every outreach email says who wrote it: an AI's campaign is ai, a person's
// edit of it ai_edited, a person's own campaign person.
func TestAuthorship(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	h.setup()
	got := h.addTo("", business("ray@a.com", "Ray", "A", "US"), business("sue@b.com", "Sue", "B", "US"))

	ai, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "AI", Steps: businessSteps, ProspectIDs: []string{got["ray@a.com"].ID}, AuthoredBy: AuthorAI})
	require.NoError(t, err)
	require.Len(t, ai.Drafts, 1)
	assert.Equal(t, AuthorAI, ai.Drafts[0].AuthoredBy)
	_, err = h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "X", Steps: businessSteps, ProspectIDs: []string{got["sue@b.com"].ID}, AuthoredBy: "robot"})
	assert.ErrorContains(t, err, "authored_by")
	person, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "Me", Steps: businessSteps, ProspectIDs: []string{got["sue@b.com"].ID}})
	require.NoError(t, err)
	assert.Equal(t, AuthorPerson, person.Drafts[0].AuthoredBy)

	id := ai.Drafts[0].MessageID
	require.NoError(t, h.svc.EditDraft(ctx, h.org, id, "", "", AuthorPerson))
	author := func(id string) string {
		m, err := h.store.GetOutreachEmail(ctx, db.GetOutreachEmailParams{ID: id, OrgID: sql.NullString{String: h.org.ID, Valid: true}})
		require.NoError(t, err)
		return m.AuthoredBy
	}
	assert.Equal(t, AuthorAI, author(id), "saving without a change changes nothing")
	require.NoError(t, h.svc.EditDraft(ctx, h.org, id, "A better subject", "", AuthorPerson))
	assert.Equal(t, AuthorEdited, author(id))
	require.NoError(t, h.svc.EditDraft(ctx, h.org, person.Drafts[0].MessageID, "", "Rewritten by the AI.", AuthorAI))
	assert.Equal(t, AuthorAI, author(person.Drafts[0].MessageID), "an AI's edit is the AI's")

	drafts, err := h.svc.ListDrafts(ctx, h.org, 10)
	require.NoError(t, err)
	for _, d := range drafts {
		assert.NotEmpty(t, d.AuthoredBy)
	}
	r, err := h.svc.CampaignResults(ctx, mustCampaign(t, h, ai.CampaignID), false)
	require.NoError(t, err)
	assert.Equal(t, AuthorAI, r.AuthoredBy)
}

// Starting from a website: the guide gives the calls in order and says what
// the brand still lacks.
func TestStartFromWebsite(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	guide, err := h.svc.Explain(ctx, h.org, ExplainQuery{Topic: "start_from_website"})
	require.NoError(t, err)
	for _, call := range []string{"targeting(resource: profile, action: update", "targeting(resource: segment, action: create", "draft_outreach(action: brief"} {
		assert.Contains(t, guide, call)
	}
	assert.Contains(t, guide, "a brand profile")
	assert.Contains(t, guide, "a mailing address")

	h.setup()
	_, err = h.svc.UpdateProfile(ctx, h.org, Profile{Website: "https://acme.com"})
	require.NoError(t, err)
	guide, err = h.svc.Explain(ctx, h.org, ExplainQuery{Topic: "start_from_website"})
	require.NoError(t, err)
	assert.Contains(t, guide, "already has a profile (https://acme.com)")
	assert.NotContains(t, guide, "Still missing")
	_, err = h.svc.Explain(ctx, h.org, ExplainQuery{Topic: "nope"})
	assert.ErrorContains(t, err, "Topics")
}

// The overview counts each person on the brand's outreach lists once, the
// number the Prospects tab shows, while prospects counts their list places.
func TestOverviewProspectsTotal(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx

	o, err := h.svc.Overview(ctx, h.org)
	require.NoError(t, err)
	assert.Zero(t, o.ProspectsTotal)

	_, err = h.svc.SaveSegment(ctx, h.org, "", SegmentInput{Name: "Utah dental", Description: "Dental clinics in Utah."})
	require.NoError(t, err)
	h.prospects(business("amy@clinic-a.com", "Amy", "Clinic A", "US"), business("bo@clinic-b.com", "Bo", "Clinic B", "US"))
	h.addTo("Utah dental", business("amy@clinic-a.com", "Amy", "Clinic A", "US"))

	o, err = h.svc.Overview(ctx, h.org)
	require.NoError(t, err)
	assert.EqualValues(t, 2, o.ProspectsTotal, "Amy is on two lists and counts once")
	var places int64
	for _, n := range o.Prospects {
		places += n
	}
	assert.EqualValues(t, 3, places, "her two list places")
	page, err := h.svc.ListProspects(ctx, h.org, ProspectQuery{})
	require.NoError(t, err)
	assert.Equal(t, page.Total, o.ProspectsTotal, "the Prospects tab's total")
}
