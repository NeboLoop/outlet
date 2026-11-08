package email

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
)

// mergeContext holds the values merge tags are filled from.
type mergeContext struct {
	BaseURL           string
	Name              string
	Email             string
	TrackingToken     string // enables {{unsubscribe_url}} and {{web_version_url}}
	VerificationToken string // enables {{confirm_url}}
	CustomFields      map[string]string
}

var mergeTagPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*(?:,\s*fallback\s*=\s*([^}]*))?\}\}`)

// fillMergeTags is the one merge-tag implementation for campaigns and
// sequences. It fills {{name}}, {{first_name}}, {{email}}, custom fields,
// {{unsubscribe_url}}, {{web_version_url}} and {{confirm_url}}; any tag may
// carry a fallback, as in {{first_name,fallback=there}}. With escapeHTML,
// subscriber-supplied values are escaped for an HTML body; subjects and plain
// text pass false. Unknown tags are left as they are.
func fillMergeTags(content string, c mergeContext, escapeHTML bool) string {
	base := strings.TrimRight(c.BaseURL, "/")
	return mergeTagPattern.ReplaceAllStringFunc(content, func(tag string) string {
		m := mergeTagPattern.FindStringSubmatch(tag)
		key, fallback := strings.ToLower(m[1]), strings.TrimSpace(m[2])

		var value string
		isURL := false
		switch key {
		case "name":
			value = c.Name
		case "first_name":
			value = strings.TrimSpace(strings.Split(strings.TrimSpace(c.Name), " ")[0])
		case "email":
			value = c.Email
		case "unsubscribe_url":
			if c.TrackingToken == "" {
				return tag
			}
			value, isURL = unsubscribeURL(c.BaseURL, c.TrackingToken), true
		case "web_version_url":
			if c.TrackingToken == "" {
				return tag
			}
			value, isURL = fmt.Sprintf("%s/w/%s", base, c.TrackingToken), true
		case "confirm_url":
			if c.VerificationToken == "" {
				return tag
			}
			value, isURL = fmt.Sprintf("%s/api/confirm-email?token=%s", base, url.QueryEscape(c.VerificationToken)), true
		default:
			v, ok := c.CustomFields[m[1]]
			if !ok && fallback == "" {
				return tag
			}
			value = v
		}
		if value == "" {
			value = fallback
		}
		if escapeHTML && !isURL {
			value = html.EscapeString(value)
		}
		return value
	})
}
