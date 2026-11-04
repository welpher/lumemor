<script lang="ts">
	// The unnamed region of the people surface (web IA §3): the
	// caller's unlabeled face clusters as a keyset-paginated card
	// grid (crop + honest face count), largest count first per the
	// server's ordering. This region carries the two organizing paths
	// of the people domain:
	//
	// - THE NAMING MAIN PATH: clicking a card opens the naming
	//   dialog; the PUT label's response is the fresh label state, and
	//   the card leaves the region immediately (the optimistic half) --
	//   the SSE person.labeled self-event converges the rest.
	// - THE BATCH ORGANIZE PATH: the select mode turns the grid into
	//   a merge surface; the merge dialog picks the target (one of the
	//   selected -- named inline first when the caller wants the
	//   merged person called something -- or any existing person) and
	//   the POST merge's response names the dissolved sources, which
	//   vanish here. A source's name dies with it (the contract's
	//   honest cost); the target's label persists.
	//
	// SSE CONVERGENCE: every person.* event and every face.clustered
	// event converges through ONE replay refetch, collapsed
	// single-flight. The replay re-walks the keyset pages from the
	// first until the loaded window's depth is covered (or the region
	// ends), so a scrolled page neither truncates nor misses the
	// fresh cluster a recognition run announced. The refetched SERVER
	// answer decides membership -- labels are private per user (B4),
	// so another account's write moves nothing the server does not
	// move.
	import { _ } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import {
		labelPerson,
		listUnnamedPeople,
		mergePeople,
		type UnnamedClusterSummary
	} from '$lib/api/people';
	import { subscribeFaceEvents, subscribePersonEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import Button from '$lib/components/Button.svelte';
	import PersonNameDialog from '$lib/people/PersonNameDialog.svelte';
	import MergeDialog from '$lib/people/MergeDialog.svelte';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	let items = $state<UnnamedClusterSummary[]>([]);
	let status = $state<ListStatus>('idle');
	/** False once the last page arrived (no `nextCursor`). */
	let hasMore = $state(true);
	/** The error-code i18n key of the last failed page load ('' when none). */
	let errorKey = $state('');
	// The continuation cursor of the next page (null while on the
	// first page); never assembled here, only passed back.
	let nextCursor: string | null = null;

	// The selection is a reactive id set: card clicks toggle it (the
	// select mode or an active selection turns cards into toggles),
	// Esc and the bar's clear drop it.
	const selection = new SvelteSet<string>();
	let selectMode = $state(false);

	// The naming dialog's write state.
	let nameDialogOpen = $state(false);
	let nameTargetId = $state('');
	let nameBusy = $state(false);
	let nameErrorKey = $state('');

	// The merge dialog's write state.
	let mergeOpen = $state(false);
	let mergeBusy = $state(false);
	let mergeErrorKey = $state('');

	// IntersectionObserver exists in every browser jsdom is not; the
	// manual load-more button is the fallback surface (and the error
	// retry lives there too, because an observer that never
	// re-intersects cannot unstick an error).
	const observerAvailable = typeof IntersectionObserver === 'function';
	let sentinel: HTMLElement | undefined = $state();

	const selectable = $derived(selectMode || selection.size > 0);
	const selectedClusters = $derived(items.filter((cluster) => selection.has(cluster.clusterId)));
	const firstLoad = $derived(items.length === 0 && status === 'loading');
	const settledEmpty = $derived(items.length === 0 && status === 'ready' && !hasMore);

	/** Fetch the next keyset page (the first page while idle). */
	async function loadMore(): Promise<void> {
		if (status === 'loading') {
			// The in-flight guard: one page fetch per trigger.
			return;
		}
		if (status === 'ready' && !hasMore) {
			return;
		}
		status = 'loading';
		const outcome = await listUnnamedPeople(nextCursor);
		if (!outcome.ok) {
			status = 'error';
			errorKey = outcome.errorKey;
			return;
		}
		// Pages append only ids not already loaded: the keyset is
		// disjoint by construction, but a convergence replay racing
		// this response must not duplicate a card.
		const seen = new Set(items.map((cluster) => cluster.clusterId));
		const appended = outcome.page.items.filter((cluster) => !seen.has(cluster.clusterId));
		items = [...items, ...appended];
		nextCursor = outcome.page.nextCursor;
		hasMore = nextCursor !== null;
		status = 'ready';
		errorKey = '';
	}

	// The single-flight collapse of the convergence replay: while one
	// replay runs, later events park in a queue and one trailing
	// replay runs when it lands.
	let converging = false;
	let convergeQueued = false;

	/**
	 * The convergence entry point behind every SSE event: replay the
	 * keyset pages from the first until the loaded window's depth is
	 * covered (or the region ends) and adopt the collected prefix.
	 * A failure keeps the last good state silently -- the next event
	 * retries.
	 */
	async function runConverge(): Promise<void> {
		if (converging) {
			convergeQueued = true;
			return;
		}
		converging = true;
		try {
			const depth = items.length;
			const seen = new Set<string>();
			let collected: UnnamedClusterSummary[] = [];
			let cursor: string | null = null;
			while (true) {
				const outcome = await listUnnamedPeople(cursor);
				if (!outcome.ok) {
					return;
				}
				for (const cluster of outcome.page.items) {
					if (!seen.has(cluster.clusterId)) {
						seen.add(cluster.clusterId);
						collected.push(cluster);
					}
				}
				cursor = outcome.page.nextCursor;
				if (cursor === null || collected.length >= depth) {
					break;
				}
			}
			items = collected;
			nextCursor = cursor;
			hasMore = cursor !== null;
			if (status === 'error') {
				status = 'ready';
				errorKey = '';
			}
		} finally {
			converging = false;
			if (convergeQueued) {
				convergeQueued = false;
				void runConverge();
			}
		}
	}

	function openNaming(cluster: UnnamedClusterSummary): void {
		nameTargetId = cluster.clusterId;
		nameErrorKey = '';
		nameDialogOpen = true;
	}

	/**
	 * The naming write: the PUT response IS the fresh label state, so
	 * the card leaves the region immediately; the SSE person.labeled
	 * self-event converges the region's counts and ordering through
	 * the replay.
	 */
	async function submitName(name: string): Promise<void> {
		if (nameBusy) {
			return;
		}
		nameBusy = true;
		nameErrorKey = '';
		const outcome = await labelPerson(nameTargetId, name);
		nameBusy = false;
		if (!outcome.ok) {
			nameErrorKey = outcome.errorKey;
			return;
		}
		items = items.filter((cluster) => cluster.clusterId !== nameTargetId);
		selection.delete(nameTargetId);
		nameDialogOpen = false;
	}

	/**
	 * The batch merge: the inline naming (a selected target) PUTs
	 * first, then the POST merge folds the other selected clusters
	 * into the target. The response's mergedClusterIds name the
	 * dissolved sources, which vanish; a target named inline left the
	 * region with its label; an unnamed survivor keeps its card with
	 * the merged count (the representative crop converges with the
	 * SSE person.merged replay -- the strongest member may be a
	 * newcomer).
	 */
	async function runMerge(target: { clusterId: string; inlineName?: string }): Promise<void> {
		if (mergeBusy) {
			return;
		}
		const sources = [...selection].filter((id) => id !== target.clusterId);
		mergeBusy = true;
		mergeErrorKey = '';
		if (target.inlineName !== undefined) {
			const labeled = await labelPerson(target.clusterId, target.inlineName);
			if (!labeled.ok) {
				mergeBusy = false;
				mergeErrorKey = labeled.errorKey;
				return;
			}
		}
		const outcome = await mergePeople(sources, target.clusterId);
		mergeBusy = false;
		if (!outcome.ok) {
			mergeErrorKey = outcome.errorKey;
			return;
		}
		const dissolved = new Set(outcome.merge.mergedClusterIds);
		let updated = items.filter((cluster) => !dissolved.has(cluster.clusterId));
		if (target.inlineName !== undefined) {
			updated = updated.filter((cluster) => cluster.clusterId !== target.clusterId);
		} else {
			const gained = sources.reduce(
				(sum, id) => sum + (items.find((cluster) => cluster.clusterId === id)?.faceCount ?? 0),
				0
			);
			updated = updated.map((cluster) =>
				cluster.clusterId === target.clusterId
					? { ...cluster, faceCount: cluster.faceCount + gained }
					: cluster
			);
		}
		items = updated;
		selection.clear();
		mergeOpen = false;
	}

	function handleCardClick(cluster: UnnamedClusterSummary): void {
		if (selectable) {
			if (selection.has(cluster.clusterId)) {
				selection.delete(cluster.clusterId);
			} else {
				selection.add(cluster.clusterId);
			}
			return;
		}
		openNaming(cluster);
	}

	function exitSelection(): void {
		selection.clear();
		selectMode = false;
	}

	function handleWindowKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && (selection.size > 0 || selectMode)) {
			exitSelection();
		}
	}

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// The first page, the SSE subscriptions and the sentinel's
	// observer live for the page's lifetime: the first subscriber of
	// any kind opens the stream, leaving the page closes it. The
	// initial page load runs untracked: its guards read the reactive
	// status flags, and a tracked read would tie this effect to the
	// fetch's own state changes (the effect re-running on every page
	// landing -- the double fetch).
	$effect(() => {
		void untrack(() => loadMore());

		const unsubscribePerson = subscribePersonEvents(() => {
			void runConverge();
		});
		const unsubscribeFace = subscribeFaceEvents(() => {
			void runConverge();
		});

		let observer: IntersectionObserver | undefined;
		if (observerAvailable && sentinel !== undefined) {
			observer = new IntersectionObserver(
				(entries) => {
					if (entries.some((entry) => entry.isIntersecting)) {
						// The page's own guards (in-flight, exhausted)
						// collapse bursts into one fetch.
						void loadMore();
					}
				},
				{ rootMargin: '600px 0px' }
			);
			observer.observe(sentinel);
		}
		return () => {
			observer?.disconnect();
			unsubscribePerson();
			unsubscribeFace();
		};
	});
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<svelte:head>
	<title>{$_('people_unnamed_title')} · lumemor</title>
</svelte:head>

<section class="unnamed max-w-[1200px] mx-auto" aria-busy={status === 'loading'}>
	<!-- The toolbar: the way back to the named region, and the
	     selection-mode toggle once clusters exist. -->
	<div class="unnamed__toolbar flex justify-end items-center gap-3 mb-2">
		<a
			class="unnamed__people-link mr-auto px-2.5 py-1.5 font-body font-semibold text-[14px] leading-[1.4] text-text-soft no-underline rounded-control transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/people"
		>{$_('nav_people')}</a>
		{#if items.length > 0}
			<Button variant="ghost" aria-pressed={selectMode} onclick={() => (selectMode = !selectMode)}>
				{selectMode ? $_('action_done') : $_('action_select')}
			</Button>
		{/if}
	</div>

	{#if firstLoad}
		<p class="unnamed__loading my-12 text-center text-text-soft" role="status">{$_('people_unnamed_loading')}</p>
	{:else if settledEmpty}
		<div class="unnamed__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card">
			<h1 class="text-[22px] mb-2">{$_('people_unnamed_title')}</h1>
			<p class="text-text-soft">{$_('people_unnamed_empty')}</p>
		</div>
	{:else if status === 'error' && items.length === 0}
		<div class="unnamed__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void loadMore()}>{$_('action_retry')}</Button>
		</div>
	{:else}
		<div class="unnamed__grid grid grid-cols-[repeat(auto-fill,minmax(min(180px,100%),1fr))] gap-3">
			{#each items as cluster (cluster.clusterId)}
				<!-- The cluster card: a button, because the naming dialog is
				     the main path; the select mode turns the same button
				     into the selection toggle (the timeline tile's
				     interaction model). -->
				<button
					class="cluster-card relative flex flex-col overflow-hidden rounded-card bg-card border text-left cursor-pointer transition-[border-color] duration-(--motion-fast) ease-standard focus-visible:border-accent {selection.has(cluster.clusterId) ? 'border-accent' : 'border-border'}"
					class:cluster-card--selected={selection.has(cluster.clusterId)}
					type="button"
					aria-pressed={selectable ? selection.has(cluster.clusterId) : undefined}
					onclick={() => handleCardClick(cluster)}
				>
					<span class="cluster-card__crop relative block aspect-square bg-accent-soft" aria-hidden="true">
						<img
							class="absolute inset-0 w-full h-full object-cover"
							src={cluster.thumbnail}
							alt=""
							loading="lazy"
							decoding="async"
							draggable="false"
						/>
						{#if selectable}
							<!-- The selection mark: a visual state of the card
							     button itself (no nested interactive element),
							     the timeline checkbox's look. -->
							<span
								class="cluster-card__check absolute top-1.5 left-1.5 z-[3] inline-flex items-center justify-center w-[26px] h-[26px] p-0 border-2 rounded-full text-transparent bg-[rgb(20_26_46/0.55)] backdrop-blur-[2px] {selection.has(cluster.clusterId) ? 'border-accent bg-accent text-accent-on' : 'border-accent-on'}"
								aria-hidden="true"
							>
								<svg viewBox="0 0 24 24" width="14" height="14" fill="none">
									<path d="M5 12.5l4.5 4.5L19 7.5" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
								</svg>
							</span>
						{/if}
					</span>
					<span class="cluster-card__count block px-3 py-2 font-body font-normal text-[14px] leading-[1.4] text-text-soft">{$_('people_face_count', { values: { count: cluster.faceCount } })}</span>
				</button>
			{/each}
		</div>
	{/if}

	{#if status === 'error' && items.length > 0}
		<!-- A failed follow-up page while cards exist: the page stays,
		     the retry re-runs the same cursor (an idempotent position). -->
		<div class="unnamed__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void loadMore()}>{$_('action_retry')}</Button>
		</div>
	{/if}

	{#if !observerAvailable && hasMore && status === 'ready'}
		<div class="unnamed__pager flex justify-center my-5">
			<Button variant="ghost" onclick={() => void loadMore()}>{$_('timeline_load_more')}</Button>
		</div>
	{/if}

	<div class="unnamed__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
</section>

{#if selection.size > 0}
	<!-- The floating merge bar: the batch organize entry, the batch
	     bar's look. The merge needs at least two clusters, so a lone
	     selection keeps the button disabled. -->
	<div
		class="mergebar fixed left-1/2 bottom-5 z-[60] box-content flex flex-wrap items-center justify-center gap-2 max-w-[min(92vw,720px)] px-3.5 py-2.5 border border-border rounded-overlay bg-card shadow-overlay [transform:translateX(-50%)]"
		role="toolbar"
		aria-label={$_('batch_selected_count', { values: { count: selection.size } })}
	>
		<p class="mergebar__count mr-1.5 font-body font-bold text-[15px] leading-[1.4] text-text whitespace-nowrap">{$_('batch_selected_count', { values: { count: selection.size } })}</p>
		<Button
			variant="primary"
			disabled={selection.size < 2 || mergeBusy}
			onclick={() => {
				mergeErrorKey = '';
				mergeOpen = true;
			}}
		>
			{$_('people_merge_action')}
		</Button>
		<Button variant="ghost" onclick={exitSelection}>{$_('action_clear_selection')}</Button>
	</div>
{/if}

<PersonNameDialog
	bind:open={nameDialogOpen}
	titleKey="people_name_dialog_title"
	submitKey="people_name_submit"
	busy={nameBusy}
	errorKey={nameErrorKey}
	onsubmit={(name) => void submitName(name)}
/>

<MergeDialog
	bind:open={mergeOpen}
	selected={selectedClusters}
	busy={mergeBusy}
	errorKey={mergeErrorKey}
	onsubmit={(target) => void runMerge(target)}
/>
