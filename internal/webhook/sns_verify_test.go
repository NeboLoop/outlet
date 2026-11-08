package webhook

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

const testCertURL = "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-test.pem"

// testSNS signs messages the way Amazon SNS does, with a throwaway key.
var testSNS = newTestSNS()

type testSNSSigner struct {
	key     *rsa.PrivateKey
	certPEM []byte
}

func newTestSNS() *testSNSSigner {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "sns.amazonaws.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	return &testSNSSigner{key: key, certPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}
}

func (s *testSNSSigner) sign(m *snsMessage) {
	m.SignatureVersion = "2"
	m.SigningCertURL = testCertURL
	digest := sha256.Sum256([]byte(snsStringToSign(m)))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, digest[:])
	if err != nil {
		panic(err)
	}
	m.Signature = base64.StdEncoding.EncodeToString(sig)
}

// verifier trusts the test certificate. urlOK is the production check, plus
// local httptest servers so subscription tests can confirm against them.
func (s *testSNSSigner) verifier() *snsVerifier {
	return &snsVerifier{
		urlOK: func(raw string) bool {
			return isSNSURL(raw) || strings.HasPrefix(raw, "http://127.0.0.1:")
		},
		fetchCert: func(ctx context.Context, certURL string) ([]byte, error) {
			return s.certPEM, nil
		},
		certs: map[string]*x509.Certificate{},
	}
}

// testSESHandler serves requests the way production routing does: the org ID
// comes from the path, and SNS messages without a signature are signed first.
func testSESHandler(svcCtx *svc.ServiceContext) http.Handler {
	h := sesHandler(svcCtx, testSNS.verifier())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			r.Body = io.NopCloser(&errorReader{})
		} else {
			var m snsMessage
			if json.Unmarshal(body, &m) == nil && m.Signature == "" {
				testSNS.sign(&m)
				body, _ = json.Marshal(m)
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		h.ServeHTTP(w, pathvar.WithVars(r, map[string]string{"orgId": "org-test"}))
	})
}

func postSNS(t *testing.T, h http.Handler, m snsMessage) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(m)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/ses/org-test", bytes.NewReader(payload))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, pathvar.WithVars(req, map[string]string{"orgId": "org-test"}))
	return rr
}

func deliveryMessage() snsMessage {
	inner, _ := json.Marshal(sesNotification{NotificationType: "Delivery", Mail: sesMailInfo{MessageId: "m1"}})
	return snsMessage{
		Type:      "Notification",
		MessageId: "msg-1",
		TopicArn:  "arn:aws:sns:us-east-1:123456789012:ses",
		Message:   string(inner),
		Timestamp: "2026-11-08T12:00:00.000Z",
	}
}

func TestSNSSignedMessageAccepted(t *testing.T) {
	h := sesHandler(createSESServiceContext(), testSNS.verifier())
	m := deliveryMessage()
	testSNS.sign(&m)
	assert.Equal(t, http.StatusOK, postSNS(t, h, m).Code)
}

func TestSNSRejectsForgedMessages(t *testing.T) {
	h := sesHandler(createSESServiceContext(), testSNS.verifier())

	t.Run("unsigned", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, postSNS(t, h, deliveryMessage()).Code)
	})

	t.Run("tampered after signing", func(t *testing.T) {
		m := deliveryMessage()
		testSNS.sign(&m)
		m.Message = strings.Replace(m.Message, "Delivery", "Complaint", 1)
		assert.Equal(t, http.StatusForbidden, postSNS(t, h, m).Code)
	})

	t.Run("certificate off Amazon", func(t *testing.T) {
		m := deliveryMessage()
		testSNS.sign(&m)
		m.SigningCertURL = "https://attacker.example.com/cert.pem"
		assert.Equal(t, http.StatusForbidden, postSNS(t, h, m).Code)
	})

	t.Run("subscribe URL off Amazon", func(t *testing.T) {
		m := snsMessage{
			Type:         "SubscriptionConfirmation",
			MessageId:    "msg-2",
			TopicArn:     "arn:aws:sns:us-east-1:123456789012:ses",
			SubscribeURL: "https://169.254.169.254/latest/meta-data/",
			Token:        "t",
			Timestamp:    "2026-11-08T12:00:00.000Z",
		}
		testSNS.sign(&m)
		assert.Equal(t, http.StatusBadRequest, postSNS(t, h, m).Code)
	})
}

func TestIsSNSURL(t *testing.T) {
	for raw, want := range map[string]bool{
		"https://sns.us-east-1.amazonaws.com/cert.pem":          true,
		"https://sns.cn-north-1.amazonaws.com.cn/cert.pem":      true,
		"http://sns.us-east-1.amazonaws.com/cert.pem":           false,
		"https://sns.us-east-1.amazonaws.com.evil.com/cert.pem": false,
		"https://evil.com/sns.us-east-1.amazonaws.com/cert.pem": false,
		"https://xsns.us-east-1.amazonaws.com/cert.pem":         false,
		"": false,
	} {
		assert.Equal(t, want, isSNSURL(raw), raw)
	}
}
