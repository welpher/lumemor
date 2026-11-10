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

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, (init?: RequestInit) => Response>;

/** The three read routes every mount fires, with default 200 bodies. */
function defaultReads(overrides: Partial<FetchRoute> = {}): FetchRoute {
	return {
		'/api/v1/admin/settings/storage': () => jsonResponse(200, settingsBody()),
		'/api/v1/admin/settings/trash': () => jsonResponse(200, trashBody(30)),
		'/api/v1/admin/settings/transcoding': () => jsonResponse(200, transcodingBody()),
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
	// The mount-roots section is GONE: the page carries exactly
	// the three sections.
	expect(document.body.querySelectorAll('.settings__card')).toHaveLength(3);
	expect(document.body.querySelector('[data-testid="admin-settings-mounts"]')).toBeNull();

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
