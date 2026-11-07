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
	// THE SERVING COPY (the 2026-11-07 blurry fix, owner pin 2): the
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
	// THE PINNED LIVE-PHOTO DECISION: TimelineItem carries
	// livePhotoVideoId, so a Live Photo tile shows its small live
	// badge here; the hover video preview arrives with the viewer and
	// the component transcode (later tasks of this milestone).
	//
	// THE HOVER AFFORDANCES (web polish batch 2, G3+G9): a restrained
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
	// no-selection walls: the album detail, the trash grid).
	import { decode } from 'blurhash';
	import { _ } from 'svelte-i18n';
	import type { TimelineItem } from '$lib/api/timeline';
	import { formatDuration } from '$lib/format';

	// The tile-relevant subset of the timeline item shape: every wall
	// item (the trash listing's rows included) structurally satisfies
	// it, so the tile renders wherever the wall's item shape lives.
	type TileItem = Pick<
		TimelineItem,
		| 'id'
		| 'kind'
		| 'originalFilename'
		| 'isFavorite'
		| 'isReady'
		| 'blurhash'
		| 'livePhotoVideoId'
		| 'durationMs'
	>;

	type Props = {
		item: TileItem;
		/** Whether the tile is in the current selection. */
		selected: boolean;
		/** Whether checkboxes show and clicks toggle (select mode or an active selection). */
		selectable: boolean;
		/**
		 * The selection toggle. ABSENT = the surface offers no
		 * selection at all (the album detail's wall, the trash grid):
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

	/** The view-size serving copy (always -- see the serving pin above). */
	const viewUrl = $derived(`/api/v1/assets/${encodeURIComponent(item.id)}/thumbnail?size=view`);

	// Reset the crossfade when the tile's image identity changes (the
	// item object is replaced by patches, not mutated).
	$effect(() => {
		void item.id;
		void item.isReady;
		loaded = false;
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
     paint (rows add their own windowing above it). -->
<div
	class="tile @container group relative w-full h-full overflow-hidden bg-accent-soft [content-visibility:auto] transition-[scale,border-radius] duration-(--motion-base) ease-standard {selected
		? 'rounded-control scale-(--wall-select-scale) [outline:var(--wall-select-ring)_solid_var(--color-accent)] [outline-offset:calc(var(--wall-select-ring)*-1)]'
		: ''}"
	class:tile--selected={selected}
	class:tile--selectable={selectable}
	class:cursor-pointer={selectable}
>
	<a
		class="tile__link absolute inset-0 focus-visible:[outline:2px_solid_var(--color-accent)] focus-visible:outline-offset-[-2px]"
		href="/a/{encodeURIComponent(item.id)}"
		onclick={handleTileClick}
		aria-label={item.originalFilename}
	>
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
				onload={() => (loaded = true)}
				onerror={() => (loaded = false)}
				draggable="false"
			/>
		{/if}

		<!-- G3's bottom scrim: the Night-based translucency that rises
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

		{#if item.kind === 'image' && item.livePhotoVideoId !== null}
			<span
				class="tile__badge tile__badge--live absolute top-1.5 left-1.5 z-(--z-tile-chip) inline-flex items-center justify-center w-6 h-6 rounded-full text-cream-text bg-night"
				title={$_('asset_live_photo_label')}
				aria-hidden="true"
			>
				<svg viewBox="0 0 24 24" width="14" height="14" fill="none" aria-hidden="true">
					<circle cx="12" cy="12" r="4.5" stroke="currentColor" stroke-width="1.75" />
					<path d="M2.5 12h3M18.5 12h3M12 2.5v3M12 18.5v3" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
				</svg>
			</span>
		{/if}

		{#if item.kind === 'video'}
			<!-- G9's duration chip: the play mark and the m:ss
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
	     absent on the no-selection walls (the album detail, the
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
				<path d="M5 12.5l4.5 4.5L19 7.5" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
			</svg>
		</button>
	{/if}
</div>
