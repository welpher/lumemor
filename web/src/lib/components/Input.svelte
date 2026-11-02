<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	type Props = HTMLInputAttributes & {
		/** Visible label; always tied to the input via id/for. */
		label: string;
		/** Validation error shown under the field. */
		error?: string;
	};

	let {
		label,
		error = '',
		value = $bindable(''),
		id = `field-${Math.random().toString(36).slice(2, 9)}`,
		class: className = '',
		...rest
	}: Props = $props();
</script>

<div class="field" class:field--error={error !== ''}>
	<label class="field__label" for={id}>{label}</label>
	<input
		class="field__input {className}"
		{id}
		bind:value
		aria-invalid={error !== '' ? 'true' : undefined}
		aria-describedby={error !== '' ? `${id}-error` : undefined}
		{...rest}
	/>
	{#if error !== ''}
		<p class="field__error" id="{id}-error">{error}</p>
	{/if}
</div>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.field__label {
		font: 600 14px/1.4 var(--font-body);
		color: var(--color-text);
	}

	.field__input {
		padding: 9px 12px;
		font: 400 15px/1.4 var(--font-body);
		color: var(--color-text);
		background-color: var(--color-card);
		border: 1px solid var(--color-border-strong);
		border-radius: var(--radius-control);
		transition:
			border-color var(--motion-fast) var(--ease-standard),
			box-shadow var(--motion-fast) var(--ease-standard);
	}

	.field__input:focus {
		border-color: var(--color-accent);
	}

	.field--error .field__input {
		border-color: var(--color-danger);
	}

	.field__error {
		margin: 0;
		font: 500 13px/1.4 var(--font-body);
		color: var(--color-danger-text);
	}
</style>
