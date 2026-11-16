// The place domain's story of the web app, walked end to end against
// the real compose stack: the flat city list (the country and capture
// years on every card, the most recent city first), the place's own
// wall (the context bar with no right-side action, the month axis,
// the curation subset), and the map that fills the viewport and opens
// its selection panel in place (70/30, the camera framed on the spot,
// the shared wall with no horizontal scrollbar), including the viewer
// round trip and the honest tile degradation.
//
// THE SEAL: the tile source is a `.invalid` host stubbed with
// `page.route` -- the TileJSON descriptor and every tile image come
// from this repository (the fixture PNG), so the scene never reaches
// the outside world.
//
// THE SCENES (serial, one stack lifetime):
//
//  1. The administrator signs in through the form.
//  2. The GPS fixture uploads through the asset API and the place
//     resolution lands (the places grouping gains its one city).
//  3. The list face: one card with the city name, the country and
//     the capture year, the honest count, the place link, no grouping
//     headers.
//  4. The place detail: the card opens the place's own wall; the bar
//     carries back + city + country + count and nothing on the right;
//     the wall's month header and tile (its row target riding the
//     container width: 192 at the 1280 viewport, the 162 floor at
//     1024, 218 at 1440, never sideways); the selection's batch
//     subset.
//  5. The map face: the stubbed source draws, the page does not
//     scroll, the footer counts, and the pin opens the side panel
//     without leaving the page (70/30, the wall, the axis, its rows
//     packing below the page wall's at the same viewport).
//  6. The viewer round trip: a panel photo opens the viewer, and
//     coming back restores the same panel.
//  7. The degradation: a source that fails withdraws the base layer
//     and keeps the pin, stated in place.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

const ADMIN_EMAIL = 'admin@e19-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The GPS fixture: its EXIF coordinates resolve to Beijing, so the
// places domain gains exactly one city after its pipeline completes.
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));
const PHOTO_GPS = path.join(repoRoot, 'server/internal/jobs/testdata/gps-beijing.jpg');
const PHOTO_GPS_NAME = 'IMG_E19_GPS.jpg';

// The stubbed tile host. `.invalid` never resolves, so a leaked
// request fails loudly instead of reaching a real tile service.
const TILE_HOST = 'https://tiles.e19-e2e.invalid';
const TILE_SOURCE = `${TILE_HOST}/tilejson.json`;
const BROKEN_SOURCE = `${TILE_HOST}/broken/tilejson.json`;

// The 256x256 fixture tile every stubbed tile request answers.
const tileFixture = await readFile(path.join(specDir, 'fixtures/tile-256.png'));

// The storage-state file the sign-in scene writes and every later
// scene replays (git-ignored, recreated per run).
const adminStatePath = `${specDir}/.auth/e19-admin-state.json`;

// The place wall's row height at the 1440 viewport, measured by
// scene 4: scene 5's panel is the same wall in a narrower container,
// so its rows must pack smaller than the page's at the same width.
let placeWallRowAt1440 = 0;

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap: the setup API creates the
	// administrator on the fresh instance.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'E19 E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

/**
 * Stub the tile host: the TileJSON descriptor for the working source,
 * a 500 for the broken one, and the fixture PNG for every tile image.
 */
async function stubTiles(page: Page): Promise<void> {
	await page.route(`${TILE_HOST}/**`, async (route) => {
		const url = new URL(route.request().url());
		if (url.pathname === '/broken/tilejson.json') {
			await route.fulfill({ status: 500, body: 'broken' });
			return;
		}
		if (url.pathname === '/tilejson.json') {
			await route.fulfill({
				status: 200,
				headers: {
					'Content-Type': 'application/json',
					'Access-Control-Allow-Origin': '*'
				},
				body: JSON.stringify({
					tilejson: '3.0.0',
					tiles: [`${TILE_HOST}/tiles/{z}/{x}/{y}.png`],
					tileSize: 256,
					minzoom: 0,
					maxzoom: 14,
					attribution: 'Test tiles'
				})
			});
			return;
		}
		await route.fulfill({ status: 200, headers: { 'Content-Type': 'image/png' }, body: tileFixture });
	});
}

/** Save the tile source through the admin API (the settings' own write). */
async function setTileSource(page: Page, tilejsonUrl: string): Promise<void> {
	const response = await page.request.put('/api/v1/admin/settings/tiles', { data: { tilejsonUrl } });
	expect(response.status()).toBe(200);
}

/** The one place's id from the grouping read. */
async function placeId(page: Page): Promise<number> {
	const response = await page.request.get('/api/v1/places');
	expect(response.status()).toBe(200);
	const body = (await response.json()) as Array<{ placeId: number }>;
	expect(body.length).toBe(1);
	return body[0]!.placeId;
}

test('scene 1: the administrator signs in and lands on the timeline', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);

	await page.context().storageState({ path: adminStatePath });
});

test.describe('the signed-in scenes', () => {
	test.use({ storageState: adminStatePath });

	test('scene 2: the GPS fixture lands and its place resolves', async ({ page }) => {
		const upload = await page.request.post('/api/v1/assets', {
			multipart: {
				deviceAssetId: 'e19-e2e-gps',
				fileName: PHOTO_GPS_NAME,
				deviceCreatedAt: '2026-06-20T12:00:00Z',
				deviceTimeZone: 'UTC',
				photo: {
					name: PHOTO_GPS_NAME,
					mimeType: 'image/jpeg',
					buffer: await readFile(PHOTO_GPS)
				}
			}
		});
		expect(upload.status()).toBe(201);

		await expect
			.poll(
				async () => {
					const places = await page.request.get('/api/v1/places');
					expect(places.status()).toBe(200);
					const body = (await places.json()) as Array<{ assetCount: number }>;
					return body[0]?.assetCount ?? 0;
				},
				{ timeout: 120_000, intervals: [1000, 2000, 5000] }
			)
			.toBe(1);
	});

	test('scene 3: the flat list carries the span, the count and the place link', async ({ page }) => {
		await page.goto('/places');
		await expect(page).toHaveTitle('Places · lumemor');
		await expect(page.getByRole('heading', { level: 1, name: 'Places · 1 city', exact: true })).toBeVisible();

		// One flat grid: no grouping headers above the cards, and the
		// grid's own children are the cards -- nothing else rides
		// between them.
		expect(await page.locator('.place-card').count()).toBe(1);
		await expect(page.locator('.places__content h2')).toHaveCount(0);
		const gridChildren = await page
			.locator('.places__grid > *')
			.evaluateAll((nodes) => nodes.map((node) => node.tagName.toLowerCase()));
		expect(gridChildren).toEqual(['a']);

		const card = page.locator('a.place-card').first();
		await expect(card.locator('.place-card__name')).toHaveText('Beijing');
		// The second line: the country and the capture years (one year
		// -- the single fixture photo's span).
		await expect(card.locator('.place-card__years')).toHaveText('China · 2026');
		await expect(card.locator('.place-card__count')).toHaveAttribute('aria-label', '1 photo');

		// The card opens the place's own page.
		const id = await placeId(page);
		await expect(card).toHaveAttribute('href', `/places/${id}`);
		await expect(page.locator('[data-places-mode="map"]')).toHaveAttribute('href', '/places?mode=map');
	});

	test('scene 4: the place page is its own wall with an actionless bar', async ({ page }) => {
		const id = await placeId(page);
		await page.goto('/places');
		await page.locator('a.place-card').first().click();
		await expect(page).toHaveURL(new RegExp(`/places/${id}$`));

		// The context bar: back, the city, the country and the count --
		// no right-side action at all.
		await expect(page.locator('.place-bar__back')).toHaveAttribute('aria-label', 'Back to places');
		await expect(page.locator('.place-bar__back')).toHaveAttribute('href', '/places');
		await expect(page.locator('.place-bar__title')).toHaveText('Beijing');
		await expect(page.locator('.place-bar__country')).toHaveText('China');
		await expect(page.locator('.place-bar__count')).toHaveAttribute('aria-label', '1 photo');
		await expect(page.locator('.place-bar button')).toHaveCount(0);

		// The wall: the month group and its axis, the place's photo.
		await expect(page.locator('.wall__month-header')).toHaveText('June 2026');
		await expect(page.locator('.scrubber')).toBeVisible();
		const tile = page.locator('.tile').first();
		await expect(tile.getByRole('link')).toHaveAccessibleName(PHOTO_GPS_NAME);

		// The curation subset on a selection: favorite / hide / trash.
		await tile.hover();
		await page.getByRole('checkbox', { name: `Select: ${PHOTO_GPS_NAME}` }).click();
		const bar = page.getByRole('toolbar');
		await expect(bar.getByRole('button', { name: 'Favorite' })).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Hide' })).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Trash' })).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Add tag' })).toHaveCount(0);
		await page.keyboard.press('Escape');

		// THE PROPORTIONAL SCALE: the row target is a fraction of the
		// wall's own content width (the viewport minus the edges and
		// the axis slot), so the photos shrink with the window instead
		// of only thinning the row -- the same tile at three widths,
		// with the narrow one resting on the scale's floor, and no
		// width overflowing (the always-full, never-sideways rule).
		const geometry = () =>
			page.evaluate(() => {
				const row = document.querySelector<HTMLElement>('.wall__row');
				const cell = document.querySelector<HTMLElement>('.wall__cell');
				const wall = document.querySelector<HTMLElement>('.wall__groups');
				return {
					row: row?.getBoundingClientRect().height ?? 0,
					cell: cell?.getBoundingClientRect().width ?? 0,
					scrollWidth: wall?.scrollWidth ?? 0,
					clientWidth: wall?.clientWidth ?? 0,
					pageOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth
				};
			});

		await page.setViewportSize({ width: 1280, height: 800 });
		await expect.poll(async () => Math.round((await geometry()).row)).toBe(192);
		const reference = await geometry();
		expect(reference.scrollWidth).toBeLessThanOrEqual(reference.clientWidth + 1);
		expect(reference.pageOverflow).toBeLessThanOrEqual(0);

		await page.setViewportSize({ width: 1024, height: 800 });
		await expect.poll(async () => Math.round((await geometry()).row)).toBe(162);
		const narrow = await geometry();
		expect(narrow.cell).toBeLessThan(reference.cell);
		expect(narrow.scrollWidth).toBeLessThanOrEqual(narrow.clientWidth + 1);

		await page.setViewportSize({ width: 1440, height: 900 });
		await expect.poll(async () => Math.round((await geometry()).row)).toBe(218);
		const wide = await geometry();
		expect(wide.cell).toBeGreaterThan(reference.cell);
		expect(wide.scrollWidth).toBeLessThanOrEqual(wide.clientWidth + 1);
		placeWallRowAt1440 = Math.round(wide.row);
		expect(placeWallRowAt1440).toBeGreaterThan(0);
		await page.setViewportSize({ width: 1280, height: 720 });
	});

	test('scene 5: the map fills the viewport and the pin opens the panel in place', async ({ page }) => {
		await stubTiles(page);
		await setTileSource(page, TILE_SOURCE);
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto('/places?mode=map');

		// The map face replaces the list and consumes the frame: the
		// page itself does not scroll.
		await expect(page.getByTestId('places-map')).toBeVisible();
		const frame = await page.evaluate(() => {
			const main = document.querySelector('main');
			return {
				scrollHeight: main?.scrollHeight ?? 0,
				clientHeight: main?.clientHeight ?? 0
			};
		});
		expect(frame.scrollHeight).toBeLessThanOrEqual(frame.clientHeight + 1);

		// The stubbed source loaded (no degraded notice) and the footer
		// carries the counts and the source's attribution.
		await expect(page.locator('[data-map-state]')).toHaveCount(0);
		await expect(page.locator('.places__map-counts')).toContainText('1 city');
		await expect(page.locator('.places__map-counts')).toContainText('1 photo');
		await expect(page.locator('.places__map-attribution')).toContainText('Test tiles');

		// One city pin, its aria-label stating the photo count.
		const pin = page.locator('.map__pin').first();
		await expect(pin).toHaveAttribute('aria-label', /1 photo/);
		await expect(pin.locator('.map__pin-name')).toHaveText('Beijing');

		// The pin click opens the panel: the address stays the map
		// face (the selection rides it), nothing navigates.
		await pin.click();
		await expect(page).toHaveURL(/\/places\?mode=map&place=\d+$/);
		const panel = page.getByTestId('places-panel');
		await expect(panel).toBeVisible();
		await expect(panel.locator('[data-place-panel-name]')).toHaveText('Beijing');
		await expect(panel.locator('.places__panel-count')).toHaveAttribute('aria-label', '1 photo');

		// The panel is the shared wall in its narrow frame: the month
		// header and the compact axis, and no selection state.
		await expect(panel.locator('.wall__month-header')).toHaveText('June 2026');
		await expect(panel.getByTestId('panel-axis')).toBeVisible();
		await expect(panel.locator('.tile__check')).toHaveCount(0);
		await expect(panel.locator('.batchbar')).toHaveCount(0);

		// The 70/30 squeeze: the map keeps ~70% of the row.
		const split = await page.evaluate(() => {
			const row = document.querySelector('[data-place-row]');
			const map = document.querySelector('.places__map-card');
			if (row === null || map === null) {
				return { row: 0, map: 0 };
			}
			return { row: row.getBoundingClientRect().width, map: map.getBoundingClientRect().width };
		});
		expect(split.map / split.row).toBeGreaterThan(0.62);
		expect(split.map / split.row).toBeLessThan(0.74);

		// The no-horizontal-scrollbar discipline: the panel's own root
		// never overflows its content box.
		const overflow = await page.evaluate(() => {
			const wall = document.querySelector('.wall__groups--panel');
			return wall === null
				? { scrollWidth: 0, clientWidth: 0 }
				: {
						scrollWidth: wall.scrollWidth,
						clientWidth: wall.clientWidth
					};
		});
		expect(overflow.scrollWidth).toBeLessThanOrEqual(overflow.clientWidth + 1);

		// THE PANEL'S OWN SCALE: the same wall in the narrow frame, so
		// its rows pack below the page wall's at the same viewport
		// (scene 4's 1440 measurement) -- the panel shrinks its photos
		// too, which is the scale reaching every container.
		const panelRow = await page.evaluate(
			() =>
				document.querySelector<HTMLElement>('.wall__groups--panel .wall__row')?.getBoundingClientRect()
					.height ?? 0
		);
		expect(panelRow).toBeGreaterThan(0);
		expect(panelRow).toBeLessThan(placeWallRowAt1440);

		// The same pin again collapses the panel (and the address).
		await pin.click();
		await expect(panel).toHaveCount(0);
		await expect(page).toHaveURL(/\/places\?mode=map$/);
	});

	test('scene 6: the viewer round trip comes back to the same panel', async ({ page }) => {
		await stubTiles(page);
		await setTileSource(page, TILE_SOURCE);
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto('/places?mode=map');

		const pin = page.locator('.map__pin').first();
		await pin.click();
		const panel = page.getByTestId('places-panel');
		await expect(panel).toBeVisible();

		// A panel photo opens the viewer; Escape returns to the map.
		await panel.locator('.tile').first().getByRole('link').click();
		await expect(page).toHaveURL(/\/a\/[0-9a-f-]+$/);
		await page.keyboard.press('Escape');
		await expect(page).toHaveURL(/\/places\?mode=map&place=\d+$/);

		// The panel is back with the same spot (the address carried it).
		await expect(page.getByTestId('places-panel')).toBeVisible();
		await expect(page.locator('[data-place-panel-name]')).toHaveText('Beijing');
	});

	test('scene 7: a failing source withdraws the base layer and keeps the pin', async ({ page }) => {
		await stubTiles(page);
		await setTileSource(page, BROKEN_SOURCE);
		await page.setViewportSize({ width: 1280, height: 800 });
		await page.goto('/places?mode=map');

		const notice = page.locator('[data-map-state="unavailable"]');
		await expect(notice).toBeVisible();
		await expect(notice).toContainText('Tiles unavailable');
		await expect(notice).toContainText('pins stay available');
		// The pin stays: the honest degradation never blanks the map.
		await expect(page.locator('.map__pin').first()).toBeVisible();

		// The pin still opens its panel without the base layer.
		await page.locator('.map__pin').first().click();
		await expect(page.getByTestId('places-panel')).toBeVisible();

		// The list face is one click away and keeps the card.
		await page.locator('[data-places-mode="list"]').click();
		await expect(page).toHaveURL(/\/places$/);
		await expect(page.locator('.place-card')).toHaveCount(1);
	});

	test('scene 8: the dark theme keeps the map face and its panel readable', async ({ page }) => {
		await stubTiles(page);
		await setTileSource(page, TILE_SOURCE);
		await page.addInitScript(() => {
			window.localStorage.setItem('lumemor:theme', 'dark');
		});
		await page.setViewportSize({ width: 1920, height: 1080 });
		await page.goto('/places?mode=map');

		await expect(page.getByTestId('places-map')).toBeVisible();
		await page.locator('.map__pin').first().click();
		const panel = page.getByTestId('places-panel');
		await expect(panel).toBeVisible();
		await expect(panel.locator('.wall__month-header')).toHaveText('June 2026');
		// The panel's root still never overflows.
		const overflow = await page.evaluate(() => {
			const wall = document.querySelector('.wall__groups--panel');
			return wall === null
				? { scrollWidth: 0, clientWidth: 0 }
				: {
						scrollWidth: wall.scrollWidth,
						clientWidth: wall.clientWidth
					};
		});
		expect(overflow.scrollWidth).toBeLessThanOrEqual(overflow.clientWidth + 1);
	});
});
