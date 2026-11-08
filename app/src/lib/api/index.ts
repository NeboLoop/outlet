export * from './generate/outlet';
export type * from './generate/outletComponents';

import { download, genUrl } from './generate/gocliRequest';

/**
 * exportListSubscribers saves a list's subscribers as CSV. The server
 * streams the file, so it isn't one of the generated JSON calls.
 */
export function exportListSubscribers(id: string, params: { status?: string } = {}): Promise<void> {
	return download(genUrl(`/api/admin/lists/${id}/export`, params), `list-${id}-subscribers.csv`);
}
