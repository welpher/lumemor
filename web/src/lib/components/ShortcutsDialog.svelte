<script lang="ts">
	// The `?` shortcut sheet (web IA §6's 键盘速查表, paid after E4):
	// the shared Modal over the shell, listing every keyboard binding
	// the app ACTUALLY carries -- collected from the code, not the
	// design's wishlist, so the sheet can never promise a stroke that
	// does nothing. Each row is the honest two-column shape the pin
	// asks for: the kbd-styled key glyphs beside their action in
	// plain language, grouped by the context the binding lives in
	// (Esc is a different verb in the viewer, in a selection and in
	// a picker, and the group heading is what keeps the list honest).
	//
	// THE `?` GATE: the sheet toggles on the `?` stroke outside every
	// editable context (typing belongs to the field -- the viewer's
	// own guard, shared shape), and defers to an open dialog (the
	// layered dismissal the viewer pins: "the open dialog owns every
	// other key") so the sheet never stacks on a dialog's own
	// keyboard domain. While the sheet itself is open the same stroke
	// closes it (a toggle, per the pin), and the Modal's own Esc and
	// focus trap close and restore it.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';

	type ShortcutRow = {
		/** The key glyphs, kbd-styled, one <kbd> per key. */
		keys: readonly string[];
		/** The action's i18n key, in plain language. */
		actionKey: string;
	};

	type ShortcutGroup = {
		/** The context heading's i18n key. */
		headingKey: string;
		rows: readonly ShortcutRow[];
	};

	// The collected bindings, live ones only:
	// - Anywhere: this sheet's own toggle, the shell's layered Esc
	//   (the account menu, the drawer, every dialog), and the `/`
	//   focus-search stroke (the content shell's topbar box).
	// - The viewer (/a/[id]): ←/→ step the wall's sequence, i toggles
	//   the info panel, +/− zoom a photo, Esc closes the panel first
	//   then leaves the route.
	// - Selection: the wall's and the unnamed region's Esc drops a
	//   live selection (the same full exit the batch bar's clear
	//   action runs).
	// - The pickers (the album path completion and the audience
	//   field): ↑/↓ move the highlight, Enter completes, Esc closes
	//   the list alone.
	const GROUPS: readonly ShortcutGroup[] = [
		{
			headingKey: 'shortcuts_anywhere_heading',
			rows: [
				{ keys: ['?'], actionKey: 'shortcuts_action_sheet' },
				{ keys: ['Esc'], actionKey: 'shortcuts_action_close' },
				{ keys: ['/'], actionKey: 'shortcuts_action_search' }
			]
		},
		{
			headingKey: 'shortcuts_viewer_heading',
			rows: [
				{ keys: ['←', '→'], actionKey: 'shortcuts_action_step' },
				{ keys: ['i'], actionKey: 'shortcuts_action_info' },
				{ keys: ['+', '−'], actionKey: 'shortcuts_action_zoom' },
				{ keys: ['Esc'], actionKey: 'shortcuts_action_exit' }
			]
		},
		{
			headingKey: 'shortcuts_selection_heading',
			rows: [{ keys: ['Esc'], actionKey: 'shortcuts_action_clear' }]
		},
		{
			headingKey: 'shortcuts_pickers_heading',
			rows: [
				{ keys: ['↑', '↓'], actionKey: 'shortcuts_action_picker_move' },
				{ keys: ['Enter'], actionKey: 'shortcuts_action_picker_pick' },
				{ keys: ['Esc'], actionKey: 'shortcuts_action_picker_close' }
			]
		}
	];

	type Props = {
		/** Open state; bind it from the shell, the modal closes itself. */
		open?: boolean;
	};

	let { open = $bindable(false) }: Props = $props();

	/** Typing belongs to the field: the viewer's own guard, shared shape. */
	function isEditableTarget(target: EventTarget | null): boolean {
		if (target === null || !(target instanceof HTMLElement)) {
			return false;
		}
		return target.closest('input, textarea, select, [contenteditable="true"], [contenteditable=""]') !== null;
	}

	function handleWindowKeydown(event: KeyboardEvent): void {
		if (event.key !== '?') {
			return;
		}
		// The toggle's own guards: never steal a stroke aimed at a
		// field, and never stack over an open dialog's keyboard
		// domain (the sheet's own panel is a dialog -- the open arm
		// checks it, the close arm does not, so `?` closes the sheet
		// exactly as the pin asks).
		if (isEditableTarget(event.target)) {
			return;
		}
		if (!open && document.querySelector('[role="dialog"]') !== null) {
			return;
		}
		open = !open;
	}
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<Modal bind:open title={$_('shortcuts_title')}>
	<div class="shortcuts flex flex-col gap-4">
		{#each GROUPS as group (group.headingKey)}
			<section class="shortcuts__group">
				<h3
					class="shortcuts__group-title m-0 mb-1.5 font-display font-bold text-[14px] leading-[1.3] text-text-soft"
				>
					{$_(group.headingKey)}
				</h3>
				<ul class="shortcuts__list m-0 p-0 list-none flex flex-col gap-1">
					{#each group.rows as row (row.actionKey)}
						<li class="shortcuts__row grid grid-cols-[auto_1fr] gap-3 items-baseline">
							<span class="shortcuts__keys flex gap-1">
								{#each row.keys as key (key)}
									<kbd
										class="shortcuts__key min-w-[22px] px-1.5 py-0.5 text-center font-mono font-normal text-[12px] leading-[1.4] text-text bg-page border border-border-strong rounded-[6px]"
										>{key}</kbd
									>
								{/each}
							</span>
							<span class="shortcuts__action font-body font-normal text-[14px] leading-[1.4] text-text"
								>{$_(row.actionKey)}</span
							>
						</li>
					{/each}
				</ul>
			</section>
		{/each}
	</div>
</Modal>
