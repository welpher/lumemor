<script lang="ts">
	// The loading line: ONE centered line of text in the content area
	// with role="status" -- no skeleton, no spinner. It stands where
	// the page body will land (the bars stay above it); the
	// anti-flicker delay is the stylesheet's, never the page's.
	type Props = {
		/** The line's own copy (the surface's loading message). */
		label: string;
	};

	let { label }: Props = $props();
</script>

<p
	class="state-loading my-12 text-center font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in"
	role="status"
>
	{label}
</p>
