import { describe, it, expect } from 'vitest';
import { mount, unmount, flushSync } from 'svelte';
import EmailEditor from './EmailEditor.svelte';

function editorRoot(target: HTMLElement): HTMLElement {
	const host = [...target.querySelectorAll('div')].find((d) => d.shadowRoot);
	return host!.shadowRoot!.querySelector<HTMLElement>('#editor-root')!;
}

describe('EmailEditor placeholder', () => {
	it('shows the placeholder without writing it into the email', () => {
		const target = document.createElement('div');
		const props = $state({ value: '', placeholder: 'Start writing your campaign email...' });
		const app = mount(EmailEditor, { target, props });
		flushSync();

		const root = editorRoot(target);
		expect(root.innerHTML).toBe('');
		expect(root.getAttribute('data-placeholder')).toBe('Start writing your campaign email...');
		expect(props.value).toBe('');
		unmount(app);
	});

	it('keeps typed content and empties a cleared editor', () => {
		const target = document.createElement('div');
		const props = $state({ value: '<p>Hello</p>', placeholder: 'Write here' });
		const app = mount(EmailEditor, { target, props });
		flushSync();

		const root = editorRoot(target);
		expect(root.innerHTML).toBe('<p>Hello</p>');

		// Deleting everything leaves <p><br></p> behind in a browser
		root.innerHTML = '<p><br></p>';
		root.dispatchEvent(new Event('input'));
		expect(root.innerHTML).toBe('');
		expect(props.value).toBe('');
		unmount(app);
	});
});
