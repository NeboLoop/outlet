// Which Help guide each screen links to, by SvelteKit route id. A screen
// not listed takes the guide of the nearest listed screen above it, so
// /[brandSlug]/lists/[id]/subscribers uses the /[brandSlug]/lists guide.
export const screenGuides: Record<string, string> = {
	'/(brand-select)': 'getting-started',
	'/(brand-select)/settings': 'getting-started',
	'/(brand-select)/settings/api': 'mcp-and-api-keys',
	'/(brand-select)/settings/backup': 'backups-and-updates',
	'/(brand-select)/settings/brands': 'getting-started',
	'/(brand-select)/settings/email': 'set-up-amazon-ses',
	'/(brand-select)/settings/mcp': 'mcp-and-api-keys',
	'/(brand-select)/settings/updates': 'backups-and-updates',
	'/(brand-select)/settings/users': 'users-and-roles',
	'/(onboarding)/[brandSlug]/welcome': 'getting-started',

	'/[brandSlug]': 'getting-started',
	'/[brandSlug]/blocklist': 'clean-lists',
	'/[brandSlug]/housekeeping': 'clean-lists',
	'/[brandSlug]/validation': 'clean-lists',
	'/[brandSlug]/campaigns': 'campaigns-and-newsletters',
	'/[brandSlug]/lists': 'lists-and-subscribers',
	'/[brandSlug]/lists/[id]/autoresponders': 'sequences-and-autoresponders',
	'/[brandSlug]/templates': 'designs',
	'/[brandSlug]/transactional': 'transactional-email',

	'/[brandSlug]/outbound': 'outreach-overview',
	'/[brandSlug]/outbound/campaigns': 'set-up-outreach',
	'/[brandSlug]/outbound/drafts': 'set-up-outreach',
	'/[brandSlug]/outbound/replies': 'set-up-outreach',
	'/[brandSlug]/outbound/prospects': 'set-up-outreach',
	'/[brandSlug]/outbound/targeting': 'outreach-targeting',
	'/[brandSlug]/outbound/materials': 'outreach-targeting',
	'/[brandSlug]/outbound/do-not-contact': 'outreach-targeting',
	'/[brandSlug]/outbound/setup': 'set-up-outreach',

	'/[brandSlug]/settings': 'getting-started',
	'/[brandSlug]/settings/email': 'set-up-amazon-ses',
	'/[brandSlug]/settings/sending': 'how-sending-works',
	'/[brandSlug]/settings/smtp': 'transactional-email',
	'/[brandSlug]/settings/webhooks': 'webhooks',
	'/[brandSlug]/settings/people': 'users-and-roles',
	'/[brandSlug]/settings/privacy': 'privacy-and-deleting-data'
};

// The Help pages themselves link to the Help index
const helpRoute = '/(brand-select)/help';

// guideForScreen returns the guide slug for a route id, walking up the
// route until a listed screen is found. Undefined means the Help index.
export function guideForScreen(routeId: string | null): string | undefined {
	let id = routeId ?? '';
	if (id === helpRoute || id.startsWith(`${helpRoute}/`)) return undefined;
	while (id) {
		const slug = screenGuides[id];
		if (slug) return slug;
		id = id.slice(0, id.lastIndexOf('/'));
	}
	return undefined;
}
