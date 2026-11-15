<script lang="ts">
	// The full-screen asset viewer: the media stage (an image's
	// blurhash placeholder with the view-size and full-resolution
	// serving copies crossfading over it, a video's transcoded
	// stream behind its poster, a Live Photo's action-row toggle
	// between the still and the component video's play stream), the gradual
	// zoom, the arrow-key walk over the timeline wall's current page
	// sequence, the floating chrome, and the layered Escape rule
	// (the page's assign dialog first, then the drawn-face mode,
	// then the edit-people panel, then the info panel, then the
	// route).
	//
	// THE CONTAIN RULE (the stage's law): the media box fits its own
	// aspect -- width-bound fills the width, height-bound fills the
	// height, only the short dimension leaves margin. The box's size
	// derives from the stage's measured box (a ResizeObserver rides
	// it: the panel toggle, the window resize, every relayout) and
	// the DECODED ratio (the loaded image's own natural sizes, the
	// video's own dimensions -- the orientation the derivatives bake
	// in comes free). No percentage inset: a portrait photo is as
	// large as the stage's height allows.
	//
	// EVERYTHING LIVE LIVES INSIDE THE BOX: the on-photo face-box
	// layer, the drawn-face drag layer, and the Live Photo's motion
	// are children of the media box, so their percentage geometry
	// follows the box through every refit -- the block's biggest
	// regression risk, pinned here. The zoom scales the BOX (not the
	// image alone), so the boxes stay glued to their faces while the
	// photo enlarges.
	//
	// THE FLOATING CHROME: the action bar is absolute over the
	// stage's top edge, no surface of its own -- only the
	// night/70 -> transparent fade keeps its buttons readable over
	// any photo, and the photo draws from y=0. The bar carries NO
	// filename (the panel's header owns the name; a second copy on
	// the bar would only cover the photo): back on the left, the
	// actions on the right, the middle open. The arrows rest at 45% opacity, hover
	// and keyboard focus return 100%, and the touch form (no hover)
	// rests at 90%. The filmstrip is the stage's bottom-left corner,
	// no card around it -- the THUMBS are the translucent part
	// (65%, hover and current 100%), the ends fade through a mask.
	//
	// THE INFO PANEL: the fixed 360px sidebar at every width from
	// 640 up (no breakpoints above it), default
	// OFF, the preference remembered in localStorage across
	// reloads, and every toggle path (the bar's info button with its
	// aria-pressed/aria-controls, the panel's close mark, the `i`
	// key) driving the same state. The ONE deviation: below 640 the
	// panel becomes a bottom drawer over a scrim (360 viewport
	// against 360 sidebar would zero the stage). Closing the panel
	// re-fits the stage through the observer.
	//
	// THE ACTION BAR'S ARMS: the close mark, the favorite toggle
	// (per-user state, everyone holds it), the info-panel toggle,
	// and the trash/restore pair (owner-only, the identity seam's
	// write arm). The permanent delete LEFT the viewer for the trash
	// page alone. The bar surfaces the last action's failure inline
	// (the mapped error line under the buttons).
	//
	// THE FACE LINKAGE: the on-photo boxes and the panel's chips are
	// two ends of one signal (./face-link.svelte.ts) -- a hover or a
	// focus on either end highlights both (the linked box carries
	// the huge-spread shadow that dims everything OUTSIDE it, the
	// box's own interior un-dimmed), with the ~220ms hide grace that
	// lets the pointer travel from chip to box. A box click mirrors
	// its chip: a named face opens the person, an unnamed one the
	// assign flow.
	//
	// THE DRAWN FACE: the panel's add-person icon arms the drag
	// layer; one drag across the photo draws a SQUARE box (the long
	// edge sets the side, anchored at the start), a release under
	// 24px is a miss (no dialog), and a real box hands its per-mille
	// geometry to the page's POST and assign flow. Esc disarms.
	//
	// OWNERSHIP SPLIT: this component owns the viewer's interaction
	// state (zoom, live playback, panel open, draw mode) and every
	// presentation; the route page owns the data (the detail, the
	// sequence, the network). The callbacks carry the page's
	// outcomes back so the surfaces here can render failures.
	//
	// THE PINNED RULES:
	//
	// - Navigation CLAMPS at the sequence's ends (ArrowLeft walks
	//   toward the newer end, ArrowRight toward the older); an asset
	//   outside the sequence (a deep link) has no neighbors, so both
	//   arrows are inert for it. Every navigation replaces the route
	//   (no history spam), which keeps Escape's history-back the
	//   one-step return to the timeline.
	// - A Live Photo's action row leads with the play/stop toggle: the
	//   first press probes the component's play stream through the
	//   page and plays the motion IN PLACE, the second press (or the
	//   stream's own end) returns the still. ASSET_NOT_READY (or a
	//   stream that refuses) keeps the still photo with no error
	//   surfaced; a later press re-probes, because the pair's own
	//   ready event says nothing about the component's transcode.
	// - Escape layers: the page's assign dialog closes first (its
	//   own Escape), then the drawn-face mode, then the edit-people
	//   panel, then the panel, then the route exits. Keys typed into
	//   an editable field (the description editor) never reach the
	//   viewer.
	// - No cover crop, no auto-hiding bar, no filmstrip auto-scroll:
	//   the stage's three not-doings (the crop cuts faces, the photo
	//   is not a video, and the reader owns the strip's scroll).
	import { decode } from 'blurhash';
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import InfoDrawer from './InfoDrawer.svelte';
	import EditFacesPanel from './EditFacesPanel.svelte';
	import { createZoomEngine, VIEWER_ZOOM_BASE, VIEWER_ZOOM_MAX, type ZoomState } from './zoom';
	import { linkFace, unlinkFace, isLinked } from './face-link.svelte';
	import {
		assetThumbnailUrl,
		assetPlayUrl,
		assetOriginalUrl,
		type AssetDetail,
		type AssetFaceBbox,
		type AssetFaceSummary
	} from '$lib/api/assets';
	import type { BatchAction, TimelineItem } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Play from '@lucide/svelte/icons/play';
	import Pause from '@lucide/svelte/icons/pause';
	import Star from '@lucide/svelte/icons/star';
	import Download from '@lucide/svelte/icons/download';
	import Info from '@lucide/svelte/icons/info';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import ZoomIn from '@lucide/svelte/icons/zoom-in';
	import ZoomOut from '@lucide/svelte/icons/zoom-out';
	import Maximize from '@lucide/svelte/icons/maximize';
	import X from '@lucide/svelte/icons/x';

	type Props = {
		/** The current asset's detail; null while loading. */
		detail: AssetDetail | null;
		/** The favorite flag (the detail carries none; the page holds it). */
		favorite: boolean;
		/**
		 * Whether the caller owns the open asset (the session's userId
		 * against the detail's ownerId): the bar's owner-only
		 * affordances (the trash/restore pair) and the drawer's
		 * owner-only edit arms render for the owner alone.
		 */
		isOwner: boolean;
		/** The wall's current page items: the navigation sequence, newest first. */
		sequence: readonly TimelineItem[];
		/** The detail lifecycle: loading, ready, or the failure's key. */
		status: 'idle' | 'loading' | 'ready' | 'error';
		/** The error-code i18n key of the failed detail load ('' when none). */
		errorKey: string;
		/** Videos: whether the play stream answered the probe. */
		videoPlayable: boolean;
		/** Any curation action or edit is in flight: controls disable. */
		busy: boolean;
		/** The error-code i18n key of the last failed action ('' when none). */
		actionErrorKey: string;
		/** A page-level modal (the face assign dialog) is open: the layered Escape defers to it. */
		modalOpen: boolean;
		/** Navigate to another asset of the sequence. */
		onnavigate: (id: string) => void;
		/** Leave the viewer (Escape and the back affordance). */
		onexit: () => void;
		/** Retry the failed detail load. */
		onretry: () => void;
		/** Probe one video-family id's play stream; true when playable. */
		onprobe: (id: string) => Promise<boolean>;
		/** Save the description through PATCH. */
		onedit: (text: string) => Promise<{ ok: boolean; errorKey: string }>;
		/** Run one curation action over the current asset. */
		onaction: (action: BatchAction) => Promise<boolean>;
		/** Open the page's assign dialog for one face (the drawer's People section). */
		onassignface: (face: AssetFaceSummary) => void;
		/**
		 * Open the page's add-tag dialog over the current asset (the
		 * drawer's tags row; per-user state, every viewer's own write).
		 */
		onassigntag: () => void;
		/** Remove one of the caller's tags (the chip's ×; the page owns the write). */
		onremovetag: (tagId: string) => void;
		/**
		 * The edit-people panel's state and flow. The PAGE owns the
		 * open flag, the staged edits and every write; THIS component
		 * owns the composition alone -- the panel stands in the
		 * information drawer's place (the same form, so the photo
		 * keeps its layout), the drawer's pencil opens it, and the
		 * layered Escape falls from the page's dialogs to the panel,
		 * then to the drawer, then to the route.
		 */
		/** Whether the edit-people panel stands in the drawer's place. */
		editFacesOpen: boolean;
		/** A staged edit's label for one face; undefined when none is staged. */
		stageLabelOf: (face: AssetFaceSummary) => string | undefined;
		/** How many faces carry a staged edit (the panel's apply gate). */
		editStagedCount: number;
		/** The edit flow's commit is in flight. */
		editBusy: boolean;
		/** The error-code i18n key of the last failed edit commit ('' when none). */
		editErrorKey: string;
		/** Open the edit-people panel (the drawer's People pencil). */
		oneditfaces: () => void;
		/** Return from the edit-people panel to the information drawer. */
		oneditfacesclose: () => void;
		/** Open the choose-person dialog for one face of the panel. */
		onreassignface: (face: AssetFaceSummary) => void;
		/** Open the detach confirm for one face of the panel. */
		ondetachface: (face: AssetFaceSummary) => void;
		/** Commit every staged edit of the panel. */
		onapplyfaces: () => void;
		/**
		 * The drawn face: the page POSTs the box and runs the assign
		 * flow (the assign picker for the new face). The outcome is
		 * the page's; the mode disarms itself on the release.
		 */
		oncreateface: (bbox: AssetFaceBbox) => void;
		/** Open the edit-date dialog (the page owns it and the write). */
		onedittime: () => void;
		/** Open the edit-location dialog (the page owns it and the write). */
		oneditlocation: () => void;
		/** Open a named face's person page (the box click's arm). */
		onopenperson: (clusterId: string) => void;
	};

	let {
		detail,
		favorite,
		isOwner,
		sequence,
		status,
		errorKey,
		videoPlayable,
		busy,
		actionErrorKey,
		modalOpen,
		onnavigate,
		onexit,
		onretry,
		onprobe,
		onedit,
		onaction,
		onassignface,
		onassigntag,
		onremovetag,
		editFacesOpen,
		stageLabelOf,
		editStagedCount,
		editBusy,
		editErrorKey,
		oneditfaces,
		oneditfacesclose,
		onreassignface,
		ondetachface,
		onapplyfaces,
		oncreateface,
		onedittime,
		oneditlocation,
		onopenperson
	}: Props = $props();

	// The panel's open-preference persistence: the pinned key, one
	// value, default closed, restored on mount. Storage can be
	// unavailable (private mode); the preference then lives for the
	// session only.
	const INFO_PANEL_STORAGE_KEY = 'lumemor:viewer:info';

	function readStoredPanelOpen(): boolean {
		try {
			return localStorage.getItem(INFO_PANEL_STORAGE_KEY) === 'true';
		} catch {
			return false;
		}
	}

	function writeStoredPanelOpen(open: boolean): void {
		try {
			localStorage.setItem(INFO_PANEL_STORAGE_KEY, open ? 'true' : 'false');
		} catch {
			// The panel still opens and closes for this session.
		}
	}

	let drawerOpen = $state(readStoredPanelOpen());
	// The value the effect last persisted: the restore itself never
	// writes, so a fresh session without a stored key stays unwritten
	// until the user actually chooses. (Read from the same source the
	// state was seeded from, not from the state itself -- the effect
	// must not observe the initializer.)
	let persistedOpen = readStoredPanelOpen();
	$effect(() => {
		if (drawerOpen !== persistedOpen) {
			persistedOpen = drawerOpen;
			writeStoredPanelOpen(drawerOpen);
		}
	});

	// The gradual zoom: the engine owns the scale and its
	// target; this component mirrors them into reactive state the
	// template rides (the affordances read the target, the media
	// transform reads the per-frame scale).
	let zoomScale = $state(VIEWER_ZOOM_BASE);
	let zoomTarget = $state(VIEWER_ZOOM_BASE);
	const zoomEngine = createZoomEngine((state: ZoomState) => {
		zoomScale = state.scale;
		zoomTarget = state.target;
	});

	// The engine's rAF loop dies with the component (a pending frame
	// after unmount would write state into a dead tree).
	$effect(() => {
		return () => {
			zoomEngine.destroy();
		};
	});

	let livePlaying = $state(false);
	/** Whether the Live Photo component's stream answered the probe. */
	let livePlayable = $state(false);
	let viewLoaded = $state(false);
	let fullLoaded = $state(false);
	/** The poster frame dropped after its request failed (the placeholder carries the stage). */
	let posterFailed = $state(false);
	/**
	 * The media itself could not be loaded (the record answered, the
	 * file did not): the stage turns into the "will not open" card
	 * while the panel keeps the record's facts -- the file is gone,
	 * the metadata is not.
	 */
	let mediaFailed = $state(false);
	/** The stage retry's own counter: a bumped value re-requests the bytes. */
	let mediaRetryTick = $state(0);
	/** The decoded ratio of the current media (null until the media answers). */
	let ratio = $state<number | null>(null);
	let canvas: HTMLCanvasElement | undefined = $state();
	let stage: HTMLDivElement | undefined = $state();

	// The stage's measured box: the refit's input. The observer rides
	// the stage element itself, so a panel toggle, a window resize or
	// any other relayout re-fits the media box without a hook into
	// the cause.
	let stageW = $state(0);
	let stageH = $state(0);
	$effect(() => {
		const element = stage;
		if (!element || typeof ResizeObserver !== 'function') {
			return;
		}
		const observer = new ResizeObserver((entries) => {
			for (const entry of entries) {
				stageW = entry.contentRect.width;
				stageH = entry.contentRect.height;
			}
		});
		observer.observe(element);
		return () => {
			observer.disconnect();
		};
	});

	/** The media box's exact size once the decoded ratio is known. */
	const mediaBox = $derived.by(() => {
		if (ratio === null || stageW === 0 || stageH === 0) {
			return null;
		}
		// The contain rule: the short dimension fits, the long one
		// fills -- the largest box of the media's own ratio inside the
		// stage.
		const width = Math.min(stageW, stageH * ratio);
		return { width, height: width / ratio };
	});

	// The drawn-face mode: the drag layer inside the media box, the
	// square in flight, and the release's verdict.
	let drawOpen = $state(false);
	let drawing = $state(false);
	let drawLayer: HTMLDivElement | undefined = $state();
	let drawStart = $state({ x: 0, y: 0 });
	let drawSide = $state(0);
	/** The square's anchor side: the start point, extended toward the pointer. */
	let drawPositiveX = $state(true);
	let drawPositiveY = $state(true);

	const isImage = $derived(detail?.kind === 'image');
	const isLivePhoto = $derived(detail !== null && detail.kind === 'image' && detail.livePhotoVideoId !== null);
	const position = $derived(detail === null ? -1 : sequence.findIndex((entry) => entry.id === detail.id));

	/**
	 * Whether the on-photo face-box layer renders at all: a loaded,
	 * visible image with faces (a trashed or hidden asset's crop
	 * endpoints answer 404, and the boxes have no stage to sit on
	 * before the media decodes).
	 */
	const faceBoxesVisible = $derived(
		detail !== null &&
			detail.kind === 'image' &&
			viewLoaded &&
			!detail.isTrashed &&
			!detail.isHidden &&
			detail.faces.length > 0
	);

	/**
	 * The drawn-face mode's gate: the owner of a live image (the
	 * write refuses every other state server-side; the affordance
	 * never arms for one).
	 */
	const canDraw = $derived(
		isOwner && detail !== null && detail.kind === 'image' && !detail.isTrashed && !detail.isHidden
	);

	/**
	 * The trash arm's gate (the read-only source rule): the caller
	 * owns the photo AND the photo reaches them through some album
	 * that is not the sync stream. The sync album is the scan's
	 * verdict -- system-managed and read-only -- so its own wall
	 * offers no curation (the album detail's empty arm), and a photo
	 * whose only home is that stream must not offer the delete here
	 * either: one photo cannot be curatable in the album and
	 * un-curatable in the viewer. A photo another album holds
	 * (a directory source, a memory, a shared album the caller
	 * owns) keeps the arm, exactly like that album's wall does.
	 *
	 * The RESTORE arm never hides: a trashed photo's reachability is
	 * the trash itself (the record keeps its live album rows), and
	 * the restore is the way back out of the state the delete made.
	 * The favorite stays for everyone -- per-user state, not a write
	 * on the collection.
	 */
	const canTrash = $derived(
		isOwner && detail !== null && (detail.isTrashed || detail.albums.some((album) => album.type !== 'sync'))
	);

	// The per-asset interaction reset: navigating replaces the asset,
	// so the crossfades, zoom, ratio and live playback start over. The
	// gate compares the id VALUE (a plain marker, not reactive state):
	// the same-asset detail replacements -- PATCH adoption, the
	// curation flips, the SSE convergence refetch -- must NOT reset
	// the stage, or a loaded photo fades back to its placeholder (the
	// img element survives without a src re-set, so nothing re-fires
	// the load). The zoom snaps to the base (never animates: the new
	// asset must not inherit the old one's zoom). The panel persists
	// across neighbors regardless (comparing metadata across
	// neighbors is the panel-open navigation's point).
	//
	// THE FRAME SURVIVES AN IMAGE-TO-IMAGE SWITCH: the media box and
	// its <img> are the same elements across a neighbor of the same
	// kind, so dropping the paint here (the placeholder flashing back
	// in at the stage's full width, then the new ratio snapping the
	// box) is the second half of the switch's flicker. A ready image
	// replacing a ready image keeps the loaded frame and the ratio:
	// the browser holds the last decoded frame while the new bytes
	// decode, the new frame paints over it, and the ratio's own load
	// event re-fits the box once. Every other transition (a
	// video, a not-yet-ready record, the first load) resets as
	// before -- there is no painted frame to keep.
	let resetFor = '';
	let resetKind = '';
	$effect(() => {
		const id = detail?.id ?? '';
		const kind = detail?.kind ?? '';
		if (id === resetFor) {
			return;
		}
		const keepFrame = resetFor !== '' && resetKind === 'image' && kind === 'image' && detail?.isReady === true;
		resetFor = id;
		resetKind = kind;
		zoomEngine.snapToBase();
		livePlaying = false;
		livePlayable = false;
		posterFailed = false;
		mediaFailed = false;
		mediaRetryTick = 0;
		drawOpen = false;
		drawing = false;
		// The full-resolution layer never carries the previous photo
		// across (it sits ABOVE the view-size copy): it hides and
		// re-reveals on its own load, while the view-size frame does
		// the carrying.
		fullLoaded = false;
		if (!keepFrame) {
			viewLoaded = false;
			ratio = null;
		}
	});

	// Paint the blurhash placeholder (the tile's guarded pattern: a
	// context-less host such as jsdom keeps the CSS skeleton fill,
	// and an unparsable hash never crashes the viewer). The canvas
	// detaches when the stage turns into the failure card, so the
	// binding may read back null as well as undefined.
	$effect(() => {
		const hash = detail?.blurhash ?? null;
		const element = canvas;
		if (hash === null || !element) {
			return;
		}
		const context = element.getContext('2d');
		if (context === null) {
			return;
		}
		try {
			const placeholder = context.createImageData(32, 32);
			placeholder.data.set(decode(hash, 32, 32));
			context.putImageData(placeholder, 0, 0);
		} catch {
			// The skeleton fill carries the stage.
		}
	});

	/** Walk the sequence by delta, clamped at both ends. */
	function navigate(delta: number): void {
		if (detail === null || position === -1) {
			return;
		}
		const next = position + delta;
		if (next < 0 || next >= sequence.length) {
			return;
		}
		const target = sequence[next];
		if (target !== undefined) {
			onnavigate(target.id);
		}
	}

	/** Toggle the Live Photo's motion: press to play, press to stop. */
	async function pressLive(): Promise<void> {
		if (!isLivePhoto || detail === null || detail.livePhotoVideoId === null) {
			return;
		}
		if (livePlaying) {
			livePlaying = false;
			return;
		}
		if (!livePlayable) {
			// Re-probe until the stream answers: the component never
			// broadcasts its own readiness, and the pair's photo row
			// counts no transcode of its own.
			livePlayable = await onprobe(detail.livePhotoVideoId);
		}
		if (livePlayable) {
			livePlaying = true;
		}
	}

	/** The on-photo box's label: the caller's name, or the unnamed state. */
	function faceLabelOf(face: AssetFaceSummary): string {
		return face.label !== null ? face.label.name : $_('viewer_faces_unnamed');
	}

	/** A box click mirrors its chip: named opens the person, unnamed the assign flow. */
	function boxClick(face: AssetFaceSummary): void {
		if (face.clusterId !== null && face.label !== null) {
			onopenperson(face.clusterId);
			return;
		}
		onassignface(face);
	}

	function isEditableTarget(target: EventTarget | null): boolean {
		if (target === null || !(target instanceof HTMLElement)) {
			return false;
		}
		return target.closest('input, textarea, select, [contenteditable="true"], [contenteditable=""]') !== null;
	}

	function handleWindowKeydown(event: KeyboardEvent): void {
		if (isEditableTarget(event.target)) {
			// Typing belongs to the field (the description editor).
			return;
		}
		if (event.key === 'Escape') {
			if (modalOpen) {
				return;
			}
			if (drawOpen) {
				disarmDraw();
				return;
			}
			if (editFacesOpen) {
				oneditfacesclose();
				return;
			}
			if (drawerOpen) {
				drawerOpen = false;
				return;
			}
			onexit();
			return;
		}
		if (modalOpen) {
			// The open dialog owns every other key.
			return;
		}
		if (drawOpen) {
			// The armed drag layer owns the stage's keys: nothing walks
			// or zooms while a box is being drawn.
			return;
		}
		if (event.key === 'ArrowLeft') {
			event.preventDefault();
			navigate(-1);
			return;
		}
		if (event.key === 'ArrowRight') {
			event.preventDefault();
			navigate(1);
			return;
		}
		if (event.key === '+' || event.key === '-') {
			// The zoom's keyboard arm: the same engine the
			// bar's buttons and the double-click ride. Videos keep
			// their native controls' keyboard handling.
			if (isImage) {
				if (event.key === '+') {
					zoomEngine.zoomIn();
				} else {
					zoomEngine.zoomOut();
				}
			}
			return;
		}
		if (event.key === 'i' || event.key === 'I') {
			drawerOpen = !drawerOpen;
		}
	}

	// The binary zoom toggle: a double-click on the media flips
	// 1x/2x through the same gradual animation (the engine's ceiling
	// stays at 3x -- the steps reach it; the double-click does not
	// jump past the readable middle). While a Live Photo's motion plays
	// it owns the stage's clicks: a double-click there is a viewer
	// double-taking at the movement, not a request to zoom.
	function handleStageDblclick(): void {
		if (!isImage || drawOpen || livePlaying) {
			return;
		}
		zoomEngine.toggleZoom();
	}

	// THE DRAWN-FACE DRAG: the layer's pointer stream, the square
	// anchored at the start and extended toward the pointer by its
	// long edge. The geometry converts to per-mille against the
	// layer's own rect (the media box's on-screen area), so the
	// drawn box addresses the same pixels the crop endpoint crops.
	function armDraw(): void {
		if (!canDraw || ratio === null) {
			return;
		}
		drawOpen = true;
		drawing = false;
		drawSide = 0;
	}

	function disarmDraw(): void {
		drawOpen = false;
		drawing = false;
		drawSide = 0;
	}

	function layerPoint(event: PointerEvent): { x: number; y: number } {
		const rect = drawLayer?.getBoundingClientRect();
		if (rect === undefined) {
			return { x: 0, y: 0 };
		}
		return { x: event.clientX - rect.left, y: event.clientY - rect.top };
	}

	function handleDrawDown(event: PointerEvent): void {
		if (event.button !== 0) {
			return;
		}
		const at = layerPoint(event);
		drawStart = at;
		drawPositiveX = true;
		drawPositiveY = true;
		drawSide = 0;
		drawing = true;
		drawLayer?.setPointerCapture(event.pointerId);
	}

	function handleDrawMove(event: PointerEvent): void {
		if (!drawing) {
			return;
		}
		const at = layerPoint(event);
		const dx = at.x - drawStart.x;
		const dy = at.y - drawStart.y;
		// THE SQUARE RULE: the long edge sets the side -- a hand drawn
		// crooked still comes out square.
		drawSide = Math.max(Math.abs(dx), Math.abs(dy));
		drawPositiveX = dx >= 0;
		drawPositiveY = dy >= 0;
	}

	function handleDrawUp(): void {
		if (!drawing) {
			return;
		}
		drawing = false;
		const rect = drawLayer?.getBoundingClientRect();
		if (rect === undefined || rect.width === 0 || rect.height === 0) {
			disarmDraw();
			return;
		}
		if (drawSide < 24) {
			// The miss: too small to be a face -- no dialog, the mode
			// just ends.
			disarmDraw();
			return;
		}
		const cornerX = drawPositiveX ? drawStart.x : drawStart.x - drawSide;
		const cornerY = drawPositiveY ? drawStart.y : drawStart.y - drawSide;
		const clamp = (value: number, low: number, high: number): number => Math.min(high, Math.max(low, value));
		const bbox: AssetFaceBbox = {
			x: clamp(Math.round((cornerX / rect.width) * 1000), 0, 1000),
			y: clamp(Math.round((cornerY / rect.height) * 1000), 0, 1000),
			w: clamp(Math.round((drawSide / rect.width) * 1000), 1, 1000),
			h: clamp(Math.round((drawSide / rect.height) * 1000), 1, 1000)
		};
		disarmDraw();
		oncreateface(bbox);
	}

	// The shared unknown-code fallback (see $lib/i18n/errors); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	/**
	 * The stage's retry: when the RECORD never arrived the page's
	 * detail refetch is the retry; when the record answered and the
	 * BYTES did not, the file is re-requested (a fresh URL, so no
	 * cached failure answers again).
	 */
	function retryStage(): void {
		if (status === 'error') {
			onretry();
			return;
		}
		mediaFailed = false;
		viewLoaded = false;
		fullLoaded = false;
		mediaRetryTick += 1;
	}

	/** The card's body: the missing-file sentence, or the record failure's own text. */
	function stageFailureBody(): string {
		return mediaFailed || errorKey === 'asset_not_found' ? $_('viewer_failure_body') : errorText(errorKey);
	}

	/** One media URL, with the retry's counter riding along when one ran. */
	function mediaUrl(base: string): string {
		if (mediaRetryTick === 0) {
			return base;
		}
		return `${base}${base.includes('?') ? '&' : '?'}retry=${mediaRetryTick}`;
	}
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<!-- The viewer's backdrop: the brand's deep night ink, opaque -- the
     photo surface needs no translucency and no blur behind it (the
     restrained surface strategy); only the controls that truly float
     over the media keep their glass. The root is the SPLIT ROW: the
     media column flexes, the docked panel (640+) rides beside it. -->
<div class="viewer fixed inset-0 z-(--z-page) flex bg-night" aria-busy={status === 'loading'}>
	<div class="viewer__main relative flex flex-col flex-1 min-w-0">
		<!-- The floating bar: absolute over the stage's top edge, no
		     surface of its own -- the night/70 -> transparent fade keeps
		     the buttons readable over any photo, and the photo draws
		     from y=0. NO filename (the panel's header owns the name):
		     back on the left, the actions on the right, the middle
		     open. -->
		<header
			class="viewer__bar absolute inset-x-0 top-0 z-(--z-stage-arrow) flex flex-wrap items-center gap-3 px-4 py-2.5 text-cream-text bg-linear-to-b from-night/70 via-night/30 to-transparent"
		>
			<button
				class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
				type="button"
				aria-label={$_('viewer_close')}
				onclick={onexit}
			>
				<ChevronLeft size={22} />
			</button>
			{#if detail !== null}
				<div class="viewer__bar-actions ms-auto flex flex-nowrap gap-1">
					{#if isLivePhoto}
						<!-- The Live Photo's toggle leads the action row: one
						     press plays the pair's motion in place, the next
						     stops it, and the stream's own end returns the
						     still (the icon-only arm: lucide play/pause, the
						     pressed state and the label both follow). It
						     renders for the shared member too -- playing is a
						     read, not a write. -->
						<button
							class="viewer__icon viewer__live-toggle inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
							type="button"
							aria-label={livePlaying ? $_('viewer_live_stop') : $_('viewer_live_play')}
							title={livePlaying ? $_('viewer_live_stop') : $_('viewer_live_play')}
							aria-pressed={livePlaying}
							onclick={() => void pressLive()}
						>
							{#if livePlaying}
								<Pause size={20} />
							{:else}
								<Play size={20} />
							{/if}
						</button>
					{/if}
					<button
						class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px] disabled:cursor-not-allowed disabled:opacity-55"
						type="button"
						aria-label={favorite ? $_('action_unfavorite') : $_('action_favorite')}
						aria-pressed={favorite}
						disabled={busy}
						onclick={() => void onaction(favorite ? 'unfavorite' : 'favorite')}
					>
						<!-- The per-user favorite: the star fills and takes the
					     star accent color when the flag is on (the gold
					     the tile's favorite badge already carries). -->
						<Star size={20} fill={favorite ? 'currentColor' : 'none'} class={favorite ? 'text-star' : ''} />
					</button>
					{#if canTrash}
						<button
							class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px] disabled:cursor-not-allowed disabled:opacity-55"
							type="button"
							aria-label={detail.isTrashed ? $_('action_restore') : $_('action_trash')}
							disabled={busy}
							onclick={() => void onaction(detail.isTrashed ? 'restore' : 'trash')}
						>
							{#if detail.isTrashed}
								<RotateCcw size={20} />
							{:else}
								<Trash2 size={20} />
							{/if}
						</button>
					{/if}
					{#if isImage}
						<!-- The zoom trio: step in, step out, and the
					     fit reset -- one step each, every change riding
					     the gradual engine; the buttons disable at their
					     clamp ends (the engine's targets, not the
					     interpolated scale, drive them). -->
						<button
							class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px] disabled:cursor-not-allowed disabled:opacity-55"
							type="button"
							aria-label={$_('viewer_zoom_in')}
							disabled={zoomTarget >= VIEWER_ZOOM_MAX}
							onclick={() => zoomEngine.zoomIn()}
						>
							<ZoomIn size={20} />
						</button>
						<button
							class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px] disabled:cursor-not-allowed disabled:opacity-55"
							type="button"
							aria-label={$_('viewer_zoom_out')}
							disabled={zoomTarget <= VIEWER_ZOOM_BASE}
							onclick={() => zoomEngine.zoomOut()}
						>
							<ZoomOut size={20} />
						</button>
						<button
							class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px] disabled:cursor-not-allowed disabled:opacity-55"
							type="button"
							aria-label={$_('viewer_zoom_reset')}
							disabled={zoomTarget <= VIEWER_ZOOM_BASE}
							onclick={() => zoomEngine.reset()}
						>
							<Maximize size={20} />
						</button>
					{/if}
					<!-- The original download: an ACTION, not metadata -- the
					     bar owns it beside the zoom utilities (the anchor
					     form keeps the native download, no script). -->
					<a
						class="viewer__icon viewer__download inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
						href={assetOriginalUrl(detail.id)}
						download
						aria-label={$_('viewer_download')}
					>
						<Download size={20} />
					</a>
					<!-- The info toggle: one of the panel's three paths --
					     pressed state synced, the panel named. -->
					<button
						class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
						type="button"
						aria-label={$_('viewer_info_open')}
						aria-pressed={drawerOpen}
						aria-controls="viewer-info"
						onclick={() => (drawerOpen = !drawerOpen)}
					>
						<Info size={20} />
					</button>
				</div>
			{/if}
			{#if actionErrorKey !== ''}
				<p
					class="viewer__action-error basis-full my-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
					role="alert"
				>
					{errorText(actionErrorKey)}
				</p>
			{/if}
		</header>

		<div class="viewer__stage-wrap relative flex-1 min-h-0 flex flex-col">
			<!-- The gradual zoom: the media box scales around the
			     stage's center through --viewer-zoom (the engine drives
			     the var per frame); the whole box carries the transform,
			     so the face boxes and the drag layer stay glued to the
			     photo while it enlarges. The zoomed marker class and the
			     cursor key on the engine's TARGET -- synchronous with the
			     intent. The double-click is a pointer convenience with
			     full keyboard parity (the +/- keys and the bar's
			     buttons), so the stage carries no widget role. -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				class="viewer__stage relative flex-1 min-h-0 flex items-center justify-center overflow-hidden"
				class:viewer__stage--zoomed={zoomTarget > VIEWER_ZOOM_BASE}
				class:cursor-zoom-out={zoomTarget > VIEWER_ZOOM_BASE}
				style:--viewer-zoom={zoomScale}
				bind:this={stage}
				ondblclick={() => handleStageDblclick()}
			>
				<!-- The stage's failure: the record answered but the bytes did
				     not (a deleted file, a vanished disk), or the record
				     itself never arrived. The card is the "will not open"
				     face; the panel beside it keeps what the record knows. -->
				{#if status === 'error' || mediaFailed}
					<div
						class="viewer__stage-failure relative z-(--z-stage-chrome) flex items-center justify-center p-6"
						data-testid="viewer-stage-error"
					>
						<div
							class="max-w-[420px] box-content rounded-card border border-border bg-card p-6 text-center shadow-card"
							role="alert"
						>
							<p class="m-0 font-display font-bold text-[16px] leading-[1.3] text-text">
								{$_('viewer_failure_title')}
							</p>
							<p class="m-0 mt-2 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
								{stageFailureBody()}
							</p>
							<Button variant="ghost" class="mt-4" onclick={retryStage}>
								<RotateCcw size={18} aria-hidden="true" />{$_('action_retry')}
							</Button>
						</div>
					</div>
				{:else if detail === null}
					<p
						class="viewer__status relative z-(--z-stage-chrome) px-5 py-3 font-body font-medium text-[14px] leading-[1.4] text-cream-text text-center rounded-control bg-night/55 backdrop-blur-[2px] animate-loading-in"
						role="status"
					>
						{$_('viewer_loading')}
					</p>
				{:else if detail.kind === 'video'}
					<!-- The video's box: the poster behind the transcoded
					     stream, both inside the same contain-fitted box; the
					     ratio rides the video element's own dimensions. -->
					<div
						class="viewer__media relative z-(--z-stage-media) {mediaBox === null
							? 'w-full h-full'
							: ''} overflow-hidden"
						style:width={mediaBox?.width !== undefined ? `${mediaBox.width}px` : undefined}
						style:height={mediaBox?.height !== undefined ? `${mediaBox.height}px` : undefined}
					>
						{#if videoPlayable}
							<!-- svelte-ignore a11y_media_has_caption -->
							<video
								class="viewer__video absolute inset-0 w-full h-full object-contain"
								controls
								src={mediaUrl(assetPlayUrl(detail.id))}
								poster={mediaUrl(assetThumbnailUrl(detail.id, 'view'))}
								onloadedmetadata={(event) => {
									const element = event.currentTarget as HTMLVideoElement;
									if (element.videoWidth > 0 && element.videoHeight > 0) {
										ratio = element.videoWidth / element.videoHeight;
									}
								}}
								onerror={() => (mediaFailed = true)}
							></video>
						{:else}
							<!-- The transcoded stream is not there yet: the
							     placeholder layer carries the stage behind the
							     poster frame until the asset's own ready event
							     lands the refetch and the re-probe. Before the
							     thumbnail job runs the poster request answers
							     ASSET_NOT_READY; the error drops the poster so
							     the placeholder (never a broken image) holds the
							     stage (the image branch's pattern). -->
							{#if detail.blurhash !== null}
								<canvas
									class="viewer__placeholder absolute inset-0 w-full h-full object-contain"
									width="32"
									height="32"
									bind:this={canvas}
									aria-hidden="true"
								></canvas>
							{:else}
								<div
									class="viewer__placeholder viewer__placeholder--empty absolute inset-0 w-full h-full bg-[linear-gradient(135deg,color-mix(in_srgb,var(--color-night-surface)_70%,var(--color-night)),var(--color-night-surface))]"
									aria-hidden="true"
								></div>
							{/if}
							{#if !posterFailed}
								<img
									class="viewer__poster relative z-(--z-stage-media) absolute inset-0 m-auto max-w-full max-h-full object-contain"
									src={assetThumbnailUrl(detail.id, 'view')}
									alt={detail.originalFilename}
									onload={(event) => {
										const element = event.currentTarget as HTMLImageElement;
										// The DECODED ratio: the derivative's own pixel
										// sizes (the orientation the derivatives bake in
										// comes free), the contain rule's input.
										if (element.naturalWidth > 0 && element.naturalHeight > 0) {
											ratio = element.naturalWidth / element.naturalHeight;
										}
										viewLoaded = true;
									}}
									onerror={() => (posterFailed = true)}
								/>
							{/if}
							<p
								class="viewer__processing absolute bottom-6 z-(--z-stage-chrome) px-5 py-3 font-body font-medium text-[14px] leading-[1.4] text-cream-text text-center rounded-control bg-night/55 backdrop-blur-[2px] animate-loading-in"
								role="status"
							>
								{$_('viewer_processing')}
							</p>
						{/if}
					</div>
				{:else}
					<!-- The image's box: the placeholder, the view-size copy
					     and the full-resolution layer all live INSIDE (the
					     contain rule: the box is the photo's own aspect, the
					     layers fill it); the face-box layer and the drag
					     layer are its children too, so their percentage
					     geometry follows every refit. The placeholder
					     cross-fades out once the view-size copy's load event
					     lands. -->
					<div
						class="viewer__media relative z-(--z-stage-media) {mediaBox === null
							? 'w-full h-full'
							: ''} overflow-hidden"
						style:width={mediaBox?.width !== undefined ? `${mediaBox.width}px` : undefined}
						style:height={mediaBox?.height !== undefined ? `${mediaBox.height}px` : undefined}
					>
						{#if detail.blurhash !== null}
							<canvas
								class="viewer__placeholder absolute inset-0 w-full h-full object-contain transition-[opacity] duration-(--motion-placeholder-fade) ease-standard"
								class:viewer__placeholder--hidden={viewLoaded}
								class:opacity-0={viewLoaded}
								width="32"
								height="32"
								bind:this={canvas}
								aria-hidden="true"
							></canvas>
						{:else}
							<div
								class="viewer__placeholder viewer__placeholder--empty absolute inset-0 w-full h-full bg-[linear-gradient(135deg,color-mix(in_srgb,var(--color-night-surface)_70%,var(--color-night)),var(--color-night-surface))] transition-[opacity] duration-(--motion-placeholder-fade) ease-standard"
								class:viewer__placeholder--hidden={viewLoaded}
								class:opacity-0={viewLoaded}
								aria-hidden="true"
							></div>
						{/if}

						{#if detail.isReady}
							<img
								class="viewer__image absolute inset-0 m-auto max-w-full max-h-full object-contain transition-[opacity] duration-(--motion-base) ease-standard"
								class:opacity-0={!viewLoaded}
								class:viewer__image--loaded={viewLoaded}
								src={mediaUrl(assetThumbnailUrl(detail.id, 'view'))}
								alt={detail.originalFilename}
								onload={(event) => {
									const element = event.currentTarget as HTMLImageElement;
									// The DECODED ratio: the derivative's own pixel
									// sizes (the orientation the derivatives bake in
									// comes free), the contain rule's input.
									if (element.naturalWidth > 0 && element.naturalHeight > 0) {
										ratio = element.naturalWidth / element.naturalHeight;
									}
									viewLoaded = true;
								}}
								onerror={() => (mediaFailed = true)}
								draggable="false"
							/>
							<!-- The full-resolution layer rides above the view-size copy. -->
							<img
								class="viewer__image viewer__image--full absolute inset-0 m-auto z-(--z-stage-media) max-w-full max-h-full object-contain transition-[opacity] duration-(--motion-base) ease-standard"
								class:opacity-0={!fullLoaded}
								class:viewer__image--loaded={fullLoaded}
								src={mediaUrl(assetThumbnailUrl(detail.id, 'original'))}
								alt=""
								onload={() => (fullLoaded = true)}
								onerror={() => (fullLoaded = false)}
								draggable="false"
							/>
						{/if}

						{#if livePlaying && detail.livePhotoVideoId !== null}
							<!-- The Live Photo's motion: the component row's play
							     stream, muted, playing ONCE (a camera's clip, not a
							     loop). The stage keeps the still's box: the video
							     letterboxes inside it, so the picture never jumps
							     when its own aspect differs from the still's. The
							     end -- or a refused stream -- returns the still
							     without a word. -->
							<video
								class="viewer__live absolute inset-0 w-full h-full object-contain z-(--z-stage-live)"
								autoplay
								muted
								playsinline
								src={assetPlayUrl(detail.livePhotoVideoId)}
								aria-label={$_('asset_live_photo_label')}
								onended={() => (livePlaying = false)}
								onerror={() => (livePlaying = false)}
							></video>
						{/if}

						{#if faceBoxesVisible}
							<!-- The face-box layer: one square box per face, hidden
							     until its linkage lights it (the chip's hover, its
							     own hover, or a keyboard focus -- the ends are one
							     state). The linked box carries the huge-spread
							     shadow that dims everything OUTSIDE it while the
							     box's own interior stays untouched; the media box's
							     overflow-hidden clips the spread. The name plate is
							     the box's child, painted above the shadow, white
							     background, dark text -- and it FLIPS below the box
							     when the box sits too close to the photo's top edge
							     to hold it above (the flip rule: the plate never
							     clips).
							     THE REVEAL'S HIT AREA: the box keeps its pointer
							     events while hidden -- the region over the face IS
							     the arm that reveals it (the pointer finds the face,
							     the face's outline and plate answer). Without it the
							     photo side of the two-way linkage is unreachable:
							     the chip could light a box the pointer could never
							     touch. The pointer leaving the region unlinks through
							     the same ~220ms grace, so the plate does not blink
							     off on a one-pixel overshoot. -->
							<div class="absolute inset-0 z-(--z-stage-chrome)" data-face-layer>
								{#each detail.faces as face (face.id)}
									{@const linked = isLinked(face.id)}
									{@const plateFlips =
										mediaBox !== null && (face.bbox.y / 1000) * mediaBox.height < 30}
									<button
										class="face-box absolute aspect-square rounded-[4px] border-2 bg-transparent cursor-pointer transition-[opacity,border-color,box-shadow] duration-(--motion-fast) ease-standard [@media(hover:none)]:pointer-events-none {linked
											? 'opacity-100 border-accent [box-shadow:0_0_0_9999px_color-mix(in_srgb,var(--color-night)_52%,transparent)]'
											: 'opacity-0 border-transparent'} focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
										style:left={`${face.bbox.x / 10}%`}
										style:top={`${face.bbox.y / 10}%`}
										style:width={`${face.bbox.w / 10}%`}
										type="button"
										data-face-box={face.id}
										aria-label={faceLabelOf(face)}
										onmouseenter={() => linkFace(face.id)}
										onmouseleave={() => unlinkFace(face.id)}
										onfocus={() => linkFace(face.id)}
										onblur={() => unlinkFace(face.id)}
										onclick={() => boxClick(face)}
									>
										<span
											class="face-label absolute {linked ? '' : 'opacity-0'} left-0 {plateFlips
												? '-bottom-7'
												: '-top-7'} px-2 py-0.5 rounded-control bg-card/95 font-body font-semibold text-[12px] leading-[1.4] text-text shadow-card whitespace-nowrap"
											data-face-label
										>
											{faceLabelOf(face)}
										</span>
									</button>
								{/each}
							</div>
						{/if}

						{#if drawOpen}
							<!-- The drawn-face drag layer: armed by the panel's
							     add-person icon, square in flight, the release
							     handing its per-mille geometry to the page's POST.
							     Esc disarms (the layered Escape's innermost arm). -->
							<div
								class="absolute inset-0 z-[calc(var(--z-stage-arrow)+1)] cursor-crosshair touch-none select-none"
								data-face-draw
								bind:this={drawLayer}
								onpointerdown={handleDrawDown}
								onpointermove={handleDrawMove}
								onpointerup={handleDrawUp}
								onpointercancel={() => (drawing = false)}
							>
								<p
									class="pointer-events-none absolute left-3 top-3 m-0 px-2.5 py-1 rounded-control bg-night/75 font-body font-semibold text-[12px] leading-[1.4] text-cream-text"
								>
									{$_('viewer_draw_hint')}
								</p>
								{#if drawing && drawSide > 0}
									<div
										class="absolute rounded-[2px] border-2 border-accent bg-[color-mix(in_srgb,var(--color-accent)_18%,transparent)]"
										data-face-rect
										style:left={`${drawPositiveX ? drawStart.x : drawStart.x - drawSide}px`}
										style:top={`${drawPositiveY ? drawStart.y : drawStart.y - drawSide}px`}
										style:width={`${drawSide}px`}
										style:height={`${drawSide}px`}
									></div>
								{/if}
							</div>
						{/if}
					</div>
				{/if}
			</div>

			{#if sequence.length > 1}
				<!-- The filmstrip: the stage's bottom-left corner, no card
				     around it -- the THUMBS are the translucent part (65%,
				     hover and current 100%, so the photo shows through),
				     the ends fade through the mask, the current thumb
				     carries the accent edge. No auto-scroll: the reader
				     owns the strip's scroll. -->
				<nav
					class="viewer__filmstrip absolute bottom-3 left-3 z-(--z-stage-arrow) max-w-[min(56%,560px)] overflow-x-auto px-1 py-1 [mask-image:linear-gradient(to_right,transparent,black_14px,black_calc(100%-14px),transparent)]"
					aria-label={$_('viewer_filmstrip_label')}
				>
					<ul class="viewer__filmstrip-list flex gap-1.5 list-none m-0 p-0">
						{#each sequence as entry (entry.id)}
							<li class="shrink-0">
								<button
									class="viewer__thumb block w-[52px] h-[52px] p-0 overflow-hidden border-2 rounded-control bg-transparent cursor-pointer transition-[opacity,border-color] duration-(--motion-fast) ease-standard {entry.id ===
									detail?.id
										? 'opacity-100 border-accent'
										: 'opacity-65 border-transparent hover:opacity-100'}"
									data-filmstrip-id={entry.id}
									type="button"
									aria-label={entry.originalFilename}
									aria-current={entry.id === detail?.id ? 'true' : undefined}
									onclick={() => onnavigate(entry.id)}
								>
									<!-- The tile's pinned gate: a thumbnail
								     request only fires once the entry is ready
								     (the wall snapshot's flag, with the page's
								     session overlay of the SSE ready events). -->
									{#if entry.isReady}
										<img
											class="w-full h-full object-cover"
											src={assetThumbnailUrl(entry.id, 'tile')}
											alt=""
											loading="lazy"
											draggable="false"
										/>
									{/if}
								</button>
							</li>
						{/each}
					</ul>
				</nav>
			{/if}

			{#if detail !== null && position !== -1 && position > 0}
				<!-- The three-step arrows: 45% at rest, hover and keyboard
				     focus 100% (the keyboard user must see the focus), the
				     touch form 90% (no hover to lift them). -->
				<button
					class="viewer__arrow viewer__arrow--left absolute left-4 top-1/2 -translate-y-1/2 z-(--z-stage-arrow) inline-flex items-center justify-center w-11 h-11 rounded-full text-cream-text bg-night/55 backdrop-blur-[2px] cursor-pointer opacity-45 transition-[background-color,opacity] duration-(--motion-fast) ease-standard hover:bg-accent hover:text-accent-on hover:opacity-100 focus-visible:opacity-100 focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px] [@media(hover:none)]:opacity-90"
					type="button"
					aria-label={$_('viewer_previous')}
					onclick={() => navigate(-1)}
				>
					<ChevronLeft size={26} />
				</button>
			{/if}
			{#if detail !== null && position !== -1 && position < sequence.length - 1}
				<button
					class="viewer__arrow viewer__arrow--right absolute right-4 top-1/2 -translate-y-1/2 z-(--z-stage-arrow) inline-flex items-center justify-center w-11 h-11 rounded-full text-cream-text bg-night/55 backdrop-blur-[2px] cursor-pointer opacity-45 transition-[background-color,opacity] duration-(--motion-fast) ease-standard hover:bg-accent hover:text-accent-on hover:opacity-100 focus-visible:opacity-100 focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px] [@media(hover:none)]:opacity-90"
					type="button"
					aria-label={$_('viewer_next')}
					onclick={() => navigate(1)}
				>
					<ChevronRight size={26} />
				</button>
			{/if}
		</div>
	</div>

	{#if drawerOpen}
		<!-- The bottom drawer's scrim (<640): one click closes, the
		     same path the panel's close mark drives. -->
		<button
			class="info-scrim fixed inset-0 z-[calc(var(--z-modal)-1)] bg-night/55 cursor-default sm:hidden"
			type="button"
			tabindex="-1"
			aria-label={$_('common_close')}
			onclick={() => (drawerOpen = false)}
		></button>
		<!-- The panel place: the fixed 360px sidebar at every width
		     from 640 up (a flex sibling -- the stage re-fits through
		     the flow), the bottom drawer below. The edit-people panel
		     stands in this region while it is open (the same form, the
		     photo's layout untouched). Before the record lands -- and
		     when it never does -- the panel place carries the load's
		     own state, so the open panel is never a silent blank
		     column. -->
		{#if detail === null}
			<aside
				id="viewer-info"
				class="info fixed inset-x-0 bottom-0 top-[14vh] z-(--z-modal) flex w-full flex-col rounded-t-overlay border-t border-border bg-card shadow-overlay sm:static sm:top-auto sm:bottom-auto sm:z-(--z-aside) sm:w-[360px] sm:shrink-0 sm:rounded-none sm:border-t-0 sm:border-l sm:shadow-none"
				aria-label={$_('viewer_info_title')}
				data-testid="viewer-info-state"
			>
				<span
					class="mx-auto mt-2 mb-1 h-1 w-10 shrink-0 rounded-full bg-border-strong sm:hidden"
					aria-hidden="true"
				></span>
				<header class="info__header flex items-center justify-between gap-2 px-4 py-3.5 border-b border-border">
					<h2 class="info__filename font-body font-semibold text-[15px] leading-[1.4] text-text">
						{$_('viewer_info_title')}
					</h2>
					<button
						class="info__close inline-flex items-center justify-center shrink-0 w-8 h-8 rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
						type="button"
						aria-label={$_('common_close')}
						onclick={() => (drawerOpen = false)}
					>
						<X size={16} />
					</button>
				</header>
				{#if status === 'error'}
					<div class="flex flex-col items-center gap-3 px-4 py-8 text-center" role="alert">
						<p class="m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
							{$_('viewer_info_failed')}
						</p>
						<Button variant="ghost" onclick={onretry}
							><RotateCcw size={18} aria-hidden="true" />{$_('action_retry')}</Button
						>
					</div>
				{:else}
					<p
						class="px-4 py-8 text-center font-body font-normal text-[14px] leading-[1.5] text-text-soft"
						role="status"
					>
						{$_('viewer_info_loading')}
					</p>
				{/if}
			</aside>
		{:else if editFacesOpen}
			<EditFacesPanel
				{detail}
				{stageLabelOf}
				stagedCount={editStagedCount}
				busy={editBusy}
				errorKey={editErrorKey}
				onclose={oneditfacesclose}
				onreassign={onreassignface}
				ondetach={ondetachface}
				onapply={onapplyfaces}
			/>
		{:else}
			<InfoDrawer
				{detail}
				{isOwner}
				onclose={() => (drawerOpen = false)}
				{onedit}
				{onassignface}
				{onassigntag}
				{onremovetag}
				{oneditfaces}
				onstartdraw={armDraw}
				{onedittime}
				{oneditlocation}
			/>
		{/if}
	{/if}
</div>

<style>
	/* Residual styles: the entrance keyframes and the zoomed stage's
	 * transform on the media box (one var, one box -- the face boxes
	 * and the drag layer ride the same transform their photo does). */
	.viewer {
		animation: viewer-in var(--motion-fast) var(--ease-standard);
	}

	/* The gradual zoom's application: the engine writes
	 * --viewer-zoom on the stage per frame and this one rule scales
	 * the media box off it. The rule is unconditional (not gated on
	 * the zoomed marker class) so a zoom-OUT animates the same way a
	 * zoom-in does -- a class gate would drop the transform
	 * mid-animation and snap the exit. */
	.viewer__media {
		transform: scale(var(--viewer-zoom, 1));
	}

	@keyframes viewer-in {
		from {
			opacity: 0;
		}
	}
</style>
