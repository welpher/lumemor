<script lang="ts">
	// The floating batch bar of the timeline wall: once at least one
	// tile is selected it rises from the bottom edge and carries the
	// six set actions of the batch API (favorite and unfavorite
	// collapse into one toggle labeled by the selection's state),
	// the selected count, a clear action (Escape does the same from
	// the page) and the last failed action's message. The bar owns
	// presentation only: the page runs the actions and owns the
	// selection. The add-tag and add-to-memory affordances ride
	// optional callbacks (each flow carries a dialog and an endpoint
	// of its own, not a batch action); the membership removal of the
	// memory wall rides one too -- the write is the memory's own
	// curation endpoint, and the wall that composes this bar runs it
	// with the same choreography as a batch action.
	//
	// THE ICON-FIRST RULE:
	// the bar is a TOOLBAR, so every action is an icon-only button
	// (lucide, 20px, decorative) whose accessible name rides the
	// aria-label -- the count line beside them carries the context,
	// and the labels are the actions' own i18n strings.
	//
	// THE LAYERING: the bar paints above the upload queue's panel
	// (--z-panel, the bottom-right status surface) -- the curation
	// bar is the ACTIVE interaction and its row must stay clickable
	// while uploads run; the passive panel yields the overlap. The
	// bar stays below the info drawer (--z-aside) and every dialog
	// (--z-modal).
	import { _ } from 'svelte-i18n';
	import IconButton from '$lib/components/IconButton.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import type { BatchAction } from '$lib/api/timeline';
	import Tag from '@lucide/svelte/icons/tag';
	import Sparkles from '@lucide/svelte/icons/sparkles';
	import Minus from '@lucide/svelte/icons/minus';
	import Star from '@lucide/svelte/icons/star';
	import Archive from '@lucide/svelte/icons/archive';
	import ArchiveRestore from '@lucide/svelte/icons/archive-restore';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import X from '@lucide/svelte/icons/x';

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
		 * (unfavorite on favorites, unhide on archived).
		 */
		actions?: readonly BatchAction[];
		/**
		 * Whether the selection holds an archived tile. The
		 * unarchive button renders only then: a family-flow wall
		 * never checks an archived row (its bar has no unarchive
		 * entry), the archived wall's rows are archived by
		 * construction, and the album wall shows the container's
		 * unarchive entry exactly when the owner checked an archived
		 * member.
		 */
		archivedSelected?: boolean;
		/**
		 * Open the add-tag dialog over the current selection (the
		 * timeline's bar offers it; the curated walls do not). The tag
		 * flow is not a batch action -- it carries its own dialog
		 * and endpoint -- so it rides this callback instead of the
		 * action set.
		 */
		onaddtag?: () => void;
		/**
		 * Open the memory picker over the current selection (the
		 * wall-selection entry): the memory flow carries its own
		 * dialog and write, not a batch action -- the same optional
		 * callback shape as the tag opener.
		 */
		onaddmemory?: () => void;
		/**
		 * Remove the selected rows' memberships from the memory the
		 * wall renders (the memory's own curation write). The wall
		 * runs it with the batch choreography -- busy, the success
		 * arm clearing the selection, a failure keeping it -- while
		 * the bar stays presentation-only.
		 */
		onremovememory?: () => void;
		/**
		 * Open the selected rows' permanent-delete confirmation (the
		 * trash view's bar carries it; every wall that runs the
		 * reversible batch actions does not). The delete flow carries
		 * its own dialog and endpoint -- the same optional-callback
		 * shape as the tag opener, and the page runs the destructive
		 * write only past its own confirm gate.
		 */
		ondeleteselected?: () => void;
		/** Run one batch action over the current selection. */
		onaction: (action: BatchAction) => void;
		/** Clear the selection. */
		onclear: () => void;
	};

	let {
		count,
		allFavorite,
		errorKey,
		busy,
		actions,
		archivedSelected = false,
		onaddtag,
		onaddmemory,
		onremovememory,
		ondeleteselected,
		onaction,
		onclear
	}: Props = $props();

	// The default is the full set of the bar's buttons; a caller's
	// subset keeps its order (the toggle, the archive pair, trash,
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
		class="batchbar fixed left-1/2 bottom-5 z-(--z-toolbar-active) box-content flex flex-wrap items-center justify-center gap-2 max-w-[min(92vw,720px)] px-3.5 py-2.5 border border-border rounded-overlay bg-card shadow-overlay [transform:translateX(-50%)]"
		role="toolbar"
		aria-label={$_('batch_selected_count', { values: { count } })}
	>
		<p
			class="batchbar__count mr-1.5 font-body font-bold text-[15px] leading-[1.4] text-text whitespace-nowrap tabular-nums"
		>
			{$_('batch_selected_count', { values: { count } })}
		</p>

		{#if onaddtag !== undefined}
			<IconButton
				class="batchbar__action"
				variant="ghost"
				disabled={busy}
				aria-label={$_('action_add_tag')}
				title={$_('action_add_tag')}
				onclick={onaddtag}
			>
				<Tag size={20} aria-hidden="true" />
			</IconButton>
		{/if}

		{#if onaddmemory !== undefined}
			<IconButton
				class="batchbar__action batchbar__action--add-memory"
				variant="ghost"
				disabled={busy}
				aria-label={$_('action_add_memory')}
				title={$_('action_add_memory')}
				onclick={onaddmemory}
			>
				<Sparkles size={20} aria-hidden="true" />
			</IconButton>
		{/if}

		{#if offers('favorite')}
			<IconButton
				class="batchbar__action batchbar__action--favorite"
				variant="ghost"
				disabled={busy}
				aria-label={allFavorite ? $_('action_unfavorite') : $_('action_favorite')}
				title={allFavorite ? $_('action_unfavorite') : $_('action_favorite')}
				onclick={() => onaction(allFavorite ? 'unfavorite' : 'favorite')}
			>
				<Star size={20} aria-hidden="true" />
			</IconButton>
		{/if}
		{#if offers('hide')}
			<IconButton
				class="batchbar__action batchbar__action--archive"
				variant="ghost"
				disabled={busy}
				aria-label={$_('asset_archive')}
				title={$_('asset_archive')}
				onclick={() => onaction('hide')}
			>
				<Archive size={20} aria-hidden="true" />
			</IconButton>
		{/if}
		{#if offers('unhide') && archivedSelected}
			<IconButton
				class="batchbar__action batchbar__action--unarchive"
				variant="ghost"
				disabled={busy}
				aria-label={$_('asset_unarchive')}
				title={$_('asset_unarchive')}
				onclick={() => onaction('unhide')}
			>
				<ArchiveRestore size={20} aria-hidden="true" />
			</IconButton>
		{/if}
		{#if offers('trash')}
			<IconButton
				class="batchbar__action batchbar__action--trash"
				variant="danger"
				disabled={busy}
				aria-label={$_('action_trash')}
				title={$_('action_trash')}
				onclick={() => onaction('trash')}
			>
				<Trash2 size={20} aria-hidden="true" />
			</IconButton>
		{/if}
		{#if offers('restore')}
			<IconButton
				class="batchbar__action batchbar__action--restore"
				variant="ghost"
				disabled={busy}
				aria-label={$_('action_restore')}
				title={$_('action_restore')}
				onclick={() => onaction('restore')}
			>
				<RotateCcw size={20} aria-hidden="true" />
			</IconButton>
		{/if}

		{#if onremovememory !== undefined}
			<IconButton
				class="batchbar__action batchbar__action--remove-memory"
				variant="danger"
				disabled={busy}
				aria-label={$_('action_remove_from_memory')}
				title={$_('action_remove_from_memory')}
				onclick={onremovememory}
			>
				<Minus size={20} aria-hidden="true" />
			</IconButton>
		{/if}

		{#if ondeleteselected !== undefined}
			<!-- The destructive arm opens the page's confirm dialog instead
			     of running (the irreversible action is never one click
			     away): the callback's presence is the capability, the
			     confirm gate stays the page's own. -->
			<IconButton
				class="batchbar__action batchbar__action--delete"
				variant="danger"
				disabled={busy}
				aria-label={$_('trash_delete_permanently')}
				title={$_('trash_delete_permanently')}
				onclick={ondeleteselected}
			>
				<Trash2 size={20} aria-hidden="true" />
			</IconButton>
		{/if}

		<IconButton
			class="batchbar__clear"
			variant="ghost"
			aria-label={$_('action_clear_selection')}
			title={$_('action_clear_selection')}
			onclick={onclear}
		>
			<X size={20} aria-hidden="true" />
		</IconButton>

		{#if errorKey !== ''}
			<p
				class="batchbar__error basis-full font-body font-normal text-[13px] leading-[1.4] text-danger-text text-center"
				role="alert"
			>
				{errorText(errorKey)}
			</p>
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
