// The admin surfaces' state family, walked end to end against the
// real compose stack: the release check's three faces, a failing job
// on the task panel, and a failed save that stays under its field.
//
// THE STACK ARRANGEMENT: one fresh lifetime owns this spec (the
// first-run bootstrap below creates the administrator, so a second
// spec on the same lifetime would answer 409). The release check is
// SEALED at the deployment level (the harness sets the check off):
// nothing here ever reaches a release feed. The three faces are the
// INJECTED state -- the check's own endpoint is answered by the
// page-level route, which is exactly the shape the card renders --
// and the manual check's busy state rides the same stub.
//
// THE SCENES (in order):
//
//  1. The administrator signs in through the form and lands on
//     /admin/settings; the sealed deployment answers the check
//     switched off, which is its own quiet face.
//  2. The three faces, one at a time: up to date with its last-checked
//     line, a newer release with the pair and the "this changes
//     nothing for you" sentence, and the failed face -- quiet, no
//     danger, no toast. The manual check shows its busy label and
//     adopts the fresh verdict.
//  3. A failed save stays under its field: the taken address (409)
//     and a wrong current password (400 naming the field) each land
//     under the input they belong to, with the typed text intact and
//     no toast. Both refusals come from the real server.
//  4. A failing job on the real stack: a directory album whose source
//     is removed fails its rescan; the task panel renders the failed
//     row with a reason and no retry (the rescan's run row carries no
//     album id, so the trigger is not this panel's), while a stubbed
//     listing shows all four states' shapes side by side.

import { copyFile, mkdir, rm } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's administrator: created through the first-run setup API
// in the suite's bootstrap (the acceptance scripts' pattern), then
// used by the login scene. The stack is always fresh, so the fixed
// identity is deterministic.
const ADMIN_EMAIL = 'admin@e20-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The second account whose address the profile save collides with.
const OTHER_EMAIL = 'grace@e20-e2e.invalid';
const OTHER_PASSWORD = 'correct-horse-battery';

// The vanishing source: a directory under the writable bind the
// container sees, holding one photo at album creation and nothing at
// rescan time (the host removes it between the two calls). The
// writable fixture is the only bind the container may write, so the
// source lives inside it; the seed photo is a genuinely distinct
// portrait the other stories never link.
const RW_MOUNT_ROOT = '/mnt/e2e-source-rw';
const RW_HOST_DIR = path.join(specDir, 'fixtures/m3-source-rw');
const VANISH_DIR_NAME = 'e20-vanish';
const VANISH_HOST_DIR = path.join(RW_HOST_DIR, VANISH_DIR_NAME);
const VANISH_MOUNT_DIR = path.join(RW_MOUNT_ROOT, VANISH_DIR_NAME);
const VANISH_SEED = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other.jpg');

// The storage state scene 1 writes and the later scenes replay
// (git-ignored, recreated per run).
const authStatePath = `${specDir}/.auth/e20-state.json`;

/** One update-check state body (the contract's UpdateCheckResponse). */
function checkState(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		enabled: true,
		status: 'up_to_date',
		currentVersion: '0.4.1',
		latestVersion: 'v0.4.1',
		checkedAt: new Date().toISOString(),
		...overrides
	};
}

/**
 * The settings cards are a collapsed index (E33): the scenes that
 * read a card's content open it through its summary first.
 */
async function openCard(page: import('@playwright/test').Page, testId: string): Promise<void> {
	const card = page.getByTestId(testId);
	// Ensure-open, not a blind toggle: the remembered expanded set
	// may already have opened this card, and a second click closes it.
	if ((await card.getAttribute('open')) === null) {
		await card.locator('summary').click();
	}
}

/** The faces present on the page, by their marker. */
async function facesOn(page: import('@playwright/test').Page): Promise<string[]> {
	return page
		.locator('[data-update-face]')
		.evaluateAll((nodes) => nodes.map((node) => node.getAttribute('data-update-face') ?? ''));
}

test.beforeAll(async ({ playwright }) => {
	// The vanishing source's seed: the directory is created, filled
	// with one photo, and removed by scene 4 -- the rescan then walks
	// a root that no longer exists. The sweep first keeps a rerun
	// deterministic on a host that kept the previous run's leftovers.
	await rm(VANISH_HOST_DIR, { recursive: true, force: true });
	await mkdir(VANISH_HOST_DIR, { recursive: true });
	await copyFile(VANISH_SEED, path.join(VANISH_HOST_DIR, 'IMG_E20_0001.jpg'));

	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance, and
	// the admin API provisions the second account the profile save
	// collides with.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'E20 E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	const login = await api.post('/api/v1/auth/login', {
		data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(login.status()).toBe(200);
	const other = await api.post('/api/v1/admin/users', {
		data: { name: 'Grace Hopper', email: OTHER_EMAIL, password: OTHER_PASSWORD }
	});
	expect(other.status()).toBe(201);
	await api.dispose();
});

test.afterAll(async () => {
	// The seeded source never outlives the story: a later lifetime
	// must start from its own bytes.
	await rm(VANISH_HOST_DIR, { recursive: true, force: true });
});

test('scene 1: the administrator signs in and the sealed check shows its off face', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Fadmin%2Fsettings');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/admin\/settings$/);
	await expect(page).toHaveTitle('Settings · lumemor');

	// The settings page is the collapsed index (E33): the release
	// card's facts live inside its fold.
	await openCard(page, 'admin-settings-update');

	// The release card renders its own row of facts: the running
	// version, the self-hosted badge, and the quiet face the sealed
	// deployment answers (the check is switched off -- not a verdict).
	await expect(page.getByTestId('update-check-version')).toContainText('Current version');
	await expect(page.getByTestId('update-check-version')).toContainText('Self-hosted');
	await expect(page.locator('[data-update-face="off"]').first()).toBeVisible();
	await expect(page.getByTestId('update-check-now')).toBeDisabled();
	// The privacy boundary rides beside the knobs.
	await expect(page.getByTestId('admin-settings-update')).toContainText('Only the version number is compared');

	// The admin session rides to every later scene.
	await page.context().storageState({ path: authStatePath });
});

test.describe('the administrator scenes', () => {
	test.use({ storageState: authStatePath });

	test('scene 2: the three faces are exclusive, and failure stays quiet', async ({ page }) => {
		// The faces, in order: up to date, a newer release, and the
		// silent failure -- each answered by the injected state, one
		// at a time.
		await page.route('**/api/v1/server/update-check', (route) => route.fulfill({ json: checkState() }));
		await page.goto('/admin/settings');
		await openCard(page, 'admin-settings-update');
		await expect(page.locator('[data-update-face="latest"]')).toBeVisible();
		await expect(page.locator('[data-update-face="latest"]')).toContainText('Up to date');
		await expect(page.locator('[data-update-face="latest"]')).toContainText('Last checked');
		expect(await facesOn(page)).toEqual(['latest']);

		await page.unroute('**/api/v1/server/update-check');
		await page.route('**/api/v1/server/update-check', (route) =>
			route.fulfill({ json: checkState({ status: 'update_available', latestVersion: 'v0.5.0' }) })
		);
		await page.goto('/admin/settings');
		await openCard(page, 'admin-settings-update');
		await expect(page.locator('[data-update-face="available"]')).toContainText('v0.5.0 available');
		await expect(page.locator('[data-update-face="available"]')).toContainText(
			'nothing is downloaded, installed or restarted for you'
		);
		// The pair: the running version stays on the card beside the offer.
		await expect(page.getByTestId('update-check-version')).toContainText('0.4.1');
		expect(await facesOn(page)).toEqual(['available']);

		await page.unroute('**/api/v1/server/update-check');
		await page.route('**/api/v1/server/update-check', (route) =>
			route.fulfill({
				json: checkState({
					status: 'unknown',
					latestVersion: null,
					checkedAt: new Date(Date.now() - 3 * 86400 * 1000).toISOString()
				})
			})
		);
		await page.goto('/admin/settings');
		await openCard(page, 'admin-settings-update');
		const failed = page.locator('[data-update-face="failed"]');
		await expect(failed).toBeVisible();
		await expect(failed).toContainText("couldn't reach the release feed");
		await expect(failed.locator('.text-danger-text')).toHaveCount(0);
		// Silence means silence: no toast, no alert, no dialog.
		await expect(page.locator('.toast')).toHaveCount(0);
		await expect(page.locator('[role="alert"]')).toHaveCount(0);
		expect(await facesOn(page)).toEqual(['failed']);

		// The manual check: the busy label while the probe runs, then
		// the fresh verdict.
		await page.unroute('**/api/v1/server/update-check');
		await page.route('**/api/v1/server/update-check', async (route) => {
			if (route.request().method() === 'POST') {
				await new Promise((resolve) => setTimeout(resolve, 600));
				await route.fulfill({ json: checkState() });
				return;
			}
			await route.fulfill({
				json: checkState({ status: 'unknown', latestVersion: null, checkedAt: null })
			});
		});
		await page.goto('/admin/settings');
		await openCard(page, 'admin-settings-update');
		await expect(page.locator('[data-update-face="failed"]')).toContainText('Not checked yet');
		await page.getByTestId('update-check-now').click();
		await expect(page.getByTestId('update-check-now')).toContainText('Checking…');
		await expect(page.getByTestId('update-check-now')).toBeDisabled();
		await expect(page.locator('[data-update-face="latest"]')).toBeVisible();
		await expect(page.getByTestId('update-check-now')).toContainText('Check now');
		expect(await facesOn(page)).toEqual(['latest']);
	});

	test('scene 3: a failed save lands under its field, with the text kept', async ({ page }) => {
		await page.goto('/settings');
		await expect(page).toHaveTitle('Settings · lumemor');

		// The taken address (the real 409): the line sits under the
		// address input, the typed text survives, nothing toasts.
		const email = page.locator('#settings-email');
		await email.fill(OTHER_EMAIL);
		await page.locator('[data-testid="settings-profile"]').getByRole('button', { name: 'Save' }).click();
		const emailError = page.locator('#settings-email-error');
		await expect(emailError).toContainText('An account with this email address already exists.');
		await expect(emailError).toHaveAttribute('role', 'alert');
		await expect(email).toHaveValue(OTHER_EMAIL);
		await expect(page.locator('#settings-name-error')).toHaveCount(0);
		await expect(page.locator('.toast')).toHaveCount(0);

		// The wrong current password (the real 400 naming `current`):
		// the line sits under the current field, the fields keep their
		// text, and the session stays signed in.
		const current = page.locator('#settings-password-current');
		await current.fill('definitely not the password');
		await page.locator('#settings-password-new').fill('another fine passphrase');
		await page.locator('#settings-password-confirm').fill('another fine passphrase');
		await page
			.locator('[data-testid="settings-password"]')
			.getByRole('button', { name: 'Change password' })
			.click();
		await expect(page.locator('#settings-password-current-error')).toContainText(
			'Please check your entries and try again.'
		);
		await expect(current).toHaveValue('definitely not the password');
		await expect(page.locator('#settings-password-new-error')).toHaveCount(0);
		await expect(page.locator('.toast')).toHaveCount(0);
		// Still signed in: the page's own content, never the login form.
		await expect(page.locator('[data-testid="settings-profile"]')).toBeVisible();
	});

	test('scene 4: a failing job says why and offers no retry it cannot run', async ({ page }) => {
		test.setTimeout(180_000);
		// The album over the seeded source: created through the real
		// API (the browser surface's own path), then its source is
		// removed from the host -- the container's view loses the root
		// the album claims.
		const created = await page.request.post('/api/v1/albums', {
			data: { title: 'E20 vanishing source', rootPath: VANISH_MOUNT_DIR }
		});
		expect(created.status()).toBe(201);
		const album = (await created.json()) as { id?: string };
		const albumId = album.id ?? '';
		expect(albumId).not.toBe('');
		await rm(VANISH_HOST_DIR, { recursive: true, force: true });

		const rescan = await page.request.post(`/api/v1/albums/${encodeURIComponent(albumId)}/rescan`);
		expect([200, 202]).toContain(rescan.status());

		// The run closes failed: the panel renders the reason, the
		// danger bar, and NO retry (the rescan's trigger belongs to the
		// album page -- this listing carries no album id to re-post).
		await page.goto('/admin/jobs');
		const failedRow = page.locator('article.jobs__row').filter({ hasText: 'Album rescan' }).first();
		await expect(failedRow).toBeVisible({ timeout: 60_000 });
		await expect(failedRow.locator('.jobs__badge--failed')).toHaveText('Failed');
		const reason = failedRow.getByTestId('job-reason');
		await expect(reason).toContainText('The run stopped before it got through its work');
		await expect(reason).toContainText('server log');
		await expect(failedRow.locator('.jobs__progress-fill--failed')).toHaveCount(1);
		await expect(failedRow.getByTestId('job-retry')).toHaveCount(0);
	});

	test('scene 5: the four states render their own shapes, not one row with a badge', async ({ page }) => {
		// The four shapes side by side, over the injected listing and
		// models read (the real stack's own steady state cannot hold a
		// queued, a running, a succeeded and a failed run at once).
		const now = Date.now();
		await page.route('**/api/v1/admin/jobs', (route) =>
			route.fulfill({
				json: {
					jobs: [
						{
							id: '0192f8b2-6c7d-8e9f-a0b1-2c3d4e5f6070',
							kind: 'clip_embed_rerun',
							state: 'running',
							total: 9820,
							done: 4210,
							failed: 0,
							startedAt: new Date(now - 40 * 60 * 1000).toISOString(),
							finishedAt: null
						},
						{
							id: '0192f8d1-2e3f-4a5b-6c7d-8e9f0a1b2c3d',
							kind: 'face_detect_rerun',
							state: 'queued',
							total: 9820,
							done: 0,
							failed: 0,
							startedAt: null,
							finishedAt: null
						},
						{
							id: '0192f8b1-4c5d-8e9f-a0b1-2c3d4e5f6a70',
							kind: 'places_rerun',
							state: 'succeeded',
							total: 12,
							done: 12,
							failed: 0,
							startedAt: new Date(now - 90 * 60 * 1000).toISOString(),
							finishedAt: new Date(now - 88 * 60 * 1000).toISOString()
						},
						{
							id: '0192f8c1-3d4e-9f0a-b1c2-3d4e5f6a7080',
							kind: 'ocr_rerun',
							state: 'failed',
							total: 5,
							done: 4,
							failed: 1,
							startedAt: new Date(now - 120 * 60 * 1000).toISOString(),
							finishedAt: new Date(now - 118 * 60 * 1000).toISOString()
						}
					]
				}
			})
		);
		await page.route('**/api/v1/admin/models', (route) =>
			route.fulfill({
				json: {
					models: [
						{
							id: 'ppocr-v4-mobile-ch',
							slot: 'ocr',
							version: 'v4-mobile',
							source: 'downloaded',
							license: 'Apache-2.0',
							notice: null,
							dim: null,
							inputSize: { width: 320, height: 320 },
							isActive: true,
							filesPresent: true,
							params: {}
						}
					],
					download: null,
					slots: [],
					official: []
				}
			})
		);
		await page.goto('/admin/jobs');

		const rowOf = (kind: string) => page.locator('article.jobs__row').filter({ hasText: kind }).first();

		// Running: the live bar, the counts in the run's own units and
		// the start.
		const running = rowOf('Embedding rerun');
		await expect(running.locator('[role="progressbar"]')).toHaveAttribute('aria-valuenow', '43');
		await expect(running).toContainText('4210 of 9820 assets');
		await expect(running).toContainText('Started');

		// Queued: no bar to draw, and the waiting sentence.
		const queued = rowOf('Face detection rerun');
		await expect(queued).toContainText('Waiting for the run ahead to finish.');
		await expect(queued.locator('[role="progressbar"]')).toHaveCount(0);

		// Succeeded: the closed bar, the finish and the duration.
		const succeeded = rowOf('Places rerun');
		await expect(succeeded.locator('[role="progressbar"]')).toHaveAttribute('aria-valuenow', '100');
		await expect(succeeded).toContainText('Finished');
		await expect(succeeded).toContainText('took 2:00');

		// Failed: the danger fill, the reason in the run's units, and
		// the retry the kind supports (its slot has an active model).
		const failed = rowOf('Text recognition rerun');
		await expect(failed.locator('.jobs__progress-fill--failed')).toHaveCount(1);
		await expect(failed.getByTestId('job-reason')).toContainText('1 item could not be processed');
		await expect(failed.getByTestId('job-retry')).toBeVisible();
	});
});
