<script lang="ts">
	// The information panel of the asset viewer: METADATA ONLY (the
	// R2-3 rework) -- the description editor (the PATCH endpoint; the
	// drawer keeps the draft on failure so a retry never retypes
	// anything, and through the same-id detail replacements of the
	// optimistic curation flips and the SSE convergence), the People
	// section (one face chip per face of the detail's faces array,
	// the caller's label state per cluster, the click delegating to
	// the page's assign dialog), the EXIF hot fields folded into
	// collapsible groups, the file facts with the tags row (the
	// caller's own tags as chips plus the assign affordance, M3d),
	// and the places placeholder pinned to its M3 domain. The
	// curation actions and the permanent delete are GONE from this
	// surface: the action bar over the media carries the favorite
	// and the trash/restore pair, and the permanent delete lives on
	// the trash page alone.
	//
	// THE FORM (R2-10): the same markup serves the two shapes the
	// viewer composes it into -- the right DOCKED split region at lg+
	// (the viewer's flex row) and the fixed overlay drawer below lg.
	// Every class below is shared; the form lives in the aside's
	// lg:* utilities alone.
	//
	// THE VIEWER'S READ-ONLY DRAWER (the M3c identity seam): a caller
	// who does not own the asset (the owner shared it to them) keeps
	// every READ and loses every owner-only write affordance -- the
	// description's Edit button (the stored text renders) never
	// renders for them. Read-only is visual, never grayed out (the
	// web IA §3's pinned rule) -- the button simply never renders.
	//
	// THE ROW LANGUAGE (R2-4): key-value rows -- the label column in
	// the faint text tone, the value in the text tone, one hairline
	// between the rows (divide-y), the grouped sections under the
	// existing uppercase micro-heads, tightened vertical rhythm per
	// brand spec §5 (no heavy borders, feather shadows): the EXIF
	// groups dropped their card borders for the same row rhythm.
	//
	// The drawer is presentation and edit-mode state only: the page
	// owns the detail and every network call, through the
	// onedit / onassignface callbacks. Escape does not close it from
	// here -- the viewer owns the layered Escape rule (the page's
	// dialog first, then this drawer, then the route) so the layers
	// cannot race.
	import { locale, _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import { type AssetDetail, type AssetFaceSummary } from '$lib/api/assets';
	import { resolveErrorText } from '$lib/i18n/errors';
	import X from '@lucide/svelte/icons/x';
	import Check from '@lucide/svelte/icons/check';
	import PenLine from '@lucide/svelte/icons/pen-line';

	type Props = {
		/** The current asset's detail (the page's local, optimistic copy). */
		detail: AssetDetail;
		/**
		 * Whether the caller owns this asset: the description's edit
		 * affordance renders for the owner alone (the identity seam's
		 * write arm; the curation pair lives in the viewer's action
		 * bar, gated there).
		 */
		isOwner: boolean;
		/** Close the drawer. */
		onclose: () => void;
		/** Save the description; the outcome decides the edit mode. */
		onedit: (text: string) => Promise<{ ok: boolean; errorKey: string }>;
		/**
		 * Open the assign dialog for one face of the People section (the
		 * page owns the dialog and the writes).
		 */
		onassignface: (face: AssetFaceSummary) => void;
		/**
		 * Open the add-tag dialog over this asset (the page owns the
		 * dialog and the writes). Tags are per-user state (B4): every
		 * caller tags with their own vocabulary, so the affordance is
		 * every viewer's, never the owner's alone.
		 */
		onassigntag: () => void;
	};

	let { detail, isOwner, onclose, onedit, onassignface, onassigntag }: Props = $props();

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

	function rows(
		entries: Array<{ label: string; value: string | null | undefined }>
	): Array<{ label: string; value: string }> {
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
				value: detail.width === null || detail.height === null ? null : `${detail.width} × ${detail.height}`
			},
			{ label: $_('viewer_file_duration'), value: durationLabel }
		])
	);

	/**
	 * The chip label of one face: the caller's own label of the face's
	 * cluster, the unnamed state of a cluster the caller has not
	 * labeled, or the not-grouped state of a face the clustering has
	 * not assigned yet (the state the assign flow's honest 404 also
	 * explains).
	 */
	function faceLabel(face: AssetFaceSummary): string {
		if (face.label !== null) {
			return face.label.name;
		}
		return face.clusterId !== null ? $_('viewer_faces_unnamed') : $_('viewer_faces_unclustered');
	}
</script>

<!-- The docked split region (R2-10): at lg+ the panel is STATIC in
     the viewer's flex row -- it occupies layout space beside the
     media column (the stage re-fits through the flow, the width is
     the same min(360px,92vw) family, no floating shadow: the
     border-l hairline separates it). Below lg the same element keeps
     the overlay-drawer form: fixed, full-height over the right edge,
     with the floating shadow the overlay language carries. -->
<aside
	class="info fixed top-0 right-0 bottom-0 z-(--z-aside) flex flex-col w-[min(360px,92vw)] bg-card border-l border-border shadow-overlay lg:static lg:shrink-0 lg:shadow-none"
	aria-label={$_('viewer_info_title')}
>
	<header class="info__header flex items-center justify-between gap-2 px-4 py-3.5 border-b border-border">
		<h2 class="info__filename font-body font-semibold text-[15px] leading-[1.4] [overflow-wrap:anywhere]">
			{detail.originalFilename}
		</h2>
		<button
			class="info__close inline-flex items-center justify-center shrink-0 w-8 h-8 rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
			type="button"
			aria-label={$_('common_close')}
			onclick={onclose}
		>
			<X size={20} />
		</button>
	</header>

	<div class="info__scroll flex-1 overflow-y-auto [scrollbar-gutter:stable] pt-1 px-4 pb-5">
		{#if detail.isTrashed}
			<p
				class="info__notice mt-3 py-2 px-3 font-body font-medium text-[13px] leading-[1.4] text-text-soft border border-notice rounded-control"
				role="status"
			>
				{$_('viewer_trashed_notice')}
			</p>
		{/if}
		{#if detail.isHidden}
			<p
				class="info__notice mt-3 py-2 px-3 font-body font-medium text-[13px] leading-[1.4] text-text-soft border border-notice rounded-control"
				role="status"
			>
				{$_('viewer_hidden_notice')}
			</p>
		{/if}

		<section class="info__section mt-4">
			<h3
				class="info__heading mb-1 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint"
			>
				{$_('viewer_description_label')}
			</h3>
			{#if editing}
				<textarea
					class="info__editor w-full mb-2 px-3 py-[9px] font-body font-normal text-[14px] leading-[1.5] text-text bg-page border border-border-strong rounded-control resize-y focus:border-accent"
					bind:value={draft}
					aria-label={$_('viewer_description_label')}
					rows="4"
					disabled={saving}></textarea>
				{#if editErrorKey !== ''}
					<p
						class="info__error mb-2 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
						role="alert"
					>
						{errorText(editErrorKey)}
					</p>
				{/if}
				<div class="info__row flex flex-wrap gap-2 mb-2">
					<Button variant="primary" disabled={saving} onclick={() => void saveEdit()}>
						<Check size={20} aria-hidden="true" />
						{$_('viewer_description_save')}
					</Button>
					<Button variant="ghost" disabled={saving} onclick={cancelEdit}>
						<X size={20} aria-hidden="true" />
						{$_('common_cancel')}
					</Button>
				</div>
			{:else}
				<p
					class="info__description mb-2 font-body font-normal text-[14px] leading-[1.5] text-text whitespace-pre-wrap [overflow-wrap:anywhere]"
				>
					{detail.description !== '' ? detail.description : $_('viewer_description_empty')}
				</p>
				{#if isOwner}
					<Button variant="ghost" onclick={startEdit}
						><PenLine size={20} aria-hidden="true" />{$_('viewer_description_edit')}</Button
					>
				{/if}
			{/if}
		</section>

		<!-- The key-value row language (R2-4): one hairline between
		     the rows, the label column faint, the value column text,
		     the tightened section rhythm -- no card borders, the
		     section's own spacing is the grouping. -->
		<section class="info__section mt-4">
			<h3
				class="info__heading mb-1 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint"
			>
				{$_('viewer_file_section')}
			</h3>
			<dl class="info__facts divide-y divide-border">
				{#each fileRows as entry (entry.label)}
					<div class="info__fact flex justify-between gap-3 py-[7px]">
						<dt class="font-body font-normal text-[13px] leading-[1.4] text-text-faint whitespace-nowrap">
							{entry.label}
						</dt>
						<dd
							class="font-body font-medium text-[13px] leading-[1.4] text-text text-right [overflow-wrap:anywhere]"
						>
							{entry.value}
						</dd>
					</div>
				{/each}
				<!-- The tags row (M3d): the CALLER's tags on the asset as
				     chips, plus the assign affordance (the add-tag dialog
				     the batch bar's flow reuses). Per-user state (B4): the
				     chips are the caller's own rows, the write is the
				     caller's own, so the affordance is every viewer's. -->
				<div class="info__tags flex items-center justify-between gap-3 py-[7px]">
					<dt class="font-body font-normal text-[13px] leading-[1.4] text-text-faint whitespace-nowrap">
						{$_('viewer_tags_label')}
					</dt>
					<dd class="info__tag-values flex flex-wrap justify-end items-center gap-1.5 m-0">
						{#if detail.tags.length === 0}
							<span
								class="info__tags-empty font-body font-normal text-[13px] leading-[1.4] text-text-faint"
								>{$_('viewer_tags_empty')}</span
							>
						{:else}
							{#each detail.tags as tag (tag.id)}
								<span
									class="info__tag-chip inline-flex items-center px-2 py-0.5 font-body font-medium text-[12px] leading-[1.4] text-text bg-transparent border border-border rounded-full whitespace-nowrap"
									data-tag-id={tag.id}>{tag.name}</span
								>
							{/each}
						{/if}
						<button
							class="info__tag-add inline-flex items-center gap-1 px-2 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-soft bg-transparent border border-border rounded-full cursor-pointer whitespace-nowrap transition-[background-color,border-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:border-accent-soft-text hover:text-accent-soft-text"
							type="button"
							onclick={onassigntag}
						>
							<svg viewBox="0 0 24 24" width="12" height="12" fill="none" aria-hidden="true">
								<path
									d="M12 5v14M5 12h14"
									stroke="currentColor"
									stroke-width="2"
									stroke-linecap="round"
								/>
							</svg>
							{$_('action_add_tag')}
						</button>
					</dd>
				</div>
			</dl>
		</section>

		<section class="info__section mt-4">
			<h3
				class="info__heading mb-1 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint"
			>
				{$_('viewer_exif_heading')}
			</h3>
			<!-- The EXIF groups: collapsible summaries over the same
			     hairline row rhythm (the card borders of the old
			     presentation are gone -- brand spec §5's no heavy
			     borders). -->
			<details class="info__group" open>
				<summary
					class="info__group-summary py-[7px] font-body font-semibold text-[13px] leading-[1.4] text-text cursor-pointer"
					>{$_('viewer_exif_camera')}</summary
				>
				{#if cameraRows.length > 0}
					<dl class="info__facts divide-y divide-border">
						{#each cameraRows as entry (entry.label)}
							<div class="info__fact flex justify-between gap-3 py-[7px]">
								<dt
									class="font-body font-normal text-[13px] leading-[1.4] text-text-faint whitespace-nowrap"
								>
									{entry.label}
								</dt>
								<dd
									class="font-body font-medium text-[13px] leading-[1.4] text-text text-right [overflow-wrap:anywhere]"
								>
									{entry.value}
								</dd>
							</div>
						{/each}
					</dl>
				{:else}
					<p
						class="info__placeholder py-[7px] font-body font-normal text-[13px] leading-[1.4] text-text-faint"
					>
						{$_('viewer_exif_none')}
					</p>
				{/if}
			</details>
			<details class="info__group">
				<summary
					class="info__group-summary py-[7px] font-body font-semibold text-[13px] leading-[1.4] text-text cursor-pointer"
					>{$_('viewer_exif_exposure')}</summary
				>
				{#if exposureRows.length > 0}
					<dl class="info__facts divide-y divide-border">
						{#each exposureRows as entry (entry.label)}
							<div class="info__fact flex justify-between gap-3 py-[7px]">
								<dt
									class="font-body font-normal text-[13px] leading-[1.4] text-text-faint whitespace-nowrap"
								>
									{entry.label}
								</dt>
								<dd
									class="font-body font-medium text-[13px] leading-[1.4] text-text text-right [overflow-wrap:anywhere]"
								>
									{entry.value}
								</dd>
							</div>
						{/each}
					</dl>
				{:else}
					<p
						class="info__placeholder py-[7px] font-body font-normal text-[13px] leading-[1.4] text-text-faint"
					>
						{$_('viewer_exif_none')}
					</p>
				{/if}
			</details>
			<details class="info__group">
				<summary
					class="info__group-summary py-[7px] font-body font-semibold text-[13px] leading-[1.4] text-text cursor-pointer"
					>{$_('viewer_exif_location')}</summary
				>
				{#if locationRows.length > 0}
					<dl class="info__facts divide-y divide-border">
						{#each locationRows as entry (entry.label)}
							<div class="info__fact flex justify-between gap-3 py-[7px]">
								<dt
									class="font-body font-normal text-[13px] leading-[1.4] text-text-faint whitespace-nowrap"
								>
									{entry.label}
								</dt>
								<dd
									class="font-body font-medium text-[13px] leading-[1.4] text-text text-right [overflow-wrap:anywhere]"
								>
									{entry.value}
								</dd>
							</div>
						{/each}
					</dl>
				{:else}
					<p
						class="info__placeholder py-[7px] font-body font-normal text-[13px] leading-[1.4] text-text-faint"
					>
						{$_('viewer_exif_none')}
					</p>
				{/if}
			</details>
		</section>

		<section class="info__section mt-4">
			<h3
				class="info__heading mb-1 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint"
			>
				{$_('nav_people')}
			</h3>
			{#if detail.isTrashed || detail.isHidden || detail.faces.length === 0}
				<!-- A trashed or hidden asset keeps its faces in the
				     detail but every crop endpoint answers 404 for it
				     -- the placeholder stands in rather than broken
				     chips. -->
				<p
					class="info__faces-empty info__placeholder py-[7px] font-body font-normal text-[13px] leading-[1.4] text-text-faint"
				>
					{$_('viewer_faces_placeholder')}
				</p>
			{:else}
				<!-- One chip per face of the detail's faces array: the crop,
				     the caller's label state of the face's cluster, and the
				     assign affordance -- the click opens the page's assign
				     dialog. -->
				<ul class="info__faces flex flex-wrap gap-2 m-0 p-0 list-none">
					{#each detail.faces as face (face.id)}
						<li>
							<button
								class="info__face flex items-center gap-2 py-1 pl-1 pr-2.5 rounded-control bg-transparent border border-border cursor-pointer transition-[background-color,border-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:border-accent-soft-text focus-visible:bg-accent-soft focus-visible:border-accent-soft-text"
								type="button"
								data-face-id={face.id}
								onclick={() => onassignface(face)}
							>
								<img
									class="info__face-crop w-9 h-9 rounded-control object-cover bg-accent-soft"
									src={face.thumbnail}
									alt=""
									loading="lazy"
									decoding="async"
									draggable="false"
								/>
								<span
									class="info__face-name max-w-[140px] font-body font-medium text-[13px] leading-[1.4] text-text truncate"
									>{faceLabel(face)}</span
								>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="info__section mt-4">
			<h3
				class="info__heading mb-1 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint"
			>
				{$_('nav_places')}
			</h3>
			<p class="info__placeholder py-[7px] font-body font-normal text-[13px] leading-[1.4] text-text-faint">
				{$_('viewer_places_placeholder')}
			</p>
		</section>
	</div>
</aside>

<style>
	/* Residual styles (verbatim): the slide-in keyframes and the
	 * drawer's animation declaration. */
	.info {
		animation: info-in var(--motion-base) var(--ease-standard);
	}

	@keyframes info-in {
		from {
			transform: translateX(16px);
			opacity: 0;
		}
	}
</style>
