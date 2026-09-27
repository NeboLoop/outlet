<script lang="ts">
	import {
		Code,
		Eye,
		FileText,
		Bold,
		Italic,
		Underline,
		List,
		ListOrdered,
		Link,
		AlignLeft,
		AlignCenter,
		AlignRight
	} from '@lucide/svelte';

	interface Variable {
		name: string;
		label?: string;
		required?: boolean;
	}

	let {
		value = $bindable(''),
		variables = [],
		placeholder = 'Start typing your email...',
		class: extraClass = '',
		showVariableInserts = true,
		onInsertVariable
	}: {
		value?: string;
		variables?: Variable[];
		placeholder?: string;
		class?: string;
		showVariableInserts?: boolean;
		onInsertVariable?: (fn: (variable: string) => void) => void;
	} = $props();

	// Expose insertVariable function to parent via callback
	$effect(() => {
		if (onInsertVariable) {
			onInsertVariable(insertVariable);
		}
	});

	// Editor mode: 'visual' | 'html' | 'text'
	let editorMode = $state<'visual' | 'html' | 'text'>('visual');
	let plainTextBody = $state('');
	let shadowHost = $state<HTMLDivElement | null>(null);
	let shadowRoot: ShadowRoot | null = null;
	let editableDiv: HTMLDivElement | null = null;

	// Initialize shadow DOM for style isolation
	function initShadowEditor(host?: HTMLDivElement | null) {
		const hostEl = host || shadowHost;
		if (!hostEl) return;

		if (!shadowRoot) {
			shadowRoot = hostEl.attachShadow({ mode: 'open' });
		}

		// Base editor styles - minimal, let email's own styles take over
		const baseStyles = `
			<style>
				:host { display: block; height: 100%; }
				#editor-root {
					height: 100%;
					padding: 24px;
					overflow-y: auto;
					font-family: Arial, sans-serif;
					font-size: 14px;
					line-height: 1.6;
					color: #333;
				}
				#editor-root:focus { outline: none; }
			</style>
		`;

		const content = value || `<p>${placeholder}</p>`;

		// Put content directly in editable div so email's styles apply
		shadowRoot.innerHTML = baseStyles + `<div id="editor-root" contenteditable="true">${content}</div>`;

		editableDiv = shadowRoot.querySelector('#editor-root');
		if (editableDiv) {
			editableDiv.addEventListener('input', syncWysiwygToHtml);
		}
	}

	// WYSIWYG formatting
	function execFormat(command: string, formatValue?: string) {
		document.execCommand(command, false, formatValue);
		editableDiv?.focus();
		syncWysiwygToHtml();
	}

	function syncWysiwygToHtml() {
		if (editableDiv) {
			value = editableDiv.innerHTML;
		}
	}

	function syncHtmlToWysiwyg() {
		if (editableDiv && editorMode === 'visual') {
			editableDiv.innerHTML = value || `<p>${placeholder}</p>`;
		}
	}

	function insertLink() {
		const url = prompt('Enter URL:');
		if (url) {
			execFormat('createLink', url);
		}
	}

	function insertVariable(variable: string) {
		if (editorMode === 'visual' && editableDiv) {
			// Get selection in shadow DOM
			const selection = (shadowRoot as unknown as { getSelection?: () => Selection | null })?.getSelection?.() || document.getSelection();
			if (selection && selection.rangeCount > 0) {
				const range = selection.getRangeAt(0);
				// Check if selection is inside our editor
				if (editableDiv.contains(range.commonAncestorContainer)) {
					range.deleteContents();
					range.insertNode(document.createTextNode(variable));
					range.collapse(false);
					syncWysiwygToHtml();
					return;
				}
			}
			// Fallback: append at end
			editableDiv.focus();
			document.execCommand('insertText', false, variable);
			syncWysiwygToHtml();
		} else if (editorMode === 'html') {
			// Insert at cursor in textarea
			const textarea = document.getElementById('email-editor-html') as HTMLTextAreaElement;
			if (textarea) {
				const start = textarea.selectionStart;
				const end = textarea.selectionEnd;
				const text = textarea.value;
				textarea.value = text.substring(0, start) + variable + text.substring(end);
				textarea.selectionStart = textarea.selectionEnd = start + variable.length;
				value = textarea.value;
				textarea.focus();
			}
		} else if (editorMode === 'text') {
			// Insert at cursor in plain text textarea
			const textarea = document.getElementById('email-editor-text') as HTMLTextAreaElement;
			if (textarea) {
				const start = textarea.selectionStart;
				const end = textarea.selectionEnd;
				const text = textarea.value;
				textarea.value = text.substring(0, start) + variable + text.substring(end);
				textarea.selectionStart = textarea.selectionEnd = start + variable.length;
				plainTextBody = textarea.value;
				value = plainTextToHtml(plainTextBody);
				textarea.focus();
			}
		}
	}

	// Svelte action to init shadow DOM on mount
	function initShadowOnMount(node: HTMLDivElement) {
		shadowHost = node;
		initShadowEditor(node);
		return {
			destroy() {
				shadowRoot = null;
				editableDiv = null;
			}
		};
	}

	function htmlToPlainText(html: string): string {
		if (!html) return '';
		return html
			.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '')
			.replace(/<head[^>]*>[\s\S]*?<\/head>/gi, '')
			.replace(/<br\s*\/?>/gi, '\n')
			.replace(/<\/p>/gi, '\n\n')
			.replace(/<\/div>/gi, '\n')
			.replace(/<\/li>/gi, '\n')
			.replace(/<[^>]*>/g, '')
			.replace(/&nbsp;/g, ' ')
			.replace(/&amp;/g, '&')
			.replace(/&lt;/g, '<')
			.replace(/&gt;/g, '>')
			.split('\n')
			.map(line => line.trim())
			.join('\n')
			.replace(/\n{3,}/g, '\n\n')
			.trim();
	}

	function plainTextToHtml(text: string): string {
		if (!text) return '';
		return text
			.split('\n\n')
			.map(para => `<p>${para.replace(/\n/g, '<br>')}</p>`)
			.join('\n');
	}

	function switchToVisual() {
		editorMode = 'visual';
		setTimeout(initShadowEditor, 0);
	}

	function switchToHtml() {
		syncWysiwygToHtml();
		editorMode = 'html';
	}

	function switchToText() {
		syncWysiwygToHtml();
		plainTextBody = htmlToPlainText(value);
		editorMode = 'text';
	}
</script>

<div class="email-editor flex flex-col h-full {extraClass}">
	<!-- Editor Mode Tabs & Variables -->
	<div class="flex items-center justify-between gap-4 mb-4 flex-shrink-0">
		<div class="flex items-center gap-4">
			<div class="flex rounded-lg border border-border overflow-hidden">
				<button
					type="button"
					onclick={switchToVisual}
					class="px-4 py-2 text-sm font-medium flex items-center gap-2 {editorMode === 'visual' ? 'bg-primary text-white' : 'bg-bg text-text hover:bg-bg-secondary'}"
				>
					<Eye size={16} />
					Visual
				</button>
				<button
					type="button"
					onclick={switchToHtml}
					class="px-4 py-2 text-sm font-medium flex items-center gap-2 border-l border-border {editorMode === 'html' ? 'bg-primary text-white' : 'bg-bg text-text hover:bg-bg-secondary'}"
				>
					<Code size={16} />
					HTML
				</button>
				<button
					type="button"
					onclick={switchToText}
					class="px-4 py-2 text-sm font-medium flex items-center gap-2 border-l border-border {editorMode === 'text' ? 'bg-primary text-white' : 'bg-bg text-text hover:bg-bg-secondary'}"
				>
					<FileText size={16} />
					Text
				</button>
			</div>
			<span class="text-xs text-text-muted">
				{editorMode === 'visual' ? 'Edit with formatting toolbar' : editorMode === 'html' ? 'Edit raw HTML' : 'Edit as plain text'}
			</span>
		</div>

		{#if showVariableInserts && variables.length > 0}
			<div class="flex items-center gap-2 flex-wrap">
				<span class="text-xs text-text-muted">Insert:</span>
				{#each variables as variable}
					<button
						type="button"
						onclick={() => insertVariable(`{{${variable.name}}}`)}
						class="px-2 py-1 text-xs font-mono rounded transition-colors {variable.required ? 'bg-yellow-100 hover:bg-yellow-200 text-yellow-800' : 'bg-bg-secondary hover:bg-border text-text-muted'}"
						title={variable.label || variable.name}
					>{`{{${variable.name}}}`}</button>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Editor Content -->
	<div class="flex-1 min-h-0">
		{#if editorMode === 'visual'}
			<div class="h-full flex flex-col bg-bg rounded-lg border border-border overflow-hidden">
				<!-- WYSIWYG Toolbar -->
				<div class="flex flex-wrap items-center gap-1 p-3 border-b border-border bg-bg-secondary flex-shrink-0">
					<button type="button" onclick={() => execFormat('bold')} class="p-2 rounded hover:bg-border" title="Bold">
						<Bold size={18} />
					</button>
					<button type="button" onclick={() => execFormat('italic')} class="p-2 rounded hover:bg-border" title="Italic">
						<Italic size={18} />
					</button>
					<button type="button" onclick={() => execFormat('underline')} class="p-2 rounded hover:bg-border" title="Underline">
						<Underline size={18} />
					</button>
					<div class="w-px h-6 bg-border mx-2"></div>
					<button type="button" onclick={() => execFormat('insertUnorderedList')} class="p-2 rounded hover:bg-border" title="Bullet List">
						<List size={18} />
					</button>
					<button type="button" onclick={() => execFormat('insertOrderedList')} class="p-2 rounded hover:bg-border" title="Numbered List">
						<ListOrdered size={18} />
					</button>
					<div class="w-px h-6 bg-border mx-2"></div>
					<button type="button" onclick={() => execFormat('justifyLeft')} class="p-2 rounded hover:bg-border" title="Align Left">
						<AlignLeft size={18} />
					</button>
					<button type="button" onclick={() => execFormat('justifyCenter')} class="p-2 rounded hover:bg-border" title="Align Center">
						<AlignCenter size={18} />
					</button>
					<button type="button" onclick={() => execFormat('justifyRight')} class="p-2 rounded hover:bg-border" title="Align Right">
						<AlignRight size={18} />
					</button>
					<div class="w-px h-6 bg-border mx-2"></div>
					<button type="button" onclick={insertLink} class="p-2 rounded hover:bg-border" title="Insert Link">
						<Link size={18} />
					</button>
					<div class="w-px h-6 bg-border mx-2"></div>
					<select
						onchange={(e) => execFormat('formatBlock', e.currentTarget.value)}
						class="text-sm border-border rounded py-1.5 pl-3 pr-8 bg-bg"
					>
						<option value="p">Paragraph</option>
						<option value="h1">Heading 1</option>
						<option value="h2">Heading 2</option>
						<option value="h3">Heading 3</option>
					</select>
				</div>
				<!-- Shadow DOM Editor -->
				<div
					bind:this={shadowHost}
					class="flex-1 bg-white min-h-0"
					use:initShadowOnMount
				></div>
			</div>
		{:else if editorMode === 'html'}
			<textarea
				id="email-editor-html"
				bind:value={value}
				class="h-full w-full rounded-lg border border-border bg-bg text-text p-4 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-primary resize-none"
				placeholder="<p>Your HTML here...</p>"
			></textarea>
		{:else}
			<div class="h-full flex flex-col">
				<textarea
					id="email-editor-text"
					bind:value={plainTextBody}
					oninput={() => { value = plainTextToHtml(plainTextBody); }}
					class="flex-1 w-full rounded-lg border border-border bg-bg text-text p-4 text-sm focus:outline-none focus:ring-2 focus:ring-primary resize-none"
					placeholder="Your plain text here..."
				></textarea>
				<p class="mt-2 text-xs text-text-muted">Plain text will be converted to HTML paragraphs.</p>
			</div>
		{/if}
	</div>
</div>
