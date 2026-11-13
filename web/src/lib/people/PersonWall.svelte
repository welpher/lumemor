<script lang="ts">
	// The person detail's photo wall: THIS PERSON'S
	// photos over the shared Wall -- month groups, the 72px axis
	// gutter, the sticky month bands under the page's context bar,
	// and the selection with its batch bar (favorite / hide /
	// trash + clear; hiding here is the PHOTO hide, not the retired
	// person hide).
	//
	// THE DATA WALK IS THIS COMPONENT'S: the timeline client over the
	// endpoint's `clusterId` narrow (one face cluster's own photos,
	// deduped by asset in the server's own query), the SSE asset
	// convergence, the scroll sentinel and the wallItems publication
	// the viewer's navigation sequence reads. The page remounts this
	// component per cluster id ({#key}), so a navigation to another
	// person always walks that person's photos -- no stale narrow.
	//
	// THE BATCH RUNNER is wired unconditionally: a person's photos
	// are curatable by anyone who can see them (the same actions the
	// timeline offers over a selection), and the wall's own hover
	// check is the selection entry -- the context bar carries no
	// selection switch (the person bar holds the person actions
	// alone).
	import { _ } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import { createTimelineClient, type BatchAction } from '$lib/api/timeline';
	import { subscribeAssetEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { publishWallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import Button from '$lib/components/Button.svelte';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	type Props = {
		/** The person whose photos this wall walks. */
		clusterId: string;
	};

	let { clusterId }: Props = $props();

	const wall = untrack(() => createTimelineClient({ path: '/api/v1/assets/timeline', view: 'timeline', clusterId }));

	const firstLoad = $derived($wall.items.length === 0 && $wall.status === 'loading');
	const settledEmpty = $derived($wall.items.length === 0 && $wall.status === 'ready' && !$wall.hasMore);

	// IntersectionObserver exists in every browser jsdom is not; the
	// manual load-more button is the fallback surface (and the error
	// retry lives there too, because an observer that never
	// re-intersects cannot unstick an error).
	const observerAvailable = typeof IntersectionObserver === 'function';
	let sentinel: HTMLElement | undefined = $state();

	// The wall's batch runner: the client's own view-aware optimistic
	// convergence (the trashed/hidden tiles leave immediately, the SSE
	// events confirm, the favourite flags flip optimistically and stay
	// -- favourites publish no events by contract). The page's detail
	// converges the count through its own asset-event subscription.
	async function runBatch(
		action: BatchAction,
		ids: readonly string[]
	): Promise<{ ok: true } | { ok: false; errorKey: string }> {
		return wall.runBatch(action, [...ids]);
	}

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon).
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// Publish the loaded items as the viewer's navigation sequence:
	// the copy outlives this component, so /a/{id} walks exactly the
	// photos this wall rendered.
	$effect(() => {
		publishWallItems($wall.items);
	});

	// The walk and the live surfaces: the first page on mount, the
	// asset convergence and the sentinel observer for the lifetime.
	$effect(() => {
		void untrack(() => wall.loadMore());

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

<section class="person-wall px-2 pe-[72px]" aria-busy={$wall.status === 'loading'}>
	{#if firstLoad}
		<p class="person-wall__loading my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('people_loading')}
		</p>
	{:else if settledEmpty}
		<div
			class="person-wall__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card"
		>
			<p class="person-wall__empty-title mb-2 font-display font-bold text-[16px] leading-[1.3] text-text">
				{$_('person_wall_empty_title')}
			</p>
			<p class="m-0 text-text-soft">{$_('person_wall_empty')}</p>
		</div>
	{:else if $wall.status === 'error' && $wall.items.length === 0}
		<div
			class="person-wall__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText($wall.errorKey)}</p>
			<Button variant="primary" onclick={() => void wall.loadMore()}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{:else}
		<Wall items={$wall.items} actions={['favorite', 'hide', 'trash']} onbatch={runBatch} />
	{/if}

	{#if $wall.status === 'error' && $wall.items.length > 0}
		<div
			class="person-wall__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText($wall.errorKey)}</p>
			<Button variant="primary" onclick={() => void wall.loadMore()}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{/if}

	{#if !observerAvailable && $wall.hasMore && $wall.status === 'ready'}
		<div class="person-wall__pager flex justify-center my-5">
			<Button variant="ghost" onclick={() => void wall.loadMore()}>
				<ChevronsDown size={20} aria-hidden="true" />
				{$_('timeline_load_more')}
			</Button>
		</div>
	{/if}

	<div class="person-wall__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
</section>
