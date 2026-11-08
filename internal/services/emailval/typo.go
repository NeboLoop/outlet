package emailval

import (
	"sort"
	"strings"
)

// commonTypos maps misspellings edit distance misses or ranks wrongly.
var commonTypos = map[string]string{
	"gmail.com.com":   "gmail.com",
	"gmail.co.uk":     "gmail.com",
	"gmailcom.com":    "gmail.com",
	"gmail.comcom":    "gmail.com",
	"googlemail.co":   "googlemail.com",
	"gogglemail.com":  "googlemail.com",
	"yahoo.com.com":   "yahoo.com",
	"yahoomail.com":   "yahoo.com",
	"yahoo.mail.com":  "yahoo.com",
	"hotmail.com.com": "hotmail.com",
	"hotmial.co.uk":   "hotmail.co.uk",
	"hotmail.co.uj":   "hotmail.co.uk",
	"outlook.com.com": "outlook.com",
	"outlok.co.uk":    "outlook.co.uk",
	"iclould.com":     "icloud.com",
	"icould.com":      "icloud.com",
	"aol.com.com":     "aol.com",
	"comcast.com":     "comcast.net",
	"comcast.nte":     "comcast.net",
	"sbcglobal.com":   "sbcglobal.net",
	"bellsouth.com":   "bellsouth.net",
	"protonmail.co":   "protonmail.com",
	"protonmail.me":   "proton.me",
}

// typoTopDomains rank first when two candidates are equally close.
var typoTopDomains = []string{
	"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "aol.com", "icloud.com", "live.com", "msn.com",
	"comcast.net", "att.net", "sbcglobal.net", "verizon.net", "bellsouth.net", "cox.net", "charter.net",
	"googlemail.com", "ymail.com", "me.com", "mac.com", "protonmail.com", "proton.me", "gmx.com", "gmx.de",
	"web.de", "mail.com", "yandex.ru", "mail.ru", "hotmail.co.uk", "yahoo.co.uk", "btinternet.com",
}

// typoExclude are short generic-word vanity domains (AOL's wow.com, mail.com's usa.com)
// that sit one edit away from too many legitimate business domains.
var typoExclude = toSet("wow.com", "love.com", "games.com", "post.com", "usa.com", "email.com", "inbox.com", "myself.com")

// typoCandidates is every popular consumer domain, top domains first, then alphabetical.
var typoCandidates = func() []string {
	seen := toSet(typoTopDomains...)
	for d := range typoExclude {
		seen[d] = true
	}
	rest := make([]string, 0, len(freeEmailDomains)+len(ispDomains))
	for _, set := range []map[string]bool{freeEmailDomains, ispDomains} {
		for d := range set {
			if !seen[d] {
				seen[d] = true
				rest = append(rest, d)
			}
		}
	}
	sort.Strings(rest)
	return append(append([]string{}, typoTopDomains...), rest...)
}()

// typoMinLen skips very short targets (qq.com, me.com, wp.pl): too many legit neighbours.
const typoMinLen = 7

// suggestDomain returns the popular domain that domain is probably a misspelling of, or "".
func suggestDomain(domain string) string {
	if s, ok := commonTypos[domain]; ok {
		return s
	}
	if freeEmailDomains[domain] || ispDomains[domain] {
		return ""
	}
	best, bestDist := "", 3
	for _, cand := range typoCandidates {
		if len(cand) < typoMinLen {
			continue
		}
		maxDist := 2
		if len(cand) < 9 {
			maxDist = 1
		}
		if d := editDistance(domain, cand); d <= maxDist && d < bestDist {
			best, bestDist = cand, d
		}
	}
	return best
}

// suggestAddress returns local@corrected-domain when the domain looks like a typo, or "".
func suggestAddress(local, domain string) string {
	if s := suggestDomain(domain); s != "" {
		return local + "@" + s
	}
	return ""
}

// editDistance is the optimal-string-alignment distance (Levenshtein plus adjacent transposition).
func editDistance(a, b string) int {
	if strings.EqualFold(a, b) {
		return 0
	}
	ra, rb := []rune(a), []rune(b)
	// Three rolling rows: i-2, i-1, i.
	prev2 := make([]int, len(rb)+1)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				cur[j] = min(cur[j], prev2[j-2]+1)
			}
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(rb)]
}
