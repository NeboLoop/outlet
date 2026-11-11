<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { login, sendSignInEmail, signInWithEmail } from '$lib/api';
	import { returnPath } from '$lib/auth';
	import { authStore } from '$lib/stores/auth.svelte';
	import { onDestroy } from 'svelte';
	import { Card, Input, Button, Alert } from '$lib/components/ui';
	import { AlertTriangle, MailCheck } from '@lucide/svelte';

	// Sign-in starts with an email: we send a link and a code. A password
	// is the other way in.
	let mode = $state<'email' | 'code' | 'password'>('email');
	let email = $state('');
	let password = $state('');
	let code = $state('');
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

	function choose(next: 'email' | 'password') {
		mode = next;
		error = '';
		code = '';
	}

	async function handleSendEmail(e?: Event) {
		e?.preventDefault();
		loading = true;
		error = '';
		try {
			await sendSignInEmail({ email: email.trim() });
			code = '';
			mode = 'code';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Could not send the sign-in email. Please try again.';
		} finally {
			loading = false;
		}
	}

	async function handleCode(e: Event) {
		e.preventDefault();
		loading = true;
		error = '';
		try {
			const response = await signInWithEmail({ email: email.trim(), code: code.trim() });
			authStore.setSession(response.token, response.user);
			goto(returnPath($page.url.searchParams.get('redirect')));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Sign-in failed. Please try again.';
		} finally {
			loading = false;
		}
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
	{#if mode === 'password'}
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
		<p class="auth-footer">
			<button type="button" class="auth-link" onclick={() => choose('email')}>Email me a sign-in link instead</button>
		</p>
	{:else}
		{#if error}
			<Alert type="error" title="Sign-in failed" class="mb-5">
				<p>{error}</p>
			</Alert>
		{/if}

		{#if mode === 'email'}
			<form class="auth-form" onsubmit={handleSendEmail}>
				<div>
					<label for="email" class="form-label">Email</label>
					<Input id="email" type="email" autocomplete="email" placeholder="you@example.com" required bind:value={email} />
				</div>

				<Button
					htmlType="submit"
					type="primary"
					size="lg"
					{loading}
					disabled={loading}
					class="mt-2 w-full justify-center"
				>
					{loading ? 'Sending...' : 'Email me a sign-in link'}
				</Button>
			</form>
		{:else}
			<div class="auth-status">
				<div class="auth-status-icon bg-success/10">
					<MailCheck class="h-6 w-6 text-success" />
				</div>
				<div>
					<h2 class="auth-status-title">Check your email</h2>
					<p class="auth-status-text">
						If an account uses <span class="font-medium text-base-content">{email.trim()}</span>, we sent it a sign-in link and a 6-digit code.
						Click the link, or enter the code below. Both expire in 15 minutes.
					</p>
				</div>
			</div>

			<form class="auth-form mt-4" onsubmit={handleCode}>
				<div>
					<label for="code" class="form-label">Code</label>
					<Input id="code" type="text" autocomplete="one-time-code" placeholder="123456" required bind:value={code} />
				</div>

				<Button
					htmlType="submit"
					type="primary"
					size="lg"
					{loading}
					disabled={loading}
					class="mt-2 w-full justify-center"
				>
					{loading ? 'Signing in...' : 'Sign in'}
				</Button>
			</form>

			<div class="auth-label-row mt-4">
				<button type="button" class="auth-link" onclick={() => choose('email')}>Use a different email</button>
				<button type="button" class="auth-link" disabled={loading} onclick={() => handleSendEmail()}>Send a new code</button>
			</div>
		{/if}

		<p class="auth-footer">
			<button type="button" class="auth-link" onclick={() => choose('password')}>Use a password instead</button>
		</p>
	{/if}
</Card>
