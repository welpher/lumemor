<script lang="ts">
	// The timeline wall, the app's home surface: keyset pages of
	// tiles stream in through the pagination client (an
	// IntersectionObserver sentinel over a 600px prefetch margin, the
	// client's in-flight guard collapsing every burst into one
	// fetch), month and day groups carry sticky headers, the right
	// rail scrubs the loaded months (the rail is the wall's own
	// organ -- the 2026-11-08 pin -- so it comes with the shared Wall
	// component), and selection turns the wall into a curation
	// surface with the floating batch bar. Everything around the
	// wall here -- the data walk, the selection state, the dialogs --
	// is the page's own concern.
	//
	// THE ADD-TO-ALBUM FLOW (M3c Task 5): the batch bar's dialog adds
	// the selection to one of the caller's own virtual albums, or
	// creates the album first -- the honest {added, skipped} report
	// of the write's own answer lands in the dialog. The ?album=
	// query parameter (the album detail's add-photos handoff) arms
	// the select mode and preselects that album in the dialog.
	//
	// THE ADD-TAG FLOW (M3d Task 4): the batch bar's second dialog
	// tags the selection with one of the caller's existing tags, or
	// creates the tag first (idempotently) -- the same honest
	// report. Tag writes are per-user state and publish nothing, so
	// the report is the outcome and the wall never converges.
	//
	// The SSE subscriptions live for the page's lifetime: the first
	// subscriber opens the stream, leaving the page closes it, and
	// every asset event converges through the client. The album
	// events restart the walk: the timeline IS the union of the
	// feeding albums, and the membership arms (a share granted or
	// revoked, an album deleted, a shared album's curation) change
	// what the union holds -- the events carry ids only, so the fresh
	// page one owns the truth.
	//
	// Render windowing is row-level: every justified row carries
	// content-visibility with its engine-computed height (see the
	// Wall component); full virtualization stays an M3 option if the
	// wall outgrows it.
	import { _ } from 'svelte-i18n';
	import { page } from '$app/state';
	import { SvelteSet } from 'svelte/reactivity';
	import { createTimelineClient, type BatchAction } from '$lib/api/timeline';
	import { addAlbumItems, createAlbum, listAlbums, type AlbumSummary } from '$lib/api/albums';
	import { addTagAssets, createTag, listTags, type TagSummary } from '$lib/api/tags';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAlbumEvents, subscribeAssetEvents } from '$lib/events';
	import { publishWallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import BatchBar from '$lib/timeline/BatchBar.svelte';
	import AddToAlbumDialog from '$lib/albums/AddToAlbumDialog.svelte';
	import AddTagDialog from '$lib/tags/AddTagDialog.svelte';
	import Button from '$lib/components/Button.svelte';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	const timeline = createTimelineClient();

	// The selection is a reactive id set: the tiles' check
	// affordances toggle it (the entry, R2-11: the check reveals on
	// hover/keyboard focus on every tile, and its first click enters
	// the selection), Esc and the bar's clear drop it, and the
	// ?album= handoff arms the select mode for the add-photos flow
	// (the touch-friendly path the toolbar button used to carry).
	const selection = new SvelteSet<string>();
	let selectMode = $state(false);
	let batchErrorKey = $state('');
	let batchBusy = $state(false);

	// The add-to-album flow's state: the dialog, the caller's own
	// virtual albums fetched on open, the honest report of the last
	// landed add, and the album a create-then-add chain already made
	// (a retry reuses it instead of creating a second album).
	let addOpen = $state(false);
	let addBusy = $state(false);
	let addErrorKey = $state('');
	let addAlbums = $state<AlbumSummary[]>([]);
	let addLoading = $state(false);
	let addReport = $state<{ added: number; skipped: number } | null>(null);
	let createdAlbumId = $state('');

	// The add-tag flow's state (M3d, the batch bar's second dialog):
	// the vocabulary fetched on open, the honest report of the last
	// landed add. The create is idempotent (a name the caller owns
	// converges on the existing row), so no created-id bookkeeping.
	let tagOpen = $state(false);
	let tagBusy = $state(false);
	let tagErrorKey = $state('');
	let tagList = $state<TagSummary[]>([]);
	let tagLoading = $state(false);
	let tagReport = $state<{ added: number; skipped: number } | null>(null);

	// The ?album= handoff: the album detail's add-photos action sends
	// the caller here to pick photos for that album -- the select
	// mode arms and the dialog preselects it.
	const preselectAlbumId = $derived(page.url.searchParams.get('album') ?? '');

	$effect(() => {
		if (preselectAlbumId !== '') {
			selectMode = true;
		}
	});

	// The scrubber's months: the wall's own organ now (the
	// 2026-11-08 pin) -- the axis folds its months from the same
	// internal groups the rendering packs, so this page's duplicate
	// fold is gone.
	const selectable = $derived(selectMode || selection.size > 0);
	const selectedItems = $derived($timeline.items.filter((timelineItem) => selection.has(timelineItem.id)));
	const allSelectedFavorite = $derived(
		selectedItems.length > 0 && selectedItems.every((timelineItem) => timelineItem.isFavorite)
	);
	const firstLoad = $derived($timeline.items.length === 0 && $timeline.status === 'loading');
	const settledEmpty = $derived(
		$timeline.items.length === 0 && $timeline.status === 'ready' && !$timeline.hasMore
	);

	// IntersectionObserver exists in every browser jsdom is not; the
	// manual load-more button is the fallback surface (and the error
	// retry lives there too, because an observer that never
	// re-intersects cannot unstick an error).
	const observerAvailable = typeof IntersectionObserver === 'function';
	let sentinel: HTMLElement | undefined = $state();

	function toggleSelected(id: string, next: boolean): void {
		if (next) {
			selection.add(id);
		} else {
			selection.delete(id);
		}
	}

	function exitSelection(): void {
		selection.clear();
		selectMode = false;
	}

	function handleWindowKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && (selection.size > 0 || selectMode)) {
			exitSelection();
		}
	}

	async function runAction(action: BatchAction): Promise<void> {
		if (batchBusy || selection.size === 0) {
			return;
		}
		batchBusy = true;
		batchErrorKey = '';
		const outcome = await timeline.runBatch(action, [...selection]);
		batchBusy = false;
		if (outcome.ok) {
			// A successful action clears the selection; the explicit
			// select mode persists for the next curation pass.
			selection.clear();
		} else {
			// The failure keeps the selection (a retry affordance) and
			// surfaces the mapped message in the bar.
			batchErrorKey = outcome.errorKey;
		}
	}

	// The add-to-album flow: open the dialog over the current
	// selection with the caller's own virtual albums (the only
	// addable-to kind) fetched fresh -- the pick list is the server's.
	function openAddDialog(): void {
		if (selection.size === 0) {
			return;
		}
		addErrorKey = '';
		addReport = null;
		createdAlbumId = '';
		addOpen = true;
		addAlbums = [];
		addLoading = true;
		void (async () => {
			const outcome = await listAlbums();
			if (!addOpen) {
				return;
			}
			addLoading = false;
			if (outcome.ok) {
				addAlbums = outcome.albums.filter(
					(album) => album.callerRole === 'owner' && album.type === 'virtual'
				);
			} else {
				addErrorKey = outcome.errorKey;
			}
		})();
	}

	// The existing-album path: the honest {added, skipped} report of
	// the write's own answer lands in the dialog.
	async function addToExisting(albumId: string): Promise<void> {
		if (addBusy || selection.size === 0) {
			return;
		}
		addBusy = true;
		addErrorKey = '';
		const outcome = await addAlbumItems(albumId, [...selection]);
		addBusy = false;
		if (!outcome.ok) {
			addErrorKey = outcome.errorKey;
			return;
		}
		addReport = { added: outcome.added, skipped: outcome.skipped };
	}

	// The create-then-add path: create the album (or reuse the one a
	// failed retry already made), then feed it the selection.
	async function addToNew(title: string): Promise<void> {
		if (addBusy || selection.size === 0) {
			return;
		}
		addBusy = true;
		addErrorKey = '';
		let albumId = createdAlbumId;
		if (albumId === '') {
			const created = await createAlbum(title);
			if (!created.ok) {
				addBusy = false;
				addErrorKey = created.errorKey;
				return;
			}
			albumId = created.album.id;
			createdAlbumId = albumId;
		}
		const outcome = await addAlbumItems(albumId, [...selection]);
		addBusy = false;
		if (!outcome.ok) {
			addErrorKey = outcome.errorKey;
			return;
		}
		addReport = { added: outcome.added, skipped: outcome.skipped };
	}

	// The report's done: close the dialog and clear the selection --
	// the add landed (the SSE items-changed echo converges the album
	// surfaces, and the timeline itself only changes when the
	// membership moved a feeding album).
	function finishAdd(): void {
		addOpen = false;
		addReport = null;
		selection.clear();
	}

	// The add-tag flow: open the dialog over the current selection
	// with the caller's vocabulary fetched fresh -- the pick list is
	// the server's. Tag writes are per-user state (B4) and publish
	// nothing, so nothing here converges the wall: the report IS the
	// outcome.
	function openTagDialog(): void {
		if (selection.size === 0) {
			return;
		}
		tagErrorKey = '';
		tagReport = null;
		tagOpen = true;
		tagList = [];
		tagLoading = true;
		void (async () => {
			const outcome = await listTags();
			if (!tagOpen) {
				return;
			}
			tagLoading = false;
			if (outcome.ok) {
				tagList = outcome.tags;
			} else {
				tagErrorKey = outcome.errorKey;
			}
		})();
	}

	// The existing-tag path: the honest {added, skipped} report of
	// the write's own answer lands in the dialog.
	async function addToExistingTag(tagId: string): Promise<void> {
		if (tagBusy || selection.size === 0) {
			return;
		}
		tagBusy = true;
		tagErrorKey = '';
		const outcome = await addTagAssets(tagId, [...selection]);
		tagBusy = false;
		if (!outcome.ok) {
			tagErrorKey = outcome.errorKey;
			return;
		}
		tagReport = outcome.report;
	}

	// The new-tag path: create the tag (idempotently -- a retry
	// converges on the same row), then tag the selection with it.
	async function addToNewTag(name: string): Promise<void> {
		if (tagBusy || selection.size === 0) {
			return;
		}
		tagBusy = true;
		tagErrorKey = '';
		const created = await createTag(name);
		if (!created.ok) {
			tagBusy = false;
			tagErrorKey = created.errorKey;
			return;
		}
		const outcome = await addTagAssets(created.tag.id, [...selection]);
		tagBusy = false;
		if (!outcome.ok) {
			tagErrorKey = outcome.errorKey;
			return;
		}
		tagReport = outcome.report;
	}

	// The tag report's done: close the dialog and clear the
	// selection -- the caller's own tags landed, no wall convergence
	// follows (per-user state, silent by contract).
	function finishTag(): void {
		tagOpen = false;
		tagReport = null;
		selection.clear();
	}

	// The month axis and its scroll wiring (the jump, the active
	// stop, the progress, the drag, the scroll-root bar hiding)
	// moved INTO the wall (the 2026-11-08 pin: the photo rendering
	// and the right-edge rail are one component). This page composes
	// the wall and keeps only its selection overlays.

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// Publish the loaded page items as the viewer's navigation
	// sequence (see $lib/timeline/wall.svelte): the copy outlives
	// this route, so the /a/{id} viewer walks exactly the items this
	// wall rendered.
	$effect(() => {
		publishWallItems($timeline.items);
	});

	$effect(() => {
		void timeline.loadMore();

		const unsubscribeAssets = subscribeAssetEvents((event) => {
			void timeline.applyEvent(event);
		});

		// The album events that can move the union's membership arms:
		// a share granted to the caller or revoked, an album deleted,
		// a shared album's curation. The events carry ids only, so
		// the walk restarts and the fresh page one owns the truth
		// (property patches and creations change nothing here).
		const unsubscribeAlbums = subscribeAlbumEvents((event) => {
			if (
				event.name === 'album.shared' ||
				event.name === 'album.unshared' ||
				event.name === 'album.deleted' ||
				event.name === 'album.items-changed'
			) {
				timeline.reset();
				void timeline.loadMore();
			}
		});

		let observer: IntersectionObserver | undefined;
		if (observerAvailable && sentinel !== undefined) {
			observer = new IntersectionObserver(
				(entries) => {
					if (entries.some((entry) => entry.isIntersecting)) {
						// The client's own guards (in-flight, exhausted)
						// collapse bursts into one fetch.
						void timeline.loadMore();
					}
				},
				{ rootMargin: '600px 0px' }
			);
			observer.observe(sentinel);
		}
		return () => {
			observer?.disconnect();
			unsubscribeAssets();
			unsubscribeAlbums();
		};
	});
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<svelte:head>
	<title>Timeline · lumemor</title>
</svelte:head>

<!-- The reserved month-axis gutter: the wall centers in the width
     LEFT of the rail's own strip, never under it. -->
<div class="timeline__gutter pe-[72px]">
	<section class="timeline px-2" aria-busy={$timeline.status === 'loading'}>
	<!-- No toolbar (R2-11): the timeline's header carries nothing but
	     what the top bar already owns (upload, search). The hidden
	     view's entry moved to the navigation drawer; the selection
	     entry is the tiles' check affordance (hover/keyboard-focus
	     revealed on every tile, the first click enters the selection)
	     -- the ?album= handoff below still arms the select mode for
	     the add-photos flow. -->

	{#if firstLoad}
		<p class="timeline__status my-12 text-center text-text-soft animate-loading-in" role="status">{$_('timeline_loading')}</p>
	{:else if settledEmpty}
		<div class="timeline__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card">
			<h1 class="text-[22px] mb-2">{$_('nav_timeline')}</h1>
			<p class="text-text-soft">{$_('timeline_empty')}</p>
		</div>
	{:else if $timeline.status === 'error' && $timeline.items.length === 0}
		<div class="timeline__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText($timeline.errorKey)}</p>
			<Button variant="primary" onclick={() => void timeline.loadMore()}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
		</div>
	{:else}
		<!-- The unified wall (its month axis is now the wall's own
		     organ -- the 2026-11-08 pin): the page keeps its toolbar
		     and its selection overlays; the grouping, the packed
		     rows, the tiles AND the right-edge rail are the one wall
		     component every view renders. -->
		<Wall
			items={$timeline.items}
			{selectable}
			isSelected={(id) => selection.has(id)}
			onselect={toggleSelected}
		/>
	{/if}

	{#if $timeline.status === 'error' && $timeline.items.length > 0}
		<div class="timeline__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText($timeline.errorKey)}</p>
			<Button variant="primary" onclick={() => void timeline.loadMore()}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
		</div>
	{/if}

	{#if !observerAvailable && $timeline.hasMore && $timeline.status === 'ready'}
		<div class="timeline__pager flex justify-center my-5">
			<Button variant="ghost" onclick={() => void timeline.loadMore()}><ChevronsDown size={20} aria-hidden="true" />{$_('timeline_load_more')}</Button>
		</div>
	{/if}

	<div class="timeline__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
	</section>
</div>

<BatchBar
	count={selectedItems.length}
	allFavorite={allSelectedFavorite}
	errorKey={batchErrorKey}
	busy={batchBusy}
	onaddtoalbum={openAddDialog}
	onaddtag={openTagDialog}
	onaction={runAction}
	onclear={exitSelection}
/>
<AddToAlbumDialog
	bind:open={addOpen}
	albums={addAlbums}
	initialAlbumId={preselectAlbumId}
	loading={addLoading}
	busy={addBusy}
	errorKey={addErrorKey}
	report={addReport}
	onpick={(albumId) => void addToExisting(albumId)}
	oncreate={(title) => void addToNew(title)}
	ondone={finishAdd}
/>
<AddTagDialog
	bind:open={tagOpen}
	tags={tagList}
	loading={tagLoading}
	busy={tagBusy}
	errorKey={tagErrorKey}
	report={tagReport}
	onpick={(tagId) => void addToExistingTag(tagId)}
	oncreate={(name) => void addToNewTag(name)}
	ondone={finishTag}
/>
