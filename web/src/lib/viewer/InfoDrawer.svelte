<script lang="ts">
	// The information panel of the asset viewer: the FOUR SECTIONS the
	// design pins -- the description (one line, click to edit), the
	// people (the chips, the two-way linked ends of the on-photo face
	// boxes, the owner's add-person icon and the edit pencil), the
	// info rows (icon + content, one per line: the capture time with
	// its zone, the size and file weight, the camera line, the tag
	// chips with their remove buttons, the album links) -- and the
	// location row (the plain place name, the 5:4 static mini map,
	// the coordinates), the admin-only path row between the albums
	// and the location.
	//
	// THE EDIT ARMS: every edit affordance in this panel delegates --
	// the description edit is the panel's own (the onedit callback),
	// while the dialogs (the date, the location, the add-tag) and the
	// writes (the tag removal, the drawn face) belong to the page,
	// which owns the network and the convergence. The pencil is the
	// OWNER's arm: a viewer the owner shared to keeps every read and
	// loses every owner-only write affordance, rendered as absence,
	// never grayed out.
	//
	// THE LINKAGE: the chips and the on-photo boxes share one signal
	// (./face-link.svelte.ts) -- hovering or focusing either end
	// highlights both, with the hide grace that lets the pointer
	// travel between them.
	import { locale, _ } from 'svelte-i18n';
	import { formatAge } from '$lib/people/age';
	import { toast } from '$lib/components/toast.svelte';
	import { type AssetAlbumRef, type AssetDetail, type AssetFaceSummary, type AssetPlaceRef } from '$lib/api/assets';
	import { linkFace, unlinkFace, isLinked } from './face-link.svelte';
	import { formatTakenAt, zoneLabel } from './zone';
	import { resolveErrorText } from '$lib/i18n/errors';
	import MapView from '$lib/map/MapView.svelte';
	import X from '@lucide/svelte/icons/x';
	import PenLine from '@lucide/svelte/icons/pen-line';
	import Folder from '@lucide/svelte/icons/folder';
	import Tag from '@lucide/svelte/icons/tag';
	import Camera from '@lucide/svelte/icons/camera';
	import Calendar from '@lucide/svelte/icons/calendar';
	import Ratio from '@lucide/svelte/icons/ratio';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import HardDrive from '@lucide/svelte/icons/hard-drive';
	import Copy from '@lucide/svelte/icons/copy';
	import Plus from '@lucide/svelte/icons/plus';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';

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
		 * dialog and the writes). Tags are per-user state: every
		 * caller tags with their own vocabulary, so the affordance is
		 * every viewer's, never the owner's alone.
		 */
		onassigntag: () => void;
		/** Remove one of the caller's tags from this asset (the page owns the write). */
		onremovetag: (tagId: string) => void;
		/**
		 * Open the edit-people panel (the page owns the panel's flow
		 * and writes; the panel replaces this drawer in the viewer's
		 * composition). The face-level correction writes are the
		 * owner's curation, so the affordance is the owner's alone --
		 * a viewer the owner shared to keeps the read-only drawer.
		 */
		oneditfaces: () => void;
		/**
		 * Start the drawn-face mode: the stage's drag layer arms (the
		 * page owns the POST and the assign flow that follows). The
		 * drawn face is the owner's curation, so the icon is the
		 * owner's alone.
		 */
		onstartdraw: () => void;
		/** Open the edit-date dialog (the page owns it and the write). */
		onedittime: () => void;
		/** Open the edit-location dialog (the page owns it and the write). */
		oneditlocation: () => void;
	};

	let {
		detail,
		isOwner,
		onclose,
		onedit,
		onassignface,
		onassigntag,
		onremovetag,
		oneditfaces,
		onstartdraw,
		onedittime,
		oneditlocation
	}: Props = $props();

	// The path row's open state (default collapsed; the row's
	// existence IS the admin gate -- the detail serves the field to
	// administrators alone).
	let pathOpen = $state(false);

	// THE DESCRIPTION: one line that becomes an input on click. Enter
	// or blur saves (an unchanged value exits silently -- no request,
	// no toast), Esc reverts. A failure keeps the edit mode and the
	// draft (a retry never retypes), and the stored description never
	// changed underneath it.
	let editing = $state(false);
	let draft = $state('');
	let saving = $state(false);
	let editErrorKey = $state('');
	let editor: HTMLInputElement | undefined = $state();

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
		if (saving || !editing) {
			return;
		}
		if (draft === detail.description) {
			// The unchanged value: no request, no toast -- the exit is
			// the whole story.
			editing = false;
			return;
		}
		saving = true;
		editErrorKey = '';
		const outcome = await onedit(draft);
		saving = false;
		if (outcome.ok) {
			// The response is the fresh detail; the page adopted it.
			editing = false;
			toast($_('viewer_description_saved'), 'success');
			return;
		}
		// The failure keeps the edit mode and the draft (a retry
		// never retypes); the stored description never changed.
		editErrorKey = outcome.errorKey;
	}

	// The edit input focuses and selects all on entry (the click that
	// opened it aims at the whole line).
	$effect(() => {
		if (editing && editor) {
			editor.focus();
			editor.select();
		}
	});

	function handleEditorKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape') {
			// Revert: the draft dies with the edit mode, the stored
			// text stands.
			event.preventDefault();
			cancelEdit();
			return;
		}
		if (event.key === 'Enter') {
			// The single-line save; the input has no Shift+Enter to
			// tell apart (the description is one line, never multi).
			event.preventDefault();
			void saveEdit();
		}
	}

	// The shared unknown-code fallback (see $lib/i18n/errors): an
	// unknown error code has no i18n key, and svelte-i18n would echo
	// the key itself -- such codes render the generic message. The
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// The panel's editable surfaces share one curation gate: the
	// owner of a LIVE asset (a trashed or archived asset's faces' crop
	// endpoints answer 404, and the write arms refuse the same states
	// server-side -- the affordances never render for them).
	const canCurate = $derived(isOwner && !detail.isTrashed && !detail.isArchived);

	/**
	 * Whether the People section lists its faces (the chips' gate):
	 * a trashed or archived asset keeps its faces in the detail but
	 * every crop endpoint answers 404 for it, and a photo with no
	 * faces has nothing to list -- the placeholder stands in. The
	 * section's edit affordance (the pencil into the edit-people
	 * panel) rides the same gate: the panel lists these very rows.
	 */
	const facesListed = $derived(!detail.isTrashed && !detail.isArchived && detail.faces.length > 0);

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

	/**
	 * The capture-time age of one named face: the
	 * person's birth date against THIS photo's takenAt. Absent when
	 * the person carries no birth date, or the photo predates it --
	 * the chip then renders its name alone (never a zero).
	 */
	function faceAge(face: AssetFaceSummary): string | null {
		if (face.label === null) {
			return null;
		}
		return formatAge(face.birthDate ?? null, detail.takenAt, $_);
	}

	/**
	 * Whether one face chip deep-links into its person page: a NAMED
	 * cluster (the caller's label sits on the face's cluster) -- the
	 * naming is done, the page owns the rename. Unnamed faces keep
	 * the assign flow (the drawer's own write).
	 */
	function faceLinks(face: AssetFaceSummary): boolean {
		return face.clusterId !== null && face.label !== null;
	}

	/** The album row's display title (the sync card's localized display name). */
	function albumTitle(album: AssetAlbumRef): string {
		return album.type === 'sync' ? $_('album_all_photos') : album.title;
	}

	const activeLocale = $derived($locale ?? 'en');

	/** The capture time in its zone's wall clock, with the zone suffix. */
	const takenLine = $derived(formatTakenAt(detail.takenAt, detail.takenAtZone, activeLocale));
	const takenZone = $derived(detail.takenAtZone === null ? '' : zoneLabel(detail.takenAtZone));

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

	/** The size line: dimensions, weight, and (for videos) the length. */
	const sizeLine = $derived.by(() => {
		const parts: string[] = [];
		if (detail.width !== null && detail.height !== null) {
			parts.push(
				`${new Intl.NumberFormat(activeLocale).format(detail.width)} × ${new Intl.NumberFormat(activeLocale).format(detail.height)}`
			);
		}
		parts.push(sizeLabel);
		if (durationLabel !== null) {
			parts.push(durationLabel);
		}
		return parts.join(' · ');
	});

	/** The camera line's parts: camera, lens, focal, aperture, shutter, ISO. */
	const cameraLine = $derived.by(() => {
		const exif = detail.exif;
		const parts: string[] = [];
		const camera = [exif.cameraMake, exif.cameraModel]
			.filter((part) => part !== null)
			.join(' ')
			.trim();
		if (camera !== '') {
			parts.push(camera);
		}
		const lens = (exif.lens ?? '').trim();
		if (lens !== '') {
			parts.push(lens);
		}
		if (exif.focal !== null) {
			parts.push(`${$_('viewer_info_focal', { values: { value: exif.focal } })}`);
		}
		if (exif.aperture !== null) {
			parts.push(`f/${exif.aperture}`);
		}
		if (exif.shutter !== null) {
			parts.push($_('viewer_info_shutter_value', { values: { value: exif.shutter } }));
		}
		if (exif.iso !== null) {
			parts.push($_('viewer_info_iso', { values: { value: exif.iso } }));
		}
		return parts.join(' · ');
	});

	const place: AssetPlaceRef | null = $derived(detail.place);

	// The photo's own position: the manual edit when one landed, the
	// camera's GPS otherwise (the detail's location face already
	// made the choice).
	const coordinates = $derived(detail.location);

	/** The altitude line under the mini map (the camera's own fact). */
	const altitudeLabel = $derived(
		detail.exif.alt === null
			? ''
			: $_('viewer_info_altitude', {
					values: {
						value: new Intl.NumberFormat(activeLocale, { maximumFractionDigits: 0 }).format(detail.exif.alt)
					}
				})
	);

	/** The path row's copy: the clipboard write and its toast. */
	async function copyPath(): Promise<void> {
		if (detail.path === null) {
			return;
		}
		try {
			await navigator.clipboard.writeText(detail.path);
			toast($_('viewer_path_copied'), 'success');
		} catch {
			toast(resolveErrorText($_, 'internal'), 'danger');
		}
	}
</script>

<!-- The docked split region: at sm+ (640) the panel is STATIC in
     the viewer's flex row -- it occupies layout space beside the
     media column (the stage re-fits through the flow), the pinned
     fixed 360px, no breakpoint above it. Below sm the same element
     keeps the BOTTOM DRAWER form: fixed to the bottom edge, full
     width, over a scrim (the one deviation the design pins -- 360
     viewport against 360 sidebar would zero the stage). -->
<aside
	id="viewer-info"
	class="info fixed inset-x-0 bottom-0 top-[14vh] z-(--z-modal) flex w-full flex-col rounded-t-overlay border-t border-border bg-card shadow-overlay sm:static sm:top-auto sm:bottom-auto sm:z-(--z-aside) sm:w-[360px] sm:shrink-0 sm:rounded-none sm:border-t-0 sm:border-l sm:shadow-none"
	aria-label={$_('viewer_info_title')}
>
	<span class="mx-auto mt-2 mb-1 h-1 w-10 shrink-0 rounded-full bg-border-strong sm:hidden" aria-hidden="true"></span>
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
		{#if detail.isArchived}
			<p
				class="info__notice mt-3 py-2 px-3 font-body font-medium text-[13px] leading-[1.4] text-text-soft border border-notice rounded-control"
				role="status"
			>
				{$_('viewer_archived_notice')}
			</p>
		{/if}

		<!-- THE DESCRIPTION: the first section, one line that becomes
		     an input on click (the owner's edit arm; a shared viewer
		     reads the stored text). -->
		<section class="info__section mt-4">
			<h3
				class="info__heading mb-1 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint"
			>
				{$_('viewer_description_label')}
			</h3>
			{#if editing}
				<!-- The contract's 10,000-rune ceiling on the description,
				     carried natively: the editor stops the typing where the
				     server would refuse the write. -->
				<input
					class="info__editor w-full px-2.5 py-1.5 font-body font-normal text-[14px] leading-[1.5] text-text bg-page border border-accent rounded-control"
					type="text"
					bind:this={editor}
					bind:value={draft}
					aria-label={$_('viewer_description_label')}
					placeholder={$_('viewer_description_empty')}
					maxlength={10000}
					disabled={saving}
					onkeydown={handleEditorKeydown}
					onblur={() => void saveEdit()}
				/>
				{#if editErrorKey !== ''}
					<p
						class="info__error mt-2 m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
						role="alert"
					>
						{errorText(editErrorKey)}
					</p>
				{/if}
			{:else if isOwner}
				{#if detail.description !== ''}
					<button
						class="info__description block w-full border-0 bg-transparent p-0 py-1.5 text-left font-body font-normal text-[14px] leading-[1.5] text-text truncate cursor-text rounded-control transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
						type="button"
						title={detail.description}
						onclick={startEdit}
					>
						{detail.description}
					</button>
				{:else}
					<button
						class="info__description block w-full border-0 bg-transparent p-0 py-1.5 text-left font-body font-normal text-[14px] leading-[1.5] text-text-faint cursor-text rounded-control transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
						type="button"
						onclick={startEdit}
					>
						{$_('viewer_description_empty')}
					</button>
				{/if}
			{:else}
				<p
					class="info__description m-0 py-1.5 font-body font-normal text-[14px] leading-[1.5] {detail.description !==
					''
						? 'text-text'
						: 'text-text-faint'} truncate"
					title={detail.description}
				>
					{detail.description !== '' ? detail.description : $_('viewer_description_empty')}
				</p>
			{/if}
		</section>

		<!-- THE PEOPLE: the chips are the panel's end of the two-way
		     linkage -- hovering or focusing a chip highlights its box on
		     the photo, and a box hover highlights the chip back. The
		     heading row carries the owner's add-person icon (the drawn
		     face) and the edit pencil. -->
		<section class="info__section mt-4">
			<div class="info__heading-row flex items-center justify-between gap-2 mb-1">
				<h3
					class="info__heading m-0 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint"
				>
					{$_('nav_people')}
				</h3>
				{#if canCurate}
					<div class="flex items-center gap-0.5">
						<button
							class="info__faces-add inline-flex items-center justify-center w-7 h-7 shrink-0 rounded-control bg-transparent text-text-faint cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
							type="button"
							aria-label={$_('viewer_faces_add')}
							title={$_('viewer_faces_add')}
							onclick={onstartdraw}
						>
							<Plus size={18} />
						</button>
						{#if facesListed}
							<button
								class="info__faces-edit inline-flex items-center justify-center w-7 h-7 shrink-0 rounded-control bg-transparent text-text-faint cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
								type="button"
								aria-label={$_('people_edit_faces')}
								title={$_('people_edit_faces')}
								onclick={oneditfaces}
							>
								<PenLine size={18} />
							</button>
						{/if}
					</div>
				{/if}
			</div>
			{#if detail.isTrashed || detail.isArchived || detail.faces.length === 0}
				<!-- A trashed or archived asset keeps its faces in the
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
				     affordance -- a NAMED cluster's chip deep-links into the
				     person page; an unnamed chip keeps the assign flow
				     (the naming write is the drawer's own, never a
				     navigation). -->
				<ul class="info__faces flex flex-wrap gap-2 m-0 p-0 list-none">
					{#each detail.faces as face (face.id)}
						<li>
							{#if faceLinks(face)}
								<a
									class="info__face flex items-center gap-2 py-1 pl-1 pr-2.5 rounded-control bg-transparent border cursor-pointer no-underline transition-[background-color,border-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:border-accent-soft-text focus-visible:bg-accent-soft focus-visible:border-accent-soft-text {isLinked(
										face.id
									)
										? 'bg-accent-soft border-accent'
										: 'border-border'}"
									href={`/people/${encodeURIComponent(face.clusterId ?? '')}`}
									data-face-id={face.id}
									onmouseenter={() => linkFace(face.id)}
									onmouseleave={() => unlinkFace(face.id)}
									onfocus={() => linkFace(face.id)}
									onblur={() => unlinkFace(face.id)}
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
									{#if faceAge(face) !== null}
										<span
											class="info__face-age flex-none font-body font-normal text-[12px] leading-[1.4] text-text-soft tabular-nums"
											>{faceAge(face)}</span
										>
									{/if}
								</a>
							{:else}
								<button
									class="info__face flex items-center gap-2 py-1 pl-1 pr-2.5 rounded-control bg-transparent border cursor-pointer transition-[background-color,border-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:border-accent-soft-text focus-visible:bg-accent-soft focus-visible:border-accent-soft-text {isLinked(
										face.id
									)
										? 'bg-accent-soft border-accent'
										: 'border-border'}"
									type="button"
									data-face-id={face.id}
									onmouseenter={() => linkFace(face.id)}
									onmouseleave={() => unlinkFace(face.id)}
									onfocus={() => linkFace(face.id)}
									onblur={() => unlinkFace(face.id)}
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
										class="info__face-name max-w-[140px] font-body font-medium text-[13px] leading-[1.4] text-text truncate {face.label ===
										null
											? 'text-text-soft'
											: ''}">{faceLabel(face)}</span
									>
									{#if faceAge(face) !== null}
										<span
											class="info__face-age flex-none font-body font-normal text-[12px] leading-[1.4] text-text-soft tabular-nums"
											>{faceAge(face)}</span
										>
									{/if}
								</button>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<!-- THE INFO: icon + content, one row per line -- the capture
		     time (its zone shown), the size and weight, the camera
		     line, the tag chips, the album links, the admin-only path
		     and the location with its mini map. No filename row (the
		     panel header carries it), no key-value columns. -->
		<section class="info__section mt-4">
			<h3
				class="info__heading mb-1 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint"
			>
				{$_('viewer_info_section')}
			</h3>
			<ul class="info__facts m-0 p-0 list-none divide-y divide-border">
				<li class="info__fact flex items-center gap-2 py-[7px]">
					<Calendar size={16} class="shrink-0 text-text-faint" aria-hidden="true" />
					<span
						class="min-w-0 flex-1 [overflow-wrap:anywhere] font-body font-normal text-[13px] leading-[1.4] text-text"
					>
						{takenLine}
						{#if takenZone !== ''}
							<span class="text-text-faint tabular-nums">· {takenZone}</span>
						{/if}
					</span>
					{#if isOwner}
						<button
							class="flex-none inline-flex items-center justify-center w-7 h-7 rounded-control bg-transparent text-text-faint cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
							type="button"
							aria-label={$_('viewer_edit_date_open')}
							title={$_('viewer_edit_date_open')}
							onclick={onedittime}
						>
							<PenLine size={14} />
						</button>
					{/if}
				</li>
				<li class="info__fact flex items-center gap-2 py-[7px]">
					<Ratio size={16} class="shrink-0 text-text-faint" aria-hidden="true" />
					<span
						class="min-w-0 flex-1 [overflow-wrap:anywhere] font-body font-normal text-[13px] leading-[1.4] text-text"
					>
						{sizeLine}
					</span>
				</li>
				{#if cameraLine !== ''}
					<li class="info__fact flex items-center gap-2 py-[7px]">
						<Camera size={16} class="shrink-0 text-text-faint" aria-hidden="true" />
						<span
							class="min-w-0 flex-1 [overflow-wrap:anywhere] font-body font-normal text-[13px] leading-[1.4] text-text"
						>
							{cameraLine}
						</span>
					</li>
				{/if}
				<li class="info__fact flex items-start gap-2 py-[7px]">
					<Tag size={16} class="shrink-0 mt-0.5 text-text-faint" aria-hidden="true" />
					<span class="min-w-0 flex-1 flex flex-wrap items-center gap-1.5">
						{#if detail.tags.length === 0}
							<span
								class="info__tags-empty font-body font-normal text-[13px] leading-[1.4] text-text-faint"
								>{$_('viewer_tags_empty')}</span
							>
						{:else}
							{#each detail.tags as tag (tag.id)}
								<!-- The chip is a span holding TWO independent
								     controls: the name is the link into the
								     tag-filtered browse, the × only removes --
								     interactive elements never nest. -->
								<span
									class="info__tag-chip inline-flex items-center gap-0.5 ps-2 pe-0.5 py-0.5 border border-border rounded-full bg-transparent whitespace-nowrap transition-[background-color,border-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:border-accent-soft-text"
									data-tag-id={tag.id}
								>
									<a
										class="font-body font-medium text-[12px] leading-[1.4] text-text no-underline transition-[color] duration-(--motion-fast) ease-standard hover:text-accent-soft-text"
										href={`/search?tagIds=${encodeURIComponent(tag.id)}`}
									>
										{tag.name}
									</a>
									<button
										class="inline-flex items-center justify-center w-4 h-4 rounded-full bg-transparent border-0 text-text-faint cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-[color-mix(in_srgb,var(--color-danger)_18%,transparent)] hover:text-danger-text focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
										type="button"
										aria-label={$_('viewer_tag_remove', { values: { name: tag.name } })}
										onclick={() => onremovetag(tag.id)}
									>
										<X size={10} aria-hidden="true" />
									</button>
								</span>
							{/each}
						{/if}
						<button
							class="info__tag-add inline-flex items-center gap-1 px-2 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-soft bg-transparent border border-border rounded-full cursor-pointer whitespace-nowrap transition-[background-color,border-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:border-accent-soft-text hover:text-accent-soft-text focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
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
					</span>
				</li>
				{#if detail.albums.length > 0}
					<!-- The albums row: the one road back from a photo to the
					     albums that hold it -- and the viewer opens them in a
					     NEW TAB, so the open photo survives the trip. -->
					<li class="info__fact flex items-center gap-2 py-[7px]">
						<Folder size={16} class="shrink-0 text-text-faint" aria-hidden="true" />
						<span class="min-w-0 flex-1 flex flex-wrap items-center gap-x-2 gap-y-1">
							{#each detail.albums as album (album.id)}<a
									class="info__album inline-flex items-center font-body font-medium text-[13px] leading-none text-accent no-underline hover:underline focus-visible:underline"
									href={album.type === 'memory'
										? `/memories?memory=${encodeURIComponent(album.id)}`
										: `/albums/${encodeURIComponent(album.id)}`}
									target="_blank"
									rel="noopener"
									data-album-id={album.id}
								>
									{albumTitle(album)}
								</a>
								{#if album !== detail.albums[detail.albums.length - 1]}
									<span class="text-text-faint" aria-hidden="true">·</span>
								{/if}
							{/each}
						</span>
					</li>
				{/if}
				{#if detail.path !== null}
					<!-- The path row: admin-only (the detail serves the field
					     to administrators alone, so the row's existence IS the
					     gate), default collapsed, the mono copy beside a copy
					     button. -->
					<li class="info__fact py-[7px]" data-testid="viewer-path">
						<button
							class="flex w-full items-center gap-2 border-0 bg-transparent p-0 text-left cursor-pointer focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
							type="button"
							aria-expanded={pathOpen}
							aria-controls="asset-path"
							onclick={() => (pathOpen = !pathOpen)}
						>
							<HardDrive size={16} class="shrink-0 text-text-faint" aria-hidden="true" />
							<span class="min-w-0 flex-1 font-body font-normal text-[13px] leading-[1.4] text-text-soft">
								{$_('viewer_path_label')}
							</span>
							<ChevronDown
								size={14}
								class="shrink-0 text-text-faint transition-transform duration-(--motion-fast) ease-standard {pathOpen
									? 'rotate-180'
									: ''}"
								aria-hidden="true"
							/>
						</button>
						{#if pathOpen}
							<div class="mt-1.5 flex items-start gap-2" id="asset-path">
								<!-- Two lines at most: the path breaks at any
								     character (the filename is the half worth
								     reading, never the half cut off). -->
								<span
									class="info__path min-w-0 flex-1 line-clamp-2 break-all font-mono font-normal text-[12px] leading-[1.5] text-text-soft"
									>{detail.path}</span
								>
								<button
									class="flex-none inline-flex items-center justify-center w-7 h-7 rounded-control bg-transparent text-text-faint cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
									type="button"
									aria-label={$_('viewer_path_copy')}
									title={$_('viewer_path_copy')}
									onclick={() => void copyPath()}
								>
									<Copy size={14} />
								</button>
							</div>
						{/if}
					</li>
				{/if}
				{#if coordinates !== null}
					<!-- The location row: the LAST item of the info -- the
					     plain place name (no link), the owner's pencil, the
					     5:4 static mini map (display only: no drag, no zoom,
					     no click) and the coordinates line. -->
					<li class="info__fact py-[7px]" data-testid="viewer-location">
						<div class="flex items-center gap-2">
							<MapPin size={16} class="shrink-0 text-text-faint" aria-hidden="true" />
							<span
								class="min-w-0 flex-1 truncate font-body font-normal text-[13px] leading-[1.4] text-text"
							>
								{place !== null ? place.name : $_('viewer_location_unnamed')}
							</span>
							{#if isOwner}
								<button
									class="flex-none inline-flex items-center justify-center w-7 h-7 rounded-control bg-transparent text-text-faint cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:[outline:2px_solid_var(--color-blue-300)] focus-visible:outline-offset-[-2px]"
									type="button"
									aria-label={$_('viewer_edit_location_open')}
									title={$_('viewer_edit_location_open')}
									onclick={oneditlocation}
								>
									<PenLine size={14} />
								</button>
							{/if}
						</div>
						<div class="relative mt-2 aspect-[5/4] overflow-hidden rounded-card border border-border">
							<!-- The map's own notice states a missing base
							     layer; the row's coordinates stand either way. -->
							<MapView
								point={coordinates}
								cluster={false}
								interactive={false}
								label={$_('viewer_location_map_label')}
							/>
						</div>
						<p
							class="m-0 mt-1.5 font-body font-normal text-[12px] leading-[1.4] text-text-faint tabular-nums"
						>
							{coordinates.lat.toFixed(4)}, {coordinates.lon.toFixed(4)}{#if altitudeLabel !== ''}
								· {altitudeLabel}
							{/if}
						</p>
					</li>
				{/if}
			</ul>
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
