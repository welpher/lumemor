<script lang="ts">
	// The icon-only action button of the paper surfaces: the square
	// a11y-safe affordance the icon-first rule sweeps
	// the toolbars, row actions and overflow menus onto. The caller
	// passes the lucide glyph as the child and the accessible name
	// as the aria-label -- the glyph itself is decorative, the label
	// is the action's own i18n string. The visual language is the
	// shell topbar's hamburger (the quiet transparent square, the
	// accent-soft hover), not the viewer's night chrome: this button
	// lives on card and page surfaces.
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type Props = HTMLButtonAttributes & {
		/**
		 * Visual variant: `ghost` is the quiet square, `danger` marks
		 * destructive row actions through the Ember outline (the
		 * Button danger variant's shape).
		 */
		variant?: 'ghost' | 'danger';
	};

	let { variant = 'ghost', type = 'button', class: className = '', children, ...rest }: Props = $props();

	const variantUtilities: Record<'ghost' | 'danger', string> = {
		ghost: 'bg-transparent text-text border-transparent hover:not-disabled:bg-accent-soft hover:not-disabled:text-accent-soft-text',
		danger: 'bg-transparent text-danger-text border-danger hover:not-disabled:bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)]'
	};
</script>

<button
	class="icon-btn icon-btn--{variant} {className} inline-flex items-center justify-center w-9 h-9 p-0 rounded-control border cursor-pointer transition-[background-color,border-color,color] duration-(--motion-fast) ease-standard disabled:opacity-55 disabled:cursor-not-allowed {variantUtilities[
		variant
	]}"
	{type}
	{...rest}
>
	{@render children?.()}
</button>
