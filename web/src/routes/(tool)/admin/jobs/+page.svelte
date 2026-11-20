<script lang="ts">
	// The task panel of the admin surface, reshaped into a fixed task
	// CATALOG with the execution history on its own tab.
	//
	// THE TWO TABS: "Tasks" (the default) is the catalog -- one row per
	// capability with its state, its counts and its actions; "Recent
	// runs" is the execution history exactly as it was (each run's
	// kind, state, progress, timestamps, cancel and retry). The address
	// carries the face (`?tab=runs`), so a reload or a shared link
	// lands on the same tab; the history's rows link here from the
	// catalog's failed lines.
	//
	// THE CATALOG IS NOT A LISTING: GET /admin/jobs/summary answers the
	// whole catalog in one read -- the queue's per-kind counts, the run
	// budget's pause switches and the latest run of each batch kind.
	// The pause switches used to sit in the budget card; they live in
	// their task's own row now (a switch belongs to its task), while
	// the card keeps the instance-wide dials (concurrency, night
	// window). The five switches write through the SAME run-budget
	// document the card saves -- one source of truth, one PUT.
	//
	// THE ONE-OFF JOBS: the catalog never duplicates a dialog. The
	// rebuild and the place re-resolution rows link to their existing
	// settings dialogs (the deep links the settings page already
	// honors), the machine learning group links to the models page, and
	// the album rescan row starts the instance-wide sweep through its
	// own endpoint (the per-album trigger stays on the album page).
	//
	// THE VISIBLE-ONLY POLL: the summary refreshes every 5 seconds
	// while the page is visible and STOPS when the tab hides -- the
	// poll is an authenticated request, and an admin leaving the page
	// open overnight must not keep the instance's activity gate awake
	// (the folder sync's idle cadence waits on that gate). The batch
	// rows also converge on the existing job.* SSE events, so a visible
	// page sees a run start, progress and finish without waiting for a
	// tick.
	//
	// THE HISTORY'S CONVERGENCE is unchanged: active rows converge
	// live through the job.* events (progress in place, start/finish by
	// refetch) with the 5-second polling fallback while any row is
	// active; history rows are static.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import Button from '$lib/components/Button.svelte';
	import AdminPageHeader from '$lib/components/AdminPageHeader.svelte';
	import { toast } from '$lib/components/toast.svelte';
	import {
		cancelJobRun,
		getJobsSummary,
		listJobRuns,
		rescanAllAlbums,
		type JobsSummary,
		type JobRun
	} from '$lib/api/jobruns';
	import {
		listModels,
		rerunModel,
		startModelDownload,
		type AdminModel,
		type ModelSlotId,
		type SlotState
	} from '$lib/api/models';
	import { reresolvePlaces } from '$lib/api/places';
	import { getRunBudgetSettings, saveRunBudgetSettings, type RunBudgetSettings } from '$lib/api/adminSettings';
	import { formatMegabytes, formatDuration } from '$lib/format';
	import { subscribeJobEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';
	type JobsTab = 'tasks' | 'runs';

	/** One pause switch's field of the budget document. */
	type PauseField = keyof RunBudgetSettings['pausedJobs'];

	/** One pipeline kind's key in the summary's `pipeline` object. */
	type PipelineKind = 'thumbnail' | 'metadata' | 'transcode' | 'faceDetect' | 'faceRecognize' | 'clipEmbed' | 'ocr';

	/** The run kinds whose label keys the history renders. */
	const KIND_LABEL_KEYS: Readonly<Record<string, string>> = {
		model_download: 'admin_jobs_kind_model_download',
		face_detect_rerun: 'admin_jobs_kind_face_detect_rerun',
		face_recognize_rerun: 'admin_jobs_kind_face_recognize_rerun',
		clip_embed_rerun: 'admin_jobs_kind_clip_embed_rerun',
		ocr_rerun: 'admin_jobs_kind_ocr_rerun',
		places_rerun: 'admin_jobs_kind_places_rerun',
		album_rescan: 'admin_jobs_kind_album_rescan',
		album_rescan_all: 'admin_jobs_kind_album_rescan_all',
		rebuild_derivatives: 'admin_jobs_kind_rebuild_derivatives'
	};

	/** The state label keys, and the badge modifier classes. */
	const STATE_LABEL_KEYS: Readonly<Record<string, string>> = {
		queued: 'admin_jobs_state_queued',
		running: 'admin_jobs_state_running',
		succeeded: 'admin_jobs_state_succeeded',
		failed: 'admin_jobs_state_failed',
		cancelled: 'admin_jobs_state_cancelled'
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

	/** The zero triple of a pipeline row (a kind the queue never saw). */
	const NO_COUNTS = { waiting: 0, active: 0, failed: 0 };

	// THE TAB: the address decides (`?tab=runs`), never local state
	// alone -- the places page's own view switch discipline.
	const tab: JobsTab = $derived(page.url.searchParams.get('tab') === 'runs' ? 'runs' : 'tasks');

	/** Switch the face and carry it in the address (the deep link both ways). */
	function selectTab(next: JobsTab): void {
		if (next === tab) {
			return;
		}
		const url = new URL(page.url);
		if (next === 'runs') {
			url.searchParams.set('tab', 'runs');
		} else {
			url.searchParams.delete('tab');
		}
		void goto(`${url.pathname}${url.search}`, { replaceState: true, noScroll: true });
	}

	// ---- the catalog's summary ----
	let summary = $state<JobsSummary | null>(null);
	let summaryStatus = $state<ListStatus>('idle');
	let summaryErrorKey = $state('');
	/** Bumping re-reads the summary (the mount, a run's boundaries, a start). */
	let summaryTick = $state(0);
	/** The sweep's start in flight (the button's disabled state). */
	let startingRescan = $state(false);
	/** The catalog's own failure line (a refused start or cancel). */
	let catalogErrorKey = $state('');

	// ---- the history listing ----
	let jobs = $state<JobRun[]>([]);
	/** The models read behind the retry matrix and the missing-dependency diagnosis. */
	let models = $state<AdminModel[]>([]);
	/** The per-slot state rows: the catalog's model-download aggregate. */
	let modelSlots = $state<SlotState[]>([]);
	let status = $state<ListStatus>('idle');
	let errorKey = $state('');
	/** Bumping re-runs the history listing (the retry, the convergence). */
	let retryTick = $state(0);
	/** Bumping re-reads the models (the mount, a retry that may have changed them). */
	let modelsTick = $state(0);
	/** The failed run whose retry is in flight (null while none). */
	let retrying = $state<string | null>(null);
	/** The last failed retry, pinned to its row (null while none). */
	let retryError = $state<{ jobId: string; errorKey: string } | null>(null);

	// THE RUN SETTINGS STRIP: the budget read stands apart from the
	// summary's tick (its failure never fails the panel -- the card
	// simply does not render and the quiet line names it). The
	// controls save on change, immediately: the card has no save
	// button, and the whole-object PUT carries every control's
	// current position (the task rows' pause switches included).
	let formBudget = $state<RunBudgetSettings>({
		concurrencyPreset: 'standard',
		nightlyEnabled: false,
		nightlyStart: '02:00',
		nightlyEnd: '06:00',
		pausedJobs: { thumbnail: false, metadata: false, transcode: false, ml: false, places: false }
	});
	let budgetLoaded = $state(false);
	let budgetLoadError = $state(false);
	let savingBudget = $state(false);
	/** A control changed while a save was in flight: one more round. */
	let budgetDirty = $state(false);
	let budgetSaved = $state(false);
	let budgetErrorKey = $state('');
	/** Bumping re-reads the budget (a failed save's refetch arm). */
	let budgetTick = $state(0);
	/** The cancel in flight, pinned to its run (null while none). */
	let cancelling = $state<string | null>(null);
	/** The last failed cancel, pinned to its history row (null while none). */
	let cancelError = $state<{ jobId: string; errorKey: string } | null>(null);

	const activeLocale = $derived($locale ?? 'en');
	/** The fallback poll's cadence, in milliseconds. */
	const POLL_INTERVAL_MS = 5000;
	const hasActiveRuns = $derived(jobs.some((job) => job.state === 'queued' || job.state === 'running'));

	// The catalog's read, keyed on its tick. A superseded read -- the
	// tick moved, or the route left -- never applies.
	$effect(() => {
		void summaryTick;
		let current = true;
		// The status read is untracked: writing it below must not
		// re-run this read (the effect keys on the tick alone).
		if (untrack(() => summaryStatus) !== 'ready') {
			summaryStatus = 'loading';
		}
		summaryErrorKey = '';
		void (async () => {
			const outcome = await getJobsSummary();
			if (!current) {
				return;
			}
			if (!outcome.ok) {
				summaryStatus = 'error';
				summaryErrorKey = outcome.errorKey;
				return;
			}
			summary = outcome.summary;
			summaryStatus = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	// THE VISIBLE-ONLY TICK: while the tab is visible, the summary
	// re-reads every 5 seconds; hidden, the timer is cleared (no
	// authenticated request rides a background tab) and a return to
	// visibility refreshes once at once.
	$effect(() => {
		if (typeof document === 'undefined') {
			return;
		}
		let timer: ReturnType<typeof setInterval> | null = null;
		const start = (): void => {
			if (timer === null) {
				timer = setInterval(() => {
					summaryTick += 1;
				}, POLL_INTERVAL_MS);
			}
		};
		const stop = (): void => {
			if (timer !== null) {
				clearInterval(timer);
				timer = null;
			}
		};
		const onVisibility = (): void => {
			if (document.visibilityState === 'visible') {
				summaryTick += 1;
				start();
			} else {
				stop();
			}
		};
		if (document.visibilityState === 'visible') {
			start();
		}
		document.addEventListener('visibilitychange', onVisibility);
		return () => {
			stop();
			document.removeEventListener('visibilitychange', onVisibility);
		};
	});

	// The history listing, keyed on the retry tick. A superseded run --
	// the retry fired again, or the route left -- never applies.
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
	// dependency probe's fact and the catalog's slot aggregate, not the
	// runs' state, so the convergence never re-reads it. Its failure
	// never fails the panel -- the rows simply lose the model-backed
	// retry, the missing-dependency wording and the slot aggregate.
	$effect(() => {
		void modelsTick;
		let current = true;
		void (async () => {
			const outcome = await listModels();
			if (!current || !outcome.ok) {
				return;
			}
			models = outcome.models;
			modelSlots = outcome.slots ?? [];
		})();
		return () => {
			current = false;
		};
	});

	// The budget read, keyed on its own tick (a failed save's
	// refetch arm re-reads the server's truth). Failure is quiet:
	// the card and the rows' switches do not render, a line names it.
	$effect(() => {
		void budgetTick;
		let current = true;
		void (async () => {
			const outcome = await getRunBudgetSettings();
			if (!current) {
				return;
			}
			if (outcome.ok) {
				formBudget = { ...outcome.settings };
				budgetLoaded = true;
				budgetLoadError = false;
			} else {
				budgetLoaded = false;
				budgetLoadError = true;
			}
		})();
		return () => {
			current = false;
		};
	});

	// THE IMMEDIATE SAVE: every control's change PUTs the whole
	// document. A change that lands mid-save marks the dirty flag and
	// one more round carries it -- the last control's position is what
	// the server ends with. A failed save stays inline (the shared
	// failure shape), and the values are not reverted: the next change
	// re-saves the fresh positions. The boolean is the caller's
	// success signal (the task rows toast only a landed switch).
	async function saveBudget(): Promise<boolean> {
		if (savingBudget) {
			budgetDirty = true;
			return false;
		}
		savingBudget = true;
		let landed = true;
		do {
			budgetDirty = false;
			budgetErrorKey = '';
			budgetSaved = false;
			const outcome = await saveRunBudgetSettings({ ...formBudget });
			if (outcome.ok) {
				formBudget = { ...outcome.settings };
				budgetSaved = true;
			} else {
				landed = false;
				budgetErrorKey = outcome.errorKey;
			}
		} while (budgetDirty);
		savingBudget = false;
		return landed;
	}

	/**
	 * One task row's pause switch: flip the field, save the whole
	 * document, and toast the landed posture. The failed save's error
	 * stays inline beside the card (the shared strip feedback); the
	 * switch keeps the flipped position -- the next change re-saves.
	 */
	async function togglePause(field: PauseField, labelKey: string): Promise<void> {
		if (savingBudget) {
			return;
		}
		const paused = !formBudget.pausedJobs[field];
		formBudget = { ...formBudget, pausedJobs: { ...formBudget.pausedJobs, [field]: paused } };
		const landed = await saveBudget();
		if (!landed) {
			return;
		}
		toast(
			$_(paused ? 'admin_jobs_task_paused_toast' : 'admin_jobs_task_resumed_toast', {
				values: { task: $_(labelKey) }
			})
		);
	}

	/**
	 * The catalog's "rescan all": one sweep over every directory
	 * album. The 202 (fresh) and the 200 (an in-flight sweep joined)
	 * are the same outcome for the caller; the row's latest line reads
	 * the run's album count off the refreshed summary, which is what
	 * the toast speaks. The history converges on the new row through
	 * its own refetch.
	 */
	async function startRescanAll(): Promise<void> {
		if (startingRescan) {
			return;
		}
		startingRescan = true;
		catalogErrorKey = '';
		const outcome = await rescanAllAlbums();
		if (!outcome.ok) {
			startingRescan = false;
			catalogErrorKey = outcome.errorKey;
			return;
		}
		const refreshed = await getJobsSummary();
		startingRescan = false;
		if (refreshed.ok) {
			summary = refreshed.summary;
			summaryStatus = 'ready';
		}
		retryTick += 1;
		const count = refreshed.ok ? (refreshed.summary.runs.albumRescanAll?.total ?? 0) : 0;
		toast(
			count > 0
				? $_('admin_jobs_task_rescan_started', { values: { count } })
				: $_('admin_jobs_task_rescan_started_plain')
		);
	}

	/**
	 * The catalog rows' cancel: the same request the history's rows
	 * send (one flag, the driver closes at its boundary). The summary
	 * re-reads on the receipt; the history converges through the
	 * run's finished event. A refusal lands on the catalog's failure
	 * line, never a toast over the page.
	 */
	async function cancelBatchRun(run: { id: string; done: number }): Promise<void> {
		if (cancelling !== null) {
			return;
		}
		cancelling = run.id;
		catalogErrorKey = '';
		const outcome = await cancelJobRun(run.id);
		cancelling = null;
		if (!outcome.ok) {
			catalogErrorKey = outcome.errorKey;
			return;
		}
		toast($_('admin_jobs_cancelled', { values: { count: run.done } }));
		summaryTick += 1;
		retryTick += 1;
	}

	/**
	 * The history's cancel (E41): asks the in-flight run to end at its
	 * next chunk boundary. The 204 is only the flag's receipt -- the
	 * toast speaks the honest account (the finished count, read off the
	 * row at the click), the row itself converges to cancelled through
	 * the driver's close and its finished event. A 409 (the run
	 * finished between the listing and the click) lands INLINE under
	 * the row it belongs to.
	 */
	async function cancelRun(job: JobRun): Promise<void> {
		if (cancelling !== null) {
			return;
		}
		cancelling = job.id;
		cancelError = null;
		const outcome = await cancelJobRun(job.id);
		cancelling = null;
		if (!outcome.ok) {
			cancelError = { jobId: job.id, errorKey: outcome.errorKey };
			return;
		}
		toast($_('admin_jobs_cancelled', { values: { count: job.done } }));
		retryTick += 1;
		summaryTick += 1;
	}

	// The SSE convergence: progress events update the named history row
	// in place (the events' counts are already numbers) and the named
	// catalog run, the start and finish of any run converge through a
	// refetch (a run started elsewhere joins both faces).
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
				applyRunProgress(event.jobRunId, event.total, event.done, event.failed, event.state);
				return;
			}
			retryTick += 1;
			summaryTick += 1;
		});
	});

	/** Fold one progress event into the catalog's matching run slot. */
	function applyRunProgress(id: string, total: number, done: number, failed: number, state: string): void {
		if (summary === null) {
			return;
		}
		const runs = summary.runs;
		const slots = ['rebuildDerivatives', 'placesRerun', 'albumRescanAll', 'folderSync', 'modelDownload'] as const;
		let changed = false;
		const next = { ...runs };
		for (const slot of slots) {
			const run = runs[slot];
			if (run !== null && run.id === id) {
				next[slot] = {
					...run,
					total,
					done,
					failed,
					state: state as typeof run.state
				};
				changed = true;
			}
		}
		if (changed) {
			summary = { ...summary, runs: next };
		}
	}

	// The history's polling fallback while any row is active: history
	// rows are static, and the panel stops asking once nothing runs.
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

	/** One run state's label (the catalog's latest lines reuse it). */
	function stateText(state: string): string {
		const key = STATE_LABEL_KEYS[state];
		return key === undefined ? state : $_(key);
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
		if (job.kind === 'album_rescan_all') {
			return $_('admin_jobs_progress_albums', {
				values: { done: job.done, total: job.total }
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
	 * model, and the places rerun has its own endpoint. The rescan
	 * kinds do not -- their run rows carry no album id (the per-album
	 * trigger belongs to the album page, the sweep to the catalog).
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
		summaryTick += 1;
	}

	// ---- the catalog's read helpers ----

	/** One pipeline kind's queue counts (zeros before the read lands). */
	function countsOf(kind: PipelineKind): { waiting: number; active: number; failed: number } {
		const counts = summary?.pipeline[kind];
		return counts === undefined ? NO_COUNTS : counts;
	}

	/**
	 * The machine learning group's counts: the four per-asset kinds
	 * summed (the group is one pause switch, so its line speaks the
	 * group's queue).
	 */
	const mlCounts = $derived.by(() => {
		const kinds: PipelineKind[] = ['faceDetect', 'faceRecognize', 'clipEmbed', 'ocr'];
		return kinds.reduce(
			(tally, kind) => {
				const counts = countsOf(kind);
				return {
					waiting: tally.waiting + counts.waiting,
					active: tally.active + counts.active,
					failed: tally.failed + counts.failed
				};
			},
			{ ...NO_COUNTS }
		);
	});

	/** Whether a counts triple carries any work at all. */
	function hasWork(counts: { waiting: number; active: number; failed: number }): boolean {
		return counts.waiting + counts.active + counts.failed > 0;
	}

	/**
	 * One task row's state line: the pause posture first (a paused
	 * kind's new jobs sit held), then the queue counts when there are
	 * any, else the honest idle word.
	 */
	function taskState(counts: { waiting: number; active: number; failed: number }, paused: boolean): string {
		const parts: string[] = [];
		if (paused) {
			parts.push($_('admin_jobs_task_paused'));
		}
		if (hasWork(counts)) {
			parts.push($_('admin_jobs_task_counts', { values: counts }));
		} else if (!paused) {
			parts.push($_('admin_jobs_task_idle'));
		}
		return parts.join(' · ');
	}

	/** One ML subitem's compact counts line. */
	function subCounts(kind: PipelineKind): string {
		return $_('admin_jobs_task_counts_compact', { values: countsOf(kind) });
	}

	/** The latest line of a one-off task (none/failed/state + time). */
	function latestText(run: { state: string; startedAt?: string | null; finishedAt?: string | null } | null): string {
		if (run === null) {
			return $_('admin_jobs_task_latest_none');
		}
		if (run.state === 'failed') {
			return $_('admin_jobs_task_latest_failed');
		}
		const boundary = run.finishedAt ?? run.startedAt ?? null;
		const state = stateText(run.state);
		if (boundary === null) {
			return $_('admin_jobs_task_latest_state', { values: { state, time: '' } }).trim();
		}
		return $_('admin_jobs_task_latest_state', { values: { state, time: timeLabel(boundary) } });
	}

	/** The rescan's latest line: the sweep's album count leads. */
	function latestRescanText(
		run: { state: string; total: number; startedAt?: string | null; finishedAt?: string | null } | null
	): string {
		if (run === null) {
			return $_('admin_jobs_task_latest_none');
		}
		if (run.state === 'failed') {
			return $_('admin_jobs_task_latest_failed');
		}
		const boundary = run.finishedAt ?? run.startedAt ?? null;
		const state = stateText(run.state);
		const time = boundary === null ? '' : timeLabel(boundary);
		return $_('admin_jobs_task_latest_rescan_all', { values: { count: run.total, state, time } }).trim();
	}

	/** Whether the latest run of this slot is still in flight. */
	function inFlight(run: { state: string } | null): boolean {
		return run !== null && (run.state === 'queued' || run.state === 'running');
	}

	/** The readied model slots (the catalog's aggregate line). */
	const readySlots = $derived(modelSlots.filter((slot) => slot.state === 'active').length);
	/** Whether a model download run is filling any slot right now. */
	const downloadingSlots = $derived(modelSlots.some((slot) => slot.state === 'downloading'));
</script>

<svelte:head>
	<title>Jobs · lumemor</title>
</svelte:head>

<div class="jobs max-w-[760px] mx-auto flex flex-col gap-5">
	<!-- The page-header row: the title and the two faces. The catalog
	     is the default; the history keeps its own address (`?tab=runs`). -->
	<AdminPageHeader title={$_('admin_jobs_title')}>
		{#snippet actions()}
			<div
				class="jobs__tabs inline-flex items-center gap-1 p-1 bg-page border border-border rounded-control"
				role="group"
				aria-label={$_('admin_jobs_tabs_label')}
			>
				<button
					class="inline-flex items-center px-2.5 py-1 font-body font-semibold text-[13px] leading-[1.4] rounded-control border-0 bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard aria-pressed:bg-accent-soft aria-pressed:text-accent-soft-text"
					type="button"
					aria-pressed={tab === 'tasks'}
					data-testid="jobs-tab-tasks"
					onclick={() => selectTab('tasks')}
				>
					{$_('admin_jobs_tab_tasks')}
				</button>
				<button
					class="inline-flex items-center px-2.5 py-1 font-body font-semibold text-[13px] leading-[1.4] rounded-control border-0 bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard aria-pressed:bg-accent-soft aria-pressed:text-accent-soft-text"
					type="button"
					aria-pressed={tab === 'runs'}
					data-testid="jobs-tab-runs"
					onclick={() => selectTab('runs')}
				>
					{$_('admin_jobs_tab_runs')}
				</button>
			</div>
		{/snippet}
	</AdminPageHeader>
	<p class="jobs__note m-0 mb-1 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
		{$_('admin_jobs_note')}
	</p>

	<!-- ===== the catalog (the default face) ===== -->
	<div class="flex flex-col gap-5" data-jobs-panel="tasks" class:hidden={tab !== 'tasks'}>
		{#if summaryStatus === 'loading' && summary === null}
			<p class="jobs__status my-12 text-center text-text-soft animate-loading-in" role="status">
				{$_('admin_jobs_loading')}
			</p>
		{:else if summaryStatus === 'error' && summary === null}
			<div
				class="jobs__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
				role="alert"
			>
				<p>{errorText(summaryErrorKey)}</p>
				<Button variant="primary" onclick={() => (summaryTick += 1)}
					><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
				>
			</div>
		{:else if summary !== null}
			<!-- THE RUN SETTINGS CARD: the instance-wide dials only. The
			     five per-kind pause switches moved into their task rows
			     (a switch belongs to its task). No save button: every
			     control saves on change, immediately. -->
			{#if budgetLoaded}
				<section
					class="jobs__run-settings bg-card border border-border rounded-card shadow-card p-5"
					data-testid="run-settings"
				>
					<div class="flex flex-wrap items-center gap-2 mb-1">
						<h2
							class="m-0 flex items-center gap-2 font-display font-bold text-[17px] leading-[1.3] text-text"
						>
							<SlidersHorizontal size={18} class="text-accent" aria-hidden="true" />
							{$_('admin_jobs_run_title')}
						</h2>
						<span class="ml-auto font-body font-normal text-[12px] leading-[1.4] text-text-faint">
							{$_('admin_jobs_run_note')}
						</span>
					</div>
					<div class="mt-3 flex flex-col gap-4">
						<!-- The concurrency dial: three steps, aria-pressed, no
						     number input. -->
						<div class="flex flex-wrap items-center gap-3">
							<span
								class="font-body font-semibold text-[14px] leading-[1.4] text-text"
								id="run-concurrency-label"
							>
								{$_('admin_jobs_run_concurrency')}
							</span>
							<div
								class="inline-flex gap-1 p-1 bg-page border border-border rounded-control"
								role="group"
								aria-labelledby="run-concurrency-label"
							>
								{#each [{ value: 'light', labelKey: 'admin_jobs_run_concurrency_light', testid: 'run-dial-light' }, { value: 'standard', labelKey: 'admin_jobs_run_concurrency_standard', testid: 'run-dial-standard' }, { value: 'full', labelKey: 'admin_jobs_run_concurrency_full', testid: 'run-dial-full' }] as step (step.value)}
									<button
										class="px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border-0 cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard bg-transparent text-text-soft aria-pressed:bg-card aria-pressed:text-text aria-pressed:shadow-card"
										type="button"
										data-testid={step.testid}
										aria-pressed={formBudget.concurrencyPreset === step.value}
										onclick={() => {
											formBudget.concurrencyPreset =
												step.value as RunBudgetSettings['concurrencyPreset'];
											void saveBudget();
										}}
									>
										{$_(step.labelKey)}
									</button>
								{/each}
							</div>
							<span class="font-body font-normal text-[12px] leading-[1.4] text-text-faint">
								{$_('admin_jobs_run_concurrency_hint')}
							</span>
						</div>
						<!-- The night window: the switch, the two times (dead
						     while the window is off), the honest scope line. -->
						<div class="flex flex-wrap items-center gap-3">
							<label
								class="settings__toggle flex items-center gap-2 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
							>
								<input
									class="settings__toggle-input w-[18px] h-[18px] accent-accent"
									type="checkbox"
									data-testid="run-nightly-toggle"
									bind:checked={formBudget.nightlyEnabled}
									onchange={() => void saveBudget()}
								/>
								<span>{$_('admin_jobs_run_nightly')}</span>
							</label>
							<label
								class="flex items-center gap-2 font-body font-normal text-[13px] leading-[1.4] text-text-soft"
							>
								{$_('admin_jobs_run_nightly_from')}
								<input
									class="px-2 py-1 w-[110px] font-body font-normal text-[13px] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard disabled:opacity-55"
									type="time"
									data-testid="run-nightly-start"
									bind:value={formBudget.nightlyStart}
									disabled={!formBudget.nightlyEnabled}
									onchange={() => void saveBudget()}
								/>
							</label>
							<label
								class="flex items-center gap-2 font-body font-normal text-[13px] leading-[1.4] text-text-soft"
							>
								{$_('admin_jobs_run_nightly_to')}
								<input
									class="px-2 py-1 w-[110px] font-body font-normal text-[13px] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard disabled:opacity-55"
									type="time"
									data-testid="run-nightly-end"
									bind:value={formBudget.nightlyEnd}
									disabled={!formBudget.nightlyEnabled}
									onchange={() => void saveBudget()}
								/>
							</label>
							<span class="font-body font-normal text-[12px] leading-[1.4] text-text-faint">
								{$_('admin_jobs_run_nightly_hint')}
							</span>
						</div>
						<!-- The card's own feedback: the quiet saved line, the
						     failed save's error (the task rows' switches ride the
						     same save). -->
						{#if savingBudget}
							<p
								class="m-0 font-body font-normal text-[12px] leading-[1.4] text-text-faint"
								role="status"
							>
								{$_('admin_jobs_run_saving')}
							</p>
						{:else if budgetSaved}
							<p class="m-0 font-body font-normal text-[12px] leading-[1.4] text-text-soft" role="status">
								{$_('admin_jobs_run_saved')}
							</p>
						{/if}
						{#if budgetErrorKey !== ''}
							<p
								class="m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
								role="alert"
							>
								{errorText(budgetErrorKey)}
							</p>
						{/if}
					</div>
				</section>
			{:else if budgetLoadError}
				<p class="m-0 font-body font-normal text-[13px] leading-[1.5] text-text-faint" role="status">
					{$_('admin_jobs_run_save_failed')}
				</p>
			{/if}

			<!-- THE FIXED CATALOG: one row per capability. A row states
			     its queue (or its latest run), carries its own pause
			     switch, and points its one-off actions at the surface
			     that already owns them. -->
			<div class="jobs__catalog flex flex-col gap-2.5" data-task-catalog>
				<!-- Thumbnails -->
				<article
					class="jobs__row jobs__task flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
					data-task="thumbnail"
				>
					<div class="flex flex-wrap items-center gap-2">
						<h2 class="m-0 font-body font-semibold text-[16px] leading-[1.4] text-text">
							{$_('admin_jobs_task_thumbnail')}
						</h2>
						<span class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
							>{taskState(countsOf('thumbnail'), formBudget.pausedJobs.thumbnail)}</span
						>
						{#if budgetLoaded}
							<span class="ml-auto">
								<button
									class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard {formBudget
										.pausedJobs.thumbnail
										? 'border-transparent bg-accent text-accent-on hover:bg-accent-strong'
										: 'border-border-strong bg-transparent text-text hover:bg-accent-soft hover:text-accent-soft-text'} disabled:opacity-55 disabled:cursor-not-allowed"
									type="button"
									disabled={savingBudget}
									data-testid="task-pause-thumbnail"
									onclick={() => void togglePause('thumbnail', 'admin_jobs_task_thumbnail')}
								>
									{formBudget.pausedJobs.thumbnail
										? $_('admin_jobs_task_resume')
										: $_('admin_jobs_task_pause')}
								</button>
							</span>
						{/if}
					</div>
				</article>

				<!-- Metadata extraction -->
				<article
					class="jobs__row jobs__task flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
					data-task="metadata"
				>
					<div class="flex flex-wrap items-center gap-2">
						<h2 class="m-0 font-body font-semibold text-[16px] leading-[1.4] text-text">
							{$_('admin_jobs_task_metadata')}
						</h2>
						<span class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
							>{taskState(countsOf('metadata'), formBudget.pausedJobs.metadata)}</span
						>
						{#if budgetLoaded}
							<span class="ml-auto">
								<button
									class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard {formBudget
										.pausedJobs.metadata
										? 'border-transparent bg-accent text-accent-on hover:bg-accent-strong'
										: 'border-border-strong bg-transparent text-text hover:bg-accent-soft hover:text-accent-soft-text'} disabled:opacity-55 disabled:cursor-not-allowed"
									type="button"
									disabled={savingBudget}
									data-testid="task-pause-metadata"
									onclick={() => void togglePause('metadata', 'admin_jobs_task_metadata')}
								>
									{formBudget.pausedJobs.metadata
										? $_('admin_jobs_task_resume')
										: $_('admin_jobs_task_pause')}
								</button>
							</span>
						{/if}
					</div>
				</article>

				<!-- Transcoding -->
				<article
					class="jobs__row jobs__task flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
					data-task="transcode"
				>
					<div class="flex flex-wrap items-center gap-2">
						<h2 class="m-0 font-body font-semibold text-[16px] leading-[1.4] text-text">
							{$_('admin_jobs_task_transcode')}
						</h2>
						<span class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
							>{taskState(countsOf('transcode'), formBudget.pausedJobs.transcode)}</span
						>
						{#if budgetLoaded}
							<span class="ml-auto">
								<button
									class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard {formBudget
										.pausedJobs.transcode
										? 'border-transparent bg-accent text-accent-on hover:bg-accent-strong'
										: 'border-border-strong bg-transparent text-text hover:bg-accent-soft hover:text-accent-soft-text'} disabled:opacity-55 disabled:cursor-not-allowed"
									type="button"
									disabled={savingBudget}
									data-testid="task-pause-transcode"
									onclick={() => void togglePause('transcode', 'admin_jobs_task_transcode')}
								>
									{formBudget.pausedJobs.transcode
										? $_('admin_jobs_task_resume')
										: $_('admin_jobs_task_pause')}
								</button>
							</span>
						{/if}
					</div>
				</article>

				<!-- The machine learning group: one shared pause switch, the
				     four per-asset kinds' counts as subitems, and the models
				     page's own surface. -->
				<article
					class="jobs__row jobs__task flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
					data-task="ml"
				>
					<div class="flex flex-wrap items-center gap-2">
						<h2 class="m-0 font-body font-semibold text-[16px] leading-[1.4] text-text">
							{$_('admin_jobs_task_ml')}
						</h2>
						<span class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
							>{taskState(mlCounts, formBudget.pausedJobs.ml)}</span
						>
						<span class="ml-auto flex items-center gap-2">
							<a
								class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border border-border-strong bg-transparent text-text no-underline hover:bg-accent-soft hover:text-accent-soft-text"
								href="/admin/models">{$_('admin_jobs_task_go_model_page')}</a
							>
							{#if budgetLoaded}
								<button
									class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard {formBudget
										.pausedJobs.ml
										? 'border-transparent bg-accent text-accent-on hover:bg-accent-strong'
										: 'border-border-strong bg-transparent text-text hover:bg-accent-soft hover:text-accent-soft-text'} disabled:opacity-55 disabled:cursor-not-allowed"
									type="button"
									disabled={savingBudget}
									data-testid="task-pause-ml"
									onclick={() => void togglePause('ml', 'admin_jobs_task_ml')}
								>
									{formBudget.pausedJobs.ml
										? $_('admin_jobs_task_resume')
										: $_('admin_jobs_task_pause')}
								</button>
							{/if}
						</span>
					</div>
					<ul
						class="m-0 p-0 list-none flex flex-wrap items-center gap-x-5 gap-y-1 font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
					>
						<li>{$_('admin_jobs_task_face_detect')} · {subCounts('faceDetect')}</li>
						<li>{$_('admin_jobs_task_face_recognize')} · {subCounts('faceRecognize')}</li>
						<li>{$_('admin_jobs_task_clip')} · {subCounts('clipEmbed')}</li>
						<li>{$_('admin_jobs_task_ocr')} · {subCounts('ocr')}</li>
					</ul>
				</article>

				<!-- Place resolution: the switch plus the settings page's
				     existing rebuild dialog. -->
				<article
					class="jobs__row jobs__task flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
					data-task="places"
				>
					<div class="flex flex-wrap items-center gap-2">
						<h2 class="m-0 font-body font-semibold text-[16px] leading-[1.4] text-text">
							{$_('admin_jobs_task_places')}
						</h2>
						<span class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
							>{latestText(summary.runs.placesRerun)}</span
						>
						<span class="ml-auto flex items-center gap-2">
							<a
								class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border border-border-strong bg-transparent text-text no-underline hover:bg-accent-soft hover:text-accent-soft-text"
								href="/admin/settings?dialog=rebuild-places">{$_('admin_jobs_task_rebuild_places')}</a
							>
							{#if budgetLoaded}
								<button
									class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard {formBudget
										.pausedJobs.places
										? 'border-transparent bg-accent text-accent-on hover:bg-accent-strong'
										: 'border-border-strong bg-transparent text-text hover:bg-accent-soft hover:text-accent-soft-text'} disabled:opacity-55 disabled:cursor-not-allowed"
									type="button"
									disabled={savingBudget}
									data-testid="task-pause-places"
									onclick={() => void togglePause('places', 'admin_jobs_task_places')}
								>
									{formBudget.pausedJobs.places
										? $_('admin_jobs_task_resume')
										: $_('admin_jobs_task_pause')}
								</button>
							{/if}
						</span>
					</div>
					{#if summary.runs.placesRerun !== null && summary.runs.placesRerun.state === 'failed'}
						<a
							class="self-start font-body font-semibold text-[13px] leading-[1.4] text-accent no-underline hover:underline"
							href="?tab=runs">{$_('admin_jobs_task_view_runs')}</a
						>
					{/if}
				</article>

				<!-- Derivative rebuild: the settings page's existing dialog. -->
				<article
					class="jobs__row jobs__task flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
					data-task="rebuild"
				>
					<div class="flex flex-wrap items-center gap-2">
						<h2 class="m-0 font-body font-semibold text-[16px] leading-[1.4] text-text">
							{$_('admin_jobs_task_rebuild')}
						</h2>
						<span class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
							>{latestText(summary.runs.rebuildDerivatives)}</span
						>
						<span class="ml-auto">
							<a
								class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border border-border-strong bg-transparent text-text no-underline hover:bg-accent-soft hover:text-accent-soft-text"
								href="/admin/settings?dialog=rebuild-derivatives"
								>{$_('admin_jobs_task_rebuild_action')}</a
							>
						</span>
					</div>
					{#if summary.runs.rebuildDerivatives !== null && summary.runs.rebuildDerivatives.state === 'failed'}
						<a
							class="self-start font-body font-semibold text-[13px] leading-[1.4] text-accent no-underline hover:underline"
							href="?tab=runs">{$_('admin_jobs_task_view_runs')}</a
						>
					{/if}
				</article>

				<!-- Album rescan: ONE sweep over every directory album; the
				     per-album trigger stays on the album page. -->
				<article
					class="jobs__row jobs__task flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
					data-task="rescan"
				>
					<div class="flex flex-wrap items-center gap-2">
						<h2 class="m-0 font-body font-semibold text-[16px] leading-[1.4] text-text">
							{$_('admin_jobs_task_rescan')}
						</h2>
						<span class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
							>{latestRescanText(summary.runs.albumRescanAll)}</span
						>
						<span class="ml-auto flex items-center gap-2">
							{#if inFlight(summary.runs.albumRescanAll)}
								<button
									class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border border-danger bg-transparent text-danger-text cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)] disabled:opacity-55 disabled:cursor-not-allowed"
									type="button"
									disabled={cancelling !== null}
									data-testid="task-cancel-rescan"
									onclick={() => {
										const run = summary?.runs.albumRescanAll;
										if (run !== null && run !== undefined) {
											void cancelBatchRun(run);
										}
									}}
								>
									{$_('admin_jobs_cancel')}
								</button>
							{/if}
							<button
								class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border-0 bg-accent text-accent-on cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-strong disabled:opacity-55 disabled:cursor-not-allowed"
								type="button"
								disabled={startingRescan}
								data-testid="task-rescan-all"
								onclick={() => void startRescanAll()}
							>
								{$_('admin_jobs_task_rescan_all')}
							</button>
						</span>
					</div>
					<span class="font-body font-normal text-[12px] leading-[1.4] text-text-faint"
						>{$_('admin_jobs_task_rescan_hint')}</span
					>
					{#if summary.runs.albumRescanAll !== null && summary.runs.albumRescanAll.state === 'failed'}
						<a
							class="self-start font-body font-semibold text-[13px] leading-[1.4] text-accent no-underline hover:underline"
							href="?tab=runs">{$_('admin_jobs_task_view_runs')}</a
						>
					{/if}
				</article>

				<!-- Folder sync: periodic and read-only -- its copy states the
				     activity-gated cadence instead of a run. -->
				<article
					class="jobs__row jobs__task flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
					data-task="foldersync"
				>
					<div class="flex flex-wrap items-center gap-2">
						<h2 class="m-0 font-body font-semibold text-[16px] leading-[1.4] text-text">
							{$_('admin_jobs_task_foldersync')}
						</h2>
						<span class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
							>{$_('admin_jobs_task_foldersync_cadence')}</span
						>
					</div>
				</article>

				<!-- Model downloads: the slots' aggregate plus the models page. -->
				<article
					class="jobs__row jobs__task flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
					data-task="models"
				>
					<div class="flex flex-wrap items-center gap-2">
						<h2 class="m-0 font-body font-semibold text-[16px] leading-[1.4] text-text">
							{$_('admin_jobs_task_models')}
						</h2>
						<span class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums">
							{#if modelSlots.length > 0}
								{$_('admin_jobs_task_models_ready', {
									values: { ready: readySlots, total: modelSlots.length }
								})}
								{#if downloadingSlots}
									· {$_('admin_jobs_task_models_downloading')}
								{/if}
							{:else}
								{$_('admin_jobs_task_idle')}
							{/if}
						</span>
						<span class="ml-auto">
							<a
								class="inline-flex items-center px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control border border-border-strong bg-transparent text-text no-underline hover:bg-accent-soft hover:text-accent-soft-text"
								href="/admin/models">{$_('admin_jobs_task_go_models')}</a
							>
						</span>
					</div>
				</article>
			</div>

			{#if catalogErrorKey !== ''}
				<p
					class="m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
					role="alert"
					data-testid="catalog-error"
				>
					{errorText(catalogErrorKey)}
				</p>
			{/if}
		{/if}
	</div>

	<!-- ===== the execution history (its own tab) ===== -->
	<div class="flex flex-col gap-3" data-jobs-panel="runs" class:hidden={tab !== 'runs'}>
		{#if status === 'loading' && jobs.length === 0}
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
									{#if job.subject !== null && job.subject !== undefined}
										<!-- The row speaks: a rescan names its album.
										     The instance-wide reruns have no subject and get
										     the kind's own label -- nothing is invented. -->
										<span
											class="jobs__subject font-body font-normal text-[14px] leading-[1.4] text-text-soft"
											>{$_('admin_jobs_subject_rescan', { values: { name: job.subject } })}</span
										>
									{:else}
										{kindLabel(job)}
									{/if}
								</h2>
								<span
									class="jobs__badge jobs__badge--{job.state} inline-block px-2.5 py-0.5 font-body font-semibold text-[12px] leading-[1.6] text-text-soft bg-accent-soft rounded-control"
									>{stateLabel(job)}</span
								>
							</header>

							{#if job.kind === 'album_rescan' && job.state !== 'queued' && job.done === job.total && job.failed === 0}
								<!-- The rescan's own line: the row reads like a log
								     -- a scan that got through its scope without a single
								     failure says what it saw, in the run's own units. -->
								<p
									class="jobs__counts font-body font-medium text-[14px] leading-[1.4] text-text tabular-nums"
								>
									{$_('admin_jobs_rescan_unchanged', { values: { count: job.done } })}
								</p>
							{:else if job.state === 'queued'}
								<!-- A queued run has touched nothing yet: the bar
								     would sit at zero and read as a stall. -->
								<p
									class="jobs__waiting m-0 font-body font-normal text-[14px] leading-[1.4] text-text-soft"
								>
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

							<!-- The cancel: the three-state action discipline --
							     running and queued carry it, the failed rows keep
							     their own retry, and a completed row carries NOTHING
							     (honesty -- there is nothing to do about finished
							     work). The request flags the run; the driver closes
							     it at the next chunk boundary with the committed
							     portion kept. -->
							{#if job.state === 'running' || job.state === 'queued'}
								<div class="jobs__actions flex flex-wrap items-center gap-2">
									<Button
										variant="ghost"
										disabled={cancelling !== null}
										data-testid="job-cancel"
										onclick={() => void cancelRun(job)}
									>
										{$_('admin_jobs_cancel')}
									</Button>
								</div>
							{/if}
							{#if cancelError !== null && cancelError.jobId === job.id}
								<!-- The 409's inline error: the refusal lands under the
								     row it belongs to, never a toast over the listing. -->
								<p
									class="jobs__cancel-error m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
									role="alert"
									data-testid="job-cancel-error"
								>
									{errorText(cancelError.errorKey)}
								</p>
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

	/* The cancelled chip: the admin's own deliberate ending -- an
	 * ended run, outlined neutral -- never the danger tones a broken
	 * run wears. */
	.jobs__badge--cancelled {
		color: var(--color-text-soft);
		background-color: transparent;
		border: 1px solid var(--color-border);
	}

	.jobs__progress-fill--failed {
		background-color: var(--color-danger);
	}
</style>
