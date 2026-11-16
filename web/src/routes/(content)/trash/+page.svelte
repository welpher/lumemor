<script lang="ts">
	// The trash page: the bounded listing of the
	// caller's soft-deleted assets with each row's retention
	// countdown, the single restore, the confirm-first permanent
	// delete, the whole-trash pair (restore all, empty trash) and the
	// multi-select batch. The SSE subscription converges through the
	// LISTING, never on an event's claim: the stream's contract replays the
	// retained ring to every fresh subscriber (a first connect
	// carries no Last-Event-ID cursor, and the refcounted stream
	// reconnects while routes swap), so a stale restored or deleted
	// event for a row that is in the trash RIGHT NOW must not erase
	// it. A restored or deleted event for a row the view holds
	// re-fetches the listing; the server's answer decides which rows
	// stay (the expiry job and other devices' actions included), and
	// every other event is not this view's business.
	//
	// THE PINNED CHOICES:
	//
	// - The restore runs through the single-asset endpoint (POST
	//   /trash/{id}/restore), not the batch surface -- one row, one
	//   click, one request. Its 404 counts as success: the row is
	//   already gone (the expiry job's purge), exactly like the
	//   permanent delete's already-cleaned reading.
	// - The permanent delete opens a confirm Modal first (the
	//   irreversible action is never one accidental click away) and
	//   reuses the viewer client's deleteAsset.
	// - The whole-trash pair is the TOOLBAR's, not the rows': the
	//   restore-all runs directly (a whole restore is reversible
	//   through the timeline's own trash flow), the empty trash
	//   confirms first with the count in the sentence, and both are
	//   disabled -- never hidden -- on an empty trash. After either
	//   one the view re-fetches once explicitly: the per-row events
	//   the server publishes would converge it too, but the
	//   sweep-sized burst of the caller's own action deserves one
	//   authoritative answer instead of a replay chase.
	// - The multi-select is the wall's selection transposition, the
	//   same vocabulary with zero new paradigms: the tiles' check
	//   affordance (hover-revealed, aria-checked), the batch bar
	//   (count, restore, the dangerous delete, clear), Escape and the
	//   bar's clear dropping the selection, and the delete arm
	//   confirming first with the count in its sentence. The row's
	//   own actions stay on the row, untouched by any of it.
	import { _ } from 'svelte-i18n';
	import { SvelteSet } from 'svelte/reactivity';
	import Button from '$lib/components/Button.svelte';
	import StateCard from '$lib/components/StateCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import BatchBar from '$lib/timeline/BatchBar.svelte';
	import PhotoTile from '$lib/timeline/PhotoTile.svelte';
	import { toast } from '$lib/components/toast.svelte';
	import { deleteAsset } from '$lib/api/assets';
	import {
		fetchTrash,
		restoreTrashedAsset,
		restoreAllTrash,
		emptyTrash,
		batchTrash,
		type TrashItem
	} from '$lib/api/trash';
	import { subscribeAssetEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import X from '@lucide/svelte/icons/x';

	type TrashStatus = 'idle' | 'loading' | 'ready' | 'error';

	let items = $state<TrashItem[]>([]);
	/** The listing's retentionDays: the note's number, the same source the countdowns use. */
	let retentionDays = $state(30);
	let status = $state<TrashStatus>('idle');
	let errorKey = $state('');
	let actionErrorKey = $state('');
	let busy = $state(false);
	/** Bumping re-runs the listing (the retry button). */
	let retryTick = $state(0);
	/** The row whose permanent delete the confirm Modal names. */
	let pendingDelete = $state<TrashItem | null>(null);
	let deleteErrorKey = $state('');
	/** The whole-trash empty's confirm gate (the count rides the body). */
	let pendingEmpty = $state(false);
	let emptyErrorKey = $state('');
	/** The multi-select batch delete's confirm gate. */
	let pendingBatchDelete = $state(false);

	const firstLoad = $derived(items.length === 0 && status === 'loading');
	const settledEmpty = $derived(items.length === 0 && status === 'ready');

	// The listing, keyed on the retry tick. A superseded run -- the
	// retry fired again, or the route left -- never applies.
	$effect(() => {
		void retryTick;
		status = 'loading';
		errorKey = '';
		let current = true;
		void (async () => {
			const outcome = await fetchTrash();
			if (!current) {
				return;
			}
			if (!outcome.ok) {
				status = 'error';
				errorKey = outcome.errorKey;
				return;
			}
			items = outcome.items;
			retentionDays = outcome.retentionDays;
			status = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	// The SSE convergence of this view is REPLAY-SAFE: restored and
	// deleted events for rows the view HOLDS re-fetch the listing,
	// and the server's answer decides whether a row leaves. The
	// event alone is a claim, never state -- the stream replays its
	// retained ring to every fresh subscriber, and the
	// restore-then-retrash flow delivers an OLD restored event for
	// an asset that is in the trash right now; the row must survive
	// it (the listing, having just returned it, says so). Events for
	// assets the view does not hold (the rest of a replayed ring)
	// cost nothing; the other asset events (uploads, edits, the
	// pipeline) do not change a row's trash membership -- the
	// countdown text is a snapshot until the next listing.
	let refreshRun = 0;

	/** Re-run the listing and adopt its answer (silent on failure). */
	async function refresh(): Promise<void> {
		// Newest-wins, the refetch discipline of the viewer's detail
		// convergence: a slower stale response never lands over a
		// newer one.
		const run = ++refreshRun;
		const outcome = await fetchTrash();
		if (run !== refreshRun) {
			return;
		}
		if (!outcome.ok) {
			// Keep the current rows: the refresh is a convergence
			// probe, not the load path -- the retry button stays a
			// failed first load's recovery.
			return;
		}
		items = outcome.items;
		retentionDays = outcome.retentionDays;
		status = 'ready';
	}

	$effect(() => {
		return subscribeAssetEvents((event) => {
			if (event.name !== 'asset.restored' && event.name !== 'asset.deleted') {
				return;
			}
			if (items.some((entry) => entry.id === event.assetId)) {
				void refresh();
			}
		});
	});

	// THE SELECTION: the wall's transposition, page-level (the trash
	// grid is a plain grid, not the justified wall). The tiles' check
	// affordances toggle it, the first checked tile arms the reveal
	// for every tile (the mid-selection reveal, the touch-friendly
	// path), and Escape and the bar's clear drop it.
	const selection = new SvelteSet<string>();

	/** Whether the selection is alive: it gates every tile's check reveal. */
	const selectable = $derived(selection.size > 0);

	function toggleSelected(id: string, next: boolean): void {
		if (next) {
			selection.add(id);
		} else {
			selection.delete(id);
		}
	}

	function clearSelection(): void {
		selection.clear();
	}

	// The Escape exit, the wall's own reading: a live selection
	// drops (the confirm dialogs own their own keys, and a closed
	// dialog's Escape never reaches a selection that the row-level
	// actions opened).
	function handleWindowKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && selection.size > 0) {
			selection.clear();
		}
	}

	// The batch run: the same choreography the walls' bars keep --
	// busy disables the bar's buttons, a landed pass clears the
	// selection and echoes the count, a failure keeps the selection
	// (the retry affordance) and surfaces the mapped error. The
	// delete arm's confirm lives in the Modal above the bar; this
	// runner fires only past it. After a landed pass one explicit
	// re-fetch bottoms out the convergence: the server's per-row
	// events would reach this view through its own subscription too,
	// but the caller's own sweep deserves one authoritative answer.
	let batchBusy = $state(false);
	let batchErrorKey = $state('');

	async function runBatch(action: 'restore' | 'delete'): Promise<void> {
		if (batchBusy || selection.size === 0) {
			return;
		}
		batchBusy = true;
		batchErrorKey = '';
		const outcome = await batchTrash(action, [...selection]);
		batchBusy = false;
		if (!outcome.ok) {
			batchErrorKey = outcome.errorKey;
			return;
		}
		const count = outcome.affected;
		selection.clear();
		if (action === 'restore') {
			toast($_('trash_batch_restore_done', { values: { count } }));
		} else {
			toast($_('trash_batch_delete_done', { values: { count } }));
		}
		void refresh();
	}

	function askDeleteSelected(): void {
		batchErrorKey = '';
		pendingBatchDelete = true;
	}

	function cancelBatchDelete(): void {
		if (batchBusy) {
			return;
		}
		pendingBatchDelete = false;
	}

	async function confirmBatchDelete(): Promise<void> {
		if (batchBusy || selection.size === 0) {
			return;
		}
		pendingBatchDelete = false;
		await runBatch('delete');
	}

	function removeRow(id: string): void {
		items = items.filter((entry) => entry.id !== id);
	}

	function retry(): void {
		retryTick += 1;
	}

	// The shared unknown-code fallback (see $lib/i18n/errors); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	async function restore(entry: TrashItem): Promise<void> {
		if (busy) {
			return;
		}
		busy = true;
		actionErrorKey = '';
		const outcome = await restoreTrashedAsset(entry.id);
		busy = false;
		// Success and 404 both drop the row: the row is back in the
		// live views, or it was purged before the request landed.
		if (outcome.ok) {
			removeRow(entry.id);
			return;
		}
		actionErrorKey = outcome.errorKey;
	}

	function askDelete(entry: TrashItem): void {
		deleteErrorKey = '';
		pendingDelete = entry;
	}

	function cancelDelete(): void {
		if (busy) {
			return;
		}
		pendingDelete = null;
		deleteErrorKey = '';
	}

	async function confirmDelete(): Promise<void> {
		const entry = pendingDelete;
		if (entry === null || busy) {
			return;
		}
		busy = true;
		deleteErrorKey = '';
		const outcome = await deleteAsset(entry.id);
		busy = false;
		// Success and 404 both drop the row: the asset is gone either
		// way, the same already-cleaned reading the viewer's delete
		// pins.
		if (outcome.ok) {
			pendingDelete = null;
			removeRow(entry.id);
			return;
		}
		deleteErrorKey = outcome.errorKey;
	}

	// THE WHOLE-TRASH PAIR: the toolbar's, always rendered, disabled
	// on an empty trash (never hidden -- the actions' place does not
	// move when the content does). The restore-all runs directly; a
	// whole restore is reversible through the timeline's own trash
	// flow. The empty confirms first, the count in the sentence.
	async function restoreAll(): Promise<void> {
		if (busy || items.length === 0) {
			return;
		}
		busy = true;
		actionErrorKey = '';
		const outcome = await restoreAllTrash();
		busy = false;
		if (!outcome.ok) {
			actionErrorKey = outcome.errorKey;
			return;
		}
		selection.clear();
		toast($_('trash_restore_all_done'));
		void refresh();
	}

	function askEmpty(): void {
		emptyErrorKey = '';
		pendingEmpty = true;
	}

	function cancelEmpty(): void {
		if (busy) {
			return;
		}
		pendingEmpty = false;
		emptyErrorKey = '';
	}

	async function confirmEmpty(): Promise<void> {
		if (busy || items.length === 0) {
			return;
		}
		busy = true;
		emptyErrorKey = '';
		const outcome = await emptyTrash();
		busy = false;
		if (!outcome.ok) {
			emptyErrorKey = outcome.errorKey;
			return;
		}
		pendingEmpty = false;
		selection.clear();
		toast($_('trash_empty_done'));
		void refresh();
	}

	// The tile takes the timeline item shape; a trash row carries
	// every field of it except the two live-view flags, and no
	// favorite badge belongs on a recovery tile.
	function tileItem(entry: TrashItem) {
		return { ...entry, isFavorite: false };
	}
</script>

<svelte:head>
	<title>Trash · lumemor</title>
</svelte:head>

<svelte:window onkeydown={handleWindowKeydown} />

<section class="trash px-4 pt-4 pb-12" aria-busy={status === 'loading'}>
	<!-- THE ONE ROW: the page title with the row count on the left,
	     the whole-trash pair on the right (justify-between -- the
	     toolbar owns the SITE-level actions; a row's actions live on
	     the row), one divider under the row. The retention note is NOT an
	     action and never rides the bar: it sits below the divider,
	     where the content it explains begins. -->
	<div class="trash__toolbar flex flex-wrap items-center justify-between gap-3 pb-4 border-b border-border">
		<h1 class="trash__title m-0 font-display font-bold text-[20px] leading-[1.3] text-text">
			{$_('nav_trash')}
			{#if status === 'ready'}
				<CountLabel
					class="trash__title-count font-body font-normal text-[15px] leading-[1.4] text-text-soft"
					count={items.length}
					label={$_('trash_bar_count', { values: { count: items.length } })}
				/>
			{/if}
		</h1>
		<!-- The whole-trash pair: order-last so a wrap lands the actions
		     under the title (the narrow-width reading), disabled on an
		     empty trash, never hidden. -->
		<div class="trash__actions order-last flex items-center gap-2">
			<Button variant="ghost" disabled={busy || items.length === 0} onclick={() => void restoreAll()}>
				<RotateCcw size={20} aria-hidden="true" />
				{$_('trash_restore_all')}
			</Button>
			<Button variant="danger-text" disabled={busy || items.length === 0} onclick={askEmpty}>
				<Trash2 size={20} aria-hidden="true" />
				{$_('trash_empty')}
			</Button>
		</div>
	</div>

	<div class="trash__content pt-4">
		<p class="trash__note m-0 pb-3 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
			{$_('trash_note_days', { values: { days: retentionDays } })}
		</p>

		{#if firstLoad}
			<LoadingLine label={$_('trash_loading')} />
		{:else if settledEmpty}
			<StateCard title={$_('trash_empty_title')} body={$_('trash_empty_body')} />
		{:else if status === 'error'}
			<StateCard tone="error" body={errorText(errorKey)} actionLabel={$_('action_retry')} onaction={retry} />
		{:else}
			<div class="trash__grid grid grid-cols-[repeat(auto-fill,minmax(min(240px,100%),1fr))] gap-3">
				{#each items as entry (entry.id)}
					<div class="trash__cell flex flex-col gap-2">
						<!-- The recovery tile keeps the management grid's square
					     crop (the wrapper owns the box now that the tile
					     renders at true aspect inside the justified walls)
					     and stacks the countdown over it and the row's
					     actions under it: the trash styling carries the
					     de-emphasized tone of a row on its way out. -->
						<div
							class="trash__tile relative aspect-square overflow-hidden rounded-control [outline:1px_solid_var(--color-border)] outline-offset-[-1px] opacity-85"
						>
							<PhotoTile
								item={tileItem(entry)}
								selected={selection.has(entry.id)}
								{selectable}
								onselect={toggleSelected}
							/>
							<p
								class="trash__days absolute bottom-1.5 left-1.5 z-(--z-tile-chip) py-0.5 px-2 font-body font-semibold text-[12px] leading-[1.5] text-cream-text bg-night rounded-control tabular-nums"
							>
								{$_('trash_remaining_days', { values: { count: entry.remainingDays } })}
							</p>
							<!-- The badge's full sentence for screen readers: the
							     chip states the remaining days alone, the file it
							     belongs to lives on the tile's own link -- the
							     sr-only line joins the two. -->
							<p class="sr-only">
								{$_('trash_remaining_days_label', {
									values: { name: entry.originalFilename, count: entry.remainingDays }
								})}
							</p>
						</div>
						<div class="trash__actions flex gap-2">
							<!-- The row's actions are icon-only (the icon-first rule):
						     the restore pair's rotate mark and the destructive
						     delete mark, named by their aria-labels. -->
							<IconButton
								class="trash__restore flex-1"
								variant="ghost"
								disabled={busy}
								aria-label={$_('action_restore')}
								title={$_('action_restore')}
								onclick={() => void restore(entry)}
							>
								<RotateCcw size={20} aria-hidden="true" />
							</IconButton>
							<IconButton
								class="trash__delete flex-1"
								variant="danger"
								disabled={busy}
								aria-label={$_('trash_delete_permanently')}
								title={$_('trash_delete_permanently')}
								onclick={() => askDelete(entry)}
							>
								<Trash2 size={20} aria-hidden="true" />
							</IconButton>
						</div>
					</div>
				{/each}
			</div>
		{/if}

		{#if actionErrorKey !== ''}
			<p
				class="trash__error max-w-[420px] mx-auto mt-4 text-center font-body font-medium text-[14px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{errorText(actionErrorKey)}
			</p>
		{/if}
	</div>
</section>

<!-- The row's confirm: the same dialog it has always been, untouched. -->
<Modal open={pendingDelete !== null} title={$_('trash_delete_title')} onclose={cancelDelete}>
	<p class="trash__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
		{$_('trash_delete_body', { values: { name: pendingDelete?.originalFilename ?? '' } })}
	</p>
	{#if deleteErrorKey !== ''}
		<p
			class="trash__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(deleteErrorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelDelete}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="danger" disabled={busy} onclick={() => void confirmDelete()}>
			<Trash2 size={20} aria-hidden="true" />
			{$_('trash_delete_confirm')}
		</Button>
	{/snippet}
</Modal>

<!-- The whole-trash empty's confirm: the count in the sentence, the
     same gate shape the row's delete keeps. -->
<Modal open={pendingEmpty} title={$_('trash_empty_confirm_title')} onclose={cancelEmpty}>
	<p class="trash__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
		{$_('trash_empty_confirm_body', { values: { count: items.length } })}
	</p>
	{#if emptyErrorKey !== ''}
		<p
			class="trash__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(emptyErrorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelEmpty}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="danger" disabled={busy} onclick={() => void confirmEmpty()}>
			<Trash2 size={20} aria-hidden="true" />
			{$_('trash_empty')}
		</Button>
	{/snippet}
</Modal>

<!-- The multi-select's batch delete confirm: the count in the
     sentence, the same gate shape the row's delete keeps. -->
<Modal open={pendingBatchDelete} title={$_('trash_batch_delete_title')} onclose={cancelBatchDelete}>
	<p class="trash__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
		{$_('trash_batch_delete_body', { values: { count: selection.size } })}
	</p>
	{#if batchErrorKey !== ''}
		<p
			class="trash__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(batchErrorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={batchBusy} onclick={cancelBatchDelete}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="danger" disabled={batchBusy} onclick={() => void confirmBatchDelete()}>
			<Trash2 size={20} aria-hidden="true" />
			{$_('trash_delete_confirm')}
		</Button>
	{/snippet}
</Modal>

{#if selection.size > 0}
	<BatchBar
		count={selection.size}
		allFavorite={false}
		errorKey={batchErrorKey}
		busy={batchBusy}
		actions={['restore']}
		onaction={(action) => void runBatch(action === 'restore' ? 'restore' : 'delete')}
		ondeleteselected={askDeleteSelected}
		onclear={clearSelection}
	/>
{/if}
