<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import * as api from '$lib/api';
	import { authStore } from '$lib/stores/auth.svelte';
	import { Card, Input, Alert, Select, Steps } from '$lib/components/ui';
	import AwsIamPolicy from '$lib/components/admin/AwsIamPolicy.svelte';
	import type { StepItem } from '$lib/components/ui/Steps.svelte';
	import { ChevronRight, Check, Loader2, HelpCircle, Shield, Key, Cloud, ExternalLink, Globe } from '@lucide/svelte';

	type Step = 'loading' | 'admin' | 'aws' | 'complete';
	let currentStep = $state<Step>('loading');
	let loading = $state(false);
	let error = $state('');

	// Track which steps have been completed (can't go back after completion)
	let completedSteps = $state<Set<Step>>(new Set());

	// Admin form fields
	let adminEmail = $state('');
	let adminPassword = $state('');
	let adminConfirmPassword = $state('');
	let adminName = $state('');
	let adminCompany = $state('');
	let adminTimezone = $state('');

	// AWS form fields
	let awsAccessKey = $state('');
	let awsSecretKey = $state('');
	let awsRegion = $state('us-east-1');

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

	// Common timezones - grouped by region
	const timezoneOptions = [
		// Americas
		{ value: 'America/New_York', label: 'Eastern Time (US & Canada)' },
		{ value: 'America/Chicago', label: 'Central Time (US & Canada)' },
		{ value: 'America/Denver', label: 'Mountain Time (US & Canada)' },
		{ value: 'America/Los_Angeles', label: 'Pacific Time (US & Canada)' },
		{ value: 'America/Anchorage', label: 'Alaska' },
		{ value: 'Pacific/Honolulu', label: 'Hawaii' },
		{ value: 'America/Toronto', label: 'Eastern Time (Canada)' },
		{ value: 'America/Vancouver', label: 'Pacific Time (Canada)' },
		{ value: 'America/Mexico_City', label: 'Mexico City' },
		{ value: 'America/Sao_Paulo', label: 'São Paulo' },
		{ value: 'America/Buenos_Aires', label: 'Buenos Aires' },
		// Europe
		{ value: 'Europe/London', label: 'London (GMT/BST)' },
		{ value: 'Europe/Paris', label: 'Paris (CET/CEST)' },
		{ value: 'Europe/Berlin', label: 'Berlin (CET/CEST)' },
		{ value: 'Europe/Amsterdam', label: 'Amsterdam (CET/CEST)' },
		{ value: 'Europe/Madrid', label: 'Madrid (CET/CEST)' },
		{ value: 'Europe/Rome', label: 'Rome (CET/CEST)' },
		{ value: 'Europe/Zurich', label: 'Zurich (CET/CEST)' },
		{ value: 'Europe/Stockholm', label: 'Stockholm (CET/CEST)' },
		{ value: 'Europe/Moscow', label: 'Moscow (MSK)' },
		// Asia & Pacific
		{ value: 'Asia/Dubai', label: 'Dubai (GST)' },
		{ value: 'Asia/Kolkata', label: 'India (IST)' },
		{ value: 'Asia/Singapore', label: 'Singapore (SGT)' },
		{ value: 'Asia/Hong_Kong', label: 'Hong Kong (HKT)' },
		{ value: 'Asia/Shanghai', label: 'Shanghai (CST)' },
		{ value: 'Asia/Tokyo', label: 'Tokyo (JST)' },
		{ value: 'Asia/Seoul', label: 'Seoul (KST)' },
		{ value: 'Australia/Sydney', label: 'Sydney (AEST/AEDT)' },
		{ value: 'Australia/Melbourne', label: 'Melbourne (AEST/AEDT)' },
		{ value: 'Australia/Perth', label: 'Perth (AWST)' },
		{ value: 'Pacific/Auckland', label: 'Auckland (NZST/NZDT)' },
		// UTC
		{ value: 'UTC', label: 'UTC (Coordinated Universal Time)' }
	];

	onMount(async () => {
		// Auto-detect user's timezone
		try {
			const detectedTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
			// Check if detected timezone is in our list
			if (timezoneOptions.some(tz => tz.value === detectedTimezone)) {
				adminTimezone = detectedTimezone;
			} else {
				// Default to UTC if not found
				adminTimezone = 'UTC';
			}
		} catch {
			adminTimezone = 'America/New_York';
		}

		try {
			const status = await api.getPublicSetupStatus();

			if (!status.has_admin) {
				currentStep = 'admin';
			} else if (!status.has_aws) {
				// Admin exists but no AWS credentials - check if logged in
				// Mark admin step as already completed
				completedSteps.add('admin');
				if (authStore.isAuthenticated) {
					currentStep = 'aws';
				} else {
					goto('/auth/login?redirect=/setup');
				}
			} else {
				// Fully configured
				goto('/');
			}
		} catch (err) {
			console.error('Failed to check setup status:', err);
			currentStep = 'admin';
		}
	});

	async function createAdmin() {
		loading = true;
		error = '';

		if (adminPassword !== adminConfirmPassword) {
			error = 'Passwords do not match';
			loading = false;
			return;
		}

		if (adminPassword.length < 8) {
			error = 'Password must be at least 8 characters';
			loading = false;
			return;
		}

		try {
			await api.createInitialAdmin({
				email: adminEmail,
				password: adminPassword,
				confirm_password: adminConfirmPassword,
				name: adminName || undefined,
				company: adminCompany || undefined,
				timezone: adminTimezone || undefined
			});

			// Auto-login
			const loginResponse = await api.login({
				email: adminEmail,
				password: adminPassword
			});

			if (loginResponse.token && loginResponse.user) {
				authStore.setSession(loginResponse.token, loginResponse.user);
			}

			// Mark admin step as completed
			completedSteps.add('admin');
			currentStep = 'aws';
		} catch (err: any) {
			console.error('Failed to create admin:', err);
			error = err.message || 'Failed to create admin account';
		} finally {
			loading = false;
		}
	}

	async function saveAwsCredentials() {
		loading = true;
		error = '';

		try {
			await api.updateEmailSettings({
				aws_access_key: awsAccessKey,
				aws_secret_key: awsSecretKey,
				aws_region: awsRegion
			});

			// Mark AWS step as completed
			completedSteps.add('aws');
			currentStep = 'complete';
		} catch (err: any) {
			console.error('Failed to save AWS credentials:', err);
			error = err.message || 'Failed to save AWS credentials';
		} finally {
			loading = false;
		}
	}

	function goToDashboard() {
		goto('/');
	}

	function skipAwsForNow() {
		goto('/');
	}

	// Steps configuration for the Steps component
	function getStepStatus(stepKey: Step, completedKey?: Step): StepItem['status'] {
		if (completedKey && completedSteps.has(completedKey)) return 'complete';
		if (currentStep === stepKey) return stepKey === 'complete' ? 'complete' : 'current';
		return 'pending';
	}

	let setupSteps = $derived<StepItem[]>([
		{ label: 'Account', status: getStepStatus('admin', 'admin') },
		{ label: 'Email', status: getStepStatus('aws', 'aws') },
		{ label: 'Complete', status: getStepStatus('complete') }
	]);

	</script>

<svelte:head>
	<title>Setup - Outlet</title>
</svelte:head>

{#if currentStep === 'loading'}
	<div class="flex items-center justify-center py-12">
		<Loader2 class="w-8 h-8 animate-spin text-primary" />
	</div>
{:else}
<div class="grid grid-cols-1 md:grid-cols-5 gap-8">
	<!-- Main Content -->
	<div class="md:col-span-3">
		<Card>
			<!-- Progress indicator -->
			<Steps steps={setupSteps} class="mb-8" />

			{#if currentStep === 'admin'}
				<!-- Admin Creation Step -->
				<div>
					<div class="text-center mb-6">
						<div class="mx-auto w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mb-6">
							<Shield class="w-8 h-8 text-primary" />
						</div>
						<h1 class="text-2xl font-bold text-text mb-2">Create Admin Account</h1>
						<p class="text-text-muted">
							Set up your administrator account to get started.
						</p>
					</div>

					{#if error}
						<Alert type="error" title="Error" class="mb-6">
							<p>{error}</p>
						</Alert>
					{/if}

					<form class="space-y-5" onsubmit={(e) => { e.preventDefault(); createAdmin(); }}>
						<div>
							<label for="admin-company" class="form-label">Company</label>
							<Input
								id="admin-company"
								type="text"
								placeholder="Your company name"
								bind:value={adminCompany}
								required
							/>
						</div>

						<div>
							<label for="admin-name" class="form-label">Name</label>
							<Input
								id="admin-name"
								type="text"
								placeholder="Your name"
								bind:value={adminName}
								required
							/>
						</div>

						<div>
							<label for="admin-email" class="form-label">Email Address</label>
							<Input
								id="admin-email"
								type="email"
								placeholder="admin@example.com"
								bind:value={adminEmail}
								required
							/>
						</div>

						<div>
							<label for="admin-password" class="form-label">Password</label>
							<Input
								id="admin-password"
								type="password"
								placeholder="Minimum 8 characters"
								bind:value={adminPassword}
								required
							/>
						</div>

						<div>
							<label for="admin-confirm-password" class="form-label">Confirm Password</label>
							<Input
								id="admin-confirm-password"
								type="password"
								placeholder="Confirm your password"
								bind:value={adminConfirmPassword}
								required
							/>
						</div>

						<div>
							<label for="admin-timezone" class="form-label">Timezone</label>
							<Select id="admin-timezone" bind:value={adminTimezone} options={timezoneOptions} />
							<p class="mt-1 text-xs text-text-muted">Used for scheduling campaigns and reports</p>
						</div>

						<div class="pt-4">
							<button
								type="submit"
								class="btn btn-primary w-full"
								disabled={loading || !adminCompany || !adminName || !adminEmail || !adminPassword || !adminConfirmPassword}
							>
								{#if loading}
									<Loader2 class="w-4 h-4 animate-spin" />
									Creating...
								{:else}
									Continue
									<ChevronRight class="w-4 h-4" />
								{/if}
							</button>
						</div>
					</form>
				</div>
			{:else if currentStep === 'aws'}
				<!-- AWS Credentials Step -->
				<div>
					<div class="text-center mb-6">
						<div class="mx-auto w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mb-6">
							<Cloud class="w-8 h-8 text-primary" />
						</div>
						<h1 class="text-2xl font-bold text-text mb-2">Connect Amazon SES</h1>
						<p class="text-text-muted">
							Enter your AWS credentials to enable email sending.
						</p>
					</div>

					{#if error}
						<Alert type="error" title="Error" class="mb-6">
							<p>{error}</p>
						</Alert>
					{/if}

					<form class="space-y-5" onsubmit={(e) => { e.preventDefault(); saveAwsCredentials(); }}>
						<div>
							<label for="aws-region" class="form-label">AWS Region</label>
							<Select id="aws-region" bind:value={awsRegion} options={regionOptions} />
							<p class="mt-1 text-xs text-text-muted">Choose the region closest to your audience</p>
						</div>

						<div>
							<label for="aws-access-key" class="form-label">Access Key ID</label>
							<Input
								id="aws-access-key"
								type="text"
								placeholder="AKIAIOSFODNN7EXAMPLE"
								bind:value={awsAccessKey}
								required
							/>
						</div>

						<div>
							<label for="aws-secret-key" class="form-label">Secret Access Key</label>
							<Input
								id="aws-secret-key"
								type="password"
								placeholder="Your secret access key"
								bind:value={awsSecretKey}
								required
							/>
						</div>

						<div class="flex flex-col gap-3 pt-4">
							<button
								type="submit"
								class="btn btn-primary w-full"
								disabled={loading || !awsAccessKey || !awsSecretKey}
							>
								{#if loading}
									<Loader2 class="w-4 h-4 animate-spin" />
									Connecting...
								{:else}
									Complete Setup
									<ChevronRight class="w-4 h-4" />
								{/if}
							</button>
							<button
								type="button"
								onclick={skipAwsForNow}
								class="btn btn-ghost btn-sm"
							>
								Skip for now
							</button>
						</div>
					</form>
				</div>
			{:else if currentStep === 'complete'}
				<!-- Complete Step -->
				<div class="text-center">
					<div class="mx-auto w-16 h-16 rounded-full bg-green-500/10 flex items-center justify-center mb-6">
						<Check class="w-8 h-8 text-green-500" />
					</div>
					<h2 class="text-2xl font-bold text-text mb-2">You're all set!</h2>
					<p class="text-text-muted mb-8">
						Your platform is configured and ready to use. Create your first brand to start sending emails.
					</p>

					<button class="btn btn-primary btn-lg w-full" onclick={goToDashboard}>
						Go to Dashboard
						<ChevronRight class="w-4 h-4" />
					</button>
				</div>
			{/if}
		</Card>
	</div>

	<!-- Help Sidebar -->
	<div class="md:col-span-2">
		<div class="sticky top-8 space-y-6">
			{#if currentStep === 'admin'}
				<!-- Admin Help -->
				<Card>
					<div class="flex items-center gap-2 mb-4">
						<Shield class="w-5 h-5 text-primary" />
						<h3 class="font-semibold text-text">Admin Account</h3>
					</div>
					<p class="text-sm text-text-muted mb-4">
						This is the first administrator account for your Outlet installation.
					</p>
					<ul class="space-y-3 text-sm text-text-muted">
						<li class="flex items-start gap-2">
							<Check class="w-4 h-4 text-green-500 mt-0.5 shrink-0" />
							<span>Full access to all features</span>
						</li>
						<li class="flex items-start gap-2">
							<Check class="w-4 h-4 text-green-500 mt-0.5 shrink-0" />
							<span>Can create additional users</span>
						</li>
						<li class="flex items-start gap-2">
							<Check class="w-4 h-4 text-green-500 mt-0.5 shrink-0" />
							<span>Manage lists and campaigns</span>
						</li>
					</ul>
				</Card>

				<Card>
					<div class="flex items-center gap-2 mb-4">
						<Globe class="w-5 h-5 text-blue-500" />
						<h3 class="font-semibold text-text">Timezone Setting</h3>
					</div>
					<p class="text-sm text-text-muted mb-2">
						Your timezone is used for:
					</p>
					<ul class="space-y-2 text-sm text-text-muted">
						<li>• Scheduling campaign sends</li>
						<li>• Displaying report timestamps</li>
						<li>• Email queue timing</li>
					</ul>
				</Card>

				<Card>
					<div class="flex items-center gap-2 mb-4">
						<HelpCircle class="w-5 h-5 text-amber-500" />
						<h3 class="font-semibold text-text">Password Requirements</h3>
					</div>
					<ul class="space-y-2 text-sm text-text-muted">
						<li>Minimum 8 characters</li>
						<li>Use a strong, unique password</li>
						<li>Consider using a password manager</li>
					</ul>
				</Card>
			{:else if currentStep === 'aws'}
				<!-- AWS Help -->
				<Card>
					<div class="flex items-center gap-2 mb-4">
						<Key class="w-5 h-5 text-primary" />
						<h3 class="font-semibold text-text">Getting AWS Credentials</h3>
					</div>
					<ol class="space-y-3 text-sm text-text-muted list-decimal list-inside">
						<li>Go to AWS IAM Console</li>
						<li>Create a new IAM user</li>
						<li>Attach the policy below</li>
						<li>Copy the Access Key ID and Secret</li>
					</ol>
					<a
						href="https://console.aws.amazon.com/iam/"
						target="_blank"
						rel="noopener noreferrer"
						class="mt-4 inline-flex items-center gap-1 text-sm text-primary hover:underline"
					>
						Open IAM Console
						<ExternalLink class="w-3 h-3" />
					</a>
				</Card>

				<AwsIamPolicy compact />
			{:else if currentStep === 'complete'}
				<!-- Complete Help -->
				<Card>
					<div class="flex items-center gap-2 mb-4">
						<HelpCircle class="w-5 h-5 text-amber-500" />
						<h3 class="font-semibold text-text">What's next?</h3>
					</div>
					<ul class="space-y-3 text-sm text-text-muted">
						<li class="flex items-start gap-2">
							<span class="text-primary font-medium shrink-0">1.</span>
							<span>Create your first brand with a domain</span>
						</li>
						<li class="flex items-start gap-2">
							<span class="text-primary font-medium shrink-0">2.</span>
							<span>Add DNS records to verify your domain</span>
						</li>
						<li class="flex items-start gap-2">
							<span class="text-primary font-medium shrink-0">3.</span>
							<span>Create email lists and start collecting subscribers</span>
						</li>
						<li class="flex items-start gap-2">
							<span class="text-primary font-medium shrink-0">4.</span>
							<span>Send your first campaign!</span>
						</li>
					</ul>
				</Card>
			{/if}
		</div>
	</div>
</div>
{/if}
