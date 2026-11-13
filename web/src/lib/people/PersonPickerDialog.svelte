<script lang="ts">
	// The person target picker: one Modal over one filterable
	// avatar-row single-select list -- the shape the two "pick a
	// person" dialogs of the people surface share (the merge face of
	// the person detail, the choose-person face of the viewer's
	// edit-people panel). The page owns the option list (it fetched
	// it and applied its own exclusions -- a face's or cluster's own
	// row is never its own target) and the write; the dialog owns
	// the choice's lifecycle: the LOCAL name filter (no request ever
	// rides the typing), the list's own FIXED height (264px -- one
	// filtered-out row or an empty match never resizes the dialog),
	// the selected row's edge-to-edge accent fill, and the in-box
	// centered empty state.
	//
	// The optional create-from-face row sits BELOW the scroll box
	// (outside it, dashed): the list can grow past every viewport
	// and the creation stays reachable.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import X from '@lucide/svelte/icons/x';
	import Check from '@lucide/svelte/icons/check';
	import Search from '@lucide/svelte/icons/search';
	import Image from '@lucide/svelte/icons/image';
	import Plus from '@lucide/svelte/icons/plus';
	import type { PersonSummary } from '$lib/api/people';
	import { resolveErrorText } from '$lib/i18n/errors';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The i18n key of the dialog title. */
		titleKey: string;
		/** The i18n key of the dialog's explanatory line, when one applies. */
		bodyKey?: string;
		/** The i18n key of the submit label. */
		submitKey: string;
		/** The i18n key of the optional visible heading over the list. */
		groupKey?: string;
		/** The pickable people, in the page's own order and exclusions. */
		people: readonly PersonSummary[];
		/**
		 * The create-from-face row below the list (the choose-person
		 * face offers it): the page stages or runs its own write.
		 */
		oncreate?: () => void;
		/** The page is still fetching the options. */
		loading?: boolean;
		/** The error-code i18n key of the failed list fetch ('' when none). */
		errorKey?: string;
		/**
		 * Notified of every close path (Escape, backdrop, the close
		 * mark) -- for callers whose open state derives from a value
		 * they own (the pick dialog's face id).
		 */
		onclose?: () => void;
		/** Run the write with the picked person's cluster id. */
		onsubmit: (clusterId: string) => void;
	};

	let {
		open = $bindable(false),
		titleKey,
		bodyKey = '',
		submitKey,
		groupKey = '',
		people,
		oncreate,
		loading = false,
		errorKey = '',
		onclose,
		onsubmit
	}: Props = $props();

	/** The local filter draft; it never leaves the dialog. */
	let filter = $state('');
	/** The picked person's cluster id ('' while nothing is pickable). */
	let selectedId = $state('');

	// The filter is LOCAL and case-insensitive on the trimmed term
	// (a substring hit on the person's name); it never fetches.
	const visible = $derived.by(() => {
		const term = filter.trim().toLowerCase();
		if (term === '') {
			return people;
		}
		return people.filter((entry) => entry.name.toLowerCase().includes(term));
	});

	$effect(() => {
		if (open) {
			filter = '';
			selectedId = '';
		}
	});

	// The selection tracks the visible rows: the reseed on open picks
	// the first, and a filter that hides the selected row falls back
	// to the first visible one (a submit can never carry a person the
	// dialog is not showing).
	$effect(() => {
		void visible;
		if (!open) {
			return;
		}
		if (selectedId !== '' && visible.some((entry) => entry.clusterId === selectedId)) {
			return;
		}
		selectedId = visible[0]?.clusterId ?? '';
	});

	/** Close through the same discipline as the modal's own paths. */
	function close(): void {
		open = false;
		onclose?.();
	}

	function submit(): void {
		if (selectedId === '') {
			return;
		}
		onsubmit(selectedId);
	}
</script>

<Modal bind:open title={$_(titleKey)} {onclose}>
	<div class="person-picker flex flex-col">
		{#if bodyKey !== ''}
			<p class="person-picker__body m-0 mb-3 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
				{$_(bodyKey)}
			</p>
		{/if}
		{#if loading}
			<p
				class="person-picker__loading m-0 font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in"
				role="status"
			>
				{$_('people_loading')}
			</p>
		{:else}
			<!-- The local filter: a leading decorative search glyph, a
			     search-type input, no request ever rides the typing. -->
			<label class="person-picker__filter-wrap relative block mb-3">
				<Search
					size={16}
					class="absolute start-3 top-1/2 -translate-y-1/2 text-text-faint pointer-events-none"
					aria-hidden="true"
				/>
				<input
					class="person-picker__filter w-full ps-9 pe-3 py-2 font-body font-normal text-[14px] leading-[1.4] text-text bg-page border border-border-strong rounded-control focus:border-accent"
					type="search"
					placeholder={$_('people_picker_filter')}
					aria-label={$_('people_picker_filter')}
					bind:value={filter}
				/>
			</label>
			{#if groupKey !== ''}
				<p class="person-picker__group m-0 mb-1.5 font-body font-semibold text-[14px] leading-[1.4] text-text">
					{$_(groupKey)}
				</p>
			{/if}
			<!-- The list's own scroll: a FIXED 264px box (filtering to one
			     row, or to none, never resizes the dialog) whose selected
			     row fills edge-to-edge (no scrollbar gutter reserved --
			     the hover-revealed bar never leaves a stripe beside the
			     accent fill). The empty state centers INSIDE the box. -->
			<ul
				class="person-picker__list m-0 p-0 list-none bg-page border border-border rounded-control overflow-y-auto h-[264px]"
				role="radiogroup"
				aria-label={$_(titleKey)}
			>
				{#if visible.length === 0}
					<li
						class="person-picker__empty h-full flex items-center justify-center px-3 text-center font-body font-normal text-[13px] leading-[1.4] text-text-faint"
						role="status"
					>
						{$_('people_picker_empty')}
					</li>
				{:else}
					{#each visible as person (person.clusterId)}
						<li data-person-id={person.clusterId}>
							<button
								class="person-picker__row group flex w-full items-center gap-2.5 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] text-text bg-transparent border-0 text-left cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text aria-checked:bg-accent-soft aria-checked:text-accent-soft-text"
								type="button"
								role="radio"
								aria-checked={selectedId === person.clusterId}
								data-person-id={person.clusterId}
								onclick={() => (selectedId = person.clusterId)}
							>
								<img
									class="flex-none w-8 h-8 rounded-full object-cover bg-accent-soft"
									src={person.thumbnail}
									alt=""
									loading="lazy"
									decoding="async"
									draggable="false"
								/>
								<span class="min-w-0 flex-1 truncate">{person.name}</span>
								<span
									class="flex-none font-normal text-[12px] leading-[1.4] text-text-faint tabular-nums inline-flex items-center gap-1"
									role="img"
									aria-label={$_('people_photo_count', { values: { count: person.photoCount } })}
									title={$_('people_photo_count', { values: { count: person.photoCount } })}
								>
									<Image size={14} class="shrink-0" aria-hidden="true" />
									{person.photoCount}
								</span>
								<!-- The picked row's trailing check, visible only on
								     the selected row (the group-aria-checked gate). -->
								<Check
									size={16}
									class="flex-none text-accent opacity-0 group-aria-checked:opacity-100"
									aria-hidden="true"
								/>
							</button>
						</li>
					{/each}
				{/if}
			</ul>
			{#if oncreate !== undefined}
				<!-- The creation row, OUTSIDE the scroll box: the list can
				     grow past every viewport and the creation stays
				     reachable. -->
				<button
					class="person-picker__create mt-3 flex w-full items-center gap-2.5 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] text-text-soft bg-transparent border border-dashed border-border-strong rounded-control text-left cursor-pointer hover:bg-accent-soft hover:text-accent-soft-text"
					type="button"
					onclick={oncreate}
				>
					<span
						class="flex-none inline-flex items-center justify-center w-8 h-8 rounded-full border border-dashed border-border-strong"
					>
						<Plus size={18} aria-hidden="true" />
					</span>
					<span class="min-w-0 flex-1">{$_('people_create_from_face')}</span>
				</button>
			{/if}
		{/if}
		{#if errorKey !== ''}
			<p
				class="person-picker__failure m-0 mt-3 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{resolveErrorText($_, errorKey)}
			</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={close}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button disabled={selectedId === ''} onclick={submit}>
			<Check size={20} aria-hidden="true" />
			{$_(submitKey)}
		</Button>
	{/snippet}
</Modal>
