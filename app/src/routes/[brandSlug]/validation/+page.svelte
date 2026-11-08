<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		adminValidateEmail,
		listValidationJobs,
		createValidationJob,
		appendValidationJobEmails,
		startValidationJob,
		getValidationUsage,
		listLists,
		type ValidationResult,
		type ValidationJobInfo,
		type ValidationUsageResponse,
		type ListInfo
	} from '$lib/api';
	import {
		Button,
		Card,
		Input,
		Alert,
		LoadingSpinner,
		Badge,
		EmptyState,
		Modal,
		Select,
		ProgressBar,
		ValidationBadge,
		ValidationBar
	} from '$lib/components/ui';
	import { MailCheck, Plus, Search, Upload, Users, ListChecks } from '@lucide/svelte';
	import {
		VALIDATION_STATUSES,
		ACTIVE_JOB_STATUSES,
		EMAIL_CHUNK_SIZE,
		subStatusLabel,
		jobStatusVariant,
		parseCSV,
		detectEmailColumn,
		extractEmails,
		formatDate
	} from '$lib/validation';

	let basePath = $derived(`/${$page.params.brandSlug}`);

	// ---------- Single check ----------
	let checkEmail = $state('');
	let checking = $state(false);
	let checkError = $state('');
	let checkResult = $state<ValidationResult | null>(null);

	async function runCheck(email?: string) {
		if (email !== undefined) checkEmail = email;
		const value = checkEmail.trim();
		if (!value) return;
		checking = true;
		checkError = '';
		try {
			checkResult = await adminValidateEmail({ email: value });
			loadUsage();
		} catch (err: any) {
			checkError = err.message || 'Validation failed';
			checkResult = null;
		} finally {
			checking = false;
		}
	}

	// ---------- Usage ----------
	let usage = $state<ValidationUsageResponse | null>(null);

	async function loadUsage() {
		try {
			usage = await getValidationUsage({});
		} catch (err) {
			console.error('Failed to load validation usage:', err);
		}
	}

	const SOURCE_LABELS: Record<string, string> = {
		api: 'API',
		admin: 'Dashboard',
		job: 'Bulk jobs',
		mcp: 'MCP'
	};

	let usageSources = $derived(
		Object.entries(usage?.by_source || {})
			.filter(([, n]) => n > 0)
			.map(([k, n]) => `${SOURCE_LABELS[k] || k} ${n.toLocaleString()}`)
			.join(' · ')
	);

	// ---------- Jobs ----------
	const JOBS_LIMIT = 20;
	let jobs = $state<ValidationJobInfo[]>([]);
	let jobsTotal = $state(0);
	let jobsPage = $state(1);
	let jobsLoading = $state(true);
	let error = $state('');

	let hasActiveJobs = $derived(jobs.some((j) => ACTIVE_JOB_STATUSES.includes(j.status)));
	let jobsPages = $derived(Math.max(1, Math.ceil(jobsTotal / JOBS_LIMIT)));

	async function loadJobs(showSpinner = false) {
		if (showSpinner) jobsLoading = true;
		try {
			const res = await listValidationJobs({ page: jobsPage, limit: JOBS_LIMIT });
			jobs = res.jobs || [];
			jobsTotal = res.total || 0;
		} catch (err: any) {
			error = err.message || 'Failed to load validation jobs';
		} finally {
			jobsLoading = false;
		}
	}

	$effect(() => {
		loadUsage();
	});

	$effect(() => {
		jobsPage;
		loadJobs(true);
	});

	// Poll every 3s while any job is pending/processing
	$effect(() => {
		if (!hasActiveJobs) return;
		const timer = setInterval(() => loadJobs(), 3000);
		return () => clearInterval(timer);
	});

	// ---------- New job dialog ----------
	let showNewJob = $state(false);
	let jobSource = $state<'list' | 'upload'>('list');
	let jobName = $state('');
	let lists = $state<ListInfo[]>([]);
	let listsLoaded = $state(false);
	let selectedListId = $state('');
	let creating = $state(false);
	let createError = $state('');

	// CSV upload state (raw: large arrays, no deep reactivity needed)
	let csvRows = $state.raw<string[][]>([]);
	let csvFileName = $state('');
	let csvHasHeader = $state(false);
	let csvColumn = $state('0');
	let uploadDone = $state(0);
	let uploadTotal = $state(0);

	let csvEmails = $derived(
		csvRows.length ? extractEmails(csvRows, Number(csvColumn), csvHasHeader) : []
	);
	let csvDataRows = $derived(Math.max(0, csvRows.length - (csvHasHeader ? 1 : 0)));
	let csvColumnOptions = $derived.by(() => {
		if (!csvRows.length) return [];
		const width = Math.max(...csvRows.slice(0, 50).map((r) => r.length));
		return Array.from({ length: width }, (_, i) => {
			const header = csvHasHeader ? (csvRows[0][i] || '').trim() : '';
			const sample = (csvRows[csvHasHeader ? 1 : 0]?.[i] || '').trim();
			const name = header || `Column ${i + 1}`;
			return { value: String(i), label: sample ? `${name} (e.g. ${sample})` : name };
		});
	});

	let canCreate = $derived(
		!creating &&
			(jobSource === 'list' ? !!selectedListId : csvEmails.length > 0)
	);

	async function openNewJob() {
		jobSource = 'list';
		jobName = '';
		selectedListId = '';
		createError = '';
		resetCsv();
		showNewJob = true;
		if (!listsLoaded) {
			try {
				const res = await listLists();
				lists = res.lists || [];
			} catch (err) {
				console.error('Failed to load lists:', err);
			} finally {
				listsLoaded = true;
			}
		}
	}

	function resetCsv() {
		csvRows = [];
		csvFileName = '';
		csvHasHeader = false;
		csvColumn = '0';
		uploadDone = 0;
		uploadTotal = 0;
	}

	async function handleFile(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		createError = '';
		if (!file) {
			resetCsv();
			return;
		}
		try {
			const text = await file.text();
			const rows = parseCSV(text);
			const detected = detectEmailColumn(rows);
			csvRows = rows;
			csvFileName = file.name;
			csvHasHeader = detected.hasHeader;
			csvColumn = String(detected.column);
			if (!jobName) jobName = file.name.replace(/\.[^.]+$/, '');
		} catch (err: any) {
			createError = err.message || 'Could not read file';
			resetCsv();
		}
	}

	async function submitNewJob() {
		creating = true;
		createError = '';
		const name = jobName.trim() || undefined;
		try {
			let job: ValidationJobInfo;
			if (jobSource === 'list') {
				job = await createValidationJob({ name, list_id: selectedListId });
			} else {
				const emails = csvEmails;
				uploadTotal = emails.length;
				uploadDone = 0;
				job = await createValidationJob({ name, emails: emails.slice(0, EMAIL_CHUNK_SIZE) });
				uploadDone = Math.min(EMAIL_CHUNK_SIZE, emails.length);
				for (let i = EMAIL_CHUNK_SIZE; i < emails.length; i += EMAIL_CHUNK_SIZE) {
					await appendValidationJobEmails({}, { emails: emails.slice(i, i + EMAIL_CHUNK_SIZE) }, job.id);
					uploadDone = Math.min(i + EMAIL_CHUNK_SIZE, emails.length);
				}
				job = await startValidationJob({}, job.id);
			}
			showNewJob = false;
			resetCsv();
			goto(`${basePath}/validation/${job.id}`);
		} catch (err: any) {
			createError = err.message || 'Failed to create validation job';
			loadJobs();
		} finally {
			creating = false;
		}
	}
</script>

<svelte:head>
	<title>Validation | Outlet</title>
</svelte:head>

<div class="p-6 max-w-5xl mx-auto">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
		<div>
			<h1 class="text-2xl font-semibold text-text">Email Validation</h1>
			<p class="mt-1 text-sm text-text-muted">
				Verify addresses before you send. Check one email or validate a whole list or CSV.
			</p>
			{#if usage}
				<p class="mt-2 text-xs text-text-muted">
					<span class="font-medium text-text">{usage.total.toLocaleString()}</span> checks this month{#if usageSources}&nbsp;({usageSources}){/if}
				</p>
			{/if}
		</div>
		<Button type="primary" onclick={openNewJob}>
			<Plus class="mr-2 h-4 w-4" />
			New Job
		</Button>
	</div>

	{#if error}
		<Alert type="error" title="Error" class="mb-4">
			<p>{error}</p>
		</Alert>
	{/if}

	<!-- Single check -->
	<Card class="mb-6">
		<h2 class="text-lg font-medium text-text mb-4">Check an email</h2>
		<form
			class="flex flex-col sm:flex-row gap-3"
			onsubmit={(e) => {
				e.preventDefault();
				runCheck();
			}}
		>
			<Input type="email" bind:value={checkEmail} placeholder="someone@example.com" class="flex-1" />
			<Button type="primary" htmlType="submit" disabled={!checkEmail.trim() || checking}>
				{#if checking}
					Checking...
				{:else}
					<Search class="mr-2 h-4 w-4" />
					Validate
				{/if}
			</Button>
		</form>

		{#if checkError}
			<Alert type="error" class="mt-4">
				<p>{checkError}</p>
			</Alert>
		{/if}

		{#if checkResult}
			<div class="mt-5 border-t border-border pt-5">
				<div class="flex flex-wrap items-center gap-3 mb-4">
					<span class="font-medium text-text break-all">{checkResult.address}</span>
					<ValidationBadge status={checkResult.status} showSubStatus={false} size="md" />
					{#if checkResult.sub_status}
						<span class="text-sm text-text-muted">{subStatusLabel(checkResult.sub_status)}</span>
					{/if}
				</div>

				{#if checkResult.did_you_mean}
					<Alert type="info" class="mb-4">
						<p>
							Did you mean
							<Button type="link" size="sm" onclick={() => runCheck(checkResult?.did_you_mean)}>
								{checkResult.did_you_mean}
							</Button>?
						</p>
					</Alert>
				{/if}

				<dl class="vresult-grid">
					<div class="flex justify-between gap-4">
						<dt>Domain</dt>
						<dd>{checkResult.domain || '-'}</dd>
					</div>
					<div class="flex justify-between gap-4">
						<dt>Account</dt>
						<dd>{checkResult.account || '-'}</dd>
					</div>
					<div class="flex justify-between gap-4">
						<dt>Free email</dt>
						<dd>{checkResult.free_email ? 'Yes' : 'No'}</dd>
					</div>
					<div class="flex justify-between gap-4">
						<dt>SMTP provider</dt>
						<dd>{checkResult.smtp_provider || '-'}</dd>
					</div>
					<div class="flex justify-between gap-4">
						<dt>MX found</dt>
						<dd>{checkResult.mx_found ? 'Yes' : 'No'}</dd>
					</div>
					<div class="flex justify-between gap-4">
						<dt>MX record</dt>
						<dd>{checkResult.mx_record || '-'}</dd>
					</div>
					<div class="flex justify-between gap-4">
						<dt>Domain age</dt>
						<dd>
							{checkResult.domain_age_days ? `${checkResult.domain_age_days.toLocaleString()} days` : 'Unknown'}
						</dd>
					</div>
					<div class="flex justify-between gap-4">
						<dt>Checked</dt>
						<dd>{formatDate(checkResult.processed_at)}</dd>
					</div>
				</dl>
			</div>
		{/if}
	</Card>

	<!-- Jobs -->
	<div class="flex items-center justify-between mb-3">
		<h2 class="text-lg font-medium text-text">Bulk jobs</h2>
		{#if hasActiveJobs}
			<span class="text-xs text-text-muted flex items-center gap-2">
				<LoadingSpinner size="small" /> Updating
			</span>
		{/if}
	</div>

	{#if jobsLoading}
		<div class="flex justify-center py-12">
			<LoadingSpinner />
		</div>
	{:else if jobs.length === 0}
		<EmptyState
			icon={ListChecks}
			title="No validation jobs yet"
			description="Validate an entire list or upload a CSV to clean it before sending."
		>
			<Button type="primary" onclick={openNewJob}>
				<Plus class="mr-2 h-4 w-4" />
				New Job
			</Button>
		</EmptyState>
	{:else}
		<div class="data-table overflow-x-auto">
			<table class="w-full text-sm">
				<thead>
					<tr>
						<th class="text-left">Job</th>
						<th class="text-left">Status</th>
						<th class="text-left w-48">Progress</th>
						<th class="text-left">Results</th>
						<th class="text-left">Created</th>
					</tr>
				</thead>
				<tbody>
					{#each jobs as job (job.id)}
						<tr>
							<td>
								<a href="{basePath}/validation/{job.id}" class="block group">
									<span class="font-medium text-text group-hover:text-primary transition-colors">
										{job.name || 'Untitled job'}
									</span>
									<span class="flex items-center gap-1 text-xs text-text-muted">
										{#if job.source === 'list'}
											<Users class="h-3 w-3" /> List
										{:else}
											<Upload class="h-3 w-3" /> Upload
										{/if}
									</span>
								</a>
							</td>
							<td>
								<Badge variant={jobStatusVariant(job.status)} size="sm">{job.status}</Badge>
							</td>
							<td>
								<ProgressBar
									value={job.processed}
									max={job.total || 1}
									showPercent={false}
									size="sm"
									variant={job.status === 'failed' ? 'error' : job.status === 'completed' ? 'success' : 'default'}
								/>
								<span class="text-xs text-text-muted">
									{job.processed.toLocaleString()} / {job.total.toLocaleString()}
								</span>
							</td>
							<td>
								{#if job.processed > 0}
									<ValidationBar {job} class="mb-1.5" />
									<div class="flex flex-wrap gap-x-3 gap-y-1 text-xs text-text-muted">
										{#each VALIDATION_STATUSES as s (s.key)}
											{#if Number(job[s.countKey]) > 0}
												<span class="flex items-center gap-1" title={s.label}>
													<span class="vstatus-dot {s.barClass}"></span>
													{Number(job[s.countKey]).toLocaleString()}
												</span>
											{/if}
										{/each}
									</div>
								{:else}
									<span class="text-text-muted">-</span>
								{/if}
							</td>
							<td class="text-text-muted whitespace-nowrap">{formatDate(job.created_at)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if jobsPages > 1}
			<div class="flex items-center justify-between mt-4 text-sm text-text-muted">
				<span>Page {jobsPage} of {jobsPages}</span>
				<div class="flex gap-2">
					<Button type="secondary" size="sm" disabled={jobsPage <= 1} onclick={() => jobsPage--}>Previous</Button>
					<Button type="secondary" size="sm" disabled={jobsPage >= jobsPages} onclick={() => jobsPage++}>Next</Button>
				</div>
			</div>
		{/if}
	{/if}
</div>

<!-- New Job Modal -->
<Modal bind:show={showNewJob} title="New Validation Job" size="lg">
	<div class="space-y-4">
		<div class="tab-nav">
			<button
				type="button"
				class="tab-nav-item"
				class:active={jobSource === 'list'}
				disabled={creating}
				onclick={() => (jobSource = 'list')}
			>
				<Users class="h-4 w-4" /> From a list
			</button>
			<button
				type="button"
				class="tab-nav-item"
				class:active={jobSource === 'upload'}
				disabled={creating}
				onclick={() => (jobSource = 'upload')}
			>
				<Upload class="h-4 w-4" /> Upload CSV
			</button>
		</div>

		{#if createError}
			<Alert type="error">
				<p>{createError}</p>
			</Alert>
		{/if}

		<div>
			<label for="job-name" class="form-label">Job name (optional)</label>
			<Input id="job-name" bind:value={jobName} placeholder="Spring cleanup" disabled={creating} />
		</div>

		{#if jobSource === 'list'}
			<div>
				<label for="job-list" class="form-label">List</label>
				{#if !listsLoaded}
					<LoadingSpinner size="small" />
				{:else if lists.length === 0}
					<p class="text-sm text-text-muted">No lists yet. Create a list first, or upload a CSV.</p>
				{:else}
					<Select
						id="job-list"
						bind:value={selectedListId}
						disabled={creating}
						options={[
							{ value: '', label: 'Select a list...' },
							...lists.map((l) => ({
								value: l.id,
								label: `${l.name} (${(l.subscriber_count || 0).toLocaleString()} subscribers)`
							}))
						]}
					/>
					<p class="form-hint">Validation starts immediately for all subscribers on the list.</p>
				{/if}
			</div>
		{:else}
			<div>
				<label for="job-file" class="form-label">CSV file</label>
				<input
					id="job-file"
					type="file"
					accept=".csv,.txt,text/csv,text/plain"
					class="file-input file-input-bordered w-full"
					disabled={creating}
					onchange={handleFile}
				/>
				<p class="form-hint">Quoted fields are supported. Duplicates and malformed addresses are skipped.</p>
			</div>

			{#if csvRows.length > 0}
				<div>
					<label for="job-column" class="form-label">Email column</label>
					<Select id="job-column" bind:value={csvColumn} options={csvColumnOptions} disabled={creating} />
				</div>
				<p class="text-sm text-text-muted">
					<span class="font-medium text-text">{csvFileName}</span>:
					{csvDataRows.toLocaleString()} rows,
					<span class="font-medium text-text">{csvEmails.length.toLocaleString()}</span> unique valid-looking emails
				</p>
			{/if}

			{#if creating && uploadTotal > 0}
				<ProgressBar
					value={uploadDone}
					max={uploadTotal}
					label="Uploading {uploadDone.toLocaleString()} / {uploadTotal.toLocaleString()}"
				/>
			{/if}
		{/if}
	</div>

	{#snippet footer()}
		<Button type="secondary" onclick={() => (showNewJob = false)} disabled={creating}>Cancel</Button>
		<Button type="primary" onclick={submitNewJob} disabled={!canCreate}>
			{#if creating}
				{jobSource === 'upload' ? 'Uploading...' : 'Creating...'}
			{:else}
				<MailCheck class="mr-2 h-4 w-4" />
				Start Validation
			{/if}
		</Button>
	{/snippet}
</Modal>
