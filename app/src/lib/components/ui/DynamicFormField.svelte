<!--
  DynamicFormField Component
  Renders form fields dynamically based on field schema
-->

<script lang="ts">
	import Input from './Input.svelte';
	import Select from './Select.svelte';
	import Button from './Button.svelte';
	import TagInput from './TagInput.svelte';
	import Checkbox from './Checkbox.svelte';
	import Spinner from './Spinner.svelte';
	import Check from '@lucide/svelte/icons/check';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';

	// Types for resource_selector (feature requires external API)
	interface ResourceInfo {
		id: string;
		name: string;
		fullName?: string;
		description?: string;
	}

	interface OrgInfo {
		id: string;
		name?: string;
		slug?: string;
	}

	interface DiscoverResourcesRequest {
		page?: number;
		org?: string;
	}

	interface DiscoverResourcesResponse {
		resources: ResourceInfo[];
		organizations?: OrgInfo[];
		hasMore?: boolean;
		page: number;
	}

	// Stub function - resource_selector requires external API integration
	async function discoverResources(_params: object, _req: DiscoverResourcesRequest, _connectionId: string): Promise<DiscoverResourcesResponse> {
		throw new Error('Resource selector requires API integration');
	}

	interface AuthField {
		name: string;
		title: string;
		description?: string;
		type: string;
		required?: boolean;
		default?: string;
		options?: string[];
		secret?: boolean;
		placeholder?: string;
		discoverType?: string;
		valueField?: string;
		labelField?: string;
	}

	let {
		field,
		value = $bindable(),
		disabled = false,
		sourceConnectionId = ''
	}: {
		field: AuthField;
		value?: any;
		disabled?: boolean;
		sourceConnectionId?: string;
	} = $props();

	// Ensure value is never undefined for form binding
	$effect(() => {
		if (value === undefined) {
			value = field.default ?? '';
		}
	});

	let showPassword = $state(false);
	let arrayValue = $state<string[]>([]);

	// Resource selector state
	let resources = $state<ResourceInfo[]>([]);
	let selectedResources = $state<Set<string>>(new Set());
	let isLoadingResources = $state(false);
	let resourceError = $state<string | null>(null);
	let resourceSearchQuery = $state('');

	// Pagination and org filter state
	let organizations = $state<OrgInfo[]>([]);
	let selectedOrg = $state<string>('');
	let hasMoreResources = $state(false);
	let nextPage = $state<number>(0);
	let currentPage = $state<number>(1);
	let loadedOrgs = $state<Set<string>>(new Set()); // Track which orgs we've loaded

	// Track if this is a masked password field (value is placeholder from server)
	const isPasswordPlaceholder = $derived(
		(field.type === 'password' || field.secret) && value === '--------'
	);

	// Clear placeholder when user focuses the field
	function handlePasswordFocus() {
		if (isPasswordPlaceholder) {
			value = '';
		}
	}

	// Initialize array value if type is array
	$effect(() => {
		if (field.type === 'array') {
			if (Array.isArray(value)) {
				arrayValue = value;
			} else if (typeof value === 'string' && value) {
				arrayValue = value.split(',').map((s: string) => s.trim()).filter(Boolean);
			} else {
				arrayValue = [];
			}
		}
	});

	// Sync array value back to main value
	$effect(() => {
		if (field.type === 'array') {
			value = arrayValue;
		}
	});

	// Resource selector: get value field (default to 'fullName')
	const valueField = $derived(field.valueField || 'fullName');
	const labelField = $derived(field.labelField || 'name');

	// Track if we've loaded resources for this connection
	let hasLoadedResources = $state(false);
	let lastSourceConnectionId = $state('');

	// Resource selector: load resources on mount (only once per sourceConnectionId)
	$effect(() => {
		if (field.type === 'resource_selector' && sourceConnectionId && sourceConnectionId !== lastSourceConnectionId) {
			lastSourceConnectionId = sourceConnectionId;
			hasLoadedResources = false;
			loadResources();
		}
	});

	// Resource selector: initialize selected from value (only on initial mount)
	let hasInitializedSelection = $state(false);
	$effect(() => {
		if (field.type === 'resource_selector' && !hasInitializedSelection && Array.isArray(value) && value.length > 0) {
			hasInitializedSelection = true;
			selectedResources = new Set(value);
		}
	});

	// Filtered resources based on search AND org filter
	const filteredResources = $derived(
		resources.filter((r) => {
			// Filter by org if one is selected
			if (selectedOrg && r.fullName) {
				const orgPrefix = selectedOrg + '/';
				if (!r.fullName.startsWith(orgPrefix)) {
					return false;
				}
			}
			// Filter by search query
			const query = resourceSearchQuery.toLowerCase();
			const labelVal = getResourceField(r, labelField);
			const descVal = r.description || '';
			return labelVal.toLowerCase().includes(query) || descVal.toLowerCase().includes(query);
		})
	);

	// Get selected resources as array with full info for display
	const selectedResourcesList = $derived(
		resources.filter((r) => selectedResources.has(getResourceField(r, valueField)))
	);

	// Get a field value from a resource
	function getResourceField(resource: ResourceInfo, fieldName: string): string {
		if (fieldName === 'fullName') return resource.fullName || resource.name;
		if (fieldName === 'name') return resource.name;
		if (fieldName === 'id') return resource.id;
		return resource.name;
	}

	// Load resources from API
	async function loadResources(page: number = 1, append: boolean = false, orgOverride?: string) {
		if (!sourceConnectionId) return;
		isLoadingResources = true;
		resourceError = null;
		const orgToLoad = orgOverride !== undefined ? orgOverride : selectedOrg;
		try {
			// Build request - only include org if it has a value
			const req: DiscoverResourcesRequest = { page };
			if (orgToLoad) {
				req.org = orgToLoad;
			}
			const response = await discoverResources({}, req, sourceConnectionId);
			const newResources = response.resources || [];

			if (append) {
				// Append and deduplicate by id
				const existingIds = new Set(resources.map((r) => r.id));
				const uniqueNew = newResources.filter((r) => !existingIds.has(r.id));
				resources = [...resources, ...uniqueNew];
			} else {
				// For initial load, just set resources
				resources = newResources;
			}

			// Always update orgs list
			if (response.organizations && response.organizations.length > 0) {
				organizations = response.organizations;
			}

			// Track pagination for current org
			hasMoreResources = response.hasMore || false;
			nextPage = response.page + 1;
			currentPage = page;
			hasLoadedResources = true;

			// Track that we've loaded this org
			loadedOrgs.add(orgToLoad || '');
			loadedOrgs = new Set(loadedOrgs);
		} catch (err: any) {
			resourceError = err.message || 'Failed to load resources';
		} finally {
			isLoadingResources = false;
		}
	}

	// Load more resources (pagination)
	function loadMoreResources() {
		if (hasMoreResources && nextPage > 0) {
			loadResources(nextPage, true);
		}
	}

	// Track previous org to detect changes
	let prevOrg = $state<string | null>(null);

	// Watch for org filter changes - load that org's repos
	$effect(() => {
		if (field.type === 'resource_selector' && hasLoadedResources && selectedOrg !== prevOrg) {
			if (selectedOrg && !loadedOrgs.has(selectedOrg)) {
				// Load this org's repos and append to existing
				loadResources(1, true, selectedOrg);
			}
			prevOrg = selectedOrg;
		}
	});

	// Toggle resource selection
	function toggleResource(resourceValue: string) {
		if (selectedResources.has(resourceValue)) {
			selectedResources.delete(resourceValue);
		} else {
			selectedResources.add(resourceValue);
		}
		selectedResources = new Set(selectedResources);
		// Sync to parent value
		value = Array.from(selectedResources);
	}

	// Select all filtered resources
	function selectAllResources() {
		for (const r of filteredResources) {
			selectedResources.add(getResourceField(r, valueField));
		}
		selectedResources = new Set(selectedResources);
		// Sync to parent value
		value = Array.from(selectedResources);
	}

	// Deselect all resources
	function deselectAllResources() {
		selectedResources = new Set();
		// Sync to parent value
		value = [];
	}

</script>

<div class="space-y-1.5">
	<label class="label pb-0">
		<span class="label-text font-medium flex items-center gap-1.5">
			{field.title}
			{#if field.required}
				<span class="text-error">*</span>
			{/if}
		</span>
	</label>

	{#if field.description}
		<p class="text-xs text-base-content/60">{field.description}</p>
	{/if}

	{#if field.type === 'password' || field.secret}
		<div class="relative">
			<Input
				type={showPassword ? 'text' : 'password'}
				class="pr-10"
				placeholder={isPasswordPlaceholder ? 'Leave blank to keep existing' : (field.placeholder || '')}
				bind:value
				onfocus={handlePasswordFocus}
				{disabled}
			/>
			<button
				type="button"
				class="absolute right-2 top-1/2 -translate-y-1/2 btn btn-ghost btn-xs btn-circle"
				onclick={() => (showPassword = !showPassword)}
			>
				{#if showPassword}
					<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path>
						<line x1="1" y1="1" x2="23" y2="23"></line>
					</svg>
				{:else}
					<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
						<circle cx="12" cy="12" r="3"></circle>
					</svg>
				{/if}
			</button>
		</div>

	{:else if field.type === 'number'}
		<Input
			type="number"
			placeholder={field.placeholder || ''}
			bind:value
			{disabled}
		/>

	{:else if field.type === 'select' && field.options}
		<Select bind:value {disabled}>
			{#if !field.required}
				<option value="">Select...</option>
			{/if}
			{#each field.options as option}
				<option value={option}>{option}</option>
			{/each}
		</Select>

	{:else if field.type === 'boolean'}
		<Checkbox bind:checked={value} {disabled} />

	{:else if field.type === 'array'}
		<TagInput bind:value={arrayValue} placeholder={field.placeholder || 'Add items...'} {disabled} />

	{:else if field.type === 'resource_selector'}
		<!-- Resource Selector - Split Screen Layout -->
		{#if resourceError}
			<div class="alert alert-error">
				<span class="text-sm">{resourceError}</span>
			</div>
		{/if}

		{#if isLoadingResources && resources.length === 0}
			<div class="flex items-center justify-center py-6">
				<Spinner />
				<span class="ml-3 text-sm text-base-content/60">Loading resources...</span>
			</div>
		{:else}
			<div class="space-y-2">
				<!-- TOP: Org filter and search stacked -->
				{#if organizations.length > 0}
					<Select bind:value={selectedOrg} {disabled}>
						<option value="">All orgs</option>
						{#each organizations as org}
							<option value={org.slug ?? org.id}>{org.name || org.slug || org.id}</option>
						{/each}
					</Select>
				{/if}
				<div class="flex items-center gap-2">
					<Input
						type="text"
						placeholder="Search..."
						bind:value={resourceSearchQuery}
						{disabled}
					/>
					<Button
						type="secondary"
						onclick={() => loadResources(1, true, selectedOrg)}
						{disabled}
					>
						<RefreshCw class="h-4 w-4 {isLoadingResources ? 'animate-spin' : ''}" />
					</Button>
				</div>

				<!-- Two equal panels -->
				<div class="grid grid-cols-2 gap-3">
					<!-- LEFT PANEL: Available -->
					<div class="space-y-1.5">
						<div class="flex items-center justify-between text-xs">
							<span class="font-medium text-base-content/60">Available ({filteredResources.length})</span>
							<Button type="link" size="sm" onclick={selectAllResources} {disabled}>
								Select all
							</Button>
						</div>
						<div class="h-48 overflow-y-auto rounded-lg bg-base-200 border border-base-300">
							{#if filteredResources.length === 0}
								<div class="p-4 text-center text-sm text-base-content/60">
									{resourceSearchQuery ? 'No matches' : 'No resources'}
								</div>
							{:else}
								{#each filteredResources as resource (resource.id)}
									{@const resourceValue = getResourceField(resource, valueField)}
									{@const resourceLabel = getResourceField(resource, labelField)}
									{@const isSelected = selectedResources.has(resourceValue)}
									<button
										type="button"
										onclick={() => toggleResource(resourceValue)}
										{disabled}
										class="w-full flex items-center gap-2 px-2 py-1.5 hover:bg-base-300 transition-colors border-b border-base-300/50 last:border-b-0 text-left {isSelected ? 'bg-primary/10' : ''}"
									>
										<div
											class="h-4 w-4 rounded border flex items-center justify-center flex-shrink-0 transition-colors {isSelected
												? 'bg-primary border-primary'
												: 'border-base-300 bg-base-100'}"
										>
											{#if isSelected}
												<Check class="h-2.5 w-2.5 text-primary-content" />
											{/if}
										</div>
										<span class="text-xs truncate">{resourceLabel}</span>
									</button>
								{/each}
								{#if hasMoreResources}
									<div class="border-t border-base-300 p-1.5">
										<Button
											type="link"
											size="sm"
											onclick={loadMoreResources}
											disabled={disabled || isLoadingResources}
											class="w-full justify-center text-xs"
										>
											{#if isLoadingResources}
												<Spinner class="h-3 w-3 mr-1" />
												Loading...
											{:else}
												Load more
											{/if}
										</Button>
									</div>
								{/if}
							{/if}
						</div>
					</div>

					<!-- RIGHT PANEL: Selected -->
					<div class="space-y-1.5">
						<div class="flex items-center justify-between text-xs">
							<span class="font-medium text-base-content/60">Selected ({selectedResources.size})</span>
							<Button type="link" size="sm" onclick={deselectAllResources} {disabled}>
								Unselect all
							</Button>
						</div>
						<div class="h-48 overflow-y-auto rounded-lg bg-base-200 border border-base-300">
							{#if selectedResources.size === 0}
								<div class="p-4 text-center text-sm text-base-content/60">
									No repositories selected
								</div>
							{:else}
								{#each selectedResourcesList as resource (resource.id)}
									{@const resourceValue = getResourceField(resource, valueField)}
									{@const resourceLabel = getResourceField(resource, labelField)}
									<button
										type="button"
										onclick={() => toggleResource(resourceValue)}
										{disabled}
										class="w-full flex items-center gap-2 px-2 py-1.5 hover:bg-base-300 transition-colors border-b border-base-300/50 last:border-b-0 text-left"
									>
										<div class="h-4 w-4 rounded border flex items-center justify-center flex-shrink-0 bg-primary border-primary">
											<Check class="h-2.5 w-2.5 text-primary-content" />
										</div>
										<span class="text-xs truncate">{resource.fullName || resourceLabel}</span>
									</button>
								{/each}
								{@const loadedValues = new Set(resources.map((r) => getResourceField(r, valueField)))}
								{#each Array.from(selectedResources) as selectedValue}
									{#if !loadedValues.has(selectedValue)}
										<button
											type="button"
											onclick={() => toggleResource(selectedValue)}
											{disabled}
											class="w-full flex items-center gap-2 px-2 py-1.5 hover:bg-base-300 transition-colors border-b border-base-300/50 last:border-b-0 text-left bg-base-200/50"
										>
											<div class="h-4 w-4 rounded border flex items-center justify-center flex-shrink-0 bg-primary border-primary">
												<Check class="h-2.5 w-2.5 text-primary-content" />
											</div>
											<span class="text-xs text-base-content/60 truncate">{selectedValue}</span>
										</button>
									{/if}
								{/each}
							{/if}
						</div>
					</div>
				</div>
			</div>
		{/if}

	{:else}
		<Input
			type="text"
			placeholder={field.placeholder || ''}
			bind:value
			{disabled}
		/>
	{/if}
</div>
