<script lang="ts">
	// The new-album dialog of the albums-as-sources design: the TITLE above, the PATH
	// below, the AUDIENCE after, and the AUTONOMY BLOCK at the
	// bottom -- creation IS the choice of an
	// existing directory (the album references it in place, zero
	// copy, zero move) plus who may see it plus the two switches the
	// edit dialog carries, landed at creation. The path field is the
	// completion combobox (PathCompletion), the audience the
	// add-style picker (AudienceField); both pickers are PORTALED
	// floating layers (the floating-layer revision), so the
	// dialog stays a fixed-height surface.
	//
	// THE AUTONOMY BLOCK (the two switches riding the create
	// request): the TIMELINE EXCLUSION toggle (the caller's own
	// per-user preference, landed in the creation itself -- it needs
	// no photos to exist) and the DELETABILITY row (「允许通过应用删除」,
	// rendered EXACTLY when the picked completion row's `writable`
	// fact is true -- the picked directory's writability is known at
	// selection time, so a read-only pick holds no row at all, no
	// dead affordance; the honest probe feedback of the landed create
	// explains the fact afterwards). The toggles' drafts seed on
	// open: the exclusion off (the album feeds its creator's
	// timeline by default), the deletability switch on (the
	// server's own default) wherever the row renders. The fact rides
	// the exact picked path: editing the field by hand drops the
	// stale fact and the row with it.
	//
	// THE DIALOG OWNS THE DRAFTS (seeded on open): the title (the
	// contract's 1-64 rune bound, checked locally so an invalid draft
	// never leaves the page), the path (non-empty and absolute -- the
	// same bound the server's clean enforces), and the audience
	// member set. THE PAGE OWNS THE WRITE: `onsubmit` receives the
	// drafts and runs the POST (an administrator's write -- the entry
	// itself is role-gated on the wall).
	//
	// THE HONEST REPORT: the creation's `skipped` list (the
	// requested ids that did not become audience -- the owner's own
	// id, an unknown or disabled account) AND the write probe's
	// `sourceWritable` fact (false is creation feedback, never
	// a failure -- a read-only directory is a legitimate album whose
	// photos cannot be deleted through the application) render in
	// the dialog before it closes, the write's own answer; the Done
	// affordance closes. The failure routing: a
	// path-matrix error renders under the path field with the
	// invalid border (the matrix's own codes all live under the
	// `album_path_` prefix), every other failure renders as the
	// dialog's general line.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import PathCompletion from './PathCompletion.svelte';
	import AudienceField, { type AudienceMember } from './AudienceField.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { sessionUserId } from '$lib/auth/session.svelte';
	import type { AlbumPathCompletion } from '$lib/api/albums';
	import FolderPlus from '@lucide/svelte/icons/folder-plus';
	import Check from '@lucide/svelte/icons/check';
	import CalendarOff from '@lucide/svelte/icons/calendar-off';
	import Trash2 from '@lucide/svelte/icons/trash-2';

	/** The two autonomy switches' drafts, as the create request carries them. */
	export type CreateAlbumSwitches = {
		/** The creator's own exclusion toggle (false = keep out of their timeline). */
		inTimeline: boolean;
		/**
		 * The deletability toggle's draft, present exactly when the
		 * picked completion row's `writable` fact rendered the row
		 * (undefined = the row does not exist, the server's default
		 * governs).
		 */
		allowAppDeletion?: boolean;
	};

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** The honest skipped report of the last landed create (null when none). */
		skipped?: readonly { userId: string; reason: 'owner' | 'invalid_account' }[] | null;
		/**
		 * The write probe's fact of the last landed create (null when
		 * none): `false` is the honest read-only feedback the report
		 * face carries -- the album exists, its photos just cannot be
		 * deleted through the application.
		 */
		sourceWritable?: boolean | null;
		/** Run the write with the drafts. */
		onsubmit: (
			title: string,
			rootPath: string,
			audience: readonly AudienceMember[],
			switches: CreateAlbumSwitches
		) => void;
		/** Acknowledge the honest report: close the dialog. */
		ondone?: () => void;
	};

	let {
		open = $bindable(false),
		busy = false,
		errorKey = '',
		skipped = null,
		sourceWritable = null,
		onsubmit,
		ondone
	}: Props = $props();

	let titleDraft = $state('');
	let pathDraft = $state('');
	let audience = $state<AudienceMember[]>([]);
	/** The exclusion toggle's draft (checked = the album feeds the creator's timeline). */
	let inTimelineDraft = $state(true);
	/** The deletability toggle's draft (undefined = the row does not render). */
	let allowDeletionDraft = $state<boolean | undefined>(undefined);
	/** The picked completion row: the write fact's source and its exact path. */
	let pickedRow = $state<AlbumPathCompletion | null>(null);
	/** Whether a submit attempt already ran for the current drafts. */
	let checked = $state(false);

	// Seed the drafts on every open; a close leaves the stale drafts
	// behind but the next open overwrites them.
	$effect(() => {
		if (open) {
			titleDraft = '';
			pathDraft = '';
			audience = [];
			inTimelineDraft = true;
			allowDeletionDraft = undefined;
			pickedRow = null;
			checked = false;
		}
	});

	/** The contract's bound, rune-counted: the trimmed title spans 1-64 characters. */
	function validTitle(): boolean {
		const runes = [...titleDraft.trim()].length;
		return runes >= 1 && runes <= 64;
	}

	/** The path's local mirror of the server's own bound: non-empty and absolute. */
	function validPath(): boolean {
		return pathDraft.trim() !== '' && pathDraft.trim().startsWith('/');
	}

	const titleViolated = $derived(checked && !validTitle());
	const pathViolated = $derived(checked && !validPath());

	// The deletability row's render gate: the picked row's write fact
	// is true AND the draft still IS that picked path -- a hand edit
	// drops the stale fact and the row with it (the server's probe
	// re-discovers the truth at creation either way).
	const deletionRowApplies = $derived(pickedRow !== null && pickedRow.writable && pathDraft === pickedRow.path);

	// A dropped row withdraws the toggle's draft too: the request
	// omits the switch, the server's default governs.
	$effect(() => {
		if (!deletionRowApplies && allowDeletionDraft !== undefined) {
			allowDeletionDraft = undefined;
		}
	});

	// The write's failure routing (the failure state): the
	// path matrix's codes all live under the album_path_ prefix and
	// render under the path field with the invalid border; anything
	// else renders as the dialog's general line.
	const pathFailure = $derived(
		errorKey !== '' && errorKey.startsWith('album_path_') && !pathViolated ? resolveErrorText($_, errorKey) : ''
	);
	const generalFailure = $derived(
		errorKey !== '' && !errorKey.startsWith('album_path_') ? resolveErrorText($_, errorKey) : ''
	);

	// The honest report's face: the landed create's own
	// answers -- the skipped audience ids and the read-only probe
	// fact -- hold the dialog until acknowledged.
	const reportFace = $derived(skipped !== null && (skipped.length > 0 || sourceWritable === false));

	function submit(): void {
		checked = true;
		if (!validTitle() || !validPath()) {
			return;
		}
		onsubmit(titleDraft.trim(), pathDraft.trim(), audience, {
			inTimeline: inTimelineDraft,
			...(allowDeletionDraft !== undefined ? { allowAppDeletion: allowDeletionDraft } : {})
		});
	}
</script>

<Modal bind:open title={$_('albums_create_title')}>
	{#if reportFace}
		<!-- The honest report of the landed create: the requested ids
		     that did not become audience and the read-only source
		     fact, the write's own answers. -->
		<div class="create-album__report flex flex-col gap-2">
			{#if skipped !== null && skipped.length > 0}
				<p
					class="create-album__report-skipped m-0 font-body font-semibold text-[15px] leading-[1.5] text-text"
					role="status"
				>
					{$_('albums_create_report', { values: { count: skipped.length } })}
				</p>
			{/if}
			{#if sourceWritable === false}
				<p
					class="create-album__report-readonly m-0 font-body font-medium text-[14px] leading-[1.5] text-text-soft"
					role="status"
				>
					{$_('albums_create_readonly')}
				</p>
			{/if}
		</div>
	{:else}
		<div class="create-album flex flex-col gap-3.5">
			<!-- The pinned order: the title ABOVE, its native ceiling
			     exactly the contract's 64 (maxlength counts UTF-16 code
			     units where the server counts runes -- a stricter cap for
			     astral text alone; the local check still owns the submit). -->
			<Input
				label={$_('albums_title_field')}
				id="create-album-title"
				name="create-album-title"
				bind:value={titleDraft}
				required
				maxlength={64}
				autocomplete="off"
				error={titleViolated ? $_('albums_title_invalid') : ''}
			/>
			<!-- ...the path BELOW: the completion combobox. The pick
			     hands the row's write fact up -- the deletability
			     row renders from it. -->
			<PathCompletion
				bind:path={pathDraft}
				id="create-album-path"
				label={$_('albums_path_field')}
				placeholder={$_('albums_path_placeholder')}
				hint={$_('albums_path_hint')}
				failure={pathViolated ? $_('albums_path_required') : pathFailure}
				{busy}
				onpick={(row) => {
					pickedRow = row;
					// The pick seeds the toggle's draft: the row exists
					// exactly for a writable pick.
					allowDeletionDraft = row.writable ? true : undefined;
				}}
			/>
			<!-- ...the audience after: the add-style picker. -->
			<AudienceField bind:members={audience} selfId={sessionUserId() ?? ''} {busy} />
			<!-- The autonomy block: the two switches ride the create
			     request -- the timeline exclusion always, the
			     deletability row exactly when the picked completion
			     row's write fact holds (a read-only or unpicked path
			     renders no row at all, no dead affordance). -->
			<div class="album-form__autonomy flex flex-col gap-2" data-testid="album-autonomy">
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
				{#if deletionRowApplies && allowDeletionDraft !== undefined}
					<!-- The deletability row: rendered ONLY for a writable
					     pick (a read-only source's row does not
					     exist -- no dead affordance); the honest probe
					     feedback of the landed create explains the fact. -->
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
								{$_('albums_deletable_label')}
							</span>
							<span
								class="album-form__toggle-note font-body font-normal text-[12px] leading-[1.4] text-text-faint"
								>{$_('albums_deletable_note')}</span
							>
						</span>
					</label>
				{/if}
			</div>
			{#if generalFailure !== ''}
				<p
					class="create-album__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
					role="alert"
				>
					{generalFailure}
				</p>
			{/if}
		</div>
	{/if}
	{#snippet footer()}
		{#if reportFace}
			<Button onclick={ondone}>
				<Check size={20} aria-hidden="true" />
				{$_('albums_create_done')}
			</Button>
		{:else}
			<Button variant="ghost" disabled={busy} onclick={() => (open = false)}>
				{$_('common_cancel')}
			</Button>
			<Button disabled={busy} onclick={submit}>
				<FolderPlus size={20} aria-hidden="true" />
				{$_('albums_create_submit')}
			</Button>
		{/if}
	{/snippet}
</Modal>
