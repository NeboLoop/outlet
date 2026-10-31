<script lang="ts">
	import { page } from '$app/stores';
	import * as api from '$lib/api';
	import type { CreateInvitationResponse, OrgInfo } from '$lib/api';
	import { Button, Card, Input, Select, Alert, LoadingSpinner } from '$lib/components/ui';
	import { UserPlus } from '@lucide/svelte';

	// Outlet is invite-only. A brand admin invites people here; the invitation
	// decides the brand and the role, and its link goes out only by email.
	let loading = $state(true);
	let org = $state<OrgInfo | null>(null);
	let error = $state('');

	let inviteEmail = $state('');
	let inviteRole = $state('member');
	let inviting = $state(false);
	let result = $state<CreateInvitationResponse | null>(null);

	$effect(() => {
		loadOrg();
	});

	async function loadOrg() {
		loading = true;
		error = '';
		const brandSlug = $page.params.brandSlug;
		if (!brandSlug) return;

		try {
			org = await api.getOrganizationBySlug({}, brandSlug);
		} catch (err: any) {
			console.error('Failed to load brand:', err);
			error = err.message || 'Could not load this brand.';
		} finally {
			loading = false;
		}
	}

	async function invite() {
		if (!org || !inviteEmail.trim()) return;

		inviting = true;
		error = '';
		result = null;
		try {
			result = await api.createInvitation({}, { email: inviteEmail.trim(), role: inviteRole }, org.id);
			inviteEmail = '';
		} catch (err: any) {
			console.error('Failed to invite:', err);
			error = err.message || 'Could not send the invitation.';
		} finally {
			inviting = false;
		}
	}
</script>

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner />
	</div>
{:else}
	<div class="space-y-6">
		{#if error}
			<Alert type="error" title="Something went wrong">
				<p>{error}</p>
			</Alert>
		{/if}

		{#if result}
			{#if result.email_sent}
				<Alert type="success" title="Invitation sent">
					<p>We emailed an invitation to {result.email}. It works once and expires {result.expires_at} UTC.</p>
				</Alert>
			{:else}
				<Alert type="warning" title="Invitation email did not go out">
					<p>The invitation email to {result.email} could not be sent. Check this brand's sending setup, then invite them again.</p>
				</Alert>
			{/if}
		{/if}

		<Card>
			<h2 class="text-lg font-medium text-text mb-1">Invite someone</h2>
			<p class="text-sm text-text-muted mb-4">
				Outlet is invite-only. Members run the brand's mail; admins also manage its settings, keys and people.
			</p>
			<div class="space-y-4">
				<div>
					<label for="invite-email" class="form-label">Email address</label>
					<Input id="invite-email" type="email" bind:value={inviteEmail} placeholder="name@example.com" />
				</div>
				<div>
					<label for="invite-role" class="form-label">Role</label>
					<Select id="invite-role" bind:value={inviteRole}>
						<option value="member">Member</option>
						<option value="admin">Admin</option>
					</Select>
				</div>
				<div class="pt-2">
					<Button type="primary" onclick={invite} disabled={inviting || !inviteEmail.trim()}>
						<UserPlus class="mr-2 h-4 w-4" />
						{inviting ? 'Inviting...' : 'Invite'}
					</Button>
				</div>
			</div>
		</Card>
	</div>
{/if}
