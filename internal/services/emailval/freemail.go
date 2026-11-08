package emailval

// freeEmailDomains are free consumer mailbox providers (Result.FreeEmail).
var freeEmailDomains = toSet(
	// Google
	"gmail.com", "googlemail.com",
	// Microsoft
	"hotmail.com", "hotmail.co.uk", "hotmail.fr", "hotmail.de", "hotmail.it", "hotmail.es", "hotmail.ca",
	"hotmail.com.au", "hotmail.com.br", "hotmail.nl", "hotmail.be", "hotmail.se", "hotmail.no", "hotmail.dk",
	"outlook.com", "outlook.fr", "outlook.de", "outlook.it", "outlook.es", "outlook.co.uk", "outlook.jp",
	"outlook.com.br", "outlook.com.au", "outlook.in", "outlook.be", "outlook.ie", "outlook.at", "outlook.dk",
	"live.com", "live.co.uk", "live.fr", "live.de", "live.it", "live.nl", "live.ca", "live.com.au", "live.com.mx",
	"live.be", "live.se", "live.dk", "live.no", "msn.com", "windowslive.com", "passport.com",
	// Yahoo / AOL
	"yahoo.com", "yahoo.co.uk", "yahoo.fr", "yahoo.de", "yahoo.it", "yahoo.es", "yahoo.ca", "yahoo.com.au",
	"yahoo.com.br", "yahoo.co.in", "yahoo.co.jp", "yahoo.com.mx", "yahoo.com.ar", "yahoo.com.sg", "yahoo.com.ph",
	"yahoo.co.id", "yahoo.gr", "yahoo.ie", "yahoo.in", "yahoo.se", "yahoo.dk", "yahoo.no", "yahoo.com.hk",
	"yahoo.com.tw", "yahoo.com.vn", "yahoo.co.nz", "yahoo.co.za", "ymail.com", "rocketmail.com",
	"aol.com", "aol.co.uk", "aol.fr", "aol.de", "aim.com", "love.com", "games.com", "wow.com",
	// Apple
	"icloud.com", "me.com", "mac.com",
	// Privacy / independent
	"protonmail.com", "protonmail.ch", "proton.me", "pm.me", "tutanota.com", "tutanota.de", "tutamail.com",
	"tuta.io", "tuta.com", "keemail.me", "fastmail.com", "fastmail.fm", "hushmail.com", "mailfence.com",
	"posteo.de", "posteo.net", "mailbox.org", "runbox.com", "disroot.org", "riseup.net", "zoho.com", "zohomail.com",
	// Portals
	"mail.com", "email.com", "usa.com", "post.com", "myself.com", "consultant.com", "gmx.com", "gmx.net",
	"gmx.de", "gmx.at", "gmx.ch", "gmx.fr", "gmx.us", "gmx.co.uk", "web.de", "inbox.com", "lycos.com",
	"excite.com", "juno.com", "netzero.net", "rediffmail.com",
	// Russia / CIS
	"yandex.ru", "yandex.com", "yandex.by", "yandex.kz", "yandex.ua", "ya.ru", "mail.ru", "inbox.ru", "list.ru",
	"bk.ru", "internet.ru", "rambler.ru", "ukr.net", "i.ua", "meta.ua",
	// Asia
	"qq.com", "foxmail.com", "163.com", "126.com", "yeah.net", "sina.com", "sina.cn", "sohu.com", "aliyun.com",
	"139.com", "naver.com", "daum.net", "hanmail.net", "nate.com",
	// Europe
	"libero.it", "virgilio.it", "tiscali.it", "alice.it", "tin.it", "laposte.net", "orange.fr", "wanadoo.fr",
	"free.fr", "sfr.fr", "neuf.fr", "t-online.de", "freenet.de", "arcor.de", "seznam.cz", "centrum.cz",
	"wp.pl", "o2.pl", "interia.pl", "onet.pl", "op.pl", "tlen.pl", "abv.bg", "mail.bg", "freemail.hu",
	"citromail.hu", "azet.sk", "zoznam.sk", "bluewin.ch", "telenet.be", "skynet.be", "home.nl", "planet.nl",
	"ziggo.nl", "kpnmail.nl", "btinternet.com", "virginmedia.com", "sky.com", "talktalk.net", "ntlworld.com",
	"blueyonder.co.uk", "eircom.net", "sapo.pt", "terra.com.br", "bol.com.br", "uol.com.br", "ig.com.br",
)

// ispDomains are popular ISP-issued consumer mailboxes. Not "free", but common typo targets.
var ispDomains = toSet(
	"comcast.net", "att.net", "sbcglobal.net", "bellsouth.net", "verizon.net", "cox.net", "charter.net",
	"spectrum.net", "earthlink.net", "optonline.net", "frontier.com", "frontiernet.net", "windstream.net",
	"centurylink.net", "q.com", "roadrunner.com", "rr.com", "twc.com", "embarqmail.com", "mchsi.com",
	"suddenlink.net", "cableone.net", "netscape.net", "prodigy.net", "pacbell.net", "ameritech.net",
	"shaw.ca", "rogers.com", "sympatico.ca", "telus.net", "videotron.ca", "bigpond.com", "bigpond.net.au",
	"optusnet.com.au", "iinet.net.au", "xtra.co.nz",
)

// isFreeEmail reports whether domain is a free consumer mailbox provider.
func isFreeEmail(domain string) bool {
	return freeEmailDomains[domain]
}

func toSet(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, s := range items {
		m[s] = true
	}
	return m
}
