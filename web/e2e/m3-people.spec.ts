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
// THE SCENES (the E12 people domain -- three pages, two states, in
// order):
//
//  1. Signing in through the form lands on the deep-linked people
//     page: the one-row bar (title + count, the name filter, the
//     badged unnamed entry) over the honest empty state.
//  2. The library bootstrap, all through the API (the M2 story's
//     seeding pattern): the model group's download run is driven to
//     succeeded, the four fixtures upload, and the unnamed listing
//     converges to exactly two clusters of two faces -- identified
//     through the cluster details so the story knows which unnamed
//     card is which person.
//  3. The unnamed region renders both clusters as cards with their
//     face counts -- as ENTRANCES: links into the person detail, no
//     selection surface anywhere (§7.20).
//  4. The naming main path moved to the detail's unnamed variant:
//     the card opens `/people/{id}?state=unnamed`, whose profile row
//     reads 「Unnamed」 and whose naming entry lands the named state
//     in place. The named list gains the person with the PHOTO
//     count; the unnamed region loses the card.
//  5. The detail is THIS PERSON'S PHOTOS (§7.17): the sticky 53px
//     context bar (no shell topbar), the profile row (name, photo
//     count, birth-date entry) and the shared wall. The photo hide
//     flows through the wall's batch bar and back (the hidden page's
//     unhide restores the tile and the count).
//  6. The person entity's attributes: the birth date renders
//     locale-aware through Intl (machine value in <time>), and the
//     avatar picker (faces only, no file names, no governance) lands
//     the pick on the profile row and the list card alike.
//  7. The merge flow: the other cluster is named through its own
//     unnamed variant, its detail's "Merge into…" (the shared person
//     picker) folds it into the first person and LANDS on the
//     survivor's own page; the survivor carries the merged wall and
//     count while the merged-away person is gone.
//  8. The viewer: a WALL TILE deep-links into the asset viewer, whose
//     info drawer carries the person's name on the face's chip and
//     the capture-time AGE the birth date exists for.
//  9. The edit-people panel: the drawer's People pencil opens the
//     per-face panel; "Create a new person from this face" stages,
//     the one-shot commit writes and echoes the count toast, and the
//     seeded cluster lands unnamed (named through its detail's
//     variant so the next scene has a second target).
// 10. The panel's reassign (the picker's local filter narrows, the
//     pick stages, the commit writes once) and the detach (the
//     confirm names the person, the row goes back among the not
//     grouped yet); the person detail's unname carries its toast
//     echo, the entity's birth date survives it, the person lands in
//     the unnamed region, and the acceptance script's final beat
//     names them again.
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
const PORTRAIT_A2 = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-orientation6.jpg');
const PORTRAIT_B1 = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other.jpg');
const PORTRAIT_B2 = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other-second.jpg');

// The uploaded file names (the detail's asset context renders them).
const PHOTO_A1_NAME = 'IMG_M3B_0001.jpg';
const PHOTO_A2_NAME = 'IMG_M3B_0002.jpg';
const PHOTO_B1_NAME = 'IMG_M3B_0003.jpg';
const PHOTO_B2_NAME = 'IMG_M3B_0004.jpg';

// The two story names, and the third person the edit-people story
// creates from one face (named so the reassign target is pickable).
const PERSON_A = 'Ada';
const PERSON_B = 'Grace';
const PERSON_C = 'Kay';

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

// The unnamed region's cluster cards are ENTRANCES (§7.20): one link
// per cluster, its accessible name the 「Unnamed」 slot beside the
// honest face count (the count is always plural -- the i18n message
// has no singular form). Addressed by the representative face's crop
// URL when the story needs one specific cluster's card.
function unnamedCardByFace(page: Page, faceId: string) {
	return page.locator('a.cluster-card').filter({ has: page.locator(`img[src="/api/v1/faces/${faceId}/thumbnail"]`) });
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

	// The one-row bar (§7.18): the title with the count, the name
	// filter and the unnamed entry (badged with the region's count --
	// zero, so the badge is absent but the entry stands).
	await expect(page.getByRole('heading', { level: 1 })).toContainText('People · 0 people');
	await expect(page.getByPlaceholder('Filter by name')).toBeVisible();
	await expect(page.getByRole('link', { name: 'Unnamed · 0 groups', exact: true })).toBeVisible();
	// The empty state carries the honest copy.
	await expect(page.getByText('No people yet')).toBeVisible();
	await expect(page.getByText('Photos with faces will show up here.')).toBeVisible();

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
		expect(faceSets.some((set) => set.every((name) => pairA.includes(name)) && set.length === pairA.length)).toBe(
			true
		);
		expect(faceSets.some((set) => set.every((name) => pairB.includes(name)) && set.length === pairB.length)).toBe(
			true
		);
		const samePerson = clusters.find((cluster) => {
			const set = faceSets[clusters.indexOf(cluster)];
			return set.every((name) => pairA.includes(name));
		});
		samePersonClusterId = samePerson.clusterId;
		samePersonRepFaceId = samePerson.representativeFaceId;
	});

	test('scene 3: the unnamed region shows both clusters as entrances', async ({ page }) => {
		await page.goto('/people/unnamed');
		await expect(page).toHaveTitle('Unnamed · lumemor');

		// The one-row bar (§7.18): back, the title and the REGION's
		// count -- no action, no select switch (the cards are
		// entrances, never a selection surface).
		await expect(page.getByRole('link', { name: 'Back to people', exact: true })).toBeVisible();
		await expect(page.getByRole('heading', { level: 1 })).toContainText('Unnamed · 2 groups');
		await expect(page.locator('.unnamed__toolbar button')).toHaveCount(0);

		// Both clusters render as cards carrying their honest face
		// counts, and the same-person cluster's card is the one whose
		// crop is its representative face.
		await expect(page.getByRole('link', { name: /2 faces/ })).toHaveCount(2);
		await expect(unnamedCardByFace(page, samePersonRepFaceId)).toBeVisible();
		// Nothing selectable: no check circle exists anywhere.
		await expect(page.locator('.cluster-card__check')).toHaveCount(0);
	});

	test('scene 4: the unnamed card opens the detail, where the naming lands', async ({ page }) => {
		await page.goto('/people/unnamed');
		const card = unnamedCardByFace(page, samePersonRepFaceId);
		await expect(card).toBeVisible();

		// The card is an ENTRANCE into the same detail page the named
		// cards open (the unnamed variant).
		await card.click();
		await expect(page).toHaveURL(new RegExp(`/people/${samePersonClusterId}\\?state=unnamed$`));
		await expect(page).toHaveTitle('Unnamed · lumemor');

		// The unnamed variant: the shell's topbar is gone (the page
		// owns its contextual bar), the profile row reads 「Unnamed」
		// and the action set swaps the unname for the naming.
		await expect(page.locator('.shell-topbar')).toHaveCount(0);
		const bar = page.locator('.person-bar');
		await expect(bar).toBeVisible();
		await expect(page.locator('.person-profile__name')).toHaveText('Unnamed');
		await expect(page.getByRole('button', { name: 'Change avatar', exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Merge into…', exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Remove name', exact: true })).toHaveCount(0);

		// The naming entry opens the dialog; the PUT lands and the
		// profile row adopts the response -- the named state arrives in
		// place (no navigation).
		await page.getByRole('button', { name: 'Name this person', exact: true }).first().click();
		const dialog = page.getByRole('dialog', { name: 'Name this person', exact: true });
		await expect(dialog).toBeVisible();
		await dialog.getByLabel('Name').fill(PERSON_A);
		await dialog.getByRole('button', { name: 'Save name', exact: true }).click();
		await expect(dialog).toHaveCount(0);
		await expect(page.locator('.person-profile__name')).toHaveText(PERSON_A);
		await expect(page.getByRole('button', { name: 'Remove name', exact: true })).toBeVisible();

		// The named region gained the person with the honest PHOTO
		// count (photos, not faces: the pair is two photos), and the
		// unnamed region lost the card.
		await page.goto('/people');
		const card2 = page.locator('a.person-card').filter({ hasText: PERSON_A });
		await expect(card2).toBeVisible();
		// The count is icon + number; its accessible name states the
		// unit (the card never spells "photos" as loose text).
		await expect(card2.locator('.person-card__count')).toHaveAttribute('aria-label', '2 photos');
		await page.goto('/people/unnamed');
		await expect(page.getByRole('heading', { level: 1 })).toContainText('Unnamed · 1 group');
		await expect(unnamedCardByFace(page, samePersonRepFaceId)).toHaveCount(0);
	});

	test('scene 5: the detail is this person’s photo wall, with the photo hide flowing through it', async ({
		page
	}) => {
		await page.goto('/people');
		await page.locator('a.person-card').filter({ hasText: PERSON_A }).click();
		await expect(page).toHaveURL(new RegExp(`/people/${samePersonClusterId}$`));

		// THE CONTEXT BAR (§7.21): sticky, 53px, back + the icon
		// actions; the shell's own chrome does not render here.
		await expect(page.locator('.shell-topbar')).toHaveCount(0);
		const bar = page.locator('.person-bar');
		await expect(bar).toBeVisible();
		expect(await bar.evaluate((element) => Math.round(element.getBoundingClientRect().height))).toBe(53);
		await expect(page.getByRole('link', { name: 'Back to people', exact: true })).toBeVisible();

		// THE PROFILE ROW: the name (with its rename pencil), the photo
		// count and the birth-date entry.
		await expect(page.locator('.person-profile__name')).toHaveText(PERSON_A);
		await expect(page.locator('.person-profile__rename')).toHaveAttribute('aria-label', 'Rename');
		await expect(page.locator('.person-profile__count')).toHaveText('2');
		await expect(page.locator('.person-profile__birthdate')).toHaveText('Add birth date');

		// THE WALL: this person's two photos, the shared wall's tiles.
		await expect(page.locator('.person-wall .tile')).toHaveCount(2);

		// THE PHOTO HIDE (the only hide left in the people domain --
		// the batch bar's): select the first photo, hide it through the
		// bar, and the wall + the profile count converge.
		const tile = page.locator('.person-wall .tile').first();
		await tile.hover();
		await tile.locator('.tile__check').click();
		const batchBar = page.locator('.batchbar');
		await expect(batchBar).toBeVisible();
		await batchBar.getByRole('button', { name: 'Hide', exact: true }).click();
		await expect(page.locator('.person-wall .tile')).toHaveCount(1);
		await expect(page.locator('.person-profile__count')).toHaveText('1');

		// The hidden photo is restorable from the hidden page and comes
		// back to this wall (the restored event's convergence).
		await page.goto('/hidden');
		await expect(page.locator('.wall .tile')).toHaveCount(1);
		const hiddenTile = page.locator('.wall .tile').first();
		await hiddenTile.hover();
		await hiddenTile.locator('.tile__check').click();
		await page.locator('.batchbar').getByRole('button', { name: 'Unhide', exact: true }).click();
		await expect(page.locator('.wall .tile')).toHaveCount(0);

		await page.goto('/people');
		await expect(
			page.locator('a.person-card').filter({ hasText: PERSON_A }).locator('.person-card__count')
		).toHaveAttribute('aria-label', '2 photos');
	});

	test('scene 6: the birth date and the avatar picker write the person entity’s attributes', async ({ page }) => {
		await page.goto('/people');
		await page.locator('a.person-card').filter({ hasText: PERSON_A }).click();
		await expect(page.locator('.person-wall .tile')).toHaveCount(2);

		// THE BIRTH DATE: the profile row's entry opens the dialog; the
		// date renders locale-aware through Intl with the machine value
		// in <time>.
		await page.locator('.person-profile__birthdate').click();
		const birthDialog = page.getByRole('dialog', { name: 'Birth date', exact: true });
		await expect(birthDialog).toBeVisible();
		await birthDialog.locator('input[type="date"]').fill('1992-03-15');
		await birthDialog.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(birthDialog).toHaveCount(0);
		const birthLine = page.locator('.person-profile__birthdate');
		await expect(birthLine).toContainText('Born 3/15/1992');
		await expect(birthLine.locator('time')).toHaveAttribute('datetime', '1992-03-15');

		// THE AVATAR PICKER: the person's faces only -- the crops, no
		// file names and no governance.
		await page.getByRole('button', { name: 'Change avatar', exact: true }).click();
		const avatarDialog = page.getByRole('dialog', { name: 'Change avatar', exact: true });
		await expect(avatarDialog).toBeVisible();
		await expect(avatarDialog).toContainText("Pick one of this person's faces as their avatar.");
		const options = avatarDialog.locator('.avatar-option');
		await expect(options).toHaveCount(2);
		await expect(avatarDialog).not.toContainText(PHOTO_A1_NAME);
		await expect(avatarDialog.locator('.avatar-option__badge')).toHaveCount(0);

		// The pick lands: the profile row's crop becomes the picked
		// face's.
		const picked = options.nth(1);
		const pickedCrop = await picked.locator('img').getAttribute('src');
		await picked.click();
		await avatarDialog.getByRole('button', { name: 'Set avatar', exact: true }).click();
		await expect(avatarDialog).toHaveCount(0);
		await expect(page.locator('.person-profile__avatar img')).toHaveAttribute('src', pickedCrop!);

		// The named list's card crop follows the stored pick (the
		// avatar is the entity's, not the caller's).
		await page.goto('/people');
		await expect(page.locator('a.person-card').filter({ hasText: PERSON_A }).locator('img')).toHaveAttribute(
			'src',
			pickedCrop!
		);
	});

	test('scene 7: the merge folds the other person into this one', async ({ page }) => {
		// Name the different person's cluster through its own unnamed
		// detail (the same entrance path).
		await page.goto('/people/unnamed');
		const remaining = page.locator('a.cluster-card').first();
		await expect(remaining).toBeVisible();
		await remaining.click();
		await expect(page.locator('.person-profile__name')).toHaveText('Unnamed');
		await page.getByRole('button', { name: 'Name this person', exact: true }).first().click();
		const naming = page.getByRole('dialog', { name: 'Name this person', exact: true });
		await expect(naming).toBeVisible();
		await naming.getByLabel('Name').fill(PERSON_B);
		await naming.getByRole('button', { name: 'Save name', exact: true }).click();
		await expect(naming).toHaveCount(0);
		await expect(page.locator('.person-profile__name')).toHaveText(PERSON_B);

		// The merge face: the shared person picker (this cluster is the
		// single source), the honest cost in the body, and the landing
		// on the SURVIVING person's page after the fold.
		await page.getByRole('button', { name: 'Merge into…', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Merge into…', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText('This person merges into the one you pick.');
		await expect(dialog).toContainText('This cluster disappears and its name is lost.');
		await dialog.getByRole('radio', { name: PERSON_A }).click();
		await dialog.getByRole('button', { name: 'Merge into…', exact: true }).click();

		// The survivor's page carries the merged wall (4 photos) and
		// the merged count; the merged-away person is gone from the
		// list and the unnamed region stays empty.
		await expect(page).toHaveURL(new RegExp(`/people/${samePersonClusterId}$`));
		await expect(page.locator('.person-profile__name')).toHaveText(PERSON_A);
		await expect(page.locator('.person-profile__count')).toHaveText('4');
		await expect(page.locator('.person-wall .tile')).toHaveCount(4);
		await page.goto('/people');
		await expect(page.locator('a.person-card')).toHaveCount(1);
		await expect(
			page.locator('a.person-card').filter({ hasText: PERSON_A }).locator('.person-card__count')
		).toHaveAttribute('aria-label', '4 photos');
		await page.goto('/people/unnamed');
		await expect(page.getByText('Face clusters appear here once faces are grouped.')).toBeVisible();
	});

	test('scene 8: a wall tile opens the viewer, whose chip carries the person and the age', async ({ page }) => {
		// The wall's tile IS the way into the photo (the faces grid is
		// gone): its link names the photo.
		await page.goto('/people');
		await page.locator('a.person-card').filter({ hasText: PERSON_A }).click();
		await page.locator('.person-wall .tile__link').first().click();
		await expect(page).toHaveURL(new RegExp(`/a/[0-9a-f-]{36}$`));

		// The info drawer's People section carries the person's name on
		// the face's chip, and -- because the entity now has a birth
		// date -- the age WHEN THIS PHOTO WAS TAKEN (the birth date's
		// whole purpose besides birthday memories).
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = page.getByRole('complementary', { name: 'Photo information' });
		await expect(drawer).toBeVisible();
		await expect(drawer.locator('.info__face-name')).toHaveText(PERSON_A);
		await expect(drawer.locator('.info__face-age')).toContainText('years');
	});

	test('scene 9: the edit-people panel creates a person from a face', async ({ page }) => {
		// The scene's own route (the spec's convention: every scene
		// navigates explicitly): the viewer with its info drawer open.
		await page.goto('/people');
		await page.locator('a.person-card').filter({ hasText: PERSON_A }).click();
		await page.locator('.person-wall .tile__link').first().click();
		await expect(page).toHaveURL(new RegExp(`/a/[0-9a-f-]{36}$`));
		await page.getByRole('button', { name: 'Photo information' }).click();
		await expect(page.getByRole('complementary', { name: 'Photo information' })).toBeVisible();

		// The People section's pencil opens the edit-people panel in
		// the drawer's exact slot -- the photo stays visible.
		await page.getByRole('button', { name: 'Edit people', exact: true }).click();
		const panel = page.getByRole('complementary', { name: 'Edit people' });
		await expect(panel).toBeVisible();
		await expect(panel.getByRole('heading', { level: 2, name: 'Edit people', exact: true })).toBeVisible();
		await expect(panel).toContainText(
			'Every face in this photo is here. A face matched to the wrong person can be reassigned to someone else, or removed from the person it sits on.'
		);
		// The row carries the age beside the name (the same entity
		// attribute the chip reads).
		await expect(panel.locator('.faces-panel__age').first()).toContainText('years');

		// The face's row actions: the reassign pencil opens the
		// choose-person picker, whose creation row STAGES a new person.
		await panel.getByRole('button', { name: 'Reassign this face', exact: true }).click();
		const picker = page.getByRole('dialog', { name: 'Choose person', exact: true });
		await expect(picker).toBeVisible();
		await expect(picker).toContainText(
			'Who does this face belong to? Pick an existing person, or create a new person from this face.'
		);
		await picker.getByRole('button', { name: 'Create a new person from this face', exact: true }).click();
		await expect(picker).toHaveCount(0);
		await expect(panel.locator('li[data-staged="true"]')).toHaveCount(1);

		// The one-shot commit writes the create and echoes the count;
		// the successful commit CLOSES the panel (the landing is the
		// feedback -- the created cluster surfaces on its own).
		await panel.getByRole('button', { name: 'Finish editing', exact: true }).click();
		await expect(page.locator('.toasts')).toContainText('Made 1 change.');
		await expect(panel).toHaveCount(0);

		// The seeded cluster lands in the unnamed region with the one
		// face; naming it through its detail's variant gives the story
		// a second named person for the reassign scene.
		await page.goto('/people/unnamed');
		await expect(page.locator('a.cluster-card')).toHaveCount(1);
		await page.locator('a.cluster-card').first().click();
		await page.getByRole('button', { name: 'Name this person', exact: true }).first().click();
		const naming = page.getByRole('dialog', { name: 'Name this person', exact: true });
		await expect(naming).toBeVisible();
		await naming.getByLabel('Name').fill(PERSON_C);
		await naming.getByRole('button', { name: 'Save name', exact: true }).click();
		await expect(naming).toHaveCount(0);
		await page.goto('/people');
		await expect(page.locator('a.person-card')).toHaveCount(2);
	});

	test('scene 10: the panel reassigns, detaches, and the unname returns the person to unnamed', async ({ page }) => {
		// The second photo of the pair (still the first person's face):
		// the reassign stages a move to the newly named person.
		await page.goto('/people');
		await page.locator('a.person-card').filter({ hasText: PERSON_A }).click();
		// The wall's second tile is the pair's other photo.
		await page.locator('.person-wall .tile__link').first().click();
		await expect(page).toHaveURL(new RegExp(`/a/[0-9a-f-]{36}$`));
		await page.getByRole('button', { name: 'Photo information' }).click();
		await expect(page.getByRole('complementary', { name: 'Photo information' })).toBeVisible();
		await page.getByRole('button', { name: 'Edit people', exact: true }).click();
		const panel = page.getByRole('complementary', { name: 'Edit people' });
		await expect(panel).toBeVisible();

		// The reassign (a CLUSTERED face moves between people): the
		// picker's local filter narrows the two named people to the one
		// typed, the pick stages, and the one-shot commit writes once
		// and CLOSES the panel with the count echo.
		await panel.getByRole('button', { name: 'Reassign this face', exact: true }).click();
		const picker = page.getByRole('dialog', { name: 'Choose person', exact: true });
		await expect(picker.getByRole('radio')).toHaveCount(1);
		await picker.getByLabel('Filter by name').fill('Ad');
		await expect(picker.getByRole('radio')).toHaveCount(0);
		await expect(picker.getByText('No people match.')).toBeVisible();
		await picker.getByLabel('Filter by name').fill('Ka');
		await expect(picker.getByRole('radio')).toHaveCount(1);
		await picker.getByRole('radio', { name: PERSON_C }).click();
		await picker.getByRole('button', { name: 'Assign', exact: true }).click();
		await expect(picker).toHaveCount(0);
		await expect(panel.locator('li[data-staged="true"]')).toHaveCount(1);
		await panel.getByRole('button', { name: 'Finish editing', exact: true }).click();
		await expect(page.locator('.toasts')).toContainText('Made 1 change.');
		await expect(panel).toHaveCount(0);

		// The detach is the immediate write: the confirm names the
		// person, the row goes back among the not grouped yet, and the
		// echo lands.
		await page.getByRole('button', { name: 'Edit people', exact: true }).click();
		const reopen = page.getByRole('complementary', { name: 'Edit people' });
		await expect(reopen).toBeVisible();
		await expect(reopen.locator('.faces-panel__name').first()).toContainText(PERSON_C);
		await reopen.getByRole('button', { name: 'Remove this face', exact: true }).click();
		const detach = page.getByRole('dialog', { name: 'Remove this face', exact: true });
		await expect(detach).toBeVisible();
		await detach.getByRole('button', { name: 'Remove', exact: true }).click();
		await expect(detach).toHaveCount(0);
		await expect(page.locator('.toasts')).toContainText('Face removed; it is back among the not grouped yet.');
		await expect(reopen.locator('.faces-panel__name').first()).toContainText('Not grouped yet');

		// THE UNNAME (§7.19): the person detail's action is immediate
		// and carries the toast echo; the person leaves the named list
		// and lands in the unnamed region as one more group.
		await page.goto('/people');
		await page.locator('a.person-card').filter({ hasText: PERSON_A }).click();
		await page.getByRole('button', { name: 'Remove name', exact: true }).click();
		await expect(page.locator('.toasts')).toContainText('Name removed; this person is unnamed again.');
		// The same page lands the unnamed variant in place.
		await expect(page.locator('.person-profile__name')).toHaveText('Unnamed');
		// The birth date is the ENTITY's: the unnamed variant still
		// carries it (it never depended on the name).
		await expect(page.locator('.person-profile__birthdate')).toContainText('Born 3/15/1992');

		await page.goto('/people');
		await expect(page.locator('a.person-card')).toHaveCount(1);
		await expect(page.locator('a.person-card').filter({ hasText: PERSON_C })).toBeVisible();
		await page.goto('/people/unnamed');
		await expect(page.getByRole('heading', { level: 1 })).toContainText('Unnamed · 1 group');

		// The final beat of the owner's acceptance script: name the
		// person again from the unnamed variant.
		await page.locator('a.cluster-card').first().click();
		await page.getByRole('button', { name: 'Name this person', exact: true }).first().click();
		const rename = page.getByRole('dialog', { name: 'Name this person', exact: true });
		await expect(rename).toBeVisible();
		await rename.getByLabel('Name').fill(PERSON_A);
		await rename.getByRole('button', { name: 'Save name', exact: true }).click();
		await expect(rename).toHaveCount(0);
		await expect(page.locator('.person-profile__name')).toHaveText(PERSON_A);
	});
});
