<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		/** The row's title: the rail's active section, echoed (the caller resolves the i18n). */
		title: string;
		/**
		 * The page's contextual actions (the users page's create
		 * button rides this slot). Settings, models and jobs carry
		 * none -- the row still stands, for the grammar's uniformity.
		 */
		actions?: Snippet;
	};

	let { title, actions }: Props = $props();
</script>

<!-- The admin content column's page-header row: the title on the
     left, the page's contextual actions on the right, and the
     hairline below -- `border-b` with the 12px bottom padding and
     the 16px margin this row pins. The tool shell is the line
     domain: this row's hairline meets the rail's border-r in a T,
     pinning the header into a belt; the content shell's
     no-dividing-line rule holds the photo faces, not this one. -->
<header class="page-header flex items-center justify-between gap-4 pb-3 mb-4 border-b border-border">
	<h1 class="page-header__title m-0 font-display font-bold text-[20px] text-text">{title}</h1>
	{#if actions}
		<div class="page-header__actions flex gap-2">
			{@render actions()}
		</div>
	{/if}
</header>
