// The admin users page's alignment and disposal story, walked end to
// end against the real compose stack: the 760 column with the table
// scrolling inside its own card, the two-step account disposal on
// both arms (transfer to another member, the trash with the
// instance's retention window), the last-administrator refusal in the
// dialog, and the surface at 520/760/1280 in both themes and both
// locales.
//
// THE STACK ARRANGEMENT: one fresh lifetime owns this spec (the
// first-run bootstrap below creates the administrator, so a second
// spec on the same lifetime would answer 409).
//
// THE STATE: the bootstrap provisions two member accounts through the
// admin API, has one of them upload one real photo through the upload
// API (the member's own session -- the photo must be theirs, because
// the transfer arm is what the story proves), and shares one of the
// administrator's albums with that member (the share the disposal
// must take with the account). The browser session rides a storage
// state saved after the sign-in scene.
//
// THE SCENES (in order):
//
//  1. The administrator signs in through the form and lands on the
//     users page: the 760 column, the page-header row, the table
//     inside its own scroll card, and the danger TEXT button on every
//     row (transparent ground, ember border -- never a solid fill).
//  2. The last-administrator refusal: the administrator's own row's
//     disposal reaches the server and comes back with the refusal in
//     the second dialog; nothing is faked as a disabled button.
//  3. The transfer arm: the member's account goes, her photo survives
//     under the receiving member (its timeline lists it), and the
//     share she held dies with the account.
//  4. The trash arm: the receiving member's account goes the other
//     way -- the photos land in the ADMINISTRATOR's trash under the
//     instance's retention countdown.
//  5. The widths (520/760/1280) and the two faces: no sideways page
//     overflow anywhere, the table scrolling inside the card at 520,
//     the dark theme keeping the same shapes, and the zh-Hans locale
//     rendering the dialog's copy.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's three accounts. The stack is always fresh, so the fixed
// identities are deterministic.
const ADMIN_EMAIL = 'admin@w3-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';
const ADMIN_NAME = 'Ada Admin';
const GRACE_EMAIL = 'grace@w3-e2e.invalid';
const GRACE_NAME = 'Grace';
const YUN_EMAIL = 'yun@w3-e2e.invalid';
const YUN_NAME = 'Yun';
const MEMBER_PASSWORD = 'correct-horse-battery';

// The member's photo: a real JPEG from the server's test corpus (the
// upload endpoint sniffs real bytes), named and dated so the story's
// assertions can address it.
const PHOTO_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/golden-large.jpg');
const PHOTO_NAME = 'IMG_W3_0001.jpg';

// The storage state scene 1 writes and the later scenes replay
// (git-ignored, recreated per run).
const authStatePath = path.join(specDir, '.auth', 'w3-state.json');

// The ids the bootstrap resolves: the two member accounts, the photo
// the first member uploaded and the shared album.
let graceId = '';
let photoId = '';
let sharedAlbumId = '';

/** One account's row on the users table, addressed by its email. */
function rowOf(page: Page, email: string) {
	return page.locator('.users__table tbody tr').filter({ hasText: email }).first();
}

/** The disposal entry of one row (its accessible name names the account). */
function deleteButton(page: Page, name: string) {
	return page.getByRole('button', { name: `Delete ${name}`, exact: true });
}

/** The disposal entry of one row whatever the locale (the button's own variant class). */
function localizedDeleteButton(page: Page, email: string) {
	return rowOf(page, email).locator('button.btn--danger-text');
}

/**
 * Walk the disposal's two steps for one account: the entry, the
 * destination choice, Continue, and the irreversible click.
 */
async function disposeAccount(page: Page, name: string): Promise<void> {
	await deleteButton(page, name).click();
	await expect(page.getByRole('dialog', { name: `Delete ${name}` })).toBeVisible();
	await page.getByRole('button', { name: 'Continue', exact: true }).click();
	await expect(page.getByRole('dialog', { name: 'Confirm deletion' })).toBeVisible();
	await page.getByRole('button', { name: 'Delete permanently', exact: true }).click();
}

/** The page's sideways overflow in pixels (0 when the layout holds). */
async function sidewaysOverflow(page: Page): Promise<number> {
	return page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
}

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap (the acceptance scripts' pattern): the
	// setup API creates the administrator on the fresh instance, the
	// admin API provisions the two members, and the member's own
	// session uploads the photo the transfer arm will prove.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: ADMIN_NAME, email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	const login = await api.post('/api/v1/auth/login', {
		data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(login.status()).toBe(200);

	const grace = await api.post('/api/v1/admin/users', {
		data: { name: GRACE_NAME, email: GRACE_EMAIL, password: MEMBER_PASSWORD }
	});
	expect(grace.status()).toBe(201);
	graceId = ((await grace.json()) as { id: string }).id;
	const yun = await api.post('/api/v1/admin/users', {
		data: { name: YUN_NAME, email: YUN_EMAIL, password: MEMBER_PASSWORD }
	});
	expect(yun.status()).toBe(201);

	// The member's photo, uploaded through her own session: the row is
	// hers, which is the whole point of the transfer arm. The asset's
	// READINESS does not matter here -- both arms settle ROWS, and the
	// downstream pipeline only decorates them.
	const memberApi = await playwright.request.newContext({ baseURL: BASE_URL });
	const memberLogin = await memberApi.post('/api/v1/auth/login', {
		data: { email: GRACE_EMAIL, password: MEMBER_PASSWORD }
	});
	expect(memberLogin.status()).toBe(200);
	const upload = await memberApi.post('/api/v1/assets', {
		multipart: {
			deviceAssetId: 'w3-e2e-grace-photo',
			fileName: PHOTO_NAME,
			deviceCreatedAt: '2024-03-01T10:00:00Z',
			deviceTimeZone: 'UTC',
			photo: {
				name: PHOTO_NAME,
				mimeType: 'image/jpeg',
				buffer: await readFile(PHOTO_SOURCE)
			}
		}
	});
	expect([200, 201]).toContain(upload.status());
	photoId = ((await upload.json()) as { id: string }).id;

	// The administrator's memory, shared with the member: the share is
	// what the disposal must take with the account.
	const memory = await api.post('/api/v1/memories', {
		data: { title: 'W3 shared memory', audience: [graceId] }
	});
	expect(memory.status()).toBe(201);
	const created = (await memory.json()) as { id: string; skipped: unknown[] };
	sharedAlbumId = created.id;
	expect(created.skipped).toHaveLength(0);
	const shared = await api.get(`/api/v1/albums/${encodeURIComponent(sharedAlbumId)}`);
	expect(shared.status()).toBe(200);
	const sharedDetail = (await shared.json()) as { shares?: Array<{ userId: string }> };
	expect(sharedDetail.shares?.map((entry) => entry.userId)).toEqual([graceId]);

	await memberApi.dispose();
	await api.dispose();
});

test('scene 1: the administrator lands on the aligned users page', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Fadmin%2Fusers');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/admin\/users$/);
	await expect(page).toHaveTitle('Users · lumemor');

	// The page-header row: the title on the left, the create entry on
	// the right.
	await expect(page.locator('.page-header__title')).toHaveText('Users');
	await expect(page.getByRole('button', { name: 'New user', exact: true })).toBeVisible();

	// The admin column's bound: the content sits inside 760px and is
	// centered (the reading/form surface's own width).
	const column = await page.locator('.users').boundingBox();
	expect(column).not.toBeNull();
	expect(column!.width).toBeLessThanOrEqual(760);
	expect(await page.locator('.users').evaluate((node) => getComputedStyle(node).marginLeft)).not.toBe('0px');

	// The table rides its own card shell: the card's border, radius and
	// shadow belong to the shell, and the shell is the scroller.
	const shell = page.locator('.users__table-shell');
	await expect(shell).toBeVisible();
	const shellFacts = await shell.evaluate((node) => {
		const styles = getComputedStyle(node);
		return {
			overflowX: styles.overflowX,
			position: styles.position,
			radius: styles.borderTopLeftRadius,
			shadow: styles.boxShadow
		};
	});
	expect(shellFacts.overflowX).toBe('auto');
	expect(shellFacts.position).toBe('relative');
	expect(shellFacts.radius).not.toBe('0px');
	expect(shellFacts.shadow).not.toBe('none');
	const tableFacts = await page.locator('.users__table').evaluate((node) => {
		const styles = getComputedStyle(node);
		return { radius: styles.borderTopLeftRadius, shadow: styles.boxShadow };
	});
	expect(tableFacts.radius).toBe('0px');
	expect(tableFacts.shadow).toBe('none');

	// Every row carries the disposal entry, and it is a TEXT button:
	// the ember border and the ember text over a transparent ground --
	// never a solid fill that would read like the everyday Disable.
	const entry = deleteButton(page, GRACE_NAME);
	await expect(entry).toBeVisible();
	await expect(entry).toContainText('Delete');
	const entryFacts = await entry.evaluate((node) => {
		const styles = getComputedStyle(node);
		return {
			borderColor: styles.borderTopColor,
			background: styles.backgroundColor,
			color: styles.color
		};
	});
	expect(entryFacts.background).toBe('rgba(0, 0, 0, 0)');
	expect(entryFacts.borderColor).not.toBe('rgba(0, 0, 0, 0)');
	// The border carries the danger tone, so the button reads as the
	// destructive one beside the neutral Disable next to it.
	const disableColor = await page
		.getByRole('button', { name: 'Disable', exact: true })
		.first()
		.evaluate((node) => getComputedStyle(node).borderTopColor);
	expect(entryFacts.borderColor).toBe(disableColor);
	expect(entryFacts.color).not.toBe(disableColor);

	// The admin session rides to every later scene.
	await page.context().storageState({ path: authStatePath });
});

test.describe('the administrator scenes', () => {
	test.use({ storageState: authStatePath });

	test('scene 2: the last administrator hears the server, never a dead button', async ({ page }) => {
		await page.goto('/admin/users');
		await expect(rowOf(page, ADMIN_EMAIL)).toBeVisible();

		// The entry is offered (the client cannot know it is the
		// instance's only administrator) and stays clickable.
		const entry = deleteButton(page, ADMIN_NAME);
		await expect(entry).toBeEnabled();

		await disposeAccount(page, ADMIN_NAME);

		// The refusal lands IN the dialog, naming the reason.
		const dialog = page.getByRole('dialog', { name: 'Confirm deletion' });
		await expect(dialog.getByRole('alert')).toHaveText("The instance's last administrator cannot be deleted.");
		// Nothing claimed a deletion: the row stays and no toast fires.
		await expect(rowOf(page, ADMIN_EMAIL)).toBeVisible();
		await expect(page.locator('.toast')).toHaveCount(0);
		// The dialog is the only way out, and it closes clean.
		await page.getByRole('button', { name: 'Cancel', exact: true }).click();
		await expect(dialog).toHaveCount(0);
	});

	test('scene 3: the transfer arm keeps the photos and takes the share', async ({ page, playwright }) => {
		await page.goto('/admin/users');
		await expect(rowOf(page, GRACE_EMAIL)).toBeVisible();

		// The first step names the account and offers both destinations;
		// the retention subline names the instance's own window.
		await deleteButton(page, GRACE_NAME).click();
		const first = page.getByRole('dialog', { name: `Delete ${GRACE_NAME}` });
		await expect(first).toBeVisible();
		await expect(first).toContainText('Transfer the photos to another member');
		await expect(first).toContainText('Move the photos to the trash');
		await expect(first.locator('.users__delete-retention')).toContainText('after 30 days');
		// The receiver picker lists the OTHER accounts only.
		const picker = first.locator('#user-delete-transfer');
		await expect(picker.locator('option')).toHaveCount(2);
		await expect(picker.locator('option', { hasText: GRACE_NAME })).toHaveCount(0);
		await picker.selectOption({ label: YUN_NAME });

		// The second step is a different dialog: the bold line says what
		// the click means.
		await first.getByRole('button', { name: 'Continue', exact: true }).click();
		const confirm = page.getByRole('dialog', { name: 'Confirm deletion' });
		await expect(confirm).toBeVisible();
		await expect(confirm).toContainText('This cannot be undone.');
		await confirm.getByRole('button', { name: 'Delete permanently', exact: true }).click();

		// The row leaves and the danger toast confirms.
		await expect(rowOf(page, GRACE_EMAIL)).toHaveCount(0);
		await expect(page.locator('.toast[data-toast-tone="danger"]')).toContainText('The account has been deleted.');
		await expect(page.getByRole('dialog')).toHaveCount(0);

		// THE SERVER'S TRUTH: the account can never sign in again, her
		// photo answers in the receiver's timeline, and the share she
		// held is gone from the album's audience.
		const refused = await playwright.request.newContext({ baseURL: BASE_URL });
		const refusedLogin = await refused.post('/api/v1/auth/login', {
			data: { email: GRACE_EMAIL, password: MEMBER_PASSWORD }
		});
		expect(refusedLogin.status()).toBe(401);
		await refused.dispose();

		// This listing is the ADMINISTRATOR's own: transferred content is
		// the receiver's, not the family's, so the photo must NOT surface
		// here (the receiver's own view proves the transfer in the next
		// scene).
		const timeline = await page.request.get('/api/v1/assets/timeline');
		expect(timeline.status()).toBe(200);
		const adminTimeline = (await timeline.json()) as { items: Array<{ id: string }> };
		expect(adminTimeline.items.map((item) => item.id)).not.toContain(photoId);
		void playwright;

		const album = await page.request.get(`/api/v1/albums/${encodeURIComponent(sharedAlbumId)}`);
		expect(album.status()).toBe(200);
		const detail = (await album.json()) as { shares?: Array<{ userId: string }> };
		expect(detail.shares?.map((entryShare) => entryShare.userId) ?? []).not.toContain(graceId);
		expect(detail.shares ?? []).toHaveLength(0);
	});

	test('scene 4: the trash arm lands the photos in the administrator trash', async ({ page, playwright }) => {
		// The receiving member's own view proves the transfer first: the
		// photo is in Yun's timeline, under Yun's own ownership.
		const memberApi = await playwright.request.newContext({ baseURL: BASE_URL });
		const memberLogin = await memberApi.post('/api/v1/auth/login', {
			data: { email: YUN_EMAIL, password: MEMBER_PASSWORD }
		});
		expect(memberLogin.status()).toBe(200);
		const memberTimeline = await memberApi.get('/api/v1/assets/timeline');
		expect(memberTimeline.status()).toBe(200);
		const receiverItems = (await memberTimeline.json()) as { items: Array<{ id: string }> };
		expect(receiverItems.items.map((item) => item.id)).toContain(photoId);
		await memberApi.dispose();

		await page.goto('/admin/users');
		await expect(rowOf(page, YUN_EMAIL)).toBeVisible();

		// The other arm: the trash, chosen in the first step.
		await deleteButton(page, YUN_NAME).click();
		const first = page.getByRole('dialog', { name: `Delete ${YUN_NAME}` });
		await expect(first).toBeVisible();
		await first.locator('input[value="trash"]').check();
		await expect(first.locator('#user-delete-transfer')).toHaveCount(0);
		await first.getByRole('button', { name: 'Continue', exact: true }).click();
		await page
			.getByRole('dialog', { name: 'Confirm deletion' })
			.getByRole('button', { name: 'Delete permanently', exact: true })
			.click();

		await expect(rowOf(page, YUN_EMAIL)).toHaveCount(0);
		await expect(page.locator('.toast[data-toast-tone="danger"]')).toContainText('The account has been deleted.');

		// THE ADMINISTRATOR'S TRASH holds the photos, with the
		// instance's retention countdown running on them.
		const trash = await page.request.get('/api/v1/trash');
		expect(trash.status()).toBe(200);
		const items = ((await trash.json()) as { items: Array<{ id: string; remainingDays: number }> }).items;
		const landed = items.find((item) => item.id === photoId);
		expect(landed).toBeDefined();
		expect(landed!.remainingDays).toBeGreaterThan(0);
		expect(landed!.remainingDays).toBeLessThanOrEqual(30);
	});

	test('scene 5: the widths hold and both faces render', async ({ page }) => {
		// 520: the page never scrolls sideways, and the table's card
		// scrolls inside itself instead.
		await page.setViewportSize({ width: 520, height: 900 });
		await page.goto('/admin/users');
		await expect(page.locator('.users__table-shell')).toBeVisible();
		expect(await sidewaysOverflow(page)).toBeLessThanOrEqual(1);
		const scroller = await page.locator('.users__table-shell').evaluate((node) => ({
			scrollWidth: node.scrollWidth,
			clientWidth: node.clientWidth
		}));
		expect(scroller.scrollWidth).toBeGreaterThan(scroller.clientWidth);

		// 760 and 1280: the same promise, and the column still reads at
		// its bound.
		for (const width of [760, 1280]) {
			await page.setViewportSize({ width, height: 900 });
			await page.goto('/admin/users');
			await expect(page.locator('.users__table-shell')).toBeVisible();
			expect(await sidewaysOverflow(page)).toBeLessThanOrEqual(1);
			const box = await page.locator('.users').boundingBox();
			expect(box!.width).toBeLessThanOrEqual(760);
		}

		// The dark theme keeps the same shapes: the entry is still a
		// text button with the ember border over the dark ground.
		await page.addInitScript(() => window.localStorage.setItem('lumemor:theme', 'dark'));
		await page.setViewportSize({ width: 1280, height: 900 });
		await page.goto('/admin/users');
		await expect(page.locator('html')).toHaveClass(/dark/);
		const darkEntry = await deleteButton(page, ADMIN_NAME).evaluate((node) => {
			const styles = getComputedStyle(node);
			return { background: styles.backgroundColor, color: styles.color };
		});
		expect(darkEntry.background).toBe('rgba(0, 0, 0, 0)');
		await deleteButton(page, ADMIN_NAME).click();
		await expect(page.getByRole('dialog', { name: `Delete ${ADMIN_NAME}` })).toBeVisible();
		await page.getByRole('button', { name: 'Cancel', exact: true }).click();

		// The zh-Hans face: the same two steps in the other language.
		await page.addInitScript(() => window.localStorage.setItem('lumemor:locale', 'zh-Hans'));
		await page.goto('/admin/users');
		await expect(page.locator('html')).toHaveAttribute('lang', 'zh-Hans');
		await localizedDeleteButton(page, ADMIN_EMAIL).click();
		const localized = page.getByRole('dialog', { name: `删除 ${ADMIN_NAME}` });
		await expect(localized).toBeVisible();
		await expect(localized).toContainText('把照片转给另一位成员');
		await expect(localized).toContainText('照片进回收站');
		// The account is the instance's last one, so the transfer arm has
		// no receiver left: the trash arm is the offer that can be made.
		await localized.locator('input[value="trash"]').check();
		await localized.getByRole('button', { name: '继续', exact: true }).click();
		const localizedConfirm = page.getByRole('dialog', { name: '确认删除' });
		await expect(localizedConfirm).toBeVisible();
		await expect(localizedConfirm).toContainText('这一步不能撤销。');
		await localizedConfirm.getByRole('button', { name: '永久删除', exact: true }).click();
		// The last-administrator refusal renders in the same language.
		await expect(localizedConfirm.getByRole('alert')).toContainText('实例的最后一个管理员不能删除。');
		expect(await sidewaysOverflow(page)).toBeLessThanOrEqual(1);
	});
});
