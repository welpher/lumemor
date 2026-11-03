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
	import { assetThumbnailUrl, assetPlayUrl, type AssetDetail } from '$lib/api/assets';
	import type { BatchAction, TimelineItem } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';

	type Props = {
		/** The current asset's detail; null while loading. */
		detail: AssetDetail | null;
		/** The favorite flag (the detail carries none; the page holds it). */
		favorite: boolean;
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
	};

	let {
		detail,
		favorite,
		sequence,
		status,
		errorKey,
		videoPlayable,
		busy,
		actionErrorKey,
		onnavigate,
		onexit,
		onretry,
		onprobe,
		onedit,
		onaction,
		ondelete
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
	// so the crossfades, zoom and live playback start over. The
	// drawer persists (comparing metadata across neighbors is the
	// drawer-open navigation's point).
	$effect(() => {
		void detail?.id;
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
			// The layered dismissal: the dialog, then the drawer, then
			// the route. The modal's own Escape handling coexists --
			// its close is idempotent and it no-ops once closed.
			if (confirmOpen) {
				confirmOpen = false;
				return;
			}
			if (drawerOpen) {
				drawerOpen = false;
				return;
			}
			onexit();
			return;
		}
		if (confirmOpen) {
			// The destructive confirm owns every other key.
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

<div class="viewer" aria-busy={status === 'loading'}>
	<header class="viewer__bar">
		<button class="viewer__icon" type="button" aria-label={$_('viewer_close')} onclick={onexit}>
			<svg viewBox="0 0 24 24" width="22" height="22" fill="none" aria-hidden="true">
				<path d="M15 5l-7 7 7 7" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" />
			</svg>
		</button>
		<p class="viewer__title">{detail?.originalFilename ?? ''}</p>
		<div class="viewer__bar-actions">
			{#if isImage && detail !== null}
				<button
					class="viewer__icon"
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
				class="viewer__icon"
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

	<div class="viewer__stage-wrap">
		<div class="viewer__stage" class:viewer__stage--zoomed={zoomed}>
			{#if status === 'error'}
				<div class="viewer__failure" role="alert">
					<p>{errorText(errorKey)}</p>
					<Button variant="primary" onclick={onretry}>{$_('action_retry')}</Button>
				</div>
			{:else if detail === null}
				<p class="viewer__status" role="status">{$_('viewer_loading')}</p>
			{:else if detail.kind === 'video'}
				<!-- svelte-ignore a11y_media_has_caption -->
				{#if videoPlayable}
					<video
						class="viewer__video"
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
						<canvas class="viewer__placeholder" width="32" height="32" bind:this={canvas} aria-hidden="true"></canvas>
					{:else}
						<div class="viewer__placeholder viewer__placeholder--empty" aria-hidden="true"></div>
					{/if}
					{#if !posterFailed}
						<img
							class="viewer__poster"
							src={assetThumbnailUrl(detail.id, 'view')}
							alt={detail.originalFilename}
							onerror={() => (posterFailed = true)}
						/>
					{/if}
					<p class="viewer__processing" role="status">{$_('viewer_processing')}</p>
				{/if}
			{:else}
				{#if detail.blurhash !== null}
					<canvas class="viewer__placeholder" width="32" height="32" bind:this={canvas} aria-hidden="true"></canvas>
				{:else}
					<div class="viewer__placeholder viewer__placeholder--empty" aria-hidden="true"></div>
				{/if}

				{#if detail.isReady}
					<img
						class="viewer__image"
						class:viewer__image--loaded={viewLoaded}
						src={assetThumbnailUrl(detail.id, 'view')}
						alt={detail.originalFilename}
						onload={() => (viewLoaded = true)}
						onerror={() => (viewLoaded = false)}
						draggable="false"
					/>
					<img
						class="viewer__image viewer__image--full"
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
						class="viewer__live"
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
						class="viewer__press"
						type="button"
						aria-label={livePlaying ? $_('viewer_live_stop') : $_('viewer_live_play')}
						aria-pressed={livePlaying}
						onclick={() => void pressLive()}
					></button>
				{/if}
			{/if}
		</div>

		{#if sequence.length > 1}
			<nav class="viewer__filmstrip" aria-label={$_('viewer_filmstrip_label')} bind:this={filmstrip}>
				<ul class="viewer__filmstrip-list">
					{#each sequence as entry (entry.id)}
						<li>
							<button
								class="viewer__thumb"
								class:viewer__thumb--current={entry.id === detail?.id}
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
				class="viewer__arrow viewer__arrow--left"
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
				class="viewer__arrow viewer__arrow--right"
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
		{busy}
		errorKey={actionErrorKey}
		onclose={() => (drawerOpen = false)}
		{onedit}
		{onaction}
		onrequestdelete={() => {
			deleteErrorKey = '';
			confirmOpen = true;
		}}
	/>
{/if}

<Modal open={confirmOpen} title={$_('viewer_delete_title')} onclose={() => (confirmOpen = false)}>
	{#snippet children()}
		<p class="viewer__confirm-body">
			{$_('viewer_delete_body', { values: { name: detail?.originalFilename ?? '' } })}
		</p>
		{#if deleteErrorKey !== ''}
			<p class="viewer__confirm-error" role="alert">{errorText(deleteErrorKey)}</p>
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
	.viewer {
		position: fixed;
		inset: 0;
		z-index: 40;
		display: flex;
		flex-direction: column;
		/* The stage's scrim: the brand night tone in both themes. */
		background-color: rgb(20 26 46 / 0.92);
		backdrop-filter: blur(6px);
		animation: viewer-in var(--motion-fast) var(--ease-standard);
	}

	.viewer__bar {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 16px;
		color: var(--color-cream-text);
	}

	.viewer__icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 38px;
		height: 38px;
		border: none;
		border-radius: var(--radius-control);
		background: none;
		color: var(--color-cream-text);
		cursor: pointer;
		transition: background-color var(--motion-fast) var(--ease-standard);
	}

	.viewer__icon:hover {
		background-color: rgb(255 255 255 / 0.12);
	}

	.viewer__icon:focus-visible {
		outline: 2px solid var(--color-blue-300);
		outline-offset: -2px;
	}

	.viewer__title {
		flex: 1;
		margin: 0;
		font: 600 15px/1.4 var(--font-body);
		overflow-wrap: anywhere;
	}

	.viewer__bar-actions {
		display: flex;
		gap: 4px;
	}

	.viewer__stage-wrap {
		position: relative;
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
	}

	.viewer__stage {
		position: relative;
		flex: 1;
		min-height: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
	}

	.viewer__stage--zoomed {
		/* Basic zoom: the stage's media doubles around its center. */
		cursor: zoom-out;
	}

	.viewer__placeholder {
		position: absolute;
		width: 100%;
		height: 100%;
		object-fit: contain;
	}

	.viewer__placeholder--empty {
		background: linear-gradient(
			135deg,
			color-mix(in srgb, var(--color-night-surface) 70%, var(--color-night)),
			var(--color-night-surface)
		);
	}

	.viewer__image,
	.viewer__video,
	.viewer__live,
	.viewer__poster {
		max-width: 100%;
		max-height: 100%;
	}

	.viewer__image {
		position: absolute;
		inset: 0;
		margin: auto;
		object-fit: contain;
		opacity: 0;
		transition:
			opacity var(--motion-base) var(--ease-standard),
			transform var(--motion-base) var(--ease-standard);
	}

	.viewer__image--loaded {
		opacity: 1;
	}

	/* The full-resolution layer rides above the view-size copy. */
	.viewer__image--full {
		z-index: 1;
	}

	.viewer__stage--zoomed .viewer__image,
	.viewer__stage--zoomed .viewer__placeholder,
	.viewer__stage--zoomed .viewer__live {
		transform: scale(2);
	}

	.viewer__video,
	.viewer__poster {
		position: relative;
		z-index: 1;
	}

	.viewer__live {
		position: relative;
		z-index: 2;
		object-fit: contain;
	}

	.viewer__status,
	.viewer__processing {
		position: relative;
		z-index: 3;
		margin: 0;
		padding: 12px 20px;
		font: 500 14px/1.4 var(--font-body);
		color: var(--color-cream-text);
		text-align: center;
		border-radius: var(--radius-control);
		background-color: rgb(20 26 46 / 0.55);
		backdrop-filter: blur(2px);
	}

	.viewer__processing {
		position: absolute;
		bottom: 24px;
	}

	.viewer__failure {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		padding: 20px;
		text-align: center;
	}

	.viewer__failure p {
		margin: 0;
		font: 500 15px/1.4 var(--font-body);
		color: var(--color-cream-text);
	}

	.viewer__press {
		position: absolute;
		inset: 0;
		z-index: 3;
		display: block;
		padding: 0;
		border: none;
		background: none;
		cursor: pointer;
	}

	.viewer__press:focus-visible {
		outline: 2px solid var(--color-blue-300);
		outline-offset: -2px;
	}

	.viewer__filmstrip {
		flex-shrink: 0;
		padding: 8px 12px 12px;
		overflow-x: auto;
	}

	.viewer__filmstrip-list {
		display: flex;
		gap: 6px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.viewer__thumb {
		width: 52px;
		height: 52px;
		padding: 0;
		overflow: hidden;
		border: 2px solid transparent;
		border-radius: var(--radius-control);
		background: none;
		cursor: pointer;
		flex-shrink: 0;
	}

	.viewer__thumb img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.viewer__thumb--current {
		border-color: var(--color-accent);
	}

	.viewer__arrow {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		z-index: 4;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 44px;
		height: 44px;
		border: none;
		border-radius: 50%;
		color: var(--color-cream-text);
		background-color: rgb(20 26 46 / 0.55);
		backdrop-filter: blur(2px);
		cursor: pointer;
		transition: background-color var(--motion-fast) var(--ease-standard);
	}

	.viewer__arrow:hover {
		background-color: var(--color-accent);
		color: var(--color-accent-on);
	}

	.viewer__arrow--left {
		left: 16px;
	}

	.viewer__arrow--right {
		right: 16px;
	}

	.viewer__confirm-body {
		margin: 0;
		font: 400 15px/1.5 var(--font-body);
	}

	.viewer__confirm-error {
		margin: 12px 0 0;
		font: 500 13px/1.4 var(--font-body);
		color: var(--color-danger-text);
	}

	@keyframes viewer-in {
		from {
			opacity: 0;
		}
	}
</style>
