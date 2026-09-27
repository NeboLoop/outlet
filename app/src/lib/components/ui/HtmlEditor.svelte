<script lang="ts">
	import {
		Code,
		Eye,
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

	let {
		value = $bindable(''),
		placeholder = 'Start typing...',
		class: extraClass = '',
		minHeight = '300px'
	}: {
		value?: string;
		placeholder?: string;
		class?: string;
		minHeight?: string;
	} = $props();

	// Editor mode: 'visual' | 'html'
	let editorMode = $state<'visual' | 'html'>('visual');
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

		// Base editor styles
		const baseStyles = `
			<style>
				:host { display: block; height: 100%; }
				#editor-root {
					height: 100%;
					min-height: ${minHeight};
					padding: 16px;
					overflow-y: auto;
					font-family: system-ui, -apple-system, sans-serif;
					font-size: 15px;
					line-height: 1.7;
					color: #e5e5e5;
					background: transparent;
				}
				#editor-root:focus { outline: none; }
				#editor-root:empty:before {
					content: attr(data-placeholder);
					color: #666;
				}
				#editor-root p { margin: 0 0 1em 0; }
				#editor-root h1 { font-size: 1.75em; font-weight: 700; margin: 0 0 0.5em 0; }
				#editor-root h2 { font-size: 1.5em; font-weight: 700; margin: 0 0 0.5em 0; }
				#editor-root h3 { font-size: 1.25em; font-weight: 600; margin: 0 0 0.5em 0; }
				#editor-root ul, #editor-root ol { margin: 0 0 1em 1.5em; padding: 0; }
				#editor-root li { margin-bottom: 0.25em; }
				#editor-root a { color: #CEB96E; text-decoration: underline; }
			</style>
		`;

		const content = value || '';
		const placeholderAttr = content ? '' : `data-placeholder="${placeholder}"`;

		shadowRoot.innerHTML = baseStyles + `<div id="editor-root" contenteditable="true" ${placeholderAttr}>${content}</div>`;

		editableDiv = shadowRoot.querySelector('#editor-root');
		if (editableDiv) {
			editableDiv.addEventListener('input', syncWysiwygToHtml);
			editableDiv.addEventListener('focus', () => {
				if (editableDiv) editableDiv.removeAttribute('data-placeholder');
			});
			editableDiv.addEventListener('blur', () => {
				if (editableDiv && !editableDiv.innerHTML.trim()) {
					editableDiv.setAttribute('data-placeholder', placeholder);
				}
			});
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
			editableDiv.innerHTML = value || '';
		}
	}

	function insertLink() {
		const url = prompt('Enter URL:');
		if (url) {
			execFormat('createLink', url);
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

	function switchToVisual() {
		editorMode = 'visual';
		setTimeout(initShadowEditor, 0);
	}

	function switchToHtml() {
		syncWysiwygToHtml();
		editorMode = 'html';
	}
</script>

<div class="flex flex-col {extraClass}">
	<!-- Editor Mode Tabs -->
	<div class="flex items-center gap-4 mb-3">
		<div class="join">
			<button
				type="button"
				onclick={switchToVisual}
				class="btn btn-sm join-item gap-2 {editorMode === 'visual' ? 'btn-primary' : 'btn-ghost'}"
			>
				<Eye size={14} />
				Visual
			</button>
			<button
				type="button"
				onclick={switchToHtml}
				class="btn btn-sm join-item gap-2 {editorMode === 'html' ? 'btn-primary' : 'btn-ghost'}"
			>
				<Code size={14} />
				HTML
			</button>
		</div>
		<span class="text-xs text-base-content/60">
			{editorMode === 'visual' ? 'Edit with formatting toolbar' : 'Edit raw HTML'}
		</span>
	</div>

	<!-- Editor Content -->
	<div class="flex-1 min-h-0">
		{#if editorMode === 'visual'}
			<div class="h-full flex flex-col bg-base-200 rounded-lg border border-base-300 overflow-hidden">
				<!-- WYSIWYG Toolbar -->
				<div class="flex flex-wrap items-center gap-1 p-2 border-b border-base-300 bg-base-200/50">
					<button type="button" onclick={() => execFormat('bold')} class="btn btn-ghost btn-xs btn-square" title="Bold">
						<Bold size={16} />
					</button>
					<button type="button" onclick={() => execFormat('italic')} class="btn btn-ghost btn-xs btn-square" title="Italic">
						<Italic size={16} />
					</button>
					<button type="button" onclick={() => execFormat('underline')} class="btn btn-ghost btn-xs btn-square" title="Underline">
						<Underline size={16} />
					</button>
					<div class="divider divider-horizontal mx-1 h-5"></div>
					<button type="button" onclick={() => execFormat('insertUnorderedList')} class="btn btn-ghost btn-xs btn-square" title="Bullet List">
						<List size={16} />
					</button>
					<button type="button" onclick={() => execFormat('insertOrderedList')} class="btn btn-ghost btn-xs btn-square" title="Numbered List">
						<ListOrdered size={16} />
					</button>
					<div class="divider divider-horizontal mx-1 h-5"></div>
					<button type="button" onclick={() => execFormat('justifyLeft')} class="btn btn-ghost btn-xs btn-square" title="Align Left">
						<AlignLeft size={16} />
					</button>
					<button type="button" onclick={() => execFormat('justifyCenter')} class="btn btn-ghost btn-xs btn-square" title="Align Center">
						<AlignCenter size={16} />
					</button>
					<button type="button" onclick={() => execFormat('justifyRight')} class="btn btn-ghost btn-xs btn-square" title="Align Right">
						<AlignRight size={16} />
					</button>
					<div class="divider divider-horizontal mx-1 h-5"></div>
					<button type="button" onclick={insertLink} class="btn btn-ghost btn-xs btn-square" title="Insert Link">
						<Link size={16} />
					</button>
					<div class="divider divider-horizontal mx-1 h-5"></div>
					<select
						onchange={(e) => execFormat('formatBlock', e.currentTarget.value)}
						class="select select-bordered select-xs"
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
					class="flex-1 min-h-0"
					style="min-height: {minHeight}"
					use:initShadowOnMount
				></div>
			</div>
		{:else}
			<textarea
				bind:value={value}
				class="textarea textarea-bordered w-full h-full font-mono text-sm resize-none"
				style="min-height: calc({minHeight} + 44px)"
				placeholder="<p>Your HTML here...</p>"
			></textarea>
		{/if}
	</div>
</div>
