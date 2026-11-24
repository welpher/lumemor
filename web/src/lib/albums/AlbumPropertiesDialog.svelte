<script lang="ts" module>
	import type { AudienceMember } from './AudienceField.svelte';

	/** The autonomy block's committed draft, handed to the page's writes. */
	export type AlbumAutonomyDraft = {
		title: string;
		audience: readonly AudienceMember[];
		/** The caller's own timeline preference draft (undefined = not excludable). */
		inTimeline: boolean | undefined;
		/** The deletability switch's draft (undefined = the row does not render). */
		allowAppWrites: boolean | undefined;
	};
</script>

<script lang="ts">
	// The edit-properties dialog of the albums surface, reshaped by the
	// albums-as-sources rework and completed by
	// album autonomy, then simplified by the lock's retirement: the
	// TITLE, the SOURCE row (the directory
	// album's root path, read-only with the "source" tag -- the path
	// is pinned at creation, the property patch carries no field for
	// it), the AUDIENCE (the current grants as chips, the add-style
	// picker for the candidates), and the two-toggle autonomy
	// block -- the TIMELINE EXCLUSION (the caller's own per-user
	// preference) and the DELETABILITY row (「允许通过应用删除」,
	// rendered EXACTLY when the source was writable at creation -- a
	// read-only album's row does not exist, no dead affordance). The
	// dialog is live for every write, every time: the retired lock's
	// unlock-first sequencing and its frozen fields went with it
	// (there is no state left that refuses a property write).
	//
	// The page owns the writes: `onsubmit` receives the whole draft
	// (the trimmed title, the FULL audience member set -- the PUT's
	// full-set semantics -- and the two toggles' states); the page
	// diffs each against the live detail and writes what changed.
	// The dialog owns the drafts' lifecycle (seeded on open from the
	// detail's own fields) and the local title bound (the server
	// trims and counts runes; the client mirrors both so an invalid
	// draft never leaves the page -- the inline message says why
	// instead, and clears the moment the draft becomes valid). The
	// write's failure renders as the dialog's general line.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import AudienceField from './AudienceField.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { sessionUserId } from '$lib/auth/session.svelte';
	import Folder from '@lucide/svelte/icons/folder';
	import CalendarOff from '@lucide/svelte/icons/calendar-off';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Save from '@lucide/svelte/icons/save';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The i18n key of the dialog title. */
		titleKey: string;
		/** The i18n key of the submit label. */
		submitKey: string;
		/** The drafts' seeds (the detail's current values). */
		initialTitle?: string;
		/**
		 * The directory album's source path; the read-only source row
		 * renders exactly when set (memory and sync albums hold none).
		 */
		rootPath?: string;
		/** The current grants -- the detail's `shares` array, the chips' seed. */
		shares: readonly { userId: string; email: string }[];
		/**
		 * The exclusion draft's seed (the detail's `inTimeline`); the
		 * row renders exactly when the album is excludable for the
		 * caller (undefined).
		 */
		initialInTimeline?: boolean;
		/**
		 * Whether the source was writable at creation (the detail's
		 * `sourceWritable`): the deletability row renders exactly
		 * when true -- a read-only album holds no row at all.
		 */
		sourceWritable?: boolean;
		/**
		 * The deletability draft's seed (the detail's
		 * `allowAppWrites`, the owner's own view).
		 */
		initialAllowDeletion?: boolean;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Run the writes with the whole draft. */
		onsubmit: (draft: AlbumAutonomyDraft) => void;
	};

	let {
		open = $bindable(false),
		titleKey,
		submitKey,
		initialTitle = '',
		rootPath = '',
		shares,
		initialInTimeline = undefined,
		sourceWritable = false,
		initialAllowDeletion = true,
		busy = false,
		errorKey = '',
		onsubmit
	}: Props = $props();

	let titleDraft = $state('');
	let audience = $state<AudienceMember[]>([]);
	let inTimelineDraft = $state<boolean | undefined>(undefined);
	let allowDeletionDraft = $state<boolean | undefined>(undefined);
	/** Whether a submit attempt already ran for the current draft. */
	let checked = $state(false);

	// Seed the drafts on every open (the current title, the current
	// grants, the two toggles); a close leaves the stale drafts
	// behind but the next open overwrites them.
	$effect(() => {
		if (open) {
			titleDraft = initialTitle;
			audience = shares.map((share) => ({ userId: share.userId, email: share.email }));
			inTimelineDraft = initialInTimeline;
			allowDeletionDraft = sourceWritable ? initialAllowDeletion : undefined;
			checked = false;
		}
	});

	/** The contract's bound, rune-counted: the trimmed title spans 1-64 characters. */
	function validDraft(): boolean {
		const runes = [...titleDraft.trim()].length;
		return runes >= 1 && runes <= 64;
	}

	const boundViolated = $derived(checked && !validDraft());

	function submit(): void {
		checked = true;
		if (!validDraft()) {
			return;
		}
		onsubmit({
			title: titleDraft.trim(),
			audience,
			inTimeline: inTimelineDraft,
			allowAppWrites: allowDeletionDraft
		});
	}
</script>

<Modal bind:open title={$_(titleKey)}>
	<div class="album-form flex flex-col gap-3.5">
		<Input
			label={$_('albums_title_field')}
			id="album-title"
			name="album-title"
			bind:value={titleDraft}
			required
			maxlength={64}
			autocomplete="off"
			error={boundViolated ? $_('albums_title_invalid') : ''}
		/>
		{#if rootPath !== ''}
			<!-- The source row: read-only with the honest tag -- the
			     album's source is pinned at creation, the edit carries
			     no field for it (the source row is read-only). -->
			<div
				class="album-form__source flex items-center gap-2 px-3 py-2.5 bg-page border border-border rounded-control"
			>
				<Folder size={18} class="text-accent shrink-0" aria-hidden="true" />
				<span
					class="album-form__source-path font-body font-normal text-[13px] leading-[1.4] text-text-soft overflow-hidden text-ellipsis whitespace-nowrap"
					>{rootPath}</span
				>
				<span
					class="album-form__source-tag shrink-0 font-body font-bold text-[11px] leading-[1.4] text-accent-soft-text bg-accent-soft px-2 py-0.5 rounded-full"
					>{$_('albums_source_tag')}</span
				>
			</div>
		{/if}
		<!-- The audience: the folded-in share management (the current
		     grants as the chips, the picker for the candidates). -->
		<AudienceField bind:members={audience} selfId={sessionUserId() ?? ''} {busy} />
		<!-- The autonomy block (the two toggles ride one
		     block -- the timeline exclusion and the deletability
		     row, the source's own setting). -->
		<div class="album-form__autonomy flex flex-col gap-2" data-testid="album-autonomy">
			{#if inTimelineDraft !== undefined}
				<label
					class="album-form__toggle flex items-start gap-2.5 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
				>
					<input
						class="album-form__toggle-input mt-0.5 w-[18px] h-[18px] accent-accent shrink-0"
						type="checkbox"
						data-testid="album-timeline-toggle"
						bind:checked={inTimelineDraft}
					/>
					<span class="flex flex-col gap-0.5">
						<span class="album-form__toggle-label flex items-center gap-1.5">
							<CalendarOff size={14} class="text-text-faint shrink-0" aria-hidden="true" />
							{$_('albums_timeline_label')}
						</span>
						<span
							class="album-form__toggle-note font-body font-normal text-[12px] leading-[1.4] text-text-faint"
							>{$_('albums_timeline_note')}</span
						>
					</span>
				</label>
			{/if}
			{#if sourceWritable && allowDeletionDraft !== undefined}
				<!-- The deletability row: rendered ONLY for a writable
				     source (a read-only album's row does not
				     exist -- no dead affordance). -->
				<label
					class="album-form__toggle flex items-start gap-2.5 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
				>
					<input
						class="album-form__toggle-input mt-0.5 w-[18px] h-[18px] accent-accent shrink-0"
						type="checkbox"
						data-testid="album-deletable-toggle"
						bind:checked={allowDeletionDraft}
					/>
					<span class="flex flex-col gap-0.5">
						<span class="album-form__toggle-label flex items-center gap-1.5">
							<Trash2 size={14} class="text-text-faint shrink-0" aria-hidden="true" />
							{$_('albums_writes_label')}
						</span>
						<span
							class="album-form__toggle-note font-body font-normal text-[12px] leading-[1.4] text-text-faint"
							>{$_('albums_writes_note')}</span
						>
					</span>
				</label>
			{/if}
		</div>
		{#if errorKey !== ''}
			<p
				class="album-form__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{resolveErrorText($_, errorKey)}
			</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={() => (open = false)}>
			{$_('common_cancel')}
		</Button>
		<Button disabled={busy} onclick={submit}>
			<Save size={20} aria-hidden="true" />
			{$_(submitKey)}
		</Button>
	{/snippet}
</Modal>
