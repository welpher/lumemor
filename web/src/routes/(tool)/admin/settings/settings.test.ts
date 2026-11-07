import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';

// The admin storage settings page: the read renders the saved
// template, the preset dropdown (the sixteen built-in presets plus
// the custom marker) and the sample preview line; choosing a preset
// fills the input; the save maps the 400's validation_failed onto the
// shared i18n table and shows the server's specific reason beneath,
// while a successful save adopts the response (the fresh sample
// render and the saved note). A non-admin caller sees the mapped
// auth_forbidden message (the page's own read is the probe).

const presets = [
	'{{y}}/{{MM}}/{{filename}}',
	'{{y}}/{{y}}-{{MM}}/{{filename}}',
	'{{assetId}}',
	'{{filename}}'
];

function settingsBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		template: '{{y}}/{{MM}}/{{filename}}',
		samplePath: 'originals/0192f8c1-0000-7000-8000-000000000001/2023/07/IMG_1234.jpg',
		presets,
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

function buttonByText(text: string): HTMLButtonElement {
	const buttons = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button'));
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

function submitForm(): void {
	const form = document.body.querySelector<HTMLFormElement>('form');
	if (form === null) {
		throw new Error('missing form');
	}
	form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
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

test('the read renders the template, the presets and the sample line', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/admin/settings/storage': () => jsonResponse(200, settingsBody()) })
	);
	mountPage();
	await settle();

	const field = document.body.querySelector<HTMLInputElement>('#admin-storage-template');
	expect(field?.value).toBe('{{y}}/{{MM}}/{{filename}}');
	const select = document.body.querySelector<HTMLSelectElement>('#admin-storage-preset');
	// The sixteen-entry fixture plus the custom marker.
	expect(select?.options.length).toBe(presets.length + 1);
	expect(samplePath()).toContain('2023/07/IMG_1234.jpg');
	expect(document.body.textContent).not.toContain('Please check your entries');
});

test('choosing a preset fills the input and stays editable', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/admin/settings/storage': () => jsonResponse(200, settingsBody()) })
	);
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
		stubFetch({
			'/api/v1/admin/settings/storage': (init?: RequestInit) =>
				init?.method === 'PUT'
					? jsonResponse(400, {
							code: 'VALIDATION_FAILED',
							message: 'storage template: unknown variable "category"',
							details: { fields: ['template'] }
						})
					: jsonResponse(200, settingsBody())
		})
	);
	mountPage();
	await settle();

	fillTemplate('{{category}}/{{filename}}');
	submitForm();
	await settle();

	expect(document.body.textContent).toContain('Please check your entries and try again.');
	const reason = document.body.querySelector<HTMLElement>('[data-testid="storage-error-reason"]');
	expect(reason?.textContent).toContain('unknown variable "category"');
	// The form keeps the admin's text for correcting.
	const field = document.body.querySelector<HTMLInputElement>('#admin-storage-template');
	expect(field?.value).toBe('{{category}}/{{filename}}');
});

test('a successful save adopts the response and shows the saved note', async () => {
	const fetchMock = stubFetch({
		'/api/v1/admin/settings/storage': (init?: RequestInit) =>
			init?.method === 'PUT'
				? jsonResponse(200, settingsBody({
						template: '{{make}}/{{model}}/{{filename}}',
						samplePath: 'originals/0192f8c1-0000-7000-8000-000000000001/Acme/Alpha 5/IMG_1234.jpg'
					}))
				: jsonResponse(200, settingsBody())
	});
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	fillTemplate('{{make}}/{{model}}/{{filename}}');
	submitForm();
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

test('a non-admin read shows the mapped auth_forbidden message', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/admin/settings/storage': () =>
				jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'administrator role required' })
		})
	);
	mountPage();
	await settle();

	expect(document.body.textContent).toContain("You don't have permission to do this.");
	expect(buttonByText('Try again')).toBeDefined();
});

test('an unparseable read maps to the generic failure', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/admin/settings/storage': () => jsonResponse(200, { nope: true }) })
	);
	mountPage();
	await settle();

	expect(document.body.textContent).toContain('Something went wrong');
});
