<script lang="ts">
	// The cover picker dialog of the album detail: one Modal over the
	// album's own members -- the wall's rendered grid is the pick
	// list, so the picker shows exactly the photos the detail walks.
	// The page owns the write (the PATCH through its callback) and
	// the members list (the wall's published items); the dialog owns
	// the choice's lifecycle (reseeded on open, the current pin
	// badged).
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import X from '@lucide/svelte/icons/x';
	import Check from '@lucide/svelte/icons/check';
	import { resolveErrorText } from '$lib/i18n/errors';
	import type { TimelineItem } from '$lib/api/timeline';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The album's rendered members (the wall's published items). */
		items: readonly TimelineItem[];
		/** The currently pinned cover asset id (null when unset). */
		currentCoverId?: string | null;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Notified of every close path (Escape, backdrop, the close mark). */
		onclose?: () => void;
		/** Run the write with the picked asset id. */
		onsubmit: (assetId: string) => void;
	};

	let {
		open = $bindable(false),
		items,
		currentCoverId = null,
		busy = false,
		errorKey = '',
		onclose,
		onsubmit
	}: Props = $props();

	let pickedId = $state('');

	// Reseed the choice on every open: the current pin starts picked
	// (submitting it again is the honest no-op re-pin), an unset pin
	// starts with the newest member.
	$effect(() => {
		if (open) {
			pickedId =
				(currentCoverId !== null && items.some((entry) => entry.id === currentCoverId)
					? currentCoverId
					: (items[0]?.id ?? '')) ?? '';
		}
	});

	/** Locale-aware capture date of one member. */
	function dateLabel(takenAt: string): string {
		return new Intl.DateTimeFormat($locale ?? 'en', { dateStyle: 'medium' }).format(
			new Date(takenAt)
		);
	}
</script>

<Modal bind:open title={$_('albums_cover_title')} {onclose}>
	<div class="cover-picker flex flex-col gap-3">
		<p class="cover-picker__body m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('albums_cover_body')}</p>
		{#if items.length === 0}
			<p class="cover-picker__empty m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('albums_cover_empty')}</p>
		{:else}
			<ul class="cover-picker__grid grid grid-cols-[repeat(auto-fill,minmax(min(140px,100%),1fr))] gap-2 m-0 p-0 list-none">
				{#each items as entry (entry.id)}
					<li>
						<button
							class="cover-option group flex flex-col overflow-hidden rounded-control bg-card border border-border cursor-pointer transition-[border-color] duration-(--motion-fast) ease-standard aria-pressed:border-accent hover:border-accent"
							type="button"
							aria-pressed={pickedId === entry.id}
							onclick={() => (pickedId = entry.id)}
						>
							<span class="cover-option__crop relative block aspect-square bg-accent-soft">
								<img
									class="absolute inset-0 w-full h-full object-cover"
									src={`/api/v1/assets/${encodeURIComponent(entry.id)}/thumbnail?size=tile`}
									alt=""
									loading="lazy"
									decoding="async"
									draggable="false"
								/>
								{#if currentCoverId === entry.id}
									<span class="cover-option__pin absolute top-1.5 left-1.5 px-1.5 py-0.5 font-body font-semibold text-[11px] leading-[1.4] text-text bg-card border border-border rounded-control">{$_('albums_cover_current_badge')}</span>
								{/if}
							</span>
							<span class="cover-option__meta flex flex-col min-w-0 px-2 py-1.5 text-left">
								<span class="cover-option__file font-body font-semibold text-[13px] leading-[1.4] text-text truncate">{entry.originalFilename}</span>
								<span class="cover-option__date font-body font-normal text-[12px] leading-[1.4] text-text-soft">{dateLabel(entry.takenAt)}</span>
							</span>
						</button>
					</li>
				{/each}
			</ul>
		{/if}
		{#if errorKey !== ''}
			<p class="cover-picker__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, errorKey)}</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => { open = false; onclose?.(); }}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button disabled={busy || items.length === 0 || pickedId === ''} onclick={() => onsubmit(pickedId)}>
			<Check size={20} aria-hidden="true" />
			{$_('albums_cover_submit')}
		</Button>
	{/snippet}
</Modal>
