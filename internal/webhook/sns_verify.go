package webhook

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// snsHostPattern matches the hosts Amazon SNS signs from and sends subscription
// links on, e.g. sns.us-east-1.amazonaws.com.
var snsHostPattern = regexp.MustCompile(`^sns\.[a-z0-9-]+\.amazonaws\.com(\.cn)?$`)

// snsVerifier checks that an SNS message really came from Amazon SNS, following
// https://docs.aws.amazon.com/sns/latest/dg/sns-verify-signature-of-message.html.
// Without it anyone could post fake bounces or complaints, or make the server
// fetch an arbitrary SubscribeURL.
type snsVerifier struct {
	urlOK     func(raw string) bool
	fetchCert func(ctx context.Context, certURL string) ([]byte, error)

	mu    sync.Mutex
	certs map[string]*x509.Certificate
}

var defaultSNSVerifier = &snsVerifier{
	urlOK:     isSNSURL,
	fetchCert: fetchURL,
	certs:     map[string]*x509.Certificate{},
}

// isSNSURL reports whether raw is an https URL on an Amazon SNS host.
func isSNSURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && snsHostPattern.MatchString(u.Hostname())
}

func fetchURL(ctx context.Context, certURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, certURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch signing certificate: status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 64<<10))
}

// verify returns nil only when m carries a valid SNS signature.
func (v *snsVerifier) verify(ctx context.Context, m *snsMessage) error {
	if !v.urlOK(m.SigningCertURL) || !strings.HasSuffix(m.SigningCertURL, ".pem") {
		return errors.New("signing certificate is not on an Amazon SNS host")
	}

	var hash crypto.Hash
	switch m.SignatureVersion {
	case "1":
		hash = crypto.SHA1
	case "2":
		hash = crypto.SHA256
	default:
		return fmt.Errorf("unsupported signature version %q", m.SignatureVersion)
	}

	sig, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	cert, err := v.cert(ctx, m.SigningCertURL)
	if err != nil {
		return err
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("signing certificate does not hold an RSA key")
	}

	msg := []byte(snsStringToSign(m))
	var digest []byte
	if hash == crypto.SHA1 {
		d := sha1.Sum(msg)
		digest = d[:]
	} else {
		d := sha256.Sum256(msg)
		digest = d[:]
	}
	if err := rsa.VerifyPKCS1v15(pub, hash, digest, sig); err != nil {
		return errors.New("signature does not match")
	}
	return nil
}

// cert returns the parsed signing certificate, fetching it once per URL.
func (v *snsVerifier) cert(ctx context.Context, certURL string) (*x509.Certificate, error) {
	v.mu.Lock()
	cached := v.certs[certURL]
	v.mu.Unlock()
	if cached != nil {
		return cached, nil
	}

	raw, err := v.fetchCert(ctx, certURL)
	if err != nil {
		return nil, fmt.Errorf("fetch signing certificate: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("signing certificate is not PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse signing certificate: %w", err)
	}

	v.mu.Lock()
	v.certs[certURL] = cert
	v.mu.Unlock()
	return cert, nil
}

// snsStringToSign builds the canonical string SNS signs: selected fields as
// "Name\nValue\n" pairs in byte order. Subscription messages sign
// SubscribeURL and Token; notifications sign Subject when present.
func snsStringToSign(m *snsMessage) string {
	var b strings.Builder
	add := func(name, value string) {
		b.WriteString(name)
		b.WriteByte('\n')
		b.WriteString(value)
		b.WriteByte('\n')
	}
	add("Message", m.Message)
	add("MessageId", m.MessageId)
	if m.Type == "SubscriptionConfirmation" || m.Type == "UnsubscribeConfirmation" {
		add("SubscribeURL", m.SubscribeURL)
		add("Timestamp", m.Timestamp)
		add("Token", m.Token)
	} else {
		if m.Subject != "" {
			add("Subject", m.Subject)
		}
		add("Timestamp", m.Timestamp)
	}
	add("TopicArn", m.TopicArn)
	add("Type", m.Type)
	return b.String()
}
