<script lang="ts">
	// The source-path completion combobox of the albums-as-sources
	// dialogs: the typed prefix in, the pickable directories out, the
	// list a FLOATING LAYER (the earlier inline-expansion design is
	// retired -- it traded an offset shift for height churn, the modal growing
	// and shrinking with every list appearance, the wrong tool). The
	// layer portals to document.body as a fixed, anchored surface
	// (the shared `anchored` action): the modal body's overflow can
	// never clip it, and the dialog NEVER changes height for it. The
	// rows: the folder glyph + the full path, the active
	// highlight, the keys footer (↑↓ / Enter / Esc). Each row also
	// carries the write fact the listing's probe discovered -- the
	// caller learns it through `onpick` at the moment of the pick,
	// the fact the create dialog's deletability row renders from.
	//
	// THE VALIDATION SLOT (the reserved-slot rule): the line under the
	// field is a reserved fixed-height slot -- the hint and the
	// failure SWAP inside it (slot-replace, never slot-add), so the
	// dialog's height is invariant to the validation's comings and
	// goings exactly as it is to the picker's open/close.
	//
	// THE REQUEST DISCIPLINE (the pinned semantics): debounced
	// ~250ms and single-flight -- a burst of keystrokes costs ONE
	// in-flight request (later triggers park behind it and one
	// trailing fetch runs when it lands), and a stale answer NEVER
	// overwrites a newer one (every run carries a generation; only
	// the newest generation's answer applies). The empty prefix
	// lists the container's root itself (the system-wide domain --
	// the allowlist and its no-roots posture retired); an empty
	// answer is the honest no-match note, never an error state.
	// The completion's own failure renders as a line that points
	// back at the completion (retry by typing or re-focusing),
	// never as a write error.
	//
	// THE KEYBOARD: ↑/↓ move the highlight over the rows (clamped,
	// the list opens closed on ArrowDown), Enter completes the
	// highlighted row into the field and closes the list, Esc closes
	// the list alone -- both stopPropagation so the surrounding
	// modal (its own Escape and any Enter-submit) never reacts to a
	// stroke the list already owned. The input KEEPS the focus while
	// the list is open (the combobox pattern); the layer floats
	// outside the Modal's focus trap (a popover's focus contract).
	import { _ } from 'svelte-i18n';
	import { completeAlbumPath, type AlbumPathCompletion } from '$lib/api/albums';
	import { anchored } from '$lib/components/anchored';
	import Folder from '@lucide/svelte/icons/folder';

	type Props = {
		/** The draft path (bindable): a completion writes its row into it. */
		path: string;
		/** The field's accessible label (the caller resolves the i18n). */
		label: string;
		/** The input's id, tied to the label's `for` (the caller owns it). */
		id: string;
		/** The input's placeholder (the caller resolves the i18n). */
		placeholder?: string;
		/** The hint line under the input (the field hint). */
		hint?: string;
		/** The write's failure text under the field; the input renders invalid while set. */
		failure?: string;
		/** Writes run one at a time: the input disables while in flight. */
		busy?: boolean;
		/**
		 * One picked row, with the write fact the listing's probe
		 * discovered for it -- the create dialog's deletability row
		 * renders exactly for a writable pick. The fact rides the
		 * exact picked path: the caller compares its own draft against
		 * the row and drops the fact the moment typing invalidates it.
		 */
		onpick?: (row: AlbumPathCompletion) => void;
	};

	let {
		path = $bindable(''),
		label,
		id,
		placeholder = '',
		hint = '',
		failure = '',
		busy = false,
		onpick
	}: Props = $props();

	/** The debounce window (~250ms). */
	const DEBOUNCE_MS = 250;

	type CompletionState = 'idle' | 'loading' | 'ready' | 'failed';

	/** The last landed answer's rows (the server's, at most 20, each with its write fact). */
	let rows = $state<AlbumPathCompletion[]>([]);
	/** Whether the list renders at all. */
	let open = $state(false);
	/** The highlighted row's index (-1 = none). */
	let activeIndex = $state(-1);
	/** The current run's face: loading, the rows, the empty note, the failure. */
	let phase = $state<CompletionState>('idle');
	// The single-flight pair: the running flag parks later triggers,
	// the queued flag runs ONE trailing fetch when the flight lands
	// (the wall's runLoad collapse, applied to a combobox).
	let running = false;
	let queued = false;
	// The generation guard: only the newest request's answer ever
	// applies, so a slow stale response cannot overwrite a newer one.
	let generation = 0;
	let timer: ReturnType<typeof setTimeout> | undefined;

	/** The combobox input: the layer's anchor and its focus home. */
	let input: HTMLInputElement | undefined = $state();
	let listbox: HTMLElement | undefined = $state();

	$effect(() => {
		return () => {
			if (timer !== undefined) {
				clearTimeout(timer);
			}
		};
	});

	/** Arm the debounced completion (every input event and focus). */
	function schedule(): void {
		if (timer !== undefined) {
			clearTimeout(timer);
		}
		open = true;
		activeIndex = -1;
		timer = setTimeout(() => {
			timer = undefined;
			void run();
		}, DEBOUNCE_MS);
	}

	/** The single-flight run: park behind an in-flight request, one trailing fetch when it lands. */
	async function run(): Promise<void> {
		if (running) {
			queued = true;
			return;
		}
		running = true;
		generation += 1;
		const mine = generation;
		const prefix = path;
		phase = 'loading';
		const outcome = await completeAlbumPath(prefix);
		running = false;
		// A stale answer never applies: a newer run owns the state.
		if (mine !== generation) {
			return;
		}
		if (outcome.ok) {
			rows = outcome.completions;
			phase = 'ready';
			activeIndex = rows.length > 0 ? 0 : -1;
		} else {
			phase = 'failed';
		}
		if (queued) {
			queued = false;
			void run();
		}
	}

	/** Complete one row into the field, hand the row to the caller, and close the list. */
	function complete(index: number): void {
		const row = rows[index];
		if (row === undefined) {
			return;
		}
		path = row.path;
		open = false;
		activeIndex = -1;
		onpick?.(row);
	}

	/** The keyboard the list owns; strokes it handles never reach the modal. */
	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
			event.preventDefault();
			event.stopPropagation();
			if (!open) {
				open = true;
				return;
			}
			if (rows.length === 0) {
				return;
			}
			const direction = event.key === 'ArrowDown' ? 1 : -1;
			activeIndex = Math.min(rows.length - 1, Math.max(0, activeIndex + direction));
			return;
		}
		if (event.key === 'Enter') {
			// The combobox's Enter is the completion key, never a
			// submit key: it completes the highlighted row when one
			// is active, and never falls through to a surrounding
			// form's implicit submit.
			event.preventDefault();
			event.stopPropagation();
			if (open && activeIndex >= 0) {
				complete(activeIndex);
			}
			return;
		}
		if (event.key === 'Escape') {
			if (open) {
				// The list's Esc closes the list alone -- the dialog
				// behind it stays (its own Escape owns the next stroke).
				event.stopPropagation();
				open = false;
				activeIndex = -1;
			}
		}
	}

	/** Close on blur unless the focus moves into the list itself. */
	function handleBlur(event: FocusEvent): void {
		const next = event.relatedTarget;
		if (next !== null && listbox !== undefined && listbox.contains(next as Node)) {
			return;
		}
		open = false;
	}

	/** One candidate's avatar-sized glyph color key class hook. */
	function rowId(index: number): string {
		return `${id}-option-${index}`;
	}
</script>

<div class="path-completion flex flex-col">
	<label class="path-completion__label font-body font-semibold text-[14px] leading-[1.4] text-text" for={id}
		>{label}</label
	>
	<input
		class="path-completion__input w-full px-3 py-[9px] font-body font-normal text-[14px] leading-[1.5] text-text bg-card border rounded-control outline-none transition-[border-color] duration-(--motion-fast) ease-standard {failure ===
		''
			? 'border-border-strong focus:border-accent'
			: 'border-danger'}"
		{id}
		bind:this={input}
		bind:value={path}
		type="text"
		{placeholder}
		disabled={busy}
		spellcheck="false"
		autocomplete="off"
		role="combobox"
		aria-expanded={open}
		aria-controls={open ? `${id}-listbox` : undefined}
		aria-activedescendant={open && activeIndex >= 0 ? rowId(activeIndex) : undefined}
		aria-autocomplete="list"
		aria-invalid={failure === '' ? undefined : 'true'}
		aria-describedby="{id}-note"
		oninput={() => schedule()}
		onfocus={() => schedule()}
		onkeydown={handleKeydown}
		onblur={handleBlur}
	/>
	<!-- THE VALIDATION SLOT (the reserved-slot rule): the
	     line under the field is a reserved fixed-height slot -- the
	     hint and the failure SWAP inside it (slot-replace, never
	     slot-add), so the dialog's height is invariant to the
	     validation's comings and goings, exactly as it is to the
	     picker's open/close. The slot reserves two lines while a
	     hint or a failure stands (the hint wraps two lines),
	     one line for a bare field (the settings' add-root row); a
	     text longer than the standing face overflows visibly rather
	     than clipping -- honest words beat clipped ones. The slot's
	     id is the input's described-by partner in both faces. -->
	<p
		class="path-completion__slot m-0 mt-1 {hint === '' && failure === ''
			? 'h-[19px]'
			: 'h-[37px]'} font-body font-normal text-[13px] leading-[1.4] {failure !== ''
			? 'path-completion__failure font-semibold text-danger-text'
			: 'path-completion__hint text-text-faint'}"
		id="{id}-note"
		role={failure !== '' ? 'alert' : undefined}
	>
		{failure !== '' ? failure : hint}
	</p>
	{#if open}
		<!-- THE FLOATING LAYER: the
		     candidate block portals to document.body as a fixed layer
		     anchored under the input (the shared `anchored` action) --
		     the modal's overflow never clips it and the dialog NEVER
		     changes height for it. The surface mirrors the account
		     dropdown (bg-card, overlay radius, overlay shadow): it is
		     a popover now, not a form block. The focus stays on the
		     input (the combobox pattern) -- the layer rides outside
		     the Modal's focus trap (a popover's focus contract), and its
		     rows are plain descendants the input's aria-* wiring
		     still points at. -->
		<div
			class="path-completion__suggest fixed z-(--z-dropdown) box-border bg-card border border-border rounded-overlay shadow-overlay overflow-y-auto [scrollbar-gutter:stable]"
			use:anchored={{ anchor: () => input }}
			bind:this={listbox}
			id="{id}-listbox"
			role="listbox"
			aria-label={$_('albums_path_suggestions_label')}
		>
			{#if phase === 'loading'}
				<p
					class="path-completion__loading m-0 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in"
					role="status"
				>
					{$_('albums_path_loading')}
				</p>
			{:else if phase === 'failed'}
				<p
					class="path-completion__failed m-0 px-3 py-2 font-body font-semibold text-[13px] leading-[1.4] text-danger-text"
					role="alert"
				>
					{$_('albums_path_completion_failed')}
				</p>
			{:else if rows.length === 0}
				<!-- The honest empty answer: nothing under the typed prefix
			         matches (the system-wide domain has no closed posture to
			         report -- the allowlist and its no-roots face retired). -->
				<p
					class="path-completion__no-match m-0 px-3 py-2 font-body font-normal text-[13px] leading-[1.5] text-text-soft"
				>
					{$_('albums_path_no_match')}
				</p>
			{:else}
				{#each rows as candidate, index (candidate.path)}
					<!-- The row: the state-selector (not an action button)
					     shape -- the folder glyph + the
					     full path, the active highlight, mousedown
					     completes before any blur can close the list. -->
					<div
						class="path-completion__row flex items-center gap-2 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] cursor-pointer {index ===
						activeIndex
							? 'bg-accent-soft text-accent-soft-text'
							: 'text-text'}"
						id={rowId(index)}
						role="option"
						tabindex="-1"
						aria-selected={index === activeIndex}
						onmousedown={(event) => {
							event.preventDefault();
							complete(index);
						}}
					>
						<Folder
							size={16}
							class="shrink-0 {index === activeIndex ? 'text-accent-soft-text' : 'text-accent'}"
							aria-hidden="true"
						/>
						<span class="path-completion__row-path overflow-hidden text-ellipsis whitespace-nowrap"
							>{candidate.path}</span
						>
					</div>
				{/each}
				<div
					class="path-completion__keys flex gap-3 px-3 py-1.5 border-t border-border font-body font-normal text-[11px] leading-[1.4] text-text-faint"
				>
					<span>{$_('albums_keys_select')}</span>
					<span>{$_('albums_keys_complete')}</span>
					<span>{$_('albums_keys_close')}</span>
				</div>
			{/if}
		</div>
	{/if}
</div>
