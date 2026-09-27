<script lang="ts">
	import { Card, Input, Button, Alert } from '$lib/components/ui';
	import { ArrowLeft, CheckCircle } from '@lucide/svelte';

	let email = $state('');
	let submitted = $state(false);

	function handleSubmit(e: Event) {
		e.preventDefault();
		console.log('Password reset request for:', email);
		submitted = true;
	}
</script>

<svelte:head>
	<title>Forgot Password - Outlet</title>
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
		<h2 class="mt-6 text-2xl font-semibold text-text">Reset your password</h2>
		<p class="mt-2 text-sm text-text-muted">
			Enter your email address and we'll send you a link to reset your password.
		</p>
	</div>

	{#if !submitted}
		<Card>
			<form class="space-y-5" onsubmit={handleSubmit}>
				<div>
					<label for="email" class="form-label">Email address</label>
					<Input
						id="email"
						type="email"
						placeholder="you@example.com"
						bind:value={email}
					/>
				</div>

				<Button
					htmlType="submit"
					type="primary"
					size="lg"
					class="w-full justify-center"
				>
					Send reset link
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
					<h3 class="text-lg font-medium text-text">Check your email</h3>
					<p class="mt-2 text-sm text-text-muted">
						We've sent a password reset link to <span class="font-medium text-text">{email}</span>.
					</p>
					<p class="mt-2 text-sm text-text-muted">
						Please check your inbox and click the link to reset your password.
					</p>
				</div>
				<div class="pt-4">
					<a href="/auth/login" class="text-sm font-medium text-primary hover:underline">
						Return to login
					</a>
				</div>
			</div>
		</Card>
	{/if}
</div>
