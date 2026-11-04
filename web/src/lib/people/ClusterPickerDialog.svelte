<script lang="ts">
	// The cluster picker dialog of the cluster detail: one Modal over
	// one grouped select, the composition the detail's two target
	// choices share -- the merge-into target (the caller's named
	// people) and the single-face move target (named people and the
	// unnamed region's first page). The page owns the option groups
	// (it knows which lists to fetch and which ids to exclude -- its
	// own cluster is never its own target) and the write; the dialog
	// owns the choice's lifecycle (reseeded on open, defaulted to the
	// first option).
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';

	/** One choosable cluster of a group. */
	type PickerOption = {
		id: string;
		label: string;
	};

	/** One labeled group of options (an <optgroup>). */
	type PickerGroup = {
		label: string;
		options: PickerOption[];
	};

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The i18n key of the dialog title. */
		titleKey: string;
		/** The i18n key of the dialog's explanatory line, when one applies. */
		bodyKey?: string;
		/** The i18n key of the select's label. */
		targetLabelKey: string;
		/** The i18n key of the submit label. */
		submitKey: string;
		/** The i18n key shown when every group is empty. */
		emptyKey: string;
		/** The option groups, in selection order. */
		groups: readonly PickerGroup[];
		/** The page is still fetching the options. */
		loading?: boolean;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/**
		 * Notified of every close path (Escape, backdrop, the close mark)
		 * -- for callers whose open state derives from a value they own
		 * (the move dialog's face id) instead of a bound flag.
		 */
		onclose?: () => void;
		/** Run the write with the chosen target id. */
		onsubmit: (targetId: string) => void;
	};

	let {
		open = $bindable(false),
		titleKey,
		bodyKey = '',
		targetLabelKey,
		submitKey,
		emptyKey,
		groups,
		loading = false,
		busy = false,
		errorKey = '',
		onclose,
		onsubmit
	}: Props = $props();

	let targetId = $state('');

	const optionIds = $derived(groups.flatMap((group) => group.options.map((option) => option.id)));

	$effect(() => {
		if (open) {
			targetId = '';
		}
	});

	// Re-pick the default target when the option set changes: the
	// dialog re-opened (targetId reset) or the page's lists arrived.
	$effect(() => {
		void optionIds;
		if (!open) {
			return;
		}
		if (!optionIds.includes(targetId)) {
			targetId = optionIds[0] ?? '';
		}
	});

	function submit(): void {
		if (targetId === '') {
			return;
		}
		onsubmit(targetId);
	}
</script>

<Modal bind:open title={$_(titleKey)} {onclose}>
	<div class="cluster-picker flex flex-col gap-3">
		{#if bodyKey !== ''}
			<p class="cluster-picker__body m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_(bodyKey)}</p>
		{/if}
		{#if loading}
			<p class="cluster-picker__loading m-0 font-body font-normal text-[14px] leading-[1.4] text-text-soft" role="status">{$_('people_loading')}</p>
		{:else if optionIds.length === 0}
			<p class="cluster-picker__empty m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_(emptyKey)}</p>
		{:else}
			<label class="cluster-picker__label font-body font-semibold text-[14px] leading-[1.4] text-text" for="cluster-picker-target">{$_(targetLabelKey)}</label>
			<select
				class="cluster-picker__target w-full px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
				id="cluster-picker-target"
				bind:value={targetId}
			>
				{#each groups as group (group.label)}
					{#if group.options.length > 0}
						<optgroup label={group.label}>
							{#each group.options as option (option.id)}
								<option value={option.id}>{option.label}</option>
							{/each}
						</optgroup>
					{/if}
				{/each}
			</select>
		{/if}
		{#if errorKey !== ''}
			<p class="cluster-picker__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, errorKey)}</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>{$_('common_cancel')}</Button>
		<Button disabled={busy || optionIds.length === 0} onclick={submit}>{$_(submitKey)}</Button>
	{/snippet}
</Modal>
