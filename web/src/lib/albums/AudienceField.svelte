<script lang="ts" module>
	/** One audience member the chips render: the id + the display email. */
	export type AudienceMember = { userId: string; email: string };
</script>

<script lang="ts">
	// The audience field of the albums-as-sources dialogs: "who can see" as ONE
	// property of the album -- the granted accounts as chips (the
	// avatar + the name + the remove mark), the dashed "Add"
	// affordance, and the candidate picker as a FLOATING LAYER (the
	// floating-layer revision: the 2026-11-08
	// inline-expansion design is retired -- it traded an offset
	// shift for height churn, the dialog growing and shrinking with
	// every open; the layer portals to document.body, anchored under
	// the chips row through the shared `anchored` action, so the
	// modal's overflow never clips it and the dialog NEVER changes
	// height for it). The candidate rows are STATE SELECTORS, not
	// action buttons (the icon-first rule's second face): the row's
	// mousedown adds, the chip's remove mark is the only button
	// among them.
	//
	// THE CANDIDATES: the instance's minimal member list (the same
	// users listing the share dialog consumed), minus the current
	// draft's members (they are chips already) and minus the CALLER
	// (the owner sees the album by construction; the server's own
	// skip report is the backstop when the session has no id). The
	// listing is fetched ONCE AT MOUNT (the pool probe -- the
	// known-pool rule needs the pool KNOWN before the first open, so
	// a single-account instance answers its honestly disabled add
	// button immediately) and FRESH on every open -- the per-open
	// discipline of the old share dialog; its honest failure renders
	// inside the picker, with the toggle affordance as the retry
	// path (which is exactly why an UNKNOWN pool never disables the
	// button: the open IS the retry).
	//
	// NO CANDIDATES = THE ADD AFFORDANCE DIES (the
	// known-pool rule): when the known pool is empty -- a single-account
	// instance, or the multi-add draining the last candidate -- the
	// add button is disabled (the honest disabled face: reduced
	// opacity, the not-allowed cursor) and an empty list earns no
	// row: the picker never OPENS empty, and a picker whose pool
	// drains mid-open COLLAPSES itself (an empty picker state is
	// never shown). The reserved no-candidates slot under the chips
	// row carries the words.
	//
	// THE VALIDATION SLOT (the reserved-slot rule): the
	// line under the chips row is a reserved fixed-height slot --
	// the no-candidates note appears and disappears inside it, so
	// the dialog's height is invariant to the note's comings and
	// goings, exactly as it is to the picker's open/close.
	//
	// THE KEYBOARD (the keys footer): the listbox takes
	// the focus on open; ↑/↓ move the highlight, Enter adds the
	// highlighted candidate (the list STAYS open -- a chip landed
	// while the picker offers the next, unless the pool drains
	// and collapses it), Esc
	// closes the picker alone and stops there -- the modal behind
	// it never reacts to a stroke the picker owned. The layer rides
	// OUTSIDE the Modal's focus trap (a popover's focus contract); closing
	// returns the focus to the trigger.
	import { _ } from 'svelte-i18n';
	import { tick } from 'svelte';
	import { listUsers, type UserSummary } from '$lib/api/albums';
	import { anchored } from '$lib/components/anchored';
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
	/** The instance's member list while it is loaded (the pool's source). */
	let users = $state<UserSummary[]>([]);
	/** Whether the member-list fetch runs; its honest failure key. */
	let loading = $state(false);
	let failed = $state(false);
	/**
	 * Whether a member listing has ever landed OK -- what makes the
	 * candidate pool KNOWN. The add button's disabled state rides on
	 * it: an unknown pool (never probed, or a failed probe) keeps the
	 * button enabled, because opening the picker IS the retry path.
	 */
	let probed = $state(false);
	/** The highlighted candidate's index (-1 = none). */
	let activeIndex = $state(-1);

	/** The chips row: the portaled picker's anchor and the drain-close's focus scope. */
	let chipsRow: HTMLElement | undefined = $state();
	let listbox: HTMLElement | undefined = $state();
	let addButton: HTMLButtonElement | undefined = $state();

	// The candidates: every account minus the draft's members (they
	// are chips) minus the caller (the owner never needs a row).
	const candidates = $derived(
		users.filter((user) => user.id !== selfId && !members.some((member) => member.userId === user.id))
	);

	// The add affordance's honest disabled state (the known-pool rule): the
	// pool is KNOWN empty -- a landed listing minus the caller minus
	// the granted chips. The granted set cannot push the pool below
	// empty on its own (granted accounts were never candidates); an
	// unknown pool keeps the button enabled as the retry door.
	const poolEmpty = $derived(probed && candidates.length === 0);

	// The mount probe: one listing fetch when the field mounts (the
	// dialog opened), so the pool is known BEFORE the first open.
	$effect(() => {
		void load();
	});

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

	/**
	 * Fetch the member list: once at mount (the pool probe) and fresh
	 * on every picker open. The answer always applies -- it feeds the
	 * add button's state whether the picker is open or not (the
	 * stale pool keeps that state honest while the fresh fetch runs;
	 * the picker shows its own loading face).
	 */
	async function load(): Promise<void> {
		loading = true;
		failed = false;
		const outcome = await listUsers();
		loading = false;
		if (outcome.ok) {
			users = outcome.users;
			probed = true;
			activeIndex = 0;
			// The known-pool rule: an empty picker state is never shown. The
			// fresh answer may reveal the pool empty mid-open (the
			// probe raced the open, or the listing changed) -- the
			// honest collapse. The focus stays put: the add button is
			// about to disable (an unfocusable target), and the
			// modal's Tab trap recovers the next stroke anyway.
			if (open && candidates.length === 0) {
				open = false;
			}
		} else {
			failed = true;
		}
	}

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
		void load();
		// The portaled listbox renders with this state change; focus
		// it once the DOM carries it (the keyboard driver of the picker).
		void tick().then(() => listbox?.focus());
	}

	/**
	 * Add one candidate to the draft audience. The picker stays open
	 * for the next -- UNLESS the pool drained (the known-pool rule): the last
	 * candidate's add collapses the picker (an empty picker state is
	 * never shown; the add button disables and the no-candidates note
	 * takes the reserved slot), and the focus lands on the newest
	 * chip's remove mark -- the one enabled button left in the
	 * section, the add button having just disabled.
	 */
	function add(index: number): void {
		const candidate = candidates[index];
		if (candidate === undefined) {
			return;
		}
		members = [...members, { userId: candidate.id, email: candidate.email }];
		if (candidates.length === 0) {
			open = false;
			void tick().then(() => {
				const marks = chipsRow?.querySelectorAll<HTMLButtonElement>('.audience-field__chip-remove');
				marks?.[marks.length - 1]?.focus();
			});
		}
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
	<p class="audience-field__label m-0 mb-2 font-body font-semibold text-[14px] leading-[1.4] text-text">
		{$_('albums_audience_label')}
	</p>
	<div class="audience-field__chips flex flex-wrap items-center gap-2" bind:this={chipsRow}>
		{#if members.length === 0}
			<!-- The honest empty audience: only the caller themselves. -->
			<p class="audience-field__only-me m-0 font-body font-normal text-[13px] leading-[1.4] text-text-faint">
				{$_('albums_audience_only_me')}
			</p>
		{:else}
			{#each members as member (member.userId)}
				<span
					class="audience-field__chip inline-flex items-center gap-1.5 py-1 ps-1 pe-1.5 bg-accent-soft text-accent-soft-text rounded-full font-body font-bold text-[13px] leading-[1.4]"
				>
					<span
						class="audience-field__chip-avatar w-5 h-5 rounded-full bg-accent text-white flex items-center justify-center text-[10px] leading-none font-bold shrink-0"
						aria-hidden="true">{initialOf(member.email)}</span
					>
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
		<!-- The dashed add affordance (icon + text), the picker's
		     toggle. The known-pool rule: the KNOWN-empty candidate
		     pool disables it (reduced opacity, the not-allowed cursor)
		     -- an empty list earns no row, and the button is the one
		     honest door that never opens onto nothing. An unknown
		     pool keeps it enabled: the open is the failed listing's
		     retry path. -->
		<button
			class="audience-field__add inline-flex items-center gap-1.5 px-2.5 py-1 border border-dashed border-border-strong bg-transparent text-text-soft rounded-full font-body font-bold text-[13px] leading-[1.4] cursor-pointer transition-[border-color,color] duration-(--motion-fast) ease-standard hover:border-accent hover:text-accent disabled:opacity-55 disabled:cursor-not-allowed"
			type="button"
			bind:this={addButton}
			aria-expanded={open}
			aria-controls={open ? 'audience-field-picker' : undefined}
			disabled={busy || poolEmpty}
			onclick={toggle}
		>
			<UserRoundPlus size={14} class="text-accent" aria-hidden="true" />
			{$_('albums_audience_add')}
		</button>
	</div>
	<!-- THE VALIDATION SLOT (the reserved-slot rule): a reserved
	     fixed-height line under the chips row -- the no-candidates
	     note appears and disappears INSIDE it, so the dialog's
	     height never moves for the note. -->
	<p
		class="audience-field__slot m-0 mt-2 h-[19px] font-body font-normal text-[13px] leading-[1.4] text-text-faint {poolEmpty
			? 'audience-field__no-candidates'
			: ''}"
	>
		{poolEmpty ? $_('albums_audience_no_candidates') : ''}
	</p>
	{#if open}
		<!-- THE FLOATING LAYER: the
		     candidate block portals to document.body as a fixed layer
		     anchored under the chips row (the shared `anchored`
		     action) -- the modal's overflow never clips it and the
		     dialog NEVER changes height for it. The surface mirrors
		     the account dropdown (bg-card, overlay radius, overlay
		     shadow): it is a popover now, not a form block. The
		     focus rides in the layer (outside the Modal's trap, a
		     popover's focus contract) and returns to the trigger on close. -->
		<div
			class="audience-field__picker fixed z-(--z-dropdown) box-border bg-card border border-border rounded-overlay shadow-overlay overflow-y-auto [scrollbar-gutter:stable]"
			use:anchored={{ anchor: () => chipsRow }}
			bind:this={listbox}
			id="audience-field-picker"
			role="listbox"
			aria-label={$_('albums_audience_candidates_label')}
			tabindex="0"
			onkeydown={handleKeydown}
		>
			{#if loading}
				<p
					class="audience-field__loading m-0 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in"
					role="status"
				>
					{$_('albums_audience_loading')}
				</p>
			{:else if failed}
				<p
					class="audience-field__failed m-0 px-3 py-2 font-body font-semibold text-[13px] leading-[1.4] text-danger-text"
					role="alert"
				>
					{$_('albums_audience_failed')}
				</p>
			{:else}
				{#each candidates as candidate, index (candidate.id)}
					<!-- The candidate row: a state selector, not an action
					     button -- mousedown adds (before any blur could
					     close the picker); the disabled account renders
					     its honest note and stays pickable (the server's
					     skip report is the authority). -->
					<div
						class="audience-field__row flex items-center gap-2.5 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] cursor-pointer {index ===
						activeIndex
							? 'bg-accent-soft text-accent-soft-text'
							: 'text-text'}"
						id="audience-field-option-{index}"
						role="option"
						tabindex="-1"
						aria-selected={index === activeIndex}
						onmousedown={(event) => {
							event.preventDefault();
							add(index);
						}}
					>
						<span
							class="audience-field__row-avatar w-7 h-7 rounded-full bg-accent-soft text-accent-soft-text flex items-center justify-center text-[12px] leading-none font-bold shrink-0"
							aria-hidden="true">{initialOf(candidate.email)}</span
						>
						<span class="audience-field__row-name truncate">{candidate.email}</span>
						{#if candidate.isDisabled}
							<span
								class="audience-field__row-note font-body font-normal text-[12px] leading-[1.4] {index ===
								activeIndex
									? 'text-accent-soft-text'
									: 'text-text-faint'}">{$_('albums_audience_disabled_note')}</span
							>
						{/if}
					</div>
				{/each}
				<div
					class="audience-field__keys flex gap-3 px-3 py-1.5 border-t border-border font-body font-normal text-[11px] leading-[1.4] text-text-faint"
				>
					<span>{$_('albums_keys_select')}</span>
					<span>{$_('albums_keys_add')}</span>
					<span>{$_('albums_keys_close')}</span>
				</div>
			{/if}
		</div>
	{/if}
</section>
