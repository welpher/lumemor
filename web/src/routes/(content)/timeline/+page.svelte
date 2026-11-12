<script lang="ts">
	// The timeline wall, the app's home surface: keyset pages of
	// tiles stream in through the pagination client (an
	// IntersectionObserver sentinel over a 600px prefetch margin, the
	// client's in-flight guard collapsing every burst into one
	// fetch), month and day groups carry sticky headers, the right
	// rail scrubs the loaded months, and selection turns the wall
	// into a curation surface with the floating batch bar. The
	// month axis AND the selection flow are the wall's own organs
	//, so they ride the
	// shared Wall component. Everything around the wall here -- the
	// data walk, the dialogs -- is the page's own concern: the page
	// runs the batch writes the wall awaits and owns the add-tag
	// dialog the wall's opener feeds with the ids captured at open.
	//
	// THE ADD-TO-ALBUM FLOW IS GONE: an album is a
	// source picked at creation, not a container photos get curated
	// into -- the bar's album button, its dialog and the ?album=
	// handoff all died together (the tag flow stays: tags are
	// vocabulary, not containers).
	//
	// THE ADD-TAG FLOW: the bar's dialog tags the
	// captured selection with one of the caller's existing tags, or
	// creates the tag first (idempotently) -- the same honest
	// report, the same `done` continuation. Tag writes are per-user
	// state and publish nothing, so the report is the outcome and
	// the wall never converges.
	//
	// THE ADD-TO-MEMORY FLOW: the bar's picker adds the captured
	// selection to one of the caller's memories, or creates the
	// memory first -- the same captured-ids + done-continuation
	// hand-over. The dialog owns the flow's writes; the wall never
	// converges on the pass (adding to a memory changes no photo's
	// state -- the memory never feeds this caller's timeline, the
	// report IS the outcome).
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
	// Wall component); full offscreen-recycling windowing stays an
	// option if the
	// wall outgrows it.
	import { _ } from 'svelte-i18n';
	import { createTimelineClient, type BatchAction, type BatchOutcome } from '$lib/api/timeline';
	import { addTagAssets, createTag, listTags, type TagSummary } from '$lib/api/tags';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAlbumEvents, subscribeAssetEvents } from '$lib/events';
	import { publishWallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import AddTagDialog from '$lib/tags/AddTagDialog.svelte';
	import AddToMemoryDialog from '$lib/memories/AddToMemoryDialog.svelte';
	import Button from '$lib/components/Button.svelte';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	const timeline = createTimelineClient();

	// The add-tag flow's state (the bar's dialog): the vocabulary
	// fetched on open, the honest report of the last landed add, and
	// the captured-ids + done-continuation pair the wall's opener
	// hands over. The create is idempotent (a name the caller owns
	// converges on the existing row), so no created-id bookkeeping.
	let tagIds: readonly string[] = [];
	let tagDone: (() => void) | undefined;
	let tagOpen = $state(false);
	let tagBusy = $state(false);
	let tagErrorKey = $state('');
	let tagList = $state<TagSummary[]>([]);
	let tagLoading = $state(false);
	let tagReport = $state<{ added: number; skipped: number } | null>(null);

	// The add-to-memory flow's state: the captured-ids +
	// done-continuation pair the wall's opener hands over -- the
	// dialog owns the flow's own state and writes.
	let memoryIds = $state<readonly string[]>([]);
	let memoryDone: (() => void) | undefined;
	let memoryOpen = $state(false);

	const firstLoad = $derived($timeline.items.length === 0 && $timeline.status === 'loading');
	const settledEmpty = $derived($timeline.items.length === 0 && $timeline.status === 'ready' && !$timeline.hasMore);

	// IntersectionObserver exists in every browser jsdom is not; the
	// manual load-more button is the fallback surface (and the error
	// retry lives there too, because an observer that never
	// re-intersects cannot unstick an error).
	const observerAvailable = typeof IntersectionObserver === 'function';
	let sentinel: HTMLElement | undefined = $state();

	// The wall's batch runner: this page's walk owns the write (the
	// client's optimistic state and rollback); the wall owns the
	// busy/error/clear choreography around the promise it awaits
	// here.
	function runBatch(action: BatchAction, ids: readonly string[]): Promise<BatchOutcome> {
		return timeline.runBatch(action, [...ids]);
	}

	// The add-tag flow: open the dialog over the wall's captured ids
	// with the caller's vocabulary fetched fresh -- the pick list is
	// the server's. Tag writes are per-user state and publish
	// nothing, so nothing here converges the wall: the report IS the
	// outcome.
	function openTagDialog(ids: readonly string[], done: () => void): void {
		tagIds = ids;
		tagDone = done;
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
		if (tagBusy || tagIds.length === 0) {
			return;
		}
		tagBusy = true;
		tagErrorKey = '';
		const outcome = await addTagAssets(tagId, [...tagIds]);
		tagBusy = false;
		if (!outcome.ok) {
			tagErrorKey = outcome.errorKey;
			return;
		}
		tagReport = outcome.report;
	}

	// The new-tag path: create the tag (idempotently -- a retry
	// converges on the same row), then tag the captured ids with it.
	async function addToNewTag(name: string): Promise<void> {
		if (tagBusy || tagIds.length === 0) {
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
		const outcome = await addTagAssets(created.tag.id, [...tagIds]);
		tagBusy = false;
		if (!outcome.ok) {
			tagErrorKey = outcome.errorKey;
			return;
		}
		tagReport = outcome.report;
	}

	// The tag report's done: close the dialog and fire the wall's
	// done continuation -- the caller's own tags landed, no wall
	// convergence follows (per-user state, silent by contract), and
	// the wall clears its selection (the success arm).
	function finishTag(): void {
		tagOpen = false;
		tagReport = null;
		tagDone?.();
		tagDone = undefined;
	}

	// The memory picker's opener: the wall's captured ids and done
	// continuation ride the dialog (the modal-pinned selection
	// discipline). The pass's landing fires the success arm -- the
	// wall clears its selection; no convergence follows.
	function openMemoryDialog(ids: readonly string[], done: () => void): void {
		memoryIds = ids;
		memoryDone = done;
		memoryOpen = true;
	}

	function finishMemory(): void {
		memoryDone?.();
		memoryDone = undefined;
	}

	// The month axis, the selection flow and the batch bar (the
	// jump, the active stop, the drag, the checks, the Esc exit, the
	// busy/error choreography, the bar itself) all moved INTO the
	// wall (the photo rendering, the right-edge
	// rail and the selection are one component). This page composes
	// the wall, runs its batch writes and owns the dialogs.

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

<svelte:head>
	<title>Timeline · lumemor</title>
</svelte:head>

<!-- The reserved month-axis gutter: the wall centers in the width
     LEFT of the rail's own strip, never under it. -->
<div class="timeline__gutter pe-[72px]">
	<section class="timeline px-2" aria-busy={$timeline.status === 'loading'}>
		<!-- No toolbar: the timeline's header carries nothing but
	     what the top bar already owns (upload, search). The hidden
	     view's entry moved to the navigation drawer; the selection
	     entry is the tiles' check affordance (hover/keyboard-focus
	     revealed on every tile, the first click enters the
	     selection). -->

		{#if firstLoad}
			<p class="timeline__status my-12 text-center text-text-soft animate-loading-in" role="status">
				{$_('timeline_loading')}
			</p>
		{:else if settledEmpty}
			<div
				class="timeline__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card"
			>
				<h1 class="text-[22px] mb-2">{$_('nav_timeline')}</h1>
				<p class="text-text-soft">{$_('timeline_empty')}</p>
			</div>
		{:else if $timeline.status === 'error' && $timeline.items.length === 0}
			<div
				class="timeline__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
				role="alert"
			>
				<p>{errorText($timeline.errorKey)}</p>
				<Button variant="primary" onclick={() => void timeline.loadMore()}
					><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
				>
			</div>
		{:else}
			<!-- The unified wall (its month axis AND its selection flow are
		     the wall's own organs): the page
		     keeps its data walk, its batch runs and its dialogs; the
		     grouping, the packed rows, the tiles, the right-edge rail,
		     the selection state and the batch bar are the one wall
		     component every curatable view renders. The dialog opener
		     carries the ids the wall captured. -->
			<Wall items={$timeline.items} onbatch={runBatch} onaddtag={openTagDialog} onaddmemory={openMemoryDialog} />
		{/if}

		{#if $timeline.status === 'error' && $timeline.items.length > 0}
			<div
				class="timeline__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
				role="alert"
			>
				<p>{errorText($timeline.errorKey)}</p>
				<Button variant="primary" onclick={() => void timeline.loadMore()}
					><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
				>
			</div>
		{/if}

		{#if !observerAvailable && $timeline.hasMore && $timeline.status === 'ready'}
			<div class="timeline__pager flex justify-center my-5">
				<Button variant="ghost" onclick={() => void timeline.loadMore()}
					><ChevronsDown size={20} aria-hidden="true" />{$_('timeline_load_more')}</Button
				>
			</div>
		{/if}

		<div class="timeline__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
	</section>
</div>

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

<AddToMemoryDialog bind:open={memoryOpen} ids={memoryIds} onpass={finishMemory} />
