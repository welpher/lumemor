<script lang="ts">
	// The filter chips row of the search page (M3d Task 4, the
	// specimen's chips): one pill per facet entry of the seven
	// dimensions, each carrying the facet's honest count (the
	// rest-of-filters visible number the server computed) and its
	// selected state. A chip is a TOGGLE: pressing it flips its
	// dimension's filter and the search re-executes -- the page owns
	// the filter state (URL-synced); this row owns the rendering.
	//
	// THE PILL LANGUAGE (the specimen's .chip): a rounded-full
	// outline in the soft text tone, the dimension-prefixed label
	// and the tabular-nums count inside; hover takes the accent
	// soft; the SELECTED chip fills with the accent and its count
	// dims to the on-accent translucency. aria-pressed carries the
	// toggle state for assistive tech.
	import { _ } from 'svelte-i18n';

	/** The seven facet dimensions (the URL's filter arms). */
	export type FacetDimension =
		| 'people'
		| 'places'
		| 'years'
		| 'cameras'
		| 'kinds'
		| 'albums'
		| 'tags';

	/** One choosable chip: the dimension, the round-trip filter
	 * value (its string form), the composed display text and the
	 * facet's count. */
	export type FacetChip = {
		dimension: FacetDimension;
		value: string;
		/** The composed label: "dimension · value" (the page i18n's it). */
		text: string;
		count: number;
		selected: boolean;
	};

	type Props = {
		/** The chips of the current facets, the page's order. */
		chips: readonly FacetChip[];
		/** Toggle one chip's filter (the page owns the state). */
		ontoggle: (chip: FacetChip) => void;
	};

	let { chips, ontoggle }: Props = $props();
</script>

<div class="search-chips flex flex-wrap gap-2 max-w-[860px] mt-3.5" role="group" aria-label={$_('search_filters_label')}>
	{#each chips as chip (chip.dimension + ':' + chip.value)}
		<button
			class="search-chip inline-flex items-center gap-1.5 px-3 py-[5px] font-body font-semibold text-[13px] leading-[1.4] bg-transparent border rounded-full cursor-pointer whitespace-nowrap transition-[background-color,border-color,color] duration-(--motion-fast) ease-standard {chip.selected
				? 'border-accent bg-accent text-accent-on'
				: 'border-border-strong text-text-soft hover:bg-accent-soft hover:text-accent-soft-text'}"
			type="button"
			aria-pressed={chip.selected}
			onclick={() => ontoggle(chip)}
		>
			<span class="search-chip__label">{chip.text}</span>
			<span class="search-chip__count text-[11px] tabular-nums {chip.selected ? 'text-accent-on/80' : 'text-text-faint'}">{chip.count}</span>
		</button>
	{/each}
</div>
