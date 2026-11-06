// The M3 model management story of the web app, walked end to end
// against the real compose stack. scripts/acceptance_m3a.sh builds
// and owns the stack (its phase 3), waits for /healthz and then runs
// this spec; the spec itself only talks to http://localhost:8244.
//
// THE STACK ARRANGEMENT (the honest reuse path, documented): the
// stack is fresh (its database volume was removed before it started)
// but the models ride the SAME storage bind mount the acceptance's
// phase 2 downloaded into -- ./lumemor persists across the script's
// stack lifetimes, and the download service's already-verified skip
// re-registers the weights without re-fetching them. The download
// this story triggers therefore completes quickly (a digest
// re-verification) unless the mount is cold, in which case it
// transfers the group from the stack's configured source. The script
// also stages two self-made constant clip-embedding fixture models
// into the models dir before the stack starts: the activation dialog
// scenes need a second model in the clip slot, and a third with a
// different dimension for the destructive branch's warning.
//
// THE SCENES (the M3 model management story, in order):
//
//  1. Signing in through the form lands on the deep-linked models
//     page, whose empty state carries the download CTA (the v1 form
//     of the first-run wizard's model download step) and the honest
//     per-slot emptiness.
//  2. The CTA drives the default group's download: the progress
//     region takes over, the bar advances (observed; a warm models
//     dir may legitimately complete before an intermediate sample
//     lands), the run completes, and the five default models render
//     with their Active badges.
//  3. The task panel lists the succeeded model_download row with its
//     converged byte counts.
//  4. The directory scan (through the admin API -- the browser
//     surface has no scan button, the same seeding pattern the M2
//     story uses for its Live Photo pair) registers the two fixture
//     models, which the page lists as inactive custom models.
//  5. The same-dimension switch: the confirm dialog renders the
//     coexistence copy (the products are kept, switching back
//     restores them); confirming moves the Active badge to the
//     fixture model.
//  6. The cross-dimension switch: the confirm dialog renders the
//     wipe warning (every embedding is deleted, the vector column is
//     re-typed); canceling leaves the badge where scene 5 put it.
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story --
// see playwright.config.ts) and share the browser session: scene 1
// signs in through the form and saves the storage state, every later
// scene replays it. Each scene navigates explicitly to its starting
// route, so a failure report names the exact scene that broke.

import { fileURLToPath } from 'node:url';
import { expect, test } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's administrator: created through the first-run setup API
// in the suite's bootstrap (the acceptance scripts' pattern), then
// used by the login scene. The stack is always fresh, so the fixed
// identity is deterministic.
const ADMIN_EMAIL = 'admin@m3a-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The five default models of the manifest, by the row identity the
// page renders (article[data-model-id]) -- the vision tower, the
// search's text tower, the face pair and the OCR pair.
const DEFAULT_MODEL_IDS = [
	'yunet-2023mar',
	'sface-2021dec',
	'cnclip-vit-b16-vision',
	'cnclip-vit-b16-text',
	'ppocr-v4-mobile-ch'
] as const;

// The two fixture models the acceptance script stages for the dialog
// scenes: the same 512 dimension as the default clip model (the
// coexistence branch) and a 768 one (the destructive branch).
const FIXTURE_512_ID = 'clip-fixture-512';
const FIXTURE_768_ID = 'clip-fixture-768';

// The download progress bound: the cold upstream transfer of the
// group (about 771 MB with the search's text tower) is minutes, the
// warm re-verification is seconds; the scene allows the cold bound.
const DOWNLOAD_TIMEOUT_MS = 900_000;

// The storage-state file scene 1 writes and every later scene replays
// (git-ignored, recreated per run).
const authStatePath = `${specDir}/.auth/m3-state.json`;

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance. The
	// browser scenes then only exercise the web app's own surfaces
	// (the login scene performs the actual sign-in through the form).
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'M3a E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

test('scene 1: signing in through the form lands on the empty models page', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Fadmin%2Fmodels');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();

	// The post-login target round-trips onto the admin models page.
	await expect(page).toHaveURL(/\/admin\/models$/);
	await expect(page.getByRole('heading', { level: 1, name: 'Models', exact: true })).toBeVisible();

	// The empty state carries the honest copy and the download CTA --
	// the v1 form of the wizard's model download step.
	await expect(page.getByRole('heading', { level: 2, name: 'No models yet', exact: true })).toBeVisible();
	await expect(
		page.getByText('The default model group has not been downloaded. Face detection, face recognition, semantic search and text recognition stay off until it lands.')
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Download the default models', exact: true })
	).toBeVisible();

	// Every slot section reports its emptiness (the five slots:
	// detection, recognition, the embedding and text towers of the
	// semantic arm, and OCR).
	await expect(page.getByText('No model registered for this slot.')).toHaveCount(5);

	// Hand the session to every later scene: the storage state rides
	// the login cookie (the fresh contexts replay it).
	await page.context().storageState({ path: authStatePath });
});

// The signed-in half of the story: every scene starts from a fresh
// context that replays the login cookie scene 1 saved. The serial
// mode guarantees the state file exists by the time these scenes run.
test.describe('the signed-in scenes', () => {
	test.use({ storageState: authStatePath });

	test('scene 2: the download CTA completes and activates the five models', async ({ page }) => {
		test.setTimeout(DOWNLOAD_TIMEOUT_MS + 60_000);
		await page.goto('/admin/models');
		const cta = page.getByRole('button', { name: 'Download the default models', exact: true });
		await expect(cta).toBeVisible();
		await cta.click();

		// The progress region takes over the action surface while the
		// run is queued or running. A warm models dir (the acceptance's
		// phase 2 already landed the weights on the shared bind mount)
		// may complete before any intermediate sample is possible, so
		// the honest convergence is: the region carries the run, and
		// either the bar advances or the run completes outright; either
		// way the five model rows land.
		const bar = page.getByRole('progressbar');
		let peak = -1;
		let sawAdvance = false;
		await expect
			.poll(
				async () => {
					if ((await bar.count()) === 1) {
						const now = Number(await bar.getAttribute('aria-valuenow'));
						if (now > peak) {
							if (peak >= 0) {
								sawAdvance = true;
							}
							peak = now;
						}
						return false;
					}
					return (await page.locator('article[data-model-id]').count()) === 5;
				},
				{ timeout: DOWNLOAD_TIMEOUT_MS, intervals: [200, 500, 1000, 2000] }
			)
			.toBe(true);
		test
			.info()
			.annotations.push({ type: 'note', description: `download progress: peak=${peak}, advanced=${sawAdvance}` });

		// The empty state's CTA is gone (the listing owns the surface
		// now) and each default model carries its Active badge.
		await expect(cta).toHaveCount(0);
		for (const id of DEFAULT_MODEL_IDS) {
			await expect(page.locator(`article[data-model-id="${id}"] .models__badge`)).toHaveText('Active');
		}
	});

	test('scene 3: the task panel lists the succeeded download row', async ({ page }) => {
		await page.goto('/admin/models');
		await page
			.getByRole('navigation', { name: 'Administration' })
			.getByRole('link', { name: 'Jobs', exact: true })
			.click();
		await expect(page).toHaveURL(/\/admin\/jobs$/);
		await expect(page.getByRole('heading', { level: 1, name: 'Jobs', exact: true })).toBeVisible();

		// The model download's row: the succeeded state and the
		// converged byte counts (the group's pinned bytes, the text
		// tower included: 808,972,863 bytes = 771.5 MB).
		const row = page.locator('article.jobs__row').filter({ hasText: 'Model download' }).first();
		await expect(row).toBeVisible();
		await expect(row.getByText('Succeeded', { exact: true })).toBeVisible();
		await expect(row.getByText(/of 771\.5 MB/)).toBeVisible();
	});

	test('scene 4: the scan registers the fixture models as inactive customs', async ({ page }) => {
		await page.goto('/admin/models');

		// The directory scan rides the admin API (the browser surface
		// has no scan button): the request rides the page's own context
		// and its login cookie, the page renders the outcome.
		const scan = await page.request.post('/api/v1/admin/models/scan');
		expect(scan.status()).toBe(200);

		// A fresh navigation re-lists: the two fixture models appear
		// under the Semantic embeddings slot, unbadged.
		await page.goto('/admin/models');
		for (const id of [FIXTURE_512_ID, FIXTURE_768_ID]) {
			const row = page.locator(`article[data-model-id="${id}"]`);
			await expect(row).toBeVisible();
			await expect(row.locator('.models__badge')).toHaveText('');
		}
	});

	test('scene 5: the same-dimension switch confirms and moves the badge', async ({ page }) => {
		await page.goto('/admin/models');

		// Asking to activate the 512-dim fixture opens the confirm
		// dialog with the coexistence copy: the previous model's
		// products are kept and switching back restores them.
		await page
			.locator(`article[data-model-id="${FIXTURE_512_ID}"]`)
			.getByRole('button', { name: 'Activate', exact: true })
			.click();
		const dialog = page.getByRole('dialog', { name: 'Activate this model?', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText("The previous model's products are kept");
		await expect(dialog).toContainText('switching back restores them');
		await expect(dialog).toContainText('needs a rerun over the library before its products appear');

		// Confirming performs the switch: the badge moves to the
		// fixture model and leaves the default clip model.
		await dialog.getByRole('button', { name: 'Activate', exact: true }).click();
		await expect(dialog).toHaveCount(0);
		await expect(page.locator(`article[data-model-id="${FIXTURE_512_ID}"] .models__badge`)).toHaveText(
			'Active'
		);
		await expect(
			page.locator('article[data-model-id="cnclip-vit-b16-vision"] .models__badge')
		).toHaveText('');
	});

	test('scene 6: the cross-dimension switch warns and canceling keeps the badge', async ({ page }) => {
		await page.goto('/admin/models');

		// Asking to activate the 768-dim fixture opens the confirm
		// dialog with the destructive branch's warning: every embedding
		// is deleted and the vector column is re-typed.
		await page
			.locator(`article[data-model-id="${FIXTURE_768_ID}"]`)
			.getByRole('button', { name: 'Activate', exact: true })
			.click();
		const dialog = page.getByRole('dialog', { name: 'Activate this model?', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText('a different embedding dimension');
		await expect(dialog).toContainText('every embedding is deleted and the vector column is re-typed');
		await expect(dialog).toContainText('switching back does not bring the deleted embeddings back');

		// Canceling keeps the activation where scene 5 left it: the
		// destructive switch never ran.
		await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
		await expect(dialog).toHaveCount(0);
		await expect(page.locator(`article[data-model-id="${FIXTURE_512_ID}"] .models__badge`)).toHaveText(
			'Active'
		);
	});
});
