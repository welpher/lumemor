<script lang="ts">
	// The edit-properties dialog of the albums surface, reshaped by
	// E4 (web IA §7.6, the specimen's edit state): the TITLE, the
	// SOURCE row (the directory album's root path, read-only with
	// the "source" tag -- the path is pinned at creation, the
	// property patch carries no field for it), and the AUDIENCE
	// (the current grants as chips, the add-style picker for the
	// candidates) -- the share dialog's whole management surface
	// folded in here, "who can see" an album property like any
	// other. Both pickers expand INLINE (the anti-occlusion pin).
	//
	// The page owns the writes: `onsubmit` receives the trimmed
	// title and the FULL audience member set (the PUT's full-set
	// semantics -- the page diffs and writes what changed); the
	// dialog owns the drafts' lifecycle (seeded on open from the
	// detail's own fields) and the local title bound (the server
	// trims and counts runes; the client mirrors both so an invalid
	// draft never leaves the page -- the inline message says why
	// instead, and clears the moment the draft becomes valid).
	// The write's failure renders as the dialog's general line.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import AudienceField, { type AudienceMember } from './AudienceField.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { sessionUserId } from '$lib/auth/session.svelte';
	import Folder from '@lucide/svelte/icons/folder';
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
		 * renders exactly when set (virtual and sync albums hold none).
		 */
		rootPath?: string;
		/** The current grants -- the detail's `shares` array, the chips' seed. */
		shares: readonly { userId: string; email: string }[];
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Run the writes with the trimmed title and the full audience set. */
		onsubmit: (title: string, audience: readonly AudienceMember[]) => void;
	};

	let {
		open = $bindable(false),
		titleKey,
		submitKey,
		initialTitle = '',
		rootPath = '',
		shares,
		busy = false,
		errorKey = '',
		onsubmit
	}: Props = $props();

	let titleDraft = $state('');
	let audience = $state<AudienceMember[]>([]);
	/** Whether a submit attempt already ran for the current draft. */
	let checked = $state(false);

	// Seed the drafts on every open (the current title and the
	// current grants); a close leaves the stale drafts behind but
	// the next open overwrites them.
	$effect(() => {
		if (open) {
			titleDraft = initialTitle;
			audience = shares.map((share) => ({ userId: share.userId, email: share.email }));
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
		onsubmit(titleDraft.trim(), audience);
	}
</script>

<Modal bind:open title={$_(titleKey)}>
	<div class="album-form flex flex-col gap-3.5">
		<Input
			label={$_('albums_title_field')}
			id="album-title"
			name="album-title"
			bind:value={titleDraft}
			maxlength={128}
			autocomplete="off"
			error={boundViolated ? $_('albums_title_invalid') : ''}
		/>
		{#if rootPath !== ''}
			<!-- The source row: read-only with the honest tag -- the
			     album's source is pinned at creation, the edit carries
			     no field for it (the specimen's 「来源 · 不可改」). -->
			<div class="album-form__source flex items-center gap-2 px-3 py-2.5 bg-page border border-border rounded-control">
				<Folder size={18} class="text-accent shrink-0" aria-hidden="true" />
				<span class="album-form__source-path font-body font-normal text-[13px] leading-[1.4] text-text-soft overflow-hidden text-ellipsis whitespace-nowrap">{rootPath}</span>
				<span class="album-form__source-tag shrink-0 font-body font-bold text-[11px] leading-[1.4] text-accent-soft-text bg-accent-soft px-2 py-0.5 rounded-full">{$_('albums_source_tag')}</span>
			</div>
		{/if}
		<!-- The audience: the folded-in share management (the current
		     grants as the chips, the picker for the candidates). -->
		<AudienceField bind:members={audience} selfId={sessionUserId() ?? ''} {busy} />
		{#if errorKey !== ''}
			<p class="album-form__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, errorKey)}</p>
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
