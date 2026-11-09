import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';

// The admin task panel: the job runs listing renders each run's kind,
// state, progress (the units per kind -- pinned bytes for the model
// download, asset counts for the reruns) and timestamps; the SSE job
// events converge the active rows live, with the 5-second polling
// fallback when the stream stays silent; history rows are static and
// the unmount unsubscribes. The events module is mocked with a
// captured handler registry, the same pattern as the trash page test.

const { handlers } = vi.hoisted(() => ({
	handlers: [] as Array<(event: unknown) => void>
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

/** One job run row (the JobRunResponse shape). */
function jobRun(id: string, overrides: Record<string, unknown> = {}) {
	return {
		id,
		kind: 'model_download',
		state: 'running',
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
// The E4 seam's pin: the contract's JobRunKind enum carries the
// rescan and the places kinds, so their rows must validate and
// render with their own labels -- a kind the validator rejects
// would collapse the whole listing into the generic failure.
const rescanRun = jobRun('0192f8e1-4f5a-6b7c-8d9e-0a1b2c3d4e5f', {
	kind: 'album_rescan',
	state: 'succeeded',
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

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function stubFetch(jobs: unknown[]): ReturnType<typeof vi.fn> {
	return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		if (String(input) === '/api/v1/admin/jobs') {
			return jsonResponse(200, { jobs });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${String(input)}`);
	});
}

async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise<void>((resolve) => {
		requestAnimationFrame(() => resolve());
	}).catch(() => undefined);
	await tick();
}

function mountPage(): ReturnType<typeof mount> {
	const page = mount(Page, { target: document.body, props: {} });
	flushSync();
	return page;
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

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	vi.clearAllMocks();
	// Boot the session store unauthenticated (the interceptor stays
	// inert; these routes answer 200s).
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })));
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
	await initI18n();
});

afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
});

test('the listing renders each run with kind, state, per-kind units and times', async () => {
	vi.stubGlobal('fetch', stubFetch([downloadRun, embedRun, ocrRun, queuedRun, rescanRun, placesRun]));
	const page = mountPage();
	await settle();

	// The kind labels of the run kinds, and the states.
	const text = document.body.textContent ?? '';
	expect(text).toContain('Model download');
	expect(text).toContain('Embedding rerun');
	expect(text).toContain('Text recognition rerun');
	expect(text).toContain('Face detection rerun');
	// The E4 seam's kinds render their own labels, not raw kind
	// strings (and the listing itself stays ready -- a rejected
	// kind would have collapsed it into the generic failure).
	expect(text).toContain('Album rescan');
	expect(text).toContain('Places rerun');
	expect(text).toContain('Running');
	expect(text).toContain('Succeeded');
	expect(text).toContain('Failed');
	expect(text).toContain('Queued');
	// The progress units per kind: bytes formatted MB for the
	// download, asset counts for the reruns and the rescan.
	expect(runCard(downloadRun.id).textContent).toContain('100.0 MB of 381.8 MB');
	expect(runCard(embedRun.id).textContent).toContain('3 of 3 assets');
	expect(runCard(ocrRun.id).textContent).toContain('4 of 5 assets');
	expect(runCard(ocrRun.id).textContent).toContain('1 failed');
	expect(runCard(rescanRun.id).textContent).toContain('2 of 2 assets');
	// The start/finish timestamps, locale-formatted; a queued run has
	// not started.
	expect(runCard(embedRun.id).textContent).toContain('Started Nov 3, 2026');
	expect(runCard(embedRun.id).textContent).toContain('Finished Nov 3, 2026');
	expect(runCard(queuedRun.id).textContent).toContain('Not started');
	// The state badge carries its color class.
	expect(
		runCard(downloadRun.id).querySelector('.jobs__badge--running')
	).not.toBeNull();
	expect(
		runCard(ocrRun.id).querySelector('.jobs__badge--failed')
	).not.toBeNull();

	unmount(page);
	flushSync();
});

test('an empty panel states it honestly', async () => {
	vi.stubGlobal('fetch', stubFetch([]));
	const page = mountPage();
	await settle();

	expect(document.body.textContent).toContain('No job runs yet');

	unmount(page);
	flushSync();
});

test('SSE progress events converge a running row without a refetch', async () => {
	const fetchMock = stubFetch([downloadRun]);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
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
	expect(fetchMock).toHaveBeenCalledTimes(1);

	unmount(page);
	flushSync();
});

test('the job.finished event converges the row through a refetch', async () => {
	let listed = 0;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		if (String(input) === '/api/v1/admin/jobs') {
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
		throw new Error(`unexpected fetch ${String(input)}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
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

	unmount(page);
	flushSync();
});

test('a running row polls the listing when the stream stays silent', async () => {
	vi.useFakeTimers();
	let listed = 0;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		if (String(input) === '/api/v1/admin/jobs') {
			listed += 1;
			const done = listed === 1 ? 104857600 : 209715200;
			return jsonResponse(200, { jobs: [jobRun(downloadRun.id, { done })] });
		}
		throw new Error(`unexpected fetch ${String(input)}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
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
	unmount(page);
	flushSync();
});

test('history rows are static: no polling without an active run', async () => {
	vi.useFakeTimers();
	const fetchMock = stubFetch([embedRun, ocrRun]);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await vi.advanceTimersByTimeAsync(0);
	await tick();

	await vi.advanceTimersByTimeAsync(10_000);
	expect(fetchMock).toHaveBeenCalledTimes(1);

	vi.useRealTimers();
	unmount(page);
	flushSync();
});

test('the unmount unsubscribes and stops the polling', async () => {
	vi.useFakeTimers();
	const fetchMock = stubFetch([downloadRun]);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await vi.advanceTimersByTimeAsync(0);
	await tick();
	expect(handlers).toHaveLength(1);

	unmount(page);
	flushSync();
	// The subscription is gone (no listener would converge anything).
	expect(handlers).toHaveLength(0);
	await vi.advanceTimersByTimeAsync(10_000);
	expect(fetchMock).toHaveBeenCalledTimes(1);

	vi.useRealTimers();
});

test('a non-admin caller sees the mapped auth_forbidden message', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () =>
			jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'administrator role required' })
		)
	);
	const page = mountPage();
	await settle();

	// The generic AUTH_FORBIDDEN message (the M3c albums branch made
	// the code cover the album share face too, so the text stopped
	// naming the admin surface).
	expect(document.body.querySelector('[role="alert"]')?.textContent).toContain(
		"You don't have permission to do this."
	);
	expect(buttonByText('Try again')).toBeTruthy();

	unmount(page);
	flushSync();
});
