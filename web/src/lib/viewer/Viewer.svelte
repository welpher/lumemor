<script lang="ts">
	// The full-screen asset viewer: the media stage (an image's
	// blurhash placeholder with the view-size and full-resolution
	// serving copies crossfading over it, a video's transcoded
	// stream behind its poster, a Live Photo's press-toggle between
	// the still and the component video's play stream), the basic
	// image zoom, the arrow-key walk over the timeline wall's
	// current page sequence with the bottom filmstrip, and the
	// layered Escape rule (the confirm dialog first, then the info
	// drawer, then the route).
	//
	// OWNERSHIP SPLIT: this component owns the viewer's interaction
	// state (zoom, live playback, drawer and dialog open) and every
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
	// - Escape layers: the delete dialog closes first, then the
	//   drawer, then the route exits. Keys typed into an editable
	//   field (the description editor) never reach the viewer.
	import { decode } from 'blurhash';
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import InfoDrawer from './InfoDrawer.svelte';
	import {
		assetThumbnailUrl,
		assetPlayUrl,
		type AssetDetail,
		type AssetFaceSummary
	} from '$lib/api/assets';
	import type { BatchAction, TimelineItem } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';

	type Props = {
		/** The current asset's detail; null while loading. */
		detail: AssetDetail | null;
		/** The favorite flag (the detail carries none; the page holds it). */
		favorite: boolean;
		/**
		 * Whether the caller owns the open asset (the session's userId
		 * against the detail's ownerId): the drawer's owner-only write
		 * affordances render for the owner alone.
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
		/** Run the permanent delete; ok means the page left the viewer. */
		ondelete: () => Promise<{ ok: boolean; errorKey: string }>;
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
		ondelete,
		onassignface
	}: Props = $props();

	let drawerOpen = $state(false);
	let confirmOpen = $state(false);
	let deleteErrorKey = $state('');
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
		deleteErrorKey = '';
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
			// The layered dismissal: the dialogs, then the drawer, then
			// the route. The modals' own Escape handling coexists --
			// their close is idempotent and it no-ops once closed. The
			// page's assign dialog (modalOpen) owns its own Escape; this
			// handler defers to it so the drawer underneath stays open.
			if (confirmOpen) {
				confirmOpen = false;
				return;
			}
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
		if (confirmOpen || modalOpen) {
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

	async function runDelete(): Promise<void> {
		if (busy) {
			return;
		}
		deleteErrorKey = '';
		const outcome = await ondelete();
		if (outcome.ok) {
			confirmOpen = false;
			return;
		}
		// The failure keeps the dialog open with the mapped message
		// (a retry affordance); nothing was deleted.
		deleteErrorKey = outcome.errorKey;
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
<div class="viewer fixed inset-0 z-[40] flex flex-col bg-night" aria-busy={status === 'loading'}>
	<header class="viewer__bar flex items-center gap-3 px-4 py-2.5 text-cream-text">
		<button
			class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
			type="button"
			aria-label={$_('viewer_close')}
			onclick={onexit}
		>
			<svg viewBox="0 0 24 24" width="22" height="22" fill="none" aria-hidden="true">
				<path d="M15 5l-7 7 7 7" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" />
			</svg>
		</button>
		<p class="viewer__title flex-1 font-body font-semibold text-[15px] leading-[1.4] [overflow-wrap:anywhere]">{detail?.originalFilename ?? ''}</p>
		<div class="viewer__bar-actions flex gap-1">
			{#if isImage && detail !== null}
				<button
					class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
					type="button"
					aria-label={zoomed ? $_('viewer_zoom_out') : $_('viewer_zoom_in')}
					aria-pressed={zoomed}
					onclick={() => (zoomed = !zoomed)}
				>
					<svg viewBox="0 0 24 24" width="20" height="20" fill="none" aria-hidden="true">
						<circle cx="11" cy="11" r="6.25" stroke="currentColor" stroke-width="1.75" />
						<path d="M8.25 11h5.5M11 8.25v5.5M16 16l4.5 4.5" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
					</svg>
				</button>
			{/if}
			<button
				class="viewer__icon inline-flex items-center justify-center w-[38px] h-[38px] rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-[rgb(255_255_255/0.12)] focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
				type="button"
				aria-label={$_('viewer_info_open')}
				aria-pressed={drawerOpen}
				onclick={() => (drawerOpen = !drawerOpen)}
			>
				<svg viewBox="0 0 24 24" width="20" height="20" fill="none" aria-hidden="true">
					<circle cx="12" cy="12" r="8.25" stroke="currentColor" stroke-width="1.75" />
					<path d="M12 11v5.5" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
					<circle cx="12" cy="7.9" r="1.05" fill="currentColor" />
				</svg>
			</button>
		</div>
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
				<p class="viewer__status relative z-[3] px-5 py-3 font-body font-medium text-[14px] leading-[1.4] text-cream-text text-center rounded-control bg-night/55 backdrop-blur-[2px]" role="status">{$_('viewer_loading')}</p>
			{:else if detail.kind === 'video'}
				<!-- svelte-ignore a11y_media_has_caption -->
				{#if videoPlayable}
					<video
						class="viewer__video relative z-[1] max-w-full max-h-full"
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
							class="viewer__poster relative z-[1] max-w-full max-h-full"
							src={assetThumbnailUrl(detail.id, 'view')}
							alt={detail.originalFilename}
							onerror={() => (posterFailed = true)}
						/>
					{/if}
					<p class="viewer__processing absolute bottom-6 z-[3] px-5 py-3 font-body font-medium text-[14px] leading-[1.4] text-cream-text text-center rounded-control bg-night/55 backdrop-blur-[2px]" role="status">{$_('viewer_processing')}</p>
				{/if}
			{:else}
				{#if detail.blurhash !== null}
					<canvas class="viewer__placeholder absolute w-full h-full object-contain" width="32" height="32" bind:this={canvas} aria-hidden="true"></canvas>
				{:else}
					<div
						class="viewer__placeholder viewer__placeholder--empty absolute w-full h-full bg-[linear-gradient(135deg,color-mix(in_srgb,var(--color-night-surface)_70%,var(--color-night)),var(--color-night-surface))]"
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
						class="viewer__image viewer__image--full absolute inset-0 m-auto z-[1] max-w-full max-h-full object-contain transition-[opacity,transform] duration-(--motion-base) ease-standard"
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
						class="viewer__live relative z-[2] max-w-full max-h-full object-contain"
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
						class="viewer__press absolute inset-0 z-[3] block p-0 bg-transparent cursor-pointer focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
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
				class="viewer__arrow viewer__arrow--left absolute left-4 top-1/2 -translate-y-1/2 z-[4] inline-flex items-center justify-center w-11 h-11 rounded-full text-cream-text bg-night/55 backdrop-blur-[2px] cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent hover:text-accent-on"
				type="button"
				aria-label={$_('viewer_previous')}
				onclick={() => navigate(-1)}
			>
				<svg viewBox="0 0 24 24" width="26" height="26" fill="none" aria-hidden="true">
					<path d="M15 5l-7 7 7 7" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" />
				</svg>
			</button>
		{/if}
		{#if detail !== null && position !== -1 && position < sequence.length - 1}
			<button
				class="viewer__arrow viewer__arrow--right absolute right-4 top-1/2 -translate-y-1/2 z-[4] inline-flex items-center justify-center w-11 h-11 rounded-full text-cream-text bg-night/55 backdrop-blur-[2px] cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent hover:text-accent-on"
				type="button"
				aria-label={$_('viewer_next')}
				onclick={() => navigate(1)}
			>
				<svg viewBox="0 0 24 24" width="26" height="26" fill="none" aria-hidden="true">
					<path d="M9 5l7 7-7 7" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" />
				</svg>
			</button>
		{/if}
	</div>
</div>

{#if drawerOpen && detail !== null}
	<InfoDrawer
		{detail}
		{favorite}
		{isOwner}
		{busy}
		errorKey={actionErrorKey}
		onclose={() => (drawerOpen = false)}
		{onedit}
		{onaction}
		onrequestdelete={() => {
			deleteErrorKey = '';
			confirmOpen = true;
		}}
		{onassignface}
	/>
{/if}

<Modal open={confirmOpen} title={$_('viewer_delete_title')} onclose={() => (confirmOpen = false)}>
	{#snippet children()}
		<p class="viewer__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
			{$_('viewer_delete_body', { values: { name: detail?.originalFilename ?? '' } })}
		</p>
		{#if deleteErrorKey !== ''}
			<p class="viewer__confirm-error mt-3 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{errorText(deleteErrorKey)}</p>
		{/if}
	{/snippet}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={() => (confirmOpen = false)}>
			{$_('common_cancel')}
		</Button>
		<Button variant="danger" disabled={busy} onclick={() => void runDelete()}>
			{$_('viewer_delete_confirm')}
		</Button>
	{/snippet}
</Modal>

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
