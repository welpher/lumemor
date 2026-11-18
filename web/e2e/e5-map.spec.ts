// The networking & map story of the web app, walked end to end
// against the real compose stack: the admin settings' three new cards
// (the release check, the map tile source, the model download policy),
// the places map tab (the tile stubs, the pins, the pin click), the
// two honest degradation faces (a source that fails, no source at
// all), and the viewer's capture-location card.
//
// THE SEAL: the tile source is a `.invalid` host stubbed with
// `page.route` -- the TileJSON document and every tile image come from
// this repository (the fixture PNG under e2e/fixtures), so the scene
// never reaches the outside world even though the page really loads
// map tiles. The release check is switched off at the deployment level
// for this stack (LUMEMOR_UPDATE_CHECK_DISABLED=1 in the run's
// environment), so the settings card renders its disabled face and no
// outbound request is made at all.
//
// THE STORY (serial, one stack lifetime):
//
//  1. Signing in lands on the timeline; the session is saved for the
//     later scenes.
//  2. Admin settings: the three cards render; the release check shows
//     the disabled face with the running version; the light and dark
//     tile sources are empty (the shipped state) and are saved
//     through the form.
//  3. The GPS fixture uploads through the asset API and its pipeline
//     completes (the place resolution needs the metadata job).
//  4. The places map tab: the stubbed source draws, the city pin shows
//     its count, and the click opens the place panel in place (the
//     place domain's own face; the pin no longer navigates).
//  5. The theme switch: the light theme loads the light source, the
//     dark theme loads the dark one, and the pin survives the swap.
//  6. A source that fails: the map states the missing base layer and
//     keeps the pin.
//  7. No source at all: the same promise, worded for the configuration.
//  8. The viewer's location card: the asset's own coordinates, the
//     mini map over them, and the point's marker on it.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

const ADMIN_EMAIL = 'admin@e5-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The GPS fixture: its EXIF coordinates resolve to Beijing, so the
// places map gains exactly one pin after its pipeline completes.
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));
const PHOTO_GPS = path.join(repoRoot, 'server/internal/jobs/testdata/gps-beijing.jpg');
const PHOTO_GPS_NAME = 'IMG_E5_GPS.jpg';

// The stubbed tile host. `.invalid` never resolves, so a leaked
// request fails loudly instead of reaching a real tile service. Two
// distinguishable sources: the light style and its dark twin, each
// with its own attribution and tile path -- the theme switch's proof.
const TILE_HOST = 'https://tiles.e5-e2e.invalid';
const TILE_SOURCE = `${TILE_HOST}/tilejson.json`;
const TILE_DARK_SOURCE = `${TILE_HOST}/dark/tilejson.json`;
const LIGHT_ATTRIBUTION = 'Test tiles';
const DARK_ATTRIBUTION = 'Test dark tiles';
// The failing source: the same host, a path the stub answers 500 for.
const BROKEN_SOURCE = `${TILE_HOST}/broken/tilejson.json`;

// The 256x256 fixture tile every stubbed tile request answers.
const tileFixture = await readFile(path.join(specDir, 'fixtures/tile-256.png'));

// The storage-state file the sign-in scene writes and every later
// scene replays (git-ignored, recreated per run).
const adminStatePath = `${specDir}/.auth/e5-admin-state.json`;

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap: the setup API creates the
	// administrator on the fresh instance.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'E5 E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

/**
 * Stub the tile host: the TileJSON descriptors for the light and dark
 * sources, a 500 for the broken one, and the fixture PNG for every
 * tile image. Cross-origin headers ride each descriptor answer (the
 * page fetches it through fetch(), which enforces them).
 */
async function stubTiles(page: Page): Promise<void> {
	await page.route(`${TILE_HOST}/**`, async (route) => {
		const url = new URL(route.request().url());
		if (url.pathname === '/broken/tilejson.json') {
			await route.fulfill({ status: 500, body: 'broken' });
			return;
		}
		const descriptor =
			url.pathname === '/tilejson.json'
				? { path: '/tiles', attribution: LIGHT_ATTRIBUTION }
				: url.pathname === '/dark/tilejson.json'
					? { path: '/dark/tiles', attribution: DARK_ATTRIBUTION }
					: null;
		if (descriptor !== null) {
			await route.fulfill({
				status: 200,
				headers: {
					'Content-Type': 'application/json',
					'Access-Control-Allow-Origin': '*'
				},
				body: JSON.stringify({
					tilejson: '3.0.0',
					tiles: [`${TILE_HOST}${descriptor.path}/{z}/{x}/{y}.png`],
					tileSize: 256,
					minzoom: 0,
					maxzoom: 14,
					attribution: descriptor.attribution
				})
			});
			return;
		}
		await route.fulfill({ status: 200, headers: { 'Content-Type': 'image/png' }, body: tileFixture });
	});
}

/**
 * The pin's box once the initial camera fit has settled. The first
 * frame draws the overlay at the pre-fit projection, so a measurement
 * taken then would read as a camera move the theme switch never made.
 */
async function settledPinBox(page: Page): Promise<{ x: number; y: number }> {
	let previous = '';
	for (let attempt = 0; attempt < 50; attempt++) {
		const box = await page.locator('.map__pin').boundingBox();
		const key = box === null ? '' : `${box.x.toFixed(1)},${box.y.toFixed(1)}`;
		if (key !== '' && key === previous) {
			return { x: box!.x, y: box!.y };
		}
		previous = key;
		await page.waitForTimeout(100);
	}
	throw new Error('the pin never settled');
}

/**
 * Save the tile sources through the admin API (the settings' own
 * write): the light style and, when given, the dark one.
 */
async function setTileSource(page: Page, tilejsonUrl: string, darkTilejsonUrl = ''): Promise<void> {
	const response = await page.request.put('/api/v1/admin/settings/tiles', {
		data: { tilejsonUrl, darkTilejsonUrl }
	});
	expect(response.status()).toBe(200);
}

test('scene 1: the administrator signs in and lands on the timeline', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);

	// Hand the admin session to every later scene: the storage state
	// rides the login cookie (the fresh contexts replay it).
	await page.context().storageState({ path: adminStatePath });
});

test.describe('the signed-in scenes', () => {
	test.use({ storageState: adminStatePath });

	test('scene 2: the admin settings render the three networking cards and save both tile sources', async ({
		page
	}) => {
		await page.goto('/admin/settings');

		// The settings page is the collapsed index (E33): each card's
		// facts live inside its fold.
		const openCard = async (testId: string): Promise<void> => {
			await page.getByTestId(testId).locator('summary').click();
		};
		await openCard('admin-settings-update');
		await openCard('admin-settings-tiles');
		await openCard('admin-settings-models');

		// The release check card: the running version, the disabled
		// face (this stack's deployment switch), and no danger tone.
		const updateCard = page.getByTestId('admin-settings-update');
		await expect(updateCard).toBeVisible();
		await expect(updateCard).toContainText('Current version');
		await expect(page.getByTestId('update-check-state')).toContainText('Checks are off.');
		await expect(page.getByTestId('update-check-now')).toBeDisabled();
		// The boundary copy stands where the knobs live: only the
		// version is compared, no photo leaves the instance.
		await expect(updateCard).toContainText('no photo');

		// The tile source card: the shipped state is empty in both
		// fields, and the privacy boundary is stated where the switch
		// lives.
		const tilesCard = page.getByTestId('admin-settings-tiles');
		await expect(tilesCard).toBeVisible();
		const sourceField = page.locator('#admin-tiles-source');
		const darkSourceField = page.locator('#admin-tiles-dark-source');
		await expect(sourceField).toHaveValue('');
		await expect(darkSourceField).toHaveValue('');
		await expect(tilesCard).toContainText('never fetches or proxies a tile');
		// The dark field names its fallback, and the recommended
		// sources ride beneath the pair (copy, not a bundled default).
		await expect(tilesCard).toContainText('Dark style (optional)');
		await expect(tilesCard).toContainText('Leave empty to reuse the light source in dark mode.');
		await expect(tilesCard).toContainText('Recommended sources: OpenFreeMap');

		// Saving both sources through the form (the admin's own path,
		// not the API seeding the later scenes use).
		await sourceField.fill(TILE_SOURCE);
		await darkSourceField.fill(TILE_DARK_SOURCE);
		await tilesCard.getByRole('button', { name: 'Save' }).click();
		await expect(tilesCard).toContainText('The map picks it up on its next load');

		// The persisted pair reads back into both fields.
		const saved = await page.request.get('/api/v1/admin/settings/tiles');
		expect(saved.status()).toBe(200);
		expect(await saved.json()).toMatchObject({
			tilejsonUrl: TILE_SOURCE,
			darkTilejsonUrl: TILE_DARK_SOURCE
		});

		// The model download policy card: the automatic fill is on by
		// default, and the copy names the one case it covers.
		const modelsCard = page.getByTestId('admin-settings-models');
		await expect(modelsCard).toBeVisible();
		await expect(page.getByTestId('models-auto-download')).toBeChecked();
		await expect(modelsCard).toContainText('wiped model store');
	});

	test('scene 3: the GPS fixture lands and its place resolves', async ({ page }) => {
		const upload = await page.request.post('/api/v1/assets', {
			multipart: {
				deviceAssetId: 'e5-e2e-gps',
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

		// The pipeline's completion: the metadata job resolves the GPS
		// position into a place, so the places read gains the city.
		await expect
			.poll(
				async () => {
					const places = await page.request.get('/api/v1/places');
					expect(places.status()).toBe(200);
					const body = (await places.json()) as Array<{ placeId: number; name: string; latitude: number }>;
					return body.length;
				},
				{ timeout: 120_000, intervals: [1000, 2000, 5000] }
			)
			.toBe(1);
	});

	test('scene 4: the places map draws the stubbed tiles and the city pin opens its assets', async ({ page }) => {
		await stubTiles(page);
		await page.goto('/places?mode=map');

		// The map card and its footer: the counts of the grouping and
		// the source's own attribution.
		await expect(page.getByTestId('places-map')).toBeVisible();
		await expect(page.locator('.places__map-counts')).toContainText('1 city');
		await expect(page.locator('.places__map-counts')).toContainText('1 photo');
		await expect(page.locator('.places__map-attribution')).toContainText('Test tiles');

		// The stubbed source loaded: no degraded notice.
		await expect(page.locator('[data-map-state]')).toHaveCount(0);
		await expect(page.locator('.maplibregl-canvas')).toBeVisible();

		// The canvas FILLS the card: the map library's container is
		// `position: relative`, so a positioned utility collapses it to
		// zero height while the tiles still fetch (the bug this scene
		// guards). A container that does not match the card is the
		// regression, whether or not the canvas element "is visible".
		const mapBox = await page.locator('.maplibregl-map').boundingBox();
		const cardBox = await page.getByTestId('places-map').boundingBox();
		expect(mapBox?.height ?? 0).toBeGreaterThan(100);
		expect(Math.abs((mapBox?.height ?? 0) - (cardBox?.height ?? 0))).toBeLessThan(8);

		// One city, one pin, its honest count in the accessible name.
		const pin = page.locator('.map__pin');
		await expect(pin).toHaveCount(1);
		await expect(pin).toContainText('1');
		await expect(pin).toHaveAttribute('aria-label', /1 photo/);
		await expect(pin.locator('.map__pin-name')).toHaveText('Beijing');

		// The point is the NUMBER alone (no icon: a map point already
		// says "photos here"), and its size reads the count -- one
		// photo is the ladder's smallest rung, 26px. Both are the
		// design's pins, not decoration.
		await expect(pin.locator('svg')).toHaveCount(0);
		const pillBox = await pin.locator('.map__pin-pill').boundingBox();
		expect(Math.round(pillBox?.height ?? 0)).toBe(26);

		// The pin click opens the place panel IN PLACE: the page never
		// navigates, and the panel's wall carries the city's photo.
		await pin.click();
		await expect(page).toHaveURL(/\/places\?mode=map&place=\d+$/);
		const panel = page.getByTestId('places-panel');
		await expect(panel).toBeVisible();
		await expect(panel.locator('[data-place-panel-name]')).toHaveText('Beijing');
		await expect(panel.locator('.tile').first().getByRole('link')).toHaveAccessibleName(PHOTO_GPS_NAME);

		// The same pin collapses it again.
		await pin.click();
		await expect(panel).toHaveCount(0);
	});

	test('scene 5: the theme switch moves the base layer and keeps the pin', async ({ page }) => {
		await stubTiles(page);
		await setTileSource(page, TILE_SOURCE, TILE_DARK_SOURCE);
		await page.goto('/places?mode=map');

		// The light theme loads the light source.
		const attribution = page.locator('.places__map-attribution');
		await expect(attribution).toContainText(LIGHT_ATTRIBUTION);
		await expect(page.locator('.map__pin')).toHaveCount(1);
		await expect(page.locator('[data-map-state]')).toHaveCount(0);

		// The system theme flips to dark: the base layer is swapped in
		// place (no reload, the pin stays a DOM marker), and the
		// untouched camera keeps the pin where it was.
		const pinBefore = await settledPinBox(page);
		await page.emulateMedia({ colorScheme: 'dark' });
		await expect(attribution).toContainText(DARK_ATTRIBUTION);
		await expect(page.locator('.map__pin')).toHaveCount(1);
		await expect(page.locator('.map__pin-name')).toHaveText('Beijing');
		await expect(page.locator('[data-map-state]')).toHaveCount(0);
		const pinAfter = await settledPinBox(page);
		expect(pinAfter.x).toBeCloseTo(pinBefore.x, 0);
		expect(pinAfter.y).toBeCloseTo(pinBefore.y, 0);

		// Back to light: the light source returns.
		await page.emulateMedia({ colorScheme: 'light' });
		await expect(attribution).toContainText(LIGHT_ATTRIBUTION);
		await expect(page.locator('.map__pin')).toHaveCount(1);
	});

	test('scene 6: a source that fails states the missing base layer and keeps the pin', async ({ page }) => {
		await stubTiles(page);
		await setTileSource(page, BROKEN_SOURCE);
		await page.goto('/places?mode=map');

		// The honest degradation: the notice names the failure, the
		// pin stays.
		const notice = page.locator('[data-map-state="unavailable"]');
		await expect(notice).toBeVisible();
		await expect(notice).toContainText('Tiles unavailable');
		await expect(notice).toContainText('pins stay available');
		await expect(page.locator('.map__pin')).toHaveCount(1);
	});

	test('scene 7: no source configured states it and keeps the pin', async ({ page }) => {
		await stubTiles(page);
		await setTileSource(page, '');
		await page.goto('/places?mode=map');

		const notice = page.locator('[data-map-state="unconfigured"]');
		await expect(notice).toBeVisible();
		await expect(notice).toContainText('No tile source configured');
		await expect(page.locator('.map__pin')).toHaveCount(1);

		// The switch is a link: the list face is one click away.
		await page.locator('[data-places-mode="list"]').click();
		await expect(page).toHaveURL(/\/places$/);
		await expect(page.locator('.place-card')).toHaveCount(1);
	});

	test('scene 8: the viewer location card carries the coordinates and the mini map', async ({ page }) => {
		await stubTiles(page);
		await setTileSource(page, TILE_SOURCE);

		// The asset's id from the timeline (the story's one photo).
		const timeline = await page.request.get('/api/v1/assets/timeline');
		expect(timeline.status()).toBe(200);
		const body = (await timeline.json()) as { items?: Array<{ id: string }> };
		const assetId = body.items?.[0]?.id;
		expect(assetId).toBeTruthy();

		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto(`/a/${assetId}`);
		// The info panel is the wide layout's docked region; the topbar
		// button opens it.
		await page.getByRole('button', { name: 'Photo information' }).click();
		const card = page.getByTestId('viewer-location');
		await expect(card).toBeVisible();

		// The coordinates are the file's own fact, printed to four
		// decimals; the mini map draws the same stubbed tiles.
		await expect(card).toContainText(/-?\d+\.\d{4}, -?\d+\.\d{4}/);
		await expect(card.locator('.maplibregl-canvas')).toBeVisible();
		// The point's own marker: the map handle is not a signal, so
		// this pins the overlay following the map's readiness (a
		// derived would have cached the first render's empty state).
		await expect(card.locator('.map__asset-pin')).toBeVisible();
		await expect(card.locator('[data-map-state]')).toHaveCount(0);
	});
});
