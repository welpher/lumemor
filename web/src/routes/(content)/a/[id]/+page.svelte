<script module lang="ts">
	// The in-app history depth of this document: the Esc fallback's
	// signal. afterNavigate resets it on 'enter' (a fresh document
	// load -- a deep link has nothing of this app before it) and
	// increments it for every completed navigation that implies an
	// in-app entry sits behind the current one (a link push, a
	// popstate landing). The viewer's own goto walks are replaceState
	// and never count: they add no history entry, so a deep link that
	// walked its arrows still has no in-app history and Escape must
	// not exit the site through history.back.
	let inAppHistory = 0;
</script>

<script lang="ts">
	// The asset viewer route: the thin data half of /a/{id}. The
	// route owns the asset's lifecycle -- the detail fetch keyed on
	// the route id, the SSE subscription that converges the open
	// asset (a video's readiness in particular; the newest event's
	// refetch response wins), the description PATCH, the single-id
	// curation batches with the optimistic local detail the drawer
	// renders, and the permanent delete -- while Viewer.svelte owns
	// the presentation and interaction.
	//
	// THE PINNED CHOICES:
	//
	// - The navigation sequence is the wall's published snapshot
	//   ($lib/timeline/wall.svelte), never a timeline fetch of this
	//   route's own; a deep link outside it still loads the detail.
	//   The snapshot is frozen while the viewer is open, so the
	//   session's asset.ready events overlay it: a sequence entry
	//   that finishes processing mid-session recovers its filmstrip
	//   thumbnail (the tile's pinned gate: no request before ready).
	// - The description edit is the refetch branch of the pinned
	//   choice: no optimistic copy, no rollback -- the PATCH response
	//   IS the fresh detail and lands on success; a failure never
	//   touched the stored text (the drawer keeps the draft).
	// - The curation actions mirror the wall's batch runner over the
	//   local detail: the optimistic flip first (favorite flag,
	//   hidden / trashed switches), the batch POST, a rollback from
	//   the pre-action snapshot on failure, and the SSE convergence
	//   confirming the success (favorites publish no events -- the
	//   optimistic flag is the final state, the wall runner's
	//   reasoning).
	// - The video's playability is decided by the play-stream probe,
	//   not the detail's isReady (a stale detail or a missed event
	//   cannot wedge the poster); a not-ready answer keeps the
	//   poster and the current asset's SSE events re-probe.
	// - The permanent delete leaves for /timeline explicitly (a deep
	//   link may have no history to back through); its 404 counts
	//   as success -- the row is gone either way, the same
	//   already-cleaned reading the upload queue's save-as pins.
	//   Escape follows the same rule: history.back() only runs when
	//   the document actually has in-app history behind the route.
	import { page } from '$app/state';
	import { afterNavigate, goto } from '$app/navigation';
	import { SvelteSet } from 'svelte/reactivity';
	import Viewer from '$lib/viewer/Viewer.svelte';
	import { wallItems } from '$lib/timeline/wall.svelte';
	import type { BatchAction } from '$lib/api/timeline';
	import {
		deleteAsset,
		fetchAssetDetail,
		patchAsset,
		probeAssetPlay,
		runAssetBatch,
		type AssetDetail
	} from '$lib/api/assets';
	import { subscribeAssetEvents } from '$lib/events';

	const assetId = $derived(page.params.id ?? '');

	// The session's ready overlay of the frozen wall snapshot: the
	// wall is unmounted while the viewer is open, so a sequence entry
	// that finishes processing mid-session can only recover through
	// the asset.ready events this route already receives (the
	// filmstrip's readiness gate then lets the thumbnail fire).
	const sequenceReady = new SvelteSet<string>();

	// The wall's current page items with that overlay applied: the
	// navigation sequence, newest first.
	const sequence = $derived(
		wallItems().map((entry) =>
			sequenceReady.has(entry.id) && !entry.isReady ? { ...entry, isReady: true } : entry
		)
	);

	type DetailStatus = 'idle' | 'loading' | 'ready' | 'error';

	let detail = $state<AssetDetail | null>(null);
	let status = $state<DetailStatus>('idle');
	let errorKey = $state('');
	/** Videos: the play stream answered the probe. */
	let videoPlayable = $state(false);
	/** The favorite flag: the detail carries none, so the page holds it. */
	let favorite = $state(false);
	let busy = $state(false);
	let actionErrorKey = $state('');
	/** Bumping re-runs the detail fetch for the same id (the retry button). */
	let retryTick = $state(0);

	// The detail fetch, keyed on the route id (and the retry tick).
	// A superseded run -- the id changed, or the route left, while
	// the fetch was in flight -- never applies.
	$effect(() => {
		const id = assetId;
		void retryTick;
		if (id === '') {
			return;
		}
		status = 'loading';
		detail = null;
		errorKey = '';
		videoPlayable = false;
		let current = true;
		void (async () => {
			const outcome = await fetchAssetDetail(id);
			if (!current) {
				return;
			}
			if (!outcome.ok) {
				status = 'error';
				errorKey = outcome.errorKey;
				return;
			}
			detail = outcome.detail;
			status = 'ready';
			// Seed the favorite flag from the wall's item for this id:
			// the wall is unmounted while the viewer is open, so the
			// snapshot's flag is the last known truth.
			favorite =
				wallItems().find((entry) => entry.id === id)?.isFavorite ?? false;
			if (outcome.detail.kind === 'video') {
				const probe = await probeAssetPlay(id);
				if (current) {
					videoPlayable = probe.ok;
				}
			}
		})();
		return () => {
			current = false;
		};
	});

	// The Esc fallback's feed (see the module script's depth
	// counter): registered at init, delivered every completed
	// navigation while the route stays mounted.
	afterNavigate((navigation) => {
		if (navigation.type === 'enter') {
			// A fresh document load: no in-app history before it.
			inAppHistory = 0;
			return;
		}
		if (navigation.type === 'goto') {
			// This route's replaceState walks (arrows, filmstrip) and
			// the exit gotos: no history entry appears behind the
			// route, so the depth never grows.
			return;
		}
		inAppHistory += 1;
	});

	// The SSE subscription of the OPEN asset only: its events
	// converge the detail (ready flips the pipeline fields, updated /
	// trashed / restored flip the owner switches). Events for other
	// assets only feed the filmstrip's ready overlay -- the wall
	// converges itself when it remounts and refetches. A deletion of
	// the open asset leaves the viewer (the detail would 404 from
	// here on).
	$effect(() => {
		const id = assetId;
		if (id === '') {
			return;
		}
		const unsubscribe = subscribeAssetEvents((event) => {
			if (event.name === 'asset.ready') {
				// The frozen snapshot's not-ready entries recover
				// their filmstrip thumbnails within the session.
				sequenceReady.add(event.assetId);
			}
			if (event.assetId !== id) {
				return;
			}
			if (event.name === 'asset.deleted') {
				goto('/timeline');
				return;
			}
			void refetchDetail();
		});
		return unsubscribe;
	});

	/** The refetch run guard: a superseded run's response never applies. */
	let refetchRun = 0;

	/** Re-fetch the current asset's detail and re-probe a waiting video. */
	async function refetchDetail(): Promise<void> {
		const id = assetId;
		if (id === '') {
			return;
		}
		// Newest-wins (the timeline client's convergence discipline,
		// at page scale): an SSE burst fires concurrent refetches, and
		// a slower stale response must not land over a newer one.
		const run = ++refetchRun;
		const outcome = await fetchAssetDetail(id);
		if (run !== refetchRun || id !== assetId || !outcome.ok) {
			return;
		}
		detail = outcome.detail;
		// A successful convergence also recovers a failed initial
		// load (the error branch would keep rendering over the fresh
		// detail).
		status = 'ready';
		errorKey = '';
		if (outcome.detail.kind === 'video' && !videoPlayable) {
			const probe = await probeAssetPlay(id);
			if (run === refetchRun && id === assetId) {
				videoPlayable = probe.ok;
			}
		}
	}

	function navigateTo(id: string): void {
		// Replace, never push: the arrows must not stack history
		// entries, so Escape's history-back stays the one-step
		// return to the timeline.
		goto(`/a/${encodeURIComponent(id)}`, { replaceState: true });
	}

	function exitViewer(): void {
		if (inAppHistory > 0) {
			history.back();
			return;
		}
		// A deep link with no in-app history behind it: back would
		// exit the site, so land on the timeline explicitly (the
		// delete flow's rule).
		goto('/timeline');
	}

	function retryDetail(): void {
		retryTick += 1;
	}

	async function probePlayable(id: string): Promise<boolean> {
		const outcome = await probeAssetPlay(id);
		return outcome.ok;
	}

	async function editDescription(text: string): Promise<{ ok: boolean; errorKey: string }> {
		const current = detail;
		if (current === null || busy) {
			return { ok: false, errorKey: 'internal' };
		}
		busy = true;
		actionErrorKey = '';
		const outcome = await patchAsset(current.id, { description: text });
		busy = false;
		if (outcome.ok) {
			if (detail !== null && detail.id === current.id) {
				// The response is the fresh detail: adopt it (the
				// refetch branch -- no optimistic copy existed).
				detail = outcome.detail;
			}
			return { ok: true, errorKey: '' };
		}
		// The stored description never changed; the drawer keeps the
		// draft and renders the mapped failure.
		return { ok: false, errorKey: outcome.errorKey };
	}

	async function runAction(action: BatchAction): Promise<boolean> {
		const current = detail;
		if (current === null || busy) {
			return false;
		}
		busy = true;
		actionErrorKey = '';

		// The optimistic mirror of the wall's batch runner, over the
		// local detail copy.
		const favoriteBefore = favorite;
		const hiddenBefore = current.isHidden;
		const trashedBefore = current.isTrashed;
		switch (action) {
			case 'favorite':
				favorite = true;
				break;
			case 'unfavorite':
				favorite = false;
				break;
			case 'hide':
				detail = { ...current, isHidden: true };
				break;
			case 'unhide':
				detail = { ...current, isHidden: false };
				break;
			case 'trash':
				detail = { ...current, isTrashed: true };
				break;
			case 'restore':
				detail = { ...current, isTrashed: false };
				break;
		}

		const outcome = await runAssetBatch(action, [current.id]);
		busy = false;
		if (outcome.ok) {
			// The SSE events converge the refetch onto the same state
			// (favorites are silent; the optimistic flag is final).
			return true;
		}
		// Roll back from the pre-action snapshot.
		favorite = favoriteBefore;
		if (detail !== null && detail.id === current.id) {
			detail = { ...detail, isHidden: hiddenBefore, isTrashed: trashedBefore };
		}
		actionErrorKey = outcome.errorKey;
		return false;
	}

	async function confirmDelete(): Promise<{ ok: boolean; errorKey: string }> {
		const current = detail;
		if (current === null || busy) {
			return { ok: false, errorKey: 'internal' };
		}
		busy = true;
		actionErrorKey = '';
		const outcome = await deleteAsset(current.id);
		busy = false;
		if (outcome.ok) {
			goto('/timeline');
			return { ok: true, errorKey: '' };
		}
		return { ok: false, errorKey: outcome.errorKey };
	}
</script>

<svelte:head>
	<title>{detail ? `${detail.originalFilename} · lumemor` : 'Asset · lumemor'}</title>
</svelte:head>

<Viewer
	{detail}
	{favorite}
	sequence={sequence}
	{status}
	{errorKey}
	{videoPlayable}
	{busy}
	actionErrorKey={actionErrorKey}
	onnavigate={navigateTo}
	onexit={exitViewer}
	onretry={retryDetail}
	onprobe={probePlayable}
	onedit={editDescription}
	onaction={runAction}
	ondelete={confirmDelete}
/>
