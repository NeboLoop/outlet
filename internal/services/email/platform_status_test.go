package email

import (
	"context"
	"reflect"
	"testing"
)

func TestPlatformStatus(t *testing.T) {
	s, store := newTestService(t)
	ctx := context.Background()
	set := func(key, value string) {
		t.Helper()
		if _, err := store.GetDB().Exec(`INSERT INTO platform_settings (key, value_text, category) VALUES (?, ?, 'email')
			ON CONFLICT (key) DO UPDATE SET value_text = excluded.value_text`, key, value); err != nil {
			t.Fatal(err)
		}
	}

	status, err := s.PlatformStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"from_email", "aws_access_key", "aws_secret_key", "smtp_host", "smtp_password", "smtp_port", "smtp_user"}
	if status.HasSES || status.HasSMTP || status.HasFrom || status.Configured() || !reflect.DeepEqual(status.Missing, want) {
		t.Errorf("nothing set: %+v", status)
	}

	// An SMTP relay without a From address: mail can go out, platform mail
	// has no sender yet.
	for key, value := range map[string]string{"smtp_host": "smtp.example", "smtp_port": "587", "smtp_user": "u", "smtp_password": "p"} {
		set(key, value)
	}
	status, err = s.PlatformStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !status.HasSMTP || status.HasFrom || status.Configured() || !reflect.DeepEqual(status.Missing, []string{"from_email"}) {
		t.Errorf("SMTP without From: %+v", status)
	}

	set("from_email", "outlet@example.com")
	status, err = s.PlatformStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !status.HasSMTP || !status.HasFrom || !status.Configured() || len(status.Missing) != 0 {
		t.Errorf("SMTP with From: %+v", status)
	}
}
