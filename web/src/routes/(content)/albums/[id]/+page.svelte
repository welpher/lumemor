<script lang="ts">
	// The album detail: the metadata context with its
	// callerRole x type action set over the wall that pages the
	// album's own timeline (the timeline endpoint's `albumId`
	// narrow -- the album's live members in the timeline's exact
	// shape and keyset order).
	//
	// THE MEMORY HAND-OVER: a memory-typed id never renders here --
	// the memory's own route owns its whole face (the curation bar,
	// the removal selection, the member's read-only view), so the
	// detail read that answers `type: memory` redirects there and
	// this page's arms serve the CONTAINERS alone (the directory
	// album's properties/rescan/remove, the sync album's read-only
	// grid, the viewer's exclusion toggle).
	//
	// THE CONTEXTUAL BAR: the page
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
	// THE WALL'S CURATION IS THE ROLE'S (the selection flow's
	// capability gate): the runner's presence is the capability, so
	// only the faces that can write wire one -- the owner of a
	// deletable directory album (the delete-only selection) and a
	// member (their own per-user favorite marks). Every other face
	// (the sync album, the owner's read-only source) passes none: no
	// hover check renders and no batch bar exists, the tiles pure
	// click-to-viewer navigation. Nothing is rendered as a disabled
	// button -- an action a caller cannot take is absent. The wall's
	// data walk (the keyset client over the album narrow, the SSE
	// convergence, the sentinel, the wallItems publication) is the
	// PAGE's own concern here, the same shape the timeline page owns
	// for its walk.
	//
	// THE PINNED CHOICES (the albums-as-sources reshape, completed
	// by the autonomy face, simplified by the lock's retirement):
	//
	// - The action set follows the role and the type: the owner of a
	//   directory album holds the properties (the edit dialog: title
	//   + the read-only source row + the audience + the autonomy
	//   block), the rescan trigger (the three-arm diff run, its
	//   progress surfaced through the job events the run's
	//   identifier keys) and the remove (the un-ingest, behind its
	//   semantics-stating confirm); a viewer
	//   holds the exclusion toggle (their own preference) alone --
	//   the sharee's departure went with the revoke endpoint it rode
	//   (the audience write is owner-domain; the member side is the
	//   read-only view). The owner of a SYNC album (the default
	//   "My photos" album) holds NOTHING: system-managed, read-only
	//   -- the explicit empty arm below, zero write affordances, and
	//   the server refuses every write face of the id anyway.
	//   Read-only is VISUAL: the member view opens exactly one
	//   writable thing -- the caller's own timeline preference --
	//   and everything the owner domain owns (the properties, the
	//   cover, the audience, the remove) is absent, never greyed.
	// - THE WALL'S ACTION SUBSET: the owner of a DELETABLE directory
	//   album (owner AND writable source AND the deletability switch
	//   -- computed server-side as `deletable`, pure data) curates a
	//   delete-only selection; a member curates the per-user
	//   favorite toggle alone. The hide switch is the owner's own
	//   row, and the batch resolves it against the caller's OWNED
	//   ids -- over another account's album it resolves nothing --
	//   so it never renders here: the bar offers what the write can
	//   actually land. The per-user favorite is the arm that
	//   resolves the caller's VISIBLE ids, so it lands for a member
	//   like it does on the aggregate timeline.
	// - Every write adopts its own response (the property patch
	//   carries the wall card after the edit, the preference
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
	import { publishWallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import Button from '$lib/components/Button.svelte';
	import StateCard from '$lib/components/StateCard.svelte';
	import UnreachableCard from '$lib/components/UnreachableCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import AlbumPropertiesDialog, { type AlbumAutonomyDraft } from '$lib/albums/AlbumPropertiesDialog.svelte';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import PenLine from '@lucide/svelte/icons/pen-line';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import CalendarOff from '@lucide/svelte/icons/calendar-off';
	import Calendar from '@lucide/svelte/icons/calendar';
	import X from '@lucide/svelte/icons/x';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	const albumId = $derived(page.params.id ?? '');

	type DetailStatus = 'idle' | 'loading' | 'ready' | 'error';

	let detail = $state<AlbumDetail | null>(null);
	let status = $state<DetailStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');
	/**
	 * The dead end: the detail read answered the byte-identical 404
	 * (or a 403). The page renders the one unreachable face then --
	 * no retry, no title, nothing that would say whether the album
	 * exists -- and the bar keeps the back affordance alone.
	 */
	let unreachable = $state(false);

	// One write at a time across every action (the dialogs and the
	// bar toggles share the flag).
	let writeBusy = $state(false);

	// The properties dialog (the edit: title + source + audience).
	let propertiesOpen = $state(false);
	let propertiesErrorKey = $state('');

	// The remove confirm (the directory album's un-ingest): the
	// semantics say themselves -- the album and its photo data go,
	// the source folder stays.
	let removeOpen = $state(false);
	let removeErrorKey = $state('');

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
	// when this page wires the batch runner below (the role's
	// action subset): every other face is pure navigation.
	const wall = untrack(() => createTimelineClient({ path: '/api/v1/assets/timeline', view: 'timeline', albumId }));
	let wallResetKey = $state(0);

	const owner = $derived(detail?.callerRole === 'owner');
	const excluded = $derived(detail?.inTimeline === false);
	const excludable = $derived(detail?.inTimeline !== undefined);
	// The wall's delete affordance: the server-derived face --
	// owner AND writable source AND the switch, pure
	// data -- decides whether this wall curates at all.
	const deletable = $derived(owner && detail?.type === 'directory' && detail.deletable === true);

	// The wall's action subset, per the caller's role (see the
	// header's pins): the owner's deletable directory album trashes
	// (the photos are theirs), a member toggles their own favorites
	// (the per-user arm the batch resolves over the caller's visible
	// ids). The arrays are constants so the bar's subset keeps a
	// stable identity across re-renders.
	const OWNER_WALL_ACTIONS: readonly BatchAction[] = ['trash'];
	const MEMBER_WALL_ACTIONS: readonly BatchAction[] = ['favorite'];
	const wallCuration = $derived.by<{ actions: readonly BatchAction[] } | undefined>(() => {
		if (detail === null) {
			return undefined;
		}
		if (detail.callerRole === 'owner') {
			return deletable ? { actions: OWNER_WALL_ACTIONS } : undefined;
		}
		return { actions: MEMBER_WALL_ACTIONS };
	});

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
			// THE MEMORY HAND-OVER: a memory-typed id bounces to its
			// own route (the memory's curation face); this page keeps
			// rendering nothing for it.
			if (outcome.detail.type === 'memory') {
				detail = null;
				status = 'loading';
				void goto(`/memories/${encodeURIComponent(outcome.detail.id)}`);
				return;
			}
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
		unreachable = false;
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

	/** Adopt a write response's wall card into the detail's fields. */
	function applyWriteFields(next: { title: string; itemCount: number }): void {
		if (detail !== null) {
			detail = {
				...detail,
				title: next.title,
				itemCount: next.itemCount
			};
		}
	}

	// The properties write (the edit dialog): the page diffs the
	// whole draft against the LIVE detail and writes only the changed
	// halves -- a same-value title PATCH would still publish its
	// album.updated event, and a same-set audience PUT is the
	// contract's silent no-op, so a no-change save writes nothing at
	// all. THE WRITE SEQUENCE is the plain diff's order: the
	// property fields (title, the deletability switch) ride ONE
	// patch, the audience PUT and the caller's own timeline
	// preference follow. Each write's PATCH response is the album's
	// wall card after the edit (the fields adopt it); the final
	// detail refetch converges the faces the cards never carry (the
	// grantee list, the deletability state, the derived affordance).
	async function submitProperties(draft: AlbumAutonomyDraft): Promise<void> {
		const id = untrack(() => albumId);
		if (writeBusy || detail === null || id === '') {
			return;
		}
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
		if (!wantsTitle && !wantsDeletable && !wantsAudience && !wantsExclusion) {
			propertiesOpen = false;
			return;
		}
		writeBusy = true;
		propertiesErrorKey = '';
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

	// The rescan trigger: POST the run, adopt the identifier it
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
			// The trigger's honest failure (a dead session): the bar's
			// shared error line.
			actionErrorKey = outcome.errorKey;
			return;
		}
		// An instant scan can finish BEFORE this answer lands (the
		// run's whole life is milliseconds on a warm stack and a small
		// source): the handler below buffered its events by run id, so
		// the line adopts the settled state instead of sitting at
		// "queued" forever.
		const tracked = outcome.run.jobRunId;
		rescanRun = settledRuns.get(tracked) ?? { id: tracked, state: 'queued', total: 0, done: 0, failed: 0 };
		if (settledRuns.has(tracked)) {
			settledRuns.delete(tracked);
		}
	}

	// The run events that arrived before the trigger's answer named
	// their run: keyed by the run id, a bounded few (the event handler
	// puts one here and the trigger's answer takes it back).
	const settledRuns = new Map<string, RescanRun>();

	// The SSE convergence of the run: progress events update the run
	// in place (the events' counts are already numbers, the state
	// already the run lifecycle's), the finish lands the final state.
	// Only the run THIS page tracked applies -- another album's
	// rescan (or any other job kind) never touches this line -- and a
	// run whose events beat the trigger's answer is buffered for it.
	$effect(() => {
		return subscribeJobEvents((event) => {
			if (event.kind !== 'album_rescan') {
				return;
			}
			const next: RescanRun = {
				id: event.jobRunId,
				state: event.state,
				total: event.total,
				done: event.done,
				failed: event.failed
			};
			const run = rescanRun;
			if (run === null || run.id !== event.jobRunId) {
				// Bound the buffer: only the runs racing this trigger
				// matter, and a page lives with a handful of them.
				if (settledRuns.size >= 8) {
					const oldest = settledRuns.keys().next().value;
					if (oldest !== undefined) {
						settledRuns.delete(oldest);
					}
				}
				settledRuns.set(event.jobRunId, next);
				return;
			}
			rescanRun = next;
		});
	});

	// The remove write (the directory album's un-ingest): the same
	// endpoint, the directory arm. Success leaves for the wall; the
	// members' own convergence (the tiles dropping everywhere) rides
	// the asset.deleted events the removal publishes.
	async function confirmRemove(): Promise<void> {
		const id = untrack(() => albumId);
		if (writeBusy || detail === null || id === '') {
			return;
		}
		writeBusy = true;
		removeErrorKey = '';
		const outcome = await deleteAlbum(id);
		writeBusy = false;
		if (!outcome.ok) {
			removeErrorKey = outcome.errorKey;
			return;
		}
		removeOpen = false;
		void goto('/albums');
	}

	// The wall's batch runner: the page's own client runs the
	// action subset the wall's bar offers -- the owner's delete-only
	// pass, or a member's per-user favorite toggle -- through the
	// same view-aware optimistic convergence every curatable wall
	// uses (a favorite flips its flag at once and stays silent with
	// the per-user write, the trashed tiles leave immediately, and
	// the SSE events of a landed pass confirm what the optimism
	// already showed). The wall owns the busy, the mapped error and
	// the selection choreography around it.
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
	<!-- THE CONTEXTUAL BAR: back + the title + the type x role
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
		     actions' own strings): the owner of a directory album
		     the edit dialog (properties: title + the source row +
		     audience + the autonomy block of exclusion /
		     deletability), the rescan trigger and the remove (the
		     un-ingest); the owner of the
		     sync album (the default album) holds nothing --
		     system-managed; a viewer the exclusion toggle alone -- the
		     ONE writable thing the member view holds (their own
		     timeline preference; the audience is owner-domain, and
		     the departure went with the revoke endpoint it rode). A
		     memory id never reaches this bar: the detail read that
		     answers one redirects to the memory's own route before
		     the arms render. -->
		<div class="album-detail__actions ml-auto flex items-center gap-1">
			{#if detail !== null && owner && detail.type === 'directory'}
				<IconButton
					disabled={writeBusy}
					aria-label={$_('albums_action_properties')}
					title={$_('albums_action_properties')}
					onclick={() => {
						propertiesErrorKey = '';
						propertiesOpen = true;
					}}
				>
					<PenLine size={20} aria-hidden="true" />
				</IconButton>
				<!-- The rescan trigger: the directory album's own
				     refresh -- the three-arm diff run over its source.
				     One run at a time (the endpoint is idempotent, the
				     button is not a spam surface). -->
				<IconButton
					disabled={writeBusy ||
						(rescanRun !== null && (rescanRun.state === 'queued' || rescanRun.state === 'running'))}
					aria-label={$_('albums_rescan_action')}
					title={$_('albums_rescan_action')}
					onclick={() => void startRescan()}
				>
					<RefreshCw size={20} aria-hidden="true" />
				</IconButton>
				<!-- The remove action: the un-ingest. The confirm dialog
				     states the semantics before the irreversible write --
				     the album and its photo data go, the source folder
				     stays. The sync album has no such action anywhere
				     (its own arm below renders nothing). -->
				<IconButton
					variant="danger"
					disabled={writeBusy}
					aria-label={$_('albums_action_remove')}
					title={$_('albums_action_remove')}
					onclick={() => {
						removeErrorKey = '';
						removeOpen = true;
					}}
				>
					<Trash2 size={20} aria-hidden="true" />
				</IconButton>
			{:else if detail !== null && owner && detail.type === 'sync'}
				<!-- The default album's arm: NOTHING. The sync album is
				     system-managed -- no rename, no cover, no
				     audience, no delete, no exclusion toggle, no rescan.
				     The arm renders zero affordances, and the server
				     refuses every write face of the id anyway; the wall
				     below is the whole page. -->
			{:else if detail !== null && !owner}
				<!-- The member view's ONE action: the timeline
				     preference -- the caller's own switch, and the
				     only writable thing a shared album gives them.
				     The label and the glyph follow the state (in the
				     timeline -> take it out; excluded -> keep it in).
				     Nothing else renders here: no properties, no
				     cover, no audience, no remove -- absent, never
				     disabled. -->
				{#if excludable}
					<IconButton
						disabled={writeBusy}
						aria-label={excluded ? $_('albums_action_include') : $_('albums_action_exclude')}
						title={excluded ? $_('albums_action_include') : $_('albums_action_exclude')}
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
		<LoadingLine label={$_('albums_detail_loading')} />
	{:else if status === 'error' && unreachable}
		<!-- The dead end: one card, no title, no count, no cover --
		     nothing that would distinguish a deleted album from one
		     this caller may not see. The bar above keeps its way
		     back alone. -->
		<UnreachableCard />
	{:else if status === 'error'}
		<StateCard
			tone="error"
			body={errorText(errorKey)}
			actionLabel={$_('action_retry')}
			onaction={() => void convergeDetail()}
		/>
	{:else if detail !== null}
		<!-- The context block (the page's card-framed text: the
		     notices and the shared error line) -- the
		     WALL below is full-flow edge-to-edge like every wall
		     (the wall is full width + 8px margins +
		     the reserved 72px month-axis slot, so this column takes
		     the same band and the page's own text never runs under
		     the axis). -->
		<div class="album-detail__body px-2 pe-[72px]">
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
			{#if excluded}
				<p
					class="album-detail__excluded-notice my-0 mb-3 font-body font-medium text-[13px] leading-[1.5] text-text-soft"
					role="status"
				>
					{$_('albums_excluded_notice')}
				</p>
			{/if}
			{#if actionErrorKey !== ''}
				<!-- The bar actions' honest failure (the
				     exclusion, the rescan trigger): the server's own answer. -->
				<p
					class="album-detail__action-error my-0 mb-3 font-body font-medium text-[14px] leading-[1.5] text-danger-text"
					role="alert"
				>
					{errorText(actionErrorKey)}
				</p>
			{/if}
		</div>

		<!-- THE WALL'S CAPABILITY GATE (the role's action subset): the
		     page's own walk (the keyset client, the SSE convergence,
		     the sentinel) around the shared Wall. The batch runner is
		     wired exactly when the role's subset holds -- the owner of
		     a writable, deletion-enabled directory album curates a
		     DELETE-ONLY selection, a member the per-user favorite
		     toggle alone; every other face passes no runner, so the
		     tiles are pure navigation and no batch bar ever rises.
		     The subset is what the write can land: the bar renders no
		     action the batch would refuse. -->
		<section class="album-detail__wall px-2 pe-[72px]" aria-busy={$wall.status === 'loading'}>
			{#if wallFirstLoad}
				<LoadingLine label={$_('albums_detail_loading')} />
			{:else if wallSettledEmpty}
				<StateCard title={$_('albums_detail_empty_title')} body={$_('albums_detail_empty_body')} />
			{:else if $wall.status === 'error' && $wall.items.length === 0}
				<StateCard
					tone="error"
					body={errorText($wall.errorKey)}
					actionLabel={$_('action_retry')}
					onaction={() => void wall.loadMore()}
				/>
			{:else}
				<Wall
					items={$wall.items}
					actions={wallCuration?.actions}
					onbatch={wallCuration !== undefined ? runWallBatch : undefined}
				/>
			{/if}

			{#if $wall.status === 'error' && $wall.items.length > 0}
				<StateCard
					tone="error"
					body={errorText($wall.errorKey)}
					actionLabel={$_('action_retry')}
					onaction={() => void wall.loadMore()}
				/>
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

{#if detail !== null && owner && detail.type === 'directory'}
	<!-- The owner faces mount for the owner's directory album alone:
	     a member's view carries no properties dialog and no remove
	     confirm at all (their entry buttons never render, and the
	     dialogs are owner-domain surfaces, not a viewer's hidden
	     pages). -->
	<AlbumPropertiesDialog
		bind:open={propertiesOpen}
		titleKey="albums_properties_title"
		submitKey="albums_properties_submit"
		initialTitle={detail.title}
		rootPath={detail.rootPath ?? ''}
		shares={detail.shares ?? []}
		initialInTimeline={detail.inTimeline}
		sourceWritable={detail.sourceWritable === true}
		initialAllowDeletion={detail.allowAppDeletion ?? true}
		busy={writeBusy}
		errorKey={propertiesErrorKey}
		onsubmit={(draft) => void submitProperties(draft)}
	/>
{/if}

<!-- The remove confirm (the directory album's un-ingest): the
     pinned semantics in the body -- the album and its photo data
     go, the source folder stays untouched. Owner domain: mounted for
     the owner's directory album alone, like the properties above. -->
{#if detail !== null && owner && detail.type === 'directory'}
	<Modal bind:open={removeOpen} title={$_('albums_remove_title')}>
		<p class="album-detail__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
			{$_('albums_remove_body')}
		</p>
		{#if removeErrorKey !== ''}
			<p
				class="album-detail__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{errorText(removeErrorKey)}
			</p>
		{/if}
		{#snippet footer()}
			<Button variant="ghost" disabled={writeBusy} onclick={() => (removeOpen = false)}>
				<X size={20} aria-hidden="true" />
				{$_('common_cancel')}
			</Button>
			<Button variant="danger" disabled={writeBusy} onclick={() => void confirmRemove()}>
				<Trash2 size={20} aria-hidden="true" />
				{$_('albums_remove_confirm')}
			</Button>
		{/snippet}
	</Modal>
{/if}
