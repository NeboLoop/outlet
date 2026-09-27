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

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
)

// SNS delivers SES bounce, complaint and delivery notifications to a public
// URL, so a message is trusted only once its signature verifies against the
// signing certificate SNS publishes, and that certificate is fetched only
// from an SNS host. Subscription confirmations are followed only to an SNS
// host too; anything else would let a stranger make this server fetch any
// URL. See https://docs.aws.amazon.com/sns/latest/dg/sns-verify-signature-of-message.html

// snsHost matches the hosts SNS serves signing certificates and subscription
// confirmations from: sns.<region>.amazonaws.com (and .amazonaws.com.cn).
var snsHost = regexp.MustCompile(`^sns\.[a-z0-9-]+\.amazonaws\.com(\.cn)?$`)

// isSNSURL reports whether raw is an https URL on an SNS host.
func isSNSURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	return snsHost.MatchString(strings.ToLower(u.Hostname()))
}

// snsStringToSign is the canonical text SNS signs: the message's fields in
// this order, each as "Name\nValue\n", Subject only when present.
func snsStringToSign(m *snsMessage) (string, error) {
	type kv struct{ k, v string }
	var fields []kv
	switch m.Type {
	case "Notification":
		fields = []kv{{"Message", m.Message}, {"MessageId", m.MessageId}}
		if m.Subject != "" {
			fields = append(fields, kv{"Subject", m.Subject})
		}
		fields = append(fields, kv{"Timestamp", m.Timestamp}, kv{"TopicArn", m.TopicArn}, kv{"Type", m.Type})
	case "SubscriptionConfirmation", "UnsubscribeConfirmation":
		fields = []kv{
			{"Message", m.Message}, {"MessageId", m.MessageId}, {"SubscribeURL", m.SubscribeURL},
			{"Timestamp", m.Timestamp}, {"Token", m.Token}, {"TopicArn", m.TopicArn}, {"Type", m.Type},
		}
	default:
		return "", fmt.Errorf("unknown SNS message type %q", m.Type)
	}
	var b strings.Builder
	for _, f := range fields {
		b.WriteString(f.k)
		b.WriteByte('\n')
		b.WriteString(f.v)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// snsCertFetcher returns the signing certificate at an SNS URL. A variable
// so tests can sign with their own key; the URL is checked before it runs.
var snsCertFetcher = fetchSNSCert

var (
	snsCertMu    sync.Mutex
	snsCertCache = map[string]*x509.Certificate{}
	// Never follow a redirect: it could leave the SNS host.
	snsClient = &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
)

// fetchSNSCert downloads and parses a signing certificate, caching it by URL
// (SNS rotates certificates by publishing them at new URLs).
func fetchSNSCert(certURL string) (*x509.Certificate, error) {
	snsCertMu.Lock()
	cert, ok := snsCertCache[certURL]
	snsCertMu.Unlock()
	if ok {
		return cert, nil
	}
	resp, err := snsClient.Get(certURL)
	if err != nil {
		return nil, fmt.Errorf("fetch signing certificate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch signing certificate: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("read signing certificate: %w", err)
	}
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, errors.New("signing certificate is not PEM")
	}
	cert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse signing certificate: %w", err)
	}
	snsCertMu.Lock()
	snsCertCache[certURL] = cert
	snsCertMu.Unlock()
	return cert, nil
}

// verifySNSMessage checks an SNS message's signature: the certificate URL
// is an SNS host, the certificate is current, and the RSA signature over
// the canonical text verifies (SignatureVersion 1 = SHA1, 2 = SHA256).
func verifySNSMessage(m *snsMessage) error {
	if !isSNSURL(m.SigningCertURL) {
		return fmt.Errorf("signing certificate URL is not an SNS host: %q", m.SigningCertURL)
	}
	var hash crypto.Hash
	switch m.SignatureVersion {
	case "1":
		hash = crypto.SHA1
	case "2":
		hash = crypto.SHA256
	default:
		return fmt.Errorf("unsupported SignatureVersion %q", m.SignatureVersion)
	}
	sig, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil || len(sig) == 0 {
		return errors.New("signature is not base64")
	}
	text, err := snsStringToSign(m)
	if err != nil {
		return err
	}
	cert, err := snsCertFetcher(m.SigningCertURL)
	if err != nil {
		return err
	}
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return errors.New("signing certificate is expired or not yet valid")
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("signing certificate key is not RSA")
	}
	var digest []byte
	if hash == crypto.SHA1 {
		d := sha1.Sum([]byte(text))
		digest = d[:]
	} else {
		d := sha256.Sum256([]byte(text))
		digest = d[:]
	}
	if err := rsa.VerifyPKCS1v15(pub, hash, digest, sig); err != nil {
		return errors.New("signature does not verify")
	}
	return nil
}

// snsAccount returns the AWS account an org sends with. A variable so tests
// need no AWS.
var snsAccount = awsAccountForOrg

var (
	snsAccountMu    sync.Mutex
	snsAccountCache = map[string]string{} // access key id -> account id
)

// awsAccountForOrg looks up (once per access key) the account of the AWS
// credentials the org sends with.
func awsAccountForOrg(ctx context.Context, svcCtx *svc.ServiceContext, orgID string) (string, error) {
	region, accessKey, secretKey, err := email.AWSCredentials(ctx, svcCtx.DB, svcCtx.CryptoService, orgID)
	if err != nil {
		return "", fmt.Errorf("AWS credentials: %w", err)
	}
	snsAccountMu.Lock()
	account, ok := snsAccountCache[accessKey]
	snsAccountMu.Unlock()
	if ok {
		return account, nil
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	if err != nil {
		return "", err
	}
	out, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("GetCallerIdentity: %w", err)
	}
	account = aws.ToString(out.Account)
	snsAccountMu.Lock()
	snsAccountCache[accessKey] = account
	snsAccountMu.Unlock()
	return account, nil
}

// topicAccount is the account id in an SNS topic ARN
// (arn:aws:sns:<region>:<account>:<name>), or "".
func topicAccount(arn string) string {
	p := strings.Split(arn, ":")
	if len(p) != 6 || p[0] != "arn" || p[2] != "sns" {
		return ""
	}
	return p[4]
}

// checkTopicAccount refuses a message from a topic outside the AWS account
// the org sends with. SNS signs every topic's messages, so a valid signature
// alone would accept a stranger's topic subscribed to this URL.
func checkTopicAccount(ctx context.Context, svcCtx *svc.ServiceContext, orgID, topicArn string) error {
	got := topicAccount(topicArn)
	if got == "" {
		return fmt.Errorf("invalid TopicArn %q", topicArn)
	}
	want, err := snsAccount(ctx, svcCtx, orgID)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("topic %s is not in the org's AWS account", topicArn)
	}
	return nil
}
