<script lang="ts" module>
	// Button variants: `primary` is the Blue 700 fill with white text
	// (the one accent of the interface), `ghost` is the outlined
	// quiet button, `danger` marks destructive actions through the
	// Ember border while keeping AA-safe text. Destructive confirms
	// may later add a filled form; the shell stays minimal here.
	export type ButtonVariant = 'primary' | 'ghost' | 'danger';
</script>

<script lang="ts">
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type Props = HTMLButtonAttributes & {
		/** Visual variant; see the module-level comment. */
		variant?: ButtonVariant;
	};

	let { variant = 'primary', type = 'button', class: className = '', children, ...rest }: Props = $props();

	// The visual half of each variant; the btn--<variant> class name
	// stays as the identity/state hook. Hovers sit behind
	// :not(:disabled) exactly like the previous :hover:not(:disabled)
	// rules, and the border color is variant-owned so no two plain
	// border-color utilities ever land on the same element.
	const variantUtilities: Record<ButtonVariant, string> = {
		primary: 'bg-accent text-accent-on border-transparent hover:not-disabled:bg-accent-strong',
		ghost: 'bg-transparent text-text border-border-strong hover:not-disabled:bg-accent-soft hover:not-disabled:text-accent-soft-text',
		danger: 'bg-transparent text-text border-danger hover:not-disabled:bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)]'
	};
</script>

<button
	class="btn btn--{variant} {className} inline-flex items-center justify-center gap-2 px-4 py-2 font-body font-semibold text-[15px] leading-[1.2] rounded-control border cursor-pointer transition-[background-color,border-color,color] duration-(--motion-fast) ease-standard disabled:opacity-55 disabled:cursor-not-allowed {variantUtilities[
		variant
	]}"
	{type}
	{...rest}
>
	{@render children?.()}
</button>
