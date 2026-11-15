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
	// curation batches with the optimistic local detail the action
	// bar renders, and the face assign flow of the drawer's People
	// section -- while Viewer.svelte owns the presentation and
	// interaction. The permanent delete is NOT this route's business
	// since the viewer rework: it lives on the trash page alone.
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
	//   reasoning). The action bar surfaces the favorite and the
	//   trash/restore pair; the runner stays the total mirror of the
	//   batch API (hide / unhide included), ready for any surface
	//   that offers them.
	// - The video's playability is decided by the play-stream probe,
	//   not the detail's isReady (a stale detail or a missed event
	//   cannot wedge the poster); a not-ready answer keeps the
	//   poster and the current asset's SSE events re-probe.
	// - The face assign flow: a chip of the drawer's
	//   People section opens the assign dialog. Picking an existing
	//   cluster POSTs the face move; choosing "new person" (offered
	//   ONLY on unnamed-cluster chips) PUTs the caller's label on
	//   the face's CURRENT cluster -- naming the whole group, the
	//   family semantics the dialog copy explains -- never a move.
	//   A face the clustering has not grouped yet has no cluster to
	//   name, and its move simply joins the picked person (the
	//   endpoint accepts a clusterless source: there is no source
	//   cluster to maintain). Every
	//   write converges through the detail refetch: the faces array
	//   IS the chips' truth, and the person.* events of other
	//   sessions refetch it the same way (the server's answer
	//   decides -- the page never patches local state).
	// - The edit-people flow (the People section's pencil): the
	//   panel lists EVERY face of this photo; reassign picks and
	//   create picks STAGE locally (the panel's one-shot commit
	//   loops the moves and creates them once, the done echo
	//   counting them); the detach is an immediate confirm-gated
	//   write (the row re-renders among the not-yet-grouped -- the
	//   face row survives, its membership does not). All three
	//   writes are the owner's curation, so the pencil never
	//   renders for a viewer the owner shared to.
	// - A deletion of the open asset (another session, the expiry
	//   job) leaves the viewer through SSE (the detail would 404
	//   from here on). Escape follows the history-back rule:
	//   history.back() only runs when the document actually has
	//   in-app history behind the route.
	import { page } from '$app/state';
	import { afterNavigate, goto } from '$app/navigation';
	import { SvelteSet } from 'svelte/reactivity';
	import { _ } from 'svelte-i18n';
	import { currentLocale } from '$lib/i18n';
	import Viewer from '$lib/viewer/Viewer.svelte';
	import ClusterPickerDialog from '$lib/people/ClusterPickerDialog.svelte';
	import PersonPickerDialog from '$lib/people/PersonPickerDialog.svelte';
	import DetachFaceDialog from '$lib/viewer/DetachFaceDialog.svelte';
	import EditDateDialog from '$lib/viewer/EditDateDialog.svelte';
	import EditLocationDialog from '$lib/viewer/EditLocationDialog.svelte';
	import AddTagDialog from '$lib/tags/AddTagDialog.svelte';
	import { toast } from '$lib/components/toast.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { wallItems } from '$lib/timeline/wall-sequence.svelte';
	import type { BatchAction } from '$lib/api/timeline';
	import {
		fetchAssetDetail,
		patchAsset,
		probeAssetPlay,
		runAssetBatch,
		createFace,
		type AssetDetail,
		type AssetFaceBbox,
		type AssetFaceSummary
	} from '$lib/api/assets';
	import { addTagAssets, createTag, listTags, removeTagAssets, type TagSummary } from '$lib/api/tags';
	import {
		clusterFace,
		detachFace,
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

	// The caller's ownership of the open asset (the identity seam):
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
		// The tag dialog is asset-scoped the same way.
		tagOpen = false;
		tagErrorKey = '';
		tagReport = null;
		// The edit-people flow is asset-scoped the same way: the
		// panel, its staging record and its two dialogs close, and
		// the staged edits die with them (a staged edit belongs to a
		// face of THIS photo).
		editFacesOpen = false;
		pickFaceId = null;
		detachConfirmId = null;
		faceStages = {};
		editErrorKey = '';
		pickErrorKey = '';
		detachErrorKey = '';
		// The date and location dialogs are asset-scoped the same way.
		editTimeOpen = false;
		editTimeErrorKey = '';
		editTimeBusy = false;
		editLocOpen = false;
		editLocErrorKey = '';
		editLocBusy = false;
		let current = true;
		void (async () => {
			const outcome = await fetchAssetDetail(id, { locale: currentLocale() });
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
			favorite = wallItems().find((entry) => entry.id === id)?.isFavorite ?? false;
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
	// not move; an event off every rendered face adds no fetch.
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
				return faces.some((face) => face.clusterId === event.clusterId);
			// The entity's attributes (birth date, avatar) change no
			// chip on this page -- they are the person detail's
			// chrome -- so the asset detail does not converge on
			// them.
			case 'person.birthdate_updated':
			case 'person.avatar_updated':
				return false;
			case 'person.merged':
				return faces.some(
					(face) =>
						face.clusterId !== null &&
						(event.survivingClusterId === face.clusterId || event.mergedClusterIds.includes(face.clusterId))
				);
			// The moved, created and detached faces all name ONE face
			// and one cluster: the creation's cluster is the brand-new
			// one (the face itself is this asset's concern), the
			// detach's is the cluster the face left (a face of ours
			// still on it, or the detached one itself, converges).
			case 'person.face_moved':
			case 'person.face_clustered':
			case 'person.face_detached':
				return faces.some((face) => face.id === event.faceId || face.clusterId === event.clusterId);
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
		const outcome = await fetchAssetDetail(id, { locale: currentLocale() });
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
	/** The face id of the assign dialog opened straight from a drawn face (null otherwise). */
	let drawnFaceId = $state<string | null>(null);

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
			const [peopleOutcome, unnamedOutcome] = await Promise.all([listPeople(), listUnnamedPeople(null)]);
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
	 * The new-person path, offered on unnamed-cluster chips AND on
	 * the drawn face the add-person flow created (a clusterless face
	 * with no cluster to name yet). The chip's naming labels the
	 * face's CURRENT cluster (the whole group shares the name -- the
	 * family semantics the dialog copy explains); the drawn face's
	 * naming seeds the brand-new cluster FIRST (the from-face
	 * creation) and labels it -- the drawn face becomes its own
	 * person in one flow. A labeled chip offers no rename here
	 * (that is the people page's surface).
	 */
	const assignNewPerson = $derived.by(() => {
		const face = assignFace;
		if (face === null || face.label !== null) {
			return undefined;
		}
		const clusterId = face.clusterId;
		if (clusterId !== null) {
			return {
				labelKey: 'viewer_assign_mode_new',
				existingLabelKey: 'viewer_assign_mode_existing',
				noteKey: 'viewer_assign_note',
				submitKey: 'people_name_submit',
				onsubmit: (name: string) => void assignName(clusterId, name)
			};
		}
		if (assignFaceId !== drawnFaceId) {
			// A clusterless face the clustering has not grouped yet has
			// nothing to name -- only the drawn face's flow seeds one.
			return undefined;
		}
		return {
			labelKey: 'viewer_assign_mode_new',
			existingLabelKey: 'viewer_assign_mode_existing',
			noteKey: 'viewer_assign_note',
			submitKey: 'people_name_submit',
			onsubmit: (name: string) => void assignCreateAndName(face.id, name)
		};
	});

	/** The drawn face's naming: seed the new cluster from the face, then label it. */
	async function assignCreateAndName(faceId: string, name: string): Promise<void> {
		if (assignFaceId === null || assignBusy) {
			return;
		}
		assignBusy = true;
		assignErrorKey = '';
		const seeded = await clusterFace(faceId);
		if (!seeded.ok) {
			assignBusy = false;
			assignErrorKey = seeded.errorKey;
			return;
		}
		const labeled = await labelPerson(seeded.cluster.clusterId, name);
		assignBusy = false;
		if (!labeled.ok) {
			assignErrorKey = labeled.errorKey;
			return;
		}
		assignFaceId = null;
		void refetchDetail();
	}

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
		drawnFaceId = null;
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
			assignErrorKey = outcome.errorKey;
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

	// The edit-people flow (the People section's pencil): the staged
	// edits, the choose-person dialog, the detach confirm, and the
	// one-shot commit. The root asymmetry holds throughout: a stage
	// or a detach acts on ONE FACE (never the cluster's name -- the
	// chips' assign flow owns the naming).
	/**
	 * One staged edit of the panel: the reassign target (the move the
	 * commit will post) or the from-face creation (the new person the
	 * commit will seed). The label is what the panel's row shows
	 * while the edit is pending.
	 */
	type FaceStage = { kind: 'move'; targetClusterId: string; label: string } | { kind: 'create'; label: string };

	/** Whether the edit-people panel stands in the drawer's place. */
	let editFacesOpen = $state(false);
	/** The face whose choose-person dialog is open; null while closed. */
	let pickFaceId = $state<string | null>(null);
	/** The face whose detach confirm is open; null while closed. */
	let detachConfirmId = $state<string | null>(null);
	/** The staged edits, keyed by face id: the panel's check commits them once. */
	let faceStages = $state<Record<string, FaceStage>>({});
	/** The commit (or the detach write) is in flight. */
	let editBusy = $state(false);
	/** The error-code i18n key of the last failed commit ('' when none). */
	let editErrorKey = $state('');
	/** The choose-person list: the existing people, fetched on open. */
	let editPeople = $state<PersonSummary[]>([]);
	let editLoading = $state(false);
	/** The error-code i18n key of the failed list fetch ('' when none). */
	let pickErrorKey = $state('');
	/** The error-code i18n key of the failed detach write ('' when none). */
	let detachErrorKey = $state('');

	/** The face behind the choose-person dialog, resolved against the detail. */
	const pickFace = $derived(
		pickFaceId === null ? null : (detail?.faces.find((face) => face.id === pickFaceId) ?? null)
	);

	/** The face behind the detach confirm, resolved against the detail. */
	const detachingFace = $derived(
		detachConfirmId === null ? null : (detail?.faces.find((face) => face.id === detachConfirmId) ?? null)
	);

	/** The person the detach confirm names (the face's cluster label). */
	const detachingName = $derived(
		detachingFace === null
			? ''
			: detachingFace.label !== null
				? detachingFace.label.name
				: $_('viewer_faces_unnamed')
	);

	/**
	 * The choose-person list: the caller's named people minus the
	 * face's own cluster (a reassign onto the current cluster is an
	 * accepted no-op -- never offered, the move dialog's discipline).
	 */
	const pickPeople = $derived(editPeople.filter((entry) => entry.clusterId !== pickFace?.clusterId));

	/** The panel's staged label for one face; undefined when no edit is pending on it. */
	function stageLabelOf(face: AssetFaceSummary): string | undefined {
		return faceStages[face.id]?.label;
	}

	/** How many faces carry a staged edit (the panel's apply gate). */
	const editStagedCount = $derived(Object.keys(faceStages).length);

	// Fetch the choose-person list on open: the caller's named
	// people, fresh every open (the fetched answer owns the options,
	// the assign flow's discipline).
	$effect(() => {
		if (pickFaceId === null) {
			return;
		}
		pickErrorKey = '';
		editPeople = [];
		editLoading = true;
		void (async () => {
			const outcome = await listPeople();
			if (pickFaceId === null) {
				// Closed mid-flight: the next open refetches.
				return;
			}
			editLoading = false;
			if (outcome.ok) {
				editPeople = outcome.people;
			} else {
				pickErrorKey = outcome.errorKey;
			}
		})();
	});

	/** Open the edit-people panel (the drawer's pencil): a fresh staging session. */
	function startEditFaces(): void {
		editErrorKey = '';
		faceStages = {};
		editFacesOpen = true;
	}

	/** Return from the panel to the information drawer (the back affordance, the layered Escape). */
	function closeEditFaces(): void {
		editFacesOpen = false;
	}

	/** Open the choose-person dialog for one face (the row's reassign pencil). */
	function startPick(face: AssetFaceSummary): void {
		pickErrorKey = '';
		pickFaceId = face.id;
	}

	/**
	 * Stage the reassign: the picked person becomes the face's
	 * pending edit. The pick stages, never writes -- the panel's
	 * check is the one writer.
	 */
	function stageReassign(targetClusterId: string): void {
		const faceId = pickFaceId;
		if (faceId === null) {
			return;
		}
		const target = editPeople.find((entry) => entry.clusterId === targetClusterId);
		if (target === undefined) {
			return;
		}
		faceStages[faceId] = { kind: 'move', targetClusterId, label: target.name };
		pickFaceId = null;
	}

	/** Stage the from-face creation: the face seeds a brand-new person at commit. */
	function stageCreate(): void {
		const faceId = pickFaceId;
		if (faceId === null) {
			return;
		}
		faceStages[faceId] = { kind: 'create', label: $_('people_create_from_face') };
		pickFaceId = null;
	}

	/** Open the detach confirm for one face (the row's trash). */
	function startDetachConfirm(face: AssetFaceSummary): void {
		detachErrorKey = '';
		detachConfirmId = face.id;
	}

	/**
	 * The detach write: immediate (a confirm-gated write, never a
	 * stage -- the membership release is one honest transaction, not
	 * a batched intent). The row re-renders among the not-yet-grouped
	 * through the convergence refetch: the face row survives on its
	 * asset, its membership does not. A reassign staged on the row
	 * dies with the detach: the membership it was staged against is
	 * gone, so keeping it would post a surprise move on the commit.
	 */
	async function runDetach(): Promise<void> {
		const faceId = detachConfirmId;
		if (faceId === null || editBusy) {
			return;
		}
		editBusy = true;
		detachErrorKey = '';
		const outcome = await detachFace(faceId);
		editBusy = false;
		if (!outcome.ok) {
			detachErrorKey = outcome.errorKey;
			return;
		}
		detachConfirmId = null;
		delete faceStages[faceId];
		toast($_('people_detach_done'), 'success');
		void refetchDetail();
	}

	/**
	 * The one-shot commit: the staged reassigns (the move loop, in
	 * the faces' own order) then the staged creates (each seeding a
	 * person from its face). The loop stops at the FIRST refusal --
	 * the writes that landed stay landed (their stages dropped),
	 * the rest stay staged for a retry, and the failure renders
	 * beside the apply button. The full commit returns to the
	 * information drawer with the count echo; every create carries
	 * its own echo (the new person exists from that write).
	 */
	async function commitEdits(): Promise<void> {
		const current = detail;
		if (current === null || editBusy) {
			return;
		}
		const entries = current.faces.flatMap((face) => {
			const stage = faceStages[face.id];
			return stage === undefined ? [] : [{ face, stage }];
		});
		if (entries.length === 0) {
			return;
		}
		editBusy = true;
		editErrorKey = '';
		let committed = 0;
		for (const { face, stage } of entries) {
			const outcome =
				stage.kind === 'move' ? await moveFace(face.id, stage.targetClusterId) : await clusterFace(face.id);
			if (!outcome.ok) {
				editErrorKey = outcome.errorKey;
				break;
			}
			delete faceStages[face.id];
			committed += 1;
			if (stage.kind === 'create') {
				toast($_('people_create_from_face_done'), 'success');
			}
		}
		editBusy = false;
		if (committed === entries.length) {
			editFacesOpen = false;
			toast($_('people_edits_count', { values: { count: committed } }), 'success');
		}
		if (committed > 0) {
			void refetchDetail();
		}
	}

	// The tag assign flow (the drawer's tags row): the dialog
	// (the batch bar's own, reused) over THIS asset's single id. The
	// write is the caller's own per-user state -- every viewer
	// holds it, never the owner alone -- and the convergence is the
	// detail refetch: the detail's tags array IS the row's truth.
	/** The add-tag dialog's open state. */
	let tagOpen = $state(false);
	let tagBusy = $state(false);
	/** The error-code i18n key of the last failed tag write ('' when none). */
	let tagErrorKey = $state('');
	let tagList = $state<TagSummary[]>([]);
	let tagLoading = $state(false);
	let tagReport = $state<{ added: number; skipped: number } | null>(null);

	/** Open the add-tag dialog: fresh list, fresh report state. */
	function openTagDialog(): void {
		if (detail === null) {
			return;
		}
		tagErrorKey = '';
		tagReport = null;
		tagOpen = true;
		tagList = [];
		tagLoading = true;
		void (async () => {
			const outcome = await listTags();
			if (!tagOpen) {
				// Closed mid-flight: the next open refetches.
				return;
			}
			tagLoading = false;
			if (outcome.ok) {
				tagList = outcome.tags;
			} else {
				tagErrorKey = outcome.errorKey;
			}
		})();
	}

	/** The existing-tag path: the honest report of the single-id add. */
	async function addToExistingTag(tagId: string): Promise<void> {
		const current = detail;
		if (current === null || tagBusy) {
			return;
		}
		tagBusy = true;
		tagErrorKey = '';
		const outcome = await addTagAssets(tagId, [current.id]);
		tagBusy = false;
		if (!outcome.ok) {
			tagErrorKey = outcome.errorKey;
			return;
		}
		tagReport = outcome.report;
	}

	/**
	 * The new-tag path: create (idempotently -- a retry converges on
	 * the same row, no created-id bookkeeping) then add.
	 */
	async function addToNewTag(name: string): Promise<void> {
		const current = detail;
		if (current === null || tagBusy) {
			return;
		}
		tagBusy = true;
		tagErrorKey = '';
		const created = await createTag(name);
		if (!created.ok) {
			tagBusy = false;
			tagErrorKey = created.errorKey;
			return;
		}
		const outcome = await addTagAssets(created.tag.id, [current.id]);
		tagBusy = false;
		if (!outcome.ok) {
			tagErrorKey = outcome.errorKey;
			return;
		}
		tagReport = outcome.report;
	}

	/** The report's done: close and converge through the detail refetch. */
	function finishTag(): void {
		tagOpen = false;
		tagReport = null;
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
		const outcome = await patchAsset(current.id, { description: text }, { locale: currentLocale() });
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

	// The date dialog (the info panel's pencil): one PATCH carrying
	// the composed pair -- the datetime-local wall clock plus the
	// zone dropdown's offset, the offset IS the display zone. The
	// response is the fresh detail (adopted like the description's),
	// the failure keeps the dialog open beside its draft.
	let editTimeOpen = $state(false);
	let editTimeBusy = $state(false);
	let editTimeErrorKey = $state('');

	function openEditTime(): void {
		if (detail === null) {
			return;
		}
		editTimeErrorKey = '';
		editTimeOpen = true;
	}

	async function editTakenAt(datetimeLocal: string, zone: string): Promise<void> {
		const current = detail;
		if (current === null || editTimeBusy) {
			return;
		}
		// Compose the wire value: the field speaks minute precision,
		// the contract's RFC 3339 wants seconds -- the zone suffix
		// carries the capture zone the panel displays.
		const composed = /:\d{2}$/.test(datetimeLocal) ? `${datetimeLocal}:00${zone}` : `${datetimeLocal}${zone}`;
		editTimeBusy = true;
		editTimeErrorKey = '';
		const outcome = await patchAsset(current.id, { takenAt: composed }, { locale: currentLocale() });
		editTimeBusy = false;
		if (outcome.ok) {
			if (detail !== null && detail.id === current.id) {
				detail = outcome.detail;
			}
			editTimeOpen = false;
			toast($_('viewer_date_saved'), 'success');
			return;
		}
		editTimeErrorKey = outcome.errorKey;
	}

	// The location dialog (the location row's pencil): one PATCH
	// carrying the picked pair. The response's `location` face and
	// `place` resolution follow the pin (the resolved city follows
	// the coordinates server-side); the failure keeps the dialog
	// open beside its pin.
	let editLocOpen = $state(false);
	let editLocBusy = $state(false);
	let editLocErrorKey = $state('');

	function openEditLocation(): void {
		if (detail === null) {
			return;
		}
		editLocErrorKey = '';
		editLocOpen = true;
	}

	async function editLocation(lat: number, lon: number): Promise<void> {
		const current = detail;
		if (current === null || editLocBusy) {
			return;
		}
		editLocBusy = true;
		editLocErrorKey = '';
		const outcome = await patchAsset(current.id, { latitude: lat, longitude: lon }, { locale: currentLocale() });
		editLocBusy = false;
		if (outcome.ok) {
			if (detail !== null && detail.id === current.id) {
				detail = outcome.detail;
			}
			editLocOpen = false;
			toast($_('viewer_location_saved'), 'success');
			return;
		}
		editLocErrorKey = outcome.errorKey;
	}

	// The tag chip's × (the tags row): the caller's own per-user
	// write over the one open asset, the convergence the detail
	// refetch (the tags array IS the row's truth). A failure rides
	// the toast's danger tone -- the chip has no inline slot to
	// fail into.
	async function removeTag(tagId: string): Promise<void> {
		const current = detail;
		if (current === null) {
			return;
		}
		const outcome = await removeTagAssets(tagId, [current.id]);
		if (outcome.ok) {
			toast($_('viewer_tag_removed'), 'success');
			void refetchDetail();
			return;
		}
		toast(resolveErrorText($_, outcome.errorKey), 'danger');
	}

	// The drawn face (the panel's add-person icon): the POST carries
	// the square's per-mille geometry; on success the new face's
	// assign flow opens over the refetched detail (the faces array
	// IS the chips' truth, and the drawn face joins it as a
	// clusterless row the picker assigns). A failure lands on the
	// danger toast (the drag layer is gone by then -- the mode
	// disarmed itself on the release).
	async function drawFace(bbox: AssetFaceBbox): Promise<void> {
		const current = detail;
		if (current === null) {
			return;
		}
		const outcome = await createFace(current.id, bbox);
		if (!outcome.ok) {
			toast(resolveErrorText($_, outcome.errorKey), 'danger');
			return;
		}
		await refetchDetail();
		const face = detail?.faces.find((entry) => entry.id === outcome.faceId) ?? null;
		if (face !== null) {
			startAssign(face);
			// AFTER startAssign, which clears the marker for every
			// other chip's flow: this dialog was opened BY a drawn
			// face, so its new-person path seeds the cluster.
			drawnFaceId = face.id;
		}
	}

	/** Open a named face's person page (the on-photo box click's arm). */
	function openPerson(clusterId: string): void {
		void goto(`/people/${encodeURIComponent(clusterId)}`);
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
</script>

<svelte:head>
	<title>{detail ? `${detail.originalFilename} · lumemor` : 'Asset · lumemor'}</title>
</svelte:head>

<Viewer
	{detail}
	{favorite}
	{isOwner}
	{sequence}
	{status}
	{errorKey}
	{videoPlayable}
	{busy}
	{actionErrorKey}
	modalOpen={assignFaceId !== null ||
		tagOpen ||
		pickFaceId !== null ||
		detachConfirmId !== null ||
		editTimeOpen ||
		editLocOpen}
	onnavigate={navigateTo}
	onexit={exitViewer}
	onretry={retryDetail}
	onprobe={probePlayable}
	onedit={editDescription}
	onaction={runAction}
	onassignface={startAssign}
	onassigntag={openTagDialog}
	onremovetag={(tagId) => void removeTag(tagId)}
	{editFacesOpen}
	{stageLabelOf}
	{editStagedCount}
	{editBusy}
	{editErrorKey}
	oneditfaces={startEditFaces}
	oneditfacesclose={closeEditFaces}
	onreassignface={startPick}
	ondetachface={startDetachConfirm}
	onapplyfaces={() => void commitEdits()}
	oncreateface={(bbox) => void drawFace(bbox)}
	onedittime={openEditTime}
	oneditlocation={openEditLocation}
	onopenperson={openPerson}
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
	emptyKey="people_picker_no_people"
	groups={assignGroups}
	newPerson={assignNewPerson}
	loading={assignLoading}
	busy={assignBusy}
	errorKey={assignErrorKey}
	onclose={() => (assignFaceId = null)}
	onsubmit={(targetId) => void assignMove(targetId)}
/>

<!-- The add-tag dialog of the drawer's tags row: the batch bar's
     own dialog reused, over this asset's single id. The write is
     the caller's own per-user state; the convergence is the detail
     refetch (the tags array IS the row's truth). -->
<AddTagDialog
	bind:open={tagOpen}
	tags={tagList}
	loading={tagLoading}
	busy={tagBusy}
	errorKey={tagErrorKey}
	report={tagReport}
	onpick={(tagId) => void addToExistingTag(tagId)}
	oncreate={(name) => void addToNewTag(name)}
	ondone={finishTag}
/>

<!-- The edit-date dialog of the info panel's date row: the wall
     clock and the zone dropdown edited as one value; the write is
     the page's PATCH (the offset in the composed value IS the
     display zone). -->
<EditDateDialog
	bind:open={editTimeOpen}
	takenAt={detail?.takenAt ?? ''}
	takenAtZone={detail?.takenAtZone ?? null}
	busy={editTimeBusy}
	errorKey={editTimeErrorKey}
	onsubmit={(datetimeLocal, zone) => void editTakenAt(datetimeLocal, zone)}
/>

<!-- The edit-location dialog of the info panel's location row: the
     place-name search and the draggable, clickable map edited as
     one pick; the write is the page's PATCH (the resolved place
     follows the pin). -->
<EditLocationDialog
	bind:open={editLocOpen}
	placeName={detail?.place?.name ?? ''}
	initial={detail?.location ?? null}
	busy={editLocBusy}
	errorKey={editLocErrorKey}
	onsubmit={(lat, lon) => void editLocation(lat, lon)}
/>

<!-- The choose-person dialog of the edit-people panel: the
     filterable target list. Picking an existing person STAGES the
     reassign; the create row STAGES the from-face creation -- the
     dialog itself never writes (the panel's check is the one
     writer). -->
<PersonPickerDialog
	open={pickFaceId !== null}
	titleKey="people_select_person_title"
	bodyKey="people_select_person_body"
	submitKey="viewer_assign_submit"
	people={pickPeople}
	loading={editLoading}
	errorKey={pickErrorKey}
	oncreate={stageCreate}
	onclose={() => (pickFaceId = null)}
	onsubmit={(clusterId) => stageReassign(clusterId)}
/>

<!-- The detach confirm of the edit-people panel: the body names the
     person, the write is immediate, and the row converges through
     the detail refetch. -->
<DetachFaceDialog
	open={detachConfirmId !== null}
	name={detachingName}
	busy={editBusy}
	errorKey={detachErrorKey}
	onclose={() => (detachConfirmId = null)}
	onsubmit={() => void runDetach()}
/>
