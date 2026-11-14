<script lang="ts">
	// The task panel of the admin surface: the job runs
	// listing with each run's kind, state, progress and timestamps.
	// Active rows (queued or running) converge live: the SSE job.*
	// events update the matching row in place (progress) or trigger a
	// refetch (start, finish -- the timestamps and any run started
	// elsewhere land through the listing), with a 5-second polling
	// fallback while any row is active. The stream's own state is not
	// observable through the subscription, so the poll runs
	// unconditionally on active rows -- a dead stream would otherwise
	// freeze the panel; the SSE events simply arrive first when the
	// stream is alive. History rows are static, and the unmount
	// unsubscribes both.
	//
	// THE FOUR STATES, FOUR SHAPES (not one row with a swapped badge):
	// a queued run draws NO progress bar -- it has not touched any
	// work yet, and a bar at zero would read as a stalled run; a
	// running run carries the live bar, its counts and its start; a
	// succeeded run carries the closed bar, its counts, the finish
	// time and how long it took; a failed run carries the bar in the
	// danger color stopping where the work broke, the REASON --
	// derived from what the panel can honestly know, never the bare
	// word "failed" -- and a retry only where the run's kind has a
	// trigger the web surface owns.
	//
	// THE PROGRESS UNITS: each run's counts carry its own work's
	// units -- the model download's are pinned bytes (rendered as
	// megabytes), the reruns' are asset counts, the rescan's are
	// resolved source entries. The kind decides the formatting;
	// the numbers themselves are the row's.
	//
	// THE FAILED REASON: the listing carries counts, never the
	// worker's error text, so the panel derives the diagnosis from
	// state it has. When the run's dependency is a model, the models
	// read says whether that slot actually has one; when it does not,
	// the row names the missing download and points at the models
	// page. Otherwise the row reports the failed count in the run's
	// units and names where the detail lives (the server's log --
	// the deployment's own surface, which no web route mirrors).
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import AdminPageHeader from '$lib/components/AdminPageHeader.svelte';
	import { listJobRuns, type JobRun } from '$lib/api/jobruns';
	import { listModels, rerunModel, startModelDownload, type AdminModel, type ModelSlotId } from '$lib/api/models';
	import { reresolvePlaces } from '$lib/api/places';
	import { formatMegabytes, formatDuration } from '$lib/format';
	import { subscribeJobEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	/** The run kinds whose label keys the panel renders. */
	const KIND_LABEL_KEYS: Readonly<Record<string, string>> = {
		model_download: 'admin_jobs_kind_model_download',
		face_detect_rerun: 'admin_jobs_kind_face_detect_rerun',
		face_recognize_rerun: 'admin_jobs_kind_face_recognize_rerun',
		clip_embed_rerun: 'admin_jobs_kind_clip_embed_rerun',
		ocr_rerun: 'admin_jobs_kind_ocr_rerun',
		places_rerun: 'admin_jobs_kind_places_rerun',
		album_rescan: 'admin_jobs_kind_album_rescan'
	};

	/** The state label keys, and the badge modifier classes. */
	const STATE_LABEL_KEYS: Readonly<Record<string, string>> = {
		queued: 'admin_jobs_state_queued',
		running: 'admin_jobs_state_running',
		succeeded: 'admin_jobs_state_succeeded',
		failed: 'admin_jobs_state_failed'
	};

	/**
	 * The rerun kinds whose work one model backs: the run's
	 * dependency, and the model the retry re-runs.
	 */
	const RERUN_SLOTS: Readonly<Record<string, ModelSlotId>> = {
		face_detect_rerun: 'face_detect',
		face_recognize_rerun: 'face_recognize',
		clip_embed_rerun: 'clip_embed',
		ocr_rerun: 'ocr'
	};

	/** The slot label keys (the models page's own vocabulary). */
	const SLOT_LABEL_KEYS: Readonly<Record<ModelSlotId, string>> = {
		face_detect: 'admin_models_slot_face_detect',
		face_recognize: 'admin_models_slot_face_recognize',
		clip_embed: 'admin_models_slot_clip_embed',
		clip_text: 'admin_models_slot_clip_text',
		ocr: 'admin_models_slot_ocr'
	};

	let jobs = $state<JobRun[]>([]);
	/** The models read behind the retry matrix and the missing-dependency diagnosis. */
	let models = $state<AdminModel[]>([]);
	let status = $state<ListStatus>('idle');
	let errorKey = $state('');
	/** Bumping re-runs the listing (the retry button, the convergence). */
	let retryTick = $state(0);
	/** Bumping re-reads the models (the mount, a retry that may have changed them). */
	let modelsTick = $state(0);
	/** The failed run whose retry is in flight (null while none). */
	let retrying = $state<string | null>(null);
	/** The last failed retry, pinned to its row (null while none). */
	let retryError = $state<{ jobId: string; errorKey: string } | null>(null);

	const activeLocale = $derived($locale ?? 'en');
	/** The fallback poll's cadence, in milliseconds. */
	const POLL_INTERVAL_MS = 5000;
	const hasActiveRuns = $derived(jobs.some((job) => job.state === 'queued' || job.state === 'running'));

	// The listing, keyed on the retry tick. A superseded run -- the
	// retry fired again, or the route left -- never applies.
	$effect(() => {
		void retryTick;
		status = 'loading';
		errorKey = '';
		let current = true;
		void (async () => {
			const outcome = await listJobRuns();
			if (!current) {
				return;
			}
			if (!outcome.ok) {
				status = 'error';
				errorKey = outcome.errorKey;
				return;
			}
			jobs = outcome.jobs;
			status = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	// The models read stands apart from the listing's tick: it is the
	// dependency probe's fact, not the runs' state, so the 5-second
	// convergence never re-reads it. Its failure never fails the panel
	// -- the rows simply lose the model-backed retry and the
	// missing-dependency wording.
	$effect(() => {
		void modelsTick;
		let current = true;
		void (async () => {
			const outcome = await listModels();
			if (!current || !outcome.ok) {
				return;
			}
			models = outcome.models;
		})();
		return () => {
			current = false;
		};
	});

	// The SSE convergence: progress events update the named row in
	// place (the events' counts are already numbers), the start and
	// finish of any run converge through a refetch (a run started
	// elsewhere joins the listing; the finish's timestamps land).
	$effect(() => {
		return subscribeJobEvents((event) => {
			if (event.name === 'job.progress') {
				jobs = jobs.map((job) =>
					job.id === event.jobRunId
						? {
								...job,
								kind: event.kind as JobRun['kind'],
								state: event.state as JobRun['state'],
								total: event.total,
								done: event.done,
								failed: event.failed
							}
						: job
				);
				return;
			}
			retryTick += 1;
		});
	});

	// The polling fallback while any row is active: history rows are
	// static, and the panel stops asking once nothing runs.
	$effect(() => {
		if (!hasActiveRuns) {
			return;
		}
		const timer = setInterval(() => {
			retryTick += 1;
		}, POLL_INTERVAL_MS);
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

	function kindLabel(job: JobRun): string {
		const key = KIND_LABEL_KEYS[job.kind];
		return key === undefined ? job.kind : $_(key);
	}

	function stateLabel(job: JobRun): string {
		const key = STATE_LABEL_KEYS[job.state];
		return key === undefined ? job.state : $_(key);
	}

	/** One row's progress bar percentage. */
	function percent(job: JobRun): number {
		if (job.total <= 0) {
			return 0;
		}
		return Math.min(100, Math.round((job.done / job.total) * 100));
	}

	/** The progress line: the units belong to the run's kind. */
	function progressText(job: JobRun): string {
		if (job.kind === 'model_download') {
			return $_('admin_jobs_progress_bytes', {
				values: { done: formatMegabytes(job.done), total: formatMegabytes(job.total) }
			});
		}
		return $_('admin_jobs_progress_assets', {
			values: { done: job.done, total: job.total }
		});
	}

	/** Locale-aware timestamp of one run boundary. */
	function timeLabel(value: string): string {
		return new Intl.DateTimeFormat(activeLocale, { dateStyle: 'medium', timeStyle: 'short' }).format(
			new Date(value)
		);
	}

	/** How long a closed run took, at the wall's m:ss rhythm. */
	function durationLabel(job: JobRun): string {
		if (job.startedAt === null || job.finishedAt === null) {
			return '';
		}
		return formatDuration(new Date(job.finishedAt).getTime() - new Date(job.startedAt).getTime());
	}

	/** The slot a run's work depends on, when one model backs it. */
	function neededSlot(job: JobRun): ModelSlotId | null {
		return RERUN_SLOTS[job.kind] ?? null;
	}

	/** The active, downloaded model of a slot (null when the slot has none). */
	function activeModelOf(slot: ModelSlotId): AdminModel | null {
		return models.find((model) => model.slot === slot && model.isActive && model.filesPresent) ?? null;
	}

	/** The slot's label, in the models page's own words. */
	function slotLabel(slot: ModelSlotId): string {
		return $_(SLOT_LABEL_KEYS[slot]);
	}

	/** Whether the failed run's dependency (its model) is missing. */
	function dependencyMissing(job: JobRun): boolean {
		const slot = neededSlot(job);
		// The models read failed or has not landed: claiming a missing
		// download without the facts would be a guess.
		if (slot === null || models.length === 0) {
			return false;
		}
		return activeModelOf(slot) === null;
	}

	/** The missing-dependency sentence of one failed run. */
	function missingSlotLabel(job: JobRun): string {
		const slot = neededSlot(job);
		return slot === null ? '' : $_('admin_jobs_reason_model_missing', { values: { slot: slotLabel(slot) } });
	}

	/**
	 * Whether this failed run has a retry the web surface owns: the
	 * download can be re-joined, the reruns re-post through their
	 * model, and the places rerun has its own endpoint. The album
	 * rescan does not -- its run row carries no album id, so the
	 * trigger belongs to the album page, not here.
	 */
	function canRetry(job: JobRun): boolean {
		if (job.state !== 'failed') {
			return false;
		}
		if (job.kind === 'model_download' || job.kind === 'places_rerun') {
			return true;
		}
		const slot = neededSlot(job);
		return slot !== null && activeModelOf(slot) !== null;
	}

	/** Re-post the failed run's own trigger; the listing converges on the new run. */
	async function retryRun(job: JobRun): Promise<void> {
		if (retrying !== null) {
			return;
		}
		retrying = job.id;
		retryError = null;
		let outcome: { ok: boolean; errorKey?: string };
		if (job.kind === 'model_download') {
			outcome = await startModelDownload();
		} else if (job.kind === 'places_rerun') {
			outcome = await reresolvePlaces();
		} else {
			const slot = neededSlot(job);
			const model = slot === null ? null : activeModelOf(slot);
			if (model === null) {
				retrying = null;
				return;
			}
			outcome = await rerunModel(model.id);
		}
		retrying = null;
		if (!outcome.ok) {
			retryError = { jobId: job.id, errorKey: outcome.errorKey ?? 'internal' };
			return;
		}
		retryTick += 1;
		modelsTick += 1;
	}
</script>

<svelte:head>
	<title>Jobs · lumemor</title>
</svelte:head>

<div class="jobs max-w-[760px] mx-auto flex flex-col gap-5">
	{#if status === 'loading'}
		<p class="jobs__status my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('admin_jobs_loading')}
		</p>
	{:else if status === 'error'}
		<div
			class="jobs__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{:else if status === 'ready'}
		<!-- The page-header row: the title
		     + the hairline; the task panel carries no page-level
		     action, the row stands for the grammar's uniformity. The
		     panel's own note rides under it as the lede. -->
		<AdminPageHeader title={$_('admin_jobs_title')} />
		<p class="jobs__note m-0 mb-1 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
			{$_('admin_jobs_note')}
		</p>

		{#if jobs.length === 0}
			<p
				class="jobs__empty box-content py-8 px-6 text-center font-body font-normal text-[14px] leading-[1.5] text-text-soft bg-card border border-border rounded-card shadow-card"
			>
				{$_('admin_jobs_empty')}
			</p>
		{:else}
			<div class="jobs__rows flex flex-col gap-2.5">
				{#each jobs as job (job.id)}
					<article
						class="jobs__row flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
						data-job-run-id={job.id}
						data-job-state={job.state}
					>
						<header class="jobs__row-head flex items-center justify-between gap-2.5">
							<h2 class="jobs__kind font-body font-semibold text-[16px] leading-[1.4] text-text">
								{kindLabel(job)}
							</h2>
							<span
								class="jobs__badge jobs__badge--{job.state} inline-block px-2.5 py-0.5 font-body font-semibold text-[12px] leading-[1.6] text-text-soft bg-accent-soft rounded-control"
								>{stateLabel(job)}</span
							>
						</header>

						{#if job.state === 'queued'}
							<!-- A queued run has touched nothing yet: the bar
							     would sit at zero and read as a stall. -->
							<p class="jobs__waiting m-0 font-body font-normal text-[14px] leading-[1.4] text-text-soft">
								{$_('admin_jobs_queued_waiting')}
							</p>
						{:else}
							<div
								class="jobs__progress w-full h-2 bg-accent-soft rounded-control overflow-hidden animate-progress-in"
								role="progressbar"
								aria-valuemin="0"
								aria-valuemax="100"
								aria-valuenow={percent(job)}
							>
								<div
									class="jobs__progress-fill jobs__progress-fill--{job.state} h-full bg-accent transition-[width] duration-(--motion-base) ease-standard"
									style="width: {percent(job)}%"
								></div>
							</div>
							<p
								class="jobs__counts font-body font-medium text-[14px] leading-[1.4] text-text tabular-nums"
							>
								{progressText(job)}
								{#if job.failed > 0}
									<span class="jobs__failed ml-2.5 text-danger-text"
										>{$_('admin_jobs_failed_count', { values: { count: job.failed } })}</span
									>
								{/if}
							</p>
						{/if}

						{#if job.state === 'failed'}
							<!-- The reason: the row says WHY, never only "failed". -->
							<p
								class="jobs__reason m-0 font-body font-normal text-[13px] leading-[1.5] text-danger-text"
								data-testid="job-reason"
							>
								{#if dependencyMissing(job)}
									{missingSlotLabel(job)}
									<a
										class="jobs__reason-link font-semibold text-accent no-underline hover:underline"
										href="/admin/models"
									>
										{$_('admin_jobs_reason_model_link')}
									</a>
								{:else if job.failed > 0}
									{$_(
										job.kind === 'model_download'
											? 'admin_jobs_reason_models'
											: 'admin_jobs_reason_items',
										{ values: { count: job.failed } }
									)}
								{:else}
									{$_('admin_jobs_reason_stopped')}
								{/if}
							</p>
							{#if canRetry(job)}
								<div class="jobs__actions flex flex-wrap items-center gap-2">
									<Button
										variant="ghost"
										disabled={retrying !== null}
										data-testid="job-retry"
										onclick={() => void retryRun(job)}
										><RotateCcw size={16} aria-hidden="true" />{$_('admin_jobs_retry')}</Button
									>
								</div>
							{/if}
							{#if retryError !== null && retryError.jobId === job.id}
								<p
									class="jobs__retry-error m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
									role="alert"
								>
									{errorText(retryError.errorKey)}
								</p>
							{/if}
						{/if}

						<p class="jobs__times font-body font-normal text-[13px] leading-[1.5] text-text-faint">
							{#if job.startedAt !== null}
								{$_('admin_jobs_started', { values: { time: timeLabel(job.startedAt) } })}
								{#if job.finishedAt !== null}
									·
									{$_('admin_jobs_finished', { values: { time: timeLabel(job.finishedAt) } })}
									·
									{$_('admin_jobs_duration', { values: { duration: durationLabel(job) } })}
								{/if}
							{/if}
						</p>
					</article>
				{/each}
			</div>
		{/if}
	{/if}
</div>

<style>
	/* Residual styles (verbatim): the state badges' colors — the
	 * modifier class names are dynamic (jobs__badge--{state}) — and
	 * the failed progress fill's background. */
	.jobs__badge--running,
	.jobs__badge--succeeded {
		color: var(--color-accent-soft-text);
	}

	.jobs__badge--failed {
		color: var(--color-danger-text);
		background-color: color-mix(in srgb, var(--color-danger) 12%, transparent);
	}

	.jobs__progress-fill--failed {
		background-color: var(--color-danger);
	}
</style>
