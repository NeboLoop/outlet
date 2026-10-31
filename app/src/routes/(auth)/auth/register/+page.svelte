<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { acceptInvitation, register, validateInvitation } from '$lib/api';
	import { authStore } from '$lib/stores/auth.svelte';
	import { onMount } from 'svelte';
	import { Card, Input, Button, Alert, Checkbox, Spinner, Badge } from '$lib/components/ui';
	import { ArrowLeft, AlertTriangle } from '@lucide/svelte';

	// Outlet is invite-only: an account starts from an invitation link, and the
	// invitation decides the email, the brand and the role. Someone who already
	// has an account opens the same link signed in and accepts it.
	let formData = $state({
		firstName: '',
		lastName: '',
		phone: '',
		password: '',
		confirmPassword: '',
		agreeToTerms: false
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

<div class="space-y-8">
	<div>
		<a href="/" class="inline-flex items-center text-sm text-text-muted hover:text-text">
			<ArrowLeft class="w-4 h-4 mr-2" />
			Back to home
		</a>
	</div>

	<div class="text-center">
		<h1 class="text-4xl font-bold text-text">Outlet</h1>
		<h2 class="mt-6 text-2xl font-semibold text-text">Create your account</h2>
	</div>

	<Card>
		{#if invitationLoading}
			<div class="text-center py-8">
				<div class="mx-auto flex items-center justify-center h-12 w-12">
					<Spinner size={32} class="text-primary" />
				</div>
				<p class="mt-4 text-sm text-text-muted">Checking your invitation...</p>
			</div>
		{:else if !invitationValid}
			<div class="text-center py-8 space-y-4">
				<Alert type="warning" title="Invitation needed" icon={AlertTriangle}>
					<p>{error}</p>
				</Alert>
				<div class="pt-4">
					<a href="/auth/login" class="text-sm font-medium text-primary hover:underline">
						Return to sign in
					</a>
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
					<div class="px-4 py-3 bg-bg-secondary border border-border rounded-xl">
						<div class="flex items-center gap-2">
							<span class="text-sm text-text">{brandName}</span>
							<Badge variant="info">{role}</Badge>
						</div>
						<p class="text-xs text-text-muted mt-1">You've been invited to join this brand on Outlet as {email}.</p>
					</div>
				</div>
				{#if signedInAsInvitee}
					<Button type="primary" size="lg" onclick={handleAccept} disabled={loading} class="w-full justify-center">
						{loading ? 'Joining...' : `Join ${brandName}`}
					</Button>
				{:else}
					<Alert type="info" title="Sign in to accept">
						<p>{email} already has an Outlet account. Sign in as {email}, then open the link in your invitation email again.</p>
					</Alert>
					<div class="text-center">
						<a href="/auth/login" class="text-sm font-medium text-primary hover:underline">Sign in</a>
					</div>
				{/if}
			</div>
		{:else}
			{#if error}
				<Alert type="error" title="Could not create your account">
					<p>{error}</p>
				</Alert>
				<div class="mt-6"></div>
			{/if}

			<form class="space-y-5" onsubmit={handleRegister}>
				<div>
					<span class="form-label">Invitation</span>
					<div class="px-4 py-3 bg-bg-secondary border border-border rounded-xl">
						<div class="flex items-center gap-2">
							<span class="text-sm text-text">{brandName}</span>
							<Badge variant="info">{role}</Badge>
						</div>
						<p class="text-xs text-text-muted mt-1">You've been invited to join this brand on Outlet.</p>
					</div>
				</div>

				<div>
					<label for="email" class="form-label">Email address</label>
					<Input id="email" type="email" value={email} readonly />
				</div>

				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="first-name" class="form-label">First name</label>
						<Input id="first-name" type="text" bind:value={formData.firstName} />
					</div>
					<div>
						<label for="last-name" class="form-label">Last name</label>
						<Input id="last-name" type="text" bind:value={formData.lastName} />
					</div>
				</div>

				<div>
					<label for="phone" class="form-label">
						Phone number <span class="text-text-muted">(optional)</span>
					</label>
					<Input id="phone" type="tel" bind:value={formData.phone} />
				</div>

				<div>
					<label for="password" class="form-label">Password</label>
					<Input id="password" type="password" bind:value={formData.password} />
				</div>

				<div>
					<label for="confirm-password" class="form-label">Confirm password</label>
					<Input id="confirm-password" type="password" bind:value={formData.confirmPassword} />
				</div>

				<div class="flex items-start gap-3">
					<Checkbox bind:checked={formData.agreeToTerms} />
					<span class="text-sm text-text">
						I agree to the
						<a href="/terms" class="text-primary hover:underline">Terms of Service</a>
						and
						<a href="/privacy" class="text-primary hover:underline">Privacy Policy</a>
					</span>
				</div>

				<Button
					htmlType="submit"
					type="primary"
					size="lg"
					disabled={loading || !formData.agreeToTerms}
					class="w-full justify-center"
				>
					{loading ? 'Creating account...' : 'Create account'}
				</Button>
			</form>
		{/if}
	</Card>
</div>
