<script lang="ts">
	import { Card, Button, Input, Alert, LoadingSpinner, SaveButton, Select, Tabs, Badge } from '$lib/components/ui';
	import AwsIamPolicy from '$lib/components/admin/AwsIamPolicy.svelte';
	import { onMount } from 'svelte';
	import { getPlatformSettingsByCategory, updateEmailSettings } from '$lib/api/generate/outlet';
	import { Cloud, Server, CheckCircle, AlertCircle, Pencil } from '@lucide/svelte';

	let saving = $state(false);
	let saved = $state(false);
	let loading = $state(true);
	let error = $state('');

	// Provider selection
	type Provider = 'ses' | 'smtp';
	let activeProvider = $state<Provider>('ses');

	// SMTP form fields
	let smtpHost = $state('');
	let smtpPort = $state(587);
	let smtpUser = $state('');
	let smtpPassword = $state('');
	let hasExistingSmtpPassword = $state(false);
	let editingSmtp = $state(false);

	// SES form fields
	let awsAccessKey = $state('');
	let awsSecretKey = $state('');
	let awsRegion = $state('us-east-1');
	let hasExistingAwsKey = $state(false);
	let editingAwsCredentials = $state(false);

	const regionOptions = [
		{ value: 'us-east-1', label: 'US East (N. Virginia)' },
		{ value: 'us-east-2', label: 'US East (Ohio)' },
		{ value: 'us-west-1', label: 'US West (N. California)' },
		{ value: 'us-west-2', label: 'US West (Oregon)' },
		{ value: 'eu-west-1', label: 'Europe (Ireland)' },
		{ value: 'eu-west-2', label: 'Europe (London)' },
		{ value: 'eu-west-3', label: 'Europe (Paris)' },
		{ value: 'eu-central-1', label: 'Europe (Frankfurt)' },
		{ value: 'ap-south-1', label: 'Asia Pacific (Mumbai)' },
		{ value: 'ap-southeast-1', label: 'Asia Pacific (Singapore)' },
		{ value: 'ap-southeast-2', label: 'Asia Pacific (Sydney)' },
		{ value: 'ap-northeast-1', label: 'Asia Pacific (Tokyo)' },
		{ value: 'ap-northeast-2', label: 'Asia Pacific (Seoul)' },
		{ value: 'sa-east-1', label: 'South America (São Paulo)' },
		{ value: 'ca-central-1', label: 'Canada (Central)' }
	];

	const providerTabs = [
		{ id: 'ses', label: 'Amazon SES' },
		{ id: 'smtp', label: 'SMTP Relay' }
	];

	onMount(async () => {
		await loadSettings();
	});

	async function loadSettings() {
		try {
			const response = await getPlatformSettingsByCategory({}, 'email');
			for (const setting of response.settings) {
				switch (setting.key) {
					case 'smtp_host':
						smtpHost = setting.value || '';
						break;
					case 'smtp_port':
						smtpPort = setting.value ? parseInt(setting.value) : 587;
						break;
					case 'smtp_user':
						smtpUser = setting.value || '';
						break;
					case 'smtp_password':
						hasExistingSmtpPassword = !!setting.value;
						break;
					case 'aws_access_key':
						awsAccessKey = setting.value || '';
						hasExistingAwsKey = !!setting.value;
						break;
					case 'aws_region':
						awsRegion = setting.value || 'us-east-1';
						break;
				}
			}
			// Determine active provider based on what's configured
			if (awsAccessKey || hasExistingAwsKey) {
				activeProvider = 'ses';
			} else if (smtpHost) {
				activeProvider = 'smtp';
			}
		} catch (err: any) {
			error = err.message || 'Failed to load settings';
		} finally {
			loading = false;
		}
	}

	async function saveSmtpSettings() {
		saving = true;
		saved = false;
		error = '';

		try {
			await updateEmailSettings({
				smtp_host: smtpHost,
				smtp_port: smtpPort,
				smtp_user: smtpUser,
				smtp_password: smtpPassword || undefined
			});
			saved = true;
			setTimeout(() => {
				saved = false;
			}, 2000);
			if (smtpPassword) {
				hasExistingSmtpPassword = true;
				smtpPassword = '';
			}
		} catch (err: any) {
			error = err.message || 'Failed to save settings';
		} finally {
			saving = false;
		}
	}

	async function saveSesSettings() {
		saving = true;
		saved = false;
		error = '';

		try {
			await updateEmailSettings({
				aws_access_key: awsAccessKey,
				aws_secret_key: awsSecretKey || undefined,
				aws_region: awsRegion
			});
			saved = true;
			setTimeout(() => {
				saved = false;
			}, 2000);
			if (awsSecretKey) {
				hasExistingAwsKey = true;
				awsSecretKey = '';
			}
		} catch (err: any) {
			error = err.message || 'Failed to save settings';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Integrations - Settings</title>
</svelte:head>

<div class="space-y-6">
	{#if error}
		<Alert type="error" title="Error" onclose={() => (error = '')}>
			<p>{error}</p>
		</Alert>
	{/if}

	{#if loading}
		<div class="flex justify-center py-8">
			<LoadingSpinner />
		</div>
	{:else}
		<Alert type="info" title="Email Sending Provider">
			<p>
				Choose how Outlet sends emails. You can use an SMTP relay service (like SendGrid, Mailgun, Postmark)
				or connect directly to Amazon SES for high-volume sending.
			</p>
		</Alert>

		<div class="mb-4">
			<Tabs tabs={providerTabs} bind:activeTab={activeProvider} variant="pills" />
		</div>

		{#if activeProvider === 'ses'}
			<!-- Amazon SES Configuration -->
			<Card>
				<div class="flex items-center gap-3 mb-4">
					<Cloud class="h-5 w-5 text-primary" />
					<div>
						<h2 class="text-lg font-medium text-text">Amazon SES</h2>
						<p class="text-sm text-text-muted">Connect directly to AWS Simple Email Service for high-volume sending</p>
					</div>
				</div>

				<div class="space-y-6">
					<div class="grid grid-cols-1 gap-6 sm:grid-cols-6">
						<div class="sm:col-span-6">
							<label for="aws-region" class="form-label">AWS Region</label>
							<Select id="aws-region" bind:value={awsRegion} options={regionOptions} />
							<p class="mt-1 text-sm text-text-muted">Choose the region closest to your audience for best delivery performance</p>
						</div>

						{#if hasExistingAwsKey && !editingAwsCredentials}
							<!-- Show configured state -->
							<div class="sm:col-span-6">
								<div class="flex items-center justify-between p-4 bg-surface-secondary rounded-lg border border-border">
									<div class="flex items-center gap-3">
										<CheckCircle class="h-5 w-5 text-green-500" />
										<div>
											<p class="font-medium text-text">AWS Credentials Configured</p>
											<p class="text-sm text-text-muted">
												Access Key ID: <code class="bg-surface-tertiary px-1.5 py-0.5 rounded text-xs">{awsAccessKey}</code>
											</p>
											<p class="text-sm text-text-muted">
												Secret Access Key: <code class="bg-surface-tertiary px-1.5 py-0.5 rounded text-xs">••••••••••••••••</code>
											</p>
										</div>
									</div>
									<Button type="secondary" size="sm" onclick={() => (editingAwsCredentials = true)}>
										<Pencil class="h-4 w-4 mr-1.5" />
										Change
									</Button>
								</div>
							</div>
						{:else}
							<!-- Show input fields for new/editing credentials -->
							<div class="sm:col-span-3">
								<label for="aws-access-key" class="form-label">Access Key ID</label>
								<Input
									type="text"
									id="aws-access-key"
									bind:value={awsAccessKey}
									placeholder="AKIAIOSFODNN7EXAMPLE"
								/>
							</div>

							<div class="sm:col-span-3">
								<label for="aws-secret-key" class="form-label">Secret Access Key</label>
								<Input
									type="password"
									id="aws-secret-key"
									bind:value={awsSecretKey}
									placeholder=""
								/>
							</div>

							{#if editingAwsCredentials}
								<div class="sm:col-span-6">
									<Button type="secondary" size="sm" onclick={() => (editingAwsCredentials = false)}>
										Cancel
									</Button>
								</div>
							{/if}
						{/if}
					</div>

					<div class="pt-4 border-t border-border">
						<h3 class="font-medium text-text mb-3">SES Sandbox Mode</h3>
						<Alert type="warning" title="New AWS accounts start in sandbox mode">
							<p>
								In sandbox mode, you can only send emails to verified email addresses.
								Request production access from your AWS console to send to any recipient.
							</p>
						</Alert>
					</div>

					<p class="text-sm text-text-muted">
						From Name and From Email are configured per organization.
					</p>

					<div class="flex justify-end">
						<SaveButton label="Save SES Settings" {saving} {saved} onclick={saveSesSettings} />
					</div>
				</div>
			</Card>

			<!-- IAM Policy Card -->
			<AwsIamPolicy />
		{:else}
			<!-- SMTP Relay Configuration -->
			<Card>
				<div class="flex items-center gap-3 mb-4">
					<Server class="h-5 w-5 text-primary" />
					<div>
						<h2 class="text-lg font-medium text-text">SMTP Relay</h2>
						<p class="text-sm text-text-muted">Connect to SendGrid, Mailgun, Postmark, or any SMTP server</p>
					</div>
				</div>

				<div class="space-y-6">
					<div class="grid grid-cols-1 gap-6 sm:grid-cols-6">
						<div class="sm:col-span-4">
							<label for="smtp-host" class="form-label">SMTP Host</label>
							<Input type="text" id="smtp-host" bind:value={smtpHost} placeholder="smtp.sendgrid.net" />
						</div>

						<div class="sm:col-span-2">
							<label for="smtp-port" class="form-label">Port</label>
							<Input type="number" id="smtp-port" bind:value={smtpPort} />
						</div>

						<div class="sm:col-span-3">
							<label for="smtp-user" class="form-label">Username</label>
							<Input type="text" id="smtp-user" bind:value={smtpUser} placeholder="apikey" />
						</div>

						{#if hasExistingSmtpPassword && !editingSmtp}
							<!-- Show configured state for password -->
							<div class="sm:col-span-3">
								<label class="form-label">Password / API Key</label>
								<div class="flex items-center gap-2 p-2 bg-surface-secondary rounded-lg border border-border">
									<CheckCircle class="h-4 w-4 text-green-500 flex-shrink-0" />
									<code class="text-sm text-text-muted">••••••••••••••••</code>
									<Button type="secondary" size="sm" class="ml-auto" onclick={() => (editingSmtp = true)}>
										<Pencil class="h-3 w-3 mr-1" />
										Change
									</Button>
								</div>
							</div>
						{:else}
							<div class="sm:col-span-3">
								<label for="smtp-password" class="form-label">Password / API Key</label>
								<Input
									type="password"
									id="smtp-password"
									bind:value={smtpPassword}
									placeholder=""
								/>
								{#if editingSmtp}
									<button
										type="button"
										class="mt-1 text-sm text-text-muted hover:text-text"
										onclick={() => (editingSmtp = false)}
									>
										Cancel
									</button>
								{/if}
							</div>
						{/if}
					</div>

					<div class="pt-4 border-t border-border">
						<h3 class="font-medium text-text mb-3">Common Providers</h3>
						<div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-sm">
							<div class="p-3 bg-surface-secondary rounded-lg">
								<p class="font-medium text-text">SendGrid</p>
								<p class="text-text-muted">smtp.sendgrid.net:587</p>
								<p class="text-text-muted text-xs">Username: apikey</p>
							</div>
							<div class="p-3 bg-surface-secondary rounded-lg">
								<p class="font-medium text-text">Mailgun</p>
								<p class="text-text-muted">smtp.mailgun.org:587</p>
								<p class="text-text-muted text-xs">Username: your login</p>
							</div>
							<div class="p-3 bg-surface-secondary rounded-lg">
								<p class="font-medium text-text">Postmark</p>
								<p class="text-text-muted">smtp.postmarkapp.com:587</p>
								<p class="text-text-muted text-xs">Username: server API token</p>
							</div>
						</div>
					</div>

					<p class="text-sm text-text-muted">
						From Name and From Email are configured per organization.
					</p>

					<div class="flex justify-end">
						<SaveButton label="Save SMTP Settings" {saving} {saved} onclick={saveSmtpSettings} />
					</div>
				</div>
			</Card>
		{/if}
	{/if}
</div>
