<script lang="ts">
	import { listUsers, createUser, updateUser, deleteUser, type UserInfo } from '$lib/api';
	import { Button, Card, Input, Select, Badge, Modal, AlertDialog, Alert, LoadingSpinner, SaveButton } from '$lib/components/ui';
	import { Plus, Users } from '@lucide/svelte';

	let loading = $state(true);
	let users = $state<UserInfo[]>([]);
	let error = $state('');

	// Add user modal
	let showAddModal = $state(false);
	let adding = $state(false);
	let newName = $state('');
	let newEmail = $state('');
	let newPassword = $state('');
	let newRole = $state('admin');

	// Edit user modal
	let showEditModal = $state(false);
	let editing = $state(false);
	let editSaved = $state(false);
	let editingUser = $state<UserInfo | null>(null);
	let editName = $state('');
	let editEmail = $state('');
	let editRole = $state('');

	// Delete confirm
	let showDeleteConfirm = $state(false);
	let deletingUser = $state<UserInfo | null>(null);
	let deleting = $state(false);

	$effect(() => {
		loadUsers();
	});

	async function loadUsers() {
		loading = true;
		error = '';

		try {
			const response = await listUsers();
			users = response.users || [];
		} catch (err) {
			console.error('Failed to load users:', err);
			error = 'Failed to load users';
		} finally {
			loading = false;
		}
	}

	function openAddModal() {
		newName = '';
		newEmail = '';
		newPassword = '';
		newRole = 'admin';
		showAddModal = true;
	}

	function closeAddModal() {
		showAddModal = false;
	}

	async function submitAdd() {
		if (!newName || !newEmail || !newPassword) return;

		adding = true;
		error = '';

		try {
			await createUser({
				name: newName,
				email: newEmail,
				password: newPassword,
				role: newRole
			});
			closeAddModal();
			await loadUsers();
		} catch (err: any) {
			console.error('Failed to add user:', err);
			error = err.message || 'Failed to add user';
		} finally {
			adding = false;
		}
	}

	function openEditModal(user: UserInfo) {
		editingUser = user;
		editName = user.name;
		editEmail = user.email;
		editRole = user.role;
		showEditModal = true;
	}

	function closeEditModal() {
		showEditModal = false;
		editingUser = null;
	}

	async function submitEdit() {
		if (!editingUser) return;

		editing = true;
		editSaved = false;
		error = '';

		try {
			await updateUser({}, {
				name: editName,
				email: editEmail,
				role: editRole
			}, editingUser.id);
			editSaved = true;
			setTimeout(() => {
				editSaved = false;
				closeEditModal();
			}, 1500);
			await loadUsers();
		} catch (err: any) {
			console.error('Failed to update user:', err);
			error = err.message || 'Failed to update user';
		} finally {
			editing = false;
		}
	}

	function confirmDelete(user: UserInfo) {
		deletingUser = user;
		showDeleteConfirm = true;
	}

	async function executeDelete() {
		if (!deletingUser) return;

		deleting = true;
		try {
			await deleteUser({}, deletingUser.id);
			showDeleteConfirm = false;
			deletingUser = null;
			await loadUsers();
		} catch (err: any) {
			console.error('Failed to delete user:', err);
			error = err.message || 'Failed to delete user';
		} finally {
			deleting = false;
		}
	}

	function getRoleVariant(role: string): 'default' | 'primary' | 'success' | 'warning' | 'error' | 'info' {
		switch (role?.toLowerCase()) {
			case 'super_admin': return 'error';
			case 'admin': return 'primary';
			case 'agent': return 'info';
			default: return 'default';
		}
	}

	function formatDate(dateString: string): string {
		if (!dateString) return 'N/A';
		return new Date(dateString).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}
</script>

<svelte:head>
	<title>Users - Outlet</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex justify-between items-center">
		<p class="text-sm text-text-secondary">Manage admin users who can access Outlet</p>
		<Button type="primary" onclick={openAddModal}>
			<Plus class="mr-2 h-4 w-4" />
			Add User
		</Button>
	</div>

	{#if error}
		<Alert type="error" title="Error">
			<p>{error}</p>
		</Alert>
	{/if}

	{#if loading}
		<LoadingSpinner size="large" />
	{:else if users.length === 0}
		<Card>
			<div class="text-center py-12">
				<Users class="mx-auto h-12 w-12 text-text-muted" />
				<h3 class="mt-2 text-sm font-medium text-text">No users</h3>
				<p class="mt-1 text-sm text-text-muted">Get started by adding your first admin user.</p>
				<div class="mt-6">
					<Button type="primary" onclick={openAddModal}>
						<Plus class="mr-2 h-4 w-4" />
						Add User
					</Button>
				</div>
			</div>
		</Card>
	{:else}
		<Card>
			<div class="divide-y divide-border">
				{#each users as user}
					<div class="flex items-center justify-between py-4 first:pt-0 last:pb-0">
						<div class="flex items-center gap-3">
							<div class="h-10 w-10 rounded-full bg-primary flex items-center justify-center">
								<span class="text-sm font-medium text-white">
									{(user.name || user.email)[0].toUpperCase()}
								</span>
							</div>
							<div>
								<div class="flex items-center gap-2">
									<span class="font-medium text-text">{user.name}</span>
									<Badge variant={getRoleVariant(user.role)} size="sm">{user.role}</Badge>
									{#if user.active}
										<Badge variant="success" size="sm">Active</Badge>
									{:else}
										<Badge variant="error" size="sm">Inactive</Badge>
									{/if}
								</div>
								<p class="text-sm text-text-muted">{user.email}</p>
							</div>
						</div>
						<div class="flex items-center gap-2">
							<span class="text-sm text-text-muted mr-4">
								Added {formatDate(user.created_at)}
							</span>
							<Button type="secondary" size="sm" onclick={() => openEditModal(user)}>
								Edit
							</Button>
							<Button type="danger" size="sm" onclick={() => confirmDelete(user)}>
								Delete
							</Button>
						</div>
					</div>
				{/each}
			</div>
		</Card>
	{/if}
</div>

<!-- Add User Modal -->
<Modal bind:show={showAddModal} title="Add User">
	<div class="space-y-4">
		<div>
			<label for="new-name" class="form-label">Name</label>
			<Input
				id="new-name"
				type="text"
				bind:value={newName}
				placeholder="John Doe"
			/>
		</div>
		<div>
			<label for="new-email" class="form-label">Email</label>
			<Input
				id="new-email"
				type="email"
				bind:value={newEmail}
				placeholder="john@example.com"
			/>
		</div>
		<div>
			<label for="new-password" class="form-label">Password</label>
			<Input
				id="new-password"
				type="password"
				bind:value={newPassword}
				placeholder="Enter password"
			/>
		</div>
		<div>
			<label for="new-role" class="form-label">Role</label>
			<Select id="new-role" bind:value={newRole}>
				<option value="admin">Admin</option>
				<option value="agent">Agent</option>
				<option value="viewer">Viewer</option>
			</Select>
		</div>
	</div>

	{#snippet footer()}
		<div class="flex justify-end gap-3">
			<Button type="secondary" onclick={closeAddModal} disabled={adding}>
				Cancel
			</Button>
			<Button type="primary" onclick={submitAdd} disabled={!newName || !newEmail || !newPassword || adding}>
				{adding ? 'Adding...' : 'Add User'}
			</Button>
		</div>
	{/snippet}
</Modal>

<!-- Edit User Modal -->
<Modal bind:show={showEditModal} title="Edit User">
	{#if editingUser}
		<div class="space-y-4">
			<div>
				<label for="edit-name" class="form-label">Name</label>
				<Input
					id="edit-name"
					type="text"
					bind:value={editName}
				/>
			</div>
			<div>
				<label for="edit-email" class="form-label">Email</label>
				<Input
					id="edit-email"
					type="email"
					bind:value={editEmail}
				/>
			</div>
			<div>
				<label for="edit-role" class="form-label">Role</label>
				<Select id="edit-role" bind:value={editRole}>
					<option value="admin">Admin</option>
					<option value="agent">Agent</option>
					<option value="viewer">Viewer</option>
				</Select>
			</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end gap-3">
			<Button type="secondary" onclick={closeEditModal} disabled={editing}>
				Cancel
			</Button>
			<SaveButton
				label="Save Changes"
				saving={editing}
				saved={editSaved}
				onclick={submitEdit}
			/>
		</div>
	{/snippet}
</Modal>

<!-- Delete Confirmation -->
<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete User"
	description={deletingUser ? `Are you sure you want to delete ${deletingUser.name}? This action cannot be undone.` : ''}
	actionLabel={deleting ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDelete}
	onCancel={() => { showDeleteConfirm = false; deletingUser = null; }}
/>
