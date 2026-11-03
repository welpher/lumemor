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

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	/** The slot sections, the contract enum's declaration order. */
	const SLOT_ORDER: ReadonlyArray<{ slot: ModelSlotId; labelKey: string }> = [
		{ slot: 'face_detect', labelKey: 'admin_models_slot_face_detect' },
		{ slot: 'face_recognize', labelKey: 'admin_models_slot_face_recognize' },
		{ slot: 'clip_embed', labelKey: 'admin_models_slot_clip_embed' },
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
	const downloadActive = $derived(
		download !== null && (download.state === 'queued' || download.state === 'running')
	);
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
		const active = models.find(
			(model) => model.slot === 'clip_embed' && model.isActive
		);
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

<div class="models">
	{#if status === 'loading'}
		<p class="models__status" role="status">{$_('admin_models_loading')}</p>
	{:else if status === 'error'}
		<div class="models__failure" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}>{$_('action_retry')}</Button>
		</div>
	{:else if status === 'ready'}
		<header class="models__header">
			<h1 class="models__title">{$_('admin_models_title')}</h1>
			<p class="models__note">{$_('admin_models_note')}</p>
		</header>

		{#if wipedNotice !== null}
			<p class="models__wiped" role="status">
				{$_('admin_models_wiped_notice', { values: wipedNotice })}
			</p>
		{/if}

		<!-- The empty card owns the action surface only while no run and
		     no failure holds it: once a download is in flight (or its
		     run failed, or the trigger failed), the progress region
		     below carries the action instead. -->
		{#if settledEmpty && !downloadActive && !downloadFailed && downloadErrorKey === ''}
			<section class="models__empty">
				<h2 class="models__empty-title">{$_('admin_models_empty_title')}</h2>
				<p class="models__empty-body">{$_('admin_models_empty_body')}</p>
				<Button
					variant="primary"
					disabled={downloading}
					onclick={() => void startDownload()}
				>
					{$_('admin_models_download_cta')}
				</Button>
			</section>
		{/if}

		{#if downloadErrorKey !== ''}
			<div class="models__download-error" role="alert">
				<p>{errorText(downloadErrorKey)}</p>
				<Button variant="primary" disabled={downloading} onclick={() => void startDownload()}>
					{$_('action_retry')}
				</Button>
			</div>
		{/if}

		{#if downloadActive || downloadFailed}
			<section class="models__progress-region" aria-live="polite">
				{#if downloadActive}
					<p class="models__progress-state">
						{download?.state === 'queued'
							? $_('admin_models_download_queued')
							: downloadProgressText()}
					</p>
					<div
						class="models__progress"
						role="progressbar"
						aria-valuemin="0"
						aria-valuemax="100"
						aria-valuenow={downloadPercent()}
					>
						<div class="models__progress-fill" style="width: {downloadPercent()}%"></div>
					</div>
				{:else if downloadFailed}
					<p class="models__download-failed" role="alert">
						{$_('admin_models_download_failed_run', {
							values: { count: download?.failed ?? 0 }
						})}
					</p>
					<Button variant="primary" disabled={downloading} onclick={() => void startDownload()}>
						{$_('action_retry')}
					</Button>
				{/if}
			</section>
		{/if}

		{#if rerunStartedFor !== ''}
			<p class="models__rerun-started">
				{$_('admin_models_rerun_started')}
				<a href="/admin/jobs">{$_('admin_models_rerun_panel')}</a>
			</p>
		{/if}

		{#each slots as group (group.slot)}
			<section class="models__slot" aria-label={$_(group.labelKey)}>
				<h2 class="models__slot-title">{$_(group.labelKey)}</h2>
				{#if group.entries.length === 0}
					<p class="models__slot-empty">{$_('admin_models_slot_empty')}</p>
				{:else}
					<div class="models__rows">
						{#each group.entries as model (model.id)}
							<article class="models__row" data-model-id={model.id}>
								<header class="models__row-head">
									<h3 class="models__row-id">{model.id}</h3>
									<span class="models__badge" class:models__badge--active={model.isActive}>
										{model.isActive ? $_('admin_models_active') : ''}
									</span>
									{#if !model.filesPresent}
										<span class="models__badge models__badge--missing">
											{$_('admin_models_files_missing')}
										</span>
									{/if}
								</header>
								<p class="models__facts">
									{$_('admin_models_version', { values: { version: model.version } })}
									·
									{$_('admin_models_license')}
									<span class="models__license">{model.license}</span>
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
									<p class="models__notice">
										{$_('admin_models_notice', { values: { notice: model.notice } })}
									</p>
								{/if}
								<div class="models__row-actions">
									{#if !model.isActive}
										<Button
											variant="ghost"
											disabled={busy}
											onclick={() => askActivate(model)}
										>
											{$_('admin_models_activate')}
										</Button>
									{:else}
										<Button
											variant="ghost"
											disabled={busy}
											onclick={() => askRerun(model)}
										>
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
	{#snippet children()}
		<p class="models__confirm-body">
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
			<p class="models__confirm-error" role="alert">{errorText(activateErrorKey)}</p>
		{/if}
	{/snippet}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelActivate}>{$_('common_cancel')}</Button>
		{#if pendingActivate !== null && needsDimWarning(pendingActivate)}
			<Button variant="danger" disabled={busy} onclick={() => void confirmActivate()}>
				{$_('admin_models_activate_confirm')}
			</Button>
		{:else}
			<Button variant="primary" disabled={busy} onclick={() => void confirmActivate()}>
				{$_('admin_models_activate_confirm')}
			</Button>
		{/if}
	{/snippet}
</Modal>

<Modal open={pendingRerun !== null} title={$_('admin_models_rerun_title')} onclose={cancelRerun}>
	{#snippet children()}
		<p class="models__confirm-body">
			{#if pendingRerun !== null}
				{$_('admin_models_rerun_body', { values: { slot: slotLabel(pendingRerun.slot) } })}
			{/if}
		</p>
		{#if rerunErrorKey !== ''}
			<p class="models__confirm-error" role="alert">{errorText(rerunErrorKey)}</p>
		{/if}
	{/snippet}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelRerun}>{$_('common_cancel')}</Button>
		<Button variant="primary" disabled={busy} onclick={() => void confirmRerun()}>
			{$_('admin_models_rerun_confirm')}
		</Button>
	{/snippet}
</Modal>

<style>
	.models {
		max-width: 760px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.models__status {
		margin: 48px 0;
		text-align: center;
		color: var(--color-text-soft);
	}

	.models__failure {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		max-width: 420px;
		margin: 48px auto;
		padding: 20px;
		text-align: center;
		border: 1px solid var(--color-danger);
		border-radius: var(--radius-card);
		color: var(--color-danger-text);
	}

	.models__failure p {
		margin: 0;
	}

	.models__header h1 {
		margin: 0 0 4px;
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 26px;
		letter-spacing: 0.01em;
		color: var(--color-text);
	}

	.models__note {
		margin: 0;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text-soft);
	}

	.models__wiped {
		margin: 0;
		padding: 12px 16px;
		font: 500 14px/1.5 var(--font-body);
		color: var(--color-danger-text);
		background-color: color-mix(in srgb, var(--color-danger) 10%, transparent);
		border: 1px solid var(--color-danger);
		border-radius: var(--radius-card);
	}

	.models__empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		max-width: 480px;
		margin: 24px auto 0;
		padding: 32px 24px;
		text-align: center;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
	}

	.models__empty-title {
		margin: 0;
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 22px;
		letter-spacing: 0.01em;
		color: var(--color-text);
	}

	.models__empty-body {
		margin: 0;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text-soft);
	}

	.models__download-error {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		max-width: 480px;
		margin: 24px auto 0;
		padding: 20px;
		text-align: center;
		border: 1px solid var(--color-danger);
		border-radius: var(--radius-card);
		color: var(--color-danger-text);
	}

	.models__download-error p {
		margin: 0;
	}

	.models__progress-region {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 20px;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
	}

	.models__progress-state {
		margin: 0;
		font: 500 14px/1.4 var(--font-body);
		color: var(--color-text);
		font-variant-numeric: tabular-nums;
	}

	.models__progress {
		width: 100%;
		height: 8px;
		background-color: var(--color-accent-soft);
		border-radius: var(--radius-control);
		overflow: hidden;
	}

	.models__progress-fill {
		height: 100%;
		background-color: var(--color-accent);
		transition: width var(--motion-base) var(--ease-standard);
	}

	.models__download-failed {
		margin: 0;
		font: 500 14px/1.4 var(--font-body);
		color: var(--color-danger-text);
	}

	.models__rerun-started {
		margin: 0;
		padding: 12px 16px;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text-soft);
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
	}

	.models__rerun-started a {
		color: var(--color-accent-soft-text);
		margin-left: 6px;
	}

	.models__slot {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.models__slot-title {
		margin: 0;
		font: 600 12px/1.4 var(--font-body);
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--color-text-faint);
	}

	.models__slot-empty {
		margin: 0;
		padding: 14px 16px;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text-soft);
		background-color: var(--color-card);
		border: 1px dashed var(--color-border);
		border-radius: var(--radius-card);
	}

	.models__rows {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.models__row {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 16px 18px;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
	}

	.models__row-head {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.models__row-id {
		margin: 0;
		font: 600 16px/1.4 var(--font-body);
		color: var(--color-text);
	}

	.models__badge {
		display: inline-block;
		padding: 2px 10px;
		font: 600 12px/1.6 var(--font-body);
		color: var(--color-text-soft);
		background-color: var(--color-accent-soft);
		border-radius: var(--radius-control);
	}

	/* An empty badge still reserves no space; the active marker is
	 * the only state a row carries. */
	.models__badge:empty {
		display: none;
	}

	.models__badge--active {
		color: var(--color-accent-soft-text);
	}

	.models__badge--missing {
		color: var(--color-danger-text);
		background-color: color-mix(in srgb, var(--color-danger) 12%, transparent);
	}

	.models__facts {
		margin: 0;
		font: 400 14px/1.6 var(--font-body);
		color: var(--color-text-soft);
	}

	.models__license {
		font-weight: 600;
	}

	.models__notice {
		margin: 0;
		font: 400 13px/1.5 var(--font-body);
		color: var(--color-text-faint);
	}

	.models__row-actions {
		display: flex;
		justify-content: flex-end;
	}

	.models__confirm-body {
		margin: 0;
		font: 400 15px/1.5 var(--font-body);
		color: var(--color-text);
	}

	.models__confirm-error {
		margin: 12px 0 0;
		font: 500 14px/1.4 var(--font-body);
		color: var(--color-danger-text);
	}
</style>
