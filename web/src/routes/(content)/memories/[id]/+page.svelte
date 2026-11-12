<script lang="ts">
	// The memory detail: the reference-set's own face. The contextual
	// bar REPLACES the shell's global topbar while the route is
	// active (the content shell hides its bar here, the album
	// detail's rule) -- back, the memory's title, the honest count,
	// and the role-split arms:
	//
	// - THE OWNER holds the edit dialog (title + audience alone: a
	//   memory has no source row and no autonomy block -- no
	//   deletability switch, because its delete never moves files; no
	//   timeline exclusion, because it never feeds the timeline by
	//   construction), the cover picker (the existing machinery over
	//   the rendered members) and the delete behind its
	//   semantics-stating confirm (the LIGHT CASCADE: the memory's
	//   references and its audience die, the photos are untouched).
	//   The owner's wall curates a REMOVE-ONLY selection: the
	//   membership removal alone rides the bar (the minus action),
	//   the references die while the photos stay -- a direct write
	//   with the success-clears/failure-keeps choreography.
	// - THE MEMBER holds nothing: the owner attribution and the
	//   read-only badge alone, a pure-navigation wall (no check ever
	//   renders). Read-only is VISUAL: the server refuses the
	//   member's every write on the memory.
	//
	// NO rescan and NO remove-flow anywhere: a memory owns no source
	// to walk and no files to un-ingest -- the memory's whole shape
	// is the reference set.
	//
	// The wall is the month-grouped walk over the timeline
	// endpoint's album narrow (the memory's live members in the
	// timeline's exact shape and keyset order, month axis included);
	// the album.* SSE events for THIS memory refetch the metadata
	// and restart the walk on membership changes.
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { _ } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import {
		deleteAlbum,
		fetchAlbumDetail,
		listAlbums,
		patchAlbum,
		removeAlbumItems,
		updateAlbumAudience,
		type AlbumDetail
	} from '$lib/api/albums';
	import { createTimelineClient } from '$lib/api/timeline';
	import { subscribeAlbumEvents, subscribeAssetEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { publishWallItems, wallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import Button from '$lib/components/Button.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import AlbumPropertiesDialog from '$lib/albums/AlbumPropertiesDialog.svelte';
	import CoverPickerDialog from '$lib/albums/CoverPickerDialog.svelte';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import PenLine from '@lucide/svelte/icons/pen-line';
	import Images from '@lucide/svelte/icons/images';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import X from '@lucide/svelte/icons/x';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';

	const memoryId = $derived(page.params.id ?? '');

	type DetailStatus = 'idle' | 'loading' | 'ready' | 'error';

	let detail = $state<AlbumDetail | null>(null);
	let status = $state<DetailStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

	// The member's attribution: the detail read carries no owner face
	// (the contract keeps the owner on the wall cards alone), so the
	// viewer's bar reads it off the matching shared card -- one wall
	// fetch when the detail answers viewer (the owner is the
	// memory's creator, an invariant; a failed probe leaves the
	// badge alone, honestly).
	let ownerEmail = $state('');

	// One write at a time across every action (the dialogs and the
	// wall's removal share the flag).
	let writeBusy = $state(false);

	// The edit dialog (title + audience).
	let editOpen = $state(false);
	let editErrorKey = $state('');

	// The cover picker.
	let coverOpen = $state(false);
	let coverErrorKey = $state('');

	// The delete confirm: the light cascade's own statement.
	let deleteOpen = $state(false);
	let deleteErrorKey = $state('');

	// THE WALL WALK (the page's own data concern, the album detail's
	// shape): the timeline client over the album narrow, constructed
	// ONCE per mount, the wallResetKey convergence (membership
	// changes bump it and the walk restarts from page one), the SSE
	// asset convergence, the sentinel and the wallItems publication.
	const wall = untrack(() =>
		createTimelineClient({ path: '/api/v1/assets/timeline', view: 'timeline', albumId: memoryId })
	);
	let wallResetKey = $state(0);

	const owner = $derived(detail?.callerRole === 'owner');

	// The wall's own states (the walk's honest faces).
	const wallFirstLoad = $derived($wall.items.length === 0 && $wall.status === 'loading');
	const wallSettledEmpty = $derived($wall.items.length === 0 && $wall.status === 'ready' && !$wall.hasMore);

	// IntersectionObserver exists in every browser jsdom is not; the
	// manual load-more button is the fallback surface (and the error
	// retry lives there too, because an observer that never
	// re-intersects cannot unstick an error).
	const observerAvailable = typeof IntersectionObserver === 'function';
	let sentinel: HTMLElement | undefined = $state();

	// The fetch and the SSE subscription are keyed on the route id; a
	// superseded run (the id changed, or the route left, while the
	// fetch was in flight) never applies.
	let fetchCurrent = true;

	/** Apply one detail outcome; the superseded runs never call this. */
	function apply(outcome: Awaited<ReturnType<typeof fetchAlbumDetail>>): void {
		if (outcome.ok) {
			detail = outcome.detail;
			errorKey = '';
			status = 'ready';
			if (outcome.detail.callerRole === 'viewer' && ownerEmail === '') {
				void convergeOwnerEmail(outcome.detail.id);
			}
			return;
		}
		detail = null;
		status = 'error';
		errorKey = outcome.errorKey;
	}

	/** Read the viewer's owner attribution off the matching wall card. */
	async function convergeOwnerEmail(id: string): Promise<void> {
		const outcome = await listAlbums();
		if (!outcome.ok) {
			return;
		}
		ownerEmail = outcome.albums.find((card) => card.id === id && card.callerRole === 'viewer')?.owner?.email ?? '';
	}

	/**
	 * The convergence refetch: keep the rendered detail while the
	 * fresh answer is in flight, replace it when it lands.
	 */
	async function convergeDetail(): Promise<void> {
		const id = untrack(() => memoryId);
		if (id === '' || !fetchCurrent) {
			return;
		}
		const outcome = await fetchAlbumDetail(id);
		if (fetchCurrent) {
			apply(outcome);
		}
	}

	$effect(() => {
		const id = memoryId;
		if (id === '') {
			return;
		}
		fetchCurrent = true;
		status = 'loading';
		detail = null;
		errorKey = '';
		void (async () => {
			const outcome = await fetchAlbumDetail(id);
			if (fetchCurrent) {
				apply(outcome);
			}
		})();

		const unsubscribe = subscribeAlbumEvents((event) => {
			if (event.albumId !== untrack(() => memoryId)) {
				return;
			}
			if (event.name === 'album.items-changed') {
				// The membership changed under the grid: restart the
				// walk -- the fresh page answer owns the tiles.
				wallResetKey += 1;
			}
			void convergeDetail();
		});
		return () => {
			fetchCurrent = false;
			unsubscribe();
		};
	});

	// The members the cover picker offers: the wall's published items
	// -- exactly the grid the page renders.
	const members = $derived(wallItems());

	/** Adopt a write response's fields into the detail. */
	function applyWriteFields(next: { title: string; itemCount: number }): void {
		if (detail !== null) {
			detail = {
				...detail,
				title: next.title,
				itemCount: next.itemCount
			};
		}
	}

	// The edit write: the page diffs the draft against the LIVE
	// detail and writes only the changed halves (a same-value title
	// PATCH would still publish its event, a same-set audience PUT
	// is the contract's silent no-op, so a no-change save writes
	// nothing at all). The final detail refetch converges the faces
	// the patch card never carries (the grantee list).
	async function submitEdit(draft: {
		title: string;
		audience: readonly { userId: string; email: string }[];
	}): Promise<void> {
		const id = untrack(() => memoryId);
		if (writeBusy || detail === null || id === '') {
			return;
		}
		const wantsTitle = draft.title !== detail.title;
		const currentIds = new Set((detail.shares ?? []).map((share) => share.userId));
		const nextIds = new Set(draft.audience.map((member) => member.userId));
		const wantsAudience =
			nextIds.size !== currentIds.size || [...nextIds].some((userId) => !currentIds.has(userId));
		if (!wantsTitle && !wantsAudience) {
			editOpen = false;
			return;
		}
		writeBusy = true;
		editErrorKey = '';
		if (wantsTitle) {
			const outcome = await patchAlbum(id, { title: draft.title });
			if (!outcome.ok) {
				writeBusy = false;
				editErrorKey = outcome.errorKey;
				return;
			}
			applyWriteFields(outcome.album);
		}
		if (wantsAudience) {
			const outcome = await updateAlbumAudience(id, [...nextIds]);
			if (!outcome.ok) {
				writeBusy = false;
				editErrorKey = outcome.errorKey;
				return;
			}
		}
		writeBusy = false;
		await convergeDetail();
		editOpen = false;
	}

	// The cover write: the 200 proves the pin landed, so the detail
	// adopts the picked id (the SSE echo refetches anyway).
	async function submitCover(assetId: string): Promise<void> {
		const id = untrack(() => memoryId);
		if (writeBusy || detail === null || id === '') {
			return;
		}
		writeBusy = true;
		coverErrorKey = '';
		const outcome = await patchAlbum(id, { coverAssetId: assetId });
		writeBusy = false;
		if (!outcome.ok) {
			coverErrorKey = outcome.errorKey;
			return;
		}
		if (detail !== null) {
			detail = { ...detail, coverAssetId: assetId };
		}
		coverOpen = false;
	}

	// The delete write (the light cascade): success leaves for the
	// memory wall (a deep link may have no history to back through).
	async function confirmDelete(): Promise<void> {
		const id = untrack(() => memoryId);
		if (writeBusy || detail === null || id === '') {
			return;
		}
		writeBusy = true;
		deleteErrorKey = '';
		const outcome = await deleteAlbum(id);
		writeBusy = false;
		if (!outcome.ok) {
			deleteErrorKey = outcome.errorKey;
			return;
		}
		deleteOpen = false;
		void goto('/memories');
	}

	// The wall's removal runner: the owner's remove-only curation.
	// The write is the membership tidy-up (the references die, the
	// photos stay); the wall owns the busy/selection choreography
	// around the promise it awaits here, and the membership's own
	// convergence rides the event the landing write publishes (the
	// walk restarts, the detail refetches).
	async function runRemoveFromMemory(
		ids: readonly string[]
	): Promise<{ ok: true } | { ok: false; errorKey: string }> {
		const id = untrack(() => memoryId);
		if (id === '') {
			return { ok: false, errorKey: 'internal' };
		}
		return removeAlbumItems(id, [...ids]);
	}

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// Publish the loaded page items as the viewer's navigation
	// sequence: the copy outlives this route, so the /a/{id} viewer
	// walks exactly the items this wall rendered.
	$effect(() => {
		publishWallItems($wall.items);
	});

	// The walk itself: the first load and every wallResetKey change
	// (the membership convergence). A reset while a page load is in
	// flight discards the stale response (the client's generation
	// guard).
	$effect(() => {
		void wallResetKey;
		untrack(() => {
			wall.reset();
			void wall.loadMore();
		});
	});

	// The live surfaces: the SSE subscription and the scroll sentinel
	// observer live for the page's lifetime.
	$effect(() => {
		const unsubscribe = subscribeAssetEvents((event) => {
			void wall.applyEvent(event);
		});

		let observer: IntersectionObserver | undefined;
		if (observerAvailable && sentinel !== undefined) {
			observer = new IntersectionObserver(
				(entries) => {
					if (entries.some((entry) => entry.isIntersecting)) {
						// The client's own guards (in-flight, exhausted)
						// collapse bursts into one fetch.
						void wall.loadMore();
					}
				},
				{ rootMargin: '600px 0px' }
			);
			observer.observe(sentinel);
		}
		return () => {
			observer?.disconnect();
			unsubscribe();
		};
	});
</script>

<svelte:head>
	<title>{detail?.title ?? $_('memories_title')} · lumemor</title>
</svelte:head>

<!-- The page root pins the wall's sticky month headers below the
     contextual bar: the bar is py-2 (16) around an h-9 row (36)
     plus its 1px border -- 53px. -->
<section class="memory-detail" style:--wall-sticky-offset="53px">
	<!-- THE CONTEXTUAL BAR: back + the title + the count, then the
	     role-split arms -- the owner's icon actions (the edit dialog,
	     the cover picker, the delete behind its confirm), the
	     member's attribution and read-only badge with NO action at
	     all. -->
	<header
		class="memory-bar sticky top-0 z-(--z-sticky-bar) flex items-center gap-2 px-4 py-2 bg-card border-b border-border"
	>
		<a
			class="memory-bar__back inline-flex items-center justify-center w-9 h-9 shrink-0 rounded-control bg-transparent text-text-soft no-underline cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/memories"
			aria-label={$_('memories_back')}
		>
			<ChevronLeft size={20} aria-hidden="true" />
		</a>

		{#if detail !== null}
			<h1
				class="memory-bar__title m-0 min-w-0 truncate font-display font-bold text-[20px] leading-[1.3] text-text"
			>
				{detail.title}
			</h1>
			<span
				class="memory-bar__count hidden sm:inline font-body font-normal text-[13px] leading-[1.4] text-text-soft whitespace-nowrap tabular-nums"
				>{$_('memories_photo_count', { values: { count: detail.itemCount } })}</span
			>
		{/if}

		<div class="memory-bar__actions ml-auto flex items-center gap-1">
			{#if detail !== null && owner}
				<IconButton
					disabled={writeBusy}
					aria-label={$_('memories_edit_title')}
					onclick={() => {
						editErrorKey = '';
						editOpen = true;
					}}
				>
					<PenLine size={20} aria-hidden="true" />
				</IconButton>
				<IconButton
					disabled={writeBusy}
					aria-label={$_('memories_action_cover')}
					onclick={() => {
						coverErrorKey = '';
						coverOpen = true;
					}}
				>
					<Images size={20} aria-hidden="true" />
				</IconButton>
				<IconButton
					variant="danger"
					disabled={writeBusy}
					aria-label={$_('memories_action_delete')}
					onclick={() => {
						deleteErrorKey = '';
						deleteOpen = true;
					}}
				>
					<Trash2 size={20} aria-hidden="true" />
				</IconButton>
			{:else if detail !== null && !owner}
				<!-- The member's arm: the honest owner attribution and the
				     read-only badge alone -- no action, no selection, the
				     server refuses the member's every write anyway. -->
				{#if ownerEmail !== ''}
					<span
						class="memory-bar__owner hidden sm:inline font-body font-normal text-[13px] leading-[1.4] text-text-soft truncate"
						>{$_('albums_shared_by', { values: { email: ownerEmail } })}</span
					>
				{/if}
				<span
					class="memory-bar__badge inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
				>
					{$_('memories_shared_badge')}
				</span>
			{/if}
		</div>
	</header>

	{#if status === 'loading'}
		<p class="memory-detail__loading my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('memories_loading')}
		</p>
	{:else if status === 'error'}
		<div
			class="memory-detail__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void convergeDetail()}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{:else if detail !== null}
		<!-- THE WALL (the month-grouped walk over the album narrow):
		     the owner's wall curates the REMOVE-ONLY selection (the
		     minus action alone -- the references die, the photos
		     stay); the member's wall passes no runner, so the tiles
		     are pure navigation and no batch bar ever rises. -->
		<section class="memory-detail__wall px-2" aria-busy={$wall.status === 'loading'}>
			{#if wallFirstLoad}
				<p class="memory-detail__wall-status my-12 text-center text-text-soft animate-loading-in" role="status">
					{$_('memories_loading')}
				</p>
			{:else if wallSettledEmpty}
				<div
					class="memory-detail__wall-empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card"
				>
					<h2 class="text-[22px] mb-2">{detail.title}</h2>
					<p class="text-text-soft">{$_('memories_detail_empty')}</p>
				</div>
			{:else if $wall.status === 'error' && $wall.items.length === 0}
				<div
					class="memory-detail__wall-failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
					role="alert"
				>
					<p>{errorText($wall.errorKey)}</p>
					<Button variant="primary" onclick={() => void wall.loadMore()}
						><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
					>
				</div>
			{:else}
				<Wall items={$wall.items} actions={[]} onremovememory={owner ? runRemoveFromMemory : undefined} />
			{/if}

			{#if $wall.status === 'error' && $wall.items.length > 0}
				<div
					class="memory-detail__wall-failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
					role="alert"
				>
					<p>{errorText($wall.errorKey)}</p>
					<Button variant="primary" onclick={() => void wall.loadMore()}
						><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
					>
				</div>
			{/if}

			{#if !observerAvailable && $wall.hasMore && $wall.status === 'ready'}
				<div class="memory-detail__wall-pager flex justify-center my-5">
					<Button variant="ghost" onclick={() => void wall.loadMore()}>
						<ChevronsDown size={20} aria-hidden="true" />
						{$_('timeline_load_more')}
					</Button>
				</div>
			{/if}

			<div class="memory-detail__wall-sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
		</section>
	{/if}
</section>

{#if detail !== null}
	<!-- The edit dialog: the title and the audience alone -- no
	     source row, no autonomy block (a memory owns neither). -->
	<AlbumPropertiesDialog
		bind:open={editOpen}
		titleKey="memories_edit_title"
		submitKey="memories_edit_submit"
		initialTitle={detail.title}
		shares={detail.shares ?? []}
		busy={writeBusy}
		errorKey={editErrorKey}
		onsubmit={(draft) => void submitEdit(draft)}
	/>

	<CoverPickerDialog
		bind:open={coverOpen}
		items={members}
		currentCoverId={detail.coverAssetId}
		busy={writeBusy}
		errorKey={coverErrorKey}
		onsubmit={(assetId) => void submitCover(assetId)}
	/>
{/if}

<!-- The delete confirm: the light cascade's own statement -- the
     memory and its references and its audience die, the photos
     themselves stay. -->
<Modal bind:open={deleteOpen} title={$_('memories_delete_title')}>
	<p class="memory-detail__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
		{$_('albums_delete_body', { values: { title: detail?.title ?? '' } })}
	</p>
	{#if deleteErrorKey !== ''}
		<p
			class="memory-detail__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(deleteErrorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={writeBusy} onclick={() => (deleteOpen = false)}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="danger" disabled={writeBusy} onclick={() => void confirmDelete()}>
			<Trash2 size={20} aria-hidden="true" />
			{$_('memories_delete_confirm')}
		</Button>
	{/snippet}
</Modal>
