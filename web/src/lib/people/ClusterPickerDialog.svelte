<script lang="ts">
	// The cluster picker dialog of the cluster detail: one Modal over
	// one grouped select, the composition the detail's two target
	// choices share -- the merge-into target (the caller's named
	// people) and the single-face move target (named people and the
	// unnamed region's first page). The page owns the option groups
	// (it knows which lists to fetch and which ids to exclude -- its
	// own cluster is never its own target) and the write; the dialog
	// owns the choice's lifecycle (reseeded on open, defaulted to the
	// first option). The optional new-person mode (M3b Task 5, the
	// viewer's assign flow) adds the second choice beside the picker:
	// typing a name instead of picking a cluster -- offered only by
	// callers that pass the `newPerson` shape, with the same local
	// 1-64 rune bound check the naming dialog applies.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Input from '$lib/components/Input.svelte';
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

	/** The optional new-person mode: naming instead of picking. */
	type NewPersonChoice = {
		/** The i18n key of the new choice's radio label. */
		labelKey: string;
		/** The i18n key of the existing choice's radio label. */
		existingLabelKey: string;
		/** The i18n key of the note under the name field. */
		noteKey: string;
		/** The i18n key of the submit label in the new mode. */
		submitKey: string;
		/** Run the write with the trimmed draft name. */
		onsubmit: (name: string) => void;
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
		/** The optional new-person mode (the assign flow's naming path). */
		newPerson?: NewPersonChoice;
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
		newPerson,
		loading = false,
		busy = false,
		errorKey = '',
		onclose,
		onsubmit
	}: Props = $props();

	let targetId = $state('');
	/** 'existing' (the picker) or 'new' (the naming path). */
	let mode = $state('existing');
	let draft = $state('');
	/** Whether a submit attempt already ran for the current draft. */
	let checked = $state(false);

	const optionIds = $derived(groups.flatMap((group) => group.options.map((option) => option.id)));

	$effect(() => {
		if (open) {
			targetId = '';
			mode = 'existing';
			draft = '';
			checked = false;
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

	/** The contract's bound, rune-counted: the trimmed name spans 1-64 characters. */
	function validDraft(): boolean {
		const runes = [...draft.trim()].length;
		return runes >= 1 && runes <= 64;
	}

	const boundViolated = $derived(checked && mode === 'new' && !validDraft());

	/** Close through the same discipline as the modal's own paths: the
	 * open flag flips AND onclose fires, so a page whose open state
	 * derives from a value it owns (the assign dialog's face id) never
	 * keeps a stale one behind a locally-closed dialog. */
	function close(): void {
		open = false;
		onclose?.();
	}

	function submit(): void {
		if (mode === 'new' && newPerson !== undefined) {
			checked = true;
			if (!validDraft()) {
				return;
			}
			newPerson.onsubmit(draft.trim());
			return;
		}
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
		{:else}
			{#if newPerson !== undefined}
				<!-- The two paths of the assign flow: picking an existing
				     cluster or typing a name for a new person. -->
				<fieldset class="cluster-picker__modes flex gap-4 m-0 p-0 border-none">
					<legend class="sr-only">{$_(titleKey)}</legend>
					<label class="cluster-picker__mode flex items-center gap-1.5 font-body font-normal text-[14px] leading-[1.4] text-text cursor-pointer">
						<input class="accent-accent" type="radio" name="cluster-picker-mode" value="existing" bind:group={mode} />
						{$_(newPerson.existingLabelKey)}
					</label>
					<label class="cluster-picker__mode flex items-center gap-1.5 font-body font-normal text-[14px] leading-[1.4] text-text cursor-pointer">
						<input class="accent-accent" type="radio" name="cluster-picker-mode" value="new" bind:group={mode} />
						{$_(newPerson.labelKey)}
					</label>
				</fieldset>
			{/if}
			{#if mode === 'new' && newPerson !== undefined}
				<Input label={$_('people_name_field_label')} name="new-person-name" bind:value={draft} maxlength={128} autocomplete="off" />
				<!-- The family-semantics note: naming labels every face of
				     the group, not just the picked one. -->
				<p class="cluster-picker__note m-0 font-body font-normal text-[13px] leading-[1.5] text-text-soft">{$_(newPerson.noteKey)}</p>
				{#if boundViolated}
					<p class="cluster-picker__bound m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text">{$_('people_name_invalid')}</p>
				{/if}
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
		{/if}
		{#if errorKey !== ''}
			<p class="cluster-picker__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, errorKey)}</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={close}>{$_('common_cancel')}</Button>
		{#if mode === 'new' && newPerson !== undefined}
			<Button disabled={busy} onclick={submit}>{$_(newPerson.submitKey)}</Button>
		{:else}
			<Button disabled={busy || optionIds.length === 0} onclick={submit}>{$_(submitKey)}</Button>
		{/if}
	{/snippet}
</Modal>
