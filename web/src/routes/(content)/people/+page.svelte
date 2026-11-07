<script lang="ts">
	// The named region of the people surface (web IA §3): the caller's
	// people as a card grid -- the representative face's crop, the
	// caller's own name, the honest per-caller face count, and the
	// hidden badge (hidden people STAY listed: the switch is a badge,
	// not a filter -- Task 1's pinned semantics). The server sorts by
	// name; the page renders that order. Every card links into the
	// cluster's detail.
	//
	// SSE CONVERGENCE: every person.* event converges through ONE
	// fresh list fetch, collapsed single-flight (a burst parks in a
	// queue and costs one trailing fetch). The refetched SERVER
	// answer decides what changed -- labels and the hidden switch are
	// PRIVATE per user (B4), so an event from another account's write
	// must move nothing the server does not move: a labeled event for
	// a cluster the caller's own list does not carry refetches the
	// same answer and the page invents no card. A failed convergence
	// keeps the last good state silently (the next event retries);
	// only the initial load's failure raises the error box.
	import { _ } from 'svelte-i18n';
	import { listPeople, type PersonSummary } from '$lib/api/people';
	import { subscribePersonEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import Button from '$lib/components/Button.svelte';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	let people = $state<PersonSummary[]>([]);
	let status = $state<ListStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

	// The single-flight guard: while one fetch runs, later triggers
	// park (the burst's tail) and one trailing fetch runs when it
	// lands -- the same collapse the timeline client applies.
	let running = false;
	let queued = false;

	/**
	 * Fetch the list and apply it. `converge` marks an SSE-driven
	 * run: the visible state stays (the items swap in place on
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
			const outcome = await listPeople();
			if (outcome.ok) {
				people = outcome.people;
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

	const firstLoad = $derived(people.length === 0 && status === 'loading');
	const settledEmpty = $derived(people.length === 0 && status === 'ready');

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// The load and the SSE subscription live for the page's lifetime:
	// the first subscriber opens the stream, leaving the page closes
	// it, and every person event converges through a refetch.
	$effect(() => {
		void runLoad(false);
		const unsubscribe = subscribePersonEvents(() => {
			void runLoad(true);
		});
		return () => unsubscribe();
	});
</script>

<svelte:head>
	<title>{$_('nav_people')} · lumemor</title>
</svelte:head>

<section class="people max-w-[1200px] mx-auto p-6" aria-busy={status === 'loading'}>
	<!-- The one toolbar of the region: the unnamed region's entry
	     point (the naming main path lives there), the timeline's
	     hidden-link pattern. -->
	<div class="people__toolbar flex justify-end items-center mb-2">
		<a
			class="people__unnamed-link mr-auto px-2.5 py-1.5 font-body font-semibold text-[14px] leading-[1.4] text-text-soft no-underline rounded-control transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/people/unnamed"
		>{$_('people_unnamed_link')}</a>
	</div>

	{#if firstLoad}
		<p class="people__loading my-12 text-center text-text-soft animate-loading-in" role="status">{$_('people_loading')}</p>
	{:else if settledEmpty}
		<div class="people__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card">
			<h1 class="text-[22px] mb-2">{$_('nav_people')}</h1>
			<p class="text-text-soft">{$_('people_empty')}</p>
		</div>
	{:else if status === 'error' && people.length === 0}
		<div class="people__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void runLoad(false)}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
		</div>
	{:else}
		<div class="people__grid grid grid-cols-[repeat(auto-fill,minmax(min(180px,100%),1fr))] gap-3">
			{#each people as person (person.clusterId)}
				<!-- The person card: the crop is the representative face's
				     (the cluster's strongest member); the hidden badge rides
				     the caption so the switch reads as text, never as a
				     filtered absence. -->
				<a
					class="person-card flex flex-col overflow-hidden rounded-card bg-card border border-border no-underline text-text cursor-pointer transition-[border-color,box-shadow] duration-(--motion-fast) ease-standard hover:border-accent hover:shadow-card focus-visible:border-accent"
					class:person-card--hidden={person.isHidden}
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
						<span class="person-card__name font-body font-semibold text-[15px] leading-[1.4] text-text truncate">{person.name}</span>
						{#if person.isHidden}
							<span class="person-card__badge inline-flex items-center px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap">{$_('people_hidden_badge')}</span>
						{/if}
						<span class="person-card__count ml-auto font-body font-normal text-[13px] leading-[1.4] text-text-soft whitespace-nowrap tabular-nums">{$_('people_face_count', { values: { count: person.faceCount } })}</span>
					</span>
				</a>
			{/each}
		</div>
	{/if}

	{#if status === 'error' && people.length > 0}
		<!-- A failed convergence while cards exist: the page stays, the
		     next event retries; the box keeps the retry affordance. -->
		<div class="people__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void runLoad(false)}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
		</div>
	{/if}
</section>
