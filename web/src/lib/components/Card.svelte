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
<div class="card {padded ? 'card--padded' : ''} {className}" {...rest}>
	{@render children?.()}
</div>

<style>
	.card {
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
	}

	.card--padded {
		padding: 20px;
	}
</style>
