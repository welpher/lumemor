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
	// renders, the permanent delete, and the face assign flow of the
	// drawer's People section -- while Viewer.svelte owns the
	// presentation and interaction.
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
	// - The face assign flow (M3b Task 5): a chip of the drawer's
	//   People section opens the assign dialog. Picking an existing
	//   cluster POSTs the face move; choosing "new person" (offered
	//   ONLY on unnamed-cluster chips) PUTs the caller's label on
	//   the face's CURRENT cluster -- naming the whole group, the
	//   family semantics the dialog copy explains -- never a move.
	//   A face the clustering has not grouped yet has no cluster to
	//   name and its move answers the people 404, which the assign
	//   context surfaces as the honest not-grouped message. Every
	//   write converges through the detail refetch: the faces array
	//   IS the chips' truth, and the person.* events of other
	//   sessions refetch it the same way (the server's answer
	//   decides, B4 -- the page never patches local state).
	// - The permanent delete leaves for /timeline explicitly (a deep
	//   link may have no history to back through); its 404 counts
	//   as success -- the row is gone either way, the same
	//   already-cleaned reading the upload queue's save-as pins.
	//   Escape follows the same rule: history.back() only runs when
	//   the document actually has in-app history behind the route.
	import { page } from '$app/state';
	import { afterNavigate, goto } from '$app/navigation';
	import { SvelteSet } from 'svelte/reactivity';
	import { _ } from 'svelte-i18n';
	import Viewer from '$lib/viewer/Viewer.svelte';
	import ClusterPickerDialog from '$lib/people/ClusterPickerDialog.svelte';
	import { wallItems } from '$lib/timeline/wall.svelte';
	import type { BatchAction } from '$lib/api/timeline';
	import {
		deleteAsset,
		fetchAssetDetail,
		patchAsset,
		probeAssetPlay,
		runAssetBatch,
		type AssetDetail,
		type AssetFaceSummary
	} from '$lib/api/assets';
	import {
		labelPerson,
		listPeople,
		listUnnamedPeople,
		moveFace,
		type PersonSummary,
		type UnnamedClusterSummary
	} from '$lib/api/people';
	import { subscribeAssetEvents, subscribePersonEvents, type PersonEvent } from '$lib/events';
	import { sessionUserId } from '$lib/auth/session.svelte';

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

	// The caller's ownership of the open asset (the M3c identity seam):
	// the session's userId against the asset's owner. A viewer -- the
	// owner's share grants them the read -- sees the drawer without its
	// owner-only write affordances (the description edit, the hide /
	// trash curation, the permanent delete); the favorite stays for
	// everyone (per-user state) and the face chips stay (their assign
	// dialog carries the per-user label write a sharee holds too).
	const isOwner = $derived.by(() => {
		const current = detail;
		return current !== null && current.ownerId === sessionUserId();
	});

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
		// The assign dialog belongs to a face of THIS asset: the
		// navigation to a neighbor (or the error retry) closes it with
		// the rest of the asset-scoped state.
		assignFaceId = null;
		assignErrorKey = '';
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

	// The person events converge the drawer's face chips: a label,
	// hidden, merge or face-move write that touches one of the open
	// asset's faces (their clusters or their ids) refetches the
	// detail, whose faces array IS the chips' truth. Foreign sessions'
	// events refetch the same answer and move nothing the server does
	// not move (B4); an event off every rendered face adds no fetch.
	// The subscription lives for the route's lifetime and unsubscribes
	// on destroy.
	$effect(() => {
		const unsubscribe = subscribePersonEvents((event) => {
			const current = detail;
			if (current === null) {
				return;
			}
			if (personEventTouchesFaces(event, current.faces)) {
				void refetchDetail();
			}
		});
		return unsubscribe;
	});

	/** Whether one person event concerns any of the open asset's faces. */
	function personEventTouchesFaces(event: PersonEvent, faces: AssetFaceSummary[]): boolean {
		switch (event.name) {
			case 'person.labeled':
			case 'person.unlabeled':
			case 'person.hidden':
				return faces.some((face) => face.clusterId === event.clusterId);
			case 'person.merged':
				return faces.some(
					(face) =>
						face.clusterId !== null &&
						(event.survivingClusterId === face.clusterId ||
							event.mergedClusterIds.includes(face.clusterId))
				);
			case 'person.face_moved':
				return faces.some(
					(face) => face.id === event.faceId || face.clusterId === event.clusterId
				);
		}
	}

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
		if (status === 'error') {
			// A successful convergence recovers a failed initial load:
			// the initial seeding (the favorite flag, which the detail
			// does not carry) runs here too, from the frozen snapshot's
			// last known truth. The flag is NOT re-seeded on the normal
			// path: the toggle's optimistic value must survive there
			// (favorites publish no SSE events, so a refetch arriving
			// between flip and settle must not clobber it).
			favorite = wallItems().find((entry) => entry.id === id)?.isFavorite ?? false;
		}
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

	// The face assign flow (the drawer's People section): the face
	// being assigned (null while the dialog is closed), the target
	// lists fetched on open, and the writes. The dialog component
	// renders the two paths; this page owns the network and the
	// convergence.
	/** The face the assign dialog is acting on; null while closed. */
	let assignFaceId = $state<string | null>(null);
	let assignBusy = $state(false);
	/** The error-code i18n key of the last failed assign write ('' when none). */
	let assignErrorKey = $state('');
	let assignPeople = $state<PersonSummary[]>([]);
	let assignUnnamed = $state<UnnamedClusterSummary[]>([]);
	let assignLoading = $state(false);

	/** The face behind the dialog, resolved against the current detail. */
	const assignFace = $derived(
		assignFaceId === null ? null : (detail?.faces.find((face) => face.id === assignFaceId) ?? null)
	);

	// Fetch the target lists on open: the caller's named people and
	// the unnamed region's first page (the people detail page's move
	// dialog discipline -- fresh lists every open, the fetched answer
	// owns the options).
	$effect(() => {
		if (assignFaceId === null) {
			return;
		}
		assignPeople = [];
		assignUnnamed = [];
		assignErrorKey = '';
		assignLoading = true;
		void (async () => {
			const [peopleOutcome, unnamedOutcome] = await Promise.all([
				listPeople(),
				listUnnamedPeople(null)
			]);
			if (assignFaceId === null) {
				// Closed mid-flight: the next open refetches.
				return;
			}
			assignLoading = false;
			if (peopleOutcome.ok) {
				assignPeople = peopleOutcome.people;
			}
			if (unnamedOutcome.ok) {
				assignUnnamed = unnamedOutcome.page.items;
			}
		})();
	});

	/**
	 * The new-person path, offered ONLY on unnamed-cluster chips: the
	 * naming labels the face's CURRENT cluster (the whole group
	 * shares the name -- the family semantics the dialog copy
	 * explains), so a labeled chip offers no rename here (that is the
	 * people page's surface) and a clusterless face has nothing to
	 * name.
	 */
	const assignNewPerson = $derived.by(() => {
		const face = assignFace;
		if (face === null) {
			return undefined;
		}
		const clusterId = face.clusterId;
		if (clusterId === null || face.label !== null) {
			return undefined;
		}
		return {
			labelKey: 'viewer_assign_mode_new',
			existingLabelKey: 'viewer_assign_mode_existing',
			noteKey: 'viewer_assign_note',
			submitKey: 'people_name_submit',
			onsubmit: (name: string) => void assignName(clusterId, name)
		};
	});

	// The picker's option groups: the named people and the unnamed
	// clusters, the face's own cluster never among the targets (a
	// move onto the current cluster is an accepted no-op -- never
	// offered).
	const assignGroups = $derived.by(() => {
		const exclude = assignFace?.clusterId ?? null;
		return [
			{
				label: $_('people_merge_group_people'),
				options: assignPeople
					.filter((entry) => entry.clusterId !== exclude)
					.map((entry) => ({ id: entry.clusterId, label: entry.name }))
			},
			{
				label: $_('people_move_group_unnamed'),
				options: assignUnnamed
					.filter((cluster) => cluster.clusterId !== exclude)
					.map((cluster) => ({
						id: cluster.clusterId,
						label: $_('people_merge_cluster_option', { values: { count: cluster.faceCount } })
					}))
			}
		];
	});

	/** Open the assign dialog for one face of the drawer's People section. */
	function startAssign(face: AssetFaceSummary): void {
		assignErrorKey = '';
		assignFaceId = face.id;
	}

	/** The move write: the face joins the picked cluster. */
	async function assignMove(targetId: string): Promise<void> {
		const faceId = assignFaceId;
		if (faceId === null || assignBusy) {
			return;
		}
		assignBusy = true;
		assignErrorKey = '';
		const outcome = await moveFace(faceId, targetId);
		assignBusy = false;
		if (!outcome.ok) {
			// The assign context gives the move's 404 its honest
			// meaning: the caller owns the asset the face sits on, so
			// a face the endpoint cannot resolve is one the clustering
			// has not grouped yet (the one message-key exception --
			// same discipline as the people client's requires-label).
			assignErrorKey =
				outcome.errorKey === 'person_not_found' ? 'viewer_face_not_grouped' : outcome.errorKey;
			return;
		}
		assignFaceId = null;
		void refetchDetail();
	}

	/** The naming write: the face's current cluster gains the caller's label. */
	async function assignName(clusterId: string, name: string): Promise<void> {
		if (assignFaceId === null || assignBusy) {
			return;
		}
		assignBusy = true;
		assignErrorKey = '';
		const outcome = await labelPerson(clusterId, name);
		assignBusy = false;
		if (!outcome.ok) {
			assignErrorKey = outcome.errorKey;
			return;
		}
		assignFaceId = null;
		void refetchDetail();
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
	{isOwner}
	sequence={sequence}
	{status}
	{errorKey}
	{videoPlayable}
	{busy}
	actionErrorKey={actionErrorKey}
	modalOpen={assignFaceId !== null}
	onnavigate={navigateTo}
	onexit={exitViewer}
	onretry={retryDetail}
	onprobe={probePlayable}
	onedit={editDescription}
	onaction={runAction}
	ondelete={confirmDelete}
	onassignface={startAssign}
/>

<!-- The assign dialog of the drawer's People section: pick an
     existing person (the move write) or, on an unnamed-cluster chip,
     name the face's current cluster as a new person (the label
     write). The page owns the writes and the convergence; the
     dialog owns the choice. -->
<ClusterPickerDialog
	open={assignFaceId !== null}
	titleKey="viewer_assign_title"
	bodyKey="viewer_assign_body"
	targetLabelKey="viewer_assign_target_label"
	submitKey="viewer_assign_submit"
	emptyKey="people_picker_empty"
	groups={assignGroups}
	newPerson={assignNewPerson}
	loading={assignLoading}
	busy={assignBusy}
	errorKey={assignErrorKey}
	onclose={() => (assignFaceId = null)}
	onsubmit={(targetId) => void assignMove(targetId)}
/>
