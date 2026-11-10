// The albums-as-sources story of the web app (reshaped by the
// album-autonomy rework), walked end to end against the real compose stack:
// an album IS a source. The create dialog picks an EXISTING directory
// on the server's filesystem (referenced in place -- zero copy, zero
// move), the rescan reconciles the album's membership with what that
// directory holds, the audience is an album property (the edit
// dialog's audience face, the full-set PUT), and the member's side of
// the share is the read-only view. Both pickers (the path completion
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
//     cross-person portraits, never the read-only fixture's pair, so
//     the rescan's owner+checksum dedup never links the two stories'
//     rows) -- and the trash scene's restore returns them, so the
//     tree self-heals.
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
// everything the audience grant gives: the wall's shared section,
// the read-only detail, the timeline's inclusion of the shared items,
// the viewer drawer's identity-aware read-only state, and the
// creation guard's refusal matrix (the member's wall carries no
// create affordance; their create-album attempt and their completion
// call both answer AUTH_FORBIDDEN).
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
//     audience stays the honest "Only you", and the POST carries the
//     pinned triple (title, rootPath, audience). The write probe
//     answers sourceWritable false -- the creation FEEDBACK carries
//     the read-only note and the Done affordance acknowledges it
//     (the honest report face; the card lands with the honest zero
//     count -- creation references, it does not scan).
//  5. The rescan: the trigger POSTs the run, the aria-live progress
//     line converges to complete through the job events, and the
//     album's membership becomes honest -- two tiles, two photos,
//     the wall ready.
//  6. The task panel: the rescan's job_runs row lists with its own
//     label (the contract's JobRunKind carries album_rescan -- the
//     row must never fall to the panel's generic failure).
//  7. The edit dialog: the autonomy block -- the lock
//     toggle and the caller's own timeline-exclusion toggle ride the
//     dialog, while the deletability row DOES NOT render (the
//     read-only source carries no row at all -- no dead affordance);
//     the audience face then grants the member the view (the source
//     row fixed, the picker, the drain collapse, the full-set PUT,
//     the unchanged title writing nothing, the chip reseed).
//  8. The member perspective: the member signs in through their own
//     form, the wall's "Shared with you" section carries the album,
//     the wall renders NO creation affordance (the guard's visual
//     half), the member's create-album attempt and completion call
//     both answer AUTH_FORBIDDEN (the refusal matrix's new arms),
//     the shared detail renders read-only (no owner affordance; the
//     exclusion toggle is the member's own), and the member's
//     timeline includes the shared items (the union's shared arm).
//  9. The viewer's identity-aware drawer: a shared photo's info
//     drawer renders the reads and the per-user favorite without any
//     owner-only write affordance.
// 10. The writable album's creation (the probe's true arm): the
//     completion's exact-match row for the writable fixture root, the
//     same pinned triple, and sourceWritable true -- the writable,
//     all-landed creation closes the dialog DIRECTLY (no report
//     face), and the card lands.
// 11. The writable album's rescan converges the seeded pair, and the
//     edit dialog's deletability row RENDERS (checked on by default)
//     beside the lock and exclusion toggles.
// 12. The deletability chain: the deletable wall curates a
//     DELETE-ONLY selection (the hover check, the batch bar offering
//     Trash alone), the trash moves the file out of the source, the
//     album's live membership converges to the survivor, the trash
//     view carries the row, and the restore returns the file -- the
//     album's fresh answer carries both photos again (the fixture
//     healed).
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story --
// see playwright.config.ts). Scenes 1-7 and 10-12 share the admin's
// browser session (scene 1 signs in and saves the storage state);
// scene 8 signs the member in through a fresh context and saves their
// own; scene 9 replays the member's. Each scene navigates explicitly
// to its starting route, so a failure report names the exact scene
// that broke.

import { copyFile, mkdir, readdir, rm } from 'node:fs/promises';
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
const ADMIN_PASSWORD = 'correct-horse-battery';
const MEMBER_EMAIL = 'grace@m3c-e2e.invalid';
const MEMBER_PASSWORD = 'correct-horse-battery';

// The read-only source fixture (the stack arrangement above): the
// bind the compose file mounts :ro, and the directory under it the
// story's first album references in place.
const MOUNT_ROOT = '/mnt/e2e-source';
const SOURCE_DIR = '/mnt/e2e-source/coastal-week';

// The writable sibling (the stack arrangement above): the bind
// without :ro, and the seeded photos the deletability chain walks.
// The seed bytes are the cross-person portrait pair -- genuinely
// distinct from the read-only fixture's pair, so the rescan's
// owner+checksum dedup never links the two stories' rows.
const RW_MOUNT_ROOT = '/mnt/e2e-source-rw';
const RW_SEED_A = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other.jpg');
const RW_SEED_B = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other-second.jpg');
const RW_SOURCE_HOST_DIR = path.join(specDir, 'fixtures/m3-source-rw');

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

// The download bound: the cold upstream transfer of the group (about
// 351 MB) is minutes, the warm re-verification is seconds; both
// allow the cold bound.
const DOWNLOAD_TIMEOUT_MS = 900_000;

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

// The member's user id, captured by the bootstrap so the audience
// write's request shape can be pinned against the real id.
let memberUserId = '';

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
		data: { name: 'M3c E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);

	const login = await api.post('/api/v1/auth/login', {
		data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(login.status()).toBe(200);

	const member = await api.post('/api/v1/admin/users', {
		data: { name: 'Grace Hopper', email: MEMBER_EMAIL, password: MEMBER_PASSWORD }
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
	// album leads the "Your albums" section with the localized
	// display name (the stored internal title never renders) and the
	// honest zero count, and the create action stays beside it -- the
	// ADMINISTRATOR'S entry (an admin session alone renders it).
	const myPhotos = page.locator('a.album-card').filter({ hasText: 'My photos' });
	await expect(myPhotos).toBeVisible();
	await expect(myPhotos).toContainText('0 photos');
	await expect(page.locator('a.album-card')).toHaveCount(1);
	await expect(page.getByText('No albums yet. Create one to gather photos around a moment.')).toBeHidden();
	await expect(page.getByRole('button', { name: 'New album', exact: true })).toBeVisible();

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

	test("scene 4: the read-only album's creation carries the probe's honest report face", async ({ page }) => {
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

		// The POST shape: the pinned triple, pinned against the real
		// request the page sends.
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

		// The report face holds the dialog: the read-only note names
		// what the fact means (the photos cannot be deleted through
		// the application), and the Done affordance acknowledges it.
		// (The skipped half of the report face is empty here -- the
		// audience was -- so the note is the face's whole content.)
		await expect(
			dialog.getByText(
				'The album was created. This folder is read-only — its photos cannot be deleted through the app.'
			)
		).toBeVisible();
		await dialog.getByRole('button', { name: 'Done', exact: true }).click();
		await expect(dialog).toHaveCount(0);

		// The SSE album.created echo converges the wall: the card
		// lands with the honest zero count -- creation references the
		// source, it does not scan it.
		const card = page.locator('a.album-card').filter({ hasText: ALBUM_TITLE });
		await expect(card).toBeVisible();
		await expect(card).toContainText('0 photos');

		// The created album's id, for the later scenes' deep links.
		albumId = (await card.getAttribute('href'))?.replace('/albums/', '') ?? '';
		expect(albumId).not.toBe('');
	});

	test("scene 5: the rescan converges the album's membership with the source", async ({ page }) => {
		await page.goto(`/albums/${albumId}`);
		await expect(page.locator('h1.album-bar__title')).toHaveText(ALBUM_TITLE);
		await expect(page.getByText('This album has no photos yet.')).toBeVisible();

		// The rescan trigger: the POST answers the run's identifier
		// (202 -- the freshly created run).
		const rescanPosted = page.waitForResponse(
			(response) =>
				response.request().method() === 'POST' && response.url().endsWith(`/api/v1/albums/${albumId}/rescan`)
		);
		await page.getByRole('button', { name: 'Rescan', exact: true }).click();
		const answer = await rescanPosted;
		expect(answer.status()).toBe(202);
		const runBody = (await answer.json()) as { jobRunId?: string };
		expect(runBody.jobRunId).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/);

		// The aria-live progress line converges through the job events
		// the run publishes: the queued/running states pass, the
		// finish lands the run's own word.
		const progress = page.locator('[data-testid="album-rescan-progress"]');
		await expect(progress).toBeVisible();
		await expect(progress).toContainText('Rescan complete.', { timeout: 120_000 });

		// The membership is honest: the bar's count and the wall's
		// tiles converge onto the two fixture photos.
		await expect(page.locator('.album-bar .album-detail__count')).toContainText('2 photos', {
			timeout: 120_000
		});
		await expect(page.locator('.tile')).toHaveCount(2, { timeout: 120_000 });
		await expect(page.getByRole('link', { name: PHOTO_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_B_NAME, exact: true })).toBeVisible();

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

		// The autonomy block: the LOCK toggle and the
		// caller's own TIMELINE-EXCLUSION toggle ride the dialog --
		// but the DELETABILITY row does not render at all. The
		// read-only source carries no row: no affordance exists for a
		// write the application can never perform (the probe's false
		// fact kills the row, scene 4's report face said why).
		const autonomy = dialog.locator('[data-testid="album-autonomy"]');
		await expect(autonomy).toBeVisible();
		await expect(autonomy.locator('[data-testid="album-lock-toggle"]')).toBeVisible();
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
		await expect(picker.getByRole('option', { name: MEMBER_EMAIL })).toBeVisible();
		await picker.press('Enter');
		await expect(dialog.locator('.audience-field__chip')).toContainText(MEMBER_EMAIL);
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
		).toContainText(MEMBER_EMAIL);
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

	// The wall's "Shared with you" section carries the album with the
	// owner attribution and the honest count.
	await page.goto('/albums');
	const card = page.locator('a.album-card').filter({ hasText: ALBUM_TITLE });
	await expect(page.getByRole('heading', { level: 2, name: 'Shared with you' })).toBeVisible();
	await expect(card).toBeVisible();
	await expect(card).toContainText(`Shared by ${ADMIN_EMAIL}`);
	await expect(card).toContainText('2 photos');

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
	// renders (the properties dialog and the rescan are the owner's),
	// while the timeline-exclusion toggle -- the member's own
	// preference -- does.
	await card.click();
	await expect(page).toHaveURL(new RegExp(`/albums/${albumId}$`));
	await expect(page.locator('h1.album-bar__title')).toHaveText(ALBUM_TITLE);
	for (const label of ['Properties', 'Rescan', 'Delete album']) {
		await expect(page.getByRole('button', { name: label, exact: true })).toHaveCount(0);
	}
	await expect(page.getByRole('button', { name: 'Keep out of timeline', exact: true })).toBeVisible();
	await expect(page.locator('.tile')).toHaveCount(2);

	// The member's timeline includes the shared items (the union's
	// shared arm -- the member owns nothing of their own).
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
		await expect(page.getByText(PHOTO_A_NAME)).toBeVisible();

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
		for (const label of ['Edit', 'Hide', 'Unhide', 'Trash', 'Restore', 'Delete permanently']) {
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

// The writable arm of the story (the deletability chain):
// the admin's session replays; the scenes walk the probe's TRUE
// fact, the deletability row it renders, the delete-only selection
// it arms, and the trash/restore round trip that moves the source
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

		// The pinned triple, the same shape as the read-only story's.
		const createPost = page.waitForRequest(
			(request) => request.method() === 'POST' && request.url().endsWith('/api/v1/albums')
		);
		await dialog.getByRole('button', { name: 'Create album', exact: true }).click();
		const post = await createPost;
		expect(post.postDataJSON()).toEqual({
			title: RW_ALBUM_TITLE,
			rootPath: RW_MOUNT_ROOT,
			audience: [],
			// The picked completion row's writable fact rendered the
			// deletability row (checked on, the server's default): the
			// switch rides the creation request.
			allowAppDeletion: true,
			inTimeline: true
		});

		// The probe's TRUE arm: the writable bind answers
		// sourceWritable true, and an all-landed, writable creation
		// holds NO report -- the dialog closes directly (the
		// acknowledgment face belongs to the honest problems alone).
		const response = await post.response();
		expect(response?.status()).toBe(201);
		const createBody = (await response?.json()) as { sourceWritable?: boolean };
		expect(createBody.sourceWritable).toBe(true);
		await expect(dialog).toHaveCount(0);

		// The card lands with the honest zero count.
		const card = page.locator('a.album-card').filter({ hasText: RW_ALBUM_TITLE });
		await expect(card).toBeVisible();
		await expect(card).toContainText('0 photos');

		// The writable album's id, for the later scenes' deep links.
		rwAlbumId = (await card.getAttribute('href'))?.replace('/albums/', '') ?? '';
		expect(rwAlbumId).not.toBe('');
	});

	test("scene 11: the writable album's rescan converges and the deletability row renders", async ({ page }) => {
		await page.goto(`/albums/${rwAlbumId}`);
		await expect(page.locator('h1.album-bar__title')).toHaveText(RW_ALBUM_TITLE);
		await expect(page.getByText('This album has no photos yet.')).toBeVisible();

		// The rescan: the trigger POSTs the run and the aria-live
		// progress line converges to complete through the job events
		// (the same walk as the read-only story's scene 5).
		const rescanPosted = page.waitForResponse(
			(response) =>
				response.request().method() === 'POST' && response.url().endsWith(`/api/v1/albums/${rwAlbumId}/rescan`)
		);
		await page.getByRole('button', { name: 'Rescan', exact: true }).click();
		const answer = await rescanPosted;
		expect(answer.status()).toBe(202);

		const progress = page.locator('[data-testid="album-rescan-progress"]');
		await expect(progress).toBeVisible();
		await expect(progress).toContainText('Rescan complete.', { timeout: 120_000 });

		// The membership converges onto the seeded pair, and the
		// scan's pipeline runs both files to ready (the models are
		// warm; the fan-out is the same one the read-only story
		// waited for).
		await expect(page.locator('.tile')).toHaveCount(2, { timeout: 120_000 });
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

		// The deletability row RENDERS for the writable source (the
		// edit dialog's third toggle, present exactly when the probe
		// answered true), on by default beside the lock and exclusion
		// toggles -- the autonomy block's full shape.
		await page.getByRole('button', { name: 'Properties', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Album properties', exact: true });
		await expect(dialog).toBeVisible();
		const autonomy = dialog.locator('[data-testid="album-autonomy"]');
		await expect(autonomy.locator('[data-testid="album-lock-toggle"]')).toBeVisible();
		await expect(autonomy.locator('[data-testid="album-timeline-toggle"]')).toBeVisible();
		const deletableToggle = autonomy.locator('[data-testid="album-deletable-toggle"]');
		await expect(deletableToggle).toBeVisible();
		await expect(deletableToggle).toBeChecked();
		await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
		await expect(dialog).toHaveCount(0);
	});

	test("scene 12: the deletable wall trashes through the selection bar and the trash view's restore returns the file", async ({
		page
	}) => {
		await page.goto(`/albums/${rwAlbumId}`);
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_RW_B_NAME, exact: true })).toBeVisible();

		// The delete-only selection (the wall's capability gate rides
		// the server-derived `deletable` face -- owner AND writable
		// AND the switch AND unlocked, all four): the hover-reveal
		// check is the selection entry, one click selects the tile.
		await page.getByRole('checkbox', { name: `Select: ${PHOTO_RW_A_NAME}`, exact: true }).click();

		// The batch bar rises with the DELETE-ONLY subset: Trash is
		// the one action it offers (the album's wall curates deletion
		// alone -- every other curatable wall's action is absent).
		const bar = page.getByRole('toolbar', { name: '1 selected' });
		await expect(bar).toBeVisible();
		await expect(bar.getByRole('button', { name: 'Trash', exact: true })).toBeVisible();
		for (const label of ['Favorite', 'Unfavorite', 'Hide', 'Unhide', 'Restore', 'Add tag']) {
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
		await expect(page.locator('.album-bar .album-detail__count')).toContainText('1 photo');
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
		await expect(page.getByRole('heading', { level: 1, name: 'Trash', exact: true })).toBeVisible();
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(page.getByText(/^(Expiring soon|\d+ days? left)$/)).toBeVisible();

		// The restore moves the file BACK to the recorded original
		// location (the external lifecycle's restore arm): the row
		// leaves the trash view and the view settles empty.
		await page.getByRole('button', { name: 'Restore', exact: true }).click();
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toHaveCount(0);
		await expect(
			page.getByText('The trash is empty. Photos you delete appear here before they are removed for good.')
		).toBeVisible();

		// The album's fresh answer carries both photos again: the
		// restored row rejoined the membership (the restore publishes
		// the same items-changed signal, so a wall that stayed on the
		// page would restart its walk too), the count is honest, and
		// the source tree holds both files for the next run (the
		// fixture healed -- the bootstrap's re-seed is a no-op copy).
		await page.goto(`/albums/${rwAlbumId}`);
		await expect(page.locator('.tile')).toHaveCount(2, { timeout: 60_000 });
		await expect(page.getByRole('link', { name: PHOTO_RW_A_NAME, exact: true })).toBeVisible();
		await expect(page.locator('.album-bar .album-detail__count')).toContainText('2 photos');
	});
});
