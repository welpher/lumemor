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
	// THE NO-SELECTION WALL (the 2026-11-07 owner pin 5, restated on
	// the 2026-11-08 selection pin's new gate): the album context
	// renders the shared Wall with NO curation -- no onbatch runner
	// is wired, so no hover checkbox renders and no batch bar
	// exists; the tiles are pure click-to-viewer navigation. The
	// wall's data walk (the keyset client over the album narrow, the
	// SSE convergence, the sentinel, the wallItems publication) is
	// the PAGE's own concern here, the same shape the timeline page
	// owns for its walk.
	//
	// THE PINNED CHOICES (E4's albums-as-sources reshape, completed
	// by E6's autonomy face):
	//
	// - The action set follows the role and the type: the owner of a
	//   virtual album holds rename / set cover (picked from the
	//   rendered members) / delete; the owner of a directory album
	//   holds the properties (the edit dialog: title + the read-only
	//   source row + the audience + the autonomy block) / the rescan
	//   trigger (E4: the three-arm diff run, its progress surfaced
	//   through the job events the run's identifier keys); a viewer
	//   holds the exclusion toggle (their own preference) alone --
	//   the sharee's departure went with the revoke endpoint it rode
	//   (the audience write is owner-domain; the member side is the
	//   read-only view). The owner of a SYNC album (the default
	//   "My photos" album) holds NOTHING: system-managed, read-only
	//   -- the explicit empty arm below, zero write affordances, and
	//   the server refuses every write face of the id anyway.
	//   Read-only is VISUAL: a sharee never sees a write affordance,
	//   the server keeps refusing them.
	// - THE LOCK LIVES IN THE EDIT DIALOG (E6, §7.9: the five-piece
	//   set's missing piece -- the same home as the timeline
	//   exclusion and the deletability switch, one dialog for every
	//   album-owned toggle), so the bar's dialog opener stays LIVE
	//   under a lock: it is the unlock's door. The OTHER contextual
	//   actions freeze under the lock honestly: the cover and the
	//   delete on the virtual arm, the rescan on the directory arm
	//   (the wall's delete affordance dies with `deletable`, the
	//   server-derived face that folds the lock in).
	// - THE WALL'S DELETE AFFORDANCE (E6, §7.9): the owner of a
	//   DELETABLE directory album (owner AND writable source AND
	//   the deletability switch AND unlocked -- all four, computed
	//   server-side as `deletable`, pure data) curates a
	//   delete-only selection: `onbatch` is wired with the trash
	//   action alone, so the wall's own selection machinery (the
	//   hover check, the batch bar) renders exactly then and never
	//   otherwise. A member's wall stays pure navigation.
	// - Every write adopts its own response (the property patch and
	//   the lock carry the wall card after the edit, the preference
	//   carries the stored switch) or converges through the detail
	//   refetch (the audience write's grantee list is the refetched
	//   `shares`); no client-side invention -- the SSE self-echoes
	//   converge the same way, collapsed.
	// - The rescan's live progress rides the activity stream's
	//   job.* events (the models download page's mechanism, keyed by
	//   the run identifier the trigger answers): progress events
	//   update the run in place, the finish lands the final state,
	//   and the membership the scan moved converges through the
	//   album events the run publishes. The run's live view is
	//   page-scoped (the honest limit: no user-domain face lists a
	//   run by album, so a navigation away loses the line while the
	//   run itself completes server-side).
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
		patchAlbum,
		rescanAlbum,
		setAlbumTimelinePreference,
		updateAlbumAudience,
		type AlbumDetail
	} from '$lib/api/albums';
	import { createTimelineClient, type BatchAction } from '$lib/api/timeline';
	import { subscribeAlbumEvents, subscribeAssetEvents, subscribeJobEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { publishWallItems, wallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import Button from '$lib/components/Button.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import AlbumPropertiesDialog, { type AlbumAutonomyDraft } from '$lib/albums/AlbumPropertiesDialog.svelte';
	import CoverPickerDialog from '$lib/albums/CoverPickerDialog.svelte';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import PenLine from '@lucide/svelte/icons/pen-line';
	import Images from '@lucide/svelte/icons/images';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import CalendarOff from '@lucide/svelte/icons/calendar-off';
	import Calendar from '@lucide/svelte/icons/calendar';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
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

	// The properties dialog (the E4 edit: title + source + audience).
	let propertiesOpen = $state(false);
	let propertiesErrorKey = $state('');

	// The cover picker.
	let coverOpen = $state(false);
	let coverErrorKey = $state('');

	// The delete confirm.
	let deleteOpen = $state(false);
	let deleteErrorKey = $state('');

	// The bar actions' honest failure (the lock, the exclusion, the
	// rescan trigger): the bar owns the line.
	let actionErrorKey = $state('');

	// The rescan run this page triggered (or joined): the run's
	// identifier keys the job events the progress rides. Null until
	// a trigger lands; the live view is page-scoped, the honest
	// limit noted above.
	type RescanRun = {
		id: string;
		state: 'queued' | 'running' | 'succeeded' | 'failed';
		total: number;
		done: number;
		failed: number;
	};
	let rescanRun = $state<RescanRun | null>(null);

	// THE WALL WALK (the page's own data concern): the timeline
	// client over the album narrow, constructed ONCE per mount (a
	// route change remounts the page), the wallResetKey convergence
	// (album.items-changed bumps it and the walk restarts from page
	// one), the SSE asset convergence, the sentinel and the
	// wallItems publication. The wall's SELECTION exists exactly
	// when this page wires the batch runner below (the `deletable`
	// gate, E6): every other face is pure navigation (R5).
	const wall = untrack(() => createTimelineClient({ path: '/api/v1/assets/timeline', view: 'timeline', albumId }));
	let wallResetKey = $state(0);

	const owner = $derived(detail?.callerRole === 'owner');
	const virtual = $derived(detail?.type === 'virtual');
	const locked = $derived(detail?.locked ?? false);
	const excluded = $derived(detail?.inTimeline === false);
	const excludable = $derived(detail?.inTimeline !== undefined);
	// The wall's delete affordance (E6): the server-derived face --
	// owner AND writable source AND the switch AND unlocked, pure
	// data -- decides whether this wall curates at all.
	const deletable = $derived(owner && detail?.type === 'directory' && detail.deletable === true);

	// The wall's own states (the walk's honest faces, inlined from
	// the curated wrapper this page used to compose).
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
	function applyWriteFields(next: { title: string; locked: boolean; itemCount: number }): void {
		if (detail !== null) {
			detail = {
				...detail,
				title: next.title,
				locked: next.locked,
				itemCount: next.itemCount
			};
		}
	}

	// The properties write (the E6 edit dialog): the page diffs the
	// whole draft against the LIVE detail and writes only the changed
	// halves -- a same-value title PATCH would still publish its
	// album.updated event, and a same-set audience PUT is the
	// contract's silent no-op, so a no-change save writes nothing at
	// all. THE WRITE SEQUENCE respects the locked gate's shape: the
	// UNLOCK writes first and ALONE (a locked album refuses a mixed
	// patch whole), the property fields (title, the deletability
	// switch) follow it, and the LOCK writes LAST (it freezes
	// everything after it); the audience PUT and the caller's own
	// timeline preference ride ungated either way (§7.9: a lock
	// protects content, not visibility; the preference is nobody's
	// content). Each write's PATCH response is the album's wall card
	// after the edit (the fields adopt it); the final detail
	// refetch converges the faces the cards never carry (the
	// grantee list, the deletability state, the derived affordance).
	async function submitProperties(draft: AlbumAutonomyDraft): Promise<void> {
		const id = untrack(() => albumId);
		if (writeBusy || detail === null || id === '') {
			return;
		}
		const wasLocked = detail.locked;
		const wantsUnlock = wasLocked && !draft.locked;
		const wantsLock = !wasLocked && draft.locked;
		const wantsTitle = draft.title !== detail.title;
		const wantsDeletable =
			detail.sourceWritable === true &&
			draft.allowAppDeletion !== undefined &&
			draft.allowAppDeletion !== (detail.allowAppDeletion ?? true);
		const currentIds = new Set((detail.shares ?? []).map((share) => share.userId));
		const nextIds = new Set(draft.audience.map((member) => member.userId));
		const wantsAudience =
			nextIds.size !== currentIds.size || [...nextIds].some((userId) => !currentIds.has(userId));
		const wantsExclusion =
			detail.inTimeline !== undefined && draft.inTimeline !== undefined && draft.inTimeline !== detail.inTimeline;
		if (!wantsUnlock && !wantsLock && !wantsTitle && !wantsDeletable && !wantsAudience && !wantsExclusion) {
			propertiesOpen = false;
			return;
		}
		writeBusy = true;
		propertiesErrorKey = '';
		if (wantsUnlock) {
			// The unlock is the one write a lock never refuses, and a
			// mixed patch carrying it is refused whole -- so it lands
			// alone, before any property field.
			const outcome = await patchAlbum(id, { locked: false });
			if (!outcome.ok) {
				writeBusy = false;
				propertiesErrorKey = outcome.errorKey;
				return;
			}
			applyWriteFields(outcome.album);
		}
		if (wantsTitle || wantsDeletable) {
			const patch: { title?: string; allowAppDeletion?: boolean } = {};
			if (wantsTitle) {
				patch.title = draft.title;
			}
			if (wantsDeletable) {
				patch.allowAppDeletion = draft.allowAppDeletion;
			}
			const outcome = await patchAlbum(id, patch);
			if (!outcome.ok) {
				writeBusy = false;
				propertiesErrorKey = outcome.errorKey;
				return;
			}
			applyWriteFields(outcome.album);
		}
		if (wantsLock) {
			// The lock lands LAST: it freezes the writes that follow
			// it, and there are none left in this save.
			const outcome = await patchAlbum(id, { locked: true });
			if (!outcome.ok) {
				writeBusy = false;
				propertiesErrorKey = outcome.errorKey;
				return;
			}
			applyWriteFields(outcome.album);
		}
		if (wantsAudience) {
			const outcome = await updateAlbumAudience(id, [...nextIds]);
			if (!outcome.ok) {
				writeBusy = false;
				propertiesErrorKey = outcome.errorKey;
				return;
			}
		}
		if (wantsExclusion) {
			const outcome = await setAlbumTimelinePreference(id, draft.inTimeline === true);
			if (!outcome.ok) {
				writeBusy = false;
				propertiesErrorKey = outcome.errorKey;
				return;
			}
		}
		writeBusy = false;
		await convergeDetail();
		propertiesOpen = false;
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

	// The rescan trigger (E4): POST the run, adopt the identifier it
	// answers (202 the freshly created run, 200 the in-flight one of
	// THIS album the request joined -- the endpoint's idempotence),
	// and let the job events the run publishes drive the progress
	// line below. The run's bookkeeping rides job_runs, not the
	// album row: the write the trigger makes is the run itself.
	async function startRescan(): Promise<void> {
		const id = untrack(() => albumId);
		if (writeBusy || detail === null || id === '' || detail.type !== 'directory') {
			return;
		}
		writeBusy = true;
		actionErrorKey = '';
		const outcome = await rescanAlbum(id);
		writeBusy = false;
		if (!outcome.ok) {
			// The trigger's honest failure (a locked album's 409, a dead
			// session): the bar's shared error line.
			actionErrorKey = outcome.errorKey;
			return;
		}
		rescanRun = { id: outcome.run.jobRunId, state: 'queued', total: 0, done: 0, failed: 0 };
	}

	// The SSE convergence of the run: progress events update the run
	// in place (the events' counts are already numbers, the state
	// already the run lifecycle's), the finish lands the final state.
	// Only the run THIS page tracked applies -- another album's
	// rescan (or any other job kind) never touches this line.
	$effect(() => {
		return subscribeJobEvents((event) => {
			if (event.kind !== 'album_rescan') {
				return;
			}
			const run = rescanRun;
			if (run === null || run.id !== event.jobRunId) {
				return;
			}
			rescanRun = {
				...run,
				total: event.total,
				done: event.done,
				failed: event.failed,
				state: event.state
			};
		});
	});

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

	// The wall's batch runner (E6): the page's own client runs the
	// delete-only batch the wall's bar offers -- the same
	// view-aware optimistic convergence every curatable wall uses
	// (the trashed tiles leave immediately, the SSE events of the
	// landed batch confirm the removal, the album's own events
	// converge the detail behind). The wall owns the busy, the
	// mapped error and the selection choreography around it.
	async function runWallBatch(
		action: BatchAction,
		ids: readonly string[]
	): Promise<{ ok: true } | { ok: false; errorKey: string }> {
		return wall.runBatch(action, [...ids]);
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
		actionErrorKey = '';
		const outcome = await setAlbumTimelinePreference(id, !detail.inTimeline);
		writeBusy = false;
		if (!outcome.ok) {
			actionErrorKey = outcome.errorKey;
			return;
		}
		if (detail !== null) {
			detail = { ...detail, inTimeline: outcome.preference.inTimeline };
		}
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
	<header
		class="album-bar sticky top-0 z-(--z-sticky-bar) flex items-center gap-2 px-4 py-2 bg-card border-b border-border"
	>
		<a
			class="album-bar__back inline-flex items-center justify-center w-9 h-9 shrink-0 rounded-control bg-transparent text-text-soft no-underline cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/albums"
			aria-label={$_('albums_back')}
		>
			<ChevronLeft size={20} aria-hidden="true" />
		</a>

		{#if detail !== null}
			<h1
				class="album-bar__title m-0 min-w-0 truncate font-display font-bold text-[20px] leading-[1.3] text-text"
			>
				{titleText}
			</h1>
			{#if locked}
				<span
					class="album-detail__badge album-detail__badge--locked hidden sm:inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
				>
					<svg viewBox="0 0 24 24" width="12" height="12" fill="none" aria-hidden="true">
						<rect x="5" y="10.5" width="14" height="9" rx="2" stroke="currentColor" stroke-width="1.75" />
						<path
							d="M8 10.5V8a4 4 0 0 1 8 0v2.5"
							stroke="currentColor"
							stroke-width="1.75"
							stroke-linecap="round"
						/>
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
				<span
					class="album-detail__badge album-detail__badge--excluded hidden sm:inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
				>
					<svg viewBox="0 0 24 24" width="12" height="12" fill="none" aria-hidden="true">
						<circle cx="12" cy="12" r="8.25" stroke="currentColor" stroke-width="1.75" />
						<path
							d="M12 7.5V12l3 2"
							stroke="currentColor"
							stroke-width="1.75"
							stroke-linecap="round"
							stroke-linejoin="round"
						/>
					</svg>
					{$_('albums_excluded_badge')}
				</span>
			{/if}
			<span
				class="album-detail__count hidden sm:inline font-body font-normal text-[13px] leading-[1.4] text-text-soft whitespace-nowrap tabular-nums"
				>{$_('albums_photo_count', { values: { count: detail.itemCount } })}</span
			>
		{/if}

		<!-- The action set, per callerRole x type, as icon-only
		     buttons (the icon-first rule; the aria-labels are the
		     actions' own strings): the owner of a virtual album holds
		     the edit dialog (rename + audience + the autonomy block),
		     the cover and the delete; the owner of a directory album
		     the edit dialog (properties: title + the source row +
		     audience + the autonomy block of lock / exclusion /
		     deletability) and the rescan trigger; the owner of the
		     sync album (the default album) holds nothing --
		     system-managed; a viewer the exclusion toggle alone (the
		     departure went with the revoke endpoint it rode; the
		     audience is owner-domain). THE LOCK'S NEW HOME is the
		     edit dialog (E6, §7.9), so the dialog opener stays LIVE
		     under a lock -- it is the unlock's door -- while the
		     other contextual actions freeze honestly: the cover and
		     the delete on the virtual arm, the rescan on the
		     directory arm. -->
		<div class="album-detail__actions ml-auto flex items-center gap-1">
			{#if detail !== null && owner && virtual}
				<IconButton
					disabled={writeBusy}
					aria-label={$_('albums_action_rename')}
					onclick={() => {
						propertiesErrorKey = '';
						propertiesOpen = true;
					}}
				>
					<PenLine size={20} aria-hidden="true" />
				</IconButton>
				<IconButton
					disabled={writeBusy || locked}
					aria-label={$_('albums_action_cover')}
					onclick={() => {
						coverErrorKey = '';
						coverOpen = true;
					}}
				>
					<Images size={20} aria-hidden="true" />
				</IconButton>
				{#if !locked}
					<!-- The permission matrix pins deletion to the UNLOCKED
					     state: the owner unlocks first (the dialog's lock
					     row), one step. -->
					<IconButton
						variant="danger"
						disabled={writeBusy}
						aria-label={$_('albums_action_delete')}
						onclick={() => {
							deleteErrorKey = '';
							deleteOpen = true;
						}}
					>
						<Trash2 size={20} aria-hidden="true" />
					</IconButton>
				{/if}
			{:else if detail !== null && owner && detail.type === 'directory'}
				<IconButton
					disabled={writeBusy}
					aria-label={$_('albums_action_properties')}
					onclick={() => {
						propertiesErrorKey = '';
						propertiesOpen = true;
					}}
				>
					<PenLine size={20} aria-hidden="true" />
				</IconButton>
				<!-- The rescan trigger (E4): the directory album's own
				     refresh -- the three-arm diff run over its source.
				     One run at a time (the endpoint is idempotent, the
				     button is not a spam surface), and a locked album
				     refuses the scan (the 409 the server answers). -->
				<IconButton
					disabled={writeBusy ||
						locked ||
						(rescanRun !== null && (rescanRun.state === 'queued' || rescanRun.state === 'running'))}
					aria-label={$_('albums_rescan_action')}
					onclick={() => void startRescan()}
				>
					<RefreshCw size={20} aria-hidden="true" />
				</IconButton>
			{:else if detail !== null && owner && detail.type === 'sync'}
				<!-- The default album's arm: NOTHING. The sync album is
				     system-managed -- no rename, no cover, no lock, no
				     audience, no delete, no exclusion toggle, no rescan.
				     The arm renders zero affordances, and the server
				     refuses every write face of the id anyway; the wall
				     below is the whole page. -->
			{:else if detail !== null && !owner}
				<!-- The read-only share's own write: the timeline
				     exclusion (the caller's preference) alone. -->
				{#if excludable}
					<IconButton
						disabled={writeBusy}
						aria-label={excluded ? $_('albums_action_include') : $_('albums_action_exclude')}
						onclick={() => void toggleExclusion()}
					>
						{#if excluded}
							<Calendar size={20} aria-hidden="true" />
						{:else}
							<CalendarOff size={20} aria-hidden="true" />
						{/if}
					</IconButton>
				{/if}
			{/if}
		</div>
	</header>

	{#if status === 'loading'}
		<p class="album-detail__loading my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('albums_detail_loading')}
		</p>
	{:else if status === 'error'}
		<div
			class="album-detail__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void convergeDetail()}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{:else if detail !== null}
		<!-- The context block (the page's card-framed text: the
		     notices and the shared error line) -- the
		     WALL below is full-flow edge-to-edge like every wall. -->
		<div class="album-detail__body max-w-[1200px] mx-auto px-2">
			{#if owner && detail.type === 'directory' && rescanRun !== null}
				<!-- The rescan's live progress (the models download page's
				     mechanism, keyed by the run identifier): the counts
				     update in place off the job events, the finish lands
				     the final state, and the membership the scan moved
				     converges through the album events below. -->
				<div
					class="album-detail__rescan my-0 mb-3 flex flex-col gap-1.5"
					aria-live="polite"
					data-testid="album-rescan-progress"
				>
					<p
						class="album-detail__rescan-state m-0 font-body font-medium text-[14px] leading-[1.4] text-text tabular-nums"
					>
						{#if rescanRun.state === 'queued'}
							{$_('albums_rescan_queued')}
						{:else if rescanRun.state === 'running'}
							{$_('albums_rescan_progress', { values: { done: rescanRun.done, total: rescanRun.total } })}
						{:else if rescanRun.state === 'succeeded'}
							{$_('albums_rescan_done')}
						{:else}
							{$_('albums_rescan_failed')}
						{/if}
					</p>
					{#if rescanRun.state === 'running' && rescanRun.total > 0}
						<div
							class="album-detail__rescan-bar w-full h-2 bg-accent-soft rounded-control overflow-hidden animate-progress-in"
							role="progressbar"
							aria-valuemin={0}
							aria-valuemax={rescanRun.total}
							aria-valuenow={rescanRun.done}
							aria-label={$_('albums_rescan_action')}
						>
							<div
								class="album-detail__rescan-fill h-full bg-accent transition-[width] duration-(--motion-base) ease-standard"
								style="width: {Math.min(100, Math.round((rescanRun.done / rescanRun.total) * 100))}%"
							></div>
						</div>
					{/if}
				</div>
			{/if}
			{#if locked && owner}
				<!-- The locked gate, honest: the frozen faces stay off until
				     the unlock (the edit dialog's lock row -- the bar's
				     dialog opener is the door). -->
				<p
					class="album-detail__locked-notice my-0 mb-3 font-body font-medium text-[13px] leading-[1.5] text-text-soft"
					role="status"
				>
					{$_('albums_locked_notice')}
				</p>
			{/if}
			{#if excluded}
				<p
					class="album-detail__excluded-notice my-0 mb-3 font-body font-medium text-[13px] leading-[1.5] text-text-soft"
					role="status"
				>
					{$_('albums_excluded_notice')}
				</p>
			{/if}
			{#if actionErrorKey !== ''}
				<!-- The bar actions' honest failure (the lock, the
				     exclusion, the rescan trigger): the server's own answer. -->
				<p
					class="album-detail__action-error my-0 mb-3 font-body font-medium text-[14px] leading-[1.5] text-danger-text"
					role="alert"
				>
					{errorText(actionErrorKey)}
				</p>
			{/if}
		</div>

		<!-- THE WALL'S CAPABILITY GATE (R5, E6's deletable face): the
		     page's own walk (the keyset client, the SSE convergence,
		     the sentinel) around the shared Wall. The batch runner is
		     wired exactly when the detail's derived `deletable` face
		     holds -- the owner of an unlocked, writable,
		     deletion-enabled directory album curates a DELETE-ONLY
		     selection (the trash action alone); every other face
		     passes no runner, so the tiles are pure navigation and no
		     batch bar ever rises. -->
		<section class="album-detail__wall px-2" aria-busy={$wall.status === 'loading'}>
			{#if wallFirstLoad}
				<p class="album-detail__wall-status my-12 text-center text-text-soft animate-loading-in" role="status">
					{$_('albums_detail_loading')}
				</p>
			{:else if wallSettledEmpty}
				<div
					class="album-detail__wall-empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card"
				>
					<h2 class="text-[22px] mb-2">{titleText}</h2>
					<p class="text-text-soft">{$_('albums_detail_empty')}</p>
				</div>
			{:else if $wall.status === 'error' && $wall.items.length === 0}
				<div
					class="album-detail__wall-failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
					role="alert"
				>
					<p>{errorText($wall.errorKey)}</p>
					<Button variant="primary" onclick={() => void wall.loadMore()}
						><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
					>
				</div>
			{:else}
				<Wall items={$wall.items} actions={['trash']} onbatch={deletable ? runWallBatch : undefined} />
			{/if}

			{#if $wall.status === 'error' && $wall.items.length > 0}
				<div
					class="album-detail__wall-failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
					role="alert"
				>
					<p>{errorText($wall.errorKey)}</p>
					<Button variant="primary" onclick={() => void wall.loadMore()}
						><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
					>
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
		rootPath={detail.rootPath ?? ''}
		shares={detail.shares ?? []}
		initialLocked={detail.locked}
		initialInTimeline={detail.inTimeline}
		sourceWritable={detail.sourceWritable === true}
		initialAllowDeletion={detail.allowAppDeletion ?? true}
		busy={writeBusy}
		errorKey={propertiesErrorKey}
		onsubmit={(draft) => void submitProperties(draft)}
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

<!-- The delete confirm: the album goes, its photo list and shares
     with it -- the photos themselves stay. -->
<Modal bind:open={deleteOpen} title={$_('albums_delete_title')}>
	{#snippet children()}
		<p class="album-detail__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
			{$_('albums_delete_body', { values: { title: detail?.title ?? '' } })}
		</p>
		{#if deleteErrorKey !== ''}
			<p
				class="album-detail__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{errorText(deleteErrorKey)}
			</p>
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
