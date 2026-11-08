package email

import (
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"testing"

	gosmtp "github.com/emersion/go-smtp"
)

// apiError is an AWS API error as the SDK returns one: it has a code.
type apiError struct{ code string }

func (e apiError) Error() string     { return e.code }
func (e apiError) ErrorCode() string { return e.code }

// httpError is an AWS response error: it has an HTTP status.
type httpError struct{ status int }

func (e httpError) Error() string       { return fmt.Sprint(e.status) }
func (e httpError) HTTPStatusCode() int { return e.status }

// Every SMTP and SES failure is told apart: the temporary ones go back to
// the queue to be tried again, the permanent ones are failed.
func TestDeliveryFailuresAreClassified(t *testing.T) {
	for _, c := range []struct {
		name      string
		err       error
		temporary bool
	}{
		{"go-smtp 451", smtpFailure(&gosmtp.SMTPError{Code: 451, Message: "later"}), true},
		{"go-smtp 550", smtpFailure(&gosmtp.SMTPError{Code: 550, Message: "no such user"}), false},
		{"net/smtp 421", smtpFailure(fmt.Errorf("send via SMTP: %w", &textproto.Error{Code: 421, Msg: "busy"})), true},
		{"net/smtp 554", smtpFailure(fmt.Errorf("send via SMTP: %w", &textproto.Error{Code: 554, Msg: "rejected"})), false},
		{"no connection", smtpFailure(&net.OpError{Op: "dial", Err: errors.New("connection refused")}), true},
		{"SES throttling", sesFailure(fmt.Errorf("SES SendRawEmail failed: %w", apiError{"Throttling"})), true},
		{"SES 503", sesFailure(httpError{503}), true},
		{"SES timeout", sesFailure(apiError{"RequestTimeout"}), true},
		{"SES rejected", sesFailure(apiError{"MessageRejected"}), false},
		{"SES unverified sender", sesFailure(apiError{"MailFromDomainNotVerifiedException"}), false},
	} {
		if got := IsTemporary(c.err); got != c.temporary {
			t.Errorf("%s: temporary = %v, want %v", c.name, got, c.temporary)
		}
	}
	if smtpFailure(nil) != nil {
		t.Error("no error is no failure")
	}
}
