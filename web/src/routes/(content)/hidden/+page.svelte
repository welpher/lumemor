<script lang="ts">
	// The hidden view page: the caller's hidden assets
	// as a curated wall -- the owner-level visibility switch turned
	// off, live rows only -- with the unhide action that returns a
	// row to the timeline and ends its membership in this view.
	import CuratedWall from '$lib/timeline/CuratedWall.svelte';
</script>

<svelte:head>
	<title>Hidden · lumemor</title>
</svelte:head>

<CuratedWall
	path="/api/v1/assets/hidden"
	view="hidden"
	titleKey="hidden_title"
	loadingKey="hidden_loading"
	emptyKey="hidden_empty"
	actions={['unhide']}
/>
