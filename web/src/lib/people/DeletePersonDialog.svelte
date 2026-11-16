<script lang="ts">
	// The delete-person confirm: the danger zone's one dialog. The
	// body is the honest cost statement -- the person AND every face's
	// detection record go away (the photos themselves untouched), the
	// unnamed pool gains nothing, and the action cannot be undone.
	// The count in the sentence is the caller's honest view: the
	// faces on assets they can see (the detail's own faces array).
	// The confirm is a real confirm (the write reshapes the lists and
	// the photos' chips) with the name and the count as its whole
	// ceremony. The page owns the write and the navigation; the
	// dialog owns nothing but the confirm's lifecycle.
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
		/** The caller's visible face count (the body counts them). */
		faceCount: number;
		/** The write is in flight: the submit disables. */
		busy?: boolean;
		/** The error-code i18n key of the failed write ('' when none). */
		errorKey?: string;
		/** Notified of every close path (Escape, backdrop, the close mark). */
		onclose?: () => void;
		/** Run the delete write. */
		onsubmit: () => void;
	};

	let { open = $bindable(false), name, faceCount, busy = false, errorKey = '', onclose, onsubmit }: Props = $props();

	/** Close through the same discipline as the modal's own paths. */
	function close(): void {
		open = false;
		onclose?.();
	}
</script>

<Modal bind:open title={$_('person_delete_title')} {onclose}>
	<div class="delete-person-dialog flex flex-col">
		<p class="delete-person-dialog__body m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
			{$_('person_delete_body', { values: { name, count: faceCount } })}
		</p>
		{#if errorKey !== ''}
			<p
				class="delete-person-dialog__failure m-0 mt-3 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
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
			<Trash2 size={18} aria-hidden="true" />
			{$_('person_delete')}
		</Button>
	{/snippet}
</Modal>
