<script lang="ts">
	// The new-memory dialog: the TITLE above, the AUDIENCE after --
	// creation is naming a reference set and naming who may see it,
	// NOTHING ELSE. A memory owns no source (no path field, no path
	// completion, no source row could ever exist), no deletability
	// switch (its delete never moves files) and no timeline switch
	// (it never feeds its creator's timeline -- the exclusion is the
	// memory's fixed semantics, landed with the row itself, so there
	// is nothing to toggle at creation). The write is EVERY
	// MEMBER'S: a memory is personal curation, not configuration --
	// the wall's button renders for every session, and the server
	// holds no role gate on the face.
	//
	// The dialog owns the drafts' lifecycle (seeded on open: an
	// empty title, an empty audience) and the title's local bound
	// (the contract's 1-64 runes, checked here so an invalid draft
	// never leaves the page). The audience is the add-style picker
	// (AudienceField), a portaled floating layer -- the dialog stays
	// a fixed-height surface. THE PAGE OWNS THE WRITE: `onsubmit`
	// receives the drafts and runs the POST.
	//
	// THE HONEST REPORT: the creation's `skipped` list (the
	// requested ids that did not become audience -- the owner's own
	// id, an unknown or disabled account) renders in the dialog
	// before it closes, the write's own answer; the Done affordance
	// closes. An all-landed creation closes directly.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import AudienceField, { type AudienceMember } from '$lib/albums/AudienceField.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { sessionUserId } from '$lib/auth/session.svelte';
	import type { AlbumAudienceSkip } from '$lib/api/albums';
	import Plus from '@lucide/svelte/icons/plus';
	import Check from '@lucide/svelte/icons/check';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** The honest skipped report of the last landed create (null when none). */
		skipped?: readonly AlbumAudienceSkip[] | null;
		/** Run the write with the drafts. */
		onsubmit: (title: string, audience: readonly AudienceMember[]) => void;
		/** Acknowledge the honest report: close the dialog. */
		ondone?: () => void;
	};

	let { open = $bindable(false), busy = false, errorKey = '', skipped = null, onsubmit, ondone }: Props = $props();

	let titleDraft = $state('');
	let audience = $state<AudienceMember[]>([]);
	/** Whether a submit attempt already ran for the current drafts. */
	let checked = $state(false);

	// Seed the drafts on every open; a close leaves the stale drafts
	// behind but the next open overwrites them.
	$effect(() => {
		if (open) {
			titleDraft = '';
			audience = [];
			checked = false;
		}
	});

	/** The contract's bound, rune-counted: the trimmed title spans 1-64 characters. */
	function validTitle(): boolean {
		const runes = [...titleDraft.trim()].length;
		return runes >= 1 && runes <= 64;
	}

	const titleViolated = $derived(checked && !validTitle());

	// The honest report's face: the landed create's own answer (the
	// skipped audience ids) holds the dialog until acknowledged.
	const reportFace = $derived(skipped !== null && skipped.length > 0);

	function submit(): void {
		checked = true;
		if (!validTitle()) {
			return;
		}
		onsubmit(titleDraft.trim(), audience);
	}
</script>

<Modal bind:open title={$_('memories_create_title')}>
	{#if reportFace}
		<!-- The honest report of the landed create: the requested ids
		     that did not become audience, the write's own answer. -->
		<p class="create-memory__report m-0 font-body font-semibold text-[15px] leading-[1.5] text-text" role="status">
			{$_('memories_create_report', { values: { count: skipped?.length ?? 0 } })}
		</p>
	{:else}
		<div class="create-memory flex flex-col gap-3.5">
			<!-- The title above, the audience after: the memory's whole
			     creation shape -- nothing else exists to ask. -->
			<Input
				label={$_('albums_title_field')}
				id="create-memory-title"
				name="create-memory-title"
				placeholder={$_('memories_title_placeholder')}
				bind:value={titleDraft}
				required
				maxlength={64}
				autocomplete="off"
				error={titleViolated ? $_('albums_title_invalid') : ''}
			/>
			<AudienceField bind:members={audience} selfId={sessionUserId() ?? ''} {busy} />
			{#if errorKey !== ''}
				<p
					class="create-memory__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
					role="alert"
				>
					{resolveErrorText($_, errorKey)}
				</p>
			{/if}
		</div>
	{/if}
	{#snippet footer()}
		{#if reportFace}
			<Button onclick={ondone}>
				<Check size={20} aria-hidden="true" />
				{$_('action_done')}
			</Button>
		{:else}
			<Button variant="ghost" disabled={busy} onclick={() => (open = false)}>
				{$_('common_cancel')}
			</Button>
			<Button disabled={busy} onclick={submit}>
				<Plus size={20} aria-hidden="true" />
				{$_('memories_create_submit')}
			</Button>
		{/if}
	{/snippet}
</Modal>
