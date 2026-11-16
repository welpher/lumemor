// The E39 people-editing story, walked end to end against the real
// compose stack: the three-tier remove vocabulary's REAL delete (the
// person page's danger zone) and the faces panel's edit-mode box
// layer (the boxes stand up on the photo, a row's Adjust box selects
// one, a drag moves it, a corner drag resizes it, the panel's
// one-shot commit PATCHes the stored box and the face's crop
// re-derives from it).
//
// THE STACK ARRANGEMENT: one fresh lifetime owns this spec (the
// first-run bootstrap creates the administrator; the models ride the
// storage root's config tree, warm or cold like every other people
// story). The seeds are the repository's own portrait fixtures: the
// same-person pair embeds together (the recognition pipeline's
// minPts 2 needs two co-embedded faces to form a cluster), and the
// genuinely-different pair forms the second person.
//
// THE TWO ACCOUNTS (the admin domain's own walkthrough): the admin
// uploads and names the first person (Ada); the MEMBER uploads and
// names the second (Grace -- naming is the member's own action).
// The member's scenes prove the domain: Grace's page renders for the
// member with NO delete affordance, and the member's DELETE answers
// the 403 whose code the body carries. The admin's scenes prove the
// rest: the box-edit chain on Ada's face, the refusal matrix of the
// PATCH, and the real delete of Ada -- the list loses the person,
// the photo loses the person's chips OUTRIGHT (the faces are
// deleted, never unassigned: the faces array empties), the unnamed
// pool gains nothing, and the photos themselves still open.
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story)
// and each signs in or replays a saved storage state; every scene
// navigates explicitly to its starting route.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's two accounts: the administrator (the real delete's
// domain) and a plain member (the domain's boundary).
const ADMIN_EMAIL = 'admin@e39-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';
const MEMBER_EMAIL = 'member@e39-e2e.invalid';
const MEMBER_PASSWORD = 'correct-horse-battery';

// The four portrait fixtures: the A pair is the same person twice
// (the embedding pair), the B pair a genuinely different person.
const PORTRAIT_A1 = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait.jpg');
const PORTRAIT_A2 = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-orientation6.jpg');
const PORTRAIT_B1 = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other.jpg');
const PORTRAIT_B2 = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other-second.jpg');

const PHOTO_A1_NAME = 'IMG_E39_0001.jpg';
const PHOTO_A2_NAME = 'IMG_E39_0002.jpg';
const PHOTO_B1_NAME = 'IMG_E39_0003.jpg';
const PHOTO_B2_NAME = 'IMG_E39_0004.jpg';

const PERSON_A = 'Ada';
const PERSON_B = 'Grace';

// The download and clustering bounds, the people stories' own.
const DOWNLOAD_TIMEOUT_MS = 900_000;
const CLUSTER_TIMEOUT_MS = 420_000;

// The storage-state files the sign-in scenes write (git-ignored).
const adminStatePath = `${specDir}/.auth/e39-admin-state.json`;
const memberStatePath = `${specDir}/.auth/e39-member-state.json`;

// The story's captured ids (the bootstrap scenes fix them).
let adaClusterId = '';
let adaFaceId = '';
let adaAssetId = '';
let adaBboxBefore: { x: number; y: number; w: number; h: number } | null = null;
let adaCropETag = '';
let graceClusterId = '';

type FaceBbox = { x: number; y: number; w: number; h: number };

/** Upload one portrait through the asset API (the people stories' seeding pattern). */
async function uploadPortrait(page: Page, file: string, name: string, takenAt: string): Promise<void> {
	const upload = await page.request.post('/api/v1/assets', {
		multipart: {
			deviceAssetId: `e39-e2e-${name}`,
			fileName: name,
			deviceCreatedAt: takenAt,
			deviceTimeZone: 'UTC',
			photo: {
				name,
				mimeType: 'image/jpeg',
				buffer: await readFile(file)
			}
		}
	});
	expect(upload.status()).toBe(201);
}

/** The caller's unnamed region as one sorted list of face counts. */
async function unnamedFaceCounts(page: Page): Promise<number[]> {
	const listing = await page.request.get('/api/v1/people/unnamed');
	expect(listing.status()).toBe(200);
	const body = (await listing.json()) as { items?: Array<{ faceCount?: number }> };
	return (body.items ?? []).map((item) => item.faceCount ?? 0).sort((a, b) => b - a);
}

/** The caller's whole unnamed region in one number (the pool's own count). */
async function unnamedTotal(page: Page): Promise<number> {
	const listing = await page.request.get('/api/v1/people/unnamed');
	expect(listing.status()).toBe(200);
	const body = (await listing.json()) as { totalCount?: number };
	return body.totalCount ?? -1;
}

/** One asset detail's faces (the chip truth the viewer renders). */
async function assetFaces(page: Page, assetId: string): Promise<Array<{ id: string; bbox: FaceBbox }>> {
	const detail = await page.request.get(`/api/v1/assets/${encodeURIComponent(assetId)}?locale=en`);
	expect(detail.status()).toBe(200);
	const body = (await detail.json()) as { faces?: Array<{ id: string; bbox: FaceBbox }> };
	return body.faces ?? [];
}

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap: the setup API creates the administrator
	// on the fresh instance.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'E39 E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

test('scene 1: the administrator signs in through the form', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Ftimeline');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);
	await page.context().storageState({ path: adminStatePath });
});

test.describe('the admin scenes', () => {
	test.use({ storageState: adminStatePath });

	test('scene 2: the model group warms through the admin API', async ({ page }) => {
		test.setTimeout(DOWNLOAD_TIMEOUT_MS + 60_000);

		const download = await page.request.post('/api/v1/admin/models/download');
		expect(download.status()).toBe(202);
		await expect
			.poll(
				async () => {
					const jobs = await page.request.get('/api/v1/admin/jobs');
					expect(jobs.status()).toBe(200);
					const body = (await jobs.json()) as { jobs?: Array<{ kind?: string; state?: string }> };
					return body.jobs?.find((job) => job.kind === 'model_download')?.state ?? 'missing';
				},
				{ timeout: DOWNLOAD_TIMEOUT_MS, intervals: [1000, 2000, 5000] }
			)
			.toBe('succeeded');
	});

	test('scene 3: the admin seeds the first person and names her', async ({ page }) => {
		test.setTimeout(CLUSTER_TIMEOUT_MS + 120_000);

		// The member account exists before the member signs in (the
		// admin surface's own create).
		const member = await page.request.post('/api/v1/admin/users', {
			data: { email: MEMBER_EMAIL, password: MEMBER_PASSWORD }
		});
		expect(member.status()).toBe(201);

		// The A pair uploads (the admin's own copies).
		await uploadPortrait(page, PORTRAIT_A1, PHOTO_A1_NAME, '2026-01-01T12:00:00Z');
		await uploadPortrait(page, PORTRAIT_A2, PHOTO_A2_NAME, '2026-02-02T12:00:00Z');

		// The recognition lands the pair as ONE unnamed cluster of two.
		await expect
			.poll(async () => unnamedFaceCounts(page), {
				timeout: CLUSTER_TIMEOUT_MS,
				intervals: [1000, 2000, 5000]
			})
			.toEqual([2]);

		// The cluster's detail fixes the story's ids: the cluster, one
		// of its faces (the box-edit's object), that face's asset and
		// the face's stored box (the drag's starting point of truth).
		const listing = await page.request.get('/api/v1/people/unnamed');
		const clusters = (
			(await listing.json()) as {
				items: Array<{ clusterId: string; faceCount: number }>;
			}
		).items;
		expect(clusters).toHaveLength(1);
		adaClusterId = clusters[0]!.clusterId;
		const detail = await page.request.get(`/api/v1/people/${adaClusterId}`);
		expect(detail.status()).toBe(200);
		const body = (await detail.json()) as {
			faces: Array<{ id: string; assetId: string; bbox: FaceBbox; asset: { originalFilename: string } }>;
		};
		expect(body.faces).toHaveLength(2);
		expect(body.faces.every((face) => [PHOTO_A1_NAME, PHOTO_A2_NAME].includes(face.asset.originalFilename))).toBe(
			true
		);
		const face = body.faces[0]!;
		adaFaceId = face.id;
		adaAssetId = face.assetId;
		adaBboxBefore = face.bbox;

		// The crop's ETag before any edit (the re-crop's change signal).
		const crop = await page.request.get(`/api/v1/faces/${adaFaceId}/thumbnail`);
		expect(crop.status()).toBe(200);
		adaCropETag = crop.headers()['etag'] ?? '';

		// The naming is the admin's own write.
		const labeled = await page.request.put(`/api/v1/people/${adaClusterId}/label`, {
			data: { name: PERSON_A }
		});
		expect(labeled.status()).toBe(200);
	});

	test('scene 4: the faces panel stands the boxes up; a row selects, drags and resizes one', async ({ page }) => {
		// The photo that carries the story's face (the pair's other
		// photo holds the cluster's other row).
		await page.goto(`/a/${adaAssetId}`);
		await expect(page).toHaveURL(new RegExp(`/a/[0-9a-f-]{36}$`));

		// The panel's two doors, then the boxes stand visible with
		// their plain borders (the edit mode's own state).
		await page.getByRole('button', { name: 'Photo information' }).click();
		await page.getByRole('button', { name: 'Edit people', exact: true }).click();
		const panel = page.getByRole('complementary', { name: 'Edit people' });
		await expect(panel).toBeVisible();
		await expect(panel).toContainText(
			"Pick Adjust box to select a face's box on the photo — drag to move, drag a corner to resize."
		);
		const boxes = page.locator('[data-face-box]');
		// The photo carries its OWN faces (one row each): the cluster's
		// other face sits on the pair's other photo.
		await expect(boxes).toHaveCount(1);
		await expect(boxes.first()).toHaveCSS('opacity', '1');

		// The row's Adjust box selects that face's box: the accent
		// state and the four corner handles arrive.
		const row = panel.locator(`li[data-face-id="${adaFaceId}"]`);
		await row.getByRole('button', { name: 'Adjust box', exact: true }).click();
		const selected = page.locator('[data-face-box][data-box-selected="true"]');
		await expect(selected).toHaveCount(1);
		await expect(selected.locator('[data-handle="nw"]')).toBeVisible();
		await expect(selected.locator('[data-handle="ne"]')).toBeVisible();
		await expect(selected.locator('[data-handle="sw"]')).toBeVisible();
		await expect(selected.locator('[data-handle="se"]')).toBeVisible();

		// THE MOVE: a pointer drag carries the box (the media box's
		// real rendered rect is the scale; the touch path is the same
		// pointer path).
		const before = await selected.boundingBox();
		expect(before).not.toBeNull();
		const cx = before!.x + before!.width / 2;
		const cy = before!.y + before!.height / 2;
		await page.mouse.move(cx, cy);
		await page.mouse.down();
		await page.mouse.move(cx + 40, cy + 30, { steps: 8 });
		await page.mouse.up();
		const afterMove = await selected.boundingBox();
		expect(Math.round(afterMove!.x - before!.x)).toBe(40);
		expect(Math.round(afterMove!.y - before!.y)).toBe(30);

		// The drag's landing staged (the pending marker on the row,
		// nothing written yet).
		await expect(row).toHaveAttribute('data-staged', 'true');

		// THE RESIZE: the south-east handle drags the corner outward;
		// the box grows by the drag's length.
		const handle = selected.locator('[data-handle="se"]');
		const handleBox = await handle.boundingBox();
		expect(handleBox).not.toBeNull();
		const hx = handleBox!.x + handleBox!.width / 2;
		const hy = handleBox!.y + handleBox!.height / 2;
		await page.mouse.move(hx, hy);
		await page.mouse.down();
		await page.mouse.move(hx + 50, hy + 50, { steps: 8 });
		await page.mouse.up();
		const afterResize = await selected.boundingBox();
		// The corner drag grows the box by the drag's length (the
		// rendered percentage carries sub-pixel rounding, and a box
		// near the frame's edge grows only to the frame).
		const growth = afterResize!.width - afterMove!.width;
		expect(growth).toBeGreaterThan(40);
		expect(growth).toBeLessThanOrEqual(50);

		// Esc clears the selection; the panel stays open (the row's
		// button re-arms).
		await page.keyboard.press('Escape');
		await expect(page.locator('[data-box-selected="true"]')).toHaveCount(0);
		await expect(panel).toBeVisible();

		// Re-select for the commit.
		await row.getByRole('button', { name: 'Adjust box', exact: true }).click();
		await expect(page.locator('[data-box-selected="true"]')).toHaveCount(1);
	});

	test('scene 5: the one-shot commit PATCHes the box and the crop re-derives', async ({ page }) => {
		await page.goto(`/a/${adaAssetId}`);
		await expect(page).toHaveURL(new RegExp(`/a/[0-9a-f-]{36}$`));
		await page.getByRole('button', { name: 'Photo information' }).click();
		await page.getByRole('button', { name: 'Edit people', exact: true }).click();
		const panel = page.getByRole('complementary', { name: 'Edit people' });
		await expect(panel).toBeVisible();
		const row = panel.locator(`li[data-face-id="${adaFaceId}"]`);
		await row.getByRole('button', { name: 'Adjust box', exact: true }).click();
		const selected = page.locator('[data-face-box][data-box-selected="true"]');
		await expect(selected).toHaveCount(1);

		// One drag is enough to stage (the story repeats scene 4's
		// move; the commit is the scene's subject).
		const before = await selected.boundingBox();
		const cx = before!.x + before!.width / 2;
		const cy = before!.y + before!.height / 2;
		await page.mouse.move(cx, cy);
		await page.mouse.down();
		await page.mouse.move(cx + 30, cy + 20, { steps: 6 });
		await page.mouse.up();
		await expect(row).toHaveAttribute('data-staged', 'true');

		// The panel's check writes once and echoes once (the staged
		// set was boxes alone).
		await panel.getByRole('button', { name: 'Finish editing', exact: true }).click();
		await expect(page.locator('.toasts')).toContainText('Face box updated.');
		await expect(panel).toHaveCount(0);

		// The PATCH landed: the stored box moved off the scene's
		// starting point (the API is the truth the boxes render).
		const faces = await assetFaces(page, adaAssetId);
		const edited = faces.find((face) => face.id === adaFaceId);
		expect(edited).toBeTruthy();
		expect(edited!.bbox).not.toEqual(adaBboxBefore);

		// The crop re-derived from the new box: the ETag moved (the
		// face row's updated_at is the crop's change signal).
		const crop = await page.request.get(`/api/v1/faces/${adaFaceId}/thumbnail`);
		expect(crop.status()).toBe(200);
		expect(crop.headers()['etag'] ?? '').not.toBe(adaCropETag);

		// The on-photo box converged (the SSE refetch re-rendered it
		// from the refetched faces array): the box layer's left edge
		// now carries the stored box's percentage.
		const box = page.locator(`[data-face-box="${adaFaceId}"]`);
		await expect(box).toHaveAttribute('style', new RegExp(`left: ${edited!.bbox.x / 10}%`));

		// The panel's rows re-request the re-derived crop (the bust
		// rides the commit).
		await page.getByRole('button', { name: 'Edit people', exact: true }).click();
		const reopened = page.getByRole('complementary', { name: 'Edit people' });
		await expect(reopened).toBeVisible();
		await expect(reopened.locator(`li[data-face-id="${adaFaceId}"] img`)).toHaveAttribute(
			'src',
			new RegExp(`\\?v=1$`)
		);
		await page.keyboard.press('Escape');
		await expect(reopened).toHaveCount(0);
	});

	test('scene 6: the PATCH refuses out-of-frame boxes and foreign faces', async ({ page }) => {
		// The stored box before the refusals (the no-side-effect
		// comparison's baseline).
		const facesBefore = await assetFaces(page, adaAssetId);
		const stored = facesBefore.find((face) => face.id === adaFaceId)!.bbox;

		// An out-of-frame box: the contract's 400 envelope.
		const outOfFrame = await page.request.patch(`/api/v1/faces/${adaFaceId}`, {
			data: { x: 0.99, y: 0.5, width: 0.05, height: 0.1 }
		});
		expect(outOfFrame.status()).toBe(400);
		expect(((await outOfFrame.json()) as { code?: string }).code).toBe('VALIDATION_FAILED');

		// A sub-floor extent: the same 400.
		const tooSmall = await page.request.patch(`/api/v1/faces/${adaFaceId}`, {
			data: { x: 0.1, y: 0.1, width: 0.005, height: 0.1 }
		});
		expect(tooSmall.status()).toBe(400);

		// An unknown face: the anti-enumeration 404.
		const unknown = await page.request.patch('/api/v1/faces/0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6fff', {
			data: { x: 0.1, y: 0.1, width: 0.1, height: 0.1 }
		});
		expect(unknown.status()).toBe(404);
		expect(((await unknown.json()) as { code?: string }).code).toBe('PERSON_NOT_FOUND');

		// The refused writes changed nothing.
		const facesAfter = await assetFaces(page, adaAssetId);
		expect(facesAfter.find((face) => face.id === adaFaceId)?.bbox).toEqual(stored);
	});
});

test('scene 7: the member signs in through the form', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Fpeople');
	await page.getByLabel('Email').fill(MEMBER_EMAIL);
	await page.getByLabel('Password').fill(MEMBER_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/people$/);
	await page.context().storageState({ path: memberStatePath });
});

test.describe('the member scenes', () => {
	test.use({ storageState: memberStatePath });

	test('scene 8: the member seeds and names the second person', async ({ page }) => {
		test.setTimeout(CLUSTER_TIMEOUT_MS + 120_000);

		// The B pair uploads (the member's own copies -- the cluster
		// is global, the member sees it through their own assets).
		await uploadPortrait(page, PORTRAIT_B1, PHOTO_B1_NAME, '2026-03-03T12:00:00Z');
		await uploadPortrait(page, PORTRAIT_B2, PHOTO_B2_NAME, '2026-04-04T12:00:00Z');

		await expect
			.poll(async () => unnamedFaceCounts(page), {
				timeout: CLUSTER_TIMEOUT_MS,
				intervals: [1000, 2000, 5000]
			})
			.toEqual([2]);

		const listing = await page.request.get('/api/v1/people/unnamed');
		const clusters = ((await listing.json()) as { items: Array<{ clusterId: string }> }).items;
		expect(clusters).toHaveLength(1);
		graceClusterId = clusters[0]!.clusterId;
		const detail = await page.request.get(`/api/v1/people/${graceClusterId}`);
		const body = (await detail.json()) as {
			faces: Array<{ asset: { originalFilename: string } }>;
		};
		expect(body.faces.map((face) => face.asset.originalFilename).sort()).toEqual(
			[PHOTO_B1_NAME, PHOTO_B2_NAME].sort()
		);

		// The naming is the member's own write (the softer tier).
		const labeled = await page.request.put(`/api/v1/people/${graceClusterId}/label`, {
			data: { name: PERSON_B }
		});
		expect(labeled.status()).toBe(200);
	});

	test("scene 9: the member's person page carries no delete affordance", async ({ page }) => {
		await page.goto('/people');
		await page.locator('a.person-card').filter({ hasText: PERSON_B }).click();
		await expect(page).toHaveURL(new RegExp(`/people/${graceClusterId}$`));
		await expect(page.locator('.person-profile__name')).toHaveText(PERSON_B);

		// The danger zone is the admin domain's: no Delete person
		// button anywhere (the member's removal action is the unname).
		await expect(page.getByRole('button', { name: 'Delete person', exact: true })).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Remove name', exact: true })).toBeVisible();

		// The guard behind the affordance: the member's DELETE
		// answers the 403 whose code the body carries.
		const forbidden = await page.request.delete(`/api/v1/people/${graceClusterId}`);
		expect(forbidden.status()).toBe(403);
		expect(((await forbidden.json()) as { code?: string }).code).toBe('AUTH_FORBIDDEN');
	});
});

test.describe('the admin delete scenes', () => {
	test.use({ storageState: adminStatePath });

	test('scene 10: the real delete removes the person, the chips and nothing else', async ({ page }) => {
		// The pool's count before: the admin's own unnamed region is
		// empty (Ada is named; the member's person is invisible here).
		expect(await unnamedTotal(page)).toBe(0);

		await page.goto('/people');
		await page.locator('a.person-card').filter({ hasText: PERSON_A }).click();
		await expect(page).toHaveURL(new RegExp(`/people/${adaClusterId}$`));

		// The danger zone's button opens the confirm: the name and
		// the caller's face count ride the sentence.
		await page.getByRole('button', { name: 'Delete person', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Delete this person?' });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText('This deletes Ada and the detection records of all 2 of their faces.');
		await expect(dialog).toContainText('The photos themselves are not affected. This cannot be undone.');

		// The confirm runs the write: the echo lands, the people
		// list takes over (the landing is the feedback).
		await dialog.getByRole('button', { name: 'Delete person', exact: true }).click();
		await expect(page.locator('.toasts')).toContainText('Person deleted.');
		await expect(page).toHaveURL(/\/people$/);
		await expect(page.locator('a.person-card')).toHaveCount(0);

		// The cluster is gone outright; the deleted id answers the
		// same dead-end 404 as an unknown one.
		const detail = await page.request.get(`/api/v1/people/${adaClusterId}`);
		expect(detail.status()).toBe(404);

		// THE CASCADE'S PROOF: the photo's faces array is EMPTY --
		// the face rows are deleted, never unassigned (nothing
		// re-rendered among the not grouped yet).
		expect(await assetFaces(page, adaAssetId)).toEqual([]);

		// The unnamed pool gained NOTHING (the pool's own count).
		expect(await unnamedTotal(page)).toBe(0);

		// The photos themselves still open: the viewer's stage loads
		// the photo and its People section reads the honest empty.
		await page.goto(`/a/${adaAssetId}`);
		await expect(page.getByRole('button', { name: 'Photo information' })).toBeVisible();
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = page.getByRole('complementary', { name: 'Photo information' });
		await expect(drawer).toBeVisible();
		await expect(drawer.locator('.info__faces-empty')).toHaveText('No people recognized in this photo.');
		await expect(page.locator('[data-face-box]')).toHaveCount(0);
	});
});
