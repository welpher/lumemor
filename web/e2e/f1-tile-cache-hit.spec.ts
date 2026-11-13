// The F1 block's browser regression: the wall tile's cache-hit race
// (an implementation defect, not a design change -- the placeholder,
// the crossfade and the serving copy all stay). THE BUG: a container
// resize is the wall's normal operation (its rows re-pack), and the
// re-pack moves tiles between rows. Svelte keys a row by its index and
// a tile by its item id, so a tile that changed rows is destroyed and
// re-created: the fresh <img> gets its src before Svelte attaches
// onload, and a fully decoded image from the browser's memory cache
// dispatches `load` synchronously inside that window. That event never
// reaches the component and nobody fires it again, so the tile sat on
// its blurhash forever -- sharp image on screen only after a reload.
// THE FIX (PhotoTile.svelte): the mount checks the element's own
// completion state (`complete && naturalWidth > 0`) instead of waiting
// for an event that may already be gone; the old unconditional
// "reset the crossfade" effect was wiping the only signal the element
// had.
//
// THE STORY: the bootstrap creates the administrator through the
// setup API (the acceptance scripts' pattern) and signs in through the
// form; the signed-in half then seeds 24 genuinely distinct photos
// through the upload API and waits for the pipeline's asset.ready.
// Each seeded file is the same fixture with a per-file trailing
// padding after the JPEG's EOI marker: the sniffing and the decode
// ignore it, and the owner-scoped checksum dedup never collapses the
// copies into one row. The photos alternate a wide (2:1) and a tall
// (4:5) fixture, so the justified rows carry the varied aspect the
// wall packs around.
//
// THE RESIZES (the regression itself): the wall is settled at the
// story's base width, then driven 720 -> 1600 -> 1280, and every leg
// asserts the same pin -- every viewport-intersecting tile carries
// `.tile__image--loaded`. Only intersecting tiles are asserted:
// offscreen tiles legally skip loading (`content-visibility: auto` on
// the tile and the row, `loading="lazy"` on the image), so an unloaded
// offscreen tile is not a defect. Each leg also pins the fix's
// bandwidth constraint: no tile the browser already holds may be
// requested again (a cache-hit remount, never a second load).
//
// THE WALKTHROUGH BEATS: the timeline's resize
// across the three widths is the first test; the favorites wall is the
// second PhotoTile surface (the same tile component, a different
// page), resized through the same legs.
//
// THE DEVICE PIXEL RATIO: this spec runs at deviceScaleFactor 2 -- the
// race reproduces there (the story's Mac conditions); the suite's own
// config stays at the 1x default and this spec overrides it for
// itself alone. The stack's lifecycle is owned by the caller (a fresh
// compose stack on http://localhost:8244); the spec only talks to it.

import { mkdir, readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type APIRequestContext, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's administrator: created through the first-run setup API,
// then signed in through the form. The stack is fresh, so the fixed
// identity is deterministic.
const ADMIN_EMAIL = 'admin@f1-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The two seed fixtures: a 1600x800 wide frame and a 500x625 portrait
// (neither carries an EXIF capture time, so the device-stamped day is
// the whole wall's day group). Both are the server's own test corpus
// (real decodable bytes, no third-party material).
const WIDE_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/golden-large.jpg');
const TALL_SOURCE = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait.jpg');

// The wall's seed: 24 photos -- enough rows at every leg's width that
// the re-pack genuinely moves tiles between rows (the race's trigger).
const PHOTO_COUNT = 24;
const PHOTO_DAY = '2026-03-14';
const PHOTO_PREFIX = 'F1_';

// The favorites wall's seed: the newest nine of them, favorited
// through the batch endpoint (the second PhotoTile surface).
const FAVORITE_COUNT = 9;

// The visible-tile floor: a wall leg that renders fewer intersecting
// tiles than this cannot prove anything (a collapsed layout would
// otherwise pass the stuck-zero assertion vacuously). The narrowest
// leg (720 wide) still shows about a day's rows -- well past it.
const MIN_VISIBLE_TILES = 6;

// The pipeline bound: 24 uploads' metadata, thumbnails and the
// asset-ready seam on a warmed model tree -- minutes at worst.
const PIPELINE_TIMEOUT_MS = 180_000;

// The storage-state file the sign-in scene writes and the wall scenes
// replay (git-ignored, recreated per run).
const authStatePath = path.join(specDir, '.auth', 'f1-state.json');

// The seeded rows, captured from the upload responses: the favorites
// batch needs the newest ids.
let seededIds: string[] = [];

/** One file name of the seed: F1_0001.jpg .. F1_0024.jpg. */
function seedName(index: number): string {
	return `${PHOTO_PREFIX}${String(index + 1).padStart(4, '0')}.jpg`;
}

/**
 * One genuinely distinct derivative of a fixture: the same decodable
 * bytes with a per-file trailer after the JPEG's EOI marker. The
 * magic-byte sniffing and the decode both stop at the marker; the
 * trailer alone makes the owner-scoped checksum dedup see 24
 * different files.
 */
function seedBytes(source: Buffer, index: number): Buffer {
	return Buffer.concat([source, Buffer.from(`\nF1-E2E-SEED-${index}\n`)]);
}

/** Upload one seed photo through the real upload API (the queue's
 *  device path: the admin's own session, multipart, the file's
 *  device-reported capture time). */
async function uploadSeed(api: APIRequestContext, index: number, wide: Buffer, tall: Buffer): Promise<string> {
	const name = seedName(index);
	const minute = String(10 + Math.floor(index / 60)).padStart(2, '0');
	const second = String(index % 60).padStart(2, '0');
	const response = await api.post('/api/v1/assets', {
		multipart: {
			deviceAssetId: `f1-e2e-${index}`,
			fileName: name,
			deviceCreatedAt: `${PHOTO_DAY}T${minute}:${second}:00Z`,
			deviceTimeZone: 'UTC',
			photo: {
				name,
				mimeType: 'image/jpeg',
				buffer: seedBytes(index % 2 === 0 ? wide : tall, index)
			}
		}
	});
	expect(response.status(), `the upload of ${name}`).toBe(201);
	const created = (await response.json()) as { id: string };
	return created.id;
}

/**
 * The tiles intersecting the viewport, and how many of those are still
 * on their placeholder. Offscreen tiles are excluded by construction
 * (content-visibility answers real boxes only for the rendered
 * rows): their lazy loads are legitimate.
 */
type WallProbe = { visible: number; stuck: number };

function probeVisibleTiles(page: Page): Promise<WallProbe> {
	return page.evaluate(() => {
		const images = Array.from(document.querySelectorAll<HTMLImageElement>('img.tile__image'));
		const width = window.innerWidth;
		const height = window.innerHeight;
		const visible = images.filter((image) => {
			const box = image.getBoundingClientRect();
			return (
				box.width > 0 &&
				box.height > 0 &&
				box.bottom > 0 &&
				box.top < height &&
				box.right > 0 &&
				box.left < width
			);
		});
		return {
			visible: visible.length,
			stuck: visible.filter((image) => !image.classList.contains('tile__image--loaded')).length
		};
	});
}

/** Wait for the wall to settle: intersecting tiles exist, and every
 *  one of them carries the loaded class (the cache hit is immediate --
 *  a stuck tile is the defect, not a slow load). */
async function settleVisibleTiles(page: Page): Promise<void> {
	await expect
		.poll(async () => (await probeVisibleTiles(page)).visible, {
			message: 'the wall renders viewport-intersecting tiles',
			timeout: 15_000
		})
		.toBeGreaterThanOrEqual(MIN_VISIBLE_TILES);
	await expect
		.poll(async () => (await probeVisibleTiles(page)).stuck, {
			message: 'every viewport-intersecting tile shows its sharp image (the cache-hit load race)',
			timeout: 15_000
		})
		.toBe(0);
}

/** The thumbnail paths of the tiles on screen, loaded: the set the
 *  next resize must reuse from the cache, never re-request. */
function loadedPaths(page: Page): Promise<string[]> {
	return page.evaluate(() =>
		Array.from(document.querySelectorAll<HTMLImageElement>('img.tile__image--loaded')).map(
			(image) => image.getAttribute('src') ?? ''
		)
	);
}

/**
 * One resize leg: drive the viewport, settle the visible tiles, and
 * pin the bandwidth constraint -- no path that was loaded before the
 * leg may be requested during it. Newly intersecting tiles (the wider
 * leg uncovers rows the narrower one hid) are first loads and may ask;
 * the ones already held must come from the browser's cache.
 */
async function resizeWall(page: Page, width: number, height: number): Promise<void> {
	const before = new Set(await loadedPaths(page));
	const requested: string[] = [];
	const record = (request: { url(): string }) => {
		const url = new URL(request.url());
		if (url.pathname.includes('/thumbnail')) {
			requested.push(`${url.pathname}${url.search}`);
		}
	};
	page.on('request', record);
	try {
		await page.setViewportSize({ width, height });
		await settleVisibleTiles(page);
	} finally {
		page.off('request', record);
	}
	const reloads = requested.filter((pathname) => before.has(pathname));
	expect(reloads, 'a resize re-requests an image the browser already holds (the fix is cache hits only)').toEqual([]);
}

test.beforeAll(async ({ playwright }) => {
	await mkdir(path.dirname(authStatePath), { recursive: true });

	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance. The
	// browser scenes then only exercise the web app's own surfaces.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'F1 E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

test('the story signs the administrator in and hands the session on', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Ftimeline');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);

	// The session rides the login cookie: the walls' scenes replay it.
	await page.context().storageState({ path: authStatePath });
});

test.describe('the cache-hit race', () => {
	// The bug's reproduction conditions (the story's Mac): a 2x device
	// pixel ratio at the story's base width. Only this spec's context
	// overrides the suite's 1x default.
	test.use({
		storageState: authStatePath,
		viewport: { width: 1280, height: 900 },
		deviceScaleFactor: 2
	});

	test.beforeAll(async ({ playwright }) => {
		// The seed rides the admin's own session (the storage state the
		// sign-in scene saved): 24 uploads, then the pipeline's
		// asset.ready settles the wall.
		const api = await playwright.request.newContext({ baseURL: BASE_URL, storageState: authStatePath });
		const wide = await readFile(WIDE_SOURCE);
		const tall = await readFile(TALL_SOURCE);

		seededIds = [];
		for (let index = 0; index < PHOTO_COUNT; index += 1) {
			seededIds.push(await uploadSeed(api, index, wide, tall));
		}

		// The pipeline's readiness: the timeline lists the rows as they
		// land, and the tile's <img> only renders once isReady flips.
		await expect
			.poll(
				async () => {
					const response = await api.get('/api/v1/assets/timeline?limit=100');
					if (response.status() !== 200) {
						return -1;
					}
					const body = (await response.json()) as { items: Array<{ isReady: boolean }> };
					return body.items.filter((item) => item.isReady).length;
				},
				{ timeout: PIPELINE_TIMEOUT_MS, intervals: [1000] }
			)
			.toBe(PHOTO_COUNT);

		// The second PhotoTile surface's seed: the newest nine are the
		// favorites wall's tiles (the batch endpoint's per-user mark).
		const favorites = await api.post('/api/v1/assets/batch', {
			data: { action: 'favorite', ids: seededIds.slice(-FAVORITE_COUNT) }
		});
		expect(favorites.status()).toBe(200);
		await api.dispose();
	});

	test('the timeline wall keeps every visible tile loaded across three widths', async ({ page }) => {
		await page.goto('/timeline');
		// The wall's first paint (the async load path) settles first;
		// the resizes below then re-pack an already-warm cache.
		await expect(page.locator('img.tile__image')).toHaveCount(PHOTO_COUNT, { timeout: 60_000 });
		await settleVisibleTiles(page);

		// 1280 -> 720: the rows halve their tiles per row, so most
		// tiles change rows and Svelte re-creates them.
		await resizeWall(page, 720, 900);
		// 720 -> 1600: the rows densify and the re-pack runs the other
		// way (the narrow leg's visible tiles are a subset of the wide
		// one's).
		await resizeWall(page, 1600, 900);
		// 1600 -> 1280: back to the story's base width.
		await resizeWall(page, 1280, 900);
	});

	test('the favorites wall carries the same recovery', async ({ page }) => {
		await page.goto('/favorites');
		await expect(page.locator('img.tile__image')).toHaveCount(FAVORITE_COUNT, { timeout: 60_000 });
		await settleVisibleTiles(page);

		await resizeWall(page, 720, 900);
		await resizeWall(page, 1600, 900);
		await resizeWall(page, 1280, 900);
	});
});
