package outbound

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
)

// timeFormat is how outbound stores times: UTC text that compares in order.
const timeFormat = "2006-01-02 15:04:05"

func ts(t time.Time) string { return t.UTC().Format(timeFormat) }

func parseTS(s string) time.Time {
	t, _ := time.ParseInLocation(timeFormat, s, time.UTC)
	return t
}

var (
	spinRe  = regexp.MustCompile(`\{([^{}]*\|[^{}]*)\}`)
	mergeRe = regexp.MustCompile(`\{\{\s*([a-z_]+)\s*\}\}`)
)

// spin picks one option from each {a|b|c} group, so no two emails are
// byte-identical. Merge tags ({{name}}) have no pipe, so they are left alone.
func spin(s string, r *rand.Rand) string {
	return spinRe.ReplaceAllStringFunc(s, func(m string) string {
		opts := strings.Split(m[1:len(m)-1], "|")
		return opts[r.IntN(len(opts))]
	})
}

// mergeFields are the values a template can name for one prospect.
func mergeFields(p db.OutboundProspect, mb db.OutboundMailbox) map[string]string {
	f := map[string]string{
		"first_name":    p.FirstName,
		"name":          p.Name,
		"email":         p.Email,
		"platform":      p.Platform,
		"channel_url":   p.ChannelUrl,
		"niche":         p.Niche,
		"recent_video":  p.RecentVideo,
		"personal_line": p.PersonalLine,
		"sender_name":   mb.FromName,
		"sender_first":  firstWord(mb.FromName),
	}
	if p.AudienceSize > 0 {
		f["audience_size"] = fmt.Sprint(p.AudienceSize)
	}
	if f["first_name"] == "" {
		f["first_name"] = firstWord(p.Name)
	}
	for k, v := range customFields(p) {
		if _, taken := f[k]; !taken {
			f[k] = v
		}
	}
	return f
}

// render fills a template for one prospect. A tag with no value is an error,
// never a blank: "Hi ," is the fastest way to look like a mass mailing.
func render(tmpl string, fields map[string]string, r *rand.Rand) (string, error) {
	out := spin(tmpl, r)
	var missing []string
	out = mergeRe.ReplaceAllStringFunc(out, func(m string) string {
		key := mergeRe.FindStringSubmatch(m)[1]
		v := strings.TrimSpace(fields[key])
		if v == "" {
			missing = append(missing, key)
		}
		return v
	})
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", fmt.Errorf("no value for %s", strings.Join(dedupe(missing), ", "))
	}
	return out, nil
}

func firstWord(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}

func dedupe(s []string) []string {
	out := s[:0]
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}

// addBusinessDays moves t forward n weekdays.
func addBusinessDays(t time.Time, n int) time.Time {
	for n > 0 {
		t = t.AddDate(0, 0, 1)
		if wd := t.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n--
		}
	}
	return t
}

// nextWindow is the earliest moment at or after t that falls inside the
// weekday sending window [startHour, endHour) in loc.
func nextWindow(t time.Time, loc *time.Location, startHour, endHour int) time.Time {
	l := t.In(loc)
	for i := 0; i < 8; i++ {
		day := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
		open := day.Add(time.Duration(startHour) * time.Hour)
		close := day.Add(time.Duration(endHour) * time.Hour)
		if wd := day.Weekday(); wd != time.Saturday && wd != time.Sunday && l.Before(close) {
			if l.Before(open) {
				return open.UTC()
			}
			return l.UTC()
		}
		l = day.AddDate(0, 0, 1)
	}
	return t.UTC()
}

func loadLocation(name, fallback string) *time.Location {
	for _, n := range []string{name, fallback} {
		if n == "" {
			continue
		}
		if loc, err := time.LoadLocation(n); err == nil {
			return loc
		}
	}
	return time.UTC
}
