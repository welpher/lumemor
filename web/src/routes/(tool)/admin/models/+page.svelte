<script lang="ts">
	// The model management page of the admin surface: the four
	// switchable slot cards (face detection, face recognition, the
	// semantic arm's embed tower, OCR) over the shared listing, the
	// per-slot switch dialog, the page footer's download-sources
	// panel, and the four-state semantics the cards speak.
	//
	// THE FOUR STATES (the cards' badge vocabulary): `active` is the
	// slot running its model (the badge renders the active mark);
	// `downloading` is a model download run in flight whose bytes the
	// slot waits for (the badge carries the run's percent, the bar
	// under the facts); `not_downloaded` is an unfilled slot -- the
	// mirror carries no weights, so this is the fresh instance's
	// truth, and the card's action is the download; `failed` is the
	// same missing-weights state with the last run failed, and the
	// card's action is the retry. The run is group-level: every
	// waiting slot renders the same group percent.
	//
	// THE SWITCH (the dialog): the official list is a CATALOG (the
	// manifest's validated entries, the downloaded/needs-download
	// marker -- the image carries no weights), the added group is the slot's
	// registered custom models (selecting one is a plain activation --
	// it was probed when it was imported), and a new source (a URL or
	// a local file) must PASS THE PROBE before the save switch
	// enables: the four-step pipeline (stage, probe, register, swap)
	// starts at the probe button, and a failed probe leaves nothing
	// saved. The face pair and OCR take the official list only; the
	// semantic arm carries the added group and the import panel.
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import AdminPageHeader from '$lib/components/AdminPageHeader.svelte';
	import {
		activateModel,
		deleteModel,
		listModels,
		probeModelSource,
		readModelProbe,
		rerunModel,
		startModelDownload,
		switchModelSource,
		type AdminModel,
		type CatalogEntry,
		type ModelProbe,
		type ModelSlotId,
		type SlotState
	} from '$lib/api/models';
	import { getModelSettings, saveModelSettings, type ModelSettings } from '$lib/api/adminSettings';
	import type { JobRun } from '$lib/api/jobruns';
	import { formatMegabytes } from '$lib/format';
	import { subscribeJobEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import ArrowLeftRight from '@lucide/svelte/icons/arrow-left-right';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import Download from '@lucide/svelte/icons/download';
	import Check from '@lucide/svelte/icons/check';
	import X from '@lucide/svelte/icons/x';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	/** The four card slots, the sample's order; the text tower rides
	 * the semantic card (it shares the joint space with the embed
	 * tower and has no switch of its own). */
	const CARD_SLOTS: ReadonlyArray<{ slot: ModelSlotId; labelKey: string; prepKey: string }> = [
		{ slot: 'face_detect', labelKey: 'admin_models_slot_face_detect', prepKey: 'admin_models_prep_detect' },
		{ slot: 'face_recognize', labelKey: 'admin_models_slot_face_recognize', prepKey: 'admin_models_prep_align' },
		{ slot: 'clip_embed', labelKey: 'admin_models_slot_clip_embed', prepKey: 'admin_models_prep_clip' },
		{ slot: 'ocr', labelKey: 'admin_models_slot_ocr', prepKey: 'admin_models_prep_ocr' }
	];

	let models = $state<AdminModel[]>([]);
	let slots = $state<SlotState[]>([]);
	let official = $state<CatalogEntry[]>([]);
	let download = $state<JobRun | null>(null);
	let status = $state<ListStatus>('idle');
	let errorKey = $state('');
	/** Bumping re-runs the listing (the retry button, the convergence). */
	let retryTick = $state(0);
	let busy = $state(false);
	/** The model whose rerun the confirm Modal names. */
	let pendingRerun = $state<AdminModel | null>(null);
	let rerunErrorKey = $state('');
	/** The download POST is in flight. */
	let downloading = $state(false);
	/** The download trigger's failure (the POST, not the run). */
	let downloadErrorKey = $state('');

	// The switch dialog's state: the slot it is open for, the
	// selection (an official entry id or a registered custom id), the
	// import panel's probe lifecycle, and the settings-backed removal
	// confirm.
	let dialogSlot = $state<ModelSlotId | null>(null);
	let selectedOfficial = $state<string>('');
	let selectedCustom = $state<string>('');
	let switchBusy = $state(false);
	let switchErrorKey = $state('');
	let switchRunning = $state(false);
	let wipedNotice = $state<{ from: number; to: number; count: number } | null>(null);
	// The import panel: which tab, the URL and pin fields, the file,
	// and the probe's lifecycle.
	let importTab = $state<'url' | 'file'>('url');
	let importURL = $state('');
	let importSHA = $state('');
	let importFile = $state<File | null>(null);
	let importErrorKey = $state('');
	let probeBusy = $state(false);
	let probe = $state<ModelProbe | null>(null);
	// The removal confirm (an added chip's x on the active model).
	let pendingRemove = $state<AdminModel | null>(null);
	let removeErrorKey = $state('');

	// The download-sources footer: the settings document it edits.
	let sourcesOpen = $state(false);
	let sourceSettings = $state<ModelSettings | null>(null);
	let hostInput = $state('');
	let savingSources = $state(false);
	let sourcesErrorKey = $state('');
	let sourcesSaved = $state(false);

	const settledEmpty = $derived(status === 'ready' && models.length === 0);
	/** The download run, live: queued or running. */
	const downloadActive = $derived(download !== null && (download.state === 'queued' || download.state === 'running'));
	const downloadFailed = $derived(download !== null && download.state === 'failed');

	// The listing, keyed on the retry tick. A superseded run -- the
	// retry fired again, or the route left -- never applies.
	$effect(() => {
		void retryTick;
		status = 'loading';
		errorKey = '';
		let current = true;
		void (async () => {
			const [outcome, sources] = await Promise.all([listModels(), getModelSettings()]);
			if (!current) {
				return;
			}
			if (!outcome.ok) {
				status = 'error';
				errorKey = outcome.errorKey;
				return;
			}
			models = outcome.models;
			slots = outcome.slots;
			official = outcome.official;
			download = outcome.download;
			if (sources.ok) {
				// The sources footer's summary renders from the read;
				// its own failure stays the footer's error, never the
				// listing's.
				sourceSettings = sources.settings;
				sourcesErrorKey = '';
			} else {
				sourceSettings = null;
				sourcesErrorKey = sources.errorKey;
			}
			status = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	// The SSE convergence of the download run: progress events update
	// the run in place, any start/finish converges through a refetch.
	$effect(() => {
		return subscribeJobEvents((event) => {
			if (event.kind !== 'model_download') {
				return;
			}
			if (event.name === 'job.progress') {
				const run = download;
				if (run !== null && run.id === event.jobRunId) {
					download = {
						...run,
						total: event.total,
						done: event.done,
						failed: event.failed,
						state: event.state
					};
				}
				return;
			}
			retryTick += 1;
		});
	});

	// The live-progress fallback: while the download runs, the listing
	// is re-read every 5 seconds.
	$effect(() => {
		if (!downloadActive) {
			return;
		}
		const timer = setInterval(() => {
			retryTick += 1;
		}, 5000);
		return () => clearInterval(timer);
	});

	// The open dialog's probe poll: while a probe runs, read it every
	// second until it settles, then keep the settled row.
	$effect(() => {
		const current = probe;
		if (current === null || (current.state !== 'queued' && current.state !== 'running')) {
			return;
		}
		const timer = setInterval(() => {
			void (async () => {
				const probeId = current.probeId;
				const outcome = await readModelProbe(probeId);
				if (!outcome.ok) {
					// An expired probe (404) is an honest retry state, not
					// a crash: the import panel says so and the dialog
					// stays usable.
					probe = { ...current, state: 'error', reason: $_('admin_models_probe_expired') };
					return;
				}
				probe = outcome.probe;
			})();
		}, 1000);
		return () => clearInterval(timer);
	});

	function retry(): void {
		retryTick += 1;
	}

	// The shared unknown-code fallback (see $lib/i18n/errors); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	/** The download progress bar's percentage. */
	function downloadPercent(): number {
		if (download === null || download.total <= 0) {
			return 0;
		}
		return Math.min(100, Math.round((download.done / download.total) * 100));
	}

	async function startDownload(): Promise<void> {
		if (downloading) {
			return;
		}
		downloading = true;
		downloadErrorKey = '';
		const outcome = await startModelDownload();
		downloading = false;
		if (outcome.ok) {
			retryTick += 1;
			return;
		}
		downloadErrorKey = outcome.errorKey;
	}

	/** The slot's state row, or a synthesized unfilled row. */
	function slotStateOf(slot: ModelSlotId): SlotState {
		const row = slots.find((candidate) => candidate.slot === slot);
		if (row !== undefined) {
			return row;
		}
		const unfilled: SlotState = {
			slot,
			state: 'not_downloaded',
			modelId: null,
			source: null,
			progress: null
		};
		return unfilled;
	}

	function slotLabel(slot: ModelSlotId): string {
		const entry = CARD_SLOTS.find((candidate) => candidate.slot === slot);
		return entry === undefined ? slot : $_(entry.labelKey);
	}

	/** The slot's active model row, when one is registered. */
	function activeModelOf(slot: ModelSlotId): AdminModel | null {
		return models.find((model) => model.slot === slot && model.isActive) ?? null;
	}

	/** The official catalog entries of one slot. */
	function officialOf(slot: ModelSlotId): CatalogEntry[] {
		return official.filter((entry) => entry.slot === slot);
	}

	/** The registered custom models of one slot (the added group). */
	function customsOf(slot: ModelSlotId): AdminModel[] {
		return models.filter((model) => model.slot === slot && model.source === 'custom');
	}

	/** The dialog's official selection default: the slot's first entry. */
	function officialDefaultOf(slot: ModelSlotId): string {
		const entries = officialOf(slot);
		return entries.length > 0 ? entries[0].id : '';
	}

	function selectedEntry(): CatalogEntry | null {
		if (dialogSlot === null) {
			return null;
		}
		return officialOf(dialogSlot).find((entry) => entry.id === selectedOfficial) ?? null;
	}

	function selectedCustomModel(): AdminModel | null {
		if (dialogSlot === null) {
			return null;
		}
		return customsOf(dialogSlot).find((model) => model.id === selectedCustom) ?? null;
	}

	/**
	 * The confirm's destructive pre-check: a clip entry whose
	 * dimension differs from the active model's warns (the column's
	 * dimension is only knowable server-side; the request's
	 * confirmWipe is the gate the server re-checks). Non-embed slots
	 * never warn.
	 */
	function needsDimWarning(): boolean {
		if (dialogSlot !== 'clip_embed') {
			return false;
		}
		const entry = selectedEntry();
		if (entry === null || entry.dim === undefined || entry.dim === null) {
			return false;
		}
		const active = activeModelOf('clip_embed');
		return active === null || active.dim === null || active.dim !== entry.dim;
	}

	/** The save switch's enablement: a selection must exist, the
	 * official entry must be on disk, and a pending import must have
	 * passed its probe. A filled URL or a chosen file means the save
	 * applies THE IMPORT -- it never silently switches to the
	 * preselected official entry instead. */
	function canApply(): boolean {
		if (switchRunning) {
			return false;
		}
		if (selectedCustom !== '') {
			return true;
		}
		const importing =
			(importTab === 'url' && importURL.trim() !== '') || (importTab === 'file' && importFile !== null);
		if (importing) {
			return probe !== null && (probe.state === 'ok' || probe.state === 'warn') && probe.sha256 !== null;
		}
		if (selectedEntry() !== null) {
			const entry = selectedEntry();
			return entry !== null && entry.downloaded;
		}
		return false;
	}

	function openDialog(slot: ModelSlotId): void {
		dialogSlot = slot;
		switchErrorKey = '';
		importErrorKey = '';
		importTab = 'url';
		importURL = '';
		importSHA = '';
		importFile = null;
		probe = null;
		probeBusy = false;
		// The official group preselects the entry the slot currently
		// runs (the catalog's id when it matches), else the first one.
		const active = activeModelOf(slot);
		const entries = officialOf(slot);
		const current = entries.find((entry) => entry.id === active?.id);
		selectedOfficial = current !== undefined ? current.id : officialDefaultOf(slot);
		selectedCustom = '';
	}

	function closeDialog(): void {
		if (switchRunning || switchBusy) {
			return;
		}
		dialogSlot = null;
	}

	/** The probe: the pipeline's first two steps. The four states
	 * render in the panel; a passed probe enables the save switch. */
	async function runProbe(): Promise<void> {
		if (dialogSlot === null || probeBusy) {
			return;
		}
		probeBusy = true;
		importErrorKey = '';
		probe = null;
		const slot = dialogSlot;
		let outcome;
		if (importTab === 'url') {
			if (importURL.trim() === '') {
				importErrorKey = 'admin_models_probe_need_url';
				probeBusy = false;
				return;
			}
			outcome = await probeModelSource(slot, { kind: 'url', url: importURL.trim(), sha256: importSHA.trim() });
		} else {
			if (importFile === null) {
				importErrorKey = 'admin_models_probe_need_file';
				probeBusy = false;
				return;
			}
			outcome = await probeModelSource(slot, { kind: 'file', file: importFile, sha256: importSHA.trim() });
		}
		probeBusy = false;
		if (!outcome.ok) {
			importErrorKey = outcome.errorKey;
			return;
		}
		// Seed the poll with the queued row; the effect takes over.
		probe = {
			probeId: outcome.probeId,
			state: 'queued',
			progress: null,
			dims: null,
			inputSize: null,
			sha256: null,
			hostDeclared: true,
			reason: null
		};
	}

	/** The save-and-switch: the pipeline's last two steps. The custom
	 * import rides its probeId; an added-chip selection re-activates
	 * the registered model; an official entry switches through the
	 * builtin kind with the confirmed wipe. */
	async function applySwitch(): Promise<void> {
		if (dialogSlot === null || !canApply() || switchBusy) {
			return;
		}
		const slot = dialogSlot;
		// A retry starts clean: the dialog's error from the previous
		// attempt clears before the request runs.
		switchErrorKey = '';
		switchRunning = true;
		let outcome;
		if (selectedCustom !== '') {
			const custom = selectedCustomModel();
			if (custom === null) {
				switchRunning = false;
				return;
			}
			switchBusy = true;
			const activation = await activateModel(custom.id);
			switchBusy = false;
			outcome = activation.ok
				? {
						ok: true as const,
						activated: activation.activated,
						dimensionChanged: activation.dimensionChanged,
						wipedEmbeddings: activation.wipedEmbeddings
					}
				: { ok: false as const, errorKey: activation.errorKey };
		} else if (probe !== null && (probe.state === 'ok' || probe.state === 'warn') && probe.sha256 !== null) {
			outcome = await switchModelSource(slot, {
				kind: importTab,
				probeId: probe.probeId,
				confirmWipe: needsDimWarning()
			});
		} else {
			const entry = selectedEntry();
			if (entry === null) {
				switchRunning = false;
				return;
			}
			outcome = await switchModelSource(slot, {
				kind: 'builtin',
				modelId: entry.id,
				confirmWipe: needsDimWarning()
			});
		}
		switchRunning = false;
		if (!outcome.ok) {
			switchErrorKey = outcome.errorKey;
			return;
		}
		dialogSlot = null;
		if (outcome.dimensionChanged !== undefined) {
			wipedNotice = {
				from: outcome.dimensionChanged.from,
				to: outcome.dimensionChanged.to,
				count: outcome.wipedEmbeddings ?? 0
			};
		}
		retryTick += 1;
	}

	/** The restore action: a custom model's card action switches the
	 * slot back onto the official entry (the same-bytes and
	 * same-dimension rules make this non-destructive for imports;
	 * the endpoint re-checks every gate). */
	async function restoreBuiltin(slot: ModelSlotId): Promise<void> {
		const entries = officialOf(slot);
		if (entries.length === 0) {
			return;
		}
		busy = true;
		const outcome = await switchModelSource(slot, { kind: 'builtin', modelId: entries[0].id });
		busy = false;
		if (!outcome.ok) {
			return;
		}
		retryTick += 1;
	}

	function askRerun(model: AdminModel): void {
		rerunErrorKey = '';
		pendingRerun = model;
	}

	function cancelRerun(): void {
		if (busy) {
			return;
		}
		pendingRerun = null;
		rerunErrorKey = '';
	}

	async function confirmRerun(): Promise<void> {
		const model = pendingRerun;
		if (model === null || busy) {
			return;
		}
		rerunErrorKey = '';
		busy = true;
		const outcome = await rerunModel(model.id);
		busy = false;
		if (!outcome.ok) {
			rerunErrorKey = outcome.errorKey;
			return;
		}
		pendingRerun = null;
	}

	// The download-sources footer: the settings document is read with
	// the listing (the summary line renders from it); the effect that
	// used to lazy-read it is the panel's own refresh on open.
	$effect(() => {
		if (sourcesOpen && sourceSettings === null && sourcesErrorKey !== '') {
			void (async () => {
				const outcome = await getModelSettings();
				if (outcome.ok) {
					sourceSettings = outcome.settings;
					sourcesErrorKey = '';
				}
			})();
		}
	});

	function addHost(): void {
		const host = hostInput.trim();
		if (host === '' || sourceSettings === null) {
			return;
		}
		if (!sourceSettings.allowedHosts.includes(host)) {
			sourceSettings = { ...sourceSettings, allowedHosts: [...sourceSettings.allowedHosts, host] };
		}
		hostInput = '';
		sourcesSaved = false;
	}

	function removeHost(host: string): void {
		if (sourceSettings === null) {
			return;
		}
		sourceSettings = {
			...sourceSettings,
			allowedHosts: sourceSettings.allowedHosts.filter((candidate) => candidate !== host)
		};
		sourcesSaved = false;
	}

	async function saveSources(): Promise<void> {
		if (sourceSettings === null || savingSources) {
			return;
		}
		savingSources = true;
		sourcesErrorKey = '';
		const outcome = await saveModelSettings({
			...sourceSettings,
			autoDownload: sourceSettings.autoDownload,
			mirrorBaseUrl: sourceSettings.mirrorBaseUrl.trim()
		});
		savingSources = false;
		if (outcome.ok) {
			sourceSettings = outcome.settings;
			sourcesSaved = true;
			retryTick += 1;
			return;
		}
		sourcesErrorKey = outcome.errorKey;
	}

	/** The footer's summary line: N hosts, the mirror host. */
	function sourcesSummary(): string {
		if (sourceSettings === null) {
			return $_('admin_models_sources_loading');
		}
		const mirror = sourceSettings.mirrorBaseUrl !== '' ? ` · ${sourceSettings.mirrorBaseUrl}` : '';
		return $_('admin_models_sources_summary', { values: { count: sourceSettings.allowedHosts.length } }) + mirror;
	}

	/** The removal confirm: an added chip's x on the ACTIVE model
	 * falls back to the official default and re-runs. */
	function askRemove(model: AdminModel): void {
		removeErrorKey = '';
		pendingRemove = model;
	}

	function cancelRemove(): void {
		if (busy) {
			return;
		}
		pendingRemove = null;
		removeErrorKey = '';
	}

	/** The removal of a custom model that is not in use: the
	 * registration and its content-named directory go; the slot's
	 * running model is untouched. */
	async function removeCustom(model: AdminModel): Promise<void> {
		busy = true;
		const outcome = await deleteModel(model.id);
		busy = false;
		if (!outcome.ok) {
			return;
		}
		retryTick += 1;
	}

	async function confirmRemove(): Promise<void> {
		const model = pendingRemove;
		if (model === null || busy || dialogSlot === null) {
			return;
		}
		removeErrorKey = '';
		busy = true;
		const entries = officialOf(dialogSlot);
		if (entries.length === 0) {
			busy = false;
			removeErrorKey = 'internal';
			return;
		}
		// The fallback first (the official default becomes the slot's
		// running model), then the removal: the running slot never
		// points at a deleted registration.
		const fallback = await switchModelSource(dialogSlot, { kind: 'builtin', modelId: entries[0].id });
		if (!fallback.ok) {
			busy = false;
			removeErrorKey = fallback.errorKey;
			return;
		}
		const outcome = await deleteModel(model.id);
		busy = false;
		if (!outcome.ok) {
			removeErrorKey = outcome.errorKey;
			retryTick += 1;
			return;
		}
		pendingRemove = null;
		retryTick += 1;
	}

	/** The sha line's short form: the first eight hex chars. */
	function shaShort(sha: string | null): string {
		return sha === null || sha.length < 8 ? '' : sha.slice(0, 8);
	}

	/** The contract line's facts for one slot: the active model's
	 * geometry (the fact the slot's card shows) and the official
	 * entry's dimension. */
	function contractLine(slot: ModelSlotId): string {
		const active = activeModelOf(slot);
		const entries = officialOf(slot);
		const entry = entries[0];
		const width = active?.inputSize?.width ?? null;
		const height = active?.inputSize?.height ?? null;
		const dim = active?.dim ?? entry?.dim ?? null;
		const prep = $_(CARD_SLOTS.find((candidate) => candidate.slot === slot)?.prepKey ?? '');
		const parts: string[] = [];
		if (width !== null && height !== null) {
			parts.push($_('admin_models_contract_input', { values: { size: `${width}×${height}` } }));
		}
		if (dim !== null && dim !== undefined) {
			parts.push($_('admin_models_contract_output', { values: { dim } }));
		}
		if (prep !== '') {
			parts.push(prep);
		}
		return parts.join(' · ');
	}
</script>

<svelte:head>
	<title>Models · lumemor</title>
</svelte:head>

<div class="models max-w-[760px] mx-auto flex flex-col gap-6">
	{#if status === 'loading'}
		<p class="models__status my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('admin_models_loading')}
		</p>
	{:else if status === 'error'}
		<div
			class="models__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}>
				<RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}
			</Button>
		</div>
	{:else if status === 'ready'}
		<AdminPageHeader title={$_('admin_models_title')} />
		<p class="models__note m-0 mb-1 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
			{$_('admin_models_note')}
		</p>

		{#if wipedNotice !== null}
			<p
				class="models__wiped px-4 py-3 font-body font-medium text-[14px] leading-[1.5] text-danger-text bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)] border border-danger rounded-card"
				role="status"
			>
				{$_('admin_models_wiped_notice', { values: wipedNotice })}
			</p>
		{/if}

		{#if settledEmpty && !downloadActive && !downloadFailed && downloadErrorKey === ''}
			<section
				class="models__empty flex flex-col items-center gap-3 max-w-[480px] box-content mx-auto mt-6 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card"
			>
				<h2 class="models__empty-title font-display font-bold text-[22px] text-text">
					{$_('admin_models_empty_title')}
				</h2>
				<p class="models__empty-body font-body font-normal text-[14px] leading-[1.5] text-text-soft">
					{$_('admin_models_empty_body')}
				</p>
				<Button variant="primary" disabled={downloading} onclick={() => void startDownload()}>
					<Download size={20} aria-hidden="true" />
					{$_('admin_models_download_cta')}
				</Button>
			</section>
		{/if}

		{#if downloadErrorKey !== ''}
			<div
				class="models__download-error flex flex-col items-center gap-3 max-w-[480px] box-content mx-auto mt-6 p-5 text-center border border-danger rounded-card text-danger-text"
				role="alert"
			>
				<p>{errorText(downloadErrorKey)}</p>
				<Button variant="primary" disabled={downloading} onclick={() => void startDownload()}>
					<RotateCcw size={20} aria-hidden="true" />
					{$_('action_retry')}
				</Button>
			</div>
		{/if}

		{#if downloadActive || downloadFailed}
			<section
				class="models__progress-region flex flex-col gap-2.5 p-5 bg-card border border-border rounded-card shadow-card"
				aria-live="polite"
			>
				{#if downloadActive}
					<p
						class="models__progress-state font-body font-medium text-[14px] leading-[1.4] text-text tabular-nums"
					>
						{download?.state === 'queued'
							? $_('admin_models_download_queued')
							: $_('admin_models_download_progress', {
									values: {
										done: formatMegabytes(download?.done ?? 0),
										total: formatMegabytes(download?.total ?? 0)
									}
								})}
					</p>
					<div
						class="models__progress w-full h-2 bg-accent-soft rounded-control overflow-hidden animate-progress-in"
						role="progressbar"
						aria-valuemin="0"
						aria-valuemax="100"
						aria-valuenow={downloadPercent()}
					>
						<div
							class="models__progress-fill h-full bg-accent transition-[width] duration-(--motion-base) ease-standard"
							style="width: {downloadPercent()}%"
						></div>
					</div>
				{:else if downloadFailed}
					<p
						class="models__download-failed font-body font-medium text-[14px] leading-[1.4] text-danger-text"
						role="alert"
					>
						{$_('admin_models_download_failed_run', {
							values: { count: download?.failed ?? 0 }
						})}
					</p>
					<Button variant="primary" disabled={downloading} onclick={() => void startDownload()}>
						<RotateCcw size={20} aria-hidden="true" />
						{$_('action_retry')}
					</Button>
				{/if}
			</section>
		{/if}

		<!-- The four switchable cards, one per pipeline slot. -->
		<div class="models__cards flex flex-col gap-4">
			{#each CARD_SLOTS as card (card.slot)}
				{@const state = slotStateOf(card.slot)}
				{@const active = activeModelOf(card.slot)}
				<article
					class="models__card flex flex-col gap-2 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
					data-slot={card.slot}
				>
					<header class="models__card-head flex flex-wrap items-center gap-2">
						<h2 class="models__card-title m-0 font-display font-bold text-[16px] leading-[1.3] text-text">
							{$_(card.labelKey)}
						</h2>
						<span
							class="models__badge inline-block px-2.5 py-0.5 font-body font-semibold text-[12px] leading-[1.6] rounded-control"
							class:bg-accent-soft={state.state === 'active'}
							class:text-accent-soft-text={state.state === 'active'}
							class:text-text-soft={state.state === 'downloading'}
							class:bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)]={state.state === 'failed'}
							class:text-danger-text={state.state === 'failed'}
							class:border={state.state === 'not_downloaded'}
							class:border-border={state.state === 'not_downloaded'}
							class:text-text-faint={state.state === 'not_downloaded'}
							data-state={state.state}
						>
							{#if state.state === 'active'}
								{$_('admin_models_active')}
							{:else if state.state === 'downloading'}
								{$_('admin_models_state_downloading', {
									values: {
										percent:
											state.progress !== null && state.progress.total > 0
												? Math.min(
														100,
														Math.round((state.progress.done / state.progress.total) * 100)
													)
												: 0
									}
								})}
							{:else if state.state === 'failed'}
								{$_('admin_models_state_failed')}
							{:else}
								{$_('admin_models_state_not_downloaded')}
							{/if}
						</span>
					</header>
					<p
						class="models__facts m-0 flex flex-wrap items-center gap-x-4 gap-y-1 font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
					>
						{#if active !== null}
							<span>{active.id}</span>
							<span>{$_('admin_models_version', { values: { version: active.version } })}</span>
							{#if active.dim !== null}
								<span>{$_('admin_models_dim', { values: { dim: active.dim } })}</span>
							{/if}
							<span class="text-text-faint">{$_('admin_models_license')} {active.license}</span>
						{:else if state.source === null}
							<span class="text-text-faint">{$_('admin_models_card_empty')}</span>
						{/if}
					</p>
					{#if active !== null && active.source === 'custom' && state.source !== null}
						<p
							class="models__custom-meta m-0 flex flex-wrap items-center gap-x-3 gap-y-1 font-body font-normal text-[12px] leading-[1.5] text-text-faint tabular-nums"
						>
							<span class="font-semibold text-text-soft">{$_('admin_models_custom_badge')}</span>
							{#if state.source.host !== null}
								<span>{$_('admin_models_custom_host', { values: { host: state.source.host } })}</span>
							{/if}
							<span
								>{$_('admin_models_custom_downloaded_at', {
									values: { date: new Date(state.source.downloadedAt).toLocaleDateString() }
								})}</span
							>
							{#if state.source.sha256 !== null}
								<span>sha256 {shaShort(state.source.sha256)}</span>
							{/if}
						</p>
					{/if}
					{#if state.state === 'downloading' && state.progress !== null}
						<div
							class="models__card-progress h-2 bg-accent-soft rounded-control overflow-hidden"
							role="progressbar"
							aria-valuemin="0"
							aria-valuemax="100"
							aria-valuenow={state.progress.total > 0
								? Math.min(100, Math.round((state.progress.done / state.progress.total) * 100))
								: 0}
						>
							<div
								class="models__card-progress-fill h-full bg-accent transition-[width] duration-(--motion-base) ease-standard"
								style="width: {state.progress.total > 0
									? Math.min(100, Math.round((state.progress.done / state.progress.total) * 100))
									: 0}%"
							></div>
						</div>
					{/if}
					{#if state.state === 'failed'}
						<p
							class="models__card-failed m-0 font-body font-normal text-[13px] leading-[1.5] text-danger-text"
						>
							{$_('admin_models_state_failed_note')}
						</p>
					{/if}
					{#if state.state === 'not_downloaded'}
						<p
							class="models__card-empty m-0 font-body font-normal text-[13px] leading-[1.5] text-text-faint"
						>
							{$_('admin_models_card_not_downloaded_note')}
						</p>
					{/if}
					<div class="models__card-actions flex flex-wrap items-center gap-2">
						<Button
							variant="primary"
							disabled={busy}
							onclick={() => openDialog(card.slot)}
							data-slot={card.slot}
						>
							<ArrowLeftRight size={20} aria-hidden="true" />
							{$_('admin_models_switch')}
						</Button>
						{#if state.state === 'active' && active !== null}
							<Button variant="ghost" disabled={busy} onclick={() => askRerun(active)}>
								<RefreshCw size={18} aria-hidden="true" />
								{$_('admin_models_rerun')}
							</Button>
							{#if active.source === 'custom'}
								<Button variant="ghost" disabled={busy} onclick={() => void restoreBuiltin(card.slot)}>
									<RotateCcw size={18} aria-hidden="true" />
									{$_('admin_models_restore')}
								</Button>
							{/if}
						{:else if state.state === 'not_downloaded' || state.state === 'failed'}
							<Button variant="primary" disabled={downloading} onclick={() => void startDownload()}>
								<Download size={18} aria-hidden="true" />
								{state.state === 'failed' ? $_('action_retry') : $_('admin_models_download_now')}
							</Button>
						{/if}
					</div>
				</article>
			{/each}
		</div>

		<!-- The download-sources footer: the collapsed rescue panel. -->
		<details class="models__sources group flex flex-col gap-3" bind:open={sourcesOpen} data-testid="models-sources">
			<summary
				class="flex cursor-pointer select-none list-none items-baseline gap-2 [&::-webkit-details-marker]:hidden"
			>
				<h2 class="models__sources-title m-0 font-display font-bold text-[17px] leading-[1.3] text-text">
					{$_('admin_models_sources_title')}
				</h2>
				<span
					class="models__sources-summary font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
					data-testid="models-sources-summary"
				>
					{sourcesSummary()}
				</span>
				<svg
					aria-hidden="true"
					width="18"
					height="18"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					class="shrink-0 text-text-faint transition-transform duration-(--motion-fast) ease-standard group-open:rotate-180"
				>
					<path d="m6 9 6 6 6-6" />
				</svg>
			</summary>
			<div
				class="models__sources-panel flex flex-col gap-3 py-4 px-4.5 bg-card border border-border rounded-card shadow-card"
			>
				{#if sourceSettings === null}
					<p class="models__sources-loading m-0 font-body font-normal text-[13px] text-text-soft">
						{sourcesErrorKey !== '' ? errorText(sourcesErrorKey) : $_('admin_models_sources_loading')}
					</p>
				{:else}
					<p class="models__sources-intro m-0 font-body font-normal text-[13px] leading-[1.5] text-text-soft">
						{$_('admin_models_sources_intro')}
					</p>
					<div
						class="models__sources-warning flex items-start gap-1.5 px-3 py-2 rounded-control bg-[color-mix(in_srgb,var(--color-warning)_22%,transparent)] border border-[color-mix(in_srgb,var(--color-warning)_55%,transparent)]"
					>
						<TriangleAlert size={14} aria-hidden="true" class="mt-0.5 shrink-0 text-warning-text" />
						<span class="font-body font-normal text-[13px] leading-[1.5] text-text">
							{$_('admin_models_sources_warning')}
						</span>
					</div>
					<div class="models__host-list flex flex-wrap gap-1.5">
						{#each sourceSettings.allowedHosts as host (host)}
							<span
								class="models__host inline-flex items-center gap-1 px-2 py-0.5 font-mono font-normal text-[12px] leading-[1.6] text-text-soft bg-page border border-border rounded-control"
							>
								{host}
								<button
									class="models__host-remove inline-flex items-center justify-center w-4 h-4 bg-transparent border-0 text-text-faint cursor-pointer hover:text-danger-text"
									type="button"
									aria-label={$_('admin_models_host_remove', { values: { host } })}
									onclick={() => removeHost(host)}
								>
									<X size={12} aria-hidden="true" />
								</button>
							</span>
						{/each}
					</div>
					<div class="models__host-add flex flex-wrap items-center gap-2">
						<input
							class="models__host-input min-w-[220px] flex-1 px-3 py-[9px] font-body font-normal text-[14px] leading-[1.4] text-text bg-page border border-border-strong rounded-control"
							type="text"
							placeholder={$_('admin_models_host_placeholder')}
							aria-label={$_('admin_models_host_add')}
							bind:value={hostInput}
						/>
						<Button variant="primary" onclick={addHost}>
							<Plus size={18} aria-hidden="true" />
							{$_('admin_models_host_add')}
						</Button>
					</div>
					<div class="models__mirror flex flex-wrap items-end gap-2">
						<label class="models__mirror-field flex min-w-[280px] flex-1 flex-col gap-1.5">
							<span
								class="models__mirror-label font-body font-semibold text-[14px] leading-[1.4] text-text"
							>
								{$_('admin_models_mirror_base')}
							</span>
							<input
								class="models__mirror-input px-3 py-[9px] font-body font-normal text-[14px] leading-[1.4] text-text bg-page border border-border-strong rounded-control"
								type="text"
								bind:value={sourceSettings.mirrorBaseUrl}
							/>
						</label>
						<Button variant="primary" disabled={savingSources} onclick={() => void saveSources()}>
							<Check size={18} aria-hidden="true" />
							{$_('action_save')}
						</Button>
					</div>
					{#if sourcesSaved}
						<p
							class="models__sources-saved m-0 font-body font-normal text-[13px] text-success"
							role="status"
						>
							{$_('admin_models_sources_saved')}
						</p>
					{/if}
					{#if sourcesErrorKey !== ''}
						<p
							class="models__sources-error m-0 font-body font-medium text-[13px] text-danger-text"
							role="alert"
						>
							{errorText(sourcesErrorKey)}
						</p>
					{/if}
					<p
						class="models__sources-fineprint m-0 font-body font-normal text-[12px] leading-[1.5] text-text-faint"
					>
						{$_('admin_models_sources_fineprint')}
					</p>
				{/if}
			</div>
		</details>
	{/if}
</div>

<!-- The switch dialog: the slot contract, the two-group select, the
     import panel and the probe's four states. -->
<Modal
	open={dialogSlot !== null}
	title={dialogSlot === null ? '' : $_('admin_models_switch_title', { values: { slot: slotLabel(dialogSlot) } })}
	onclose={closeDialog}
>
	{#if dialogSlot !== null}
		<div class="models__dialog flex flex-col gap-4">
			<div class="models__contract flex flex-col gap-1 rounded-control border border-border bg-page px-3 py-2.5">
				<span class="models__contract-label font-body font-semibold text-[12px] leading-[1.4] text-text-soft">
					{$_('admin_models_slot_contract')}
				</span>
				<span
					class="models__contract-facts font-body font-normal text-[13px] leading-[1.5] text-text tabular-nums"
				>
					{contractLine(dialogSlot)}
				</span>
				<span class="models__contract-note font-body font-normal text-[12px] leading-[1.5] text-danger-text">
					{#if dialogSlot === 'clip_embed'}
						{$_('admin_models_contract_wipe')}
					{:else if dialogSlot === 'face_detect'}
						{$_('admin_models_contract_detect')}
					{:else if dialogSlot === 'face_recognize'}
						{$_('admin_models_contract_recognize')}
					{:else}
						{$_('admin_models_contract_ocr')}
					{/if}
				</span>
			</div>

			<div class="models__select flex flex-col gap-2">
				<span class="models__select-label font-body font-semibold text-[13px] leading-[1.4] text-text">
					{$_('admin_models_models')}
				</span>
				<select
					class="models__official-select w-full px-3 py-[9px] pe-9 font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control"
					aria-label={$_('admin_models_models')}
					bind:value={selectedOfficial}
					onchange={() => (selectedCustom = '')}
				>
					<optgroup label={$_('admin_models_group_official')}>
						{#each officialOf(dialogSlot) as entry (entry.id)}
							<option value={entry.id}>
								{entry.name} · {entry.license} · {entry.downloaded
									? $_('admin_models_catalog_downloaded')
									: $_(
											'admin_models_catalog_need_download'
										)}{#if entry.dim !== null && entry.dim !== undefined}
									· {$_('admin_models_dim', { values: { dim: entry.dim } })}{/if}
							</option>
						{/each}
					</optgroup>
					{#if dialogSlot === 'clip_embed' && customsOf(dialogSlot).length > 0}
						<optgroup label={$_('admin_models_group_custom')}>
							{#each customsOf(dialogSlot) as custom (custom.id)}
								<option value={custom.id}>
									{custom.id} · {$_('admin_models_custom_badge')} · {custom.filesPresent
										? $_('admin_models_catalog_downloaded')
										: $_('admin_models_catalog_need_download')}
								</option>
							{/each}
						</optgroup>
					{/if}
				</select>
				<div class="models__custom-chips flex flex-wrap items-center gap-1.5">
					{#each customsOf(dialogSlot) as custom (custom.id)}
						<span
							class="models__chip inline-flex items-center gap-1 px-2 py-0.5 font-body font-normal text-[12px] leading-[1.6] text-text-soft bg-page border border-border rounded-control"
						>
							{custom.id}
							{#if custom.filesPresent}
								<span class="text-text-faint">· {shaShort(custom.id.split('@')[1] ?? null) || ''}</span>
							{/if}
							<button
								class="models__chip-remove inline-flex items-center justify-center w-4 h-4 bg-transparent border-0 text-text-faint cursor-pointer hover:text-danger-text"
								type="button"
								aria-label={$_('admin_models_remove', { values: { name: custom.id } })}
								onclick={() => (custom.isActive ? askRemove(custom) : void removeCustom(custom))}
							>
								<X size={12} aria-hidden="true" />
							</button>
						</span>
					{/each}
				</div>
			</div>

			{#if dialogSlot === 'clip_embed'}
				<div class="models__import flex flex-col gap-2">
					<div class="models__import-head flex items-center gap-2 pt-1 border-t border-border">
						<span
							class="models__import-title pt-2 font-body font-semibold text-[13px] leading-[1.4] text-text"
						>
							{$_('admin_models_import_title')}
						</span>
						<span
							class="models__import-hint pt-2 font-body font-normal text-[12px] leading-[1.4] text-text-faint"
						>
							{$_('admin_models_import_hint')}
						</span>
					</div>
					<div
						class="models__import-tabs flex gap-1 p-1 bg-page border border-border rounded-control"
						role="tablist"
						aria-label={$_('admin_models_import_title')}
					>
						<button
							class="models__import-tab flex-1 inline-flex items-center justify-center gap-2 px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control bg-transparent border-0 text-text-soft cursor-pointer aria-selected:bg-card aria-selected:text-text aria-selected:shadow-card"
							type="button"
							role="tab"
							aria-selected={importTab === 'url'}
							onclick={() => (importTab = 'url')}
						>
							{$_('admin_models_source_url')}
						</button>
						<button
							class="models__import-tab flex-1 inline-flex items-center justify-center gap-2 px-3 py-1.5 font-body font-semibold text-[13px] leading-[1.4] rounded-control bg-transparent border-0 text-text-soft cursor-pointer aria-selected:bg-card aria-selected:text-text aria-selected:shadow-card"
							type="button"
							role="tab"
							aria-selected={importTab === 'file'}
							onclick={() => (importTab = 'file')}
						>
							{$_('admin_models_source_file')}
						</button>
					</div>
					{#if importTab === 'url'}
						<div class="models__import-url flex flex-col gap-2">
							<label class="models__field flex flex-col gap-1.5">
								<span
									class="models__field-label font-body font-semibold text-[14px] leading-[1.4] text-text"
								>
									{$_('admin_models_import_url_label')}
								</span>
								<input
									class="models__url-input px-3 py-[9px] font-body font-normal text-[14px] leading-[1.4] text-text bg-page border border-border-strong rounded-control"
									type="url"
									placeholder="https://…/model.onnx"
									bind:value={importURL}
								/>
							</label>
							<label class="models__field flex flex-col gap-1.5">
								<span
									class="models__field-label font-body font-semibold text-[14px] leading-[1.4] text-text"
								>
									{$_('admin_models_import_sha_label')}
								</span>
								<input
									class="models__sha-input px-3 py-[9px] font-body font-normal text-[14px] leading-[1.4] text-text bg-page border border-border-strong rounded-control"
									type="text"
									placeholder="64"
									bind:value={importSHA}
								/>
							</label>
						</div>
					{:else}
						<label
							class="models__drop flex flex-col items-center justify-center gap-1 px-4 py-6 text-center bg-page border border-dashed border-border-strong rounded-control cursor-pointer"
						>
							<span
								class="models__drop-title font-body font-semibold text-[14px] leading-[1.4] text-text"
							>
								{importFile === null ? $_('admin_models_import_file_hint') : importFile.name}
							</span>
							<span
								class="models__drop-note font-body font-normal text-[12px] leading-[1.5] text-text-faint"
							>
								{$_('admin_models_import_file_bound')}
							</span>
							<input
								class="hidden"
								type="file"
								accept=".onnx"
								onchange={(event) => {
									const files = event.currentTarget.files;
									importFile = files !== null && files.length > 0 ? files[0] : null;
								}}
							/>
						</label>
					{/if}
				</div>
			{/if}

			<!-- The probe's four states. -->
			{#if probe !== null || importErrorKey !== ''}
				{#if importErrorKey !== ''}
					<p
						class="models__probe-error m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
						role="alert"
					>
						{errorText(importErrorKey)}
					</p>
				{:else if probe !== null && (probe.state === 'queued' || probe.state === 'running')}
					<div
						class="models__probe-running flex flex-col gap-2 rounded-control border border-border bg-page px-3 py-2.5"
						data-testid="probe-state"
						data-probe-state="running"
					>
						<div class="flex items-center gap-2 font-body text-[13px] leading-[1.4] text-text-soft">
							<span
								class="inline-block w-3.5 h-3.5 rounded-full border-2 border-accent border-t-transparent animate-spin"
								aria-hidden="true"
							></span>
							<span class="tabular-nums">
								{$_('admin_models_probe_running')}
								{#if probe.state === 'running' && probe.progress !== null && probe.progress.total > 0}
									{Math.min(100, Math.round((probe.progress.done / probe.progress.total) * 100))}%
								{/if}
							</span>
						</div>
						{#if probe.state === 'running' && probe.progress !== null && probe.progress.total > 0}
							<div class="h-1.5 bg-accent-soft rounded-control overflow-hidden">
								<div
									class="h-full bg-accent transition-[width] duration-(--motion-base) ease-standard"
									style="width: {Math.min(
										100,
										Math.round((probe.progress.done / probe.progress.total) * 100)
									)}%"
								></div>
							</div>
						{/if}
					</div>
				{:else if probe !== null && probe.state === 'ok'}
					<div
						class="models__probe-ok flex flex-col gap-1 rounded-control border border-success/40 bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] px-3 py-2.5"
						data-testid="probe-state"
						data-probe-state="ok"
					>
						<span class="font-body font-semibold text-[13px] leading-[1.4] text-text"
							>{$_('admin_models_probe_ok')}</span
						>
						<span class="font-body font-normal text-[12px] leading-[1.5] text-text-soft tabular-nums">
							{probe.dims !== null
								? $_('admin_models_probe_ok_detail', { values: { dim: probe.dims } })
								: ''}
							{#if probe.sha256 !== null}
								· sha256 {$_('admin_models_probe_sha_checked')}{/if}
						</span>
					</div>
				{:else if probe !== null && probe.state === 'warn'}
					<div
						class="models__probe-warn flex flex-col gap-1 rounded-control border border-[color-mix(in_srgb,var(--color-warning)_60%,transparent)] bg-[color-mix(in_srgb,var(--color-warning)_22%,transparent)] px-3 py-2.5"
						data-testid="probe-state"
						data-probe-state="warn"
					>
						<span class="font-body font-semibold text-[13px] leading-[1.4] text-text"
							>{$_('admin_models_probe_warn')}</span
						>
						<span class="font-body font-normal text-[12px] leading-[1.5] text-text-soft">
							{$_('admin_models_probe_warn_detail')}
						</span>
					</div>
				{:else if probe !== null && probe.state === 'error'}
					<div
						class="models__probe-failed flex flex-col gap-1 rounded-control border border-danger/40 bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)] px-3 py-2.5"
						data-testid="probe-state"
						data-probe-state="error"
					>
						<span class="font-body font-semibold text-[13px] leading-[1.4] text-danger-text"
							>{$_('admin_models_probe_failed')}</span
						>
						<span class="font-body font-normal text-[12px] leading-[1.5] text-text-soft"
							>{probe.reason}</span
						>
					</div>
				{/if}
			{/if}

			{#if switchErrorKey !== ''}
				<p
					class="models__switch-error m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
					role="alert"
				>
					{errorText(switchErrorKey)}
				</p>
			{/if}

			<div
				class="models__trust flex items-start gap-1.5 px-3 py-2 rounded-control bg-[color-mix(in_srgb,var(--color-warning)_22%,transparent)] border border-[color-mix(in_srgb,var(--color-warning)_55%,transparent)]"
			>
				<TriangleAlert size={14} aria-hidden="true" class="mt-0.5 shrink-0 text-warning-text" />
				<span class="font-body font-normal text-[12px] leading-[1.5] text-text">
					{$_('admin_models_trust')}
				</span>
			</div>
		</div>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={switchRunning || switchBusy} onclick={closeDialog}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="primary" disabled={probeBusy || dialogSlot !== 'clip_embed'} onclick={() => void runProbe()}>
			<Check size={18} aria-hidden="true" />
			{probeBusy ? $_('admin_models_probe_running') : $_('admin_models_probe')}
		</Button>
		<Button variant="primary" disabled={!canApply()} onclick={() => void applySwitch()}>
			<ArrowLeftRight size={18} aria-hidden="true" />
			{switchRunning ? $_('admin_models_switch_running') : $_('admin_models_save_and_switch')}
		</Button>
	{/snippet}
</Modal>

<!-- The removal confirm: removing the ACTIVE custom model falls back
     to the official default and re-runs the slot. -->
<Modal open={pendingRemove !== null} title={$_('admin_models_remove_active_title')} onclose={cancelRemove}>
	<p class="models__remove-body font-body font-normal text-[15px] leading-[1.5] text-text">
		{$_('admin_models_remove_active_warning', { values: { name: pendingRemove?.id ?? '' } })}
	</p>
	{#if removeErrorKey !== ''}
		<p
			class="models__remove-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(removeErrorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelRemove}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="danger" disabled={busy} onclick={() => void confirmRemove()}>
			<Trash2 size={20} aria-hidden="true" />
			{$_('admin_models_remove_confirm')}
		</Button>
	{/snippet}
</Modal>

<Modal open={pendingRerun !== null} title={$_('admin_models_rerun_title')} onclose={cancelRerun}>
	<p class="models__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
		{#if pendingRerun !== null}
			{$_('admin_models_rerun_body', { values: { slot: slotLabel(pendingRerun.slot) } })}
		{/if}
	</p>
	{#if rerunErrorKey !== ''}
		<p
			class="models__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(rerunErrorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelRerun}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="primary" disabled={busy} onclick={() => void confirmRerun()}>
			<RefreshCw size={20} aria-hidden="true" />
			{$_('admin_models_rerun_confirm')}
		</Button>
	{/snippet}
</Modal>
