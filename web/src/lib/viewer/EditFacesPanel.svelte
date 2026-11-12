<script lang="ts">
	// The edit-people side panel of the asset viewer: the face-level
	// correction surface. It slides in REPLACING the information
	// panel (the same docked/overlay form, the photo always visible
	// beside it) and lists EVERY face of this photo -- named,
	// unnamed, not-yet-grouped -- one row per face with the 40px
	// crop, the face's person, and the row's actions: the reassign
	// pencil (every row -- picking the target is the choose-person
	// dialog, which also holds the create-from-face row) and the
	// detach trash (NAMED rows only: an unnamed or not-yet-grouped
	// face is nobody's member, so there is nothing to remove it
	// from -- the write would answer the honest 404, and the
	// affordance never lies about that).
	//
	// THE ROOT ASYMMETRY, kept visible in every label: naming acts
	// on the CLUSTER (the whole group shares the name); this panel
	// acts on ONE FACE -- reassign moves this face, detach releases
	// this face, the create seeds a new person with this face.
	//
	// THE STAGING: the panel renders the page's staged edits (the
	// pencil's dialog stages them; the top-right check commits them
	// once -- a pending row shows its staged label with the accent
	// dot). The page owns the staging record and every network
	// call; this panel owns presentation alone.
	import { _ } from 'svelte-i18n';
	import { type AssetDetail, type AssetFaceSummary } from '$lib/api/assets';
	import { resolveErrorText } from '$lib/i18n/errors';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import PenLine from '@lucide/svelte/icons/pen-line';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Check from '@lucide/svelte/icons/check';

	type Props = {
		/** The current asset's detail (the page's local copy). */
		detail: AssetDetail;
		/**
		 * The staged label of one face, when an edit is pending on it
		 * (the reassign target's name, or the create row's own label);
		 * undefined when the face carries no staged edit.
		 */
		stageLabelOf: (face: AssetFaceSummary) => string | undefined;
		/** How many faces carry a staged edit (the apply button's gate). */
		stagedCount: number;
		/** A commit is in flight: the rows and the apply button disable. */
		busy: boolean;
		/** The error-code i18n key of the last failed commit ('' when none). */
		errorKey: string;
		/** Return to the information panel. */
		onclose: () => void;
		/** Open the choose-person dialog for one face (reassign or create). */
		onreassign: (face: AssetFaceSummary) => void;
		/** Open the detach confirm for one face (named rows only). */
		ondetach: (face: AssetFaceSummary) => void;
		/** Commit every staged edit (the one-shot apply). */
		onapply: () => void;
	};

	let { detail, stageLabelOf, stagedCount, busy, errorKey, onclose, onreassign, ondetach, onapply }: Props = $props();

	/**
	 * The row's person label, the chips' own vocabulary: the caller's
	 * label of the face's cluster, the unnamed state, or the
	 * not-grouped state of a face the clustering has not assigned yet.
	 */
	function faceLabel(face: AssetFaceSummary): string {
		if (face.label !== null) {
			return face.label.name;
		}
		return face.clusterId !== null ? $_('viewer_faces_unnamed') : $_('viewer_faces_unclustered');
	}

	/**
	 * The detach affordance's gate: a NAMED person's row alone. The
	 * label is the person -- the confirm names it, and the write
	 * releases the face from it; an unnamed cluster holds nobody to
	 * release from (the reassign pencil is that row's one action).
	 */
	function isDetachable(face: AssetFaceSummary): boolean {
		return face.clusterId !== null && face.label !== null;
	}
</script>

<!-- The information panel's own form (the identity seam of the two
     asides): the fixed right-edge overlay below lg, the STATIC
     docked region of the viewer's flex row at lg+, so the photo
     stays visible and re-fits while this panel stands in the
     drawer's place. -->
<aside
	class="faces-panel fixed top-0 right-0 bottom-0 z-(--z-aside) flex flex-col w-[min(360px,92vw)] bg-card border-l border-border shadow-overlay lg:static lg:shrink-0 lg:shadow-none"
	aria-label={$_('people_edit_faces')}
>
	<header class="faces-panel__header flex items-center gap-2 px-3 py-2.5 border-b border-border">
		<!-- Back to the information panel (the destination is the info
		     panel's own accessible name). -->
		<button
			class="faces-panel__back inline-flex items-center justify-center w-8 h-8 shrink-0 rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
			type="button"
			aria-label={$_('viewer_info_title')}
			onclick={onclose}
		>
			<ChevronLeft size={20} />
		</button>
		<h2 class="faces-panel__title m-0 flex-1 truncate font-body font-semibold text-[15px] leading-[1.4] text-text">
			{$_('people_edit_faces')}
		</h2>
		<!-- The one-shot commit: the staged edits (reassigns and
		     creates) write here, once, and the done echo counts them. -->
		<button
			class="faces-panel__apply inline-flex items-center justify-center w-8 h-8 shrink-0 rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text disabled:cursor-not-allowed disabled:opacity-55"
			type="button"
			aria-label={$_('people_edit_faces_apply')}
			title={$_('people_edit_faces_apply')}
			data-staged-count={stagedCount}
			disabled={busy || stagedCount === 0}
			onclick={onapply}
		>
			<Check size={18} />
		</button>
	</header>

	<div class="faces-panel__scroll flex-1 overflow-y-auto [scrollbar-gutter:stable]">
		<!-- The commit's honest failure: the loop stopped at the first
		     refusal; the uncommitted stages stay staged for a retry. -->
		{#if errorKey !== ''}
			<p
				class="faces-panel__error m-0 mt-3 px-4 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{resolveErrorText($_, errorKey)}
			</p>
		{/if}
		<p class="faces-panel__hint m-0 px-4 pt-3 pb-2 font-body font-normal text-[13px] leading-[1.5] text-text-soft">
			{$_('people_edit_faces_hint')}
		</p>

		<ul class="faces-panel__rows m-0 p-0 list-none divide-y divide-border">
			{#each detail.faces as face (face.id)}
				{@const stage = stageLabelOf(face)}
				{@const detachable = isDetachable(face)}
				<li
					class="faces-panel__row flex items-center gap-2.5 px-4 py-2.5"
					data-face-id={face.id}
					data-staged={stage !== undefined}
				>
					<img
						class="faces-panel__crop w-10 h-10 shrink-0 rounded-control object-cover bg-accent-soft"
						src={face.thumbnail}
						alt=""
						loading="lazy"
						decoding="async"
						draggable="false"
					/>
					<span class="min-w-0 flex-1 flex items-center gap-1.5">
						<!-- The pending marker: a staged edit swaps the row's
						     label for its staged one and carries the accent
						     dot until the commit lands. -->
						{#if stage !== undefined}
							<span
								class="faces-panel__pending w-1.5 h-1.5 rounded-full bg-accent shrink-0"
								aria-hidden="true"
							></span>
						{/if}
						<span
							class="faces-panel__name block truncate font-body font-semibold text-[14px] leading-[1.4] {face.label ===
							null
								? 'text-text-soft'
								: 'text-text'}"
						>
							{stage ?? faceLabel(face)}
						</span>
					</span>
					<button
						class="faces-panel__reassign inline-flex items-center justify-center w-8 h-8 shrink-0 rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text disabled:cursor-not-allowed disabled:opacity-55"
						type="button"
						aria-label={$_('people_face_reassign')}
						title={$_('people_face_reassign')}
						disabled={busy}
						onclick={() => onreassign(face)}
					>
						<PenLine size={18} />
					</button>
					{#if detachable}
						<!-- The destructive hover tint carries the danger tone
						     (the trash releases the face from its person). -->
						<button
							class="faces-panel__detach inline-flex items-center justify-center w-8 h-8 shrink-0 rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)] hover:text-danger-text disabled:cursor-not-allowed disabled:opacity-55"
							type="button"
							aria-label={$_('people_face_detach')}
							title={$_('people_face_detach')}
							disabled={busy}
							onclick={() => ondetach(face)}
						>
							<Trash2 size={18} />
						</button>
					{/if}
				</li>
			{/each}
		</ul>
	</div>
</aside>

<style>
	/* Residual styles (the drawer's own): the slide-in keyframes and
	 * the panel's animation declaration. */
	.faces-panel {
		animation: faces-panel-in var(--motion-base) var(--ease-standard);
	}

	@keyframes faces-panel-in {
		from {
			transform: translateX(16px);
			opacity: 0;
		}
	}
</style>
