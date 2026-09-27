<script lang="ts">
	import * as api from '$lib/api';
	import type { BackupInfo, BackupSettingsResponse } from '$lib/api';
	import { getToken } from '$lib/auth';
	import { Button, Card, Input, Alert, LoadingSpinner, Badge, SaveButton, Toggle, AlertDialog } from '$lib/components/ui';
	import { Plus, Download, Trash2, RefreshCw, HardDrive, Cloud, Copy, ExternalLink, CheckCircle, Pencil } from '@lucide/svelte';
	import { getWebSocketClient } from '$lib/websocket/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { onDestroy, onMount } from 'svelte';

	let loading = $state(true);
	let backups = $state<BackupInfo[]>([]);
	let settings = $state<BackupSettingsResponse | null>(null);
	let error = $state('');

	// Create backup state
	let creating = $state(false);
	let uploadToS3 = $state(false);

	// Settings state
	let saving = $state(false);
	let saved = $state(false);
	let editS3Enabled = $state(false);
	let editS3Bucket = $state('');
	let editS3Region = $state('');
	let editS3AccessKey = $state('');
	let editS3SecretKey = $state('');
	let editS3Prefix = $state('');
	let editScheduleEnabled = $state(false);
	let editScheduleCron = $state('0 3 * * *');
	let editRetentionDays = $state(30);

	// Delete confirmation
	let showDeleteConfirm = $state(false);
	let backupToDelete = $state<BackupInfo | null>(null);
	let deleting = $state(false);

	// Edit S3 credentials state
	let editingS3Credentials = $state(false);

	// IAM Policy copy state
	let copiedPolicy = $state(false);

	// S3 Backup IAM Policy - uses bucket placeholder that user should replace
	const s3IamPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::YOUR-BUCKET-NAME",
        "arn:aws:s3:::YOUR-BUCKET-NAME/*"
      ]
    }
  ]
}`;

	async function copyPolicy() {
		await navigator.clipboard.writeText(s3IamPolicy);
		copiedPolicy = true;
		setTimeout(() => copiedPolicy = false, 2000);
	}

	// WebSocket cleanup function
	let wsCleanup: (() => void) | null = null;

	onMount(() => {
		// Connect to WebSocket and listen for backup updates
		const wsClient = getWebSocketClient();
		wsClient.connect(authStore.user?.id);

		wsCleanup = wsClient.on('backup_update', (data: { id: string; status: string; filename: string; file_size: number; error?: string }) => {
			console.log('[Backup] Received backup update:', data);
			// Refresh the backup list when we receive an update
			loadData();
		});
	});

	onDestroy(() => {
		// Clean up WebSocket listener
		if (wsCleanup) {
			wsCleanup();
		}
	});

	$effect(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = '';
		try {
			const [backupsRes, settingsRes] = await Promise.all([
				api.listBackups({ page: 1, page_size: 50 }),
				api.getBackupSettings()
			]);
			backups = backupsRes.backups || [];
			settings = settingsRes;

			// Populate edit state
			editS3Enabled = settingsRes.s3_enabled;
			editS3Bucket = settingsRes.s3_bucket || '';
			editS3Region = settingsRes.s3_region || '';
			editS3Prefix = settingsRes.s3_prefix || '';
			editScheduleEnabled = settingsRes.schedule_enabled;
			editScheduleCron = settingsRes.schedule_cron || '0 3 * * *';
			editRetentionDays = settingsRes.retention_days || 30;
		} catch (err: any) {
			console.error('Failed to load backup data:', err);
			error = err.message || 'Failed to load backup data';
		} finally {
			loading = false;
		}
	}

	async function createBackup() {
		creating = true;
		error = '';
		try {
			await api.createBackup({
				upload_to_s3: uploadToS3 && editS3Enabled
			});
			await loadData();
		} catch (err: any) {
			console.error('Failed to create backup:', err);
			error = err.message || 'Failed to create backup';
		} finally {
			creating = false;
		}
	}

	async function saveSettings() {
		saving = true;
		saved = false;
		error = '';
		try {
			const updated = await api.updateBackupSettings({
				s3_enabled: editS3Enabled,
				s3_bucket: editS3Bucket,
				s3_region: editS3Region,
				s3_access_key: editS3AccessKey,
				s3_secret_key: editS3SecretKey,
				s3_prefix: editS3Prefix,
				schedule_enabled: editScheduleEnabled,
				schedule_cron: editScheduleCron,
				retention_days: editRetentionDays
			});
			settings = updated;
			saved = true;
			setTimeout(() => { saved = false; }, 2000);
		} catch (err: any) {
			console.error('Failed to save settings:', err);
			error = err.message || 'Failed to save settings';
		} finally {
			saving = false;
		}
	}

	function confirmDelete(backup: BackupInfo) {
		backupToDelete = backup;
		showDeleteConfirm = true;
	}

	async function executeDelete() {
		if (!backupToDelete) return;
		deleting = true;
		try {
			await api.deleteBackup({}, backupToDelete.id);
			showDeleteConfirm = false;
			backupToDelete = null;
			await loadData();
		} catch (err: any) {
			console.error('Failed to delete backup:', err);
			error = err.message || 'Failed to delete backup';
		} finally {
			deleting = false;
		}
	}

	async function downloadBackup(backup: BackupInfo) {
		try {
			const token = getToken();
			if (!token) {
				throw new Error('Not authenticated');
			}

			// Fetch with auth headers
			const response = await fetch(`/api/admin/backup/${backup.id}/download`, {
				method: 'GET',
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});

			if (!response.ok) {
				const errData = await response.json().catch(() => null);
				throw new Error(errData?.error || `Download failed: ${response.status}`);
			}

			// Create blob and trigger download
			const blob = await response.blob();
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = backup.filename;
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			document.body.removeChild(a);
		} catch (err: any) {
			console.error('Download failed:', err);
			error = err.message || 'Failed to download backup';
		}
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function formatDate(dateStr: string): string {
		if (!dateStr) return '-';
		return new Date(dateStr).toLocaleString();
	}
</script>

<svelte:head>
	<title>Backup - Settings</title>
</svelte:head>

{#if error}
	<div class="mb-6">
		<Alert type="error" title="Error">
			<p>{error}</p>
		</Alert>
	</div>
{/if}

{#if loading}
	<LoadingSpinner size="large" />
{:else}
	<div class="space-y-6">
		<Card>
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-medium text-text">Create Backup</h2>
			</div>
			<div class="space-y-4">
				<p class="text-sm text-text-muted">
					Creates a compressed backup containing both SQLite binary (.db) and SQL dump (.sql) formats.
				</p>
				{#if settings?.s3_enabled}
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" bind:checked={uploadToS3} class="checkbox checkbox-primary" />
						<span class="text-sm text-text">Also upload to S3</span>
					</label>
				{/if}
				<Button
					type="primary"
					onclick={createBackup}
					disabled={creating}
				>
					<Plus class="mr-2 h-4 w-4" />
					{creating ? 'Creating Backup...' : 'Create Backup Now'}
				</Button>
			</div>
		</Card>

		<Card>
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-medium text-text">Backup History</h2>
				<Button type="ghost" onclick={loadData}>
					<RefreshCw class="h-4 w-4" />
				</Button>
			</div>

			{#if backups.length === 0}
				<p class="text-sm text-text-muted text-center py-8">No backups yet. Create your first backup above.</p>
			{:else}
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="border-b border-border">
								<th class="text-left py-2 px-3 text-sm font-medium text-text-muted">Filename</th>
								<th class="text-left py-2 px-3 text-sm font-medium text-text-muted">Size</th>
								<th class="text-left py-2 px-3 text-sm font-medium text-text-muted">Type</th>
								<th class="text-left py-2 px-3 text-sm font-medium text-text-muted">Storage</th>
								<th class="text-left py-2 px-3 text-sm font-medium text-text-muted">Status</th>
								<th class="text-left py-2 px-3 text-sm font-medium text-text-muted">Created</th>
								<th class="text-right py-2 px-3 text-sm font-medium text-text-muted">Actions</th>
							</tr>
						</thead>
						<tbody>
							{#each backups as backup}
								<tr class="border-b border-border hover:bg-bg-secondary">
									<td class="py-2 px-3 text-sm font-mono">{backup.filename}</td>
									<td class="py-2 px-3 text-sm">{formatBytes(backup.file_size)}</td>
									<td class="py-2 px-3 text-sm">
										<Badge type="secondary">{backup.backup_type}</Badge>
									</td>
									<td class="py-2 px-3 text-sm">
										{#if backup.storage_type === 's3'}
											<Cloud class="h-4 w-4 inline text-primary" />
											<span class="ml-1">S3</span>
										{:else}
											<HardDrive class="h-4 w-4 inline text-text-muted" />
											<span class="ml-1">Local</span>
										{/if}
									</td>
									<td class="py-2 px-3 text-sm">
										{#if backup.status === 'completed'}
											<Badge type="success">Completed</Badge>
										{:else if backup.status === 'in_progress'}
											<Badge type="warning">Working</Badge>
										{:else if backup.status === 'failed'}
											<Badge type="error">Failed</Badge>
										{:else}
											<Badge type="secondary">{backup.status}</Badge>
										{/if}
									</td>
									<td class="py-2 px-3 text-sm text-text-muted">{formatDate(backup.started_at)}</td>
									<td class="py-2 px-3 text-right">
										<div class="flex items-center justify-end gap-1">
											{#if backup.status === 'completed'}
												<Button type="ghost" size="icon" onclick={() => downloadBackup(backup)}>
													<Download class="h-4 w-4" />
												</Button>
											{/if}
											<Button type="ghost" size="icon" onclick={() => confirmDelete(backup)}>
												<Trash2 class="h-4 w-4 text-red-500" />
											</Button>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</Card>

		<Card>
			<h2 class="text-lg font-medium text-text mb-4">S3 Storage Settings</h2>
			<div class="space-y-4">
				<Toggle
					bind:checked={editS3Enabled}
					label="Enable S3 backup storage"
				/>

				{#if editS3Enabled}
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="s3-bucket" class="form-label">S3 Bucket</label>
							<Input
								id="s3-bucket"
								type="text"
								bind:value={editS3Bucket}
								placeholder="my-backup-bucket"
							/>
						</div>
						<div>
							<label for="s3-region" class="form-label">S3 Region</label>
							<Input
								id="s3-region"
								type="text"
								bind:value={editS3Region}
								placeholder="us-east-1"
							/>
						</div>
					</div>
					<div>
						<label for="s3-prefix" class="form-label">S3 Key Prefix (optional)</label>
						<Input
							id="s3-prefix"
							type="text"
							bind:value={editS3Prefix}
							placeholder="backups/outlet/"
						/>
					</div>
					{#if settings?.has_s3_creds && !editingS3Credentials}
						<!-- Show configured state -->
						<div class="flex items-center justify-between p-4 bg-surface-secondary rounded-lg border border-border">
							<div class="flex items-center gap-3">
								<CheckCircle class="h-5 w-5 text-green-500" />
								<div>
									<p class="font-medium text-text">AWS Credentials Configured</p>
									<p class="text-sm text-text-muted">
										Access Key ID: <code class="bg-surface-tertiary px-1.5 py-0.5 rounded text-xs">••••••••</code>
									</p>
									<p class="text-sm text-text-muted">
										Secret Access Key: <code class="bg-surface-tertiary px-1.5 py-0.5 rounded text-xs">••••••••</code>
									</p>
								</div>
							</div>
							<Button type="secondary" size="sm" onclick={() => (editingS3Credentials = true)}>
								<Pencil class="h-4 w-4 mr-1.5" />
								Change
							</Button>
						</div>
					{:else}
						<!-- Show input fields for new/editing credentials -->
						<div class="grid grid-cols-2 gap-4">
							<div>
								<label for="s3-access-key" class="form-label">Access Key ID</label>
								<Input
									id="s3-access-key"
									type="text"
									bind:value={editS3AccessKey}
									placeholder="AKIAIOSFODNN7EXAMPLE"
								/>
							</div>
							<div>
								<label for="s3-secret-key" class="form-label">Secret Access Key</label>
								<Input
									id="s3-secret-key"
									type="password"
									bind:value={editS3SecretKey}
									placeholder=""
								/>
							</div>
						</div>
						{#if editingS3Credentials}
							<div>
								<Button type="secondary" size="sm" onclick={() => (editingS3Credentials = false)}>
									Cancel
								</Button>
							</div>
						{/if}
					{/if}

					<div class="mt-6 pt-4 border-t border-border">
						<div class="flex items-center justify-between mb-3">
							<div class="flex items-center gap-2">
								<h3 class="font-medium text-sm text-text">Required IAM Policy</h3>
								<a
									href="https://console.aws.amazon.com/iam/"
									target="_blank"
									rel="noopener noreferrer"
									class="text-xs text-primary hover:underline flex items-center gap-1"
								>
									Open IAM Console
									<ExternalLink class="w-3 h-3" />
								</a>
							</div>
							<button
								type="button"
								onclick={copyPolicy}
								class="text-xs text-primary hover:underline flex items-center gap-1"
							>
								<Copy class="w-3 h-3" />
								{copiedPolicy ? 'Copied!' : 'Copy'}
							</button>
						</div>
						<p class="text-xs text-text-muted mb-2">
							Create an IAM user with this policy. Replace <code class="bg-base-200 px-1 rounded">YOUR-BUCKET-NAME</code> with your bucket name.
						</p>
						<pre class="text-xs bg-base-200 p-3 rounded-lg overflow-x-auto text-text-muted">{s3IamPolicy}</pre>
					</div>
				{/if}
			</div>
		</Card>

		<Card>
			<h2 class="text-lg font-medium text-text mb-4">Scheduled Backups</h2>
			<div class="space-y-4">
				<Toggle
					bind:checked={editScheduleEnabled}
					label="Enable scheduled backups"
				/>

				{#if editScheduleEnabled}
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="schedule-cron" class="form-label">Schedule (Cron)</label>
							<Input
								id="schedule-cron"
								type="text"
								bind:value={editScheduleCron}
								placeholder="0 3 * * *"
							/>
							<p class="mt-1 text-xs text-text-muted">Default: 3 AM daily (0 3 * * *)</p>
						</div>
						<div>
							<label for="retention-days" class="form-label">Retention (days)</label>
							<Input
								id="retention-days"
								type="number"
								bind:value={editRetentionDays}
								min="1"
								max="365"
							/>
							<p class="mt-1 text-xs text-text-muted">Backups older than this will be deleted</p>
						</div>
					</div>
					{#if settings?.last_backup_at}
						<p class="text-sm text-text-muted">
							Last backup: {formatDate(settings.last_backup_at)}
						</p>
					{/if}
				{/if}
			</div>
		</Card>

		<div class="flex justify-end">
			<SaveButton
				label="Save Settings"
				{saving}
				{saved}
				onclick={saveSettings}
			/>
		</div>
	</div>
{/if}

<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete Backup"
	description={`Are you sure you want to delete "${backupToDelete?.filename}"? This action cannot be undone.`}
	actionLabel={deleting ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDelete}
	onCancel={() => { showDeleteConfirm = false; backupToDelete = null; }}
/>
