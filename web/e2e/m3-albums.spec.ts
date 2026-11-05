// The M3c albums and sharing story of the web app, walked end to end
// against the real compose stack. scripts/acceptance_m3c.sh chains
// the M3b acceptance (which itself chains the whole M1/M2/M3a line)
// and then starts this spec's own fresh stack; the spec itself only
// talks to http://localhost:8244.
//
// THE STACK ARRANGEMENT (the honest reuse path, documented): the
// stack is fresh (its database volume was removed before it started)
// but the models ride the SAME storage bind mount the chained
// acceptances downloaded into -- ./lumemor persists across the
// script's stack lifetimes, and the download service's
// already-verified skip re-registers the weights without re-fetching
// them. The bootstrap scene therefore triggers the default group's
// download through the admin API (the albums story has no download
// surface of its own): warm, it is a digest re-verification; cold
// (CI's first run), it is the real transfer, bounded like the M3
// model story's cold bound.
//
// THE TWO-ACCOUNT DANCE (the sharing story's shape): the first-run
// administrator owns the library and the album; a member account,
// created through the admin API in the suite's bootstrap, is the
// sharee. The story's scenes sign each account in through the login
// form in SEPARATE browser contexts -- the admin's storage state
// first, then the member's own sign-in scene (the two sessions never
// share a context) -- and the member's scenes assert everything the
// share grants: the wall's shared section, the read-only detail, the
// timeline's inclusion of the shared items, and the viewer drawer's
// identity-aware read-only state (the M3c identity seam's browser
// proof).
//
// THE SCENES (the M3c sharing surface, in order):
//
//  1. The administrator signs in through the form and lands on the
//     empty albums wall.
//  2. The library bootstrap, all through the API (the M2 story's
//     seeding pattern): the model group's download run is driven to
//     succeeded and two photos upload, converging to ready timeline
//     items.
//  3. The create flow: the wall's "New album" dialog names a virtual
//     album whose card appears in the "Your albums" section.
//  4. The curation handoff: the detail's "Add photos" lands on the
//     timeline with the selection mode armed (?album=), both photos
//     are selected, the batch bar's add-to-album dialog submits, and
//     the album's grid holds the two photos.
//  5. The share flow: the share dialog offers the member (the
//     caller's own account filtered out -- the identity seam's self
//     filter) and the grant lands the member's row.
//  6. The member perspective: the member signs in through their own
//     form, the wall's "Shared with you" section carries the album,
//     the shared detail renders read-only (no write affordance, the
//     leave affordance present), and the member's timeline includes
//     the shared items (the union's shared arm).
//  7. The viewer's identity-aware drawer: a shared photo's info
//     drawer renders the reads and the per-user favorite without
//     any owner-only write affordance.
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story --
// see playwright.config.ts). Scenes 1-5 share the admin's browser
// session (scene 1 signs in and saves the storage state); scene 6
// signs the member in through a fresh context and saves their own;
// scene 7 replays the member's. Each scene navigates explicitly to
// its starting route, so a failure report names the exact scene that
// broke.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's two accounts: the first-run administrator (the setup
// API's, the album's owner) and the member the administrator
// provisions through the admin API in the suite's bootstrap. The
// stack is always fresh, so the fixed identities are deterministic.
const ADMIN_EMAIL = 'admin@m3c-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';
const MEMBER_EMAIL = 'grace@m3c-e2e.invalid';
const MEMBER_PASSWORD = 'correct-horse-battery';

// The two seeded photos: the face portrait and its EXIF-orientation
// twin (a known-good same-person pair; the albums story needs no
// face structure, only two deterministic JPEGs whose timeline order
// is pinned by their taken-at stamps).
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));
const PHOTO_A = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait.jpg');
const PHOTO_B = path.join(
	repoRoot,
	'server/internal/ml/testdata/face-portrait-orientation6.jpg'
);
const PHOTO_A_NAME = 'IMG_M3C_0001.jpg';
const PHOTO_B_NAME = 'IMG_M3C_0002.jpg';

// The story's album title.
const ALBUM_TITLE = 'Coastal week 2026';

// The download bound: the cold upstream transfer of the group (about
// 351 MB) is minutes, the warm re-verification is seconds; both
// allow the cold bound.
const DOWNLOAD_TIMEOUT_MS = 900_000;

// The storage-state files scene 1 (the admin) and scene 6 (the
// member) write and the later scenes replay (git-ignored, recreated
// per run).
const adminStatePath = `${specDir}/.auth/m3c-admin-state.json`;
const memberStatePath = `${specDir}/.auth/m3c-member-state.json`;

// The created album's id, captured by the create scene through the
// API so the later scenes can deep-link deterministically.
let albumId = '';

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance; the
	// signed-in admin then provisions the member account through the
	// admin surface (the two-account dance's setup). The browser
	// scenes then only exercise the web app's own surfaces.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'M3c E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);

	const login = await api.post('/api/v1/auth/login', {
		data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(login.status()).toBe(200);

	const member = await api.post('/api/v1/admin/users', {
		data: { name: 'Grace Hopper', email: MEMBER_EMAIL, password: MEMBER_PASSWORD }
	});
	expect(member.status()).toBe(201);
	await api.dispose();
});

test('scene 1: the administrator signs in and lands on the empty albums wall', async ({
	page
}) => {
	await page.goto('/login?redirectTo=%2Falbums');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();

	// The post-login target round-trips onto the albums wall.
	await expect(page).toHaveURL(/\/albums$/);
	await expect(page).toHaveTitle('Albums · lumemor');

	// The empty state carries the honest copy and the create action.
	await expect(
		page.getByText('No albums yet. Create one to gather photos around a moment.')
	).toBeVisible();
	await expect(page.getByRole('button', { name: 'New album', exact: true })).toBeVisible();

	// Hand the admin session to every later admin scene: the storage
	// state rides the login cookie (the fresh contexts replay it).
	await page.context().storageState({ path: adminStatePath });
});

// The admin half of the story: every scene starts from a fresh
// context that replays the admin's login cookie scene 1 saved. The
// serial mode guarantees the state file exists by the time these
// scenes run.
test.describe('the administrator scenes', () => {
	test.use({ storageState: adminStatePath });

	test('scene 2: the library bootstrap seeds two photos through the API', async ({ page }) => {
		test.setTimeout(DOWNLOAD_TIMEOUT_MS + 120_000);

		// The model group's download first (the albums story has no
		// download surface of its own; the admin API is the M2 story's
		// seeding pattern -- the request rides the page's own context
		// and its login cookie). Warm mount or cold, the run must
		// reach succeeded.
		const download = await page.request.post('/api/v1/admin/models/download');
		expect(download.status()).toBe(202);
		await expect
			.poll(
				async () => {
					const jobs = await page.request.get('/api/v1/admin/jobs');
					expect(jobs.status()).toBe(200);
					const body = (await jobs.json()) as {
						jobs?: Array<{ kind?: string; state?: string }>;
					};
					const row = body.jobs?.find((job) => job.kind === 'model_download');
					return row?.state ?? 'missing';
				},
				{ timeout: DOWNLOAD_TIMEOUT_MS, intervals: [1000, 2000, 5000] }
			)
			.toBe('succeeded');

		// The two fixtures upload through the asset API (the same
		// seeding the acceptance script applies).
		const seeds: Array<{ file: string; name: string; takenAt: string }> = [
			{ file: PHOTO_A, name: PHOTO_A_NAME, takenAt: '2026-05-01T12:00:00Z' },
			{ file: PHOTO_B, name: PHOTO_B_NAME, takenAt: '2026-05-02T12:00:00Z' }
		];
		for (const seed of seeds) {
			const upload = await page.request.post('/api/v1/assets', {
				multipart: {
					deviceAssetId: `m3c-e2e-${seed.name}`,
					fileName: seed.name,
					deviceCreatedAt: seed.takenAt,
					deviceTimeZone: 'UTC',
					photo: {
						name: seed.name,
						mimeType: 'image/jpeg',
						buffer: await readFile(seed.file)
					}
				}
			});
			expect(upload.status()).toBe(201);
		}

		// The convergence: both assets reach the ready state and the
		// timeline lists them (the wall's tile gate needs readiness).
		type TimelineBody = { items?: Array<{ id: string; isReady: boolean }> };
		await expect
			.poll(
				async () => {
					const timeline = await page.request.get('/api/v1/assets/timeline');
					expect(timeline.status()).toBe(200);
					const body = (await timeline.json()) as TimelineBody;
					return (body.items ?? []).filter((item) => item.isReady).length;
				},
				{ timeout: 120_000, intervals: [1000, 2000, 5000] }
			)
			.toBe(2);
	});

	test('scene 3: the create dialog names a virtual album whose card appears', async ({ page }) => {
		await page.goto('/albums');
		await expect(page).toHaveTitle('Albums · lumemor');

		await page.getByRole('button', { name: 'New album', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'New album', exact: true });
		await expect(dialog).toBeVisible();
		await dialog.getByLabel('Title').fill(ALBUM_TITLE);
		await dialog.getByRole('button', { name: 'Create album', exact: true }).click();
		await expect(dialog).toHaveCount(0);

		// The SSE album.created echo converges the wall: the card
		// lands in the "Your albums" section with the honest count.
		const card = page.locator('a.album-card').filter({ hasText: ALBUM_TITLE });
		await expect(card).toBeVisible();
		await expect(page.getByRole('heading', { level: 1, name: 'Your albums' })).toBeVisible();
		await expect(card).toContainText('0 photos');

		// The created album's id, for the later scenes' deep links.
		albumId = (await card.getAttribute('href'))?.replace('/albums/', '') ?? '';
		expect(albumId).not.toBe('');
	});

	test('scene 4: the add-photos handoff curates both photos into the album', async ({ page }) => {
		await page.goto(`/albums/${albumId}`);
		// The detail's header title (the wall's empty state renders the
		// album title in its own h1 too -- the class scopes the assert).
		await expect(page.locator('h1.album-detail__title')).toHaveText(ALBUM_TITLE);
		await expect(page.getByText('This album has no photos yet. Add some from your timeline.')).toBeVisible();

		// The handoff: the detail's add-photos action lands on the
		// timeline with the selection mode armed (?album=).
		await page.getByRole('button', { name: 'Add photos', exact: true }).click();
		await expect(page).toHaveURL(new RegExp(`/timeline\\?album=${albumId}$`));

		// Both tiles join the selection through their checkboxes.
		await page
			.getByRole('checkbox', { name: `Select: ${PHOTO_A_NAME}` })
			.click();
		await page
			.getByRole('checkbox', { name: `Select: ${PHOTO_B_NAME}` })
			.click();

		// The batch bar rises with the honest count and the add
		// affordance leads it.
		const bar = page.getByRole('toolbar');
		await expect(bar).toContainText('2 selected');
		await bar.getByRole('button', { name: 'Add to album', exact: true }).click();

		// The dialog preselects the handoff's album (the detail's own
		// flow), the submit lands, and the honest report answers.
		const dialog = page.getByRole('dialog', { name: 'Add to album', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog.locator('#add-album-target')).toContainText(ALBUM_TITLE);
		await dialog.getByRole('button', { name: 'Add photos', exact: true }).click();
		await expect(dialog.getByRole('status')).toHaveText('Added 2, skipped 0.');
		await dialog.getByRole('button', { name: 'Done', exact: true }).click();
		await expect(dialog).toHaveCount(0);

		// The album's grid holds the two photos (the membership
		// convergence restarts the walk -- the fresh answer owns it).
		await page.goto(`/albums/${albumId}`);
		await expect(page.locator('.tile')).toHaveCount(2);
		await expect(page.getByRole('link', { name: PHOTO_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_B_NAME, exact: true })).toBeVisible();
		await expect(page.getByText('2 photos')).toBeVisible();
	});

	test('scene 5: the share dialog grants the member the view', async ({ page }) => {
		await page.goto(`/albums/${albumId}`);
		await page.getByRole('button', { name: 'Share…', exact: true }).click();

		const dialog = page.getByRole('dialog', { name: 'Share this album', exact: true });
		await expect(dialog).toBeVisible();

		// The identity seam's self filter: the caller's own account
		// never appears among the pick options; the member does.
		const options = dialog.locator('#share-dialog-target option');
		await expect(options).toHaveCount(1);
		await expect(options).toHaveText(MEMBER_EMAIL);

		await dialog.getByRole('button', { name: 'Share', exact: true }).click();
		await expect(dialog).toContainText(MEMBER_EMAIL);
	});
});

// The member half of the story: scene 6 signs the member in through
// their own form (a fresh context -- the two accounts never share a
// browser session) and saves their storage state; scene 7 replays it.
test('scene 6: the member signs in and sees the shared album read-only', async ({ browser }) => {
	// The config's use-locale does not reach manually created
	// contexts; pin it so the asserted strings render in English.
	const context = await browser.newContext({ locale: 'en-US', timezoneId: 'UTC' });
	const page = await context.newPage();

	await page.goto('/login');
	await page.getByLabel('Email').fill(MEMBER_EMAIL);
	await page.getByLabel('Password').fill(MEMBER_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);

	// The wall's "Shared with you" section carries the album with the
	// owner attribution.
	await page.goto('/albums');
	const card = page.locator('a.album-card').filter({ hasText: ALBUM_TITLE });
	await expect(page.getByRole('heading', { level: 2, name: 'Shared with you' })).toBeVisible();
	await expect(card).toBeVisible();
	await expect(card).toContainText(`Shared by ${ADMIN_EMAIL}`);
	await expect(card).toContainText('2 photos');

	// The shared detail reads as viewer: no write affordance ever
	// renders, the timeline-exclusion toggle and the leave
	// affordance do.
	await card.click();
	await expect(page).toHaveURL(new RegExp(`/albums/${albumId}$`));
	await expect(page.locator('h1.album-detail__title')).toHaveText(ALBUM_TITLE);
	for (const label of ['Rename', 'Set cover…', 'Lock', 'Share…', 'Add photos', 'Delete album']) {
		await expect(page.getByRole('button', { name: label, exact: true })).toHaveCount(0);
	}
	await expect(page.getByRole('button', { name: 'Keep out of timeline', exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Leave album', exact: true })).toBeVisible();
	await expect(page.locator('.tile')).toHaveCount(2);

	// The member's timeline includes the shared items (the union's
	// shared arm -- the member owns nothing of their own).
	await page.goto('/timeline');
	await expect(page.getByRole('link', { name: PHOTO_A_NAME, exact: true })).toBeVisible();
	await expect(page.getByRole('link', { name: PHOTO_B_NAME, exact: true })).toBeVisible();
	await expect(page.locator('.tile')).toHaveCount(2);

	// Hand the member session to scene 7.
	await context.storageState({ path: memberStatePath });
	await context.close();
});

test.describe('the member viewer scenes', () => {
	test.use({ storageState: memberStatePath });

	test("scene 7: the viewer's drawer keeps the reads and the per-user favorite, never the owner-only writes", async ({
		page
	}) => {
		// A shared photo's viewer, deep-linked from the album detail.
		await page.goto(`/albums/${albumId}`);
		await page.getByRole('link', { name: PHOTO_A_NAME, exact: true }).click();
		await expect(page).toHaveURL(new RegExp(`/a/[0-9a-f-]{36}$`));
		await expect(page.getByText(PHOTO_A_NAME)).toBeVisible();

		// The info drawer plus the action bar (the R2-3 rework: the
		// drawer is metadata-only, the per-user favorite rides the
		// bar's icon button): the reads render (the file facts, the
		// download affordance) and the favorite stays, while every
		// owner-only write affordance is absent (the identity seam's
		// browser proof -- the session's userId is not the asset's
		// owner).
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = page.getByRole('complementary', { name: 'Photo information' });
		await expect(drawer).toBeVisible();
		await expect(drawer.getByRole('link', { name: 'Download the original' })).toBeVisible();
		const favorite = page.getByRole('button', { name: 'Favorite', exact: true });
		await expect(favorite).toBeVisible();
		for (const label of [
			'Edit',
			'Hide',
			'Unhide',
			'Trash',
			'Restore',
			'Delete permanently'
		]) {
			// The owner-only writes never render in either surface:
			// no trash/restore in the BAR (the drawer never carried a
			// curation button since the rework), no description edit.
			await expect(page.getByRole('button', { name: label, exact: true })).toHaveCount(0);
		}

		// The per-user favorite works for the viewer: the flag flips
		// without a page reload (the optimistic per-user state).
		await favorite.click();
		await expect(page.getByRole('button', { name: 'Unfavorite', exact: true })).toBeVisible();
	});
});
