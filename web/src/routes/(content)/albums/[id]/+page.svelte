<script lang="ts">
	// The album detail (web IA §2-§3): the metadata header with its
	// callerRole x type action set over the CuratedWall grid that
	// pages the album's own timeline (the timeline endpoint's
	// `albumId` narrow -- the album's live members in the timeline's
	// exact shape and keyset order).
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
	//   seam's departure). Read-only is VISUAL: a sharee never sees a
	//   write affordance, the server keeps refusing them.
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
	import { subscribeAlbumEvents } from '$lib/events';
	import { sessionUserId } from '$lib/auth/session.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { wallItems } from '$lib/timeline/wall.svelte';
	import CuratedWall from '$lib/timeline/CuratedWall.svelte';
	import Button from '$lib/components/Button.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import AlbumPropertiesDialog from '$lib/albums/AlbumPropertiesDialog.svelte';
	import CoverPickerDialog from '$lib/albums/CoverPickerDialog.svelte';
	import ShareDialog from '$lib/albums/ShareDialog.svelte';

	const albumId = $derived(page.params.id ?? '');

	type DetailStatus = 'idle' | 'loading' | 'ready' | 'error';

	let detail = $state<AlbumDetail | null>(null);
	let status = $state<DetailStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

	// One write at a time across every action (the dialogs and the
	// header toggles share the flag).
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

	// The exclusion toggle's honest failure (the header owns the line).
	let exclusionErrorKey = $state('');	// The items walk's membership convergence key: album.items-changed
	// bumps it and the CuratedWall restarts from page one.
	let wallResetKey = $state(0);

	const owner = $derived(detail?.callerRole === 'owner');
	const virtual = $derived(detail?.type === 'virtual');
	const locked = $derived(detail?.locked ?? false);
	const excluded = $derived(detail?.inTimeline === false);
	const excludable = $derived(detail?.inTimeline !== undefined);

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

	/** Adopt a write response's wall card into the detail's header fields. */
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
	// card after the edit -- the header adopts its fields.
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
			// The toggle's honest failure: the header owns the line
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
	// plain-button shape the owner's own revoke rows carry). Success
	// leaves for the wall: the album left the caller's view, the
	// detail would 404 from here on. A failure renders through the
	// header's shared error line.
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

	const titleText = $derived(detail?.title ?? $_('albums_title'));
</script>

<svelte:head>
	<title>{titleText} · lumemor</title>
</svelte:head>

<section class="album-detail max-w-[1200px] mx-auto">
	<div class="album-detail__top flex items-center gap-2 mb-1">
		<a
			class="album-detail__back inline-flex items-center px-2.5 py-1.5 font-body font-semibold text-[14px] leading-[1.4] text-text-soft no-underline rounded-control transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/albums"
		>
			<svg viewBox="0 0 24 24" width="16" height="16" fill="none" aria-hidden="true">
				<path d="M14.5 6.5L8 12l6.5 5.5" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" />
			</svg>
			{$_('albums_title')}</a>
	</div>

	{#if status === 'loading'}
		<p class="album-detail__loading my-12 text-center text-text-soft" role="status">{$_('albums_detail_loading')}</p>
	{:else if status === 'error'}
		<div class="album-detail__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void convergeDetail()}>{$_('action_retry')}</Button>
		</div>
	{:else if detail !== null}
		<header class="album-detail__header flex flex-wrap items-center gap-3 mb-2">
			<h1 class="album-detail__title m-0 font-display font-bold text-[24px] leading-[1.3] text-text">{detail.title}</h1>
			{#if locked}
				<span class="album-detail__badge album-detail__badge--locked inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap">
					<svg viewBox="0 0 24 24" width="12" height="12" fill="none" aria-hidden="true">
						<rect x="5" y="10.5" width="14" height="9" rx="2" stroke="currentColor" stroke-width="1.75" />
						<path d="M8 10.5V8a4 4 0 0 1 8 0v2.5" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
					</svg>
					{$_('albums_locked_badge')}
				</span>
			{/if}
			{#if excluded}
				<!-- The muted clock: the caller's own preference keeps this
				     album's photos out of their aggregated timeline. -->
				<span class="album-detail__badge album-detail__badge--excluded inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap">
					<svg viewBox="0 0 24 24" width="12" height="12" fill="none" aria-hidden="true">
						<circle cx="12" cy="12" r="8.25" stroke="currentColor" stroke-width="1.75" />
						<path d="M12 7.5V12l3 2" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" />
					</svg>
					{$_('albums_excluded_badge')}
				</span>
			{/if}
			<span class="album-detail__count ml-1 font-body font-normal text-[13px] leading-[1.4] text-text-soft whitespace-nowrap">{$_('albums_photo_count', { values: { count: detail.itemCount } })}</span>

			<!-- The action set, per callerRole x type: the owner of a
			     virtual album holds the full write set; the owner of a
			     directory album the properties / exclusion / rescan
			     placeholder; a viewer the exclusion toggle and the
			     leave affordance -- both self-affecting writes the
			     share holds. -->
			<div class="album-detail__actions ml-auto flex flex-wrap items-center gap-2">
				{#if owner && virtual}
					<Button variant="ghost" disabled={writeBusy || locked} onclick={() => { propertiesErrorKey = ''; propertiesOpen = true; }}>{$_('albums_action_rename')}</Button>
					<Button variant="ghost" disabled={writeBusy || locked} onclick={() => { coverErrorKey = ''; coverOpen = true; }}>{$_('albums_action_cover')}</Button>
					<Button variant="ghost" disabled={writeBusy} onclick={() => void toggleLock()}>{locked ? $_('albums_action_unlock') : $_('albums_action_lock')}</Button>
					<Button variant="ghost" onclick={() => { shareErrorKey = ''; shareOpen = true; }}>{$_('albums_action_share')}</Button>
					<Button variant="ghost" disabled={writeBusy || locked} onclick={() => void goto(`/timeline?album=${encodeURIComponent(albumId)}`)}>{$_('albums_action_add_photos')}</Button>
					<Button variant="danger" disabled={writeBusy} onclick={() => { deleteErrorKey = ''; deleteOpen = true; }}>{$_('albums_action_delete')}</Button>
				{:else if owner && detail.type === 'directory'}
					<Button variant="ghost" disabled={writeBusy || locked} onclick={() => { propertiesErrorKey = ''; propertiesOpen = true; }}>{$_('albums_action_properties')}</Button>
					{#if excludable}
						<Button variant="ghost" disabled={writeBusy} onclick={() => void toggleExclusion()}>{excluded ? $_('albums_action_include') : $_('albums_action_exclude')}</Button>
					{/if}
					<!-- The rescan placeholder: honest and disabled -- the
					     directory album's own flow arrives with a later
					     milestone, and the tooltip says so. -->
					<Button variant="ghost" disabled title={$_('albums_rescan_note')} aria-describedby="album-detail-rescan-note">{$_('albums_rescan_action')}</Button>
				{:else if !owner}
					<!-- The read-only share's own writes: the timeline
					     exclusion (the caller's preference) and the
					     departure (the caller's own share row). -->
					{#if excludable}
						<Button variant="ghost" disabled={writeBusy} onclick={() => void toggleExclusion()}>{excluded ? $_('albums_action_include') : $_('albums_action_exclude')}</Button>
					{/if}
					<Button variant="ghost" disabled={writeBusy} onclick={() => void leaveAlbum()}>{$_('albums_leave_action')}</Button>
				{/if}
			</div>
		</header>
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
			<!-- The header toggles' honest failure (the lock, the
			     exclusion): the server's own answer. -->
			<p class="album-detail__action-error my-0 mb-3 font-body font-medium text-[14px] leading-[1.5] text-danger-text" role="alert">{errorText(exclusionErrorKey)}</p>
		{/if}

		<CuratedWall
			path="/api/v1/assets/timeline"
			view="timeline"
			albumId={albumId}
			heading={detail.title}
			loadingKey="albums_detail_loading"
			emptyKey="albums_detail_empty"
			actions={['favorite']}
			resetKey={wallResetKey}
		/>
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
		<Button variant="ghost" disabled={writeBusy} onclick={() => (deleteOpen = false)}>{$_('common_cancel')}</Button>
		<Button variant="danger" disabled={writeBusy} onclick={() => void confirmDelete()}>
			{$_('albums_delete_confirm')}
		</Button>
	{/snippet}
</Modal>
