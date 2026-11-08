<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { acceptInvitation, register, validateInvitation } from '$lib/api';
	import { authStore } from '$lib/stores/auth.svelte';
	import { loginPath } from '$lib/auth';
	import { onMount } from 'svelte';
	import { Card, Input, Button, Alert, Spinner, Badge } from '$lib/components/ui';
	import { ArrowLeft, AlertTriangle } from '@lucide/svelte';

	// Outlet is invite-only: an account starts from an invitation link, and the
	// invitation decides the email, the brand and the role. Someone who already
	// has an account opens the same link signed in and accepts it.
	let formData = $state({
		firstName: '',
		lastName: '',
		phone: '',
		password: '',
		confirmPassword: ''
	});

	let loading = $state(false);
	let error = $state('');
	let invitationToken = $state('');
	let invitationValid = $state(false);
	let invitationLoading = $state(true);
	let email = $state('');
	let brandName = $state('');
	let role = $state('');
	let hasAccount = $state(false);
	let signedInAsInvitee = $derived(
		authStore.isAuthenticated && authStore.user?.email?.toLowerCase() === email.toLowerCase()
	);

	onMount(async () => {
		const token = $page.url.searchParams.get('token');
		if (!token) {
			error = 'Outlet is invite-only. Ask a brand admin for an invitation.';
			invitationLoading = false;
			return;
		}

		invitationToken = token;

		try {
			const invitation = await validateInvitation({ token });
			if (invitation.valid) {
				invitationValid = true;
				email = invitation.email;
				brandName = invitation.brand_name;
				role = invitation.role;
				hasAccount = invitation.has_account;
			} else {
				error = invitation.message || "This invitation isn't valid.";
			}
		} catch (err: any) {
			console.error('Failed to validate invitation:', err);
			error = 'Could not check this invitation. Try again.';
		} finally {
			invitationLoading = false;
		}
	});

	async function handleAccept() {
		loading = true;
		error = '';
		try {
			await acceptInvitation({ token: invitationToken });
			goto('/');
		} catch (err: any) {
			console.error('Accept invitation error:', err);
			error = err.message || 'Could not accept the invitation. Try again.';
		} finally {
			loading = false;
		}
	}

	async function handleRegister(e: Event) {
		e.preventDefault();
		loading = true;
		error = '';

		if (formData.password !== formData.confirmPassword) {
			error = 'The passwords do not match.';
			loading = false;
			return;
		}

		try {
			const response = await register({
				token: invitationToken,
				password: formData.password,
				first_name: formData.firstName,
				last_name: formData.lastName,
				phone: formData.phone || undefined
			});

			if (response.token && response.user) {
				authStore.setSession(response.token, response.user);
				goto('/');
			} else {
				error = 'Something went wrong. Try again.';
			}
		} catch (err: any) {
			console.error('Registration error:', err);
			error = err.message || 'Could not create your account. Try again.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Create Account - Outlet</title>
</svelte:head>

<header class="auth-header">
	<h1 class="auth-title">Create your account</h1>
</header>

<Card class="auth-card">
	{#if invitationLoading}
		<div class="auth-status">
			<div class="auth-status-icon">
				<Spinner size={32} class="text-primary" />
			</div>
			<p class="auth-status-text">Checking your invitation...</p>
		</div>
	{:else if !invitationValid}
		<div class="space-y-5">
			<Alert type="warning" title="Invitation needed" icon={AlertTriangle}>
				<p>{error}</p>
			</Alert>
			<div class="text-center">
				<a href="/auth/login" class="auth-link">Return to sign in</a>
			</div>
		</div>
	{:else if hasAccount}
		<div class="space-y-5">
			{#if error}
				<Alert type="error" title="Could not accept the invitation">
					<p>{error}</p>
				</Alert>
			{/if}
			<div>
				<span class="form-label">Invitation</span>
				<div class="invitation-summary">
					<div class="flex items-center gap-2">
						<span class="text-sm font-medium">{brandName}</span>
						<Badge variant="info" size="sm">{role}</Badge>
					</div>
					<p class="auth-hint">You've been invited to join this brand on Outlet as {email}.</p>
				</div>
			</div>
			{#if signedInAsInvitee}
				<Button type="primary" size="lg" onclick={handleAccept} {loading} disabled={loading} class="w-full justify-center">
					{loading ? 'Joining...' : `Join ${brandName}`}
				</Button>
			{:else}
				<Alert type="info" title="Sign in to accept">
					<p>{email} already has an Outlet account. Sign in as {email} and you come back here to join.</p>
				</Alert>
				<Button href={loginPath($page.url.pathname + $page.url.search)} type="primary" size="lg" class="w-full justify-center">
					Sign in
				</Button>
			{/if}
		</div>
	{:else}
		{#if error}
			<Alert type="error" title="Could not create your account" class="mb-5">
				<p>{error}</p>
			</Alert>
		{/if}

		<form class="auth-form" onsubmit={handleRegister}>
			<div>
				<span class="form-label">Invitation</span>
				<div class="invitation-summary">
					<div class="flex items-center gap-2">
						<span class="text-sm font-medium">{brandName}</span>
						<Badge variant="info" size="sm">{role}</Badge>
					</div>
					<p class="auth-hint">You've been invited to join this brand on Outlet.</p>
				</div>
			</div>

			<div>
				<label for="email" class="form-label">Email address</label>
				<Input id="email" type="email" autocomplete="email" value={email} readonly />
			</div>

			<div class="grid grid-cols-2 gap-3">
				<div>
					<label for="first-name" class="form-label">First name</label>
					<Input id="first-name" type="text" autocomplete="given-name" bind:value={formData.firstName} />
				</div>
				<div>
					<label for="last-name" class="form-label">Last name</label>
					<Input id="last-name" type="text" autocomplete="family-name" bind:value={formData.lastName} />
				</div>
			</div>

			<div>
				<label for="phone" class="form-label">
					Phone number <span class="font-normal opacity-60">(optional)</span>
				</label>
				<Input id="phone" type="tel" autocomplete="tel" bind:value={formData.phone} />
			</div>

			<div>
				<label for="password" class="form-label">Password</label>
				<Input id="password" type="password" autocomplete="new-password" reveal bind:value={formData.password} />
			</div>

			<div>
				<label for="confirm-password" class="form-label">Confirm password</label>
				<Input id="confirm-password" type="password" autocomplete="new-password" reveal bind:value={formData.confirmPassword} />
			</div>

			<Button
				htmlType="submit"
				type="primary"
				size="lg"
				{loading}
				disabled={loading}
				class="mt-2 w-full justify-center"
			>
				{loading ? 'Creating account...' : 'Create account'}
			</Button>
		</form>
	{/if}
</Card>

<p class="auth-footer">
	<a href="/" class="auth-back">
		<ArrowLeft class="h-4 w-4" />
		Back to home
	</a>
</p>
