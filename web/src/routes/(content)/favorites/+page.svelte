<script lang="ts">
	// The favorites page: the caller's favorites as a
	// curated wall -- the same shape, ordering and pagination as the
	// timeline, every tile a favorite by construction, so the bar's
	// single action is the unfavorite toggle that ends a row's
	// membership in the view.
	import CuratedWall from '$lib/timeline/CuratedWall.svelte';
</script>

<svelte:head>
	<title>Favorites · lumemor</title>
</svelte:head>

<CuratedWall
	path="/api/v1/assets/favorites"
	view="favorites"
	titleKey="nav_favorites"
	loadingKey="favorites_loading"
	emptyKey="favorites_empty"
	actions={['favorite']}
/>
