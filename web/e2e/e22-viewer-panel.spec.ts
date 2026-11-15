// The viewer's final form, walked end to end against the real compose
// stack: the info panel's four sections with their five edit arms, and
// the stage's chrome (the contain rule, the floating bar, the
// card-less filmstrip, the three-step arrows, the panel switch).
//
// THE STACK ARRANGEMENT: one fresh lifetime owns this spec (the
// first-run bootstrap creates the administrator, and the seeds upload
// through the asset API). The photos are the repository's own fixtures
// -- a 2:1 landscape and an EXIF-rotated portrait (the contain rule
// needs both shapes), plus a GPS-carrying landscape whose pipeline
// resolves a place (the location row and its dialog).
//
// THE STORY (serial, one stack lifetime):
//
//  1. The administrator signs in through the form and lands on the
//     timeline; the session is saved for the later scenes.
//  2. The stage: the media box wears the photo's own aspect and touches
//     one stage edge, the bar floats over the photo with no filename,
//     the filmstrip sits card-less in the bottom-left corner with
//     translucent thumbs, the arrows rest at 45%, and the zoom walks
//     its ladder (the `+` key, the double-click, the reset on a switch).
//  3. The panel: four sections in order (the one-line description, the
//     people, the info rows), the description's click-edit with Enter
//     and Esc, the tag chips' removal and the add-tag dialog, the
//     admin-only path row, and the remembered open preference.
//  4. The date row: the zone on screen, the pencil's dialog writing the
//     wall clock and its zone as one value.
//  5. The drawn face: the add-person icon arms the drag layer, a square
//     drawn on the photo becomes a face row, its chip joins the People
//     section, and the assign flow opens over it.
//  6. The panel is one switch: the bar's pressed state, the panel's
//     close mark and the `i` key drive the same state, and closing it
//     widens the stage (the media re-fits, the filmstrip stays put).
//  7. The location row: the honest degradation with no tile source (the
//     coordinates stand, the map says what is missing), then the
//     instance's own tile source draws the mini map, and the edit
//     dialog's search and map click move the pin.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type APIRequestContext, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's administrator: created through the first-run setup API
// in the suite's bootstrap. The stack is always fresh, so the fixed
// identity is deterministic.
const ADMIN_EMAIL = 'admin@e22-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The seeds. The landscape is 2:1 (width-bound on a 1280x720 stage),
// the portrait is an EXIF-rotated 200x100 frame (the derivatives bake
// the orientation in, so the viewer sees 1:2), and the GPS photo
// resolves Beijing.
const LAND_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/golden-large.jpg');
const PORT_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/portrait-orientation6.jpg');
const GPS_SOURCE = path.join(repoRoot, 'server/internal/jobs/testdata/gps-beijing.jpg');
const LAND_NAME = 'IMG_E22_LAND.jpg';
const PORT_NAME = 'IMG_E22_PORT.jpg';
const GPS_NAME = 'IMG_E22_GPS.jpg';
const LAND_DAY = '2026-09-10T09:00:00Z';
const PORT_DAY = '2026-09-11T09:00:00Z';
const GPS_DAY = '2026-09-12T09:00:00Z';

// The storage-state file the sign-in scene writes and every later
// scene replays (git-ignored, recreated per run).
const authStatePath = path.join(specDir, '.auth', 'e22-state.json');

// The stubbed tile host of the location scene. `.invalid` never
// resolves, so a leaked request fails loudly instead of reaching a
// real tile service.
const TILE_HOST = 'https://tiles.e22-e2e.invalid';
const TILE_SOURCE = `${TILE_HOST}/tilejson.json`;
const tileFixture = await readFile(path.join(specDir, 'fixtures/tile-256.png'));

let landId = '';
let portId = '';
let gpsId = '';

/** One wall tile, addressed by its accessible name (the file name). */
function tile(page: Page, name: string) {
	return page.getByRole('link', { name, exact: true });
}

/** The viewer's info panel (the aside the bar's toggle controls). */
function panel(page: Page) {
	return page.getByRole('complementary', { name: 'Photo information' });
}

/** The media box's size (the contain rule's output). */
async function mediaBox(page: Page): Promise<{ width: number; height: number }> {
	const box = await page.locator('.viewer__media').boundingBox();
	if (box === null) {
		throw new Error('the media box never landed');
	}
	return box;
}

/** Upload one photo through the real upload API, returning the asset id. */
async function uploadPhoto(
	api: APIRequestContext,
	name: string,
	source: string,
	takenAt: string,
	index: number
): Promise<string> {
	// Distinct bytes per seed: the content-addressed dedupe would
	// otherwise answer with the existing asset instead of a new one.
	const bytes = Buffer.concat([await readFile(source), Buffer.from(`\nE22-E2E-SEED-${index}\n`)]);
	const response = await api.post('/api/v1/assets', {
		multipart: {
			deviceAssetId: `e22-e2e-${index}`,
			fileName: name,
			deviceCreatedAt: takenAt,
			deviceTimeZone: 'UTC',
			photo: { name, mimeType: 'image/jpeg', buffer: bytes }
		}
	});
	expect(response.status(), `the upload of ${name}`).toBe(201);
	const created = (await response.json()) as { id: string };
	return created.id;
}

/** The pipeline's completion signal: both derivative sizes answer 200. */
async function waitPipelineDone(api: APIRequestContext, assetId: string): Promise<void> {
	for (const size of ['tile', 'view']) {
		await expect
			.poll(async () => (await api.get(`/api/v1/assets/${assetId}/thumbnail?size=${size}`)).status(), {
				timeout: 180_000,
				intervals: [500, 1000, 2000]
			})
			.toBe(200);
	}
}

/**
 * Stub the tile host: the TileJSON descriptor and the fixture PNG for
 * every tile image. Cross-origin headers ride the descriptor answer
 * (the page fetches it through fetch(), which enforces them).
 */
async function stubTiles(page: Page): Promise<void> {
	await page.route(`${TILE_HOST}/**`, async (route) => {
		const url = new URL(route.request().url());
		if (url.pathname === '/tilejson.json') {
			await route.fulfill({
				status: 200,
				headers: { 'Content-Type': 'application/json', 'Access-Control-Allow-Origin': '*' },
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

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap, the acceptance scripts' pattern.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'E22 E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	const login = await api.post('/api/v1/auth/login', {
		data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(login.status()).toBe(200);
	landId = await uploadPhoto(api, LAND_NAME, LAND_SOURCE, LAND_DAY, 1);
	portId = await uploadPhoto(api, PORT_NAME, PORT_SOURCE, PORT_DAY, 2);
	gpsId = await uploadPhoto(api, GPS_NAME, GPS_SOURCE, GPS_DAY, 3);
	await waitPipelineDone(api, landId);
	await waitPipelineDone(api, portId);
	await waitPipelineDone(api, gpsId);
	await api.dispose();
});

test('scene 1: the administrator signs in and lands on the timeline', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Ftimeline');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);
	await expect(tile(page, LAND_NAME)).toBeVisible();
	await expect(tile(page, PORT_NAME)).toBeVisible();
	await expect(tile(page, GPS_NAME)).toBeVisible();
	await page.context().storageState({ path: authStatePath });
});

test.describe('the viewer', () => {
	test.use({ storageState: authStatePath, permissions: ['clipboard-read', 'clipboard-write'] });

	test('scene 2: the stage fits each photo to its own aspect, the chrome floats over it', async ({ page }) => {
		await page.goto('/timeline');
		await tile(page, LAND_NAME).click();
		await expect(page).toHaveURL(/\/a\/[0-9a-f-]{36}$/);

		// The floating bar: no filename anywhere in it (the name is the
		// tab's and the panel's), and it draws OVER the photo -- the
		// media starts at the stage's top edge.
		const bar = page.locator('.viewer__bar');
		await expect(bar).toBeVisible();
		await expect(page).toHaveTitle(`${LAND_NAME} · lumemor`);
		await expect(bar).not.toContainText(LAND_NAME);
		const stageBox = await page.locator('.viewer__stage').boundingBox();
		const barBox = await bar.boundingBox();
		if (stageBox === null || barBox === null) {
			throw new Error('the stage geometry never landed');
		}
		// The stage starts at the viewport's top edge: the bar does not
		// push the photo down.
		expect(stageBox.y).toBe(0);

		// The contain rule: the 2:1 photo's box keeps its aspect and
		// touches the stage on one dimension (no percentage inset).
		await expect.poll(async () => (await mediaBox(page)).width).toBeGreaterThan(100);
		const land = await mediaBox(page);
		expect(land.width / land.height).toBeCloseTo(2, 1);
		const touches = (value: number, edge: number) => Math.abs(value - edge) < 2;
		expect(touches(land.width, stageBox.width) || touches(land.height, stageBox.height)).toBe(true);
		// The bar overlaps the photo's top edge: it floats, it does not
		// take a row of its own.
		expect(barBox.y + barBox.height).toBeGreaterThan(stageBox.y + 8);

		// The filmstrip: bottom-left, no card, the thumbs translucent
		// (the photo shows through) and the current one at full
		// strength with the accent edge.
		const filmstrip = page.getByRole('navigation', { name: 'Photos in view' });
		await expect(filmstrip).toBeVisible();
		const stripBox = await filmstrip.boundingBox();
		if (stripBox === null) {
			throw new Error('the filmstrip never landed');
		}
		expect(stripBox.x).toBeLessThan(stageBox.width * 0.3);
		expect(stripBox.y + stripBox.height).toBeGreaterThan(stageBox.y + stageBox.height - 40);
		const currentThumb = filmstrip.getByRole('button', { name: LAND_NAME, exact: true });
		const otherThumb = filmstrip.getByRole('button', { name: PORT_NAME, exact: true });
		await expect(currentThumb).toHaveAttribute('aria-current', 'true');
		await expect(currentThumb).toHaveCSS('opacity', '1');
		await expect(otherThumb).toHaveCSS('opacity', '0.65');

		// The three-step arrows: 45% at rest, 100% under the pointer.
		const next = page.getByRole('button', { name: 'Next photo', exact: true });
		await expect(next).toHaveCSS('opacity', '0.45');
		await next.hover();
		await expect(next).toHaveCSS('opacity', '1');

		// The zoom ladder: the `+` key one step, the double-click back
		// to 1x (and up to 2x from rest), every change the same
		// gradual engine the bar's buttons ride.
		const zoom = () =>
			page.evaluate(() =>
				(document.querySelector('.viewer__stage') as HTMLElement).style.getPropertyValue('--viewer-zoom')
			);
		await page.mouse.move(10, 10);
		await page.keyboard.press('+');
		await expect.poll(zoom).toBe('1.5');
		await page.locator('.viewer__media').dblclick();
		await expect.poll(zoom).toBe('1');
		await page.locator('.viewer__media').dblclick();
		await expect.poll(zoom).toBe('2');

		// A switch resets the zoom and re-fits the box to the new
		// photo's own aspect: the portrait's height fills the stage.
		// (The filmstrip is the switch here -- the wall's order comes
		// from the fixtures' own capture times.)
		await filmstrip.getByRole('button', { name: PORT_NAME, exact: true }).click();
		await expect(page).toHaveTitle(`${PORT_NAME} · lumemor`);
		await expect.poll(zoom).toBe('1');
		await expect.poll(async () => (await mediaBox(page)).height).toBeGreaterThan(100);
		const port = await mediaBox(page);
		expect(port.height / port.width).toBeCloseTo(2, 1);
		expect(touches(port.height, stageBox.height) || touches(port.width, stageBox.width)).toBe(true);
	});

	test('scene 2b: switching photos swaps the content, never the window', async ({ page }) => {
		// The acceptance walk's flicker: the switch used to empty the
		// route's record, which unmounted the drawer and the stage's
		// media box -- the window read as closing and reopening between
		// photos. The pin is DOM IDENTITY: the panel element that is
		// open before the switch is the same element after it, with the
		// new photo's facts as content.
		// Entered through the wall's tile (the switch needs the wall's
		// sequence: a deep link carries no neighbors).
		await page.goto('/timeline');
		await tile(page, LAND_NAME).click();
		await expect(page).toHaveURL(/\/a\/[0-9a-f-]{36}$/);
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = panel(page);
		await expect(drawer).toBeVisible();
		await expect(drawer.locator('.info__filename')).toHaveText(LAND_NAME);

		await page.evaluate(() => {
			(window as unknown as { __drawer?: Element | null }).__drawer = document.getElementById('viewer-info');
		});

		const filmstrip = page.getByRole('navigation', { name: 'Photos in view' });
		await filmstrip.getByRole('button', { name: PORT_NAME, exact: true }).click();
		await expect(page).toHaveTitle(`${PORT_NAME} · lumemor`);
		await expect(drawer.locator('.info__filename')).toHaveText(PORT_NAME);

		const sameElement = await page.evaluate(() => {
			const stash = (window as unknown as { __drawer?: Element | null }).__drawer;
			return stash !== null && stash !== undefined && stash === document.getElementById('viewer-info');
		});
		expect(sameElement).toBe(true);
		// The drawer's own loading face never stepped in for the
		// switch: that aside is a DIFFERENT element, and its presence
		// is the remount the pin forbids.
		await expect(page.getByTestId('viewer-info-state')).toHaveCount(0);

		// The stage's frame carried the swap too: the media box is the
		// same element across the switch (no tear-down per photo), and
		// the PAINTED FRAME survives -- the image element is not
		// replaced (a keyed destruction would flash the placeholder
		// between photos), and the box's width never collapses to the
		// stage's.
		const mediaIdentity = await page.evaluate(() => {
			const box = document.querySelector('.viewer__media');
			const image = box?.querySelector('.viewer__image');
			return { box: box ?? null, image: image ?? null };
		});
		await filmstrip.getByRole('button', { name: LAND_NAME, exact: true }).click();
		await expect(page).toHaveTitle(`${LAND_NAME} · lumemor`);
		const identityAfter = await page.evaluate(() => {
			const box = document.querySelector('.viewer__media');
			const image = box?.querySelector('.viewer__image');
			return { box: box ?? null, image: image ?? null };
		});
		expect(identityAfter.box).not.toBeNull();
		expect(identityAfter.image).not.toBeNull();
		expect(mediaIdentity.image === identityAfter.image).toBe(true);
		const boxWidth = await page.locator('.viewer__media').evaluate((node) => node.getBoundingClientRect().width);
		expect(boxWidth).toBeGreaterThan(100);
	});

	test('scene 3: the panel is four sections and five edit arms', async ({ page }) => {
		await page.goto(`/a/${landId}`);

		// The panel is CLOSED by default (the remembered preference):
		// the photo gets the whole stage.
		await expect(panel(page)).toHaveCount(0);
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = panel(page);
		await expect(drawer).toBeVisible();

		// The four sections in their pinned order: description, people,
		// information (the location is the info's last row, not a
		// section of its own), under the filename header.
		const headings = await drawer.getByRole('heading').allTextContents();
		expect(headings).toEqual([LAND_NAME, 'Description', 'People', 'Information']);

		// The description is ONE line that becomes an input on click;
		// Enter saves it, Escape reverts, an unchanged value writes
		// nothing.
		await expect(drawer.locator('.info__description')).toHaveText('No description yet.');
		await drawer.locator('.info__description').click();
		const editor = drawer.locator('input.info__editor');
		await expect(editor).toBeFocused();
		await editor.fill('Original caption.');
		await page.keyboard.press('Escape');
		await expect(drawer.locator('.info__description')).toHaveText('No description yet.');
		await drawer.locator('.info__description').click();
		await editor.fill('Sunset over the lake.');
		await page.keyboard.press('Enter');
		await expect(page.locator('.toasts')).toContainText('Description saved.');
		await expect(drawer.locator('.info__description')).toHaveText('Sunset over the lake.');
		// The line is a click-edit, never a standing textarea.
		await expect(drawer.locator('textarea')).toHaveCount(0);

		// The info rows: the size line, the tag row, and no tag yet.
		await expect(drawer.locator('.info__facts')).toContainText('MB');
		await expect(drawer.locator('.info__tags-empty')).toHaveText('No tags yet.');

		// The add-tag dialog: a new tag lands on the chip row without a
		// reload, and the chip's own × takes it away again.
		await drawer.getByRole('button', { name: 'Add tag', exact: true }).click();
		const tagDialog = page.getByRole('dialog', { name: 'Add tag', exact: true });
		await expect(tagDialog).toBeVisible();
		await tagDialog.getByRole('radio', { name: 'New tag', exact: true }).check();
		await tagDialog.getByLabel('Tag name').fill('Beach');
		await tagDialog.getByRole('button', { name: 'Create and add', exact: true }).click();
		await expect(tagDialog).toContainText('Added 1, skipped 0.');
		await tagDialog.getByRole('button', { name: 'Done', exact: true }).click();
		const chip = drawer.locator('.info__tag-chip').first();
		await expect(chip).toContainText('Beach');
		await expect(chip.locator('a')).toHaveAttribute('href', /\/search\?tagIds=/);
		await chip.getByRole('button', { name: 'Remove tag Beach', exact: true }).click();
		await expect(page.locator('.toasts')).toContainText('Tag removed.');
		await expect(drawer.locator('.info__tags-empty')).toHaveText('No tags yet.');

		// The path row is the administrator's, collapsed until asked,
		// and its copy arm writes the path to the clipboard.
		const pathRow = drawer.getByTestId('viewer-path');
		await expect(pathRow).toBeVisible();
		await expect(pathRow).toContainText('Path');
		await expect(pathRow.getByRole('button', { name: 'Copy path', exact: true })).toHaveCount(0);
		await pathRow.getByRole('button', { name: 'Path', exact: true }).click();
		await expect(pathRow).toContainText('originals/');
		await pathRow.getByRole('button', { name: 'Copy path', exact: true }).click();
		await expect(page.locator('.toasts')).toContainText('Path copied.');
		expect(await page.evaluate(() => navigator.clipboard.readText())).toContain('originals/');

		// No GPS and no manual pin: the location row never renders.
		await expect(drawer.getByTestId('viewer-location')).toHaveCount(0);
		// The album row is the road back from a photo to its albums, and
		// it opens in a NEW TAB so the open photo survives the trip (the
		// instance's sync card is every asset's one album).
		const album = drawer.locator('.info__album');
		await expect(album).toHaveCount(1);
		await expect(album).toHaveText('My photos');
		await expect(album).toHaveAttribute('target', '_blank');
		await expect(album).toHaveAttribute('rel', 'noopener');

		// The open preference is remembered across a reload (the same
		// tab session): the panel is still open, the description too.
		await page.reload();
		await expect(panel(page)).toBeVisible();
		await expect(panel(page).locator('.info__description')).toHaveText('Sunset over the lake.');
	});

	test('scene 4: the date row shows its zone and its dialog writes one value', async ({ page }) => {
		await page.goto(`/a/${landId}`);
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = panel(page);

		// The pencil opens the dialog seeded from the stored truth: the
		// wall clock in the asset's zone, that zone selected.
		await drawer.getByRole('button', { name: 'Edit date', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Edit date', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText('The zone decides where the photo sits on the timeline');
		await dialog.locator('#viewer-edit-date-value').fill('2026-08-12T18:42');
		await dialog.locator('#viewer-edit-date-zone').selectOption('+08:00');
		await dialog.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(page.locator('.toasts')).toContainText('Date updated.');

		// The row now reads the edited wall clock WITH its zone: the
		// offset is the display zone, not decoration.
		await expect(drawer.locator('.info__facts')).toContainText('UTC+8');
		await expect(drawer.locator('.info__facts')).toContainText('6:42');
	});

	test('scene 5: a drawn box becomes a face, its chip joins the People section', async ({ page }) => {
		await page.goto(`/a/${portId}`);
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = panel(page);
		await expect(drawer.locator('.info__faces-empty')).toHaveText('No people recognized in this photo.');

		// The add-person icon arms the drag layer INSIDE the media box;
		// the arrow keys belong to the drag while it is armed.
		await drawer.getByRole('button', { name: 'Add person', exact: true }).click();
		const layer = page.locator('[data-face-draw]');
		await expect(layer).toBeVisible();
		await expect(layer).toContainText('Draw a box around the face');

		const box = await mediaBox(page);
		const originX = box.x + box.width * 0.35;
		const originY = box.y + box.height * 0.3;
		await page.mouse.move(originX, originY);
		await page.mouse.down();
		await page.mouse.move(originX + 90, originY + 60, { steps: 6 });
		await expect(page.locator('[data-face-rect]')).toBeVisible();
		await page.mouse.up();

		// The assign flow opens over the new face (its next question is
		// who it is); closing it leaves the face on the photo.
		const assign = page.getByRole('dialog', { name: 'Assign person', exact: true });
		await expect(assign).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(assign).toHaveCount(0);

		// The chip joins the People section and the box exists on the
		// photo; hovering the chip lights that box (the two ends of one
		// state) with its name plate.
		const chip = drawer.locator('.info__face').first();
		await expect(chip).toContainText('Not grouped yet');
		await expect(page.locator('[data-face-box]')).toHaveCount(1);
		await expect(page.locator('[data-face-box]').first()).toHaveCSS('opacity', '0');
		await chip.hover();
		await expect(page.locator('[data-face-box]').first()).toHaveCSS('opacity', '1');
		// The box names its face: an unlabeled face reads as unnamed.
		await expect(page.locator('[data-face-label]').first()).toContainText('Unnamed');

		// THE REVERSE DIRECTION (the acceptance walk's finding): the
		// face's own region on the photo is the other arm. The pointer
		// leaves the chip, the ~220ms grace lets the highlight go, and
		// then hovering the REGION over the face reveals its box and
		// lights its chip -- the two ends are one state, whichever one
		// the pointer finds first.
		await page.mouse.move(4, 4);
		await expect(page.locator('[data-face-box]').first()).toHaveCSS('opacity', '0');
		const faceBox = page.locator('[data-face-box]').first();
		const faceRect = await faceBox.boundingBox();
		if (faceRect === null) {
			throw new Error('the face box never landed');
		}
		await page.mouse.move(faceRect.x + faceRect.width / 2, faceRect.y + faceRect.height / 2);
		await expect(faceBox).toHaveCSS('opacity', '1');
		await expect(chip).toHaveClass(/border-accent/);
		await expect(page.locator('[data-face-label]').first()).toContainText('Unnamed');
	});

	test('scene 6: the panel is one switch and the stage re-fits when it closes', async ({ page }) => {
		await page.goto(`/a/${landId}`);
		const toggle = page.getByRole('button', { name: 'Photo information' });
		await expect(toggle).toHaveAttribute('aria-pressed', 'false');

		// The bar's button opens it; the media box narrows by the
		// panel's fixed 360px (the stage re-fits through the flow).
		await toggle.click();
		await expect(toggle).toHaveAttribute('aria-pressed', 'true');
		await expect(panel(page)).toBeVisible();
		const withPanel = (await mediaBox(page)).width;

		// The panel's own close mark is the same path: pressed state
		// off, the stage widens back.
		await panel(page).getByRole('button', { name: 'Close', exact: true }).click();
		await expect(toggle).toHaveAttribute('aria-pressed', 'false');
		await expect(panel(page)).toHaveCount(0);
		await expect.poll(async () => (await mediaBox(page)).width).toBeGreaterThan(withPanel + 200);

		// The i key is the third door, and it also closes.
		await page.keyboard.press('i');
		await expect(panel(page)).toBeVisible();
		await expect(toggle).toHaveAttribute('aria-pressed', 'true');
		await page.keyboard.press('i');
		await expect(panel(page)).toHaveCount(0);
		// The preference is a stored value, not a per-visit guess.
		expect(await page.evaluate(() => localStorage.getItem('lumemor:viewer:info'))).toBe('false');
	});

	test('scene 7: the location row states what is missing, then draws and edits the pin', async ({ page }) => {
		// No tile source is configured on this fresh instance: the
		// coordinates still stand and the map says which half is gone
		// (the honest degradation -- never a blank square).
		await page.goto(`/a/${gpsId}`);
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = panel(page);
		const row = drawer.getByTestId('viewer-location');
		await expect(row).toBeVisible();
		await expect(row).toContainText(/\d+\.\d{4}, \d+\.\d{4}/);
		await expect(row.locator('[data-map-state="unconfigured"]')).toBeVisible();

		// The instance's own tile source (stubbed here) draws the same
		// box in the same place: the mini map is display-only.
		await stubTiles(page);
		const saved = await page.request.put('/api/v1/admin/settings/tiles', { data: { tilejsonUrl: TILE_SOURCE } });
		expect(saved.status()).toBe(200);
		// The reload also proves the open preference: the panel comes
		// back open by itself, the same instance's tiles now behind the
		// mini map.
		await page.reload();
		await expect(panel(page)).toBeVisible();
		const live = panel(page).getByTestId('viewer-location');
		await expect(live.locator('.maplibregl-canvas')).toBeVisible();
		await expect(live.locator('.map__asset-pin')).toBeVisible();
		await expect(live.locator('[data-map-state]')).toHaveCount(0);

		// The pencil opens the edit dialog: the search box is the
		// keyboard path, and picking a place pins its coordinates.
		await live.getByRole('button', { name: 'Edit location', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Edit location', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText('Drag to move the map');
		await dialog.getByRole('searchbox').fill('Beij');
		const option = dialog.getByRole('button', { name: /Beijing/ });
		await expect(option).toBeVisible();
		await option.click();
		await expect(dialog.getByTestId('viewer-location-coords-line')).toContainText(/\d+\.\d{4}, \d+\.\d{4}/);

		// A click on the map drops the pin somewhere else: the
		// coordinates line follows it.
		const before = await dialog.getByTestId('viewer-location-coords-line').textContent();
		const canvas = dialog.locator('.maplibregl-canvas');
		const canvasBox = await canvas.boundingBox();
		if (canvasBox === null) {
			throw new Error('the picker canvas never landed');
		}
		// The click is dispatched at the canvas's own coordinates: the
		// library reads the pointer's viewport position off the event,
		// and this is the same click the pointer would send.
		await canvas.dispatchEvent('click', {
			clientX: canvasBox.x + 60,
			clientY: canvasBox.y + 60,
			bubbles: true
		});
		await expect(dialog.getByTestId('viewer-location-coords-line')).not.toHaveText(before ?? '');

		// The save writes the pair and the row converges on it.
		await dialog.getByRole('button', { name: 'Use this location', exact: true }).click();
		await expect(page.locator('.toasts')).toContainText('Location updated.');
		await expect(panel(page).getByTestId('viewer-location')).toContainText(/\d+\.\d{4}, \d+\.\d{4}/);
	});

	test('scene 8: the floor protection -- a drawer below 640, the sidebar above it', async ({ page }) => {
		// The one deviation from the reference model: a 360px sidebar
		// against a 360px viewport would zero the stage, so below 640
		// the panel becomes a bottom drawer over a scrim and the photo
		// keeps the width.
		await page.setViewportSize({ width: 520, height: 800 });
		await page.goto(`/a/${landId}`);
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = panel(page);
		await expect(drawer).toBeVisible();
		await expect(drawer).toHaveCSS('position', 'fixed');
		const drawerBox = await drawer.boundingBox();
		if (drawerBox === null) {
			throw new Error('the drawer never landed');
		}
		expect(drawerBox.width).toBeGreaterThan(500);
		// The drawer leaves the top of the photo visible.
		expect(drawerBox.y).toBeGreaterThan(80);
		// The scrim behind it closes on a click (the same path the
		// panel's close mark drives).
		await page.locator('.info-scrim').click({ position: { x: 20, y: 20 } });
		await expect(panel(page)).toHaveCount(0);
		// The minimum width holds: no horizontal overflow at 360.
		await page.setViewportSize({ width: 360, height: 780 });
		await page.getByRole('button', { name: 'Photo information' }).click();
		await expect(panel(page)).toBeVisible();
		expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1)).toBe(true);
		await expect(page.locator('.info-scrim')).toBeVisible();

		// Above 640 the same element is the static 360px sidebar: no
		// scrim, the photo beside it.
		await page.setViewportSize({ width: 900, height: 800 });
		await expect(panel(page)).toBeVisible();
		await expect(page.locator('.info-scrim')).toBeHidden();
		await expect(panel(page)).toHaveCSS('position', 'static');
		const sidebar = await panel(page).boundingBox();
		if (sidebar === null) {
			throw new Error('the sidebar never landed');
		}
		expect(Math.round(sidebar.width)).toBe(360);
		// The docked panel sits against the viewport's right edge (a
		// scrollbar's width is the only tolerated gap).
		expect(Math.abs(sidebar.x + sidebar.width - 900)).toBeLessThanOrEqual(16);
	});
});
