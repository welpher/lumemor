// The M3d search and places story of the web app, walked end to end
// against the real compose stack. scripts/acceptance_m3d.sh chains
// the M3c acceptance (which itself chains the whole M1/M2/M3a/M3b
// line) and then starts this spec's own fresh stack; the spec itself
// only talks to http://localhost:8244.
//
// THE STACK ARRANGEMENT (the honest reuse path, the m3-albums
// arrangement): the stack is fresh (its database volume was removed
// before it started) but the models ride the SAME storage bind mount
// the chained acceptances downloaded into -- ./lumemor persists
// across the script's stack lifetimes, and the download service's
// already-verified skip re-registers the weights without re-fetching
// them. The bootstrap scene therefore triggers the default group's
// download through the admin API (the search story has no download
// surface of its own): warm, it is a digest re-verification; cold
// (CI's first run), it is the real transfer, bounded like the M3
// model story's cold bound. The group now carries FIVE models -- the
// semantic arm's text tower among them -- and the download's empty-
// slot auto-activation makes it ACTIVE, so the search page's Chinese
// query exercises BOTH merged paths from the first scene.
//
// THE SEEDED LIBRARY (the admin's own, three photos through the
// asset API): the Beijing GPS photo (the EXIF-bearing golden
// derivative whose coordinates resolve to Beijing -- the places
// page's city), the Chinese OCR photo (the rendered text image whose
// extracted text carries 海边 -- the text path's hit), and the face
// portrait (the third wall member). The 海边 query hits the OCR text
// through the TEXT path and every photo through the SEMANTIC path,
// so the fused page answers all three with the OCR photo first --
// the merge rule's browser proof (an asset both paths hit outranks
// the single-path hits around it, and appears exactly once).
//
// THE SCENES (the M3d search surface, in order):
//
//  1. The administrator signs in through the form and lands on the
//     timeline.
//  2. The library bootstrap, all through the API (the M2 story's
//     seeding pattern): the model group's download run is driven to
//     succeeded and the three photos upload, converging to ready
//     timeline items.
//  3. The topbar's search box: typing 海边 and committing Enter
//     navigates to /search?query=… -- the flat wall answers in RANK
//     order (the OCR photo first: both paths hit it) with NO month
//     or day headers (grouping would lie about the rank order).
//  4. The filter browse: the bare /search (no query) renders the
//     timeline's exact month-grouped wall, and a facet chip (the
//     Places · Beijing chip) toggles: the URL gains placeIds, the
//     wall re-executes to the one Beijing photo, the chip stays
//     pressed.
//  5. The places page: the city grid renders the Beijing card (the
//     locale name, the country label, the honest count), and the
//     card deep-links into the filtered browse (/search?placeIds=)
//     with the month-grouped wall.
//  6. The tags flow on the timeline: the batch bar's "Add tag"
//     opens the dialog, the "New tag" path names the tag, the
//     honest {added, skipped} report lands, Done closes and clears
//     the selection.
//  7. The tag joins the search vocabulary: the browse chips row
//     carries the new tag with its honest count.
//  8. The browse wall curates the per-user subset (the search
//     walls' curation ruling: BOTH walls carry [favorite, tag] for
//     EVERY role): the bar offers the favorite toggle and the tag
//     affordance and never the ownership actions; the favorite
//     batch lands, the bar's success arm clears, and a reload
//     answers the server-held flip.
//  9. The query wall carries the same subset through its own
//     organ: the flat wall's check entry, the same subset bar, and
//     the tag flow riding the opener over the captured id (the
//     existing-tag path, the scene-6 tag).
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story --
// see playwright.config.ts). Scene 1 signs in and saves the storage
// state; every later scene replays it. Each scene navigates
// explicitly to its starting route, so a failure report names the
// exact scene that broke.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's single account: the first-run administrator (the setup
// API's). The stack is always fresh, so the fixed identity is
// deterministic.
const ADMIN_EMAIL = 'admin@m3d-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The three seeded photos: the Beijing GPS photo (the golden
// derivative whose EXIF coordinates resolve to Beijing), the Chinese
// OCR text image (whose extracted text carries 海边), and the face
// portrait. The GPS photo's taken-at comes from its EXIF
// (2026:06:20 10:15:30 +05:30); the other two ride their upload
// stamps.
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));
const PHOTO_GPS = path.join(repoRoot, 'server/internal/jobs/testdata/gps-beijing.jpg');
const PHOTO_OCR = path.join(repoRoot, 'server/internal/ml/testdata/ocr-zh-text.png');
const PHOTO_FACE = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait.jpg');
const PHOTO_GPS_NAME = 'IMG_M3D_GPS.jpg';
const PHOTO_OCR_NAME = 'IMG_M3D_OCR.png';
const PHOTO_FACE_NAME = 'IMG_M3D_FACE.jpg';

// The story's Chinese query: the acceptance chain's own line -- a
// search for 海边 must answer photos. The OCR photo's extracted text
// carries it, so the text path hits it; the semantic path ranks the
// whole library; the fused page answers all three photos with the
// OCR photo first.
const QUERY = '海边';

// The tags flow's new tag name.
const TAG_NAME = 'seaside';

// The download bound: the cold upstream transfer of the group (about
// 771 MB with the text tower) is minutes, the warm re-verification
// is seconds; both allow the cold bound.
const DOWNLOAD_TIMEOUT_MS = 900_000;

// The storage-state file scene 1 writes and every later scene
// replays (git-ignored, recreated per run).
const adminStatePath = `${specDir}/.auth/m3d-admin-state.json`;

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance. The
	// browser scenes then only exercise the web app's own surfaces.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'M3d E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

test('scene 1: the administrator signs in and lands on the timeline', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);
	await expect(page).toHaveTitle('Timeline · lumemor');

	// Hand the admin session to every later scene: the storage state
	// rides the login cookie (the fresh contexts replay it).
	await page.context().storageState({ path: adminStatePath });
});

// The signed-in half of the story: every scene starts from a fresh
// context that replays the admin's login cookie scene 1 saved. The
// serial mode guarantees the state file exists by the time these
// scenes run.
test.describe('the signed-in scenes', () => {
	test.use({ storageState: adminStatePath });

	test('scene 2: the library bootstrap seeds three photos through the API', async ({ page }) => {
		test.setTimeout(DOWNLOAD_TIMEOUT_MS + 120_000);

		// The model group's download first (the search story has no
		// download surface of its own; the admin API is the M2 story's
		// seeding pattern -- the request rides the page's own context
		// and its login cookie). Warm mount or cold, the run must
		// reach succeeded. The five-model group (the text tower among
		// them) auto-activates the semantic arm.
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

		// The three fixtures upload through the asset API (the same
		// seeding the acceptance script applies).
		const seeds: Array<{ file: string; name: string; mime: string; takenAt: string }> = [
			{ file: PHOTO_GPS, name: PHOTO_GPS_NAME, mime: 'image/jpeg', takenAt: '2026-06-20T12:00:00Z' },
			{ file: PHOTO_OCR, name: PHOTO_OCR_NAME, mime: 'image/png', takenAt: '2026-07-15T12:00:00Z' },
			{ file: PHOTO_FACE, name: PHOTO_FACE_NAME, mime: 'image/jpeg', takenAt: '2026-05-10T12:00:00Z' }
		];
		for (const seed of seeds) {
			const upload = await page.request.post('/api/v1/assets', {
				multipart: {
					deviceAssetId: `m3d-e2e-${seed.name}`,
					fileName: seed.name,
					deviceCreatedAt: seed.takenAt,
					deviceTimeZone: 'UTC',
					photo: {
						name: seed.name,
						mimeType: seed.mime,
						buffer: await readFile(seed.file)
					}
				}
			});
			expect(upload.status()).toBe(201);
		}

		// The convergence: all three assets reach the ready state
		// (the OCR and embedding pipelines included -- the search's
		// two paths need their products) and the timeline lists them.
		type TimelineBody = { items?: Array<{ id: string; isReady: boolean }> };
		await expect
			.poll(
				async () => {
					const timeline = await page.request.get('/api/v1/assets/timeline');
					expect(timeline.status()).toBe(200);
					const body = (await timeline.json()) as TimelineBody;
					return (body.items ?? []).filter((item) => item.isReady).length;
				},
				{ timeout: 180_000, intervals: [1000, 2000, 5000] }
			)
			.toBe(3);
	});

	test('scene 3: the topbar search navigates and the Chinese query answers the flat rank-ordered wall', async ({
		page
	}) => {
		await page.goto('/timeline');
		await expect(page.getByRole('link', { name: PHOTO_GPS_NAME, exact: true })).toBeVisible();

		// The topbar's box: typing and committing (Enter) navigates
		// to /search with the encoded query.
		await page.locator('.topbar-search').fill(QUERY);
		await page.locator('.topbar-search').press('Enter');
		await expect(page).toHaveURL(`/search?query=${encodeURIComponent(QUERY)}`);
		await expect(page).toHaveTitle('Search · lumemor');

		// The count line is honest: all three photos answered (the
		// semantic path ranks the whole visible library, the text
		// path boosts the OCR hit).
		await expect(page.locator('.search-meta b')).toHaveText('3 results');

		// QUERY MODE renders the FLAT wall: rank order, no month or
		// day headers (grouping would lie about the rank order).
		await expect(page.locator('.flat-wall')).toBeVisible();
		await expect(page.locator('.wall__month-header')).toHaveCount(0);
		await expect(page.locator('.wall__day-header')).toHaveCount(0);

		// The rank order: the OCR photo -- hit by BOTH paths -- leads
		// the page (the merge rule: a both-path hit outranks the
		// single-path hits around it), and every photo answers
		// exactly once (the fusion's dedup).
		const tiles = page.locator('.flat-wall .tile');
		await expect(tiles).toHaveCount(3);
		await expect(tiles.first().getByRole('link')).toHaveAccessibleName(PHOTO_OCR_NAME);
		const names = await tiles
			.getByRole('link')
			.evaluateAll((links) => links.map((link) => link.getAttribute('aria-label')));
		expect(new Set(names).size).toBe(3);
		expect(names).toContain(PHOTO_GPS_NAME);
		expect(names).toContain(PHOTO_FACE_NAME);
	});

	test('scene 4: the filter browse renders the month-grouped wall and a facet chip re-executes', async ({ page }) => {
		// The bare /search (no query): BROWSE mode, the timeline's
		// exact month-grouped day-block wall over the time-ordered
		// items.
		await page.goto('/search');
		await expect(page.locator('.flat-wall')).toHaveCount(0);
		await expect(page.locator('.search-meta b')).toHaveText('3 results');
		for (const month of ['July 2026', 'June 2026', 'May 2026']) {
			await expect(page.locator('.wall__month-header', { hasText: month })).toBeVisible();
		}

		// A facet chip toggles: the Places · Beijing chip carries the
		// honest rest-of-filters count, the toggle lands the filter
		// in the URL, and the wall re-executes to the one Beijing
		// photo.
		const chip = page.getByRole('button', { name: 'Places · Beijing' });
		await expect(chip).toBeVisible();
		await expect(chip.locator('.search-chip__count')).toHaveText('1');
		await expect(chip).toHaveAttribute('aria-pressed', 'false');
		await chip.click();

		await expect(page).toHaveURL(/\/search\?placeIds=\d+$/);
		await expect(page.locator('.tile')).toHaveCount(1);
		await expect(page.locator('.tile').first().getByRole('link')).toHaveAccessibleName(PHOTO_GPS_NAME);
		await expect(page.locator('.search-meta b')).toHaveText('1 result');
		await expect(chip).toHaveAttribute('aria-pressed', 'true');

		// The filtered browse keeps the month-grouped wall (the
		// filter never changes the mode).
		await expect(page.locator('.wall__month-header', { hasText: 'June 2026' })).toBeVisible();
	});

	test('scene 5: the places page renders the city grid and deep-links into the filtered browse', async ({ page }) => {
		await page.goto('/places');
		await expect(page).toHaveTitle('Places · lumemor');
		await expect(page.getByRole('heading', { level: 1, name: 'Places · 1 city', exact: true })).toBeVisible();

		// The count rides the title in the page bar (one city here,
		// counted by CARDS, never a second line under the title).
		await expect(page.locator('.places__toolbar')).toHaveClass(/border-b/);

		// The Beijing card: the locale name over the cover, the
		// country label and the honest per-caller count below.
		const card = page.locator('a.place-card');
		await expect(card).toHaveCount(1);
		await expect(card.locator('.place-card__name')).toHaveText('Beijing');
		await expect(card.locator('.place-card__region')).toHaveText('China');
		// The count is the glyph and the number; the wording is its
		// accessible name.
		await expect(card.locator('.place-card__count')).toHaveAttribute('aria-label', '1 photo');

		// The view switch: the list is the current face, the map is the
		// page's other half (its own address).
		await expect(page.locator('[data-places-mode="list"]')).toHaveAttribute('aria-current', 'page');
		await expect(page.locator('[data-places-mode="map"]')).toHaveAttribute('href', '/places?mode=map');

		// The deep link: the card navigates into the filtered browse
		// -- the search page owns the filter state.
		await card.click();
		await expect(page).toHaveURL(/\/search\?placeIds=\d+$/);
		await expect(page.locator('.tile')).toHaveCount(1);
		await expect(page.locator('.tile').first().getByRole('link')).toHaveAccessibleName(PHOTO_GPS_NAME);
		await expect(page.locator('.wall__month-header', { hasText: 'June 2026' })).toBeVisible();
	});

	test('scene 6: the timeline batch bar tags the selection through the new-tag path', async ({ page }) => {
		await page.goto('/timeline');
		await expect(page.getByRole('link', { name: PHOTO_OCR_NAME, exact: true })).toBeVisible();

		// The selection joins through the tile's checkbox; the batch
		// bar rises with the honest count and the add-tag affordance.
		await page.getByRole('checkbox', { name: `Select: ${PHOTO_OCR_NAME}` }).click();
		const bar = page.getByRole('toolbar');
		await expect(bar).toContainText('1 selected');
		await bar.getByRole('button', { name: 'Add tag', exact: true }).click();

		// The dialog's new-tag path: name the tag, create-and-add.
		// The creation is idempotent, so the retry needs no
		// bookkeeping; the honest report lands in place of the form.
		const dialog = page.getByRole('dialog', { name: 'Add tag', exact: true });
		await expect(dialog).toBeVisible();
		await dialog.getByLabel('New tag').check();
		await dialog.getByLabel('Tag name').fill(TAG_NAME);
		await dialog.getByRole('button', { name: 'Create and add', exact: true }).click();
		await expect(dialog.getByRole('status')).toHaveText('Added 1, skipped 0.');

		// Done closes the dialog and clears the selection.
		await dialog.getByRole('button', { name: 'Done', exact: true }).click();
		await expect(dialog).toHaveCount(0);
		await expect(page.getByRole('toolbar')).toHaveCount(0);
	});

	test('scene 7: the new tag joins the search vocabulary as a facet chip', async ({ page }) => {
		await page.goto('/search');
		await expect(page.locator('.search-meta b')).toHaveText('3 results');

		// The tags facet is the caller's own vocabulary: the
		// scene-6 tag lists with the honest count over the browse
		// scope.
		const chip = page.getByRole('button', { name: `Tags · ${TAG_NAME}` });
		await expect(chip).toBeVisible();
		await expect(chip.locator('.search-chip__count')).toHaveText('1');
	});

	test('scene 8: the browse wall curates the per-user subset -- favorite and tag, the favorite batch lands', async ({
		page
	}) => {
		await page.goto('/search');
		await expect(page.locator('.search-meta b')).toHaveText('3 results');

		// The selection joins through the tile's hover check; the bar
		// rises with the honest count.
		await page.getByRole('checkbox', { name: `Select: ${PHOTO_GPS_NAME}`, exact: true }).click();
		const bar = page.getByRole('toolbar', { name: '1 selected' });
		await expect(bar).toBeVisible();

		// THE SUBSET (the curation ruling: [favorite, tag] for EVERY
		// role): the favorite toggle and the tag affordance render;
		// the ownership actions stay off the search walls' bars -- the
		// search surface is everyone's, the endpoint's owner-per-caller
		// resolution included.
		await expect(bar.getByRole('button', { name: 'Favorite', exact: true })).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Add tag', exact: true })).toBeVisible();
		for (const absent of ['Hide', 'Unhide', 'Trash', 'Restore']) {
			await expect(bar.getByRole('button', { name: absent, exact: true })).toHaveCount(0);
		}

		// The favorite batch lands: the optimistic flag is the final
		// state (favorites publish no events) and the bar's success
		// arm clears the selection.
		await bar.getByRole('button', { name: 'Favorite', exact: true }).click();
		await expect(page.getByRole('toolbar')).toHaveCount(0);
		await expect(
			page.getByRole('link', { name: PHOTO_GPS_NAME, exact: true }).locator('.tile__badge--favorite')
		).toBeAttached();

		// The server holds the flip: a reload answers the favorited
		// item (the wall's fresh page owns the truth).
		await page.reload();
		await expect(page.locator('.search-meta b')).toHaveText('3 results');
		await expect(
			page.getByRole('link', { name: PHOTO_GPS_NAME, exact: true }).locator('.tile__badge--favorite')
		).toBeAttached();
	});

	test('scene 9: the query wall carries the same subset through its own organ', async ({ page }) => {
		await page.goto('/timeline');
		await page.locator('.topbar-search').fill(QUERY);
		await page.locator('.topbar-search').press('Enter');
		await expect(page.locator('.flat-wall')).toBeVisible();

		// The selection joins through the FLAT wall's own check
		// affordance (the query wall's selection organ); the same
		// subset bar rises over the rank order.
		await page.getByRole('checkbox', { name: `Select: ${PHOTO_FACE_NAME}`, exact: true }).click();
		const bar = page.getByRole('toolbar', { name: '1 selected' });
		await expect(bar).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Favorite', exact: true })).toBeVisible();
		for (const absent of ['Hide', 'Unhide', 'Trash', 'Restore']) {
			await expect(bar.getByRole('button', { name: absent, exact: true })).toHaveCount(0);
		}

		// The tag flow rides the opener over the captured id: the
		// existing-tag path tags the selection with the scene-6 tag,
		// the honest report lands, Done clears.
		await bar.getByRole('button', { name: 'Add tag', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Add tag', exact: true });
		await expect(dialog).toBeVisible();
		await dialog.getByLabel('Existing tag').check();
		await dialog.getByLabel('Tag', { exact: true }).selectOption({ label: TAG_NAME });
		await dialog.getByRole('button', { name: 'Add tag', exact: true }).click();
		await expect(dialog.getByRole('status')).toHaveText('Added 1, skipped 0.');
		await dialog.getByRole('button', { name: 'Done', exact: true }).click();
		await expect(page.getByRole('toolbar')).toHaveCount(0);
	});
});
