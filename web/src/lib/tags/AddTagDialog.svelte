<script lang="ts">
	// The add-tag dialog (M3d Task 4): pick one of the caller's
	// existing tags, or name a new one (created first, then fed the
	// ids) -- the two paths the add-to-album dialog's pattern
	// shares. The page owns the writes and the tag list; the dialog
	// owns the choice's lifecycle (reseeded on open, defaulted to
	// the first option). After a successful add the dialog swaps to
	// the honest {added, skipped} report -- the write's own answer,
	// never an invented count.
	//
	// The dialog serves two surfaces: the timeline's batch bar (the
	// selection's ids) and the viewer's info drawer (one asset's
	// id); both pass the same props, the report reads the same.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import X from '@lucide/svelte/icons/x';
	import Check from '@lucide/svelte/icons/check';
	import Plus from '@lucide/svelte/icons/plus';
	import { resolveErrorText } from '$lib/i18n/errors';
	import type { TagSummary } from '$lib/api/tags';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The caller's tag vocabulary (the pick list). */
		tags: readonly TagSummary[];
		/** The page is still fetching the vocabulary. */
		loading?: boolean;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/**
		 * The honest report of the last successful add (null while none
		 * landed); the dialog shows it in place of the choice.
		 */
		report?: { added: number; skipped: number } | null;
		/** Notified of every close path (Escape, backdrop, the close mark). */
		onclose?: () => void;
		/** Add the picked existing tag. */
		onpick: (tagId: string) => void;
		/** Create a new tag with the drafted name, then add to it. */
		oncreate: (name: string) => void;
		/** Acknowledge the report: close (the caller clears its state). */
		ondone: () => void;
	};

	let {
		open = $bindable(false),
		tags,
		loading = false,
		busy = false,
		errorKey = '',
		report = null,
		onclose,
		onpick,
		oncreate,
		ondone
	}: Props = $props();

	/** 'existing' (the picker) or 'new' (the create-then-add path). */
	let mode = $state('existing');
	let targetId = $state('');
	let draft = $state('');
	/** Whether a submit attempt already ran for the current draft. */
	let checked = $state(false);

	const optionIds = $derived(tags.map((tag) => tag.id));

	$effect(() => {
		if (open) {
			mode = 'existing';
			targetId = '';
			draft = '';
			checked = false;
		}
	});

	// Re-pick the default target when the option set changes: the
	// dialog re-opened (targetId reset) or the page's list arrived.
	$effect(() => {
		void optionIds;
		if (!open) {
			return;
		}
		if (targetId === '' || !optionIds.includes(targetId)) {
			targetId = optionIds[0] ?? '';
		}
	});

	/** The contract's bound, rune-counted: the trimmed name spans 1-64 characters. */
	function validDraft(): boolean {
		const runes = [...draft.trim()].length;
		return runes >= 1 && runes <= 64;
	}

	const boundViolated = $derived(checked && mode === 'new' && !validDraft());

	/** Close through the same discipline as the modal's own paths. */
	function close(): void {
		open = false;
		onclose?.();
	}

	function submit(): void {
		if (mode === 'new') {
			checked = true;
			if (!validDraft()) {
				return;
			}
			oncreate(draft.trim());
			return;
		}
		if (targetId === '') {
			return;
		}
		onpick(targetId);
	}
</script>

<Modal bind:open title={$_('tags_add_title')} {onclose}>
	<div class="add-tag flex flex-col gap-3">
		{#if report !== null}
			<!-- The honest report of the landed add: the write's own
			     {added, skipped} answer, with the done affordance. -->
			<p class="add-tag__report m-0 font-body font-semibold text-[15px] leading-[1.5] text-text" role="status">{$_('tags_add_report', { values: { added: report.added, skipped: report.skipped } })}</p>
		{:else if loading}
			<p class="add-tag__loading m-0 font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in" role="status">{$_('tags_add_pick_label')}…</p>
		{:else}
			<!-- The two paths: picking an existing tag or naming a new one. -->
			<fieldset class="add-tag__modes flex gap-4 m-0 p-0 border-none">
				<legend class="sr-only">{$_('tags_add_title')}</legend>
				<label class="add-tag__mode flex items-center gap-1.5 font-body font-normal text-[14px] leading-[1.4] text-text cursor-pointer">
					<input class="accent-accent" type="radio" name="add-tag-mode" value="existing" bind:group={mode} />
					{$_('tags_add_mode_existing')}
				</label>
				<label class="add-tag__mode flex items-center gap-1.5 font-body font-normal text-[14px] leading-[1.4] text-text cursor-pointer">
					<input class="accent-accent" type="radio" name="add-tag-mode" value="new" bind:group={mode} />
					{$_('tags_add_mode_new')}
				</label>
			</fieldset>
			{#if mode === 'new'}
				<Input label={$_('tags_name_field')} name="add-tag-name" bind:value={draft} maxlength={128} autocomplete="off" />
				{#if boundViolated}
					<p class="add-tag__bound m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text">{$_('tags_name_invalid')}</p>
				{/if}
			{:else if optionIds.length === 0}
				<p class="add-tag__empty m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('tags_add_empty')}</p>
			{:else}
				<label class="add-tag__label font-body font-semibold text-[14px] leading-[1.4] text-text" for="add-tag-target">{$_('tags_add_pick_label')}</label>
				<select
					class="add-tag__target w-full px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
					id="add-tag-target"
					bind:value={targetId}
				>
					{#each tags as tag (tag.id)}
						<option value={tag.id}>{tag.name}</option>
					{/each}
				</select>
			{/if}
		{/if}
		{#if errorKey !== ''}
			<p class="add-tag__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, errorKey)}</p>
		{/if}
	</div>
	{#snippet footer()}
		{#if report !== null}
			<Button onclick={ondone}>
				<Check size={20} aria-hidden="true" />
				{$_('tags_add_done')}
			</Button>
		{:else}
			<Button variant="ghost" onclick={close}>
				<X size={20} aria-hidden="true" />
				{$_('common_cancel')}
			</Button>
			{#if mode === 'new'}
				<Button disabled={busy} onclick={submit}>
					<Plus size={20} aria-hidden="true" />
					{$_('tags_add_create_submit')}
				</Button>
			{:else}
				<Button disabled={busy || loading || optionIds.length === 0} onclick={submit}>
					<Check size={20} aria-hidden="true" />
					{$_('tags_add_submit')}
				</Button>
			{/if}
		{/if}
	{/snippet}
</Modal>
