<script lang="ts">
	// One tile of the timeline wall: the blurhash placeholder paints
	// first (a 32x32 decode stretched over the square tile), the
	// thumbnail request only fires once the pipeline marked the asset
	// ready (before that the endpoint answers ASSET_NOT_READY -- pure
	// noise), and the image crossfades in over the placeholder on its
	// load event. The browser's cache does the ETag/immutable work;
	// the tile just sets the src.
	//
	// THE PINNED LIVE-PHOTO DECISION: TimelineItem carries
	// livePhotoVideoId, so a Live Photo tile shows its small live
	// badge here; the hover video preview arrives with the viewer and
	// the component transcode (later tasks of this milestone).
	//
	// Selection: the checkbox appears on hover and whenever a
	// selection or the select mode is active (the touch-friendly
	// path); with selection active the whole tile toggles, otherwise
	// the tile is the anchor into the asset viewer.
	import { decode } from 'blurhash';
	import { _ } from 'svelte-i18n';
	import type { TimelineItem } from '$lib/api/timeline';

	// The tile-relevant subset of the timeline item shape: every wall
	// item (the trash listing's rows included) structurally satisfies
	// it, so the tile renders wherever the wall's item shape lives.
	type TileItem = Pick<
		TimelineItem,
		'id' | 'kind' | 'originalFilename' | 'isFavorite' | 'isReady' | 'blurhash' | 'livePhotoVideoId'
	>;

	type Props = {
		item: TileItem;
		/** Whether the tile is in the current selection. */
		selected: boolean;
		/** Whether checkboxes show and clicks toggle (select mode or an active selection). */
		selectable: boolean;
		/** Toggle callback; the checkbox and the tile body both call it. */
		onselect: (id: string, next: boolean) => void;
	};

	let { item, selected, selectable, onselect }: Props = $props();

	let loaded = $state(false);
	let canvas: HTMLCanvasElement | undefined = $state();

	/** The tile-size serving copy (the timeline box size). */
	const thumbnailUrl = $derived(`/api/v1/assets/${encodeURIComponent(item.id)}/thumbnail?size=tile`);

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
		onselect(item.id, !selected);
	}

	function handleTileClick(event: MouseEvent): void {
		if (selectable) {
			event.preventDefault();
			toggle();
		}
	}
</script>

<!-- One square cell of the wall grid: the anchor fills it with the
     stacked placeholder and image; the checkbox is a sibling, not
     nested interactive content. The group class feeds the hover and
     focus-within reveal of the checkbox; the content-visibility pair
     is the batched-render windowing of v0 (tiles offscreen skip
     their layout and paint, the estimated size keeps the scrollbar
     honest). -->
<div
	class="tile group relative aspect-square overflow-hidden rounded-control bg-accent-soft [content-visibility:auto] [contain-intrinsic-size:auto_240px] {selected ? '[outline:3px_solid_var(--color-accent)] outline-offset-[-3px]' : ''}"
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
			<canvas class="tile__placeholder absolute inset-0 w-full h-full object-cover" width="32" height="32" bind:this={canvas} aria-hidden="true"></canvas>
		{:else}
			<div
				class="tile__placeholder tile__placeholder--empty absolute inset-0 w-full h-full bg-[linear-gradient(135deg,color-mix(in_srgb,var(--color-accent-soft)_60%,var(--color-card)),var(--color-accent-soft))]"
				aria-hidden="true"
			></div>
		{/if}

		{#if item.isReady}
			<img
				class="tile__image absolute inset-0 w-full h-full object-cover transition-[opacity] duration-(--motion-base) ease-standard"
				class:opacity-0={!loaded}
				class:tile__image--loaded={loaded}
				src={thumbnailUrl}
				alt=""
				loading="lazy"
				decoding="async"
				onload={() => (loaded = true)}
				onerror={() => (loaded = false)}
				draggable="false"
			/>
		{/if}

		{#if item.isFavorite}
			<span
				class="tile__badge tile__badge--favorite absolute top-1.5 right-1.5 z-[2] inline-flex items-center justify-center w-6 h-6 rounded-full text-star bg-[rgb(20_26_46/0.55)] backdrop-blur-[2px]"
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
				class="tile__badge tile__badge--live absolute top-1.5 left-1.5 z-[2] inline-flex items-center justify-center w-6 h-6 rounded-full text-accent-on bg-[rgb(20_26_46/0.55)] backdrop-blur-[2px]"
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
			<span
				class="tile__badge tile__badge--video absolute bottom-1.5 right-1.5 z-[2] inline-flex items-center justify-center w-6 h-6 rounded-full text-accent-on bg-[rgb(20_26_46/0.55)] backdrop-blur-[2px]"
				title={$_('asset_video_label')}
				aria-hidden="true"
			>
				<svg viewBox="0 0 24 24" width="14" height="14" fill="none" aria-hidden="true">
					<path d="M9 8.2v7.6l6-3.8z" fill="currentColor" />
					<circle cx="12" cy="12" r="8.25" stroke="currentColor" stroke-width="1.75" />
				</svg>
			</span>
		{/if}
	</a>

	<!-- The selection checkbox: revealed on hover (the group on the
	     tile, for pointer users) and whenever selection is active
	     (touch users, mid-selection). -->
	<button
		class="tile__check absolute top-1.5 left-1.5 z-[3] inline-flex items-center justify-center w-[26px] h-[26px] p-0 border-2 border-accent-on rounded-full text-transparent bg-[rgb(20_26_46/0.55)] backdrop-blur-[2px] cursor-pointer transition-[opacity] duration-(--motion-fast) ease-standard group-hover:opacity-100 group-focus-within:opacity-100 aria-checked:bg-accent aria-checked:border-accent aria-checked:text-accent-on"
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
</div>
