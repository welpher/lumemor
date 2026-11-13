<script lang="ts">
	// The catch-all: every address the app has no route for lands
	// here, inside the content shell -- the top bar stays, the
	// content area is the unreachable card. It renders the same face
	// an unknown, foreign or withdrawn resource does; nothing on the
	// page reads the path, so the address can never narrow the answer.
	import UnreachableCard from '$lib/components/UnreachableCard.svelte';
</script>

<svelte:head>
	<title>lumemor</title>
</svelte:head>

<div class="unreachable-page px-4">
	<UnreachableCard />
</div>
