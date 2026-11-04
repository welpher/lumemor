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

<div class="field flex flex-col gap-1.5" class:field--error={error !== ''}>
	<label class="field__label font-body font-semibold text-[14px] leading-[1.4] text-text" for={id}>{label}</label>
	<input
		class="field__input {className} px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border rounded-control transition-[border-color,box-shadow] duration-(--motion-fast) ease-standard {error === '' ? 'border-border-strong focus:border-accent' : 'border-danger'}"
		{id}
		bind:value
		aria-invalid={error !== '' ? 'true' : undefined}
		aria-describedby={error !== '' ? `${id}-error` : undefined}
		{...rest}
	/>
	{#if error !== ''}
		<p class="field__error font-body font-medium text-[13px] leading-[1.4] text-danger-text" id="{id}-error">{error}</p>
	{/if}
</div>
