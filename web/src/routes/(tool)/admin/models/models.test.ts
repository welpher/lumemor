import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';

// The admin models page: the four switchable slot cards speak the
// four states (active, downloading N%, not downloaded, failed), the
// switch dialog carries the slot contract and the two-group select,
// a custom import must PASS its probe before the save enables (the
// failed probe leaves nothing saved), the sources footer edits the
// allowed hosts and the mirror base, and every failure maps onto the
// shared i18n table. The events module is mocked with a captured
// handler registry, the same pattern as the trash page test.

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

/** One official catalog entry (the ModelCatalogEntry shape). */
function catalog(id: string, overrides: Record<string, unknown> = {}) {
	return {
		id,
		slot: 'clip_embed',
		name: 'Chinese-CLIP ViT-B/16',
		license: 'unknown',
		dim: 512,
		bytes: 345839119,
		downloaded: true,
		...overrides
	};
}

/** One slot state row (the SlotStateResponse shape). */
function slotState(slot: string, overrides: Record<string, unknown> = {}) {
	return {
		slot,
		state: 'not_downloaded',
		modelId: null,
		source: null,
		progress: null,
		...overrides
	};
}

const clipActive = model('clip-a', { isActive: true, source: 'custom' });
const detector = model('detect-fx', {
	id: 'detect-fx',
	slot: 'face_detect',
	version: '2023mar',
	license: 'MIT',
	dim: null,
	inputSize: { width: 640, height: 640 },
	isActive: true
});

const listing = {
	models: [detector, clipActive],
	download: null,
	slots: [
		slotState('face_detect', {
			state: 'active',
			modelId: 'detect-fx',
			source: {
				kind: 'official',
				host: 'huggingface.co',
				sha256: '8f2383e4',
				downloadedAt: '2026-11-03T09:14:00Z'
			}
		}),
		slotState('face_recognize'),
		slotState('clip_embed', {
			state: 'active',
			modelId: 'clip-a',
			source: {
				kind: 'custom',
				host: 'models.example.net',
				sha256: '92d6b93b',
				downloadedAt: '2026-11-10T09:14:00Z'
			}
		}),
		slotState('ocr', { state: 'downloading', progress: { done: 100, total: 400 } })
	],
	official: [
		catalog('cnclip-vit-b16-vision'),
		catalog('yunet-2023mar', { slot: 'face_detect', name: 'YuNet', license: 'MIT', dim: null })
	]
};

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
		'/api/v1/admin/models': () =>
			jsonResponse(200, {
				models,
				download: null,
				slots: listing.slots,
				official: listing.official
			}),
		'/api/v1/admin/settings/models': () =>
			jsonResponse(200, {
				autoDownload: true,
				allowedHosts: ['huggingface.co'],
				mirrorBaseUrl: '',
				params: { faceDetectMinScore: 0.6, faceRecognizeThreshold: 0.55 }
			}),
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

/** The slot card whose data-slot names the pipeline slot. */
function slotCard(slot: string): HTMLElement {
	const cards = Array.from(document.body.querySelectorAll<HTMLElement>('.models__card'));
	const found = cards.find((card) => card.dataset.slot === slot);
	if (found === undefined) {
		throw new Error(`missing slot card ${slot}`);
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
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
	await initI18n();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the listing renders the four switchable cards with their state badges and facts', async () => {
	vi.stubGlobal('fetch', stubFetch(modelsRoutes([detector, clipActive])));
	const page = mountPage();
	await settle();

	// The page-header row names the domain; the cards carry the slot
	// labels and the switch action first.
	const header = document.body.querySelector<HTMLElement>('.page-header');
	expect(header?.querySelector('h1')?.textContent).toBe('Machine learning');

	const text = document.body.textContent ?? '';
	expect(text).toContain('Face detection');
	expect(text).toContain('Face recognition');
	expect(text).toContain('Semantic embeddings');
	expect(text).toContain('Text recognition');
	expect((text.match(/Swap model/g) ?? []).length).toBe(4);

	// The four states speak their badges: the active slots carry the
	// active badge, the OCR slot renders the group percent.
	expect(slotCard('face_detect').querySelector('[data-state="active"]')?.textContent).toContain('Active');
	expect(slotCard('face_recognize').querySelector('[data-state="not_downloaded"]')?.textContent).toContain(
		'Not downloaded'
	);
	expect(slotCard('ocr').querySelector('[data-state="downloading"]')?.textContent).toContain('Downloading 25%');

	// The custom slot's meta line: the source host, the digest's
	// short form.
	const custom = slotCard('clip_embed');
	expect(custom.textContent).toContain('Custom');
	expect(custom.textContent).toContain('models.example.net');
	expect(custom.textContent).toContain('92d6b93b');

	unmount(page);
});

test('the empty state offers the download; SSE progress advances it and the finish refreshes the listing', async () => {
	const posted: string[] = [];
	let listingCount = 0;
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/admin/models': () => {
				listingCount += 1;
				if (listingCount === 1) {
					return jsonResponse(200, { models: [], download: null, slots: [], official: listing.official });
				}
				return jsonResponse(200, {
					models: listing.models,
					download: downloadRun({ state: 'succeeded', done: 400371536 }),
					slots: listing.slots,
					official: listing.official
				});
			},
			'/api/v1/admin/models/download': () => {
				posted.push('download');
				return jsonResponse(202, { jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10' });
			}
		})
	);
	const page = mountPage();
	await settle();

	// The empty card owns the action surface: the download CTA.
	const cta = buttonByText('Download the default models');
	await cta.click();
	await settle();
	expect(posted).toEqual(['download']);

	// The SSE progress advances the bar; the finish refreshes the
	// listing. The convergence also rides the 5s polling fallback:
	// the read lands the finished run and the active cards without
	// waiting for the fallback (the poll fires at 5s, past the test's
	// bound; the assertions hold on the first refetch).
	deliverJobEvent({
		name: 'job.progress',
		jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10',
		kind: 'model_download',
		total: '400371536',
		done: '200185768',
		failed: '0',
		state: 'running'
	});
	const bar = document.body.querySelector<HTMLElement>('[role="progressbar"]');
	expect(bar?.getAttribute('aria-valuenow')).toBe('50');
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
	await settle();
	expect(slotCard('face_detect').querySelector('[data-state="active"]')).not.toBeNull();

	unmount(page);
}, 15_000);

test('a failed download start renders the mapped error; the retry re-posts (idempotent)', async () => {
	const posted: string[] = [];
	let attempt = 0;
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/admin/models': () =>
				jsonResponse(200, { models: [], download: null, slots: [], official: listing.official }),
			'/api/v1/admin/models/download': () => {
				posted.push('download');
				attempt += 1;
				if (attempt === 1) {
					return jsonResponse(502, { code: 'MODEL_DOWNLOAD_FAILED', message: 'the run failed' });
				}
				return jsonResponse(202, { jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10' });
			}
		})
	);
	const page = mountPage();
	await settle();

	await buttonByText('Download the default models').click();
	await settle();
	expect(document.body.textContent).toContain('The model download could not be started');

	await buttonByText('Try again').click();
	await settle();
	expect(posted).toEqual(['download', 'download']);

	unmount(page);
});

test('the switch dialog carries the slot contract and the official group; the probed import enables the save', async () => {
	const calls: Array<{ url: string; method: string }> = [];
	let probeReads = 0;
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/admin/models': () =>
				jsonResponse(200, {
					models: listing.models,
					download: null,
					slots: listing.slots,
					official: listing.official
				}),
			'/api/v1/admin/models/clip_embed/probe': () => {
				calls.push({ url: 'probe', method: 'POST' });
				return jsonResponse(202, { probeId: '0192f8b0-0000-7000-8000-00000000000a' });
			},
			'/api/v1/admin/models/probes/0192f8b0-0000-7000-8000-00000000000a': () => {
				probeReads += 1;
				if (probeReads === 1) {
					return jsonResponse(200, {
						probeId: '0192f8b0-0000-7000-8000-00000000000a',
						state: 'running',
						progress: { done: 10, total: 100 },
						dims: null,
						inputSize: null,
						sha256: null,
						hostDeclared: true,
						reason: null
					});
				}
				return jsonResponse(200, {
					probeId: '0192f8b0-0000-7000-8000-00000000000a',
					state: 'ok',
					progress: null,
					dims: 512,
					inputSize: { width: 224, height: 224 },
					sha256: '92d6b93b89bf6ccec24222be86689b6ff18370503d3d71316d5d325bd2116766',
					hostDeclared: true,
					reason: null
				});
			},
			'/api/v1/admin/models/clip_embed/source': () => {
				calls.push({ url: 'source', method: 'POST' });
				return jsonResponse(202, {
					activated: { id: 'clip-a@92d6b93b', slot: 'clip_embed' },
					dimensionChanged: null,
					wipedEmbeddings: null,
					recalcRunId: null
				});
			}
		})
	);
	const page = mountPage();
	await settle();

	// Open the semantic slot's dialog: the contract names the output
	// dimension and the red consequence line; the official group
	// preselects the running model.
	await slotCard('clip_embed').querySelector<HTMLButtonElement>('button[data-slot]')!.click();
	await settle();
	const panel = document.body.querySelector<HTMLElement>('.modal__panel')!;
	expect(panel.getAttribute('aria-label')).toBe('Swap the Semantic embeddings model');
	expect(panel.textContent).toContain('Output 512 dims');
	expect(panel.textContent).toContain('cannot be undone');

	// The select's two groups: the official list and the added
	// custom.
	const select = panel.querySelector<HTMLSelectElement>('select')!;
	const groups = Array.from(select.querySelectorAll('optgroup'));
	expect(groups[0]?.label).toBe('Official list');
	expect(groups[1]?.label).toBe('Added');

	// The save switch starts ENABLED: the official group preselects
	// the slot's running entry (a switch to it is one click), and a
	// filled URL takes over the save's meaning -- until the probe
	// passes it stays disabled.
	const save = modalButton('Save and switch');
	expect(save.disabled).toBe(false);

	// Fill the URL and run the probe: the running state, then the
	// settled ok enables the save.
	const urlField = panel.querySelector<HTMLInputElement>('.models__url-input')!;
	urlField.value = 'https://models.example.net/my.onnx';
	urlField.dispatchEvent(new Event('input', { bubbles: true }));
	await modalButton('Check').click();
	await settle();
	expect(calls.some((call) => call.url === 'probe')).toBe(true);
	expect(panel.querySelector('[data-probe-state="running"]')).not.toBeNull();
	// The probe settles through its poll: wait for the ok state (the
	// stub answers running once, then the settled facts).
	const deadline = Date.now() + 5000;
	let settled = false;
	while (Date.now() < deadline) {
		await new Promise((resolve) => setTimeout(resolve, 300));
		await settle();
		if (panel.querySelector('[data-probe-state="ok"]') !== null) {
			settled = true;
			break;
		}
	}
	expect(settled).toBe(true);
	expect(save.disabled).toBe(false);

	await save.click();
	await settle();
	expect(calls.some((call) => call.url === 'source')).toBe(true);
	// The dialog closed and the listing converged.
	expect(document.body.querySelector('.modal__panel')).toBeNull();

	unmount(page);
});

test('a failed probe leaves the save disabled and nothing saved', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/admin/models': () =>
				jsonResponse(200, {
					models: listing.models,
					download: null,
					slots: listing.slots,
					official: listing.official
				}),
			'/api/v1/admin/models/clip_embed/probe': () =>
				jsonResponse(202, { probeId: '0192f8b0-0000-7000-8000-00000000000b' }),
			'/api/v1/admin/models/probes/0192f8b0-0000-7000-8000-00000000000b': () =>
				jsonResponse(200, {
					probeId: '0192f8b0-0000-7000-8000-00000000000b',
					state: 'error',
					progress: null,
					dims: 384,
					inputSize: null,
					sha256: null,
					hostDeclared: true,
					reason: 'the model emits 384 dimensions, this slot needs 512'
				})
		})
	);
	const page = mountPage();
	await settle();

	await slotCard('clip_embed').querySelector<HTMLButtonElement>('button[data-slot]')!.click();
	await settle();
	const mismatchField = document.body.querySelector<HTMLInputElement>('.modal__panel .models__url-input')!;
	mismatchField.value = 'https://models.example.net/mismatch.onnx';
	mismatchField.dispatchEvent(new Event('input', { bubbles: true }));
	await modalButton('Check').click();
	await settle();
	// The probe settles into the error state through its poll: the
	// save switch stays disabled (a failed probe saves nothing).
	const deadline = Date.now() + 5000;
	while (Date.now() < deadline) {
		await new Promise((resolve) => setTimeout(resolve, 300));
		await settle();
		const panelRef = document.body.querySelector<HTMLElement>('.modal__panel');
		if (panelRef !== null && panelRef.querySelector('[data-probe-state="error"]') !== null) {
			break;
		}
	}
	const panelRef = document.body.querySelector<HTMLElement>('.modal__panel')!;
	expect(panelRef.querySelector('[data-probe-state="error"]')?.textContent).toContain('384');
	expect(modalButton('Save and switch').disabled).toBe(true);
	unmount(page);
});

test('the sources footer edits the allowed hosts and saves the settings', async () => {
	const puts: Array<{ body: unknown }> = [];
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/admin/models': () =>
				jsonResponse(200, {
					models: listing.models,
					download: null,
					slots: listing.slots,
					official: listing.official
				}),
			'/api/v1/admin/settings/models': () => {
				if (puts.length > 0) {
					return jsonResponse(200, {
						autoDownload: true,
						allowedHosts: ['huggingface.co', 'models.example.net'],
						mirrorBaseUrl: '',
						params: { faceDetectMinScore: 0.6, faceRecognizeThreshold: 0.55 }
					});
				}
				return jsonResponse(200, {
					autoDownload: true,
					allowedHosts: ['huggingface.co'],
					mirrorBaseUrl: '',
					params: { faceDetectMinScore: 0.6, faceRecognizeThreshold: 0.55 }
				});
			}
		})
	);
	const page = mountPage();
	await settle();

	// The footer is collapsed by default: the summary line names the
	// count without the panel being open.
	const details = document.body.querySelector<HTMLDetailsElement>('details[data-testid="models-sources"]')!;
	expect(details.open).toBe(false);
	expect(document.body.querySelector('[data-testid="models-sources-summary"]')?.textContent).toBe('1 hosts');

	// Expanding reads the settings; adding a host and saving PUTs the
	// whole document.
	details.querySelector('summary')!.click();
	await settle();
	await settle();
	const input = document.body.querySelector<HTMLInputElement>('.models__host-input')!;
	input.value = 'models.example.net';
	input.dispatchEvent(new Event('input', { bubbles: true }));
	await buttonByText('Add host').click();
	await settle();
	await buttonByText('Save').click();
	await settle();
	expect(document.body.textContent).toContain('The download sources are saved.');

	unmount(page);
});
