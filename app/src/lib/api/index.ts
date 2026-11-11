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

/**
 * downloadBackup saves a backup's file. The server streams the file, so it
 * isn't one of the generated JSON calls.
 */
export function downloadBackup(id: string, filename: string): Promise<void> {
	return download(`/api/admin/backup/${encodeURIComponent(id)}/download`, filename);
}

/**
 * exportOutboundLeads saves the brand's outreach leads as CSV: one row per
 * person per campaign, with segment, status and reply classification
 * (interested: the hot leads). The server streams the file.
 */
export function exportOutboundLeads(params: { campaign_id?: string; classification?: string } = {}): Promise<void> {
	return download(genUrl(`/api/admin/outbound/leads/export`, params), 'leads.csv');
}
