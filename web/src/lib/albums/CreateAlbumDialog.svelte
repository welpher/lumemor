<script lang="ts">
	// The new-album dialog of the albums-as-sources design (web IA
	// §7.5, the specimen's pinned order): the TITLE above, the PATH
	// below, the AUDIENCE after -- creation IS the choice of an
	// existing directory (the album references it in place, zero
	// copy, zero move) plus who may see it. The path field is the
	// completion combobox (PathCompletion), the audience the
	// add-style picker (AudienceField); both pickers are PORTALED
	// floating layers (§7.5's 2026-11-09 second revision), so the
	// dialog stays a fixed-height surface.
	//
	// THE DIALOG OWNS THE DRAFTS (seeded on open): the title (the
	// contract's 1-64 rune bound, checked locally so an invalid draft
	// never leaves the page), the path (non-empty and absolute -- the
	// same bound the server's clean enforces), and the audience
	// member set. THE PAGE OWNS THE WRITE: `onsubmit` receives the
	// triple and runs the POST (an administrator's write -- the entry
	// itself is role-gated on the wall).
	//
	// THE HONEST REPORT (E6): the creation's `skipped` list (the
	// requested ids that did not become audience -- the owner's own
	// id, an unknown or disabled account) AND the write probe's
	// `sourceWritable` fact (§7.9: false is creation feedback, never
	// a failure -- a read-only directory is a legitimate album whose
	// photos cannot be deleted through the application) render in
	// the dialog before it closes, the write's own answer; the Done
	// affordance closes. The failure routing follows the specimen: a
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
	import FolderPlus from '@lucide/svelte/icons/folder-plus';
	import Check from '@lucide/svelte/icons/check';

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
		onsubmit: (title: string, rootPath: string, audience: readonly AudienceMember[]) => void;
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
	/** Whether a submit attempt already ran for the current drafts. */
	let checked = $state(false);

	// Seed the drafts on every open; a close leaves the stale drafts
	// behind but the next open overwrites them.
	$effect(() => {
		if (open) {
			titleDraft = '';
			pathDraft = '';
			audience = [];
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

	// The write's failure routing (the specimen's state four): the
	// path matrix's codes all live under the album_path_ prefix and
	// render under the path field with the invalid border; anything
	// else renders as the dialog's general line.
	const pathFailure = $derived(
		errorKey !== '' && errorKey.startsWith('album_path_') && !pathViolated ? resolveErrorText($_, errorKey) : ''
	);
	const generalFailure = $derived(
		errorKey !== '' && !errorKey.startsWith('album_path_') ? resolveErrorText($_, errorKey) : ''
	);

	// The honest report's face (E6, §7.9): the landed create's own
	// answers -- the skipped audience ids and the read-only probe
	// fact -- hold the dialog until acknowledged.
	const reportFace = $derived(skipped !== null && (skipped.length > 0 || sourceWritable === false));

	function submit(): void {
		checked = true;
		if (!validTitle() || !validPath()) {
			return;
		}
		onsubmit(titleDraft.trim(), pathDraft.trim(), audience);
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
			<!-- The pinned order: the title ABOVE... -->
			<Input
				label={$_('albums_title_field')}
				id="create-album-title"
				name="create-album-title"
				bind:value={titleDraft}
				maxlength={128}
				autocomplete="off"
				error={titleViolated ? $_('albums_title_invalid') : ''}
			/>
			<!-- ...the path BELOW: the completion combobox. -->
			<PathCompletion
				bind:path={pathDraft}
				id="create-album-path"
				label={$_('albums_path_field')}
				placeholder={$_('albums_path_placeholder')}
				hint={$_('albums_path_hint')}
				failure={pathViolated ? $_('albums_path_required') : pathFailure}
				{busy}
			/>
			<!-- ...the audience after: the add-style picker. -->
			<AudienceField bind:members={audience} selfId={sessionUserId() ?? ''} {busy} />
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
