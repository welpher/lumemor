// The manual memories story of the web app, walked end to end against
// the real compose stack: a memory is a REFERENCE SET -- the photos
// do not live inside it, they are pointed at, and the memory's whole
// life is curation: photo by photo in, photo by photo out, an
// audience beside it. The story walks the page family the concept
// owns (the memories wall, the memory's own detail route, the wall
// selection's picker) and the semantics that make it a memory rather
// than an album: EVERY member creates (personal curation, never a
// configuration act), the photos a member gathers may include the
// ones others shared to them (the re-share ground -- a row lands
// only for a photo the creator CURRENTLY sees, and the shared views
// re-check that sight on every read, so a revoked photo leaves the
// audience's view with no snapshot residue), the memory NEVER feeds
// its creator's timeline (it is not a feed -- the photos appear
// there exactly once each, through their own arms), a directory
// album refuses the same curation (its membership is its scan's
// verdict -- the container/reference-set divide), and the memory's
// delete takes only its own rows (the references and the audience
// die, the photos stay).
//
// THE STACK ARRANGEMENT (the honest reuse path, the m3-albums
// arrangement): the stack is fresh (its database volume was removed
// before it started) but the models ride the SAME storage bind mount
// the chained acceptances downloaded into -- ./lumemor persists
// across the script's stack lifetimes, and the download service's
// already-verified skip re-registers the weights without re-fetching
// them. The fixture tree web/e2e/fixtures/m3-source rides its
// read-only bind at /mnt/e2e-source as every e2e story's source
// ground; this story's directory album references coastal-week under
// it (referenced in place, zero copy) and the read-only bind answers
// the creation-time write probe with the honest false fact.
//
// THE THREE-ACCOUNT DANCE (the sharing story's family shape): the
// first-run administrator owns the source half -- the directory
// album over the fixture, its audience grant, and the revocation
// that ends it; the CREATOR (a plain member, provisioned through the
// admin API) is the story's subject -- the every-member creation,
// the wall-selection curation, the audience edit; the SHAREE
// (another plain member) is the audience -- the read-only memory
// view whose live recomputation is the sharing story's own. The
// creator's sight of one shared photo rides the directory album's
// grant: that photo is the memory's re-shared member, and the
// revocation of the grant is the cascade's trigger.
//
// THE SCENES (the memories surface, in order):
//
//  1. The administrator signs in through the form and lands on the
//     timeline (the source half's session, saved for its scenes).
//  2. The source half through the API: the model group's download
//     run driven to succeeded, the directory album created over the
//     fixture with the CREATOR in the audience at create (the grant
//     lands with the row), the rescan converging the fixture's two
//     photos to ready members, and the curation refusal -- the
//     owner's own add-items attempt on the directory album folds
//     into the single-shape 404 (a source's membership is its
//     scan's verdict, never a hand-picked set).
//  3. The creator signs in through their own form and lands on the
//     memories wall: the fully-empty state's copy points at the
//     wall-selection entry, and the New memory button renders for
//     the plain member (the every-member pin -- the opposite gate
//     of the album wall's admin-only creation).
//  4. The creator's seed through the API: their own three photos
//     uploaded and converged to ready, the third HIDDEN through the
//     batch write (the owner-level visibility switch -- the honest
//     skip's setup), leaving the timeline at the own pair plus the
//     shared pair.
//  5. The creation: the New memory dialog (the title and the
//     audience alone -- a memory owns no source row and no
//     switches), the POST carrying exactly the dialog's shape, the
//     all-landed creation closing directly, and the SSE echo
//     converging the wall: the card with the honest zero count and
//     the manual origin badge.
//  6. The wall-selection entry: the timeline's selection (two own
//     photos and the shared one -- the re-share ground's add arm),
//     the batch bar's sparkles action, the picker (the captured-ids
//     intro, the memory row), the pick's POST and its honest
//     {added, skipped} report, the row flipping to the added state,
//     and Done closing with the wall's selection cleared -- no
//     convergence follows (adding to a memory changes no photo's
//     state; the tiles stay exactly as they were).
//  7. The honest skip: the hidden wall carries the hidden photo
//     (a photo the creator cannot currently see), its batch bar's
//     same sparkles entry, and the picker's pick answers
//     {added 0, skipped 1} with the skip-note honesty line -- the
//     row never lands, the memory's count is unchanged.
//  8. The audience share: the memory detail's edit dialog (the
//     title field and the audience chips alone -- no source row, no
//     autonomy toggles), the candidate picker adding the sharee's
//     chip, and the save writing the audience PUT alone (the
//     unchanged title writes nothing).
//  9. The sharee's read-only view: their own sign-in, the memories
//     wall's one list carrying the shared card (the owner's name,
//     the read-only chip, the honest count), and the
//     detail's member bar -- the attribution and the badge with NO
//     action at all, the wall pure navigation (no check ever
//     renders).
// 10. The source revocation cascade: the sharee's memories wall
//     held open (the shared card, its count and its collage), the
//     creator's own wall and timeline held open, and the
//     administrator emptying the source album's audience -- the
//     grant's death drops the re-shared photo from the sharee's
//     card LIVE (the album event converges the open wall, no
//     reload), the sharee's fresh detail visit answers the grounded
//     count beside a wall that agrees (the walk drops the revoked
//     photo too -- the page never contradicts itself), the
//     creator's own card keeps the plain live membership (the row
//     stays; the set is the owner's to tidy), and the creator's
//     timeline loses the shared pair (the memory's row does not
//     hold a photo there -- it is not a feed).
// 11. The timeline exclusion, the fresh view: the creator's
//     aggregated timeline holds exactly their own pair, each photo
//     exactly once -- the memory's three rows inject nothing, and
//     the revoked photo's row never kept it on the feed.
// 12. The removal: the memory detail's owner bar (the edit, cover
//     and delete arms; no rescan, no un-ingest -- a memory owns no
//     source), the wall curating the REMOVE-ONLY subset (the danger
//     minus action alone), the dead-source row selected and removed
//     (the DELETE's honest report, the tile leaving through the
//     membership event, the count dropping) -- and the photo itself
//     stays (the library row lives on, read by its owner).
// 13. The hand-over: the albums wall never carries the memory (its
//     home is the memories wall), and a memory id visited on the
//     album detail route lands on the memory's own route.
// 14. The delete: the confirm stating the light cascade (the
//     references and the audience die, the photos stay), the DELETE
//     leaving for the memories wall, the sharee's held wall losing
//     the shared card live (the audience died with the memory), the
//     creator's own photos still on the timeline (the library keeps
//     them), and the memory answering the single-shape 404 after.
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story --
// see playwright.config.ts). Scene 1 signs the administrator in and
// saves the storage state; scene 2 replays it. Scene 3 signs the
// creator in through a fresh context and saves their own; scenes
// 4-8 and 10-14 replay it. Scene 9 signs the sharee in through
// another fresh context and saves theirs; scenes 10 and 14 replay it
// inside browser contexts created within those scenes (the three
// accounts never share a browser session). Each scene navigates
// explicitly to its starting route, so a failure report names the
// exact scene that broke.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's three accounts: the first-run administrator (the
// source half's owner), the member who curates the memory, and the
// member the memory is shared to. The stack is always fresh, so the
// fixed identities are deterministic.
const ADMIN_EMAIL = 'admin@m3m-e2e.invalid';
const ADMIN_NAME = 'Mei Chen';
const ADMIN_PASSWORD = 'correct-horse-battery';
const CREATOR_EMAIL = 'grace@m3m-e2e.invalid';
const CREATOR_NAME = 'Grace Hopper';
const CREATOR_PASSWORD = 'correct-horse-battery';
const SHAREE_EMAIL = 'hedy@m3m-e2e.invalid';
const SHAREE_NAME = 'Hedy Lamarr';
const SHAREE_PASSWORD = 'correct-horse-battery';

// The read-only source fixture (the m3 story family's arrangement):
// the compose file's bind, and the directory under it the story's
// directory album references in place. Its two photos are the
// creator's sight-through-the-share arm: the re-shared member and
// its sibling.
const SOURCE_DIR = '/mnt/e2e-source/coastal-week';
const VIA_SHARE_A_NAME = 'IMG_E4_0001.jpg';
const VIA_SHARE_B_NAME = 'IMG_E4_0002.jpg';

// The creator's own seed (genuinely distinct bytes from the fixture
// pair, so the story's rows never collide with the scan's): two
// photos the memory gathers, and one that starts HIDDEN -- the
// honest skip's own row.
const PHOTO_A = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other.jpg');
const PHOTO_B = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other-second.jpg');
const PHOTO_C = path.join(repoRoot, 'server/internal/jobs/testdata/golden.jpg');
const OWN_A_NAME = 'IMG_M3M_A.jpg';
const OWN_B_NAME = 'IMG_M3M_B.jpg';
const OWN_C_NAME = 'IMG_M3M_C.jpg';

// The story's titles.
const SOURCE_ALBUM_TITLE = 'Coastal days 2026';
const MEMORY_TITLE = 'Late summer 2026';

// The skip-note honesty line (the picker's report face).
const SKIP_NOTE =
	"1 photo is not visible right now and stayed out — a shared photo's visibility is re-checked at add time.";

// The download bound: the cold upstream transfer of the group is
// minutes, the warm re-verification is seconds; both allow the cold
// bound.
const DOWNLOAD_TIMEOUT_MS = 900_000;

// The storage-state files scenes 1, 3 and 9 write and the later
// scenes replay (git-ignored, recreated per run).
const adminStatePath = `${specDir}/.auth/m3m-admin-state.json`;
const creatorStatePath = `${specDir}/.auth/m3m-creator-state.json`;
const shareeStatePath = `${specDir}/.auth/m3m-sharee-state.json`;

// The identities the scenes capture: the provisioned members' user
// ids (the audience write's shape), the source album's id, the
// memory's id, and the photo rows the assertions address.
let creatorUserId = '';
let shareeUserId = '';
let sourceAlbumId = '';
let memoryId = '';
let ownAId = '';
let ownBId = '';
let ownCId = '';
let viaShareAId = '';

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance; the
	// signed-in admin then provisions the two member accounts through
	// the admin surface. The browser scenes then only exercise the
	// web app's own surfaces.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: ADMIN_NAME, email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);

	const login = await api.post('/api/v1/auth/login', {
		data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(login.status()).toBe(200);

	const creator = await api.post('/api/v1/admin/users', {
		data: { name: CREATOR_NAME, email: CREATOR_EMAIL, password: CREATOR_PASSWORD }
	});
	expect(creator.status()).toBe(201);
	creatorUserId = ((await creator.json()) as { id?: string }).id ?? '';
	expect(creatorUserId).not.toBe('');

	const sharee = await api.post('/api/v1/admin/users', {
		data: { name: SHAREE_NAME, email: SHAREE_EMAIL, password: SHAREE_PASSWORD }
	});
	expect(sharee.status()).toBe(201);
	shareeUserId = ((await sharee.json()) as { id?: string }).id ?? '';
	expect(shareeUserId).not.toBe('');
	await api.dispose();
});

test('scene 1: the administrator signs in and lands on the timeline', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);
	await expect(page).toHaveTitle('Timeline · lumemor');

	// Hand the admin session to scene 2: the storage state rides the
	// login cookie (the fresh contexts replay it).
	await page.context().storageState({ path: adminStatePath });
});

// The admin half of the story: the source album the creator's sight
// rides on, landed through the API (the memory story's own surfaces
// are the members'; the source half is the arrangement behind them).
test.describe('the administrator scenes', () => {
	test.use({ storageState: adminStatePath });

	test('scene 2: the source half lands through the API and the directory album refuses the curation', async ({
		page
	}) => {
		test.setTimeout(DOWNLOAD_TIMEOUT_MS + 120_000);

		// The model group's download first (the uploads and the scan
		// run the same pipeline fan-out; the request rides the page's
		// own context and its login cookie). Warm mount or cold, the
		// run must reach succeeded.
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

		// The directory album over the read-only fixture, the creator
		// in the audience AT CREATE (the grant lands with the row --
		// the story's sight arm). The probe answers the honest false
		// fact of the read-only bind.
		const create = await page.request.post('/api/v1/albums', {
			data: { title: SOURCE_ALBUM_TITLE, rootPath: SOURCE_DIR, audience: [creatorUserId] }
		});
		expect(create.status()).toBe(201);
		const createBody = (await create.json()) as {
			id?: string;
			sourceWritable?: boolean;
			skipped?: string[];
		};
		expect(createBody.sourceWritable).toBe(false);
		expect(createBody.skipped).toEqual([]);
		sourceAlbumId = createBody.id ?? '';
		expect(sourceAlbumId).not.toBe('');

		// The rescan: the membership converges onto the fixture's two
		// photos, and the scan's pipeline runs both files to ready
		// (the models are warm).
		const rescan = await page.request.post(`/api/v1/albums/${sourceAlbumId}/rescan`);
		expect(rescan.status()).toBe(202);
		type DetailBody = {
			itemCount?: number;
			items?: Array<{ assetId?: string; asset?: { originalFilename?: string } }>;
		};
		await expect
			.poll(
				async () => {
					const detail = await page.request.get(`/api/v1/albums/${sourceAlbumId}`);
					expect(detail.status()).toBe(200);
					const body = (await detail.json()) as DetailBody;
					if (body.itemCount !== 2) {
						return 0;
					}
					const names = (body.items ?? []).map((item) => item.asset?.originalFilename ?? '').sort();
					return names.join(',') === [VIA_SHARE_A_NAME, VIA_SHARE_B_NAME].sort().join(',') ? 2 : 1;
				},
				{ timeout: 120_000, intervals: [1000, 2000, 5000] }
			)
			.toBe(2);
		await expect
			.poll(
				async () => {
					const timeline = await page.request.get(
						`/api/v1/assets/timeline?albumId=${encodeURIComponent(sourceAlbumId)}`
					);
					expect(timeline.status()).toBe(200);
					const body = (await timeline.json()) as { items?: Array<{ isReady: boolean }> };
					return (body.items ?? []).filter((item) => item.isReady).length;
				},
				{ timeout: 120_000, intervals: [500, 1000, 2000, 5000] }
			)
			.toBe(2);

		// The members' ids, captured for the later scenes' requests.
		const detail = await page.request.get(`/api/v1/albums/${sourceAlbumId}`);
		expect(detail.status()).toBe(200);
		const detailBody = (await detail.json()) as DetailBody;
		const members = detailBody.items ?? [];
		viaShareAId = members.find((item) => item.asset?.originalFilename === VIA_SHARE_A_NAME)?.assetId ?? '';
		expect(viaShareAId).not.toBe('');

		// The curation refusal (the container/reference-set divide):
		// the ADMINISTRATOR'S own add-items attempt on their
		// directory album folds into the single-shape 404 -- a
		// source's membership is its scan's verdict, never a
		// hand-picked set. Nothing lands.
		const attempt = await page.request.post(`/api/v1/albums/${sourceAlbumId}/items`, {
			data: { assetIds: [viaShareAId] }
		});
		expect(attempt.status()).toBe(404);
		const attemptBody = (await attempt.json()) as { code?: string };
		expect(attemptBody.code).toBe('ALBUM_NOT_FOUND');
		const membership = await page.request.get(`/api/v1/albums/${sourceAlbumId}`);
		const membershipBody = (await membership.json()) as DetailBody;
		expect(membershipBody.itemCount).toBe(2);
	});
});

// The member half of the story: the creator's sign-in through their
// own form (a fresh context -- the accounts never share a browser
// session), the memories wall's every-member face, and the state
// hand-off to their scenes.
test('scene 3: the member signs in and the memories wall carries the every-member entry', async ({ browser }) => {
	// The config's use-locale does not reach manually created
	// contexts; pin it so the asserted strings render in English.
	const context = await browser.newContext({ locale: 'en-US', timezoneId: 'UTC' });
	const page = await context.newPage();

	await page.goto('/login');
	await page.getByLabel('Email').fill(CREATOR_EMAIL);
	await page.getByLabel('Password').fill(CREATOR_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);

	// The memories wall: the fully-empty state (no memory of the
	// caller's own, none shared to them yet) whose copy points at
	// the wall-selection entry -- the first memory begins from the
	// photos, not from this page.
	await page.goto('/memories');
	await expect(page).toHaveTitle('Memories · lumemor');
	// The page bar's one row: the title with its count on the left,
	// the creation at the right -- the count never takes a line of
	// its own, and the settled empty wall counts zero honestly.
	await expect(page.locator('.memories__title')).toHaveText('Memories · 0 memories');
	await expect(page.locator('.memories__toolbar')).toHaveClass(/border-b/);
	await expect(page.getByText('No memories yet')).toBeVisible();

	// The every-member pin: the New memory button renders for a
	// plain member -- a memory is personal curation, never a
	// configuration act (the album wall's creation is the
	// administrator's alone; this one is everyone's).
	await expect(page.locator('.state-card').getByRole('button', { name: 'New memory', exact: true })).toBeVisible();

	await context.storageState({ path: creatorStatePath });
	await context.close();
});

// The creator's scenes: every scene starts from a fresh context that
// replays the creator's login cookie scene 3 saved.
test.describe('the creator scenes', () => {
	test.use({ storageState: creatorStatePath });

	test('scene 4: the creator seeds their own library and hides the third photo', async ({ page }) => {
		// The three own photos through the asset API (the story's
		// seeding pattern): two the memory gathers, and the third
		// that starts HIDDEN -- the honest skip's own row.
		const seeds: Array<{ file: string; name: string; mime: string; takenAt: string }> = [
			{ file: PHOTO_A, name: OWN_A_NAME, mime: 'image/jpeg', takenAt: '2026-09-10T12:00:00Z' },
			{ file: PHOTO_B, name: OWN_B_NAME, mime: 'image/jpeg', takenAt: '2026-09-11T12:00:00Z' },
			{ file: PHOTO_C, name: OWN_C_NAME, mime: 'image/jpeg', takenAt: '2026-09-12T12:00:00Z' }
		];
		for (const seed of seeds) {
			const upload = await page.request.post('/api/v1/assets', {
				multipart: {
					deviceAssetId: `m3m-e2e-${seed.name}`,
					fileName: seed.name,
					deviceCreatedAt: seed.takenAt,
					deviceTimeZone: 'UTC',
					photo: {
						name: seed.name,
						mimeType: seed.mime,
						buffer: await readFile(seed.file)
					}
				}
			});
			expect(upload.status()).toBe(201);
			const body = (await upload.json()) as { id?: string };
			if (seed.name === OWN_A_NAME) {
				ownAId = body.id ?? '';
			} else if (seed.name === OWN_B_NAME) {
				ownBId = body.id ?? '';
			} else {
				ownCId = body.id ?? '';
			}
		}
		expect(ownAId).not.toBe('');
		expect(ownBId).not.toBe('');
		expect(ownCId).not.toBe('');

		// The convergence: all three reach the ready state (the
		// timeline's ready count carries the own three PLUS the
		// shared pair -- the source grant's arm, already converged in
		// scene 2).
		type TimelineBody = { items?: Array<{ id: string; isReady: boolean }> };
		await expect
			.poll(
				async () => {
					const timeline = await page.request.get('/api/v1/assets/timeline');
					expect(timeline.status()).toBe(200);
					const body = (await timeline.json()) as TimelineBody;
					return (body.items ?? []).filter((item) => item.isReady).length;
				},
				{ timeout: 180_000, intervals: [1000, 2000, 5000] }
			)
			.toBe(5);

		// The hide of the third photo (the owner-level visibility
		// switch, the batch surface's own write): the row leaves the
		// caller's sight -- the honest skip's ground.
		const hide = await page.request.post('/api/v1/assets/batch', {
			data: { action: 'hide', ids: [ownCId] }
		});
		expect(hide.status()).toBe(200);
		const hideBody = (await hide.json()) as { affected?: number };
		expect(hideBody.affected).toBe(1);

		// The creator's aggregated timeline after the seed: their own
		// pair plus the shared pair (the union's shared arm -- the
		// directory album's grant), the hidden photo gone.
		const timeline = await page.request.get('/api/v1/assets/timeline');
		expect(timeline.status()).toBe(200);
		const timelineBody = (await timeline.json()) as TimelineBody;
		expect(timelineBody.items ?? []).toHaveLength(4);
	});

	test('scene 5: the every-member creation lands the memory through the dialog', async ({ page }) => {
		await page.goto('/memories');
		await page.getByRole('button', { name: 'New memory', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'New memory', exact: true });
		await expect(dialog).toBeVisible();

		// The creation shape: the title and the audience alone -- no
		// path field, no switches (a memory owns neither). The
		// audience stays the honest "Only you": the story grants
		// through the edit dialog's audience face.
		await dialog.getByLabel('Title').fill(MEMORY_TITLE);
		await expect(dialog.getByText('Only you')).toBeVisible();

		// The POST carries exactly the dialog's shape.
		const createPost = page.waitForRequest(
			(request) => request.method() === 'POST' && request.url().endsWith('/api/v1/memories')
		);
		await dialog.getByRole('button', { name: 'Create', exact: true }).click();
		const post = await createPost;
		expect(post.postDataJSON()).toEqual({ title: MEMORY_TITLE, audience: [] });

		// The all-landed creation holds no report: the dialog closes
		// directly (the acknowledgment face belongs to the honest
		// problems alone).
		const response = await post.response();
		expect(response?.status()).toBe(201);
		const createBody = (await response?.json()) as { skipped?: string[] };
		expect(createBody.skipped).toEqual([]);
		await expect(dialog).toHaveCount(0);

		// The SSE album.created echo converges the wall: the card
		// lands with the honest zero count and the manual origin
		// badge.
		const card = page.locator('a.memory-card').filter({ hasText: MEMORY_TITLE });
		await expect(card).toBeVisible();
		await expect(card.locator('.memory-card__count')).toHaveAttribute('aria-label', '0 photos');
		await expect(card).toContainText('Manual');

		// The created memory's id, for the later scenes' deep links
		// and requests.
		memoryId = (await card.getAttribute('href'))?.replace('/memories/', '') ?? '';
		expect(memoryId).not.toBe('');
	});

	test('scene 6: the wall-selection entry adds the selected photos through the picker', async ({ page }) => {
		await page.goto('/timeline');

		// The creator's timeline: their own pair plus the shared pair
		// (the union's shared arm), four tiles.
		await expect(page.locator('.tile')).toHaveCount(4);

		// The selection: two of their own photos and the shared one
		// -- the re-share ground's add arm (a photo another account
		// shared to the creator lands as a member exactly because
		// the creator currently sees it).
		for (const name of [OWN_A_NAME, OWN_B_NAME, VIA_SHARE_A_NAME]) {
			await page.getByRole('checkbox', { name: `Select: ${name}`, exact: true }).click();
		}
		const bar = page.getByRole('toolbar', { name: '3 selected' });
		await expect(bar).toBeVisible();

		// The batch bar's sparkles action opens the picker.
		await expect(bar.getByRole('button', { name: 'Add to memory', exact: true })).toBeVisible();
		await bar.getByRole('button', { name: 'Add to memory', exact: true }).click();

		// The picker: the captured-ids intro, the caller's own
		// memory as the one row.
		const dialog = page.getByRole('dialog', { name: 'Add to memory', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog.getByText('3 photos are selected — pick a memory, or create one.')).toBeVisible();
		const row = dialog.locator('.add-memory__row').filter({ hasText: MEMORY_TITLE });
		await expect(row).toBeVisible();
		await expect(row.locator('.count')).toHaveAttribute('aria-label', '0 photos');

		// The pick: the POST carries the ids captured at open, and
		// the answer is the honest counts-only report.
		const addPosted = page.waitForRequest(
			(request) => request.method() === 'POST' && request.url().endsWith(`/api/v1/albums/${memoryId}/items`)
		);
		await row.click();
		const add = await addPosted;
		expect(add.postDataJSON()).toEqual({ assetIds: [ownAId, ownBId, viaShareAId] });
		const addResponse = await add.response();
		expect(addResponse?.status()).toBe(200);
		expect(await addResponse?.json()).toEqual({ added: 3, skipped: 0 });

		// The row flips to the added state; no skip note renders (the
		// report had nothing to confess).
		await expect(row).toContainText('Added');
		await expect(dialog.locator('.add-memory__skip-note')).toHaveCount(0);

		// Done closes the picker and clears the wall's selection; no
		// convergence follows -- adding to a memory changes no
		// photo's state, and the timeline's tiles stay exactly as
		// they were.
		await dialog.getByRole('button', { name: 'Done', exact: true }).click();
		await expect(dialog).toHaveCount(0);
		await expect(bar).toHaveCount(0);
		await expect(page.locator('.tile')).toHaveCount(4);

		// The memory's count is the honest three.
		const detail = await page.request.get(`/api/v1/albums/${memoryId}`);
		expect(detail.status()).toBe(200);
		const detailBody = (await detail.json()) as { itemCount?: number };
		expect(detailBody.itemCount).toBe(3);
	});

	test('scene 7: the hidden wall answers the honest skip note', async ({ page }) => {
		// The hidden wall carries the hidden photo -- a row the
		// creator cannot currently see.
		await page.goto('/hidden');
		await expect(page.getByRole('link', { name: OWN_C_NAME, exact: true })).toBeVisible();

		// The same wall-selection entry rides the hidden wall's bar.
		await page.getByRole('checkbox', { name: `Select: ${OWN_C_NAME}`, exact: true }).click();
		const bar = page.getByRole('toolbar', { name: '1 selected' });
		await expect(bar).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Add to memory', exact: true })).toBeVisible();
		await bar.getByRole('button', { name: 'Add to memory', exact: true }).click();

		const dialog = page.getByRole('dialog', { name: 'Add to memory', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog.getByText('1 photo is selected — pick a memory, or create one.')).toBeVisible();

		// The pick POSTs the captured id; the write answers the
		// honest counts-only report -- the hidden row never lands.
		const addPosted = page.waitForRequest(
			(request) => request.method() === 'POST' && request.url().endsWith(`/api/v1/albums/${memoryId}/items`)
		);
		await dialog.locator('.add-memory__row').filter({ hasText: MEMORY_TITLE }).click();
		const add = await addPosted;
		expect(add.postDataJSON()).toEqual({ assetIds: [ownCId] });
		const addResponse = await add.response();
		expect(addResponse?.status()).toBe(200);
		expect(await addResponse?.json()).toEqual({ added: 0, skipped: 1 });

		// The skip-note honesty line: the write's own answer, with
		// the ground it enforced.
		await expect(dialog.getByText(SKIP_NOTE)).toBeVisible();

		await dialog.getByRole('button', { name: 'Done', exact: true }).click();
		await expect(dialog).toHaveCount(0);

		// The memory's count is unchanged: the row never landed.
		const detail = await page.request.get(`/api/v1/albums/${memoryId}`);
		expect(detail.status()).toBe(200);
		const detailBody = (await detail.json()) as { itemCount?: number };
		expect(detailBody.itemCount).toBe(3);
	});

	test('scene 8: the edit dialog grants the audience through the full-set PUT', async ({ page }) => {
		await page.goto(`/memories/${memoryId}`);
		await page.getByRole('button', { name: 'Edit memory', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Edit memory', exact: true });
		await expect(dialog).toBeVisible();

		// The memory's whole edit shape: the title field and the
		// audience alone -- no source row, no autonomy toggles (a
		// memory owns neither).
		await expect(dialog.getByLabel('Title')).toHaveValue(MEMORY_TITLE);
		await expect(dialog.locator('.album-form__source')).toHaveCount(0);
		await expect(dialog.locator('[data-testid="album-timeline-toggle"]')).toHaveCount(0);
		await expect(dialog.locator('[data-testid="album-deletable-toggle"]')).toHaveCount(0);

		// The audience face: the honest "Only you", the picker (a
		// portaled floating layer at page level) offering the two
		// other accounts, the sharee's option picked, the chip
		// landing.
		await expect(dialog.getByText('Only you')).toBeVisible();
		await dialog.getByRole('button', { name: 'Add', exact: true }).click();
		const picker = page.getByRole('listbox', { name: 'Candidate members' });
		// The rows name the people (the identity rule: the display
		// name leads, never the login address).
		await expect(picker.getByRole('option', { name: SHAREE_NAME })).toBeVisible();
		await picker.getByRole('option', { name: SHAREE_NAME }).click();
		await expect(dialog.locator('.audience-field__chip')).toContainText(SHAREE_NAME);
		// The picker stays open (the instance's third account keeps
		// the candidate pool alive); the layer's own Esc closes it
		// alone -- the dialog behind it stays.
		await page.keyboard.press('Escape');
		await expect(picker).toHaveCount(0);

		// The save writes the audience PUT alone (the unchanged title
		// writes nothing -- the page diffs against the live detail),
		// and the PUT carries the full set.
		const patchUrls: string[] = [];
		page.on('request', (request) => {
			if (request.method() === 'PATCH' && request.url().includes(`/api/v1/albums/${memoryId}`)) {
				patchUrls.push(request.url());
			}
		});
		const audiencePut = page.waitForRequest(
			(request) => request.method() === 'PUT' && request.url().endsWith(`/api/v1/albums/${memoryId}/audience`)
		);
		await dialog.getByRole('button', { name: 'Save changes', exact: true }).click();
		const put = await audiencePut;
		expect(put.postDataJSON()).toEqual({ audience: [shareeUserId] });
		const putResponse = await put.response();
		expect(putResponse?.status()).toBe(200);

		await expect(dialog).toHaveCount(0);
		expect(patchUrls).toEqual([]);
	});
});

// The sharee half of the story: their own sign-in through a fresh
// context, the read-only memory view, and the state hand-off.
test('scene 9: the sharee signs in and the shared memory reads view-only', async ({ browser }) => {
	const context = await browser.newContext({ locale: 'en-US', timezoneId: 'UTC' });
	const page = await context.newPage();

	await page.goto('/login');
	await page.getByLabel('Email').fill(SHAREE_EMAIL);
	await page.getByLabel('Password').fill(SHAREE_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);

	// The memories wall is ONE list: the card carries the owner's NAME
	// beside the read-only chip and the honest count (the live members
	// the memory's owner currently sees -- all three still pass the
	// owner's sight). The address never renders.
	await page.goto('/memories');
	await expect(page.getByRole('heading', { level: 2 })).toHaveCount(0);
	const card = page.locator('a.memory-card').filter({ hasText: MEMORY_TITLE });
	await expect(card).toBeVisible();
	await expect(card.locator('.memory-card__count')).toHaveAttribute('aria-label', '3 photos');
	await expect(card.locator('.memory-card__badge--owner')).toHaveText(`Shared by ${CREATOR_NAME}`);
	await expect(card.locator('.memory-card__badge--owner')).not.toContainText(CREATOR_EMAIL);
	await expect(card.locator('.memory-card__badge--readonly')).toHaveText('Read-only');

	// The detail: the member bar carries the owner attribution and
	// the read-only badge with NO action at all, and the wall is
	// pure navigation -- no check ever renders.
	await card.click();
	await expect(page).toHaveURL(new RegExp(`/memories/${memoryId}$`));
	await expect(page.locator('h1.memory-bar__title')).toHaveText(MEMORY_TITLE);
	await expect(page.locator('.memory-bar__count')).toHaveAttribute('aria-label', '3 photos');
	await expect(page.locator('.memory-bar__owner')).toHaveText(`Shared by ${CREATOR_NAME}`);
	await expect(page.locator('.memory-bar__badge')).toContainText('Read-only');
	for (const label of [
		'Edit memory',
		'Change cover',
		'Delete memory',
		'Rescan',
		'Remove album',
		'Keep out of timeline'
	]) {
		await expect(page.getByRole('button', { name: label, exact: true })).toHaveCount(0);
	}
	await expect(page.locator('.tile')).toHaveCount(3);
	await expect(page.getByRole('checkbox')).toHaveCount(0);

	await context.storageState({ path: shareeStatePath });
	await context.close();
});

// The closing scenes: the creator's session replays; the sharee's
// and the admin's contexts are created inside the scenes that need
// them (the held surfaces and the source-side writes).
test.describe('the closing scenes', () => {
	test.use({ storageState: creatorStatePath });

	test('scene 10: the source revocation cascades into the open sharee view', async ({ page, browser }) => {
		// The held surfaces, opened BEFORE the write so their
		// convergence is the live kind: the sharee's memories wall
		// (the shared card with its count and its collage), the
		// creator's own memories wall (the own card), and the
		// creator's timeline (the union's shared arm).
		const shareeContext = await browser.newContext({
			locale: 'en-US',
			timezoneId: 'UTC',
			storageState: shareeStatePath
		});
		const shareePage = await shareeContext.newPage();
		await shareePage.goto('/memories');
		const shareeCard = shareePage.locator('a.memory-card').filter({ hasText: MEMORY_TITLE });
		await expect(shareeCard).toBeVisible();
		await expect(shareeCard.locator('.memory-card__count')).toHaveAttribute('aria-label', '3 photos');
		await expect(shareeCard.locator('img')).toHaveCount(3);

		const creatorList = await page.context().newPage();
		await creatorList.goto('/memories');
		const ownCard = creatorList.locator('a.memory-card').filter({ hasText: MEMORY_TITLE });
		await expect(ownCard).toBeVisible();
		await expect(ownCard.locator('.memory-card__count')).toHaveAttribute('aria-label', '3 photos');

		const creatorTimeline = await page.context().newPage();
		await creatorTimeline.goto('/timeline');
		await expect(creatorTimeline.locator('.tile')).toHaveCount(4);

		// The revocation: the source album's owner empties its
		// audience -- the creator's sight of the shared photo dies
		// with the grant (the request rides the admin's own
		// context).
		const adminContext = await browser.newContext({
			locale: 'en-US',
			timezoneId: 'UTC',
			storageState: adminStatePath
		});
		const revoke = await adminContext.request.put(`/api/v1/albums/${sourceAlbumId}/audience`, {
			data: { audience: [] }
		});
		expect(revoke.status()).toBe(200);
		await adminContext.close();

		// The sharee's held wall converges LIVE (the album event
		// refetches the open page, no reload): the shared card's
		// count drops and the collage loses the re-shared photo --
		// the shared view recomputes against the owner's current
		// sight, with no snapshot residue.
		await expect(shareeCard.locator('.memory-card__count')).toHaveAttribute('aria-label', '2 photos');
		await expect(shareeCard.locator('img')).toHaveCount(2);

		// The sharee's detail wall, the FRESH visit: the count and
		// the tiles agree -- the walk (the timeline endpoint's album
		// narrow) drops the revoked photo too, so the memory's own
		// face never renders a grounded count beside the revoked
		// photo's tile. (A held detail page would not converge here:
		// its event filter keys on the memory's own id while the
		// revocation's event names the source album, so the fresh
		// visit -- the recompute-on-read kind -- is the honest
		// witness.)
		const shareeDetail = await shareeContext.newPage();
		await shareeDetail.goto(`/memories/${memoryId}`);
		await expect(shareeDetail.locator('h1.memory-bar__title')).toHaveText(MEMORY_TITLE);
		await expect(shareeDetail.locator('.memory-bar__count')).toHaveAttribute('aria-label', '2 photos');
		await expect(shareeDetail.locator('.tile')).toHaveCount(2);
		await expect(shareeDetail.getByRole('link', { name: VIA_SHARE_A_NAME, exact: true })).toHaveCount(0);
		await shareeDetail.close();

		// The creator's own card keeps the plain live membership:
		// the row stays (the set is the owner's to tidy).
		await expect(ownCard.locator('.memory-card__count')).toHaveAttribute('aria-label', '3 photos');

		// The creator's held timeline drops the shared pair (the
		// grant's death ends the shared arm) -- and the memory's row
		// never holds a photo on the feed.
		await expect(creatorTimeline.locator('.tile')).toHaveCount(2, { timeout: 30_000 });
		await expect(creatorTimeline.getByRole('link', { name: OWN_A_NAME, exact: true })).toBeVisible();
		await expect(creatorTimeline.getByRole('link', { name: OWN_B_NAME, exact: true })).toBeVisible();
		await expect(creatorTimeline.getByRole('link', { name: VIA_SHARE_A_NAME, exact: true })).toHaveCount(0);
		await expect(creatorTimeline.getByRole('link', { name: VIA_SHARE_B_NAME, exact: true })).toHaveCount(0);

		await shareeContext.close();
		await creatorList.close();
		await creatorTimeline.close();
	});

	test("scene 11: the memory never feeds the creator's timeline", async ({ page }) => {
		// The fresh view agrees with the held one: exactly the
		// creator's own pair, each photo exactly once -- the
		// memory's three rows inject nothing into the aggregate,
		// and the revoked photo's row never kept it on the feed.
		await page.goto('/timeline');
		await expect(page.locator('.tile')).toHaveCount(2);
		await expect(page.getByRole('link', { name: OWN_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: OWN_B_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: VIA_SHARE_A_NAME, exact: true })).toHaveCount(0);
		await expect(page.getByRole('link', { name: OWN_C_NAME, exact: true })).toHaveCount(0);

		// The server's own answer: exactly the two own photos, no
		// duplicates, no memory-injected rows.
		await expect
			.poll(async () => {
				const timeline = await page.request.get('/api/v1/assets/timeline');
				expect(timeline.status()).toBe(200);
				const body = (await timeline.json()) as { items?: Array<{ id?: string }> };
				return (body.items ?? []).length;
			})
			.toBe(2);

		// And the memory itself still holds its three rows (the
		// owner's plain live membership -- the dead-source row
		// stays until the owner tidies it, the next scene's walk).
		const detail = await page.request.get(`/api/v1/albums/${memoryId}`);
		expect(detail.status()).toBe(200);
		const detailBody = (await detail.json()) as { itemCount?: number };
		expect(detailBody.itemCount).toBe(3);
	});

	test('scene 12: the removal curates the remove-only subset and the photo stays', async ({ page, browser }) => {
		await page.goto(`/memories/${memoryId}`);

		// The owner's bar: the edit, cover and delete arms alone --
		// no rescan and no un-ingest anywhere (a memory owns no
		// source to walk and no files to purge).
		await expect(page.locator('h1.memory-bar__title')).toHaveText(MEMORY_TITLE);
		await expect(page.locator('.memory-bar__count')).toHaveAttribute('aria-label', '3 photos');
		for (const label of ['Edit memory', 'Change cover', 'Delete memory']) {
			await expect(page.getByRole('button', { name: label, exact: true })).toBeVisible();
		}
		for (const label of ['Rescan', 'Remove album', 'Add to memory']) {
			await expect(page.getByRole('button', { name: label, exact: true })).toHaveCount(0);
		}

		// The owner's wall keeps the dead-source row (the plain live
		// membership): the tile is still here to select and tidy.
		await expect(page.locator('.tile')).toHaveCount(3);
		await expect(page.getByRole('link', { name: VIA_SHARE_A_NAME, exact: true })).toBeVisible();

		// The selection rises the REMOVE-ONLY subset: the danger
		// minus action alone -- no favorite, no hide, no trash, no
		// tag, no sparkles.
		await page.getByRole('checkbox', { name: `Select: ${VIA_SHARE_A_NAME}`, exact: true }).click();
		const bar = page.getByRole('toolbar', { name: '1 selected' });
		await expect(bar).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Remove from memory', exact: true })).toBeVisible();
		for (const label of [
			'Favorite',
			'Unfavorite',
			'Hide',
			'Unhide',
			'Trash',
			'Restore',
			'Add tag',
			'Add to memory'
		]) {
			await expect(bar.getByRole('button', { name: label, exact: true })).toHaveCount(0);
		}

		// The write: the membership removal with its honest report.
		const removePosted = page.waitForRequest(
			(request) => request.method() === 'DELETE' && request.url().endsWith(`/api/v1/albums/${memoryId}/items`)
		);
		await bar.getByRole('button', { name: 'Remove from memory', exact: true }).click();
		const remove = await removePosted;
		expect(remove.postDataJSON()).toEqual({ assetIds: [viaShareAId] });
		const removeResponse = await remove.response();
		expect(removeResponse?.status()).toBe(200);
		expect(await removeResponse?.json()).toEqual({ removed: 1, skipped: 0 });

		// The membership event converges the held page: the tile
		// leaves, the bar's count drops, the selection clears.
		await expect(page.getByRole('link', { name: VIA_SHARE_A_NAME, exact: true })).toHaveCount(0);
		await expect(page.locator('.tile')).toHaveCount(2);
		await expect(page.locator('.memory-bar__count')).toHaveAttribute('aria-label', '2 photos');
		await expect(bar).toHaveCount(0);

		// The photo itself stays: the library row lives on, its
		// owner still reads it (the request rides the admin's own
		// context -- the photo is the admin's, and the creator's
		// sight of it died with the source grant).
		const adminContext = await browser.newContext({
			locale: 'en-US',
			timezoneId: 'UTC',
			storageState: adminStatePath
		});
		const survivor = await adminContext.request.get(`/api/v1/assets/${viaShareAId}`);
		expect(survivor.status()).toBe(200);
		await adminContext.close();
	});

	test("scene 13: a memory id on the album route lands on the memory's own face", async ({ page }) => {
		// The albums wall never carries the memory (its home is the
		// memories wall): the creator's wall holds exactly the sync
		// default -- the source grant died in scene 10, and the
		// memory never leaked here.
		await page.goto('/albums');
		await expect(page.locator('a.album-card').filter({ hasText: MEMORY_TITLE })).toHaveCount(0);
		await expect(page.locator('a.album-card')).toHaveCount(1);

		// The deep link: the album detail's read answers the memory
		// type and the page hands the route over to the memory's own
		// face -- one face per concept.
		await page.goto(`/albums/${memoryId}`);
		await expect(page).toHaveURL(new RegExp(`/memories/${memoryId}$`));
		await expect(page.locator('h1.memory-bar__title')).toHaveText(MEMORY_TITLE);
		await expect(page.locator('.memory-bar__count')).toHaveAttribute('aria-label', '2 photos');
	});

	test('scene 14: the delete takes the references and the audience, never the photos', async ({ page, browser }) => {
		// The sharee's held memories wall: the live convergence
		// witness of the audience's death.
		const shareeContext = await browser.newContext({
			locale: 'en-US',
			timezoneId: 'UTC',
			storageState: shareeStatePath
		});
		const shareePage = await shareeContext.newPage();
		await shareePage.goto('/memories');
		const shareeCard = shareePage.locator('a.memory-card').filter({ hasText: MEMORY_TITLE });
		await expect(shareeCard).toBeVisible();
		await expect(shareeCard.locator('.memory-card__count')).toHaveAttribute('aria-label', '2 photos');

		// The delete: the confirm states the light cascade -- the
		// memory's references and its audience die, the photos
		// themselves stay.
		await page.goto(`/memories/${memoryId}`);
		await page.getByRole('button', { name: 'Delete memory', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Delete this memory?', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText('goes, together with its photo list and its shares');
		await expect(dialog).toContainText('The photos themselves stay in your library');

		const deleteAnswered = page.waitForResponse(
			(response) =>
				response.request().method() === 'DELETE' && response.url().endsWith(`/api/v1/albums/${memoryId}`)
		);
		await dialog.getByRole('button', { name: 'Delete memory', exact: true }).click();
		const answer = await deleteAnswered;
		expect(answer.status()).toBe(204);

		// Success leaves for the memories wall: the creator's own
		// section is empty again (the fully-empty state's copy).
		await expect(page).toHaveURL(/\/memories$/);
		await expect(page.locator('a.memory-card')).toHaveCount(0);
		await expect(page.getByText('No memories yet', { exact: true })).toBeVisible();

		// The sharee's held wall loses the shared card LIVE (the
		// audience died with the memory) and settles at its own
		// fully-empty state.
		await expect(shareeCard).toHaveCount(0, { timeout: 30_000 });
		await expect(shareePage.getByText('No memories yet', { exact: true })).toBeVisible();

		// The photos stay: the creator's own pair is still exactly
		// the timeline -- the library kept every row the memory
		// pointed at.
		await page.goto('/timeline');
		await expect(page.locator('.tile')).toHaveCount(2);
		await expect(page.getByRole('link', { name: OWN_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: OWN_B_NAME, exact: true })).toBeVisible();

		// The aftermath: the memory answers the single-shape 404 to
		// the creator and to the sharee alike, like an unknown id.
		const gone = await page.request.get(`/api/v1/albums/${memoryId}`);
		expect(gone.status()).toBe(404);
		const shareeGone = await shareeContext.request.get(`/api/v1/albums/${memoryId}`);
		expect(shareeGone.status()).toBe(404);

		// The dead share's DEEP LINK: the address the sharee held
		// (yesterday's shared memory, today gone) renders the one
		// unreachable card -- no title, no count, no name -- the same
		// face an address the app never had gets.
		const deadLink = await shareeContext.newPage();
		await deadLink.goto(`/memories/${memoryId}`);
		await expect(deadLink.locator('.unreachable-card')).toBeVisible();
		await expect(deadLink.locator('.unreachable-card')).not.toContainText(MEMORY_TITLE);
		await expect(deadLink.locator('.memory-bar__title')).toHaveCount(0);
		await expect(deadLink.locator('.memory-bar__count')).toHaveCount(0);
		await expect(deadLink.getByRole('link', { name: 'Back to the timeline' })).toBeVisible();
		await deadLink.close();
		await shareeContext.close();
	});
});
