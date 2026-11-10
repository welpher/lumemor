<script lang="ts">
	// The model management page of the admin surface (M3a Task 5):
	// the slot-grouped listing of every registered model with its
	// facts, the empty state whose CTA starts the default group's
	// download (the v1 form of the wizard's model download step), the
	// live progress region driven by the SSE job events with a 5s
	// polling fallback, the activation switch behind a confirm dialog
	// (the same-dimension coexistence copy, the cross-dimension wipe
	// warning on the danger variant), and the rerun trigger that
	// hands off to the task panel.
	//
	// THE CROSS-DIMENSION CHOICE: the clip_embeddings column's
	// current dimension is not knowable before the PATCH (it lives in
	// the database catalog), so the confirm dialog warns whenever a
	// clip model's dimension differs from the slot's active model's
	// -- or cannot be compared. The response's dimensionChanged then
	// reports what actually happened.
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import AdminPageHeader from '$lib/components/AdminPageHeader.svelte';
	import {
		activateModel,
		listModels,
		rerunModel,
		startModelDownload,
		type AdminModel,
		type ModelSlotId
	} from '$lib/api/models';
	import type { JobRun } from '$lib/api/jobruns';
	import { formatMegabytes } from '$lib/format';
	import { subscribeJobEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import Download from '@lucide/svelte/icons/download';
	import Check from '@lucide/svelte/icons/check';
	import X from '@lucide/svelte/icons/x';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	/** The slot sections, the contract enum's declaration order. The
	 * clip_text slot (the search's semantic arm -- the CLIP text
	 * tower) sits between the embedding slot and OCR, exactly where
	 * the contract's enum declares it. */
	const SLOT_ORDER: ReadonlyArray<{ slot: ModelSlotId; labelKey: string }> = [
		{ slot: 'face_detect', labelKey: 'admin_models_slot_face_detect' },
		{ slot: 'face_recognize', labelKey: 'admin_models_slot_face_recognize' },
		{ slot: 'clip_embed', labelKey: 'admin_models_slot_clip_embed' },
		{ slot: 'clip_text', labelKey: 'admin_models_slot_clip_text' },
		{ slot: 'ocr', labelKey: 'admin_models_slot_ocr' }
	];

	let models = $state<AdminModel[]>([]);
	let download = $state<JobRun | null>(null);
	let status = $state<ListStatus>('idle');
	let errorKey = $state('');
	/** Bumping re-runs the listing (the retry button, the convergence). */
	let retryTick = $state(0);
	let busy = $state(false);
	/** The model whose activation the confirm Modal names. */
	let pendingActivate = $state<AdminModel | null>(null);
	let activateErrorKey = $state('');
	/** The cross-dimension arm's report, rendered after the switch. */
	let wipedNotice = $state<{ from: number; to: number; count: number } | null>(null);
	/** The model whose rerun the confirm Modal names. */
	let pendingRerun = $state<AdminModel | null>(null);
	let rerunErrorKey = $state('');
	/** The id of the model whose rerun started (the task panel handoff). */
	let rerunStartedFor = $state('');
	/** The download POST is in flight. */
	let downloading = $state(false);
	/** The download trigger's failure (the POST, not the run). */
	let downloadErrorKey = $state('');

	const settledEmpty = $derived(status === 'ready' && models.length === 0);
	/** The download run, live: queued or running. */
	const downloadActive = $derived(download !== null && (download.state === 'queued' || download.state === 'running'));
	const downloadFailed = $derived(download !== null && download.state === 'failed');
	/** The slot sections with their models, declaration order. */
	const slots = $derived(
		SLOT_ORDER.map((entry) => ({
			...entry,
			entries: models.filter((model) => model.slot === entry.slot)
		}))
	);

	// The listing, keyed on the retry tick. A superseded run -- the
	// retry fired again, or the route left -- never applies.
	$effect(() => {
		void retryTick;
		status = 'loading';
		errorKey = '';
		let current = true;
		void (async () => {
			const outcome = await listModels();
			if (!current) {
				return;
			}
			if (!outcome.ok) {
				status = 'error';
				errorKey = outcome.errorKey;
				return;
			}
			models = outcome.models;
			download = outcome.download;
			status = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	// The SSE convergence of the download run: progress events update
	// the run in place (the events' counts are already numbers), the
	// start and finish of any model download run converge through a
	// refetch (the run may have started elsewhere; the finish's row
	// and the registered models land together).
	$effect(() => {
		return subscribeJobEvents((event) => {
			if (event.kind !== 'model_download') {
				return;
			}
			if (event.name === 'job.progress') {
				const run = download;
				if (run !== null && run.id === event.jobRunId) {
					download = {
						...run,
						total: event.total,
						done: event.done,
						failed: event.failed,
						state: event.state
					};
				}
				return;
			}
			retryTick += 1;
		});
	});

	// The live-progress fallback: while the download runs, the listing
	// is re-read every 5 seconds. The subscription's own state is not
	// observable (a dead stream would otherwise freeze the wizard's
	// download step); the SSE events simply arrive first when the
	// stream is alive.
	$effect(() => {
		if (!downloadActive) {
			return;
		}
		const timer = setInterval(() => {
			retryTick += 1;
		}, 5000);
		return () => clearInterval(timer);
	});

	function retry(): void {
		retryTick += 1;
	}

	// The shared unknown-code fallback (see $lib/i18n/errors); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	/** The download progress bar's percentage. */
	function downloadPercent(): number {
		if (download === null || download.total <= 0) {
			return 0;
		}
		return Math.min(100, Math.round((download.done / download.total) * 100));
	}

	/** The progress line's pre-formatted byte counts. */
	function downloadProgressText(): string {
		if (download === null) {
			return '';
		}
		return $_('admin_models_download_progress', {
			values: { done: formatMegabytes(download.done), total: formatMegabytes(download.total) }
		});
	}

	async function startDownload(): Promise<void> {
		if (downloading) {
			return;
		}
		downloading = true;
		downloadErrorKey = '';
		const outcome = await startModelDownload();
		downloading = false;
		if (outcome.ok) {
			// The run row lands through the refetch (the idempotent
			// join answered either the fresh or the existing run).
			retryTick += 1;
			return;
		}
		downloadErrorKey = outcome.errorKey;
	}

	/** The dimension of the slot's currently active clip model, or null. */
	function activeClipDim(): number | null {
		const active = models.find((model) => model.slot === 'clip_embed' && model.isActive);
		return active?.dim ?? null;
	}

	/**
	 * The destructive arm's pre-flight guess: a clip model whose
	 * dimension differs from the active one's -- or cannot be
	 * compared with it -- warns, because the column's dimension is
	 * only knowable server-side. Non-clip slots have no dimension
	 * semantics and never warn.
	 */
	function needsDimWarning(model: AdminModel): boolean {
		if (model.slot !== 'clip_embed') {
			return false;
		}
		const activeDim = activeClipDim();
		return model.dim === null || activeDim === null || model.dim !== activeDim;
	}

	function slotLabel(slot: ModelSlotId): string {
		const entry = SLOT_ORDER.find((candidate) => candidate.slot === slot);
		return entry === undefined ? slot : $_(entry.labelKey);
	}

	/** The model's params, rendered as a compact summary. */
	function paramsSummary(model: AdminModel): string {
		return Object.entries(model.params)
			.map(([key, value]) => `${key} ${String(value)}`)
			.join(', ');
	}

	function askActivate(model: AdminModel): void {
		activateErrorKey = '';
		pendingActivate = model;
	}

	function cancelActivate(): void {
		if (busy) {
			return;
		}
		pendingActivate = null;
		activateErrorKey = '';
	}

	async function confirmActivate(): Promise<void> {
		const model = pendingActivate;
		if (model === null || busy) {
			return;
		}
		// A retry starts clean: the modal's error from the previous
		// attempt clears before the request runs (the trash page's
		// confirm pattern).
		activateErrorKey = '';
		wipedNotice = null;
		// The modal stays open through the request: a failed switch
		// recovers in place with the mapped error inside the dialog.
		busy = true;
		const outcome = await activateModel(model.id);
		busy = false;
		if (!outcome.ok) {
			activateErrorKey = outcome.errorKey;
			return;
		}
		pendingActivate = null;
		if (outcome.dimensionChanged !== undefined) {
			wipedNotice = {
				from: outcome.dimensionChanged.from,
				to: outcome.dimensionChanged.to,
				count: outcome.wipedEmbeddings ?? 0
			};
		}
		// The refetch converges the activation badges and the listing.
		retryTick += 1;
	}

	function askRerun(model: AdminModel): void {
		rerunErrorKey = '';
		pendingRerun = model;
	}

	function cancelRerun(): void {
		if (busy) {
			return;
		}
		pendingRerun = null;
		rerunErrorKey = '';
	}

	async function confirmRerun(): Promise<void> {
		const model = pendingRerun;
		if (model === null || busy) {
			return;
		}
		rerunErrorKey = '';
		busy = true;
		const outcome = await rerunModel(model.id);
		busy = false;
		if (!outcome.ok) {
			rerunErrorKey = outcome.errorKey;
			return;
		}
		pendingRerun = null;
		rerunStartedFor = model.id;
	}
</script>

<svelte:head>
	<title>Models · lumemor</title>
</svelte:head>

<div class="models max-w-[760px] mx-auto flex flex-col gap-6">
	{#if status === 'loading'}
		<p class="models__status my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('admin_models_loading')}
		</p>
	{:else if status === 'error'}
		<div
			class="models__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{:else if status === 'ready'}
		<!-- The page-header row (the 2026-11-09 specimen): the title
		     + the hairline; the workbench carries no page-level
		     action, the row stands for the grammar's uniformity. The
		     page's own note rides under it as the lede. -->
		<AdminPageHeader title={$_('admin_models_title')} />
		<p class="models__note m-0 mb-1 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
			{$_('admin_models_note')}
		</p>

		{#if wipedNotice !== null}
			<p
				class="models__wiped px-4 py-3 font-body font-medium text-[14px] leading-[1.5] text-danger-text bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)] border border-danger rounded-card"
				role="status"
			>
				{$_('admin_models_wiped_notice', { values: wipedNotice })}
			</p>
		{/if}

		<!-- The empty card owns the action surface only while no run and
		     no failure holds it: once a download is in flight (or its
		     run failed, or the trigger failed), the progress region
		     below carries the action instead. -->
		{#if settledEmpty && !downloadActive && !downloadFailed && downloadErrorKey === ''}
			<section
				class="models__empty flex flex-col items-center gap-3 max-w-[480px] box-content mx-auto mt-6 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card"
			>
				<h2 class="models__empty-title font-display font-bold text-[22px] text-text">
					{$_('admin_models_empty_title')}
				</h2>
				<p class="models__empty-body font-body font-normal text-[14px] leading-[1.5] text-text-soft">
					{$_('admin_models_empty_body')}
				</p>
				<Button variant="primary" disabled={downloading} onclick={() => void startDownload()}>
					<Download size={20} aria-hidden="true" />
					{$_('admin_models_download_cta')}
				</Button>
			</section>
		{/if}

		{#if downloadErrorKey !== ''}
			<div
				class="models__download-error flex flex-col items-center gap-3 max-w-[480px] box-content mx-auto mt-6 p-5 text-center border border-danger rounded-card text-danger-text"
				role="alert"
			>
				<p>{errorText(downloadErrorKey)}</p>
				<Button variant="primary" disabled={downloading} onclick={() => void startDownload()}>
					<RotateCcw size={20} aria-hidden="true" />
					{$_('action_retry')}
				</Button>
			</div>
		{/if}

		{#if downloadActive || downloadFailed}
			<section
				class="models__progress-region flex flex-col gap-2.5 p-5 bg-card border border-border rounded-card shadow-card"
				aria-live="polite"
			>
				{#if downloadActive}
					<p
						class="models__progress-state font-body font-medium text-[14px] leading-[1.4] text-text tabular-nums"
					>
						{download?.state === 'queued' ? $_('admin_models_download_queued') : downloadProgressText()}
					</p>
					<div
						class="models__progress w-full h-2 bg-accent-soft rounded-control overflow-hidden animate-progress-in"
						role="progressbar"
						aria-valuemin="0"
						aria-valuemax="100"
						aria-valuenow={downloadPercent()}
					>
						<div
							class="models__progress-fill h-full bg-accent transition-[width] duration-(--motion-base) ease-standard"
							style="width: {downloadPercent()}%"
						></div>
					</div>
				{:else if downloadFailed}
					<p
						class="models__download-failed font-body font-medium text-[14px] leading-[1.4] text-danger-text"
						role="alert"
					>
						{$_('admin_models_download_failed_run', {
							values: { count: download?.failed ?? 0 }
						})}
					</p>
					<Button variant="primary" disabled={downloading} onclick={() => void startDownload()}>
						<RotateCcw size={20} aria-hidden="true" />
						{$_('action_retry')}
					</Button>
				{/if}
			</section>
		{/if}

		{#if rerunStartedFor !== ''}
			<p
				class="models__rerun-started px-4 py-3 font-body font-normal text-[14px] leading-[1.5] text-text-soft bg-card border border-border rounded-card"
			>
				{$_('admin_models_rerun_started')}
				<a class="text-accent-soft-text ml-1.5" href="/admin/jobs">{$_('admin_models_rerun_panel')}</a>
			</p>
		{/if}

		{#each slots as group (group.slot)}
			<section class="models__slot flex flex-col gap-2.5" aria-label={$_(group.labelKey)}>
				<h2
					class="models__slot-title font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint"
				>
					{$_(group.labelKey)}
				</h2>
				{#if group.entries.length === 0}
					<p
						class="models__slot-empty px-4 py-3.5 font-body font-normal text-[14px] leading-[1.5] text-text-soft bg-card border border-dashed border-border rounded-card"
					>
						{$_('admin_models_slot_empty')}
					</p>
				{:else}
					<div class="models__rows flex flex-col gap-2.5">
						{#each group.entries as model (model.id)}
							<article
								class="models__row flex flex-col gap-1.5 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
								data-model-id={model.id}
							>
								<header class="models__row-head flex items-center gap-2.5">
									<h3
										class="models__row-id font-body font-semibold text-[16px] leading-[1.4] text-text"
									>
										{model.id}
									</h3>
									<!-- An empty badge still reserves no space (empty:hidden);
									     the active marker is the only state a row carries. -->
									<span
										class="models__badge inline-block px-2.5 py-0.5 font-body font-semibold text-[12px] leading-[1.6] bg-accent-soft rounded-control empty:hidden"
										class:models__badge--active={model.isActive}
										class:text-accent-soft-text={model.isActive}
										class:text-text-soft={!model.isActive}
									>
										{model.isActive ? $_('admin_models_active') : ''}
									</span>
									{#if !model.filesPresent}
										<span
											class="models__badge models__badge--missing inline-block px-2.5 py-0.5 font-body font-semibold text-[12px] leading-[1.6] text-danger-text bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)] rounded-control empty:hidden"
										>
											{$_('admin_models_files_missing')}
										</span>
									{/if}
								</header>
								<p class="models__facts font-body font-normal text-[14px] leading-[1.6] text-text-soft">
									{$_('admin_models_version', { values: { version: model.version } })}
									·
									{$_('admin_models_license')}
									<span class="models__license font-semibold">{model.license}</span>
									·
									{model.source === 'downloaded'
										? $_('admin_models_source_downloaded')
										: $_('admin_models_source_custom')}
									{#if model.dim !== null}
										· {$_('admin_models_dim', { values: { dim: model.dim } })}
									{/if}
									{#if model.inputSize !== null}
										·
										{$_('admin_models_input_size', {
											values: { size: `${model.inputSize.width}×${model.inputSize.height}` }
										})}
									{/if}
									{#if Object.keys(model.params).length > 0}
										· {$_('admin_models_params', { values: { params: paramsSummary(model) } })}
									{/if}
								</p>
								{#if model.notice !== null && model.notice !== ''}
									<p
										class="models__notice font-body font-normal text-[13px] leading-[1.5] text-text-faint"
									>
										{$_('admin_models_notice', { values: { notice: model.notice } })}
									</p>
								{/if}
								<div class="models__row-actions flex justify-end">
									{#if !model.isActive}
										<Button variant="ghost" disabled={busy} onclick={() => askActivate(model)}>
											<Check size={20} aria-hidden="true" />
											{$_('admin_models_activate')}
										</Button>
									{:else}
										<Button variant="ghost" disabled={busy} onclick={() => askRerun(model)}>
											<RefreshCw size={20} aria-hidden="true" />
											{$_('admin_models_rerun')}
										</Button>
									{/if}
								</div>
							</article>
						{/each}
					</div>
				{/if}
			</section>
		{/each}
	{/if}
</div>

<Modal open={pendingActivate !== null} title={$_('admin_models_activate_title')} onclose={cancelActivate}>
	<p class="models__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
		{#if pendingActivate !== null && needsDimWarning(pendingActivate)}
			{$_('admin_models_activate_body_cross', {
				values: { name: pendingActivate.id, slot: slotLabel(pendingActivate.slot) }
			})}
		{:else if pendingActivate !== null}
			{$_('admin_models_activate_body_same', {
				values: { name: pendingActivate.id, slot: slotLabel(pendingActivate.slot) }
			})}
		{/if}
	</p>
	{#if activateErrorKey !== ''}
		<p
			class="models__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(activateErrorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelActivate}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		{#if pendingActivate !== null && needsDimWarning(pendingActivate)}
			<Button variant="danger" disabled={busy} onclick={() => void confirmActivate()}>
				<Check size={20} aria-hidden="true" />
				{$_('admin_models_activate_confirm')}
			</Button>
		{:else}
			<Button variant="primary" disabled={busy} onclick={() => void confirmActivate()}>
				<Check size={20} aria-hidden="true" />
				{$_('admin_models_activate_confirm')}
			</Button>
		{/if}
	{/snippet}
</Modal>

<Modal open={pendingRerun !== null} title={$_('admin_models_rerun_title')} onclose={cancelRerun}>
	<p class="models__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
		{#if pendingRerun !== null}
			{$_('admin_models_rerun_body', { values: { slot: slotLabel(pendingRerun.slot) } })}
		{/if}
	</p>
	{#if rerunErrorKey !== ''}
		<p
			class="models__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(rerunErrorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelRerun}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="primary" disabled={busy} onclick={() => void confirmRerun()}>
			<RefreshCw size={20} aria-hidden="true" />
			{$_('admin_models_rerun_confirm')}
		</Button>
	{/snippet}
</Modal>
