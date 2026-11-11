<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { signInWithEmail } from '$lib/api';
	import { authStore } from '$lib/stores/auth.svelte';
	import { Card, Button, Alert } from '$lib/components/ui';
	import { ArrowLeft } from '@lucide/svelte';

	// The emailed sign-in link opens this page. It signs in when the person
	// presses the button, not on load: mail scanners that open links would
	// otherwise use it up before the person gets to it.
	const token = $page.url.searchParams.get('token') ?? '';
	let loading = $state(false);
	let error = $state(token ? '' : 'This sign-in link is missing its token. Send yourself a new one.');

	async function handleSignIn() {
		loading = true;
		error = '';
		try {
			const response = await signInWithEmail({ token });
			authStore.setSession(response.token, response.user);
			goto('/');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Sign-in failed. Please try again.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Sign In - Outlet</title>
</svelte:head>

<header class="auth-header">
	<h1 class="auth-title">Sign in to Outlet</h1>
	<p class="auth-subtitle">Continue to finish signing in from your email link.</p>
</header>

<Card class="auth-card">
	{#if error}
		<Alert type="error" title="Sign-in failed" class="mb-5">
			<p>{error}</p>
		</Alert>
	{/if}

	{#if token}
		<Button
			type="primary"
			size="lg"
			{loading}
			disabled={loading}
			class="w-full justify-center"
			onclick={handleSignIn}
		>
			{loading ? 'Signing in...' : 'Continue'}
		</Button>
	{/if}

	{#if error}
		<p class="auth-footer">
			<a href="/auth/login" class="auth-link">Send a new sign-in link</a>
		</p>
	{/if}
</Card>

<p class="auth-footer">
	<a href="/auth/login" class="auth-back">
		<ArrowLeft class="h-4 w-4" />
		Back to login
	</a>
</p>
