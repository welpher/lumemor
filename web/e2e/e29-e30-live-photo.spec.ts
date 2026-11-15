// The Live Photo's whole life, walked end to end against the real
// compose stack (a fresh lifetime; the spec talks to
// http://localhost:8244 only).
//
// THE STORY: a family's library holds both Live Photo shapes -- a
// paired upload (a still plus its own video file) and a single-file
// "motion photo" (an Android camera file whose video is embedded). The
// wall must show each as ONE tile carrying the type mark, hovering it
// must play the moment, the viewer's first action must play it in
// place and return the still, a download must hand over the pair, and
// the ingest must recognize the embedded shape without anyone asking.
//
// THE SCENES:
//
//  1. The bootstrap: the administrator, the session, one plain photo
//     and one Live Photo pair (seeded through the upload API -- the
//     device path the web never sends).
//  2. The wall: the Live Photo's tile carries the type mark and plays
//     its moment on hover; a plain photo's tile does not.
//  3. The viewer: the action row's FIRST button plays and stops the
//     motion in place, and the stream's own end returns the still.
//  4. The download: one archive of the pair, the motion always inside.
//  5. The single-file ingest: a Pixel-style motion photo uploaded as
//     one file grows its pair silently, and deleting the motion
//     degrades the tile to a plain photo with no error.
//
// THE STATE HANDOFF mirrors the neighbouring story: the server-side
// state is the story, the browser session rides a storage state saved
// after the bootstrap.
import { copyFile, mkdir, mkdtemp, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type APIRequestContext, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const BASE_URL = 'http://localhost:8244';
const ADMIN_EMAIL = 'admin@e29e30-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));
const PLAIN_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/portrait-orientation6.jpg');
const LIVE_PHOTO_SOURCE = path.join(repoRoot, 'server/internal/jobs/testdata/golden.jpg');
const LIVE_VIDEO_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/tiny.mp4');
const PLAIN_NAME = 'LIVE_0002.jpg';
const LIVE_PHOTO_NAME = 'LIVE_0003.jpg';
const LIVE_VIDEO_NAME = 'LIVE_0003.mp4';
const MOTION_PHOTO_NAME = 'MOTION_0001.jpg';

const authStatePath = path.join(specDir, '.auth', 'live-photo-state.json');

let livePhotoId = '';
let liveVideoId = '';
let motionPhotoId = '';
let motionPhotoVideoId = '';
let plainId = '';

/** One wall tile by its file name: a Live Photo's accessible name
 *  carries its type as a suffix ("... · Live Photo"), so the lookup
 *  anchors on the name itself rather than the whole string. */
function tile(page: Page, name: string) {
	return page.locator(`a.tile__link[aria-label="${name}"], a.tile__link[aria-label^="${name} · "]`).first();
}

/** One tile's Live Photo type mark. */
function liveMark(page: Page, name: string) {
	return tile(page, name).locator('.tile__live');
}

/** The viewer's Live Photo toggle (the action row's first button). */
function liveToggle(page: Page) {
	return page.locator('.viewer__bar-actions button').first();
}

/** The wall's tile count: one tile per photo, the hidden motion never
 *  among them. */
function tileCount(page: Page): Promise<number> {
	return page.locator('a.tile__link').count();
}

/**
 * A Pixel-style single-file motion photo: the golden JPEG with an XMP
 * packet declaring the motion container, and the video appended on the
 * tail -- exactly the shape the server's extraction recognizes
 * (size - Length = the video's window).
 */
async function buildPixelMotionPhoto(video: Buffer): Promise<Buffer> {
	const jpeg = await readFile(LIVE_PHOTO_SOURCE);
	const xmp = `<?xpacket begin="\ufeff" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about=""
    xmlns:GCamera="http://ns.google.com/photos/1.0/camera/"
    xmlns:Container="http://ns.google.com/photos/1.0/container/"
    xmlns:Item="http://ns.google.com/photos/1.0/container/item/"
    GCamera:MotionPhoto="1" GCamera:MotionPhotoVersion="1"
    GCamera:MotionPhotoPresentationTimestampUs="0">
   <GCamera:ContainerDirectory>
    <rdf:Seq>
     <rdf:li rdf:parseType="Resource">
      <Container:Directory><rdf:Seq><rdf:li rdf:parseType="Resource">
        <Container:Item Item:Mime="image/jpeg" Item:Semantic="Primary" Item:Length="0" Item:Padding="0"/>
      </rdf:li></rdf:Seq></Container:Directory>
     </rdf:li>
     <rdf:li rdf:parseType="Resource">
      <Container:Directory><rdf:Seq><rdf:li rdf:parseType="Resource">
        <Container:Item Item:Mime="video/mp4" Item:Semantic="MotionPhoto" Item:Length="${video.length}" Item:Padding="0"/>
      </rdf:li></rdf:Seq></Container:Directory>
     </rdf:li>
    </rdf:Seq>
   </GCamera:ContainerDirectory>
  </rdf:Description>
 </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`;
	const header = Buffer.from('http://ns.adobe.com/xap/1.0/\u0000', 'latin1');
	const payload = Buffer.concat([header, Buffer.from(xmp, 'utf8')]);
	const segment = Buffer.from([0xff, 0xe1, (payload.length + 2) >> 8, (payload.length + 2) & 0xff]);
	return Buffer.concat([jpeg.subarray(0, 2), segment, payload, jpeg.subarray(2), video]);
}

/** Wait until the asset detail answers a Live Photo's motion id. */
async function waitForPair(api: APIRequestContext, assetId: string): Promise<string> {
	let motionId = '';
	await expect
		.poll(
			async () => {
				const detail = await api.get(`/api/v1/assets/${assetId}`);
				if (detail.status() !== 200) {
					return '';
				}
				const body = (await detail.json()) as { livePhotoVideoId: string | null };
				motionId = body.livePhotoVideoId ?? '';
				return motionId;
			},
			{ timeout: 90_000 }
		)
		.not.toBe('');
	return motionId;
}

test.beforeAll(async ({ playwright }) => {
	const staging = await mkdtemp(path.join(tmpdir(), 'lumemor-live-e2e-'));
	const plain = path.join(staging, PLAIN_NAME);
	await copyFile(PLAIN_SOURCE, plain);
	await mkdir(path.dirname(authStatePath), { recursive: true });

	// An EMPTY state on purpose: the request context inherits the file's
	// use options, and the file this bootstrap is about to write does not
	// exist yet.
	const api = await playwright.request.newContext({
		baseURL: BASE_URL,
		storageState: { cookies: [], origins: [] }
	});
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'Live E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);

	const login = await api.post('/api/v1/auth/login', {
		data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(login.status()).toBe(200);
	await api.storageState({ path: authStatePath });

	// The plain photo: the wall's control case.
	const plainUpload = await api.post('/api/v1/assets', {
		multipart: {
			deviceAssetId: 'e29e30-plain',
			fileName: PLAIN_NAME,
			deviceCreatedAt: '2023-01-10T12:00:00Z',
			deviceTimeZone: 'UTC',
			photo: { name: PLAIN_NAME, mimeType: 'image/jpeg', buffer: await readFile(plain) }
		}
	});
	expect(plainUpload.status()).toBe(201);
	plainId = ((await plainUpload.json()) as { id: string }).id;

	// The pair: the still plus its own motion file (the iPhone shape).
	const pairUpload = await api.post('/api/v1/assets', {
		multipart: {
			deviceAssetId: 'e29e30-live',
			fileName: LIVE_PHOTO_NAME,
			deviceCreatedAt: '2023-06-06T12:00:00Z',
			deviceTimeZone: 'UTC',
			photo: { name: LIVE_PHOTO_NAME, mimeType: 'image/jpeg', buffer: await readFile(LIVE_PHOTO_SOURCE) },
			liveVideo: {
				name: LIVE_VIDEO_NAME,
				mimeType: 'video/mp4',
				buffer: await readFile(LIVE_VIDEO_SOURCE)
			}
		}
	});
	expect(pairUpload.status()).toBe(201);
	livePhotoId = ((await pairUpload.json()) as { id: string }).id;

	// The single-file motion photo: one upload, no second part.
	const motionBytes = await buildPixelMotionPhoto(await readFile(LIVE_VIDEO_SOURCE));
	const motionUpload = await api.post('/api/v1/assets', {
		multipart: {
			deviceAssetId: 'e29e30-motion',
			fileName: MOTION_PHOTO_NAME,
			deviceCreatedAt: '2023-08-08T12:00:00Z',
			deviceTimeZone: 'UTC',
			photo: { name: MOTION_PHOTO_NAME, mimeType: 'image/jpeg', buffer: motionBytes }
		}
	});
	expect(motionUpload.status()).toBe(201);
	motionPhotoId = ((await motionUpload.json()) as { id: string }).id;

	liveVideoId = await waitForPair(api, livePhotoId);
	motionPhotoVideoId = await waitForPair(api, motionPhotoId);
	await api.dispose();
});

// The signed-in scenes: every one replays the session the bootstrap
// saved (the worker fixture resolves the state file per test, after the
// bootstrap wrote it).
test.describe('the live photo story', () => {
	test.use({ storageState: authStatePath });

	test('scene 1: the wall marks a live photo and leaves a plain photo bare', async ({ page }) => {
		await page.goto('/timeline');
		await expect(tile(page, LIVE_PHOTO_NAME)).toBeVisible({ timeout: 60_000 });
		await expect(tile(page, MOTION_PHOTO_NAME)).toBeVisible({ timeout: 60_000 });
		await expect(tile(page, PLAIN_NAME)).toBeVisible({ timeout: 60_000 });

		// The type mark: a bare glyph at the tile's top-right, named for the
		// screen reader through the tile's own accessible name.
		await expect(liveMark(page, LIVE_PHOTO_NAME)).toBeVisible();
		await expect(liveMark(page, MOTION_PHOTO_NAME)).toBeVisible();
		await expect(tile(page, PLAIN_NAME).locator('.tile__live')).toHaveCount(0);
		await expect(tile(page, LIVE_PHOTO_NAME)).toHaveAttribute('aria-label', `${LIVE_PHOTO_NAME} · Live Photo`);

		// ONE TILE per photo: the hidden motion never becomes a tile of its
		// own, and the count counts the photo once.
		const tiles = page.locator('a.tile__link');
		await expect(tiles.filter({ has: page.locator(`[aria-label^="${LIVE_VIDEO_NAME}"]`) })).toHaveCount(0);
		expect(await tileCount(page)).toBe(3);
	});

	test('scene 2: hovering the live tile plays the moment and flips the mark', async ({ page }) => {
		await page.goto('/timeline');
		await expect(tile(page, LIVE_PHOTO_NAME)).toBeVisible({ timeout: 60_000 });

		// At rest: no stream mounted (the motion costs no byte until asked).
		await expect(page.locator('video.tile__motion')).toHaveCount(0);

		// The component's play stream answers once its transcode drained;
		// the hover's own element then mounts against that stream.
		await expect
			.poll(
				async () => {
					const probe = await page.request.get(`/api/v1/assets/${liveVideoId}/play`, {
						headers: { Range: 'bytes=0-0' }
					});
					return probe.status();
				},
				{ timeout: 90_000 }
			)
			.toBe(206);

		await tile(page, LIVE_PHOTO_NAME).hover();
		const motion = page.locator(`video.tile__motion[src="/api/v1/assets/${liveVideoId}/play"]`);
		await expect(motion).toHaveCount(1);
		// Playback flags are element properties, not attributes.
		await expect(motion).toHaveJSProperty('muted', true);
		await expect(motion).toHaveJSProperty('loop', true);
		await expect(motion).toHaveJSProperty('controls', false);
		// The mark flips to the pause pair once the stream actually plays,
		// and the still takes its slow push.
		await expect(liveMark(page, LIVE_PHOTO_NAME).locator('path')).toHaveCount(1, { timeout: 30_000 });

		// Leaving drops the stream and restores the type glyph.
		await page.mouse.move(0, 0);
		await expect(page.locator('video.tile__motion')).toHaveCount(0);
		await expect(liveMark(page, LIVE_PHOTO_NAME).locator('circle')).toHaveCount(2);
		// The plain tile never asks for a stream.
		await tile(page, PLAIN_NAME).hover();
		await expect(page.locator('video.tile__motion')).toHaveCount(0);
		await expect(liveMark(page, PLAIN_NAME)).toHaveCount(0);
	});

	test('scene 3: the viewer plays the motion in place and returns the still', async ({ page }) => {
		await page.goto('/timeline');
		await expect(tile(page, LIVE_PHOTO_NAME)).toBeVisible({ timeout: 60_000 });
		await tile(page, LIVE_PHOTO_NAME).click();
		await expect(page).toHaveTitle(`${LIVE_PHOTO_NAME} · lumemor`);

		// The action row's FIRST button is the Live Photo's toggle: the
		// play arm at rest, named and unpressed.
		const toggle = liveToggle(page);
		await expect(toggle).toHaveAttribute('aria-label', 'Play the Live Photo');
		await expect(toggle).toHaveAttribute('aria-pressed', 'false');
		await expect(page.locator('video.viewer__live')).toHaveCount(0);

		// One press plays the component's stream inside the still's box.
		await toggle.click();
		const motion = page.locator('video.viewer__live');
		await expect(motion).toBeVisible();
		await expect(motion).toHaveAttribute('src', `/api/v1/assets/${liveVideoId}/play`);
		await expect(liveToggle(page)).toHaveAttribute('aria-pressed', 'true');
		await expect(liveToggle(page)).toHaveAttribute('aria-label', 'Stop the Live Photo');

		// The clip ends by itself and the still comes back -- no second
		// press, no error surfaced.
		await expect(page.locator('video.viewer__live')).toHaveCount(0, { timeout: 30_000 });
		await expect(liveToggle(page)).toHaveAttribute('aria-pressed', 'false');
		await expect(page.locator('[role="alert"]')).toHaveCount(0);

		// Click-click: the second press stops it early.
		await liveToggle(page).click();
		await expect(page.locator('video.viewer__live')).toBeVisible();
		await liveToggle(page).click();
		await expect(page.locator('video.viewer__live')).toHaveCount(0);

		// A plain photo's action row carries no toggle at all.
		await page.keyboard.press('Escape');
		await expect(page).toHaveURL(/\/timeline$/);
		await tile(page, PLAIN_NAME).click();
		await expect(page).toHaveTitle(`${PLAIN_NAME} · lumemor`);
		await expect(page.locator('.viewer__live-toggle')).toHaveCount(0);
	});

	test('scene 4: the download hands over one archive of the pair', async ({ page }) => {
		await page.goto('/timeline');

		// The pair's motion arrived as its own file, so it always rides
		// along: the archive holds both originals under their own names.
		const download = await page.request.get(`/api/v1/assets/${livePhotoId}/original`);
		expect(download.status()).toBe(200);
		expect(download.headers()['content-type']).toBe('application/zip');
		expect(download.headers()['content-disposition']).toContain('LIVE_0003.zip');
		const archive = await download.body();
		expect(archive.includes(Buffer.from(LIVE_PHOTO_NAME, 'latin1'))).toBe(true);
		expect(archive.includes(Buffer.from(LIVE_VIDEO_NAME, 'latin1'))).toBe(true);

		// The same request on the single-file motion photo: its extracted
		// motion is a copy of bytes the still already carries, so the
		// archive holds the still alone until the account asks for it.
		const embeddedOff = await page.request.get(`/api/v1/assets/${motionPhotoId}/original`);
		expect(embeddedOff.status()).toBe(200);
		const rest = await embeddedOff.body();
		expect(rest.includes(Buffer.from(MOTION_PHOTO_NAME, 'latin1'))).toBe(true);
		expect(rest.includes(Buffer.from(`${MOTION_PHOTO_NAME.replace('.jpg', '')}.mp4`, 'latin1'))).toBe(false);

		// The preference flips it on: the extracted video joins the archive.
		const patch = await page.request.patch('/api/v1/users/me', {
			data: { downloadIncludeEmbeddedVideos: true }
		});
		expect(patch.status()).toBe(200);
		const embeddedOn = await (await page.request.get(`/api/v1/assets/${motionPhotoId}/original`)).body();
		expect(embeddedOn.includes(Buffer.from(`MOTION_0001.mp4`, 'latin1'))).toBe(true);
		await page.request.patch('/api/v1/users/me', { data: { downloadIncludeEmbeddedVideos: false } });

		// A plain photo keeps its plain original.
		const plain = await page.request.get(`/api/v1/assets/${plainId}/original`);
		expect(plain.status()).toBe(200);
		expect(plain.headers()['content-type']).toBe('image/jpeg');
	});

	test('scene 5: deleting the motion degrades the tile without an error', async ({ page }) => {
		await page.goto('/timeline');
		await expect(tile(page, MOTION_PHOTO_NAME)).toBeVisible({ timeout: 60_000 });
		await expect(liveMark(page, MOTION_PHOTO_NAME)).toBeVisible();

		// The motion's row is deleted for good: the still's link nulls out
		// through the schema, and the type mark goes with it.
		const deleted = await page.request.delete(`/api/v1/assets/${motionPhotoVideoId}`);
		expect(deleted.status()).toBe(204);

		await expect(liveMark(page, MOTION_PHOTO_NAME)).toHaveCount(0, { timeout: 60_000 });
		// The still is still one tile, still openable, and nothing shouts.
		await expect(tile(page, MOTION_PHOTO_NAME)).toBeVisible();
		expect(await tileCount(page)).toBe(3);
		await expect(page.locator('[role="alert"]')).toHaveCount(0);
	});
});
