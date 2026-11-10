<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';

	type Props = HTMLAttributes<HTMLDivElement> & {
		/** Interior padding; cards default to comfortable density. */
		padded?: boolean;
	};

	let { padded = true, class: className = '', children, ...rest }: Props = $props();
</script>

<!-- Card: the base surface for grouped content (spec §5: card radii
     16-20, very light shadow, no heavy borders). -->
<div
	class="card {padded ? 'card--padded' : ''} {className} bg-card border border-border rounded-card shadow-card {padded
		? 'p-5'
		: ''}"
	{...rest}
>
	{@render children?.()}
</div>
