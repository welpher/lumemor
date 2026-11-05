<script lang="ts">
	// The full-screen asset viewer: the media stage (an image's
	// blurhash placeholder with the view-size and full-resolution
	// serving copies crossfading over it, a video's transcoded
	// stream behind its poster, a Live Photo's press-toggle between
	// the still and the component video's play stream), the basic
	// image zoom, the arrow-key walk over the timeline wall's
	// current page sequence with the bottom filmstrip, and the
	// layered Escape rule (the page's assign dialog first, then the
	// info drawer, then the route).
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
	// OWNERSHIP SPLIT: this component owns the viewer's interaction
	// state (zoom, live playback, drawer open) and every
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
	//   own Escape), then the drawer, then the route exits. Keys
	//   typed into an editable field (the description editor) never
	//   reach the viewer.
	import { decode } from 'blurhash';
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import InfoDrawer from './InfoDrawer.svelte';
	import {
		assetThumbnailUrl,
		assetPlayUrl,
		type AssetDetail,
		type AssetFaceSummary
	} from '$lib/api/assets';
	import type { BatchAction, TimelineItem } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Star from '@lucide/svelte/icons/star';
	import Info from '@lucide/svelte/icons/info';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import ZoomIn from '@lucide/svelte/icons/zoom-in';
	import ZoomOut from '@lucide/svelte/icons/zoom-out';

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
		onassignface
	}: Props = $props();

	let drawerOpen = $state(false);
	let zoomed = $state(false);
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
	// The drawer persists across neighbors regardless (comparing
	// metadata across neighbors is the drawer-open navigation's
	// point).
	let resetFor = '';
	$effect(() => {
		const id = detail?.id ?? '';
		if (id === resetFor) {
			return;
		}
		resetFor = id;
		zoomed = false;
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
			// drawer, then the route. The assign dialog (modalOpen)
			// owns its own Escape; this handler defers to it so the
			// drawer underneath stays open.
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
		if (event.key === 'i' || event.key === 'I') {
			drawerOpen = !drawerOpen;
		}
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
     over the media keep their glass. -->
<div class="viewer fixed inset-0 z-(--z-page) flex flex-col bg-night" aria-busy={status === 'loading'}>
	<!-- The action bar (R2-3): the top chrome over the media, Night
	     surface, icon buttons named by their aria-labels -- the close
	     mark, the favorite toggle, the trash/restore pair (owner-only)
	     and the info-panel toggle; the zoom rides with them on images.
	     The last failed action's mapped error lands on the bar's wrap
	     row (the batch bar's pattern). -->
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
			<!-- The action cluster shifts left of the open info panel (the
			     panel is full-height over the right edge; without the
			     shift its 360px would cover these buttons -- the zoom and
			     info toggles were covered before the rework, the favorite
			     and trash would have joined them). -->
			<div class="viewer__bar-actions flex gap-1 {drawerOpen ? 'me-[calc(min(360px,92vw)+1rem)]' : ''}">
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
					<button
						class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
						type="button"
						aria-label={zoomed ? $_('viewer_zoom_out') : $_('viewer_zoom_in')}
						aria-pressed={zoomed}
						onclick={() => (zoomed = !zoomed)}
					>
						{#if zoomed}
							<ZoomOut size={20} />
						{:else}
							<ZoomIn size={20} />
						{/if}
					</button>
				{/if}
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
		<!-- Basic zoom: the stage's media doubles around its center (the
		     zoomed children carry the transform in the residual styles
		     below). -->
		<div
			class="viewer__stage relative flex-1 min-h-0 flex items-center justify-center overflow-hidden"
			class:viewer__stage--zoomed={zoomed}
			class:cursor-zoom-out={zoomed}
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
						class="viewer__image absolute inset-0 m-auto max-w-full max-h-full object-contain transition-[opacity,transform] duration-(--motion-base) ease-standard"
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
						class="viewer__image viewer__image--full absolute inset-0 m-auto z-(--z-stage-media) max-w-full max-h-full object-contain transition-[opacity,transform] duration-(--motion-base) ease-standard"
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
	<InfoDrawer
		{detail}
		{isOwner}
		onclose={() => (drawerOpen = false)}
		{onedit}
		{onassignface}
	/>
{/if}

<style>
	/* Residual styles (verbatim): the entrance keyframes and the
	 * zoomed stage's transform on its media children (a parent state
	 * class reaching three different elements). */
	.viewer {
		animation: viewer-in var(--motion-fast) var(--ease-standard);
	}

	.viewer__stage--zoomed .viewer__image,
	.viewer__stage--zoomed .viewer__placeholder,
	.viewer__stage--zoomed .viewer__live {
		transform: scale(2);
	}

	@keyframes viewer-in {
		from {
			opacity: 0;
		}
	}
</style>
