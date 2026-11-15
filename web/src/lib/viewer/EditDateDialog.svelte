<script lang="ts">
	// The "edit date" dialog of the viewer's info panel: the capture
	// time and its zone edited as one value -- a `datetime-local`
	// field plus the zone dropdown, and one PATCH carrying the pair.
	// The offset in the composed value IS the displayed zone: the
	// same value decides where the photo sits on the timeline (the
	// instant) and which suffix renders beside the time (the zone).
	//
	// The zone list: hourly offsets across the world's range, with
	// the asset's current zone pinned into the list when the stored
	// offset is off-hour (an EXIF half-hour zone must never silently
	// become an hour). The dialog's own default is the asset's
	// current zone, and the local field defaults to the stored time
	// rendered in it -- the edit starts from the truth the panel
	// displayed.
	//
	// The page owns the write: `onsubmit` receives the composed pair;
	// the dialog owns the draft (seeded on every open) and closes
	// itself on the caller's success.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { zoneLabel, localInputValue } from './zone';
	import Check from '@lucide/svelte/icons/check';

	type Props = {
		/** Open state; bind it from the caller, the dialog closes itself. */
		open?: boolean;
		/** The stored capture time (RFC 3339), rendered in the zone below. */
		takenAt: string;
		/** The stored display zone (EXIF offset notation, null when none resolved). */
		takenAtZone: string | null;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Run the write with the composed field pair. */
		onsubmit: (datetimeLocal: string, zone: string) => void;
	};

	let { open = $bindable(false), takenAt, takenAtZone, busy = false, errorKey = '', onsubmit }: Props = $props();

	/** The offset suffixes the zone dropdown offers, hour by hour. */
	const ZONES: string[] = [];
	for (let hours = -12; hours <= 14; hours += 1) {
		const sign = hours < 0 ? '-' : '+';
		ZONES.push(`${sign}${String(Math.abs(hours)).padStart(2, '0')}:00`);
	}

	/** The stored time rendered into the zone's wall clock, `datetime-local` form. */
	function seedLocal(): string {
		return localInputValue(takenAt, takenAtZone);
	}

	let draftLocal = $state('');
	let draftZone = $state('+00:00');

	// Each open seeds the fields from the stored truth (the asset's
	// current time and zone), so a retry and a fresh edit both start
	// from what the panel displayed.
	$effect(() => {
		if (!open) {
			return;
		}
		draftZone = takenAtZone ?? '+00:00';
		draftLocal = seedLocal();
	});

	const validDraft = $derived(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/.test(draftLocal));
</script>

<Modal bind:open title={$_('viewer_edit_date_title')} maxWidth="420px">
	<div class="viewer-date-dialog flex flex-col gap-1.5">
		<p class="m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
			{$_('viewer_edit_date_note')}
		</p>
		<label class="font-body font-semibold text-[14px] leading-[1.4] text-text" for="viewer-edit-date-value"
			>{$_('viewer_edit_date_field')}</label
		>
		<input
			class="w-full px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-page border border-border-strong rounded-control"
			id="viewer-edit-date-value"
			type="datetime-local"
			bind:value={draftLocal}
		/>
		<label class="mt-2 font-body font-semibold text-[14px] leading-[1.4] text-text" for="viewer-edit-date-zone"
			>{$_('viewer_edit_date_zone')}</label
		>
		<select
			class="w-full px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-page border border-border-strong rounded-control"
			id="viewer-edit-date-zone"
			bind:value={draftZone}
		>
			{#each ZONES as zone (zone)}
				<option value={zone}>{zoneLabel(zone)}</option>
			{/each}
			{#if takenAtZone !== null && !ZONES.includes(takenAtZone)}
				<!-- The stored zone is off-hour (an EXIF half-hour offset):
				     it rides the list so the truth stays selectable. -->
				<option value={takenAtZone}>{zoneLabel(takenAtZone)}</option>
			{/if}
		</select>
		{#if errorKey !== ''}
			<p
				class="viewer-date-dialog__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{resolveErrorText($_, errorKey)}
			</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>{$_('common_cancel')}</Button>
		<Button variant="primary" disabled={busy || !validDraft} onclick={() => onsubmit(draftLocal, draftZone)}>
			<Check size={20} aria-hidden="true" />
			{$_('action_save')}
		</Button>
	{/snippet}
</Modal>
