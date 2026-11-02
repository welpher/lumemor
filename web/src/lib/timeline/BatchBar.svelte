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
		/** Run one batch action over the current selection. */
		onaction: (action: BatchAction) => void;
		/** Clear the selection. */
		onclear: () => void;
	};

	let { count, allFavorite, errorKey, busy, onaction, onclear }: Props = $props();

	// The shared unknown-code fallback (see $lib/i18n/errors): an
	// unknown error code has no i18n key, and svelte-i18n would echo
	// the key itself -- such codes render the generic message. The
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}
</script>

{#if count > 0}
	<div class="batchbar" role="toolbar" aria-label={$_('batch_selected_count', { values: { count } })}>
		<p class="batchbar__count">{$_('batch_selected_count', { values: { count } })}</p>

		<Button variant="ghost" disabled={busy} onclick={() => onaction(allFavorite ? 'unfavorite' : 'favorite')}>
			{allFavorite ? $_('action_unfavorite') : $_('action_favorite')}
		</Button>
		<Button variant="ghost" disabled={busy} onclick={() => onaction('hide')}>{$_('action_hide')}</Button>
		<Button variant="ghost" disabled={busy} onclick={() => onaction('unhide')}>{$_('action_unhide')}</Button>
		<Button variant="danger" disabled={busy} onclick={() => onaction('trash')}>{$_('action_trash')}</Button>
		<Button variant="ghost" disabled={busy} onclick={() => onaction('restore')}>{$_('action_restore')}</Button>

		<button class="batchbar__clear" type="button" aria-label={$_('action_clear_selection')} onclick={onclear}>
			<svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden="true">
				<path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
			</svg>
		</button>

		{#if errorKey !== ''}
			<p class="batchbar__error" role="alert">{errorText(errorKey)}</p>
		{/if}
	</div>
{/if}

<style>
	.batchbar {
		position: fixed;
		left: 50%;
		bottom: 20px;
		transform: translateX(-50%);
		z-index: 60;
		display: flex;
		align-items: center;
		gap: 8px;
		max-width: min(92vw, 720px);
		padding: 10px 14px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-overlay);
		background-color: var(--color-card);
		box-shadow: var(--shadow-overlay);
		animation: batchbar-in var(--motion-base) var(--ease-standard);
		flex-wrap: wrap;
		justify-content: center;
	}

	.batchbar__count {
		margin: 0 6px 0 0;
		font: 700 15px/1.4 var(--font-body);
		color: var(--color-text);
		white-space: nowrap;
	}

	.batchbar__clear {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 34px;
		height: 34px;
		border: none;
		border-radius: var(--radius-control);
		color: var(--color-text-soft);
		background: none;
		cursor: pointer;
		transition:
			background-color var(--motion-fast) var(--ease-standard),
			color var(--motion-fast) var(--ease-standard);
	}

	.batchbar__clear:hover {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}

	.batchbar__error {
		flex-basis: 100%;
		margin: 0;
		font: 400 13px/1.4 var(--font-body);
		color: var(--color-danger-text);
		text-align: center;
	}

	@keyframes batchbar-in {
		from {
			opacity: 0;
			transform: translate(-50%, 8px);
		}
	}
</style>
