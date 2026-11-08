import { error } from '@sveltejs/kit';
import { findGuide, guides } from '$lib/help/guides';
import type { EntryGenerator, PageLoad } from './$types';

// Every guide is built into its own page
export const entries: EntryGenerator = () => guides.map((guide) => ({ slug: guide.slug }));

export const load: PageLoad = ({ params }) => {
	const guide = findGuide(params.slug);
	if (!guide) error(404, 'There is no help guide by that name');
	return { guide };
};
