<script lang="ts">
	// The admin settings page (the geometry:
	// the page-header row with its hairline, the centered 760
	// column): the collapsed index of the instance's setting cards --
	// the storage template (the storage-template feature: the
	// instance's upload path template with the built-in presets
	// filling the input, a live sample preview of the SAVED template,
	// and the save's validation errors), the trash retention (whole
	// days 0-365, 0 being immediate permanent delete, with the
	// destructive warning line), the video transcoding profile (the
	// preset, the concurrency bound, the poster-frame switch and the
	// failure policy; the contract's new-ingests-only scope note),
	// the image card (the derivative cost: the sizes and encoding
	// every NEW derivation runs at, the honest consequence line, and
	// the explicit rebuild with its two scopes and its estimate), the
	// map's tile source, the model store's policy, and the release
	// check. The mount-root allowlist section is GONE: the allowlist
	// retired whole (the boundary is the compose mount, nothing left
	// to list).
	//
	// THE COLLAPSE (the settings index): every card is a native
	// <details> with NO open attribute in the markup -- the first
	// visit lands on six collapsed rows (the index), the collapsed
	// row names its card's CURRENT value (data, not new copy), and
	// the EXPANDED SET is remembered in localStorage (storing the
	// expanded set, not the collapsed one, is what keeps a future
	// card collapsed by default). The fold does not depend on
	// scripts: JS-off, the native summary still opens and closes.
	//
	// THE 403 PAGE: a signed-in non-admin landing on this route sees
	// the mapped auth_forbidden message -- the session contract
	// carries no role, so this page's own read is the probe.
	import { _, locale } from 'svelte-i18n';
	import { page } from '$app/state';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import AdminPageHeader from '$lib/components/AdminPageHeader.svelte';
	import { toast } from '$lib/components/toast.svelte';
	import {
		getDerivativesSummary,
		getImageSettings,
		getModelSettings,
		getStorageSettings,
		getTilesSettings,
		getTranscodingSettings,
		getTrashSettings,
		getUpdateCheckSettings,
		rebuildDerivatives,
		saveImageSettings,
		saveModelSettings,
		saveStorageSettings,
		saveTilesSettings,
		saveTranscodingSettings,
		saveTrashSettings,
		saveUpdateCheckSettings,
		type DerivativesSummary,
		type ImageSettings,
		type ModelSettings,
		type StorageSettings,
		type TilesSettings,
		type TranscodingSettings,
		type UpdateCheckSettings
	} from '$lib/api/adminSettings';
	import { getUpdateCheck, runUpdateCheck, type UpdateCheckState } from '$lib/api/updateCheck';
	import { resolveErrorText } from '$lib/i18n/errors';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import Check from '@lucide/svelte/icons/check';
	import { isKnownNonWgs84Basemap } from '$lib/map/wgs84';
	import FolderTree from '@lucide/svelte/icons/folder-tree';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import MonitorPlay from '@lucide/svelte/icons/monitor-play';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import Map from '@lucide/svelte/icons/map';
	import PackageCheck from '@lucide/svelte/icons/package-check';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import Image from '@lucide/svelte/icons/image';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	// THE EXPANDED SET: localStorage stores the ids of the cards the
	// admin left open -- storing the expanded set (not the collapsed
	// one) is what keeps a card added later collapsed by default.
	const EXPANDED_STORAGE_KEY = 'lumemor:settings:expanded';

	type CardId = 'storage' | 'trash' | 'transcoding' | 'image' | 'tiles' | 'models' | 'update';

	/** The per-card open state; the markup starts every card collapsed. */
	let expanded = $state<Record<CardId, boolean>>({
		storage: false,
		trash: false,
		transcoding: false,
		image: false,
		tiles: false,
		models: false,
		update: false
	});

	/** The remembered set's parse: only ids of cards the page carries. */
	function isCardId(value: string): value is CardId {
		return value in expanded;
	}

	// The remembered set rides ONCE at the mount: the fold's default
	// stays in the markup (no open attribute), so a JS-off browser
	// sees the collapsed index and can still open cards natively.
	$effect(() => {
		let stored: string[] = [];
		try {
			const raw = window.localStorage.getItem(EXPANDED_STORAGE_KEY);
			if (raw !== null) {
				const parsed: unknown = JSON.parse(raw);
				if (Array.isArray(parsed)) {
					stored = parsed.filter((entry): entry is string => typeof entry === 'string' && isCardId(entry));
				}
			}
		} catch {
			// An unreadable or unparsable set reads as none: the index.
		}
		for (const id of stored) {
			if (isCardId(id)) {
				expanded[id] = true;
			}
		}
	});

	/** One card's toggle lands in the remembered set (the expanded ids only).
	 * The ids read off the DOM's open states: the toggle event and the
	 * bind's state write race, and the elements are the truth. */
	function rememberExpanded(): void {
		const open = Array.from(document.querySelectorAll<HTMLDetailsElement>('details[data-card]'))
			.filter((element) => element.open)
			.map((element) => element.dataset.card)
			.filter((id): id is CardId => typeof id === 'string' && id in expanded);
		try {
			window.localStorage.setItem(EXPANDED_STORAGE_KEY, JSON.stringify(open));
		} catch {
			// A private-mode or full store loses the memory quietly:
			// the fold itself still works.
		}
	}

	let settings = $state<StorageSettings | null>(null);
	let status = $state<ListStatus>('idle');
	let errorKey = $state('');
	/** Bumping re-runs the read (the retry button). */
	let retryTick = $state(0);

	// The storage form: the template input and the preset that fills
	// it. The preset select follows the input (a hand-edited template
	// reads back as "Custom"); choosing a preset writes it into the
	// input, where it stays editable.
	let formTemplate = $state('');
	let formPreset = $state('custom');
	let saving = $state(false);
	let saveErrorKey = $state('');
	/** The server's specific validation reason of the last failed save. */
	let saveReason = $state('');
	let savedNote = $state(false);

	// The trash retention form.
	let formRetentionDays = $state(30);
	let savingTrash = $state(false);
	let trashErrorKey = $state('');
	let trashSaved = $state(false);

	// The transcoding form.
	let formTranscoding = $state<TranscodingSettings>({
		preset: 'auto',
		concurrency: 2,
		thumbnailFrame: true,
		keepOriginal: true
	});
	let savingTranscoding = $state(false);
	let transcodingErrorKey = $state('');
	let transcodingSaved = $state(false);

	// The image card: the settings form, the cost line and the
	// rebuild dialog. The save recomputes nothing -- the dialog's
	// rebuild is the explicit action.
	let formImage = $state<ImageSettings>({
		tileSize: 250,
		previewSize: 1080,
		format: 'avif',
		fullSize: false,
		rawEmbedded: true
	});
	let savingImage = $state(false);
	let imageErrorKey = $state('');
	let imageSaved = $state(false);
	let derivativesSummary = $state<DerivativesSummary | null>(null);
	let summaryError = $state(false);

	// The rebuild dialog: the two scopes are the honest pair, the
	// estimate rides each scope's note, and the start is one POST.
	let rebuildOpen = $state(false);
	let rebuildScope = $state<'missing' | 'all'>('missing');
	let rebuilding = $state(false);
	let rebuildErrorKey = $state('');

	// The release check: the settings form and the cached state.
	let formUpdateCheck = $state<UpdateCheckSettings>({
		enabled: true,
		endpoint: '',
		intervalHours: 24
	});
	let savingUpdateCheck = $state(false);
	let updateCheckErrorKey = $state('');
	let updateCheckSaved = $state(false);
	/** The cached check state (null until the read lands). */
	let updateState = $state<UpdateCheckState | null>(null);
	/** True while the manual check runs. */
	let checking = $state(false);

	// The map's tile source form: the light style and the optional
	// dark one, saved as one document. The library is WGS-84 only; a
	// known non-WGS-84 host raises the offset warning below.
	let formTilejsonUrl = $state('');
	let formDarkTilejsonUrl = $state('');
	const tileWarning = $derived(
		isKnownNonWgs84Basemap(formTilejsonUrl) || isKnownNonWgs84Basemap(formDarkTilejsonUrl)
	);
	let savingTiles = $state(false);
	let tilesErrorKey = $state('');
	let tilesSaved = $state(false);

	// The model store's policy form.
	let formAutoDownload = $state(true);
	/** The model settings' read document, the sources included: the
	 * card's save is the whole object (the PUT replaces wholesale). */
	let formModels = $state<ModelSettings>({
		autoDownload: true,
		allowedHosts: [],
		mirrorBaseUrl: '',
		params: { faceDetectMinScore: 0.6, faceRecognizeThreshold: 0.55 }
	});
	let savingModels = $state(false);
	let modelsErrorKey = $state('');
	let modelsSaved = $state(false);

	// The read, keyed on the retry tick. A superseded run -- the
	// retry fired again, or the route left -- never applies.
	$effect(() => {
		void retryTick;
		status = 'loading';
		errorKey = '';
		let current = true;
		void (async () => {
			const [storage, trash, transcoding, updateCheck, tiles, models, checkState, image, summary] =
				await Promise.all([
					getStorageSettings(),
					getTrashSettings(),
					getTranscodingSettings(),
					getUpdateCheckSettings(),
					getTilesSettings(),
					getModelSettings(),
					getUpdateCheck(),
					getImageSettings(),
					getDerivativesSummary()
				]);
			if (!current) {
				return;
			}
			// The reads share one gate: they are one page, and the
			// admin-role probe belongs to whichever read answers first
			// (any failure -- the role refusal, a session loss, an
			// unparseable body -- is the page's failure). The summary's
			// read stands apart: its failure dims the image card's
			// cost line, it never fails the page.
			if (!storage.ok) {
				status = 'error';
				errorKey = storage.errorKey;
				return;
			}
			if (!trash.ok) {
				status = 'error';
				errorKey = trash.errorKey;
				return;
			}
			if (!transcoding.ok) {
				status = 'error';
				errorKey = transcoding.errorKey;
				return;
			}
			if (!updateCheck.ok) {
				status = 'error';
				errorKey = updateCheck.errorKey;
				return;
			}
			if (!tiles.ok) {
				status = 'error';
				errorKey = tiles.errorKey;
				return;
			}
			if (!models.ok) {
				status = 'error';
				errorKey = models.errorKey;
				return;
			}
			if (!checkState.ok) {
				status = 'error';
				errorKey = checkState.errorKey;
				return;
			}
			if (!image.ok) {
				status = 'error';
				errorKey = image.errorKey;
				return;
			}
			adoptSettings(storage.settings, trash.settings.retentionDays, transcoding.settings);
			adoptImageSettings(image.settings);
			adoptNetworkingSettings(updateCheck.settings, tiles.settings, models.settings);
			updateState = checkState.state;
			if (summary.ok) {
				derivativesSummary = summary.settings;
				summaryError = false;
			} else {
				summaryError = true;
			}
			status = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	// The dialog deep link (?dialog=rebuild-derivatives): the dialog
	// opens over the collapsed index -- the collapse and the dialog
	// are independent layers (the design's boundary table).
	$effect(() => {
		if (status === 'ready' && page.url.searchParams.get('dialog') === 'rebuild-derivatives') {
			rebuildOpen = true;
		}
	});

	/** Adopt the image card's read document as its fresh state. */
	function adoptImageSettings(next: ImageSettings): void {
		formImage = { ...next };
		imageErrorKey = '';
		imageSaved = false;
	}

	/** Adopt the networking sections' read documents as their fresh state. */
	function adoptNetworkingSettings(
		updateCheck: UpdateCheckSettings,
		tiles: TilesSettings,
		models: ModelSettings
	): void {
		formUpdateCheck = { ...updateCheck };
		updateCheckErrorKey = '';
		updateCheckSaved = false;
		formTilejsonUrl = tiles.tilejsonUrl;
		formDarkTilejsonUrl = tiles.darkTilejsonUrl;
		tilesErrorKey = '';
		tilesSaved = false;
		formModels = { ...models };
		modelsErrorKey = '';
		modelsSaved = false;
	}

	/** Adopt the three read documents as the sections' fresh state. */
	function adoptSettings(next: StorageSettings, retentionDays: number, transcoding: TranscodingSettings): void {
		settings = next;
		formTemplate = next.template;
		formPreset = presetOf(next.template);
		saveErrorKey = '';
		saveReason = '';
		savedNote = false;
		formRetentionDays = retentionDays;
		trashErrorKey = '';
		trashSaved = false;
		formTranscoding = { ...transcoding };
		transcodingErrorKey = '';
		transcodingSaved = false;
	}

	/** The preset id of one template text, or the custom marker. */
	function presetOf(template: string): string {
		const found = settings?.presets.find((preset) => preset === template);
		return found ?? 'custom';
	}

	/** Choosing a preset fills the input with it, ready to edit. */
	function pickPreset(event: Event): void {
		const select = event.currentTarget as HTMLSelectElement;
		if (select.value === 'custom' || select.value === '') {
			return;
		}
		formPreset = select.value;
		formTemplate = select.value;
		saveErrorKey = '';
		saveReason = '';
		savedNote = false;
	}

	function retry(): void {
		retryTick += 1;
	}

	// The shared unknown-code fallback (see $lib/i18n/errors); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	async function handleSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (saving) {
			return;
		}
		saving = true;
		saveErrorKey = '';
		saveReason = '';
		savedNote = false;
		const outcome = await saveStorageSettings(formTemplate);
		saving = false;
		if (outcome.ok) {
			// The save adopts the fresh answer (the other sections keep
			// their own drafts: the trash and transcoding forms are
			// untouched by this write).
			adoptSettings(outcome.settings, formRetentionDays, formTranscoding);
			savedNote = true;
			return;
		}
		saveErrorKey = outcome.errorKey;
		saveReason = outcome.reason ?? '';
	}

	async function handleTrashSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (savingTrash) {
			return;
		}
		const days = Number(formRetentionDays);
		if (!Number.isInteger(days)) {
			return;
		}
		savingTrash = true;
		trashErrorKey = '';
		trashSaved = false;
		const outcome = await saveTrashSettings(days);
		savingTrash = false;
		if (outcome.ok) {
			formRetentionDays = outcome.settings.retentionDays;
			trashSaved = true;
			return;
		}
		trashErrorKey = outcome.errorKey;
	}

	async function handleTranscodingSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (savingTranscoding) {
			return;
		}
		savingTranscoding = true;
		transcodingErrorKey = '';
		transcodingSaved = false;
		const outcome = await saveTranscodingSettings({ ...formTranscoding });
		savingTranscoding = false;
		if (outcome.ok) {
			formTranscoding = { ...outcome.settings };
			transcodingSaved = true;
			return;
		}
		transcodingErrorKey = outcome.errorKey;
	}

	async function handleImageSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (savingImage) {
			return;
		}
		savingImage = true;
		imageErrorKey = '';
		imageSaved = false;
		const outcome = await saveImageSettings({ ...formImage });
		savingImage = false;
		if (outcome.ok) {
			formImage = { ...outcome.settings };
			imageSaved = true;
			return;
		}
		imageErrorKey = outcome.errorKey;
	}

	/**
	 * The rebuild start: one POST with the dialog's chosen scope, the
	 * honest toast on the way out, the error line in the dialog.
	 */
	async function handleRebuildStart(): Promise<void> {
		if (rebuilding) {
			return;
		}
		rebuilding = true;
		rebuildErrorKey = '';
		const outcome = await rebuildDerivatives(rebuildScope);
		rebuilding = false;
		if (outcome.ok) {
			rebuildOpen = false;
			toast($_('admin_rebuild_started'));
			return;
		}
		rebuildErrorKey = outcome.errorKey;
	}

	async function handleUpdateCheckSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (savingUpdateCheck) {
			return;
		}
		savingUpdateCheck = true;
		updateCheckErrorKey = '';
		updateCheckSaved = false;
		const outcome = await saveUpdateCheckSettings({ ...formUpdateCheck });
		savingUpdateCheck = false;
		if (outcome.ok) {
			formUpdateCheck = { ...outcome.settings };
			updateCheckSaved = true;
			return;
		}
		updateCheckErrorKey = outcome.errorKey;
	}

	/**
	 * The manual check: one server-side probe, then the fresh state
	 * replaces the card's. A failed probe is not an error here -- the
	 * answer's status stays `unknown` and the card keeps its quiet
	 * "last checked" line.
	 */
	async function handleCheckNow(): Promise<void> {
		if (checking) {
			return;
		}
		checking = true;
		const outcome = await runUpdateCheck();
		checking = false;
		if (outcome.ok) {
			updateState = outcome.state;
		}
	}

	async function handleTilesSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (savingTiles) {
			return;
		}
		savingTiles = true;
		tilesErrorKey = '';
		tilesSaved = false;
		const outcome = await saveTilesSettings(formTilejsonUrl.trim(), formDarkTilejsonUrl.trim());
		savingTiles = false;
		if (outcome.ok) {
			formTilejsonUrl = outcome.settings.tilejsonUrl;
			formDarkTilejsonUrl = outcome.settings.darkTilejsonUrl;
			tilesSaved = true;
			return;
		}
		tilesErrorKey = outcome.errorKey;
	}

	/**
	 * The reset is a DRAFT edit, not a save: both fields become the
	 * unconfigured state and the existing Save persists it, so the
	 * administrator still sees what the action did before it lands.
	 */
	function resetTileSources(): void {
		formTilejsonUrl = '';
		formDarkTilejsonUrl = '';
		tilesErrorKey = '';
		tilesSaved = false;
	}

	async function handleModelsSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (savingModels) {
			return;
		}
		savingModels = true;
		modelsErrorKey = '';
		modelsSaved = false;
		// The save is the whole object: the read document carried the
		// download sources too, and the PUT replaces them wholesale.
		const outcome = await saveModelSettings({ ...formModels, autoDownload: formAutoDownload });
		savingModels = false;
		if (outcome.ok) {
			formModels = { ...outcome.settings };
			formAutoDownload = outcome.settings.autoDownload;
			modelsSaved = true;
			return;
		}
		modelsErrorKey = outcome.errorKey;
	}

	/**
	 * One relative time in the active locale (the Intl table's own
	 * wording): the card's quiet last-checked line. Minute granularity
	 * is the finest the cadence needs.
	 */
	function relativeTime(iso: string): string {
		const seconds = Math.round((new Date(iso).getTime() - Date.now()) / 1000);
		const formatter = new Intl.RelativeTimeFormat($locale ?? 'en', { numeric: 'auto' });
		const magnitude = Math.abs(seconds);
		if (magnitude < 3600) {
			return formatter.format(Math.round(seconds / 60), 'minute');
		}
		if (magnitude < 86400) {
			return formatter.format(Math.round(seconds / 3600), 'hour');
		}
		return formatter.format(Math.round(seconds / 86400), 'day');
	}

	const presetChoices: ReadonlyArray<{ value: TranscodingSettings['preset']; labelKey: string }> = [
		{ value: 'auto', labelKey: 'admin_settings_transcode_preset_auto' },
		{ value: '1080p', labelKey: 'admin_settings_transcode_preset_1080p' },
		{ value: '720p', labelKey: 'admin_settings_transcode_preset_720p' },
		{ value: '480p', labelKey: 'admin_settings_transcode_preset_480p' },
		{ value: 'off', labelKey: 'admin_settings_transcode_preset_off' }
	];

	const concurrencyChoices: ReadonlyArray<{ value: number; labelKey: string }> = [
		{ value: 1, labelKey: 'admin_settings_transcode_concurrency_1' },
		{ value: 2, labelKey: 'admin_settings_transcode_concurrency_2' },
		{ value: 4, labelKey: 'admin_settings_transcode_concurrency_4' }
	];

	/** The collapsed row's value for one card: NOW what is, never what will be. */
	function metaOf(id: CardId): string {
		switch (id) {
			case 'storage':
				return settings?.template ?? '';
			case 'trash':
				return $_('admin_settings_collapse_days', { values: { days: formRetentionDays } });
			case 'transcoding':
				return `${$_(presetOf(formTranscoding.preset) === 'custom' ? 'admin_settings_transcode_preset_auto' : `admin_settings_transcode_preset_${formTranscoding.preset}`)} · ${formTranscoding.concurrency}`;
			case 'image':
				return `${$_('admin_settings_image_thumb_label')} ${formImage.tileSize} · ${formImage.previewSize} · ${formImage.format.toUpperCase()}`;
			case 'tiles':
				return formTilejsonUrl === '' && formDarkTilejsonUrl === ''
					? $_('admin_settings_collapse_unconfigured')
					: $_('admin_settings_collapse_configured');
			case 'models':
				return formAutoDownload ? $_('admin_settings_collapse_on') : $_('admin_settings_collapse_off');
			case 'update':
				return `${updateState?.currentVersion ?? ''} · ${formUpdateCheck.enabled ? $_('admin_settings_collapse_on') : $_('admin_settings_collapse_off')}`;
		}
	}
</script>

<svelte:head>
	<title>Settings · lumemor</title>
</svelte:head>

<div class="settings max-w-[760px] mx-auto flex flex-col gap-4">
	{#if status === 'loading'}
		<p class="settings__status my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('admin_settings_loading')}
		</p>
	{:else if status === 'error'}
		<div
			class="settings__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{:else if status === 'ready' && settings !== null}
		<!-- The page-header row: the title
		     echoing the rail's active section, the right action slot
		     -- settings carries no page-level action, each section
		     owns its save, but the row stands for uniformity -- and
		     the hairline that pins the row into a belt with the
		     rail's border). -->
		<AdminPageHeader title={$_('admin_nav_settings')} />

		<!-- The storage layout card: a native details fold, no open
		     attribute in the markup (the index is the first visit's
		     face), the collapsed row naming the CURRENT template. -->
		<details
			class="settings__card group bg-card border border-border rounded-card shadow-card p-5"
			data-testid="admin-settings-storage"
			data-card="storage"
			bind:open={expanded.storage}
			ontoggle={rememberExpanded}
		>
			<summary
				class="flex cursor-pointer select-none list-none items-center gap-2 -mx-2 px-2 -my-1 py-1 rounded-control transition-colors duration-(--motion-fast) ease-standard hover:bg-accent-soft/40"
			>
				<h2
					class="settings__section-title flex items-center gap-2 font-display font-bold text-[17px] leading-[1.3] text-text"
				>
					<FolderTree size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('admin_settings_storage_title')}
				</h2>
				<span
					class="settings__section-meta ml-1 font-body font-normal text-[13px] leading-[1.4] text-text-faint tabular-nums group-open:hidden"
				>
					<code class="font-mono text-[12px]">{metaOf('storage')}</code>
				</span>
				<ChevronDown
					size={18}
					class="ml-auto shrink-0 text-text-faint transition-transform duration-(--motion-fast) ease-standard group-open:rotate-180"
					aria-hidden="true"
				/>
			</summary>
			<div class="mt-3">
				<p
					class="settings__section-note mb-3.5 font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_storage_note')}
				</p>
				<form class="settings__form flex flex-col gap-3.5" onsubmit={handleSubmit}>
					<div class="settings__field flex flex-col gap-1.5">
						<label
							class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text"
							for="admin-storage-preset">{$_('admin_settings_preset_label')}</label
						>
						<select
							class="settings__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard"
							id="admin-storage-preset"
							bind:value={formPreset}
							onchange={pickPreset}
							data-testid="storage-preset"
						>
							<option value="custom">{$_('admin_settings_preset_custom')}</option>
							{#each settings.presets as preset (preset)}
								<option value={preset}>{preset}</option>
							{/each}
						</select>
					</div>
					<Input
						id="admin-storage-template"
						label={$_('admin_settings_template_label')}
						type="text"
						autocomplete="off"
						spellcheck="false"
						class="font-mono"
						bind:value={formTemplate}
					/>
					<!-- The sample preview of the SAVED template: the save adopts the
					     fresh render, so the line always names the layout on disk. -->
					<div class="settings__sample flex flex-col gap-1 pt-1" data-testid="storage-sample">
						<p
							class="settings__sample-label font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint"
						>
							{$_('admin_settings_sample_label')}
						</p>
						<p
							class="settings__sample-path font-mono font-normal text-[13px] leading-[1.5] text-text-soft break-all"
						>
							{settings.samplePath}
						</p>
					</div>
					{#if savedNote}
						<p
							class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft"
							role="status"
						>
							{$_('admin_settings_saved_note')}
						</p>
					{/if}
					{#if saveErrorKey !== ''}
						<div class="settings__error flex flex-col gap-0.5" role="alert">
							<p class="font-body font-medium text-[14px] leading-[1.4] text-danger-text">
								{errorText(saveErrorKey)}
							</p>
							{#if saveReason !== ''}
								<p
									class="settings__error-reason font-mono font-normal text-[13px] leading-[1.4] text-danger-text break-all"
									data-testid="storage-error-reason"
								>
									{saveReason}
								</p>
							{/if}
						</div>
					{/if}
					<div class="settings__form-actions flex justify-end mt-1">
						<Button type="submit" disabled={saving}
							><Check size={20} aria-hidden="true" />{$_('admin_settings_submit')}</Button
						>
					</div>
				</form>
				<p
					class="settings__contract mt-4 pt-4 border-t border-border font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_contract_note')}
				</p>
			</div>
		</details>

		<!-- The trash retention card: the number input, the save,
		     and the destructive warning line. -->
		<details
			class="settings__card group bg-card border border-border rounded-card shadow-card p-5"
			data-testid="admin-settings-trash"
			data-card="trash"
			bind:open={expanded.trash}
			ontoggle={rememberExpanded}
		>
			<summary
				class="flex cursor-pointer select-none list-none items-center gap-2 -mx-2 px-2 -my-1 py-1 rounded-control transition-colors duration-(--motion-fast) ease-standard hover:bg-accent-soft/40"
			>
				<h2
					class="settings__section-title flex items-center gap-2 font-display font-bold text-[17px] leading-[1.3] text-text"
				>
					<Trash2 size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('admin_settings_trash_title')}
				</h2>
				<span
					class="settings__section-meta ml-1 font-body font-normal text-[13px] leading-[1.4] text-text-faint tabular-nums group-open:hidden"
				>
					{metaOf('trash')}
				</span>
				<ChevronDown
					size={18}
					class="ml-auto shrink-0 text-text-faint transition-transform duration-(--motion-fast) ease-standard group-open:rotate-180"
					aria-hidden="true"
				/>
			</summary>
			<div class="mt-3">
				<p
					class="settings__section-note mb-3.5 font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_trash_note')}
				</p>
				<form class="settings__trash flex flex-col gap-3.5" onsubmit={handleTrashSubmit}>
					<div class="settings__field flex flex-col gap-1.5">
						<label
							class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text"
							for="admin-trash-days">{$_('admin_settings_trash_days')}</label
						>
						<input
							class="settings__number w-[120px] px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard"
							id="admin-trash-days"
							type="number"
							min="0"
							max="365"
							step="1"
							data-testid="trash-retention"
							bind:value={formRetentionDays}
						/>
					</div>
					{#if trashSaved}
						<p
							class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft"
							role="status"
						>
							{$_('admin_settings_trash_saved')}
						</p>
					{/if}
					{#if trashErrorKey !== ''}
						<p
							class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(trashErrorKey)}
						</p>
					{/if}
					<div class="settings__form-actions flex justify-end mt-1">
						<Button type="submit" disabled={savingTrash}
							><Check size={20} aria-hidden="true" />{$_('admin_settings_trash_save')}</Button
						>
					</div>
				</form>
				<p
					class="settings__warning mt-4 pt-4 border-t border-border flex items-start gap-2 font-body font-medium text-[12px] leading-[1.5] text-danger-text"
				>
					<TriangleAlert size={14} class="settings__warning-icon shrink-0 mt-0.5" aria-hidden="true" />
					{$_('admin_settings_trash_warning')}
				</p>
			</div>
		</details>

		<!-- The video transcoding card: the preset and concurrency
		     dropdowns, the two toggles, the save, and the
		     new-ingests-only scope note. -->
		<details
			class="settings__card group bg-card border border-border rounded-card shadow-card p-5"
			data-testid="admin-settings-transcoding"
			data-card="transcoding"
			bind:open={expanded.transcoding}
			ontoggle={rememberExpanded}
		>
			<summary
				class="flex cursor-pointer select-none list-none items-center gap-2 -mx-2 px-2 -my-1 py-1 rounded-control transition-colors duration-(--motion-fast) ease-standard hover:bg-accent-soft/40"
			>
				<h2
					class="settings__section-title flex items-center gap-2 font-display font-bold text-[17px] leading-[1.3] text-text"
				>
					<MonitorPlay size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('admin_settings_transcode_title')}
				</h2>
				<span
					class="settings__section-meta ml-1 font-body font-normal text-[13px] leading-[1.4] text-text-faint tabular-nums group-open:hidden"
				>
					{metaOf('transcoding')}
				</span>
				<ChevronDown
					size={18}
					class="ml-auto shrink-0 text-text-faint transition-transform duration-(--motion-fast) ease-standard group-open:rotate-180"
					aria-hidden="true"
				/>
			</summary>
			<div class="mt-3">
				<p
					class="settings__section-note mb-3.5 font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_transcode_note')}
				</p>
				<form class="settings__transcoding flex flex-col gap-3.5" onsubmit={handleTranscodingSubmit}>
					<div class="settings__field flex flex-col gap-1.5">
						<label
							class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text"
							for="admin-transcode-preset">{$_('admin_settings_transcode_preset')}</label
						>
						<select
							class="settings__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard"
							id="admin-transcode-preset"
							data-testid="transcode-preset"
							bind:value={formTranscoding.preset}
						>
							{#each presetChoices as choice (choice.value)}
								<option value={choice.value}>{$_(choice.labelKey)}</option>
							{/each}
						</select>
					</div>
					<div class="settings__field flex flex-col gap-1.5">
						<label
							class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text"
							for="admin-transcode-concurrency">{$_('admin_settings_transcode_concurrency')}</label
						>
						<select
							class="settings__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard"
							id="admin-transcode-concurrency"
							data-testid="transcode-concurrency"
							bind:value={formTranscoding.concurrency}
						>
							{#each concurrencyChoices as choice (choice.value)}
								<option value={choice.value}>{$_(choice.labelKey)}</option>
							{/each}
						</select>
					</div>
					<div class="settings__toggles flex flex-col gap-2">
						<label
							class="settings__toggle flex items-center gap-2 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
						>
							<input
								class="settings__toggle-input w-[18px] h-[18px] accent-accent"
								type="checkbox"
								data-testid="transcode-thumbnail-frame"
								bind:checked={formTranscoding.thumbnailFrame}
							/>
							{$_('admin_settings_transcode_thumbnail')}
						</label>
						<label
							class="settings__toggle flex items-center gap-2 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
						>
							<input
								class="settings__toggle-input w-[18px] h-[18px] accent-accent"
								type="checkbox"
								data-testid="transcode-keep-original"
								bind:checked={formTranscoding.keepOriginal}
							/>
							{$_('admin_settings_transcode_keep')}
						</label>
					</div>
					{#if transcodingSaved}
						<p
							class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft"
							role="status"
						>
							{$_('admin_settings_transcode_saved')}
						</p>
					{/if}
					{#if transcodingErrorKey !== ''}
						<p
							class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(transcodingErrorKey)}
						</p>
					{/if}
					<div class="settings__form-actions flex justify-end mt-1">
						<Button type="submit" disabled={savingTranscoding}
							><Check size={20} aria-hidden="true" />{$_('admin_settings_transcode_save')}</Button
						>
					</div>
				</form>
				<p
					class="settings__contract mt-4 pt-4 border-t border-border font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_transcode_contract')}
				</p>
			</div>
		</details>

		<!-- The image card: the derivative cost. The four knobs, the
		     honest consequence line (the save recomputes nothing), and
		     the card's own rebuild entry with its cost line. -->
		<details
			class="settings__card group bg-card border border-border rounded-card shadow-card p-5"
			data-testid="admin-settings-image"
			data-card="image"
			bind:open={expanded.image}
			ontoggle={rememberExpanded}
		>
			<summary
				class="flex cursor-pointer select-none list-none items-center gap-2 -mx-2 px-2 -my-1 py-1 rounded-control transition-colors duration-(--motion-fast) ease-standard hover:bg-accent-soft/40"
			>
				<h2
					class="settings__section-title flex items-center gap-2 font-display font-bold text-[17px] leading-[1.3] text-text"
				>
					<Image size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('admin_settings_image_title')}
				</h2>
				<span
					class="settings__section-meta ml-1 font-body font-normal text-[13px] leading-[1.4] text-text-faint tabular-nums group-open:hidden"
				>
					{metaOf('image')}
				</span>
				<ChevronDown
					size={18}
					class="ml-auto shrink-0 text-text-faint transition-transform duration-(--motion-fast) ease-standard group-open:rotate-180"
					aria-hidden="true"
				/>
			</summary>
			<div class="mt-3">
				<p
					class="settings__section-note mb-3.5 font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_image_note')}
				</p>
				<form class="settings__image flex flex-col gap-3.5" onsubmit={handleImageSubmit}>
					<div class="settings__field flex flex-col gap-1.5">
						<label
							class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text"
							for="admin-image-thumb">{$_('admin_settings_image_thumb_label')}</label
						>
						<select
							class="settings__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard"
							id="admin-image-thumb"
							data-testid="image-tile-size"
							bind:value={formImage.tileSize}
						>
							<option value={250}>{$_('admin_settings_image_thumb_250')}</option>
							<option value={320}>{$_('admin_settings_image_thumb_320')}</option>
						</select>
					</div>
					<div class="settings__field flex flex-col gap-1.5">
						<label
							class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text"
							for="admin-image-preview">{$_('admin_settings_image_preview_label')}</label
						>
						<select
							class="settings__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard"
							id="admin-image-preview"
							data-testid="image-preview-size"
							bind:value={formImage.previewSize}
						>
							<option value={1080}>{$_('admin_settings_image_preview_1080')}</option>
							<option value={1440}>{$_('admin_settings_image_preview_1440')}</option>
							<option value={2048}>{$_('admin_settings_image_preview_2048')}</option>
						</select>
					</div>
					<div class="settings__field flex flex-col gap-1.5">
						<label
							class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text"
							for="admin-image-format">{$_('admin_settings_image_format_label')}</label
						>
						<select
							class="settings__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard"
							id="admin-image-format"
							data-testid="image-format"
							bind:value={formImage.format}
						>
							<option value="avif">{$_('admin_settings_image_format_avif')}</option>
							<option value="webp">{$_('admin_settings_image_format_webp')}</option>
						</select>
						<span class="font-body font-normal text-[12px] leading-[1.5] text-text-faint">
							{$_('admin_settings_image_format_hint')}
						</span>
					</div>
					<div class="flex flex-col gap-2">
						<label
							class="settings__toggle flex items-start gap-2.5 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
						>
							<input
								class="settings__toggle-input mt-0.5 w-[18px] h-[18px] accent-accent shrink-0"
								type="checkbox"
								data-testid="image-fullsize"
								bind:checked={formImage.fullSize}
							/>
							<span>{$_('admin_settings_image_fullsize')}</span>
						</label>
						<span class="ml-7 font-body font-normal text-[12px] leading-[1.5] text-text-faint">
							{$_('admin_settings_image_fullsize_hint')}
						</span>
					</div>
					<label
						class="settings__toggle flex items-start gap-2.5 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
					>
						<input
							class="settings__toggle-input mt-0.5 w-[18px] h-[18px] accent-accent shrink-0"
							type="checkbox"
							data-testid="image-raw-embedded"
							bind:checked={formImage.rawEmbedded}
						/>
						<span>{$_('admin_settings_image_raw_embedded')}</span>
					</label>
					{#if imageSaved}
						<p
							class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft"
							role="status"
							data-testid="image-saved"
						>
							{$_('admin_settings_image_saved')}
						</p>
					{/if}
					{#if imageErrorKey !== ''}
						<p
							class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(imageErrorKey)}
						</p>
					{/if}
					<div class="settings__form-actions flex justify-end mt-1">
						<Button type="submit" disabled={savingImage}
							><Check size={20} aria-hidden="true" />{$_('admin_settings_submit')}</Button
						>
					</div>
				</form>
				<div class="mt-4 pt-4 border-t border-border flex flex-col gap-2">
					<!-- The consequence line: the honest cost statement,
					     the card never redoes work behind your back. -->
					<div
						class="flex items-start gap-1.5 px-3 py-2 rounded-control bg-[color-mix(in_srgb,var(--color-warning)_22%,transparent)] border border-[color-mix(in_srgb,var(--color-warning)_55%,transparent)]"
					>
						<TriangleAlert
							size={14}
							class="settings__warning-icon shrink-0 mt-0.5 text-warning-text"
							aria-hidden="true"
						/>
						<span class="font-body font-normal text-[12px] leading-[1.5] text-text">
							{$_('admin_settings_image_consequence')}
						</span>
					</div>
					<!-- The cost line: the library's size and the declared
					     estimate. A failed summary read dims the line, never
					     the card. -->
					<div class="flex flex-wrap items-center justify-between gap-2">
						<span
							class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
							data-testid="image-summary"
						>
							{#if derivativesSummary !== null}
								{$_('admin_settings_image_library_line', {
									values: {
										count: derivativesSummary.imageCount,
										minutes: derivativesSummary.estimateMinutes
									}
								})}
							{:else if summaryError}
								{$_('admin_settings_image_summary_error')}
							{/if}
						</span>
						<Button variant="ghost" onclick={() => (rebuildOpen = true)} data-testid="image-rebuild-open">
							<RefreshCw size={20} aria-hidden="true" />
							{$_('admin_settings_image_rebuild')}
						</Button>
					</div>
				</div>
			</div>
		</details>

		<!-- The release check: the running version, the last answer's
		     face, the manual check, and the three knobs. A failed check
		     is silent -- quiet text, no danger color, no toast. -->
		<details
			class="settings__card group bg-card border border-border rounded-card shadow-card p-5"
			data-testid="admin-settings-update"
			data-card="update"
			bind:open={expanded.update}
			ontoggle={rememberExpanded}
		>
			<summary
				class="flex cursor-pointer select-none list-none items-center gap-2 -mx-2 px-2 -my-1 py-1 rounded-control transition-colors duration-(--motion-fast) ease-standard hover:bg-accent-soft/40"
			>
				<h2
					class="settings__section-title flex items-center gap-2 font-display font-bold text-[17px] leading-[1.3] text-text"
				>
					<RefreshCw size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('admin_settings_update_title')}
				</h2>
				<span
					class="settings__section-meta ml-1 font-body font-normal text-[13px] leading-[1.4] text-text-faint tabular-nums group-open:hidden"
				>
					{metaOf('update')}
				</span>
				<ChevronDown
					size={18}
					class="ml-auto shrink-0 text-text-faint transition-transform duration-(--motion-fast) ease-standard group-open:rotate-180"
					aria-hidden="true"
				/>
			</summary>
			<div class="mt-3">
				<p
					class="settings__section-note mb-3.5 font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_update_note')}
				</p>

				<!-- The running version, with the deployment's own answer to
				     "who updates this": nobody here. -->
				<div
					class="settings__version flex flex-wrap items-center justify-between gap-2 pb-3 border-b border-border"
					data-testid="update-check-version"
				>
					<div class="flex flex-col gap-0.5">
						<span class="font-body font-semibold text-[14px] leading-[1.4] text-text">
							{$_('admin_settings_update_version_label')}
						</span>
						<span class="font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums">
							{updateState?.currentVersion ?? ''}
						</span>
					</div>
					<span
						class="settings__deployment inline-flex items-center h-8 px-3 rounded-control border border-border bg-page font-body font-semibold text-[13px] leading-[1.4] text-text-soft"
					>
						{$_('admin_settings_update_badge')}
					</span>
				</div>

				<!-- The three faces: exactly one renders. A failed check is
				     SILENT -- quiet text, no danger color, no toast, no
				     dialog. -->
				<div class="settings__update-face mt-3" data-testid="update-check-state">
					{#if updateState !== null}
						{#if updateState.status === 'update_available'}
							<div class="settings__face flex flex-col gap-2" data-update-face="available">
								<div class="flex flex-wrap items-center gap-2">
									<span
										class="settings__update-chip inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-notice text-navy-900 font-body font-semibold text-[12px] leading-[1.4]"
									>
										{$_('admin_settings_update_available', {
											values: { version: updateState.latestVersion ?? '' }
										})}
									</span>
									{#if updateState.checkedAt !== null}
										<span
											class="font-body font-normal text-[13px] leading-[1.4] text-text-faint tabular-nums"
										>
											{$_('admin_settings_update_checked_at', {
												values: { when: relativeTime(updateState.checkedAt) }
											})}
										</span>
									{/if}
								</div>
								<p class="m-0 font-body font-normal text-[13px] leading-[1.5] text-text-soft">
									{$_('admin_settings_update_not_automatic')}
								</p>
							</div>
						{:else if updateState.status === 'up_to_date'}
							<div class="settings__face flex flex-wrap items-center gap-2" data-update-face="latest">
								<span
									class="settings__update-chip inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-accent-soft text-accent-soft-text font-body font-semibold text-[12px] leading-[1.4]"
								>
									<CircleCheck size={13} aria-hidden="true" />
									{$_('admin_settings_update_latest')}
								</span>
								{#if updateState.checkedAt !== null}
									<span
										class="font-body font-normal text-[13px] leading-[1.4] text-text-faint tabular-nums"
									>
										{$_('admin_settings_update_checked_at', {
											values: { when: relativeTime(updateState.checkedAt) }
										})}
									</span>
								{/if}
							</div>
						{:else if updateState.status === 'disabled'}
							<div class="settings__face" data-update-face="off">
								<span class="font-body font-normal text-[13px] leading-[1.4] text-text-faint">
									{$_('admin_settings_update_off')}
								</span>
							</div>
						{:else}
							<!-- The silent failure: the state is unknown, and the
							     only thing the card says is when it last asked.
							     The retry sentence is the affordance, not a
							     warning. -->
							<div class="settings__face flex flex-col gap-2" data-update-face="failed">
								<span
									class="settings__update-quiet font-body font-normal text-[13px] leading-[1.4] text-text-faint tabular-nums"
								>
									{#if updateState.checkedAt !== null}
										{$_('admin_settings_update_unknown', {
											values: { when: relativeTime(updateState.checkedAt) }
										})}
									{:else}
										{$_('admin_settings_update_never')}
									{/if}
								</span>
								<p class="m-0 font-body font-normal text-[13px] leading-[1.5] text-text-faint">
									{$_('admin_settings_update_failed_note')}
								</p>
							</div>
						{/if}
					{/if}
					<div class="settings__update-actions flex justify-end mt-3">
						<Button
							variant="ghost"
							disabled={checking || updateState?.enabled === false}
							onclick={() => void handleCheckNow()}
							data-testid="update-check-now"
							aria-busy={checking}
							><RefreshCw size={20} aria-hidden="true" />{checking
								? $_('admin_settings_update_checking')
								: $_('admin_settings_update_check_now')}</Button
						>
					</div>
				</div>

				<form class="settings__update-form flex flex-col gap-3.5 mt-4" onsubmit={handleUpdateCheckSubmit}>
					<label
						class="settings__toggle flex items-center gap-2 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
					>
						<input
							class="settings__toggle-input w-[18px] h-[18px] accent-accent"
							type="checkbox"
							data-testid="update-check-enabled"
							bind:checked={formUpdateCheck.enabled}
						/>
						{$_('admin_settings_update_enabled')}
					</label>
					<Input
						id="admin-update-endpoint"
						label={$_('admin_settings_update_endpoint')}
						type="url"
						autocomplete="off"
						spellcheck="false"
						class="font-mono"
						bind:value={formUpdateCheck.endpoint}
					/>
					<div class="settings__field flex flex-col gap-1.5">
						<label
							class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text"
							for="admin-update-interval">{$_('admin_settings_update_interval')}</label
						>
						<input
							class="settings__number w-[120px] px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard"
							id="admin-update-interval"
							type="number"
							min="1"
							max="168"
							step="1"
							data-testid="update-check-interval"
							bind:value={formUpdateCheck.intervalHours}
						/>
					</div>
					{#if updateCheckSaved}
						<p
							class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft"
							role="status"
						>
							{$_('admin_settings_update_saved')}
						</p>
					{/if}
					{#if updateCheckErrorKey !== ''}
						<p
							class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(updateCheckErrorKey)}
						</p>
					{/if}
					<div class="settings__form-actions flex justify-end mt-1">
						<Button type="submit" disabled={savingUpdateCheck}
							><Check size={20} aria-hidden="true" />{$_('admin_settings_update_save')}</Button
						>
					</div>
				</form>
				<p
					class="settings__contract mt-4 pt-4 border-t border-border font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_update_contract')}
				</p>
			</div>
		</details>

		<!-- The map's tile sources: the browser loads them, so LAN
		     addresses are legitimate; empty values are the shipped
		     state and the map's stated absence. The dark style is
		     optional -- an empty one reuses the light source. -->
		<details
			class="settings__card group bg-card border border-border rounded-card shadow-card p-5"
			data-testid="admin-settings-tiles"
			data-card="tiles"
			bind:open={expanded.tiles}
			ontoggle={rememberExpanded}
		>
			<summary
				class="flex cursor-pointer select-none list-none items-center gap-2 -mx-2 px-2 -my-1 py-1 rounded-control transition-colors duration-(--motion-fast) ease-standard hover:bg-accent-soft/40"
			>
				<h2
					class="settings__section-title flex items-center gap-2 font-display font-bold text-[17px] leading-[1.3] text-text"
				>
					<Map size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('admin_settings_tiles_title')}
				</h2>
				<span
					class="settings__section-meta ml-1 font-body font-normal text-[13px] leading-[1.4] text-text-faint tabular-nums group-open:hidden"
				>
					{metaOf('tiles')}
				</span>
				<ChevronDown
					size={18}
					class="ml-auto shrink-0 text-text-faint transition-transform duration-(--motion-fast) ease-standard group-open:rotate-180"
					aria-hidden="true"
				/>
			</summary>
			<div class="mt-3">
				<p
					class="settings__section-note mb-3.5 font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_tiles_note')}
				</p>
				<form class="settings__tiles flex flex-col gap-3.5" onsubmit={handleTilesSubmit}>
					<div class="settings__tiles-pair flex flex-col gap-3.5">
						<Input
							id="admin-tiles-source"
							label={$_('admin_settings_tiles_source')}
							type="url"
							autocomplete="off"
							spellcheck="false"
							class="font-mono"
							bind:value={formTilejsonUrl}
						/>
						<Input
							id="admin-tiles-dark-source"
							label={$_('admin_settings_tiles_dark_label')}
							type="url"
							autocomplete="off"
							spellcheck="false"
							class="font-mono"
							placeholder="https://…/style-dark.json"
							bind:value={formDarkTilejsonUrl}
						/>
						<p
							class="settings__field-hint m-0 font-body font-normal text-[12px] leading-[1.5] text-text-faint"
						>
							{$_('admin_settings_tiles_dark_hint')}
						</p>
					</div>
					<p
						class="settings__tiles-recommended m-0 font-body font-normal text-[12px] leading-[1.5] text-text-faint"
					>
						{$_('admin_settings_tiles_recommended')}
					</p>
					{#if tileWarning}
						<div
							class="flex items-start gap-1.5 px-3 py-2 rounded-control bg-[color-mix(in_srgb,var(--color-warning)_22%,transparent)] border border-[color-mix(in_srgb,var(--color-warning)_55%,transparent)]"
							role="status"
						>
							<TriangleAlert size={14} class="shrink-0 mt-0.5 text-warning-text" aria-hidden="true" />
							<span class="font-body font-normal text-[12px] leading-[1.5] text-text">
								{$_('admin_settings_tiles_non_wgs84_warning')}
							</span>
						</div>
					{/if}
					{#if tilesSaved}
						<p
							class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft"
							role="status"
						>
							{$_('admin_settings_tiles_saved')}
						</p>
					{/if}
					{#if tilesErrorKey !== ''}
						<p
							class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(tilesErrorKey)}
						</p>
					{/if}
					<div class="settings__form-actions flex flex-wrap items-center justify-between gap-2 mt-1">
						<Button type="button" variant="ghost" onclick={resetTileSources}
							>{$_('admin_settings_tiles_reset')}</Button
						>
						<Button type="submit" disabled={savingTiles}
							><Check size={20} aria-hidden="true" />{$_('admin_settings_tiles_save')}</Button
						>
					</div>
				</form>
				<p
					class="settings__contract mt-4 pt-4 border-t border-border font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_tiles_contract')}
				</p>
			</div>
		</details>

		<!-- The model store's policy: the automatic fill of a slot that
		     is configured but whose weights are gone. -->
		<details
			class="settings__card group bg-card border border-border rounded-card shadow-card p-5"
			data-testid="admin-settings-models"
			data-card="models"
			bind:open={expanded.models}
			ontoggle={rememberExpanded}
		>
			<summary
				class="flex cursor-pointer select-none list-none items-center gap-2 -mx-2 px-2 -my-1 py-1 rounded-control transition-colors duration-(--motion-fast) ease-standard hover:bg-accent-soft/40"
			>
				<h2
					class="settings__section-title flex items-center gap-2 font-display font-bold text-[17px] leading-[1.3] text-text"
				>
					<PackageCheck size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('admin_settings_models_title')}
				</h2>
				<span
					class="settings__section-meta ml-1 font-body font-normal text-[13px] leading-[1.4] text-text-faint tabular-nums group-open:hidden"
				>
					{metaOf('models')}
				</span>
				<ChevronDown
					size={18}
					class="ml-auto shrink-0 text-text-faint transition-transform duration-(--motion-fast) ease-standard group-open:rotate-180"
					aria-hidden="true"
				/>
			</summary>
			<div class="mt-3">
				<p
					class="settings__section-note mb-3.5 font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_models_note')}
				</p>
				<form class="settings__models flex flex-col gap-3.5" onsubmit={handleModelsSubmit}>
					<label
						class="settings__toggle flex items-center gap-2 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
					>
						<input
							class="settings__toggle-input w-[18px] h-[18px] accent-accent"
							type="checkbox"
							data-testid="models-auto-download"
							bind:checked={formAutoDownload}
						/>
						{$_('admin_settings_models_auto')}
					</label>
					{#if modelsSaved}
						<p
							class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft"
							role="status"
						>
							{$_('admin_settings_models_saved')}
						</p>
					{/if}
					{#if modelsErrorKey !== ''}
						<p
							class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(modelsErrorKey)}
						</p>
					{/if}
					<div class="settings__form-actions flex justify-end mt-1">
						<Button type="submit" disabled={savingModels}
							><Check size={20} aria-hidden="true" />{$_('admin_settings_models_save')}</Button
						>
					</div>
				</form>
				<p
					class="settings__contract mt-4 pt-4 border-t border-border font-body font-normal text-[13px] leading-[1.5] text-text-faint"
				>
					{$_('admin_settings_models_contract')}
				</p>
			</div>
		</details>
	{/if}
</div>

<!-- The rebuild dialog: the honest pair of scopes, the estimate
     riding each, the warning that names where it can be stopped. The
     dialog is a page-level layer: it opens over the collapsed index
     exactly as it opens over an expanded card. -->
<Modal
	bind:open={rebuildOpen}
	title={$_('admin_rebuild_title')}
	onclose={() => {
		rebuildErrorKey = '';
	}}
>
	<div class="flex flex-col gap-3" data-testid="rebuild-dialog">
		<label class="flex items-start gap-2 px-3 py-2.5 bg-page border border-border rounded-control cursor-pointer">
			<input
				class="mt-0.5 w-[16px] h-[16px] accent-accent"
				type="radio"
				name="rebuild-scope"
				value="missing"
				bind:group={rebuildScope}
			/>
			<span class="flex flex-col">
				<span class="font-body font-semibold text-[14px] leading-[1.4] text-text">
					{$_('admin_rebuild_scope_missing')}
				</span>
				<span class="font-body font-normal text-[12px] leading-[1.5] text-text-faint">
					{$_('admin_rebuild_scope_missing_note')}
				</span>
			</span>
		</label>
		<label class="flex items-start gap-2 px-3 py-2.5 bg-page border border-border rounded-control cursor-pointer">
			<input
				class="mt-0.5 w-[16px] h-[16px] accent-accent"
				type="radio"
				name="rebuild-scope"
				value="all"
				bind:group={rebuildScope}
			/>
			<span class="flex flex-col">
				<span class="font-body font-semibold text-[14px] leading-[1.4] text-text">
					{$_('admin_rebuild_scope_all')}
				</span>
				<span class="font-body font-normal text-[12px] leading-[1.5] text-text-faint">
					{derivativesSummary === null
						? $_('admin_rebuild_scope_all_note_unavailable')
						: $_('admin_rebuild_scope_all_note', {
								values: {
									count: derivativesSummary.imageCount,
									minutes: derivativesSummary.estimateMinutes
								}
							})}
				</span>
			</span>
		</label>
		<div
			class="flex items-start gap-1.5 px-3 py-2 rounded-control bg-[color-mix(in_srgb,var(--color-warning)_22%,transparent)] border border-[color-mix(in_srgb,var(--color-warning)_55%,transparent)]"
		>
			<TriangleAlert size={14} class="shrink-0 mt-0.5 text-warning-text" aria-hidden="true" />
			<span class="font-body font-normal text-[12px] leading-[1.5] text-text">
				{$_('admin_rebuild_warning')}
			</span>
		</div>
		{#if rebuildErrorKey !== ''}
			<p class="m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">
				{errorText(rebuildErrorKey)}
			</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (rebuildOpen = false)}>
			{$_('common_cancel')}
		</Button>
		<Button
			variant="primary"
			disabled={rebuilding}
			onclick={() => void handleRebuildStart()}
			data-testid="rebuild-start"
			aria-busy={rebuilding}
		>
			{$_('admin_rebuild_start')}
		</Button>
	{/snippet}
</Modal>
