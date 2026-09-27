package webhook

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tests sign SNS messages with their own key, served as the signing
// certificate at testCertURL, and send with an org in testAccount.
const (
	testAccount      = "123456789"
	testTopic        = "arn:aws:sns:us-east-1:" + testAccount + ":outlet-ses-example-com"
	testCertURL      = "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-test.pem"
	testSubscribeURL = "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&TopicArn=" + testTopic + "&Token=t"
)

var (
	testKey  *rsa.PrivateKey
	testCert *x509.Certificate
)

func TestMain(m *testing.M) {
	var err error
	testKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "sns.amazonaws.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &testKey.PublicKey, testKey)
	if err != nil {
		panic(err)
	}
	testCert, _ = x509.ParseCertificate(der)

	snsCertFetcher = func(u string) (*x509.Certificate, error) {
		if u != testCertURL {
			return nil, fmt.Errorf("unexpected certificate URL %s", u)
		}
		return testCert, nil
	}
	snsAccount = func(context.Context, *svc.ServiceContext, string) (string, error) { return testAccount, nil }
	snsConfirm = func(string) error { return fmt.Errorf("no confirmation expected") }
	os.Exit(m.Run())
}

// stubConfirm stands in for SNS's SubscribeURL; the returned func restores.
func stubConfirm(f func(string) error) func() {
	prev := snsConfirm
	snsConfirm = f
	return func() { snsConfirm = prev }
}

// sign signs m as SNS does, with the given SignatureVersion.
func sign(t testing.TB, m *snsMessage, version string) {
	if err := signMsg(m, version); err != nil {
		t.Fatal(err)
	}
}

func signMsg(m *snsMessage, version string) error {
	if m.TopicArn == "" {
		m.TopicArn = testTopic
	}
	if m.Timestamp == "" {
		m.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	m.SigningCertURL = testCertURL
	m.SignatureVersion = version
	text, err := snsStringToSign(m)
	if err != nil {
		// A type SNS does not sign: leave the message unsigned.
		return nil
	}
	var sig []byte
	if version == "1" {
		d := sha1.Sum([]byte(text))
		sig, err = rsa.SignPKCS1v15(rand.Reader, testKey, crypto.SHA1, d[:])
	} else {
		d := sha256.Sum256([]byte(text))
		sig, err = rsa.SignPKCS1v15(rand.Reader, testKey, crypto.SHA256, d[:])
	}
	if err != nil {
		return err
	}
	m.Signature = base64.StdEncoding.EncodeToString(sig)
	return nil
}

// signedSNS is m signed (SignatureVersion 2) and encoded as SNS posts it.
func signedSNS(m snsMessage) ([]byte, error) {
	if err := signMsg(&m, "2"); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func notification() snsMessage {
	inner, _ := json.Marshal(sesNotification{NotificationType: "Delivery", Mail: sesMailInfo{MessageId: "m-1"}})
	return snsMessage{Type: "Notification", MessageId: "n-1", Subject: "Amazon SES Email Event Notification", Message: string(inner)}
}

func TestVerifySNSMessage(t *testing.T) {
	for _, v := range []string{"1", "2"} {
		m := notification()
		sign(t, &m, v)
		assert.NoError(t, verifySNSMessage(&m), "SignatureVersion %s", v)
	}

	sub := snsMessage{Type: "SubscriptionConfirmation", MessageId: "s-1", Message: "confirm", SubscribeURL: testSubscribeURL, Token: "t"}
	sign(t, &sub, "2")
	assert.NoError(t, verifySNSMessage(&sub))

	cases := map[string]func(m *snsMessage){
		"tampered message": func(m *snsMessage) { m.Message = `{"notificationType":"Bounce"}` },
		"tampered topic":   func(m *snsMessage) { m.TopicArn = "arn:aws:sns:us-east-1:999999999:x" },
		"tampered subject": func(m *snsMessage) { m.Subject = "" },
		"cert not on SNS":  func(m *snsMessage) { m.SigningCertURL = "https://evil.example.com/cert.pem" },
		"cert over http":   func(m *snsMessage) { m.SigningCertURL = "http://sns.us-east-1.amazonaws.com/cert.pem" },
		"lookalike cert host": func(m *snsMessage) {
			m.SigningCertURL = "https://sns.us-east-1.amazonaws.com.evil.example/cert.pem"
		},
		"unknown version": func(m *snsMessage) { m.SignatureVersion = "3" },
		"no signature":    func(m *snsMessage) { m.Signature = "" },
		"bad base64":      func(m *snsMessage) { m.Signature = "!!" },
	}
	for name, mutate := range cases {
		m := notification()
		sign(t, &m, "2")
		mutate(&m)
		assert.Error(t, verifySNSMessage(&m), name)
	}
}

func TestVerifySNSMessageExpiredCert(t *testing.T) {
	prev := snsCertFetcher
	defer func() { snsCertFetcher = prev }()
	expired := *testCert
	expired.NotAfter = time.Now().Add(-time.Minute)
	snsCertFetcher = func(string) (*x509.Certificate, error) { return &expired, nil }

	m := notification()
	sign(t, &m, "2")
	assert.Error(t, verifySNSMessage(&m))
}

func TestTopicAccount(t *testing.T) {
	assert.Equal(t, "123456789012", topicAccount("arn:aws:sns:us-east-1:123456789012:outlet-ses-nebo-bot"))
	assert.Equal(t, "", topicAccount("arn:aws:sqs:us-east-1:123456789012:q"))
	assert.Equal(t, "", topicAccount("not-an-arn"))
}

// The webhook refuses what it cannot trust, before it acts on it.
func TestSESHandler_RejectsUntrusted(t *testing.T) {
	handler := SESHandler(createSESServiceContext())
	post := func(body []byte) int {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, sesRequest(bytes.NewReader(body)))
		return rr.Code
	}

	// Unsigned.
	unsigned, _ := json.Marshal(notification())
	assert.Equal(t, http.StatusForbidden, post(unsigned))

	// Signed, but tampered with after signing.
	m := notification()
	sign(t, &m, "2")
	m.Message = `{"notificationType":"Bounce","bounce":{"bouncedRecipients":[{"emailAddress":"victim@example.com"}]}}`
	tampered, _ := json.Marshal(m)
	assert.Equal(t, http.StatusForbidden, post(tampered))

	// Validly signed by SNS, from a topic in someone else's AWS account.
	other := notification()
	other.TopicArn = "arn:aws:sns:us-east-1:999999999:attacker-topic"
	foreign, _ := signedSNS(other)
	assert.Equal(t, http.StatusForbidden, post(foreign))

	// A subscription from someone else's account is never confirmed.
	confirmed := false
	defer stubConfirm(func(string) error { confirmed = true; return nil })()
	sub := snsMessage{Type: "SubscriptionConfirmation", MessageId: "s-2", Message: "confirm",
		TopicArn: "arn:aws:sns:us-east-1:999999999:attacker-topic", SubscribeURL: testSubscribeURL, Token: "t"}
	foreignSub, _ := signedSNS(sub)
	assert.Equal(t, http.StatusForbidden, post(foreignSub))
	assert.False(t, confirmed)

	// A signed subscription whose SubscribeURL leaves SNS is refused by the
	// real confirmer.
	snsConfirm = confirmSNSSubscription
	evil := snsMessage{Type: "SubscriptionConfirmation", MessageId: "s-3", Message: "confirm",
		SubscribeURL: "https://169.254.169.254/latest/meta-data/", Token: "t"}
	evilSub, _ := signedSNS(evil)
	assert.Equal(t, http.StatusInternalServerError, post(evilSub))

	// And a trusted one is accepted.
	good, err := signedSNS(notification())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, post(good))
}
