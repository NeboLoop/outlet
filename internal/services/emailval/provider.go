package emailval

import "strings"

// mxProviders maps MX host suffixes to the mailbox/gateway provider behind them.
// Order matters: more specific suffixes come before their parents (AOL lives under yahoodns.net).
var mxProviders = []struct{ suffix, provider string }{
	{"mx-aol.mail.gm0.yahoodns.net", "aol"},
	{"aol.com", "aol"},
	{"google.com", "google"},
	{"googlemail.com", "google"},
	{"outlook.com", "microsoft"}, // *.mail.protection.outlook.com, *.olc.protection.outlook.com
	{"hotmail.com", "microsoft"},
	{"outlook.cn", "microsoft"},
	{"yahoodns.net", "yahoo"},
	{"yahoo.com", "yahoo"},
	{"icloud.com", "apple"},
	{"me.com", "apple"},
	{"zoho.com", "zoho"},
	{"zoho.eu", "zoho"},
	{"zoho.in", "zoho"},
	{"zoho.com.au", "zoho"},
	{"zohomail.com", "zoho"},
	{"pphosted.com", "proofpoint"},
	{"ppe-hosted.com", "proofpoint"},
	{"mimecast.com", "mimecast"},
	{"mimecast.co.za", "mimecast"},
	{"mimecast-offshore.com", "mimecast"},
	{"barracudanetworks.com", "barracuda"},
	{"cudasvc.com", "barracuda"},
	{"iphmx.com", "cisco"},
	{"messagingengine.com", "fastmail"},
	{"fastmail.com", "fastmail"},
	{"protonmail.ch", "protonmail"},
	{"proton.me", "protonmail"},
	{"gmx.net", "gmx"},
	{"gmx.com", "gmx"},
	{"web.de", "gmx"},
	{"ionos.com", "ionos"},
	{"ionos.de", "ionos"},
	{"ionos.co.uk", "ionos"},
	{"1and1.com", "ionos"},
	{"kundenserver.de", "ionos"},
	{"perfora.net", "ionos"},
	{"yandex.net", "yandex"},
	{"yandex.ru", "yandex"},
	{"mail.ru", "mailru"},
	{"mailgun.org", "mailgun"},
	{"amazonaws.com", "amazon_ses"}, // inbound-smtp.<region>.amazonaws.com
	{"amazonses.com", "amazon_ses"},
	{"secureserver.net", "godaddy"},
	{"privateemail.com", "namecheap"},
	{"registrar-servers.com", "namecheap"},
	{"emailsrvr.com", "rackspace"},
	{"ovh.net", "ovh"},
	{"gandi.net", "gandi"},
	{"mailchannels.net", "mailchannels"},
	{"hydra.sophos.com", "sophos"},
	{"tmes.trendmicro.com", "trendmicro"},
	{"tmes.trendmicro.eu", "trendmicro"},
	{"mailcontrol.com", "forcepoint"},
	{"messagelabs.com", "broadcom"},
	{"hornetsecurity.com", "hornetsecurity"},
	{"spamexperts.com", "spamexperts"},
	{"tutanota.de", "tutanota"},
	{"qq.com", "tencent"},
	{"163.com", "netease"},
	{"naver.com", "naver"},
	{"improvmx.com", "improvmx"},
	{"forwardemail.net", "forwardemail"},
	{"mx.cloudflare.net", "cloudflare"},
	{"simplelogin.co", "simplelogin"},
	{"anonaddy.me", "addy"},
	{"addy.io", "addy"},
	{"agentmail.to", "agentmail"},
}

// detectProvider names the provider of the first MX host (preference order) it recognizes, or "".
func detectProvider(hosts []string) string {
	for _, h := range hosts {
		for _, p := range mxProviders {
			if hasDomainSuffix(h, p.suffix) {
				return p.provider
			}
		}
	}
	return ""
}

// hasDomainSuffix reports whether host is suffix or a subdomain of it.
func hasDomainSuffix(host, suffix string) bool {
	return host == suffix || strings.HasSuffix(host, "."+suffix)
}

// anyHostSuffix reports whether host matches any of suffixes.
func anyHostSuffix(host string, suffixes []string) bool {
	for _, s := range suffixes {
		if hasDomainSuffix(host, s) {
			return true
		}
	}
	return false
}
