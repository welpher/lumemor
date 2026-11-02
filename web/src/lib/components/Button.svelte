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
</script>

<button class="btn btn--{variant} {className}" {type} {...rest}>
	{@render children?.()}
</button>

<style>
	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 8px 16px;
		font: 600 15px/1.2 var(--font-body);
		border-radius: var(--radius-control);
		border: 1px solid transparent;
		cursor: pointer;
		transition:
			background-color var(--motion-fast) var(--ease-standard),
			border-color var(--motion-fast) var(--ease-standard),
			color var(--motion-fast) var(--ease-standard);
	}

	.btn:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}

	/* Primary: the accent fill (Blue 700 + white in light, Blue 300 +
	 * Navy 900 in dark, per the brand contrast rules). */
	.btn--primary {
		background-color: var(--color-accent);
		color: var(--color-accent-on);
	}

	.btn--primary:hover:not(:disabled) {
		background-color: var(--color-accent-strong);
	}

	/* Ghost: quiet outline for secondary actions. */
	.btn--ghost {
		background-color: transparent;
		color: var(--color-text);
		border-color: var(--color-border-strong);
	}

	.btn--ghost:hover:not(:disabled) {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}

	/* Danger: destructive actions, marked by the Ember border. The
	 * label keeps the regular text color for AA contrast; Ember text
	 * on light backgrounds stays below 4.5:1. */
	.btn--danger {
		background-color: transparent;
		color: var(--color-text);
		border-color: var(--color-danger);
	}

	.btn--danger:hover:not(:disabled) {
		background-color: color-mix(in srgb, var(--color-danger) 12%, transparent);
	}
</style>
