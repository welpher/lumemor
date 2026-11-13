<script lang="ts">
	// The upload queue overlay of the (content) shell: a bottom-right
	// panel listing the singleton queue's rows (the determinate bar of
	// an in-flight XHR, the SSE-confirmed states, the retry affordance
	// of a failed row), the full-page drag-and-drop hot zone (with an
	// edge-counted drag hint), and the three-state conflict dialog
	// (restore the trashed copy / save as new / keep -- the pinned
	// pinned conflict semantics live in the queue module).
	//
	// The queue survives route navigation (the module singleton); this
	// component only renders and feeds it.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import {
		uploadQueueItems,
		uploadQueueConflictId,
		uploadQueueConflictErrorKey,
		enqueueUploadFiles,
		closeUploadConflict,
		openUploadConflict,
		resolveUploadByRestore,
		resolveUploadBySaveAs,
		retryUpload,
		dismissUpload,
		uploadFormatSupported
	} from '$lib/upload/queue.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import X from '@lucide/svelte/icons/x';
	import FilePlus from '@lucide/svelte/icons/file-plus';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';

	// The hint-level front end of the server's upload whitelist:
	// dropped files whose extension is unknown are parked failed
	// before any bytes move (the server stays authoritative).
	// The bottom-right panel and the dialog carry no picker markup of
	// their own: the shell layout's Upload button owns the hidden
	// file input (next to its trigger), mirroring the same
	// hint-level whitelist in its accept attribute.
	let dragDepth = $state(0);
	const items = $derived(uploadQueueItems());
	const conflictId = $derived(uploadQueueConflictId());
	const conflictItem = $derived(items.find((item) => item.id === conflictId) ?? null);
	const activeCount = $derived(items.filter((item) => item.state === 'uploading').length);

	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	/** The drag hint counts edges, so crossing into child elements
	 * during the drag never flickers the overlay away. */
	function handlesFiles(event: DragEvent): boolean {
		return Array.from(event.dataTransfer?.types ?? []).includes('Files');
	}

	function handleDragEnter(event: DragEvent): void {
		if (handlesFiles(event)) {
			dragDepth += 1;
		}
	}

	function handleDragLeave(event: DragEvent): void {
		if (handlesFiles(event) && dragDepth > 0) {
			dragDepth -= 1;
		}
	}

	function handleDragOver(event: DragEvent): void {
		if (handlesFiles(event)) {
			// The default (navigating on a dropped file) is prevented on
			// dragover: only a cancelable dragover makes the drop legal.
			event.preventDefault();
		}
	}

	function handleDrop(event: DragEvent): void {
		if (!handlesFiles(event)) {
			return;
		}
		event.preventDefault();
		dragDepth = 0;
		const files = event.dataTransfer?.files;
		if (files !== undefined && files.length > 0) {
			enqueueUploadFiles(files);
		}
	}

	// The retry affordance of a failed row (the queue module
	// re-validates the hint-level whitelist, so a hint-rejected row
	// reports its rejection but never re-attempts the wire).

	// The panel's collapse: the header stays as the queue's ledger
	// (its title and the in-flight count), the rows fold away. The
	// fold is manual -- a completed row waits for it -- and a NEW row
	// re-opens the panel, so a fresh drop is never uploaded behind a
	// folded header.
	let collapsed = $state(false);
	let seenCount = 0;
	$effect(() => {
		const count = items.length;
		if (count > seenCount) {
			collapsed = false;
		}
		seenCount = count;
	});

	let resolving = $state(false);
	const conflictError = $derived(uploadQueueConflictErrorKey());

	async function chooseRestore(): Promise<void> {
		if (resolving || conflictId === null) {
			return;
		}
		resolving = true;
		await resolveUploadByRestore(conflictId);
		resolving = false;
	}

	async function chooseSaveAs(): Promise<void> {
		if (resolving || conflictId === null) {
			return;
		}
		resolving = true;
		await resolveUploadBySaveAs(conflictId);
		resolving = false;
	}

	function stateLabel(item: (typeof items)[number]): string {
		switch (item.state) {
			case 'queued':
				return $_('upload_state_waiting');
			case 'uploading':
				return $_('upload_state_uploading');
			case 'conflict':
				return $_('upload_state_conflict');
			case 'processing':
				return $_('upload_state_processing');
			case 'done':
				return $_('upload_state_done');
			case 'failed':
				return $_('upload_state_failed');
		}
	}
</script>

<svelte:window
	ondragenter={handleDragEnter}
	ondragleave={handleDragLeave}
	ondragover={handleDragOver}
	ondrop={handleDrop}
/>

{#if dragDepth > 0}
	<!-- The hint is visual only: the drag events stay on the window,
	     and a drop landing on the hint must still reach the page. -->
	<div
		class="drophint fixed inset-0 z-(--z-scrim) flex items-center justify-center bg-[color-mix(in_srgb,var(--color-accent-soft)_40%,transparent)] border-3 border-dashed border-accent pointer-events-none"
		aria-hidden="true"
	>
		<p
			class="py-4 px-7 rounded-overlay bg-card font-body font-bold text-[18px] leading-[1.4] text-accent-on shadow-overlay"
		>
			{$_('upload_drop_hint')}
		</p>
	</div>
{/if}

{#if items.length > 0}
	<section
		class="overlay fixed right-4 bottom-4 z-(--z-panel) w-[min(360px,calc(100vw-32px))] max-h-[min(60vh,480px)] box-content flex flex-col p-3 bg-card border border-border rounded-overlay shadow-overlay"
		aria-label={$_('upload_queue_title')}
	>
		<header class="overlay__head flex items-baseline gap-2 pb-2">
			<h2 class="overlay__title font-body font-bold text-[14px] leading-[1.4] text-text">
				{$_('upload_queue_title')}
			</h2>
			{#if activeCount > 0}
				<span class="overlay__count font-body font-normal text-[12px] leading-[1.4] text-text-soft tabular-nums"
					>{$_('upload_active_count', { values: { count: activeCount } })}</span
				>
			{/if}
			<button
				class="overlay__collapse ml-auto inline-flex items-center justify-center w-7 h-7 rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
				type="button"
				aria-expanded={!collapsed}
				aria-label={collapsed ? $_('upload_expand') : $_('upload_collapse')}
				onclick={() => (collapsed = !collapsed)}
			>
				<ChevronDown size={16} aria-hidden="true" class={collapsed ? 'rotate-180' : ''} />
			</button>
		</header>
		{#if !collapsed}
			<ul class="overlay__rows flex flex-col gap-2 overflow-y-auto [scrollbar-gutter:stable] list-none">
				{#each items as item (item.id)}
					<li
						class="row row--{item.state} grid grid-cols-[1fr_auto] [grid-template-areas:'main_close''bar_close''error_close''actions_close'] items-center gap-x-2 p-2 border border-border rounded-control"
					>
						<div class="row__main flex flex-col min-w-0 [grid-area:main]">
							<span
								class="row__name font-body font-semibold text-[13px] leading-[1.4] text-text truncate"
								title={item.fileName}>{item.fileName}</span
							>
							<span class="row__state font-body font-normal text-[11px] leading-[1.4] text-text-soft"
								>{stateLabel(item)}</span
							>
						</div>
						{#if item.state === 'uploading' || item.state === 'processing'}
							<!-- The determinate bar of an uploading row carries its
						     live percentage; the processing row's bar is
						     indeterminate, so it names its own state and
						     reports no value (an aria-valuenow of a settled
						     100% would misstate an unknown-duration wait).
						     Both bars enter through the anti-flicker delays:
						     the determinate progress waits out its 100ms
						     grace and the indeterminate processing bar (the
						     spinner class) the loading line's 400ms -- an
						     upload that errors or converges inside the
						     window never flashes a bar. -->
							{@const indeterminate = item.state === 'processing'}
							<div
								class="row__bar [grid-area:bar] mt-1.5 h-1 rounded-[2px] bg-border overflow-hidden"
								class:row__bar--indeterminate={indeterminate}
								class:animate-progress-in={!indeterminate}
								class:animate-loading-in={indeterminate}
								role="progressbar"
								aria-label={stateLabel(item)}
								aria-valuemin={indeterminate ? undefined : 0}
								aria-valuemax={indeterminate ? undefined : 100}
								aria-valuenow={indeterminate ? undefined : Math.round(item.progress * 100)}
							>
								<div
									class="row__bar-fill h-full bg-accent transition-[width] duration-(--motion-fast) ease-standard"
									style={`width:${Math.round(item.progress * 100)}%`}
								></div>
							</div>
						{/if}
						{#if item.state === 'failed' && item.errorKey !== ''}
							<p
								class="row__error [grid-area:error] mt-1.5 font-body font-normal text-[12px] leading-[1.45] text-danger-text"
								role="alert"
							>
								{errorText(item.errorKey)}
							</p>
						{/if}
						<div class="row__actions [grid-area:actions] flex items-center gap-1.5 mt-1.5">
							{#if item.state === 'failed' && uploadFormatSupported(item.fileName)}
								<button
									class="row__action inline-flex items-center gap-1 py-[3px] px-2 border border-border-strong rounded-control bg-transparent font-body font-semibold text-[12px] leading-[1.4] text-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft"
									type="button"
									onclick={() => retryUpload(item.id)}
								>
									<RotateCcw size={14} aria-hidden="true" />
									{$_('action_retry')}
								</button>
							{/if}
							{#if item.state === 'conflict'}
								<button
									class="row__action inline-flex items-center gap-1 py-[3px] px-2 border border-border-strong rounded-control bg-transparent font-body font-semibold text-[12px] leading-[1.4] text-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft"
									type="button"
									onclick={() => openUploadConflict(item.id)}
								>
									<FilePlus size={14} aria-hidden="true" />
									{$_('upload_resolve_conflict')}
								</button>
							{/if}
							<button
								class="row__close [grid-area:close] inline-flex items-center justify-center w-[26px] h-[26px] rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
								type="button"
								aria-label={$_('upload_dismiss')}
								onclick={() => dismissUpload(item.id)}
							>
								<svg viewBox="0 0 24 24" width="14" height="14" fill="none" aria-hidden="true">
									<path
										d="M6 6l12 12M18 6L6 18"
										stroke="currentColor"
										stroke-width="1.75"
										stroke-linecap="round"
									/>
								</svg>
							</button>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
{/if}

<!-- The three-state conflict decision (the pinned semantics): the
     Modal now carries the focus trap, so keyboard users stay inside. -->
<Modal open={conflictItem !== null} title={$_('upload_conflict_title')} onclose={() => closeUploadConflict()}>
	{#if conflictItem !== null && conflictItem.asset !== null}
		<p class="conflict__body font-body font-normal text-[15px] leading-[1.5] text-text whitespace-pre-line">
			{$_('upload_conflict_body', { values: { name: conflictItem.fileName } })}
		</p>
		{#if conflictError !== ''}
			<p
				class="conflict__error mt-2.5 font-body font-normal text-[13px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{errorText(conflictError)}
			</p>
		{/if}
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={resolving} onclick={() => closeUploadConflict()}
			><X size={20} aria-hidden="true" />{$_('common_cancel')}</Button
		>
		<Button variant="danger" disabled={resolving} onclick={() => void chooseSaveAs()}
			><FilePlus size={20} aria-hidden="true" />{$_('upload_conflict_save_as')}</Button
		>
		<Button variant="primary" disabled={resolving} onclick={() => void chooseRestore()}
			><RotateCcw size={20} aria-hidden="true" />{$_('upload_conflict_restore')}</Button
		>
	{/snippet}
</Modal>

<style>
	/* Residual styles (verbatim): the row-state compounds whose class
	 * names are dynamic (row--{state}), the indeterminate bar's fill
	 * rule, and its keyframes. */
	.row--done .row__state,
	.row--processing .row__state {
		color: var(--color-teal, var(--color-text-soft));
	}

	.row--failed .row__state {
		color: var(--color-danger-text);
	}

	.row__bar--indeterminate .row__bar-fill {
		width: 40%;
		animation: indeterminate 1.1s var(--ease-standard) infinite;
	}

	@keyframes indeterminate {
		from {
			transform: translateX(-100%);
		}
		to {
			transform: translateX(250%);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.row__bar--indeterminate .row__bar-fill,
		.row__bar-fill,
		.drophint,
		.overlay {
			animation: none;
			transition: none;
		}
	}
</style>
