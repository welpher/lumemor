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
// into the models dir before the stack starts (clip-fixture-512 and
// clip-fixture-768, the second at a different dimension) and uploads
// the face-portrait fixtures.
//
// THE SCENES (the M3 model sources story, in order):
//
//  1. Signing in through the form lands on the models page whose
//     four slot cards honestly read Not downloaded (the mirror
//     carries no weights: an unregistered group is the fresh
//     instance's truth) and the empty state's download CTA. The
//     admin rail's items carry their 18px glyphs.
//  2. The CTA drives the default group's download: the progress
//     region takes over, the run completes, and the four cards read
//     Active.
//  3. The task panel lists the succeeded model_download row with its
//     converged byte counts.
//  4. The directory scan (through the admin API -- the browser
//     surface has no scan button, the same seeding pattern the M2
//     story uses) registers the two fixture models. The semantic
//     slot's switch dialog shows them in the Added group; selecting
//     the 512 fixture and saving moves the card onto the custom
//     model (the Custom badge, the source meta line).
//  5. The failed probe: a 128-dim file cannot pass the embed slot's
//     contract -- the probe's error state names the check, and the
//     save switch stays disabled (a failed probe leaves nothing
//     saved).
//  6. The passing probe and the same-bytes reuse: the card switches
//     back onto the official entry first; the 512 fixture's bytes
//     (the file the dialog drops) then pass the probe and the save
//     activates the ALREADY-REGISTERED model with the same digest --
//     the same bytes are one model, never a second copy.
//  7. The download-sources footer: collapsed by default, its summary
//     names the count; adding a host and saving persists it (the
//     summary follows).
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story --
// see playwright.config.ts) and share the browser session: scene 1
// signs in through the form and saves the storage state, every later
// scene replays it. Each scene navigates explicitly to its starting
// route, so a failure report names the exact scene that broke.

import { fileURLToPath } from 'node:url';
import { expect, test } from '@playwright/test';
import path from 'node:path';

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

// The fixture models the acceptance script stages for the dialog
// scenes: the same 512 dimension as the default clip model (the
// coexistence branch) and a 768 one (the destructive branch).
const FIXTURE_512_ID = 'clip-fixture-512';

// The ONNX fixtures the probe scenes drop into the file input: the
// 128-dim one fails the embed slot's contract, the 512 one passes and
// reuses the already-registered bytes.
const FIXTURE_128_FILE = path.join(specDir, '..', '..', 'server', 'internal', 'ml', 'testdata', 'const128.onnx');
const FIXTURE_512_FILE = path.join(specDir, '..', '..', 'server', 'internal', 'ml', 'testdata', 'const512.onnx');

// The download progress bound: the cold upstream transfer of the
// group (about 771 MB with the search's text tower) is minutes, the
// warm re-verification is seconds; the scene allows the cold bound.
const DOWNLOAD_TIMEOUT_MS = 900_000;

// The storage-state file scene 1 writes and every later scene replays
// (git-ignored, recreated per run).
const authStatePath = `${specDir}/.auth/m3-state.json`;

// The four slot cards, by their data-slot (the pipeline slots the
// four-card face covers; the text tower rides the semantic card).
const CARD_SLOTS = ['face_detect', 'face_recognize', 'clip_embed', 'ocr'] as const;

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

test('scene 1: signing in through the form lands on the honest un-downloaded models page', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Fadmin%2Fmodels');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();

	// The post-login target round-trips onto the admin models page,
	// whose page-header row names the renamed surface (the 2026-11-09
	// naming pin: the domain-named "Machine learning").
	await expect(page).toHaveURL(/\/admin\/models$/);
	await expect(page.getByRole('heading', { level: 1, name: 'Machine learning', exact: true })).toBeVisible();

	// The empty state carries the honest copy and the download CTA --
	// the v1 form of the wizard's model download step.
	await expect(page.getByRole('heading', { level: 2, name: 'No models yet', exact: true })).toBeVisible();
	await expect(
		page.getByText(
			'The default model group has not been downloaded. Face detection, face recognition, semantic search and text recognition stay off until it lands.'
		)
	).toBeVisible();
	await expect(page.getByRole('button', { name: 'Download the default models', exact: true })).toBeVisible();

	// The four slot cards each honestly read Not downloaded: the
	// mirror carries no weights and nothing is registered yet.
	for (const slot of CARD_SLOTS) {
		const card = page.locator(`.models__card[data-slot="${slot}"]`);
		await expect(card).toBeVisible();
		await expect(card.locator('[data-state="not_downloaded"]')).toHaveText('Not downloaded');
		// The switch action sits first in every card's action row.
		await expect(card.getByRole('button', { name: 'Swap model' })).toBeVisible();
	}

	// The admin rail's items name themselves with their glyphs (the
	// icon-first rule): every item carries an 18px svg before the
	// label.
	const rail = page.getByRole('navigation', { name: 'Administration' });
	for (const label of ['Users', 'Instance settings', 'Machine learning', 'Jobs']) {
		const item = rail.getByRole('link', { name: label, exact: true });
		await expect(item).toBeVisible();
		expect(await item.locator('svg').count()).toBe(1);
	}

	// The download-sources footer: collapsed, its summary line names
	// the shipped host count (the manifest's own hosts plus the
	// mirror host).
	const sources = page.locator('details[data-testid="models-sources"]');
	await expect(sources).toBeVisible();
	await expect(sources).not.toHaveAttribute('open');
	await expect(sources.locator('[data-testid="models-sources-summary"]')).toHaveText(/hosts$/);

	// Hand the session to every later scene: the storage state rides
	// the login cookie (the fresh contexts replay it).
	await page.context().storageState({ path: authStatePath });
});

// The signed-in half of the story: every scene starts from a fresh
// context that replays the login cookie scene 1 saved. The serial
// mode guarantees the state file exists by the time these scenes run.
test.describe('the signed-in scenes', () => {
	test.use({ storageState: authStatePath });

	test('scene 2: the download CTA completes and the four cards read Active', async ({ page }) => {
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
		// way the four cards land Active.
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
					let active = 0;
					for (const slot of CARD_SLOTS) {
						if (
							(await page.locator(`.models__card[data-slot="${slot}"] [data-state="active"]`).count()) ===
							1
						) {
							active += 1;
						}
					}
					return active === CARD_SLOTS.length;
				},
				{ timeout: DOWNLOAD_TIMEOUT_MS, intervals: [200, 500, 1000, 2000] }
			)
			.toBe(true);
		test.info().annotations.push({
			type: 'note',
			description: `download progress: peak=${peak}, advanced=${sawAdvance}`
		});

		// The empty state's CTA is gone (the listing owns the surface
		// now) and the four cards carry the Active badge.
		await expect(cta).toHaveCount(0);
		for (const slot of CARD_SLOTS) {
			await expect(page.locator(`.models__card[data-slot="${slot}"] [data-state="active"]`)).toHaveText('Active');
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

	test('scene 4: the scan registers the fixtures and the switch moves the semantic card onto a custom model', async ({
		page
	}) => {
		await page.goto('/admin/models');

		// The directory scan rides the admin API (the browser surface
		// has no scan button): the request rides the page's own context
		// and its login cookie.
		const scan = await page.request.post('/api/v1/admin/models/scan');
		expect(scan.status()).toBe(200);

		// The listing's models array carries the fixtures; the
		// switch dialog's Added group offers them for the semantic
		// slot.
		await page.goto('/admin/models');
		await page.locator('.models__card[data-slot="clip_embed"]').getByRole('button', { name: 'Swap model' }).click();
		const dialog = page.getByRole('dialog', { name: 'Swap the Semantic embeddings model', exact: true });
		await expect(dialog).toBeVisible();

		// The slot contract names the output dimension and the red
		// consequence line.
		await expect(dialog).toContainText('Output 512 dims');
		await expect(dialog).toContainText('cannot be undone');

		const select = dialog.locator('select');
		// The two groups: the optgroup labels live on the DOM
		// attributes, not the options' text.
		const groupLabels = await select
			.locator('optgroup')
			.evaluateAll((nodes) => nodes.map((node) => (node as HTMLOptGroupElement).label));
		expect(groupLabels).toContain('Official list');
		expect(groupLabels).toContain('Added');

		// Selecting the 512 fixture and saving performs the switch:
		// the card's badge reads Active and its meta line carries the
		// custom source facts.
		await select.selectOption(FIXTURE_512_ID);
		await dialog.getByRole('button', { name: 'Save and switch', exact: true }).click();
		await expect(dialog).toHaveCount(0);

		const card = page.locator('.models__card[data-slot="clip_embed"]');
		await expect(card.locator('[data-state="active"]')).toHaveText('Active');
		await expect(card).toContainText('Custom');
		await expect(card).toContainText('downloaded');
	});

	test('scene 5: the failed probe names the check and the save stays disabled', async ({ page }) => {
		await page.goto('/admin/models');
		await page.locator('.models__card[data-slot="clip_embed"]').getByRole('button', { name: 'Swap model' }).click();
		const dialog = page.getByRole('dialog', { name: 'Swap the Semantic embeddings model', exact: true });
		await expect(dialog).toBeVisible();

		// A 128-dim file cannot pass the embed slot's 512 contract: the
		// probe's error state names the check, and the save switch
		// never enables.
		// The file tab first: the URL tab is the import panel's default.
		await dialog.getByRole('tab', { name: 'Local file', exact: true }).click();
		await dialog.locator('input[type="file"]').setInputFiles(FIXTURE_128_FILE);
		await dialog.getByRole('button', { name: 'Check', exact: true }).click();
		const failure = dialog.locator('[data-probe-state="error"]');
		await expect(failure).toBeVisible({ timeout: 60_000 });
		await expect(failure).toContainText('This model cannot be used');
		await expect(dialog).toContainText(/dimensions?/);
		await expect(dialog.getByRole('button', { name: 'Save and switch', exact: true })).toBeDisabled();

		// Canceling keeps everything as it was: nothing was saved.
		await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
		await expect(dialog).toHaveCount(0);
		const card = page.locator('.models__card[data-slot="clip_embed"]');
		await expect(card.locator('[data-state="active"]')).toHaveText('Active');
	});

	test('scene 6: the passing probe switches back through the same bytes, never a second copy', async ({ page }) => {
		await page.goto('/admin/models');

		// First the card switches back onto the official entry: the
		// dialog's select names the official entry (it preselects the
		// running custom), the save performs the switch, and the
		// custom meta line is gone.
		await page.locator('.models__card[data-slot="clip_embed"]').getByRole('button', { name: 'Swap model' }).click();
		const dialog = page.getByRole('dialog', { name: 'Swap the Semantic embeddings model', exact: true });
		await expect(dialog).toBeVisible();
		await dialog.locator('select').selectOption('cnclip-vit-b16-vision');
		await dialog.getByRole('button', { name: 'Save and switch', exact: true }).click();
		await expect(dialog).toHaveCount(0);
		const card = page.locator('.models__card[data-slot="clip_embed"]');
		await expect(card.locator('[data-state="active"]')).toHaveText('Active');
		await expect(card).not.toContainText('Custom');

		// The 512 fixture's bytes pass the probe (they match the slot's
		// contract), and the save activates the ALREADY-REGISTERED
		// model carrying the same digest -- the same bytes are one
		// model, never a second copy.
		await card.getByRole('button', { name: 'Swap model' }).click();
		const dialog2 = page.getByRole('dialog', { name: 'Swap the Semantic embeddings model', exact: true });
		await expect(dialog2).toBeVisible();
		// The file tab first: the URL tab is the import panel's default.
		await dialog2.getByRole('tab', { name: 'Local file', exact: true }).click();
		await dialog2.locator('input[type="file"]').setInputFiles(FIXTURE_512_FILE);
		await dialog2.getByRole('button', { name: 'Check', exact: true }).click();
		const ok = dialog2.locator('[data-probe-state="ok"]');
		await expect(ok).toBeVisible({ timeout: 60_000 });
		await expect(ok).toContainText('The checks passed');
		await dialog2.getByRole('button', { name: 'Save and switch', exact: true }).click();
		await expect(dialog2).toHaveCount(0);
		await expect(card.locator('[data-state="active"]')).toHaveText('Active');
		await expect(card).toContainText('Custom');
		// The active model is the fixture's registration, not a
		// content-named duplicate (the listing's models carry it).
		const listing = await page.request.get('/api/v1/admin/models');
		const body = (await listing.json()) as { models: Array<{ id: string; slot: string; isActive: boolean }> };
		const activeIds = body.models.filter((model) => model.isActive).map((model) => model.id);
		expect(activeIds).toContain(FIXTURE_512_ID);
	});

	test('scene 7: the download-sources footer edits the hosts and the summary follows', async ({ page }) => {
		await page.goto('/admin/models');

		const sources = page.locator('details[data-testid="models-sources"]');
		await sources.locator('summary').click();
		await expect(sources).toHaveAttribute('open');

		// The shipped chips render; adding a host appends one, and the
		// save persists the whole document (the summary's count
		// follows).
		const chips = sources.locator('.models__host');
		const before = await chips.count();
		expect(before).toBeGreaterThanOrEqual(4);
		await sources.locator('.models__host-input').fill('models.example.net');
		await sources.getByRole('button', { name: 'Add host', exact: true }).click();
		await expect(chips).toHaveCount(before + 1);
		await sources.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(sources).toContainText('The download sources are saved.');
		await expect(sources.locator('[data-testid="models-sources-summary"]')).toHaveText(
			new RegExp(`^${before + 1} hosts`)
		);

		// The persisted answer: a fresh navigation reads the saved
		// list back.
		await page.goto('/admin/models');
		const sourcesAfter = page.locator('details[data-testid="models-sources"]');
		await expect(sourcesAfter.locator('[data-testid="models-sources-summary"]')).toHaveText(
			new RegExp(`^${before + 1} hosts`)
		);
	});
});
