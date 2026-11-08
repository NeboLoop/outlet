export * from './generate/outlet';
export type * from './generate/outletComponents';

import { download } from './generate/gocliRequest';

/**
 * downloadBackup saves a backup's file. The server streams the file, so it
 * isn't one of the generated JSON calls.
 */
export function downloadBackup(id: string, filename: string): Promise<void> {
	return download(`/api/admin/backup/${encodeURIComponent(id)}/download`, filename);
}
