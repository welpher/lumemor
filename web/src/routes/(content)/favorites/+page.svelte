<script lang="ts">
	// The favorites page: the caller's favorites as a
	// curated wall -- the same shape, ordering and pagination as the
	// timeline, every tile a favorite by construction, so the bar's
	// single action is the unfavorite toggle that ends a row's
	// membership in the view. The wall-selection entry joins the
	// bar: a favorite is in the caller's current sight, so adding it
	// to a memory is exactly the entry's ground.
	import CuratedWall from '$lib/timeline/CuratedWall.svelte';
	import AddToMemoryDialog from '$lib/memories/AddToMemoryDialog.svelte';

	// The add-to-memory flow's state: the captured-ids +
	// done-continuation pair the wall's opener hands over -- the
	// dialog owns the flow's own state and writes.
	let memoryIds = $state<readonly string[]>([]);
	let memoryDone: (() => void) | undefined;
	let memoryOpen = $state(false);

	function openMemoryDialog(ids: readonly string[], done: () => void): void {
		memoryIds = ids;
		memoryDone = done;
		memoryOpen = true;
	}

	function finishMemory(): void {
		memoryDone?.();
		memoryDone = undefined;
	}
</script>

<svelte:head>
	<title>Favorites · lumemor</title>
</svelte:head>

<CuratedWall
	path="/api/v1/assets/favorites"
	view="favorites"
	titleKey="nav_favorites"
	countKey="favorites_bar_count"
	loadingKey="favorites_loading"
	emptyKey="favorites_empty"
	actions={['favorite']}
	onaddmemory={openMemoryDialog}
/>

<AddToMemoryDialog bind:open={memoryOpen} ids={memoryIds} onpass={finishMemory} />
