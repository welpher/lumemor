<script lang="ts">
	// The detach confirm of the edit-people panel: one Modal naming
	// the person whose face is about to be released. The body is the
	// honest cost statement -- the face returns to the not-yet-grouped
	// pool, the photo itself is untouched, and a later assign can
	// re-home it. The confirm is a real confirm (the membership
	// change touches another surface's counts) but the lightest one:
	// no typed name, no checkbox -- the person's name in the body is
	// the whole ceremony. The page owns the write; the dialog owns
	// nothing but the confirm's lifecycle.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import X from '@lucide/svelte/icons/x';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { resolveErrorText } from '$lib/i18n/errors';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The person's display name (the body names them). */
		name: string;
		/** The write is in flight: the submit disables. */
		busy?: boolean;
		/** The error-code i18n key of the failed write ('' when none). */
		errorKey?: string;
		/** Notified of every close path (Escape, backdrop, the close mark). */
		onclose?: () => void;
		/** Run the detach write. */
		onsubmit: () => void;
	};

	let { open = $bindable(false), name, busy = false, errorKey = '', onclose, onsubmit }: Props = $props();

	/** Close through the same discipline as the modal's own paths. */
	function close(): void {
		open = false;
		onclose?.();
	}
</script>

<Modal bind:open title={$_('people_detach_dialog_title')} {onclose}>
	<div class="detach-dialog flex flex-col">
		<p class="detach-dialog__body m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
			{$_('people_detach_dialog_body', { values: { name } })}
		</p>
		{#if errorKey !== ''}
			<p
				class="detach-dialog__failure m-0 mt-3 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{resolveErrorText($_, errorKey)}
			</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={close}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="danger" disabled={busy} onclick={onsubmit}>
			<Trash2 size={20} aria-hidden="true" />
			{$_('people_detach_submit')}
		</Button>
	{/snippet}
</Modal>
