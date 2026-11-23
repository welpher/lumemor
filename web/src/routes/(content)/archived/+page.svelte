<script lang="ts">
	// The archived view page: the caller's archived assets
	// as a curated wall -- the owner-level archive switch turned
	// on, untrashed rows only -- with the unarchive action that
	// returns a row to every family flow and ends its membership in
	// this view. No add-to-memory entry: an archived row has left the
	// family flows (the memory add's own filter would reject it), so
	// the batch bar offers the unarchive action alone.
	import CuratedWall from '$lib/timeline/CuratedWall.svelte';
</script>

<svelte:head>
	<title>Archive · lumemor</title>
</svelte:head>

<CuratedWall
	path="/api/v1/assets/archived"
	view="archived"
	titleKey="nav_archived"
	countKey="archive_bar_count"
	loadingKey="archive_loading"
	emptyTitleKey="archive_empty_title"
	emptyBodyKey="archive_empty_body"
	actions={['unhide']}
/>
