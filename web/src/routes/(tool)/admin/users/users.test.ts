import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearToasts, toasts } from '$lib/components/toast.svelte';

// The admin user management page (the 2026-11-09 modal pin, the
// alignment and disposal update): the listing renders the accounts
// with their role and disabled badges, a non-admin caller sees the
// mapped auth_forbidden message (the page's own listing is the probe
// -- the session contract carries no role), the create flow rides the
// page-header row's "New user" button (UserPlus + text) into a MODAL
// whose fields and failure surface are the inline form's own --
// success closes it with the listing converging, a failure keeps it
// open with the mapped error inside -- and the disable toggle
// confirms first, adopts the PATCH response, and surfaces the
// server's self-disable refusal as the mapped message. The geometry
// is the admin column's (the 760 bound, the table's own scroll card),
// and the disposal is two steps: the destination of the photos, then
// the irreversible confirmation, with every refusal staying in the
// second dialog.

const adminDefaults = {
	name: 'Ada Lovelace',
	role: 'admin' as const,
	isDisabled: false,
	createdAt: '2026-10-29T08:30:00Z'
};

const grace = {
	id: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70',
	email: 'grace@example.com',
	name: 'Grace Hopper',
	role: 'user' as const,
	isDisabled: false,
	createdAt: '2026-10-30T11:04:05Z'
};

const adminUser = {
	id: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	email: 'ada@example.com',
	...adminDefaults
};

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, () => Response>;

function usersRoutes(overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/admin/users': () => jsonResponse(200, { items: [adminUser, grace] }),
		'/api/v1/admin/settings/trash': () => jsonResponse(200, { retentionDays: 30 }),
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

function fillField(selector: string, value: string): void {
	const field = document.body.querySelector<HTMLInputElement>(selector);
	if (field === null) {
		throw new Error(`missing field ${selector}`);
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
}

/** Open the creation modal through the header row's button. */
function openCreateModal(): void {
	const button = buttonByText('New user');
	button.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
}

/** Submit the creation modal through its Create button. */
function submitCreate(): void {
	modalButton('Create user').dispatchEvent(new MouseEvent('click', { bubbles: true }));
}

function rows(): HTMLElement[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.users__table tbody tr'));
}

/** The single action button of the row naming one account's email. */
function rowAction(email: string): HTMLButtonElement {
	const row = rows().find((tr) => (tr.textContent ?? '').includes(email));
	const button = row?.querySelector<HTMLButtonElement>('button');
	if (button === null || button === undefined) {
		throw new Error(`missing action button of the row of ${email}`);
	}
	return button;
}

/** The disposal entry of the row naming one account's email. */
function deleteButton(email: string): HTMLButtonElement {
	const row = rows().find((tr) => (tr.textContent ?? '').includes(email));
	const buttons = Array.from(row?.querySelectorAll<HTMLButtonElement>('button') ?? []);
	const found = buttons.find((button) => button.textContent?.trim() === 'Delete');
	if (found === undefined) {
		throw new Error(`missing delete button of the row of ${email}`);
	}
	return found;
}

/** Click one element the way the browser does. */
function click(element: Element): void {
	element.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
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

beforeEach(async () => {
	document.body.textContent = '';
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
	clearToasts();
	vi.unstubAllGlobals();
});

test('the listing renders the accounts with role and status badges', async () => {
	vi.stubGlobal('fetch', stubFetch(usersRoutes()));
	const page = mountPage();
	await settle();

	expect(rows()).toHaveLength(2);
	const text = document.body.textContent ?? '';
	// Both accounts render with their identity columns.
	expect(text).toContain('Ada Lovelace');
	expect(text).toContain('ada@example.com');
	expect(text).toContain('Grace Hopper');
	// The role and status badges.
	expect(text).toContain('Administrator');
	expect(text).toContain('User');
	expect(text).toContain('Active');
	// Every enabled row carries the disable action.
	expect(buttonByText('Disable')).toBeTruthy();
	// The creation date is locale-formatted.
	expect(text).toContain('Oct 29, 2026');

	unmount(page);
	flushSync();
});

test('a non-admin caller sees the mapped auth_forbidden message', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/admin/users': () =>
				jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'administrator role required' })
		})
	);
	const page = mountPage();
	await settle();

	expect(rows()).toHaveLength(0);
	// The generic AUTH_FORBIDDEN message (the albums branch made
	// the code cover the album share face too, so the text stopped
	// naming the admin surface).
	expect(document.body.querySelector('[role="alert"]')?.textContent).toContain(
		"You don't have permission to do this."
	);
	// The retry affordance is present (the probe may succeed later).
	expect(buttonByText('Try again')).toBeTruthy();

	unmount(page);
	flushSync();
});

test('the create modal posts the payload, closes on success and appends the created account', async () => {
	const bodies: unknown[] = [];
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (url === '/api/v1/admin/users' && init?.method === 'POST') {
			bodies.push(JSON.parse(String(init.body)));
			return jsonResponse(201, {
				id: '0192f5b2-6c7d-8e9f-a0b1-2c3d4e5f6070',
				name: 'Kid Hopper',
				email: 'kid@example.com',
				role: 'user',
				isDisabled: false,
				createdAt: '2026-10-31T09:15:00Z'
			});
		}
		if (url === '/api/v1/admin/users') {
			return jsonResponse(200, { items: [adminUser, grace] });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The header row's button (UserPlus + text, the primary-flow
	// shape) opens the modal -- the inline section is gone.
	const entry = buttonByText('New user');
	expect(entry.querySelector('svg.lucide-user-plus')).not.toBeNull();
	openCreateModal();
	expect(document.body.querySelector('.modal__panel')?.getAttribute('aria-label')).toBe('Add a user');

	fillField('#admin-user-name', 'Kid Hopper');
	fillField('#admin-user-email', 'kid@example.com');
	fillField('#admin-user-password', 'correct horse battery staple');
	submitCreate();
	await settle();

	// The posted payload: the optional name, the credentials, and
	// the default role.
	expect(bodies).toEqual([
		{
			name: 'Kid Hopper',
			email: 'kid@example.com',
			password: 'correct horse battery staple',
			role: 'user'
		}
	]);
	// The created account joins the listing without a refetch, and
	// success closed the modal (the fields reset with the next open).
	expect(rows()).toHaveLength(3);
	expect(document.body.textContent).toContain('Kid Hopper');
	expect(document.body.querySelector('.modal__panel')).toBeNull();

	unmount(page);
	flushSync();
});

test('a duplicate email maps to the user_email_taken message', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (url === '/api/v1/admin/users' && init?.method === 'POST') {
				return jsonResponse(409, { code: 'USER_EMAIL_TAKEN', message: 'duplicate' });
			}
			return jsonResponse(200, { items: [adminUser, grace] });
		})
	);
	const page = mountPage();
	await settle();

	openCreateModal();
	fillField('#admin-user-email', 'grace@example.com');
	fillField('#admin-user-password', 'correct horse battery staple');
	submitCreate();
	await settle();

	// A failed create keeps the modal open with the mapped error
	// inside (the disable confirm's pattern), the fields kept for
	// correcting.
	expect(document.body.querySelector('.modal__panel [role="alert"]')?.textContent).toContain(
		'An account with this email address already exists.'
	);
	expect(document.body.querySelector<HTMLInputElement>('#admin-user-email')?.value).toBe('grace@example.com');
	// The listing is untouched.
	expect(rows()).toHaveLength(2);

	unmount(page);
	flushSync();
});

test('a validation failure maps to the validation message', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (url === '/api/v1/admin/users' && init?.method === 'POST') {
				return jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: 'invalid account payload',
					details: { fields: ['password'] }
				});
			}
			return jsonResponse(200, { items: [adminUser, grace] });
		})
	);
	const page = mountPage();
	await settle();

	openCreateModal();
	fillField('#admin-user-email', 'kid@example.com');
	fillField('#admin-user-password', 'short');
	submitCreate();
	await settle();

	expect(document.body.querySelector('.modal__panel [role="alert"]')?.textContent).toContain(
		'Please check your entries and try again.'
	);

	unmount(page);
	flushSync();
});

test('the disable toggle confirms, patches, and adopts the fresh state', async () => {
	const patches: Array<{ id: string; body: unknown }> = [];
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'PATCH' && url === `/api/v1/admin/users/${grace.id}`) {
			patches.push({ id: url, body: JSON.parse(String(init.body)) });
			return jsonResponse(200, { ...grace, isDisabled: true });
		}
		if (url === '/api/v1/admin/users') {
			return jsonResponse(200, { items: [adminUser, grace] });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The confirm-first gate: the disable button opens the modal with
	// the account's name.
	rowAction('grace@example.com').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(document.body.textContent).toContain('Disable this account?');
	expect(document.body.textContent).toContain('Grace Hopper');
	expect(patches).toEqual([]);

	modalButton('Disable').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(patches).toEqual([{ id: `/api/v1/admin/users/${grace.id}`, body: { isDisabled: true } }]);
	// The row adopted the response: the disabled badge and the
	// enable action.
	expect(document.body.textContent).toContain('Disabled');
	expect(buttonByText('Enable')).toBeTruthy();
	expect(rows()).toHaveLength(2);

	unmount(page);
	flushSync();
});

test('the enable action patches directly and restores the active badge', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'PATCH' && url === `/api/v1/admin/users/${grace.id}`) {
			return jsonResponse(200, { ...grace, isDisabled: false });
		}
		if (url === '/api/v1/admin/users') {
			return jsonResponse(200, { items: [adminUser, { ...grace, isDisabled: true }] });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The already-disabled row carries the enable action (no
	// confirmation: the action restores access).
	rowAction('grace@example.com').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(document.body.textContent).toContain('Active');
	expect(buttonByText('Disable')).toBeTruthy();

	unmount(page);
	flushSync();
});

test('the self-disable refusal surfaces the mapped message (the guard is server-side)', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (init?.method === 'PATCH' && url === `/api/v1/admin/users/${adminUser.id}`) {
				return jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: 'an administrator cannot disable their own account',
					details: { fields: ['id'] }
				});
			}
			if (url === '/api/v1/admin/users') {
				return jsonResponse(200, { items: [adminUser, grace] });
			}
			throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
		})
	);
	const page = mountPage();
	await settle();

	// The web client cannot know its own user id (the listing carries
	// no caller marker), so the toggle is offered; the server's
	// refusal is the guard and its mapped message the visible half.
	rowAction('ada@example.com').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	modalButton('Disable').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(document.body.querySelector('[role="alert"]')?.textContent).toContain(
		'Please check your entries and try again.'
	);
	// The listing kept the row's untouched state.
	expect(document.body.textContent).toContain('Active');

	unmount(page);
	flushSync();
});

test('a failed disable keeps the confirm dialog open with the mapped error', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (init?.method === 'PATCH' && url === `/api/v1/admin/users/${grace.id}`) {
				return jsonResponse(500, { code: 'INTERNAL', message: 'boom' });
			}
			if (url === '/api/v1/admin/users') {
				return jsonResponse(200, { items: [adminUser, grace] });
			}
			throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
		})
	);
	const page = mountPage();
	await settle();

	rowAction('grace@example.com').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	modalButton('Disable').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The confirm dialog stays open with the mapped error inside it
	// (the trash page's confirm pattern: a failed confirm-able write
	// recovers in place, not under the table).
	expect(document.body.querySelector('.modal__panel')).not.toBeNull();
	expect(document.body.querySelector('.modal__panel [role="alert"]')?.textContent).toContain('Something went wrong');
	// The row is untouched: still active, still offered the toggle.
	expect(document.body.textContent).toContain('Active');
	expect(buttonByText('Disable')).toBeTruthy();

	unmount(page);
	flushSync();
});

test('a stale row (404) reconciles through a refetch', async () => {
	let listed = 0;
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (init?.method === 'PATCH' && url === `/api/v1/admin/users/${grace.id}`) {
			return jsonResponse(404, { code: 'USER_NOT_FOUND', message: 'no account' });
		}
		if (url === '/api/v1/admin/users') {
			listed += 1;
			// The first listing carries Grace's row; the refetch (the
			// 404's reconciliation) answers without it.
			return jsonResponse(200, { items: listed === 1 ? [adminUser, grace] : [adminUser] });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(listed).toBe(1);

	rowAction('grace@example.com').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	modalButton('Disable').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The 404 triggered the listing's reconciliation.
	expect(listed).toBe(2);
	expect(rows()).toHaveLength(1);
	// The already-gone reading (the trash page's 404 precedent): the
	// confirm dialog closed with the reconciliation -- there is
	// nothing left to confirm against a row the listing no longer
	// carries.
	expect(document.body.querySelector('.modal__panel')).toBeNull();

	unmount(page);
	flushSync();
});

test('the column is the 760 one and the table scrolls inside its own card', async () => {
	vi.stubGlobal('fetch', stubFetch(usersRoutes()));
	const page = mountPage();
	await settle();

	// The admin content column's bound (the reading/form surfaces'
	// shared width) and its centering.
	const column = document.body.querySelector<HTMLElement>('.users');
	expect(column?.className).toContain('max-w-[760px]');
	expect(column?.className).toContain('mx-auto');
	expect(column?.className).toContain('gap-6');

	// The card moved off the table and onto the scroll shell: a narrow
	// viewport scrolls the table inside the card instead of pushing
	// the page sideways.
	const shell = document.body.querySelector<HTMLElement>('.users__table-shell');
	expect(shell?.className).toContain('relative');
	expect(shell?.className).toContain('overflow-x-auto');
	expect(shell?.className).toContain('rounded-card');
	expect(shell?.className).toContain('border');
	expect(shell?.className).toContain('shadow-card');
	const table = shell?.querySelector<HTMLElement>('table.users__table');
	expect(table).not.toBeNull();
	expect(table?.className).not.toContain('shadow-card');
	expect(table?.className).not.toContain('rounded-card');

	unmount(page);
	flushSync();
});

test('the disposal asks twice, then hands the photos to the chosen member', async () => {
	const deletes: Array<{ url: string; body: unknown }> = [];
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (init?.method === 'DELETE') {
				deletes.push({ url, body: JSON.parse(String(init.body)) });
				return new Response(null, { status: 204 });
			}
			const answer = usersRoutes()[url];
			if (answer === undefined) {
				throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
			}
			return answer();
		})
	);
	const page = mountPage();
	await settle();

	// The row's disposal entry is a text button in the danger tone
	// (red border and red text, transparent ground) -- never a solid
	// fill, which beside Disable would read like an everyday action.
	const entry = deleteButton('grace@example.com');
	expect(entry.className).toContain('btn--danger-text');
	expect(entry.querySelector('svg.lucide-user-minus')).not.toBeNull();
	expect(entry.disabled).toBe(false);
	click(entry);

	// Step one: the title names the account, both destinations stand
	// side by side, and the trash arm's subline carries the instance's
	// own retention window (read when the dialog opens).
	const panel = document.body.querySelector<HTMLElement>('.modal__panel');
	expect(panel?.getAttribute('aria-label')).toBe('Delete Grace Hopper');
	expect(document.body.textContent).toContain('Transfer the photos to another member');
	expect(document.body.textContent).toContain('Move the photos to the trash');
	await settle();
	expect(document.body.querySelector('.users__delete-retention')?.textContent).toContain('after 30 days');
	// The receiver picker lists the OTHER accounts only.
	const select = document.body.querySelector<HTMLSelectElement>('#user-delete-transfer');
	expect(Array.from(select?.options ?? []).map((option) => option.textContent)).toEqual(['Ada Lovelace']);
	// Nothing has been written yet.
	expect(deletes).toEqual([]);

	click(modalButton('Continue'));

	// Step two: the irreversible confirmation, still without a write.
	expect(document.body.querySelector('.modal__panel')?.getAttribute('aria-label')).toBe('Confirm deletion');
	expect(document.body.textContent).toContain('Grace Hopper');
	expect(document.body.textContent).toContain('This cannot be undone.');
	expect(deletes).toEqual([]);

	click(modalButton('Delete permanently'));
	await settle();

	expect(deletes).toEqual([{ url: `/api/v1/admin/users/${grace.id}`, body: { transferTo: adminUser.id } }]);
	// The row left the listing and the dialog closed; the danger toast
	// carries the acknowledgment.
	expect(rows()).toHaveLength(1);
	expect(document.body.querySelector('.modal__panel')).toBeNull();
	const live = toasts();
	expect(live.at(-1)?.text).toBe('The account has been deleted.');
	expect(live.at(-1)?.tone).toBe('danger');

	unmount(page);
	flushSync();
});

test('the trash arm posts the disposal and drops the receiver picker', async () => {
	const deletes: Array<{ url: string; body: unknown }> = [];
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (init?.method === 'DELETE') {
				deletes.push({ url, body: JSON.parse(String(init.body)) });
				return new Response(null, { status: 204 });
			}
			const answer = usersRoutes()[url];
			if (answer === undefined) {
				throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
			}
			return answer();
		})
	);
	const page = mountPage();
	await settle();

	click(deleteButton('grace@example.com'));
	const trashRadio = document.body.querySelector<HTMLInputElement>('input[value="trash"]');
	expect(trashRadio).not.toBeNull();
	trashRadio!.checked = true;
	trashRadio!.dispatchEvent(new Event('change', { bubbles: true }));
	flushSync();

	// The receiver picker belongs to the transfer arm alone.
	expect(document.body.querySelector('#user-delete-transfer')).toBeNull();

	click(modalButton('Continue'));
	click(modalButton('Delete permanently'));
	await settle();

	expect(deletes).toEqual([{ url: `/api/v1/admin/users/${grace.id}`, body: { dispose: 'trash' } }]);
	expect(rows()).toHaveLength(1);

	unmount(page);
	flushSync();
});

test('a refused disposal stays in the dialog with its mapped reason', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (init?.method === 'DELETE') {
				return jsonResponse(409, {
					code: 'USER_LAST_ADMIN',
					message: 'the last administrator cannot be deleted'
				});
			}
			const answer = usersRoutes()[url];
			if (answer === undefined) {
				throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
			}
			return answer();
		})
	);
	const page = mountPage();
	await settle();

	// The entry is offered on every row: the guard is the server's,
	// and the interface never fakes a disabled button for a rule it
	// cannot see.
	const entry = deleteButton('ada@example.com');
	expect(entry.disabled).toBe(false);
	click(entry);
	click(modalButton('Continue'));
	click(modalButton('Delete permanently'));
	await settle();

	const panel = document.body.querySelector<HTMLElement>('.modal__panel');
	expect(panel).not.toBeNull();
	expect(panel?.querySelector('[role="alert"]')?.textContent).toContain(
		"The instance's last administrator cannot be deleted."
	);
	// The row survives and no toast claims a deletion.
	expect(rows()).toHaveLength(2);
	expect(toasts()).toHaveLength(0);

	unmount(page);
	flushSync();
});

test('a row another administrator removed reconciles through the listing', async () => {
	let listed = 0;
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (init?.method === 'DELETE') {
				return jsonResponse(404, { code: 'USER_NOT_FOUND', message: 'no account' });
			}
			if (url === '/api/v1/admin/users') {
				listed += 1;
				return jsonResponse(200, { items: listed === 1 ? [adminUser, grace] : [adminUser] });
			}
			const answer = usersRoutes()[url];
			if (answer === undefined) {
				throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
			}
			return answer();
		})
	);
	const page = mountPage();
	await settle();
	expect(listed).toBe(1);

	click(deleteButton('grace@example.com'));
	click(modalButton('Continue'));
	click(modalButton('Delete permanently'));
	await settle();

	// The already-gone reading: the listing reconciles and the dialog
	// closes -- there is nothing left to confirm against.
	expect(listed).toBe(2);
	expect(rows()).toHaveLength(1);
	expect(document.body.querySelector('.modal__panel')).toBeNull();
	expect(toasts()).toHaveLength(0);

	unmount(page);
	flushSync();
});
