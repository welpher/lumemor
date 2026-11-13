<script lang="ts">
	// The named region of the people surface (the design's action
	// bar over a card grid): the one-row bar carries the title with
	// the person count, the name filter and the unnamed region's
	// entry (badged with the unnamed group count); the cards below
	// are square crops with the caller's own name and the honest
	// PHOTO count -- photos, not faces, so the number the card shows
	// is the number of photos clicking through will render.
	//
	// THE TWO STATES ONLY: the people domain is named
	// (this page) or unnamed (`/people/unnamed`). Hiding a person
	// retired whole -- there is no badge and no third region; a
	// person leaves this list by removing the caller's name
	// (the detail's remove-name action), which sends the cluster back
	// to the caller's unnamed region.
	//
	// SSE CONVERGENCE: every person.* event and every asset
	// deletion converges through ONE fresh load of both reads (the
	// named list and the unnamed region's count), collapsed
	// single-flight (a burst parks in a queue and costs one trailing
	// load). The refetched SERVER answer decides what changed --
	// labels are PRIVATE per user, so an event from another
	// account's write must move nothing the server does not move.
	import { _ } from 'svelte-i18n';
	import { listPeople, listUnnamedPeople, type PersonSummary } from '$lib/api/people';
	import { subscribeAssetEvents, subscribePersonEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import StateCard from '$lib/components/StateCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import Search from '@lucide/svelte/icons/search';
	import ScanFace from '@lucide/svelte/icons/scan-face';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	let people = $state<PersonSummary[]>([]);
	/** The caller's whole unnamed region (the entry's badge). */
	let unnamedCount = $state(0);
	let status = $state<ListStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

	// The name filter is the page's own narrowing: the server sorts
	// the caller's people by name and the list is small (a family),
	// so the filter runs locally and reacts on every keystroke --
	// no request per letter.
	let filter = $state('');

	// The single-flight guard: while one load runs, later triggers
	// park (the burst's tail) and one trailing load runs when it
	// lands -- the same collapse the timeline client applies.
	let running = false;
	let queued = false;

	/**
	 * Load the page's two reads and apply them. `converge` marks an
	 * SSE-driven run: the visible state stays and a failure keeps the
	 * last good answer instead of raising the error box -- later
	 * events retry.
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
			const [peopleOutcome, unnamedOutcome] = await Promise.all([listPeople(), listUnnamedPeople(null)]);
			if (peopleOutcome.ok && unnamedOutcome.ok) {
				people = peopleOutcome.people;
				unnamedCount = unnamedOutcome.page.totalCount;
				errorKey = '';
				status = 'ready';
			} else if (!converge) {
				status = 'error';
				errorKey = peopleOutcome.ok
					? unnamedOutcome.ok
						? ''
						: unnamedOutcome.errorKey
					: peopleOutcome.errorKey;
			}
		} finally {
			running = false;
			if (queued) {
				queued = false;
				void runLoad(true);
			}
		}
	}

	const shown = $derived(
		filter.trim() === ''
			? people
			: people.filter((person) => person.name.toLowerCase().includes(filter.trim().toLowerCase()))
	);
	const firstLoad = $derived(people.length === 0 && status === 'loading');
	const settledEmpty = $derived(people.length === 0 && status === 'ready');

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// The unnamed entry's accessible name: the region's title beside
	// its count (the bar's own read, composed).
	const unnamedLabel = $derived(
		`${$_('people_unnamed_title')} · ${$_('people_unnamed_count', { values: { count: unnamedCount } })}`
	);

	// The load and the SSE subscriptions live for the page's
	// lifetime: the first subscriber opens the stream, leaving the
	// page closes it. Asset deletions converge too: a purged asset's
	// faces died with it (clusters lost members or dissolved), so the
	// counts and the membership are stale until the refetch.
	$effect(() => {
		void runLoad(false);
		const unsubscribe = subscribePersonEvents(() => {
			void runLoad(true);
		});
		const unsubscribeAssets = subscribeAssetEvents((event) => {
			if (event.name !== 'asset.deleted') {
				return;
			}
			void runLoad(true);
		});
		return () => {
			unsubscribe();
			unsubscribeAssets();
		};
	});
</script>

<svelte:head>
	<title>{$_('nav_people')} · lumemor</title>
</svelte:head>

<section class="people max-w-[1200px] mx-auto px-4 pt-4 pb-12" aria-busy={status === 'loading'}>
	<!-- THE ONE ROW: the title with its count on the left, the page
	     filter and the unnamed entry on the right, one divider under
	     the row. The count renders only on a settled answer -- a
	     loading face counts nothing. -->
	<div class="people__toolbar flex items-center gap-3 pb-4 border-b border-border">
		<h1 class="people__title m-0 font-display font-bold text-[20px] leading-[1.3] text-text">
			{$_('nav_people')}
			{#if status === 'ready'}
				<span
					class="people__title-count font-body font-normal text-[15px] leading-[1.4] text-text-soft tabular-nums"
				>
					· {$_('people_bar_count', { values: { count: people.length } })}
				</span>
			{/if}
		</h1>
		<div class="people__actions ml-auto flex items-center gap-2">
			<label class="people__filter relative block w-full max-w-[220px]">
				<span class="absolute start-3 top-1/2 -translate-y-1/2 text-text-faint pointer-events-none">
					<Search size={16} aria-hidden="true" />
				</span>
				<input
					class="w-full ps-9 pe-3 py-2 font-body font-normal text-[14px] leading-[1.4] text-text bg-page border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
					type="search"
					bind:value={filter}
					placeholder={$_('people_filter_placeholder')}
					aria-label={$_('people_filter_placeholder')}
				/>
			</label>
			<!-- The unnamed region's entry: the naming main path's door,
			     badged with the region's own count. An icon affordance
			     (the icon-first rule), named by its aria-label. -->
			<a
				class="people__unnamed-link relative inline-flex items-center justify-center w-9 h-9 p-0 rounded-control border border-transparent bg-transparent text-text no-underline cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
				href="/people/unnamed"
				aria-label={unnamedLabel}
				title={unnamedLabel}
			>
				<ScanFace size={20} aria-hidden="true" />
				{#if unnamedCount > 0}
					<span
						class="people__unnamed-badge absolute -top-1 -right-1 min-w-[18px] h-[18px] px-1 inline-flex items-center justify-center rounded-full bg-accent text-accent-on font-body font-semibold text-[11px] leading-none tabular-nums pointer-events-none"
						aria-hidden="true">{unnamedCount}</span
					>
				{/if}
			</a>
		</div>
	</div>

	<div class="people__content pt-4">
		{#if firstLoad}
			<LoadingLine label={$_('people_loading')} />
		{:else if settledEmpty}
			<StateCard title={$_('people_empty_title')} body={$_('people_empty')} />
		{:else if status === 'error' && people.length === 0}
			<StateCard
				tone="error"
				body={errorText(errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void runLoad(false)}
			/>
		{:else if shown.length === 0}
			<!-- The filter matched nothing: the grid empties, the copy
			     says why (the server's list is untouched). -->
			<p
				class="people__no-match my-12 text-center font-body font-normal text-[14px] leading-[1.4] text-text-soft"
				role="status"
			>
				{$_('people_picker_empty')}
			</p>
		{:else}
			<div class="people__grid grid grid-cols-[repeat(auto-fill,minmax(min(160px,100%),1fr))] gap-3">
				{#each shown as person (person.clusterId)}
					<!-- The person card: the avatar crop (the person's picked
					     face, or the best the caller can see), the caller's own
					     name, and the photo count as icon + number. -->
					<a
						class="person-card flex flex-col overflow-hidden rounded-card bg-card border border-border no-underline text-text cursor-pointer transition-[border-color,box-shadow] duration-(--motion-fast) ease-standard hover:border-accent hover:shadow-card focus-visible:border-accent"
						href="/people/{encodeURIComponent(person.clusterId)}"
					>
						<span class="person-card__crop relative block aspect-square bg-accent-soft" aria-hidden="true">
							<img
								class="absolute inset-0 w-full h-full object-cover"
								src={person.thumbnail}
								alt=""
								loading="lazy"
								decoding="async"
								draggable="false"
							/>
						</span>
						<span class="person-card__caption flex items-center gap-2 px-3 py-2.5">
							<span
								class="person-card__name min-w-0 font-body font-semibold text-[15px] leading-[1.4] text-text truncate"
								>{person.name}</span
							>
							<CountLabel
								class="person-card__count ml-auto font-body font-normal text-[13px] leading-[1.4] text-text-soft"
								count={person.photoCount}
								label={$_('people_photo_count', { values: { count: person.photoCount } })}
							/>
						</span>
					</a>
				{/each}
			</div>
		{/if}

		{#if status === 'error' && people.length > 0}
			<!-- A failed convergence while cards exist: the page stays,
			     the next event retries; the box keeps the retry affordance. -->
			<StateCard
				tone="error"
				body={errorText(errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void runLoad(false)}
			/>
		{/if}
	</div>
</section>
