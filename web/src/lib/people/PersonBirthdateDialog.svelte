<script lang="ts">
	// The birth date dialog of the person detail (web-ia §7.17): one
	// date field, the honest note about what the value is for, and
	// the save / clear pair. The date is the person ENTITY's
	// attribute -- an unnamed cluster carries one too -- and one
	// value serves the whole family.
	//
	// THE FORMAT DISCIPLINE: the wire and the `<input type="date">`
	// both speak the machine's `YYYY-MM-DD`; the DISPLAY never
	// hand-assembles a date -- the page renders the profile line
	// through `Intl.DateTimeFormat` (numeric year/month/day) with the
	// machine value in `<time datetime>`. This dialog only ever shows
	// the input's own locale-driven presentation.
	//
	// The page owns the write: `onsubmit` receives the picked date or
	// null (the clear) and runs the PUT; the dialog owns the draft
	// (seeded on every open) and closes itself on the caller's
	// success.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import X from '@lucide/svelte/icons/x';
	import Check from '@lucide/svelte/icons/check';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { resolveErrorText } from '$lib/i18n/errors';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The stored date in the wire's `YYYY-MM-DD` form (null when unset). */
		initialDate: string | null;
		/** Writes run one at a time: the submits disable while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Run the write with the picked date, or null for the clear. */
		onsubmit: (birthDate: string | null) => void;
	};

	let { open = $bindable(false), initialDate, busy = false, errorKey = '', onsubmit }: Props = $props();

	let draft = $state('');

	// Seed the draft on every open (the clear leaves the field empty
	// for the next open, never a stale date).
	$effect(() => {
		if (open) {
			draft = initialDate ?? '';
		}
	});

	const validDraft = $derived(/^\d{4}-\d{2}-\d{2}$/.test(draft));
</script>

<Modal bind:open title={$_('person_birthdate_dialog_title')}>
	<div class="person-birthdate-dialog flex flex-col gap-1.5">
		<label class="font-body font-semibold text-[14px] leading-[1.4] text-text" for="person-birth-date"
			>{$_('person_birthdate_dialog_title')}</label
		>
		<input
			class="w-full px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control focus:border-accent"
			id="person-birth-date"
			type="date"
			bind:value={draft}
		/>
		<p class="m-0 font-body font-normal text-[12px] leading-[1.4] text-text-faint">
			{$_('person_birthdate_dialog_note')}
		</p>
		{#if errorKey !== ''}
			<p
				class="person-birthdate-dialog__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{resolveErrorText($_, errorKey)}
			</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<!-- The clear is an act of its own (the wire's explicit null),
		     offered exactly when a value is stored. -->
		{#if initialDate !== null}
			<Button variant="ghost" disabled={busy} onclick={() => onsubmit(null)}>
				<Trash2 size={20} aria-hidden="true" />
				{$_('person_birthdate_clear')}
			</Button>
		{/if}
		<Button disabled={busy || !validDraft} onclick={() => onsubmit(draft)}>
			<Check size={20} aria-hidden="true" />
			{$_('person_birthdate_save')}
		</Button>
	{/snippet}
</Modal>
