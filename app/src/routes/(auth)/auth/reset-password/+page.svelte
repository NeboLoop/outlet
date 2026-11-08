<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { resetPassword } from '$lib/api';
	import { Card, Input, Button, Alert } from '$lib/components/ui';
	import { ArrowLeft, CheckCircle } from '@lucide/svelte';

	let password = $state('');
	let confirmPassword = $state('');
	let submitted = $state(false);
	let loading = $state(false);
	let error = $state('');

	// Read in the browser: the page is prerendered, so the URL has no query
	// string when it is built.
	let token = $state('');
	onMount(() => {
		token = $page.url.searchParams.get('token') ?? '';
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';

		if (!token) {
			error = 'This reset link is incomplete. Request a new one.';
			return;
		}
		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}
		if (password.length < 8) {
			error = 'Password must be at least 8 characters';
			return;
		}

		loading = true;
		try {
			await resetPassword({ token, new_password: password, confirm_password: confirmPassword });
			submitted = true;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Could not reset your password. Please try again.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Reset Password - Outlet</title>
</svelte:head>

<header class="auth-header">
	<h1 class="auth-title">Create new password</h1>
	<p class="auth-subtitle">
		Enter your new password below.
	</p>
</header>

{#if !submitted}
	<Card class="auth-card">
		{#if error}
			<Alert type="error" title="Error" class="mb-5">
				<p>{error}</p>
				{#if !token || error.includes('expired')}
					<p class="mt-2"><a href="/auth/forgot-password" class="auth-link">Request a new link</a></p>
				{/if}
			</Alert>
		{/if}

		<form class="auth-form" onsubmit={handleSubmit}>
			<div>
				<label for="password" class="form-label">New password</label>
				<Input
					id="password"
					type="password"
					autocomplete="new-password"
					reveal
					bind:value={password}
				/>
				<p class="auth-hint">Must be at least 8 characters</p>
			</div>

			<div>
				<label for="confirm-password" class="form-label">Confirm new password</label>
				<Input
					id="confirm-password"
					type="password"
					autocomplete="new-password"
					reveal
					bind:value={confirmPassword}
				/>
			</div>

			<Button
				htmlType="submit"
				type="primary"
				size="lg"
				class="mt-2 w-full justify-center"
				{loading}
				disabled={loading}
			>
				Reset password
			</Button>
		</form>
	</Card>
{:else}
	<Card class="auth-card">
		<div class="auth-status">
			<div class="auth-status-icon bg-success/10">
				<CheckCircle class="h-6 w-6 text-success" />
			</div>
			<div>
				<h2 class="auth-status-title">Password reset successful</h2>
				<p class="auth-status-text">
					Your password has been successfully reset. You can now sign in with your new password.
				</p>
			</div>
			<div class="pt-2">
				<Button
					href="/auth/login"
					type="primary"
					size="lg"
					class="w-full justify-center"
				>
					Continue to login
				</Button>
			</div>
		</div>
	</Card>
{/if}

<p class="auth-footer">
	<a href="/auth/login" class="auth-back">
		<ArrowLeft class="h-4 w-4" />
		Back to login
	</a>
</p>
