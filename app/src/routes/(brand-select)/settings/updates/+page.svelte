<script lang="ts">
	import * as api from '$lib/api';
	import type { VersionInfo, UpdateCheckResponse } from '$lib/api';
	import { Button, Card, Alert, LoadingSpinner, Badge } from '$lib/components/ui';
	import { RefreshCw, Download, CheckCircle, AlertCircle } from '@lucide/svelte';

	let loading = $state(true);
	let checking = $state(false);
	let versionInfo = $state<VersionInfo | null>(null);
	let updateInfo = $state<UpdateCheckResponse | null>(null);
	let error = $state('');

	$effect(() => {
		loadVersionInfo();
	});

	async function loadVersionInfo() {
		loading = true;
		error = '';
		try {
			versionInfo = await api.getVersion();
		} catch (err: any) {
			console.error('Failed to load version info:', err);
			error = err.message || 'Failed to load version information';
		} finally {
			loading = false;
		}
	}

	async function checkForUpdates() {
		checking = true;
		error = '';
		try {
			updateInfo = await api.checkForUpdates();
		} catch (err: any) {
			console.error('Failed to check for updates:', err);
			error = err.message || 'Failed to check for updates';
		} finally {
			checking = false;
		}
	}
</script>

<svelte:head>
	<title>Updates - Settings</title>
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
			<h2 class="text-lg font-medium text-text mb-4">Current Version</h2>
			{#if versionInfo}
				<div class="space-y-3">
					<div class="flex items-center justify-between">
						<span class="text-sm text-text-muted">Version</span>
						<Badge type="primary">{versionInfo.version}</Badge>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-sm text-text-muted">Commit</span>
						<code class="text-sm font-mono text-text">{versionInfo.commit}</code>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-sm text-text-muted">Build Date</span>
						<span class="text-sm text-text">{versionInfo.build_date}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-sm text-text-muted">Go Version</span>
						<span class="text-sm text-text">{versionInfo.go_version}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-sm text-text-muted">Platform</span>
						<span class="text-sm text-text">{versionInfo.os}/{versionInfo.arch}</span>
					</div>
				</div>
			{/if}
		</Card>

		<Card>
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-medium text-text">Update Status</h2>
				<Button
					type="secondary"
					onclick={checkForUpdates}
					disabled={checking}
				>
					<RefreshCw class="mr-2 h-4 w-4 {checking ? 'animate-spin' : ''}" />
					{checking ? 'Checking...' : 'Check for Updates'}
				</Button>
			</div>

			{#if updateInfo}
				{#if updateInfo.update_available}
					<div class="bg-primary/10 border border-primary/20 rounded-lg p-4">
						<div class="flex items-start gap-3">
							<AlertCircle class="h-5 w-5 text-primary flex-shrink-0 mt-0.5" />
							<div class="flex-1">
								<h3 class="font-medium text-text">Update Available</h3>
								<p class="text-sm text-text-muted mt-1">
									A new version <Badge type="success">{updateInfo.latest_version}</Badge> is available.
									You are currently running <Badge type="secondary">{updateInfo.current_version}</Badge>.
								</p>
								{#if updateInfo.release_date}
									<p class="text-sm text-text-muted mt-2">
										Released: {updateInfo.release_date}
									</p>
								{/if}
								{#if updateInfo.changelog}
									<div class="mt-4">
										<h4 class="text-sm font-medium text-text mb-2">What's New</h4>
										<div class="text-sm text-text-muted bg-bg-secondary rounded p-3 whitespace-pre-wrap">
											{updateInfo.changelog}
										</div>
									</div>
								{/if}
								{#if updateInfo.download_url}
									<div class="mt-4">
										<Button
											type="primary"
											onclick={() => window.open(updateInfo?.download_url, '_blank')}
										>
											<Download class="mr-2 h-4 w-4" />
											Download Update
										</Button>
									</div>
								{/if}
								<div class="mt-4 text-sm text-text-muted">
									<p>To update via CLI, run:</p>
									<code class="block bg-bg-secondary rounded p-2 mt-1 font-mono">outlet update</code>
								</div>
							</div>
						</div>
					</div>
				{:else}
					<div class="bg-green-500/10 border border-green-500/20 rounded-lg p-4">
						<div class="flex items-center gap-3">
							<CheckCircle class="h-5 w-5 text-green-500" />
							<div>
								<h3 class="font-medium text-text">You're up to date!</h3>
								<p class="text-sm text-text-muted mt-1">
									Outlet.sh {updateInfo.current_version} is the latest version.
								</p>
							</div>
						</div>
					</div>
				{/if}
			{:else}
				<div class="text-center py-8 text-text-muted">
					<p>Click "Check for Updates" to see if a new version is available.</p>
				</div>
			{/if}
		</Card>

		<Card>
			<h2 class="text-lg font-medium text-text mb-4">Update Instructions</h2>
			<div class="space-y-4 text-sm text-text-muted">
				<div>
					<h3 class="font-medium text-text mb-2">CLI Update (Recommended)</h3>
					<p>Run the update command to automatically download and install the latest version:</p>
					<code class="block bg-bg-secondary rounded p-3 mt-2 font-mono">
						outlet update
					</code>
					<p class="mt-2 text-xs">Use <code class="bg-bg-secondary px-1 rounded">--check</code> to check without installing.</p>
				</div>
				<div>
					<h3 class="font-medium text-text mb-2">Docker Update</h3>
					<p>If running via Docker, pull the latest image:</p>
					<code class="block bg-bg-secondary rounded p-3 mt-2 font-mono">
						docker pull ghcr.io/localrivet/outlet:latest
					</code>
				</div>
				<div>
					<h3 class="font-medium text-text mb-2">Manual Update</h3>
					<ol class="list-decimal list-inside space-y-1">
						<li>Download the latest release from <a href="https://github.com/localrivet/outlet/releases" target="_blank" class="text-primary hover:underline">GitHub Releases</a></li>
						<li>Stop the current Outlet instance</li>
						<li>Replace the binary with the new version</li>
						<li>Start Outlet again</li>
					</ol>
				</div>
			</div>
		</Card>
	</div>
{/if}
