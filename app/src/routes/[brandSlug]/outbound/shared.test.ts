import { describe, expect, it } from 'vitest';
import { pageRange } from './shared';

describe('pageRange', () => {
	it('says which rows a page shows', () => {
		expect(pageRange(0, 50, 120)).toBe('1 to 50 of 120');
		expect(pageRange(100, 20, 120)).toBe('101 to 120 of 120');
	});

	it('says when a page shows none', () => {
		expect(pageRange(0, 0, 0)).toBe('0 of 0');
	});
});
