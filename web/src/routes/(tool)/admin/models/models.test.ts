import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';

// The admin models page: the slot-grouped listing renders every
// model's facts, the empty state's download CTA starts the run and
// follows it through the SSE job events (the all-strings payload) to
// the converged listing, the activation switch confirms first with
// the same-dimension and cross-dimension copies (the destructive arm
// on the danger variant), the rerun hands off to the task panel, and
// every failure maps onto the shared i18n table. The events module is
// mocked with a captured handler registry, the same pattern as the
// trash page test.

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

// The listing's download run row (the JobRunResponse shape).
function downloadRun(overrides: Record<string, unknown> = {}) {
	return {
		id: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10',
		kind: 'model_download',
		state: 'running',
		total: 400371536,
		done: 0,
		failed: 0,
		startedAt: '2026-11-03T09:12:00Z',
		finishedAt: null,
		...overrides
	};
}

/** One model row (the ModelResponse shape). */
function model(id: string, overrides: Record<string, unknown> = {}) {
	return {
		id,
		slot: 'clip_embed',
		version: 'base-patch16',
		source: 'downloaded',
		license: 'Apache-2.0',
		notice: null,
		dim: 512,
		inputSize: { width: 224, height: 224 },
		isActive: false,
		filesPresent: true,
		params: {},
		...overrides
	};
}

const clipA = model('clip-a', { isActive: true, params: { scoreThreshold: 0.6 } });
const clipB = model('clip-b', { version: 'alt-2024' });
const clipWide = model('clip-wide', { dim: 768 });
// The search's text tower: the clip_text slot the semantic arm embeds
// queries through (the manifest's fifth default model).
const textTower = model('cnclip-vit-b16-text', {
	slot: 'clip_text',
	inputSize: { width: 52, height: 1 },
	isActive: true,
	params: { contextLength: 52 }
});
const detector = model('detect-fx', {
	id: 'detect-fx',
	slot: 'face_detect',
	version: '2023mar',
	license: 'MIT',
	dim: null,
	inputSize: { width: 320, height: 320 },
	isActive: true
});

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, () => Response>;

function modelsRoutes(models: unknown[], overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/admin/models': () => jsonResponse(200, { models, download: null }),
		...overrides
	};
}

function stubFetch(route: FetchRoute): ReturnType<typeof vi.fn> {
	return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		const answer = route[url];
		if (answer === undefined) {
			throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
		}
		return answer();
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

/** A button of the open modal's panel (the confirm surface). */
function modalButton(text: string): HTMLButtonElement {
	const panel = document.body.querySelector<HTMLElement>('.modal__panel');
	const found = Array.from(panel?.querySelectorAll<HTMLButtonElement>('button') ?? []).find(
		(button) => button.textContent?.trim() === text
	);
	if (found === undefined) {
		throw new Error(`missing modal button ${text}`);
	}
	return found;
}

/** The model row (card) whose text names the model id. */
function modelCard(id: string): HTMLElement {
	const cards = Array.from(document.body.querySelectorAll<HTMLElement>('[data-model-id]'));
	const found = cards.find((card) => card.dataset.modelId === id);
	if (found === undefined) {
		throw new Error(`missing model card ${id}`);
	}
	return found;
}

/** The Activate button of one model's row. */
function activateButton(id: string): HTMLButtonElement {
	const button = modelCard(id).querySelector<HTMLButtonElement>('button');
	if (button === null) {
		throw new Error(`missing activate button of ${id}`);
	}
	return button;
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
	vi.unstubAllGlobals();
});

test('the listing renders slot-grouped cards with every model fact', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(modelsRoutes([detector, clipA, clipB, textTower]))
	);
	const page = mountPage();
	await settle();

	const text = document.body.textContent ?? '';
	// The slot sections, in the enum's declaration order (the
	// clip_text slot between the embedding slot and OCR).
	expect(text).toContain('Face detection');
	expect(text).toContain('Semantic embeddings');
	expect(text).toContain('Search text encoder');
	// The empty slots say so honestly.
	expect(text).toContain('No model registered for this slot.');
	// The detector's facts: id, version, license badge, source label,
	// input geometry -- and no dimension line (detectors do not embed).
	const detectorText = modelCard('detect-fx').textContent ?? '';
	expect(detectorText).toContain('detect-fx');
	expect(detectorText).toContain('2023mar');
	expect(detectorText).toContain('MIT');
	expect(detectorText).toContain('Downloaded');
	expect(detectorText).toContain('320×320');
	expect(detectorText).not.toContain('Dimension');
	// The active badge.
	expect(detectorText).toContain('Active');
	// The clip model's facts: dimension, params summary and the
	// provenance notice line.
	const clipText = modelCard('clip-a').textContent ?? '';
	expect(clipText).toContain('Dimension 512');
	expect(clipText).toContain('scoreThreshold 0.6');
	// The text tower's own card: the token geometry (52×1 -- the
	// context length, not pixels) and its params summary.
	const towerText = modelCard('cnclip-vit-b16-text').textContent ?? '';
	expect(towerText).toContain('52×1');
	expect(towerText).toContain('contextLength 52');
	expect(towerText).toContain('Active');
	// An inactive model offers the activation switch.
	expect(modelCard('clip-b').textContent).toContain('Activate');

	unmount(page);
	flushSync();
});

test('a model with missing files shows the files-missing indicator', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(modelsRoutes([{ ...clipB, filesPresent: false }]))
	);
	const page = mountPage();
	await settle();

	expect(modelCard('clip-b').textContent).toContain('Files missing');

	unmount(page);
	flushSync();
});

test('the empty state offers the download; SSE progress advances it and the finish refreshes the listing', async () => {
	let listed = 0;
	const posted: number[] = [];
	let runningState: Record<string, unknown> = {
		state: 'running',
		total: 400371536,
		done: 0,
		failed: 0
	};
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'POST' && url === '/api/v1/admin/models/download') {
			posted.push(202);
			return jsonResponse(202, { jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10' });
		}
		if (url === '/api/v1/admin/models') {
			listed += 1;
			// The second listing (after the POST) carries the running
			// download row; the third (after job.finished) the landed
			// models.
			if (listed === 1) {
				return jsonResponse(200, { models: [], download: null });
			}
			if (listed === 2) {
				return jsonResponse(200, { models: [], download: downloadRun(runningState) });
			}
			return jsonResponse(200, { models: [clipA], download: downloadRun({ state: 'succeeded', done: 400371536, finishedAt: '2026-11-03T09:14:30Z' }) });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The empty state names the honest degradation and offers the CTA.
	expect(document.body.textContent).toContain('No models yet');
	expect(document.body.textContent).toContain('stay off');
	buttonByText('Download the default models').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(posted).toEqual([202]);
	expect(listed).toBe(2);

	// The progress region: the run's totals, the bytes formatted MB.
	expect(document.body.textContent).toContain('0.0 MB of 381.8 MB');
	// The SSE progress events (all-strings payloads) advance it.
	deliverJobEvent({
		name: 'job.progress',
		jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10',
		kind: 'model_download',
		total: '400371536',
		done: '104857600',
		failed: '0',
		state: 'running'
	});
	expect(document.body.textContent).toContain('100.0 MB of 381.8 MB');

	// The finish converges the listing: the run landed, the model
	// registered and active.
	deliverJobEvent({
		name: 'job.finished',
		jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10',
		kind: 'model_download',
		total: '400371536',
		done: '400371536',
		failed: '0',
		state: 'succeeded'
	});
	await settle();
	expect(listed).toBe(3);
	expect(modelCard('clip-a')).toBeTruthy();
	expect(modelCard('clip-a').textContent).toContain('Active');

	unmount(page);
	flushSync();
});

test('a failed download start renders the mapped error; the retry re-posts (idempotent)', async () => {
	const posted: number[] = [];
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'POST' && url === '/api/v1/admin/models/download') {
			if (posted.length === 0) {
				posted.push(500);
				return jsonResponse(500, { code: 'MODEL_DOWNLOAD_FAILED', message: 'no queue' });
			}
			// The idempotent join: 200, nothing new started.
			posted.push(200);
			return jsonResponse(200, { jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10' });
		}
		if (url === '/api/v1/admin/models') {
			const listed = fetchMock.mock.calls.filter(
				(call) => String(call[0]) === '/api/v1/admin/models'
			).length;
			// The initial listing knows no run; the refetch after the
			// idempotent join carries the running one.
			if (listed === 1) {
				return jsonResponse(200, { models: [], download: null });
			}
			return jsonResponse(200, { models: [], download: downloadRun() });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Download the default models').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(posted).toEqual([500]);
	// The mapped model_download_failed message.
	expect(document.body.querySelector('[role="alert"]')?.textContent).toContain(
		'The model download could not be started'
	);

	// The retry re-posts and joins the run idempotently.
	buttonByText('Try again').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(posted).toEqual([500, 200]);
	expect(document.body.textContent).toContain('0.0 MB of 381.8 MB');

	unmount(page);
	flushSync();
});

test('a same-dimension switch confirms with the kept-products copy, patches and flips the badge', async () => {
	const patches: string[] = [];
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'PATCH' && url === '/api/v1/admin/models/clip-b') {
			patches.push(String(init.body));
			return jsonResponse(200, { activated: { id: 'clip-b', slot: 'clip_embed' } });
		}
		if (url === '/api/v1/admin/models') {
			const listed = fetchMock.mock.calls.filter(
				(call) => String(call[0]) === '/api/v1/admin/models'
			).length;
			// The refetch after the PATCH carries the flipped flags.
			const models =
				listed <= 1
					? [detector, clipA, clipB]
					: [detector, { ...clipA, isActive: false }, { ...clipB, isActive: true }];
			return jsonResponse(200, { models, download: null });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	activateButton('clip-b').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	// The confirm-first gate: the same-dimension copy states the
	// coexistence semantics -- kept products, rollback by switching
	// back, the manual rerun.
	const modalText = document.body.querySelector('.modal__panel')?.textContent ?? '';
	expect(modalText).toContain('products are kept');
	expect(modalText).toContain('switching back restores them');
	expect(modalText).toContain('needs a rerun');
	expect(patches).toEqual([]);

	modalButton('Activate').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(patches).toEqual(['{}']);
	// The refetched listing flips the badge.
	expect(modelCard('clip-b').textContent).toContain('Active');
	expect(modelCard('clip-a').textContent).not.toContain('Active');
	expect(document.body.querySelector('.modal__panel')).toBeNull();

	unmount(page);
	flushSync();
});

test('a cross-dimension switch warns on the danger variant and reports the wipe', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'PATCH' && url === '/api/v1/admin/models/clip-wide') {
			return jsonResponse(200, {
				activated: { id: 'clip-wide', slot: 'clip_embed' },
				dimensionChanged: { from: 512, to: 768 },
				wipedEmbeddings: 14002
			});
		}
		if (url === '/api/v1/admin/models') {
			const listed = fetchMock.mock.calls.filter(
				(call) => String(call[0]) === '/api/v1/admin/models'
			).length;
			const models =
				listed <= 1
					? [clipA, clipWide]
					: [{ ...clipA, isActive: false }, { ...clipWide, isActive: true }];
			return jsonResponse(200, { models, download: null });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	activateButton('clip-wide').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	// The destructive arm's copy and the danger variant: the column's
	// dimension is not knowable before the PATCH, so a clip model of a
	// different dimension than the active one's warns.
	const modalText = document.body.querySelector('.modal__panel')?.textContent ?? '';
	expect(modalText).toContain('every embedding is deleted');
	expect(modalText).toContain('re-typed');
	const confirm = modalButton('Activate');
	expect(confirm.className).toContain('btn--danger');

	modalButton('Activate').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	// The response's facts surface: the wipe happened, the rerun is
	// the rebuild.
	expect(document.body.textContent).toContain('14002 embeddings were removed');
	expect(modelCard('clip-wide').textContent).toContain('Active');

	unmount(page);
	flushSync();
});

test('an unknown model on the switch maps the not-found message inside the modal', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'PATCH' && url === '/api/v1/admin/models/clip-b') {
			return jsonResponse(404, { code: 'MODEL_NOT_FOUND', message: 'no model' });
		}
		if (url === '/api/v1/admin/models') {
			return jsonResponse(200, { models: [detector, clipA, clipB], download: null });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	activateButton('clip-b').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	modalButton('Activate').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The confirm dialog stays open with the mapped failure (the
	// trash page's confirm pattern); the row is untouched.
	expect(document.body.querySelector('.modal__panel')).not.toBeNull();
	expect(document.body.querySelector('.modal__panel [role="alert"]')?.textContent).toContain(
		'This model is not registered.'
	);
	expect(modelCard('clip-b').textContent).not.toContain('Active');

	unmount(page);
	flushSync();
});

test('a model with drifted files maps the invalid message inside the modal', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'PATCH' && url === '/api/v1/admin/models/clip-b') {
			return jsonResponse(400, { code: 'MODEL_INVALID', message: 'digest mismatch' });
		}
		if (url === '/api/v1/admin/models') {
			return jsonResponse(200, { models: [detector, clipA, clipB], download: null });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	activateButton('clip-b').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	modalButton('Activate').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(document.body.querySelector('.modal__panel [role="alert"]')?.textContent).toContain(
		'This model cannot be used for the requested action.'
	);

	unmount(page);
	flushSync();
});

test('the rerun confirms, posts and hands off to the task panel', async () => {
	const reruns: string[] = [];
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'POST' && url === '/api/v1/admin/models/clip-a/rerun') {
			reruns.push(url);
			return jsonResponse(202, { jobRunId: '0192f8b1-4c5d-8e9f-a0b1-2c3d4e5f6a70' });
		}
		if (url === '/api/v1/admin/models') {
			return jsonResponse(200, { models: [detector, clipA, clipB], download: null });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The rerun is offered on the active model's row.
	const card = modelCard('clip-a');
	const rerun = Array.from(card.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.textContent?.trim() === 'Rerun for the library'
	);
	if (rerun === undefined) {
		throw new Error('missing rerun button of clip-a');
	}
	rerun.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const modalText = document.body.querySelector('.modal__panel')?.textContent ?? '';
	expect(modalText).toContain('every ready image');
	expect(modalText).toContain('task panel');
	expect(reruns).toEqual([]);

	modalButton('Start the rerun').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(reruns).toEqual(['/api/v1/admin/models/clip-a/rerun']);
	// The handoff: the started line links to the task panel.
	const link = document.body.querySelector<HTMLAnchorElement>('a[href="/admin/jobs"]');
	if (link === null) {
		throw new Error('missing the task panel link');
	}
	expect(link.textContent).toContain('Open the task panel');
	expect(document.body.querySelector('.modal__panel')).toBeNull();

	unmount(page);
	flushSync();
});

test('a non-admin caller sees the mapped auth_forbidden message', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/admin/models': () =>
				jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'administrator role required' })
		})
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

test('while a download runs the page polls the listing when the stream stays silent', async () => {
	vi.useFakeTimers();
	let listed = 0;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		if (String(input) === '/api/v1/admin/models') {
			listed += 1;
			// The first listing carries a running download at zero; the
			// poll's refetch finds it advanced.
			const done = listed === 1 ? 0 : 209715200;
			return jsonResponse(200, { models: [], download: downloadRun({ done }) });
		}
		throw new Error(`unexpected fetch ${String(input)}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await vi.advanceTimersByTimeAsync(0);
	await tick();

	expect(document.body.textContent).toContain('0.0 MB of 381.8 MB');

	// No SSE block ever arrives (the stream is down); the 5-second
	// fallback poll converges the progress.
	await vi.advanceTimersByTimeAsync(5000);
	await tick();
	flushSync();
	expect(document.body.textContent).toContain('200.0 MB of 381.8 MB');

	vi.useRealTimers();
	unmount(page);
	flushSync();
});
