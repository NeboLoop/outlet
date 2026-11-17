package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	emailsvc "github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// SignInTTL is how long an emailed sign-in link and code work.
	SignInTTL = 15 * time.Minute
	// signInCodeTries is how many codes may be checked against one email;
	// the last wrong one retires it.
	signInCodeTries = 5
)

// SignInEmailSent is the answer to a request for a sign-in email. It is the
// same whether or not the address has an account, so the form can't be used
// to find out who has one.
const SignInEmailSent = "If that email has an account, we sent it a sign-in link and code"

var (
	errSignInEmailInput = errorx.NewBadRequestError("Open the link in the email, or enter your email and the code from it.")
	errSignInEmailBad   = errorx.NewUnauthorizedError("That sign-in link or code is wrong or has expired. Send yourself a new one.")
)

// signInHash is the keyed hash a sign-in link token or code is kept as: what
// is stored can't be turned back into a working link or code, nor checked
// against guesses without the server's secret.
func signInHash(secret, value string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(value))
	return hex.EncodeToString(m.Sum(nil))
}

// IssueSignIn gives a person a new emailed sign-in: a link token and a
// 6-digit code, either good once within 15 minutes. It replaces any earlier
// one, so only the newest email works.
func IssueSignIn(ctx context.Context, q db.Querier, secret, userID string) (token, code string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(b)
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", "", err
	}
	code = fmt.Sprintf("%06d", n.Int64())
	if err := q.ReplaceSignInCode(ctx, db.ReplaceSignInCodeParams{
		ID:        generateUUID(),
		UserID:    userID,
		TokenHash: signInHash(secret, "link:"+token),
		CodeHash:  signInHash(secret, "code:"+userID+":"+code),
		ExpiresAt: time.Now().UTC().Add(SignInTTL).Format(time.RFC3339),
	}); err != nil {
		return "", "", fmt.Errorf("store sign-in code: %w", err)
	}
	return token, code, nil
}

// RedeemSignIn uses an emailed sign-in, by its link token or by the email
// address and its code, and returns whose it was. Using it retires both the
// link and the code. Every refusal is the same, whether the address has no
// account, no sign-in was sent, it expired, or the code was wrong.
func RedeemSignIn(ctx context.Context, q db.Querier, secret, token, email, code string) (string, error) {
	token = strings.TrimSpace(token)
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)

	var row db.SignInCode
	switch {
	case token != "":
		r, err := q.GetSignInCodeByTokenHash(ctx, signInHash(secret, "link:"+token))
		if errors.Is(err, sql.ErrNoRows) {
			return "", errSignInEmailBad
		}
		if err != nil {
			return "", err
		}
		row = r
	case email != "" && code != "":
		user, err := q.GetUserByEmail(ctx, email)
		if errors.Is(err, sql.ErrNoRows) {
			return "", errSignInEmailBad
		}
		if err != nil {
			return "", err
		}
		// The try is counted before the code is checked, so no number of
		// guesses at once gets more than signInCodeTries checks.
		r, err := q.CountSignInCodeAttempt(ctx, user.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return "", errSignInEmailBad
		}
		if err != nil {
			return "", err
		}
		if r.Attempts > signInCodeTries {
			_, _ = q.DeleteSignInCode(ctx, r.ID)
			return "", errSignInEmailBad
		}
		if !hmac.Equal([]byte(r.CodeHash), []byte(signInHash(secret, "code:"+user.ID+":"+code))) {
			if r.Attempts == signInCodeTries {
				if _, err := q.DeleteSignInCode(ctx, r.ID); err != nil {
					return "", err
				}
			}
			return "", errSignInEmailBad
		}
		row = r
	default:
		return "", errSignInEmailInput
	}

	// Removing the row is what uses it: of two uses at once, one removes it
	// and the other is refused.
	n, err := q.DeleteSignInCode(ctx, row.ID)
	if err != nil {
		return "", err
	}
	if n != 1 {
		return "", errSignInEmailBad
	}
	return row.UserID, nil
}

// SendSignIn emails a sign-in link and code to the address, if it belongs to
// an active account, and otherwise does nothing; the caller answers
// SignInEmailSent either way. link gives the page the emailed link opens for
// a link token: the app's sign-in page, or the MCP sign-in that started it.
func SendSignIn(ctx context.Context, svcCtx *svc.ServiceContext, email string, link func(token string) string) error {
	log := logx.WithContext(ctx)
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return fmt.Errorf("email is required")
	}

	u, err := svcCtx.DB.GetUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		log.Infof("Sign-in email requested for an address with no account")
		return nil
	}
	if err != nil {
		log.Errorf("Database error checking email: %v", err)
		return nil
	}
	// Only an active account can sign in, so only it gets the email.
	user, err := access.Active(ctx, svcCtx.DB, u.ID)
	if err != nil {
		log.Infof("Sign-in email requested for an account that can't sign in: %s", u.ID)
		return nil
	}

	token, code, err := IssueSignIn(ctx, svcCtx.DB, svcCtx.Config.Auth.AccessSecret, user.ID)
	if err != nil {
		log.Errorf("Error issuing sign-in code: %v", err)
		return nil
	}

	// Sent asynchronously, the way the reset email is, so the answer takes as
	// long whether or not the address has an account.
	msg := signInMessage(link(token), user.Email, user.Name, code)
	go func() {
		if err := svcCtx.EmailService.Send(context.Background(), msg); err != nil {
			log.Errorf("Error sending sign-in email: %v", err)
		} else {
			log.Infof("Sign-in email sent to: %s", user.Email)
		}
	}()
	return nil
}

// SignInByEmail signs a person in with an emailed sign-in, by its link token
// or by the email and its code (RedeemSignIn), and returns them if they are
// still active. The emailed sign-in proves the inbox, as a reset link does:
// it records the sign-in and clears any lock from wrong passwords.
func SignInByEmail(ctx context.Context, svcCtx *svc.ServiceContext, token, email, code string) (db.User, error) {
	userID, err := RedeemSignIn(ctx, svcCtx.DB, svcCtx.Config.Auth.AccessSecret, token, email, code)
	if err != nil {
		return db.User{}, err
	}
	user, err := access.Active(ctx, svcCtx.DB, userID)
	if err != nil {
		return db.User{}, err
	}
	if err := svcCtx.DB.UpdateLastLogin(ctx, user.ID); err != nil {
		logx.WithContext(ctx).Errorf("Could not record the sign-in for %s: %v", user.ID, err)
	}
	return user, nil
}

// SignInURL is the app's page that finishes an emailed sign-in.
func SignInURL(baseURL, token string) string {
	return fmt.Sprintf("%s/auth/sign-in?token=%s", strings.TrimSuffix(baseURL, "/"), url.QueryEscape(token))
}

// signInMessage is the email with a person's sign-in link and code. Like the
// reset email it is platform mail: Send gives it the platform's sender.
func signInMessage(link, to, name, code string) emailsvc.Message {
	body := fmt.Sprintf(`
<html>
<body>
<h2>Sign in to Outlet</h2>
<p>Hi %s,</p>
<p>Click the link below to sign in:</p>
<p><a href="%s">Sign in to Outlet</a></p>
<p>Or enter this code on the sign-in page:</p>
<p style="font-size:24px;font-weight:bold;letter-spacing:4px;">%s</p>
<p>The link and the code work once and expire in 15 minutes. Using one retires the other.</p>
<p>If you didn't ask to sign in, you can safely ignore this email.</p>
<br>
<p>Best regards,<br>The Outlet Team</p>
</body>
</html>`, html.EscapeString(name), html.EscapeString(link), code)
	return emailsvc.Message{To: to, Subject: "Your Outlet sign-in code: " + code, HTML: body}
}
