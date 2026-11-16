// The E41 admin-surfaces story, walked end to end against the real
// compose stack: the users page's breathing list rows, the task
// panel's speaking rows (a rescan names its album, a clean scan reads
// its own unchanged count), the cancel's faces (the queued run
// cancelled for real through the panel, the finished run's 409, the
// 409's inline face), and the model params' domain (the defaults, the
// out-of-bounds refusal, the dispatch reading the fresh value).
//
// THE STACK ARRANGEMENT: one fresh lifetime owns this spec (the
// first-run bootstrap below creates the administrator, so a second
// spec on the same lifetime would answer 409). The model group's
// download rides the offline mirror (the acceptance chain's own
// arrangement); the scenes that need the models run AFTER the
// download scene, and the params-dispatch scene is the reason the
// download happens here at all.
//
// THE SCENES (in order):
//
//  1. The administrator signs in through the form and lands on the
//     users page: the row's stacked identity -- the email sits UNDER
//     the name -- with the role/status chips inline, the actions
//     right, and the joined date's wrapper hiding at 520 and showing
//     at 760.
//  2. The models page before any model exists: the two face slots'
//     param blocks render the pipeline's own defaults, the
//     out-of-bounds edit refuses INLINE (the real 400 behind it,
//     proven through the API), and the valid save toasts and
//     persists.
//  3. The task panel's speaking rows: the directory album's rescan
//     names its album, the clean second scan reads "photos
//     unchanged", and a finished run answers the cancel with 409
//     JOB_NOT_CANCELLABLE.
//  4. The default group's download completes and the four cards read
//     Active (the models the later scenes' dispatch needs).
//  5. The queued-run cancel, for real: the ML pause holds a face
//     detect rerun queued, the panel's cancel closes it cancelled at
//     once, and nothing ever ran.
//  6. The running row's cancel faces over the injected listing: the
//     204 toasts the honest account, the row converges to the
//     cancelled chip, and a 409 lands INLINE under its own row.
//  7. The params' dispatch: the minimum score raised past any real
//     detection's confidence re-runs the face detect slot -- the
//     portrait's face leaves the library; restored, the rerun brings
//     it back. The dispatch read the fresh value.
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story)
// and share the browser session: scene 1 signs in through the form
// and saves the storage state, every later scene replays it.

import { fileURLToPath } from 'node:url';
import { expect, test, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's administrator: created through the first-run setup API
// in the suite's bootstrap, then used by the login scene.
const ADMIN_EMAIL = 'admin@e41-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The directory album's source: the read-only fixture bind (four
// real photos), mounted at /mnt/e2e-source.
const SOURCE_DIR = '/mnt/e2e-source/trip-2026';
const ALBUM_TITLE = 'Trip 2026';

// The download progress bound: the offline mirror serves the group
// from the local cache, so the transfer is seconds-to-minutes; the
// budget absorbs a cold mirror start.
const DOWNLOAD_TIMEOUT_MS = 900_000;

// The storage state scene 1 writes and the later scenes replay
// (git-ignored, recreated per run).
const authStatePath = `${specDir}/.auth/e41-state.json`;

/** One account's row on the users list, addressed by its email. */
function userRow(page: Page, email: string) {
	return page.locator('.users__row').filter({ hasText: email }).first();
}

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'E41 E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

test('scene 1: the users rows breathe -- the email under the name, the chips inline, the date on its wrapper', async ({
	page
}) => {
	await page.goto('/login?redirectTo=%2Fadmin%2Fusers');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/admin\/users$/);
	await expect(page).toHaveTitle('Users · lumemor');

	// THE DECISIVE ASSERTION (the plan's own words): the email sits
	// under the name -- the stacked identity of the breathing row.
	// One real account renders (the fresh instance's administrator).
	const row = userRow(page, ADMIN_EMAIL);
	await expect(row).toBeVisible();
	const name = row.locator('.users__name');
	const email = row.locator('.users__email');
	await expect(name).toHaveText('E41 E2E');
	await expect(email).toHaveText(ADMIN_EMAIL);
	const nameBox = await name.boundingBox();
	const emailBox = await email.boundingBox();
	expect(nameBox).not.toBeNull();
	expect(emailBox).not.toBeNull();
	expect(emailBox!.y).toBeGreaterThanOrEqual(nameBox!.y + nameBox!.height - 1);

	// The chips ride the name's own line (inline, not their own
	// column): the role chip and the status chip sit between the
	// name's line's top and the email's top.
	const roleChip = row.locator('.users__badge--role');
	const statusChip = row.locator('.users__badge--status');
	await expect(roleChip).toHaveText('Administrator');
	await expect(statusChip).toHaveText('Active');
	const roleBox = await roleChip.boundingBox();
	expect(roleBox!.y).toBeLessThanOrEqual(emailBox!.y);

	// The actions sit right of the identity (the row's last block).
	const actionsBox = await row.locator('.users__actions').boundingBox();
	expect(actionsBox!.x).toBeGreaterThanOrEqual(emailBox!.x + emailBox!.width);

	// The joined date's WRAPPER carries the narrow hiding (the E37
	// review's N7 lesson): hidden at 520, shown at 760 -- the wrapper
	// element itself, never a fight with a component base class.
	const joined = row.locator('.users__joined');
	await expect(joined).toContainText('Joined');
	await page.setViewportSize({ width: 520, height: 900 });
	await expect(joined).toBeHidden();
	await page.setViewportSize({ width: 760, height: 900 });
	await expect(joined).toBeVisible();
	await page.setViewportSize({ width: 1280, height: 900 });

	// The admin session rides to every later scene.
	await page.context().storageState({ path: authStatePath });
});

test.describe('the administrator scenes', () => {
	test.use({ storageState: authStatePath });

	test('scene 2: the model params render the pipeline defaults and refuse out-of-bounds', async ({ page }) => {
		await page.goto('/admin/models');

		// The two face slots carry their param blocks (the honest set:
		// the other two slots render none), seeded with the pipeline's
		// own defaults.
		const minScore = page.getByTestId('models-param-min-score');
		const cluster = page.getByTestId('models-param-cluster-threshold');
		await expect(minScore).toHaveValue('0.6');
		await expect(cluster).toHaveValue('0.55');
		const detectCard = page.locator('.models__card[data-slot="face_detect"]');
		const recognizeCard = page.locator('.models__card[data-slot="face_recognize"]');
		await expect(detectCard).toContainText('Faces below this score never enter the library');
		await expect(recognizeCard).toContainText('Embeddings closer than this belong to one person');
		await expect(page.locator('.models__card[data-slot="clip_embed"] .models__params')).toHaveCount(0);
		await expect(page.locator('.models__card[data-slot="ocr"] .models__params')).toHaveCount(0);

		// The out-of-bounds edit: the refusal lands INLINE under the
		// slot's own row, and nothing is written.
		await minScore.fill('1.5');
		await minScore.dispatchEvent('change');
		await expect(page.locator('.models__param-error').first()).toBeVisible();

		// THE SERVER'S OWN 400: the same out-of-bounds body through
		// the API answers VALIDATION_FAILED naming the field.
		const read = await page.request.get('/api/v1/admin/settings/models');
		expect(read.status()).toBe(200);
		const settings = (await read.json()) as {
			autoDownload: boolean;
			allowedHosts: string[];
			mirrorBaseUrl: string;
			params: { faceDetectMinScore: number; faceRecognizeThreshold: number };
		};
		const refused = await page.request.put('/api/v1/admin/settings/models', {
			data: { ...settings, params: { ...settings.params, faceDetectMinScore: 1.5 } }
		});
		expect(refused.status()).toBe(400);
		const envelope = (await refused.json()) as { code?: string; details?: { fields?: string[] } };
		expect(envelope.code).toBe('VALIDATION_FAILED');
		expect(envelope.details?.fields ?? []).toContain('params.faceDetectMinScore');

		// The valid save: the toast speaks, and a fresh navigation
		// reads the saved value back.
		await minScore.fill('0.8');
		await minScore.dispatchEvent('change');
		await expect(page.locator('.toast')).toContainText('The model params are saved.');
		await page.goto('/admin/models');
		await expect(page.getByTestId('models-param-min-score')).toHaveValue('0.8');
		await expect(page.getByTestId('models-param-cluster-threshold')).toHaveValue('0.55');

		// The pipeline's own default rides back (the later scenes'
		// detections run at the shipped 0.6).
		await page.getByTestId('models-param-min-score').fill('0.6');
		await page.getByTestId('models-param-min-score').dispatchEvent('change');
		await expect(page.locator('.toast')).toContainText('The model params are saved.');
		await page.goto('/admin/models');
		await expect(page.getByTestId('models-param-min-score')).toHaveValue('0.6');
	});

	test('scene 3: the rescan rows speak -- the album named, the clean scan counted', async ({ page }) => {
		// The directory album over the read-only fixture bind, created
		// through the API (the browser surface's own path): the
		// creation enqueues the album's first scan.
		const created = await page.request.post('/api/v1/albums', {
			data: { title: ALBUM_TITLE, rootPath: SOURCE_DIR, audience: [] }
		});
		expect(created.status()).toBe(201);

		// The first scan converges succeeded (four photos).
		await expect
			.poll(
				async () => {
					const listing = await page.request.get('/api/v1/admin/jobs');
					const body = (await listing.json()) as {
						jobs?: Array<{ kind?: string; state?: string }>;
					};
					return body.jobs?.find((job) => job.kind === 'album_rescan')?.state ?? 'missing';
				},
				{ timeout: 120_000, intervals: [500, 1000, 2000] }
			)
			.toBe('succeeded');

		// The second scan: the source is unchanged, so the run is the
		// clean repeat the panel's rows must speak.
		const wall = await page.request.get('/api/v1/albums');
		expect(wall.status()).toBe(200);
		const albumId = ((await wall.json()) as { albums?: Array<{ id?: string; title?: string }> }).albums?.find(
			(album) => album.title === ALBUM_TITLE
		)?.id;
		expect(albumId).not.toBe('');
		const rescan = await page.request.post(`/api/v1/albums/${encodeURIComponent(albumId!)}/rescan`);
		expect([200, 202]).toContain(rescan.status());
		await expect
			.poll(
				async () => {
					const listing = await page.request.get('/api/v1/admin/jobs');
					const body = (await listing.json()) as {
						jobs?: Array<{ kind?: string; state?: string; subject?: string | null }>;
					};
					const runs = body.jobs?.filter((job) => job.kind === 'album_rescan') ?? [];
					return runs.filter((job) => job.state === 'succeeded').length;
				},
				{ timeout: 120_000, intervals: [500, 1000, 2000] }
			)
			.toBe(2);

		// THE ROW SPEAKS: the newest rescan row names its album (the
		// subject rides the listing read) and reads its own clean
		// count.
		await page.goto('/admin/jobs');
		const row = page.locator('article.jobs__row').first();
		await expect(row).toContainText(`Album rescan · ${ALBUM_TITLE}`);
		await expect(row).toContainText('4 photos unchanged');

		// The finished run's cancel: the API answers 409
		// JOB_NOT_CANCELLABLE -- the row's own state is the honest
		// answer.
		const listing = await page.request.get('/api/v1/admin/jobs');
		const body = (await listing.json()) as {
			jobs?: Array<{ id?: string; kind?: string; state?: string }>;
		};
		const succeededRun = body.jobs?.find((job) => job.kind === 'album_rescan' && job.state === 'succeeded');
		expect(succeededRun?.id).not.toBe('');
		const cancelled = await page.request.post(`/api/v1/admin/jobs/${succeededRun!.id}/cancel`);
		expect(cancelled.status()).toBe(409);
		const envelope = (await cancelled.json()) as { code?: string };
		expect(envelope.code).toBe('JOB_NOT_CANCELLABLE');

		// An unknown id answers 404.
		const unknown = await page.request.post('/api/v1/admin/jobs/0192f8e9-0abc-8f0a-b1c2-3d4e5f6ce401/cancel');
		expect(unknown.status()).toBe(404);
	});

	test('scene 4: the download completes and the four cards read Active', async ({ page }) => {
		test.setTimeout(DOWNLOAD_TIMEOUT_MS + 60_000);
		await page.goto('/admin/models');
		const cta = page.getByRole('button', { name: 'Download the default models', exact: true });
		await expect(cta).toBeVisible();
		await cta.click();

		// The offline mirror serves the group: the progress region
		// takes over, the run completes, the four cards land Active.
		await expect
			.poll(
				async () => {
					let active = 0;
					for (const slot of ['face_detect', 'face_recognize', 'clip_embed', 'ocr']) {
						if (
							(await page.locator(`.models__card[data-slot="${slot}"] [data-state="active"]`).count()) ===
							1
						) {
							active += 1;
						}
					}
					return active;
				},
				{ timeout: DOWNLOAD_TIMEOUT_MS, intervals: [1000, 2000, 5000] }
			)
			.toBe(4);
		await expect(cta).toHaveCount(0);
	});

	test('scene 5: the queued rerun cancels for real -- closed cancelled, nothing ever ran', async ({ page }) => {
		// The ML pause holds a fresh rerun queued (the paused class's
		// new jobs sit held): the run the panel then cancels is a run
		// the queue never started.
		await page.goto('/admin/jobs');
		await page.getByTestId('run-pause-ml').check();
		await expect(page.getByTestId('run-settings')).toContainText('Saved');

		// The face detect rerun (the models page's own trigger): under
		// the pause it sits queued.
		await page.goto('/admin/models');
		await page
			.locator('.models__card[data-slot="face_detect"]')
			.getByRole('button', { name: 'Rerun for the library', exact: true })
			.click();
		await page
			.getByRole('dialog', { name: 'Rerun this model over the library?' })
			.getByRole('button', { name: 'Start the rerun', exact: true })
			.click();
		await page.goto('/admin/jobs');
		const rerunRow = page.locator('article.jobs__row').filter({ hasText: 'Face detection rerun' }).first();
		await expect(rerunRow).toBeVisible({ timeout: 30_000 });
		await expect(rerunRow.locator('.jobs__badge')).toHaveText('Queued');

		// THE CANCEL, for real: the panel's own button. A queued run
		// closes at once -- nothing ever started, the counts stay
		// zero, and the row reads cancelled.
		await rerunRow.getByTestId('job-cancel').click();
		await expect(rerunRow.locator('.jobs__badge--cancelled')).toHaveText('Cancelled');

		// The pause lifts: the run stays cancelled (a cancelled run is
		// never picked back up).
		await page.getByTestId('run-pause-ml').uncheck();
		await expect(page.getByTestId('run-settings')).toContainText('Saved');
		await page.waitForTimeout(4000);
		await expect(rerunRow.locator('.jobs__badge--cancelled')).toHaveText('Cancelled');
	});

	test("scene 6: the running row's cancel -- the honest toast, the convergence, the inline 409", async ({ page }) => {
		// The real stack's steady state cannot hold a running run at
		// click time (the scenes' runs close in seconds), so this
		// scene injects the running row and answers the cancel POST
		// both ways: the 204's toast + the finished event's
		// convergence to the cancelled chip, and the 409's INLINE
		// refusal under its own row.
		const runId = '0192f8e9-0abc-8f0a-b1c2-3d4e5f6ce402';
		const now = Date.now();
		let refuse = false;
		await page.route('**/api/v1/admin/jobs', (route) =>
			route.fulfill({
				json: {
					jobs: [
						{
							id: runId,
							kind: 'clip_embed_rerun',
							state: refuse ? 'cancelled' : 'running',
							subject: null,
							total: 9820,
							done: 4210,
							failed: 0,
							startedAt: new Date(now - 40 * 60 * 1000).toISOString(),
							finishedAt: null
						}
					]
				}
			})
		);
		await page.route('**/api/v1/admin/models', (route) =>
			route.fulfill({
				json: { models: [], download: null, slots: [], official: [] }
			})
		);
		await page.route('**/api/v1/admin/jobs/*/cancel', async (route) => {
			if (refuse) {
				await route.fulfill({
					status: 409,
					json: { code: 'JOB_NOT_CANCELLABLE', message: 'this run has already finished' }
				});
				return;
			}
			refuse = true;
			await route.fulfill({ status: 204, body: '' });
		});

		await page.goto('/admin/jobs');
		const row = page.locator('article.jobs__row').first();
		await expect(row).toContainText('4210 of 9820 assets');

		// The cancel: the 204's toast speaks the honest account read
		// off the row; the listing's refetch lands the cancelled chip.
		await row.getByTestId('job-cancel').click();
		await expect(page.locator('.toast')).toContainText('Cancelled. The 4210 finished items are kept.');
		await expect(row.locator('.jobs__badge--cancelled')).toHaveText('Cancelled');

		// The 409's face: a stale running row's cancel lands the
		// refusal INLINE under the row it belongs to -- never a toast
		// over the listing.
		await page.unroute('**/api/v1/admin/jobs');
		await page.unroute('**/api/v1/admin/jobs/*/cancel');
		await page.route('**/api/v1/admin/jobs', (route) =>
			route.fulfill({
				json: {
					jobs: [
						{
							id: runId,
							kind: 'clip_embed_rerun',
							state: 'running',
							subject: null,
							total: 9820,
							done: 4210,
							failed: 0,
							startedAt: new Date(now - 40 * 60 * 1000).toISOString(),
							finishedAt: null
						}
					]
				}
			})
		);
		await page.route('**/api/v1/admin/jobs/*/cancel', (route) =>
			route.fulfill({
				status: 409,
				json: { code: 'JOB_NOT_CANCELLABLE', message: 'this run has already finished' }
			})
		);
		await page.goto('/admin/jobs');
		await page.getByTestId('job-cancel').first().click();
		await expect(page.getByTestId('job-cancel-error')).toContainText(
			'This run has already finished and cannot be cancelled.'
		);
		await expect(page.locator('.toast')).toHaveCount(0);
	});

	test('scene 7: the dispatch reads the fresh params -- the raised minimum keeps the faces out', async ({ page }) => {
		test.setTimeout(600_000);
		// THE OBSERVABLE: the UNNAMED region of the people surface
		// (`/people/unnamed` — the cluster carries no label row here,
		// so the named region's `/people` answers an empty array by
		// construction). The library's images are the trip fixtures'
		// -- two of which ARE the people story's portrait pair (the
		// same person twice: the recognition's clustering needs two
		// close faces for one cluster, minPts 2). The models landed in
		// scene 4, AFTER the rescan's ingest, so the library's first
		// dispatch is a RERUN (the model rerun surface owns the redo
		// -- the model download never re-dispatches the library).
		const peopleOf = async (): Promise<number> => {
			const people = await page.request.get('/api/v1/people/unnamed');
			expect(people.status()).toBe(200);
			const body = (await people.json()) as { totalCount?: number };
			return body.totalCount ?? 0;
		};
		const detectModel = await (async () => {
			const models = await page.request.get('/api/v1/admin/models');
			const body = (await models.json()) as {
				models: Array<{ id: string; slot: string; isActive: boolean }>;
			};
			return body.models.find((model) => model.slot === 'face_detect' && model.isActive)!.id;
		})();
		const rerunCount = async (): Promise<number> => {
			const listing = await page.request.get('/api/v1/admin/jobs');
			const body = (await listing.json()) as {
				jobs?: Array<{ kind?: string; state?: string }>;
			};
			const runs = body.jobs?.filter((job) => job.kind === 'face_detect_rerun') ?? [];
			return runs.filter((job) => job.state === 'succeeded').length;
		};
		const rerunAndWait = async (): Promise<void> => {
			const before = await rerunCount();
			const rerun = await page.request.post(`/api/v1/admin/models/${detectModel}/rerun`);
			expect([200, 202]).toContain(rerun.status());
			await expect
				.poll(async () => rerunCount(), { timeout: 180_000, intervals: [1000, 2000, 5000] })
				.toBe(before + 1);
		};

		// The baseline rerun at the shipped 0.6: the faces land, the
		// recognize chain clusters the pair, the listing carries it.
		await rerunAndWait();
		await expect
			.poll(async () => peopleOf(), { timeout: 300_000, intervals: [2000, 5000] })
			.toBeGreaterThanOrEqual(1);
		const before = await peopleOf();

		// Raise the minimum past any real detection: the NEXT
		// dispatched job reads 0.99 at its own start and keeps every
		// face out of the library.
		const raised = await page.request.put('/api/v1/admin/settings/models', {
			data: {
				autoDownload: true,
				allowedHosts: ['huggingface.co'],
				mirrorBaseUrl: '',
				params: { faceDetectMinScore: 0.99, faceRecognizeThreshold: 0.55 }
			}
		});
		expect(raised.status()).toBe(200);
		await rerunAndWait();
		await expect.poll(async () => peopleOf(), { timeout: 120_000 }).toBe(0);

		// Restored, the next dispatch brings the clusters back: the
		// same images, the same detector, the value the job read
		// deciding.
		const restored = await page.request.put('/api/v1/admin/settings/models', {
			data: {
				autoDownload: true,
				allowedHosts: ['huggingface.co'],
				mirrorBaseUrl: '',
				params: { faceDetectMinScore: 0.6, faceRecognizeThreshold: 0.55 }
			}
		});
		expect(restored.status()).toBe(200);
		await rerunAndWait();
		await expect.poll(async () => peopleOf(), { timeout: 300_000 }).toBe(before);
	});
});
