<script lang="ts">
	// The avatar picker of the person detail (web-ia §7.17): the
	// person's FACES are the candidates -- each face's crop, square,
	// no file name and no source photo name (a name helps nobody
	// recognize a face and only turns the grid into a table). The
	// current avatar carries the "Avatar" badge; the footer's two
	// buttons are the only chrome.
	//
	// THIS IS NOT FACE GOVERNANCE: the picker chooses which face
	// renders as the person's avatar, nothing else. Reassigning and
	// detaching faces stays in the viewer's edit-people panel (the
	// photo's own context, where the whole photo is visible) -- the
	// picker never grows those actions.
	//
	// The page owns the write: `onsubmit` receives the picked face id
	// and runs the PUT; the dialog owns the draft (the pick, seeded
	// from the current avatar on every open) and closes itself on the
	// caller's success.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import X from '@lucide/svelte/icons/x';
	import Check from '@lucide/svelte/icons/check';
	import { resolveErrorText } from '$lib/i18n/errors';
	import type { FaceDetail } from '$lib/api/people';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The person's faces on caller-visible assets (the candidates). */
		faces: readonly FaceDetail[];
		/** The currently stored avatar face id (null when none was picked). */
		currentFaceId: string | null;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Run the write with the picked face id. */
		onsubmit: (faceId: string) => void;
	};

	let { open = $bindable(false), faces, currentFaceId, busy = false, errorKey = '', onsubmit }: Props = $props();

	// The draft pick: seeded from the stored avatar on every open, so
	// a reopened dialog never carries a stale choice.
	let picked = $state<string | null>(null);

	$effect(() => {
		if (open) {
			picked = currentFaceId;
		}
	});

	// The visibility gate: the page passes the STORED pick verbatim,
	// and a stored pick the caller cannot see never renders a row --
	// so the dialog opens with nothing pressed and the submit stays
	// armed only for a pick the caller could actually write (the
	// server would refuse an invisible face; the button never offers
	// that write).
	const pickIsVisible = $derived(faces.some((face) => face.id === picked));

	function submit(): void {
		if (picked === null || !pickIsVisible) {
			return;
		}
		onsubmit(picked);
	}
</script>

<Modal bind:open title={$_('person_avatar_action')}>
	<div class="person-avatar-dialog flex flex-col gap-3">
		<p class="m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
			{$_('person_avatar_dialog_body')}
		</p>
		<div class="person-avatar-dialog__grid grid grid-cols-[repeat(auto-fill,minmax(min(96px,100%),1fr))] gap-2">
			{#each faces as face (face.id)}
				<!-- The pick is one radio group of face crops: the
				     pressed state is the pick, the badge marks the
				     STORED avatar (so a fresh pick reads as a change
				     against the current one). -->
				<button
					class="avatar-option group relative block w-full aspect-square overflow-hidden rounded-control bg-accent-soft border cursor-pointer transition-[border-color] duration-(--motion-fast) ease-standard {picked ===
					face.id
						? 'border-accent'
						: 'border-border hover:border-accent'}"
					type="button"
					aria-pressed={picked === face.id}
					aria-label={$_('person_avatar_option')}
					onclick={() => (picked = face.id)}
				>
					<img
						class="absolute inset-0 w-full h-full object-cover"
						src={face.thumbnail}
						alt=""
						loading="lazy"
						decoding="async"
						draggable="false"
					/>
					{#if face.id === currentFaceId}
						<span
							class="avatar-option__badge absolute top-1.5 left-1.5 px-1.5 py-0.5 font-body font-semibold text-[11px] leading-[1.4] text-text bg-card border border-border rounded-control"
							>{$_('person_avatar_badge')}</span
						>
					{/if}
				</button>
			{/each}
		</div>
		{#if errorKey !== ''}
			<p
				class="person-avatar-dialog__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
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
		<Button disabled={busy || picked === null || !pickIsVisible} onclick={submit}>
			<Check size={20} aria-hidden="true" />
			{$_('person_avatar_set')}
		</Button>
	{/snippet}
</Modal>
