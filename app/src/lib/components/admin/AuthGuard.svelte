<script lang="ts">
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { isAuthenticated, getCurrentUser, isAdmin, loginPath } from '$lib/auth';

	interface Props {
		requireAdmin?: boolean;
		redirectTo?: string;
		children: Snippet;
	}

	let { requireAdmin = false, redirectTo, children }: Props = $props();

	let isLoading = $state(true);
	let isAuthorized = $state(false);

	$effect(() => {
		checkAuth();
	});

	function checkAuth() {
		// Check if user is authenticated
		if (!isAuthenticated()) {
			// Sign in, then come back here
			goto(redirectTo ?? loginPath($page.url.pathname + $page.url.search));
			return;
		}

		// Check admin requirement
		if (requireAdmin && !isAdmin()) {
			goto('/'); // Redirect non-admins to home
			return;
		}

		isAuthorized = true;
		isLoading = false;
	}
</script>

{#if isLoading}
	<div class="flex items-center justify-center min-h-screen bg-base-50">
		<div class="text-center">
			<div class="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
			<p class="mt-4 text-text-muted">Loading...</p>
		</div>
	</div>
{:else if isAuthorized}
	{@render children()}
{/if}
