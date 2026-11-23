<script lang="ts">
	// The map's place panel: ONE spot's photos beside the map. The
	// head names the selection (a city pin's place, or a bubble's
	// group -- its shared country, else the honest area label) and
	// counts its photos (never its cities, the map's own reading);
	// the body is the SHARED wall in its panel frame -- the same
	// month/day headers, the same justified rows, the same tiles,
	// with the panel's own scroll root and compact axis. There is no
	// selection state and no asset action here: the panel is for
	// looking, not curating.
	//
	// THE WALK: one keyset client over the endpoint's `placeIds`
	// narrow, the SSE convergence every wall shares, and the page's
	// next-batch hook the wall reports when its own root reaches the
	// bottom. A photo click is the tile's own viewer navigation; the
	// panel stays open.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import { createTimelineClient } from '$lib/api/timeline';
	import type { PlaceSummary } from '$lib/api/places';
	import { subscribeAssetEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { publishWallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import StateCard from '$lib/components/StateCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import { selectionCount, selectionName } from './places';
	import X from '@lucide/svelte/icons/x';

	type Props = {
		/** The selected places (the page's grouping rows for the marker). */
		places: readonly PlaceSummary[];
		/** Collapse the panel (the page returns the map to its full width). */
		onclose: () => void;
	};

	let { places, onclose }: Props = $props();

	const placeIds = untrack(() => places.map((place) => place.placeId));
	const wall = untrack(() => createTimelineClient({ placeIds }));

	const title = $derived(selectionName(places, $locale ?? 'en', $_('places_panel_area')));
	const count = $derived(selectionCount(places));

	const firstLoad = $derived($wall.items.length === 0 && $wall.status === 'loading');
	const settledEmpty = $derived($wall.items.length === 0 && $wall.status === 'ready' && !$wall.hasMore);

	/** The shared unknown-code fallback (see $lib/i18n/errors). */
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// Publish the loaded page items as the viewer's navigation
	// sequence: a photo opened from the panel walks exactly the
	// panel's loaded photos.
	$effect(() => {
		publishWallItems($wall.items);
	});

	$effect(() => {
		void wall.loadMore();
		const unsubscribe = subscribeAssetEvents((event) => {
			void wall.applyEvent(event);
		});
		return unsubscribe;
	});
</script>

<aside
	class="places__panel flex h-1/2 min-h-0 w-full flex-none flex-col overflow-hidden rounded-card border border-border bg-card shadow-card sm:h-full sm:w-[30%] sm:min-w-[300px] sm:max-w-[600px]"
	aria-label={$_('places_panel_label')}
	data-testid="places-panel"
	data-place-panel
>
	<header class="places__panel-head flex flex-none items-center gap-2 px-3 py-2 border-b border-border">
		<h2
			class="places__panel-name m-0 min-w-0 flex-1 truncate font-display font-bold text-[16px] leading-[1.3] text-text"
			data-place-panel-name
		>
			{title}
		</h2>
		<CountLabel
			class="places__panel-count flex-none font-body font-normal text-[13px] leading-[1.4] text-text-soft"
			{count}
			label={$_('places_photo_count', { values: { count } })}
		/>
		<button
			class="places__panel-close flex-none inline-flex items-center justify-center w-8 h-8 rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
			type="button"
			aria-label={$_('common_close')}
			onclick={onclose}
		>
			<X size={16} aria-hidden="true" />
		</button>
	</header>

	<div class="places__panel-body relative min-h-0 flex-1" aria-busy={$wall.status === 'loading'}>
		{#if firstLoad}
			<div class="px-3 py-4">
				<LoadingLine label={$_('places_loading')} />
			</div>
		{:else if settledEmpty}
			<p
				class="places__panel-empty m-0 px-4 py-6 text-center font-body font-normal text-[13px] leading-[1.5] text-text-soft"
			>
				{$_('place_empty_body')}
			</p>
		{:else if $wall.status === 'error' && $wall.items.length === 0}
			<StateCard
				tone="error"
				body={errorText($wall.errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void wall.loadMore()}
			/>
		{:else}
			<!-- The shared wall, the panel frame: no runner, so no
			     selection state and no batch bar ever rise here. -->
			<Wall items={$wall.items} variant="panel" onreachbottom={() => void wall.loadMore()} />
		{/if}
	</div>
</aside>
