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
 * formatDate shows a server time in the viewer's time zone. It returns ''
 * for no value, and the value itself when it isn't a time.
 */
export function formatDate(value: string | null | undefined, style: DateStyle = 'date'): string {
	if (!value) return '';
	const text = value.trim();
	if (DATE_ONLY.test(text)) {
		const day = new Date(`${text}T00:00:00Z`);
		return day.toLocaleString('en-US', { ...STYLES[style === 'day' ? 'day' : 'date'], timeZone: 'UTC' });
	}
	let iso = text.replace(' ', 'T');
	if (!HAS_ZONE.test(iso)) iso += 'Z';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return value;
	return d.toLocaleString('en-US', STYLES[style]);
}
