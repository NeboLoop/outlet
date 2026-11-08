<script lang="ts">
	import { forgotPassword } from '$lib/api';
	import { Card, Input, Button, Alert } from '$lib/components/ui';
	import { ArrowLeft, CheckCircle } from '@lucide/svelte';

	let email = $state('');
	let submitted = $state(false);
	let loading = $state(false);
	let error = $state('');

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			await forgotPassword({ email: email.trim() });
			submitted = true;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Could not send the reset link. Please try again.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Forgot Password - Outlet</title>
</svelte:head>

<header class="auth-header">
	<h1 class="auth-title">Reset your password</h1>
	<p class="auth-subtitle">
		Enter your email address and we'll send you a link to reset your password.
	</p>
</header>

{#if !submitted}
	<Card class="auth-card">
		{#if error}
			<Alert type="error" title="Error" class="mb-5">
				<p>{error}</p>
			</Alert>
		{/if}

		<form class="auth-form" onsubmit={handleSubmit}>
			<div>
				<label for="email" class="form-label">Email address</label>
				<Input
					id="email"
					type="email"
					autocomplete="email"
					placeholder="you@example.com"
					required
					bind:value={email}
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
				Send reset link
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
				<h2 class="auth-status-title">Check your email</h2>
				<p class="auth-status-text">
					If an account uses <span class="font-medium text-base-content">{email}</span>, we've sent it a password reset link.
				</p>
				<p class="auth-status-text">
					The link works once and expires in 1 hour.
				</p>
			</div>
			<div class="pt-2">
				<a href="/auth/login" class="auth-link">Return to login</a>
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
