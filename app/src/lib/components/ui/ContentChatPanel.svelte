<script lang="ts">
	import { ArrowUp, Square, Sparkles, RotateCcw, Copy, Check } from '@lucide/svelte';
	import { Markdown } from '$lib/components/ui';

	interface Message {
		id: string;
		role: 'user' | 'assistant';
		content: string;
	}

	interface Props {
		onInsertContent?: (content: string) => void;
		contentContext?: string;
		class?: string;
		hideHeader?: boolean;
	}

	let { onInsertContent, contentContext = '', class: className = '', hideHeader = false }: Props = $props();

	let messages = $state<Message[]>([]);
	let userInput = $state('');
	let isLoading = $state(false);
	let streamingContent = $state('');
	let error = $state('');
	let textareaRef: HTMLTextAreaElement;
	let messagesContainer: HTMLDivElement;
	let copiedId = $state<string | null>(null);

	const quickPrompts = [
		{ label: 'Write intro', prompt: 'Write an engaging introduction paragraph for this content.' },
		{ label: 'Expand section', prompt: 'Expand on the current content with more detail and examples.' },
		{ label: 'Make concise', prompt: 'Rewrite this content to be more concise and impactful.' },
		{ label: 'Add examples', prompt: 'Add relevant examples to illustrate the key points.' },
		{ label: 'SEO optimize', prompt: 'Optimize this content for SEO while keeping it natural.' }
	];

	async function sendMessage() {
		if (!userInput.trim() || isLoading) return;

		const userMessage: Message = {
			id: `user-${Date.now()}`,
			role: 'user',
			content: userInput
		};

		messages = [...messages, userMessage];
		const currentInput = userInput;
		userInput = '';
		isLoading = true;
		error = '';
		streamingContent = '';

		// Reset textarea height
		if (textareaRef) {
			textareaRef.style.height = 'auto';
		}

		try {
			// Build context for the AI
			const systemContext = contentContext
				? `You are helping write content for a blog post or page. Here is the current content for context:\n\n${contentContext}\n\nProvide helpful, well-written content in Markdown format. Be creative but stay on topic.`
				: 'You are a helpful content writing assistant. Provide well-written content in Markdown format.';

			const response = await fetch('/api/ai/generate-content', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					messages: [
						{ role: 'system', content: systemContext },
						...messages.map(m => ({ role: m.role, content: m.content })),
						{ role: 'user', content: currentInput }
					]
				})
			});

			if (!response.ok) {
				throw new Error('Failed to generate content');
			}

			// Handle streaming response
			const reader = response.body?.getReader();
			const decoder = new TextDecoder();

			if (!reader) {
				throw new Error('No response body');
			}

			while (true) {
				const { done, value } = await reader.read();
				if (done) break;

				const chunk = decoder.decode(value, { stream: true });
				streamingContent += chunk;
				scrollToBottom();
			}

			// Add assistant message
			const assistantMessage: Message = {
				id: `assistant-${Date.now()}`,
				role: 'assistant',
				content: streamingContent
			};

			messages = [...messages, assistantMessage];
			streamingContent = '';
		} catch (err: any) {
			console.error('Failed to generate content:', err);
			error = err.message || 'Failed to generate content. Please try again.';
		} finally {
			isLoading = false;
		}
	}

	function handleKeyPress(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			sendMessage();
		}
	}

	function handleInput(e: Event) {
		const target = e.target as HTMLTextAreaElement;
		target.style.height = 'auto';
		target.style.height = Math.min(target.scrollHeight, 120) + 'px';
	}

	function scrollToBottom() {
		if (messagesContainer) {
			messagesContainer.scrollTo({
				top: messagesContainer.scrollHeight,
				behavior: 'smooth'
			});
		}
	}

	function insertContent(content: string) {
		if (onInsertContent) {
			onInsertContent(content);
		}
	}

	async function copyContent(id: string, content: string) {
		try {
			await navigator.clipboard.writeText(content);
			copiedId = id;
			setTimeout(() => {
				copiedId = null;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	}

	function useQuickPrompt(prompt: string) {
		userInput = prompt;
		sendMessage();
	}

	function clearChat() {
		messages = [];
		error = '';
	}
</script>

<div class="flex flex-col h-full bg-gray-50 {className}">
	<!-- Header (optional) -->
	{#if !hideHeader}
		<div class="flex items-center justify-between px-4 py-3 border-b border-gray-200 bg-white">
			<div class="flex items-center gap-2">
				<Sparkles size={18} class="text-blue-600" />
				<span class="font-medium text-gray-900">AI Assistant</span>
			</div>
			{#if messages.length > 0}
				<button
					type="button"
					onclick={clearChat}
					class="text-xs text-gray-500 hover:text-gray-700"
				>
					Clear
				</button>
			{/if}
		</div>
	{/if}

	<!-- Messages Area -->
	<div
		bind:this={messagesContainer}
		class="flex-1 overflow-y-auto p-4 space-y-4"
	>
		{#if messages.length === 0 && !isLoading}
			<!-- Empty state with quick prompts -->
			<div class="text-center py-8">
				<Sparkles size={32} class="mx-auto text-gray-300 mb-3" />
				<p class="text-sm text-gray-500 mb-4">Ask AI to help write your content</p>
				<div class="flex flex-wrap gap-2 justify-center">
					{#each quickPrompts as prompt}
						<button
							type="button"
							onclick={() => useQuickPrompt(prompt.prompt)}
							class="px-3 py-1.5 text-xs bg-white border border-gray-200 rounded-full text-gray-600 hover:bg-gray-50 hover:border-gray-300 transition-colors"
						>
							{prompt.label}
						</button>
					{/each}
				</div>
			</div>
		{:else}
			{#each messages as message (message.id)}
				<div class="flex {message.role === 'user' ? 'justify-end' : 'justify-start'}">
					{#if message.role === 'user'}
						<div class="max-w-[85%] rounded-2xl bg-blue-600 px-4 py-2">
							<p class="text-white text-sm whitespace-pre-wrap">{message.content}</p>
						</div>
					{:else}
						<div class="max-w-[95%] space-y-2">
							<div class="rounded-2xl bg-white border border-gray-200 px-4 py-3 shadow-sm">
								<Markdown content={message.content} class="text-sm" />
							</div>
							<div class="flex items-center gap-1">
								<button
									type="button"
									onclick={() => insertContent(message.content)}
									class="flex items-center gap-1 px-2 py-1 text-xs text-blue-600 hover:bg-blue-50 rounded transition-colors"
								>
									<ArrowUp size={12} />
									Insert
								</button>
								<button
									type="button"
									onclick={() => copyContent(message.id, message.content)}
									class="flex items-center gap-1 px-2 py-1 text-xs text-gray-500 hover:bg-gray-100 rounded transition-colors"
								>
									{#if copiedId === message.id}
										<Check size={12} class="text-green-500" />
									{:else}
										<Copy size={12} />
									{/if}
									Copy
								</button>
							</div>
						</div>
					{/if}
				</div>
			{/each}

			{#if isLoading && streamingContent}
				<div class="flex justify-start">
					<div class="max-w-[95%] rounded-2xl bg-white border border-gray-200 px-4 py-3 shadow-sm">
						<Markdown content={streamingContent} class="text-sm" />
						<span class="animate-pulse">▊</span>
					</div>
				</div>
			{:else if isLoading}
				<div class="flex justify-start">
					<div class="flex gap-1 px-4 py-3">
						<span class="w-2 h-2 bg-blue-400 rounded-full animate-bounce" style="animation-delay: 0ms"></span>
						<span class="w-2 h-2 bg-blue-400 rounded-full animate-bounce" style="animation-delay: 150ms"></span>
						<span class="w-2 h-2 bg-blue-400 rounded-full animate-bounce" style="animation-delay: 300ms"></span>
					</div>
				</div>
			{/if}
		{/if}

		{#if error}
			<div class="rounded-lg bg-red-50 border border-red-200 px-4 py-2">
				<p class="text-sm text-red-600">{error}</p>
			</div>
		{/if}
	</div>

	<!-- Input Area -->
	<div class="border-t border-gray-200 bg-white p-3">
		{#if messages.length > 0}
			<div class="flex flex-wrap gap-1.5 mb-2">
				{#each quickPrompts.slice(0, 3) as prompt}
					<button
						type="button"
						onclick={() => useQuickPrompt(prompt.prompt)}
						class="px-2 py-1 text-xs bg-gray-100 rounded-full text-gray-600 hover:bg-gray-200 transition-colors"
						disabled={isLoading}
					>
						{prompt.label}
					</button>
				{/each}
			</div>
		{/if}

		<div class="flex items-end gap-2">
			<div class="flex-1 relative">
				<textarea
					bind:this={textareaRef}
					bind:value={userInput}
					onkeypress={handleKeyPress}
					oninput={handleInput}
					placeholder="Ask AI to help write..."
					rows="1"
					disabled={isLoading}
					class="w-full resize-none rounded-2xl border border-gray-300 bg-white px-4 py-2.5 pr-12 text-sm text-gray-900 placeholder-gray-400 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
				></textarea>
			</div>
			<button
				type="button"
				onclick={sendMessage}
				disabled={!userInput.trim() || isLoading}
				class="flex-shrink-0 rounded-full p-2.5 transition-colors disabled:cursor-not-allowed disabled:opacity-40 {userInput.trim() && !isLoading ? 'bg-blue-600 text-white hover:bg-blue-700' : 'bg-gray-200 text-gray-400'}"
			>
				{#if isLoading}
					<Square size={16} class="fill-current" />
				{:else}
					<ArrowUp size={16} />
				{/if}
			</button>
		</div>
	</div>
</div>
