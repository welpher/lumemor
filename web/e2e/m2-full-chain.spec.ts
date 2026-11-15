// The M2 full-chain story of the web app, walked end to end against
// the real compose stack (scripts/acceptance_m2d.sh builds the stack
// and waits for /healthz before running this suite; the spec itself
// only talks to http://localhost:8244). The stack is fresh and
// uninitialized when the suite starts -- the story performs the
// first-run bootstrap itself, exactly like the shell acceptance
// scripts do, through the setup API.
//
// THE SCENES (the full-chain script, in order):
//
//  1. A deep link to /timeline while signed out redirects to the
//     login page carrying the path as the post-login target.
//  2. The first-run admin (created through the setup API -- the
//     acceptance scripts' pattern) signs in through the form and
//     lands on the deep-linked timeline's empty state.
//  3. Uploading one plain photo through the top-bar picker settles
//     its queue row (the SSE-confirmed Done) and gains its tile.
//  4. The trashed-conflict-restore flow: a second photo is uploaded,
//     batch-trashed from the wall's selection bar, and the SAME file
//     re-uploaded -- the conflict dialog appears, Restore is chosen,
//     the row completes and the tile is live again.
//  5. The wall asserts the full order: the two uploaded photos plus
//     a Live Photo pair seeded through the upload API (the web never
//     sends the pairing video -- that is the device path), newest
//     first, folded into month groups.
//  6. The viewer: opening a tile, the arrow keys walk the wall's
//     sequence in both directions, Escape returns to the timeline.
//  7. The Live Photo press: the affordance swaps the still for the
//     component video's play stream (the component row's own id,
//     never the photo's), and pressing again returns the still.
//  8. Batch hide moves a tile to the hidden view; unhiding from
//     there returns it to the timeline.
//  9. The trash page's confirmed delete removes one asset for good
//     (trashed there through the viewer's action bar, the viewer
//     rework: the permanent delete lives on the trash page alone).
// 10. The trash view shows the retention countdown of the asset
//     trashed through the viewer's action bar, and restores it.
// 11. The interface-debt surfaces (the 2026-11-09 block): the
//     drawer carries no settings door (the account dropdown is its
//     only home), `?` opens the keyboard cheat sheet
//     and `/` focuses the search box, and dropping the network
//     raises the offline strip until the browser's own SSE retry
//     recovers.
// 12. Signing out re-arms the route guard: the deep link redirects
//     to the login page again.
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story --
// see playwright.config.ts) and share exactly two kinds of state.
// The server-side state (the uploads, the curation decisions) is the
// story itself; the browser-side session rides the login cookie,
// saved as a storage state after scene 2 and replayed by every later
// scene's fresh context. Each scene navigates explicitly to its
// starting route, so a scene's failure report names the exact step
// that broke.
//
// Determinism pins: the two uploaded photos carry no EXIF capture
// time, so their taken_at is the staged file mtime (the queue sends
// it as deviceCreatedAt); the staged mtimes pin the timeline order.
// The Live Photo pair's photo carries an EXIF DateTimeOriginal
// (2023-07-14 10:15:30 +05:30 -> 2023-07-14T04:45:30Z), which the
// metadata job applies over the device time -- its position is fixed
// by the fixture's bytes. The browser context runs in the UTC
// timezone so the device times group onto the same calendar dates
// the wire values name.

import { copyFile, mkdir, mkdtemp, readFile, utimes } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's administrator: created through the first-run setup API
// in the suite's bootstrap, then used by the login scene. The stack
// is always fresh, so the fixed identity is deterministic.
const ADMIN_EMAIL = 'admin@m2d-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The staged fixture files. The two browser-uploaded photos are real
// JPEGs from the server's test corpus (the upload endpoint sniffs
// real bytes); their staged mtimes become the queue's deviceCreatedAt
// values (neither file carries an EXIF capture time), pinning the
// wall order. The Live Photo pair rides the upload API: the photo
// part is another real JPEG (distinct bytes, so the owner-scoped
// checksum never collapses it into the browser uploads) whose EXIF
// time places it newest; the pairing video is the tiny real MP4 the
// media pipeline transcodes for browser playback.
const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));
const PHOTO_A_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/portrait-orientation6.jpg');
const PHOTO_B_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/golden-large.jpg');
const LIVE_PHOTO_SOURCE = path.join(repoRoot, 'server/internal/jobs/testdata/golden.jpg');
const LIVE_VIDEO_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/tiny.mp4');
const PHOTO_A_NAME = 'IMG_0001.jpg';
const PHOTO_B_NAME = 'IMG_0002.jpg';
const LIVE_PHOTO_NAME = 'LIVE_0001.jpg';
const LIVE_VIDEO_NAME = 'LIVE_0001.mp4';
const PHOTO_A_TAKEN_AT = new Date('2023-01-15T12:00:00Z');
const PHOTO_B_TAKEN_AT = new Date('2023-02-15T12:00:00Z');

// The storage-state file the login scene writes and every later
// scene replays (git-ignored, recreated per run).
const authStatePath = path.join(specDir, '.auth', 'story-state.json');

// The Live Photo pair's ids, captured from the seeding response: the
// photo row's id and its component video row's id (the press
// assertion needs the component's -- the pair's motion plays through
// the component's own id, never the photo's).
let livePhotoId = '';
let liveVideoId = '';

// The staged fixture paths, filled by the bootstrap.
let photoA = '';
let photoB = '';

/** One wall tile, addressed by its accessible name (the file name). */
function tile(page: Page, name: string) {
	return page.getByRole('link', { name, exact: true });
}

/** One tile's selection checkbox (revealed on hover or keyboard focus
 *  on every tile -- the check IS the selection entry, so it
 *  surfaces regardless of any select mode). */
function tileCheckbox(page: Page, name: string) {
	return page.getByRole('checkbox', { name: `Select: ${name}`, exact: true });
}

/** The floating batch bar of a wall, once at least one tile is selected. */
function batchBar(page: Page) {
	return page.getByRole('toolbar', { name: '1 selected' });
}

/** Open the navigation drawer and follow one of its links. */
async function navigateViaDrawer(page: Page, linkName: string): Promise<void> {
	await page.getByRole('button', { name: 'Open navigation' }).click();
	await page
		.getByRole('navigation', { name: 'Navigation' })
		.getByRole('link', { name: linkName, exact: true })
		.click();
}

/** Upload files through the top-bar picker (the real Upload button;
 *  the bar's own, since a page body may carry the same affordance). */
async function uploadThroughPicker(page: Page, ...files: string[]): Promise<void> {
	const [chooser] = await Promise.all([
		page.waitForEvent('filechooser'),
		page.getByRole('banner').getByRole('button', { name: 'Upload', exact: true }).click()
	]);
	await chooser.setFiles(files);
}

/** The upload overlay's row for one file name. */
function uploadRow(page: Page, fileName: string) {
	return page.getByRole('region', { name: 'Uploads' }).getByRole('listitem').filter({ hasText: fileName }).last();
}

test.beforeAll(async ({ playwright }) => {
	// Stage the browser-upload fixtures into a temp directory with
	// pinned mtimes: the file chooser turns the mtime into the File's
	// lastModified, which the queue sends as deviceCreatedAt, which
	// (no EXIF time in either file) becomes the asset's taken_at.
	const staging = await mkdtemp(path.join(tmpdir(), 'lumemor-e2e-'));
	photoA = path.join(staging, PHOTO_A_NAME);
	photoB = path.join(staging, PHOTO_B_NAME);
	await copyFile(PHOTO_A_SOURCE, photoA);
	await copyFile(PHOTO_B_SOURCE, photoB);
	await utimes(photoA, PHOTO_A_TAKEN_AT, PHOTO_A_TAKEN_AT);
	await utimes(photoB, PHOTO_B_TAKEN_AT, PHOTO_B_TAKEN_AT);

	await mkdir(path.dirname(authStatePath), { recursive: true });

	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance. The
	// browser scenes then only exercise the web app's own surfaces.
	// (The login scene performs the actual sign-in through the form.)
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'M2d E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

test('scene 1: a deep link while signed out redirects to the login page', async ({ page }) => {
	await page.goto('/timeline');

	// The guard's redirect carries the deep-linked path as the
	// post-login target (the instance is initialized, so the login
	// form stays put -- the first-run handoff to /setup is not this
	// scene's path).
	await expect(page).toHaveURL(/\/login\?redirectTo=%2Ftimeline$/);
	await expect(page.getByRole('heading', { level: 1, name: 'Sign in', exact: true })).toBeVisible();
});

test('scene 2: signing in through the form lands on the deep-linked timeline', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Ftimeline');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();

	// The post-login target round-trips: the deep link from scene 1
	// is honored, and the fresh instance's timeline is empty.
	await expect(page).toHaveURL(/\/timeline$/);
	// The empty wall is the one state card: a 16px line, no second
	// page heading above it (the shell's own bar is the page's head).
	await expect(page.locator('.state-card')).toBeVisible();
	await expect(page.getByText('No photos yet', { exact: true })).toBeVisible();
	await expect(page.locator('.state-card').getByRole('heading')).toHaveCount(0);

	// Hand the session to every later scene: the storage state rides
	// the login cookie (the fresh contexts replay it).
	await page.context().storageState({ path: authStatePath });
});

// The signed-in half of the story: every scene starts from a fresh
// context that replays the login cookie scene 2 saved. The serial
// mode guarantees the state file exists by the time these scenes run.
test.describe('the signed-in scenes', () => {
	test.use({ storageState: authStatePath });

	test('scene 3: uploading a photo through the picker settles its row and tile', async ({ page }) => {
		await page.goto('/timeline');
		await expect(tile(page, PHOTO_A_NAME)).toHaveCount(0);

		// The empty wall's own way in: the no-data card's Upload starts
		// the same queue the top bar does (the timeline carries no page
		// bar, so the card is its one title-and-action surface).
		await expect(page.locator('.state-card')).toBeVisible();
		const [chooser] = await Promise.all([
			page.waitForEvent('filechooser'),
			page.locator('.state-card').getByRole('button', { name: 'Upload', exact: true }).click()
		]);
		await chooser.setFiles([photoA]);

		// The queue row settles only on the SSE asset.ready event (the
		// pipeline's metadata and thumbnail jobs must drain first) -- the
		// auto-wait covers the whole pipeline.
		await expect(uploadRow(page, PHOTO_A_NAME)).toBeVisible();
		await expect(uploadRow(page, PHOTO_A_NAME).getByText('Done', { exact: true })).toBeVisible({
			timeout: 60_000
		});

		// The timeline gains the tile (the SSE convergence upserts it).
		await expect(tile(page, PHOTO_A_NAME)).toBeVisible({ timeout: 60_000 });

		// Keep the story's pages clean of the overlay panel: terminal
		// rows persist until dismissed, and the panel would sit over the
		// wall's lower-right corner for the rest of the story.
		await uploadRow(page, PHOTO_A_NAME).getByRole('button', { name: 'Remove from list' }).click();
		await expect(page.getByRole('region', { name: 'Uploads' })).toHaveCount(0);
	});

	test('scene 4: re-uploading a trashed photo offers the restore decision', async ({ page }) => {
		await page.goto('/timeline');
		await expect(tile(page, PHOTO_A_NAME)).toBeVisible();

		// Upload the second photo; its row settles and its tile appears.
		await uploadThroughPicker(page, photoB);
		await expect(uploadRow(page, PHOTO_B_NAME).getByText('Done', { exact: true })).toBeVisible({
			timeout: 60_000
		});
		await expect(tile(page, PHOTO_B_NAME)).toBeVisible({ timeout: 60_000 });

		// Batch-trash it from the wall's selection bar: select the tile's
		// checkbox, the bar rises, the Trash action removes the tile.
		await tileCheckbox(page, PHOTO_B_NAME).click();
		await expect(batchBar(page)).toBeVisible();
		await batchBar(page).getByRole('button', { name: 'Trash', exact: true }).click();
		await expect(tile(page, PHOTO_B_NAME)).toHaveCount(0);

		// Re-upload the SAME file: the owner-scoped checksum matches the
		// trashed row, the queue parks the row and the conflict dialog
		// appears.
		await uploadThroughPicker(page, photoB);
		const dialog = page.getByRole('dialog', { name: 'Duplicate in trash', exact: true });
		await expect(dialog).toBeVisible({ timeout: 60_000 });
		await expect(dialog).toContainText(`"${PHOTO_B_NAME}"`);

		// Choose Restore: the trashed row is restored through the batch
		// endpoint, the queue row completes, the tile is live again.
		await dialog.getByRole('button', { name: 'Restore the trash copy', exact: true }).click();
		await expect(dialog).toHaveCount(0);
		await expect(uploadRow(page, PHOTO_B_NAME).getByText('Done', { exact: true })).toBeVisible({
			timeout: 60_000
		});
		await expect(tile(page, PHOTO_B_NAME)).toBeVisible({ timeout: 60_000 });

		await uploadRow(page, PHOTO_B_NAME).getByRole('button', { name: 'Remove from list' }).click();
	});

	test('scene 5: the wall orders the photos newest first with month groups', async ({ page }) => {
		await page.goto('/timeline');
		await expect(tile(page, PHOTO_A_NAME)).toBeVisible();
		await expect(tile(page, PHOTO_B_NAME)).toBeVisible();

		// Seed the Live Photo pair through the upload API -- the pairing
		// video is the device path the web never sends. The request rides
		// the page's own context (its login cookie), and the browser sees
		// the pair through the same SSE stream the timeline subscribes to.
		const seed = await page.request.post('/api/v1/assets', {
			multipart: {
				deviceAssetId: 'm2d-e2e-live-photo',
				fileName: LIVE_PHOTO_NAME,
				deviceCreatedAt: '2023-06-06T12:00:00Z',
				deviceTimeZone: 'UTC',
				photo: {
					name: LIVE_PHOTO_NAME,
					mimeType: 'image/jpeg',
					buffer: await readFile(LIVE_PHOTO_SOURCE)
				},
				liveVideo: {
					name: LIVE_VIDEO_NAME,
					mimeType: 'video/mp4',
					buffer: await readFile(LIVE_VIDEO_SOURCE)
				}
			}
		});
		expect(seed.status()).toBe(201);
		const seeded = (await seed.json()) as { id: string };
		livePhotoId = seeded.id;
		// The upload response carries the row's identity but not the
		// pair's linkage (AssetResponse has no livePhotoVideoId; the
		// detail endpoint does) -- read the component's id from the
		// detail. The typeof check is exact on purpose: a missing key
		// would read as undefined and slip past a null comparison.
		const detail = await page.request.get(`/api/v1/assets/${livePhotoId}`);
		expect(detail.status()).toBe(200);
		const pair = (await detail.json()) as { livePhotoVideoId: string | null };
		expect(typeof pair.livePhotoVideoId).toBe('string');
		liveVideoId = pair.livePhotoVideoId as string;

		// The convergence brings the photo row's tile onto the wall (the
		// component row is never a timeline item), newest first: the EXIF
		// time (July 2023) places it above the staged device times.
		await expect(tile(page, LIVE_PHOTO_NAME)).toBeVisible({ timeout: 60_000 });
		// The tile first converges with the device-reported time; the
		// pipeline's metadata job then refines taken_at onto the EXIF
		// time and the asset.ready event re-converges the tile. The
		// month group's settled label is the visible signal of that
		// state (the tile's own thumbnail rides the same event).
		await expect(page.getByRole('heading', { level: 2, name: 'July 2023', exact: true })).toBeVisible({
			timeout: 60_000
		});
		const names = await page
			.getByRole('link', { name: new RegExp(`^(${LIVE_PHOTO_NAME}|${PHOTO_B_NAME}|${PHOTO_A_NAME})$`) })
			.evaluateAll((nodes) => nodes.map((node) => (node as HTMLAnchorElement).getAttribute('aria-label')));
		expect(names).toEqual([LIVE_PHOTO_NAME, PHOTO_B_NAME, PHOTO_A_NAME]);

		// The wall folds the same order into sticky month groups (the
		// fixtures are few -- one keyset page, no load-more to walk).
		const months = await page
			.getByRole('heading', { level: 2 })
			.filter({ hasText: /^(July|February|January) 2023$/ })
			.evaluateAll((nodes) => nodes.map((node) => node.textContent));
		expect(months).toEqual(['July 2023', 'February 2023', 'January 2023']);

		// The Live Photo's tile carries the wall's live badge -- the
		// timeline's only Live Photo affordance before the viewer.
		await expect(tile(page, LIVE_PHOTO_NAME).locator('[title="Live Photo"]')).toBeVisible();
	});

	test('scene 6: the viewer walks the wall and Escape returns', async ({ page }) => {
		await page.goto('/timeline');
		await expect(tile(page, LIVE_PHOTO_NAME)).toBeVisible();

		// Open a tile from the middle of the sequence. The floating bar
		// carries NO filename (it would cover the photo): the open
		// asset's name is the browser tab's and the panel's header's.
		await tile(page, PHOTO_B_NAME).click();
		await expect(page).toHaveTitle(`${PHOTO_B_NAME} · lumemor`);

		// The filmstrip carries the wall's whole sequence; the current
		// thumb is marked.
		const filmstrip = page.getByRole('navigation', { name: 'Photos in view' });
		await expect(filmstrip).toBeVisible();
		await expect(filmstrip.getByRole('button', { name: PHOTO_B_NAME, exact: true })).toHaveAttribute(
			'aria-current',
			'true'
		);

		// The stage's contain rule: the media box fits the photo's own
		// aspect and touches the stage on one dimension -- no percentage
		// inset. The stage is the media column's whole box, not the bar's
		// (the bar floats over the photo now).
		await expect
			.poll(async () => (await page.locator('.viewer__media').boundingBox())?.width ?? 0)
			.toBeGreaterThan(100);
		const stageBox = await page.locator('.viewer__stage').boundingBox();
		const mediaBox = await page.locator('.viewer__media').boundingBox();
		if (stageBox === null || mediaBox === null) {
			throw new Error('the stage geometry never landed');
		}
		const photoRatio = await page
			.locator('img.viewer__image')
			.first()
			.evaluate((element) => {
				const image = element as HTMLImageElement;
				return image.naturalWidth / image.naturalHeight;
			});
		expect(Math.abs(mediaBox.width / mediaBox.height - photoRatio)).toBeLessThan(0.02);
		const touchesWidth = Math.abs(mediaBox.width - stageBox.width) < 2;
		const touchesHeight = Math.abs(mediaBox.height - stageBox.height) < 2;
		expect(touchesWidth || touchesHeight).toBe(true);

		// The info panel is CLOSED by default (the pinned preference):
		// the photo gets the whole stage until something asks for it.
		await expect(page.getByRole('button', { name: 'Photo information' })).toHaveAttribute('aria-pressed', 'false');

		// ArrowRight walks toward the older end, ArrowLeft back toward
		// the newer one; both replace the open asset (the tab's title
		// follows, and the filmstrip's current thumb with it).
		await page.keyboard.press('ArrowRight');
		await expect(page).toHaveTitle(`${PHOTO_A_NAME} · lumemor`);
		await expect(filmstrip.getByRole('button', { name: PHOTO_A_NAME, exact: true })).toHaveAttribute(
			'aria-current',
			'true'
		);
		await page.keyboard.press('ArrowLeft');
		await expect(page).toHaveTitle(`${PHOTO_B_NAME} · lumemor`);
		await page.keyboard.press('ArrowLeft');
		await expect(page).toHaveTitle(`${LIVE_PHOTO_NAME} · lumemor`);

		// Escape returns to the timeline (the history-back rule: the tile
		// click pushed exactly one entry).
		await page.keyboard.press('Escape');
		await expect(page).toHaveURL(/\/timeline$/);
	});

	test('scene 7: pressing a Live Photo plays the component stream', async ({ page }) => {
		await page.goto('/timeline');
		await tile(page, LIVE_PHOTO_NAME).click();

		// The press affordance exists on the still (the pair's component
		// id is part of the photo row's detail).
		const press = page.getByRole('button', { name: 'Play the Live Photo', exact: true });
		await expect(press).toBeVisible();

		// The component never broadcasts its own readiness, and the photo
		// row's events say nothing about the transcode -- the same range
		// probe the viewer's press runs settles the wait here: the
		// component's play stream answers 206 once its transcode job has
		// drained.
		const componentPlayUrl = `/api/v1/assets/${liveVideoId}/play`;
		await expect
			.poll(
				async () => {
					const probe = await page.request.get(componentPlayUrl, {
						headers: { Range: 'bytes=0-0' }
					});
					return probe.status();
				},
				// The scene's other pipeline waits allow a cold runner a
				// full minute; the transcode drain this probes is the
				// same class of wait.
				{ timeout: 60_000 }
			)
			.toBe(206);

		// Pressing swaps the still for the component's video: the element
		// names the Live Photo and its src is the COMPONENT row's play
		// stream -- never the photo's own id.
		await press.click();
		const liveVideo = page.locator('video[aria-label="Live Photo"]');
		await expect(liveVideo).toBeVisible();
		await expect(liveVideo).toHaveAttribute('src', componentPlayUrl);
		expect(componentPlayUrl).not.toContain(livePhotoId);

		// Pressing again stops the motion and returns the still.
		await page.getByRole('button', { name: 'Stop the Live Photo', exact: true }).click();
		await expect(liveVideo).toHaveCount(0);
		await expect(press).toBeVisible();
	});

	test('scene 8: batch hide parks a tile in the hidden view and unhiding returns it', async ({ page }) => {
		await page.goto('/timeline');
		await expect(tile(page, PHOTO_A_NAME)).toBeVisible();

		// Select the tile and hide it through the batch bar: the tile
		// leaves the timeline.
		await tileCheckbox(page, PHOTO_A_NAME).click();
		await batchBar(page).getByRole('button', { name: 'Hide', exact: true }).click();
		await expect(tile(page, PHOTO_A_NAME)).toHaveCount(0);

		// The navigation drawer's Hidden entry opens the hidden view
		// (the timeline's toolbar link is gone -- the drawer
		// owns the entry, beside favorites and trash): the tile lives
		// there now.
		await navigateViaDrawer(page, 'Hidden');
		await expect(page).toHaveURL(/\/hidden$/);
		await expect(page).toHaveTitle('Hidden · lumemor');
		// The converged page bar: the title with its count on the
		// left, the selection switch at the far right -- one row with
		// one divider under it.
		await expect(page.locator('.wall-page__title')).toHaveText('Hidden 1');
		await expect(page.locator('.wall-page__title-count')).toHaveAttribute('aria-label', '1 photo');
		await expect(page.locator('.wall-page__toolbar')).toHaveClass(/border-b/);
		await expect(tile(page, PHOTO_A_NAME)).toBeVisible();

		// The switch is a REAL selection state: pressed on it marks the
		// wall and reveals every check, pressed off it clears the
		// selection and the batch bar leaves the surface.
		const selectSwitch = page.locator('.wall-page__actions button');
		await expect(selectSwitch).toHaveAttribute('aria-pressed', 'false');
		await selectSwitch.click();
		await expect(selectSwitch).toHaveAttribute('aria-pressed', 'true');
		await expect(page.locator('.wall__groups')).toHaveClass(/is-selecting/);
		await selectSwitch.click();
		await expect(selectSwitch).toHaveAttribute('aria-pressed', 'false');
		await expect(page.locator('.wall__groups')).not.toHaveClass(/is-selecting/);
		await expect(page.locator('.batchbar')).toHaveCount(0);

		// Unhide from the hidden view's own batch bar: the tile leaves,
		// the view settles empty.
		await tileCheckbox(page, PHOTO_A_NAME).click();
		await batchBar(page).getByRole('button', { name: 'Unhide', exact: true }).click();
		await expect(tile(page, PHOTO_A_NAME)).toHaveCount(0);
		await expect(page.getByText('Nothing hidden')).toBeVisible();

		// Back on the timeline (through the drawer's navigation), the
		// unhidden tile is back with the other two.
		await navigateViaDrawer(page, 'Timeline');
		await expect(page).toHaveURL(/\/timeline$/);
		await expect(tile(page, PHOTO_A_NAME)).toBeVisible();
		await expect(tile(page, PHOTO_B_NAME)).toBeVisible();
		await expect(tile(page, LIVE_PHOTO_NAME)).toBeVisible();
	});

	test("scene 9: the trash page's confirmed delete removes the asset for good", async ({ page }) => {
		await page.goto('/timeline');
		await tile(page, PHOTO_B_NAME).click();

		// Trash it through the viewer's ACTION BAR (the viewer rework:
		// the curation affordances ride the bar's icon buttons and the
		// info drawer is metadata-only; the permanent delete lives on
		// the trash page alone): the drawer's trashed notice carries
		// the optimistic flip, and the back mark returns to the wall.
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = page.getByRole('complementary', { name: 'Photo information' });
		await expect(drawer).toBeVisible();
		await page.getByRole('button', { name: 'Trash', exact: true }).click();
		await expect(drawer.getByRole('status').getByText('This photo is in the trash.')).toBeVisible();
		await page.getByRole('button', { name: 'Back to the timeline' }).click();
		await expect(page).toHaveURL(/\/timeline$/);
		await expect(tile(page, PHOTO_B_NAME)).toHaveCount(0);

		// The trash page's recovery card carries the permanent delete.
		await navigateViaDrawer(page, 'Trash');
		await expect(page).toHaveURL(/\/trash$/);
		await expect(page.getByRole('heading', { level: 1, name: /^Trash \d+ photos?$/ })).toBeVisible();
		await page.getByRole('button', { name: 'Delete permanently', exact: true }).click();

		// The confirm dialog names the asset and requires the explicit
		// confirmation.
		const dialog = page.getByRole('dialog', { name: 'Delete permanently?', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText(`"${PHOTO_B_NAME}"`);
		await dialog.getByRole('button', { name: 'Delete forever', exact: true }).click();

		// The row is gone for good: the trash settles empty and the
		// survivors remain on the timeline.
		await expect(tile(page, PHOTO_B_NAME)).toHaveCount(0);
		await expect(page.getByText('The trash is empty', { exact: true })).toBeVisible();
		await navigateViaDrawer(page, 'Timeline');
		await expect(page).toHaveURL(/\/timeline$/);
		await expect(tile(page, PHOTO_A_NAME)).toBeVisible();
		await expect(tile(page, LIVE_PHOTO_NAME)).toBeVisible();
	});

	test('scene 10: the trash view counts down retention and restores', async ({ page }) => {
		await page.goto('/timeline');
		await tile(page, PHOTO_A_NAME).click();

		// Trash it through the viewer's ACTION BAR (the viewer rework
		// moved the curation affordances from the drawer to the bar's
		// icon buttons): the trashed notice appears in the drawer, and
		// the timeline drops the tile.
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = page.getByRole('complementary', { name: 'Photo information' });
		await page.getByRole('button', { name: 'Trash', exact: true }).click();
		await expect(drawer.getByRole('status').getByText('This photo is in the trash.')).toBeVisible();
		await page.getByRole('button', { name: 'Back to the timeline' }).click();
		await expect(page).toHaveURL(/\/timeline$/);
		await expect(tile(page, PHOTO_A_NAME)).toHaveCount(0);

		// The drawer's Trash entry opens the trash view: the trashed
		// asset's recovery tile carries the retention countdown (the
		// default retention is 30 days).
		await navigateViaDrawer(page, 'Trash');
		await expect(page).toHaveURL(/\/trash$/);
		await expect(page.getByRole('heading', { level: 1, name: /^Trash \d+ photos?$/ })).toBeVisible();
		await expect(tile(page, PHOTO_A_NAME)).toBeVisible();
		await expect(page.getByText(/^(Expiring soon|\d+ days? left)$/)).toBeVisible();

		// Restore returns the asset to the timeline; the trash settles
		// empty.
		await page.getByRole('button', { name: 'Restore', exact: true }).click();
		await expect(tile(page, PHOTO_A_NAME)).toHaveCount(0);
		await expect(page.getByText('The trash is empty', { exact: true })).toBeVisible();

		// The restored tile is back on the timeline.
		await navigateViaDrawer(page, 'Timeline');
		await expect(tile(page, PHOTO_A_NAME)).toBeVisible();
		await expect(tile(page, LIVE_PHOTO_NAME)).toBeVisible();
	});

	test('scene 11: the interface-debt surfaces — the drawer has no settings door, `?` opens the sheet, the offline strip rises', async ({
		page
	}) => {
		await page.goto('/timeline');
		await expect(tile(page, PHOTO_A_NAME)).toBeVisible();

		// The drawer (the single-home rule): the content
		// sections carry NO settings entry for any role -- the
		// account dropdown is settings' ONLY home.
		await page.getByRole('button', { name: 'Open navigation' }).click();
		const drawer = page.getByRole('navigation', { name: 'Navigation' });
		await expect(drawer).toBeVisible();
		await expect(drawer.getByRole('link', { name: 'Settings', exact: true })).toHaveCount(0);
		await page.keyboard.press('Escape');
		await expect(drawer).toHaveCount(0);

		// The account dropdown carries the settings link instead --
		// the single entry the pin leaves in place.
		await page.getByRole('button', { name: 'Account' }).click();
		await expect(page.getByRole('link', { name: 'Settings', exact: true })).toBeVisible();
		await page.keyboard.press('Escape');

		// The `?` cheat sheet: the collected bindings render in
		// the two-column shape, grouped by context; Esc closes.
		await page.keyboard.press('?');
		const sheet = page.getByRole('dialog', { name: 'Keyboard shortcuts' });
		await expect(sheet).toBeVisible();
		await expect(sheet.getByText('Focus the search box')).toBeVisible();
		await expect(sheet.getByText('Move to the previous or next photo')).toBeVisible();
		await expect(sheet.getByText('Clear the selection')).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(sheet).toHaveCount(0);

		// The `/` focus-search binding: the stroke lands the
		// focus on the topbar's search box.
		await page.keyboard.press('/');
		await expect(page.locator('input.topbar-search')).toBeFocused();

		// The offline strip: the SSE stream dies at
		// the network layer -- the events request is aborted at the
		// route (the surgical equivalent of the connection dropping:
		// the page itself stays healthy, only the stream is gone).
		// The reload re-opens the stream into the abort, the yellow
		// band rises with the honest reconnecting count, and
		// un-routing lets the browser's OWN EventSource retry land --
		// the strip falls with a recovery nobody had to wire, because
		// the browser owns the retry. (A context-wide setOffline
		// leaves the ALREADY-ESTABLISHED stream untouched -- CDP's
		// network emulation gates new requests only -- so the route
		// abort is the deterministic drop.)
		await page.route('**/api/v1/events', (route) => route.abort());
		await page.reload();
		await expect(page.getByRole('status').filter({ hasText: 'Connection lost — reconnecting' })).toBeVisible();
		await page.unroute('**/api/v1/events');
		await expect(page.getByRole('status').filter({ hasText: 'Connection lost — reconnecting' })).toHaveCount(0);
	});

	test('scene 12: signing out re-arms the route guard', async ({ page }) => {
		await page.goto('/timeline');

		// The account menu's sign-out revokes the session and lands on
		// the login page.
		await page.getByRole('button', { name: 'Account' }).click();
		await page.getByRole('button', { name: 'Sign out', exact: true }).click();
		await expect(page).toHaveURL(/\/login/);
		await expect(page.getByRole('heading', { level: 1, name: 'Sign in', exact: true })).toBeVisible();

		// The deep link now redirects to the login page again, carrying
		// the path as the post-login target -- the guard of scene 1, on
		// the signed-out side of the story's end.
		await page.goto('/timeline');
		await expect(page).toHaveURL(/\/login\?redirectTo=%2Ftimeline$/);
	});
});
