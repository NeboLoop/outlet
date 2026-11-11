package outbound

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// done maps each setup step to whether it is done.
func done(steps []SetupStep) map[string]bool {
	out := map[string]bool{}
	for _, s := range steps {
		out[s.Step] = s.Done
	}
	return out
}

// The steps come in order, each says what it still needs, and a domain that
// fails its check is named apart from having no domain.
func TestSetupStepsFromFacts(t *testing.T) {
	steps := setupSteps(setupFacts{})
	require.Len(t, steps, 5)
	assert.Equal(t, []string{SetupBusiness, SetupTargeting, SetupSending, SetupDoNotContact, SetupCampaign},
		[]string{steps[0].Step, steps[1].Step, steps[2].Step, steps[3].Step, steps[4].Step})
	for _, s := range steps {
		assert.False(t, s.Done, s.Step)
		assert.NotEmpty(t, s.Needs, s.Step)
	}
	assert.Equal(t, "outbound turned on; a mailing address; an outreach domain; an active mailbox", steps[2].Needs)

	steps = setupSteps(setupFacts{enabled: true, mailingAddress: true, anyDomain: true, activeMailbox: true})
	assert.False(t, steps[2].Done)
	assert.Equal(t, "a domain that passes its DNS check", steps[2].Needs)

	steps = setupSteps(setupFacts{profile: Profile{Website: "https://acme.com"}})
	assert.False(t, steps[0].Done, "a website alone doesn't say what the business sells or to whom")

	steps = setupSteps(setupFacts{outsideCheck: true})
	assert.True(t, steps[3].Done, "an outside check is a do-not-contact source")
	assert.Empty(t, steps[3].Needs)
}

// Each step is done exactly while its data exists: making the data marks it
// done, and taking the data away marks it not done again.
func TestSetupFromData(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx

	steps, err := h.svc.Setup(ctx, h.org)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{SetupBusiness: false, SetupTargeting: false, SetupSending: false, SetupDoNotContact: false, SetupCampaign: false}, done(steps))

	_, err = h.svc.UpdateProfile(ctx, h.org, Profile{Website: "https://acme.com"})
	require.NoError(t, err)
	steps, _ = h.svc.Setup(ctx, h.org)
	assert.False(t, done(steps)[SetupBusiness])
	_, err = h.svc.UpdateProfile(ctx, h.org, Profile{Sells: "Booking software", Audience: "Small clinics"})
	require.NoError(t, err)
	steps, _ = h.svc.Setup(ctx, h.org)
	assert.True(t, done(steps)[SetupBusiness])

	// A list made by importing prospects says nothing about who it targets.
	h.prospects(business("amy@clinic-a.com", "Amy", "Clinic A", "US"))
	steps, _ = h.svc.Setup(ctx, h.org)
	assert.False(t, done(steps)[SetupTargeting])
	_, err = h.svc.SaveSegment(ctx, h.org, "", SegmentInput{Name: "Utah dental", Description: "Dental clinics in Utah."})
	require.NoError(t, err)
	steps, _ = h.svc.Setup(ctx, h.org)
	assert.True(t, done(steps)[SetupTargeting])

	sam, _ := h.setup()
	steps, _ = h.svc.Setup(ctx, h.org)
	assert.True(t, done(steps)[SetupSending])
	require.NoError(t, h.svc.Pause(ctx, h.org, "mailbox", sam.ID, false))
	steps, _ = h.svc.Setup(ctx, h.org)
	assert.True(t, done(steps)[SetupSending], "one active mailbox is enough")
	require.NoError(t, h.svc.SetEnabled(ctx, h.org.ID, false))
	h.refresh()
	steps, _ = h.svc.Setup(ctx, h.org)
	assert.False(t, done(steps)[SetupSending], "outbound turned off")
	require.NoError(t, h.svc.SetEnabled(ctx, h.org.ID, true))
	h.refresh()

	_, err = h.svc.AddDoNotContact(ctx, h.org, "Customers", []string{"ann@client.com"})
	require.NoError(t, err)
	steps, _ = h.svc.Setup(ctx, h.org)
	assert.True(t, done(steps)[SetupDoNotContact])
	_, err = h.svc.DeleteDoNotContactList(ctx, h.org, "Customers")
	require.NoError(t, err)
	steps, _ = h.svc.Setup(ctx, h.org)
	assert.False(t, done(steps)[SetupDoNotContact], "the list is gone, so the step is open again")
	_, err = h.svc.AddDoNotContact(ctx, h.org, "Customers", []string{"client.com"})
	require.NoError(t, err)

	_, err = h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "First", List: DefaultOutreachList, Steps: businessSteps})
	require.NoError(t, err)
	o, err := h.svc.Overview(ctx, h.org)
	require.NoError(t, err)
	for _, s := range o.Setup {
		assert.True(t, s.Done, s.Step)
		assert.Empty(t, s.Needs, s.Step)
	}
}

// A segment can be removed while nothing depends on it; one with people,
// a campaign or materials of its own is refused.
func TestDeleteSegment(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx

	_, err := h.svc.SaveSegment(ctx, h.org, "", SegmentInput{Name: "Spare", Description: "Nobody yet."})
	require.NoError(t, err)
	require.NoError(t, h.svc.DeleteSegment(ctx, h.org, "Spare"))
	segs, err := h.svc.Segments(ctx, h.org)
	require.NoError(t, err)
	assert.Empty(t, segs)
	assert.Error(t, h.svc.DeleteSegment(ctx, h.org, "Spare"), "it is gone")

	_, err = h.svc.SaveSegment(ctx, h.org, "", SegmentInput{Name: "Cited", Description: "Has a case study."})
	require.NoError(t, err)
	_, err = h.svc.SaveMaterial(ctx, h.org, "", MaterialInput{Kind: "case_study", Title: "Clinic Z", Summary: "Bookings doubled.", Segment: "Cited"})
	require.NoError(t, err)
	assert.ErrorContains(t, h.svc.DeleteSegment(ctx, h.org, "Cited"), "materials of its own")

	h.addTo("Peopled", business("amy@clinic-a.com", "Amy", "Clinic A", "US"))
	assert.ErrorContains(t, h.svc.DeleteSegment(ctx, h.org, "Peopled"), "has people in it")
	segs, err = h.svc.Segments(ctx, h.org)
	require.NoError(t, err)
	assert.Len(t, segs, 2, "refused segments stay")
}
