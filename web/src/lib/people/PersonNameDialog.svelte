<script lang="ts">
	// The naming dialog of the people surface: one Modal over one
	// Input and the submit pair, the composition the unnamed region's
	// naming main path and the cluster detail's naming CTA / rename
	// share. The page owns the write: `onsubmit` receives the trimmed
	// name, runs the PUT label, and adopts the response; the dialog
	// owns the draft's lifecycle (seeded on open) and the local bound
	// check (the server trims and counts runes; the client mirrors
	// both so an invalid draft never leaves the page -- the inline
	// message says why instead, and clears the moment the draft
	// becomes valid).
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import X from '@lucide/svelte/icons/x';
	import Check from '@lucide/svelte/icons/check';
	import { resolveErrorText } from '$lib/i18n/errors';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The i18n key of the dialog title (naming vs renaming). */
		titleKey: string;
		/** The i18n key of the submit label. */
		submitKey: string;
		/** The draft's seed (the rename path passes the current name). */
		initialName?: string;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Run the write with the trimmed draft. */
		onsubmit: (name: string) => void;
	};

	let {
		open = $bindable(false),
		titleKey,
		submitKey,
		initialName = '',
		busy = false,
		errorKey = '',
		onsubmit
	}: Props = $props();

	let draft = $state('');
	/** Whether a submit attempt already ran for the current draft. */
	let checked = $state(false);

	// Seed the draft on every open (the rename path reseeds the
	// current name); a close leaves the stale draft behind but the
	// next open overwrites it.
	$effect(() => {
		if (open) {
			draft = initialName;
			checked = false;
		}
	});

	/** The contract's bound, rune-counted: the trimmed name spans 1-64 characters. */
	function validDraft(): boolean {
		const runes = [...draft.trim()].length;
		return runes >= 1 && runes <= 64;
	}

	const boundViolated = $derived(checked && !validDraft());

	function submit(): void {
		checked = true;
		if (!validDraft()) {
			return;
		}
		onsubmit(draft.trim());
	}
</script>

<Modal bind:open title={$_(titleKey)}>
	<div class="person-name-dialog flex flex-col gap-2">
		<Input
			label={$_('people_name_field_label')}
			name="name"
			bind:value={draft}
			maxlength={128}
			autocomplete="off"
		/>
		<!-- The local bound check and the write's error line share the
		     dialog body; both stay locale-tracked through the reactive
		     store reads of $_. -->
		{#if boundViolated}
			<p class="person-name-dialog__bound m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text">{$_('people_name_invalid')}</p>
		{/if}
		{#if errorKey !== ''}
			<p class="person-name-dialog__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, errorKey)}</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button disabled={busy} onclick={submit}>
			<Check size={20} aria-hidden="true" />
			{$_(submitKey)}
		</Button>
	{/snippet}
</Modal>
