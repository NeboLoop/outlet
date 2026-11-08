package emailval

// Forwarding rule:
//   - Privacy relays (Apple Hide My Email, DuckDuckGo, Firefox Relay, SimpleLogin, addy.io, Proton Pass,
//     33mail) hand out throwaway aliases that hide the real inbox and can be switched off by the owner
//     at any time. Mail to them is do_not_mail/mx_forward whether the address uses the relay's own domain
//     or a custom domain pointed at the relay's MX.
//   - Plain forwarders (ImprovMX, ForwardEmail, Cloudflare Email Routing) relay a custom domain the
//     owner controls to a real mailbox. They are not risky on their own: the address is graded by the
//     normal pipeline and, when nothing else applies, reported as valid/alias_address.

// privacyRelayDomains are address domains (and their subdomains) owned by privacy relay services.
var privacyRelayDomains = []string{
	"privaterelay.appleid.com",
	"duck.com",
	"mozmail.com",
	"relay.firefox.com",
	"simplelogin.com", "simplelogin.fr", "aleeas.com", "slmail.me", "slmails.com", "silomails.com",
	"8alias.com", "8shield.net", "dralias.com",
	"anonaddy.com", "anonaddy.me", "addy.io",
	"passmail.net", "passmail.com", "passinbox.com",
	"33mail.com",
}

// privacyRelayMX are MX suffixes of privacy relays (catches custom domains pointed at them).
var privacyRelayMX = []string{"simplelogin.co", "anonaddy.me", "addy.io", "33mail.com"}

// plainForwarderMX are MX suffixes of plain forwarding services.
var plainForwarderMX = []string{"improvmx.com", "forwardemail.net", "mx.cloudflare.net"}

// isPrivacyRelay reports whether the address belongs to a privacy relay service.
func isPrivacyRelay(domain string, hosts []string) bool {
	if anyHostSuffix(domain, privacyRelayDomains) {
		return true
	}
	for _, h := range hosts {
		if anyHostSuffix(h, privacyRelayMX) {
			return true
		}
	}
	return false
}

// isPlainForwarder reports whether the domain's preferred MX is a plain forwarding service.
func isPlainForwarder(hosts []string) bool {
	return len(hosts) > 0 && anyHostSuffix(hosts[0], plainForwarderMX)
}

// aiAgentDomains are mailbox services built for AI agents (address domain or MX suffix).
// ponytail: tiny on purpose; extend as agent-mail providers appear.
var aiAgentDomains = []string{"agentmail.to"}

// isAIAgentMailbox reports whether the address is hosted by an AI-agent mailbox service.
func isAIAgentMailbox(domain string, hosts []string) bool {
	if anyHostSuffix(domain, aiAgentDomains) {
		return true
	}
	for _, h := range hosts {
		if anyHostSuffix(h, aiAgentDomains) {
			return true
		}
	}
	return false
}
