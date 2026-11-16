<script lang="ts">
	// The archived view page: the caller's archived assets
	// as a curated wall -- the owner-level archive switch turned
	// on, untrashed rows only -- with the unarchive action that
	// returns a row to every family flow and ends its membership in
	// this view. The wall-selection entry joins the bar: an archived
	// row is still in the caller's sight here, so adding it to a
	// memory is exactly the entry's ground (the memory add's own
	// family-flow filter decides what lands).
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
	onaddmemory={openMemoryDialog}
/>

<AddToMemoryDialog bind:open={memoryOpen} ids={memoryIds} onpass={finishMemory} />
