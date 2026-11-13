<script lang="ts">
	// The memory wall: the reference-set cards in ONE list (the design's
	// single list, carried to the memories page: no own/shared sections,
	// no group labels, most recently updated first) -- the manual origin
	// badge on an own card, the honest owner attribution beside the
	// read-only badge on a shared one (the attribution names the
	// person, the email only falls back) -- with the same collage cover
	// language the album cards carry. A memory card NEVER
	// carries the timeline-exclusion badge: a memory never feeds its
	// creator's timeline by construction (the exclusion is the
	// memory's fixed semantics, not a signal worth a badge).
	// Card clicks open the memory's own detail (the reference-set
	// face -- the curation bar and the removal selection).
	//
	// THE CREATION ENTRY IS EVERY MEMBER'S: a memory is personal
	// curation, the OPPOSITE gate of the album wall's
	// administrator-only "New album" (pointing at the file system is
	// configuration; naming a reference set of one's own sighted
	// photos is not). The "New memory" button renders for every
	// session and the server holds no role gate on the write.
	//
	// SSE CONVERGENCE: every album.* event converges through ONE
	// fresh wall fetch, collapsed single-flight (the albums page's
	// pattern): the memories ride the album event family, and the
	// refetched SERVER answer decides what changed. A failed
	// convergence keeps the last good state silently; only the
	// initial load's failure raises the error box.
	import { _ } from 'svelte-i18n';
	import { listAlbums, createMemory, type AlbumSummary, type AlbumAudienceSkip } from '$lib/api/albums';
	import { subscribeAlbumEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { ambiguousLabels, displayNameOf } from '$lib/identity';
	import Button from '$lib/components/Button.svelte';
	import StateCard from '$lib/components/StateCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import CreateMemoryDialog from '$lib/memories/CreateMemoryDialog.svelte';
	import Plus from '@lucide/svelte/icons/plus';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	// The memories are the wall's memory-typed rows: the server's
	// own group answers the caller's memories, the shared group the
	// memories others granted them.
	let memories = $state<AlbumSummary[]>([]);
	let status = $state<ListStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

	// The creation dialog (title + audience); the created card lands
	// through the refetch the write triggers, and the write's honest
	// `skipped` report (the requested audience ids that did not
	// land) holds the dialog open until acknowledged.
	let createOpen = $state(false);
	let createBusy = $state(false);
	let createErrorKey = $state('');
	let createSkipped = $state<readonly AlbumAudienceSkip[] | null>(null);

	// The single-flight guard: while one fetch runs, later triggers
	// park (the burst's tail) and one trailing fetch runs when it
	// lands -- the albums page's collapse.
	let running = false;
	let queued = false;

	/**
	 * Fetch the wall and keep the memory rows. `converge` marks an
	 * SSE-driven run: the visible state stays and a failure keeps the
	 * last good answer -- later events retry.
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
				memories = outcome.albums.filter((album) => album.type === 'memory');
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

	// The wall's ONE list, most recently updated first (the design:
	// own and shared mixed, no group labels; the sort is stable, so the
	// server's order breaks equal timestamps). A memory has no pinned
	// sync card to lead it.
	const memoryCards = $derived(
		[...memories].sort((left, right) => Date.parse(right.updatedAt) - Date.parse(left.updatedAt))
	);

	// The attribution chips' disambiguator: a display name TWO accounts
	// carry gets the email as its hover title (the name is not a key).
	const ambiguousOwners = $derived(
		ambiguousLabels(memories.flatMap((memory) => (memory.owner === undefined ? [] : [memory.owner])))
	);

	const firstLoad = $derived(memories.length === 0 && status === 'loading');
	const settledEmpty = $derived(memories.length === 0 && status === 'ready');

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// The shared card's attribution: the owner's display name, the email
	// when the account set none (a blank chip never renders).
	function ownerLabel(memory: AlbumSummary): string {
		return memory.owner === undefined ? '' : displayNameOf(memory.owner);
	}

	// The attribution chip's hover title: the owner's email when another
	// account of this wall carries the same name.
	function ownerTitle(memory: AlbumSummary): string | undefined {
		return memory.owner !== undefined && ambiguousOwners.has(ownerLabel(memory)) ? memory.owner.email : undefined;
	}

	// The cover layout class of one card: the pinned cover and the
	// single newest member render full-bleed, two split, three the
	// 2+1 fall, four the quad; a bare memory renders the placeholder.
	function coverClass(count: number): string {
		switch (count) {
			case 0:
				return 'memory-card__cover--empty';
			case 1:
				return 'memory-card__cover--single';
			case 2:
				return 'memory-card__cover--split';
			case 3:
				return 'memory-card__cover--duo-plus-one';
			default:
				return 'memory-card__cover--quad';
		}
	}

	// The creation write: the refetch the write's SSE echo triggers
	// places the new card in the server's order (the single-flight
	// collapses the pair). The write's honest `skipped` report (the
	// requested audience ids that did not land) holds the dialog
	// open until acknowledged; an all-landed creation closes it
	// directly.
	async function submitCreate(title: string, audience: readonly { userId: string; email: string }[]): Promise<void> {
		if (createBusy) {
			return;
		}
		createBusy = true;
		createErrorKey = '';
		const outcome = await createMemory({
			title,
			audience: audience.map((member) => member.userId)
		});
		createBusy = false;
		if (!outcome.ok) {
			createErrorKey = outcome.errorKey;
			return;
		}
		createSkipped = outcome.skipped;
		void runLoad(true);
		if (outcome.skipped.length === 0) {
			createOpen = false;
		}
	}

	// The report's done: close the dialog and clear the report (the
	// wall behind already converged through the write's refetch).
	function finishCreate(): void {
		createOpen = false;
		createSkipped = null;
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
	<title>{$_('memories_title')} · lumemor</title>
</svelte:head>

<section class="memories max-w-[1200px] mx-auto px-4 pt-4 pb-12" aria-busy={status === 'loading'}>
	<!-- THE ONE ROW: the page title with its count on the left, the
	     creation on the right, one divider under the row (the count
	     never takes a row of its own, and the loading face counts
	     nothing). The creation is EVERY MEMBER'S (a memory is
	     personal curation, not configuration; the server holds no
	     role gate on the write). -->
	<div class="memories__toolbar flex items-center gap-3 pb-4 border-b border-border">
		<h1 class="memories__title m-0 font-display font-bold text-[20px] leading-[1.3] text-text">
			{$_('memories_title')}
			{#if status === 'ready'}
				<span
					class="memories__title-count font-body font-normal text-[15px] leading-[1.4] text-text-soft tabular-nums"
				>
					· {$_('memories_bar_count', { values: { count: memories.length } })}
				</span>
			{/if}
		</h1>
		<div class="memories__actions ml-auto flex items-center gap-2">
			<Button
				onclick={() => {
					createErrorKey = '';
					createSkipped = null;
					createOpen = true;
				}}><Plus size={20} aria-hidden="true" />{$_('memories_new')}</Button
			>
		</div>
	</div>

	<div class="memories__content pt-4">
		{#if firstLoad}
			<LoadingLine label={$_('memories_loading')} />
		{:else if settledEmpty}
			<StateCard
				title={$_('memories_empty_title')}
				body={$_('memories_empty_body')}
				actionLabel={$_('memories_new')}
				onaction={() => {
					createErrorKey = '';
					createSkipped = null;
					createOpen = true;
				}}
			/>
		{:else if status === 'error' && memories.length === 0}
			<StateCard
				tone="error"
				body={errorText(errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void runLoad(false)}
			/>
		{:else}
			<!-- ONE grid: own and shared memories mixed by recency, no
			     group labels. -->
			<div class="memories__grid grid grid-cols-[repeat(auto-fill,minmax(min(220px,100%),1fr))] gap-3">
				{#each memoryCards as memory (memory.id)}
					{@render card(memory)}
				{/each}
			</div>
		{/if}

		{#if status === 'error' && memories.length > 0}
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

{#snippet card(memory: AlbumSummary)}
	<!-- One memory card: the collage cover, the title, the count, then the
	     badge row -- the manual origin badge on an own card, the owner
	     attribution beside the read-only badge on a shared one (the
	     attribution rules). No exclusion badge anywhere: a memory never feeds
	     its creator's timeline. -->
	<a
		class="memory-card flex flex-col overflow-hidden rounded-card bg-card border border-border no-underline text-text cursor-pointer transition-[border-color,box-shadow] duration-(--motion-fast) ease-standard hover:border-accent hover:shadow-card focus-visible:border-accent"
		href="/memories/{encodeURIComponent(memory.id)}"
	>
		<span
			class="memory-card__cover {coverClass(
				memory.coverAssetIds.length
			)} relative block aspect-[4/3] bg-accent-soft"
			aria-hidden="true"
		>
			{#if memory.coverAssetIds.length === 1}
				<img
					class="absolute inset-0 w-full h-full object-cover"
					src={`/api/v1/assets/${encodeURIComponent(memory.coverAssetIds[0] ?? '')}/thumbnail?size=tile`}
					alt=""
					loading="lazy"
					decoding="async"
					draggable="false"
				/>
			{:else if memory.coverAssetIds.length > 1}
				<!-- The collage cells: the 2+1 fall's third cell spans both
				     columns, the quad fills the 2x2 grid. -->
				<span class="memory-card__cells grid grid-cols-2 absolute inset-0">
					{#each memory.coverAssetIds.slice(0, 4) as coverId, index (coverId)}
						<span
							class="memory-card__cell {memory.coverAssetIds.length === 3 && index === 2
								? 'memory-card__cell--wide col-span-2'
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
				<!-- A bare memory: the honest placeholder, no requests. -->
				<span class="memory-card__placeholder absolute inset-0 grid place-items-center text-text-faint">
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
		<span class="memory-card__caption flex flex-col gap-0.5 px-3 py-2.5">
			<span class="memory-card__title font-body font-semibold text-[15px] leading-[1.4] text-text truncate"
				>{memory.title}</span
			>
			<CountLabel
				class="memory-card__count font-body font-normal text-[13px] leading-[1.4] text-text-soft"
				count={memory.itemCount}
				label={$_('memories_photo_count', { values: { count: memory.itemCount } })}
			/>
			<span class="memory-card__badges mt-0.5 flex flex-wrap items-center gap-1.5">
				{#if memory.callerRole === 'owner'}
					<!-- The manual origin badge: the reference set the owner
					     curated photo by photo. -->
					<span
						class="memory-card__badge memory-card__badge--manual inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
					>
						{$_('memories_manual_badge')}
					</span>
				{:else}
					<!-- The shared face: who shared it, then what it means for
					     the member -- the attribution and the read-only mark
					     ride one line. -->
					{#if memory.owner !== undefined}
						<span
							class="memory-card__badge memory-card__badge--owner inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
							title={ownerTitle(memory)}
						>
							{$_('albums_shared_by', { values: { name: ownerLabel(memory) } })}
						</span>
					{/if}
					<span
						class="memory-card__badge memory-card__badge--readonly inline-flex items-center gap-1 px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
					>
						{$_('shared_readonly_badge')}
					</span>
				{/if}
			</span>
		</span>
	</a>
{/snippet}

<CreateMemoryDialog
	bind:open={createOpen}
	busy={createBusy}
	errorKey={createErrorKey}
	skipped={createSkipped}
	onsubmit={(title, audience) => void submitCreate(title, audience)}
	ondone={finishCreate}
/>
