<script lang="ts">
	// The source-path completion combobox of the albums-as-sources
	// dialogs (the specimen's inline-expansion pin, 2026-11-08): the
	// typed prefix in, the pickable directories out, the list
	// expanding INLINE under the input -- the dialog grows and its
	// footer moves down, zero occlusion by construction, no floating
	// layer a modal body's overflow could clip. The rows are the
	// specimen's: the folder glyph + the full path, the active
	// highlight, the keys footer (↑↓ / Enter / Esc).
	//
	// THE REQUEST DISCIPLINE (the plan's pinned semantics): debounced
	// ~250ms and single-flight -- a burst of keystrokes costs ONE
	// in-flight request (later triggers park behind it and one
	// trailing fetch runs when it lands), and a stale answer NEVER
	// overwrites a newer one (every run carries a generation; only
	// the newest generation's answer applies). The empty prefix
	// lists the mount roots themselves (the contract's "start
	// here"); an empty answer over an empty prefix is the honest
	// NO-ROOTS state -- the allowlist is closed, and the dialog says
	// so instead of pretending the world is empty. The completion's
	// own failure renders as a line that points back at the
	// completion (retry by typing or re-focusing), never as a write
	// error.
	//
	// THE KEYBOARD: ↑/↓ move the highlight over the rows (clamped,
	// the list opens closed on ArrowDown), Enter completes the
	// highlighted row into the field and closes the list, Esc closes
	// the list alone -- both stopPropagation so the surrounding
	// modal (its own Escape and any Enter-submit) never reacts to a
	// stroke the list already owned.
	import { _ } from 'svelte-i18n';
	import { completeAlbumPath } from '$lib/api/albums';
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
		/** The hint line under the input (the specimen's field hint). */
		hint?: string;
		/** The write's failure text under the field; the input renders invalid while set. */
		failure?: string;
		/** Writes run one at a time: the input disables while in flight. */
		busy?: boolean;
	};

	let { path = $bindable(''), label, id, placeholder = '', hint = '', failure = '', busy = false }: Props = $props();

	/** The debounce window the plan pins (~250ms). */
	const DEBOUNCE_MS = 250;

	type CompletionState = 'idle' | 'loading' | 'ready' | 'failed';

	/** The last landed answer's rows (the server's, at most 20). */
	let paths = $state<string[]>([]);
	/** Whether the list renders at all. */
	let open = $state(false);
	/** The highlighted row's index (-1 = none). */
	let activeIndex = $state(-1);
	/** The current run's face: loading, the rows, the empty notes, the failure. */
	let phase = $state<CompletionState>("idle");
	/**
	 * Whether the last answer was the EMPTY-PREFIX empty list -- the
	 * no-roots posture (the allowlist is closed), surfaced honestly
	 * instead of a bare "nothing matched".
	 */
	let emptyPrefixAnswer = $state(false);

	// The single-flight pair: the running flag parks later triggers,
	// the queued flag runs ONE trailing fetch when the flight lands
	// (the wall's runLoad collapse, applied to a combobox).
	let running = false;
	let queued = false;
	// The generation guard: only the newest request's answer ever
	// applies, so a slow stale response cannot overwrite a newer one.
	let generation = 0;
	let timer: ReturnType<typeof setTimeout> | undefined;

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
		phase = "loading";
		const outcome = await completeAlbumPath(prefix);
		running = false;
		// A stale answer never applies: a newer run owns the state.
		if (mine !== generation) {
			return;
		}
		if (outcome.ok) {
			paths = outcome.paths;
			emptyPrefixAnswer = prefix.trim() === '';
			phase = "ready";
			activeIndex = paths.length > 0 ? 0 : -1;
		} else {
			phase = "failed";
		}
		if (queued) {
			queued = false;
			void run();
		}
	}

	/** Complete one row into the field and close the list. */
	function complete(index: number): void {
		const row = paths[index];
		if (row === undefined) {
			return;
		}
		path = row;
		open = false;
		activeIndex = -1;
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
			if (paths.length === 0) {
				return;
			}
			const direction = event.key === 'ArrowDown' ? 1 : -1;
			activeIndex = Math.min(paths.length - 1, Math.max(0, activeIndex + direction));
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
	<label class="path-completion__label font-body font-semibold text-[14px] leading-[1.4] text-text" for={id}>{label}</label>
	<input
		class="path-completion__input w-full px-3 py-[9px] font-body font-normal text-[14px] leading-[1.5] text-text bg-card border rounded-control outline-none transition-[border-color] duration-(--motion-fast) ease-standard {failure === '' ? 'border-border-strong focus:border-accent' : 'border-danger'}"
		{id}
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
		aria-describedby={failure !== '' ? `${id}-failure` : undefined}
		oninput={() => schedule()}
		onfocus={() => schedule()}
		onkeydown={handleKeydown}
		onblur={handleBlur}
	/>
	{#if hint !== ''}
		<p class="path-completion__hint m-0 mt-1 font-body font-normal text-[13px] leading-[1.4] text-text-faint">{hint}</p>
	{/if}
	{#if failure !== ''}
		<p class="path-completion__failure m-0 mt-1 font-body font-semibold text-[13px] leading-[1.4] text-danger-text" id="{id}-failure" role="alert">{failure}</p>
	{/if}
	{#if open}
		<!-- THE INLINE EXPANSION (the anti-occlusion pin): the list is
		     a flow child of the field -- the dialog grows, its footer
		     moves down, and no overflow rule can ever clip it. -->
		<div
			class="path-completion__suggest mt-1.5 bg-page border border-border rounded-control overflow-hidden"
			bind:this={listbox}
			id="{id}-listbox"
			role="listbox"
			aria-label={$_('albums_path_suggestions_label')}
		>
			{#if phase === 'loading'}
				<p class="path-completion__loading m-0 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in" role="status">{$_('albums_path_loading')}</p>
			{:else if phase === 'failed'}
				<p class="path-completion__failed m-0 px-3 py-2 font-body font-semibold text-[13px] leading-[1.4] text-danger-text" role="alert">{$_('albums_path_completion_failed')}</p>
			{:else if paths.length === 0}
				{#if emptyPrefixAnswer}
					<!-- The honest no-roots posture: the allowlist is closed,
					     the admin must open a root first. -->
					<p class="path-completion__no-roots m-0 px-3 py-2 font-body font-normal text-[13px] leading-[1.5] text-text-soft">{$_('albums_path_no_roots')}</p>
				{:else}
					<p class="path-completion__no-match m-0 px-3 py-2 font-body font-normal text-[13px] leading-[1.5] text-text-soft">{$_('albums_path_no_match')}</p>
				{/if}
			{:else}
				{#each paths as candidate, index (candidate)}
					<!-- The row: the state-selector (not an action button)
					     shape of the specimen -- the folder glyph + the
					     full path, the active highlight, mousedown
					     completes before any blur can close the list. -->
					<div
						class="path-completion__row flex items-center gap-2 px-3 py-2 font-body font-normal text-[14px] leading-[1.4] cursor-pointer {index === activeIndex ? 'bg-accent-soft text-accent-soft-text' : 'text-text'}"
						id={rowId(index)}
						role="option"
						tabindex="-1"
						aria-selected={index === activeIndex}
						onmousedown={(event) => {
							event.preventDefault();
							complete(index);
						}}
					>
						<Folder size={16} class="shrink-0 {index === activeIndex ? 'text-accent-soft-text' : 'text-accent'}" aria-hidden="true" />
						<span class="path-completion__row-path overflow-hidden text-ellipsis whitespace-nowrap">{candidate}</span>
					</div>
				{/each}
				<div class="path-completion__keys flex gap-3 px-3 py-1.5 border-t border-border font-body font-normal text-[11px] leading-[1.4] text-text-faint">
					<span>{$_('albums_keys_select')}</span>
					<span>{$_('albums_keys_complete')}</span>
					<span>{$_('albums_keys_close')}</span>
				</div>
			{/if}
		</div>
	{/if}
</div>
