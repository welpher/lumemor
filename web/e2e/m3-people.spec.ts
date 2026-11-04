// The M3b people story of the web app, walked end to end against the
// real compose stack. scripts/acceptance_m3b.sh chains the M3a
// acceptance (which itself chains the whole M1/M2 line) and then
// starts this spec's own fresh stack; the spec itself only talks to
// http://localhost:8244.
//
// THE STACK ARRANGEMENT (the honest reuse path, documented): the
// stack is fresh (its database volume was removed before it started)
// but the models ride the SAME storage bind mount the chained M3a
// phases downloaded into -- ./lumemor persists across the script's
// stack lifetimes, and the download service's already-verified skip
// re-registers the weights without re-fetching them. The bootstrap
// scene therefore triggers the default group's download through the
// admin API (the people story has no download surface of its own):
// warm, it is a digest re-verification; cold (CI's first run), it is
// the real transfer, bounded like the M3 model story's cold bound.
//
// THE SEEDING (four photos, two people): the recognition pipeline
// clusters faces by density (minPts 2), so a person's cluster needs
// TWO photos of that person -- a lone face is noise with no cluster
// id. The story therefore seeds a pair of the SAME person (the
// portrait and its EXIF-orientation twin, the pair the M3a acceptance
// proved embeds at cosine distance 0.007) AND a pair of a genuinely
// DIFFERENT person (two public-domain NASA portraits of the STS-107
// commander). The pipeline answers two clusters of two faces each --
// the same-person co-clustering and the cross-person separation the
// acceptance script's database asserts prove at the embedding level.
//
// THE SCENES (the M3b people surface, in order):
//
//  1. Signing in through the form lands on the deep-linked people
//     page, whose empty state carries the honest copy and the
//     unnamed region's entry.
//  2. The library bootstrap, all through the API (the M2 story's
//     seeding pattern): the model group's download run is driven to
//     succeeded, the four fixtures upload, and the unnamed listing
//     converges to exactly two clusters of two faces -- identified
//     through the cluster details so the story knows which unnamed
//     card is which person.
//  3. The unnamed region renders both clusters as cards with their
//     face counts.
//  4. The naming main path: clicking the same-person cluster's card
//     opens the naming dialog; the PUT lands and the card leaves the
//     region immediately; the named region gains the person.
//  5. The detail navigation: the person card opens the cluster
//     detail with the faces grid and the asset context.
//  6. The merge flow: the other cluster is named through the same
//     dialog, its detail's "Merge into…" folds it into the first
//     person, and the named region shows the survivor with the
//     merged face count while the merged-away person is gone.
//  7. The viewer's face chips (the M3b Task 5 surface): a face card
//     of the detail deep-links into the asset viewer, whose info
//     drawer carries the person's name on the face's chip.
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story --
// see playwright.config.ts) and share the browser session: scene 1
// signs in through the form and saves the storage state, every later
// scene replays it. Each scene navigates explicitly to its starting
// route, so a failure report names the exact scene that broke.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's administrator: created through the first-run setup API
// in the suite's bootstrap (the acceptance scripts' pattern), then
// used by the login scene. The stack is always fresh, so the fixed
// identity is deterministic.
const ADMIN_EMAIL = 'admin@m3b-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The four fixtures of the people story: the portrait and its
// EXIF-orientation twin are the SAME person, and the two "other"
// portraits are a second, genuinely different person (see the header
// comment's seeding note).
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));
const PORTRAIT_A1 = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait.jpg');
const PORTRAIT_A2 = path.join(
	repoRoot,
	'server/internal/ml/testdata/face-portrait-orientation6.jpg'
);
const PORTRAIT_B1 = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other.jpg');
const PORTRAIT_B2 = path.join(
	repoRoot,
	'server/internal/ml/testdata/face-portrait-other-second.jpg'
);

// The uploaded file names (the detail's asset context renders them).
const PHOTO_A1_NAME = 'IMG_M3B_0001.jpg';
const PHOTO_A2_NAME = 'IMG_M3B_0002.jpg';
const PHOTO_B1_NAME = 'IMG_M3B_0003.jpg';
const PHOTO_B2_NAME = 'IMG_M3B_0004.jpg';

// The two story names.
const PERSON_A = 'Ada';
const PERSON_B = 'Grace';

// The download and clustering bounds: the cold upstream transfer of
// the group (about 351 MB) is minutes, the warm re-verification is
// seconds, and the recognition of four faces rides the pipeline's
// own pace; both allow the cold bound.
const DOWNLOAD_TIMEOUT_MS = 900_000;
const CLUSTER_TIMEOUT_MS = 420_000;

// The storage-state file scene 1 writes and every later scene replays
// (git-ignored, recreated per run).
const authStatePath = `${specDir}/.auth/m3b-state.json`;

// The same-person cluster's identity, captured by the bootstrap scene
// through the API: the cluster's id and its representative face's id
// (the unnamed card renders the representative's crop, so the card is
// addressable by its thumbnail URL).
let samePersonClusterId = '';
let samePersonRepFaceId = '';

// The unnamed region's cluster cards (the page renders one button per
// cluster whose accessible name is the honest face count; the count
// line is always plural -- the i18n message has no singular form).
// Addressed by the representative face's crop URL when the story
// needs one specific cluster's card.
function clusterCard(page: Page, count: number) {
	return page.getByRole('button', { name: `${count} faces`, exact: true });
}

function clusterCardByFace(page: Page, faceId: string) {
	return page
		.locator('button.cluster-card')
		.filter({ has: page.locator(`img[src="/api/v1/faces/${faceId}/thumbnail"]`) });
}

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance. The
	// browser scenes then only exercise the web app's own surfaces
	// (the login scene performs the actual sign-in through the form).
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'M3b E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

test('scene 1: signing in through the form lands on the empty people page', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Fpeople');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();

	// The post-login target round-trips onto the people page.
	await expect(page).toHaveURL(/\/people$/);
	await expect(page).toHaveTitle('People · lumemor');

	// The empty state carries the honest copy and the unnamed
	// region's entry (the naming main path lives there).
	await expect(
		page.getByRole('heading', { level: 1, name: 'People', exact: true })
	).toBeVisible();
	await expect(page.getByText('No people yet. Photos with faces will show up here.')).toBeVisible();
	await expect(page.getByRole('link', { name: 'Unnamed', exact: true })).toBeVisible();

	// Hand the session to every later scene: the storage state rides
	// the login cookie (the fresh contexts replay it).
	await page.context().storageState({ path: authStatePath });
});

// The signed-in half of the story: every scene starts from a fresh
// context that replays the login cookie scene 1 saved. The serial
// mode guarantees the state file exists by the time these scenes run.
test.describe('the signed-in scenes', () => {
	test.use({ storageState: authStatePath });

	test('scene 2: the library bootstrap seeds two clusters through the API', async ({ page }) => {
		test.setTimeout(DOWNLOAD_TIMEOUT_MS + CLUSTER_TIMEOUT_MS + 60_000);

		// The model group's download first (the people story has no
		// download surface of its own; the admin API is the M2 story's
		// seeding pattern -- the request rides the page's own context
		// and its login cookie). Warm mount or cold, the run must
		// reach succeeded.
		const download = await page.request.post('/api/v1/admin/models/download');
		expect(download.status()).toBe(202);
		await expect
			.poll(
				async () => {
					const jobs = await page.request.get('/api/v1/admin/jobs');
					expect(jobs.status()).toBe(200);
					const body = (await jobs.json()) as {
						jobs?: Array<{ kind?: string; state?: string }>;
					};
					const row = body.jobs?.find((job) => job.kind === 'model_download');
					return row?.state ?? 'missing';
				},
				{ timeout: DOWNLOAD_TIMEOUT_MS, intervals: [1000, 2000, 5000] }
			)
			.toBe('succeeded');

		// The four fixtures upload through the asset API (the same
		// seeding the acceptance script applies).
		const seeds: Array<{ file: string; name: string; takenAt: string }> = [
			{ file: PORTRAIT_A1, name: PHOTO_A1_NAME, takenAt: '2026-01-01T12:00:00Z' },
			{ file: PORTRAIT_A2, name: PHOTO_A2_NAME, takenAt: '2026-02-02T12:00:00Z' },
			{ file: PORTRAIT_B1, name: PHOTO_B1_NAME, takenAt: '2026-03-03T12:00:00Z' },
			{ file: PORTRAIT_B2, name: PHOTO_B2_NAME, takenAt: '2026-04-04T12:00:00Z' }
		];
		for (const seed of seeds) {
			const upload = await page.request.post('/api/v1/assets', {
				multipart: {
					deviceAssetId: `m3b-e2e-${seed.name}`,
					fileName: seed.name,
					deviceCreatedAt: seed.takenAt,
					deviceTimeZone: 'UTC',
					photo: {
						name: seed.name,
						mimeType: 'image/jpeg',
						buffer: await readFile(seed.file)
					}
				}
			});
			expect(upload.status()).toBe(201);
		}

		// The convergence: the recognition runs land the faces and the
		// clusters, so the unnamed listing answers exactly two
		// clusters whose honest counts are one pair per person.
		type UnnamedItem = { clusterId: string; faceCount: number; representativeFaceId: string };
		await expect
			.poll(
				async () => {
					const listing = await page.request.get('/api/v1/people/unnamed');
					expect(listing.status()).toBe(200);
					const body = (await listing.json()) as { items?: UnnamedItem[] };
					return (body.items ?? []).map((item) => item.faceCount).sort((a, b) => b - a);
				},
				{ timeout: CLUSTER_TIMEOUT_MS, intervals: [1000, 2000, 5000] }
			)
			.toEqual([2, 2]);

		// Identify which cluster is the same-person pair (the story
		// must name the right card): each cluster's detail carries its
		// faces' asset file names, so the pair sets tell the two
		// people apart -- and the disjoint pairs are the API-level
		// cross-person separation evidence.
		const listing = await page.request.get('/api/v1/people/unnamed');
		expect(listing.status()).toBe(200);
		const clusters = ((await listing.json()) as { items: UnnamedItem[] }).items;
		expect(clusters).toHaveLength(2);
		const faceSets: string[][] = [];
		for (const cluster of clusters) {
			const detail = await page.request.get(`/api/v1/people/${cluster.clusterId}`);
			expect(detail.status()).toBe(200);
			const body = (await detail.json()) as {
				label: unknown;
				faces: Array<{ asset: { originalFilename: string } }>;
			};
			// The unnamed state's detail carries no label.
			expect(body.label).toBeNull();
			expect(body.faces).toHaveLength(2);
			faceSets.push(body.faces.map((face) => face.asset.originalFilename));
		}
		const pairA = [PHOTO_A1_NAME, PHOTO_A2_NAME];
		const pairB = [PHOTO_B1_NAME, PHOTO_B2_NAME];
		expect(
			faceSets.some(
				(set) => set.every((name) => pairA.includes(name)) && set.length === pairA.length
			)
		).toBe(true);
		expect(
			faceSets.some(
				(set) => set.every((name) => pairB.includes(name)) && set.length === pairB.length
			)
		).toBe(true);
		const samePerson = clusters.find((cluster) => {
			const set = faceSets[clusters.indexOf(cluster)];
			return set.every((name) => pairA.includes(name));
		});
		samePersonClusterId = samePerson.clusterId;
		samePersonRepFaceId = samePerson.representativeFaceId;
	});

	test('scene 3: the unnamed region shows both clusters', async ({ page }) => {
		await page.goto('/people/unnamed');
		await expect(page).toHaveTitle('Unnamed · lumemor');

		// Both clusters render as cards with their honest face
		// counts: one pair per person, and the same-person cluster's
		// card is the one whose crop is its representative face.
		await expect(clusterCard(page, 2)).toHaveCount(2);
		await expect(clusterCardByFace(page, samePersonRepFaceId)).toBeVisible();
	});

	test('scene 4: the naming dialog moves a cluster into the named region', async ({ page }) => {
		await page.goto('/people/unnamed');
		const samePersonCard = clusterCardByFace(page, samePersonRepFaceId);
		await expect(samePersonCard).toBeVisible();

		// Clicking the card opens the naming dialog (the main path).
		await samePersonCard.click();
		const dialog = page.getByRole('dialog', { name: 'Name this person', exact: true });
		await expect(dialog).toBeVisible();
		await dialog.getByLabel('Name').fill(PERSON_A);
		await dialog.getByRole('button', { name: 'Save name', exact: true }).click();

		// The PUT lands and the card leaves the region immediately
		// (the response is the fresh label state); the different
		// person's card stays.
		await expect(dialog).toHaveCount(0);
		await expect(samePersonCard).toHaveCount(0);
		await expect(page.locator('button.cluster-card')).toHaveCount(1);

		// The named region gained the person with the honest count.
		await page.goto('/people');
		const card = page.locator('a.person-card').filter({ hasText: PERSON_A });
		await expect(card).toBeVisible();
		await expect(card).toContainText('2 faces');
	});

	test('scene 5: the person card opens the cluster detail', async ({ page }) => {
		await page.goto('/people');
		await page.locator('a.person-card').filter({ hasText: PERSON_A }).click();
		await expect(page).toHaveURL(new RegExp(`/people/${samePersonClusterId}$`));

		// The detail carries the name, the faces heading and the two
		// faces of the same-person pair, each with its asset context
		// (the file name) and the link into the viewer.
		await expect(page.getByRole('heading', { level: 1, name: PERSON_A, exact: true })).toBeVisible();
		await expect(page.getByRole('heading', { level: 2, name: 'Faces', exact: true })).toBeVisible();
		await expect(page.locator('.face-card')).toHaveCount(2);
		await expect(page.getByText(PHOTO_A1_NAME)).toBeVisible();
		await expect(page.getByText(PHOTO_A2_NAME)).toBeVisible();
		await expect(
			page.getByRole('link', { name: `Face in ${PHOTO_A1_NAME}`, exact: true })
		).toBeVisible();
		await expect(page.getByRole('link', { name: 'People', exact: true })).toBeVisible();
	});

	test('scene 6: the merge folds the other person into the first', async ({ page }) => {
		// Name the different person's cluster through the same
		// dialog path.
		await page.goto('/people/unnamed');
		const remaining = page.locator('button.cluster-card').first();
		await expect(remaining).toBeVisible();
		await remaining.click();
		const naming = page.getByRole('dialog', { name: 'Name this person', exact: true });
		await expect(naming).toBeVisible();
		await naming.getByLabel('Name').fill(PERSON_B);
		await naming.getByRole('button', { name: 'Save name', exact: true }).click();
		await expect(naming).toHaveCount(0);

		// The region settles empty (every visible cluster is named).
		await expect(
			page.getByText('Nothing to name yet. Face clusters appear here once faces are grouped.')
		).toBeVisible();

		// Both people are listed; the merged-away person's detail
		// carries the merge-into action.
		await page.goto('/people');
		await expect(page.locator('a.person-card')).toHaveCount(2);
		await page.locator('a.person-card').filter({ hasText: PERSON_B }).click();
		await expect(page.getByRole('heading', { level: 1, name: PERSON_B, exact: true })).toBeVisible();
		await page.getByRole('button', { name: 'Merge into…', exact: true }).click();

		// The merge dialog states the honest cost and offers the
		// first person as the target (the only other named person).
		const dialog = page.getByRole('dialog', { name: 'Merge people', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText('This person merges into the one you pick.');
		await expect(dialog).toContainText('This cluster disappears and its name is lost.');
		const target = dialog.locator('#cluster-picker-target');
		await expect(target).toContainText(PERSON_A);
		await dialog.getByRole('button', { name: 'Merge', exact: true }).click();

		// The merge leaves for the named region: the survivor carries
		// the merged face count (2 + 2), the merged-away person is
		// gone, and the unnamed region stays empty.
		await expect(page).toHaveURL(/\/people$/);
		const survivor = page.locator('a.person-card').filter({ hasText: PERSON_A });
		await expect(survivor).toBeVisible();
		await expect(survivor).toContainText('4 faces');
		await expect(page.locator('a.person-card')).toHaveCount(1);
		await page.goto('/people/unnamed');
		await expect(
			page.getByText('Nothing to name yet. Face clusters appear here once faces are grouped.')
		).toBeVisible();
	});

	test('scene 7: the viewer carries the person on the face chips', async ({ page }) => {
		// A face card of the survivor's detail deep-links into the
		// asset viewer.
		await page.goto('/people');
		await page.locator('a.person-card').filter({ hasText: PERSON_A }).click();
		await page.getByRole('link', { name: `Face in ${PHOTO_A1_NAME}`, exact: true }).click();
		await expect(page).toHaveURL(new RegExp(`/a/[0-9a-f-]{36}$`));
		await expect(page.getByText(PHOTO_A1_NAME)).toBeVisible();

		// The info drawer's People section carries the person's name
		// on the face's chip (the caller's own label of the face's
		// cluster -- the merged person, every face of the merged set).
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = page.getByRole('complementary', { name: 'Photo information' });
		await expect(drawer).toBeVisible();
		await expect(drawer.locator('.info__face-name')).toHaveText(PERSON_A);
	});
});
