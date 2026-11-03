<script lang="ts">
	// The information drawer of the asset viewer: the owner-level
	// editing surfaces of one asset -- the description editor (the
	// PATCH endpoint; the drawer keeps the draft on failure so a
	// retry never retypes anything, and through the same-id detail
	// replacements of the optimistic curation flips and the SSE
	// convergence), the curation action bar
	// (favorite / hide / trash, the batch endpoint, optimistic on
	// the page's local state), and the permanent delete's confirm
	// dialog trigger -- plus the read-only panels: the file facts,
	// the EXIF hot fields folded into collapsible groups, and the
	// faces / places placeholders pinned to their M3 domains.
	//
	// The drawer is presentation and edit-mode state only: the page
	// owns the detail, the optimistic state and every network call,
	// through the onedit / onaction / onrequestdelete callbacks.
	// Escape does not close it from here -- the viewer owns the
	// layered Escape rule (modal first, then this drawer, then the
	// route) so the layers cannot race.
	import { locale, _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import { assetOriginalUrl, type AssetDetail } from '$lib/api/assets';
	import type { BatchAction } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';

	type Props = {
		/** The current asset's detail (the page's local, optimistic copy). */
		detail: AssetDetail;
		/** The favorite flag (the detail carries none; the page holds it). */
		favorite: boolean;
		/** Any curation action is in flight: buttons disable. */
		busy: boolean;
		/** The error-code i18n key of the last failed action ('' when none). */
		errorKey: string;
		/** Close the drawer. */
		onclose: () => void;
		/** Save the description; the outcome decides the edit mode. */
		onedit: (text: string) => Promise<{ ok: boolean; errorKey: string }>;
		/** Run one curation action over this asset. */
		onaction: (action: BatchAction) => Promise<boolean>;
		/** Open the permanent-delete confirm dialog (the viewer owns it). */
		onrequestdelete: () => void;
	};

	let { detail, favorite, busy, errorKey, onclose, onedit, onaction, onrequestdelete }: Props =
		$props();

	let editing = $state(false);
	let draft = $state('');
	let saving = $state(false);
	let editErrorKey = $state('');

	// The asset id the current edit state belongs to. The page
	// replaces the detail REFERENCE for the SAME asset (the
	// optimistic curation flip, the SSE convergence refetch, the
	// PATCH adoption), and only an actual asset change -- the
	// navigation to a neighbor -- drops the edit mode: the id VALUE
	// decides, never the object identity. resetFor is deliberately a
	// plain (non-reactive) let -- the reset must not observe its own
	// writes.
	let resetFor = '';

	$effect(() => {
		const id = detail.id;
		if (id === resetFor) {
			// The same asset behind a new object: an in-progress
			// draft survives the replacement.
			return;
		}
		resetFor = id;
		editing = false;
		editErrorKey = '';
	});

	function startEdit(): void {
		draft = detail.description;
		editing = true;
		editErrorKey = '';
	}

	function cancelEdit(): void {
		editing = false;
		editErrorKey = '';
	}

	async function saveEdit(): Promise<void> {
		if (saving) {
			return;
		}
		saving = true;
		editErrorKey = '';
		const outcome = await onedit(draft);
		saving = false;
		if (outcome.ok) {
			// The response is the fresh detail; the page adopted it.
			editing = false;
			return;
		}
		// The failure keeps the edit mode and the draft (a retry
		// never retypes); the stored description never changed.
		editErrorKey = outcome.errorKey;
	}

	// The shared unknown-code fallback (see $lib/i18n/errors): an
	// unknown error code has no i18n key, and svelte-i18n would echo
	// the key itself -- such codes render the generic message. The
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	/** One label/value row of the read-only panels; nulls drop out. */
	function row(label: string, value: string | null | undefined): { label: string; value: string } | null {
		if (value === null || value === undefined || value === '') {
			return null;
		}
		return { label, value };
	}

	function rows(entries: Array<{ label: string; value: string | null | undefined }>): Array<{ label: string; value: string }> {
		return entries
			.map((entry) => row(entry.label, entry.value))
			.filter((entry): entry is { label: string; value: string } => entry !== null);
	}

	/** Render a nullable text field ('' and null both drop the row). */
	function text(value: string | null | undefined): string | null {
		return value === null || value === undefined || value === '' ? null : value;
	}

	/** Render a nullable numeric field with a prefix / suffix. */
	function number(value: number | null | undefined, prefix: string, suffix: string): string | null {
		return value === null || value === undefined ? null : `${prefix}${value}${suffix}`;
	}

	// The EXIF groups: the hot fields of the detail, each group
	// showing its empty note while it has nothing to show (the
	// pipeline has not written the extraction, or the file carries
	// no such tag -- the two are indistinguishable on the wire).
	const exif = $derived(detail.exif);
	const cameraRows = $derived(
		rows([
			{ label: $_('viewer_exif_camera_make'), value: text(exif?.cameraMake) },
			{ label: $_('viewer_exif_camera_model'), value: text(exif?.cameraModel) },
			{ label: $_('viewer_exif_lens'), value: text(exif?.lens) }
		])
	);
	const exposureRows = $derived(
		rows([
			{ label: $_('viewer_exif_iso'), value: number(exif?.iso, '', '') },
			{ label: $_('viewer_exif_aperture'), value: number(exif?.aperture, 'f/', '') },
			{ label: $_('viewer_exif_shutter'), value: text(exif?.shutter) },
			{ label: $_('viewer_exif_focal'), value: number(exif?.focal, '', ' mm') }
		])
	);
	const locationRows = $derived(
		rows([
			{ label: $_('viewer_exif_latitude'), value: number(exif?.lat, '', '') },
			{ label: $_('viewer_exif_longitude'), value: number(exif?.lon, '', '') },
			{ label: $_('viewer_exif_altitude'), value: number(exif?.alt, '', ' m') }
		])
	);

	const activeLocale = $derived($locale ?? 'en');

	/** Locale-aware capture time of the wire takenAt. */
	const capturedLabel = $derived(
		new Intl.DateTimeFormat(activeLocale, { dateStyle: 'medium', timeStyle: 'short' }).format(
			new Date(detail.takenAt)
		)
	);

	/** Locale-aware file size (decimal megabytes). */
	const sizeLabel = $derived(
		`${new Intl.NumberFormat(activeLocale, { maximumFractionDigits: 1 }).format(detail.fileSize / 1_000_000)} MB`
	);

	/** Locale-aware playback duration; null for images. */
	const durationLabel = $derived(
		detail.durationMs === null
			? null
			: `${new Intl.NumberFormat(activeLocale, { maximumFractionDigits: 1 }).format(detail.durationMs / 1000)} s`
	);

	const fileRows = $derived(
		rows([
			{ label: $_('viewer_file_name'), value: detail.originalFilename },
			{ label: $_('viewer_file_captured'), value: capturedLabel },
			{ label: $_('viewer_file_size'), value: sizeLabel },
			{
				label: $_('viewer_file_dimensions'),
				value:
					detail.width === null || detail.height === null
						? null
						: `${detail.width} × ${detail.height}`
			},
			{ label: $_('viewer_file_duration'), value: durationLabel }
		])
	);
</script>

<aside class="info" aria-label={$_('viewer_info_title')}>
	<header class="info__header">
		<h2 class="info__filename">{detail.originalFilename}</h2>
		<button class="info__close" type="button" aria-label={$_('common_close')} onclick={onclose}>
			<svg viewBox="0 0 24 24" width="20" height="20" fill="none" aria-hidden="true">
				<path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
			</svg>
		</button>
	</header>

	<div class="info__scroll">
		{#if detail.isTrashed}
			<p class="info__notice" role="status">{$_('viewer_trashed_notice')}</p>
		{/if}
		{#if detail.isHidden}
			<p class="info__notice" role="status">{$_('viewer_hidden_notice')}</p>
		{/if}

		<section class="info__section">
			<h3 class="info__heading">{$_('viewer_description_label')}</h3>
			{#if editing}
				<textarea
					class="info__editor"
					bind:value={draft}
					aria-label={$_('viewer_description_label')}
					rows="4"
					disabled={saving}
				></textarea>
				{#if editErrorKey !== ''}
					<p class="info__error" role="alert">{errorText(editErrorKey)}</p>
				{/if}
				<div class="info__row">
					<Button variant="primary" disabled={saving} onclick={() => void saveEdit()}>
						{$_('viewer_description_save')}
					</Button>
					<Button variant="ghost" disabled={saving} onclick={cancelEdit}>
						{$_('common_cancel')}
					</Button>
				</div>
			{:else}
				<p class="info__description">
					{detail.description !== '' ? detail.description : $_('viewer_description_empty')}
				</p>
				<Button variant="ghost" onclick={startEdit}>{$_('viewer_description_edit')}</Button>
			{/if}
		</section>

		<section class="info__section">
			<h3 class="info__heading">{$_('viewer_actions_label')}</h3>
			<div class="info__row">
				<Button
					variant="ghost"
					disabled={busy}
					onclick={() => void onaction(favorite ? 'unfavorite' : 'favorite')}
				>
					{favorite ? $_('action_unfavorite') : $_('action_favorite')}
				</Button>
				<Button
					variant="ghost"
					disabled={busy}
					onclick={() => void onaction(detail.isHidden ? 'unhide' : 'hide')}
				>
					{detail.isHidden ? $_('action_unhide') : $_('action_hide')}
				</Button>
				<Button
					variant="danger"
					disabled={busy}
					onclick={() => void onaction(detail.isTrashed ? 'restore' : 'trash')}
				>
					{detail.isTrashed ? $_('action_restore') : $_('action_trash')}
				</Button>
			</div>
			<Button variant="danger" disabled={busy} onclick={onrequestdelete}>
				{$_('viewer_delete_permanently')}
			</Button>
			{#if errorKey !== ''}
				<p class="info__error" role="alert">{errorText(errorKey)}</p>
			{/if}
		</section>

		<section class="info__section">
			<h3 class="info__heading">{$_('viewer_file_section')}</h3>
			<dl class="info__facts">
				{#each fileRows as entry (entry.label)}
					<div class="info__fact">
						<dt>{entry.label}</dt>
						<dd>{entry.value}</dd>
					</div>
				{/each}
			</dl>
			<a class="info__download" href={assetOriginalUrl(detail.id)} download>
				{$_('viewer_download')}
			</a>
		</section>

		<section class="info__section">
			<h3 class="info__heading">{$_('viewer_exif_heading')}</h3>
			<details class="info__group" open>
				<summary>{$_('viewer_exif_camera')}</summary>
				{#if cameraRows.length > 0}
					<dl class="info__facts">
						{#each cameraRows as entry (entry.label)}
							<div class="info__fact">
								<dt>{entry.label}</dt>
								<dd>{entry.value}</dd>
							</div>
						{/each}
					</dl>
				{:else}
					<p class="info__placeholder">{$_('viewer_exif_none')}</p>
				{/if}
			</details>
			<details class="info__group">
				<summary>{$_('viewer_exif_exposure')}</summary>
				{#if exposureRows.length > 0}
					<dl class="info__facts">
						{#each exposureRows as entry (entry.label)}
							<div class="info__fact">
								<dt>{entry.label}</dt>
								<dd>{entry.value}</dd>
							</div>
						{/each}
					</dl>
				{:else}
					<p class="info__placeholder">{$_('viewer_exif_none')}</p>
				{/if}
			</details>
			<details class="info__group">
				<summary>{$_('viewer_exif_location')}</summary>
				{#if locationRows.length > 0}
					<dl class="info__facts">
						{#each locationRows as entry (entry.label)}
							<div class="info__fact">
								<dt>{entry.label}</dt>
								<dd>{entry.value}</dd>
							</div>
						{/each}
					</dl>
				{:else}
					<p class="info__placeholder">{$_('viewer_exif_none')}</p>
				{/if}
			</details>
		</section>

		<section class="info__section">
			<h3 class="info__heading">{$_('nav_people')}</h3>
			<p class="info__placeholder">{$_('viewer_faces_placeholder')}</p>
		</section>

		<section class="info__section">
			<h3 class="info__heading">{$_('nav_places')}</h3>
			<p class="info__placeholder">{$_('viewer_places_placeholder')}</p>
		</section>
	</div>
</aside>

<style>
	.info {
		position: fixed;
		top: 0;
		right: 0;
		bottom: 0;
		z-index: 80;
		display: flex;
		flex-direction: column;
		width: min(360px, 92vw);
		background-color: var(--color-card);
		border-left: 1px solid var(--color-border);
		box-shadow: var(--shadow-overlay);
		animation: info-in var(--motion-base) var(--ease-standard);
	}

	.info__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
		padding: 14px 16px;
		border-bottom: 1px solid var(--color-border);
	}

	.info__filename {
		margin: 0;
		font: 600 15px/1.4 var(--font-body);
		/* Long names wrap instead of pushing the close mark out. */
		overflow-wrap: anywhere;
	}

	.info__close {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		width: 32px;
		height: 32px;
		border: none;
		border-radius: var(--radius-control);
		background: none;
		color: var(--color-text-soft);
		cursor: pointer;
		transition:
			background-color var(--motion-fast) var(--ease-standard),
			color var(--motion-fast) var(--ease-standard);
	}

	.info__close:hover {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}

	.info__scroll {
		flex: 1;
		overflow-y: auto;
		padding: 4px 16px 20px;
	}

	.info__notice {
		margin: 12px 0 0;
		padding: 8px 12px;
		font: 500 13px/1.4 var(--font-body);
		color: var(--color-text-soft);
		border: 1px solid var(--color-notice);
		border-radius: var(--radius-control);
	}

	.info__section {
		margin-top: 20px;
	}

	.info__heading {
		margin: 0 0 8px;
		font: 600 12px/1.4 var(--font-body);
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--color-text-faint);
	}

	.info__description {
		margin: 0 0 8px;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text);
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.info__editor {
		width: 100%;
		box-sizing: border-box;
		margin-bottom: 8px;
		padding: 9px 12px;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text);
		background-color: var(--color-page);
		border: 1px solid var(--color-border-strong);
		border-radius: var(--radius-control);
		resize: vertical;
	}

	.info__editor:focus {
		border-color: var(--color-accent);
	}

	.info__row {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-bottom: 8px;
	}

	.info__error {
		margin: 0 0 8px;
		font: 500 13px/1.4 var(--font-body);
		color: var(--color-danger-text);
	}

	.info__facts {
		margin: 0;
	}

	.info__fact {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		padding: 5px 0;
	}

	.info__fact dt {
		font: 400 13px/1.4 var(--font-body);
		color: var(--color-text-faint);
		white-space: nowrap;
	}

	.info__fact dd {
		margin: 0;
		font: 500 13px/1.4 var(--font-body);
		color: var(--color-text);
		text-align: right;
		overflow-wrap: anywhere;
	}

	.info__group {
		border: 1px solid var(--color-border);
		border-radius: var(--radius-control);
	}

	.info__group summary {
		padding: 8px 12px;
		font: 600 13px/1.4 var(--font-body);
		color: var(--color-text);
		cursor: pointer;
	}

	.info__group[open] summary {
		border-bottom: 1px solid var(--color-border);
	}

	.info__group .info__facts {
		padding: 4px 12px;
	}

	.info__download {
		display: inline-block;
		margin-top: 4px;
		padding: 8px 12px;
		font: 600 13px/1.4 var(--font-body);
		color: var(--color-link);
		text-decoration: none;
		border: 1px solid var(--color-border-strong);
		border-radius: var(--radius-control);
		transition: background-color var(--motion-fast) var(--ease-standard);
	}

	.info__download:hover {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}

	.info__placeholder {
		margin: 0;
		font: 400 13px/1.4 var(--font-body);
		color: var(--color-text-faint);
	}

	@keyframes info-in {
		from {
			transform: translateX(16px);
			opacity: 0;
		}
	}
</style>
