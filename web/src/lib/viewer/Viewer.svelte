<script lang="ts">
	// The full-screen asset viewer: the media stage (an image's
	// blurhash placeholder with the view-size and full-resolution
	// serving copies crossfading over it, a video's transcoded
	// stream behind its poster, a Live Photo's press-toggle between
	// the still and the component video's play stream), the gradual
	// zoom, the arrow-key walk over the timeline wall's current page
	// sequence with the bottom filmstrip, and the layered Escape rule
	// (the page's assign dialog first, then the info panel, then the
	// route).
	//
	// THE ACTION BAR (R2-3): the top chrome carries the curation
	// affordances as icon buttons -- the close mark, the favorite
	// toggle (per-user state, everyone holds it), the info-panel
	// toggle, and the trash/restore pair (owner-only, the identity
	// seam's write arm). The permanent delete LEFT the viewer for
	// the trash page alone: the destructive irreversible action
	// stays behind the trash listing's confirm-first flow, and this
	// surface never carries it. The bar surfaces the last action's
	// failure inline (the mapped error line under the buttons).
	//
	// THE SPLIT INFO PANEL (R2-9/10): the panel is a right DOCKED
	// region at lg+ -- the viewer root is a flex row, the media
	// column flexes (flex-1 min-w-0) and the panel rides beside it
	// occupying layout space, so the media stage re-fits through the
	// flow (the img's max-w/max-h constraints re-resolve against the
	// shrunken stage -- the resize-aware fit, no observer needed
	// because nothing about the fit is JS-computed). Below lg the
	// same panel falls back to the overlay-drawer form (fixed,
	// full-height over the right edge) and the action cluster keeps
	// R2-3's left shift to clear it. The open preference persists in
	// localStorage (one key, default closed) and restores on mount.
	//
	// THE GRADUAL ZOOM (R2-7/8): the in/out/reset buttons, the +/- keys
	// and the stage's double-click all drive one engine (see ./zoom.ts)
	// -- every change animates through rAF interpolation (~300ms,
	// cubicOut, interruptible: an input mid-flight takes over from the
	// current interpolated scale) and the scale clamps to the same 2x
	// ceiling the old binary zoom had. The affordances key on the
	// engine's TARGET (synchronous with the intent) while the media
	// transform rides the per-frame --viewer-zoom var.
	//
	// OWNERSHIP SPLIT: this component owns the viewer's interaction
	// state (zoom, live playback, panel open) and every
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
	// - A Live Photo's press probes the component's play stream
	//   through the page; ASSET_NOT_READY keeps the still photo (a
	//   later press re-probes -- the pair's own ready event says
	//   nothing about the component's transcode).
	// - Escape layers: the page's assign dialog closes first (its
	//   own Escape), then the panel, then the route exits. Keys
	//   typed into an editable field (the description editor) never
	//   reach the viewer.
	import { decode } from 'blurhash';
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import InfoDrawer from './InfoDrawer.svelte';
	import {
		createZoomEngine,
		VIEWER_ZOOM_BASE,
		VIEWER_ZOOM_MAX,
		type ZoomState
	} from './zoom';
	import {
		assetThumbnailUrl,
		assetPlayUrl,
		assetOriginalUrl,
		type AssetDetail,
		type AssetFaceSummary
	} from '$lib/api/assets';
	import type { BatchAction, TimelineItem } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Star from '@lucide/svelte/icons/star';
	import Download from '@lucide/svelte/icons/download';
	import Info from '@lucide/svelte/icons/info';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import ZoomIn from '@lucide/svelte/icons/zoom-in';
	import ZoomOut from '@lucide/svelte/icons/zoom-out';
	import Maximize from '@lucide/svelte/icons/maximize';

	type Props = {
		/** The current asset's detail; null while loading. */
		detail: AssetDetail | null;
		/** The favorite flag (the detail carries none; the page holds it). */
		favorite: boolean;
		/**
		 * Whether the caller owns the open asset (the session's userId
		 * against the detail's ownerId): the bar's owner-only
		 * affordances (the trash/restore pair) and the drawer's
		 * description editor render for the owner alone.
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
		onassigntag
	}: Props = $props();

	// The panel's open-preference persistence (R2-9): one key, default
	// closed, restored on mount. Storage can be unavailable (private
	// mode); the preference then lives for the session only.
	const INFO_PANEL_STORAGE_KEY = 'lumemor:viewer:info-panel';

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

	// The gradual zoom (R2-7/8): the engine owns the scale and its
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
	let canvas: HTMLCanvasElement | undefined = $state();
	let filmstrip: HTMLElement | undefined = $state();

	const isImage = $derived(detail?.kind === 'image');
	const isLivePhoto = $derived(
		detail !== null && detail.kind === 'image' && detail.livePhotoVideoId !== null
	);
	const position = $derived(
		detail === null ? -1 : sequence.findIndex((entry) => entry.id === detail.id)
	);

	// The per-asset interaction reset: navigating replaces the asset,
	// so the crossfades, zoom and live playback start over. The gate
	// compares the id VALUE (a plain marker, not reactive state): the
	// same-asset detail replacements -- PATCH adoption, the curation
	// flips, the SSE convergence refetch -- must NOT reset the stage,
	// or a loaded photo fades back to its placeholder (the img element
	// survives without a src re-set, so nothing re-fires the load).
	// The zoom snaps to the base (never animates: the new asset must
	// not inherit the old one's zoom). The panel persists across
	// neighbors regardless (comparing metadata across neighbors is
	// the panel-open navigation's point).
	let resetFor = '';
	$effect(() => {
		const id = detail?.id ?? '';
		if (id === resetFor) {
			return;
		}
		resetFor = id;
		zoomEngine.snapToBase();
		livePlaying = false;
		livePlayable = false;
		viewLoaded = false;
		fullLoaded = false;
		posterFailed = false;
	});

	// Paint the blurhash placeholder (the tile's guarded pattern: a
	// context-less host such as jsdom keeps the CSS skeleton fill,
	// and an unparsable hash never crashes the viewer).
	$effect(() => {
		const hash = detail?.blurhash ?? null;
		const element = canvas;
		if (hash === null || element === undefined) {
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

	// Keep the filmstrip's current thumb in view; the id gate keeps
	// this off every unrelated state change.
	$effect(() => {
		const id = detail?.id;
		if (id === undefined) {
			return;
		}
		const button = filmstrip?.querySelector<HTMLButtonElement>(
			`[data-filmstrip-id="${CSS.escape(id)}"]`
		);
		if (button !== null && button !== undefined && typeof button.scrollIntoView === 'function') {
			button.scrollIntoView({ block: 'nearest', inline: 'nearest' });
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

	function isEditableTarget(target: EventTarget | null): boolean {
		if (target === null || !(target instanceof HTMLElement)) {
			return false;
		}
		return (
			target.closest('input, textarea, select, [contenteditable="true"], [contenteditable=""]') !==
			null
		);
	}

	function handleWindowKeydown(event: KeyboardEvent): void {
		if (isEditableTarget(event.target)) {
			// Typing belongs to the field (the description editor).
			return;
		}
		if (event.key === 'Escape') {
			// The layered dismissal: the page's dialog, then the
			// panel, then the route. The assign dialog (modalOpen)
			// owns its own Escape; this handler defers to it so the
			// panel underneath stays open.
			if (modalOpen) {
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
			// The zoom's keyboard arm (R2-7): the same engine the
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

	// The binary zoom toggle (R2-8): a double-click on the media flips
	// 1x/the zoomed ceiling through the same gradual animation. The
	// Live Photo's press affordance owns its own clicks -- a double
	// press is the press pair (play/stop), never a zoom toggle.
	function handleStageDblclick(event: MouseEvent): void {
		if (!isImage) {
			return;
		}
		const target = event.target;
		if (target instanceof HTMLElement && target.closest('.viewer__press') !== null) {
			return;
		}
		zoomEngine.toggleZoom();
	}

	// The shared unknown-code fallback (see $lib/i18n/errors); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<!-- The viewer's backdrop: the brand's deep night ink, opaque -- the
     photo surface needs no translucency and no blur behind it (the
     restrained surface strategy); only the controls that truly float
     over the media keep their glass. The root is the SPLIT ROW
     (R2-10): the media column flexes, the docked panel (lg+) rides
     beside it. -->
<div class="viewer fixed inset-0 z-(--z-page) flex bg-night" aria-busy={status === 'loading'}>
	<div class="viewer__main flex flex-col flex-1 min-w-0">
	<!-- The action bar (R2-3): the top chrome over the media, Night
	     surface, icon buttons named by their aria-labels -- the close
	     mark, the favorite toggle, the trash/restore pair (owner-only),
	     the zoom trio (R2-7, images) and the info-panel toggle; the
	     last failed action's mapped error lands on the bar's wrap row
	     (the batch bar's pattern). At lg+ the bar spans only the media
	     column, so the docked panel can never cover it (the split's
	     formalization of R2-3's left shift). -->
	<header class="viewer__bar flex flex-wrap items-center gap-3 px-4 py-2.5 text-cream-text">
		<button
			class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
			type="button"
			aria-label={$_('viewer_close')}
			onclick={onexit}
		>
			<ChevronLeft size={22} />
		</button>
		<p class="viewer__title flex-1 min-w-[120px] font-body font-semibold text-[15px] leading-[1.4] [overflow-wrap:anywhere]">{detail?.originalFilename ?? ''}</p>
		{#if detail !== null}
			<!-- The action cluster's left shift (R2-3), now the OVERLAY
			     fallback's guard alone: below lg the panel is a fixed
			     overlay over the right edge and would cover these
			     buttons without the shift; at lg+ the split layout owns
			     the clearance (lg:me-0) and the shift is inert. -->
			<div class="viewer__bar-actions flex flex-nowrap gap-1 {drawerOpen ? 'me-[calc(min(360px,92vw)+1rem)] lg:me-0' : ''}">
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
				{#if isOwner}
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
					<!-- The zoom trio (R2-7): step in, step out, and the
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
				<button
					class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
					type="button"
					aria-label={$_('viewer_info_open')}
					aria-pressed={drawerOpen}
					onclick={() => (drawerOpen = !drawerOpen)}
				>
					<Info size={20} />
				</button>
			</div>
		{/if}
		{#if actionErrorKey !== ''}
			<p class="viewer__action-error basis-full my-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{errorText(actionErrorKey)}</p>
		{/if}
	</header>

	<div class="viewer__stage-wrap relative flex-1 min-h-0 flex flex-col">
		<!-- The gradual zoom (R2-8): the stage's media children scale
		     around the stage's center through --viewer-zoom (the
		     engine drives the var per frame, the residual style rule
		     below applies it to the image, placeholder and live
		     layers); the double-click flips the 1x/zoomed binary. The
		     zoomed marker class and the cursor key on the engine's
		     TARGET -- synchronous with the intent -- while the var
		     carries the interpolation. The double-click is a pointer
		     convenience with full keyboard parity (the +/- keys and
		     the bar's buttons), so the stage carries no widget role. -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="viewer__stage relative flex-1 min-h-0 flex items-center justify-center overflow-hidden"
			class:viewer__stage--zoomed={zoomTarget > VIEWER_ZOOM_BASE}
			class:cursor-zoom-out={zoomTarget > VIEWER_ZOOM_BASE}
			style:--viewer-zoom={zoomScale}
			ondblclick={handleStageDblclick}
		>
			{#if status === 'error'}
				<div class="viewer__failure flex flex-col items-center gap-3 p-5 text-center" role="alert">
					<p class="font-body font-medium text-[15px] leading-[1.4] text-cream-text">{errorText(errorKey)}</p>
					<Button variant="primary" onclick={onretry}>{$_('action_retry')}</Button>
				</div>
			{:else if detail === null}
				<p class="viewer__status relative z-(--z-stage-chrome) px-5 py-3 font-body font-medium text-[14px] leading-[1.4] text-cream-text text-center rounded-control bg-night/55 backdrop-blur-[2px] animate-loading-in" role="status">{$_('viewer_loading')}</p>
			{:else if detail.kind === 'video'}
				<!-- svelte-ignore a11y_media_has_caption -->
				{#if videoPlayable}
					<video
						class="viewer__video relative z-(--z-stage-media) max-w-full max-h-full"
						controls
						src={assetPlayUrl(detail.id)}
						poster={assetThumbnailUrl(detail.id, 'view')}
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
						<canvas class="viewer__placeholder absolute w-full h-full object-contain" width="32" height="32" bind:this={canvas} aria-hidden="true"></canvas>
					{:else}
						<div
							class="viewer__placeholder viewer__placeholder--empty absolute w-full h-full bg-[linear-gradient(135deg,color-mix(in_srgb,var(--color-night-surface)_70%,var(--color-night)),var(--color-night-surface))]"
							aria-hidden="true"
						></div>
					{/if}
					{#if !posterFailed}
						<img
							class="viewer__poster relative z-(--z-stage-media) max-w-full max-h-full"
							src={assetThumbnailUrl(detail.id, 'view')}
							alt={detail.originalFilename}
							onerror={() => (posterFailed = true)}
						/>
					{/if}
					<p class="viewer__processing absolute bottom-6 z-(--z-stage-chrome) px-5 py-3 font-body font-medium text-[14px] leading-[1.4] text-cream-text text-center rounded-control bg-night/55 backdrop-blur-[2px] animate-loading-in" role="status">{$_('viewer_processing')}</p>
				{/if}
			{:else}
				<!-- The placeholder cross-fades out (G6's placeholder
				     fade) once the view-size copy's load event lands --
				     the blurhash approximates the image's own average
				     colors, so the exit rides under the image's fade-in
				     without a visible dip; a failed load gives the
				     placeholder back. -->
				{#if detail.blurhash !== null}
					<canvas
						class="viewer__placeholder absolute w-full h-full object-contain transition-[opacity] duration-(--motion-placeholder-fade) ease-standard"
						class:viewer__placeholder--hidden={viewLoaded}
						class:opacity-0={viewLoaded}
						width="32"
						height="32"
						bind:this={canvas}
						aria-hidden="true"
					></canvas>
				{:else}
					<div
						class="viewer__placeholder viewer__placeholder--empty absolute w-full h-full bg-[linear-gradient(135deg,color-mix(in_srgb,var(--color-night-surface)_70%,var(--color-night)),var(--color-night-surface))] transition-[opacity] duration-(--motion-placeholder-fade) ease-standard"
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
						src={assetThumbnailUrl(detail.id, 'view')}
						alt={detail.originalFilename}
						onload={() => (viewLoaded = true)}
						onerror={() => (viewLoaded = false)}
						draggable="false"
					/>
					<!-- The full-resolution layer rides above the view-size copy. -->
					<img
						class="viewer__image viewer__image--full absolute inset-0 m-auto z-(--z-stage-media) max-w-full max-h-full object-contain transition-[opacity] duration-(--motion-base) ease-standard"
						class:opacity-0={!fullLoaded}
						class:viewer__image--loaded={fullLoaded}
						src={assetThumbnailUrl(detail.id, 'original')}
						alt=""
						onload={() => (fullLoaded = true)}
						onerror={() => (fullLoaded = false)}
						draggable="false"
					/>
				{/if}

				{#if livePlaying && detail.livePhotoVideoId !== null}
					<!-- The Live Photo's motion: the component row's play
					     stream, muted and looping like a camera's live
					     capture; the end returns to the still. -->
					<!-- svelte-ignore a11y_media_has_caption -->
					<video
						class="viewer__live relative z-(--z-stage-live) max-w-full max-h-full object-contain"
						autoplay
						muted
						loop
						playsinline
						src={assetPlayUrl(detail.livePhotoVideoId)}
						aria-label={$_('asset_live_photo_label')}
						onended={() => (livePlaying = false)}
					></video>
				{/if}

				{#if isLivePhoto}
					<!-- The press affordance (the plan-pinned Live Photo
					     interaction): a transparent button covering the
					     still; pressing swaps the still for the component
					     video's motion and pressing again swaps back. -->
					<button
						class="viewer__press absolute inset-0 z-(--z-stage-chrome) block p-0 bg-transparent cursor-pointer focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
						type="button"
						aria-label={livePlaying ? $_('viewer_live_stop') : $_('viewer_live_play')}
						aria-pressed={livePlaying}
						onclick={() => void pressLive()}
					></button>
				{/if}
			{/if}
		</div>

		{#if sequence.length > 1}
			<nav class="viewer__filmstrip shrink-0 pt-2 px-3 pb-3 overflow-x-auto" aria-label={$_('viewer_filmstrip_label')} bind:this={filmstrip}>
				<ul class="viewer__filmstrip-list flex gap-1.5 list-none">
					{#each sequence as entry (entry.id)}
						<li>
							<button
								class="viewer__thumb w-[52px] h-[52px] p-0 overflow-hidden border-2 rounded-control bg-transparent cursor-pointer shrink-0"
								class:viewer__thumb--current={entry.id === detail?.id}
								class:border-accent={entry.id === detail?.id}
								class:border-transparent={entry.id !== detail?.id}
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
			<button
				class="viewer__arrow viewer__arrow--left absolute left-4 top-1/2 -translate-y-1/2 z-(--z-stage-arrow) inline-flex items-center justify-center w-11 h-11 rounded-full text-cream-text bg-night/55 backdrop-blur-[2px] cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent hover:text-accent-on"
				type="button"
				aria-label={$_('viewer_previous')}
				onclick={() => navigate(-1)}
			>
				<ChevronLeft size={26} />
			</button>
		{/if}
		{#if detail !== null && position !== -1 && position < sequence.length - 1}
			<button
				class="viewer__arrow viewer__arrow--right absolute right-4 top-1/2 -translate-y-1/2 z-(--z-stage-arrow) inline-flex items-center justify-center w-11 h-11 rounded-full text-cream-text bg-night/55 backdrop-blur-[2px] cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent hover:text-accent-on"
				type="button"
				aria-label={$_('viewer_next')}
				onclick={() => navigate(1)}
			>
				<ChevronRight size={26} />
			</button>
		{/if}
	</div>
	</div>

	{#if drawerOpen && detail !== null}
		<!-- The docked split region (R2-10): a flex sibling of the media
		     column at lg+ (lg:static inside the row -- the stage re-fits
		     through the flow), the fixed overlay drawer below lg. -->
		<InfoDrawer
			{detail}
			{isOwner}
			onclose={() => (drawerOpen = false)}
			{onedit}
			{onassignface}
			{onassigntag}
		/>
	{/if}
</div>

<style>
	/* Residual styles: the entrance keyframes and the zoomed stage's
	 * transform on its media children (one rule reaching three
	 * different elements). */
	.viewer {
		animation: viewer-in var(--motion-fast) var(--ease-standard);
	}

	/* The gradual zoom's application (R2-8): the engine writes
	 * --viewer-zoom on the stage per frame and this one rule scales
	 * the three media children off it. The rule is unconditional
	 * (not gated on the zoomed marker class) so a zoom-OUT animates
	 * the same way a zoom-in does -- a class gate would drop the
	 * transform mid-animation and snap the exit. */
	.viewer__image,
	.viewer__placeholder,
	.viewer__live {
		transform: scale(var(--viewer-zoom, 1));
	}

	@keyframes viewer-in {
		from {
			opacity: 0;
		}
	}
</style>
