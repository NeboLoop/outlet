/**
 * Email validation helpers (ZeroBounce v2 compatible statuses).
 * Shared by the validation pages and the ValidationBadge component.
 */
import type { ValidationJobInfo, ValidationResult } from '$lib/api';

export type BadgeVariant = 'success' | 'error' | 'warning' | 'neutral' | 'info';

export interface ValidationStatusMeta {
	key: string;
	label: string;
	variant: BadgeVariant;
	outline: boolean;
	/** CSS class (app.css) used for chart/bar segments */
	barClass: string;
	/** Field on ValidationJobInfo holding this status's count */
	countKey: keyof ValidationJobInfo;
}

export const VALIDATION_STATUSES: ValidationStatusMeta[] = [
	{ key: 'valid', label: 'Valid', variant: 'success', outline: false, barClass: 'vstatus-bar-valid', countKey: 'valid' },
	{ key: 'invalid', label: 'Invalid', variant: 'error', outline: false, barClass: 'vstatus-bar-invalid', countKey: 'invalid' },
	{ key: 'catch-all', label: 'Catch-all', variant: 'warning', outline: true, barClass: 'vstatus-bar-catch-all', countKey: 'catch_all' },
	{ key: 'unknown', label: 'Unknown', variant: 'neutral', outline: false, barClass: 'vstatus-bar-unknown', countKey: 'unknown' },
	{ key: 'spamtrap', label: 'Spamtrap', variant: 'error', outline: false, barClass: 'vstatus-bar-spamtrap', countKey: 'spamtrap' },
	{ key: 'abuse', label: 'Abuse', variant: 'error', outline: false, barClass: 'vstatus-bar-abuse', countKey: 'abuse' },
	{ key: 'do_not_mail', label: 'Do not mail', variant: 'warning', outline: false, barClass: 'vstatus-bar-do-not-mail', countKey: 'do_not_mail' }
];

/** Statuses suggested for suppression by default. */
export const DEFAULT_SUPPRESS_STATUSES = ['invalid', 'spamtrap', 'abuse', 'do_not_mail'];

/** Job states in which the job is still running and should be polled. */
export const ACTIVE_JOB_STATUSES = ['pending', 'processing'];

/** Max emails per create/append call. */
export const EMAIL_CHUNK_SIZE = 10000;

/** Badge variant for a validation job's lifecycle status. */
export function jobStatusVariant(status: string): BadgeVariant {
	switch (status) {
		case 'completed':
			return 'success';
		case 'processing':
			return 'info';
		case 'failed':
			return 'error';
		case 'pending':
			return 'warning';
		default:
			return 'neutral';
	}
}

export function validationStatusMeta(status: string): ValidationStatusMeta {
	const s = (status || '').toLowerCase().replace('catch_all', 'catch-all');
	return (
		VALIDATION_STATUSES.find((m) => m.key === s) ?? {
			key: s,
			label: humanize(s) || 'Unknown',
			variant: 'neutral',
			outline: false,
			barClass: 'vstatus-bar-unknown',
			countKey: 'unknown'
		}
	);
}

const SUB_STATUS_LABELS: Record<string, string> = {
	mx_forward: 'MX forward',
	unroutable_ip_address: 'Unroutable IP address',
	failed_smtp_connection: 'Failed SMTP connection',
	no_dns_entries: 'No DNS entries',
	ai_agent_mailbox: 'AI agent mailbox',
	mail_server_did_not_respond: 'Mail server did not respond'
};

function humanize(value: string): string {
	if (!value) return '';
	const words = value.replace(/[_-]+/g, ' ').trim();
	return words.charAt(0).toUpperCase() + words.slice(1);
}

/** Human-readable sub-status, e.g. mailbox_not_found -> "Mailbox not found". */
export function subStatusLabel(subStatus: string | undefined | null): string {
	if (!subStatus) return '';
	return SUB_STATUS_LABELS[subStatus] ?? humanize(subStatus);
}

export function percent(part: number, total: number): number {
	if (!total) return 0;
	return (part / total) * 100;
}

/**
 * Minimal RFC 4180 CSV parser: handles quoted fields, escaped quotes ("")
 * and newlines inside quotes. The delimiter (comma, semicolon or tab) is
 * detected from the first line. Returns rows of cells.
 */
export function parseCSV(text: string): string[][] {
	const rows: string[][] = [];
	let row: string[] = [];
	let field = '';
	let inQuotes = false;
	const src = text.charCodeAt(0) === 0xfeff ? text.slice(1) : text;
	// Detect delimiter from the first line (ignoring quoted sections)
	const nl = src.search(/\r|\n/);
	const firstLine = (nl < 0 ? src : src.slice(0, nl)).replace(/"[^"]*"/g, '');
	const delimiter = [',', ';', '\t'].reduce((best, d) =>
		firstLine.split(d).length > firstLine.split(best).length ? d : best
	);

	for (let i = 0; i < src.length; i++) {
		const c = src[i];
		if (inQuotes) {
			if (c === '"') {
				if (src[i + 1] === '"') {
					field += '"';
					i++;
				} else {
					inQuotes = false;
				}
			} else {
				field += c;
			}
			continue;
		}
		if (c === '"') {
			inQuotes = true;
		} else if (c === delimiter) {
			row.push(field);
			field = '';
		} else if (c === '\n' || c === '\r') {
			if (c === '\r' && src[i + 1] === '\n') i++;
			row.push(field);
			field = '';
			if (row.some((v) => v.trim() !== '')) rows.push(row);
			row = [];
		} else {
			field += c;
		}
	}
	row.push(field);
	if (row.some((v) => v.trim() !== '')) rows.push(row);
	return rows;
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function looksLikeEmail(value: string): boolean {
	return EMAIL_RE.test(value.trim());
}

/**
 * Pick the most likely email column: a header named like "email",
 * otherwise the column with the most email-looking values.
 */
export function detectEmailColumn(rows: string[][]): { column: number; hasHeader: boolean } {
	if (rows.length === 0) return { column: 0, hasHeader: false };
	const first = rows[0];
	const hasHeader = !first.some((v) => looksLikeEmail(v));
	if (hasHeader) {
		const byName = first.findIndex((h) => /^e-?mail(\s*address)?$/i.test(h.trim()));
		if (byName >= 0) return { column: byName, hasHeader };
		const partial = first.findIndex((h) => /e-?mail/i.test(h));
		if (partial >= 0) return { column: partial, hasHeader };
	}
	const sample = rows.slice(hasHeader ? 1 : 0, 200);
	const width = Math.max(...rows.slice(0, 200).map((r) => r.length));
	let best = 0;
	let bestScore = -1;
	for (let col = 0; col < width; col++) {
		const score = sample.filter((r) => looksLikeEmail(r[col] ?? '')).length;
		if (score > bestScore) {
			best = col;
			bestScore = score;
		}
	}
	return { column: best, hasHeader };
}

/** Extract unique, trimmed, lowercased emails from a column. */
export function extractEmails(rows: string[][], column: number, hasHeader: boolean): string[] {
	const seen = new Set<string>();
	for (const r of rows.slice(hasHeader ? 1 : 0)) {
		const v = (r[column] ?? '').trim().toLowerCase();
		if (v && looksLikeEmail(v)) seen.add(v);
	}
	return [...seen];
}

function csvCell(value: unknown): string {
	const s = value === undefined || value === null ? '' : String(value);
	return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
}

/** Serialize validation results to CSV (ZeroBounce-style columns). */
export function resultsToCSV(results: ValidationResult[]): string {
	const header = [
		'email',
		'status',
		'sub_status',
		'free_email',
		'did_you_mean',
		'account',
		'domain',
		'domain_age_days',
		'smtp_provider',
		'mx_found',
		'mx_record',
		'processed_at'
	];
	const lines = [header.join(',')];
	for (const r of results) {
		lines.push(
			[
				r.address,
				r.status,
				r.sub_status,
				r.free_email,
				r.did_you_mean,
				r.account,
				r.domain,
				r.domain_age_days ?? '',
				r.smtp_provider,
				r.mx_found,
				r.mx_record,
				r.processed_at
			]
				.map(csvCell)
				.join(',')
		);
	}
	return lines.join('\n');
}

export function downloadFile(filename: string, content: string, type = 'text/csv;charset=utf-8') {
	const blob = new Blob([content], { type });
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = filename;
	document.body.appendChild(a);
	a.click();
	a.remove();
	URL.revokeObjectURL(url);
}

export function formatDate(value?: string): string {
	if (!value) return '-';
	return new Date(value).toLocaleString('en-US', {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
		hour: 'numeric',
		minute: '2-digit'
	});
}
