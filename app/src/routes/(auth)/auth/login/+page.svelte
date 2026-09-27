<script lang="ts">
	import { goto } from '$app/navigation';
	import { login } from '$lib/api';
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
				goto('/');
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

<div class="min-h-screen flex items-center justify-center">
	<div class="w-full max-w-sm space-y-8">
		<div class="text-center">
			<h1 class="text-4xl font-bold text-text">Outlet</h1>
			<p class="mt-3 text-text-muted">Sign in to your account</p>
		</div>

		<Card>
			{#if blockedUntil && retrySeconds > 0}
				<Alert type="warning" title="Account temporarily locked" icon={AlertTriangle}>
					<p>Try again in <span class="font-mono font-semibold">{formatTime(retrySeconds)}</span></p>
				</Alert>
				<div class="mt-6"></div>
			{:else if error}
				<Alert type="error" title="Login failed">
					<p>{error}</p>
				</Alert>
				<div class="mt-6"></div>
			{/if}

			<form class="space-y-5" onsubmit={handleLogin}>
				<div>
					<label for="email" class="form-label">Email</label>
					<Input
						id="email"
						type="email"
						placeholder="you@example.com"
						bind:value={email}
						disabled={blockedUntil !== null && retrySeconds > 0}
					/>
				</div>

				<div>
					<label for="password" class="form-label">Password</label>
					<Input
						id="password"
						type="password"
						placeholder="Enter your password"
						bind:value={password}
						disabled={blockedUntil !== null && retrySeconds > 0}
					/>
				</div>

				<Button
					htmlType="submit"
					type="primary"
					size="lg"
					disabled={loading || (blockedUntil !== null && retrySeconds > 0)}
					class="w-full justify-center"
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

			<div class="mt-6 text-center">
				<a href="/auth/forgot-password" class="text-sm text-primary hover:underline">
					Forgot your password?
				</a>
			</div>
		</Card>
	</div>
</div>
