<script lang="ts">
	// The properties dialog of the albums surface: one Modal over the
	// title field and, in the properties mode, the description
	// field -- the composition the wall's creation flow (title only,
	// the pinned dialog of the web IA) and the detail's rename /
	// properties action share. The page owns the write: `onsubmit`
	// receives the trimmed pair, runs the POST or the PATCH, and
	// adopts the response; the dialog owns the draft's lifecycle
	// (seeded on open) and the local title bound (the server trims
	// and counts runes; the client mirrors both so an invalid draft
	// never leaves the page -- the inline message says why instead,
	// and clears the moment the draft becomes valid).
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The i18n key of the dialog title (creation vs properties). */
		titleKey: string;
		/** The i18n key of the submit label. */
		submitKey: string;
		/** The drafts' seeds (the properties path passes the current values). */
		initialTitle?: string;
		initialDescription?: string;
		/** Whether the description field renders (the properties mode). */
		showDescription?: boolean;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Run the write with the trimmed drafts. */
		onsubmit: (title: string, description: string) => void;
	};

	let {
		open = $bindable(false),
		titleKey,
		submitKey,
		initialTitle = '',
		initialDescription = '',
		showDescription = false,
		busy = false,
		errorKey = '',
		onsubmit
	}: Props = $props();

	let titleDraft = $state('');
	let descriptionDraft = $state('');
	/** Whether a submit attempt already ran for the current draft. */
	let checked = $state(false);

	// Seed the drafts on every open (the properties path reseeds the
	// current values); a close leaves the stale drafts behind but the
	// next open overwrites them.
	$effect(() => {
		if (open) {
			titleDraft = initialTitle;
			descriptionDraft = initialDescription;
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
		onsubmit(titleDraft.trim(), descriptionDraft.trim());
	}
</script>

<Modal bind:open title={$_(titleKey)}>
	<div class="album-form flex flex-col gap-2">
		<Input
			label={$_('albums_title_field')}
			name="album-title"
			bind:value={titleDraft}
			maxlength={128}
			autocomplete="off"
		/>
		{#if showDescription}
			<label class="album-form__description-label font-body font-semibold text-[14px] leading-[1.4] text-text" for="album-description">{$_('albums_description_field')}</label>
			<textarea
				class="album-form__description w-full px-3 py-[9px] font-body font-normal text-[14px] leading-[1.5] text-text bg-card border border-border-strong rounded-control resize-y focus:border-accent"
				id="album-description"
				bind:value={descriptionDraft}
				aria-label={$_('albums_description_field')}
				rows="3"
			></textarea>
		{/if}
		<!-- The local bound check and the write's error line share the
		     dialog body; both stay locale-tracked through the reactive
		     store reads of $_. -->
		{#if boundViolated}
			<p class="album-form__bound m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text">{$_('albums_title_invalid')}</p>
		{/if}
		{#if errorKey !== ''}
			<p class="album-form__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, errorKey)}</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>{$_('common_cancel')}</Button>
		<Button disabled={busy} onclick={submit}>{$_(submitKey)}</Button>
	{/snippet}
</Modal>
