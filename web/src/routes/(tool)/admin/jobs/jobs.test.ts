import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearToasts, toasts } from '$lib/components/toast.svelte';

// The admin task panel, reshaped into the fixed catalog plus the
// history tab: the catalog's rows (counts, pause switches, latest
// runs, links), the two-tab address, the rescan-all trigger and the
// catalog cancel, the visible-only summary poll; and the history's
// unchanged behaviour (per-kind units, SSE convergence, the fallback
// poll, cancel and retry). The events module is mocked with a
// captured handler registry, and the route (`$app/state` /
// `$app/navigation`) with a runes holder, the same patterns the
// places and viewer pages use.

const { handlers } = vi.hoisted(() => ({
	handlers: [] as Array<(event: unknown) => void>
}));

const { gotoMock } = vi.hoisted(() => ({
	gotoMock: vi.fn(async (..._args: unknown[]) => undefined)
}));

vi.mock('$lib/events', () => ({
	subscribeJobEvents: (listener: (event: unknown) => void) => {
		handlers.push(listener);
		return () => {
			const index = handlers.indexOf(listener);
			if (index !== -1) {
				handlers.splice(index, 1);
			}
		};
	}
}));

vi.mock('$app/state', async () => ({
	page: (await import('./jobs-page-state.svelte')).jobsPageState
}));
vi.mock('$app/navigation', async () => {
	const { jobsPageState } = await import('./jobs-page-state.svelte');
	return {
		goto: (url: URL | string, options?: { replaceState?: boolean }) => {
			jobsPageState.url = new URL(String(url), 'http://localhost');
			return gotoMock(url, options);
		}
	};
});

/** One job run row (the JobRunResponse shape). */
function jobRun(id: string, overrides: Record<string, unknown> = {}) {
	return {
		id,
		kind: 'model_download',
		state: 'running',
		subject: null,
		total: 400371536,
		done: 104857600,
		failed: 0,
		startedAt: '2026-11-03T09:12:00Z',
		finishedAt: null,
		...overrides
	};
}

const downloadRun = jobRun('0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10');
const embedRun = jobRun('0192f8b1-4c5d-8e9f-a0b1-2c3d4e5f6a70', {
	kind: 'clip_embed_rerun',
	state: 'succeeded',
	total: 3,
	done: 3,
	startedAt: '2026-11-03T10:00:00Z',
	finishedAt: '2026-11-03T10:02:30Z'
});
const ocrRun = jobRun('0192f8c1-3d4e-9f0a-b1c2-3d4e5f6a7080', {
	kind: 'ocr_rerun',
	state: 'failed',
	total: 5,
	done: 4,
	failed: 1,
	startedAt: '2026-11-03T11:00:00Z',
	finishedAt: '2026-11-03T11:01:10Z'
});
const queuedRun = jobRun('0192f8d1-2e3f-4a5b-6c7d-8e9f0a1b2c3d', {
	kind: 'face_detect_rerun',
	state: 'queued',
	total: 10,
	done: 0,
	startedAt: null
});
const rescanRun = jobRun('0192f8e1-4f5a-6b7c-8d9e-0a1b2c3d4e5f', {
	kind: 'album_rescan',
	state: 'succeeded',
	subject: 'Trip 2026',
	total: 2,
	done: 2,
	startedAt: '2026-11-03T12:00:00Z',
	finishedAt: '2026-11-03T12:00:04Z'
});
const placesRun = jobRun('0192f8f1-5a6b-7c8d-9e0f-1a2b3c4d5e6a', {
	kind: 'places_rerun',
	state: 'succeeded',
	total: 6,
	done: 6,
	startedAt: '2026-11-03T13:00:00Z',
	finishedAt: '2026-11-03T13:00:12Z'
});

/** One catalog run summary (the JobRunSummary shape). */
function runSummary(id: string, overrides: Record<string, unknown> = {}) {
	return {
		id,
		state: 'succeeded',
		total: 12,
		done: 12,
		failed: 0,
		startedAt: '2026-11-19T02:00:00Z',
		finishedAt: '2026-11-19T02:04:10Z',
		...overrides
	};
}

/** The catalog's summary document (the JobsSummaryResponse shape). */
function summaryBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		pipeline: {
			thumbnail: { waiting: 3, active: 2, failed: 0 },
			metadata: { waiting: 0, active: 0, failed: 0 },
			transcode: { waiting: 4, active: 0, failed: 0 },
			faceDetect: { waiting: 3, active: 0, failed: 0 },
			faceRecognize: { waiting: 2, active: 0, failed: 0 },
			clipEmbed: { waiting: 0, active: 0, failed: 0 },
			ocr: { waiting: 0, active: 0, failed: 0 }
		},
		paused: { thumbnail: false, metadata: false, transcode: false, ml: false, places: false },
		runs: {
			rebuildDerivatives: runSummary('0192f900-0000-7000-8000-000000000001', { total: 1240, done: 1240 }),
			placesRerun: runSummary('0192f900-0000-7000-8000-000000000002', { total: 12847, done: 12847 }),
			albumRescanAll: null,
			folderSync: null,
			modelDownload: runSummary('0192f900-0000-7000-8000-000000000003', {
				total: 400371536,
				done: 400371536
			})
		},
		...overrides
	};
}

/** One model slot row (the SlotStateResponse shape). */
function slot(slotId: string, state: string) {
	return { slot: slotId, state, modelId: null, source: null, progress: null };
}

/** The budget document (the RunBudgetSettingsResponse shape). */
function budgetBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		concurrencyPreset: 'standard',
		nightlyEnabled: false,
		nightlyStart: '02:00',
		nightlyEnd: '06:00',
		pausedJobs: { thumbnail: false, metadata: false, transcode: false, ml: false, places: false },
		...overrides
	};
}

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/**
 * One model row (the ModelResponse shape), for the dependency probes.
 */
function model(id: string, overrides: Record<string, unknown> = {}) {
	return {
		id,
		slot: 'ocr',
		version: 'v4-mobile',
		source: 'downloaded',
		license: 'Apache-2.0',
		notice: null,
		dim: null,
		inputSize: { width: 320, height: 320 },
		isActive: true,
		filesPresent: true,
		params: {},
		...overrides
	};
}

/** The full panel's fetch stub: summary, listing, models and budget. */
function stubFetch(
	jobs: unknown[],
	options: {
		summary?: unknown | ((call: number) => unknown);
		models?: unknown[];
		slots?: unknown[];
		budget?: unknown;
		extra?: (input: RequestInfo | URL, init?: RequestInit) => Response | undefined;
	} = {}
): ReturnType<typeof vi.fn> {
	let summaryCalls = 0;
	const summary = options.summary ?? summaryBody();
	const budget = options.budget ?? budgetBody();
	const modelSlots = options.slots ?? [
		slot('face_detect', 'active'),
		slot('face_recognize', 'active'),
		slot('clip_embed', 'active'),
		slot('ocr', 'downloading'),
		slot('clip_text', 'not_downloaded')
	];
	return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		const extra = options.extra?.(input, init);
		if (extra !== undefined) {
			return extra;
		}
		if (url === '/api/v1/admin/jobs/summary') {
			summaryCalls += 1;
			return jsonResponse(200, typeof summary === 'function' ? summary(summaryCalls) : summary);
		}
		if (url === '/api/v1/admin/jobs') {
			return jsonResponse(200, { jobs });
		}
		if (url === '/api/v1/admin/models') {
			return jsonResponse(200, {
				models: options.models ?? [],
				download: null,
				slots: modelSlots,
				official: []
			});
		}
		if (url === '/api/v1/admin/settings/run-budget') {
			if (init?.method === 'PUT') {
				return jsonResponse(200, JSON.parse(String(init.body)));
			}
			return jsonResponse(200, budget);
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
}

/** The summary reads so far. */
function summaryCalls(fetchMock: ReturnType<typeof vi.fn>): number {
	return fetchMock.mock.calls.filter((call) => String(call[0]) === '/api/v1/admin/jobs/summary').length;
}

/** The history listing reads so far. */
function listingCalls(fetchMock: ReturnType<typeof vi.fn>): number {
	return fetchMock.mock.calls.filter((call) => String(call[0]) === '/api/v1/admin/jobs').length;
}

async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise<void>((resolve) => {
		requestAnimationFrame(() => resolve());
	}).catch(() => undefined);
	await tick();
}

// The runes holder the mock factory shares: the address decides the
// face, so a mount sets it first.
let jobsState: { url: URL };

/** The page mounted by the current test (the unmount helper's handle). */
let mountedPage: ReturnType<typeof mount> | null = null;

/** Mount the page on one tab (the address carries it). */
function mountPage(tab: 'tasks' | 'runs' = 'tasks'): ReturnType<typeof mount> {
	jobsState.url = new URL(`http://localhost/admin/jobs${tab === 'runs' ? '?tab=runs' : ''}`);
	const mounted = mount(Page, { target: document.body, props: {} });
	flushSync();
	// Every mount is tracked: a test that forgets to unmount would
	// otherwise leave its effects (and the visibility poll's timer)
	// listening for the NEXT test's events.
	mountedPage = mounted;
	return mounted;
}

/** The current test's page (the tests that unmount through the handle). */
function page(): ReturnType<typeof mount> {
	if (mountedPage === null) {
		throw new Error('no page mounted');
	}
	return mountedPage;
}

function buttonByText(text: string): HTMLButtonElement {
	const buttons = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === text);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

/** The run row whose data attribute names the run. */
function runCard(id: string): HTMLElement {
	const cards = Array.from(document.body.querySelectorAll<HTMLElement>('[data-job-run-id]'));
	const found = cards.find((card) => card.dataset.jobRunId === id);
	if (found === undefined) {
		throw new Error(`missing job run card ${id}`);
	}
	return found;
}

/** One catalog row whose data attribute names the task. */
function taskRow(task: string): HTMLElement {
	const row = document.body.querySelector<HTMLElement>(`[data-task="${task}"]`);
	if (row === null) {
		throw new Error(`missing task row ${task}`);
	}
	return row;
}

/** Deliver one SSE job event to the page's subscription. */
function deliverJobEvent(event: {
	name: string;
	jobRunId: string;
	kind: string;
	total: string;
	done: string;
	failed: string;
	state: string;
}): void {
	for (const handler of handlers) {
		handler({ id: 1, ...event });
	}
	flushSync();
}

/** Pin the document's visibility for the poll's test. */
function setVisibility(value: 'visible' | 'hidden'): void {
	Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => value });
	document.dispatchEvent(new Event('visibilitychange'));
}

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	clearToasts();
	vi.clearAllMocks();
	// Boot the session store unauthenticated (the interceptor stays
	// inert; these routes answer 200s).
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
	await initI18n();
	const state = await import('./jobs-page-state.svelte');
	jobsState = state.jobsPageState;
	jobsState.url = new URL('http://localhost/admin/jobs');
	mountedPage = null;
	Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'visible' });
});

afterEach(() => {
	if (mountedPage !== null) {
		unmount(mountedPage);
		flushSync();
		mountedPage = null;
	}
	vi.useRealTimers();
	vi.unstubAllGlobals();
});

// ---- the catalog ----

test('the catalog renders every task row with its counts, pauses and latest runs', async () => {
	const summary = summaryBody({
		paused: { thumbnail: false, metadata: false, transcode: true, ml: false, places: false },
		runs: {
			rebuildDerivatives: runSummary('0192f900-0000-7000-8000-000000000011', {
				state: 'succeeded',
				total: 1240,
				done: 1240
			}),
			placesRerun: runSummary('0192f900-0000-7000-8000-000000000012', {
				state: 'failed',
				failed: 2,
				finishedAt: '2026-11-19T03:00:00Z'
			}),
			albumRescanAll: null,
			folderSync: null,
			modelDownload: runSummary('0192f900-0000-7000-8000-000000000013', { state: 'succeeded' })
		}
	});
	const fetchMock = stubFetch([], {
		summary,
		budget: budgetBody({
			pausedJobs: { thumbnail: false, metadata: false, transcode: true, ml: false, places: false }
		})
	});
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	// The queue counts, in the catalog's own words.
	expect(taskRow('thumbnail').textContent).toContain('Running 2 · Waiting 3 · Failed 0');
	// A quiet kind says idle; a paused kind leads with the posture.
	expect(taskRow('metadata').textContent).toContain('Idle');
	expect(taskRow('transcode').textContent).toContain('Paused');
	expect(taskRow('transcode').textContent).toContain('Waiting 4');
	// The pause switch follows the budget document.
	expect(
		taskRow('transcode')
			.querySelector<HTMLButtonElement>('[data-testid="task-pause-transcode"]')
			?.textContent?.trim()
	).toBe('Resume');
	expect(
		taskRow('thumbnail')
			.querySelector<HTMLButtonElement>('[data-testid="task-pause-thumbnail"]')
			?.textContent?.trim()
	).toBe('Pause');

	// The machine learning group: the shared switch plus the four
	// per-kind subitems and the models link.
	const ml = taskRow('ml');
	expect(ml.textContent).toContain('Machine learning');
	expect(ml.textContent).toContain('Face detection · Running 0 · Waiting 3');
	expect(ml.textContent).toContain('Face recognition · Running 0 · Waiting 2');
	expect(ml.textContent).toContain('Smart search');
	expect(ml.textContent).toContain('OCR');
	expect(ml.querySelector('a')?.getAttribute('href')).toBe('/admin/models');

	// The one-off rows: the latest line, the failed line's link to the
	// history, and the existing surfaces' deep links.
	expect(taskRow('places').textContent).toContain('Latest: failed');
	expect(taskRow('places').querySelector('a')?.getAttribute('href')).toBe('/admin/settings?dialog=rebuild-places');
	expect(taskRow('rebuild').textContent).toContain('Latest: Succeeded');
	expect(taskRow('rebuild').querySelector('a')?.getAttribute('href')).toBe(
		'/admin/settings?dialog=rebuild-derivatives'
	);
	expect(taskRow('rescan').textContent).toContain('Latest: none');
	expect(taskRow('rescan').querySelector<HTMLButtonElement>('[data-testid="task-rescan-all"]')).not.toBeNull();
	expect(taskRow('foldersync').textContent).toContain('Live watch · every 60s');
	// The model slots' aggregate: three active, one downloading, one
	// unfilled.
	expect(taskRow('models').textContent).toContain('3 of 5 slots ready');
	expect(taskRow('models').textContent).toContain('Downloading');
	expect(taskRow('models').querySelector('a')?.getAttribute('href')).toBe('/admin/models');

	// The catalog asks for no estimate and copies no dialog: the only
	// one-off requests are its own read, the listing, the models and
	// the budget.
	const urls = fetchMock.mock.calls.map((call) => String(call[0]));
	expect(urls).not.toContain('/api/v1/admin/derivatives/summary');
	expect(urls).not.toContain('/api/v1/admin/settings/image');
});

test('a failed latest run links to the history tab', async () => {
	const summary = summaryBody({
		runs: {
			rebuildDerivatives: runSummary('0192f900-0000-7000-8000-000000000021', { state: 'failed' }),
			placesRerun: null,
			albumRescanAll: runSummary('0192f900-0000-7000-8000-000000000022', { state: 'failed', total: 12 }),
			folderSync: null,
			modelDownload: null
		}
	});
	vi.stubGlobal('fetch', stubFetch([], { summary }));
	mountPage();
	await settle();

	expect(taskRow('rebuild').textContent).toContain('Latest: failed');
	expect(taskRow('rebuild').querySelectorAll('a')[1]?.getAttribute('href')).toBe('?tab=runs');
	expect(taskRow('rescan').textContent).toContain('Latest: failed');
	expect(taskRow('rescan').querySelectorAll('a')[0]?.getAttribute('href') ?? '').toBe('?tab=runs');
	// A null latest line is the honest "none", never a failed link.
	expect(taskRow('places').textContent).toContain('Latest: none');
});

test('the pause switches live in their task rows and save the whole budget', async () => {
	const fetchMock = stubFetch([], { budget: budgetBody() });
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	// The card no longer carries the five switches; the rows do.
	expect(document.body.querySelector('[data-testid="run-pause-thumbnail"]')).toBeNull();
	for (const task of ['thumbnail', 'metadata', 'transcode', 'ml', 'places']) {
		expect(taskRow(task).querySelector('[data-testid^="task-pause-"]')).not.toBeNull();
	}

	taskRow('places').querySelector<HTMLButtonElement>('[data-testid="task-pause-places"]')?.click();
	await settle();

	const put = fetchMock.mock.calls.find(
		(call) => String(call[0]) === '/api/v1/admin/settings/run-budget' && call[1]?.method === 'PUT'
	);
	expect(put).toBeDefined();
	const saved = JSON.parse(String(put![1].body));
	expect(saved.pausedJobs.places).toBe(true);
	expect(saved.concurrencyPreset).toBe('standard');
	expect(toasts().map((entry) => entry.text)).toContain('Place resolution jobs paused.');
	// The switch itself now reads the resume posture.
	expect(
		taskRow('places').querySelector<HTMLButtonElement>('[data-testid="task-pause-places"]')?.textContent?.trim()
	).toBe('Resume');
});

test('rescan all posts the sweep endpoint and toasts the refreshed album count', async () => {
	const posted: string[] = [];
	let summaryCall = 0;
	const fetchMock = stubFetch([], {
		summary: () => {
			summaryCall += 1;
			// The start's refresh sees the fresh sweep's total.
			return summaryBody({
				runs: {
					rebuildDerivatives: null,
					placesRerun: null,
					albumRescanAll:
						summaryCall === 1
							? null
							: runSummary('0192f900-0000-7000-8000-000000000031', {
									state: 'queued',
									total: 12,
									done: 0,
									startedAt: null,
									finishedAt: null
								}),
					folderSync: null,
					modelDownload: null
				}
			});
		},
		extra: (input, init) => {
			if (String(input) === '/api/v1/admin/albums/rescan' && init?.method === 'POST') {
				posted.push(String(input));
				return jsonResponse(202, { jobRunId: '0192f900-0000-7000-8000-000000000031' });
			}
			return undefined;
		}
	});
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	taskRow('rescan').querySelector<HTMLButtonElement>('[data-testid="task-rescan-all"]')?.click();
	await settle();

	expect(posted).toEqual(['/api/v1/admin/albums/rescan']);
	expect(toasts().map((entry) => entry.text)).toContain('Rescanning all 12 albums.');
	// The refreshed summary now carries the sweep, and its cancel
	// shows up in the row.
	expect(taskRow('rescan').textContent).toContain('Latest: all 12 albums');
	expect(taskRow('rescan').querySelector<HTMLButtonElement>('[data-testid="task-cancel-rescan"]')).not.toBeNull();
});

test('a refused rescan-all start lands on the catalog error line', async () => {
	const fetchMock = stubFetch([], {
		extra: (input, init) => {
			if (String(input) === '/api/v1/admin/albums/rescan' && init?.method === 'POST') {
				return jsonResponse(500, { code: 'INTERNAL', message: 'the album rescan could not be started' });
			}
			return undefined;
		}
	});
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	taskRow('rescan').querySelector<HTMLButtonElement>('[data-testid="task-rescan-all"]')?.click();
	await settle();

	expect(document.body.querySelector('[data-testid="catalog-error"]')?.textContent).toContain('Something went wrong');
	expect(toasts()).toHaveLength(0);
});

test('a running sweep carries the cancel in its catalog row', async () => {
	const cancelled: string[] = [];
	const fetchMock = stubFetch([], {
		summary: summaryBody({
			runs: {
				rebuildDerivatives: null,
				placesRerun: null,
				albumRescanAll: runSummary('0192f900-0000-7000-8000-000000000041', {
					state: 'running',
					total: 12,
					done: 4,
					finishedAt: null
				}),
				folderSync: null,
				modelDownload: null
			}
		}),
		extra: (input, init) => {
			const url = String(input);
			if (url.startsWith('/api/v1/admin/jobs/') && url.endsWith('/cancel') && init?.method === 'POST') {
				cancelled.push(url);
				return new Response(null, { status: 204 });
			}
			return undefined;
		}
	});
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	taskRow('rescan').querySelector<HTMLButtonElement>('[data-testid="task-cancel-rescan"]')?.click();
	await settle();

	expect(cancelled).toEqual(['/api/v1/admin/jobs/0192f900-0000-7000-8000-000000000041/cancel']);
	// The honest account: the sweep's done counts ALBUMS.
	expect(toasts().map((entry) => entry.text)).toContain('Cancelled. The 4 finished items are kept.');
});

// ---- the tabs ----

test('the tabs: tasks is the default, ?tab=runs opens the history, the switch carries the address', async () => {
	vi.stubGlobal('fetch', stubFetch([embedRun]));
	mountPage();
	await settle();

	// The default face is the catalog.
	expect(document.body.querySelector<HTMLButtonElement>('[data-testid="jobs-tab-tasks"]')?.ariaPressed).toBe('true');
	expect(document.body.querySelector<HTMLButtonElement>('[data-testid="jobs-tab-runs"]')?.ariaPressed).toBe('false');
	expect(document.body.querySelector('[data-jobs-panel="tasks"]')?.classList.contains('hidden')).toBe(false);
	expect(document.body.querySelector('[data-jobs-panel="runs"]')?.classList.contains('hidden')).toBe(true);

	document.body.querySelector<HTMLButtonElement>('[data-testid="jobs-tab-runs"]')?.click();
	await settle();

	// The address carries the face; the history comes forward.
	const { jobsPageState } = await import('./jobs-page-state.svelte');
	expect(jobsPageState.url.searchParams.get('tab')).toBe('runs');
	expect(document.body.querySelector('[data-jobs-panel="runs"]')?.classList.contains('hidden')).toBe(false);
	expect(document.body.querySelector('[data-jobs-panel="tasks"]')?.classList.contains('hidden')).toBe(true);
});

test('a ?tab=runs mount opens the history face directly', async () => {
	vi.stubGlobal('fetch', stubFetch([embedRun]));
	mountPage('runs');
	await settle();

	expect(document.body.querySelector<HTMLButtonElement>('[data-testid="jobs-tab-runs"]')?.ariaPressed).toBe('true');
	expect(document.body.querySelector('[data-jobs-panel="runs"]')?.classList.contains('hidden')).toBe(false);
	expect(runCard(embedRun.id).textContent).toContain('Embedding rerun');
});

// ---- the summary's convergence ----

test('the summary poll runs only while the page is visible', async () => {
	vi.useFakeTimers();
	const fetchMock = stubFetch([embedRun]);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await vi.advanceTimersByTimeAsync(0);
	await tick();
	expect(summaryCalls(fetchMock)).toBe(1);

	// Visible: one read per 5 seconds.
	await vi.advanceTimersByTimeAsync(5000);
	await tick();
	expect(summaryCalls(fetchMock)).toBe(2);

	// Hidden: the timer stops -- no authenticated request rides a
	// background tab.
	setVisibility('hidden');
	await vi.advanceTimersByTimeAsync(15_000);
	await tick();
	expect(summaryCalls(fetchMock)).toBe(2);

	// Visible again: one refresh at once, then the cadence resumes.
	setVisibility('visible');
	await vi.advanceTimersByTimeAsync(0);
	await tick();
	expect(summaryCalls(fetchMock)).toBe(3);
	await vi.advanceTimersByTimeAsync(5000);
	await tick();
	expect(summaryCalls(fetchMock)).toBe(4);
});

test('a job.finished event refetches the catalog', async () => {
	const fetchMock = stubFetch([]);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();
	expect(summaryCalls(fetchMock)).toBe(1);

	deliverJobEvent({
		name: 'job.finished',
		jobRunId: '0192f900-0000-7000-8000-000000000051',
		kind: 'album_rescan_all',
		total: '12',
		done: '12',
		failed: '0',
		state: 'succeeded'
	});
	await settle();
	expect(summaryCalls(fetchMock)).toBe(2);

	// A progress event folds into the matching catalog slot without a
	// read.
	deliverJobEvent({
		name: 'job.progress',
		jobRunId: '0192f900-0000-7000-8000-000000000003',
		kind: 'model_download',
		total: '400371536',
		done: '209715200',
		failed: '0',
		state: 'running'
	});
	expect(summaryCalls(fetchMock)).toBe(2);
});

// ---- the history (ported) ----

test('the listing renders each run with kind, state, per-kind units and times', async () => {
	vi.stubGlobal('fetch', stubFetch([downloadRun, embedRun, ocrRun, queuedRun, rescanRun, placesRun]));
	mountedPage = mountPage('runs');
	await settle();

	const text = document.body.textContent ?? '';
	expect(text).toContain('Model download');
	expect(text).toContain('Embedding rerun');
	expect(text).toContain('Text recognition rerun');
	expect(text).toContain('Face detection rerun');
	expect(text).toContain('Album rescan');
	expect(text).toContain('Places rerun');
	expect(text).toContain('Running');
	expect(text).toContain('Succeeded');
	expect(text).toContain('Failed');
	expect(text).toContain('Queued');
	expect(runCard(downloadRun.id).textContent).toContain('100.0 MB of 381.8 MB');
	expect(runCard(embedRun.id).textContent).toContain('3 of 3 assets');
	expect(runCard(ocrRun.id).textContent).toContain('4 of 5 assets');
	expect(runCard(ocrRun.id).textContent).toContain('1 failed');
	// The rescan's own line: a clean scan reads like a log.
	expect(runCard(rescanRun.id).textContent).toContain('2 photos unchanged');
	expect(runCard(embedRun.id).textContent).toContain('Started Nov 3, 2026');
	expect(runCard(embedRun.id).textContent).toContain('Finished Nov 3, 2026');
	expect(runCard(embedRun.id).textContent).toContain('took 2:30');
	expect(runCard(queuedRun.id).textContent).toContain('Waiting for the run ahead to finish');
	expect(runCard(queuedRun.id).querySelector('[role="progressbar"]')).toBeNull();
	expect(runCard(downloadRun.id).querySelector('[role="progressbar"]')).not.toBeNull();
	expect(runCard(embedRun.id).querySelector('[role="progressbar"]')).not.toBeNull();
	expect(runCard(downloadRun.id).querySelector('.jobs__badge--running')).not.toBeNull();
	expect(runCard(ocrRun.id).querySelector('.jobs__badge--failed')).not.toBeNull();

	unmount(page());
	flushSync();
	mountedPage = null;
});

test('an empty history states it honestly', async () => {
	vi.stubGlobal('fetch', stubFetch([]));
	mountedPage = mountPage('runs');
	await settle();

	expect(document.body.textContent).toContain('No job runs yet');

	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('SSE progress events converge a running row without a refetch', async () => {
	const fetchMock = stubFetch([downloadRun]);
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await settle();

	deliverJobEvent({
		name: 'job.progress',
		jobRunId: downloadRun.id,
		kind: 'model_download',
		total: '400371536',
		done: '209715200',
		failed: '0',
		state: 'running'
	});
	expect(runCard(downloadRun.id).textContent).toContain('200.0 MB of 381.8 MB');
	// No refetch for a progress event: the row converged in place.
	expect(listingCalls(fetchMock)).toBe(1);

	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('the job.finished event converges the row through a refetch', async () => {
	let listed = 0;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		const url = String(input);
		if (url === '/api/v1/admin/jobs') {
			listed += 1;
			if (listed === 1) {
				return jsonResponse(200, { jobs: [downloadRun] });
			}
			return jsonResponse(200, {
				jobs: [
					jobRun(downloadRun.id, {
						state: 'succeeded',
						done: 400371536,
						finishedAt: '2026-11-03T09:14:30Z'
					})
				]
			});
		}
		if (url === '/api/v1/admin/jobs/summary') {
			return jsonResponse(200, summaryBody());
		}
		if (url === '/api/v1/admin/settings/run-budget') {
			return jsonResponse(200, budgetBody());
		}
		if (url === '/api/v1/admin/models') {
			return jsonResponse(200, { models: [], download: null, slots: [], official: [] });
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await settle();

	deliverJobEvent({
		name: 'job.finished',
		jobRunId: downloadRun.id,
		kind: 'model_download',
		total: '400371536',
		done: '400371536',
		failed: '0',
		state: 'succeeded'
	});
	await settle();
	expect(listed).toBe(2);
	expect(runCard(downloadRun.id).textContent).toContain('Succeeded');
	expect(runCard(downloadRun.id).textContent).toContain('Finished Nov 3, 2026');

	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('a running row polls the listing when the stream stays silent', async () => {
	vi.useFakeTimers();
	let listed = 0;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		const url = String(input);
		if (url === '/api/v1/admin/jobs') {
			listed += 1;
			const done = listed === 1 ? 104857600 : 209715200;
			return jsonResponse(200, { jobs: [jobRun(downloadRun.id, { done })] });
		}
		if (url === '/api/v1/admin/jobs/summary') {
			return jsonResponse(200, summaryBody());
		}
		if (url === '/api/v1/admin/settings/run-budget') {
			return jsonResponse(200, budgetBody());
		}
		if (url === '/api/v1/admin/models') {
			return jsonResponse(200, { models: [], download: null, slots: [], official: [] });
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await vi.advanceTimersByTimeAsync(0);
	await tick();

	expect(runCard(downloadRun.id).textContent).toContain('100.0 MB of 381.8 MB');

	// No SSE block ever arrives (the stream is down); the 5-second
	// fallback poll converges the row.
	await vi.advanceTimersByTimeAsync(5000);
	await tick();
	flushSync();
	expect(listed).toBe(2);
	expect(runCard(downloadRun.id).textContent).toContain('200.0 MB of 381.8 MB');

	vi.useRealTimers();
	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('history rows are static: no polling without an active run', async () => {
	vi.useFakeTimers();
	const fetchMock = stubFetch([embedRun, ocrRun]);
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await vi.advanceTimersByTimeAsync(0);
	await tick();

	await vi.advanceTimersByTimeAsync(10_000);
	expect(listingCalls(fetchMock)).toBe(1);

	vi.useRealTimers();
	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('the unmount unsubscribes and stops the polling', async () => {
	vi.useFakeTimers();
	const fetchMock = stubFetch([downloadRun]);
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await vi.advanceTimersByTimeAsync(0);
	await tick();
	expect(handlers).toHaveLength(1);

	unmount(mountedPage);
	flushSync();
	mountedPage = null;
	expect(handlers).toHaveLength(0);
	await vi.advanceTimersByTimeAsync(10_000);
	expect(listingCalls(fetchMock)).toBe(1);

	vi.useRealTimers();
});

test('a failed run states its reason and retries through its own trigger', async () => {
	const posted: string[] = [];
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'POST') {
			posted.push(url);
			return jsonResponse(202, { jobRunId: '0192f900-0000-7000-8000-000000000061' });
		}
		if (url === '/api/v1/admin/jobs') {
			return jsonResponse(200, { jobs: [ocrRun] });
		}
		if (url === '/api/v1/admin/models') {
			return jsonResponse(200, {
				models: [model('ppocr-v4-mobile-ch')],
				download: null,
				slots: [],
				official: []
			});
		}
		if (url === '/api/v1/admin/jobs/summary') {
			return jsonResponse(200, summaryBody());
		}
		if (url === '/api/v1/admin/settings/run-budget') {
			return jsonResponse(200, budgetBody());
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await settle();

	const reason = runCard(ocrRun.id).querySelector('[data-testid="job-reason"]');
	expect(reason?.textContent).toContain('1 item could not be processed');
	expect(reason?.textContent).toContain('server log');

	buttonByText('Retry').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(posted).toEqual(['/api/v1/admin/models/ppocr-v4-mobile-ch/rerun']);
	expect(listingCalls(fetchMock)).toBe(2);

	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('a failed run whose model is missing names the download instead of retrying', async () => {
	const fetchMock = stubFetch([ocrRun], { models: [model('ppocr-v4-mobile-ch', { isActive: false })] });
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await settle();

	const reason = runCard(ocrRun.id).querySelector('[data-testid="job-reason"]');
	expect(reason?.textContent).toContain('Text recognition has no downloaded model');
	expect(reason?.querySelector('a')?.getAttribute('href')).toBe('/admin/models');
	expect(runCard(ocrRun.id).querySelector('[data-testid="job-retry"]')).toBeNull();

	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('a failed download and a failed places run retry through their own endpoints', async () => {
	const failedDownload = jobRun(downloadRun.id, {
		state: 'failed',
		failed: 2,
		finishedAt: '2026-11-03T09:20:00Z'
	});
	const failedPlaces = jobRun(placesRun.id, {
		kind: 'places_rerun',
		state: 'failed',
		failed: 0,
		done: 3,
		finishedAt: '2026-11-03T13:10:00Z'
	});
	const posted: string[] = [];
	const fetchMock = stubFetch([failedDownload, failedPlaces], {
		extra: (input, init) => {
			const url = String(input);
			if (init?.method === 'POST') {
				posted.push(url);
				return jsonResponse(202, { jobRunId: '0192f900-0000-7000-8000-000000000062' });
			}
			return undefined;
		}
	});
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await settle();

	expect(runCard(failedDownload.id).querySelector('[data-testid="job-reason"]')?.textContent).toContain(
		'2 models could not be downloaded'
	);
	expect(runCard(failedPlaces.id).querySelector('[data-testid="job-reason"]')?.textContent).toContain(
		'stopped before it got through its work'
	);

	const downloadRetry = runCard(failedDownload.id).querySelector<HTMLButtonElement>('[data-testid="job-retry"]');
	downloadRetry?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	const placesRetry = runCard(failedPlaces.id).querySelector<HTMLButtonElement>('[data-testid="job-retry"]');
	placesRetry?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(posted).toEqual(['/api/v1/admin/models/download', '/api/v1/admin/places/reresolve']);

	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('a failed rescan offers no retry: its album id is not in the listing', async () => {
	const failedRescan = jobRun(rescanRun.id, {
		kind: 'album_rescan',
		state: 'failed',
		failed: 0,
		finishedAt: '2026-11-03T12:10:00Z'
	});
	vi.stubGlobal('fetch', stubFetch([failedRescan]));
	mountedPage = mountPage('runs');
	await settle();

	expect(runCard(failedRescan.id).querySelector('[data-testid="job-reason"]')).not.toBeNull();
	expect(runCard(failedRescan.id).querySelector('[data-testid="job-retry"]')).toBeNull();

	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('the all-albums sweep row reads its own kind and album units', async () => {
	const sweep = jobRun('0192f900-0000-7000-8000-000000000063', {
		kind: 'album_rescan_all',
		state: 'succeeded',
		total: 12,
		done: 12,
		startedAt: '2026-11-03T12:00:00Z',
		finishedAt: '2026-11-03T12:05:00Z'
	});
	vi.stubGlobal('fetch', stubFetch([sweep]));
	mountedPage = mountPage('runs');
	await settle();

	expect(runCard(sweep.id).textContent).toContain('Album rescan (all)');
	expect(runCard(sweep.id).textContent).toContain('12 of 12 albums');

	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('a rejected retry reports in place, above the row it belongs to', async () => {
	const fetchMock = stubFetch([ocrRun], {
		models: [model('ppocr-v4-mobile-ch')],
		extra: (input, init) => {
			if (init?.method === 'POST') {
				return jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'administrator role required' });
			}
			return undefined;
		}
	});
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await settle();

	buttonByText('Retry').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(runCard(ocrRun.id).textContent).toContain("You don't have permission to do this.");

	unmount(mountedPage);
	flushSync();
	mountedPage = null;
});

test('the run-settings card renders the dials and saves the whole document on change', async () => {
	const fetchMock = stubFetch([], { budget: budgetBody() });
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	const card = document.body.querySelector<HTMLElement>('[data-testid="run-settings"]');
	expect(card).not.toBeNull();
	expect(document.body.querySelector<HTMLButtonElement>('[data-testid="run-dial-standard"]')?.ariaPressed).toBe(
		'true'
	);
	expect(document.body.querySelector<HTMLButtonElement>('[data-testid="run-dial-light"]')?.ariaPressed).toBe('false');

	document.body.querySelector<HTMLButtonElement>('[data-testid="run-dial-light"]')?.click();
	await settle();
	const put = fetchMock.mock.calls.find(
		(call) => String(call[0]) === '/api/v1/admin/settings/run-budget' && call[1]?.method === 'PUT'
	);
	expect(put).toBeDefined();
	const saved = JSON.parse(String(put![1].body));
	expect(saved.concurrencyPreset).toBe('light');
	// The whole document rides: the pauses and the window too.
	expect(saved.pausedJobs).toEqual(budgetBody().pausedJobs);
	expect(saved.nightlyStart).toBe('02:00');

	unmount(page());
	mountedPage = null;
});

test("the night window's time inputs go dead while the switch is off", async () => {
	const fetchMock = stubFetch([], { budget: budgetBody({ nightlyEnabled: false }) });
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	expect(document.body.querySelector<HTMLInputElement>('[data-testid="run-nightly-start"]')?.disabled).toBe(true);
	expect(document.body.querySelector<HTMLInputElement>('[data-testid="run-nightly-end"]')?.disabled).toBe(true);

	document.body.querySelector<HTMLInputElement>('[data-testid="run-nightly-toggle"]')?.click();
	await settle();
	expect(document.body.querySelector<HTMLInputElement>('[data-testid="run-nightly-start"]')?.disabled).toBe(false);
	expect(document.body.querySelector<HTMLInputElement>('[data-testid="run-nightly-end"]')?.disabled).toBe(false);

	unmount(page());
	mountedPage = null;
});

test('a running row carries the cancel, and cancelling POSTs the run id, toasts and refetches', async () => {
	const rebuildRun = jobRun('0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f11', {
		kind: 'rebuild_derivatives',
		state: 'running',
		subject: null,
		total: 12,
		done: 4,
		startedAt: '2026-11-03T14:00:00Z'
	});
	let cancels = 0;
	let cancelUrl = '';
	const fetchMock = stubFetch([rebuildRun], {
		extra: (input, init) => {
			const url = String(input);
			if (url === `/api/v1/admin/jobs/${rebuildRun.id}/cancel` && init?.method === 'POST') {
				cancels += 1;
				cancelUrl = url;
				return new Response(null, { status: 204 });
			}
			return undefined;
		}
	});
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await settle();

	const cancel = document.body.querySelector<HTMLButtonElement>('[data-testid="job-cancel"]');
	expect(cancel).not.toBeNull();
	cancel?.click();
	await settle();
	expect(cancels).toBe(1);
	expect(cancelUrl).toBe(`/api/v1/admin/jobs/${rebuildRun.id}/cancel`);
	expect(toasts().map((entry) => entry.text)).toContain('Cancelled. The 4 finished items are kept.');

	unmount(page());
	mountedPage = null;
});

test('a finished run answers with no cancel on its row', async () => {
	const closedRun = jobRun('0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f12', {
		kind: 'clip_embed_rerun',
		state: 'succeeded',
		subject: null,
		total: 3,
		done: 3,
		startedAt: '2026-11-03T14:00:00Z',
		finishedAt: '2026-11-03T14:03:00Z'
	});
	const fetchMock = stubFetch([closedRun]);
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await settle();

	expect(runCard(closedRun.id).querySelector('[data-testid="job-cancel"]')).toBeNull();

	unmount(page());
	mountedPage = null;
});

test('a queued row carries the cancel too, and a 409 lands under its own row', async () => {
	const queued = jobRun('0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f13', {
		kind: 'face_detect_rerun',
		state: 'queued',
		subject: null,
		total: 10,
		done: 0,
		startedAt: null
	});
	const fetchMock = stubFetch([queued], {
		extra: (input, init) => {
			const url = String(input);
			if (url === `/api/v1/admin/jobs/${queued.id}/cancel` && init?.method === 'POST') {
				return jsonResponse(409, { code: 'JOB_NOT_CANCELLABLE', message: 'this run has already finished' });
			}
			return undefined;
		}
	});
	vi.stubGlobal('fetch', fetchMock);
	mountedPage = mountPage('runs');
	await settle();

	const cancel = document.body.querySelector<HTMLButtonElement>('[data-testid="job-cancel"]');
	expect(cancel).not.toBeNull();
	cancel?.click();
	await settle();
	const inline = document.body.querySelector<HTMLElement>('[data-testid="job-cancel-error"]');
	expect(inline).not.toBeNull();
	expect(inline?.textContent).toContain('This run has already finished and cannot be cancelled.');
	expect(document.body.querySelector('.toast')).toBeNull();

	unmount(page());
	mountedPage = null;
});
