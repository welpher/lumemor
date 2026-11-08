<script lang="ts" module>
	/** One audience member the chips render: the id + the display email. */
	export type AudienceMember = { userId: string; email: string };
</script>

<script lang="ts">
	// The audience field of the albums-as-sources dialogs (web IA
	// §7.6, the specimen's add-style pin): "who can see" as ONE
	// property of the album -- the granted accounts as chips (the
	// avatar + the name + the remove mark), the dashed "Add"
	// affordance, and the candidate picker that opens INLINE (the
	// same zero-occlusion construction as the path completion: the
	// dialog grows, its footer moves down, no floating layer). The
	// candidate rows are STATE SELECTORS, not action buttons (the
	// icon-first rule's second face): the row's mousedown adds, the
	// chip's remove mark is the only button among them.
	//
	// THE CANDIDATES: the instance's minimal member list (the same
	// users listing the share dialog consumed), minus the current
	// draft's members (they are chips already) and minus the CALLER
	// (the owner sees the album by construction; the server's own
	// skip report is the backstop when the session has no id). The
	// listing is fetched on every open -- the per-open discipline of
	// the old share dialog -- and its honest failure renders inside
	// the picker, with the toggle affordance as the retry path.
	//
	// THE KEYBOARD (the specimen's keys footer): the listbox takes
	// the focus on open; ↑/↓ move the highlight, Enter adds the
	// highlighted candidate (the list STAYS open -- the specimen's
	// state two shows a chip landed while the picker offers the
	// next; Esc is the close), Esc closes the picker alone and
	// stops there -- the modal behind it never reacts to a stroke
	// the picker owned.
	import { _ } from 'svelte-i18n';
	import { tick } from 'svelte';
	import { listUsers, type UserSummary } from '$lib/api/albums';
	import UserRoundPlus from '@lucide/svelte/icons/user-round-plus';
	import X from '@lucide/svelte/icons/x';

	type Props = {
		/** The draft audience (bindable): the chips, and the write's full set. */
		members: AudienceMember[];
		/** The caller's own account id ('' when the session holds none). */
		selfId?: string;
		/** Writes run one at a time: the affordances disable while in flight. */
		busy?: boolean;
	};

	let { members = $bindable([]), selfId = '', busy = false }: Props = $props();

	/** The candidate picker's face. */
	let open = $state(false);
	/** The instance's member list while it is loaded ('' state below). */
	let users = $state<UserSummary[]>([]);
	/** Whether the member-list fetch runs; its honest failure key. */
	let loading = $state(false);
	let failed = $state(false);
	/** The highlighted candidate's index (-1 = none). */
	let activeIndex = $state(-1);

	let listbox: HTMLElement | undefined = $state();
	let addButton: HTMLButtonElement | undefined = $state();

	// The candidates: every account minus the draft's members (they
	// are chips) minus the caller (the owner never needs a row).
	const candidates = $derived(
		users.filter(
			(user) =>
				user.id !== selfId && !members.some((member) => member.userId === user.id)
		)
	);

	// Keep the highlight valid as the set re-derives (a just-added
	// candidate leaves the list; the highlight steps forward on its
	// own row).
	$effect(() => {
		void candidates;
		if (activeIndex >= candidates.length) {
			activeIndex = candidates.length - 1;
		}
		if (activeIndex < 0 && candidates.length > 0) {
			activeIndex = 0;
		}
	});

	// The members the dialog passed stay referenced; nothing to
	// clean up beyond the picker's own state on close.
	$effect(() => {
		if (!open) {
			activeIndex = -1;
		}
	});

	/** Toggle the candidate picker; every open fetches the member list fresh. */
	function toggle(): void {
		if (busy) {
			return;
		}
		if (open) {
			open = false;
			addButton?.focus();
			return;
		}
		open = true;
		users = [];
		failed = false;
		loading = true;
		void (async () => {
			const outcome = await listUsers();
			if (!open) {
				return;
			}
			loading = false;
			if (outcome.ok) {
				users = outcome.users;
				activeIndex = 0;
			} else {
				failed = true;
			}
		})();
		// The listbox renders with this state change; focus it once
		// the DOM carries it (the keyboard driver of the picker).
		void tick().then(() => listbox?.focus());
	}

	/** Add one candidate to the draft audience (the picker stays open). */
	function add(index: number): void {
		const candidate = candidates[index];
		if (candidate === undefined) {
			return;
		}
		members = [...members, { userId: candidate.id, email: candidate.email }];
	}

	/** Remove one member (the chip's mark). */
	function remove(userId: string): void {
		members = members.filter((member) => member.userId !== userId);
	}

	/** The keyboard the picker owns; strokes it handles never reach the modal. */
	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
			event.preventDefault();
			event.stopPropagation();
			if (candidates.length === 0) {
				return;
			}
			const direction = event.key === 'ArrowDown' ? 1 : -1;
			activeIndex = Math.min(candidates.length - 1, Math.max(0, activeIndex + direction));
			return;
		}
		if (event.key === 'Enter') {
			if (activeIndex >= 0) {
				event.preventDefault();
				event.stopPropagation();
				add(activeIndex);
			}
			return;
		}
		if (event.key === 'Escape') {
			// The picker's Esc closes the picker alone -- the dialog
			// behind it stays (its own Escape owns the next stroke).
			event.stopPropagation();
			open = false;
			addButton?.focus();
		}
	}

	/** One account's avatar initial: the email local part's first rune. */
	function initialOf(email: string): string {
		const local = email.split('@')[0] ?? '';
		return [...local][0]?.toUpperCase() ?? '?';
	}
</script>

<section class="audience-field border-t border-border pt-3" aria-label={$_('albums_audience_label')}>
	<p class="audience-field__label m-0 mb-2 font-body font-semibold text-[14px] leading-[1.4] text-text">{$_('albums_audience_label')}</p>
	<div class="audience-field__chips flex flex-wrap items-center gap-2">
		{#if members.length === 0}
			<!-- The honest empty audience: only the caller themselves. -->
			<p class="audience-field__only-me m-0 font-body font-normal text-[13px] leading-[1.4] text-text-faint">{$_('albums_audience_only_me')}</p>
		{:else}
			{#each members as member (member.userId)}
				<span class="audience-field__chip inline-flex items-center gap-1.5 py-1 ps-1 pe-1.5 bg-accent-soft text-accent-soft-text rounded-full font-body font-bold text-[13px] leading-[1.4]">
					<span class="audience-field__chip-avatar w-5 h-5 rounded-full bg-accent text-white flex items-center justify-center text-[10px] leading-none font-bold shrink-0" aria-hidden="true">{initialOf(member.email)}</span>
					<span class="audience-field__chip-name truncate">{member.email}</span>
					<!-- The chip's remove mark: the one action button among
					     the state-selector shapes, icon-only and named. -->
					<button
						class="audience-field__chip-remove inline-flex items-center justify-center w-[18px] h-[18px] rounded-full bg-transparent text-accent-soft-text border-none cursor-pointer hover:bg-[color-mix(in_srgb,var(--color-night)_12%,transparent)] disabled:opacity-55 disabled:cursor-not-allowed"
						type="button"
						aria-label={$_('albums_audience_remove', { values: { name: member.email } })}
						disabled={busy}
						onclick={() => remove(member.userId)}
					>
						<X size={10} stroke-width={3} aria-hidden="true" />
					</button>
				</span>
			{/each}
		{/if}
		<!-- The dashed add affordance (icon + text), the picker's toggle. -->
		<button
			class="audience-field__add inline-flex items-center gap-1.5 px-2.5 py-1 border border-dashed border-border-strong bg-transparent text-text-soft rounded-full font-body font-bold text-[13px] leading-[1.4] cursor-pointer transition-[border-color,color] duration-(--motion-fast) ease-standard hover:border-accent hover:text-accent disabled:opacity-55 disabled:cursor-not-allowed"
			type="button"
			bind:this={addButton}
			aria-expanded={open}
			aria-controls={open ? 'audience-field-picker' : undefined}
			disabled={busy}
			onclick={toggle}
		>
			<UserRoundPlus size={14} class="text-accent" aria-hidden="true" />
			{$_('albums_audience_add')}
		</button>
	</div>
	{#if open}
		<!-- THE INLINE EXPANSION (the same anti-occlusion construction
		     as the path completion): the picker is a flow child of the
		     audience section. -->
		<div
			class="audience-field__picker mt-2 bg-page border border-border rounded-control overflow-hidden"
			bind:this={listbox}
			id="audience-field-picker"
			role="listbox"
			aria-label={$_('albums_audience_candidates_label')}
			tabindex="0"
			onkeydown={handleKeydown}
		>
			{#if loading}
				<p class="audience-field__loading m-0 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in" role="status">{$_('albums_audience_loading')}</p>
			{:else if failed}
				<p class="audience-field__failed m-0 px-3 py-2 font-body font-semibold text-[13px] leading-[1.4] text-danger-text" role="alert">{$_('albums_audience_failed')}</p>
			{:else if candidates.length === 0}
				<p class="audience-field__no-candidates m-0 px-3 py-2 font-body font-normal text-[13px] leading-[1.5] text-text-soft">{$_('albums_audience_no_candidates')}</p>
			{:else}
				{#each candidates as candidate, index (candidate.id)}
					<!-- The candidate row: a state selector, not an action
					     button -- mousedown adds (before any blur could
					     close the picker); the disabled account renders
					     its honest note and stays pickable (the server's
					     skip report is the authority). -->
					<div
						class="audience-field__row flex items-center gap-2.5 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] cursor-pointer {index === activeIndex ? 'bg-accent-soft text-accent-soft-text' : 'text-text'}"
						id="audience-field-option-{index}"
						role="option"
						tabindex="-1"
						aria-selected={index === activeIndex}
						onmousedown={(event) => {
							event.preventDefault();
							add(index);
						}}
					>
						<span class="audience-field__row-avatar w-7 h-7 rounded-full bg-accent-soft text-accent-soft-text flex items-center justify-center text-[12px] leading-none font-bold shrink-0" aria-hidden="true">{initialOf(candidate.email)}</span>
						<span class="audience-field__row-name truncate">{candidate.email}</span>
						{#if candidate.isDisabled}
							<span class="audience-field__row-note font-body font-normal text-[12px] leading-[1.4] {index === activeIndex ? 'text-accent-soft-text' : 'text-text-faint'}">{$_('albums_audience_disabled_note')}</span>
						{/if}
					</div>
				{/each}
				<div class="audience-field__keys flex gap-3 px-3 py-1.5 border-t border-border font-body font-normal text-[11px] leading-[1.4] text-text-faint">
					<span>{$_('albums_keys_select')}</span>
					<span>{$_('albums_keys_add')}</span>
					<span>{$_('albums_keys_close')}</span>
				</div>
			{/if}
		</div>
	{/if}
</section>
