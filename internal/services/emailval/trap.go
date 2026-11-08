package emailval

import (
	"regexp"
	"strings"
)

// trapKeywords mark local parts or domain labels that announce themselves as traps.
var trapKeywords = []string{
	"spamtrap", "spam-trap", "spam_trap", "spam.trap",
	"honeypot", "honey-pot", "honey_pot",
	"abuse-report", "abuse_report", "abusereport",
	"blackhole", "black-hole",
}

// trapLocals are exact local parts used for traps (matched after stripping +tags).
var trapLocals = toSet("trap", "traps", "spam", "spambox", "spamreport", "spam-report", "junk-trap")

// isPossibleTrap is a data-free heuristic: addresses that name themselves as traps.
// Real spamtraps are unannounced; this only catches the self-describing ones planted in lists.
func isPossibleTrap(local, domain string) bool {
	base := local
	if i := strings.IndexByte(base, '+'); i > 0 {
		base = base[:i]
	}
	if trapLocals[base] {
		return true
	}
	for _, kw := range trapKeywords {
		if strings.Contains(local, kw) {
			return true
		}
	}
	for _, label := range strings.Split(domain, ".") {
		for _, kw := range trapKeywords {
			if strings.Contains(label, kw) {
				return true
			}
		}
	}
	return false
}

var (
	hexLocal  = regexp.MustCompile(`^[0-9a-f]{24,}$`)
	uuidLocal = regexp.MustCompile(`^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$`)
)

// isToxic flags obviously machine-generated mailboxes on free providers:
// a 24+ char pure-hex local part mixing letters and digits, or a UUID.
// Ceiling: this is deliberately narrow. Humans do not pick 24 hex chars or UUIDs on gmail/yahoo,
// so false positives are rare; anything fuzzier (random-looking words, digit runs) is left to
// Signal Network data rather than guessed here.
func isToxic(local string, free bool) bool {
	if !free {
		return false
	}
	if i := strings.IndexByte(local, '+'); i > 0 {
		local = local[:i]
	}
	if uuidLocal.MatchString(local) {
		return true
	}
	return hexLocal.MatchString(local) &&
		strings.ContainsAny(local, "0123456789") && strings.ContainsAny(local, "abcdef")
}
