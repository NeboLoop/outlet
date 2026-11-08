/**
 * Client-side authentication utilities
 * Handles JWT token management and user state
 */

/**
 * Get JWT token from localStorage
 */
export function getToken(): string | null {
	if (typeof window === 'undefined') return null;
	return localStorage.getItem('auth_token');
}

/**
 * Set JWT token in localStorage
 */
export function setToken(token: string): void {
	if (typeof window === 'undefined') return;
	localStorage.setItem('auth_token', token);
}

/**
 * Remove JWT token from localStorage
 */
export function removeToken(): void {
	if (typeof window === 'undefined') return;
	localStorage.removeItem('auth_token');
}

/**
 * Decode JWT token to get user info
 */
export function decodeToken(token: string): any {
	try {
		const base64Url = token.split('.')[1];
		const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
		const jsonPayload = decodeURIComponent(
			atob(base64)
				.split('')
				.map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
				.join('')
		);
		return JSON.parse(jsonPayload);
	} catch (error) {
		console.error('Failed to decode token:', error);
		return null;
	}
}

/**
 * Get current user from JWT token
 */
export function getCurrentUser(): any {
	const token = getToken();
	if (!token) return null;
	return decodeToken(token);
}

/**
 * Check if user is authenticated
 */
export function isAuthenticated(): boolean {
	const token = getToken();
	if (!token) return false;

	const user = decodeToken(token);
	if (!user || !user.exp) return false;

	// Check if token is expired
	const now = Math.floor(Date.now() / 1000);
	return user.exp > now;
}

/**
 * Check if user has admin role
 */
export function isAdmin(): boolean {
	const user = getCurrentUser();
	return user?.role === 'admin' || user?.role === 'super_admin';
}

/**
 * Logout - remove token and redirect to login
 */
export function logout(): void {
	removeToken();
	if (typeof window !== 'undefined') {
		window.location.href = '/auth/login';
	}
}

/**
 * The login page, asked to return to `returnTo` (a path in this app) after
 * signing in.
 */
export function loginPath(returnTo: string): string {
	const back = returnPath(returnTo);
	return back === '/' ? '/auth/login' : `/auth/login?redirect=${encodeURIComponent(back)}`;
}

/**
 * Where to go after signing in: the `redirect` value when it is a path in
 * this app, otherwise home. Another site's address is never followed.
 */
export function returnPath(redirect: string | null | undefined): string {
	if (!redirect || !redirect.startsWith('/') || redirect.startsWith('//') || redirect.startsWith('/\\')) {
		return '/';
	}
	if (redirect.startsWith('/auth/login')) return '/';
	return redirect;
}
