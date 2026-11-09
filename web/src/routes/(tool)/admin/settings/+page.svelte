<script lang="ts">
	// The admin settings page (the 2026-11-09 specimen's geometry:
	// the page-header row with its hairline, the centered 760
	// column): the three settings sections -- the storage template
	// (the storage-template feature: the instance's upload path
	// template with the built-in presets filling the input, a live
	// sample preview of the SAVED template, and the save's
	// validation errors -- the shared i18n text plus the server's
	// specific reason line, the reason naming the admin's own
	// template text; no translation table can pre-render it), the
	// trash retention (the settings feature: whole days 0-365, 0
	// being immediate permanent delete, with the destructive
	// warning line), and the video transcoding profile (the preset,
	// the concurrency bound, the poster-frame switch and the failure
	// policy; the contract's new-ingests-only scope note). E4's
	// mount-root allowlist section is GONE: the allowlist retired
	// whole with E6 (web IA §7.5's third revision -- the boundary is
	// the compose mount itself, nothing left to list).
	//
	// THE 403 PAGE: a signed-in non-admin landing on this route sees
	// the mapped auth_forbidden message -- the session contract
	// carries no role, so this page's own read is the probe.
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import Input from '$lib/components/Input.svelte';
	import AdminPageHeader from '$lib/components/AdminPageHeader.svelte';
	import {
		getStorageSettings,
		getTrashSettings,
		getTranscodingSettings,
		saveStorageSettings,
		saveTrashSettings,
		saveTranscodingSettings,
		type StorageSettings,
		type TranscodingSettings
	} from '$lib/api/adminSettings';
	import { resolveErrorText } from '$lib/i18n/errors';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import Check from '@lucide/svelte/icons/check';
	import FolderTree from '@lucide/svelte/icons/folder-tree';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import MonitorPlay from '@lucide/svelte/icons/monitor-play';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

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

	// The read, keyed on the retry tick. A superseded run -- the
	// retry fired again, or the route left -- never applies.
	$effect(() => {
		void retryTick;
		status = 'loading';
		errorKey = '';
		let current = true;
		void (async () => {
			const [storage, trash, transcoding] = await Promise.all([
				getStorageSettings(),
				getTrashSettings(),
				getTranscodingSettings()
			]);
			if (!current) {
				return;
			}
			// The three reads share one gate: they are one page, and
			// the admin-role probe belongs to whichever read answers
			// first (any failure -- the role refusal, a session loss,
			// an unparseable body -- is the page's failure).
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
			adoptSettings(storage.settings, trash.settings.retentionDays, transcoding.settings);
			status = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	/** Adopt the three read documents as the sections' fresh state. */
	function adoptSettings(
		next: StorageSettings,
		retentionDays: number,
		transcoding: TranscodingSettings
	): void {
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
</script>

<svelte:head>
	<title>Settings · lumemor</title>
</svelte:head>

<div class="settings max-w-[760px] mx-auto flex flex-col gap-4">
	{#if status === 'loading'}
		<p class="settings__status my-12 text-center text-text-soft animate-loading-in" role="status">{$_('admin_settings_loading')}</p>
	{:else if status === 'error'}
		<div class="settings__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
		</div>
	{:else if status === 'ready' && settings !== null}
		<!-- The page-header row (the 2026-11-09 specimen: the title
		     echoing the rail's active section, the right action slot
		     -- settings carries no page-level action, each section
		     owns its save, but the row stands for uniformity -- and
		     the hairline that pins the row into a belt with the
		     rail's border). -->
		<AdminPageHeader title={$_('admin_nav_settings')} />

		<section class="settings__card" data-testid="admin-settings-storage">
			<Card>
				<h2 class="settings__section-title flex items-center gap-2 mb-1 font-display font-bold text-[17px] text-text">
					<FolderTree size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('admin_settings_storage_title')}
				</h2>
				<p class="settings__section-note mb-3.5 font-body font-normal text-[13px] leading-[1.5] text-text-faint">{$_('admin_settings_storage_note')}</p>
				<form class="settings__form flex flex-col gap-3.5" onsubmit={handleSubmit}>
					<div class="settings__field flex flex-col gap-1.5">
						<label class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text" for="admin-storage-preset">{$_('admin_settings_preset_label')}</label>
						<select
							class="settings__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
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
						<p class="settings__sample-label font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint">{$_('admin_settings_sample_label')}</p>
						<p class="settings__sample-path font-mono font-normal text-[13px] leading-[1.5] text-text-soft break-all">{settings.samplePath}</p>
					</div>
					{#if savedNote}
						<p class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft" role="status">{$_('admin_settings_saved_note')}</p>
					{/if}
					{#if saveErrorKey !== ''}
						<div class="settings__error flex flex-col gap-0.5" role="alert">
							<p class="font-body font-medium text-[14px] leading-[1.4] text-danger-text">{errorText(saveErrorKey)}</p>
							{#if saveReason !== ''}
								<p class="settings__error-reason font-mono font-normal text-[13px] leading-[1.4] text-danger-text break-all" data-testid="storage-error-reason">{saveReason}</p>
							{/if}
						</div>
					{/if}
					<div class="settings__form-actions flex justify-end mt-1">
						<Button type="submit" disabled={saving}><Check size={20} aria-hidden="true" />{$_('admin_settings_submit')}</Button>
					</div>
				</form>
				<p class="settings__contract mt-4 pt-4 border-t border-border font-body font-normal text-[13px] leading-[1.5] text-text-faint">{$_('admin_settings_contract_note')}</p>
			</Card>
		</section>

		<!-- The trash retention section: the number input, the save,
		     and the destructive warning line. -->
		<section class="settings__card" data-testid="admin-settings-trash">
			<Card>
				<h2 class="settings__section-title flex items-center gap-2 mb-1 font-display font-bold text-[17px] text-text">
					<Trash2 size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('admin_settings_trash_title')}
				</h2>
				<p class="settings__section-note mb-3.5 font-body font-normal text-[13px] leading-[1.5] text-text-faint">{$_('admin_settings_trash_note')}</p>
				<form class="settings__trash flex flex-col gap-3.5" onsubmit={handleTrashSubmit}>
					<div class="settings__field flex flex-col gap-1.5">
						<label class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text" for="admin-trash-days">{$_('admin_settings_trash_days')}</label>
						<input
							class="settings__number w-[120px] px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
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
						<p class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft" role="status">{$_('admin_settings_trash_saved')}</p>
					{/if}
					{#if trashErrorKey !== ''}
						<p class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text" role="alert">{errorText(trashErrorKey)}</p>
					{/if}
					<div class="settings__form-actions flex justify-end mt-1">
						<Button type="submit" disabled={savingTrash}><Check size={20} aria-hidden="true" />{$_('admin_settings_trash_save')}</Button>
					</div>
				</form>
				<p class="settings__warning mt-4 pt-4 border-t border-border flex items-start gap-2 font-body font-medium text-[12px] leading-[1.5] text-danger-text">
					<TriangleAlert size={14} class="settings__warning-icon shrink-0 mt-0.5" aria-hidden="true" />
					{$_('admin_settings_trash_warning')}
				</p>
			</Card>
		</section>

		<!-- The video transcoding section: the preset and concurrency
		     dropdowns, the two toggles, the save, and the
		     new-ingests-only scope note. -->
		<section class="settings__card" data-testid="admin-settings-transcoding">
			<Card>
				<h2 class="settings__section-title flex items-center gap-2 mb-1 font-display font-bold text-[17px] text-text">
					<MonitorPlay size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('admin_settings_transcode_title')}
				</h2>
				<p class="settings__section-note mb-3.5 font-body font-normal text-[13px] leading-[1.5] text-text-faint">{$_('admin_settings_transcode_note')}</p>
				<form class="settings__transcoding flex flex-col gap-3.5" onsubmit={handleTranscodingSubmit}>
					<div class="settings__field flex flex-col gap-1.5">
						<label class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text" for="admin-transcode-preset">{$_('admin_settings_transcode_preset')}</label>
						<select
							class="settings__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
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
						<label class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text" for="admin-transcode-concurrency">{$_('admin_settings_transcode_concurrency')}</label>
						<select
							class="settings__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
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
						<label class="settings__toggle flex items-center gap-2 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer">
							<input
								class="settings__toggle-input w-[18px] h-[18px] accent-accent"
								type="checkbox"
								data-testid="transcode-thumbnail-frame"
								bind:checked={formTranscoding.thumbnailFrame}
							/>
							{$_('admin_settings_transcode_thumbnail')}
						</label>
						<label class="settings__toggle flex items-center gap-2 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer">
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
						<p class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft" role="status">{$_('admin_settings_transcode_saved')}</p>
					{/if}
					{#if transcodingErrorKey !== ''}
						<p class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text" role="alert">{errorText(transcodingErrorKey)}</p>
					{/if}
					<div class="settings__form-actions flex justify-end mt-1">
						<Button type="submit" disabled={savingTranscoding}><Check size={20} aria-hidden="true" />{$_('admin_settings_transcode_save')}</Button>
					</div>
				</form>
				<p class="settings__contract mt-4 pt-4 border-t border-border font-body font-normal text-[13px] leading-[1.5] text-text-faint">{$_('admin_settings_transcode_contract')}</p>
			</Card>
		</section>
	{/if}
</div>
