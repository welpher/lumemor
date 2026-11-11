<script lang="ts">
	// The hidden view page: the caller's hidden assets
	// as a curated wall -- the owner-level visibility switch turned
	// off, live rows only -- with the unhide action that returns a
	// row to the timeline and ends its membership in this view. The
	// wall-selection entry joins the bar: a hidden row is still in
	// the caller's sight here, so adding it to a memory is exactly
	// the entry's ground.
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
	<title>Hidden · lumemor</title>
</svelte:head>

<CuratedWall
	path="/api/v1/assets/hidden"
	view="hidden"
	titleKey="hidden_title"
	loadingKey="hidden_loading"
	emptyKey="hidden_empty"
	actions={['unhide']}
	onaddmemory={openMemoryDialog}
/>

<AddToMemoryDialog bind:open={memoryOpen} ids={memoryIds} onpass={finishMemory} />
