<script lang="ts">
	// The upload queue overlay of the (content) shell: a bottom-right
	// panel listing the singleton queue's rows (the determinate bar of
	// an in-flight XHR, the SSE-confirmed states, the retry affordance
	// of a failed row), the full-page drag-and-drop hot zone (with an
	// edge-counted drag hint), and the three-state conflict dialog
	// (restore the trashed copy / save as new / keep -- the pinned
	// Review Focus ③ semantics live in the queue module).
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
	<div class="drophint" aria-hidden="true">
		<p>{$_('upload_drop_hint')}</p>
	</div>
{/if}

{#if items.length > 0}
	<section class="overlay" aria-label={$_('upload_queue_title')}>
		<header class="overlay__head">
			<h2 class="overlay__title">{$_('upload_queue_title')}</h2>
			{#if activeCount > 0}
				<span class="overlay__count">{$_('upload_active_count', { values: { count: activeCount } })}</span>
			{/if}
		</header>
		<ul class="overlay__rows">
			{#each items as item (item.id)}
				<li class="row row--{item.state}">
					<div class="row__main">
						<span class="row__name" title={item.fileName}>{item.fileName}</span>
						<span class="row__state">{stateLabel(item)}</span>
					</div>
					{#if item.state === 'uploading' || item.state === 'processing'}
						<div
							class="row__bar"
							class:row__bar--indeterminate={item.state === 'processing'}
							role="progressbar"
							aria-label={$_('upload_state_uploading')}
							aria-valuemin="0"
							aria-valuemax="100"
							aria-valuenow={Math.round(item.progress * 100)}
						>
							<div class="row__bar-fill" style={`width:${Math.round(item.progress * 100)}%`}></div>
						</div>
					{/if}
					{#if item.state === 'failed' && item.errorKey !== ''}
						<p class="row__error" role="alert">{errorText(item.errorKey)}</p>
					{/if}
					<div class="row__actions">
						{#if item.state === 'failed' && uploadFormatSupported(item.fileName)}
							<button class="row__action" type="button" onclick={() => retryUpload(item.id)}>
								{$_('action_retry')}
							</button>
						{/if}
						{#if item.state === 'conflict'}
							<button class="row__action" type="button" onclick={() => openUploadConflict(item.id)}>
								{$_('upload_resolve_conflict')}
							</button>
						{/if}
						<button
							class="row__close"
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
	</section>
{/if}

<!-- The three-state conflict decision (the pinned semantics): the
     Modal now carries the focus trap, so keyboard users stay inside. -->
<Modal
	open={conflictItem !== null}
	title={$_('upload_conflict_title')}
	onclose={() => closeUploadConflict()}
>
	{#snippet children()}
		{#if conflictItem !== null && conflictItem.asset !== null}
			<p class="conflict__body">
				{$_('upload_conflict_body', { values: { name: conflictItem.fileName } })}
			</p>
			{#if conflictError !== ''}
				<p class="conflict__error" role="alert">{errorText(conflictError)}</p>
			{/if}
		{/if}
	{/snippet}
	{#snippet footer()}
		<Button
			variant="ghost"
			disabled={resolving}
			onclick={() => closeUploadConflict()}>{$_('common_cancel')}</Button
		>
		<Button
			variant="danger"
			disabled={resolving}
			onclick={() => void chooseSaveAs()}>{$_('upload_conflict_save_as')}</Button
		>
		<Button
			variant="primary"
			disabled={resolving}
			onclick={() => void chooseRestore()}>{$_('upload_conflict_restore')}</Button
		>
	{/snippet}
</Modal>

<style>
	.drophint {
		position: fixed;
		inset: 0;
		z-index: 90;
		display: flex;
		align-items: center;
		justify-content: center;
		background-color: color-mix(in srgb, var(--color-accent-soft) 40%, transparent);
		border: 3px dashed var(--color-accent);
		/* The hint is visual only: the drag events stay on the window,
		 * and a drop landing on the hint must still reach the page. */
		pointer-events: none;
	}

	.drophint p {
		margin: 0;
		padding: 16px 28px;
		border-radius: var(--radius-overlay);
		background-color: var(--color-card);
		font: 700 18px/1.4 var(--font-body);
		color: var(--color-accent-on);
		box-shadow: var(--shadow-overlay);
	}

	.overlay {
		position: fixed;
		right: 16px;
		bottom: 16px;
		z-index: 70;
		width: min(360px, calc(100vw - 32px));
		max-height: min(60vh, 480px);
		display: flex;
		flex-direction: column;
		padding: 12px;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-overlay);
		box-shadow: var(--shadow-overlay);
	}

	.overlay__head {
		display: flex;
		align-items: baseline;
		gap: 8px;
		padding-bottom: 8px;
	}

	.overlay__title {
		margin: 0;
		font: 700 14px/1.4 var(--font-body);
		color: var(--color-text);
	}

	.overlay__count {
		font: 400 12px/1.4 var(--font-body);
		color: var(--color-text-soft);
	}

	.overlay__rows {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
		overflow-y: auto;
	}

	.row {
		display: grid;
		grid-template-columns: 1fr auto;
		grid-template-areas:
			'main close'
			'bar close'
			'error close'
			'actions close';
		align-items: center;
		column-gap: 8px;
		padding: 8px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-control);
	}

	.row__main {
		grid-area: main;
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.row__name {
		font: 600 13px/1.4 var(--font-body);
		color: var(--color-text);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.row__state {
		font: 400 11px/1.4 var(--font-body);
		color: var(--color-text-soft);
	}

	.row--done .row__state,
	.row--processing .row__state {
		color: var(--color-teal, var(--color-text-soft));
	}

	.row--failed .row__state {
		color: var(--color-danger-text);
	}

	.row__bar {
		grid-area: bar;
		margin-top: 6px;
		height: 4px;
		border-radius: 2px;
		background-color: var(--color-border);
		overflow: hidden;
	}

	.row__bar-fill {
		height: 100%;
		background-color: var(--color-accent);
		transition: width var(--motion-fast) var(--ease-standard);
	}

	.row__bar--indeterminate .row__bar-fill {
		width: 40%;
		animation: indeterminate 1.1s var(--ease-standard) infinite;
	}

	.row__error {
		grid-area: error;
		margin: 6px 0 0;
		font: 400 12px/1.45 var(--font-body);
		color: var(--color-danger-text);
	}

	.row__actions {
		grid-area: actions;
		display: flex;
		align-items: center;
		gap: 6px;
		margin-top: 6px;
	}

	.row__action {
		padding: 3px 8px;
		border: 1px solid var(--color-border-strong);
		border-radius: var(--radius-control);
		background: none;
		font: 600 12px/1.4 var(--font-body);
		color: var(--color-text);
		cursor: pointer;
		transition: background-color var(--motion-fast) var(--ease-standard);
	}

	.row__action:hover {
		background-color: var(--color-accent-soft);
	}

	.row__close {
		grid-area: close;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 26px;
		height: 26px;
		border: none;
		border-radius: var(--radius-control);
		background: none;
		color: var(--color-text-soft);
		cursor: pointer;
		transition:
			background-color var(--motion-fast) var(--ease-standard),
			color var(--motion-fast) var(--ease-standard);
	}

	.row__close:hover {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}

	.conflict__body {
		margin: 0;
		font: 400 15px/1.5 var(--font-body);
		color: var(--color-text);
		white-space: pre-line;
	}

	.conflict__error {
		margin: 10px 0 0;
		font: 400 13px/1.4 var(--font-body);
		color: var(--color-danger-text);
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

	@keyframes indeterminate {
		from {
			transform: translateX(-100%);
		}
		to {
			transform: translateX(250%);
		}
	}
</style>
