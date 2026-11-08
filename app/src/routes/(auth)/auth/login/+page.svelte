<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { login } from '$lib/api';
	import { returnPath } from '$lib/auth';
	import { authStore } from '$lib/stores/auth.svelte';
	import { onDestroy } from 'svelte';
	import { Card, Input, Button, Alert } from '$lib/components/ui';
	import { AlertTriangle } from '@lucide/svelte';

	let email = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state('');
	let blockedUntil = $state<Date | null>(null);
	let retrySeconds = $state(0);
	let countdownInterval: ReturnType<typeof setInterval> | null = null;

	function startCountdown(seconds: number) {
		retrySeconds = seconds;
		if (countdownInterval) clearInterval(countdownInterval);
		countdownInterval = setInterval(() => {
			retrySeconds--;
			if (retrySeconds <= 0) {
				if (countdownInterval) clearInterval(countdownInterval);
				blockedUntil = null;
				error = '';
			}
		}, 1000);
	}

	function formatTime(seconds: number): string {
		const mins = Math.floor(seconds / 60);
		const secs = seconds % 60;
		if (mins > 0) {
			return `${mins}m ${secs}s`;
		}
		return `${secs}s`;
	}

	async function handleLogin(e: Event) {
		e.preventDefault();
		if (blockedUntil && new Date() < blockedUntil) {
			return;
		}

		loading = true;
		error = '';
		blockedUntil = null;

		try {
			const response = await login({ email, password });

			if (response.token && response.user) {
				authStore.setSession(response.token, response.user);
				goto(returnPath($page.url.searchParams.get('redirect')));
			} else {
				error = 'Invalid response from server';
			}
		} catch (err: any) {
			console.error('Login error:', err);
			if (err.blocked_until) {
				blockedUntil = new Date(err.blocked_until);
				const retryAfter = err.retry_after || Math.ceil((blockedUntil.getTime() - Date.now()) / 1000);
				startCountdown(retryAfter);
				error = err.error || 'Too many failed attempts';
			} else {
				error = err.message || 'Login failed. Please check your credentials.';
			}
		} finally {
			loading = false;
		}
	}

	onDestroy(() => {
		if (countdownInterval) clearInterval(countdownInterval);
	});
</script>

<svelte:head>
	<title>Login - Outlet</title>
</svelte:head>

<header class="auth-header">
	<h1 class="auth-title">Sign in to your account</h1>
</header>

<Card class="auth-card">
	{#if blockedUntil && retrySeconds > 0}
		<Alert type="warning" title="Account temporarily locked" icon={AlertTriangle} class="mb-5">
			<p>Try again in <span class="font-mono font-semibold">{formatTime(retrySeconds)}</span></p>
		</Alert>
	{:else if error}
		<Alert type="error" title="Login failed" class="mb-5">
			<p>{error}</p>
		</Alert>
	{/if}

	<form class="auth-form" onsubmit={handleLogin}>
		<div>
			<label for="email" class="form-label">Email</label>
			<Input
				id="email"
				type="email"
				autocomplete="email"
				placeholder="you@example.com"
				bind:value={email}
				disabled={blockedUntil !== null && retrySeconds > 0}
			/>
		</div>

		<div>
			<div class="auth-label-row">
				<label for="password" class="form-label">Password</label>
				<a href="/auth/forgot-password" class="auth-link">Forgot your password?</a>
			</div>
			<Input
				id="password"
				type="password"
				autocomplete="current-password"
				reveal
				placeholder="Enter your password"
				bind:value={password}
				disabled={blockedUntil !== null && retrySeconds > 0}
			/>
		</div>

		<Button
			htmlType="submit"
			type="primary"
			size="lg"
			{loading}
			disabled={loading || (blockedUntil !== null && retrySeconds > 0)}
			class="mt-2 w-full justify-center"
		>
			{#if blockedUntil && retrySeconds > 0}
				Locked ({formatTime(retrySeconds)})
			{:else if loading}
				Signing in...
			{:else}
				Sign in
			{/if}
		</Button>
	</form>
</Card>
