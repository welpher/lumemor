<script lang="ts">
	// The trash page (M2d Task 8): the bounded listing of the
	// caller's soft-deleted assets with each row's retention
	// countdown, the single restore, and the confirm-first permanent
	// delete. The SSE subscription converges the rows asset.deleted
	// and asset.restored events remove (the expiry job and other
	// devices' actions), so the view never shows a row that left the
	// trash; every other event is not this view's business.
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
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import PhotoTile from '$lib/timeline/PhotoTile.svelte';
	import { deleteAsset } from '$lib/api/assets';
	import { fetchTrash, restoreTrashedAsset, type TrashItem } from '$lib/api/trash';
	import { subscribeAssetEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';

	type TrashStatus = 'idle' | 'loading' | 'ready' | 'error';

	let items = $state<TrashItem[]>([]);
	let status = $state<TrashStatus>('idle');
	let errorKey = $state('');
	let actionErrorKey = $state('');
	let busy = $state(false);
	/** Bumping re-runs the listing (the retry button). */
	let retryTick = $state(0);
	/** The row whose permanent delete the confirm Modal names. */
	let pendingDelete = $state<TrashItem | null>(null);
	let deleteErrorKey = $state('');

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
			status = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	// The SSE convergence of this view: restored and deleted events
	// remove the named row. The other asset events (uploads, edits,
	// the pipeline) do not change a row's trash membership -- the
	// countdown text is a snapshot until the next listing.
	$effect(() => {
		return subscribeAssetEvents((event) => {
			if (event.name !== 'asset.restored' && event.name !== 'asset.deleted') {
				return;
			}
			removeRow(event.assetId);
		});
	});

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

	// The tile takes the timeline item shape; a trash row carries
	// every field of it except the two live-view flags, and no
	// favorite badge belongs on a recovery tile.
	function tileItem(entry: TrashItem) {
		return { ...entry, isFavorite: false };
	}

	function noopSelect(): void {
		// The trash view has no selection mode; PhotoTile's toggle
		// stays inert (selectable is always false).
	}
</script>

<svelte:head>
	<title>Trash · lumemor</title>
</svelte:head>

<section class="trash" aria-busy={status === 'loading'}>
	<header class="trash__header">
		<h1>{$_('nav_trash')}</h1>
		<p class="trash__note">{$_('trash_note')}</p>
	</header>

	{#if firstLoad}
		<p class="trash__status" role="status">{$_('trash_loading')}</p>
	{:else if settledEmpty}
		<div class="trash__empty">
			<p>{$_('trash_empty')}</p>
		</div>
	{:else if status === 'error'}
		<div class="trash__failure" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}>{$_('action_retry')}</Button>
		</div>
	{:else}
		<div class="trash__grid">
			{#each items as entry (entry.id)}
				<div class="trash__cell">
					<div class="trash__tile">
						<PhotoTile item={tileItem(entry)} selected={false} selectable={false} onselect={noopSelect} />
						<p class="trash__days">{$_('trash_remaining_days', { values: { count: entry.remainingDays } })}</p>
					</div>
					<div class="trash__actions">
						<Button variant="ghost" disabled={busy} onclick={() => void restore(entry)}>
							{$_('action_restore')}
						</Button>
						<Button variant="danger" disabled={busy} onclick={() => askDelete(entry)}>
							{$_('trash_delete_permanently')}
						</Button>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	{#if actionErrorKey !== ''}
		<p class="trash__error" role="alert">{errorText(actionErrorKey)}</p>
	{/if}
</section>

<Modal open={pendingDelete !== null} title={$_('trash_delete_title')} onclose={cancelDelete}>
	{#snippet children()}
		<p class="trash__confirm-body">
			{$_('trash_delete_body', { values: { name: pendingDelete?.originalFilename ?? '' } })}
		</p>
		{#if deleteErrorKey !== ''}
			<p class="trash__confirm-error" role="alert">{errorText(deleteErrorKey)}</p>
		{/if}
	{/snippet}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelDelete}>{$_('common_cancel')}</Button>
		<Button variant="danger" disabled={busy} onclick={() => void confirmDelete()}>
			{$_('trash_delete_confirm')}
		</Button>
	{/snippet}
</Modal>

<style>
	.trash {
		max-width: 1200px;
		margin: 0 auto;
	}

	.trash__header {
		margin-bottom: 16px;
	}

	.trash__header h1 {
		margin: 0 0 4px;
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 26px;
		letter-spacing: 0.01em;
		color: var(--color-text);
	}

	.trash__note {
		margin: 0;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text-soft);
	}

	.trash__status {
		margin: 48px 0;
		text-align: center;
		color: var(--color-text-soft);
	}

	.trash__empty {
		max-width: 420px;
		margin: 48px auto;
		padding: 32px 24px;
		text-align: center;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
	}

	.trash__empty p {
		margin: 0;
		color: var(--color-text-soft);
	}

	.trash__failure {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		max-width: 420px;
		margin: 48px auto;
		padding: 20px;
		text-align: center;
		border: 1px solid var(--color-danger);
		border-radius: var(--radius-card);
		color: var(--color-danger-text);
	}

	.trash__failure p {
		margin: 0;
	}

	.trash__grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(240px, 100%), 1fr));
		gap: 12px;
	}

	.trash__cell {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	/* The recovery tile stacks the countdown over the tile and the
	 * row's actions under it: the trash styling carries the
	 * de-emphasized tone of a row on its way out. */
	.trash__tile {
		position: relative;
		border-radius: var(--radius-control);
		outline: 1px solid var(--color-border);
		outline-offset: -1px;
		opacity: 0.85;
	}

	.trash__days {
		position: absolute;
		bottom: 6px;
		left: 6px;
		z-index: 2;
		margin: 0;
		padding: 2px 8px;
		font: 600 12px/1.5 var(--font-body);
		color: var(--color-accent-on);
		background-color: rgb(20 26 46 / 0.65);
		backdrop-filter: blur(2px);
		border-radius: var(--radius-control);
	}

	.trash__actions {
		display: flex;
		gap: 8px;
	}

	.trash__actions :global(button) {
		flex: 1;
	}

	.trash__error {
		max-width: 420px;
		margin: 16px auto 0;
		font: 500 14px/1.4 var(--font-body);
		color: var(--color-danger-text);
		text-align: center;
	}

	.trash__confirm-body {
		margin: 0;
		font: 400 15px/1.5 var(--font-body);
		color: var(--color-text);
	}

	.trash__confirm-error {
		margin: 12px 0 0;
		font: 500 14px/1.4 var(--font-body);
		color: var(--color-danger-text);
	}
</style>
