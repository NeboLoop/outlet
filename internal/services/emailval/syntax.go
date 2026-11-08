package emailval

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// Simple, user-friendly syntax regex (not full RFC 5322).
	// Matches: local@domain.tld where local and domain have no spaces.
	simpleRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

// normalize lowercases and trims whitespace from an email.
func normalize(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validateSyntax checks email syntax and returns local and domain parts.
func validateSyntax(email string) (local, domain string, err error) {
	if !simpleRegex.MatchString(email) {
		return "", "", errors.New("does not match email pattern")
	}

	local, domain, err = splitEmail(email)
	if err != nil {
		return "", "", err
	}

	// Check for invalid local-part structures.
	if strings.Contains(local, "..") {
		return "", "", errors.New("local-part contains consecutive dots")
	}
	if strings.HasPrefix(local, ".") {
		return "", "", errors.New("local-part starts with dot")
	}
	if strings.HasSuffix(local, ".") {
		return "", "", errors.New("local-part ends with dot")
	}
	if len(local) > 64 || len(email) > 254 {
		return "", "", errors.New("address too long")
	}
	if strings.ContainsAny(local, `"(),:;<>[\]`) {
		return "", "", errors.New("local-part contains characters that require quoting")
	}
	for _, label := range strings.Split(domain, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", "", errors.New("domain has an invalid label")
		}
		if strings.ContainsAny(label, "_!#$%&'*+/=?^`{|}~\"(),:;<>[\\]") {
			return "", "", errors.New("domain contains invalid characters")
		}
	}

	return local, domain, nil
}

// leadingPeriodDomains drop a leading period from the local part when delivering.
var leadingPeriodDomains = map[string]bool{"gmail.com": true, "googlemail.com": true}

// stripLeadingPeriod removes leading periods from the local part for Gmail/Googlemail,
// which ignore them; it reports whether anything was removed.
func stripLeadingPeriod(email string) (string, bool) {
	local, domain, ok := strings.Cut(email, "@")
	if !ok || !leadingPeriodDomains[domain] || !strings.HasPrefix(local, ".") {
		return email, false
	}
	trimmed := strings.TrimLeft(local, ".")
	if trimmed == "" {
		return email, false
	}
	return trimmed + "@" + domain, true
}

// splitEmail splits an email into local and domain parts.
func splitEmail(email string) (local, domain string, err error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", "", errors.New("must contain single @")
	}
	local = parts[0]
	domain = parts[1]
	if local == "" {
		return "", "", errors.New("empty local-part")
	}
	if domain == "" {
		return "", "", errors.New("empty domain")
	}
	return local, domain, nil
}
