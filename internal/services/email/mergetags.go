package email

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
)

// MergeContext holds the values merge tags are filled from.
type MergeContext struct {
	BaseURL           string
	Name              string
	Email             string
	TrackingToken     string // enables {{unsubscribe_url}} and {{web_version_url}}
	VerificationToken string // enables {{confirm_url}}
	CustomFields      map[string]string
}

var mergeTagPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*(?:,\s*fallback\s*=\s*([^}]*))?\}\}`)

// FillMergeTags is the one merge-tag implementation for campaigns,
// sequences and transactional emails. It fills {{name}}, {{first_name}},
// {{email}}, custom fields (a transactional email's variables),
// {{unsubscribe_url}}, {{web_version_url}} and {{confirm_url}}; any tag may
// carry a fallback, as in {{first_name,fallback=there}}. With escapeHTML,
// subscriber-supplied values are escaped for an HTML body; subjects and plain
// text pass false. Unknown tags are left as they are.
func FillMergeTags(content string, c MergeContext, escapeHTML bool) string {
	out, _ := fill(content, c, escapeHTML)
	return out
}

// FillTags fills merge tags in plain text for one person and reports the tags
// that came out empty (no value and no fallback). One-to-one outreach refuses
// to send those rather than write "Hi ,".
func FillTags(content, name, address string, fields map[string]string) (string, []string) {
	return fill(content, MergeContext{Name: name, Email: address, CustomFields: fields}, false)
}

func fill(content string, c MergeContext, escapeHTML bool) (string, []string) {
	base := strings.TrimRight(c.BaseURL, "/")
	var empty []string
	out := mergeTagPattern.ReplaceAllStringFunc(content, func(tag string) string {
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
				empty = append(empty, m[1])
				return tag
			}
			value = v
		}
		if value == "" {
			value = fallback
		}
		if strings.TrimSpace(value) == "" {
			empty = append(empty, m[1])
		}
		if escapeHTML && !isURL {
			value = html.EscapeString(value)
		}
		return value
	})
	return out, empty
}
