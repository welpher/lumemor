<script lang="ts">
	// The memory picker: the wall-selection entry's dialog. The page
	// hands it the ids CAPTURED AT OPEN (the modal-pinned selection
	// discipline -- the wall's opener hands the pair) and the done
	// continuation the pass's landing fires; the dialog owns the
	// rest of the flow -- the caller's own memories fetched fresh on
	// open, the pick-then-add write, the inline create-then-add
	// write, the honest counts-only report -- because four pages
	// compose this one flow (the timeline, the search, the favorites
	// and hidden walls), and a component owning its walk and writes
	// is the curated-wall wrapper's precedent.
	//
	// THE ROWS ARE STATE SELECTORS (the audience picker's face):
	// the row's mousedown picks, the keyboard is the dialog's own
	// window handler -- no row is a button.
	//
	// THE TWO PATHS:
	// - PICK: one row pick POSTs the captured ids to the memory's
	//   items face; the row flips to the accent added state (an
	//   added row is inert -- a re-add would be an all-skipped no-op,
	//   the row already says it), and the write's {added, skipped}
	//   answer renders the skip-note honesty line (the ids that
	//   stayed out: not visible to the caller, not live rows). A
	//   later row in the same session adds the SAME ids again -- the
	//   reference sets are not exclusive.
	// - CREATE: the inline row expands to the title input + the
	//   create-and-add action -- POST /memories (the picker's
	//   creation carries the title alone; the audience is the
	//   memory's later edit), then the same items POST; the new row
	//   lands at the head of the list already added.
	//
	// THE KEYBOARD: ↑/↓ move the highlight over the rows and the
	// create row, Enter activates the highlighted row (the create
	// input keeps its own typing -- Enter there submits the create,
	// never a pick), Esc closes the whole dialog (the modal's own
	// stroke). The wall never converges on this flow: adding to a
	// memory changes no photo's state, so the report IS the outcome
	// -- the done continuation fires the wall's success arm (clear
	// the selection) from the Done affordance alone.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import { addAlbumItems, createMemory, listAlbums, type AlbumSummary } from '$lib/api/albums';
	import { resolveErrorText } from '$lib/i18n/errors';
	import Plus from '@lucide/svelte/icons/plus';
	import Check from '@lucide/svelte/icons/check';
	import X from '@lucide/svelte/icons/x';
	import Info from '@lucide/svelte/icons/info';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The ids the wall captured at open -- the whole session's set. */
		ids: readonly string[];
		/**
		 * The pass's landing: the page fires the wall's done
		 * continuation (the success arm clears the selection) when
		 * the report's Done affordance closes the dialog.
		 */
		onpass?: () => void;
	};

	let { open = $bindable(false), ids, onpass }: Props = $props();

	// The caller's own memories (the pick list), fetched fresh on
	// open; the loading and failure states are the list's own lines.
	let memories = $state<AlbumSummary[]>([]);
	let listLoading = $state(false);
	/** The ids that landed an add in this session (the rows' added state). */
	let addedIds = $state<string[]>([]);
	/** The honest counts-only report of the last landed add (null when none). */
	let report = $state<{ added: number; skipped: number } | null>(null);
	/** The error-code i18n key of the last failed write ('' when none). */
	let errorKey = $state('');
	/** Writes run one at a time: the affordances disable while in flight. */
	let busy = $state(false);
	/** The inline create form's visibility (the create row's expansion). */
	let createOpen = $state(false);
	/** The create form's title draft. */
	let titleDraft = $state('');
	/** Whether a create attempt already ran for the current draft. */
	let checked = $state(false);
	/** The highlighted row's index (the keyboard's cursor). */
	let highlight = $state(0);

	// The seed on every open: a fresh list fetch, an empty added set,
	// a cleared report. The ids stay the pair the wall handed over.
	$effect(() => {
		if (open) {
			addedIds = [];
			report = null;
			errorKey = '';
			busy = false;
			createOpen = false;
			titleDraft = '';
			checked = false;
			highlight = 0;
			memories = [];
			listLoading = true;
			void (async () => {
				const outcome = await listAlbums();
				if (!open) {
					return;
				}
				listLoading = false;
				if (outcome.ok) {
					memories = outcome.albums.filter(
						(album) => album.type === 'memory' && album.callerRole === 'owner'
					);
				} else {
					errorKey = outcome.errorKey;
				}
			})();
		}
	});

	/** The contract's bound, rune-counted: the trimmed title spans 1-64 characters. */
	function validDraft(): boolean {
		const runes = [...titleDraft.trim()].length;
		return runes >= 1 && runes <= 64;
	}

	const boundViolated = $derived(checked && !validDraft());

	// The rows the keyboard walks: the memories plus the create row.
	const rowCount = $derived(memories.length + 1);

	/** The highlighted memory row, if the highlight sits on one. */
	function highlightedMemory(): AlbumSummary | null {
		return memories[highlight] ?? null;
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (!open) {
			return;
		}
		// The create input keeps its own keys: Enter submits the
		// create (the button's activation), never a pick.
		if (event.target instanceof HTMLElement && event.target.id === 'add-memory-create-title') {
			return;
		}
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			highlight = Math.min(rowCount - 1, highlight + 1);
			return;
		}
		if (event.key === 'ArrowUp') {
			event.preventDefault();
			highlight = Math.max(0, highlight - 1);
			return;
		}
		if (event.key === 'Enter') {
			// Enter activates the highlighted row: a memory pick, or
			// the create row's expansion.
			event.preventDefault();
			if (highlightedMemory() !== null) {
				void addTo(highlightedMemory() as AlbumSummary);
			} else {
				openCreate();
			}
		}
	}

	/** The pick path: POST the captured ids, adopt the honest report. */
	async function addTo(memory: AlbumSummary): Promise<void> {
		if (busy || addedIds.includes(memory.id) || ids.length === 0) {
			return;
		}
		busy = true;
		errorKey = '';
		const outcome = await addAlbumItems(memory.id, [...ids]);
		busy = false;
		if (!outcome.ok) {
			errorKey = outcome.errorKey;
			return;
		}
		addedIds = [...addedIds, memory.id];
		report = { added: outcome.added, skipped: outcome.skipped };
	}

	/** The create row's activation: expand the inline form. */
	function openCreate(): void {
		if (busy) {
			return;
		}
		createOpen = true;
	}

	/** The create path: create the memory, then feed it the same ids. */
	async function createAndAdd(): Promise<void> {
		if (busy || ids.length === 0) {
			return;
		}
		checked = true;
		if (!validDraft()) {
			return;
		}
		busy = true;
		errorKey = '';
		const created = await createMemory({ title: titleDraft.trim() });
		if (!created.ok) {
			busy = false;
			errorKey = created.errorKey;
			return;
		}
		// The new card lands at the head of the list already added;
		// the items write follows immediately.
		memories = [created.memory, ...memories];
		const outcome = await addAlbumItems(created.memory.id, [...ids]);
		busy = false;
		if (!outcome.ok) {
			errorKey = outcome.errorKey;
			return;
		}
		addedIds = [...addedIds, created.memory.id];
		highlight = 0;
		report = { added: outcome.added, skipped: outcome.skipped };
	}

	// The shared unknown-code fallback; the reactive `$_` keeps the
	// text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	/** One row's thumb: the newest member's serving copy, the same
	 * face the memory cards carry. */
	function rowThumb(memory: AlbumSummary): string {
		const coverId = memory.coverAssetIds[0];
		return coverId === undefined ? '' : `/api/v1/assets/${encodeURIComponent(coverId)}/thumbnail?size=tile`;
	}

	function done(): void {
		open = false;
		onpass?.();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<Modal bind:open title={$_('action_add_memory')}>
	<p class="add-memory__intro m-0 mb-2.5 font-body font-normal text-[13px] leading-[1.4] text-text-faint">
		{$_('memories_picker_intro', { values: { count: ids.length } })}
	</p>
	<div class="add-memory__list bg-page border border-border rounded-control overflow-hidden">
		{#if listLoading}
			<p
				class="add-memory__loading m-0 px-3 py-2.5 font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in"
				role="status"
			>
				{$_('memories_loading')}…
			</p>
		{:else}
			<ul class="add-memory__rows m-0 p-0 list-none" role="listbox" aria-label={$_('memories_own_section')}>
				{#each memories as memory, index (memory.id)}
					<li
						class="add-memory__row flex items-center gap-2.5 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] cursor-pointer {addedIds.includes(
							memory.id
						)
							? 'bg-accent-soft text-accent-soft-text'
							: highlight === index
								? 'bg-accent-soft text-accent-soft-text'
								: 'text-text'}"
						role="option"
						aria-selected={addedIds.includes(memory.id) || highlight === index}
						onmousedown={() => void addTo(memory)}
					>
						<span
							class="flex-none relative block w-8 h-8 rounded-control overflow-hidden bg-accent-soft"
							aria-hidden="true"
						>
							{#if rowThumb(memory) !== ''}
								<img
									class="absolute inset-0 w-full h-full object-cover"
									src={rowThumb(memory)}
									alt=""
									loading="lazy"
									decoding="async"
									draggable="false"
								/>
							{/if}
						</span>
						<span class="min-w-0 flex-1 truncate">{memory.title}</span>
						<CountLabel
							class="flex-none font-normal text-[12px] leading-[1.4] {addedIds.includes(memory.id)
								? ''
								: 'text-text-faint'}"
							count={memory.itemCount}
							label={$_('memories_photo_count', { values: { count: memory.itemCount } })}
						/>
						{#if addedIds.includes(memory.id)}
							<!-- The accent added state: the row's write landed; the
							     row is inert (a re-add would be the all-skipped
							     no-op the state already says). -->
							<span
								class="flex-none inline-flex items-center gap-1 font-body font-semibold text-[12px] leading-[1.4]"
							>
								<Check size={14} aria-hidden="true" />
								{$_('memories_picker_added')}
							</span>
						{/if}
					</li>
				{/each}
				<li
					class="add-memory__create-row flex items-center gap-2.5 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] text-text-soft cursor-pointer border-t border-border {highlight ===
					memories.length
						? 'bg-accent-soft text-accent-soft-text'
						: ''}"
					role="option"
					aria-selected={highlight === memories.length}
					onmousedown={openCreate}
				>
					<Plus size={16} class="flex-none text-accent" aria-hidden="true" />
					{$_('memories_picker_create_row')}
				</li>
			</ul>
			{#if createOpen}
				<!-- The inline create form: the create row's expansion --
				     the title input and the create-and-add action. -->
				<div class="add-memory__create-form flex items-center gap-2 px-3 pb-2.5 border-t border-border">
					<input
						class="add-memory__create-input flex-1 min-w-0 px-2.5 py-1.5 font-body font-normal text-[13px] leading-[1.4] text-text bg-card border border-border-strong rounded-control"
						type="text"
						id="add-memory-create-title"
						aria-label={$_('memories_picker_create_label')}
						bind:value={titleDraft}
						maxlength={128}
						autocomplete="off"
					/>
					<Button class="add-memory__create-submit" disabled={busy} onclick={() => void createAndAdd()}>
						{$_('memories_picker_create_submit')}
					</Button>
				</div>
				{#if boundViolated}
					<p
						class="add-memory__bound m-0 px-3 pb-2 font-body font-medium text-[12px] leading-[1.4] text-danger-text"
					>
						{$_('albums_title_invalid')}
					</p>
				{/if}
			{/if}
			<div
				class="add-memory__keys flex gap-3 px-3 py-1.5 border-t border-border font-body font-normal text-[11px] leading-[1.4] text-text-faint"
			>
				<span>{$_('albums_keys_select')}</span>
				<span>{$_('memories_keys_add')}</span>
				<span>{$_('albums_keys_close')}</span>
			</div>
		{/if}
	</div>
	{#if report !== null && report.skipped > 0}
		<!-- The skip-note honesty line: the write's own counts-only
		     answer -- the ids that stayed out, with the ground. -->
		<p
			class="add-memory__skip-note flex items-start gap-1.5 m-0 mt-2.5 font-body font-normal text-[13px] leading-[1.4] text-text-faint"
			role="status"
		>
			<Info size={14} class="flex-none mt-0.5" aria-hidden="true" />
			<span>{$_('memories_picker_skip_note', { values: { count: report.skipped } })}</span>
		</p>
	{/if}
	{#if errorKey !== ''}
		<p
			class="add-memory__failure m-0 mt-2.5 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(errorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={() => (open = false)}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button disabled={busy} onclick={done}>
			<Check size={20} aria-hidden="true" />
			{$_('action_done')}
		</Button>
	{/snippet}
</Modal>
