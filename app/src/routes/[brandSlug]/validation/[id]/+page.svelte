<script lang="ts">
	import { formatDate } from '$lib/dates';
	import { page } from '$app/stores';
	import {
		getValidationJob,
		listValidationResults,
		cancelValidationJob,
		applyValidationJob,
		type ValidationJobInfo,
		type ValidationResult
	} from '$lib/api';
	import {
		Button,
		Card,
		Alert,
		LoadingSpinner,
		Badge,
		EmptyState,
		Select,
		ProgressBar,
		Checkbox,
		AlertDialog,
		Toast,
		ValidationBadge,
		ValidationBar
	} from '$lib/components/ui';
	import { ArrowLeft, Download, ShieldBan, XCircle, Users, Upload, Search } from '@lucide/svelte';
	import {
		VALIDATION_STATUSES,
		DEFAULT_SUPPRESS_STATUSES,
		ACTIVE_JOB_STATUSES,
		subStatusLabel,
		jobStatusVariant,
		percent,
		resultsToCSV,
		downloadFile
	} from '$lib/validation';

	let jobId = $derived($page.params.id!);
	let basePath = $derived(`/${$page.params.brandSlug}`);

	let job = $state<ValidationJobInfo | null>(null);
	let loading = $state(true);
	let error = $state('');

	let isActive = $derived(!!job && ACTIVE_JOB_STATUSES.includes(job.status));

	// ---------- Results ----------
	const RESULTS_LIMIT = 50;
	let results = $state<ValidationResult[]>([]);
	let resultsTotal = $state(0);
	let resultsPage = $state(1);
	let resultsLoading = $state(true);
	let statusFilter = $state('');

	let resultsPages = $derived(Math.max(1, Math.ceil(resultsTotal / RESULTS_LIMIT)));
	let rangeStart = $derived(resultsTotal === 0 ? 0 : (resultsPage - 1) * RESULTS_LIMIT + 1);
	let rangeEnd = $derived(Math.min(resultsPage * RESULTS_LIMIT, resultsTotal));

	const filterOptions = [
		{ value: '', label: 'All statuses' },
		...VALIDATION_STATUSES.map((s) => ({ value: s.key, label: s.label }))
	];

	async function loadJob() {
		try {
			job = await getValidationJob({}, jobId);
		} catch (err: any) {
			error = err.message || 'Failed to load validation job';
		} finally {
			loading = false;
		}
	}

	async function loadResults(showSpinner = false) {
		if (showSpinner) resultsLoading = true;
		try {
			const res = await listValidationResults(
				{ ...(statusFilter ? { status: statusFilter } : {}), page: resultsPage, limit: RESULTS_LIMIT },
				jobId
			);
			results = res.results || [];
			resultsTotal = res.total || 0;
		} catch (err: any) {
			error = err.message || 'Failed to load results';
		} finally {
			resultsLoading = false;
		}
	}

	$effect(() => {
		jobId;
		loading = true;
		loadJob();
	});

	$effect(() => {
		jobId;
		statusFilter;
		resultsPage;
		loadResults(true);
	});

	// Poll every 3s while the job is pending/processing
	$effect(() => {
		if (!isActive) return;
		const timer = setInterval(() => {
			loadJob();
			loadResults();
		}, 3000);
		return () => clearInterval(timer);
	});

	function setFilter(status: string) {
		statusFilter = statusFilter === status ? '' : status;
		resultsPage = 1;
	}

	function onFilterChange(value: string | number | undefined) {
		statusFilter = String(value ?? '');
		resultsPage = 1;
	}

	// ---------- Breakdown ----------
	let breakdown = $derived(
		job
			? VALIDATION_STATUSES.map((s) => {
					const count = Number(job![s.countKey] || 0);
					return { ...s, count, pct: percent(count, job!.processed) };
				})
			: []
	);

	// ---------- Cancel ----------
	let showCancelConfirm = $state(false);
	let cancelling = $state(false);

	async function executeCancel() {
		cancelling = true;
		try {
			job = await cancelValidationJob({}, jobId);
			showToast('Validation job cancelled', 'info');
		} catch (err: any) {
			error = err.message || 'Failed to cancel job';
		} finally {
			cancelling = false;
		}
	}

	// ---------- Export ----------
	let exporting = $state(false);
	let exportProgress = $state(0);

	async function exportCSV() {
		if (!job) return;
		exporting = true;
		exportProgress = 0;
		error = '';
		const limit = 500;
		const all: ValidationResult[] = [];
		try {
			for (let p = 1; ; p++) {
				const res = await listValidationResults(
					{ ...(statusFilter ? { status: statusFilter } : {}), page: p, limit },
					jobId
				);
				const batch = res.results || [];
				all.push(...batch);
				exportProgress = all.length;
				if (batch.length < limit || all.length >= (res.total || 0)) break;
			}
			const safeName = (job.name || 'validation').replace(/[^a-z0-9-_]+/gi, '-').toLowerCase();
			const suffix = statusFilter ? `-${statusFilter}` : '';
			downloadFile(`${safeName}${suffix}-results.csv`, resultsToCSV(all));
		} catch (err: any) {
			error = err.message || 'Failed to export results';
		} finally {
			exporting = false;
		}
	}

	// ---------- Suppress ----------
	let suppressStatuses = $state<Record<string, boolean>>(
		Object.fromEntries(VALIDATION_STATUSES.map((s) => [s.key, DEFAULT_SUPPRESS_STATUSES.includes(s.key)]))
	);
	let showSuppressConfirm = $state(false);
	let suppressing = $state(false);

	let selectedSuppress = $derived(VALIDATION_STATUSES.filter((s) => suppressStatuses[s.key]));
	let suppressCount = $derived(selectedSuppress.reduce((sum, s) => sum + Number(job?.[s.countKey] || 0), 0));

	async function executeSuppress() {
		if (selectedSuppress.length === 0) return;
		suppressing = true;
		try {
			const res = await applyValidationJob({}, { statuses: selectedSuppress.map((s) => s.key) }, jobId);
			showToast(`${res.suppressed.toLocaleString()} addresses added to the suppression list`, 'success');
		} catch (err: any) {
			error = err.message || 'Failed to suppress addresses';
		} finally {
			suppressing = false;
		}
	}

	// ---------- Toast ----------
	let toastShow = $state(false);
	let toastMessage = $state('');
	let toastType = $state<'success' | 'info'>('success');

	function showToast(message: string, type: 'success' | 'info') {
		toastMessage = message;
		toastType = type;
		toastShow = true;
	}
</script>

<svelte:head>
	<title>{job?.name || 'Validation Job'} | Outlet</title>
</svelte:head>

<div class="p-6 max-w-5xl mx-auto">
	<a href="{basePath}/validation" class="inline-flex items-center gap-1 text-sm text-text-muted hover:text-text mb-4">
		<ArrowLeft class="h-4 w-4" /> Validation
	</a>

	{#if error}
		<Alert type="error" title="Error" class="mb-4">
			<p>{error}</p>
		</Alert>
	{/if}

	{#if loading}
		<div class="flex justify-center py-12">
			<LoadingSpinner size="large" />
		</div>
	{:else if job}
		<!-- Header -->
		<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
			<div>
				<div class="flex items-center gap-3 flex-wrap">
					<h1 class="text-2xl font-semibold text-text">{job.name || 'Untitled job'}</h1>
					<Badge variant={jobStatusVariant(job.status)} size="sm">{job.status}</Badge>
				</div>
				<p class="mt-1 text-sm text-text-muted flex items-center gap-1 flex-wrap">
					{#if job.source === 'list'}
						<Users class="h-3.5 w-3.5" /> List
					{:else}
						<Upload class="h-3.5 w-3.5" /> Upload
					{/if}
					· {job.total.toLocaleString()} emails · Created {formatDate(job.created_at, 'datetime') || '-'}
					{#if job.completed_at}
						· Completed {formatDate(job.completed_at, 'datetime') || '-'}
					{/if}
				</p>
			</div>
			<div class="flex gap-2 flex-wrap">
				{#if isActive}
					<Button type="danger" outline onclick={() => (showCancelConfirm = true)} disabled={cancelling}>
						<XCircle class="mr-2 h-4 w-4" />
						{cancelling ? 'Cancelling...' : 'Cancel'}
					</Button>
				{/if}
				<Button type="secondary" onclick={exportCSV} disabled={exporting || job.processed === 0}>
					<Download class="mr-2 h-4 w-4" />
					{exporting ? `Exporting ${exportProgress.toLocaleString()}...` : 'Export CSV'}
				</Button>
				<Button type="primary" onclick={() => (showSuppressConfirm = true)} disabled={suppressing || job.processed === 0}>
					<ShieldBan class="mr-2 h-4 w-4" />
					{suppressing ? 'Suppressing...' : 'Suppress'}
				</Button>
			</div>
		</div>

		{#if job.error}
			<Alert type="error" title="Job error" class="mb-4">
				<p>{job.error}</p>
			</Alert>
		{/if}

		<!-- Summary -->
		<Card class="mb-6">
			<div class="flex items-center justify-between mb-2">
				<h2 class="text-lg font-medium text-text">Summary</h2>
				<span class="text-sm text-text-muted">
					{job.processed.toLocaleString()} of {job.total.toLocaleString()} processed
				</span>
			</div>

			{#if isActive}
				<div class="mb-4">
					<ProgressBar value={job.processed} max={job.total || 1} size="sm" />
				</div>
			{/if}

			<ValidationBar {job} class="mb-4" />

			<div class="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-3">
				{#each breakdown as s (s.key)}
					<button
						type="button"
						class="vstatus-tile"
						class:active={statusFilter === s.key}
						onclick={() => setFilter(s.key)}
						title="Filter results by {s.label}"
					>
						<span class="flex items-center gap-1.5 text-xs text-text-muted">
							<span class="vstatus-dot {s.barClass}"></span>
							{s.label}
						</span>
						<span class="text-lg font-semibold text-text">{s.count.toLocaleString()}</span>
						<span class="text-xs text-text-muted">{s.pct.toFixed(1)}%</span>
					</button>
				{/each}
			</div>
		</Card>

		<!-- Results -->
		<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-3">
			<h2 class="text-lg font-medium text-text">Results</h2>
			<div class="w-full sm:w-56">
				<Select
					size="sm"
					options={filterOptions}
					bind:value={() => statusFilter, onFilterChange}
				/>
			</div>
		</div>

		{#if resultsLoading}
			<div class="flex justify-center py-12">
				<LoadingSpinner />
			</div>
		{:else if results.length === 0}
			<EmptyState
				icon={Search}
				title={isActive ? 'Results are on the way' : 'No results'}
				description={isActive
					? 'Results appear here as addresses are validated.'
					: statusFilter
						? 'No addresses with this status.'
						: 'This job has no results.'}
			/>
		{:else}
			<div class="data-table overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr>
							<th class="text-left">Email</th>
							<th class="text-left">Status</th>
							<th class="text-left">Reason</th>
							<th class="text-left">Domain</th>
							<th class="text-left">Free</th>
							<th class="text-left">Suggestion</th>
						</tr>
					</thead>
					<tbody>
						{#each results as r (r.address)}
							<tr>
								<td class="font-medium text-text break-all">{r.address}</td>
								<td><ValidationBadge status={r.status} showSubStatus={false} /></td>
								<td class="text-text-muted">{subStatusLabel(r.sub_status) || '-'}</td>
								<td class="text-text-muted">{r.domain || '-'}</td>
								<td class="text-text-muted">{r.free_email ? 'Yes' : 'No'}</td>
								<td class="text-text-muted">{r.did_you_mean || '-'}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<div class="flex items-center justify-between mt-4 text-sm text-text-muted">
				<span>{rangeStart.toLocaleString()}–{rangeEnd.toLocaleString()} of {resultsTotal.toLocaleString()}</span>
				{#if resultsPages > 1}
					<div class="flex gap-2">
						<Button type="secondary" size="sm" disabled={resultsPage <= 1} onclick={() => resultsPage--}>Previous</Button>
						<Button type="secondary" size="sm" disabled={resultsPage >= resultsPages} onclick={() => resultsPage++}>Next</Button>
					</div>
				{/if}
			</div>
		{/if}
	{/if}
</div>

<AlertDialog
	bind:open={showCancelConfirm}
	title="Cancel Validation Job"
	description="Stop validating the remaining addresses? Results so far are kept."
	actionLabel="Cancel Job"
	cancelLabel="Keep Running"
	actionType="danger"
	onAction={executeCancel}
	onCancel={() => (showCancelConfirm = false)}
/>

<AlertDialog
	bind:open={showSuppressConfirm}
	title="Suppress Addresses"
	description="Add addresses with the selected statuses to this brand's suppression list. Suppressed addresses won't receive any email."
	actionLabel={selectedSuppress.length === 0 ? 'Select a status' : `Suppress ${suppressCount.toLocaleString()}`}
	actionType="danger"
	loading={suppressing}
	onAction={executeSuppress}
	onCancel={() => (showSuppressConfirm = false)}
>
	<div class="grid grid-cols-1 sm:grid-cols-2 gap-2 mb-2">
		{#each VALIDATION_STATUSES as s (s.key)}
			<Checkbox
				bind:checked={suppressStatuses[s.key]}
				label="{s.label} ({Number(job?.[s.countKey] || 0).toLocaleString()})"
				size="sm"
			/>
		{/each}
	</div>
</AlertDialog>

<Toast bind:show={toastShow} message={toastMessage} type={toastType} />
