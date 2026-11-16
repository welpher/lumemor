// The albums-as-sources story of the web app (reshaped by the
// album-autonomy rework and the folder live sync), walked end to end
// against the real compose stack:
// an album IS a source. The create dialog picks an EXISTING directory
// on the server's filesystem (referenced in place -- zero copy, zero
// move), the CREATION scans it (the album's first scan is the
// creation's own -- the wall never waits for a manual rescan), and
// the periodic folder sync then keeps the album's membership equal to
// the folder: a file dropped into the source reaches the wall, a file
// removed from it leaves. The audience is an album property (the edit
// dialog's audience face, the full-set PUT), and the member's side of
// the share is the read-only view. THE AUTONOMY BLOCK rides BOTH
// dialogs (the lock's retirement reshaped it): the timeline-exclusion
// toggle everywhere, the deletability row exactly where the source's
// writable fact holds -- the create dialog learns the fact from the
// picked completion row, the edit dialog from the stored album -- and
// NO lock affordance anywhere (the setting gates the delete
// affordance, the operating system gates the execution, the trash
// catches everything). Both pickers (the path completion
// and the audience candidates) are PORTALED floating layers (the
// floating-layer revision) -- anchored under their triggers,
// living in the document outside the dialogs' DOM, so the dialogs
// never change height for them. The stack's lifecycle is owned by the
// caller (the acceptance scripts' chaining pattern, or one fresh
// stack per spec -- the first-run bootstrap below expects its 201);
// the spec itself only talks to http://localhost:8244.
//
// THE STACK ARRANGEMENT (the honest source fixtures, documented):
// the repository's compose file binds TWO fixture trees into the
// lumemor container, and the binds themselves are the boundary
// (the mount-boundary revision: the mount-root allowlist retired
// whole -- the operator controls the container's view, the
// completion's domain is the container's REAL filesystem, and the
// managed data tree never lists).
//   - web/e2e/fixtures/m3-source, bound READ-ONLY at /mnt/e2e-source:
//     the story's first album points at coastal-week under it -- the
//     two face-portrait JPEGs the upload stories seed, arriving
//     through the source path this time. The read-only bind pins the
//     zero-copy ingest by construction AND answers the creation-time
//     write probe with the honest false fact (a read-only
//     directory is a legitimate album whose photos cannot be deleted
//     through the application -- feedback, never a failure).
//   - web/e2e/fixtures/m3-source-rw, bound WITHOUT :ro at
//     /mnt/e2e-source-rw: the probe's true arm and the deletability
//     chain (see that fixture's README). The bootstrap below seeds
//     its photos per run -- genuinely distinct bytes (the
//     cross-person portraits, never the read-only fixture's pair: the
//     scan keys rows by source POSITION, so identical bytes elsewhere
//     would land a second row of one content and blur the two
//     stories' identities) -- and the trash scene's restore returns
//     them, so the tree self-heals; the removal scene later asserts
//     the same files byte for byte on the host after the un-ingest.
// The models ride the SAME storage bind mount the chained
// acceptances downloaded into -- ./lumemor persists across stack
// lifetimes, so the download run the bootstrap triggers is warm (a
// digest re-verification); cold (CI's first run), it is the real
// transfer, bounded like every e2e story's cold bound.
//
// THE TWO-ACCOUNT DANCE (the sharing story's shape): the first-run
// administrator owns the library and the albums (mounting an
// existing photo directory is a configuration act, so the creation
// entry and the completion are the ADMINISTRATOR'S alone); a member
// account, created through the admin API in the suite's bootstrap,
// becomes the audience. The story's scenes sign each account in
// through the login form in SEPARATE browser contexts -- the admin's
// storage state first, then the member's own sign-in scene (the two
// sessions never share a context) -- and the member's scenes assert
// everything the audience grant gives: the wall's shared card (the
// owner's name beside the read-only chip), the read-only detail, the timeline's inclusion of the shared items,
// the viewer drawer's identity-aware read-only state, and the
// creation guard's refusal matrix (the member's wall carries no
// create affordance; their create-album attempt and their completion
// call both answer AUTH_FORBIDDEN).
//
// THE REMOVAL rides the same writable album: a
// directory album's removal is the UN-INGEST -- the album row, its
// membership and its audience die, and every member no other album
// references purges from the library whole (the asset row, the
// faces with their cluster sweep, the embedding, the OCR text, the
// per-user states, the generated thumbnail/encoded files) while the
// SOURCE FILES ARE NEVER TOUCHED -- the fixture dir is checked on
// the host after, byte for byte. The write's confirm dialog states
// the pinned semantics before the irreversible call, the sync album
// folds into the single-shape 404 (no album resource), the sharee's
// attempt answers the honest 403, and the published event family
// (album.deleted + asset.deleted) converges the held-open surfaces
// live: the member's timeline drops the shared tiles, the search
// wall empties, the named person whose faces died with the purge
// falls off the people page.
//
// THE SCENES (the albums and autonomy surface, in order):
//
//  1. The administrator signs in through the form and lands on the
//     albums wall holding exactly the pinned "My photos" card (the
//     default sync album -- a fresh account's wall is never empty;
//     the count is honest at zero).
//  2. The library bootstrap: the model group's download run is driven
//     to succeeded through the admin API (the scan's pipeline needs
//     it; this story uploads nothing -- both albums' members arrive
//     through their sources).
//  3. The completion's domain: the create dialog's path field asks
//     the system-wide completion -- the empty prefix lists the
//     filesystem root itself, the trailing-separator drill lists the
//     container's /mnt children (both fixture mounts visible: the
//     boundary is the compose bind), and the managed data tree NEVER
//     lists (the storage root's own directory answers its config
//     sibling alone). The closing discipline: the list's Esc closes
//     the list alone, the modal's Esc owns the next stroke.
//  4. The read-only album's creation: the completion lists
//     coastal-week for a typed prefix, Enter completes it, the
//     audience stays the honest "Only you", and the create dialog's
//     autonomy block holds its read-only shape -- the
//     timeline-exclusion toggle present and checked (the feed's
//     default), NO deletability row (the picked row's writable fact
//     is false: no dead affordance). The POST carries exactly what
//     the dialog holds (title, rootPath, audience, inTimeline true
//     -- and no allowAppDeletion key). The write probe
//     answers sourceWritable false -- the creation FEEDBACK is the
//     toast carrying the read-only note (both sentences, one
//     acknowledgment; the dialog closes on the landed write). The
//     card lands, and the CREATION'S
//     OWN SCAN converges it to the folder's two photos -- no rescan
//     is pressed anywhere in this scene.
//  5. The creation scan's result: the album page already holds the
//     two tiles (the scan the creation enqueued), both ready through
//     the same pipeline fan-out an upload walks; then the manual
//     rescan re-runs the same directory idempotently -- the trigger
//     POSTs the run and the aria-live progress line converges to
//     complete through the job events, the membership unchanged.
//     The bar's count holds at a narrow viewport too (it rides the
//     title's truncation instead of vanishing).
//  6. The task panel: the rescan's job_runs row lists with its own
//     label (the contract's JobRunKind carries album_rescan -- the
//     row must never fall to the panel's generic failure).
//  7. The edit dialog: the autonomy block -- the caller's own
//     timeline-exclusion toggle rides the dialog and the LOCK
//     TOGGLE IS GONE (the lock retired whole: no lock affordance
//     renders anywhere), while the deletability row DOES NOT render
//     (the read-only source carries no row at all -- no dead
//     affordance); the audience face then grants the member the view
//     (the source row fixed, the picker, the drain collapse, the
//     full-set PUT, the unchanged title writing nothing, the chip
//     reseed).
//  8. The member perspective: the member signs in through their own
//     form, the wall's one list carries the album's shared card
//     (the owner's name, the read-only chip),
//     the wall renders NO creation affordance (the guard's visual
//     half), the member's create-album attempt and completion call
//     both answer AUTH_FORBIDDEN (the refusal matrix's new arms),
//     the shared detail renders read-only (one action in the bar --
//     the exclusion toggle, the member's own -- with no owner
//     affordance beside it), the member's wall curates the per-user
//     favorite alone (the check, the bar's two buttons, the silent
//     write's flip), the toggle excludes the album from the member's
//     own timeline and keeping it in brings the shared tiles back
//     (the union's shared arm).
//  9. The viewer's identity-aware drawer: a shared photo's info
//     drawer renders the reads and the per-user favorite without any
//     owner-only write affordance.
// 10. The writable album's creation (the probe's true arm): the
//     completion's exact-match row for the writable fixture root, the
//     same pinned triple, and the create dialog's autonomy block in
//     its writable shape -- the deletability row RENDERS (checked on)
//     from the picked row's writable fact, and the timeline-exclusion
//     toggle is UNCHECKED: the album is created kept out of its
//     creator's timeline. The POST carries both switches
//     (allowAppDeletion true, inTimeline false), the probe answers
//     sourceWritable true -- the writable, all-landed creation closes
//     the dialog DIRECTLY and the house toast confirms it. The card
//     lands.
// 11. The writable album's creation scan converges the seeded pair,
//     and the edit dialog's autonomy block holds its writable shape --
//     the deletability row rendered (checked on) beside the
//     exclusion toggle (reading unchecked: the creation landed the
//     exclusion), the lock toggle gone.
// 12. The create-time exclusion's convergence: the administrator's
//     aggregated timeline holds exactly the read-only album's pair
//     (created with the feed's default) and NEVER the writable
//     album's pair (created excluded) -- the preference landed in
//     the creation itself scopes the aggregate, never the album's
//     own view (scene 11's membership is complete either way).
// 13. The deletability chain: the deletable wall curates a
//     DELETE-ONLY selection (the hover check, the batch bar offering
//     Trash alone), the trash moves the file out of the source, the
//     album's live membership converges to the survivor, the trash
//     view carries the row, and the restore returns the file -- the
//     album's fresh answer carries both photos again (the fixture
//     healed).
// 13b. The folder live sync on one held-open album wall: a NEW photo
//     written straight into the bind-mounted source appears on the
//     wall with no manual action (the periodic snapshot diff enqueues
//     the album's own scan inside the shortened window), and REMOVING
//     the file takes the tile away again through the pinned orphan
//     semantics -- the row is soft-deleted, never the trash view's
//     (the file was never the server's). The fixture heals itself.
// 14. The removal's stage: the audience grant arms the member (the
//     full-set PUT the edit dialog drives), the two portrait pairs'
//     clusters converge (the cross-person separation is the
//     identification), the writable pair's cluster is NAMED through
//     the unnamed region's dialog -- the people card whose faces sit
//     wholly on the purged assets, the removal's people-convergence
//     witness -- and the member's shared side is asserted before the
//     write: the wall's shared card, the four-tile timeline, and the
//     member's own removal attempt answering the honest 403 with
//     nothing landed.
// 15. The removal itself: the detail bar's remove action opens the
//     confirm whose body states the pinned semantics, the confirm
//     drives the DELETE, the wall loses the card -- and every
//     held-open surface converges on the events without a reload:
//     the member's timeline drops the shared tiles (the read-only
//     pair stays), the member's wall loses the shared card (the
//     audience died with the album), the admin's search wall
//     empties, the people page drops the named person. The scene
//     closes on the zero-contact pin's host-side proof: the source
//     files are byte for byte the seeds.
// 16. The aftermath: the album answers 404 to the read and to the
//     repeat delete (the single-shape refusal), the sync album's own
//     removal attempt folds into the same 404 with the default card
//     staying on the wall, the aggregated timeline is the read-only
//     pair's alone, the search answers the purged name with nothing
//     and the survivor's name with the survivor, the named people
//     listing is empty, the unnamed region carries exactly the
//     read-only pair's cluster, and the purged cluster's detail is a
//     404 like an unknown id.
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story --
// see playwright.config.ts). Scenes 1-7, 10-13 and 14-16 share the
// admin's browser session (scene 1 signs in and saves the storage
// state); scene 8 signs the member in through a fresh context and
// saves their own; scene 9 replays the member's, and the removal
// scenes replay it again for the member's side of the grant (the
// contexts are created inside those scenes). Each scene navigates
// explicitly to its starting route, so a failure report names the
// exact scene that broke.

import { copyFile, mkdir, readFile, readdir, rm } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's two accounts: the first-run administrator (the setup
// API's, the albums' owner) and the member the administrator
// provisions through the admin API in the suite's bootstrap. The
// stack is always fresh, so the fixed identities are deterministic.
const ADMIN_EMAIL = 'admin@m3c-e2e.invalid';
const ADMIN_NAME = 'Mei Chen';
const ADMIN_PASSWORD = 'correct-horse-battery';
const MEMBER_EMAIL = 'grace@m3c-e2e.invalid';
const MEMBER_NAME = 'Grace Hopper';
const MEMBER_PASSWORD = 'correct-horse-battery';

// The read-only source fixture (the stack arrangement above): the
// bind the compose file mounts :ro, and the directory under it the
// story's first album references in place.
const MOUNT_ROOT = '/mnt/e2e-source';
const SOURCE_DIR = '/mnt/e2e-source/coastal-week';

// The writable sibling (the stack arrangement above): the bind
// without :ro, and the seeded photos the deletability chain walks.
// The seed bytes are the cross-person portrait pair -- genuinely
// distinct from the read-only fixture's pair, so the scan's
// position-keyed rows keep the two stories' identities apart.
const RW_MOUNT_ROOT = '/mnt/e2e-source-rw';
const RW_SEED_A = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other.jpg');
const RW_SEED_B = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other-second.jpg');
const RW_SOURCE_HOST_DIR = path.join(specDir, 'fixtures/m3-source-rw');

// The folder sync scene's dropped photo: a plain PNG with no faces
// (the story's people assertions count the portrait pairs' faces
// alone), and bytes no other scene of this stack ever ingests -- the
// tile's name is the file's, so a collision would blur the counts the
// folder-sync scene reads.
const FOLDER_SYNC_PHOTO = path.join(specDir, 'fixtures/tile-256.png');
const PHOTO_SYNC_NAME = 'IMG_SYNC_0001.png';

// The cover story's source: the repository's four-photo trip fixture,
// bound read-only at the mount root. The album card holds four frames,
// so the cover scene needs a collection of exactly that size -- the
// scan ingests the four files in place (no other story in this stack
// references them, so the card's four frames are the album's own).
const COVER_SOURCE_DIR = '/mnt/e2e-source/trip-2026';
const COVER_ALBUM_TITLE = 'Trip frames 2026';

// The two fixture photos the read-only album gathers (the source
// tree's own names -- the story asserts them as the tiles' links).
const PHOTO_A_NAME = 'IMG_E4_0001.jpg';
const PHOTO_B_NAME = 'IMG_E4_0002.jpg';

// The two seeded photos the writable album gathers (the fixture
// README's arrangement; the trash scene moves the first and the
// restore returns it).
const PHOTO_RW_A_NAME = 'IMG_E6_0001.jpg';
const PHOTO_RW_B_NAME = 'IMG_E6_0002.jpg';

// The story's album titles.
const ALBUM_TITLE = 'Coastal week 2026';
const RW_ALBUM_TITLE = 'Field day 2026';

// The mixed count scene's seeds: one photo and one video uploaded into
// the sync album (the only album uploads land in). Two live members,
// ONE photo -- the numbers the card's and the bar's "N photos" must
// part ways on.
const MIXED_PHOTO_NAME = 'IMG_COUNT_PHOTO.jpg';
const MIXED_VIDEO_NAME = 'CLIP_COUNT_VIDEO.mp4';
const MIXED_PHOTO_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/golden-large.jpg');
const MIXED_VIDEO_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/tiny.mp4');

// The download bound: the cold upstream transfer of the group (about
// 351 MB) is minutes, the warm re-verification is seconds; both
// allow the cold bound.
const DOWNLOAD_TIMEOUT_MS = 900_000;

// The creation-time scan's bound: the creation enqueues the album's
// first scan, and the wall converges as its ingest lands -- the walk
// plus the pipeline's first jobs on a two-photo source is seconds on
// a warm stack, and the cold bounds above already allow minutes.
const CREATION_SCAN_TIMEOUT_MS = 120_000;

// The folder live sync's bound: the periodic tick re-reads every
// directory album's source and enqueues the album's scan when it
// changed. The harness shortens the interval (the compose stack's
// LUMEMOR_FOLDER_SYNC_INTERVAL, 3s in the e2e run), so one change
// needs a tick plus the scan's own ingest -- well inside this bound
// even with the pipeline's cold steps.
const FOLDER_SYNC_TIMEOUT_MS = 120_000;

// The storage-state files scene 1 (the admin) and scene 8 (the
// member) write and the later scenes replay (git-ignored, recreated
// per run).
const adminStatePath = `${specDir}/.auth/m3c-admin-state.json`;
const memberStatePath = `${specDir}/.auth/m3c-member-state.json`;

// The read-only album's id, captured by the create scene through the
// wall card so the later scenes can deep-link deterministically.
let albumId = '';

// The writable album's id, captured the same way by its create scene.
let rwAlbumId = '';

// The cover scene's album id (created through the API -- the create
// dialog's flow is scene 4's and 10's subject, the cover's is this
// one's) and the four member ids it gathers, in the cover's pick
// order.
let coverAlbumId = '';
let coverMemberIds: string[] = [];

// The member's user id, captured by the bootstrap so the audience
// write's request shape can be pinned against the real id.
let memberUserId = '';

// The two face clusters' identities, captured by scene 14 through
// the unnamed listing and the cluster details (the faces' file
// names tell the cross-person pairs apart): the writable pair's
// cluster is the removal's casualty (named, then dissolved with
// its label row), the read-only pair's the survivor. The
// representative face's id addresses the writable cluster's
// unnamed card (the card renders the representative's crop).
let rwClusterId = '';
let rwRepFaceId = '';
let roClusterId = '';

test.beforeAll(async ({ playwright }) => {
	// The writable fixture's seeding (the arrangement its README
	// documents): the photos are never committed -- the bootstrap
	// copies them in, the trash scene moves one into the managed
	// trash area, and the restore returns it, so the next run's
	// re-seed is a no-op copy. The sweep first is the determinism:
	// the directory persists across every run on the host (this
	// spec's two seeds, the acceptance chain's one), and a stale
	// photo would leave the rescan's membership counting leftovers.
	// The bind is live, so the container sees the files the moment
	// they land.
	await mkdir(RW_SOURCE_HOST_DIR, { recursive: true });
	for (const entry of await readdir(RW_SOURCE_HOST_DIR)) {
		if (entry !== 'README.md') {
			await rm(path.join(RW_SOURCE_HOST_DIR, entry), { force: true });
		}
	}
	await copyFile(RW_SEED_A, path.join(RW_SOURCE_HOST_DIR, PHOTO_RW_A_NAME));
	await copyFile(RW_SEED_B, path.join(RW_SOURCE_HOST_DIR, PHOTO_RW_B_NAME));

	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance; the
	// signed-in admin then provisions the member account through the
	// admin surface (the two-account dance's setup). The browser
	// scenes then only exercise the web app's own surfaces.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: ADMIN_NAME, email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);

	const login = await api.post('/api/v1/auth/login', {
		data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(login.status()).toBe(200);

	const member = await api.post('/api/v1/admin/users', {
		data: { name: MEMBER_NAME, email: MEMBER_EMAIL, password: MEMBER_PASSWORD }
	});
	expect(member.status()).toBe(201);
	const memberBody = (await member.json()) as { id?: string };
	memberUserId = memberBody.id ?? '';
	expect(memberUserId).not.toBe('');
	await api.dispose();
});

test('scene 1: the administrator signs in and lands on the pinned My photos card', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Falbums');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();

	// The post-login target round-trips onto the albums wall.
	await expect(page).toHaveURL(/\/albums$/);
	await expect(page).toHaveTitle('Albums · lumemor');

	// The wall is never empty on a fresh account: the pinned default
	// album leads the wall with the localized display name (the
	// stored internal title never renders) and the honest zero count,
	// and the create action rides the page bar's right edge -- the
	// ADMINISTRATOR'S entry (an admin session alone renders it). The
	// bar's left side is the page's one title with its own count.
	await expect(page.locator('.albums__title')).toHaveText('Albums · 1 album');
	await expect(page.locator('.albums__toolbar')).toHaveClass(/border-b/);
	const myPhotos = page.locator('a.album-card').filter({ hasText: 'My photos' });
	await expect(myPhotos).toBeVisible();
	await expect(myPhotos.locator('.album-card__count')).toHaveAttribute('aria-label', '0 photos');
	await expect(page.locator('a.album-card')).toHaveCount(1);
	await expect(page.getByText('Gather a trip or an occasion into an album.')).toBeHidden();
	await expect(page.getByRole('button', { name: 'New album', exact: true })).toBeVisible();

	// An address the app has no route for renders the content shell
	// plus the one unreachable card: the same face a dead resource id
	// gets, with the bar above it intact.
	await page.goto('/no-such-page');
	await expect(page.locator('.unreachable-card')).toBeVisible();
	await expect(page.getByRole('banner')).toBeVisible();
	await expect(page.getByRole('link', { name: 'Back to the timeline' })).toBeVisible();
	await page.goto('/albums');
	await expect(page).toHaveURL(/\/albums$/);

	// Hand the admin session to every later admin scene: the storage
	// state rides the login cookie (the fresh contexts replay it).
	await page.context().storageState({ path: adminStatePath });
});

// The admin half of the story: every scene starts from a fresh
// context that replays the admin's login cookie scene 1 saved. The
// serial mode guarantees the state file exists by the time these
// scenes run.
test.describe('the administrator scenes', () => {
	test.use({ storageState: adminStatePath });

	test('scene 2: the library bootstrap converges the model download', async ({ page }) => {
		test.setTimeout(DOWNLOAD_TIMEOUT_MS + 120_000);

		// The model group's download first (this story uploads nothing
		// -- both albums' members arrive through their source scans --
		// but the scans' pipelines run the same metadata/thumbnail/ML
		// fan-out an upload's does, so the models must be ready for
		// the tiles to converge the same way; the admin API is the upload
		// story's seeding pattern -- the request rides the page's own
		// context and its login cookie). Warm mount or cold, the run
		// must reach succeeded.
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
	});

	test("scene 3: the completion drills the container's real filesystem and never lists the managed tree", async ({
		page
	}) => {
		await page.goto('/albums');
		await page.getByRole('button', { name: 'New album', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'New album', exact: true });
		await expect(dialog).toBeVisible();

		// The system-wide domain (the mount-boundary revision): the
		// empty prefix on focus lists the filesystem ROOT ITSELF --
		// the combobox's one "start here" row, the honest reflection
		// of what the container sees (the allowlist's closed posture
		// died with the allowlist). The candidate block is a PORTALED
		// floating layer (the floating-layer revision), so its
		// rows live in the document at page level, outside the
		// dialog's DOM.
		const pathField = dialog.getByLabel('Path');
		await pathField.click();
		const suggest = page.locator('.path-completion__suggest');
		await expect(suggest.getByRole('option', { name: '/', exact: true })).toBeVisible();

		// The boundary is the compose bind: the trailing-separator
		// drill into the container's /mnt lists BOTH fixture mounts --
		// read-only and writable -- because the mounts themselves are
		// what the container's world holds.
		await pathField.fill('/mnt/');
		await expect(suggest.getByRole('option', { name: MOUNT_ROOT, exact: true })).toBeVisible();
		await expect(suggest.getByRole('option', { name: RW_MOUNT_ROOT, exact: true })).toBeVisible();

		// The managed tree never lists (the one exclusion that stayed
		// through the retirement): drilling the storage root's own
		// directory answers its config sibling alone -- the data tree
		// (originals, derivatives, trash) has no row, ever.
		await pathField.fill('/var/lib/lumemor/');
		await expect(suggest.getByRole('option', { name: '/var/lib/lumemor/config', exact: true })).toBeVisible();
		await expect(suggest.getByRole('option', { name: '/var/lib/lumemor/data', exact: true })).toHaveCount(0);

		// The closing discipline (the keys footer's promise, walked):
		// the list's Esc closes the list ALONE -- the dialog behind it
		// stays -- and the modal's own Esc owns the next stroke. (The
		// portaled layer leaves the document with the close.)
		await pathField.press('Escape');
		await expect(suggest).toHaveCount(0);
		await expect(dialog).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(dialog).toHaveCount(0);
	});

	test("scene 4: the read-only album's creation carries the probe's honest report in the toast", async ({ page }) => {
		await page.goto('/albums');
		await page.getByRole('button', { name: 'New album', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'New album', exact: true });
		await expect(dialog).toBeVisible();

		await dialog.getByLabel('Title').fill(ALBUM_TITLE);

		// The path completion against the mounted fixture tree: the
		// typed prefix's parent lists the matching directory, the row
		// appears (in the PORTALED floating layer, at page level --
		// the dialog's DOM never carries it), Enter completes it into
		// the field.
		const pathField = dialog.getByLabel('Path');
		await pathField.fill(`${MOUNT_ROOT}/coas`);
		await expect(page.getByRole('option', { name: SOURCE_DIR })).toBeVisible();
		await pathField.press('Enter');
		await expect(pathField).toHaveValue(SOURCE_DIR);

		// The audience stays the honest "Only you": this story grants
		// through the edit dialog's audience face, so the create
		// carries the empty set.
		await expect(dialog.getByText('Only you')).toBeVisible();

		// The create dialog's autonomy block in its read-only shape,
		// learned from the pick: the timeline-exclusion toggle present
		// and checked (the feed's default), NO deletability row -- the
		// picked completion row's writable fact is false, and a
		// read-only source renders no dead affordance.
		const autonomy = dialog.locator('[data-testid="album-autonomy"]');
		await expect(autonomy).toBeVisible();
		await expect(autonomy.locator('[data-testid="album-timeline-toggle"]')).toBeChecked();
		await expect(autonomy.locator('[data-testid="album-deletable-toggle"]')).toHaveCount(0);

		// The POST shape: exactly what the dialog holds, pinned against
		// the real request the page sends (the read-only pick's row is
		// absent, so no allowAppDeletion key rides along).
		const createPost = page.waitForRequest(
			(request) => request.method() === 'POST' && request.url().endsWith('/api/v1/albums')
		);
		await dialog.getByRole('button', { name: 'Create album', exact: true }).click();
		const post = await createPost;
		expect(post.postDataJSON()).toEqual({
			title: ALBUM_TITLE,
			rootPath: SOURCE_DIR,
			audience: [],
			inTimeline: true
		});

		// The write probe's honest face: the read-only bind
		// answers sourceWritable FALSE -- a fact about the album's
		// source, never a creation failure. The 201 landed the album;
		// the response carries the fact.
		const response = await post.response();
		expect(response?.status()).toBe(201);
		const createBody = (await response?.json()) as { sourceWritable?: boolean };
		expect(createBody.sourceWritable).toBe(false);

		// The creation's confirmation is the toast (the sample's
		// create button is the design): the read-only note rides the
		// SAME organ -- both sentences, one acknowledgment -- and the
		// dialog closes on the landed write. (The skipped-audience
		// report alone still holds the dialog: a list cannot ride a
		// toast; this story's audience was empty.)
		await expect(dialog).toHaveCount(0);
		await expect(
			page.getByRole('status').filter({
				hasText:
					'The album was created. This folder is read-only — its photos cannot be deleted through the app.'
			})
		).toBeVisible();

		// The SSE album.created echo converges the wall, and the
		// creation's OWN scan converges the card: no rescan is pressed
		// here, and the folder's two photos land (the creation-time
		// scan is the album's first scan). The count is the wall's own
		// live number -- album.items-changed refetches the grid.
		const card = page.locator('a.album-card').filter({ hasText: ALBUM_TITLE });
		await expect(card).toBeVisible();
		await expect(card.locator('.album-card__count')).toHaveAttribute('aria-label', '2 photos', {
			timeout: CREATION_SCAN_TIMEOUT_MS
		});

		// The created album's id, for the later scenes' deep links.
		albumId = (await card.getAttribute('href'))?.replace('/albums/', '') ?? '';
		expect(albumId).not.toBe('');
	});

	test("scene 5: the creation's scan converges the album, and the manual rescan re-runs it idempotently", async ({
		page
	}) => {
		await page.goto(`/albums/${albumId}`);
		await expect(page.locator('h1.album-bar__title')).toHaveText(ALBUM_TITLE);

		// THE CREATION'S OWN SCAN: the album page holds the folder's
		// two photos already -- the scan the creation enqueued, never
		// a button -- and the bar's count agrees with the grid.
		await expect(page.locator('.album-bar .album-detail__count')).toHaveAttribute('aria-label', '2 photos', {
			timeout: CREATION_SCAN_TIMEOUT_MS
		});
		await expect(page.locator('.tile')).toHaveCount(2, { timeout: CREATION_SCAN_TIMEOUT_MS });
		await expect(page.getByRole('link', { name: PHOTO_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_B_NAME, exact: true })).toBeVisible();

		// The manual rescan is the owner's explicit re-run -- still the
		// endpoint and still the same three-arm diff: the POST answers
		// the run's identifier (202 the fresh run; 200 joins a run of
		// this album still in flight, the endpoint's idempotence).
		const rescanPosted = page.waitForResponse(
			(response) =>
				response.request().method() === 'POST' && response.url().endsWith(`/api/v1/albums/${albumId}/rescan`)
		);
		await page.getByRole('button', { name: 'Rescan', exact: true }).click();
		const answer = await rescanPosted;
		expect([200, 202]).toContain(answer.status());
		const runBody = (await answer.json()) as { jobRunId?: string };
		expect(runBody.jobRunId).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/);

		// The aria-live progress line converges through the job events
		// the run publishes: the queued/running states pass, the
		// finish lands the run's own word.
		const progress = page.locator('[data-testid="album-rescan-progress"]');
		await expect(progress).toBeVisible();
		await expect(progress).toContainText('Rescan complete.', { timeout: 120_000 });

		// The idempotent re-run changes nothing the user sees: the
		// membership is still the two fixture photos.
		await expect(page.locator('.album-bar .album-detail__count')).toHaveAttribute('aria-label', '2 photos', {
			timeout: 120_000
		});
		await expect(page.locator('.tile')).toHaveCount(2, { timeout: 120_000 });

		// The bar's count is unconditional: a narrow window keeps it
		// beside the truncating title (the one numeric statement the
		// bar carries at every width).
		await page.setViewportSize({ width: 560, height: 800 });
		const barCount = page.locator('.album-bar .album-detail__count');
		await expect(barCount).toBeVisible();
		await expect(barCount).toHaveAttribute('aria-label', '2 photos');
		await page.setViewportSize({ width: 1280, height: 720 });
		// At the desktop width the count rides the bar as the sample
		// draws it: the glyph BESIDE the digits on ONE line (the
		// retired gate blockified the count here -- the variant's
		// display:inline beat the component's inline-flex and stacked
		// the svg over the number, a 35px-tall count). The height is
		// the decisive check: a flex child's computed display always
		// blockifies (inline-flex reads as flex), so the one-line
		// geometry -- not the display value -- is what the pin can
		// honestly assert.
		await expect(barCount).toBeVisible();
		expect(await barCount.evaluate((el) => el.getBoundingClientRect().height)).toBeLessThan(24);

		// And the scan's pipeline ran to its end: the source ingest
		// enqueued the same fan-out an upload's does, and both files
		// reach the ready state (the tiles' thumbnails are real, the
		// tiles are ready -- not merely present).
		await expect
			.poll(
				async () => {
					const timeline = await page.request.get(
						`/api/v1/assets/timeline?albumId=${encodeURIComponent(albumId)}`
					);
					expect(timeline.status()).toBe(200);
					const body = (await timeline.json()) as { items?: Array<{ isReady: boolean }> };
					return (body.items ?? []).filter((item) => item.isReady).length;
				},
				{ timeout: 120_000, intervals: [500, 1000, 2000, 5000] }
			)
			.toBe(2);
	});

	test("scene 6: the task panel lists the rescan's row with its own label", async ({ page }) => {
		await page.goto('/admin/jobs');
		await expect(page.getByRole('heading', { level: 1, name: 'Jobs', exact: true })).toBeVisible();

		// The rescan's job_runs row: its own kind label (the
		// contract's JobRunKind carries album_rescan -- the panel's
		// listing validator accepts the row, so the panel renders the
		// runs instead of the generic failure), the succeeded state,
		// and the resolved-entry counts in the run's own units.
		const row = page.locator('article.jobs__row').filter({ hasText: 'Album rescan' }).first();
		await expect(row).toBeVisible();
		await expect(row.getByText('Succeeded', { exact: true })).toBeVisible();
		await expect(row.getByText('2 of 2 assets')).toBeVisible();
	});

	test("scene 7: the edit dialog's autonomy block rides the audience grant", async ({ page }) => {
		await page.goto(`/albums/${albumId}`);

		// The PATCH collector: the save must write the audience PUT
		// alone -- the unchanged title writes nothing (the page diffs
		// against the live detail).
		const patchUrls: string[] = [];
		page.on('request', (request) => {
			if (request.method() === 'PATCH' && request.url().includes(`/api/v1/albums/${albumId}`)) {
				patchUrls.push(request.url());
			}
		});

		await page.getByRole('button', { name: 'Properties', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Album properties', exact: true });
		await expect(dialog).toBeVisible();

		// The source row: the directory the album references, read
		// only -- the path is pinned at creation.
		await expect(dialog.locator('.album-form__source')).toContainText(SOURCE_DIR);
		await expect(dialog.getByText('Source · fixed')).toBeVisible();

		// The autonomy block: the caller's own TIMELINE-EXCLUSION
		// toggle rides the dialog, the LOCK TOGGLE IS GONE (the lock's
		// retirement: no lock affordance renders anywhere), and the
		// DELETABILITY row does not render at all. The read-only
		// source carries no row: no affordance exists for a write the
		// application can never perform (the probe's false fact kills
		// the row, scene 4's toast said why).
		const autonomy = dialog.locator('[data-testid="album-autonomy"]');
		await expect(autonomy).toBeVisible();
		await expect(autonomy.locator('[data-testid="album-lock-toggle"]')).toHaveCount(0);
		await expect(autonomy.locator('[data-testid="album-timeline-toggle"]')).toBeVisible();
		await expect(autonomy.locator('[data-testid="album-deletable-toggle"]')).toHaveCount(0);

		// The audience face: the honest "Only you", then the picker
		// (a PORTALED floating layer at page level -- the dialog's
		// DOM never carries it) offers the one other account (the
		// identity seam's self filter), Enter adds the chip, and the
		// DRAIN collapses the picker: this instance holds exactly two
		// accounts, so the member's add empties the candidate pool,
		// and the known-pool rule never shows an empty picker -- the
		// layer falls, the add button disables, and the reserved
		// no-candidates slot carries the words.
		await expect(dialog.getByText('Only you')).toBeVisible();
		await dialog.getByRole('button', { name: 'Add', exact: true }).click();
		const picker = page.getByRole('listbox', { name: 'Candidate members' });
		// The candidate row names the person (the identity rule: the
		// display name leads, never the login address).
		await expect(picker.getByRole('option', { name: MEMBER_NAME })).toBeVisible();
		await picker.press('Enter');
		await expect(dialog.locator('.audience-field__chip')).toContainText(MEMBER_NAME);
		await expect(picker).toHaveCount(0);
		await expect(dialog.getByRole('button', { name: 'Add', exact: true })).toBeDisabled();
		await expect(dialog.getByText('No other account is available.')).toBeVisible();

		// The audience write's request shape: the FULL set, pinned
		// against the real PUT the page sends.
		const audiencePut = page.waitForRequest(
			(request) => request.method() === 'PUT' && request.url().endsWith(`/api/v1/albums/${albumId}/audience`)
		);
		await dialog.getByRole('button', { name: 'Save changes', exact: true }).click();
		const put = await audiencePut;
		expect(put.postDataJSON()).toEqual({ audience: [memberUserId] });

		// The write converged: the dialog closes, and no title PATCH
		// ever left the page.
		await expect(dialog).toHaveCount(0);
		expect(patchUrls).toEqual([]);

		// The honest convergence: the refetched detail's grants reseed
		// the chips -- the member's row is the audience now. The
		// reseeded face also carries the known-pool rule the other way round:
		// with the only other account already granted, the probed
		// pool is KNOWN empty and the add affordance opens nothing.
		await page.getByRole('button', { name: 'Properties', exact: true }).click();
		await expect(page.getByRole('dialog', { name: 'Album properties', exact: true })).toBeVisible();
		await expect(
			page.getByRole('dialog', { name: 'Album properties', exact: true }).locator('.audience-field__chip')
		).toContainText(MEMBER_NAME);
		await expect(
			page
				.getByRole('dialog', { name: 'Album properties', exact: true })
				.getByRole('button', { name: 'Add', exact: true })
		).toBeDisabled();
		await page
			.getByRole('dialog', { name: 'Album properties', exact: true })
			.getByRole('button', { name: 'Cancel', exact: true })
			.click();
	});

	test("scene 7b: the cover picker pins the album card's four frames and clears back to the rotation", async ({
		page
	}) => {
		// The four-frame collection: the trip fixture referenced in place
		// (the read-only bind), scanned into the album. The creation and
		// the rescan ride the API -- the dialogs those flows own are
		// scenes 4 and 10's subject.
		const created = await page.request.post('/api/v1/albums', {
			data: { title: COVER_ALBUM_TITLE, rootPath: COVER_SOURCE_DIR, audience: [] }
		});
		expect(created.status()).toBe(201);
		coverAlbumId = ((await created.json()) as { id?: string }).id ?? '';
		expect(coverAlbumId).not.toBe('');
		const rescan = await page.request.post(`/api/v1/albums/${coverAlbumId}/rescan`);
		expect([200, 202]).toContain(rescan.status());

		// The scan's convergence: four live members, then their ids in
		// the timeline's order (newest first) -- the pick order the
		// scene pins is its own, so the exact ids are read once here.
		await expect
			.poll(
				async () => {
					const detail = await page.request.get(`/api/v1/albums/${coverAlbumId}?limit=200`);
					expect(detail.status()).toBe(200);
					const body = (await detail.json()) as {
						itemCount?: number;
						items?: Array<{ assetId?: string }>;
					};
					if ((body.itemCount ?? 0) !== 4) {
						return 0;
					}
					coverMemberIds = (body.items ?? []).map((item) => item.assetId ?? '');
					return coverMemberIds.filter((id) => id !== '').length;
				},
				{ timeout: 60_000, intervals: [500, 1000, 2000] }
			)
			.toBe(4);

		// The automatic group, read twice in one week: the same four
		// frames (the rotation's freeze -- a family device that refreshes
		// never sees the cover move).
		const groupOf = async (): Promise<string[]> => {
			const wall = await page.request.get('/api/v1/albums');
			expect(wall.status()).toBe(200);
			const body = (await wall.json()) as {
				albums?: Array<{ id?: string; coverAssetIds?: string[] }>;
			};
			return (body.albums ?? []).find((album) => album.id === coverAlbumId)?.coverAssetIds ?? [];
		};
		const automatic = await groupOf();
		expect(automatic).toHaveLength(4);
		expect(automatic.every((id) => coverMemberIds.includes(id))).toBe(true);
		expect(await groupOf()).toEqual(automatic);

		// The card draws the automatic group in the answered order, the
		// collage labelled as one image (each cell silent).
		await page.goto('/albums');
		const card = page.locator('a.album-card').filter({ hasText: COVER_ALBUM_TITLE });
		await expect(card).toBeVisible();
		await expect(card.locator('.album-card__cover')).toHaveAttribute('aria-label', 'Cover: 4 photos');
		await expect(card.locator('.album-card__cell img')).toHaveCount(4);
		const cardFrames = async (): Promise<string[]> =>
			card
				.locator('.album-card__cell img')
				.evaluateAll((images) =>
					images.map((image) => /\/assets\/([^/]+)\//.exec(image.getAttribute('src') ?? '')?.[1] ?? '')
				);
		expect(await cardFrames()).toEqual(automatic);

		// The owner's pick: four frames in an order of the scene's own
		// (the reverse of the wall's group), saved through the dialog.
		await card.click();
		await expect(page.locator('h1.album-bar__title')).toHaveText(COVER_ALBUM_TITLE);
		await page.getByRole('button', { name: 'Set cover', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Set the cover', exact: true });
		await expect(dialog).toBeVisible();
		// The stored pin is empty: the dialog opens on the automatic
		// state.
		await expect(dialog.locator('.cover-picker__count')).toContainText('Automatic');
		const options = dialog.locator('.cover-option');
		await expect(options).toHaveCount(4);
		const pin = [...coverMemberIds].reverse();
		for (const id of pin) {
			await options.filter({ has: page.locator(`img[src*="/assets/${id}/"]`) }).click();
		}
		await expect(dialog.locator('.cover-option__badge')).toHaveCount(4);
		await expect(dialog.locator('.cover-picker__count')).toContainText('Selected 4 / 4');

		const patched = page.waitForRequest(
			(request) => request.method() === 'PATCH' && request.url().endsWith(`/api/v1/albums/${coverAlbumId}`)
		);
		await dialog.getByRole('button', { name: 'Save cover', exact: true }).click();
		const patch = await patched;
		// The write carries the pick order exactly (the server stores
		// and returns it as given).
		expect(patch.postDataJSON()).toEqual({ coverAssetIds: pin });
		expect((await patch.response())?.status()).toBe(200);
		await expect(dialog).toHaveCount(0);
		await expect(page.getByText('Cover updated.', { exact: true })).toBeVisible();

		// The pinned card: the four frames in the PICK order, not the
		// automatic group's.
		await page.goto('/albums');
		await expect(card.locator('.album-card__cell img')).toHaveCount(4);
		expect(await cardFrames()).toEqual(pin);

		// The partial state: reopening seeds the stored pin (its order,
		// its numbers), and one removed frame blocks the save -- a cover
		// is the full set or nothing.
		await card.click();
		await page.getByRole('button', { name: 'Set cover', exact: true }).click();
		await expect(dialog).toBeVisible();
		await expect(dialog.locator('.cover-picker__count')).toContainText('Selected 4 / 4');
		// The numbers follow the PICK order, not the grid's own order:
		// the frame picked first wears 1.
		await expect(
			options.filter({ has: page.locator(`img[src*="/assets/${pin[0]}/"]`) }).locator('.cover-option__badge')
		).toHaveText('1');
		await expect(
			options.filter({ has: page.locator(`img[src*="/assets/${pin[3]}/"]`) }).locator('.cover-option__badge')
		).toHaveText('4');
		await options.filter({ has: page.locator(`img[src*="/assets/${pin[0]}/"]`) }).click();
		await expect(dialog.locator('.cover-picker__count')).toContainText('Selected 3 / 4 · 1 more to pick');
		await expect(dialog.getByRole('button', { name: 'Save cover', exact: true })).toBeDisabled();

		// The way back: "Use the weekly rotation" empties the pick, the
		// save writes the empty set, and the album returns to the
		// automatic group.
		await dialog.getByRole('button', { name: 'Use the weekly rotation', exact: true }).click();
		await expect(dialog.locator('.cover-picker__count')).toContainText('Automatic');
		const cleared = page.waitForRequest(
			(request) => request.method() === 'PATCH' && request.url().endsWith(`/api/v1/albums/${coverAlbumId}`)
		);
		await dialog.getByRole('button', { name: 'Save cover', exact: true }).click();
		expect((await cleared).postDataJSON()).toEqual({ coverAssetIds: [] });
		await expect(dialog).toHaveCount(0);

		const detail = await page.request.get(`/api/v1/albums/${coverAlbumId}`);
		expect(((await detail.json()) as { coverAssetIds?: string[] }).coverAssetIds).toEqual([]);
		expect(await groupOf()).toEqual(automatic);

		// The contract's refusal matrix, through the API: a partial set
		// names the field, a set with a frame outside the album names
		// the count, and an over-full set never fits the card.
		const partial = await page.request.patch(`/api/v1/albums/${coverAlbumId}`, {
			data: { coverAssetIds: [coverMemberIds[0]] }
		});
		expect(partial.status()).toBe(400);
		expect(await partial.json()).toMatchObject({
			code: 'VALIDATION_FAILED',
			details: { fields: ['coverAssetIds'] }
		});

		const stranger = await page.request.patch(`/api/v1/albums/${coverAlbumId}`, {
			data: {
				coverAssetIds: [
					coverMemberIds[0],
					coverMemberIds[1],
					coverMemberIds[2],
					'0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f60ff'
				]
			}
		});
		expect(stranger.status()).toBe(400);
		expect(await stranger.json()).toMatchObject({
			code: 'VALIDATION_FAILED',
			details: { key: 'album_cover_not_member', count: 1 }
		});

		const overfull = await page.request.patch(`/api/v1/albums/${coverAlbumId}`, {
			data: { coverAssetIds: [...coverMemberIds, '0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f60ff'] }
		});
		expect(overfull.status()).toBe(400);
		expect(await overfull.json()).toMatchObject({
			code: 'VALIDATION_FAILED',
			details: { fields: ['coverAssetIds'] }
		});

		// The album goes before the removal scenes (each of them counts
		// the wall): the un-ingest purges the four ingested rows, and
		// the wall is back to its two albums.
		const removed = await page.request.delete(`/api/v1/albums/${coverAlbumId}`);
		expect(removed.status()).toBe(204);
		await expect(page.locator('a.album-card').filter({ hasText: COVER_ALBUM_TITLE })).toHaveCount(0);
	});
});

// The member half of the story: scene 8 signs the member in through
// their own form (a fresh context -- the two accounts never share a
// browser session) and saves their storage state; scene 9 replays
// it.
test('scene 8: the member signs in, the creation guard refuses them, and the shared album reads view-only', async ({
	browser
}) => {
	// The config's use-locale does not reach manually created
	// contexts; pin it so the asserted strings render in English.
	const context = await browser.newContext({ locale: 'en-US', timezoneId: 'UTC' });
	const page = await context.newPage();

	await page.goto('/login');
	await page.getByLabel('Email').fill(MEMBER_EMAIL);
	await page.getByLabel('Password').fill(MEMBER_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);

	// The wall is ONE list (no own/shared sections): the shared album's
	// card carries the owner's NAME beside the read-only chip and the
	// honest count -- the address never renders.
	await page.goto('/albums');
	const card = page.locator('a.album-card').filter({ hasText: ALBUM_TITLE });
	await expect(page.getByRole('heading', { level: 2 })).toHaveCount(0);
	await expect(card).toBeVisible();
	await expect(card.locator('.album-card__badge--owner')).toHaveText(`Shared by ${ADMIN_NAME}`);
	await expect(card.locator('.album-card__badge--owner')).not.toContainText(ADMIN_EMAIL);
	await expect(card.locator('.album-card__badge--readonly')).toHaveText('Read-only');
	await expect(card.locator('.album-card__count')).toHaveAttribute('aria-label', '2 photos');

	// The creation guard's visual half: mounting an existing
	// photo directory is a configuration act, so the member's wall
	// renders NO creation affordance at all.
	await expect(page.getByRole('button', { name: 'New album', exact: true })).toHaveCount(0);

	// The guard's server half, the refusal matrix's new arms: the
	// member's create-album attempt and their completion call both
	// answer AUTH_FORBIDDEN (the requests ride the member's own page
	// context and its login cookie -- the same session the browser
	// scenes walk).
	const memberCreate = await page.request.post('/api/v1/albums', {
		data: { title: 'Member attempt', rootPath: SOURCE_DIR, audience: [] }
	});
	expect(memberCreate.status()).toBe(403);
	const memberCreateBody = (await memberCreate.json()) as { code?: string };
	expect(memberCreateBody.code).toBe('AUTH_FORBIDDEN');

	const memberCompletion = await page.request.get(
		`/api/v1/albums/path-completions?prefix=${encodeURIComponent(MOUNT_ROOT)}`
	);
	expect(memberCompletion.status()).toBe(403);
	const memberCompletionBody = (await memberCompletion.json()) as { code?: string };
	expect(memberCompletionBody.code).toBe('AUTH_FORBIDDEN');

	// The shared detail reads as viewer: no owner affordance ever
	// renders (the properties dialog, the rescan and the remove are
	// the owner's), while the timeline-exclusion toggle -- the
	// member's own preference -- does.
	await card.click();
	await expect(page).toHaveURL(new RegExp(`/albums/${albumId}$`));
	await expect(page.locator('h1.album-bar__title')).toHaveText(ALBUM_TITLE);
	for (const label of ['Properties', 'Rescan', 'Delete album', 'Remove album']) {
		await expect(page.getByRole('button', { name: label, exact: true })).toHaveCount(0);
	}
	// The member view's action cluster holds EXACTLY one button --
	// the caller's own timeline preference. Nothing the owner domain
	// owns is rendered as a disabled control.
	const cluster = page.locator('.album-detail__actions');
	await expect(cluster.locator('button')).toHaveCount(1);
	await expect(page.getByRole('button', { name: 'Keep out of timeline', exact: true })).toBeVisible();
	await expect(page.locator('.tile')).toHaveCount(2);

	// The unreachable face, live: an album id this caller cannot
	// reach answers the contract's 404, and the page renders the one
	// card. The bar keeps the way back alone, nothing of a resource
	// renders, and two different dead ids read byte for byte the
	// same -- the address can never narrow the answer.
	await page.goto('/albums/0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60');
	await expect(page.locator('.unreachable-card')).toBeVisible();
	await expect(page.locator('.unreachable-card')).toContainText('This page will not open');
	await expect(page.locator('.unreachable-card')).toContainText(
		'It may have been deleted, or you may not have access to it.'
	);
	await expect(page.locator('.album-bar__back')).toBeVisible();
	await expect(page.locator('.album-bar__title')).toHaveCount(0);
	await expect(page.locator('.album-detail__count')).toHaveCount(0);
	await expect(page.locator('.album-detail__actions button')).toHaveCount(0);
	const unknownFace = (await page.locator('main').innerText()).trim();
	await page.goto('/albums/0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f61');
	await expect(page.locator('.unreachable-card')).toBeVisible();
	expect((await page.locator('main').innerText()).trim()).toBe(unknownFace);
	await page.goto(`/albums/${albumId}`);
	await expect(page.locator('h1.album-bar__title')).toHaveText(ALBUM_TITLE);

	// The member's wall curates their own favorites -- and nothing
	// else: the trash would delete the owner's files, and the
	// archive switch writes the owner's row (the batch resolves it
	// over the caller's own ids, so a shared album can never land it).
	const check = page.locator('.tile__check').first();
	await check.click();
	const bar = page.getByRole('toolbar');
	await expect(bar).toContainText('1 selected');
	await expect(bar.getByRole('button', { name: 'Favorite', exact: true })).toBeVisible();
	await expect(bar.getByRole('button', { name: 'Clear selection', exact: true })).toBeVisible();
	await expect(bar.locator('button')).toHaveCount(2);
	for (const label of ['Trash', 'Archive', 'Unarchive', 'Restore']) {
		await expect(bar.getByRole('button', { name: label, exact: true })).toHaveCount(0);
	}

	// The per-user write lands (the silent arm): the pass clears the
	// selection, and the flipped flag reads back as the unfavorite.
	await bar.getByRole('button', { name: 'Favorite', exact: true }).click();
	await expect(page.locator('.batchbar')).toHaveCount(0);
	await check.click();
	await expect(bar.getByRole('button', { name: 'Unfavorite', exact: true })).toBeVisible();
	// Toggling back restores the unflagged state the later scenes
	// (and scene 9's favorite proof) expect.
	await bar.getByRole('button', { name: 'Unfavorite', exact: true }).click();
	await expect(page.locator('.batchbar')).toHaveCount(0);

	// The one writable thing flips both ways, and the badge follows
	// the response (the write publishes no event -- the page's own
	// state is the answer).
	await page.getByRole('button', { name: 'Keep out of timeline', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Keep in timeline', exact: true })).toBeVisible();
	await expect(page.locator('.album-detail__badge--excluded')).toHaveText('Not in timeline');

	// The exclusion reaches the caller's own aggregate timeline: the
	// member owns nothing else, so the timeline settles empty.
	await page.goto('/timeline');
	await expect(page.locator('.state-card')).toBeVisible();
	await expect(page.locator('.tile')).toHaveCount(0);

	// Keeping it in restores the shared items (the union's shared
	// arm -- the member owns nothing of their own).
	await page.goto(`/albums/${albumId}`);
	await page.getByRole('button', { name: 'Keep in timeline', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Keep out of timeline', exact: true })).toBeVisible();
	await page.goto('/timeline');
	await expect(page.getByRole('link', { name: PHOTO_A_NAME, exact: true })).toBeVisible();
	await expect(page.getByRole('link', { name: PHOTO_B_NAME, exact: true })).toBeVisible();
	await expect(page.locator('.tile')).toHaveCount(2);

	// Hand the member session to scene 9.
	await context.storageState({ path: memberStatePath });
	await context.close();
});

test.describe('the member viewer scenes', () => {
	test.use({ storageState: memberStatePath });

	test("scene 9: the viewer's drawer keeps the reads and the per-user favorite, never the owner-only writes", async ({
		page
	}) => {
		// A shared photo's viewer, deep-linked from the album detail.
		await page.goto(`/albums/${albumId}`);
		await page.getByRole('link', { name: PHOTO_A_NAME, exact: true }).click();
		await expect(page).toHaveURL(new RegExp(`/a/[0-9a-f-]{36}$`));
		// The floating bar carries no filename: the open asset's name
		// is the tab's, and the drawer's header repeats it below.
		await expect(page).toHaveTitle(`${PHOTO_A_NAME} · lumemor`);

		// The info drawer plus the action bar (the viewer rework: the
		// drawer is metadata-only, the per-user favorite and the
		// original download ride the bar's icon affordances): the reads
		// render (the file facts, the download affordance) and the
		// favorite stays, while every owner-only write affordance is
		// absent (the identity seam's browser proof -- the session's
		// userId is not the asset's owner).
		await page.getByRole('button', { name: 'Photo information' }).click();
		const drawer = page.getByRole('complementary', { name: 'Photo information' });
		await expect(drawer).toBeVisible();
		await expect(page.getByRole('link', { name: 'Download the original' })).toBeVisible();
		const favorite = page.getByRole('button', { name: 'Favorite', exact: true });
		await expect(favorite).toBeVisible();
		for (const label of ['Edit', 'Archive', 'Unarchive', 'Trash', 'Restore', 'Delete permanently']) {
			// The owner-only writes never render in either surface: no
			// trash/restore in the BAR (the drawer never carried a
			// curation button since the rework), no description edit.
			await expect(page.getByRole('button', { name: label, exact: true })).toHaveCount(0);
		}

		// The per-user favorite works for the viewer: the flag flips
		// without a page reload (the optimistic per-user state).
		await favorite.click();
		await expect(page.getByRole('button', { name: 'Unfavorite', exact: true })).toBeVisible();
	});
});

// The writable arm of the story (the autonomy chain): the admin's
// session replays; the scenes walk the probe's TRUE fact, the
// deletability row it renders (both dialogs), the create-time
// exclusion's convergence, the delete-only selection the derived
// face arms, and the trash/restore round trip that moves the source
// file itself.
test.describe('the writable source scenes', () => {
	test.use({ storageState: adminStatePath });

	test("scene 10: the writable source's creation answers the probe's true fact and closes directly", async ({
		page
	}) => {
		await page.goto('/albums');
		await page.getByRole('button', { name: 'New album', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'New album', exact: true });
		await expect(dialog).toBeVisible();

		await dialog.getByLabel('Title').fill(RW_ALBUM_TITLE);

		// The completion's exact-match row: typing the writable
		// fixture's full path answers the directory itself (the
		// parent /mnt holds exactly one entry whose name starts with
		// the typed last segment), Enter completes it.
		const pathField = dialog.getByLabel('Path');
		await pathField.fill(RW_MOUNT_ROOT);
		await expect(page.getByRole('option', { name: RW_MOUNT_ROOT, exact: true })).toBeVisible();
		await pathField.press('Enter');
		await expect(pathField).toHaveValue(RW_MOUNT_ROOT);

		await expect(dialog.getByText('Only you')).toBeVisible();

		// The create dialog's autonomy block in its writable shape,
		// learned from the pick: the deletability row RENDERS (seeded
		// on) -- the picked row's writable fact is true -- beside the
		// timeline-exclusion toggle, which seeds checked too. The
		// story then FLIPS the exclusion: this album is created kept
		// out of its creator's timeline (scene 12 walks the
		// convergence).
		const autonomy = dialog.locator('[data-testid="album-autonomy"]');
		await expect(autonomy).toBeVisible();
		const timelineToggle = autonomy.locator('[data-testid="album-timeline-toggle"]');
		await expect(timelineToggle).toBeChecked();
		const deletableToggle = autonomy.locator('[data-testid="album-deletable-toggle"]');
		await expect(deletableToggle).toBeVisible();
		await expect(deletableToggle).toBeChecked();
		await timelineToggle.uncheck();

		// The pinned body: the triple plus BOTH switches, exactly as
		// the dialog's drafts hold them (the deletability row's seed
		// on, the exclusion flipped off).
		const createPost = page.waitForRequest(
			(request) => request.method() === 'POST' && request.url().endsWith('/api/v1/albums')
		);
		await dialog.getByRole('button', { name: 'Create album', exact: true }).click();
		const post = await createPost;
		expect(post.postDataJSON()).toEqual({
			title: RW_ALBUM_TITLE,
			rootPath: RW_MOUNT_ROOT,
			audience: [],
			allowAppDeletion: true,
			inTimeline: false
		});

		// The probe's TRUE arm: the writable bind answers
		// sourceWritable true, and an all-landed, writable creation
		// holds NO report -- the dialog closes directly (the
		// acknowledgment face belongs to the honest problems alone).
		// The response card also carries the landed exclusion: the
		// album was born out of its creator's timeline.
		const response = await post.response();
		expect(response?.status()).toBe(201);
		const createBody = (await response?.json()) as {
			sourceWritable?: boolean;
			inTimeline?: boolean;
		};
		expect(createBody.sourceWritable).toBe(true);
		expect(createBody.inTimeline).toBe(false);
		await expect(dialog).toHaveCount(0);

		// The clean creation's confirmation is the house toast (the
		// skipped-audience report is for the honest problems alone --
		// a list cannot ride a toast -- so the path that closes
		// silently is the one that speaks here).
		await expect(page.locator('.toasts')).toContainText('Album created.');

		// The card lands, and the creation's own scan converges it to
		// the seeded pair -- the writable album's first scan is the
		// creation's too (nothing is pressed here).
		const card = page.locator('a.album-card').filter({ hasText: RW_ALBUM_TITLE });
		await expect(card).toBeVisible();
		await expect(card.locator('.album-card__count')).toHaveAttribute('aria-label', '2 photos', {
			timeout: CREATION_SCAN_TIMEOUT_MS
		});

		// The writable album's id, for the later scenes' deep links.
		rwAlbumId = (await card.getAttribute('href'))?.replace('/albums/', '') ?? '';
		expect(rwAlbumId).not.toBe('');
	});

	test("scene 11: the writable album's creation scan converges the pair and the deletability row renders", async ({
		page
	}) => {
		await page.goto(`/albums/${rwAlbumId}`);
		await expect(page.locator('h1.album-bar__title')).toHaveText(RW_ALBUM_TITLE);

		// The creation's own scan, the same beat as the read-only
		// story's scene 5: the seeded pair is already on the wall, and
		// the scan's pipeline runs both files to ready (the models are
		// warm; the fan-out is the one every ingest walks).
		await expect(page.locator('.tile')).toHaveCount(2, { timeout: CREATION_SCAN_TIMEOUT_MS });
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_RW_B_NAME, exact: true })).toBeVisible();
		await expect
			.poll(
				async () => {
					const timeline = await page.request.get(
						`/api/v1/assets/timeline?albumId=${encodeURIComponent(rwAlbumId)}`
					);
					expect(timeline.status()).toBe(200);
					const body = (await timeline.json()) as { items?: Array<{ isReady: boolean }> };
					return (body.items ?? []).filter((item) => item.isReady).length;
				},
				{ timeout: 120_000, intervals: [500, 1000, 2000, 5000] }
			)
			.toBe(2);

		// The edit dialog's autonomy block holds its writable shape:
		// the deletability row RENDERS for the writable source
		// (present exactly when the probe answered true), checked on,
		// beside the exclusion toggle -- and the LOCK TOGGLE IS GONE
		// (the lock's retirement). The exclusion toggle reads
		// UNCHECKED: the creation landed the creator's own exclusion
		// (scene 10's flipped draft), and the dialog seeds from the
		// detail's own fields.
		await page.getByRole('button', { name: 'Properties', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Album properties', exact: true });
		await expect(dialog).toBeVisible();
		const autonomy = dialog.locator('[data-testid="album-autonomy"]');
		await expect(autonomy.locator('[data-testid="album-lock-toggle"]')).toHaveCount(0);
		const editTimelineToggle = autonomy.locator('[data-testid="album-timeline-toggle"]');
		await expect(editTimelineToggle).toBeVisible();
		await expect(editTimelineToggle).not.toBeChecked();
		const deletableToggle = autonomy.locator('[data-testid="album-deletable-toggle"]');
		await expect(deletableToggle).toBeVisible();
		await expect(deletableToggle).toBeChecked();
		await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
		await expect(dialog).toHaveCount(0);
	});

	test('scene 12: the excluded creation keeps the writable album out of the aggregated timeline', async ({
		page
	}) => {
		// The create-time exclusion's convergence, the creator's own
		// aggregated view: the read-only album (created with the
		// feed's default, scene 4) feeds its two photos; the writable
		// album (created excluded, scene 10) feeds NOTHING -- the
		// preference landed in the creation itself. The wall is the
		// product surface's honest answer: exactly the read-only
		// pair's tiles.
		await page.goto('/timeline');
		await expect(page.getByRole('link', { name: PHOTO_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_B_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toHaveCount(0);
		await expect(page.getByRole('link', { name: PHOTO_RW_B_NAME, exact: true })).toHaveCount(0);
		await expect(page.locator('.tile')).toHaveCount(2);

		// The server's own answer agrees: exactly two items, the
		// read-only pair's -- and never the writable album's ids.
		await expect
			.poll(async () => {
				const timeline = await page.request.get('/api/v1/assets/timeline');
				expect(timeline.status()).toBe(200);
				const body = (await timeline.json()) as {
					items?: Array<{ id?: string; originalFilename?: string }>;
				};
				return (body.items ?? [])
					.map((item) => item.originalFilename ?? item.id ?? '')
					.sort()
					.join(',');
			})
			.toBe([PHOTO_A_NAME, PHOTO_B_NAME].sort().join(','));
	});

	test("scene 13: the deletable wall trashes through the selection bar and the trash view's restore returns the file", async ({
		page
	}) => {
		await page.goto(`/albums/${rwAlbumId}`);
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_RW_B_NAME, exact: true })).toBeVisible();

		// The delete-only selection (the wall's capability gate rides
		// the server-derived `deletable` face -- owner AND writable
		// AND the switch, the two-layer story's data half: the setting
		// gates the affordance, the operating system gates the
		// execution): the hover-reveal check is the selection entry,
		// one click selects the tile.
		await page.getByRole('checkbox', { name: `Select: ${PHOTO_RW_A_NAME}`, exact: true }).click();

		// The batch bar rises with the DELETE-ONLY subset: Trash is
		// the one action it offers (the album's wall curates deletion
		// alone -- every other curatable wall's action is absent).
		const bar = page.getByRole('toolbar', { name: '1 selected' });
		await expect(bar).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Trash', exact: true })).toBeVisible();
		for (const label of ['Favorite', 'Unfavorite', 'Archive', 'Unarchive', 'Restore', 'Add tag']) {
			await expect(bar.getByRole('button', { name: label, exact: true })).toHaveCount(0);
		}

		// The trash through the wall: the batch POST answers its
		// honest report (one affected id), the tile leaves the wall
		// immediately (the optimistic convergence), the survivor
		// stays, and the selection bar falls (the success arm). The
		// file itself MOVES into the managed trash area -- the
		// external lifecycle's own delete, the operating system's
		// authority over a writable source.
		const batchPosted = page.waitForResponse(
			(response) => response.request().method() === 'POST' && response.url().endsWith('/api/v1/assets/batch')
		);
		await bar.getByRole('button', { name: 'Trash', exact: true }).click();
		const batch = await batchPosted;
		expect(batch.status()).toBe(200);
		const batchBody = (await batch.json()) as { affected?: number };
		expect(batchBody.affected).toBe(1);
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toHaveCount(0);
		await expect(page.getByRole('link', { name: PHOTO_RW_B_NAME, exact: true })).toBeVisible();
		await expect(bar).toHaveCount(0);

		// The honest convergence, both faces: the album's own
		// items-changed event (the trash lifecycle's new convergence
		// signal) restarts the walk AND refetches the detail, so the
		// bar's count drops with the wall's tile -- the server's own
		// answer confirms it.
		await expect(page.locator('.album-bar .album-detail__count')).toHaveAttribute('aria-label', '1 photo');
		await expect
			.poll(
				async () => {
					const detail = await page.request.get(`/api/v1/albums/${rwAlbumId}`);
					expect(detail.status()).toBe(200);
					const body = (await detail.json()) as { itemCount?: number };
					return body.itemCount ?? 0;
				},
				{ timeout: 30_000, intervals: [500, 1000, 2000] }
			)
			.toBe(1);

		// The trash view carries the row with its retention countdown
		// -- the moved file waits there for the expiry or the restore.
		await page.goto('/trash');
		// The page's one row: the trash title with its count (the
		// count belongs to the title, never a row of its own).
		await expect(page.getByRole('heading', { level: 1, name: /^Trash 1 photo$/ })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByText(/^(Expiring soon|\d+ days? left)$/)).toBeVisible();

		// The restore moves the file BACK to the recorded original
		// location (the external lifecycle's restore arm): the row
		// leaves the trash view and the view settles empty.
		await page.getByRole('button', { name: 'Restore', exact: true }).click();
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toHaveCount(0);
		await expect(page.getByText('The trash is empty', { exact: true })).toBeVisible();

		// The album's fresh answer carries both photos again: the
		// restored row rejoined the membership (the restore publishes
		// the same items-changed signal, so a wall that stayed on the
		// page would restart its walk too), the count is honest, and
		// the source tree holds both files for the next run (the
		// fixture healed -- the bootstrap's re-seed is a no-op copy).
		await page.goto(`/albums/${rwAlbumId}`);
		await expect(page.locator('.tile')).toHaveCount(2, { timeout: 60_000 });
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(page.locator('.album-bar .album-detail__count')).toHaveAttribute('aria-label', '2 photos');
	});

	test('scene 13c: the archive empties the timeline, keeps the album member, and the album wall unarchives it', async ({
		page
	}) => {
		await page.goto(`/albums/${rwAlbumId}`);
		await expect(page.locator('.tile')).toHaveCount(2, { timeout: 60_000 });

		// Archive one member through the batch endpoint (the same write
		// the timeline's bar runs): the family flows lose the row, the
		// album keeps it -- the container rule. The album read names
		// the member's id.
		const albumRead = await page.request.get(`/api/v1/albums/${rwAlbumId}`);
		expect(albumRead.status()).toBe(200);
		const albumBody = (await albumRead.json()) as {
			items?: Array<{ assetId?: string; asset?: { originalFilename?: string } }>;
		};
		const memberId = (albumBody.items ?? []).find(
			(item) => item.asset?.originalFilename === PHOTO_RW_A_NAME
		)?.assetId;
		expect(memberId).toBeTruthy();
		const archived = await page.request.post('/api/v1/assets/batch', {
			data: { action: 'hide', ids: [memberId] }
		});
		expect(archived.status()).toBe(200);
		expect(((await archived.json()) as { affected?: number }).affected).toBe(1);

		// The archived view is where the row lives while archived (the
		// family flows exclude it; this album is the excluded creation,
		// so the archived view is the flow-side evidence).
		const archivedPage = await page.request.get('/api/v1/assets/archived');
		expect(archivedPage.status()).toBe(200);
		const archivedBody = (await archivedPage.json()) as {
			items?: Array<{ originalFilename?: string }>;
		};
		expect((archivedBody.items ?? []).map((item) => item.originalFilename)).toContain(PHOTO_RW_A_NAME);

		// The album's own wall keeps the member (the reference
		// container) and its count with it.
		await page.reload();
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(page.locator('.album-bar .album-detail__count')).toHaveAttribute('aria-label', '2 photos');

		// The container's unarchive entry: the button renders for the
		// archived selection alone, and the landing confirms.
		await page.getByRole('checkbox', { name: `Select: ${PHOTO_RW_A_NAME}`, exact: true }).click();
		const bar = page.getByRole('toolbar', { name: '1 selected' });
		await expect(bar).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Unarchive', exact: true })).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Archive', exact: true })).toHaveCount(0);
		await bar.getByRole('button', { name: 'Unarchive', exact: true }).click();
		await expect(page.getByText('Unarchived 1 photo.')).toBeVisible();
		// The container keeps the tile (unarchiving ends no album
		// membership), and the row leaves the archived view -- it is
		// back in the family flows.
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect
			.poll(async () => {
				const fresh = await page.request.get('/api/v1/assets/archived');
				expect(fresh.status()).toBe(200);
				const body = (await fresh.json()) as { items?: Array<{ originalFilename?: string }> };
				return (body.items ?? []).some((item) => item.originalFilename === PHOTO_RW_A_NAME);
			})
			.toBe(false);
	});

	test('scene 13b: the folder sync follows the folder — a dropped photo appears, a removed one leaves', async ({
		page
	}) => {
		// The held-open album wall: everything below converges through
		// the activity stream (the album's items-changed refetch and
		// the vanish arm's asset.trashed), never a reload.
		await page.goto(`/albums/${rwAlbumId}`);
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_RW_B_NAME, exact: true })).toBeVisible();
		await expect(page.locator('.album-bar .album-detail__count')).toHaveAttribute('aria-label', '2 photos');

		// THE ADD: the test writes straight into the bind-mounted
		// source -- the host side of the mount the container sees. No
		// trigger fires in the app: the periodic directory-snapshot
		// diff notices a file the last scan never saw and enqueues the
		// album's OWN scan, whose add arm ingests it through the same
		// pipeline a manual rescan walks.
		const dropped = path.join(RW_SOURCE_HOST_DIR, PHOTO_SYNC_NAME);
		await copyFile(FOLDER_SYNC_PHOTO, dropped);

		await expect(page.getByRole('link', { name: PHOTO_SYNC_NAME, exact: true })).toBeVisible({
			timeout: FOLDER_SYNC_TIMEOUT_MS
		});
		await expect(page.locator('.album-bar .album-detail__count')).toHaveAttribute('aria-label', '3 photos', {
			timeout: FOLDER_SYNC_TIMEOUT_MS
		});
		await expect(page.locator('.tile')).toHaveCount(3, { timeout: FOLDER_SYNC_TIMEOUT_MS });

		// THE REMOVE: the file leaves the folder, and the wall loses
		// the tile through the pinned orphan semantics -- the row is
		// marked deleted_at directly, and the trash view NEVER lists
		// it: the file was never the server's, so there is nothing of
		// ours in the trash to restore or expire away.
		await rm(dropped, { force: true });
		await expect(page.getByRole('link', { name: PHOTO_SYNC_NAME, exact: true })).toHaveCount(0, {
			timeout: FOLDER_SYNC_TIMEOUT_MS
		});
		await expect(page.locator('.album-bar .album-detail__count')).toHaveAttribute('aria-label', '2 photos', {
			timeout: FOLDER_SYNC_TIMEOUT_MS
		});
		await page.goto('/trash');
		await expect(page.getByText('The trash is empty', { exact: true })).toBeVisible();

		// The fixture healed itself (the source tree is the seeded pair
		// again), and the story's later scenes start from the two
		// photos they already expect.
		await page.goto(`/albums/${rwAlbumId}`);
		await expect(page.getByRole('link', { name: PHOTO_SYNC_NAME, exact: true })).toHaveCount(0);
		await expect(page.locator('.tile')).toHaveCount(2, { timeout: 60_000 });
	});
});

// The removal arm of the story: the admin's session
// replays; the scenes stage the removal's pre-state (the share, the
// named person, the two clusters' identities), walk the confirm's
// pinned semantics through the detail bar's remove action, and pin
// the un-ingest's whole aftermath -- the sources byte-intact on the
// host, every held-open surface's live convergence, the library's
// fresh answers, and the refusal matrix's two standing arms (the
// member's 403 and the sync album's single-shape 404).
test.describe('the removal scenes', () => {
	test.use({ storageState: adminStatePath });

	test("scene 14: the share arms the member, the second person is named, and the member's removal attempt refuses", async ({
		page,
		browser
	}) => {
		test.setTimeout(600_000);

		// The audience grant rides the same full-set PUT the edit
		// dialog's audience face drives (scene 7 walked that dialog;
		// this scene needs the grant, not the dialog): the report
		// names the granted member, the set is theirs.
		const share = await page.request.put(`/api/v1/albums/${rwAlbumId}/audience`, {
			data: { audience: [memberUserId] }
		});
		expect(share.status()).toBe(200);
		const shareBody = (await share.json()) as { granted?: string[] };
		expect(shareBody.granted).toEqual([memberUserId]);

		// The face clusters: the story's two portrait pairs are two
		// people (the writable seeds are the cross-person pair, never
		// the read-only fixture's), so the unnamed listing converges
		// to two clusters of two faces each. Each cluster's detail
		// names its faces' files, so the story can tell the pairs
		// apart: the writable pair's cluster is the removal's
		// casualty, the read-only pair's the survivor.
		type UnnamedItem = { clusterId: string; faceCount: number; representativeFaceId: string };
		await expect
			.poll(
				async () => {
					const listing = await page.request.get('/api/v1/people/unnamed');
					expect(listing.status()).toBe(200);
					const body = (await listing.json()) as { items?: UnnamedItem[] };
					return (body.items ?? []).map((item) => item.faceCount).sort((a, b) => b - a);
				},
				{ timeout: 420_000, intervals: [1000, 2000, 5000] }
			)
			.toEqual([2, 2]);

		const listing = await page.request.get('/api/v1/people/unnamed');
		expect(listing.status()).toBe(200);
		const clusters = ((await listing.json()) as { items: UnnamedItem[] }).items;
		expect(clusters).toHaveLength(2);
		const rwNames = [PHOTO_RW_A_NAME, PHOTO_RW_B_NAME];
		const roNames = [PHOTO_A_NAME, PHOTO_B_NAME];
		for (const cluster of clusters) {
			const detail = await page.request.get(`/api/v1/people/${cluster.clusterId}`);
			expect(detail.status()).toBe(200);
			const body = (await detail.json()) as {
				faces: Array<{ asset: { originalFilename: string } }>;
			};
			expect(body.faces).toHaveLength(2);
			const names = body.faces.map((face) => face.asset.originalFilename);
			// The separation itself: each cluster's faces come wholly
			// from one pair -- the cross-person evidence riding the
			// identification.
			if (names.every((name) => rwNames.includes(name))) {
				rwClusterId = cluster.clusterId;
				rwRepFaceId = cluster.representativeFaceId;
			} else if (names.every((name) => roNames.includes(name))) {
				roClusterId = cluster.clusterId;
			}
		}
		expect(rwClusterId).not.toBe('');
		expect(rwRepFaceId).not.toBe('');
		expect(roClusterId).not.toBe('');

		// The naming main path: the writable pair's cluster is named
		// (its card addressed by the representative's crop), so the
		// people page (the named surface) carries a card whose faces
		// sit wholly on the purged assets -- the removal's
		// people-convergence witness. The unnamed region keeps the
		// read-only pair's cluster alone.
		await page.goto('/people/unnamed');
		const unnamedCards = page.locator('a.cluster-card');
		await expect(unnamedCards).toHaveCount(2);
		const rwUnnamedCard = unnamedCards.filter({
			has: page.locator(`img[src="/api/v1/faces/${rwRepFaceId}/thumbnail"]`)
		});
		await expect(rwUnnamedCard).toHaveCount(1);
		// The card is an ENTRANCE into the person detail's unnamed
		// variant, whose naming entry opens the dialog.
		await rwUnnamedCard.click();
		await expect(page.locator('.person-profile__name')).toHaveText('Unnamed');
		await page.getByRole('button', { name: 'Name this person', exact: true }).first().click();
		const naming = page.getByRole('dialog', { name: 'Name this person', exact: true });
		await expect(naming).toBeVisible();
		await naming.getByLabel('Name').fill('Field person');
		await naming.getByRole('button', { name: 'Save name', exact: true }).click();
		await expect(naming).toHaveCount(0);
		await expect(page.locator('.person-profile__name')).toHaveText('Field person');
		await page.goto('/people/unnamed');
		await expect(page.locator('a.cluster-card')).toHaveCount(1, { timeout: 30_000 });

		// The member's side of the grant: the wall's shared group
		// carries the writable album, and the member's timeline (the
		// union's shared arms) now holds all four photos -- the two
		// albums' pairs. The member's own removal attempt answers the
		// honest 403 (the two-tier gate's sharee arm) and lands
		// nothing: the album stands. The context replays the member's
		// saved storage state (scene 8's sign-in owns it).
		const context = await browser.newContext({
			locale: 'en-US',
			timezoneId: 'UTC',
			storageState: memberStatePath
		});
		const memberPage = await context.newPage();
		await memberPage.goto('/albums');
		const memberCard = memberPage.locator('a.album-card').filter({ hasText: RW_ALBUM_TITLE });
		await expect(memberCard).toBeVisible();
		await expect(memberCard.locator('.album-card__count')).toHaveAttribute('aria-label', '2 photos');
		await memberPage.goto('/timeline');
		await expect(memberPage.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(memberPage.getByRole('link', { name: PHOTO_RW_B_NAME, exact: true })).toBeVisible();
		await expect(memberPage.locator('.tile')).toHaveCount(4);

		const attempt = await memberPage.request.delete(`/api/v1/albums/${rwAlbumId}`);
		expect(attempt.status()).toBe(403);
		const attemptBody = (await attempt.json()) as { code?: string };
		expect(attemptBody.code).toBe('AUTH_FORBIDDEN');
		const standing = await memberPage.request.get(`/api/v1/albums/${rwAlbumId}`);
		expect(standing.status()).toBe(200);
		await context.close();
	});

	test('scene 15: the removal un-ingests — the sources stay and every held surface converges', async ({
		page,
		browser
	}) => {
		// The held surfaces, opened BEFORE the write so their
		// convergence is the live kind (the SSE subscriptions ride
		// the events the removal publishes; no reload happens): the
		// member's timeline (the union's shared arm), the admin's
		// search wall over the writable pair's names, and the admin's
		// people page over the named person. The member's context
		// replays their saved storage state.
		const memberContext = await browser.newContext({
			locale: 'en-US',
			timezoneId: 'UTC',
			storageState: memberStatePath
		});
		const memberPage = await memberContext.newPage();
		await memberPage.goto('/timeline');
		await expect(memberPage.locator('.tile')).toHaveCount(4);

		const searchPage = await page.context().newPage();
		await searchPage.goto('/search?query=IMG_E6');
		// The unified search's semantic path ranks the whole visible
		// library beside the text hits (the m3-search story's pin), so
		// the writable pair answers among the results -- the held wall
		// carries them until the removal's events converge it.
		await expect(searchPage.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(searchPage.getByRole('link', { name: PHOTO_RW_B_NAME, exact: true })).toBeVisible();

		const peoplePage = await page.context().newPage();
		await peoplePage.goto('/people');
		await expect(peoplePage.locator('a.person-card').filter({ hasText: 'Field person' })).toBeVisible();

		// The removal walk: the detail bar's remove action (the
		// directory arm's danger affordance) opens its own confirm --
		// THE PINNED SEMANTICS in the body -- and the confirm drives
		// the DELETE. Success leaves for the wall.
		await page.goto(`/albums/${rwAlbumId}`);
		await expect(page.getByRole('button', { name: 'Remove album', exact: true })).toBeVisible();
		await page.getByRole('button', { name: 'Remove album', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Remove this album?', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText('library data will be removed');
		await expect(dialog).toContainText('the source folder on disk stays untouched');

		const removalAnswered = page.waitForResponse(
			(response) =>
				response.request().method() === 'DELETE' && response.url().endsWith(`/api/v1/albums/${rwAlbumId}`)
		);
		await dialog.getByRole('button', { name: 'Remove album', exact: true }).click();
		const answer = await removalAnswered;
		expect(answer.status()).toBe(204);

		// The wall's fresh answer: the removed album's card is gone,
		// exactly the sync default and the read-only album remain.
		await expect(page).toHaveURL(/\/albums$/);
		await expect(page.locator('a.album-card').filter({ hasText: RW_ALBUM_TITLE })).toHaveCount(0);
		await expect(page.locator('a.album-card').filter({ hasText: ALBUM_TITLE })).toBeVisible();
		await expect(page.locator('a.album-card')).toHaveCount(2);

		// The member's held timeline: the shared tiles drop live (the
		// removal's events ride the member's own stream), the
		// read-only pair stays -- the audience died with the album,
		// and so did the member's timeline arm.
		await expect(memberPage.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toHaveCount(0, {
			timeout: 30_000
		});
		await expect(memberPage.getByRole('link', { name: PHOTO_RW_B_NAME, exact: true })).toHaveCount(0);
		await expect(memberPage.locator('.tile')).toHaveCount(2, { timeout: 30_000 });
		await expect(memberPage.getByRole('link', { name: PHOTO_A_NAME, exact: true })).toBeVisible();
		await expect(memberPage.getByRole('link', { name: PHOTO_B_NAME, exact: true })).toBeVisible();

		// The member's wall: the shared card is gone with the album
		// (the read-only grant is untouched).
		await memberPage.goto('/albums');
		await expect(memberPage.locator('a.album-card').filter({ hasText: RW_ALBUM_TITLE })).toHaveCount(0);
		await expect(memberPage.locator('a.album-card').filter({ hasText: ALBUM_TITLE })).toBeVisible();
		await memberContext.close();

		// The admin's held search wall: the purged tiles fall (the
		// refetch IS the search's convergence) and the answer shrinks
		// to the surviving read-only pair.
		await expect(searchPage.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toHaveCount(0, {
			timeout: 30_000
		});
		await expect(searchPage.getByRole('link', { name: PHOTO_RW_B_NAME, exact: true })).toHaveCount(0);
		await expect(searchPage.locator('.tile')).toHaveCount(2, { timeout: 30_000 });
		await searchPage.close();

		// The admin's held people page: the named person's card drops
		// live (the asset.deleted refetch) -- the faces died with the
		// purge, the label row cascaded with the dissolved cluster,
		// and the surface settles at its honest empty state.
		await expect(peoplePage.locator('a.person-card').filter({ hasText: 'Field person' })).toHaveCount(0, {
			timeout: 30_000
		});
		// The empty state's title + body (the settled page splits them).
		await expect(peoplePage.getByText('No people yet', { exact: true })).toBeVisible();
		await expect(peoplePage.getByText('Photos with faces will show up here.')).toBeVisible();
		await peoplePage.close();

		// THE ZERO-CONTACT PIN, the host's own judgment: the source
		// files the removed album referenced are byte for byte the
		// seeds the bootstrap laid down -- the library data went, the
		// family's files did not.
		const [seedA, seedB, sourceA, sourceB] = await Promise.all([
			readFile(RW_SEED_A),
			readFile(RW_SEED_B),
			readFile(path.join(RW_SOURCE_HOST_DIR, PHOTO_RW_A_NAME)),
			readFile(path.join(RW_SOURCE_HOST_DIR, PHOTO_RW_B_NAME))
		]);
		expect(sourceA.equals(seedA)).toBe(true);
		expect(sourceB.equals(seedB)).toBe(true);
	});

	test('scene 16: the aftermath — the library answers empty, the refusals stand, the survivor cluster remains', async ({
		page
	}) => {
		// The removal is not repeatable: the album answers the
		// single-shape 404 both to the read and to the repeat delete,
		// exactly like an unknown id.
		const detail = await page.request.get(`/api/v1/albums/${rwAlbumId}`);
		expect(detail.status()).toBe(404);
		const repeat = await page.request.delete(`/api/v1/albums/${rwAlbumId}`);
		expect(repeat.status()).toBe(404);
		const repeatBody = (await repeat.json()) as { code?: string };
		expect(repeatBody.code).toBe('ALBUM_NOT_FOUND');

		// The refusal matrix's system arm: the sync album is no album
		// resource -- the owner's own removal attempt folds into the
		// same single-shape 404, and the default card stays on the
		// wall (the fresh answer carries exactly the sync card and
		// the read-only album; scene 15's removal is the only
		// absence).
		const wall = await page.request.get('/api/v1/albums');
		expect(wall.status()).toBe(200);
		const wallBody = (await wall.json()) as { albums?: Array<{ id?: string; type?: string }> };
		const syncAlbum = (wallBody.albums ?? []).find((card) => card.type === 'sync');
		expect(syncAlbum?.id).toBeTruthy();
		const syncDelete = await page.request.delete(`/api/v1/albums/${syncAlbum?.id}`);
		expect(syncDelete.status()).toBe(404);
		const syncBody = (await syncDelete.json()) as { code?: string };
		expect(syncBody.code).toBe('ALBUM_NOT_FOUND');
		await page.goto('/albums');
		await expect(page.locator('a.album-card').filter({ hasText: 'My photos' })).toBeVisible();
		await expect(page.locator('a.album-card')).toHaveCount(2);

		// The library's fresh answers: the aggregated timeline is the
		// read-only pair's alone, and the search's merged answer (the
		// text path's zero hits beside the semantic path's whole
		// visible library) carries exactly the surviving pair -- the
		// purged rows answer nothing on any path.
		await expect
			.poll(async () => {
				const timeline = await page.request.get('/api/v1/assets/timeline');
				expect(timeline.status()).toBe(200);
				const body = (await timeline.json()) as { items?: Array<{ originalFilename?: string }> };
				return (body.items ?? [])
					.map((item) => item.originalFilename ?? '')
					.sort()
					.join(',');
			})
			.toBe([PHOTO_A_NAME, PHOTO_B_NAME].sort().join(','));

		await expect
			.poll(async () => {
				const search = await page.request.get('/api/v1/search?query=IMG_E6');
				expect(search.status()).toBe(200);
				const body = (await search.json()) as { items?: Array<{ originalFilename?: string }> };
				return (body.items ?? [])
					.map((item) => item.originalFilename ?? '')
					.sort()
					.join(',');
			})
			.toBe([PHOTO_A_NAME, PHOTO_B_NAME].sort().join(','));

		// The people surface: the named person is gone (the label row
		// cascaded with the dissolved cluster), the unnamed region
		// carries exactly the read-only pair's cluster, and the
		// purged cluster's detail is a 404 like an unknown id -- the
		// purged photos' faces dropped off the surface whole.
		const named = await page.request.get('/api/v1/people');
		expect(named.status()).toBe(200);
		expect((await named.json()) as unknown[]).toEqual([]);

		const unnamed = await page.request.get('/api/v1/people/unnamed');
		expect(unnamed.status()).toBe(200);
		const unnamedBody = (await unnamed.json()) as { items?: Array<{ clusterId: string }> };
		expect((unnamedBody.items ?? []).map((item) => item.clusterId)).toEqual([roClusterId]);

		const purgedCluster = await page.request.get(`/api/v1/people/${rwClusterId}`);
		expect(purgedCluster.status()).toBe(404);

		// The unnamed region's own page: one cluster card, the
		// read-only pair's.
		await page.goto('/people/unnamed');
		await expect(page.locator('a.cluster-card')).toHaveCount(1);
	});

	test('scene 17: the count beside the image glyph counts photos, never members', async ({ page }) => {
		// The acceptance walk's finding: the wall's count idiom reads
		// "N photos" beside the IMAGE glyph, but it rendered the live
		// member count -- videos included -- so a mixed collection
		// claimed photos it does not hold. The sync album is the only
		// album uploads land in, and this scene uploads one photo and
		// one video: two live members, one photo. (It runs last: the
		// earlier scenes upload nothing, and the extra members disturb
		// no earlier count.)
		const photoUpload = await page.request.post('/api/v1/assets', {
			multipart: {
				deviceAssetId: 'm3c-count-photo',
				fileName: MIXED_PHOTO_NAME,
				deviceCreatedAt: '2026-10-01T09:00:00Z',
				deviceTimeZone: 'UTC',
				photo: {
					name: MIXED_PHOTO_NAME,
					mimeType: 'image/jpeg',
					buffer: await readFile(MIXED_PHOTO_SOURCE)
				}
			}
		});
		expect(photoUpload.status()).toBe(201);
		const videoUpload = await page.request.post('/api/v1/assets', {
			multipart: {
				deviceAssetId: 'm3c-count-video',
				fileName: MIXED_VIDEO_NAME,
				deviceCreatedAt: '2026-10-02T09:00:00Z',
				deviceTimeZone: 'UTC',
				photo: {
					name: MIXED_VIDEO_NAME,
					mimeType: 'video/mp4',
					buffer: await readFile(MIXED_VIDEO_SOURCE)
				}
			}
		});
		expect(videoUpload.status()).toBe(201);

		// The contract's two numbers, honestly apart: `itemCount` is the
		// collection's own size (the cover pool's cardinality),
		// `photoCount` the image-kind members the count idiom renders.
		const wall = await page.request.get('/api/v1/albums');
		expect(wall.status()).toBe(200);
		const wallBody = (await wall.json()) as {
			albums?: Array<{ id: string; type: string; itemCount: number; photoCount: number }>;
		};
		const syncCard = (wallBody.albums ?? []).find((card) => card.type === 'sync');
		expect(syncCard?.itemCount).toBe(2);
		expect(syncCard?.photoCount).toBe(1);

		const syncId = syncCard?.id ?? '';
		expect(syncId).not.toBe('');
		const detail = await page.request.get(`/api/v1/albums/${syncId}?limit=50`);
		expect(detail.status()).toBe(200);
		const detailBody = (await detail.json()) as { itemCount: number; photoCount: number };
		expect(detailBody.itemCount).toBe(2);
		expect(detailBody.photoCount).toBe(1);

		// The wall's card renders the PHOTO count beside the image
		// glyph (the singular plural form proves the number is 1).
		await page.goto('/albums');
		const card = page.locator('a.album-card').filter({ hasText: 'My photos' });
		await expect(card.locator('.album-card__count')).toHaveAttribute('aria-label', '1 photo');

		// The detail bar renders the same number over a wall that shows
		// BOTH members: the count and the grid are never one number
		// pretending to be the other -- the video is a tile with its
		// own glyph, never a photo.
		await page.goto(`/albums/${syncId}`);
		await expect(page.locator('.album-detail__count')).toHaveAttribute('aria-label', '1 photo');
		await expect(page.locator('a.tile__link')).toHaveCount(2);
		await expect(page.locator('.tile__badge--video')).toHaveCount(1);
	});
});
