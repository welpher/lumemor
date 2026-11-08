<script lang="ts">
	// The task panel of the admin surface (M3a Task 5): the job runs
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
	// THE PROGRESS UNITS: each run's counts carry its own work's
	// units -- the model download's are pinned bytes (rendered as
	// megabytes), the reruns' are asset counts, the rescan's are
	// resolved source entries. The kind decides the formatting;
	// the numbers themselves are the row's.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import { listJobRuns, type JobRun } from '$lib/api/jobruns';
	import { formatMegabytes } from '$lib/format';
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

	let jobs = $state<JobRun[]>([]);
	let status = $state<ListStatus>('idle');
	let errorKey = $state('');
	/** Bumping re-runs the listing (the retry button, the convergence). */
	let retryTick = $state(0);

	const activeLocale = $derived($locale ?? 'en');
	/** The fallback poll's cadence, in milliseconds. */
	const POLL_INTERVAL_MS = 5000;
	const hasActiveRuns = $derived(
		jobs.some((job) => job.state === 'queued' || job.state === 'running')
	);

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
</script>

<svelte:head>
	<title>Jobs · lumemor</title>
</svelte:head>

<div class="jobs max-w-[680px] mx-auto flex flex-col gap-5">
	{#if status === 'loading'}
		<p class="jobs__status my-12 text-center text-text-soft animate-loading-in" role="status">{$_('admin_jobs_loading')}</p>
	{:else if status === 'error'}
		<div class="jobs__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
		</div>
	{:else if status === 'ready'}
		<header class="jobs__header">
			<h1 class="jobs__title mb-1 font-display font-bold text-[26px] text-text">{$_('admin_jobs_title')}</h1>
			<p class="jobs__note font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('admin_jobs_note')}</p>
		</header>

		{#if jobs.length === 0}
			<p class="jobs__empty box-content py-8 px-6 text-center font-body font-normal text-[14px] leading-[1.5] text-text-soft bg-card border border-border rounded-card shadow-card">{$_('admin_jobs_empty')}</p>
		{:else}
			<div class="jobs__rows flex flex-col gap-2.5">
				{#each jobs as job (job.id)}
					<article class="jobs__row flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card" data-job-run-id={job.id}>
						<header class="jobs__row-head flex items-center justify-between gap-2.5">
							<h2 class="jobs__kind font-body font-semibold text-[16px] leading-[1.4] text-text">{kindLabel(job)}</h2>
							<span class="jobs__badge jobs__badge--{job.state} inline-block px-2.5 py-0.5 font-body font-semibold text-[12px] leading-[1.6] text-text-soft bg-accent-soft rounded-control">{stateLabel(job)}</span>
						</header>
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
						<p class="jobs__counts font-body font-medium text-[14px] leading-[1.4] text-text tabular-nums">
							{progressText(job)}
							{#if job.failed > 0}
								<span class="jobs__failed ml-2.5 text-danger-text">{$_('admin_jobs_failed_count', { values: { count: job.failed } })}</span>
							{/if}
						</p>
						<p class="jobs__times font-body font-normal text-[13px] leading-[1.5] text-text-faint">
							{#if job.startedAt !== null}
								{$_('admin_jobs_started', { values: { time: timeLabel(job.startedAt) } })}
							{:else}
								{$_('admin_jobs_not_started')}
							{/if}
							{#if job.finishedAt !== null}
								·
								{$_('admin_jobs_finished', { values: { time: timeLabel(job.finishedAt) } })}
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
