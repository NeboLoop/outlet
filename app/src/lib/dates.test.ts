import { describe, it, expect } from 'vitest';
import { formatDate, parseDate } from './dates';

// Run 7 hours behind UTC, the way the bug showed: a UTC time read as local
// time was 7 hours off. Node applies a TZ change at once.
process.env.TZ = 'America/Phoenix';

describe('formatDate', () => {
	it('reads a time without a zone as UTC', () => {
		const utc = new Date(Date.UTC(2026, 10, 8, 19, 14));
		const expected = utc.toLocaleString('en-US', { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
		expect(formatDate('2026-11-08 19:14:00', 'short')).toBe(expected);
		expect(formatDate('2026-11-08T19:14:00', 'short')).toBe(expected);
		expect(formatDate('2026-11-08T19:14:00Z', 'short')).toBe(expected);
	});

	it('keeps a zone the value names', () => {
		const utc = new Date(Date.UTC(2026, 10, 8, 19, 14));
		const expected = utc.toLocaleString('en-US', { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
		expect(formatDate('2026-11-08T12:14:00-07:00', 'short')).toBe(expected);
	});

	it('shows the viewer\'s local time', () => {
		expect(formatDate('2026-11-08 19:14:00', 'short')).toBe('Nov 8, 12:14 PM');
	});

	it('shows a bare date as that calendar day', () => {
		expect(formatDate('2026-11-08')).toBe('Nov 8, 2026');
		expect(formatDate('2026-11-08', 'day')).toBe('Nov 8');
	});

	it('returns empty for no value and the value when it is not a time', () => {
		expect(formatDate('')).toBe('');
		expect(formatDate(null)).toBe('');
		expect(formatDate(undefined)).toBe('');
		expect(formatDate('soon')).toBe('soon');
	});

	it('formats each style', () => {
		expect(formatDate('2026-11-08 19:14:00', 'date')).toMatch(/^Nov \d{1,2}, 2026$/);
		expect(formatDate('2026-11-08 19:14:00', 'datetime')).toMatch(/^Nov \d{1,2}, 2026, \d{1,2}:14\s[AP]M$/);
		expect(formatDate('2026-11-08 19:14:00', 'day')).toMatch(/^Nov \d{1,2}$/);
	});
});

describe('parseDate', () => {
	it('reads a time without a zone as UTC', () => {
		expect(parseDate('2026-11-08 19:14:00')?.toISOString()).toBe('2026-11-08T19:14:00.000Z');
		expect(parseDate('nope')).toBeNull();
		expect(parseDate(undefined)).toBeNull();
	});
});
