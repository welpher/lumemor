// The E43 task-catalog story, walked end to end against the real
// compose stack: the jobs page's two faces (the fixed catalog and the
// recent-runs history, addressed by `?tab=runs`), the catalog's rows
// with their counts and pause switches, a pause/resume round trip
// through the run-budget document, and the instance-wide album rescan
// sweep -- the start, the history's one row, and the cancel of a
// running sweep through its catalog row.
//
// THE STACK ARRANGEMENT: one fresh lifetime owns this spec (the
// first-run bootstrap below creates the administrator, so a second
// spec on the same lifetime would answer 409). No model download is
// needed: the catalog's rows and the sweep's ingest pipeline run
// without the ML slots.
//
// THE SWEEP'S CANCEL WINDOW: cancelling a sweep of an unchanged album
// would race a near-instant diff, so the cancel scene stages a
// second album whose files all CHANGE between its creation scan and
// the sweep -- the sweep's change arm reads every file, keeping the
// run in flight long enough for the row's cancel to land at its own
// boundary. The staged tree lives in the writable fixture bind (the
// container sees it at /mnt/e2e-source-rw) and is removed by the
// spec; nothing new is committed.
//
// THE SCENES (in order):
//
//  1. The administrator signs in through the form and lands on the
//     jobs page: the catalog is the default face, its rows and the
//     budget card render, the five pause switches live in their rows
//     (never the card), and `?tab=runs` opens the history face.
//  2. The pause round trip: the place-resolution row's switch pauses
//     (the budget document carries it, the toast speaks it), and the
//     resume reverses both.
//  3. The sweep: a directory album over the read-only fixture is
//     created (its first scan is the creation's), "Rescan all" starts
//     the one run, the toast speaks the album count, the catalog's
//     latest line reads the sweep, and the history tab shows the ONE
//     "Album rescan (all)" row (never one per album).
//  4. The cancel: the staged second album's files change, the sweep
//     starts, the catalog row's cancel lands, and the run closes
//     cancelled with the committed album kept.

import { mkdir, readFile, rm, utimes, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's administrator: created through the first-run setup API
// in the suite's bootstrap, then used by the login scene.
const ADMIN_EMAIL = 'admin@e43-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The read-only fixture source (the compose bind): four real photos.
const SOURCE_DIR = '/mnt/e2e-source/trip-2026';
const ALBUM_TITLE = 'Trip 2026';

// The staged cancel-window album: an empty directory first (so the
// creation scan is instant), then many files, then a stamp change
// before the sweep.
const STAGED_DIR = '/mnt/e2e-source-rw/e43-sweep';
const STAGED_HOST_DIR = path.join(specDir, 'fixtures/m3-source-rw/e43-sweep');
const STAGED_ALBUM = 'Sweep window 2026';
const STAGED_FILES = 400;

// The scene bounds: the creation scan's pipeline runs inside the
// first two albums' scans, the sweep's cancel window inside the
// change storm.
const SCAN_TIMEOUT_MS = 180_000;
const SWEEP_TIMEOUT_MS = 180_000;

// The storage-state file the first scene writes and the later scenes
// replay (git-ignored, recreated per run).
const authStatePath = `${specDir}/.auth/e43-state.json`;

/** The jobs listing's runs, read through the API. */
async function listRuns(page: Page): Promise<Array<{ id?: string; kind?: string; state?: string }>> {
	const listing = await page.request.get('/api/v1/admin/jobs?limit=200');
	expect(listing.status()).toBe(200);
	const body = (await listing.json()) as { jobs?: Array<{ id?: string; kind?: string; state?: string }> };
	return body.jobs ?? [];
}

/** The ids of the runs of one kind already recorded. */
async function runIds(page: Page, kind: string): Promise<Set<string>> {
	const runs = await listRuns(page);
	return new Set(runs.filter((entry) => entry.kind === kind && entry.id !== undefined).map((entry) => entry.id!));
}

/**
 * Poll the API until a run of the kind reaches the state, EXCLUDING
 * the ids already recorded: a repeated sweep must not match the
 * earlier, already-closed one.
 */
async function waitRun(
	page: Page,
	kind: string,
	state: string,
	exclude: Set<string>,
	timeout: number
): Promise<string> {
	let id = '';
	await expect
		.poll(
			async () => {
				const runs = await listRuns(page);
				const run = runs.find(
					(entry) => entry.kind === kind && entry.state === state && !exclude.has(entry.id ?? '')
				);
				id = run?.id ?? '';
				return run?.state ?? 'missing';
			},
			{ timeout, intervals: [500, 1000, 2000] }
		)
		.toBe(state);
	expect(id).not.toBe('');
	return id;
}

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'E43 E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

test('scene 1: the catalog is the default face and ?tab=runs opens the history', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Fadmin%2Fjobs');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/admin\/jobs$/);
	await expect(page).toHaveTitle('Jobs · lumemor');

	// The two-tab control: tasks pressed by default.
	await expect(page.getByTestId('jobs-tab-tasks')).toHaveAttribute('aria-pressed', 'true');
	await expect(page.getByTestId('jobs-tab-runs')).toHaveAttribute('aria-pressed', 'false');

	// The fixed catalog's rows.
	for (const task of [
		'thumbnail',
		'metadata',
		'transcode',
		'ml',
		'places',
		'rebuild',
		'rescan',
		'foldersync',
		'models'
	]) {
		await expect(page.locator(`[data-task="${task}"]`)).toBeVisible();
	}
	// A fresh instance's queue rows are idle and the budget card is in.
	await expect(page.locator('[data-task="thumbnail"]')).toContainText('Idle');
	await expect(page.getByTestId('run-settings')).toBeVisible();
	// The five pause switches live in their rows now -- the card
	// carries none.
	await expect(page.getByTestId('task-pause-thumbnail')).toBeVisible();
	await expect(page.getByTestId('task-pause-places')).toBeVisible();
	await expect(page.locator('[data-testid="run-settings"] [data-testid^="run-pause-"]')).toHaveCount(0);

	// The catalog's one-off actions point at the existing surfaces.
	await expect(page.locator('[data-task="places"] a').first()).toHaveAttribute(
		'href',
		'/admin/settings?dialog=rebuild-places'
	);
	await expect(page.locator('[data-task="rebuild"] a').first()).toHaveAttribute(
		'href',
		'/admin/settings?dialog=rebuild-derivatives'
	);

	// The deep link opens the history face; the tasks face folds away.
	await page.goto('/admin/jobs?tab=runs');
	await expect(page.getByTestId('jobs-tab-runs')).toHaveAttribute('aria-pressed', 'true');
	await expect(page.locator('[data-jobs-panel="tasks"]')).toBeHidden();

	// And the switch back carries the address.
	await page.getByTestId('jobs-tab-tasks').click();
	await expect(page).toHaveURL(/\/admin\/jobs$/);
	await expect(page.locator('[data-jobs-panel="tasks"]')).toBeVisible();

	await page.context().storageState({ path: authStatePath });
});

test.describe('the administrator scenes', () => {
	test.use({ storageState: authStatePath });

	test('scene 2: the place-resolution pause round trip writes the budget', async ({ page }) => {
		await page.goto('/admin/jobs');
		await expect(page.getByTestId('task-pause-places')).toBeVisible();
		await expect(page.getByTestId('task-pause-places')).toHaveText('Pause');

		await page.getByTestId('task-pause-places').click();
		await expect(page.locator('.toast').filter({ hasText: 'Place resolution jobs paused.' })).toBeVisible();
		await expect(page.getByTestId('task-pause-places')).toHaveText('Resume');
		await expect
			.poll(async () => {
				const response = await page.request.get('/api/v1/admin/settings/run-budget');
				const body = (await response.json()) as { pausedJobs?: { places?: boolean } };
				return body.pausedJobs?.places;
			})
			.toBe(true);

		await page.getByTestId('task-pause-places').click();
		await expect(page.locator('.toast').filter({ hasText: 'Place resolution jobs resumed.' })).toBeVisible();
		await expect(page.getByTestId('task-pause-places')).toHaveText('Pause');
		await expect
			.poll(async () => {
				const response = await page.request.get('/api/v1/admin/settings/run-budget');
				const body = (await response.json()) as { pausedJobs?: { places?: boolean } };
				return body.pausedJobs?.places;
			})
			.toBe(false);
	});

	test('scene 3: the rescan-all sweep starts, and the history keeps one row', async ({ page }) => {
		// The directory album over the read-only fixture; its creation
		// enqueues the first scan.
		const created = await page.request.post('/api/v1/albums', {
			data: { title: ALBUM_TITLE, rootPath: SOURCE_DIR, audience: [] }
		});
		expect(created.status()).toBe(201);
		await waitRun(page, 'album_rescan', 'succeeded', new Set(), SCAN_TIMEOUT_MS);

		await page.goto('/admin/jobs');
		// The sweep's start: the toast speaks the album count and the row
		// converges on the fresh run.
		const beforeStart = await runIds(page, 'album_rescan_all');
		await page.getByTestId('task-rescan-all').click();
		await expect(page.locator('.toast').filter({ hasText: 'Rescanning all 1 albums.' })).toBeVisible();
		const sweepId = await waitRun(page, 'album_rescan_all', 'succeeded', beforeStart, SWEEP_TIMEOUT_MS);
		expect(sweepId).not.toBe('');

		// The history tab: ONE sweep row, its units are albums.
		await page.goto('/admin/jobs?tab=runs');
		const sweepRow = page.locator('article.jobs__row').filter({ hasText: 'Album rescan (all)' });
		await expect(sweepRow).toHaveCount(1);
		await expect(sweepRow).toContainText('1 of 1 albums');
		await expect(sweepRow).toContainText('Succeeded');

		// The catalog's latest line reads the closed sweep.
		await page.goto('/admin/jobs');
		await expect(page.locator('[data-task="rescan"]')).toContainText('Latest: all 1 albums');
		await expect(page.locator('[data-task="rescan"]')).toContainText('Succeeded');
	});

	test('scene 4: a running sweep is cancellable from its catalog row', async ({ page }) => {
		test.setTimeout(SCAN_TIMEOUT_MS + SWEEP_TIMEOUT_MS);

		// The staged album: an empty directory first (instant creation
		// scan), then many files that all change before the sweep.
		await rm(STAGED_HOST_DIR, { recursive: true, force: true });
		await mkdir(STAGED_HOST_DIR, { recursive: true });
		const created = await page.request.post('/api/v1/albums', {
			data: { title: STAGED_ALBUM, rootPath: STAGED_DIR, audience: [] }
		});
		expect(created.status()).toBe(201);
		await waitRun(page, 'album_rescan', 'succeeded', new Set(), SCAN_TIMEOUT_MS);

		// The bytes: one real fixture repeated under distinct names.
		const photo = await readFile(path.join(repoRoot, 'server/internal/media/testdata/portrait-orientation6.jpg'));
		for (let index = 0; index < STAGED_FILES; index += 1) {
			await writeFile(path.join(STAGED_HOST_DIR, `sweep-${String(index).padStart(4, '0')}.jpg`), photo);
		}

		// The album's first scan ingests the new files (the sweep's window
		// must not be the ingest's own). The creation scan ran over the
		// empty directory, so this is the album's SECOND scan.
		await page.goto('/admin/jobs');
		const beforeIngest = await runIds(page, 'album_rescan_all');
		await page.getByTestId('task-rescan-all').click();
		await expect(page.locator('.toast').filter({ hasText: 'Rescanning all 2 albums.' })).toBeVisible();
		await waitRun(page, 'album_rescan_all', 'succeeded', beforeIngest, SWEEP_TIMEOUT_MS);

		// Stamp every file: the sweep's change arm re-reads each one, so
		// the run stays in flight long enough for the cancel.
		const stamp = new Date(Date.now() + 60_000);
		for (let index = 0; index < STAGED_FILES; index += 1) {
			await utimes(path.join(STAGED_HOST_DIR, `sweep-${String(index).padStart(4, '0')}.jpg`), stamp, stamp);
		}

		await page.goto('/admin/jobs');
		const beforeCancel = await runIds(page, 'album_rescan_all');
		await page.getByTestId('task-rescan-all').click();
		// The cancel rides the row the moment the fresh summary lands.
		await expect(page.getByTestId('task-cancel-rescan')).toBeVisible({ timeout: 30_000 });
		await page.getByTestId('task-cancel-rescan').click();
		await expect(page.locator('.toast').filter({ hasText: 'Cancelled.' })).toBeVisible();

		// The run closes cancelled; the committed album's work stays.
		const cancelledId = await waitRun(page, 'album_rescan_all', 'cancelled', beforeCancel, SWEEP_TIMEOUT_MS);
		expect(cancelledId).not.toBe('');
		await page.goto('/admin/jobs?tab=runs');
		const cancelledRow = page.locator('article.jobs__row').filter({ hasText: 'Album rescan (all)' }).first();
		await expect(cancelledRow).toContainText('Cancelled');

		// The staged tree is the spec's own: remove it.
		await rm(STAGED_HOST_DIR, { recursive: true, force: true });
	});
});
