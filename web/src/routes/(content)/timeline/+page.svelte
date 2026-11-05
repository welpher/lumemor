<script lang="ts">
	// The timeline wall, the app's home surface: keyset pages of
	// tiles stream in through the pagination client (an
	// IntersectionObserver sentinel over a 600px prefetch margin, the
	// client's in-flight guard collapsing every burst into one
	// fetch), month and day groups carry sticky headers, the right
	// rail scrubs the loaded months, and selection turns the wall
	// into a curation surface with the floating batch bar.
	//
	// THE ADD-TO-ALBUM FLOW (M3c Task 5): the batch bar's dialog adds
	// the selection to one of the caller's own virtual albums, or
	// creates the album first -- the honest {added, skipped} report
	// of the write's own answer lands in the dialog. The ?album=
	// query parameter (the album detail's add-photos handoff) arms
	// the select mode and preselects that album in the dialog.
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
	// content-visibility with its engine-computed height (see
	// MonthGroups); full virtualization stays an M3 option if the
	// wall outgrows it.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { page } from '$app/state';
	import { SvelteSet } from 'svelte/reactivity';
	import { createTimelineClient, type BatchAction } from '$lib/api/timeline';
	import { addAlbumItems, createAlbum, listAlbums, type AlbumSummary } from '$lib/api/albums';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAlbumEvents, subscribeAssetEvents } from '$lib/events';
	import { groupTimelineItems } from '$lib/timeline/grouping';
	import { publishWallItems } from '$lib/timeline/wall.svelte';
	import MonthGroups from '$lib/timeline/MonthGroups.svelte';
	import MonthScrubber from '$lib/timeline/MonthScrubber.svelte';
	import BatchBar from '$lib/timeline/BatchBar.svelte';
	import AddToAlbumDialog from '$lib/albums/AddToAlbumDialog.svelte';
	import Button from '$lib/components/Button.svelte';

	const timeline = createTimelineClient();

	// The selection is a reactive id set: checkbox and tile clicks
	// toggle it, Esc and the bar's clear drop it, and the select mode
	// (the touch-friendly entry) keeps the checkboxes revealed.
	const selection = new SvelteSet<string>();
	let selectMode = $state(false);
	let batchErrorKey = $state('');
	let batchBusy = $state(false);
	let activeMonthKey = $state<string | null>(null);

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

	// The ?album= handoff: the album detail's add-photos action sends
	// the caller here to pick photos for that album -- the select
	// mode arms and the dialog preselects it.
	const preselectAlbumId = $derived(page.url.searchParams.get('album') ?? '');

	$effect(() => {
		if (preselectAlbumId !== '') {
			selectMode = true;
		}
	});

	const groups = $derived(groupTimelineItems($timeline.items, $locale ?? 'en'));
	const months = $derived(groups.map((group) => ({ key: group.key, label: group.label })));
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

	/** Scroll one month group into view (the scrubber jump target). */
	function jumpToMonth(key: string): void {
		const element = document.querySelector<HTMLElement>(`[data-month-key="${key}"]`);
		if (element !== null && typeof element.scrollIntoView === 'function') {
			element.scrollIntoView({ block: 'start', behavior: 'smooth' });
		}
	}

	let scrollScheduled = false;

	/** Track which month the viewport reads, for the scrubber's stop. */
	function updateActiveMonth(): void {
		const sections = Array.from(document.querySelectorAll<HTMLElement>('[data-month-key]'));
		if (sections.length === 0) {
			return;
		}
		const first = sections[0]?.getBoundingClientRect();
		// A layout-less host (tests) answers zero rects: keep no stop.
		if (first === undefined || (first.top === 0 && first.height === 0)) {
			return;
		}
		const anchor = 100;
		let current: string | null = null;
		for (const section of sections) {
			if (section.getBoundingClientRect().top <= anchor) {
				current = section.dataset.monthKey ?? null;
			} else {
				break;
			}
		}
		activeMonthKey = current;
	}

	function handleWindowScroll(): void {
		if (scrollScheduled) {
			return;
		}
		scrollScheduled = true;
		requestAnimationFrame(() => {
			scrollScheduled = false;
			updateActiveMonth();
		});
	}

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

<svelte:window onkeydown={handleWindowKeydown} onscroll={handleWindowScroll} />

<svelte:head>
	<title>Timeline · lumemor</title>
</svelte:head>

<section class="timeline max-w-[1200px] mx-auto" aria-busy={$timeline.status === 'loading'}>
	<!-- The one toolbar of the wall: the hidden view's entry point
	     (the IA's timeline-header entry, a lightweight link while the
	     dropdown pattern it names waits for the settings surface of
	     M3) and, once tiles exist, the selection-mode toggle. -->
	<div class="timeline__toolbar flex justify-end items-center gap-3 mb-2">
		<a
			class="timeline__hidden-link mr-auto px-2.5 py-1.5 font-body font-semibold text-[14px] leading-[1.4] text-text-soft no-underline rounded-control transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/hidden"
		>{$_('hidden_title')}</a>
		{#if $timeline.items.length > 0}
			<Button
				variant="ghost"
				aria-pressed={selectMode}
				onclick={() => (selectMode = !selectMode)}
			>
				{selectMode ? $_('action_done') : $_('action_select')}
			</Button>
		{/if}
	</div>

	{#if firstLoad}
		<p class="timeline__status my-12 text-center text-text-soft" role="status">{$_('timeline_loading')}</p>
	{:else if settledEmpty}
		<div class="timeline__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card">
			<h1 class="text-[22px] mb-2">{$_('nav_timeline')}</h1>
			<p class="text-text-soft">{$_('timeline_empty')}</p>
		</div>
	{:else if $timeline.status === 'error' && $timeline.items.length === 0}
		<div class="timeline__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText($timeline.errorKey)}</p>
			<Button variant="primary" onclick={() => void timeline.loadMore()}>{$_('action_retry')}</Button>
		</div>
	{:else}
		<!-- The month-grouped justified wall (the shared renderer): the
		     page keeps its toolbar, its scrubber and its scroll
		     tracking; the groups, the sticky headers and the packed
		     rows are the one wall component every view renders. -->
		<MonthGroups {groups} isSelected={(id) => selection.has(id)} {selectable} onselect={toggleSelected} />
	{/if}

	{#if $timeline.status === 'error' && $timeline.items.length > 0}
		<div class="timeline__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText($timeline.errorKey)}</p>
			<Button variant="primary" onclick={() => void timeline.loadMore()}>{$_('action_retry')}</Button>
		</div>
	{/if}

	{#if !observerAvailable && $timeline.hasMore && $timeline.status === 'ready'}
		<div class="timeline__pager flex justify-center my-5">
			<Button variant="ghost" onclick={() => void timeline.loadMore()}>{$_('timeline_load_more')}</Button>
		</div>
	{/if}

	<div class="timeline__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
</section>

<MonthScrubber {months} activeKey={activeMonthKey} onjump={jumpToMonth} />
<BatchBar
	count={selectedItems.length}
	allFavorite={allSelectedFavorite}
	errorKey={batchErrorKey}
	busy={batchBusy}
	onaddtoalbum={openAddDialog}
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
