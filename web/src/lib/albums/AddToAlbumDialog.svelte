<script lang="ts">
	// The add-to-album dialog of the timeline's batch bar (web IA
	// §5): pick one of the caller's own virtual albums, or name a new
	// one (created first, then fed the selection) -- the two modes
	// the cluster picker's assign flow pattern shares. The page owns
	// the writes (the POST items, the POST create + items chain) and
	// the album list; the dialog owns the choice's lifecycle
	// (reseeded on open, defaulted to the first option or the route's
	// ?album= preselect). After a successful add the dialog swaps to
	// the honest {added, skipped} report -- the write's own answer,
	// never an invented count.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import type { AlbumSummary } from '$lib/api/albums';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The caller's own virtual albums (the only addable-to kind). */
		albums: readonly AlbumSummary[];
		/** The route's ?album= preselect (the detail's add-photos handoff). */
		initialAlbumId?: string;
		/** The page is still fetching the album list. */
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
		/** Add the selection to the picked existing album. */
		onpick: (albumId: string) => void;
		/** Create a new album with the drafted title, then add to it. */
		oncreate: (title: string) => void;
		/** Acknowledge the report: close and clear the selection. */
		ondone: () => void;
	};

	let {
		open = $bindable(false),
		albums,
		initialAlbumId = '',
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

	const optionIds = $derived(albums.map((album) => album.id));

	$effect(() => {
		if (open) {
			mode = 'existing';
			targetId = '';
			draft = '';
			checked = false;
		}
	});

	// Re-pick the default target when the option set changes: the
	// dialog re-opened (targetId reset) or the page's list arrived --
	// the route's preselect wins over the first option.
	$effect(() => {
		void optionIds;
		if (!open) {
			return;
		}
		if (targetId === '' || !optionIds.includes(targetId)) {
			targetId =
				(initialAlbumId !== '' && optionIds.includes(initialAlbumId)
					? initialAlbumId
					: (optionIds[0] ?? '')) ?? '';
		}
	});

	/** The contract's bound, rune-counted: the trimmed title spans 1-64 characters. */
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

<Modal bind:open title={$_('albums_add_title')} {onclose}>
	<div class="add-album flex flex-col gap-3">
		{#if report !== null}
			<!-- The honest report of the landed add: the write's own
			     {added, skipped} answer, with the done affordance. -->
			<p class="add-album__report m-0 font-body font-semibold text-[15px] leading-[1.5] text-text" role="status">{$_('albums_add_report', { values: { added: report.added, skipped: report.skipped } })}</p>
		{:else if loading}
			<p class="add-album__loading m-0 font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in" role="status">{$_('albums_add_pick_label')}…</p>
		{:else}
			<!-- The two paths: picking an existing album or naming a new one. -->
			<fieldset class="add-album__modes flex gap-4 m-0 p-0 border-none">
				<legend class="sr-only">{$_('albums_add_title')}</legend>
				<label class="add-album__mode flex items-center gap-1.5 font-body font-normal text-[14px] leading-[1.4] text-text cursor-pointer">
					<input class="accent-accent" type="radio" name="add-album-mode" value="existing" bind:group={mode} />
					{$_('albums_add_mode_existing')}
				</label>
				<label class="add-album__mode flex items-center gap-1.5 font-body font-normal text-[14px] leading-[1.4] text-text cursor-pointer">
					<input class="accent-accent" type="radio" name="add-album-mode" value="new" bind:group={mode} />
					{$_('albums_add_mode_new')}
				</label>
			</fieldset>
			{#if mode === 'new'}
				<Input label={$_('albums_title_field')} name="add-album-title" bind:value={draft} maxlength={128} autocomplete="off" />
				{#if boundViolated}
					<p class="add-album__bound m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text">{$_('albums_title_invalid')}</p>
				{/if}
			{:else if optionIds.length === 0}
				<p class="add-album__empty m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('albums_add_empty')}</p>
			{:else}
				<label class="add-album__label font-body font-semibold text-[14px] leading-[1.4] text-text" for="add-album-target">{$_('albums_add_pick_label')}</label>
				<select
					class="add-album__target w-full px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
					id="add-album-target"
					bind:value={targetId}
				>
					{#each albums as album (album.id)}
						<option value={album.id}>{album.title}</option>
					{/each}
				</select>
			{/if}
		{/if}
		{#if errorKey !== ''}
			<p class="add-album__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, errorKey)}</p>
		{/if}
	</div>
	{#snippet footer()}
		{#if report !== null}
			<Button onclick={ondone}>{$_('albums_add_done')}</Button>
		{:else}
			<Button variant="ghost" onclick={close}>{$_('common_cancel')}</Button>
			{#if mode === 'new'}
				<Button disabled={busy} onclick={submit}>{$_('albums_add_create_submit')}</Button>
			{:else}
				<Button disabled={busy || loading || optionIds.length === 0} onclick={submit}>{$_('albums_add_submit')}</Button>
			{/if}
		{/if}
	{/snippet}
</Modal>
