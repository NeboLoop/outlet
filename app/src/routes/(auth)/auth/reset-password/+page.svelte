<script lang="ts">
	import { page } from '$app/stores';
	import { Card, Input, Button, Alert } from '$lib/components/ui';
	import { ArrowLeft, CheckCircle } from '@lucide/svelte';

	let password = $state('');
	let confirmPassword = $state('');
	let submitted = $state(false);
	let error = $state('');

	function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';

		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}

		if (password.length < 8) {
			error = 'Password must be at least 8 characters';
			return;
		}

		console.log('Password reset with token:', $page.url.searchParams.get('token'));
		console.log('New password:', password);
		submitted = true;
	}
</script>

<svelte:head>
	<title>Reset Password - Outlet</title>
</svelte:head>

<div class="space-y-8">
	<div>
		<a href="/auth/login" class="inline-flex items-center text-sm text-text-muted hover:text-text">
			<ArrowLeft class="w-4 h-4 mr-2" />
			Back to login
		</a>
	</div>

	<div class="text-center">
		<h1 class="text-4xl font-bold text-text">Outlet</h1>
		<h2 class="mt-6 text-2xl font-semibold text-text">Create new password</h2>
		<p class="mt-2 text-sm text-text-muted">
			Enter your new password below.
		</p>
	</div>

	{#if !submitted}
		<Card>
			{#if error}
				<Alert type="error" title="Error">
					<p>{error}</p>
				</Alert>
				<div class="mt-6"></div>
			{/if}

			<form class="space-y-5" onsubmit={handleSubmit}>
				<div>
					<label for="password" class="form-label">New password</label>
					<Input
						id="password"
						type="password"
						bind:value={password}
					/>
					<p class="mt-1 text-xs text-text-muted">Must be at least 8 characters</p>
				</div>

				<div>
					<label for="confirm-password" class="form-label">Confirm new password</label>
					<Input
						id="confirm-password"
						type="password"
						bind:value={confirmPassword}
					/>
				</div>

				<Button
					htmlType="submit"
					type="primary"
					size="lg"
					class="w-full justify-center"
				>
					Reset password
				</Button>
			</form>
		</Card>
	{:else}
		<Card>
			<div class="text-center space-y-4">
				<div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-success/10">
					<CheckCircle class="h-6 w-6 text-success" />
				</div>
				<div>
					<h3 class="text-lg font-medium text-text">Password reset successful</h3>
					<p class="mt-2 text-sm text-text-muted">
						Your password has been successfully reset. You can now sign in with your new password.
					</p>
				</div>
				<div class="pt-4">
					<Button
						href="/auth/login"
						type="primary"
						size="lg"
						class="justify-center"
					>
						Continue to login
					</Button>
				</div>
			</div>
		</Card>
	{/if}
</div>
