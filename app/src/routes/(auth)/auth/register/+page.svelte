<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { register, validateInvitation } from '$lib/api';
	import { authStore } from '$lib/stores/auth.svelte';
	import { onMount } from 'svelte';
	import { Card, Input, Button, Alert, Checkbox, Spinner, Badge } from '$lib/components/ui';
	import { ArrowLeft, AlertTriangle } from '@lucide/svelte';

	let formData = $state({
		firstName: '',
		lastName: '',
		email: '',
		phone: '',
		password: '',
		confirmPassword: '',
		userType: 'homeowner',
		agreeToTerms: false
	});

	let loading = $state(false);
	let error = $state('');
	let invitationToken = $state<string | null>(null);
	let invitationType = $state<'homeowner' | 'agent' | 'partner' | null>(null);
	let invitationLoading = $state(false);

	onMount(async () => {
		const token = $page.url.searchParams.get('token');
		if (!token) {
			error = 'Registration requires an invitation. Please contact an administrator.';
			invitationLoading = false;
			return;
		}

		invitationToken = token;
		invitationLoading = true;

		try {
			const invitationData = await validateInvitation({ token });

			if (invitationData.valid) {
				formData.email = invitationData.email;
				formData.firstName = invitationData.first_name || '';
				formData.lastName = invitationData.last_name || '';
				formData.phone = invitationData.phone || '';

				if (invitationData.role === 'partner_manager') {
					invitationType = 'partner';
					formData.userType = 'partner';
				} else if (invitationData.role === 'sales_agent') {
					invitationType = 'agent';
					formData.userType = 'agent';
				} else if (invitationData.role === 'homeowner') {
					invitationType = 'homeowner';
					formData.userType = 'homeowner';
				}
			} else {
				error = invitationData.message || 'Invalid invitation token';
			}
		} catch (err: any) {
			console.error('Failed to validate invitation:', err);
			error = 'Failed to validate invitation. Please try again or contact support.';
		} finally {
			invitationLoading = false;
		}
	});

	async function handleRegister(e: Event) {
		e.preventDefault();
		loading = true;
		error = '';

		if (formData.password !== formData.confirmPassword) {
			error = 'Passwords do not match';
			loading = false;
			return;
		}

		try {
			const response = await register({
				email: formData.email,
				password: formData.password,
				first_name: formData.firstName,
				last_name: formData.lastName,
				phone: formData.phone || undefined,
				token: invitationToken || undefined
			});

			if (response.token && response.user) {
				authStore.setSession(response.token, response.user);

				if (response.user.role === 'sales_agent') {
					goto('/agent');
				} else if (response.user.role === 'partner_manager') {
					goto('/partner');
				} else if (response.user.role === 'homeowner') {
					goto('/homeowner');
				} else if (response.user.role === 'admin') {
					goto('/');
				} else {
					goto('/');
				}
			} else {
				error = 'Invalid response from server';
			}
		} catch (err: any) {
			console.error('Registration error:', err);
			error = err.message || 'Registration failed. Please try again.';
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
		<h2 class="mt-6 text-2xl font-semibold text-text">Complete your registration</h2>
	</div>

	<Card>
		{#if invitationLoading}
			<div class="text-center py-8">
				<div class="mx-auto flex items-center justify-center h-12 w-12">
					<Spinner size={32} class="text-primary" />
				</div>
				<p class="mt-4 text-sm text-text-muted">Loading invitation details...</p>
			</div>
		{:else if error && !invitationToken}
			<div class="text-center py-8 space-y-4">
				<Alert type="warning" title="Invitation Required" icon={AlertTriangle}>
					<p>{error}</p>
				</Alert>
				<div class="pt-4">
					<a href="/auth/login" class="text-sm font-medium text-primary hover:underline">
						Return to login
					</a>
				</div>
			</div>
		{:else}
			{#if error}
				<Alert type="error" title="Registration failed">
					<p>{error}</p>
				</Alert>
				<div class="mt-6"></div>
			{/if}

			<form class="space-y-5" onsubmit={handleRegister}>
				{#if invitationType}
					<div>
						<label class="form-label">Account Type</label>
						<div class="px-4 py-3 bg-bg-secondary border border-border rounded-xl">
							<div class="flex items-center gap-2">
								<Badge variant="info">
									{#if invitationType === 'partner'}
										Partner Manager
									{:else if invitationType === 'agent'}
										Sales Agent
									{:else}
										Homeowner
									{/if}
								</Badge>
							</div>
							<p class="text-xs text-text-muted mt-1">You've been invited to join Outlet</p>
						</div>
					</div>
				{/if}

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
					<label for="email" class="form-label">Email address</label>
					<Input id="email" type="email" bind:value={formData.email} />
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
