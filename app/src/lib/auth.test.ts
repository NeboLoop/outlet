import { describe, it, expect } from 'vitest';
import { loginPath, returnPath } from './auth';

describe('login return', () => {
	it('sends you back to the page you were sent away from', () => {
		const login = loginPath('/acme/campaigns/42?tab=stats');
		expect(login).toBe('/auth/login?redirect=%2Facme%2Fcampaigns%2F42%3Ftab%3Dstats');
		const redirect = new URL(login, 'https://outlet.test').searchParams.get('redirect');
		expect(returnPath(redirect)).toBe('/acme/campaigns/42?tab=stats');
	});

	it('goes home without a page to return to', () => {
		expect(loginPath('/')).toBe('/auth/login');
		expect(returnPath(null)).toBe('/');
		expect(returnPath('')).toBe('/');
	});

	it('never follows another site', () => {
		expect(returnPath('https://evil.example/')).toBe('/');
		expect(returnPath('//evil.example/')).toBe('/');
		expect(returnPath('/\\evil.example/')).toBe('/');
		expect(returnPath('javascript:alert(1)')).toBe('/');
	});

	it('does not return to the login page itself', () => {
		expect(returnPath('/auth/login?redirect=/x')).toBe('/');
	});
});
