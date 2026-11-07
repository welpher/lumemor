<script lang="ts">
	// The album detail (web IA §2-§3): the metadata context with its
	// callerRole x type action set over the wall that pages the
	// album's own timeline (the timeline endpoint's `albumId`
	// narrow -- the album's live members in the timeline's exact
	// shape and keyset order).
	//
	// THE CONTEXTUAL BAR (the 2026-11-07 owner pin 3): the page
	// REPLACES the shell's global topbar while it is active -- the
	// content shell hides its own bar on this route, and this page's
	// bar (back, the album's title, the type x role action set as
	// icon-only buttons) pins over the wall in its place. The visual
	// language is the viewer's bar VOCABULARY (a chrome bar with icon
	// buttons) on the PAPER palette (bg-card, the quiet accent-soft
	// hover) -- not the viewer's Night chrome, because this page is a
	// paper page. The drawer nav stays reachable through the back
	// affordance (one step into the albums wall, which carries the
	// shell bar again). The bar pins by stickiness at the ladder's
	// --z-sticky-bar rung, above the wall's own sticky month headers;
	// the page root sets --wall-sticky-offset so those headers stick
	// BELOW the bar instead of under it.
	//
	// THE NO-SELECTION WALL (the 2026-11-07 owner pin 5): the album
	// context renders the shared Wall with NO selection -- no
	// onselect callback is wired, so no hover checkbox renders and no
	// batch bar exists; the tiles are pure click-to-viewer
	// navigation. The wall's data walk (the keyset client over the
	// album narrow, the SSE convergence, the sentinel, the
	// wallItems publication) is the PAGE's own concern here, the
	// same shape the timeline page owns for its walk.
	//
	// THE PINNED CHOICES:
	//
	// - The action set follows the role and the type: the owner of a
	//   virtual album holds rename / set cover (picked from the
	//   rendered members) / lock toggle / share management / add
	//   photos (the handoff to the timeline's selection flow) /
	//   delete; the owner of a directory album holds the properties /
	//   the timeline-exclusion toggle and the rescan placeholder
	//   (disabled with the honest "later update" note -- directory
	//   albums arrive with their own milestone); a viewer holds the
	//   exclusion toggle (their own preference) and the leave
	//   affordance (the revoke of their own share row, the identity
	//   seam's departure). The owner of a SYNC album (the default
	//   "My photos" album) holds NOTHING: system-managed, read-only
	//   -- the explicit empty arm below, zero write affordances, and
	//   the server refuses every write face of the id anyway.
	//   Read-only is VISUAL: a sharee never sees a write affordance,
	//   the server keeps refusing them.
	// - The locked gate is visual too: while locked, the curation and
	//   property actions disable (the unlock is the lock toggle
	//   itself, never gated), while the delete and the share stay
	//   (locked gates neither).
	// - Every write adopts its own response (the property patch and
	//   the lock carry the wall card after the edit, the preference
	//   carries the stored switch) or converges through the detail
	//   refetch (the share pair's grantee list is the refetched
	//   `shares`); no client-side invention -- the SSE self-echoes
	//   converge the same way, collapsed.
	// - SSE: the album events that touch THIS album refetch the
	//   metadata (album.deleted's 404 renders the honest error state);
	//   album.items-changed additionally restarts the items walk --
	//   the event carries ids only, so the wall's fresh answer owns
	//   the grid.
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { _ } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import {
		deleteAlbum,
		fetchAlbumDetail,
		grantAlbumShare,
		listUsers,
		patchAlbum,
		revokeAlbumShare,
		setAlbumTimelinePreference,
		type AlbumDetail,
		type UserSummary
	} from '$lib/api/albums';
	import { createTimelineClient } from '$lib/api/timeline';
	import { subscribeAlbumEvents, subscribeAssetEvents } from '$lib/events';
	import { sessionUserId } from '$lib/auth/session.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { publishWallItems, wallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import Button from '$lib/components/Button.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import AlbumPropertiesDialog from '$lib/albums/AlbumPropertiesDialog.svelte';
	import CoverPickerDialog from '$lib/albums/CoverPickerDialog.svelte';
	import ShareDialog from '$lib/albums/ShareDialog.svelte';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import PenLine from '@lucide/svelte/icons/pen-line';
	import Images from '@lucide/svelte/icons/images';
	import Lock from '@lucide/svelte/icons/lock';
	import LockOpen from '@lucide/svelte/icons/lock-open';
	import Share2 from '@lucide/svelte/icons/share-2';
	import ImagePlus from '@lucide/svelte/icons/image-plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import CalendarOff from '@lucide/svelte/icons/calendar-off';
	import Calendar from '@lucide/svelte/icons/calendar';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import LogOut from '@lucide/svelte/icons/log-out';
	import X from '@lucide/svelte/icons/x';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';

	const albumId = $derived(page.params.id ?? '');

	type DetailStatus = 'idle' | 'loading' | 'ready' | 'error';

	let detail = $state<AlbumDetail | null>(null);
	let status = $state<DetailStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

	// One write at a time across every action (the dialogs and the
	// bar toggles share the flag).
	let writeBusy = $state(false);

	// The properties dialog (rename / properties).
	let propertiesOpen = $state(false);
	let propertiesErrorKey = $state('');

	// The cover picker.
	let coverOpen = $state(false);
	let coverErrorKey = $state('');

	// The share dialog: the member list fetched on open, the grantee
	// list riding the detail's own `shares`.
	let shareOpen = $state(false);
	let shareErrorKey = $state('');
	let shareUsers = $state<UserSummary[]>([]);
	let shareLoading = $state(false);

	// The share dialog: the member list is the server's; the pick list
	// filters the current grantees AND the caller's own account -- the
	// identity seam's userId names the one account the grant's own
	// target rule refuses (`album_share_to_owner`), so the dialog
	// never offers it in the first place.
	const shareCandidates = $derived(
		shareUsers.filter(
			(user) =>
				user.id !== sessionUserId() &&
				!(detail?.shares ?? []).some((share) => share.userId === user.id)
		)
	);

	// The delete confirm.
	let deleteOpen = $state(false);
	let deleteErrorKey = $state('');

	// The exclusion toggle's honest failure (the bar owns the line).
	let exclusionErrorKey = $state('');

	// THE WALL WALK (the page's own data concern): the timeline
	// client over the album narrow, constructed ONCE per mount (a
	// route change remounts the page), the wallResetKey convergence
	// (album.items-changed bumps it and the walk restarts from page
	// one), the SSE asset convergence, the sentinel and the
	// wallItems publication. NO selection state exists here (R5):
	// the Wall below receives no onselect, so its tiles are pure
	// navigation.
	const wall = untrack(() =>
		createTimelineClient({ path: '/api/v1/assets/timeline', view: 'timeline', albumId })
	);
	let wallResetKey = $state(0);

	const owner = $derived(detail?.callerRole === 'owner');
	const virtual = $derived(detail?.type === 'virtual');
	const locked = $derived(detail?.locked ?? false);
	const excluded = $derived(detail?.inTimeline === false);
	const excludable = $derived(detail?.inTimeline !== undefined);

	// The wall's own states (the walk's honest faces, inlined from
	// the curated wrapper this page used to compose).
	const wallFirstLoad = $derived($wall.items.length === 0 && $wall.status === 'loading');
	const wallSettledEmpty = $derived(
		$wall.items.length === 0 && $wall.status === 'ready' && !$wall.hasMore
	);

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
			return;
		}
		detail = null;
		status = 'error';
		errorKey = outcome.errorKey;
	}

	/**
	 * The convergence refetch: keep the rendered detail while the
	 * fresh answer is in flight (a flash of the loading line would
	 * flicker every SSE event), replace it when it lands.
	 */
	async function convergeDetail(): Promise<void> {
		const id = untrack(() => albumId);
		if (id === '' || !fetchCurrent) {
			return;
		}
		const outcome = await fetchAlbumDetail(id);
		if (fetchCurrent) {
			apply(outcome);
		}
	}

	$effect(() => {
		const id = albumId;
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
			if (event.albumId !== untrack(() => albumId)) {
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
	// -- exactly the grid the detail renders (the picker itself only
	// renders while the detail is loaded).
	const members = $derived(wallItems());

	/** Adopt a write response's wall card into the detail's fields. */
	function applyWriteFields(next: {
		title: string;
		description: string;
		locked: boolean;
		itemCount: number;
	}): void {
		if (detail !== null) {
			detail = {
				...detail,
				title: next.title,
				description: next.description,
				locked: next.locked,
				itemCount: next.itemCount
			};
		}
	}

	// The properties write: the PATCH response is the album's wall
	// card after the edit -- the fields adopt it.
	async function submitProperties(title: string, description: string): Promise<void> {
		const id = untrack(() => albumId);
		if (writeBusy || detail === null || id === '') {
			return;
		}
		writeBusy = true;
		propertiesErrorKey = '';
		const outcome = await patchAlbum(id, { title, description });
		writeBusy = false;
		if (!outcome.ok) {
			propertiesErrorKey = outcome.errorKey;
			return;
		}
		applyWriteFields(outcome.album);
		propertiesOpen = false;
	}

	// The lock toggle: `locked` itself is never gated -- a patch of
	// only `locked: false` IS the unlock.
	async function toggleLock(): Promise<void> {
		const id = untrack(() => albumId);
		if (writeBusy || detail === null || id === '') {
			return;
		}
		writeBusy = true;
		const outcome = await patchAlbum(id, { locked: !detail.locked });
		writeBusy = false;
		if (!outcome.ok) {
			// The toggle's honest failure: the bar owns the line
			// through the shared error area.
			exclusionErrorKey = outcome.errorKey;
			return;
		}
		applyWriteFields(outcome.album);
	}

	// The cover write: the 200 proves the pin landed, so the detail
	// adopts the picked id (the SSE echo refetches anyway).
	async function submitCover(assetId: string): Promise<void> {
		const id = untrack(() => albumId);
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

	// The share dialog: the member list is the server's minimal face
	// (every account); the pick list's filtering -- the current
	// grantees and the caller's own account -- lives in the
	// shareCandidates derivation above.
	$effect(() => {
		if (!shareOpen) {
			return;
		}
		shareUsers = [];
		shareErrorKey = '';
		shareLoading = true;
		void (async () => {
			const outcome = await listUsers();
			if (!shareOpen) {
				return;
			}
			shareLoading = false;
			if (outcome.ok) {
				shareUsers = outcome.users;
			}
		})();
	});

	// The grant write: the refetched detail's `shares` array is the
	// dialog's list (the SSE album.shared echo converges the same
	// way, collapsed).
	async function grantShare(userId: string): Promise<void> {
		const id = untrack(() => albumId);
		if (writeBusy || id === '') {
			return;
		}
		writeBusy = true;
		shareErrorKey = '';
		const outcome = await grantAlbumShare(id, userId);
		writeBusy = false;
		if (!outcome.ok) {
			shareErrorKey = outcome.errorKey;
			return;
		}
		await convergeDetail();
	}

	// The revoke write: the same convergence.
	async function revokeShare(userId: string): Promise<void> {
		const id = untrack(() => albumId);
		if (writeBusy || id === '') {
			return;
		}
		writeBusy = true;
		shareErrorKey = '';
		const outcome = await revokeAlbumShare(id, userId);
		writeBusy = false;
		if (!outcome.ok) {
			shareErrorKey = outcome.errorKey;
			return;
		}
		await convergeDetail();
	}

	// The delete write: success leaves for the wall (a deep link may
	// have no history to back through).
	async function confirmDelete(): Promise<void> {
		const id = untrack(() => albumId);
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
		void goto('/albums');
	}

	// The exclusion toggle: self-affecting state -- the write
	// publishes no event, the response's stored switch is the truth
	// the badge and notice follow.
	async function toggleExclusion(): Promise<void> {
		const id = untrack(() => albumId);
		if (writeBusy || detail === null || id === '' || detail.inTimeline === undefined) {
			return;
		}
		writeBusy = true;
		exclusionErrorKey = '';
		const outcome = await setAlbumTimelinePreference(id, !detail.inTimeline);
		writeBusy = false;
		if (!outcome.ok) {
			exclusionErrorKey = outcome.errorKey;
			return;
		}
		if (detail !== null) {
			detail = { ...detail, inTimeline: outcome.preference.inTimeline };
		}
	}

	// The sharee's departure (the identity seam's second consumer):
	// the revoke of the CALLER'S OWN share row -- DELETE
	// /albums/{id}/shares/{own id}. The owner can re-share at any
	// time, so no confirm dialog stands in the way (the same
	// icon-button shape the owner's own revoke rows carry). Success
	// leaves for the wall: the album left the caller's view, the
	// detail would 404 from here on. A failure renders through the
	// bar's shared error line.
	async function leaveAlbum(): Promise<void> {
		const id = untrack(() => albumId);
		const userId = sessionUserId();
		if (writeBusy || id === '' || userId === null) {
			return;
		}
		writeBusy = true;
		exclusionErrorKey = '';
		const outcome = await revokeAlbumShare(id, userId);
		writeBusy = false;
		if (!outcome.ok) {
			exclusionErrorKey = outcome.errorKey;
			return;
		}
		void goto('/albums');
	}

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// The page's display title: the localized default-album name for
	// the caller's own sync album, the stored title for every other
	// album -- the contract's client-side display-name convention
	// (the sync album's stored title is the system's internal label
	// and never renders).
	const titleText = $derived(
		detail?.type === 'sync' ? $_('album_all_photos') : (detail?.title ?? $_('albums_title'))
	);

	// Publish the loaded page items as the viewer's navigation
	// sequence (see $lib/timeline/wall.svelte): the copy outlives
	// this route, so the /a/{id} viewer walks exactly the items this
	// wall rendered.
	$effect(() => {
		publishWallItems($wall.items);
	});

	// The walk itself: the first load and every wallResetKey change
	// (the album's membership convergence). A reset while a page load
	// is in flight discards the stale response (the client's
	// generation guard), so the fresh walk's answer is the only one
	// that lands.
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
	<title>{titleText} · lumemor</title>
</svelte:head>

<!-- The page root pins the wall's sticky month headers below the
     contextual bar: the bar is py-2 (16) around an h-9 row (36)
     plus its 1px border -- 53px. -->
<section class="album-detail" style:--wall-sticky-offset="53px">
	<!-- THE CONTEXTUAL BAR (R3): back + the title + the type x role
	     icon actions, pinned over the wall on the card palette. The
	     shell's own topbar hides while this route is active (the
	     content shell's route rule). -->
	<header class="album-bar sticky top-0 z-(--z-sticky-bar) flex items-center gap-2 px-4 py-2 bg-card border-b border-border">
		<a
			class="album-bar__back inline-flex items-center justify-center w-9 h-9 shrink-0 rounded-control bg-transparent text-text-soft no-underline cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/albums"
			aria-label={$_('albums_back')}
		>
			<ChevronLeft size={20} aria-hidden="true" />
		</a>

		{#if detail !== null}
			<h1 class="album-bar__title m-0 min-w-0 truncate font-display font-bold text-[20px] leading-[1.3] text-text">{titleText}</h1>
			{#if locked}
				<span class="album-detail__badge album-detail__badge--locked hidden sm:inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap">
					<svg viewBox="0 0 24 24" width="12" height="12" fill="none" aria-hidden="true">
						<rect x="5" y="10.5" width="14" height="9" rx="2" stroke="currentColor" stroke-width="1.75" />
						<path d="M8 10.5V8a4 4 0 0 1 8 0v2.5" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
					</svg>
					{$_('albums_locked_badge')}
				</span>
			{/if}
			{#if excluded}
				<!-- The muted clock: the caller's own preference keeps this
				     album's photos out of their aggregated timeline. (Below sm
				     the badge yields -- the body's notice carries the state,
				     and the one-row bar keeps its height for the sticky
				     offset's math.) -->
				<span class="album-detail__badge album-detail__badge--excluded hidden sm:inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap">
					<svg viewBox="0 0 24 24" width="12" height="12" fill="none" aria-hidden="true">
						<circle cx="12" cy="12" r="8.25" stroke="currentColor" stroke-width="1.75" />
						<path d="M12 7.5V12l3 2" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" />
					</svg>
					{$_('albums_excluded_badge')}
				</span>
			{/if}
			<span class="album-detail__count hidden sm:inline font-body font-normal text-[13px] leading-[1.4] text-text-soft whitespace-nowrap tabular-nums">{$_('albums_photo_count', { values: { count: detail.itemCount } })}</span>
		{/if}

		<!-- The action set, per callerRole x type, as icon-only
		     buttons (the icon-first rule; the aria-labels are the
		     actions' own strings): the owner of a virtual album holds
		     the full write set; the owner of a directory album the
		     properties / exclusion / rescan placeholder; the owner of
		     the sync album (the default album) holds nothing --
		     system-managed; a viewer the exclusion toggle and the
		     leave affordance -- both self-affecting writes the share
		     holds. -->
		<div class="album-detail__actions ml-auto flex items-center gap-1">
			{#if detail !== null && owner && virtual}
				<IconButton disabled={writeBusy || locked} aria-label={$_('albums_action_rename')} onclick={() => { propertiesErrorKey = ''; propertiesOpen = true; }}>
					<PenLine size={20} aria-hidden="true" />
				</IconButton>
				<IconButton disabled={writeBusy || locked} aria-label={$_('albums_action_cover')} onclick={() => { coverErrorKey = ''; coverOpen = true; }}>
					<Images size={20} aria-hidden="true" />
				</IconButton>
				<IconButton disabled={writeBusy} aria-label={locked ? $_('albums_action_unlock') : $_('albums_action_lock')} onclick={() => void toggleLock()}>
					{#if locked}
						<LockOpen size={20} aria-hidden="true" />
					{:else}
						<Lock size={20} aria-hidden="true" />
					{/if}
				</IconButton>
				<IconButton aria-label={$_('albums_action_share')} onclick={() => { shareErrorKey = ''; shareOpen = true; }}>
					<Share2 size={20} aria-hidden="true" />
				</IconButton>
				<IconButton disabled={writeBusy || locked} aria-label={$_('albums_action_add_photos')} onclick={() => void goto(`/timeline?album=${encodeURIComponent(albumId)}`)}>
					<ImagePlus size={20} aria-hidden="true" />
				</IconButton>
				{#if !locked}
					<!-- The permission matrix pins deletion to the UNLOCKED
					     state: the owner unlocks first, one step. -->
					<IconButton variant="danger" disabled={writeBusy} aria-label={$_('albums_action_delete')} onclick={() => { deleteErrorKey = ''; deleteOpen = true; }}>
						<Trash2 size={20} aria-hidden="true" />
					</IconButton>
				{/if}
			{:else if detail !== null && owner && detail.type === 'directory'}
				<IconButton disabled={writeBusy || locked} aria-label={$_('albums_action_properties')} onclick={() => { propertiesErrorKey = ''; propertiesOpen = true; }}>
					<PenLine size={20} aria-hidden="true" />
				</IconButton>
				{#if excludable}
					<IconButton disabled={writeBusy} aria-label={excluded ? $_('albums_action_include') : $_('albums_action_exclude')} onclick={() => void toggleExclusion()}>
						{#if excluded}
							<Calendar size={20} aria-hidden="true" />
						{:else}
							<CalendarOff size={20} aria-hidden="true" />
						{/if}
					</IconButton>
				{/if}
				<!-- The rescan placeholder: honest and disabled -- the
				     directory album's own flow arrives with a later
				     milestone, and the tooltip says so. -->
				<IconButton disabled title={$_('albums_rescan_note')} aria-describedby="album-detail-rescan-note" aria-label={$_('albums_rescan_action')}>
					<RefreshCw size={20} aria-hidden="true" />
				</IconButton>
			{:else if detail !== null && owner && detail.type === 'sync'}
				<!-- The default album's arm: NOTHING. The sync album is
				     system-managed -- no rename, no cover, no lock, no
				     share, no add-photos, no delete, no exclusion
				     toggle. The arm renders zero affordances, and the
				     server refuses every write face of the id anyway;
				     the wall below is the whole page. -->
			{:else if detail !== null && !owner}
				<!-- The read-only share's own writes: the timeline
				     exclusion (the caller's preference) and the
				     departure (the caller's own share row). -->
				{#if excludable}
					<IconButton disabled={writeBusy} aria-label={excluded ? $_('albums_action_include') : $_('albums_action_exclude')} onclick={() => void toggleExclusion()}>
						{#if excluded}
							<Calendar size={20} aria-hidden="true" />
						{:else}
							<CalendarOff size={20} aria-hidden="true" />
						{/if}
					</IconButton>
				{/if}
				<IconButton disabled={writeBusy} aria-label={$_('albums_leave_action')} onclick={() => void leaveAlbum()}>
					<LogOut size={20} aria-hidden="true" />
				</IconButton>
			{/if}
		</div>
	</header>

	{#if status === 'loading'}
		<p class="album-detail__loading my-12 text-center text-text-soft animate-loading-in" role="status">{$_('albums_detail_loading')}</p>
	{:else if status === 'error'}
		<div class="album-detail__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void convergeDetail()}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
		</div>
	{:else if detail !== null}
		<!-- The context block (the page's card-framed text: the
		     description, the notices, the shared error line) -- the
		     WALL below is full-flow edge-to-edge like every wall. -->
		<div class="album-detail__body max-w-[1200px] mx-auto px-2">
			{#if owner && detail.type === 'directory'}
				<p id="album-detail-rescan-note" class="album-detail__rescan-note my-0 mb-3 font-body font-normal text-[13px] leading-[1.5] text-text-soft">{$_('albums_rescan_note')}</p>
			{/if}
			{#if detail.description !== ''}
				<p class="album-detail__description my-0 mb-3 font-body font-normal text-[15px] leading-[1.5] text-text-soft">{detail.description}</p>
			{/if}
			{#if locked && owner && virtual}
				<!-- The locked gate, honest: curation and property writes stay
				     off until the unlock (the toggle above). -->
				<p class="album-detail__locked-notice my-0 mb-3 font-body font-medium text-[13px] leading-[1.5] text-text-soft" role="status">{$_('albums_locked_notice')}</p>
			{/if}
			{#if excluded}
				<p class="album-detail__excluded-notice my-0 mb-3 font-body font-medium text-[13px] leading-[1.5] text-text-soft" role="status">{$_('albums_excluded_notice')}</p>
			{/if}
			{#if exclusionErrorKey !== ''}
				<!-- The bar toggles' honest failure (the lock, the
				     exclusion): the server's own answer. -->
				<p class="album-detail__action-error my-0 mb-3 font-body font-medium text-[14px] leading-[1.5] text-danger-text" role="alert">{errorText(exclusionErrorKey)}</p>
			{/if}
		</div>

		<!-- THE NO-SELECTION WALL (R5): the page's own walk (the keyset
		     client, the SSE convergence, the sentinel) around the
		     shared Wall's pure rendering -- no onselect wired, so the
		     tiles are pure navigation and no batch bar exists. -->
		<section class="album-detail__wall px-2" aria-busy={$wall.status === 'loading'}>
			{#if wallFirstLoad}
				<p class="album-detail__wall-status my-12 text-center text-text-soft animate-loading-in" role="status">{$_('albums_detail_loading')}</p>
			{:else if wallSettledEmpty}
				<div class="album-detail__wall-empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card">
					<h2 class="text-[22px] mb-2">{titleText}</h2>
					<p class="text-text-soft">{$_('albums_detail_empty')}</p>
				</div>
			{:else if $wall.status === 'error' && $wall.items.length === 0}
				<div class="album-detail__wall-failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
					<p>{errorText($wall.errorKey)}</p>
					<Button variant="primary" onclick={() => void wall.loadMore()}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
				</div>
			{:else}
				<Wall items={$wall.items} />
			{/if}

			{#if $wall.status === 'error' && $wall.items.length > 0}
				<div class="album-detail__wall-failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
					<p>{errorText($wall.errorKey)}</p>
					<Button variant="primary" onclick={() => void wall.loadMore()}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
				</div>
			{/if}

			{#if !observerAvailable && $wall.hasMore && $wall.status === 'ready'}
				<div class="album-detail__wall-pager flex justify-center my-5">
					<Button variant="ghost" onclick={() => void wall.loadMore()}>
						<ChevronsDown size={20} aria-hidden="true" />
						{$_('timeline_load_more')}
					</Button>
				</div>
			{/if}

			<div class="album-detail__wall-sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
		</section>
	{/if}
</section>

{#if detail !== null}
	<AlbumPropertiesDialog
		bind:open={propertiesOpen}
		titleKey="albums_properties_title"
		submitKey="albums_properties_submit"
		initialTitle={detail.title}
		initialDescription={detail.description}
		showDescription={true}
		busy={writeBusy}
		errorKey={propertiesErrorKey}
		onsubmit={(title, description) => void submitProperties(title, description)}
	/>

	<CoverPickerDialog
		bind:open={coverOpen}
		items={members}
		currentCoverId={detail.coverAssetId}
		busy={writeBusy}
		errorKey={coverErrorKey}
		onsubmit={(assetId) => void submitCover(assetId)}
	/>

	<ShareDialog
		bind:open={shareOpen}
		shares={detail.shares ?? []}
		candidates={shareCandidates}
		loading={shareLoading}
		busy={writeBusy}
		errorKey={shareErrorKey}
		ongrant={(userId) => void grantShare(userId)}
		onrevoke={(userId) => void revokeShare(userId)}
	/>
{/if}

<!-- The delete confirm: the album goes, its photo list and shares
     with it -- the photos themselves stay. -->
<Modal bind:open={deleteOpen} title={$_('albums_delete_title')}>
	{#snippet children()}
		<p class="album-detail__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
			{$_('albums_delete_body', { values: { title: detail?.title ?? '' } })}
		</p>
		{#if deleteErrorKey !== ''}
			<p class="album-detail__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text" role="alert">{errorText(deleteErrorKey)}</p>
		{/if}
	{/snippet}
	{#snippet footer()}
		<Button variant="ghost" disabled={writeBusy} onclick={() => (deleteOpen = false)}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="danger" disabled={writeBusy} onclick={() => void confirmDelete()}>
			<Trash2 size={20} aria-hidden="true" />
			{$_('albums_delete_confirm')}
		</Button>
	{/snippet}
</Modal>
