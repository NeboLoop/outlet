package email

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
)

// SetLinkSecret sets the secret that signs click-tracking links, so the click
// endpoint only redirects to URLs Outlet itself put in an email.
func (s *Service) SetLinkSecret(secret string) {
	mac := sha256.Sum256([]byte("outlet-click-links\n" + secret))
	s.linkKey = mac[:]
}

// GetTrackingPixelURL returns the URL for a tracking pixel
func (s *Service) GetTrackingPixelURL(trackingToken string) string {
	return fmt.Sprintf("%s/api/e/o/%s", strings.TrimRight(s.baseURL, "/"), trackingToken)
}

// RewriteLinksForTracking replaces links with signed click-tracking URLs.
func (s *Service) RewriteLinksForTracking(htmlBody, trackingToken string) string {
	return rewriteLinks(htmlBody, s.baseURL, s.linkKey, trackingToken)
}

// ClickLink is the signed click-tracking URL for one target, on host (a full
// origin like https://go.example.com), or this server's own URL when host is
// empty. The click endpoint verifies the signature, so it only ever redirects
// where the email pointed.
func (s *Service) ClickLink(host, trackingToken, target string) string {
	if host == "" {
		host = s.baseURL
	}
	return clickLink(host, s.linkKey, trackingToken, target)
}

func clickLink(base string, key []byte, trackingToken, target string) string {
	sig := hex.EncodeToString(clickSignature(key, trackingToken, target))
	return fmt.Sprintf("%s/api/e/c/%s?url=%s&sig=%s", strings.TrimRight(base, "/"), trackingToken, url.QueryEscape(target), sig)
}

// VerifyClickLink reports whether sig is the signature Outlet made for this
// token and target URL.
func (s *Service) VerifyClickLink(trackingToken, target, sig string) bool {
	want, err := hex.DecodeString(sig)
	return err == nil && len(s.linkKey) > 0 && hmac.Equal(want, clickSignature(s.linkKey, trackingToken, target))
}

func clickSignature(key []byte, trackingToken, target string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(trackingToken + "\n" + target))
	return mac.Sum(nil)[:16]
}

var hrefPattern = regexp.MustCompile(`href="([^"]+)"`)

// rewriteLinks is the one link rewriter for campaigns and sequences. It leaves
// mailto, tel, anchors, merge tags, existing tracking links and the
// unsubscribe page alone. Without a key it leaves every link alone, since an
// unsigned tracking link would not redirect.
func rewriteLinks(htmlBody, baseURL string, key []byte, trackingToken string) string {
	if len(key) == 0 || trackingToken == "" {
		return htmlBody
	}
	base := strings.TrimRight(baseURL, "/")
	unsubscribe := unsubscribeURL(baseURL, trackingToken)

	return hrefPattern.ReplaceAllStringFunc(htmlBody, func(match string) string {
		// Attribute values are HTML-escaped (&amp;); sign and redirect to the
		// URL the browser would actually open
		target := html.UnescapeString(hrefPattern.FindStringSubmatch(match)[1])
		lower := strings.ToLower(target)
		if !(strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")) ||
			strings.Contains(target, "/api/e/c/") || target == unsubscribe {
			return match
		}
		return `href="` + html.EscapeString(clickLink(base, key, trackingToken, target)) + `"`
	})
}
