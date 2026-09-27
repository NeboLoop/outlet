<script lang="ts">
	import { marked } from 'marked';
	import {
		Bold,
		Italic,
		Heading1,
		Heading2,
		Heading3,
		List,
		ListOrdered,
		Quote,
		Code,
		Link,
		Image,
		Eye,
		Code2
	} from '@lucide/svelte';

	interface Props {
		value: string;
		placeholder?: string;
		class?: string;
	}

	let { value = $bindable(''), placeholder = 'Start writing...', class: className = '' }: Props = $props();

	let mode = $state<'visual' | 'source'>('visual');
	let textareaRef = $state<HTMLTextAreaElement | null>(null);

	// Configure marked
	marked.setOptions({
		breaks: true,
		gfm: true
	});

	let previewHtml = $derived(marked.parse(value || '') as string);

	function insertMarkdown(prefix: string, suffix: string = '', placeholder: string = '') {
		if (!textareaRef) return;

		const start = textareaRef.selectionStart;
		const end = textareaRef.selectionEnd;
		const selectedText = value.substring(start, end) || placeholder;

		const before = value.substring(0, start);
		const after = value.substring(end);

		value = before + prefix + selectedText + suffix + after;

		// Set cursor position after the inserted text
		setTimeout(() => {
			if (textareaRef) {
				const newPos = start + prefix.length + selectedText.length + suffix.length;
				textareaRef.focus();
				textareaRef.setSelectionRange(newPos, newPos);
			}
		}, 0);
	}

	function insertHeading(level: number) {
		const prefix = '#'.repeat(level) + ' ';
		const start = textareaRef?.selectionStart || 0;

		// Find start of current line
		let lineStart = start;
		while (lineStart > 0 && value[lineStart - 1] !== '\n') {
			lineStart--;
		}

		// Check if line already has a heading
		const lineEnd = value.indexOf('\n', start);
		const line = value.substring(lineStart, lineEnd === -1 ? value.length : lineEnd);
		const headingMatch = line.match(/^#+\s*/);

		if (headingMatch) {
			// Replace existing heading
			const before = value.substring(0, lineStart);
			const after = value.substring(lineStart + headingMatch[0].length);
			value = before + prefix + after;
		} else {
			// Insert heading at line start
			const before = value.substring(0, lineStart);
			const after = value.substring(lineStart);
			value = before + prefix + after;
		}
	}

	function insertList(ordered: boolean) {
		const prefix = ordered ? '1. ' : '- ';
		const start = textareaRef?.selectionStart || 0;

		// Find start of current line
		let lineStart = start;
		while (lineStart > 0 && value[lineStart - 1] !== '\n') {
			lineStart--;
		}

		const before = value.substring(0, lineStart);
		const after = value.substring(lineStart);

		// Add newline before if not at start and previous char isn't newline
		const needsNewline = lineStart > 0 && value[lineStart - 1] !== '\n';
		value = before + (needsNewline ? '\n' : '') + prefix + after;
	}

	function insertBlockquote() {
		const start = textareaRef?.selectionStart || 0;
		const end = textareaRef?.selectionEnd || 0;
		const selectedText = value.substring(start, end);

		if (selectedText) {
			// Quote selected text
			const quotedText = selectedText.split('\n').map(line => '> ' + line).join('\n');
			const before = value.substring(0, start);
			const after = value.substring(end);
			value = before + quotedText + after;
		} else {
			// Insert quote at current position
			insertMarkdown('> ', '', 'Quote');
		}
	}

	function insertCodeBlock() {
		insertMarkdown('\n```\n', '\n```\n', 'code');
	}

	function insertLink() {
		const selectedText = textareaRef ? value.substring(textareaRef.selectionStart, textareaRef.selectionEnd) : '';
		if (selectedText) {
			insertMarkdown('[', '](url)', '');
		} else {
			insertMarkdown('[', '](url)', 'link text');
		}
	}

	function insertImage() {
		insertMarkdown('![', '](image-url)', 'alt text');
	}

	const toolbarButtons = [
		{ icon: Bold, action: () => insertMarkdown('**', '**', 'bold'), title: 'Bold' },
		{ icon: Italic, action: () => insertMarkdown('*', '*', 'italic'), title: 'Italic' },
		{ separator: true },
		{ icon: Heading1, action: () => insertHeading(1), title: 'Heading 1' },
		{ icon: Heading2, action: () => insertHeading(2), title: 'Heading 2' },
		{ icon: Heading3, action: () => insertHeading(3), title: 'Heading 3' },
		{ separator: true },
		{ icon: List, action: () => insertList(false), title: 'Bullet List' },
		{ icon: ListOrdered, action: () => insertList(true), title: 'Numbered List' },
		{ icon: Quote, action: insertBlockquote, title: 'Quote' },
		{ separator: true },
		{ icon: Code, action: () => insertMarkdown('`', '`', 'code'), title: 'Inline Code' },
		{ icon: Code2, action: insertCodeBlock, title: 'Code Block' },
		{ separator: true },
		{ icon: Link, action: insertLink, title: 'Link' },
		{ icon: Image, action: insertImage, title: 'Image' }
	];
</script>

<div class="flex flex-col h-full bg-white rounded-lg border border-gray-200 overflow-hidden {className}">
	<!-- Toolbar -->
	<div class="flex items-center justify-between px-3 py-2 border-b border-gray-200 bg-gray-50">
		<div class="flex items-center gap-1">
			{#each toolbarButtons as btn}
				{#if btn.separator}
					<div class="w-px h-5 bg-gray-300 mx-1"></div>
				{:else}
					<button
						type="button"
						onclick={btn.action}
						class="p-1.5 rounded hover:bg-gray-200 text-gray-600 hover:text-gray-900 transition-colors"
						title={btn.title}
					>
						<btn.icon size={16} />
					</button>
				{/if}
			{/each}
		</div>

		<!-- Mode Toggle -->
		<div class="flex items-center gap-1 bg-gray-200 rounded-md p-0.5">
			<button
				type="button"
				onclick={() => mode = 'visual'}
				class="flex items-center gap-1 px-2.5 py-1 text-xs font-medium rounded transition-colors {mode === 'visual' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-600 hover:text-gray-900'}"
			>
				<Eye size={14} />
				Visual
			</button>
			<button
				type="button"
				onclick={() => mode = 'source'}
				class="flex items-center gap-1 px-2.5 py-1 text-xs font-medium rounded transition-colors {mode === 'source' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-600 hover:text-gray-900'}"
			>
				<Code2 size={14} />
				Source
			</button>
		</div>
	</div>

	<!-- Editor/Preview Area -->
	<div class="flex-1 overflow-hidden">
		{#if mode === 'source'}
			<textarea
				bind:this={textareaRef}
				bind:value
				{placeholder}
				class="w-full h-full p-4 resize-none border-0 focus:outline-none focus:ring-0 font-mono text-sm text-gray-900 bg-white"
			></textarea>
		{:else}
			<!-- Split view: editor on left, preview on right -->
			<div class="flex h-full">
				<div class="flex-1 border-r border-gray-200">
					<textarea
						bind:this={textareaRef}
						bind:value
						{placeholder}
						class="w-full h-full p-4 resize-none border-0 focus:outline-none focus:ring-0 text-sm text-gray-900 bg-white"
					></textarea>
				</div>
				<div class="flex-1 overflow-y-auto p-4 bg-gray-50">
					{#if value}
						<div class="prose prose-sm max-w-none">
							{@html previewHtml}
						</div>
					{:else}
						<p class="text-gray-400 text-sm italic">Preview will appear here...</p>
					{/if}
				</div>
			</div>
		{/if}
	</div>
</div>
