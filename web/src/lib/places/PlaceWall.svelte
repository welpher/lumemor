<script lang="ts">
	// The place detail's photo wall: THIS PLACE'S photos over the
	// shared Wall -- month groups, the 72px axis gutter, the sticky
	// month bands under the page's context bar, and the selection
	// with its batch bar (favorite / archive / trash + clear).
	//
	// THE DATA WALK IS THIS COMPONENT'S: the timeline client over the
	// endpoint's `placeIds` narrow (the caller-visible live assets
	// resolved to this place), the SSE asset convergence, the scroll
	// sentinel and the wallItems publication the viewer's navigation
	// sequence reads. The page keys this component per place id, so a
	// navigation to another place always walks that place's photos --
	// no stale narrow.
	import { _ } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import { createTimelineClient, type BatchAction } from '$lib/api/timeline';
	import { subscribeAssetEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { publishWallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import Button from '$lib/components/Button.svelte';
	import StateCard from '$lib/components/StateCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	type Props = {
		/** The place whose photos this wall walks (the GeoNames id). */
		placeId: number;
	};

	let { placeId }: Props = $props();

	const wall = untrack(() =>
		createTimelineClient({ path: '/api/v1/assets/timeline', view: 'timeline', placeIds: [placeId] })
	);

	const firstLoad = $derived($wall.items.length === 0 && $wall.status === 'loading');
	const settledEmpty = $derived($wall.items.length === 0 && $wall.status === 'ready' && !$wall.hasMore);

	// IntersectionObserver exists in every browser jsdom is not; the
	// manual load-more button is the fallback surface (and the error
	// retry lives there too, because an observer that never
	// re-intersects cannot unstick an error).
	const observerAvailable = typeof IntersectionObserver === 'function';
	let sentinel: HTMLElement | undefined = $state();

	// The wall's batch runner: the client's own view-aware optimistic
	// convergence (the trashed/archived tiles leave immediately, the SSE
	// events confirm, the favourite flags flip optimistically and stay
	// -- favourites publish no events by contract).
	async function runBatch(
		action: BatchAction,
		ids: readonly string[]
	): Promise<{ ok: true } | { ok: false; errorKey: string }> {
		return wall.runBatch(action, [...ids]);
	}

	/** The shared unknown-code fallback (see $lib/i18n/errors). */
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

<section class="place-wall px-2 pe-[72px]" aria-busy={$wall.status === 'loading'}>
	{#if firstLoad}
		<LoadingLine label={$_('place_loading')} />
	{:else if settledEmpty}
		<StateCard title={$_('place_empty_title')} body={$_('place_empty_body')} />
	{:else if $wall.status === 'error' && $wall.items.length === 0}
		<StateCard
			tone="error"
			body={errorText($wall.errorKey)}
			actionLabel={$_('action_retry')}
			onaction={() => void wall.loadMore()}
		/>
	{:else}
		<Wall items={$wall.items} actions={['favorite', 'hide', 'trash']} onbatch={runBatch} />
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
		<div class="place-wall__pager flex justify-center my-5">
			<Button variant="ghost" onclick={() => void wall.loadMore()}>
				<ChevronsDown size={20} aria-hidden="true" />
				{$_('timeline_load_more')}
			</Button>
		</div>
	{/if}

	<div class="place-wall__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
</section>
