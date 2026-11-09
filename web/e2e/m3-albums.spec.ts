// The albums-as-sources story of the web app (E4), walked end to
// end against the real compose stack: an album IS a source. The
// create dialog picks an EXISTING directory on the server's
// filesystem (referenced in place -- zero copy, zero move), the
// rescan reconciles the album's membership with what that
// directory holds, the audience is an album property (the edit
// dialog's audience face, the full-set PUT), and the member's side
// of the share is the read-only view. Both pickers (the path
// completion and the audience candidates) are PORTALED floating
// layers (web IA §7.5's 2026-11-09 revision) -- anchored under
// their triggers, living in the document outside the dialogs' DOM,
// so the dialogs never change height for them. The stack's
// lifecycle is owned by the caller (the acceptance scripts'
// chaining pattern, or one fresh stack per spec -- the first-run
// bootstrap below expects its 201); the spec itself only talks to
// http://localhost:8244.
//
// THE STACK ARRANGEMENT (the honest source fixture, documented):
// the repository's compose file binds
// web/e2e/fixtures/m3-source into the lumemor container READ-ONLY
// at /mnt/e2e-source (see that directory's README). The story's
// directory album points at coastal-week under that root -- the
// two face-portrait JPEGs the upload stories seed, arriving
// through the source path this time. The read-only mount is the
// zero-copy pin held by construction: a scan that tried to write
// into the source would fail loudly against it. The mount rides
// ONLY the repository's own compose file (the dogfood stack
// composes its own file elsewhere and never sees it), and it is
// inert until an administrator opens the root in the instance's
// mount-root allowlist -- the default allowlist is empty, which is
// exactly where this story starts. The models ride the SAME
// storage bind mount the chained acceptances downloaded into --
// ./lumemor persists across stack lifetimes, so the download run
// the bootstrap triggers is warm (a digest re-verification); cold
// (CI's first run), it is the real transfer, bounded like every
// M3 story's cold bound.
//
// THE TWO-ACCOUNT DANCE (the sharing story's shape): the first-run
// administrator owns the library and the album; a member account,
// created through the admin API in the suite's bootstrap, becomes
// the audience. The story's scenes sign each account in through
// the login form in SEPARATE browser contexts -- the admin's
// storage state first, then the member's own sign-in scene (the
// two sessions never share a context) -- and the member's scenes
// assert everything the audience grant gives: the wall's shared
// section, the read-only detail, the timeline's inclusion of the
// shared items, and the viewer drawer's identity-aware read-only
// state.
//
// THE SCENES (the E4 surface, in order):
//
//  1. The administrator signs in through the form and lands on the
//     albums wall holding exactly the pinned "My photos" card (the
//     default sync album -- a fresh account's wall is never empty;
//     the count is honest at zero).
//  2. The library bootstrap: the model group's download run is
//     driven to succeeded through the admin API (the scan's
//     pipeline needs it; this story uploads nothing -- the album's
//     members arrive through the source).
//  3. The create dialog's closed posture: with no mount root open
//     (the fresh instance's default), the path completion answers
//     the honest no-roots note -- the refusal face of the empty
//     allowlist -- and the administrator's face carries the direct
//     jump to the instance settings (the 2026-11-09 dogfood pin).
//  4. The instance settings open the mount root: the "Album
//     sources" section's closed posture, the hand-typed first root
//     (the allowlist is empty, so nothing completes yet), the
//     complete-replacement save, and the live-domain proof that
//     the root now answers the completion.
//  5. The create flow: the path completion lists coastal-week for
//     a typed prefix, Enter completes it, the audience stays the
//     honest "Only you", and the POST carries the pinned triple
//     (title, rootPath, audience) -- the card lands with the
//     honest zero count (creation references, it does not scan).
//  6. The rescan: the trigger POSTs the run, the aria-live
//     progress line converges to complete through the job events,
//     and the album's membership becomes honest -- two tiles, two
//     photos, the wall ready.
//  7. The task panel: the rescan's job_runs row lists with its own
//     label (the contract's JobRunKind carries album_rescan -- the
//     row must never fall to the panel's generic failure).
//  8. The audience grant: the edit dialog's audience face -- the
//     source row reads fixed, the picker offers the member, the
//     add drains the two-account pool and collapses the picker
//     (§7.6's 2026-11-09 pin: an empty picker state is never
//     shown), the save PUTs the full set (and the unchanged title
//     writes nothing), and the chip reseeds from the refetched
//     shares with the add affordance honestly disabled.
//  9. The member perspective: the member signs in through their
//     own form, the wall's "Shared with you" section carries the
//     album, the shared detail renders read-only (no owner
//     affordance; the exclusion toggle is the member's own), and
//     the member's timeline includes the shared items (the
//     union's shared arm).
// 10. The viewer's identity-aware drawer: a shared photo's info
//     drawer renders the reads and the per-user favorite without
//     any owner-only write affordance.
//
// THE STATE HANDOFF: the scenes are serial (one stack, one story --
// see playwright.config.ts). Scenes 1-8 share the admin's browser
// session (scene 1 signs in and saves the storage state); scene 9
// signs the member in through a fresh context and saves their own;
// scene 10 replays the member's. Each scene navigates explicitly
// to its starting route, so a failure report names the exact scene
// that broke.

import { fileURLToPath } from 'node:url';
import { expect, test } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

// The story's two accounts: the first-run administrator (the setup
// API's, the album's owner) and the member the administrator
// provisions through the admin API in the suite's bootstrap. The
// stack is always fresh, so the fixed identities are deterministic.
const ADMIN_EMAIL = 'admin@m3c-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';
const MEMBER_EMAIL = 'grace@m3c-e2e.invalid';
const MEMBER_PASSWORD = 'correct-horse-battery';

// The source fixture (the stack arrangement above): the mount root
// the compose file binds read-only, and the directory under it the
// story's album references in place.
const MOUNT_ROOT = '/mnt/e2e-source';
const SOURCE_DIR = '/mnt/e2e-source/coastal-week';

// The two fixture photos the album gathers (the source tree's own
// names -- the story asserts them as the tiles' links).
const PHOTO_A_NAME = 'IMG_E4_0001.jpg';
const PHOTO_B_NAME = 'IMG_E4_0002.jpg';

// The story's album title.
const ALBUM_TITLE = 'Coastal week 2026';

// The download bound: the cold upstream transfer of the group (about
// 351 MB) is minutes, the warm re-verification is seconds; both
// allow the cold bound.
const DOWNLOAD_TIMEOUT_MS = 900_000;

// The storage-state files scene 1 (the admin) and scene 9 (the
// member) write and the later scenes replay (git-ignored, recreated
// per run).
const adminStatePath = `${specDir}/.auth/m3c-admin-state.json`;
const memberStatePath = `${specDir}/.auth/m3c-member-state.json`;

// The created album's id, captured by the create scene through the
// wall card so the later scenes can deep-link deterministically.
let albumId = '';

// The member's user id, captured by the bootstrap so the audience
// write's request shape can be pinned against the real id.
let memberUserId = '';

test.beforeAll(async ({ playwright }) => {
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

test('scene 1: the administrator signs in and lands on the pinned My photos card', async ({
	page
}) => {
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
	// honest zero count, and the create action stays beside it.
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
		// -- the album's members arrive through the source scan -- but
		// the scan's pipeline runs the same metadata/thumbnail/ML
		// fan-out an upload's does, so the models must be ready for
		// the tiles to converge the same way; the admin API is the M2
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

	test('scene 3: the create dialog answers the honest no-roots note while the allowlist is closed', async ({
		page
	}) => {
		await page.goto('/albums');
		await page.getByRole('button', { name: 'New album', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'New album', exact: true });
		await expect(dialog).toBeVisible();

		// Focusing the path field alone asks the completion for the
		// roots (the empty prefix lists them); the fresh instance's
		// allowlist is empty, and the honest answer is the closed
		// posture -- the note that names the administrator's move,
		// never a pretense that the world is empty. The candidate
		// block is a PORTALED floating layer (web IA §7.5's
		// 2026-11-09 revision: fixed, anchored under the input, the
		// dialog never changing height for it), so its rows live in
		// the document at page level, outside the dialog's DOM.
		await dialog.getByLabel('Path').click();
		const suggest = page.locator('.path-completion__suggest');
		await expect(suggest.getByText('No folders can be picked yet — an administrator must first open a mount root in the instance settings.')).toBeVisible();

		// The admin's direct jump (the 2026-11-09 dogfood pin): the
		// no-roots face carries an icon+text action linking to the
		// instance settings, where the "Album sources" section owns
		// the allowlist -- the note is no longer a dead end for the
		// administrator who can act on it. (Scene 4 walks the
		// settings themselves; the link's target is pinned, not
		// walked, so the closing discipline below stays intact.)
		const openRoots = page.getByRole('link', { name: 'Open mount roots' });
		await expect(openRoots).toBeVisible();
		await expect(openRoots).toHaveAttribute('href', '/admin/settings');

		// The closing discipline (the keys footer's promise, walked):
		// the list's Esc closes the list ALONE -- the dialog behind it
		// stays -- and the modal's own Esc owns the next stroke.
		// (The portaled layer leaves the document with the close.)
		await dialog.getByLabel('Path').press('Escape');
		await expect(suggest).toHaveCount(0);
		await expect(dialog).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(dialog).toHaveCount(0);
	});

	test('scene 4: the instance settings open the e2e mount root', async ({ page }) => {
		await page.goto('/admin/settings');
		await expect(page).toHaveTitle('Settings · lumemor');

		// The "Album sources" section renders the same closed posture
		// the create dialog answered: no root is open yet.
		const section = page.locator('[data-testid="admin-settings-mounts"]');
		await expect(section.getByRole('heading', { name: 'Album sources' })).toBeVisible();
		await expect(section.locator('[data-testid="mount-roots-empty"]')).toHaveText(
			'No mount roots yet — directory album creation is closed until one is added.'
		);

		// The first root is typed by hand (the allowlist is empty, so
		// the add row's own completion has nothing to suggest -- its
		// layer renders the quiet no-match note and stays out of the
		// way). The Esc before the click is the settled discipline:
		// the completion's list closes on the input's blur, and the
		// portaled layer leaves the document with the close (the
		// dialog's height never moved for it either way -- §7.5's
		// revision -- so the layout is already settled; the key simply
		// keeps the scene's walk honest).
		const draft = section.getByLabel('Add a root');
		await draft.fill(MOUNT_ROOT);
		await expect(
			page.getByText('No matching folders under the allowed roots.')
		).toBeVisible();
		await draft.press('Escape');
		await section.getByRole('button', { name: 'Add root', exact: true }).click();
		await expect(section.locator('[data-testid="mount-roots-list"]')).toContainText(MOUNT_ROOT);
		await section.getByRole('button', { name: 'Save roots', exact: true }).click();
		await expect(page.getByText('Saved. New directory albums may reference these roots.')).toBeVisible();

		// The root is not merely stored but LIVE: the completion's
		// domain (exactly the creation validation's) now answers the
		// mounted root -- the honest proof the fixture mount rides
		// the container before the create dialog leans on it.
		const completions = await page.request.get(
			`/api/v1/albums/path-completions?prefix=${encodeURIComponent(MOUNT_ROOT)}`
		);
		expect(completions.status()).toBe(200);
		const body = (await completions.json()) as { paths?: string[] };
		expect(body.paths).toEqual([MOUNT_ROOT]);
	});

	test('scene 5: the create dialog completes the source path and posts the pinned triple', async ({
		page
	}) => {
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
			audience: []
		});

		// The all-landed creation (nothing to skip) closes the dialog
		// directly, and the SSE album.created echo converges the wall:
		// the card lands with the honest zero count -- creation
		// references the source, it does not scan it.
		await expect(dialog).toHaveCount(0);
		const card = page.locator('a.album-card').filter({ hasText: ALBUM_TITLE });
		await expect(card).toBeVisible();
		await expect(card).toContainText('0 photos');

		// The created album's id, for the later scenes' deep links.
		albumId = (await card.getAttribute('href'))?.replace('/albums/', '') ?? '';
		expect(albumId).not.toBe('');
	});

	test('scene 6: the rescan converges the album\'s membership with the source', async ({ page }) => {
		await page.goto(`/albums/${albumId}`);
		await expect(page.locator('h1.album-bar__title')).toHaveText(ALBUM_TITLE);
		await expect(page.getByText('This album has no photos yet.')).toBeVisible();

		// The rescan trigger: the POST answers the run's identifier
		// (202 -- the freshly created run).
		const rescanPosted = page.waitForResponse(
			(response) =>
				response.request().method() === 'POST' &&
				response.url().endsWith(`/api/v1/albums/${albumId}/rescan`)
		);
		await page.getByRole('button', { name: 'Rescan', exact: true }).click();
		const answer = await rescanPosted;
		expect(answer.status()).toBe(202);
		const runBody = (await answer.json()) as { jobRunId?: string };
		expect(runBody.jobRunId).toMatch(
			/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/
		);

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

		// And the scan's pipeline ran to its end: the reference ingest
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

	test('scene 7: the task panel lists the rescan\'s row with its own label', async ({ page }) => {
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

	test('scene 8: the edit dialog\'s audience grants the member the view', async ({ page }) => {
		await page.goto(`/albums/${albumId}`);

		// The PATCH collector: the save must write the audience PUT
		// alone -- the unchanged title writes nothing (the page diffs
		// against the live detail).
		const patchUrls: string[] = [];
		page.on('request', (request) => {
			if (
				request.method() === 'PATCH' &&
				request.url().includes(`/api/v1/albums/${albumId}`)
			) {
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

		// The audience face: the honest "Only you", then the picker
		// (a PORTALED floating layer at page level -- the dialog's
		// DOM never carries it) offers the one other account (the
		// caller's own is filtered out -- the identity seam's self
		// filter), Enter adds the chip, and the DRAIN collapses the
		// picker: this instance holds exactly two accounts, so the
		// member's add empties the candidate pool, and §7.6's
		// 2026-11-09 pin never shows an empty picker -- the layer
		// falls, the add button disables, and the reserved
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
			(request) =>
				request.method() === 'PUT' &&
				request.url().endsWith(`/api/v1/albums/${albumId}/audience`)
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
		// reseeded face also carries §7.6's pin the other way round:
		// with the only other account already granted, the probed
		// pool is KNOWN empty and the add affordance opens nothing.
		await page.getByRole('button', { name: 'Properties', exact: true }).click();
		await expect(page.getByRole('dialog', { name: 'Album properties', exact: true })).toBeVisible();
		await expect(
			page.getByRole('dialog', { name: 'Album properties', exact: true }).locator('.audience-field__chip')
		).toContainText(MEMBER_EMAIL);
		await expect(
			page.getByRole('dialog', { name: 'Album properties', exact: true }).getByRole('button', { name: 'Add', exact: true })
		).toBeDisabled();
		await page
			.getByRole('dialog', { name: 'Album properties', exact: true })
			.getByRole('button', { name: 'Cancel', exact: true })
			.click();
	});
});

// The member half of the story: scene 9 signs the member in through
// their own form (a fresh context -- the two accounts never share a
// browser session) and saves their storage state; scene 10 replays
// it.
test('scene 9: the member signs in and sees the shared album read-only', async ({ browser }) => {
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

	// Hand the member session to scene 10.
	await context.storageState({ path: memberStatePath });
	await context.close();
});

test.describe('the member viewer scenes', () => {
	test.use({ storageState: memberStatePath });

	test("scene 10: the viewer's drawer keeps the reads and the per-user favorite, never the owner-only writes", async ({
		page
	}) => {
		// A shared photo's viewer, deep-linked from the album detail.
		await page.goto(`/albums/${albumId}`);
		await page.getByRole('link', { name: PHOTO_A_NAME, exact: true }).click();
		await expect(page).toHaveURL(new RegExp(`/a/[0-9a-f-]{36}$`));
		await expect(page.getByText(PHOTO_A_NAME)).toBeVisible();

		// The info drawer plus the action bar (the R2-3 rework: the
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
		for (const label of [
			'Edit',
			'Hide',
			'Unhide',
			'Trash',
			'Restore',
			'Delete permanently'
		]) {
			// The owner-only writes never render in either surface:
			// no trash/restore in the BAR (the drawer never carried a
			// curation button since the rework), no description edit.
			await expect(page.getByRole('button', { name: label, exact: true })).toHaveCount(0);
		}

		// The per-user favorite works for the viewer: the flag flips
		// without a page reload (the optimistic per-user state).
		await favorite.click();
		await expect(page.getByRole('button', { name: 'Unfavorite', exact: true })).toBeVisible();
	});
});
