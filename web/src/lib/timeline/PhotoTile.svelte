<script lang="ts">
	// One tile of the photo walls: the blurhash placeholder paints
	// first (a 32x32 decode stretched over the tile box), the
	// thumbnail request only fires once the pipeline marked the asset
	// ready (before that the endpoint answers ASSET_NOT_READY -- pure
	// noise), and the image crossfades in over the placeholder on its
	// load event. The browser's cache does the ETag/immutable work;
	// the tile just sets the src.
	//
	// THE TILE'S BOX (the justified wall): the tile renders at its
	// TRUE aspect, but it no longer sizes itself -- the row layout
	// engine (see ./layout.ts) computes the box from the item's
	// dimensions and the caller wraps this tile in a sized cell. A
	// RESTING tile is pure photo edge-to-edge: no radius, no border,
	// no frame -- the photo IS the surface. Selection is geometric:
	// the tile retreats (a scale), gains its radius, and an accent
	// ring draws at its edge -- the whitespace IS the selected state,
	// no reflow, no checkbox geometry change.
	//
	// THE SERVING COPY (the 2026-11-07 blurry fix): the
	// tile ALWAYS serves the view derivative -- the tile/view DPR
	// srcset pair is gone. The pair relied on the browser
	// re-evaluating its responsive-image candidate after the row
	// engine re-packed and the `sizes` hint changed on an
	// already-loaded <img>, which it does not reliably do -- so a
	// window resize re-packed the wall and left the soft candidate
	// on screen. The tile derivative itself was sized for a uniform
	// ~250px grid, not the justified wall's 200-500px variable
	// cells, so the sharper candidate was doing the real work
	// anyway; a self-hosted LAN app's bandwidth cost is negligible
	// and the photos are ALWAYS sharp. (The viewer's image branch
	// keeps its own view/original pair -- untouched.)
	//
	// THE LOST LOAD EVENT (the 2026-11-12 resize blur): a DIFFERENT
	// problem from the serving copy above, never to be conflated with
	// it. That pin was the browser re-estimating a responsive
	// candidate; this one is the `load` event itself going missing
	// when the wall's re-pack re-creates a tile whose image already
	// sits in the memory cache (the effect below reads the element's
	// own completion state instead of waiting for an event nobody
	// fires).
	//
	// THE LIVE PHOTO (the still is the asset, the moment is an
	// attachment): TimelineItem carries livePhotoVideoId, so the tile
	// marks the type with a concentric-ring glyph at the top-right --
	// a BARE glyph with a drop shadow, no plate: the selection check
	// already owns a circle at the top-left, and a second circle there
	// would read as a second control. The glyph is decoration
	// (aria-hidden); the anchor's accessible name carries the type
	// ("... · live photo").
	//
	// HOVER PLAYS THE MOMENT: entering the tile (pointer or keyboard
	// focus) mounts the component video's play stream -- muted,
	// looping, no controls -- over the still; once it is actually
	// playing the tile takes its slow push (1 -> 1.07 with a micro
	// pan) and the glyph flips to the pause mark, so the mark always
	// tells the truth about what is moving. Leaving resets both. A
	// motion that cannot play (the component's transcode still
	// running, the stream refusing) keeps the still and the type
	// glyph: no push, no toast, and the next hover tries again.
	// `prefers-reduced-motion` collapses the push through the theme's
	// global rule; the mark still flips (the glyph is information,
	// not decoration). The hidden video never becomes a tile of its
	// own -- it is not even fetched until a hover asks.
	//
	// THE HOVER AFFORDANCES: a restrained
	// bottom scrim of translucent Night rises on hover and keyboard
	// focus — the photo never scales, it stays authoritative. The
	// scrim reveals the tile's STATE (the favorite star, bottom-left)
	// while the TYPE marks stay readable at rest (the live badge
	// top-left, the video chip bottom-right). The video chip carries
	// the duration: an opaque night chip (the batch-1 ink-chip
	// vocabulary), tabular numerals, a subtle text shadow, and a
	// container-query scale-down so narrow portrait tiles shrink it
	// instead of overflowing. There is no stack count yet — nothing
	// invents data the contract does not carry.
	//
	// Selection interaction: the check circle appears on hover and
	// whenever a selection or the select mode is active (the
	// touch-friendly path); with selection active the whole tile
	// toggles, otherwise the tile is the anchor into the asset
	// viewer. `aria-checked` and the keyboard semantics stay exactly
	// as they were -- only the visuals moved. A caller that wires NO
	// onselect callback gets NO check circle at all (the
	// no-selection walls: the sync album, the trash grid).
	import { decode } from 'blurhash';
	import { _ } from 'svelte-i18n';
	import type { TimelineItem } from '$lib/api/timeline';
	import { assetPlayUrl } from '$lib/api/assets';
	import { formatDuration } from '$lib/format';

	// The tile-relevant subset of the timeline item shape: every wall
	// item (the trash listing's rows included) structurally satisfies
	// it, so the tile renders wherever the wall's item shape lives.
	type TileItem = Pick<
		TimelineItem,
		'id' | 'kind' | 'originalFilename' | 'isFavorite' | 'isReady' | 'blurhash' | 'livePhotoVideoId' | 'durationMs'
	>;

	type Props = {
		item: TileItem;
		/** Whether the tile is in the current selection. */
		selected: boolean;
		/** Whether checkboxes show and clicks toggle (select mode or an active selection). */
		selectable: boolean;
		/**
		 * The selection toggle. ABSENT = the surface offers no
		 * selection at all (the sync album's wall, the trash grid):
		 * NO check circle renders -- not even the hover reveal -- and
		 * the tile body stays the pure viewer anchor. The callback's
		 * presence is the capability, so no caller ever wires a dead
		 * affordance.
		 */
		onselect?: (id: string, selected: boolean) => void;
	};

	let { item, selected, selectable, onselect }: Props = $props();

	let loaded = $state(false);
	let canvas: HTMLCanvasElement | undefined = $state();
	let image: HTMLImageElement | undefined = $state();

	/** Whether this tile is a Live Photo: an image with its moment attached. */
	const isLive = $derived(item.kind === 'image' && item.livePhotoVideoId !== null);

	/** A hover/focus asked for the moment (the stream mounts here alone). */
	let motionAsked = $state(false);
	/** The moment answered and is actually playing (the push's gate). */
	let motionPlaying = $state(false);
	/** The moment's stream refused (transcode pending, file gone): stay on the still. */
	let motionBroken = $state(false);

	/** Ask for the moment: mount the stream and wait for it to play. */
	function startMotion(): void {
		if (!isLive || motionBroken) {
			return;
		}
		motionAsked = true;
	}

	/**
	 * Drop the moment: unmount the stream and reset the push. A
	 * refusal is forgotten here, so the next hover asks again -- the
	 * component's transcode may have landed in between.
	 */
	function stopMotion(): void {
		motionAsked = false;
		motionPlaying = false;
		motionBroken = false;
	}

	/** The view-size serving copy (always -- see the serving pin above). */
	const viewUrl = $derived(`/api/v1/assets/${encodeURIComponent(item.id)}/thumbnail?size=view`);

	// THE CACHE-HIT RACE (the resize blur): the wall re-packs on a
	// container resize and moves tiles between rows, so Svelte
	// destroys and re-creates the tile; the fresh <img> gets its src
	// before Svelte attaches onload, and a fully decoded image from
	// the memory cache can dispatch `load` synchronously inside that
	// window. That event never reaches the component and nobody
	// fires it again, so the tile would sit on its blurhash forever.
	// The old unconditional `loaded = false` then wiped the only
	// signal the element had. The mount check reads the element's own
	// completion state instead: a cache hit shows immediately, an
	// async load still lands through onload.
	$effect(() => {
		void item.id;
		void item.isReady;
		const element = image;
		loaded = element !== undefined && element.complete && element.naturalWidth > 0;
	});

	// Paint the blurhash placeholder. jsdom (and any context-less
	// host) answers a null context: the tile then keeps its CSS
	// skeleton fill. A hash that fails validation leaves the skeleton
	// too -- never a crash.
	$effect(() => {
		const hash = item.blurhash;
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
			// An unparsable hash: the skeleton fill carries the tile.
		}
	});

	function toggle(): void {
		if (onselect !== undefined) {
			onselect(item.id, !selected);
		}
	}

	function handleTileClick(event: MouseEvent): void {
		if (selectable && onselect !== undefined) {
			event.preventDefault();
			toggle();
		}
	}
</script>

<!-- One cell of the justified wall: the caller's sized cell carries
     the box (the row's height and the engine's width); this tile
     fills it edge-to-edge. The resting tile is the photo alone; the
     selected state is geometric -- the retreat scale, the radius and
     the accent ring at the retreated edge, all transform/outline so
     nothing reflows. The ink chips (badges, check) are opaque night
     surfaces over the photo, no blur. The tile is a size container
     so its duration chip can scale itself down on narrow tiles. The
     content-visibility pair keeps offscreen tiles out of layout and
     paint (rows add their own windowing above it).

     The hover/focus pair previews the media -- it starts no action
     -- so the envelope needs no widget role; the anchor inside it is
     the tile's one interactive element. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="tile @container group relative w-full h-full overflow-hidden bg-accent-soft [content-visibility:auto] transition-[scale,border-radius] duration-(--motion-base) ease-standard {selected
		? 'rounded-control scale-(--wall-select-scale) [outline:var(--wall-select-ring)_solid_var(--color-accent)] [outline-offset:calc(var(--wall-select-ring)*-1)]'
		: ''}"
	class:tile--selected={selected}
	class:tile--selectable={selectable}
	class:cursor-pointer={selectable}
	onmouseenter={startMotion}
	onmouseleave={stopMotion}
	onfocusin={startMotion}
	onfocusout={stopMotion}
>
	<a
		class="tile__link absolute inset-0 focus-visible:[outline:2px_solid_var(--color-accent)] focus-visible:outline-offset-[-2px]"
		href="/a/{encodeURIComponent(item.id)}"
		onclick={handleTileClick}
		aria-label={isLive ? `${item.originalFilename} · ${$_('asset_live_photo_label')}` : item.originalFilename}
	>
		<!-- The media region: the placeholder, the still and (on a
		     hover) the moment. The slow push lives here, so the still
		     and the motion breathe as ONE picture -- the chips and the
		     check ride outside the push, since a control that drifts
		     under the pointer is a control that fights it. -->
		<div class="tile__media absolute inset-0" class:tile__media--moving={motionPlaying} aria-hidden="true">
			{#if item.blurhash !== null}
				<canvas
					class="tile__placeholder absolute inset-0 w-full h-full object-cover transition-[opacity] duration-(--motion-placeholder-fade) ease-standard"
					class:tile__placeholder--hidden={loaded}
					class:opacity-0={loaded}
					width="32"
					height="32"
					bind:this={canvas}
					aria-hidden="true"
				></canvas>
			{:else}
				<div
					class="tile__placeholder tile__placeholder--empty absolute inset-0 w-full h-full bg-[linear-gradient(135deg,color-mix(in_srgb,var(--color-accent-soft)_60%,var(--color-card)),var(--color-accent-soft))] transition-[opacity] duration-(--motion-placeholder-fade) ease-standard"
					class:tile__placeholder--hidden={loaded}
					class:opacity-0={loaded}
					aria-hidden="true"
				></div>
			{/if}

			{#if item.isReady}
				<img
					class="tile__image absolute inset-0 w-full h-full object-cover transition-[opacity] duration-(--motion-base) ease-standard"
					class:opacity-0={!loaded}
					class:tile__image--loaded={loaded}
					src={viewUrl}
					alt=""
					loading="lazy"
					decoding="async"
					bind:this={image}
					onload={() => (loaded = true)}
					onerror={() => (loaded = false)}
					draggable="false"
				/>
			{/if}

			{#if motionAsked && !motionBroken && item.livePhotoVideoId !== null}
				<!-- The moment: the component row's play stream, muted and
				     looping, no controls. It is fetched HERE alone -- a
				     hover asks, leaving drops the element -- so the hidden
				     video never costs a byte at rest. -->
				<video
					class="tile__motion absolute inset-0 w-full h-full object-cover z-(--z-stage-media)"
					src={assetPlayUrl(item.livePhotoVideoId)}
					muted
					loop
					autoplay
					playsinline
					onplaying={() => (motionPlaying = true)}
					onerror={() => {
						motionBroken = true;
						motionPlaying = false;
					}}
					aria-hidden="true"
				></video>
			{/if}
		</div>

		<!-- The bottom scrim: the Night-based translucency that rises
		     with the tile's state affordances on hover and keyboard
		     focus. Pure opacity -- the photo never scales, and the
		     pointer never interacts with the veil itself. -->
		<div
			class="tile__scrim absolute inset-x-0 bottom-0 h-1/3 pointer-events-none bg-linear-to-t from-night/55 to-transparent opacity-0 transition-[opacity] duration-(--motion-fast) ease-standard group-hover:opacity-100 group-focus-within:opacity-100"
			aria-hidden="true"
		></div>

		{#if item.isFavorite}
			<!-- The state affordance the scrim reveals: the star rests
			     invisible in the scrim zone (bottom-left) and fades in
			     with it -- an opaque night chip with the gold star, no
			     blur, readable in both themes. -->
			<span
				class="tile__badge tile__badge--favorite absolute bottom-1.5 left-1.5 z-(--z-tile-chip) inline-flex items-center justify-center w-6 h-6 rounded-full text-star bg-night opacity-0 transition-[opacity] duration-(--motion-fast) ease-standard group-hover:opacity-100 group-focus-within:opacity-100"
				title={$_('asset_favorite_label')}
				aria-hidden="true"
			>
				<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor" aria-hidden="true">
					<path d="M12 3.6l2.5 5.2 5.7.7-4.2 4 1.1 5.6-5.1-2.8-5.1 2.8 1.1-5.6-4.2-4 5.7-.7z" />
				</svg>
			</span>
		{/if}

		{#if isLive}
			<!-- The type mark: a BARE glyph with a drop shadow, at the
			     top-right -- no plate behind it. The selection check
			     already draws a filled circle at the top-left; a second
			     disc beside it would read as a second control, and the
			     shadow alone keeps the rings legible on a bright sky
			     and a dark frame alike. The glyph says "this still has
			     a moment attached"; while the moment plays it says
			     "pause", so the mark never lies about what is moving.
			     Decoration, not an action: aria-hidden, the anchor's
			     name carries the type. -->
			<span
				class="tile__live absolute top-2 right-2 z-(--z-tile-chip) inline-flex items-center justify-center text-cream-text drop-shadow-[0_1px_4px_rgb(0_0_0/0.55)]"
				title={$_('asset_live_photo_label')}
				aria-hidden="true"
			>
				{#if motionPlaying}
					<svg viewBox="0 0 24 24" width="15" height="15" fill="none" aria-hidden="true">
						<path d="M9 6v12M15 6v12" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" />
					</svg>
				{:else}
					<svg viewBox="0 0 24 24" width="15" height="15" fill="none" aria-hidden="true">
						<circle
							cx="12"
							cy="12"
							r="9.2"
							stroke="currentColor"
							stroke-width="1.8"
							stroke-dasharray="2.4 3.4"
							stroke-linecap="round"
						/>
						<circle cx="12" cy="12" r="3.6" fill="currentColor" />
					</svg>
				{/if}
			</span>
		{/if}

		{#if item.kind === 'video'}
			<!-- The duration chip: the play mark and the m:ss
			     duration on one opaque night chip (the ink-chip
			     vocabulary), with tabular numerals and the subtle
			     text shadow; the container query scales the whole
			     chip down on narrow (portrait-tile) widths instead
			     of overflowing. An unknown duration (null on the
			     wire) keeps the mark alone. -->
			<span
				class="tile__badge tile__badge--video absolute bottom-1.5 right-1.5 z-(--z-tile-chip) inline-flex items-center gap-1 h-6 px-1.5 rounded-control font-body font-semibold text-xs text-cream-text bg-night tabular-nums text-shadow-badge @max-tile-compact:scale-75"
				title={$_('asset_video_label')}
				aria-hidden="true"
			>
				<svg viewBox="0 0 24 24" width="12" height="12" fill="none" aria-hidden="true">
					<path d="M9 8.2v7.6l6-3.8z" fill="currentColor" />
					<circle cx="12" cy="12" r="8.25" stroke="currentColor" stroke-width="1.75" />
				</svg>
				{#if item.durationMs !== null}
					<span class="tile__duration">{formatDuration(item.durationMs)}</span>
				{/if}
			</span>
		{/if}
	</a>

	<!-- The selection checkbox: rendered ONLY on surfaces that wired
	     a selection callback (the timeline, the curated walls) --
	     absent on the no-selection walls (the sync album, the
	     trash grid), which never show a check at all. On the walls
	     that carry it, the check is revealed on hover (the group on
	     the tile, for pointer users) and whenever selection is
	     active (touch users, mid-selection) -- an opaque night
	     surface with a translucent cream rim, readable over any
	     photo, in both themes, without a blur in sight. -->
	{#if onselect !== undefined}
		<button
			class="tile__check absolute top-1.5 left-1.5 z-(--z-tile-check) inline-flex items-center justify-center w-[26px] h-[26px] p-0 border border-cream-text/35 rounded-full text-transparent bg-night cursor-pointer transition-[opacity,background-color,border-color,color] duration-(--motion-fast) ease-standard group-hover:opacity-100 group-focus-within:opacity-100 aria-checked:bg-accent aria-checked:border-accent aria-checked:text-accent-on"
			class:opacity-0={!selectable}
			class:tile__check--shown={selectable}
			type="button"
			role="checkbox"
			aria-checked={selected}
			aria-label="{$_('action_select')}: {item.originalFilename}"
			onclick={toggle}
		>
			<svg viewBox="0 0 24 24" width="14" height="14" fill="none" aria-hidden="true">
				<path
					d="M5 12.5l4.5 4.5L19 7.5"
					stroke="currentColor"
					stroke-width="2.5"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
			</svg>
		</button>
	{/if}
</div>

<style>
	/* The moment's slow push: the still and the motion breathe as one
	 * picture while the motion plays -- 3.2s alternate, the photo
	 * enlarging a touch while it drifts, so a hovering pointer reads
	 * "this is moving" before it reads any glyph. Reduced motion keeps
	 * the mark and drops the push: the global theme rule already
	 * collapses animation durations, and the explicit rule keeps the
	 * intent legible where the theme's blanket is read as an
	 * implementation detail rather than a decision. */
	.tile__media--moving {
		animation: tile-motion-push 3.2s var(--ease-standard) infinite alternate;
	}

	@keyframes tile-motion-push {
		from {
			transform: scale(1) translate3d(0, 0, 0);
		}
		to {
			transform: scale(1.07) translate3d(-1.6%, -1%, 0);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.tile__media--moving {
			animation: none;
		}
	}
</style>
