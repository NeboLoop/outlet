package public

import (
	"html"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A campaign's web view is the email its recipient got: placed in the
// campaign's design with its merge tags filled, by CampaignMessage.
func TestWebView_ShowsTheCampaignInItsDesign(t *testing.T) {
	h, store := newSubscribeFixture(t)
	for _, q := range []string{
		`INSERT INTO email_designs (id, org_id, name, slug, html_body) VALUES (7, 'org1', 'D', 'd', '<html><head><style>.brand{color:red}</style></head><body><div class="brand">ACME</div>{{content}}</body></html>')`,
		`INSERT INTO email_campaigns (id, org_id, design_id, name, subject, html_body) VALUES ('camp', 'org1', 7, 'C', 'News for {{name}}', '<p>Hello {{name}}</p>')`,
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('ana', 'org1', 'Ana', 'ana@example.com')`,
		`INSERT INTO campaign_sends (id, campaign_id, contact_id, tracking_token) VALUES ('cs', 'camp', 'ana', 'tok')`,
	} {
		_, err := store.GetDB().Exec(q)
		require.NoError(t, err, q)
	}

	rec := httptest.NewRecorder()
	h.HandleWebView(rec, httptest.NewRequest(http.MethodGet, "/w/tok", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	page := html.UnescapeString(rec.Body.String())
	assert.Contains(t, page, `<div class="brand">ACME</div><p>Hello Ana</p>`)
	assert.Contains(t, page, "News for Ana")
	assert.Contains(t, page, "/u/tok", "the unsubscribe link the email carries")
}
