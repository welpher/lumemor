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
	// megabytes), the reruns' are asset counts. The kind decides the
	// formatting; the numbers themselves are the row's.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import { listJobRuns, type JobRun } from '$lib/api/jobruns';
	import { formatMegabytes } from '$lib/format';
	import { subscribeJobEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	/** The run kinds whose label keys the panel renders. */
	const KIND_LABEL_KEYS: Readonly<Record<string, string>> = {
		model_download: 'admin_jobs_kind_model_download',
		face_detect_rerun: 'admin_jobs_kind_face_detect_rerun',
		face_recognize_rerun: 'admin_jobs_kind_face_recognize_rerun',
		clip_embed_rerun: 'admin_jobs_kind_clip_embed_rerun',
		ocr_rerun: 'admin_jobs_kind_ocr_rerun'
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

<div class="jobs">
	{#if status === 'loading'}
		<p class="jobs__status" role="status">{$_('admin_jobs_loading')}</p>
	{:else if status === 'error'}
		<div class="jobs__failure" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}>{$_('action_retry')}</Button>
		</div>
	{:else if status === 'ready'}
		<header class="jobs__header">
			<h1 class="jobs__title">{$_('admin_jobs_title')}</h1>
			<p class="jobs__note">{$_('admin_jobs_note')}</p>
		</header>

		{#if jobs.length === 0}
			<p class="jobs__empty">{$_('admin_jobs_empty')}</p>
		{:else}
			<div class="jobs__rows">
				{#each jobs as job (job.id)}
					<article class="jobs__row" data-job-run-id={job.id}>
						<header class="jobs__row-head">
							<h2 class="jobs__kind">{kindLabel(job)}</h2>
							<span class="jobs__badge jobs__badge--{job.state}">{stateLabel(job)}</span>
						</header>
						<div
							class="jobs__progress"
							role="progressbar"
							aria-valuemin="0"
							aria-valuemax="100"
							aria-valuenow={percent(job)}
						>
							<div
								class="jobs__progress-fill jobs__progress-fill--{job.state}"
								style="width: {percent(job)}%"
							></div>
						</div>
						<p class="jobs__counts">
							{progressText(job)}
							{#if job.failed > 0}
								<span class="jobs__failed">{$_('admin_jobs_failed_count', { values: { count: job.failed } })}</span>
							{/if}
						</p>
						<p class="jobs__times">
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
	.jobs {
		max-width: 680px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.jobs__status {
		margin: 48px 0;
		text-align: center;
		color: var(--color-text-soft);
	}

	.jobs__failure {
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

	.jobs__failure p {
		margin: 0;
	}

	.jobs__header h1 {
		margin: 0 0 4px;
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 26px;
		letter-spacing: 0.01em;
		color: var(--color-text);
	}

	.jobs__note {
		margin: 0;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text-soft);
	}

	.jobs__empty {
		margin: 0;
		padding: 32px 24px;
		text-align: center;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text-soft);
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
	}

	.jobs__rows {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.jobs__row {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 16px 18px;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
	}

	.jobs__row-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
	}

	.jobs__kind {
		margin: 0;
		font: 600 16px/1.4 var(--font-body);
		color: var(--color-text);
	}

	.jobs__badge {
		display: inline-block;
		padding: 2px 10px;
		font: 600 12px/1.6 var(--font-body);
		color: var(--color-text-soft);
		background-color: var(--color-accent-soft);
		border-radius: var(--radius-control);
	}

	.jobs__badge--running {
		color: var(--color-accent-soft-text);
	}

	.jobs__badge--succeeded {
		color: var(--color-accent-soft-text);
	}

	.jobs__badge--failed {
		color: var(--color-danger-text);
		background-color: color-mix(in srgb, var(--color-danger) 12%, transparent);
	}

	.jobs__progress {
		width: 100%;
		height: 8px;
		background-color: var(--color-accent-soft);
		border-radius: var(--radius-control);
		overflow: hidden;
	}

	.jobs__progress-fill {
		height: 100%;
		background-color: var(--color-accent);
		transition: width var(--motion-base) var(--ease-standard);
	}

	.jobs__progress-fill--failed {
		background-color: var(--color-danger);
	}

	.jobs__counts {
		margin: 0;
		font: 500 14px/1.4 var(--font-body);
		color: var(--color-text);
		font-variant-numeric: tabular-nums;
	}

	.jobs__failed {
		margin-left: 10px;
		color: var(--color-danger-text);
	}

	.jobs__times {
		margin: 0;
		font: 400 13px/1.5 var(--font-body);
		color: var(--color-text-faint);
	}
</style>
