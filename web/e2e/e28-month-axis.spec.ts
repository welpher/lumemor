// The month axis's story of the web app, walked end to end against
// the real compose stack: the axis is ONE scrollbar control on
// every month-grouped wall (the timeline, the album detail, the
// place detail, and the map panel's compact pickup). The press
// jumps where you point (on the down, not the release), the hold
// drags the wall along with the solid bar standing down, the
// readout follows the pointer and rides the bar while the wall
// scrolls alone, the keyboard is one tab stop walking the months
// with the readout as its focus indicator, and the wall's tail
// lets the LAST month land on the reading line. The dots, the
// press, the readout and the accessible name all read ONE
// mapping (the wall's own geometry), so pressing a dot's height
// lands exactly that month.
//
// THE SCENES (serial, one stack lifetime):
//
//  1. The administrator signs in through the form.
//  2. The fixtures upload through the asset API: twelve photos
//     across five months, two of them carrying the Beijing GPS
//     EXIF (June and May -- the place domain's own two-month wall).
//  3. The timeline wall's control: the rest shape, the hover
//     readout, the press jump at the screen edge, the drag with
//     the bar's stand-down, the dot-press mapping, the keyboard
//     walk and the reading line, the scroll readout's ride and
//     fade, the reduced-motion walk and the touch case.
//  4. The sync album's detail wall: the same control over a
//     sticky-bar page (the pinned offset shifts the reading line
//     for dots, press and landing alike).
//  5. The place detail's wall: the same control on the place's
//     own page.
//  6. The map panel's compact pickup: the SAME control in its
//     24px slot -- the press moves the panel's own root.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

const ADMIN_EMAIL = 'admin@e28-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

const repoRoot = fileURLToPath(new URL('../..', import.meta.url));
const PHOTO_A_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/portrait-orientation6.jpg');
const PHOTO_B_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/golden-large.jpg');
// The GPS fixture: its EXIF coordinates resolve to Beijing, so the
// places domain gains its one city from the two suffixed copies.
const PHOTO_GPS_SOURCE = path.join(repoRoot, 'server/internal/jobs/testdata/gps-beijing.jpg');

const TILE_HOST = 'https://tiles.e28-e2e.invalid';
const TILE_SOURCE = `${TILE_HOST}/tilejson.json`;

const tileFixture = await readFile(path.join(specDir, 'fixtures/tile-256.png'));

const adminStatePath = `${specDir}/.auth/e28-admin-state.json`;

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap: the setup API creates the
	// administrator on the fresh instance.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'E28 E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

/** One API upload: the device timestamp IS the taken-at (the
 * testdata photos carry no EXIF date), so the wall folds by it.
 * Every upload carries a distinct trailing byte block (distinct
 * checksums) -- the ingest's dedup would answer the repeated
 * fixtures with the first row instead of a new one. */
async function uploadAsset(
	page: Page,
	assetId: string,
	fileName: string,
	takenAt: string,
	source: string | Buffer,
	seed: number
): Promise<void> {
	const base = typeof source === 'string' ? await readFile(source) : source;
	const buffer = Buffer.concat([base, Buffer.alloc(256, seed)]);
	const upload = await page.request.post('/api/v1/assets', {
		multipart: {
			deviceAssetId: assetId,
			fileName,
			deviceCreatedAt: takenAt,
			deviceTimeZone: 'UTC',
			photo: { name: fileName, mimeType: 'image/jpeg', buffer }
		}
	});
	expect(upload.status()).toBe(201);
}

/** The GPS fixture with its EXIF capture date rewritten in place
 * (a length-stable ASCII string): the fixture's own EXIF date is
 * the taken-at, so a copy in another month gives the place domain a
 * wall that spans two months -- the span the axis's tail and press
 * need to be walked at all. */
async function gpsPhotoInMonth(month: string): Promise<Buffer> {
	const buffer = Buffer.from(await readFile(PHOTO_GPS_SOURCE));
	const date = Buffer.from('2026:06:20 10:15:30');
	const at = buffer.indexOf(date);
	expect(at).toBeGreaterThan(0);
	Buffer.from(`2026:${month}:20 10:15:30`).copy(buffer, at);
	return buffer;
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

	test('scene 2: the fixtures land and the timeline folds into five months', async ({ page }) => {
		// Five months, newest first on the wall; every upload carries
		// its own seed byte block (distinct checksums). The two GPS
		// photos resolve one Beijing place from their shared EXIF;
		// the second one's EXIF month is May, so the place wall spans
		// two months.
		const months = [
			'2026-09-15T12:00:00Z',
			'2026-08-15T12:00:00Z',
			'2026-07-15T12:00:00Z',
			'2026-06-15T12:00:00Z',
			'2026-05-15T12:00:00Z'
		];
		for (const [index, takenAt] of months.entries()) {
			await uploadAsset(
				page,
				`e28-e2e-a${index + 1}`,
				`IMG_E28_${index + 1}.jpg`,
				takenAt,
				index % 2 === 0 ? PHOTO_A_SOURCE : PHOTO_B_SOURCE,
				index + 1
			);
			await uploadAsset(
				page,
				`e28-e2e-b${index + 1}`,
				`IMG_E28_B${index + 1}.jpg`,
				takenAt,
				index % 2 === 0 ? PHOTO_B_SOURCE : PHOTO_A_SOURCE,
				11 + index
			);
		}
		await uploadAsset(page, 'e28-e2e-gps-1', 'IMG_E28_GPS1.jpg', '2026-06-20T12:00:00Z', PHOTO_GPS_SOURCE, 21);
		await uploadAsset(
			page,
			'e28-e2e-gps-2',
			'IMG_E28_GPS2.jpg',
			'2026-05-20T12:00:00Z',
			await gpsPhotoInMonth('05'),
			22
		);

		// The timeline read carries all twelve once the pipeline's
		// listing catches up (the wall's geometry needs the sections).
		await expect
			.poll(
				async () => {
					const response = await page.request.get('/api/v1/assets/timeline?limit=100');
					expect(response.status()).toBe(200);
					const body = (await response.json()) as { items: unknown[] };
					return body.items.length;
				},
				{ timeout: 120_000, intervals: [1000, 2000, 5000] }
			)
			.toBe(12);

		// The place domain resolves from the GPS pair (its wall reads
		// the two oldest months).
		await expect
			.poll(
				async () => {
					const response = await page.request.get('/api/v1/places');
					expect(response.status()).toBe(200);
					const body = (await response.json()) as Array<{ assetCount: number }>;
					return body[0]?.assetCount ?? 0;
				},
				{ timeout: 120_000, intervals: [1000, 2000, 5000] }
			)
			.toBe(2);
	});

	test('scene 3: the timeline wall carries the one control end to end', async ({ page }) => {
		await page.goto('/timeline');
		await expect(page).toHaveTitle('Timeline · lumemor');

		const control = page.locator('.scrubber__rail');
		await expect(control).toBeVisible();

		// THE REST SHAPE: a scrollbar with one tab stop, the resize
		// cursor over the whole rail, the controlled scroller named,
		// and no month button anywhere (the dots are decoration).
		await expect(control).toHaveAttribute('role', 'scrollbar');
		await expect(control).toHaveAttribute('aria-orientation', 'vertical');
		await expect(control).toHaveAttribute('aria-controls', 'shell-main');
		await expect(control).toHaveAttribute('aria-valuemin', '0');
		await expect(control).toHaveAttribute('aria-valuemax', '100');
		expect(await control.getAttribute('aria-valuenow')).toBe('0');
		const valuetext = await control.getAttribute('aria-valuetext');
		expect(valuetext).toMatch(/^(September|August|July|June|May) 2026$/);
		expect(await page.locator('.scrubber__rail button').count()).toBe(0);
		// The marks: five months, five dots at strictly increasing
		// heights, one faint year.
		expect(await page.locator('.scrubber__stop').count()).toBe(5);
		expect(await page.locator('.scrubber__year').count()).toBe(1);
		const markTops = await page
			.locator('.scrubber__stop')
			.evaluateAll((nodes) => nodes.map((node) => Number.parseFloat((node as HTMLElement).style.top)));
		expect(markTops).toHaveLength(5);
		for (let i = 1; i < markTops.length; i += 1) {
			expect(markTops[i]!).toBeGreaterThan(markTops[i - 1]!);
		}
		// THE TAIL: the runway that lets the last month reach the
		// reading line (the wall no longer ends flush).
		const tail = page.locator('.wall__tail');
		await expect(tail).toBeAttached();
		expect(await tail.evaluate((node) => (node as HTMLElement).getBoundingClientRect().height)).toBeGreaterThan(0);
		// The touch case: the rail keeps the gesture for itself (the
		// page never scrolls under a held drag).
		const touch = await control.evaluate((node) => getComputedStyle(node).touchAction);
		expect(touch).toBe('none');
		// The readout is mounted but transparent at rest (the fades are
		// transitions, not pops).
		await expect(page.locator('.scrubber__readout')).toHaveAttribute('aria-hidden', 'true');
		expect(await page.locator('.scrubber__readout').evaluate((node) => (node as HTMLElement).style.opacity)).toBe(
			'0'
		);

		const box = (await control.boundingBox())!;
		expect(box).toBeTruthy();

		// THE HOVER: the readout follows the pointer, names the month a
		// press there would land, and the pointer's tick shows beside
		// it -- the wall itself does not move.
		await page.mouse.move(box.x + box.width / 2, box.y + box.height * 0.6);
		const hoverReadout = page.locator('.scrubber__readout');
		await expect
			.poll(() => hoverReadout.evaluate((node) => (node as HTMLElement).style.opacity), { timeout: 5_000 })
			.toBe('1');
		expect(await hoverReadout.textContent()).toMatch(/^(September|August|July|June|May) 2026$/);
		expect(await page.locator('.scrubber__tick').evaluate((node) => (node as HTMLElement).style.opacity)).toBe('1');
		const mainScrollAtHover = await page.evaluate(() => document.querySelector('main')?.scrollTop ?? -1);
		expect(mainScrollAtHover).toBe(0);
		// Leaving the rail fades both marks.
		await page.mouse.move(box.x - 120, box.y + box.height / 2);
		await expect
			.poll(() => hoverReadout.evaluate((node) => (node as HTMLElement).style.opacity), { timeout: 5_000 })
			.toBe('0');

		// THE PRESS IS THE JUMP: a press at 60% of the rail -- at the
		// screen's very edge (the rail is flush to it) -- takes the wall
		// there before the release.
		await page.mouse.move(box.x + box.width - 1, box.y + box.height * 0.6);
		await page.mouse.down();
		const pressScroll = await page.evaluate(() => document.querySelector('main')?.scrollTop ?? -1);
		expect(pressScroll).toBeGreaterThan(0);
		await expect
			.poll(async () => Number(await control.getAttribute('aria-valuenow')), { timeout: 5_000 })
			.toBeGreaterThanOrEqual(55);
		await page.mouse.up();

		// THE DRAG: a held press drags the wall continuously; the solid
		// bar stands down mid-drag (the tick is the position marker) and
		// comes back on the release.
		await page.mouse.move(box.x + box.width / 2, box.y + box.height * 0.6);
		await page.mouse.down();
		const dragFrom = await page.evaluate(() => document.querySelector('main')?.scrollTop ?? -1);
		await page.mouse.move(box.x + box.width / 2, box.y + box.height * 0.15, { steps: 8 });
		const dragTo = await page.evaluate(() => document.querySelector('main')?.scrollTop ?? -1);
		expect(dragTo).toBeLessThan(dragFrom);
		// The bar stands down by a fade: poll while the press still
		// holds, so the transition's last frame is not the assertion.
		await expect
			.poll(() => page.locator('.scrubber__indicator').evaluate((node) => (node as HTMLElement).style.opacity), {
				timeout: 5_000
			})
			.toBe('0');
		await page.mouse.up();
		await expect
			.poll(() => page.locator('.scrubber__indicator').evaluate((node) => (node as HTMLElement).style.opacity), {
				timeout: 5_000
			})
			.toBe('1');

		// THE DOT-PRESS MAPPING: a press at a dot's height lands exactly
		// that month's reading position -- the dot you see IS the place
		// the press takes you, and the accessible month name follows.
		const secondDot = page.locator('.scrubber__stop').nth(1);
		const dotBox = (await secondDot.boundingBox())!;
		await page.mouse.move(dotBox.x + dotBox.width / 2, dotBox.y + dotBox.height / 2);
		await page.mouse.down();
		await page.mouse.up();
		await expect
			.poll(async () => await control.getAttribute('aria-valuetext'), { timeout: 5_000 })
			.toBe('August 2026');

		// THE KEYBOARD: one tab stop; the readout is the focus
		// indicator (no box ring), the arrows walk on from the month the
		// wall reads. The pointer's own press left the rail focused
		// without the readout (a click never parks a bubble), so step
		// out and back in -- the Tab arrival.
		await control.evaluate((node) => (node as HTMLElement).blur());
		await control.focus();
		await expect
			.poll(() => hoverReadout.evaluate((node) => (node as HTMLElement).style.opacity), { timeout: 5_000 })
			.toBe('1');
		await page.keyboard.press('ArrowDown');
		await expect
			.poll(async () => await control.getAttribute('aria-valuetext'), { timeout: 10_000 })
			.toBe('July 2026');

		// End takes the last month, whose landing the tail makes
		// exactly the reading line: 92px under the scrollport top, with
		// the accessible name and the progress at the far end.
		await page.keyboard.press('End');
		await expect
			.poll(async () => await control.getAttribute('aria-valuetext'), { timeout: 10_000 })
			.toBe('May 2026');
		await expect(control).toHaveAttribute('aria-valuenow', '100');
		// The smooth landing is settled by the poll: the accessible
		// name turns at the anchor crossing, the geometry a frame or
		// two later.
		await expect
			.poll(
				async () =>
					await page.evaluate(() => {
						const main = document.querySelector('main');
						const last = Array.from(document.querySelectorAll('.wall__month')).at(-1);
						if (main === null || last === undefined) {
							return Number.NaN;
						}
						return Math.abs(last.getBoundingClientRect().top - main.getBoundingClientRect().top - 92);
					}),
				{ timeout: 5_000 }
			)
			.toBeLessThanOrEqual(3);

		// THE SCROLL READOUT: the wall alone scrolling wakes the readout
		// riding the bar (never the pointer's tick), and nine hundred
		// quiet milliseconds fade it again. The keyboard's focus owns
		// the readout, so the focus steps off first -- otherwise the
		// fade is testing the focus, not the scroll.
		await control.evaluate((node) => (node as HTMLElement).blur());
		await page.mouse.move(box.x - 200, box.y + box.height / 2);
		await page.mouse.wheel(0, -400);
		await expect
			.poll(() => hoverReadout.evaluate((node) => (node as HTMLElement).style.opacity), { timeout: 5_000 })
			.toBe('1');
		expect(await page.locator('.scrubber__tick').evaluate((node) => (node as HTMLElement).style.opacity)).toBe('0');
		await expect
			.poll(() => hoverReadout.evaluate((node) => (node as HTMLElement).style.opacity), { timeout: 6_000 })
			.toBe('0');

		// THE REDUCED-MOTION WALK: the keyboard jump lands instantly
		// (the smooth courtesy is the pointer path's).
		await page.emulateMedia({ reducedMotion: 'reduce' });
		await control.focus();
		await page.keyboard.press('Home');
		await expect
			.poll(async () => await control.getAttribute('aria-valuetext'), { timeout: 5_000 })
			.toBe('September 2026');
		await page.emulateMedia({ reducedMotion: null });
	});

	test('scene 4: the sync album detail wears the same control under its pinned bar', async ({ page }) => {
		// The caller's own sync album is the default wall card (a fresh
		// account's albums wall is never empty).
		const albums = await page.request.get('/api/v1/albums');
		expect(albums.status()).toBe(200);
		const body = (await albums.json()) as { albums?: Array<{ id?: string; type?: string }> };
		const sync = (body.albums ?? []).find((album) => album.type === 'sync');
		expect(sync?.id).toBeTruthy();

		await page.goto(`/albums/${sync?.id}`);
		await expect(page).toHaveTitle(/· lumemor$/);

		const control = page.locator('.scrubber__rail');
		await expect(control).toBeVisible();
		await expect(control).toHaveAttribute('role', 'scrollbar');
		// The album's pinned bar shifts the reading line for the dots,
		// the press and the landing alike -- the control still answers.
		const box = (await control.boundingBox())!;
		await page.mouse.move(box.x + box.width - 1, box.y + box.height * 0.7);
		await page.mouse.down();
		const pressScroll = await page.evaluate(() => document.querySelector('main')?.scrollTop ?? -1);
		expect(pressScroll).toBeGreaterThan(0);
		await page.mouse.up();
		await expect
			.poll(async () => await control.getAttribute('aria-valuetext'), { timeout: 5_000 })
			.toMatch(/^(September|August|July|June|May) 2026$/);
		await expect(page.locator('.wall__tail')).toBeAttached();
	});

	test('scene 5: the place detail wears the same control on its own wall', async ({ page }) => {
		const places = await page.request.get('/api/v1/places');
		expect(places.status()).toBe(200);
		const body = (await places.json()) as Array<{ placeId: number }>;
		expect(body.length).toBe(1);

		await page.goto(`/places/${body[0]!.placeId}`);
		const control = page.locator('.scrubber__rail');
		await expect(control).toBeVisible();
		// The place wall reads two months (the GPS pair); a press at
		// the rail's bottom takes it to the last month's reading line.
		const box = (await control.boundingBox())!;
		await page.mouse.move(box.x + box.width - 1, box.y + box.height - 2);
		await page.mouse.down();
		const pressScroll = await page.evaluate(() => document.querySelector('main')?.scrollTop ?? -1);
		expect(pressScroll).toBeGreaterThan(0);
		await page.mouse.up();
		await expect
			.poll(async () => await control.getAttribute('aria-valuetext'), { timeout: 5_000 })
			.toBe('May 2026');
	});

	test('scene 6: the map panel picks up the same control in its compact slot', async ({ page }) => {
		await page.route(`${TILE_HOST}/**`, async (route) => {
			if (route.request().url().includes('/tilejson.json')) {
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
		await page.request.put('/api/v1/admin/settings/tiles', { data: { tilejsonUrl: TILE_SOURCE } });
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto('/places?mode=map');

		const pin = page.locator('.map__pin').first();
		await expect(pin).toBeVisible();
		await pin.click();
		const panel = page.getByTestId('places-panel');
		await expect(panel).toBeVisible();

		// The compact axis is the SAME control: a scrollbar in the
		// panel's own slot, driving the panel's own root.
		const control = panel.locator('.scrubber__rail');
		await expect(control).toHaveAttribute('role', 'scrollbar');
		await expect(control).toHaveAttribute('aria-controls', 'wall-panel-scroll');
		await expect(control).toHaveAttribute('aria-orientation', 'vertical');
		expect(await control.getAttribute('aria-valuetext')).toMatch(/^(June|May) 2026$/);
		// The compact slot's own tail (two months in a tall panel).
		await expect(panel.locator('.wall__tail')).toBeAttached();

		// A press at the compact rail's bottom takes the panel's wall to
		// its last month's reading position.
		const box = (await control.boundingBox())!;
		await page.mouse.move(box.x + box.width / 2, box.y + box.height - 2);
		await page.mouse.down();
		const pressScroll = await page.evaluate(() => document.querySelector('#wall-panel-scroll')?.scrollTop ?? -1);
		expect(pressScroll).toBeGreaterThan(0);
		await page.mouse.up();
		await expect
			.poll(async () => await control.getAttribute('aria-valuetext'), { timeout: 5_000 })
			.toBe('May 2026');
	});
});
