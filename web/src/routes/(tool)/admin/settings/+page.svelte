<script lang="ts">
	// The storage settings section of the admin surface (the
	// storage-template feature): the instance's upload path template
	// with the sixteen built-in presets filling the input, a live
	// sample preview of the SAVED template, and the save's validation
	// errors -- the shared i18n text plus the server's specific reason
	// line (the reason names the admin's own template text; no
	// translation table can pre-render it).
	//
	// THE 403 PAGE: a signed-in non-admin landing on this route sees
	// the mapped auth_forbidden message -- the session contract
	// carries no role, so this page's own read is the probe.
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import Input from '$lib/components/Input.svelte';
	import {
		getStorageSettings,
		saveStorageSettings,
		type StorageSettings
	} from '$lib/api/adminSettings';
	import { resolveErrorText } from '$lib/i18n/errors';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	let settings = $state<StorageSettings | null>(null);
	let status = $state<ListStatus>('idle');
	let errorKey = $state('');
	/** Bumping re-runs the read (the retry button). */
	let retryTick = $state(0);

	// The form: the template input and the preset that fills it. The
	// preset select follows the input (a hand-edited template reads
	// back as "Custom"); choosing a preset writes it into the input,
	// where it stays editable.
	let formTemplate = $state('');
	let formPreset = $state('custom');
	let saving = $state(false);
	let saveErrorKey = $state('');
	/** The server's specific validation reason of the last failed save. */
	let saveReason = $state('');
	let savedNote = $state(false);

	// The read, keyed on the retry tick. A superseded run -- the
	// retry fired again, or the route left -- never applies.
	$effect(() => {
		void retryTick;
		status = 'loading';
		errorKey = '';
		let current = true;
		void (async () => {
			const outcome = await getStorageSettings();
			if (!current) {
				return;
			}
			if (!outcome.ok) {
				status = 'error';
				errorKey = outcome.errorKey;
				return;
			}
			adoptSettings(outcome.settings);
			status = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	function adoptSettings(next: StorageSettings): void {
		settings = next;
		formTemplate = next.template;
		formPreset = presetOf(next.template);
		saveErrorKey = '';
		saveReason = '';
		savedNote = false;
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
			adoptSettings(outcome.settings);
			savedNote = true;
			return;
		}
		saveErrorKey = outcome.errorKey;
		saveReason = outcome.reason ?? '';
	}
</script>

<svelte:head>
	<title>Settings · lumemor</title>
</svelte:head>

<div class="settings max-w-[680px] mx-auto flex flex-col gap-5">
	{#if status === 'loading'}
		<p class="settings__status my-12 text-center text-text-soft animate-loading-in" role="status">{$_('admin_settings_loading')}</p>
	{:else if status === 'error'}
		<div class="settings__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}>{$_('action_retry')}</Button>
		</div>
	{:else if status === 'ready' && settings !== null}
		<header class="settings__header">
			<h1 class="settings__title mb-1 font-display font-bold text-[26px] text-text">{$_('admin_settings_storage_title')}</h1>
			<p class="settings__note font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('admin_settings_storage_note')}</p>
		</header>

		<section class="settings__card w-[min(560px,100%)]">
			<Card>
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
						<Button type="submit" disabled={saving}>{$_('admin_settings_submit')}</Button>
					</div>
				</form>
				<p class="settings__contract mt-4 pt-4 border-t border-border font-body font-normal text-[13px] leading-[1.5] text-text-faint">{$_('admin_settings_contract_note')}</p>
			</Card>
		</section>
	{/if}
</div>
