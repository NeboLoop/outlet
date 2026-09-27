import '@testing-library/jest-dom';
import { vi } from 'vitest';

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
	writable: true,
	value: vi.fn().mockImplementation((query) => ({
		matches: false,
		media: query,
		onchange: null,
		addListener: vi.fn(),
		removeListener: vi.fn(),
		addEventListener: vi.fn(),
		removeEventListener: vi.fn(),
		dispatchEvent: vi.fn()
	}))
});

// Mock localStorage with Map-based implementation
class LocalStorageMock {
	private store: Map<string, string>;

	constructor() {
		this.store = new Map();
	}

	getItem(key: string): string | null {
		return this.store.get(key) || null;
	}

	setItem(key: string, value: string): void {
		this.store.set(key, String(value));
	}

	removeItem(key: string): void {
		this.store.delete(key);
	}

	clear(): void {
		this.store.clear();
	}

	get length(): number {
		return this.store.size;
	}

	key(index: number): string | null {
		return Array.from(this.store.keys())[index] || null;
	}
}

// jsdom exposes localStorage as a getter-only property, so define it.
Object.defineProperty(globalThis, 'localStorage', {
	configurable: true,
	value: new LocalStorageMock()
});

// Mock fetch
global.fetch = vi.fn();

// Mock WebSocket
global.WebSocket = vi.fn() as any;

// Mock import.meta.env
if (typeof import.meta.env === 'undefined') {
	(import.meta as any).env = {
		VITE_API_URL: 'http://localhost:8080'
	};
}
