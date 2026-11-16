<script lang="ts">
	// The album wall: ONE cover-card grid -- the caller's own cards (the
	// pinned sync card of the default "My photos" album first) and the
	// albums shared to them, mixed by the cards' own recency (the
	// single list: no own/shared sections, no group labels) --
	// with the collage covers (the pinned cover full-bleed, otherwise
	// the newest members: two split, three the 2+1 fall, four the
	// quad) and one badge row under each count: a shared card carries
	// its owner's attribution beside the read-only badge, an own card
	// its shared / excluded state (the attribution names
	// the person, the email only falls back). THE MEMORY ROWS ARE NOT
	// THIS WALL'S: the reference sets moved to their own route (the
	// memories wall), so the grid filters the memory type out. Card
	// clicks open the album's
	// detail. The sync card's title is the system's internal label
	// and never renders: `type === 'sync'` keys the localized
	// display name (album_all_photos), the contract's client-side
	// convention.
	//
	// THE CREATION ENTRY IS THE ADMINISTRATOR'S : pointing at the file system is configuration,
	// so the "New album" button renders for an admin's session alone
	// -- a member never sees it (the server refuses their POST with
	// 403 regardless; the visual gate is the honest half). Members
	// contribute photos through upload and sync, never through album
	// creation.
	//
	// SSE CONVERGENCE: every album.* event converges through ONE
	// fresh wall fetch, collapsed single-flight (a burst parks in a
	// queue and costs one trailing fetch). The refetched SERVER
	// answer decides what changed -- an event for an album this wall
	// never sees refetches the same answer and the page invents no
	// card. A failed convergence keeps the last good state silently
	// (the next event retries); only the initial load's failure
	// raises the error box.
	import { _ } from 'svelte-i18n';
	import { listAlbums, createAlbum, type AlbumSummary, type AlbumAudienceSkip } from '$lib/api/albums';
	import { subscribeAlbumEvents } from '$lib/events';
	import { sessionRole } from '$lib/auth/session.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { ambiguousLabels, displayNameOf } from '$lib/identity';
	import Button from '$lib/components/Button.svelte';
	import StateCard from '$lib/components/StateCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import CreateAlbumDialog, { type CreateAlbumSwitches } from '$lib/albums/CreateAlbumDialog.svelte';
	import { toast } from '$lib/components/toast.svelte';
	import { albumCoverCell, albumCoverGrid } from '$lib/albums/cover-layout';
	import Plus from '@lucide/svelte/icons/plus';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	let albums = $state<AlbumSummary[]>([]);
	let status = $state<ListStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

	// The creation dialog (title + source path + audience); the
	// created card lands through the refetch the write triggers, and
	// the write's honest answers -- the `skipped` report, the write
	// probe's `sourceWritable` fact (a read-only source is a
	// legitimate album, the fact is creation feedback, never a
	// failure) -- hold the dialog open until acknowledged.
	let createOpen = $state(false);
	let createBusy = $state(false);
	let createErrorKey = $state('');
	let createSkipped = $state<readonly AlbumAudienceSkip[] | null>(null);
	let createWritable = $state<boolean | null>(null);

	// The admin entry: the session's own role read, the drawer's
	// pattern -- components read the store directly, no prop
	// shadowing. A member's wall renders no creation affordance at
	// all.
	const adminSession = $derived(sessionRole() === 'admin');

	// The single-flight guard: while one fetch runs, later triggers
	// park (the burst's tail) and one trailing fetch runs when it
	// lands -- the same collapse the timeline client applies.
	let running = false;
	let queued = false;

	/**
	 * Fetch the wall and apply it. `converge` marks an SSE-driven
	 * run: the visible state stays (the cards swap in place on
	 * success) and a failure keeps the last good answer instead of
	 * raising the error box -- later events retry.
	 */
	async function runLoad(converge: boolean): Promise<void> {
		if (running) {
			queued = true;
			return;
		}
		running = true;
		if (!converge) {
			status = 'loading';
		}
		try {
			const outcome = await listAlbums();
			if (outcome.ok) {
				albums = outcome.albums;
				errorKey = '';
				status = 'ready';
			} else if (!converge) {
				status = 'error';
				errorKey = outcome.errorKey;
			}
		} finally {
			running = false;
			if (queued) {
				queued = false;
				void runLoad(true);
			}
		}
	}

	// The wall's ONE list: the album cards (the memory rows leave --
	// the reference sets' home is the memories wall), the pinned sync
	// card first and everything else by the card's own `updatedAt`
	// descending (own and shared mixed, most recently
	// updated first; the sort is stable, so the server's arm order
	// breaks equal timestamps).
	const albumCards = $derived.by(() => {
		const cards = albums.filter((album) => album.type !== 'memory');
		const sync = cards.filter((album) => album.type === 'sync');
		const rest = cards
			.filter((album) => album.type !== 'sync')
			.sort((left, right) => Date.parse(right.updatedAt) - Date.parse(left.updatedAt));
		return [...sync, ...rest];
	});

	// The attribution chips' disambiguator: a display name TWO accounts
	// carry gets the email as its hover title (the name is not a key).
	// The check rides the cards this wall renders -- only a name the
	// reader can see twice is ambiguous here.
	const ambiguousOwners = $derived(
		ambiguousLabels(albumCards.flatMap((album) => (album.owner === undefined ? [] : [album.owner])))
	);

	const firstLoad = $derived(albums.length === 0 && status === 'loading');
	// The honest empty: no ALBUM row at all (a caller whose wall holds
	// only memories still sees the empty state here -- their
	// reference sets live on the memories wall).
	const settledEmpty = $derived(albumCards.length === 0 && status === 'ready');

	// The bar's count: the album CARDS this wall holds (the memory rows
	// excluded -- memories count on their own page).
	const albumCount = $derived(albumCards.length);

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// The card's display title: the localized default-album name for
	// the pinned sync card, the stored title for every other album
	// (the client-side display-name convention of the contract -- no
	// guessing beyond the type).
	function cardTitle(album: AlbumSummary): string {
		return album.type === 'sync' ? $_('album_all_photos') : album.title;
	}

	// The shared card's attribution: the owner's display name, the email
	// when the account set none (a blank chip never renders).
	function ownerLabel(album: AlbumSummary): string {
		return album.owner === undefined ? '' : displayNameOf(album.owner);
	}

	// The attribution chip's hover title: the owner's email when another
	// account of this wall carries the same name -- the reader then has
	// a way to tell them apart.
	function ownerTitle(album: AlbumSummary): string | undefined {
		return album.owner !== undefined && ambiguousOwners.has(ownerLabel(album)) ? album.owner.email : undefined;
	}

	// The cover layout class of one card: the pinned cover and the
	// single newest member render full-bleed, two split, three the
	// 2+1 fall, four the quad; a bare album renders the placeholder.
	function coverClass(count: number): string {
		switch (count) {
			case 0:
				return 'album-card__cover--empty';
			case 1:
				return 'album-card__cover--single';
			case 2:
				return 'album-card__cover--split';
			case 3:
				return 'album-card__cover--duo-plus-one';
			default:
				return 'album-card__cover--quad';
		}
	}

	// The creation write: the response card IS the wall's truth for
	// the new album, but the refetch the write triggers places it in
	// the server's order -- the SSE album.created echo converges the
	// same way (the single-flight collapses the pair). The write's
	// honest `skipped` report (the requested audience ids that did
	// not land) holds the dialog open until acknowledged; an
	// all-landed writable creation closes it directly. The two
	// autonomy switches ride the POST: the timeline exclusion (the
	// creator's own preference, landed in the creation) and the
	// deletability switch (present exactly when the picked
	// completion row's write fact rendered the row).
	async function submitCreate(
		title: string,
		rootPath: string,
		audience: readonly { userId: string; email: string }[],
		switches: CreateAlbumSwitches
	): Promise<void> {
		if (createBusy) {
			return;
		}
		createBusy = true;
		createErrorKey = '';
		const outcome = await createAlbum({
			title,
			rootPath,
			audience: audience.map((member) => member.userId),
			allowAppDeletion: switches.allowAppDeletion,
			inTimeline: switches.inTimeline
		});
		createBusy = false;
		if (!outcome.ok) {
			createErrorKey = outcome.errorKey;
			return;
		}
		createSkipped = outcome.skipped;
		createWritable = outcome.sourceWritable;
		void runLoad(true);
		// The creation's confirmation is the toast (the sample's
		// create button is the design): the clean write says so, and
		// the read-only source's fact rides the SAME organ -- the
		// toast text carries both sentences, one acknowledgment,
		// never two. The skipped-audience report alone still holds
		// the dialog: a LIST of who did not land is the dialog's
		// business (with the read-only fact beside it when both
		// answers came back), and a toast cannot carry a list.
		if (outcome.skipped.length === 0) {
			createOpen = false;
			toast($_(outcome.sourceWritable ? 'albums_created' : 'albums_create_readonly'));
		}
	}

	// The report's done: close the dialog and clear the report (the
	// wall behind already converged through the write's refetch).
	function finishCreate(): void {
		createOpen = false;
		createSkipped = null;
		createWritable = null;
	}

	// The load and the SSE subscription live for the page's lifetime:
	// the first subscriber opens the stream, leaving the page closes
	// it, and every album event converges through a refetch.
	$effect(() => {
		void runLoad(false);
		const unsubscribe = subscribeAlbumEvents(() => {
			void runLoad(true);
		});
		return () => unsubscribe();
	});
</script>

<svelte:head>
	<title>{$_('albums_title')} · lumemor</title>
</svelte:head>

<!-- The card grid's container ): full width + 16px margins --
     a wider window adds columns, the cards' own size stays bounded. -->
<section class="albums px-4 pt-4 pb-12" aria-busy={status === 'loading'}>
	<!-- THE ONE ROW: the page title with its count on the left, the
	     page-level action on the right, one divider under the row.
	     The count is the title's own (it never takes a row of its
	     own), and it renders only on a settled answer -- a loading
	     face counts nothing. The creation is the ADMIN'S ALONE
	     (mounting an existing photo directory is configuration, a
	     member never sees the affordance). -->
	<div class="albums__toolbar flex items-center gap-3 pb-4 border-b border-border">
		<h1 class="albums__title m-0 font-display font-bold text-[20px] leading-[1.3] text-text">
			{$_('albums_title')}
			{#if status === 'ready'}
				<span
					class="albums__title-count font-body font-normal text-[15px] leading-[1.4] text-text-soft tabular-nums"
				>
					· {$_('albums_bar_count', { values: { count: albumCount } })}
				</span>
			{/if}
		</h1>
		{#if adminSession}
			<div class="albums__actions ml-auto flex items-center gap-2">
				<Button
					onclick={() => {
						createErrorKey = '';
						createSkipped = null;
						createWritable = null;
						createOpen = true;
					}}><Plus size={20} aria-hidden="true" />{$_('albums_new')}</Button
				>
			</div>
		{/if}
	</div>

	<div class="albums__content pt-4">
		{#if firstLoad}
			<LoadingLine label={$_('albums_loading')} />
		{:else if settledEmpty}
			<!-- The no-data card offers the way in, and only to a
			     caller the creation write can land (the create endpoint
			     is the administrator's): a member's card carries no
			     action at all. -->
			<StateCard
				title={$_('albums_empty_title')}
				body={$_('albums_empty_body')}
				actionLabel={adminSession ? $_('albums_new') : ''}
				onaction={() => {
					createErrorKey = '';
					createSkipped = null;
					createWritable = null;
					createOpen = true;
				}}
			/>
		{:else if status === 'error' && albumCards.length === 0}
			<StateCard
				tone="error"
				body={errorText(errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void runLoad(false)}
			/>
		{:else}
			<!-- ONE grid: own and shared cards mixed by recency, no group
			     labels. -->
			<div class="albums__grid grid grid-cols-[repeat(auto-fill,minmax(min(220px,100%),1fr))] gap-3">
				{#each albumCards as album (album.id)}
					{@render card(album)}
				{/each}
			</div>
		{/if}

		{#if status === 'error' && albumCards.length > 0}
			<!-- A failed convergence while cards exist: the page stays, the
		     next event retries; the box keeps the retry affordance. -->
			<StateCard
				tone="error"
				body={errorText(errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void runLoad(false)}
			/>
		{/if}
	</div>
</section>

{#snippet card(album: AlbumSummary)}
	<!-- One album card, three lines: the title, the count, and the badge
	     row -- a shared card's attribution beside its read-only badge, an
	     own card's shared / excluded state. The
	     badge row renders even when empty: the cards stay equal height. -->
	<a
		class="album-card flex flex-col overflow-hidden rounded-card bg-card border border-border no-underline text-text cursor-pointer transition-[border-color,box-shadow] duration-(--motion-fast) ease-standard hover:border-accent hover:shadow-card"
		href="/albums/{encodeURIComponent(album.id)}"
	>
		<span
			class="album-card__cover {coverClass(
				album.coverAssetIds.length
			)} relative block aspect-[4/3] bg-accent-soft"
			role={album.coverAssetIds.length > 0 ? 'img' : undefined}
			aria-label={album.coverAssetIds.length > 0
				? $_('cover_collage_label', { values: { count: album.coverAssetIds.length } })
				: undefined}
			aria-hidden={album.coverAssetIds.length === 0 ? 'true' : undefined}
		>
			{#if album.coverAssetIds.length === 1}
				<img
					class="absolute inset-0 w-full h-full object-cover"
					src={`/api/v1/assets/${encodeURIComponent(album.coverAssetIds[0] ?? '')}/thumbnail?size=tile`}
					alt=""
					loading="lazy"
					decoding="async"
					draggable="false"
				/>
			{:else if album.coverAssetIds.length > 1}
				<!-- The collage cells: the 2+1 fall's third cell spans both
				     columns, the quad fills the 2x2 grid (the cover's
				     degradation ladder, see lib/albums/cover-layout). -->
				<span class="album-card__cells absolute inset-0 {albumCoverGrid(album.coverAssetIds.length)}">
					{#each album.coverAssetIds.slice(0, 4) as coverId, index (coverId)}
						<span
							class="album-card__cell {albumCoverCell(album.coverAssetIds.length, index)} overflow-hidden"
						>
							<img
								class="w-full h-full object-cover"
								src={`/api/v1/assets/${encodeURIComponent(coverId)}/thumbnail?size=tile`}
								alt=""
								loading="lazy"
								decoding="async"
								draggable="false"
							/>
						</span>
					{/each}
				</span>
			{:else}
				<!-- A bare album: the honest placeholder, no requests. -->
				<span class="album-card__placeholder absolute inset-0 grid place-items-center text-text-faint">
					<svg viewBox="0 0 24 24" width="28" height="28" fill="none">
						<path
							d="M4 7c0-1.1.9-2 2-2l1.7.0a2 2 0 0 1 1.7 1l.8 1.4a2 2 0 0 0 1.7 1H18a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V7Z"
							stroke="currentColor"
							stroke-width="1.5"
							stroke-linejoin="round"
						/>
					</svg>
				</span>
			{/if}
		</span>
		<span class="album-card__caption flex flex-col gap-0.5 px-3 py-2.5">
			<span class="album-card__title font-body font-semibold text-[15px] leading-[1.4] text-text truncate"
				>{cardTitle(album)}</span
			>
			<!-- The card's count is the PHOTO count (the glyph is the
			     image glyph and the title says "photos"): videos are
			     members but never photos, so the number beside this
			     card's cover is `photoCount`, not the live-member
			     `itemCount` the cover pool consults. -->
			<CountLabel
				class="album-card__count font-body font-normal text-[13px] leading-[1.4] text-text-soft"
				count={album.photoCount}
				label={$_('albums_photo_count', { values: { count: album.photoCount } })}
			/>
			<!-- The badge row: the attribution and the read-only mark ride
			     one line, so a shared card says who and what it means
			     without a fourth line. -->
			<span class="album-card__badges mt-0.5 flex flex-wrap items-center gap-1.5">
				{#if album.callerRole === 'viewer' && album.owner !== undefined}
					<span
						class="album-card__badge album-card__badge--owner inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
						title={ownerTitle(album)}
					>
						{$_('albums_shared_by', { values: { name: ownerLabel(album) } })}
					</span>
					<span
						class="album-card__badge album-card__badge--readonly inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
					>
						{$_('shared_readonly_badge')}
					</span>
				{:else if album.isShared}
					<!-- The owner's own "someone else sees this" mark; the
					     shared face above says the same thing in the
					     reader's direction instead. -->
					<span
						class="album-card__badge album-card__badge--shared inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
					>
						<svg viewBox="0 0 24 24" width="12" height="12" fill="none" aria-hidden="true">
							<circle cx="9" cy="8.5" r="3.25" stroke="currentColor" stroke-width="1.75" />
							<path
								d="M3.5 19c.9-2.8 3-4.2 5.5-4.2s4.6 1.4 5.5 4.2"
								stroke="currentColor"
								stroke-width="1.75"
								stroke-linecap="round"
							/>
							<path
								d="M16 5.6a3.25 3.25 0 0 1 0 5.8M17.5 14.9c2 .5 3.4 1.8 4 4.1"
								stroke="currentColor"
								stroke-width="1.75"
								stroke-linecap="round"
							/>
						</svg>
						{$_('albums_shared_badge')}
					</span>
				{/if}
				{#if album.inTimeline === false}
					<!-- The muted clock: the caller's own preference keeps this
					     album's photos out of their aggregated timeline. -->
					<span
						class="album-card__badge album-card__badge--excluded inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
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
			</span>
		</span>
	</a>
{/snippet}

<CreateAlbumDialog
	bind:open={createOpen}
	busy={createBusy}
	errorKey={createErrorKey}
	skipped={createSkipped}
	sourceWritable={createWritable}
	onsubmit={(title, rootPath, audience, switches) => void submitCreate(title, rootPath, audience, switches)}
	ondone={finishCreate}
/>
