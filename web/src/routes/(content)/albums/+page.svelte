<script lang="ts">
	// The album wall: the cover-card grid in its two
	// sections -- the caller's own albums (the pinned sync card of
	// the default "My photos" album first, then the
	// directory albums in the server's own-group order) then the
	// albums shared to them (with the honest owner attribution) --
	// with the collage covers (the pinned cover full-bleed,
	// otherwise the newest members: two split, three the 2+1 fall,
	// four the quad), the shared / excluded badges, and the
	// "New album" creation dialog. THE MEMORY ROWS ARE NOT THIS
	// WALL'S: the reference sets moved to their own route (the
	// memories wall), so both sections filter the memory type out.
	// Card clicks open the album's
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
	import Button from '$lib/components/Button.svelte';
	import CreateAlbumDialog, { type CreateAlbumSwitches } from '$lib/albums/CreateAlbumDialog.svelte';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
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

	// The sections ride the items' own `callerRole` discriminator:
	// the server's concatenation is own-first, shared-after, and each
	// group's order is the server's. The memory rows leave both
	// groups -- the reference sets' home is the memories wall.
	const ownAlbums = $derived(albums.filter((album) => album.callerRole === 'owner' && album.type !== 'memory'));
	const sharedAlbums = $derived(albums.filter((album) => album.callerRole === 'viewer' && album.type !== 'memory'));

	const firstLoad = $derived(albums.length === 0 && status === 'loading');
	// The honest empty: no ALBUM row at all (a caller whose wall holds
	// only memories still sees the empty state here -- their
	// reference sets live on the memories wall).
	const settledEmpty = $derived(ownAlbums.length === 0 && sharedAlbums.length === 0 && status === 'ready');

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
		// The honest answers hold the dialog: any skipped id, or the
		// read-only source fact, keeps the report face until the
		// Done affordance acknowledges it.
		if (outcome.skipped.length === 0 && outcome.sourceWritable) {
			createOpen = false;
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

<section class="albums max-w-[1200px] mx-auto p-6" aria-busy={status === 'loading'}>
	<!-- The one toolbar of the wall: the directory-album creation, the
	     primary action -- THE ADMIN'S ALONE (mounting an
	     existing photo directory is configuration, a member never
	     sees the affordance). -->
	{#if adminSession}
		<div class="albums__toolbar flex justify-end items-center mb-3">
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

	{#if firstLoad}
		<p class="albums__loading my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('albums_loading')}
		</p>
	{:else if settledEmpty}
		<div
			class="albums__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card"
		>
			<h1 class="text-[22px] mb-2">{$_('albums_title')}</h1>
			<p class="text-text-soft">{$_('albums_empty')}</p>
		</div>
	{:else if status === 'error' && ownAlbums.length === 0 && sharedAlbums.length === 0}
		<div
			class="albums__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void runLoad(false)}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{:else}
		<section class="albums__own-section">
			<h1 class="albums__section-title mt-2 mb-3 font-display font-bold text-[20px] leading-[1.3] text-text">
				{$_('albums_own_section')}
			</h1>
			<div class="albums__grid grid grid-cols-[repeat(auto-fill,minmax(min(220px,100%),1fr))] gap-3">
				{#each ownAlbums as album (album.id)}
					{@render card(album)}
				{/each}
			</div>
		</section>
		{#if sharedAlbums.length > 0}
			<section class="albums__shared-section">
				<h2 class="albums__section-title mt-6 mb-3 font-display font-bold text-[20px] leading-[1.3] text-text">
					{$_('albums_shared_section')}
				</h2>
				<div class="albums__grid grid grid-cols-[repeat(auto-fill,minmax(min(220px,100%),1fr))] gap-3">
					{#each sharedAlbums as album (album.id)}
						{@render card(album)}
					{/each}
				</div>
			</section>
		{/if}
	{/if}

	{#if status === 'error' && (ownAlbums.length > 0 || sharedAlbums.length > 0)}
		<!-- A failed convergence while cards exist: the page stays, the
		     next event retries; the box keeps the retry affordance. -->
		<div
			class="albums__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void runLoad(false)}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{/if}
</section>

{#snippet card(album: AlbumSummary)}
	<!-- One album card: the collage cover, the title with its badges,
	     the count, and the owner attribution on the shared cards. -->
	<a
		class="album-card flex flex-col overflow-hidden rounded-card bg-card border border-border no-underline text-text cursor-pointer transition-[border-color,box-shadow] duration-(--motion-fast) ease-standard hover:border-accent hover:shadow-card focus-visible:border-accent"
		href="/albums/{encodeURIComponent(album.id)}"
	>
		<span
			class="album-card__cover {coverClass(
				album.coverAssetIds.length
			)} relative block aspect-[4/3] bg-accent-soft"
			aria-hidden="true"
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
				     columns, the quad fills the 2x2 grid. -->
				<span class="album-card__cells grid grid-cols-2 absolute inset-0">
					{#each album.coverAssetIds.slice(0, 4) as coverId, index (coverId)}
						<span
							class="album-card__cell {album.coverAssetIds.length === 3 && index === 2
								? 'album-card__cell--wide col-span-2'
								: ''} overflow-hidden"
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
			{#if album.owner !== undefined}
				<span class="album-card__owner font-body font-normal text-[13px] leading-[1.4] text-text-soft truncate"
					>{$_('albums_shared_by', { values: { email: album.owner.email } })}</span
				>
			{/if}
			<span class="album-card__meta flex items-center gap-1.5 mt-0.5">
				<span
					class="album-card__count font-body font-normal text-[13px] leading-[1.4] text-text-soft whitespace-nowrap tabular-nums"
					>{$_('albums_photo_count', { values: { count: album.itemCount } })}</span
				>
				{#if album.isShared}
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
