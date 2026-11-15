<script lang="ts">
	// The cover picker: a cover is a SET of frames, four on the album
	// card's 2x2 collage and six on the memory band's 3-across strip,
	// and the order the owner clicks is the order the frames render.
	// The dialog is the collection's grid (the members the page walks)
	// and owns the choice's lifecycle: it seeds the current pin, counts
	// the picks, refuses the over-full click with a toast, and keeps
	// Save off while the set is half-picked -- a stored cover is either
	// empty (the automatic weekly rotation) or full.
	//
	// THE DEAD CELL: a pinned frame that has left the album has no row
	// in the grid, so it renders as its own marked tile ("no longer in
	// this album") and starts picked -- the honest state of the stored
	// pin. Deselecting it is the owner's fix; saving with it selected
	// answers the server's refusal, whose count the dialog states.
	// The mark only renders when the page's pool is the whole
	// collection (the loaded grid may be one page of many; a frame
	// from a later page must never be called missing).
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import { toast } from '$lib/components/toast.svelte';
	import X from '@lucide/svelte/icons/x';
	import Check from '@lucide/svelte/icons/check';
	import ImageOff from '@lucide/svelte/icons/image-off';
	import { resolveErrorText } from '$lib/i18n/errors';
	import type { TimelineItem } from '$lib/api/timeline';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The collection's members (the page's walk, or the memory's own). */
		items: readonly TimelineItem[];
		/** The album's live member count: the pool is complete only when the grid holds them all. */
		itemCount?: number;
		/** The card's frame count: four (album card) or six (memory band). */
		maxItems?: number;
		/** The stored pin, in pick order (empty when the album is automatic). */
		currentCoverIds?: readonly string[];
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** The refused frames the last failed write named (0 when none). */
		invalidCount?: number;
		/** Notified of every close path (Escape, backdrop, the close mark). */
		onclose?: () => void;
		/** Run the write with the picked frames, in pick order (empty restores the rotation). */
		onsubmit: (assetIds: readonly string[]) => void;
	};

	let {
		open = $bindable(false),
		items,
		itemCount = 0,
		maxItems = 4,
		currentCoverIds = [],
		busy = false,
		errorKey = '',
		invalidCount = 0,
		onclose,
		onsubmit
	}: Props = $props();

	/** The pick, in click order (the render order). */
	let picked = $state<string[]>([]);
	/** Whether the album opened with a stored pin (the empty collection's clear path). */
	let hadPin = $state(false);

	const pool = $derived(new Set(items.map((entry) => entry.id)));
	/** Whether the grid holds the whole collection (a later page must not read as a dead pin). */
	const poolComplete = $derived(items.length >= itemCount);
	/** The stored frames the collection no longer holds. */
	const goneIds = $derived(poolComplete ? currentCoverIds.filter((id) => !pool.has(id) && picked.includes(id)) : []);
	const remaining = $derived(Math.max(0, maxItems - picked.length));

	// Reseed the choice on the OPEN EDGE alone: the stored pin in its
	// own order, an unset pin empty (the automatic state). Seeding on
	// every prop change would wipe a half-made pick the moment a
	// live event refreshes the parent's detail -- the dialog exists
	// to hold the in-progress pick, so the seed reads the prop
	// through untrack and only the false-to-true transition plants
	// it.
	let wasOpen = false;
	$effect(() => {
		const isOpen = open;
		if (isOpen && !wasOpen) {
			untrack(() => {
				picked = [...currentCoverIds];
				hadPin = currentCoverIds.length > 0;
			});
		}
		wasOpen = isOpen;
	});

	/** Toggle one frame in or out of the pick; the over-full click is refused. */
	function toggle(id: string): void {
		if (picked.includes(id)) {
			picked = picked.filter((entry) => entry !== id);
			return;
		}
		if (picked.length >= maxItems) {
			toast($_('albums_cover_full', { values: { count: maxItems } }), 'danger');
			return;
		}
		picked = [...picked, id];
	}

	/** The pick's number of one frame (0 when unpicked). */
	function orderOf(id: string): number {
		return picked.indexOf(id) + 1;
	}

	/** Locale-aware capture date of one member. */
	function dateLabel(takenAt: string): string {
		return new Intl.DateTimeFormat($locale ?? 'en', { dateStyle: 'medium' }).format(new Date(takenAt));
	}

	/** The failure's line: the cover refusal states how many frames died. */
	const failureText = $derived(
		invalidCount > 0
			? $_('albums_cover_invalid', { values: { count: invalidCount } })
			: resolveErrorText($_, errorKey)
	);
</script>

<Modal bind:open title={$_('albums_cover_title')} {onclose}>
	<div class="cover-picker flex flex-col gap-3" data-cover-picker data-max={maxItems}>
		<p class="cover-picker__body m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
			{$_('albums_cover_body', { values: { count: maxItems } })}
		</p>
		<div class="cover-picker__status flex items-center justify-between gap-2">
			<span
				class="cover-picker__count font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
			>
				{#if picked.length === 0}
					{$_('albums_cover_automatic')}
				{:else}
					{$_('albums_cover_count', { values: { count: picked.length, max: maxItems } })}
					{#if remaining > 0}
						<span class="cover-picker__hint">
							· {$_('albums_cover_remaining', { values: { count: remaining } })}</span
						>
					{/if}
				{/if}
			</span>
			{#if picked.length > 0}
				<button
					class="cover-picker__clear font-body font-semibold text-[13px] leading-[1.4] text-accent bg-transparent border-0 cursor-pointer hover:underline"
					type="button"
					onclick={() => (picked = [])}
				>
					{$_('albums_cover_clear')}
				</button>
			{/if}
		</div>
		{#if items.length === 0 && goneIds.length === 0}
			<p class="cover-picker__empty m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
				{$_('albums_cover_empty')}
			</p>
		{:else}
			<ul
				class="cover-picker__grid grid grid-cols-[repeat(auto-fill,minmax(min(120px,100%),1fr))] gap-2 m-0 p-0 list-none"
			>
				{#each goneIds as goneId (goneId)}
					<li>
						<button
							class="cover-option cover-option--gone group relative flex w-full flex-col overflow-hidden rounded-control bg-card border border-border cursor-pointer transition-[border-color] duration-(--motion-fast) ease-standard hover:border-accent aria-pressed:border-accent"
							type="button"
							aria-pressed={picked.includes(goneId)}
							onclick={() => toggle(goneId)}
						>
							<span
								class="cover-option__crop relative grid aspect-square place-items-center bg-accent-soft text-text-faint"
							>
								<ImageOff size={20} aria-hidden="true" />
								<span
									class="cover-option__badge absolute top-1.5 left-1.5 inline-flex items-center justify-center w-6 h-6 rounded-full bg-accent text-accent-on font-body font-bold text-[12px] leading-none tabular-nums"
									>{orderOf(goneId)}</span
								>
							</span>
							<span class="cover-option__meta flex flex-col min-w-0 px-2 py-1.5 text-left">
								<span
									class="cover-option__file font-body font-semibold text-[13px] leading-[1.4] text-text-faint truncate"
									>{$_('albums_cover_gone')}</span
								>
							</span>
						</button>
					</li>
				{/each}
				{#each items as entry (entry.id)}
					<li>
						<button
							class="cover-option group relative flex flex-col overflow-hidden rounded-control bg-card border border-border cursor-pointer transition-[border-color] duration-(--motion-fast) ease-standard hover:border-accent aria-pressed:border-accent aria-pressed:[outline:2px_solid_var(--color-accent)] [outline-offset:-2px]"
							type="button"
							aria-pressed={picked.includes(entry.id)}
							onclick={() => toggle(entry.id)}
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
								{#if picked.includes(entry.id)}
									<span
										class="cover-option__badge absolute top-1.5 left-1.5 inline-flex items-center justify-center w-6 h-6 rounded-full bg-accent text-accent-on font-body font-bold text-[12px] leading-none tabular-nums"
										>{orderOf(entry.id)}</span
									>
								{/if}
							</span>
							<span class="cover-option__meta flex flex-col min-w-0 px-2 py-1.5 text-left">
								<span
									class="cover-option__file font-body font-semibold text-[13px] leading-[1.4] text-text truncate"
									>{entry.originalFilename}</span
								>
								<span
									class="cover-option__date font-body font-normal text-[12px] leading-[1.4] text-text-soft"
									>{dateLabel(entry.takenAt)}</span
								>
							</span>
						</button>
					</li>
				{/each}
			</ul>
		{/if}
		{#if errorKey !== ''}
			<p
				class="cover-picker__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{failureText}
			</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button
			variant="ghost"
			onclick={() => {
				open = false;
				onclose?.();
			}}
		>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<!-- 0 or N saves; a half-picked set cannot (the server refuses a
		     partial cover, and the status line already says how many are
		     missing). A collection with no photos has nothing to save --
		     unless a fully dead pin still holds cells: clearing those is
		     the way back to the rotation, so the empty collection's grid
		     renders them and the save stays live once they are gone. -->
		<Button
			disabled={busy ||
				(items.length === 0 && picked.length === 0 && !hadPin) ||
				(picked.length > 0 && picked.length < maxItems)}
			onclick={() => onsubmit(picked)}
		>
			<Check size={20} aria-hidden="true" />
			{$_('albums_cover_submit')}
		</Button>
	{/snippet}
</Modal>
