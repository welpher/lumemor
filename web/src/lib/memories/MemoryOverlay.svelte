<script lang="ts">
	// The memory playback overlay: the reference set's own face, opened
	// OVER the memories page instead of routed to. A memory is not a
	// destination to browse -- it is something to put on -- so the
	// overlay is an opaque night stage, one photo at a time, with the
	// grid as its management face (multi-select, remove).
	//
	// TWO FACES, ONE BAR: the segmented switch at the bar's right end
	// flips the PLAY stage (a 3:2 contain frame, the ‹ › flanks, the
	// N / total row) and the GRID (square thumbnails at a 150px floor,
	// selection always armed, the batch bar carrying the removal).
	// The removal ends references, never photos: the copy names the
	// act as leaving the memory, and the photos stay where they live.
	//
	// THE PLAYBACK DISCIPLINE (the design's pin): opening starts the
	// four-second advance immediately; ANY manual action stops on the
	// current photo -- an arrow, a step button, the space key, the
	// switch to the grid, Escape, the tab going to the background
	// (visibilitychange, so a hidden tab never spins); resuming is
	// only ever the user's Play or Space. The advance stops at the
	// last photo rather than looping (no invented return to the
	// start), and switching back from the grid resumes the automatic
	// advance -- the spec's one exception, because choosing to watch
	// again IS the user's start.
	//
	// THE REACHABILITY RULE: the memories wall is the authority on
	// what this caller may open (it lists every memory the caller owns
	// or is shared), so an id the settled wall does not carry answers
	// the SAME dead-end card every unknown, foreign or withdrawn id
	// does -- no title, no count, no cover, and the bar keeps its way
	// back alone (the in-view variant of the one dead end). The detail
	// read's own 404 lands there too (defense in depth: the wall and
	// the read agree on the contract's byte-identical answer).
	//
	// The owner's bar carries the memory's whole action set (edit,
	// cover, delete); a member's bar carries the attribution, the
	// read-only badge and the one writable thing a shared memory
	// gives them (the caller's own timeline preference). Read-only is
	// visual: the server refuses the member's every other write.
	import { get } from 'svelte/store';
	import { _ } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import {
		deleteAlbum,
		fetchAlbumDetail,
		patchAlbum,
		removeAlbumItems,
		setAlbumTimelinePreference,
		updateAlbumAudience,
		type AlbumDetail,
		type AlbumSummary
	} from '$lib/api/albums';
	import { createTimelineClient } from '$lib/api/timeline';
	import { subscribeAlbumEvents, subscribeAssetEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { displayNameOf } from '$lib/identity';
	import { toast } from '$lib/components/toast.svelte';
	import Button from '$lib/components/Button.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import StateCard from '$lib/components/StateCard.svelte';
	import UnreachableCard from '$lib/components/UnreachableCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import AlbumPropertiesDialog from '$lib/albums/AlbumPropertiesDialog.svelte';
	import CoverPickerDialog from '$lib/albums/CoverPickerDialog.svelte';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import PenLine from '@lucide/svelte/icons/pen-line';
	import Images from '@lucide/svelte/icons/images';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import X from '@lucide/svelte/icons/x';
	import Play from '@lucide/svelte/icons/play';
	import Pause from '@lucide/svelte/icons/pause';
	import LayoutGrid from '@lucide/svelte/icons/layout-grid';
	import Calendar from '@lucide/svelte/icons/calendar';
	import CalendarOff from '@lucide/svelte/icons/calendar-off';
	import CircleMinus from '@lucide/svelte/icons/circle-minus';
	import Check from '@lucide/svelte/icons/check';

	/** The two faces the bar's segmented switch flips. */
	export type MemoryMode = 'play' | 'grid';

	/** One photo advance's beat (the design's four seconds). */
	const PLAYBACK_MS = 4000;

	type Props = {
		/** The overlay's open state (the page derives it from the URL). */
		open: boolean;
		/** The memory the overlay renders. */
		memoryId: string;
		/** The wall card the overlay opened from (may lag a deep link). */
		card: AlbumSummary | null;
		/** The wall's load status: it decides reachability. */
		wallStatus: 'idle' | 'loading' | 'ready' | 'error';
		/** The wall's last failure key (the overlay's retryable face). */
		wallErrorKey?: string;
		/** The attribution chip's hover title (the names-not-keys rule). */
		ownerTitle?: string;
		/** The active face (the URL's `mode`). */
		mode: MemoryMode;
		/** Close the overlay (the page drops the URL parameters). */
		onclose: () => void;
		/** Flip the face (the page writes the URL). */
		onmodechange: (mode: MemoryMode) => void;
		/** Retry the wall's load (the reachability gate). */
		onretrywall?: () => void;
	};

	let {
		open,
		memoryId,
		card,
		wallStatus,
		wallErrorKey = '',
		ownerTitle,
		mode,
		onclose,
		onmodechange,
		onretrywall
	}: Props = $props();

	type DetailStatus = 'idle' | 'loading' | 'ready' | 'error';

	let detail = $state<AlbumDetail | null>(null);
	let status = $state<DetailStatus>('idle');
	/** The error-code i18n key of the last failed read ('' when none). */
	let errorKey = $state('');
	/**
	 * The dead end: the wall does not carry the id, or the detail read
	 * answered the contract's byte-identical 404. The overlay renders
	 * the one unreachable face then -- nothing on it names the memory.
	 */
	let unreachable = $state(false);

	// The playback state: the current index, whether the four-second
	// advance runs, and whether the opening start is still pending
	// (the user's first action cancels it, so a slow first page can
	// never start playing after they took the wheel).
	let index = $state(0);
	let playing = $state(false);
	let autoStartPending = $state(true);

	// The grid's selection (always armed: no select mode to enter).
	let selection = $state<ReadonlySet<string>>(new Set());
	let removeBusy = $state(false);
	let removeErrorKey = $state('');

	// The owner dialogs (they ride inside the overlay).
	let editOpen = $state(false);
	let editErrorKey = $state('');
	let coverOpen = $state(false);
	let coverErrorKey = $state('');
	let deleteOpen = $state(false);
	let deleteErrorKey = $state('');

	/** One write at a time across every action. */
	let writeBusy = $state(false);

	// THE WALK: the memory's own items in the timeline's shape and
	// keyset order, paged to completion (a memory is small; the
	// overlay renders the whole set, and the total is honest). The
	// store IS the list -- no second copy to keep in step.
	const walk = untrack(() => createTimelineClient({ path: '/api/v1/assets/timeline', albumId: memoryId }));
	const items = $derived($walk.items);
	const walkStatus = $derived($walk.status);

	/** The overlay's focus trap root. */
	let panel: HTMLElement | undefined = $state();
	let previouslyFocused: HTMLElement | null = null;

	const owner = $derived(detail?.callerRole === 'owner');
	/** Whether the memory's kind is a preference target for this caller
	 *  (the owner's own reference set never feeds their timeline). */
	const excludable = $derived(detail !== null && !owner && detail.inTimeline !== undefined);
	const excluded = $derived(detail?.inTimeline === false);
	/** The bar's title: the detail's, the card's while it loads. */
	const titleText = $derived(detail?.title ?? card?.title ?? '');
	/** The bar's count: the server's own number, the card's while it loads. */
	const count = $derived(detail?.itemCount ?? card?.itemCount ?? 0);
	/** The member's attribution: the owner's display name (the email falls back). */
	const ownerLabel = $derived(card?.owner === undefined ? '' : displayNameOf(card.owner));

	/** The bar's facts render once something knows them. */
	const settled = $derived(status === 'ready' && detail !== null && !unreachable);
	/** The dead end: the settled wall omits the id, or the read answered 404. */
	const deadEnd = $derived(unreachable || (card === null && wallStatus === 'ready'));
	/** The stage's load face: nothing to show yet, nothing gone wrong. */
	const stageLoading = $derived(
		!deadEnd && (wallStatus === 'loading' || status === 'loading' || walkStatus === 'loading')
	);
	/** The stage's retryable failure (the read's own first, the wall's fallback). */
	const stageErrorKey = $derived(
		!deadEnd && status === 'error'
			? errorKey
			: !deadEnd && card === null && wallStatus === 'error'
				? wallErrorKey
				: ''
	);

	/** The current photo and the walk's total. */
	const current = $derived(items[index] ?? null);
	const total = $derived(items.length);

	/** The selection's photos, in walk order (the batch bar's ids). */
	const selectedIds = $derived(items.filter((item) => selection.has(item.id)).map((item) => item.id));

	/** The stage's serving copy: the view derivative, the inline size. */
	function stageUrl(id: string): string {
		return `/api/v1/assets/${encodeURIComponent(id)}/thumbnail?size=view`;
	}

	/** The grid's serving copy: the tile derivative. */
	function tileUrl(id: string): string {
		return `/api/v1/assets/${encodeURIComponent(id)}/thumbnail?size=tile`;
	}

	/**
	 * Walk the memory's items to the end: the overlay renders the whole
	 * set (a memory is small, and the N / total is only honest with
	 * every page in hand). `fresh` starts from the first page again.
	 */
	async function walkToEnd(fresh: boolean): Promise<void> {
		if (fresh) {
			walk.reset();
		}
		for (;;) {
			await walk.loadMore();
			const state = get(walk);
			if (state.status !== 'ready' || !state.hasMore) {
				return;
			}
		}
	}

	/** Clamp the playhead when the set shrinks under it. */
	$effect(() => {
		if (index >= items.length && items.length > 0) {
			index = items.length - 1;
		}
		if (items.length === 0 && index !== 0) {
			index = 0;
		}
	});

	/** Apply one detail outcome (the superseded runs never call this). */
	function apply(outcome: Awaited<ReturnType<typeof fetchAlbumDetail>>): void {
		if (outcome.ok) {
			detail = outcome.detail;
			errorKey = '';
			status = 'ready';
			return;
		}
		detail = null;
		status = 'error';
		errorKey = outcome.errorKey;
		unreachable = outcome.unreachable;
	}

	/**
	 * The convergence refetch: keep the rendered detail while the fresh
	 * answer is in flight, replace it when it lands (the write's echo).
	 */
	async function convergeDetail(): Promise<void> {
		const outcome = await fetchAlbumDetail(untrack(() => memoryId));
		if (open) {
			apply(outcome);
		}
	}

	// The open lifecycle: the detail read, the walk, the SSE
	// subscriptions. A close clears everything (the next open starts
	// from the server's answer again).
	$effect(() => {
		if (!open || memoryId === '') {
			return;
		}
		const id = memoryId;
		let live = true;
		status = 'loading';
		detail = null;
		errorKey = '';
		unreachable = false;
		index = 0;
		playing = false;
		autoStartPending = true;
		selection = new Set();
		removeErrorKey = '';
		void (async () => {
			const outcome = await fetchAlbumDetail(id);
			if (live) {
				apply(outcome);
			}
		})();
		void walkToEnd(true);

		const unsubscribeAlbums = subscribeAlbumEvents((event) => {
			if (event.albumId !== id) {
				return;
			}
			if (event.name === 'album.items-changed' || event.name === 'album.deleted') {
				// The membership changed (a removal here, another
				// session's curation, the memory's death): the walk and
				// the metadata both re-answer.
				void walkToEnd(true);
				void convergeDetail();
			} else {
				void convergeDetail();
			}
		});
		// A photo's own changes (a re-processed row, a trashed one) patch
		// or drop the item the overlay renders; the walk store is the
		// rendered list, so no copy follows.
		const unsubscribeAssets = subscribeAssetEvents((event) => {
			void walk.applyEvent(event);
		});
		return () => {
			live = false;
			unsubscribeAlbums();
			unsubscribeAssets();
		};
	});

	// The autoplay start: the first page of the walk arms it; the
	// user's first action disarms it. Playback begins only in the play
	// face (a grid deep link stays still until the switch).
	$effect(() => {
		if (!open || !autoStartPending || mode !== 'play' || items.length === 0) {
			return;
		}
		autoStartPending = false;
		playing = true;
	});

	// The advance: one four-second step at a time, rescheduled off the
	// index it landed on. The last photo stops the run (no looping).
	$effect(() => {
		if (!open || !playing || mode !== 'play' || items.length === 0) {
			return;
		}
		const at = index;
		if (at >= items.length - 1) {
			playing = false;
			return;
		}
		const timer = setTimeout(() => {
			index = at + 1;
		}, PLAYBACK_MS);
		return () => clearTimeout(timer);
	});

	// The tab's own stop: a background tab never advances.
	$effect(() => {
		if (typeof document === 'undefined') {
			return;
		}
		const onHidden = (): void => {
			if (document.hidden) {
				playing = false;
			}
		};
		document.addEventListener('visibilitychange', onHidden);
		return () => document.removeEventListener('visibilitychange', onHidden);
	});

	// The focus trap: opening saves the focus, targets the panel and
	// cycles Tab inside it; closing restores the invoking control.
	$effect(() => {
		if (!open) {
			return;
		}
		previouslyFocused = document.activeElement as HTMLElement | null;
		panel?.focus();
		return () => {
			previouslyFocused?.focus();
		};
	});

	/** The focusable elements inside the panel, document order. */
	function focusables(): HTMLElement[] {
		if (panel === undefined) {
			return [];
		}
		return Array.from(
			panel.querySelectorAll<HTMLElement>(
				'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
			)
		);
	}

	function trapTab(event: KeyboardEvent): void {
		const candidates = focusables();
		if (candidates.length === 0) {
			event.preventDefault();
			panel?.focus();
			return;
		}
		const first = candidates[0];
		const last = candidates[candidates.length - 1];
		const active = document.activeElement;
		const inside = panel !== undefined && active !== null && panel.contains(active);
		if (!inside) {
			event.preventDefault();
			(event.shiftKey ? last : first)?.focus();
			return;
		}
		if (event.shiftKey) {
			if (active === first || active === panel) {
				event.preventDefault();
				last?.focus();
			}
			return;
		}
		if (active === last) {
			event.preventDefault();
			first?.focus();
		}
	}

	/** A dialog riding above the overlay owns its keys (the layered Escape). */
	function dialogOpen(): boolean {
		return editOpen || coverOpen || deleteOpen;
	}

	/** A typing field owns its keys (the edit dialog's title box). */
	function isEditableTarget(target: EventTarget | null): boolean {
		if (target === null || !(target instanceof HTMLElement)) {
			return false;
		}
		return target.closest('input, textarea, select, [contenteditable="true"], [contenteditable=""]') !== null;
	}

	/** The playhead's one-step walk: any manual step stops the run. */
	function step(delta: number): void {
		if (items.length === 0) {
			return;
		}
		autoStartPending = false;
		playing = false;
		index = Math.min(items.length - 1, Math.max(0, index + delta));
	}

	/** Start/stop the four-second advance (the user's own switch). */
	function togglePlay(): void {
		if (items.length === 0) {
			return;
		}
		autoStartPending = false;
		playing = !playing;
	}

	/** The face switch: the grid stops the run; the play face resumes it. */
	function switchMode(next: MemoryMode): void {
		if (next === mode) {
			return;
		}
		autoStartPending = false;
		playing = next === 'play' && items.length > 0;
		onmodechange(next);
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (!open || dialogOpen() || isEditableTarget(event.target)) {
			return;
		}
		if (event.key === 'Escape') {
			event.preventDefault();
			playing = false;
			onclose();
			return;
		}
		if (mode !== 'play' || deadEnd) {
			// The grid curates; the keys belong to the playback face.
			return;
		}
		if (event.key === 'Tab') {
			trapTab(event);
			return;
		}
		if (event.key === 'ArrowLeft') {
			event.preventDefault();
			step(-1);
			return;
		}
		if (event.key === 'ArrowRight') {
			event.preventDefault();
			step(1);
			return;
		}
		if (event.key === ' ') {
			// A focused control activates natively on Space; the key
			// reaches the stage alone (the panel itself, a photo, the
			// backdrop).
			const target = event.target;
			if (target instanceof HTMLElement && target.closest('button, a[href], input, textarea, select') !== null) {
				return;
			}
			event.preventDefault();
			togglePlay();
		}
	}

	/** The grid cell's toggle: the selection is always armed. */
	function toggleSelected(id: string): void {
		const next = new Set(selection);
		if (next.has(id)) {
			next.delete(id);
		} else {
			next.add(id);
		}
		selection = next;
	}

	/** Drop the selection (the batch bar's cancel and a landed removal). */
	function clearSelection(): void {
		selection = new Set();
		removeErrorKey = '';
	}

	// The removal: a landed pass clears the selection, confirms through
	// one toast, and re-answers both the walk and the count (the
	// references left; the photos are untouched).
	async function removeSelected(): Promise<void> {
		const ids = selectedIds;
		if (removeBusy || ids.length === 0) {
			return;
		}
		removeBusy = true;
		removeErrorKey = '';
		const outcome = await removeAlbumItems(memoryId, ids);
		removeBusy = false;
		if (!outcome.ok) {
			// The failure keeps the selection (the retry affordance).
			removeErrorKey = outcome.errorKey;
			return;
		}
		clearSelection();
		toast($_('memories_removed_toast'));
		await walkToEnd(true);
		await convergeDetail();
	}

	// The edit write: the page diffs the draft against the LIVE detail
	// and writes only the changed halves (the unchanged half never
	// publishes an event).
	async function submitEdit(draft: {
		title: string;
		audience: readonly { userId: string; email: string }[];
	}): Promise<void> {
		if (writeBusy || detail === null) {
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
			const outcome = await patchAlbum(memoryId, { title: draft.title });
			if (!outcome.ok) {
				writeBusy = false;
				editErrorKey = outcome.errorKey;
				return;
			}
			if (detail !== null) {
				detail = { ...detail, title: outcome.album.title, itemCount: outcome.album.itemCount };
			}
		}
		if (wantsAudience) {
			const outcome = await updateAlbumAudience(memoryId, [...nextIds]);
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
	// adopts the picked id (the event echo refetches anyway).
	async function submitCover(assetId: string): Promise<void> {
		if (writeBusy || detail === null) {
			return;
		}
		writeBusy = true;
		coverErrorKey = '';
		const outcome = await patchAlbum(memoryId, { coverAssetId: assetId });
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

	// The delete write (the light cascade): success closes the overlay;
	// the wall behind converges through the event and drops the card.
	async function confirmDelete(): Promise<void> {
		if (writeBusy || detail === null) {
			return;
		}
		writeBusy = true;
		deleteErrorKey = '';
		const outcome = await deleteAlbum(memoryId);
		writeBusy = false;
		if (!outcome.ok) {
			deleteErrorKey = outcome.errorKey;
			return;
		}
		deleteOpen = false;
		onclose();
	}

	// The member's ONE action: the caller's own timeline preference --
	// the same self-affecting switch a shared album carries (a shared
	// memory IS a legal target: the endpoint's refusals cover the
	// caller's own sync and own memory albums, never a shared one).
	// The write publishes no event, so the response's stored switch is
	// what the bar flips to.
	async function toggleExclusion(): Promise<void> {
		if (writeBusy || detail === null || detail.inTimeline === undefined) {
			return;
		}
		writeBusy = true;
		const outcome = await setAlbumTimelinePreference(memoryId, !detail.inTimeline);
		writeBusy = false;
		if (!outcome.ok) {
			return;
		}
		if (detail !== null) {
			detail = { ...detail, inTimeline: outcome.preference.inTimeline };
		}
	}

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon).
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<!-- The opaque night stage: the whole window, never a percentage
	     scrim. The paper cards (the dead end, the states, the owner's
	     dialogs) sit on it as the surface's guests. -->
	<div class="memory-overlay fixed inset-0 z-(--z-modal) flex bg-night" role="presentation">
		<div
			class="memory-overlay__panel flex h-full w-full flex-col text-cream-text"
			role="dialog"
			aria-modal="true"
			aria-label={$_('memories_overlay_label')}
			tabindex="-1"
			data-mode={mode}
			bind:this={panel}
		>
			<!-- THE BAR: the way back, the title with its count, the face
			     switch, then the role's arms. The dead end keeps the way
			     back alone (title, count, switch and actions all yield). -->
			<header class="memory-overlay__bar flex items-center gap-3 px-4 py-2 border-b border-white/10">
				<button
					class="memory-overlay__close inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-control border border-transparent bg-transparent text-cream-text no-underline cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-white/10"
					type="button"
					aria-label={$_('memories_back')}
					title={$_('memories_back')}
					onclick={onclose}
				>
					<ChevronLeft size={20} aria-hidden="true" />
				</button>

				{#if !deadEnd && titleText !== ''}
					<h2
						class="memory-overlay__title m-0 min-w-0 truncate font-display font-bold text-[20px] leading-[1.3] text-cream-text"
					>
						{titleText}
					</h2>
					{#if settled}
						<CountLabel
							class="memory-overlay__count hidden sm:inline font-body font-normal text-[13px] leading-[1.4] text-cream-text/70"
							{count}
							label={$_('memories_photo_count', { values: { count } })}
						/>
					{/if}
				{/if}

				<div class="memory-overlay__actions ml-auto flex items-center gap-1">
					{#if settled && !deadEnd}
						<div
							class="memory-overlay__modes mr-1 flex items-center gap-0.5 rounded-control bg-white/10 p-0.5"
							role="group"
							aria-label={$_('memories_mode_label')}
						>
							<button
								class="memory-overlay__mode inline-flex items-center gap-1 border-0 bg-transparent px-2.5 py-1 font-body font-semibold text-[13px] leading-[1.4] rounded-control text-cream-text/70 cursor-pointer aria-pressed:bg-white/20 aria-pressed:text-cream-text"
								type="button"
								aria-pressed={mode === 'play'}
								onclick={() => switchMode('play')}
							>
								<Play size={14} aria-hidden="true" />{$_('memories_mode_play')}
							</button>
							<button
								class="memory-overlay__mode inline-flex items-center gap-1 border-0 bg-transparent px-2.5 py-1 font-body font-semibold text-[13px] leading-[1.4] rounded-control text-cream-text/70 cursor-pointer aria-pressed:bg-white/20 aria-pressed:text-cream-text"
								type="button"
								aria-pressed={mode === 'grid'}
								onclick={() => switchMode('grid')}
							>
								<LayoutGrid size={14} aria-hidden="true" />{$_('memories_mode_grid')}
							</button>
						</div>

						{#if owner}
							<!-- The owner's memory-level actions, both faces (a
							     curation act is never a face's privilege). -->
							<button
								class="memory-overlay__action inline-flex h-9 w-9 items-center justify-center rounded-control border border-transparent bg-transparent text-cream-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-white/10"
								type="button"
								disabled={writeBusy}
								aria-label={$_('memories_edit_title')}
								title={$_('memories_edit_title')}
								onclick={() => {
									editErrorKey = '';
									editOpen = true;
								}}
							>
								<PenLine size={18} aria-hidden="true" />
							</button>
							<button
								class="memory-overlay__action inline-flex h-9 w-9 items-center justify-center rounded-control border border-transparent bg-transparent text-cream-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-white/10"
								type="button"
								disabled={writeBusy}
								aria-label={$_('memories_action_cover')}
								title={$_('memories_action_cover')}
								onclick={() => {
									coverErrorKey = '';
									coverOpen = true;
								}}
							>
								<Images size={18} aria-hidden="true" />
							</button>
							<button
								class="memory-overlay__action memory-overlay__action--danger inline-flex h-9 w-9 items-center justify-center rounded-control border border-transparent bg-transparent text-danger cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-white/10"
								type="button"
								disabled={writeBusy}
								aria-label={$_('memories_action_delete')}
								title={$_('memories_action_delete')}
								onclick={() => {
									deleteErrorKey = '';
									deleteOpen = true;
								}}
							>
								<Trash2 size={18} aria-hidden="true" />
							</button>
						{:else}
							<!-- The member's bar: the honest owner attribution and
							     the read-only badge with NO curation arm (the
							     server refuses the member's writes). -->
							{#if ownerLabel !== ''}
								<span
									class="memory-overlay__owner hidden sm:inline font-body font-normal text-[13px] leading-[1.4] text-cream-text/70 truncate"
									title={ownerTitle}
								>
									{$_('albums_shared_by', { values: { name: ownerLabel } })}
								</span>
							{/if}
							<span
								class="memory-overlay__badge inline-flex items-center gap-1 whitespace-nowrap rounded-control border border-cream-text/30 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-cream-text/70"
							>
								{$_('shared_readonly_badge')}
							</span>
							<!-- The one writable thing a shared memory gives the
							     caller: their own timeline preference. The glyph
							     and the name follow the state (in the timeline ->
							     take it out; excluded -> keep it in). -->
							{#if excludable}
								<button
									class="memory-overlay__exclusion inline-flex h-9 w-9 items-center justify-center rounded-control border border-transparent bg-transparent text-cream-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-white/10 disabled:opacity-55 disabled:cursor-not-allowed"
									type="button"
									disabled={writeBusy}
									aria-label={excluded ? $_('albums_action_include') : $_('albums_action_exclude')}
									title={excluded ? $_('albums_action_include') : $_('albums_action_exclude')}
									onclick={() => void toggleExclusion()}
								>
									{#if excluded}
										<Calendar size={18} aria-hidden="true" />
									{:else}
										<CalendarOff size={18} aria-hidden="true" />
									{/if}
								</button>
							{/if}
						{/if}
					{/if}
				</div>
			</header>

			{#if deadEnd}
				<!-- The one dead end, in-view: the same card every unknown,
				     foreign or withdrawn id answers, with no title, no count
				     and no cover. The bar above keeps the way back alone. -->
				<div class="memory-overlay__deadend min-h-0 flex-1 overflow-y-auto px-4 py-6">
					<UnreachableCard />
				</div>
			{:else if stageErrorKey !== ''}
				<div class="memory-overlay__problem min-h-0 flex-1 overflow-y-auto px-4">
					<StateCard
						tone="error"
						body={errorText(stageErrorKey)}
						actionLabel={$_('action_retry')}
						onaction={() => {
							if (status === 'error') {
								void convergeDetail();
							} else {
								void walkToEnd(true);
								onretrywall?.();
							}
						}}
					/>
				</div>
			{:else if stageLoading}
				<!-- The loading state: one line, no spinner, no skeleton. -->
				<div class="memory-overlay__loading grid min-h-0 flex-1 place-items-center">
					<LoadingLine label={$_('memories_loading')} />
				</div>
			{:else if items.length === 0}
				<!-- An empty memory: the state card in the stage's place, and
				     the playback controls below it with the play disabled
				     (there is no frame to advance to). -->
				<div class="memory-overlay__empty min-h-0 flex-1 overflow-y-auto px-4">
					<StateCard title={$_('memories_detail_empty_title')} body={$_('memories_detail_empty_body')} />
				</div>
				<footer class="memory-overlay__footer flex items-center justify-center gap-3 pb-4">
					<span
						class="memory-overlay__index font-body font-normal text-[13px] leading-[1.4] tabular-nums text-cream-text/70"
					>
						0 / 0
					</span>
					<button
						class="memory-overlay__play inline-flex h-9 w-9 items-center justify-center rounded-full border-0 bg-white/10 text-cream-text cursor-pointer hover:bg-white/20 disabled:opacity-55 disabled:cursor-not-allowed"
						type="button"
						disabled
						aria-label={$_('memories_play_action')}
					>
						<Play size={18} aria-hidden="true" />
					</button>
					<span
						class="memory-overlay__hints hidden md:inline font-body font-normal text-[12px] leading-[1.4] text-cream-text/50"
					>
						{$_('memories_keys_hint')}
					</span>
				</footer>
			{:else if mode === 'grid'}
				<!-- THE GRID: one thumbnail per member, square, the selection
				     always armed (its check is part of the cell, not a hover
				     reveal). The batch bar carries the removal. -->
				<div class="memory-overlay__grid-wrap min-h-0 flex-1 overflow-y-auto px-4 py-4 pb-24">
					<div
						class="memory-overlay__grid grid grid-cols-[repeat(auto-fill,minmax(min(150px,100%),1fr))] gap-2"
					>
						{#each items as item (item.id)}
							{@const on = selection.has(item.id)}
							<button
								class="memory-overlay__cell group relative aspect-square overflow-hidden rounded-control bg-accent-soft cursor-pointer"
								class:is-selected={on}
								type="button"
								aria-pressed={on}
								aria-label={item.originalFilename}
								onclick={() => toggleSelected(item.id)}
							>
								{#if item.isReady}
									<img
										class="absolute inset-0 h-full w-full object-cover"
										src={tileUrl(item.id)}
										alt=""
										loading="lazy"
										decoding="async"
										draggable="false"
									/>
								{/if}
								<span
									class="memory-overlay__check absolute top-1.5 left-1.5 z-(--z-tile-check) inline-flex h-[26px] w-[26px] items-center justify-center rounded-full border border-cream-text/35 bg-night text-transparent"
									class:is-on={on}
									aria-hidden="true"
								>
									<Check size={14} />
								</span>
							</button>
						{/each}
					</div>
				</div>

				{#if selectedIds.length > 0}
					<!-- The batch bar: the count, the removal, the cancel. The
					     removal ends the REFERENCES; the photos stay where they
					     live, so the act never borrows the delete vocabulary. -->
					<div
						class="memory-overlay__batchbar fixed left-1/2 bottom-5 z-(--z-toolbar-active) box-content flex max-w-[min(92vw,720px)] flex-wrap items-center justify-center gap-2 rounded-overlay border border-border bg-card px-3.5 py-2.5 shadow-overlay [transform:translateX(-50%)]"
						role="toolbar"
						aria-label={$_('batch_selected_count', { values: { count: selectedIds.length } })}
					>
						<p
							class="memory-overlay__batchbar-count m-0 mr-1.5 whitespace-nowrap font-body font-bold text-[15px] leading-[1.4] text-text tabular-nums"
						>
							{$_('batch_selected_count', { values: { count: selectedIds.length } })}
						</p>
						<IconButton
							class="memory-overlay__remove"
							variant="danger"
							disabled={removeBusy}
							aria-label={$_('action_remove_from_memory')}
							title={$_('action_remove_from_memory')}
							onclick={() => void removeSelected()}
						>
							<CircleMinus size={20} aria-hidden="true" />
						</IconButton>
						<IconButton
							class="memory-overlay__clear"
							disabled={removeBusy}
							aria-label={$_('action_clear_selection')}
							title={$_('action_clear_selection')}
							onclick={clearSelection}
						>
							<X size={20} aria-hidden="true" />
						</IconButton>
						{#if removeErrorKey !== ''}
							<p
								class="memory-overlay__batchbar-error basis-full text-center font-body font-normal text-[13px] leading-[1.4] text-danger-text"
								role="alert"
							>
								{errorText(removeErrorKey)}
							</p>
						{/if}
					</div>
				{/if}
			{:else}
				<!-- THE PLAY STAGE: the 3:2 contain frame, the ‹ › flanks, the
				     N / total row with the play switch and the key hints. -->
				<div
					class="memory-overlay__stage relative flex min-h-0 flex-1 items-center justify-center gap-2 px-2 py-3"
				>
					<button
						class="memory-overlay__step inline-flex h-9 w-9 flex-none items-center justify-center rounded-control border border-transparent bg-transparent text-cream-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-white/10 disabled:opacity-45 disabled:cursor-not-allowed"
						type="button"
						disabled={index === 0}
						aria-label={$_('memories_step_previous')}
						title={$_('memories_step_previous')}
						onclick={() => step(-1)}
					>
						<ChevronLeft size={22} aria-hidden="true" />
					</button>
					<div
						class="relative flex h-full min-h-0 w-full max-w-[1100px] items-center justify-center overflow-hidden"
					>
						<div
							class="memory-overlay__frame relative aspect-[3/2] max-h-full w-full overflow-hidden rounded-card bg-night-surface"
						>
							{#if current !== null}
								{#if current.isReady}
									<img
										class="absolute inset-0 m-auto h-full w-full object-contain"
										src={stageUrl(current.id)}
										alt={current.originalFilename}
										draggable="false"
									/>
								{:else}
									<span
										class="absolute inset-0 grid place-items-center px-6 text-center font-body font-normal text-[13px] leading-[1.4] text-cream-text/60"
									>
										{$_('viewer_processing')}
									</span>
								{/if}
							{/if}
						</div>
					</div>
					<button
						class="memory-overlay__step inline-flex h-9 w-9 flex-none items-center justify-center rounded-control border border-transparent bg-transparent text-cream-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-white/10 disabled:opacity-45 disabled:cursor-not-allowed"
						type="button"
						disabled={index >= items.length - 1}
						aria-label={$_('memories_step_next')}
						title={$_('memories_step_next')}
						onclick={() => step(1)}
					>
						<ChevronRight size={22} aria-hidden="true" />
					</button>
				</div>

				<footer class="memory-overlay__footer flex items-center justify-center gap-3 pb-4">
					<span
						class="memory-overlay__index font-body font-normal text-[13px] leading-[1.4] tabular-nums text-cream-text/70"
					>
						{index + 1} / {total}
					</span>
					<button
						class="memory-overlay__play inline-flex h-9 w-9 items-center justify-center rounded-full border-0 bg-white/10 text-cream-text cursor-pointer hover:bg-white/20"
						type="button"
						aria-pressed={playing}
						aria-label={playing ? $_('memories_pause_action') : $_('memories_play_action')}
						title={playing ? $_('memories_pause_action') : $_('memories_play_action')}
						onclick={togglePlay}
					>
						{#if playing}
							<Pause size={18} aria-hidden="true" />
						{:else}
							<Play size={18} aria-hidden="true" />
						{/if}
					</button>
					<span
						class="memory-overlay__hints hidden md:inline font-body font-normal text-[12px] leading-[1.4] text-cream-text/50"
					>
						{$_('memories_keys_hint')}
					</span>
				</footer>
			{/if}
		</div>
	</div>
{/if}

{#if detail !== null}
	<!-- The edit dialog: the title and the audience alone (a memory owns
	     no source row and no autonomy block). -->
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
		{items}
		currentCoverId={detail.coverAssetId}
		busy={writeBusy}
		errorKey={coverErrorKey}
		onsubmit={(assetId) => void submitCover(assetId)}
	/>
{/if}

<!-- The delete confirm: the light cascade's own statement -- the memory
     and its references and its audience die, the photos themselves stay
     in the albums and the timeline they live in. -->
<Modal bind:open={deleteOpen} title={$_('memories_delete_title')}>
	<p class="memory-overlay__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
		{$_('albums_delete_body', { values: { title: titleText } })}
	</p>
	{#if deleteErrorKey !== ''}
		<p
			class="memory-overlay__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
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

<style>
	/* The night stage's own legibility rules: the loading line and the
	 * state cards keep the paper components' shapes, but the one line
	 * of copy that stands directly on the stage takes the stage's
	 * cream ink (the paper card's text colors already read on their
	 * own surface). */
	.memory-overlay__loading :global(.state-loading) {
		color: var(--color-cream-text);
	}

	/* The grid's selection ring: the accent ring drawn INSIDE the cell
	 * (no reflow, the geometry never moves -- the wall's selected
	 * language translated to a night stage). */
	.memory-overlay__cell.is-selected {
		outline: 2px solid var(--color-accent);
		outline-offset: -2px;
	}

	.memory-overlay__check.is-on {
		border-color: var(--color-accent);
		background: var(--color-accent);
		color: var(--color-accent-on);
	}
</style>
