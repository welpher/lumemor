<script lang="ts">
	// The floating batch bar of the timeline wall: once at least one
	// tile is selected it rises from the bottom edge and carries the
	// six set actions of the batch API (favorite and unfavorite
	// collapse into one toggle labeled by the selection's state),
	// the selected count, a clear action (Escape does the same from
	// the page) and the last failed action's message. The bar owns
	// presentation only: the page runs the actions and owns the
	// selection.
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import type { BatchAction } from '$lib/api/timeline';

	type Props = {
		/** How many tiles the selection holds. */
		count: number;
		/** Whether every selected tile is already a favorite. */
		allFavorite: boolean;
		/** The error-code i18n key of the last failed action ('' when none). */
		errorKey: string;
		/** Actions run one at a time: buttons disable while in flight. */
		busy: boolean;
		/**
		 * Which of the bar's actions render, keyed like the batch API
		 * ('favorite' names the favorite/unfavorite toggle). The
		 * timeline passes nothing and gets every action; the curated
		 * walls pass the subset that their view makes sense of
		 * (unfavorite on favorites, unhide on hidden).
		 */
		actions?: readonly BatchAction[];
		/** Run one batch action over the current selection. */
		onaction: (action: BatchAction) => void;
		/** Clear the selection. */
		onclear: () => void;
	};

	let { count, allFavorite, errorKey, busy, actions, onaction, onclear }: Props = $props();

	// The default is the full set of the bar's buttons; a caller's
	// subset keeps its order (the toggle, hide, unhide, trash,
	// restore -- the timeline's order).
	const shown = $derived(actions ?? ['favorite', 'hide', 'unhide', 'trash', 'restore']);

	/** Whether one action's button renders. */
	function offers(action: BatchAction): boolean {
		return shown.includes(action);
	}

	// The shared unknown-code fallback (see $lib/i18n/errors): an
	// unknown error code has no i18n key, and svelte-i18n would echo
	// the key itself -- such codes render the generic message. The
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}
</script>

{#if count > 0}
	<div
		class="batchbar fixed left-1/2 bottom-5 z-[60] flex flex-wrap items-center justify-center gap-2 max-w-[min(92vw,720px)] px-3.5 py-2.5 border border-border rounded-overlay bg-card shadow-overlay [transform:translateX(-50%)]"
		role="toolbar"
		aria-label={$_('batch_selected_count', { values: { count } })}
	>
		<p class="batchbar__count mr-1.5 font-body font-bold text-[15px] leading-[1.4] text-text whitespace-nowrap">{$_('batch_selected_count', { values: { count } })}</p>

		{#if offers('favorite')}
			<Button variant="ghost" disabled={busy} onclick={() => onaction(allFavorite ? 'unfavorite' : 'favorite')}>
				{allFavorite ? $_('action_unfavorite') : $_('action_favorite')}
			</Button>
		{/if}
		{#if offers('hide')}
			<Button variant="ghost" disabled={busy} onclick={() => onaction('hide')}>{$_('action_hide')}</Button>
		{/if}
		{#if offers('unhide')}
			<Button variant="ghost" disabled={busy} onclick={() => onaction('unhide')}>{$_('action_unhide')}</Button>
		{/if}
		{#if offers('trash')}
			<Button variant="danger" disabled={busy} onclick={() => onaction('trash')}>{$_('action_trash')}</Button>
		{/if}
		{#if offers('restore')}
			<Button variant="ghost" disabled={busy} onclick={() => onaction('restore')}>{$_('action_restore')}</Button>
		{/if}

		<button
			class="batchbar__clear inline-flex items-center justify-center w-[34px] h-[34px] rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
			type="button"
			aria-label={$_('action_clear_selection')}
			onclick={onclear}
		>
			<svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden="true">
				<path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
			</svg>
		</button>

		{#if errorKey !== ''}
			<p class="batchbar__error basis-full font-body font-normal text-[13px] leading-[1.4] text-danger-text text-center" role="alert">{errorText(errorKey)}</p>
		{/if}
	</div>
{/if}

<style>
	/* The rise-in motion (verbatim): the keyframes and the bar's
	 * animation declaration are the residual styles. The keyframes
	 * animate `transform` over the element's [transform:translateX]
	 * utility, exactly like the previous rule pair. */
	.batchbar {
		animation: batchbar-in var(--motion-base) var(--ease-standard);
	}

	@keyframes batchbar-in {
		from {
			opacity: 0;
			transform: translate(-50%, 8px);
		}
	}
</style>
