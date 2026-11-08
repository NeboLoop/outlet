<script lang="ts">
	import { formatDate } from '$lib/dates';
	import { page } from '$app/stores';
	import * as api from '$lib/api';
	import type { CreateInvitationResponse, ListOrgPeopleResponse, OrgInfo, OrgPendingInvitation } from '$lib/api';
	import { Badge, Button, Card, Input, Select, Alert, LoadingSpinner } from '$lib/components/ui';
	import { UserPlus } from '@lucide/svelte';

	// Outlet is invite-only. A brand admin invites people here; the invitation
	// decides the brand and the role, and its link goes out only by email.
	let loading = $state(true);
	let org = $state<OrgInfo | null>(null);
	let people = $state<ListOrgPeopleResponse | null>(null);
	let error = $state('');

	let inviteEmail = $state('');
	let inviteRole = $state('member');
	let inviting = $state(false);
	let resending = $state('');
	let result = $state<CreateInvitationResponse | null>(null);

	$effect(() => {
		loadOrg();
	});

	function message(err: unknown, fallback: string): string {
		return err instanceof Error && err.message ? err.message : fallback;
	}

	function roleLabel(role: string): string {
		return role === 'admin' ? 'Admin' : 'Member';
	}

	async function loadOrg() {
		loading = true;
		error = '';
		const brandSlug = $page.params.brandSlug;
		if (!brandSlug) return;

		try {
			org = await api.getOrganizationBySlug({}, brandSlug);
			await loadPeople();
		} catch (err) {
			console.error('Failed to load brand:', err);
			error = message(err, 'Could not load this brand.');
		} finally {
			loading = false;
		}
	}

	async function loadPeople() {
		if (!org) return;
		people = await api.listOrgPeople({}, org.id);
	}

	// sendInvitation invites someone, or invites them again: a new invitation
	// replaces the earlier one and sends a fresh link.
	async function sendInvitation(email: string, role: string) {
		if (!org) return;
		error = '';
		result = null;
		result = await api.createInvitation({}, { email, role }, org.id);
		await loadPeople();
	}

	async function invite() {
		if (!inviteEmail.trim()) return;
		inviting = true;
		try {
			await sendInvitation(inviteEmail.trim(), inviteRole);
			inviteEmail = '';
		} catch (err) {
			console.error('Failed to invite:', err);
			error = message(err, 'Could not send the invitation.');
		} finally {
			inviting = false;
		}
	}

	async function resend(invitation: OrgPendingInvitation) {
		resending = invitation.email;
		try {
			await sendInvitation(invitation.email, invitation.role);
		} catch (err) {
			console.error('Failed to resend the invitation:', err);
			error = message(err, 'Could not send the invitation again.');
		} finally {
			resending = '';
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
					<p>We emailed an invitation to {result.email}. It works once and expires {formatDate(result.expires_at, 'datetime')}.</p>
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

		{#if people}
			<Card>
				<h2 class="text-lg font-medium text-text mb-4">Members</h2>
				{#if people.members.length === 0}
					<p class="text-sm text-text-muted">Nobody has joined this brand yet.</p>
				{:else}
					<div class="divide-y divide-border">
						{#each people.members as member (member.user_id)}
							<div class="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0">
								<div class="min-w-0">
									<div class="flex items-center gap-2">
										<span class="font-medium text-text">{member.name || member.email}</span>
										<Badge variant={member.role === 'admin' ? 'primary' : 'neutral'} size="sm">{roleLabel(member.role)}</Badge>
									</div>
									<p class="text-sm text-text-muted">{member.email}</p>
								</div>
								<span class="text-sm text-text-muted">Joined {formatDate(member.joined_at)}</span>
							</div>
						{/each}
					</div>
				{/if}
			</Card>

			<Card>
				<h2 class="text-lg font-medium text-text mb-1">Pending invitations</h2>
				<p class="text-sm text-text-muted mb-4">
					Invitations not yet accepted. <strong>Resend</strong> emails a new link that replaces the old one.
				</p>
				{#if people.invitations.length === 0}
					<p class="text-sm text-text-muted">No invitations are waiting.</p>
				{:else}
					<div class="divide-y divide-border">
						{#each people.invitations as invitation (invitation.email)}
							<div class="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0">
								<div class="min-w-0">
									<div class="flex items-center gap-2">
										<span class="font-medium text-text">{invitation.email}</span>
										<Badge variant={invitation.role === 'admin' ? 'primary' : 'neutral'} size="sm">{roleLabel(invitation.role)}</Badge>
										{#if invitation.expired}
											<Badge variant="warning" size="sm">Expired</Badge>
										{/if}
									</div>
									<p class="text-sm text-text-muted">
										{invitation.expired ? 'Expired' : 'Expires'} {formatDate(invitation.expires_at, 'datetime')}
									</p>
								</div>
								<Button
									type="secondary"
									size="sm"
									onclick={() => resend(invitation)}
									disabled={resending === invitation.email}
								>
									{resending === invitation.email ? 'Sending...' : 'Resend'}
								</Button>
							</div>
						{/each}
					</div>
				{/if}
			</Card>
		{/if}
	</div>
{/if}
