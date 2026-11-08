/**
 * The one way the app shows a time.
 *
 * The server stores times as UTC text, mostly without a zone
 * ("2026-11-08 19:14:00"), which `new Date()` would read as local time. A
 * time without a zone is read here as UTC and shown in the viewer's time
 * zone. A bare date ("2026-11-08") is a calendar day and is shown as that day.
 */

export type DateStyle =
	| 'date' // Nov 8, 2026
	| 'datetime' // Nov 8, 2026, 12:14 PM
	| 'short' // Nov 8, 12:14 PM
	| 'day'; // Nov 8

const STYLES: Record<DateStyle, Intl.DateTimeFormatOptions> = {
	date: { month: 'short', day: 'numeric', year: 'numeric' },
	datetime: { month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' },
	short: { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' },
	day: { month: 'short', day: 'numeric' }
};

const DATE_ONLY = /^\d{4}-\d{2}-\d{2}$/;
const HAS_ZONE = /(Z|[+-]\d{2}:?\d{2})$/i;

/**
 * parseDate reads a server time (UTC when it names no zone) as a Date, or
 * null when it isn't one.
 */
export function parseDate(value: string | null | undefined): Date | null {
	if (!value) return null;
	let iso = value.trim().replace(' ', 'T');
	if (DATE_ONLY.test(iso)) iso += 'T00:00:00';
	if (!HAS_ZONE.test(iso)) iso += 'Z';
	const d = new Date(iso);
	return Number.isNaN(d.getTime()) ? null : d;
}

/**
 * formatDate shows a server time in the viewer's time zone. It returns ''
 * for no value, and the value itself when it isn't a time.
 */
export function formatDate(value: string | null | undefined, style: DateStyle = 'date'): string {
	if (!value) return '';
	const d = parseDate(value);
	if (!d) return value;
	if (DATE_ONLY.test(value.trim())) {
		return d.toLocaleString('en-US', { ...STYLES[style === 'day' ? 'day' : 'date'], timeZone: 'UTC' });
	}
	return d.toLocaleString('en-US', STYLES[style]);
}
