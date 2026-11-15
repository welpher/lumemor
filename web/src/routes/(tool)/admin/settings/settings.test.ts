import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';

// The admin settings page (the geometry: the
// page-header row with its hairline, the centered 760 column): the
// storage section's read renders the saved template, the preset
// dropdown (the built-in presets plus the custom marker) and the
// sample preview line; choosing a preset fills the input; the save
// maps the 400's validation_failed onto the shared i18n table and
// shows the server's specific reason beneath, while a successful
// save adopts the response (the fresh sample render and the saved
// note). The trash and transcoding sections (the settings feature)
// render their read documents, save their PUT bodies, and adopt the
// saved answers. The mount-roots section is GONE (the allowlist
// retired -- the boundary is the compose mount, nothing left
// to list). A non-admin caller sees the mapped auth_forbidden
// message (the page's own read is the probe).

const presets = ['{{y}}/{{MM}}/{{filename}}', '{{y}}/{{y}}-{{MM}}/{{filename}}', '{{assetId}}', '{{filename}}'];

function settingsBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		template: '{{y}}/{{MM}}/{{filename}}',
		samplePath: 'originals/0192f8c1-0000-7000-8000-000000000001/2023/07/IMG_1234.jpg',
		presets,
		...overrides
	};
}

function trashBody(retentionDays: number): Record<string, unknown> {
	return { retentionDays };
}

function transcodingBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		preset: 'auto',
		concurrency: 2,
		thumbnailFrame: true,
		keepOriginal: true,
		...overrides
	};
}

function updateCheckSettingsBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		enabled: true,
		endpoint: 'https://api.github.com/repos/welpher/lumemor/releases/latest',
		intervalHours: 24,
		...overrides
	};
}

function tilesSettingsBody(tilejsonUrl = ''): Record<string, unknown> {
	return { tilejsonUrl };
}

function modelSettingsBody(autoDownload = true): Record<string, unknown> {
	return { autoDownload, allowedHosts: ['huggingface.co'], mirrorBaseUrl: '' };
}

function imageSettingsBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		tileSize: 250,
		previewSize: 1080,
		format: 'avif',
		fullSize: false,
		rawEmbedded: true,
		...overrides
	};
}

function derivativesSummaryBody(imageCount = 0, estimateMinutes = 1): Record<string, unknown> {
	return { imageCount, estimateMinutes };
}

function updateCheckStateBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		enabled: true,
		status: 'up_to_date',
		currentVersion: '0.4.1',
		latestVersion: 'v0.4.1',
		checkedAt: new Date(Date.now() - 3 * 3600 * 1000).toISOString(),
		...overrides
	};
}

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, (init?: RequestInit) => Response>;

/** The read routes every mount fires, with default 200 bodies. */
function defaultReads(overrides: Partial<FetchRoute> = {}): FetchRoute {
	return {
		'/api/v1/admin/settings/storage': () => jsonResponse(200, settingsBody()),
		'/api/v1/admin/settings/trash': () => jsonResponse(200, trashBody(30)),
		'/api/v1/admin/settings/transcoding': () => jsonResponse(200, transcodingBody()),
		'/api/v1/admin/settings/update-check': () => jsonResponse(200, updateCheckSettingsBody()),
		'/api/v1/admin/settings/tiles': () => jsonResponse(200, tilesSettingsBody()),
		'/api/v1/admin/settings/models': () => jsonResponse(200, modelSettingsBody()),
		'/api/v1/admin/settings/image': () => jsonResponse(200, imageSettingsBody()),
		'/api/v1/admin/derivatives/summary': () => jsonResponse(200, derivativesSummaryBody()),
		'/api/v1/server/update-check': () => jsonResponse(200, updateCheckStateBody()),
		...overrides
	};
}

/** The mounted page under test, for the afterEach unmount. */
let page: ReturnType<typeof mount> | null = null;

function stubFetch(route: FetchRoute): ReturnType<typeof vi.fn> {
	return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		const answer = route[url];
		if (answer === undefined) {
			throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
		}
		return answer(init);
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
	page = mount(Page, { target: document.body, props: {} });
	flushSync();
	return page;
}

function buttonByText(text: string, scope: ParentNode = document.body): HTMLButtonElement {
	const buttons = Array.from(scope.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === text);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

function fillTemplate(value: string): void {
	const field = document.body.querySelector<HTMLInputElement>('#admin-storage-template');
	if (field === null) {
		throw new Error('missing template field');
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
}

function pickPreset(value: string): void {
	const select = document.body.querySelector<HTMLSelectElement>('#admin-storage-preset');
	if (select === null) {
		throw new Error('missing preset select');
	}
	select.value = value;
	select.dispatchEvent(new Event('change', { bubbles: true }));
}

function submitForm(form: HTMLFormElement): void {
	form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
}

function submitStorageForm(): void {
	const forms = Array.from(document.body.querySelectorAll<HTMLFormElement>('form'));
	const form = forms[0];
	if (form === undefined) {
		throw new Error('missing storage form');
	}
	submitForm(form);
}

function submitFormBy(testId: string): void {
	const section = document.body.querySelector<HTMLElement>(`[data-testid="${testId}"]`);
	const form = section?.querySelector<HTMLFormElement>('form');
	if (form === null || form === undefined) {
		throw new Error(`missing form of ${testId}`);
	}
	submitForm(form);
}

function samplePath(): string {
	const line = document.body.querySelector<HTMLElement>('[data-testid="storage-sample"]');
	return line?.textContent ?? '';
}

/** The update card's faces, in render order: exactly one renders. */
function faceNames(): string[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('[data-update-face]')).map(
		(face) => face.dataset.updateFace ?? ''
	);
}

beforeEach(async () => {
	await initI18n();
});

afterEach(() => {
	if (page !== null) {
		unmount(page);
		page = null;
	}
	document.body.innerHTML = '';
	vi.unstubAllGlobals();
});

test('the read renders the page-header row, the template, the presets and the sample line', async () => {
	vi.stubGlobal('fetch', stubFetch(defaultReads()));
	mountPage();
	await settle();

	// THE PAGE-HEADER ROW: the title
	// echoing the rail's active section above the hairline; the
	// centered 760 column owns the page below it.
	const header = document.body.querySelector<HTMLElement>('.page-header');
	expect(header?.querySelector('h1')?.textContent).toBe('Instance settings');
	expect(header?.className).toContain('border-b');
	const column = document.body.querySelector<HTMLElement>('.settings');
	expect(column?.className).toContain('max-w-[760px]');
	// The mount-roots section is GONE; the settings sections are the
	// storage template, the trash retention, transcoding, the image
	// card, the release check, the map tile source and the model
	// download policy.
	expect(document.body.querySelectorAll('.settings__card')).toHaveLength(7);
	// The new image card rides between transcoding and the release
	// check, folded like every other card.
	expect(document.body.querySelector('[data-testid="admin-settings-image"]')).not.toBeNull();
	expect(document.body.querySelector('[data-testid="admin-settings-mounts"]')).toBeNull();
	expect(document.body.querySelector('[data-testid="admin-settings-update"]')).not.toBeNull();
	expect(document.body.querySelector('[data-testid="admin-settings-tiles"]')).not.toBeNull();
	expect(document.body.querySelector('[data-testid="admin-settings-models"]')).not.toBeNull();

	const field = document.body.querySelector<HTMLInputElement>('#admin-storage-template');
	expect(field?.value).toBe('{{y}}/{{MM}}/{{filename}}');
	const select = document.body.querySelector<HTMLSelectElement>('#admin-storage-preset');
	// The sixteen-entry fixture plus the custom marker.
	expect(select?.options.length).toBe(presets.length + 1);
	expect(samplePath()).toContain('2023/07/IMG_1234.jpg');
	expect(document.body.textContent).not.toContain('Please check your entries');
});

test('choosing a preset fills the input and stays editable', async () => {
	vi.stubGlobal('fetch', stubFetch(defaultReads()));
	mountPage();
	await settle();

	pickPreset('{{assetId}}');
	await settle();
	let field = document.body.querySelector<HTMLInputElement>('#admin-storage-template');
	expect(field?.value).toBe('{{assetId}}');

	// The filled preset stays editable: hand-editing is possible.
	fillTemplate('{{assetId}}/{{filename}}');
	field = document.body.querySelector<HTMLInputElement>('#admin-storage-template');
	expect(field?.value).toBe('{{assetId}}/{{filename}}');
});

test('a rejected save shows the mapped error and the server reason, writing nothing', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/admin/settings/storage': (init?: RequestInit) =>
					init?.method === 'PUT'
						? jsonResponse(400, {
								code: 'VALIDATION_FAILED',
								message: 'storage template: unknown variable "category"',
								details: { fields: ['template'] }
							})
						: jsonResponse(200, settingsBody())
			})
		)
	);
	mountPage();
	await settle();

	fillTemplate('{{category}}/{{filename}}');
	submitStorageForm();
	await settle();

	expect(document.body.textContent).toContain('Please check your entries and try again.');
	const reason = document.body.querySelector<HTMLElement>('[data-testid="storage-error-reason"]');
	expect(reason?.textContent).toContain('unknown variable "category"');
	// The form keeps the admin's text for correcting.
	const field = document.body.querySelector<HTMLInputElement>('#admin-storage-template');
	expect(field?.value).toBe('{{category}}/{{filename}}');
});

test('a successful save adopts the response and shows the saved note', async () => {
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/admin/settings/storage': (init?: RequestInit) =>
				init?.method === 'PUT'
					? jsonResponse(
							200,
							settingsBody({
								template: '{{make}}/{{model}}/{{filename}}',
								samplePath: 'originals/0192f8c1-0000-7000-8000-000000000001/Acme/Alpha 5/IMG_1234.jpg'
							})
						)
					: jsonResponse(200, settingsBody())
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	fillTemplate('{{make}}/{{model}}/{{filename}}');
	submitStorageForm();
	await settle();

	// The PUT carried the edited template.
	const put = fetchMock.mock.calls.find((call) => (call[1] as RequestInit | undefined)?.method === 'PUT');
	expect(put).toBeDefined();
	expect(JSON.parse(String((put?.[1] as RequestInit).body))).toEqual({
		template: '{{make}}/{{model}}/{{filename}}'
	});
	// The sample line adopted the response's fresh render.
	expect(samplePath()).toContain('Acme/Alpha 5/IMG_1234.jpg');
	expect(document.body.textContent).toContain('New uploads follow the new layout');
	// The preset select follows the saved text (not a preset: custom).
	const select = document.body.querySelector<HTMLSelectElement>('#admin-storage-preset');
	expect(select?.value).toBe('custom');
});

test('the trash section renders the retention and saves the PUT body', async () => {
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/admin/settings/trash': (init?: RequestInit) =>
				init?.method === 'PUT' ? jsonResponse(200, trashBody(0)) : jsonResponse(200, trashBody(30))
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	const field = document.body.querySelector<HTMLInputElement>('[data-testid="trash-retention"]');
	expect(field?.value).toBe('30');
	// The destructive warning line renders beneath the section.
	expect(document.body.textContent).toContain('cannot be undone');

	field!.value = '0';
	field!.dispatchEvent(new Event('input', { bubbles: true }));
	submitFormBy('admin-settings-trash');
	await settle();

	const put = fetchMock.mock.calls.find(
		(call) =>
			String(call[0]).endsWith('/admin/settings/trash') && (call[1] as RequestInit | undefined)?.method === 'PUT'
	);
	expect(put).toBeDefined();
	expect(JSON.parse(String((put?.[1] as RequestInit).body))).toEqual({ retentionDays: 0 });
	// The saved note follows the adopted answer.
	expect(document.body.textContent).toContain('The next expiry run applies the new window');
	expect(field?.value).toBe('0');
});

test('the transcoding section renders the defaults and saves the PUT body', async () => {
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/admin/settings/transcoding': (init?: RequestInit) =>
				init?.method === 'PUT'
					? jsonResponse(200, transcodingBody({ preset: '720p', concurrency: 1, keepOriginal: false }))
					: jsonResponse(200, transcodingBody())
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	const preset = document.body.querySelector<HTMLSelectElement>('[data-testid="transcode-preset"]');
	const concurrency = document.body.querySelector<HTMLSelectElement>('[data-testid="transcode-concurrency"]');
	const thumbnail = document.body.querySelector<HTMLInputElement>('[data-testid="transcode-thumbnail-frame"]');
	const keep = document.body.querySelector<HTMLInputElement>('[data-testid="transcode-keep-original"]');
	expect(preset?.value).toBe('auto');
	expect(concurrency?.value).toBe('2');
	expect(thumbnail?.checked).toBe(true);
	expect(keep?.checked).toBe(true);

	preset!.value = '720p';
	preset!.dispatchEvent(new Event('change', { bubbles: true }));
	concurrency!.value = '1';
	concurrency!.dispatchEvent(new Event('change', { bubbles: true }));
	keep!.checked = false;
	keep!.dispatchEvent(new Event('change', { bubbles: true }));
	submitFormBy('admin-settings-transcoding');
	await settle();

	const put = fetchMock.mock.calls.find(
		(call) =>
			String(call[0]).endsWith('/admin/settings/transcoding') &&
			(call[1] as RequestInit | undefined)?.method === 'PUT'
	);
	expect(put).toBeDefined();
	expect(JSON.parse(String((put?.[1] as RequestInit).body))).toEqual({
		preset: '720p',
		concurrency: 1,
		thumbnailFrame: true,
		keepOriginal: false
	});
	// The scope note renders; the saved note follows the save.
	expect(document.body.textContent).toContain('never regenerated');
	expect(document.body.textContent).toContain('Newly uploaded videos follow the new profile');
});

test('a rejected transcoding save shows the mapped error', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/admin/settings/transcoding': (init?: RequestInit) =>
					init?.method === 'PUT'
						? jsonResponse(400, {
								code: 'VALIDATION_FAILED',
								message: 'preset must be one of auto, 1080p, 720p, 480p, off',
								details: { fields: ['preset'] }
							})
						: jsonResponse(200, transcodingBody())
			})
		)
	);
	mountPage();
	await settle();

	submitFormBy('admin-settings-transcoding');
	await settle();

	expect(document.body.textContent).toContain('Please check your entries and try again.');
});

test('a non-admin read shows the mapped auth_forbidden message', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/admin/settings/storage': () =>
					jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'administrator role required' }),
				'/api/v1/admin/settings/trash': () =>
					jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'administrator role required' }),
				'/api/v1/admin/settings/transcoding': () =>
					jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'administrator role required' })
			})
		)
	);
	mountPage();
	await settle();

	expect(document.body.textContent).toContain("You don't have permission to do this.");
	expect(buttonByText('Try again')).toBeDefined();
});

test('an unparseable read maps to the generic failure', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(defaultReads({ '/api/v1/admin/settings/storage': () => jsonResponse(200, { nope: true }) }))
	);
	mountPage();
	await settle();

	expect(document.body.textContent).toContain('Something went wrong');
});

test('the release card renders the last verdict, the last-checked line and the manual check', async () => {
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/server/update-check': (init?: RequestInit) =>
				init?.method === 'POST'
					? jsonResponse(
							200,
							updateCheckStateBody({
								status: 'update_available',
								latestVersion: 'v0.5.0',
								checkedAt: new Date().toISOString()
							})
						)
					: jsonResponse(200, updateCheckStateBody())
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	// The first face: up to date, with the quiet last-checked line and
	// exactly one face on the page.
	expect(faceNames()).toEqual(['latest']);
	expect(document.querySelector('[data-update-face="latest"]')?.textContent).toContain('Up to date');
	expect(document.querySelector('[data-update-face="latest"]')?.textContent).toContain('Last checked');
	expect(document.body.querySelector('[data-testid="update-check-version"]')?.textContent).toContain('0.4.1');
	expect(document.body.querySelector('[data-testid="update-check-version"]')?.textContent).toContain('Self-hosted');
	// The privacy boundary stands beside the knobs.
	expect(document.body.textContent).toContain('Only the version number is compared');

	// The manual check probes through the server and adopts the fresh
	// verdict, including the "nothing is installed for you" line.
	buttonByText('Check now').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	const post = fetchMock.mock.calls.find((call) => (call[1] as RequestInit | undefined)?.method === 'POST');
	expect(post).toBeDefined();
	expect(String(post?.[0])).toBe('/api/v1/server/update-check');
	expect(faceNames()).toEqual(['available']);
	const available = document.querySelector('[data-update-face="available"]');
	expect(available?.textContent).toContain('v0.5.0 available');
	// The pair: the running version stays on the card beside the offer.
	expect(document.body.querySelector('[data-testid="update-check-version"]')?.textContent).toContain('0.4.1');
	expect(document.body.textContent).toContain('nothing is downloaded, installed or restarted for you');
	expect(document.body.textContent).toContain('pull');
});

test('the manual check shows its busy state while the probe runs', async () => {
	let release = (): void => undefined;
	const held = new Promise<void>((resolve) => {
		release = resolve;
	});
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (url === '/api/v1/server/update-check' && init?.method === 'POST') {
				await held;
				return jsonResponse(
					200,
					updateCheckStateBody({ status: 'up_to_date', checkedAt: new Date().toISOString() })
				);
			}
			const answer = defaultReads()[url];
			if (answer === undefined) {
				throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
			}
			return answer(init);
		})
	);
	mountPage();
	await settle();

	const button = buttonByText('Check now');
	button.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await tick();

	// The busy state: the label names the work, the button refuses a
	// second click, and the face keeps its last verdict while the
	// probe runs.
	const busy = buttonByText('Checking…');
	expect(busy.disabled).toBe(true);
	expect(busy.getAttribute('aria-busy')).toBe('true');
	expect(faceNames()).toEqual(['latest']);

	release();
	await settle();
	expect(buttonByText('Check now').disabled).toBe(false);
});

test('a failed check stays silent: the quiet last-checked line, no danger text', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/server/update-check': () =>
					jsonResponse(
						200,
						updateCheckStateBody({
							status: 'unknown',
							latestVersion: null,
							checkedAt: new Date(Date.now() - 3 * 86400 * 1000).toISOString()
						})
					)
			})
		)
	);
	mountPage();
	await settle();

	// The third face is the silent one: it says when it last asked and
	// that the feed could not be reached -- nothing red, no chip, no
	// button-styled warning.
	expect(faceNames()).toEqual(['failed']);
	const failed = document.querySelector('[data-update-face="failed"]');
	expect(failed?.textContent).toContain("couldn't reach the release feed");
	expect(failed?.querySelector('.settings__update-chip')).toBeNull();
	expect(failed?.querySelector('.text-danger-text')).toBeNull();
	expect(document.querySelector('.settings__update-face .text-danger-text')).toBeNull();
	expect(document.body.querySelector('.toast, [role="alert"]')).toBeNull();
});

test('a check that never ran says so instead of inventing a verdict', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/server/update-check': () =>
					jsonResponse(200, updateCheckStateBody({ status: 'unknown', latestVersion: null, checkedAt: null }))
			})
		)
	);
	mountPage();
	await settle();

	expect(faceNames()).toEqual(['failed']);
	expect(document.querySelector('[data-update-face="failed"]')?.textContent).toContain('Not checked yet.');
	expect(document.querySelector('.settings__update-face .text-danger-text')).toBeNull();
});

test('a check switched off renders its own quiet face, not a verdict', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/server/update-check': () =>
					jsonResponse(200, updateCheckStateBody({ enabled: false, status: 'disabled', latestVersion: null }))
			})
		)
	);
	mountPage();
	await settle();

	expect(faceNames()).toEqual(['off']);
	expect(document.body.querySelector('[data-testid="update-check-now"]')?.hasAttribute('disabled')).toBe(true);
});

test('the tile source and the model policy save their own bodies', async () => {
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/admin/settings/tiles': (init?: RequestInit) =>
				init?.method === 'PUT'
					? jsonResponse(200, tilesSettingsBody('https://tiles.example.net/osm/tilejson.json'))
					: jsonResponse(200, tilesSettingsBody()),
			'/api/v1/admin/settings/models': (init?: RequestInit) =>
				init?.method === 'PUT'
					? jsonResponse(200, modelSettingsBody(false))
					: jsonResponse(200, modelSettingsBody())
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	// The shipped state: no source configured, the automatic policy on.
	const tilesField = document.body.querySelector<HTMLInputElement>('#admin-tiles-source');
	expect(tilesField?.value).toBe('');
	expect(document.body.querySelector<HTMLInputElement>('[data-testid="models-auto-download"]')?.checked).toBe(true);
	// The boundary copy rides both cards.
	expect(document.body.textContent).toContain('never fetches or proxies a tile');
	expect(document.body.textContent).toContain('a wiped model store');

	tilesField!.value = 'https://tiles.example.net/osm/tilejson.json';
	tilesField!.dispatchEvent(new Event('input', { bubbles: true }));
	submitFormBy('admin-settings-tiles');
	await settle();

	const tilesPut = fetchMock.mock.calls.find(
		(call) =>
			String(call[0]).endsWith('/admin/settings/tiles') && (call[1] as RequestInit | undefined)?.method === 'PUT'
	);
	expect(JSON.parse(String((tilesPut?.[1] as RequestInit).body))).toEqual({
		tilejsonUrl: 'https://tiles.example.net/osm/tilejson.json'
	});
	expect(document.body.textContent).toContain('The map picks it up on its next load');

	const auto = document.body.querySelector<HTMLInputElement>('[data-testid="models-auto-download"]');
	auto!.checked = false;
	auto!.dispatchEvent(new Event('change', { bubbles: true }));
	submitFormBy('admin-settings-models');
	await settle();

	const modelsPut = fetchMock.mock.calls.find(
		(call) =>
			String(call[0]).endsWith('/admin/settings/models') && (call[1] as RequestInit | undefined)?.method === 'PUT'
	);
	expect(JSON.parse(String((modelsPut?.[1] as RequestInit).body))).toEqual({
		autoDownload: false,
		allowedHosts: ['huggingface.co'],
		mirrorBaseUrl: ''
	});
});

// THE COLLAPSE AND THE IMAGE CARD (E33): the settings page is an
// index of folded cards -- the first visit lands collapsed with
// every summary naming its card's current value, the expanded set is
// remembered in localStorage (stored as the EXPANDED ids, so a card
// added later stays collapsed), and the image card carries the
// honest cost line with its explicit rebuild.

test('the first visit is the collapsed index: every card folded, every summary naming its value', async () => {
	window.localStorage.removeItem('lumemor:settings:expanded');
	vi.stubGlobal('fetch', stubFetch(defaultReads()));
	mountPage();
	await settle();

	const cards = Array.from(document.body.querySelectorAll<HTMLDetailsElement>('.settings__card'));
	expect(cards).toHaveLength(7);
	// No card opens by default, and the fold is in the MARKUP (no
	// open attribute): a JS-off browser still sees the index.
	for (const card of cards) {
		expect(card.hasAttribute('open')).toBe(false);
	}
	// Each collapsed row states its card's current value.
	const meta = (testId: string): string =>
		document.body.querySelector<HTMLElement>(`[data-testid="${testId}"] .settings__section-meta`)?.textContent ??
		'';
	expect(meta('admin-settings-storage')).toContain('{{y}}/{{MM}}/{{filename}}');
	expect(meta('admin-settings-trash')).toContain('30');
	expect(meta('admin-settings-image')).toContain('250');
	expect(meta('admin-settings-image')).toContain('1080');
	expect(meta('admin-settings-image')).toContain('AVIF');
	expect(meta('admin-settings-tiles')).toContain('not configured');
	expect(meta('admin-settings-models')).toContain('on');
});

test('expanding cards persists the expanded set, and a fresh mount restores it', async () => {
	window.localStorage.removeItem('lumemor:settings:expanded');
	vi.stubGlobal('fetch', stubFetch(defaultReads()));
	mountPage();
	await settle();

	// Open two cards by toggling their summaries (the native fold).
	const image = document.body.querySelector<HTMLDetailsElement>('[data-testid="admin-settings-image"]');
	const update = document.body.querySelector<HTMLDetailsElement>('[data-testid="admin-settings-update"]');
	if (image === null || update === null) {
		throw new Error('missing cards');
	}
	image.open = true;
	image.dispatchEvent(new Event('toggle', { bubbles: false }));
	update.open = true;
	update.dispatchEvent(new Event('toggle', { bubbles: false }));
	await tick();

	// The stored set names the EXPANDED ids only.
	const stored = JSON.parse(window.localStorage.getItem('lumemor:settings:expanded') ?? '[]');
	expect(stored).toEqual(expect.arrayContaining(['image', 'update']));
	expect(stored).not.toContain('storage');

	// A fresh mount (the refresh) restores exactly those.
	unmount(page!);
	page = null;
	document.body.innerHTML = '';
	mountPage();
	await settle();
	expect(document.body.querySelector<HTMLDetailsElement>('[data-testid="admin-settings-image"]')?.open).toBe(true);
	expect(document.body.querySelector<HTMLDetailsElement>('[data-testid="admin-settings-update"]')?.open).toBe(true);
	expect(document.body.querySelector<HTMLDetailsElement>('[data-testid="admin-settings-storage"]')?.open).toBe(false);
});

test('the image card saves its PUT body and the consequence line names the no-recompute rule', async () => {
	window.localStorage.removeItem('lumemor:settings:expanded');
	const reads = defaultReads();
	let savedBody: unknown = null;
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (url === '/api/v1/admin/settings/image' && init?.method === 'PUT') {
				savedBody = JSON.parse(String(init.body));
				return jsonResponse(200, imageSettingsBody({ tileSize: 320, format: 'webp' }));
			}
			const answer = reads[url];
			if (answer === undefined) {
				throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
			}
			return answer(init);
		})
	);
	mountPage();
	await settle();

	// The consequence line rides in the card: the save recomputes
	// nothing, the rebuild is the explicit action.
	const card = document.body.querySelector<HTMLElement>('[data-testid="admin-settings-image"]');
	expect(card?.textContent).toContain('Saving recomputes nothing');
	// The cost line names the summary's declared estimate.
	expect(card?.textContent).toContain('a full rebuild takes about');

	// The save PUTs the whole document.
	submitFormBy('admin-settings-image');
	await settle();
	expect(savedBody).toEqual({
		tileSize: 250,
		previewSize: 1080,
		format: 'avif',
		fullSize: false,
		rawEmbedded: true
	});
	// The adopted answer fills the card and the collapsed row.
	const tileSelect = document.body.querySelector<HTMLSelectElement>('#admin-image-thumb');
	expect(tileSelect?.value).toBe('320');
	const meta = document.body.querySelector<HTMLElement>(
		'[data-testid="admin-settings-image"] .settings__section-meta'
	);
	expect(meta?.textContent).toContain('WEBP');
});

test('the rebuild dialog opens over the collapsed card, defaults to the missing scope and POSTs it', async () => {
	window.localStorage.removeItem('lumemor:settings:expanded');
	let postedScope: unknown = null;
	const reads = defaultReads();
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (url === '/api/v1/admin/derivatives/rebuild' && init?.method === 'POST') {
				postedScope = JSON.parse(String(init.body)).scope;
				return jsonResponse(202, { jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10' });
			}
			const answer = reads[url];
			if (answer === undefined) {
				throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
			}
			return answer(init);
		})
	);
	mountPage();
	await settle();

	// The card stays folded while the dialog opens (the dialog is a
	// page-level layer).
	const image = document.body.querySelector<HTMLDetailsElement>('[data-testid="admin-settings-image"]');
	expect(image?.hasAttribute('open')).toBe(false);
	buttonByText('Rebuild derivatives').click();
	await settle();
	const dialog = document.body.querySelector<HTMLElement>('[data-testid="rebuild-dialog"]');
	expect(dialog).not.toBeNull();
	// The missing scope is the default radio.
	const missing = dialog?.querySelector<HTMLInputElement>('input[value="missing"]');
	expect(missing?.checked).toBe(true);
	// The scope note carries the summary's declared estimate.
	expect(dialog?.textContent).toContain('Re-derives all 0 images');

	buttonByText('Start rebuild').click();
	await settle();
	expect(postedScope).toBe('missing');
});
