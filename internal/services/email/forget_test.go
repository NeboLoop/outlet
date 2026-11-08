package email

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/outlet-sh/outlet/internal/services/crypto"
)

// Forgetting an address asks each SES account once to take it off its
// suppression list, the one per-recipient record SES's API deletes, and says
// plainly that an SMTP relay and mailboxes have no such API.
func TestForgetRecipient(t *testing.T) {
	var mu sync.Mutex
	var deletes []string
	ses := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deletes = append(deletes, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(ses.Close)
	t.Setenv("AWS_ENDPOINT_URL", ses.URL)
	// Nothing from this machine's own AWS setup
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_CONFIG_FILE", t.TempDir()+"/none")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", t.TempDir()+"/none")

	_, store := newTestService(t)
	cs, err := crypto.NewService(strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(store, cs)
	ctx := context.Background()
	for _, in := range []PathwayInput{
		{Provider: ProviderSES, Name: "SES one", AWSAccessKey: "AKIA1", AWSSecretKey: "secret"},
		{Provider: ProviderSES, Name: "SES again", AWSAccessKey: "AKIA1", AWSSecretKey: "secret"},
		{Provider: ProviderSMTP, Name: "Relay", SMTPHost: "smtp.example", SMTPUsername: "u", SMTPPassword: "p"},
		{Provider: ProviderMailboxes, Name: "Inboxes"},
	} {
		if _, err := s.CreatePathway(ctx, "org-a", in); err != nil {
			t.Fatal(in.Name, err)
		}
	}

	got, err := s.ForgetRecipient(ctx, "org-a", "gone@x.example")
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]ProviderErasure{}
	for _, r := range got {
		byName[r.Pathway] = r
	}
	if len(deletes) != 1 || deletes[0] != "DELETE /v2/email/suppression/addresses/gone@x.example" {
		t.Errorf("SES calls: %v", deletes)
	}
	// The two SES pathways share one account: one asks it, the other says so
	removed, same := 0, 0
	for _, name := range []string{"SES one", "SES again"} {
		r := byName[name]
		switch {
		case !r.Failed && strings.HasPrefix(r.Result, "Removed the address from the Amazon SES account's suppression list"):
			removed++
		case strings.HasPrefix(r.Result, "The same Amazon SES account as "):
			same++
		default:
			t.Errorf("%s: %+v", name, r)
		}
	}
	if removed != 1 || same != 1 {
		t.Errorf("removed %d, same account %d", removed, same)
	}
	if r := byName["Relay"]; r.Result != smtpNoAPI {
		t.Errorf("Relay: %+v", r)
	}
	if r := byName["Inboxes"]; r.Result != mailboxesNoAPI {
		t.Errorf("Inboxes: %+v", r)
	}
	// The brand's platform pathway, with no platform SES keys, sends through
	// the platform's SMTP server
	if r := byName["Platform email"]; r.Provider != "SMTP relay" || r.Result != smtpNoAPI {
		t.Errorf("Platform email: %+v (all: %+v)", r, got)
	}
}

// An address SES doesn't have on its suppression list isn't a failure.
func TestDeleteSESSuppression_NotOnTheList(t *testing.T) {
	ses := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Amzn-ErrorType", "NotFoundException")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Email address not found"}`))
	}))
	t.Cleanup(ses.Close)
	t.Setenv("AWS_ENDPOINT_URL", ses.URL)
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_CONFIG_FILE", t.TempDir()+"/none")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", t.TempDir()+"/none")

	result, failed := deleteSESSuppression(context.Background(), &SESConfig{Region: "us-east-1", AccessKey: "a", SecretKey: "s"}, "never@x.example")
	if failed || !strings.HasPrefix(result, "The address wasn't on the Amazon SES account's suppression list.") {
		t.Errorf("got %q, failed %v", result, failed)
	}
}
