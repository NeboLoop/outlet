// Visits every admin screen at phone width and fails when a page scrolls
// sideways or cuts content off inside a clipping box. Runs against a live,
// signed-in Outlet:
//
//   OUTLET_URL=http://localhost:8888 OUTLET_EMAIL=you@example.com \
//   OUTLET_PASSWORD=... OUTLET_BRAND=acme node scripts/check-mobile-overflow.mjs
//
// Optional: WIDTH and HEIGHT (default 390x844, an iPhone), SHOTS=<dir> to save
// a screenshot per screen. It uses the system Chrome, or Playwright's own
// Chromium when PLAYWRIGHT_CHROMIUM=1.
import { mkdirSync } from 'node:fs';
import { chromium } from 'playwright';

const base = (process.env.OUTLET_URL ?? 'http://localhost:8888').replace(/\/$/, '');
const email = process.env.OUTLET_EMAIL;
const password = process.env.OUTLET_PASSWORD;
const brand = process.env.OUTLET_BRAND;
const width = Number(process.env.WIDTH ?? 390);
const height = Number(process.env.HEIGHT ?? 844);
const shots = process.env.SHOTS;

if (!email || !password || !brand) {
	console.error('Set OUTLET_EMAIL, OUTLET_PASSWORD and OUTLET_BRAND (a brand slug).');
	process.exit(2);
}

// The first id of each kind of thing in the brand, so detail screens get
// visited too. A kind with nothing in it is skipped.
async function firstIds() {
	const login = await fetch(`${base}/api/v1/auth/login`, {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ email, password })
	}).then((r) => r.json());
	const auth = { Authorization: `Bearer ${login.token}` };
	const orgs = await fetch(`${base}/api/admin/organizations`, { headers: auth }).then((r) => r.json());
	const org = (orgs.organizations ?? []).find((o) => o.slug === brand);
	if (!org) throw new Error(`No brand with the slug "${brand}"`);
	const headers = { ...auth, 'X-Org-Id': org.id };
	const first = async (path, key) => {
		const body = await fetch(`${base}${path}`, { headers }).then((r) => r.json());
		return body[key]?.[0];
	};
	return {
		list: await first('/api/admin/lists', 'lists'),
		sequence: await first('/api/admin/sequences', 'sequences'),
		campaign: await first('/api/admin/campaigns', 'campaigns'),
		design: await first('/api/admin/email-designs', 'designs'),
		transactional: await first('/api/admin/transactional-emails', 'templates'),
		validation: await first('/api/admin/validation/jobs', 'jobs')
	};
}

function routes(ids) {
	const b = `/${brand}`;
	const list = ids.list && `${b}/lists/${ids.list.id}`;
	return [
		'/',
		'/help',
		'/help/getting-started',
		...['mcp', 'api', 'email', 'users', 'brands', 'backup', 'updates'].map((t) => `/settings/${t}`),
		b,
		`${b}/welcome`,
		`${b}/lists`,
		...(list
			? ['subscribers', 'autoresponders', 'custom-fields', 'settings', 'subscribe-form'].map((t) => `${list}/${t}`)
			: []),
		ids.sequence && `${b}/lists/${ids.sequence.list_id}/autoresponders/${ids.sequence.id}`,
		`${b}/campaigns`,
		`${b}/campaigns/new`,
		ids.campaign && `${b}/campaigns/${ids.campaign.id}`,
		`${b}/templates`,
		ids.design && `${b}/templates/${ids.design.id}`,
		`${b}/transactional`,
		ids.transactional && `${b}/transactional/${ids.transactional.id}`,
		`${b}/validation`,
		ids.validation && `${b}/validation/${ids.validation.id}`,
		`${b}/blocklist`,
		`${b}/housekeeping`,
		...['', '/setup', '/prospects', '/campaigns', '/campaigns/new', '/drafts', '/replies'].map((t) => `${b}/outbound${t}`),
		...['', '/email', '/sending', '/smtp', '/webhooks', '/privacy', '/people'].map((t) => `${b}/settings${t}`)
	].filter(Boolean);
}

// Runs in the page: how wide the page is, and boxes that hide content
// sideways (overflow hidden) without an ellipsis.
function inspect(vw) {
	const describe = (el) => `${el.tagName.toLowerCase()}.${String(el.className).slice(0, 70)}`;
	const clipped = [];
	for (const el of document.querySelectorAll('body *')) {
		const cs = getComputedStyle(el);
		if (cs.display === 'none' || cs.visibility === 'hidden') continue;
		const r = el.getBoundingClientRect();
		if (r.width === 0 || r.height === 0 || r.left >= vw || r.right <= 0) continue;
		const hides = cs.overflowX === 'hidden' || cs.overflowX === 'clip';
		if (!hides || el.scrollWidth <= el.clientWidth + 2 || cs.textOverflow === 'ellipsis') continue;
		const cut = [...el.querySelectorAll('*')].find((k) => {
			const kr = k.getBoundingClientRect();
			const pos = getComputedStyle(k).position;
			return kr.width > 0 && kr.right > r.right + 2 && pos !== 'absolute' && pos !== 'fixed';
		});
		if (cut) clipped.push(`${describe(el)} cuts off ${describe(cut)}`);
	}
	return { scrollWidth: document.documentElement.scrollWidth, clipped, path: location.pathname };
}

const channel = process.env.PLAYWRIGHT_CHROMIUM === '1' ? undefined : 'chrome';
const browser = await chromium.launch({ channel, headless: true });
const page = await browser.newPage({ viewport: { width, height }, hasTouch: true });
await page.goto(`${base}/auth/login`);
await page.fill('input[type=email]', email);
await page.fill('input[type=password]', password);
await page.click('button[type=submit]');
await page.waitForURL((u) => !u.pathname.startsWith('/auth'), { timeout: 15000 });

if (shots) mkdirSync(shots, { recursive: true });
let failures = 0;
for (const route of routes(await firstIds())) {
	await page.goto(base + route, { waitUntil: 'networkidle', timeout: 15000 }).catch(() => {});
	await page.waitForTimeout(300);
	const result = await page.evaluate(inspect, width);
	const ok = result.scrollWidth <= width && result.clipped.length === 0;
	if (!ok) failures++;
	console.log(`${ok ? 'PASS' : 'FAIL'} ${route}${result.path !== route ? ` -> ${result.path}` : ''} (page width ${result.scrollWidth})`);
	for (const c of result.clipped.slice(0, 5)) console.log(`     ${c}`);
	if (shots) await page.screenshot({ path: `${shots}/${route.replace(/\//g, '_').replace(/^_/, '') || 'root'}.png` });
}
await browser.close();
console.log(failures ? `${failures} screen(s) don't fit ${width}px` : `Every screen fits ${width}px`);
process.exit(failures ? 1 : 0);
