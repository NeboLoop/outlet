// The Help guides: one Markdown file per guide in ./guides, listed here in
// the order the Help page shows them. A guide's title is its first "# "
// heading and its summary is the first paragraph after it.
import { fillLabels } from './labels';

export interface Guide {
	slug: string;
	section: string;
	title: string;
	summary: string;
	markdown: string;
	// The words of the guide, lowercased, for the search box
	text: string;
}

const order: { section: string; slugs: string[] }[] = [
	{
		section: 'Start here',
		slugs: [
			'getting-started',
			'how-sending-works',
			'set-up-amazon-ses',
			'use-your-own-smtp-server',
			'sending-domain-and-dns'
		]
	},
	{
		section: 'Outreach',
		slugs: [
			'outreach-overview',
			'set-up-outreach',
			'connect-a-mailbox-service',
			'several-mailbox-services',
			'google-workspace-and-microsoft-365',
			'warm-up-and-limits'
		]
	},
	{
		section: 'Email marketing',
		slugs: [
			'campaigns-and-newsletters',
			'sequences-and-autoresponders',
			'transactional-email',
			'lists-and-subscribers',
			'clean-lists',
			'designs'
		]
	},
	{
		section: 'Connect and manage',
		slugs: [
			'webhooks',
			'mcp-and-api-keys',
			'users-and-roles',
			'privacy-and-deleting-data',
			'backups-and-updates'
		]
	},
	{
		section: 'When something goes wrong',
		slugs: ['troubleshooting']
	}
];

const files = import.meta.glob('./guides/*.md', {
	query: '?raw',
	import: 'default',
	eager: true
}) as Record<string, string>;

export const guideFiles: string[] = Object.keys(files).map((path) =>
	path.replace('./guides/', '').replace(/\.md$/, '')
);

function parse(slug: string, section: string): Guide {
	const raw = files[`./guides/${slug}.md`];
	if (raw === undefined) throw new Error(`Help guide ${slug} has no file`);
	const markdown = fillLabels(raw);
	const lines = markdown.split('\n');
	const titleLine = lines.findIndex((line) => line.startsWith('# '));
	const title = titleLine >= 0 ? lines[titleLine].slice(2).trim() : '';
	const rest = lines.slice(titleLine + 1);
	const start = rest.findIndex((line) => line.trim() !== '');
	const end = rest.findIndex((line, i) => i > start && line.trim() === '');
	const summary = rest
		.slice(start, end < 0 ? undefined : end)
		.join(' ')
		.replace(/\*\*|`|\[|\]\([^)]*\)/g, '')
		.trim();
	return { slug, section, title, summary, markdown, text: markdown.toLowerCase() };
}

export const guides: Guide[] = order.flatMap(({ section, slugs }) =>
	slugs.map((slug) => parse(slug, section))
);

export const sections: string[] = order.map(({ section }) => section);

export function findGuide(slug: string): Guide | undefined {
	return guides.find((guide) => guide.slug === slug);
}

// searchGuides keeps the guides that hold every word of the query, in the
// title or the text; title matches come first.
export function searchGuides(query: string): Guide[] {
	const words = query.toLowerCase().split(/\s+/).filter(Boolean);
	if (words.length === 0) return guides;
	const hits = guides.filter((guide) => words.every((word) => guide.text.includes(word)));
	const inTitle = (guide: Guide) => words.every((word) => guide.title.toLowerCase().includes(word));
	return [...hits.filter(inTitle), ...hits.filter((guide) => !inTitle(guide))];
}

// snippet returns a short piece of the guide around the first query word,
// so a search result shows why it matched.
export function snippet(guide: Guide, query: string): string {
	const word = query.toLowerCase().split(/\s+/).find(Boolean);
	const plain = guide.markdown
		.replace(/^#+ .*$/gm, '')
		.replace(/^\s*(?:[-*]|\d+\.)\s+/gm, '')
		.replace(/\|/g, ' ')
		.replace(/\*\*|`|\[|\]\([^)]*\)/g, '')
		.replace(/\s+/g, ' ')
		.trim();
	if (!word) return guide.summary;
	const at = plain.toLowerCase().indexOf(word);
	if (at < 0) return guide.summary;
	const from = Math.max(0, at - 60);
	const to = Math.min(plain.length, at + 140);
	return `${from > 0 ? '...' : ''}${plain.slice(from, to)}${to < plain.length ? '...' : ''}`;
}
