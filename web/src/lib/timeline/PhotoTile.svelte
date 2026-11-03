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

<div class="tile" class:tile--selected={selected} class:tile--selectable={selectable}>
	<a
		class="tile__link"
		href="/a/{encodeURIComponent(item.id)}"
		onclick={handleTileClick}
		aria-label={item.originalFilename}
	>
		{#if item.blurhash !== null}
			<canvas class="tile__placeholder" width="32" height="32" bind:this={canvas} aria-hidden="true"></canvas>
		{:else}
			<div class="tile__placeholder tile__placeholder--empty" aria-hidden="true"></div>
		{/if}

		{#if item.isReady}
			<img
				class="tile__image"
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
			<span class="tile__badge tile__badge--favorite" title={$_('asset_favorite_label')} aria-hidden="true">
				<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor" aria-hidden="true">
					<path d="M12 3.6l2.5 5.2 5.7.7-4.2 4 1.1 5.6-5.1-2.8-5.1 2.8 1.1-5.6-4.2-4 5.7-.7z" />
				</svg>
			</span>
		{/if}

		{#if item.kind === 'image' && item.livePhotoVideoId !== null}
			<span class="tile__badge tile__badge--live" title={$_('asset_live_photo_label')} aria-hidden="true">
				<svg viewBox="0 0 24 24" width="14" height="14" fill="none" aria-hidden="true">
					<circle cx="12" cy="12" r="4.5" stroke="currentColor" stroke-width="1.75" />
					<path d="M2.5 12h3M18.5 12h3M12 2.5v3M12 18.5v3" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
				</svg>
			</span>
		{/if}

		{#if item.kind === 'video'}
			<span class="tile__badge tile__badge--video" title={$_('asset_video_label')} aria-hidden="true">
				<svg viewBox="0 0 24 24" width="14" height="14" fill="none" aria-hidden="true">
					<path d="M9 8.2v7.6l6-3.8z" fill="currentColor" />
					<circle cx="12" cy="12" r="8.25" stroke="currentColor" stroke-width="1.75" />
				</svg>
			</span>
		{/if}
	</a>

	<button
		class="tile__check"
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

<style>
	/* One square cell of the wall grid: the anchor fills it with the
	 * stacked placeholder and image; the checkbox is a sibling, not
	 * nested interactive content. */
	.tile {
		position: relative;
		aspect-ratio: 1;
		overflow: hidden;
		border-radius: var(--radius-control);
		background-color: var(--color-accent-soft);
	}

	/* The batched-render windowing of v0: tiles offscreen skip their
	 * layout and paint (the estimated size keeps the scrollbar
	 * honest); full virtualization stays an M3 optimization. */
	.tile {
		content-visibility: auto;
		contain-intrinsic-size: auto 240px;
	}

	.tile__link {
		position: absolute;
		inset: 0;
	}

	.tile__link:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: -2px;
	}

	.tile__placeholder,
	.tile__image {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
	}

	/* The decoded blurhash stretches smoothly over the tile; an empty
	 * hash keeps a quiet diagonal fill as the skeleton. */
	.tile__placeholder {
		object-fit: cover;
	}

	.tile__placeholder--empty {
		background: linear-gradient(
			135deg,
			color-mix(in srgb, var(--color-accent-soft) 60%, var(--color-card)),
			var(--color-accent-soft)
		);
	}

	.tile__image {
		opacity: 0;
		object-fit: cover;
		transition: opacity var(--motion-base) var(--ease-standard);
	}

	.tile__image--loaded {
		opacity: 1;
	}

	/* Badges sit on the scrim corners of the tile. */
	.tile__badge {
		position: absolute;
		z-index: 2;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		color: var(--color-accent-on);
		background-color: rgb(20 26 46 / 0.55);
		backdrop-filter: blur(2px);
	}

	.tile__badge--favorite {
		top: 6px;
		right: 6px;
		color: var(--color-star);
	}

	.tile__badge--live {
		top: 6px;
		left: 6px;
	}

	.tile__badge--video {
		bottom: 6px;
		right: 6px;
	}

	/* The selection checkbox: revealed on hover (pointer users) and
	 * whenever selection is active (touch users, mid-selection). */
	.tile__check {
		position: absolute;
		top: 6px;
		left: 6px;
		z-index: 3;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 26px;
		height: 26px;
		padding: 0;
		border: 2px solid var(--color-accent-on);
		border-radius: 50%;
		color: transparent;
		background-color: rgb(20 26 46 / 0.55);
		backdrop-filter: blur(2px);
		cursor: pointer;
		opacity: 0;
		transition: opacity var(--motion-fast) var(--ease-standard);
	}

	.tile:hover .tile__check,
	.tile:focus-within .tile__check,
	.tile__check--shown {
		opacity: 1;
	}

	.tile__check[aria-checked='true'] {
		color: var(--color-accent-on);
		background-color: var(--color-accent);
		border-color: var(--color-accent);
	}

	/* The selected tile itself carries the accent ring. */
	.tile--selected {
		outline: 3px solid var(--color-accent);
		outline-offset: -3px;
	}

	/* While selection is active the pointer shows the toggle intent. */
	.tile--selectable {
		cursor: pointer;
	}
</style>
