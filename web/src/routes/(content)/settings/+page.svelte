<script lang="ts">
	// The user settings page (the settings feature): the five
	// sections of the owner-approved specimen -- the profile (name
	// and email), the password change (the revocation contract's
	// door), the appearance pair (theme and locale, applied
	// IMMEDIATELY: the store's locale and the html class react on
	// the pick, the users row persists for the next sign-in), the
	// download preferences (declarative account state; the download
	// flows do not exist yet, so the preferences persist for when
	// they do) and the signed-in devices (the isCurrent marker and
	// the revoke button; the current device's own exit is the
	// sign-out flow, so its row carries no revoke).
	//
	// THE ME-SCOPING: every call goes through /users/me -- the
	// session IS the identity, no userId ever appears.
	import { _, locale } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import Input from '$lib/components/Input.svelte';
	import {
		changeUserPassword,
		getUserSelf,
		listUserDevices,
		revokeUserDevice,
		updateUserSelf,
		type DeviceSession,
		type UserSelf
	} from '$lib/api/userSettings';
	import { setLocale, type AppLocale } from '$lib/i18n';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { setThemePreference, type ThemePreference } from '$lib/theme/theme.svelte';
	import User from '@lucide/svelte/icons/user';
	import Lock from '@lucide/svelte/icons/lock';
	import Sun from '@lucide/svelte/icons/sun';
	import Moon from '@lucide/svelte/icons/moon';
	import SunMoon from '@lucide/svelte/icons/sun-moon';
	import Download from '@lucide/svelte/icons/download';
	import Monitor from '@lucide/svelte/icons/monitor';
	import Smartphone from '@lucide/svelte/icons/smartphone';
	import Archive from '@lucide/svelte/icons/archive';
	import FileText from '@lucide/svelte/icons/file-text';
	import LogOut from '@lucide/svelte/icons/log-out';
	import Check from '@lucide/svelte/icons/check';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';

	type LoadStatus = 'idle' | 'loading' | 'ready' | 'error';

	// The account document and its load state: one GET /users/me
	// feeds the profile, appearance and download sections (the
	// devices section owns its own list read).
	let self = $state<UserSelf | null>(null);
	let status = $state<LoadStatus>('idle');
	let errorKey = $state('');
	/** Bumping re-runs the read (the retry button). */
	let retryTick = $state(0);

	// The profile form.
	let formName = $state('');
	let formEmail = $state('');
	let savingProfile = $state(false);
	let profileErrorKey = $state('');
	let profileSaved = $state(false);

	// The password form. The confirm field is the client's own
	// guard; the server owns the real bounds.
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let changingPassword = $state(false);
	let passwordErrorKey = $state('');
	let passwordMismatch = $state(false);
	let passwordChanged = $state(false);

	// The appearance and download sections' own error lines: the
	// immediate-apply patches are small and their failures stay
	// beside the control that fired them.
	let appearanceErrorKey = $state('');
	let downloadErrorKey = $state('');

	// The devices section.
	let devices = $state<DeviceSession[]>([]);
	let devicesStatus = $state<LoadStatus>('idle');
	let devicesErrorKey = $state('');
	let devicesRetryTick = $state(0);
	let revoking = $state<string | null>(null);
	let revokeErrorKey = $state('');

	// The account read, keyed on the retry tick. A superseded run --
	// the retry fired again, or the route left -- never applies.
	$effect(() => {
		void retryTick;
		status = 'loading';
		errorKey = '';
		let current = true;
		void (async () => {
			const outcome = await getUserSelf();
			if (!current) {
				return;
			}
			if (!outcome.ok) {
				status = 'error';
				errorKey = outcome.errorKey;
				return;
			}
			adoptSelf(outcome.self);
			status = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	// The device list read, keyed on its own retry tick.
	$effect(() => {
		void devicesRetryTick;
		devicesStatus = 'loading';
		devicesErrorKey = '';
		let current = true;
		void (async () => {
			const outcome = await listUserDevices();
			if (!current) {
				return;
			}
			if (!outcome.ok) {
				devicesStatus = 'error';
				devicesErrorKey = outcome.errorKey;
				return;
			}
			devices = outcome.devices;
			devicesStatus = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	/** Adopt one account document as the sections' fresh state. */
	function adoptSelf(next: UserSelf): void {
		self = next;
		formName = next.name;
		formEmail = next.email;
		profileErrorKey = '';
		profileSaved = false;
		passwordErrorKey = '';
		passwordMismatch = false;
		passwordChanged = false;
		appearanceErrorKey = '';
		downloadErrorKey = '';
	}

	function retry(): void {
		retryTick += 1;
	}

	function retryDevices(): void {
		devicesRetryTick += 1;
	}

	// The shared unknown-code fallback (see $lib/i18n/errors); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// THE APPEARANCE PAIR'S IMMEDIATE APPLY: the theme's pick maps
	// onto the shared store's preference (the server's `auto` is the
	// store's `system` -- follow the OS) and flips the html class
	// through it; the locale's pick reloads the i18n store through
	// setLocale. The users row persists the same value for the next
	// sign-in; a failed patch keeps the local apply (the preference
	// is the user's stated intent) and shows the error beside the
	// control.
	const themeChoices: ReadonlyArray<{
		value: UserSelf['theme'];
		labelKey: string;
		apply: ThemePreference;
		icon: typeof Sun;
	}> = [
		{ value: 'light', labelKey: 'theme_light', apply: 'light', icon: Sun },
		{ value: 'dark', labelKey: 'theme_dark', apply: 'dark', icon: Moon },
		{ value: 'auto', labelKey: 'theme_system', apply: 'system', icon: SunMoon }
	];

	/** One theme pick: apply locally, persist on the row. */
	async function pickTheme(theme: UserSelf['theme']): Promise<void> {
		const choice = themeChoices.find((candidate) => candidate.value === theme);
		if (self === null || choice === undefined || self.theme === theme) {
			return;
		}
		self = { ...self, theme };
		setThemePreference(choice.apply);
		appearanceErrorKey = '';
		const outcome = await updateUserSelf({ theme });
		if (!outcome.ok) {
			appearanceErrorKey = outcome.errorKey;
			return;
		}
		adoptSelf(outcome.self);
	}

	/** One locale pick: apply locally, persist on the row. */
	async function pickLocale(next: AppLocale): Promise<void> {
		if (self === null || self.locale === next) {
			return;
		}
		self = { ...self, locale: next };
		setLocale(next);
		appearanceErrorKey = '';
		const outcome = await updateUserSelf({ locale: next });
		if (!outcome.ok) {
			appearanceErrorKey = outcome.errorKey;
			return;
		}
		adoptSelf(outcome.self);
	}

	// The download preferences: applied on every change like the
	// appearance pair (the specimen carries no save button for the
	// section; the preferences ARE the state).
	const formatChoices: ReadonlyArray<{
		value: UserSelf['downloadFormat'];
		labelKey: string;
		icon: typeof Archive;
	}> = [
		{ value: 'zip', labelKey: 'settings_download_format_zip', icon: Archive },
		{ value: 'tar', labelKey: 'settings_download_format_tar', icon: FileText }
	];

	async function pickDownloadFormat(format: UserSelf['downloadFormat']): Promise<void> {
		if (self === null || self.downloadFormat === format) {
			return;
		}
		self = { ...self, downloadFormat: format };
		downloadErrorKey = '';
		const outcome = await updateUserSelf({ downloadFormat: format });
		if (!outcome.ok) {
			downloadErrorKey = outcome.errorKey;
			return;
		}
		adoptSelf(outcome.self);
	}

	async function persistDownloadFlag(flag: 'downloadIncludeVideos' | 'downloadIncludeSidecars'): Promise<void> {
		if (self === null) {
			return;
		}
		downloadErrorKey = '';
		const outcome = await updateUserSelf({ [flag]: self[flag] });
		if (!outcome.ok) {
			downloadErrorKey = outcome.errorKey;
			return;
		}
		adoptSelf(outcome.self);
	}

	async function handleProfileSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (self === null || savingProfile) {
			return;
		}
		savingProfile = true;
		profileErrorKey = '';
		profileSaved = false;
		const outcome = await updateUserSelf({
			name: formName.trim(),
			email: formEmail.trim()
		});
		savingProfile = false;
		if (!outcome.ok) {
			profileErrorKey = outcome.errorKey;
			return;
		}
		adoptSelf(outcome.self);
		profileSaved = true;
	}

	async function handlePasswordSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (changingPassword) {
			return;
		}
		// The client's own guard first: the confirm field must agree
		// with the new one before anything leaves the browser.
		passwordMismatch = newPassword !== confirmPassword;
		if (passwordMismatch) {
			return;
		}
		changingPassword = true;
		passwordErrorKey = '';
		passwordChanged = false;
		const outcome = await changeUserPassword(currentPassword, newPassword);
		changingPassword = false;
		if (!outcome.ok) {
			passwordErrorKey = outcome.errorKey;
			return;
		}
		// The contract's revocation note: the other devices were
		// signed out; this one stays. Refresh the list so the rows
		// match reality, and clear the form.
		passwordChanged = true;
		currentPassword = '';
		newPassword = '';
		confirmPassword = '';
		devicesRetryTick += 1;
	}

	/** One device's revoke; the list refetch adopts the server's answer. */
	async function revoke(device: DeviceSession): Promise<void> {
		if (revoking !== null) {
			return;
		}
		revoking = device.id;
		revokeErrorKey = '';
		const outcome = await revokeUserDevice(device.id);
		revoking = null;
		if (!outcome.ok) {
			revokeErrorKey = outcome.errorKey;
			return;
		}
		const list = await listUserDevices();
		if (list.ok) {
			devices = list.devices;
			devicesStatus = 'ready';
		}
	}

	// The device label: a web session's stored label is the login's
	// raw user agent, condensed to the browser · platform pair the
	// specimen renders; a paired device's label is the name the
	// pairing supplied. An unrecognized agent falls back to the raw
	// string (honest over clever).
	function deviceLabel(device: DeviceSession): string {
		if (device.kind === 'web' && device.deviceName !== undefined && device.deviceName !== '') {
			return condenseUserAgent(device.deviceName);
		}
		// The fallback keys by kind: a web session with no stored
		// label is a browser, not a paired device.
		return (
			device.deviceName ??
			$_(device.kind === 'web' ? 'settings_devices_kind_web' : 'settings_devices_kind_mobile')
		);
	}

	function condenseUserAgent(agent: string): string {
		let browser = '';
		if (agent.includes('Firefox/')) {
			browser = 'Firefox';
		} else if (agent.includes('Edg/')) {
			browser = 'Edge';
		} else if (agent.includes('Chrome/')) {
			browser = 'Chrome';
		} else if (agent.includes('Safari/')) {
			browser = 'Safari';
		}
		let platform = '';
		if (agent.includes('iPhone')) {
			platform = 'iPhone';
		} else if (agent.includes('iPad')) {
			platform = 'iPad';
		} else if (agent.includes('Android')) {
			platform = 'Android';
		} else if (agent.includes('Macintosh')) {
			platform = 'macOS';
		} else if (agent.includes('Windows')) {
			platform = 'Windows';
		} else if (agent.includes('Linux')) {
			platform = 'Linux';
		}
		if (browser === '' && platform === '') {
			return agent;
		}
		return platform === '' ? browser : `${browser} · ${platform}`;
	}

	// The device row's meta line: the last use through the active
	// locale's medium date, and the creation-time address when the
	// row carries one.
	function deviceMeta(device: DeviceSession): string {
		const activeLocale = $locale ?? 'en';
		const seen = new Intl.DateTimeFormat(activeLocale, {
			dateStyle: 'medium',
			timeStyle: 'short'
		}).format(new Date(device.lastSeenAt));
		return device.ipAddress === undefined ? seen : `${seen} · ${device.ipAddress}`;
	}

	// The locale dropdown renders the installed locales in their OWN
	// language (the shared table's per-locale names are already the
	// native ones); the current locale's own entry leads.
	const localeChoices: ReadonlyArray<{ value: AppLocale; labelKey: string }> = [
		{ value: 'en', labelKey: 'language_english' },
		{ value: 'zh-Hans', labelKey: 'language_chinese' }
	];
</script>

<svelte:head>
	<title>Settings · lumemor</title>
</svelte:head>

<div class="settings max-w-[640px] mx-auto flex flex-col gap-4 p-6" aria-busy={status === 'loading'}>
	{#if status === 'loading'}
		<p class="settings__status my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('settings_loading')}
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
	{:else if status === 'ready' && self !== null}
		<header class="settings__header">
			<h1 class="settings__title mb-1 font-display font-bold text-[26px] text-text">{$_('settings_title')}</h1>
		</header>

		<!-- The profile section: the identity fields and their save. -->
		<section class="settings__section" data-testid="settings-profile">
			<Card>
				<h2
					class="settings__section-title mb-3.5 flex items-center gap-2 font-display font-bold text-[17px] text-text"
				>
					<User size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('settings_profile_title')}
				</h2>
				<form class="settings__profile flex flex-col gap-3.5" onsubmit={handleProfileSubmit}>
					<Input
						id="settings-name"
						label={$_('settings_profile_name')}
						type="text"
						autocomplete="name"
						bind:value={formName}
					/>
					<Input
						id="settings-email"
						label={$_('settings_profile_email')}
						type="email"
						autocomplete="email"
						bind:value={formEmail}
					/>
					{#if profileSaved}
						<p
							class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft"
							role="status"
						>
							{$_('settings_profile_saved')}
						</p>
					{/if}
					{#if profileErrorKey !== ''}
						<p
							class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(profileErrorKey)}
						</p>
					{/if}
					<div class="settings__actions flex justify-end">
						<Button type="submit" disabled={savingProfile}
							><Check size={20} aria-hidden="true" />{$_('settings_profile_save')}</Button
						>
					</div>
				</form>
			</Card>
		</section>

		<!-- The password section: the revocation contract's door. The
		     current session stays; the note says what happened to the
		     others. -->
		<section class="settings__section" data-testid="settings-password">
			<Card>
				<h2
					class="settings__section-title mb-3.5 flex items-center gap-2 font-display font-bold text-[17px] text-text"
				>
					<Lock size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('settings_password_title')}
				</h2>
				<form class="settings__password flex flex-col gap-3.5" onsubmit={handlePasswordSubmit}>
					<Input
						id="settings-password-current"
						label={$_('settings_password_current')}
						type="password"
						autocomplete="current-password"
						bind:value={currentPassword}
					/>
					<Input
						id="settings-password-new"
						label={$_('settings_password_new')}
						type="password"
						autocomplete="new-password"
						placeholder={$_('settings_password_new_hint')}
						bind:value={newPassword}
					/>
					<Input
						id="settings-password-confirm"
						label={$_('settings_password_confirm')}
						type="password"
						autocomplete="new-password"
						bind:value={confirmPassword}
						error={passwordMismatch ? $_('settings_password_mismatch') : ''}
					/>
					{#if passwordChanged}
						<p
							class="settings__saved font-body font-normal text-[13px] leading-[1.5] text-text-soft"
							role="status"
						>
							{$_('settings_password_changed')}
						</p>
					{/if}
					{#if passwordErrorKey !== ''}
						<p
							class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(passwordErrorKey)}
						</p>
					{/if}
					<div class="settings__actions flex justify-end">
						<Button type="submit" disabled={changingPassword}
							><Lock size={20} aria-hidden="true" />{$_('settings_password_submit')}</Button
						>
					</div>
				</form>
			</Card>
		</section>

		<!-- The appearance section: the immediate-apply pair. Each
		     control applies on the pick -- the html class flips, the
		     i18n store reloads -- and the row persists the choice. -->
		<section class="settings__section" data-testid="settings-appearance">
			<Card>
				<h2
					class="settings__section-title mb-3.5 flex items-center gap-2 font-display font-bold text-[17px] text-text"
				>
					<Sun size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('settings_appearance_title')}
				</h2>
				<div class="settings__appearance flex flex-col gap-4">
					<div class="settings__field flex flex-col gap-1.5">
						<p class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text">
							{$_('settings_appearance_theme')}
						</p>
						<div
							class="settings__theme-chips flex gap-2"
							role="radiogroup"
							aria-label={$_('settings_appearance_theme')}
							data-testid="settings-theme"
						>
							{#each themeChoices as choice (choice.value)}
								<button
									class="settings__chip flex-1 flex items-center justify-center gap-2 px-3 py-2.5 font-body font-semibold text-[14px] leading-[1.4] rounded-control border cursor-pointer transition-[background-color,border-color,color] duration-(--motion-fast) ease-standard {self.theme ===
									choice.value
										? 'bg-accent-soft border-accent text-accent-soft-text'
										: 'bg-transparent border-border-strong text-text-soft hover:not-disabled:bg-accent-soft hover:not-disabled:text-accent-soft-text'}"
									type="button"
									role="radio"
									aria-checked={self.theme === choice.value}
									data-testid="settings-theme-{choice.value}"
									onclick={() => void pickTheme(choice.value)}
								>
									<choice.icon size={16} aria-hidden="true" />
									{$_(choice.labelKey)}
								</button>
							{/each}
						</div>
					</div>
					<div class="settings__field flex flex-col gap-1.5">
						<label
							class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text"
							for="settings-locale">{$_('settings_appearance_locale')}</label
						>
						<select
							class="settings__select w-full px-3 py-[9px] pe-9 font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control cursor-pointer transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
							id="settings-locale"
							data-testid="settings-locale"
							value={self.locale}
							onchange={(event) =>
								void pickLocale((event.currentTarget as HTMLSelectElement).value as AppLocale)}
						>
							{#each localeChoices as choice (choice.value)}
								<option value={choice.value}>{$_(choice.labelKey)}</option>
							{/each}
						</select>
						<p class="settings__note font-body font-normal text-[12px] leading-[1.5] text-text-faint">
							{$_('settings_appearance_note')}
						</p>
					</div>
					{#if appearanceErrorKey !== ''}
						<p
							class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(appearanceErrorKey)}
						</p>
					{/if}
				</div>
			</Card>
		</section>

		<!-- The download preferences section: declarative account
		     state, applied on every change like the appearance pair.
		     The note states the honest consumption state. -->
		<section class="settings__section" data-testid="settings-download">
			<Card>
				<h2
					class="settings__section-title mb-3.5 flex items-center gap-2 font-display font-bold text-[17px] text-text"
				>
					<Download size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('settings_download_title')}
				</h2>
				<div class="settings__download flex flex-col gap-4">
					<div class="settings__field flex flex-col gap-1.5">
						<p class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text">
							{$_('settings_download_format')}
						</p>
						<div
							class="settings__format-chips flex gap-2"
							role="radiogroup"
							aria-label={$_('settings_download_format')}
							data-testid="settings-format"
						>
							{#each formatChoices as choice (choice.value)}
								<button
									class="settings__chip flex-1 flex items-center justify-center gap-2 px-3 py-2.5 font-body font-semibold text-[14px] leading-[1.4] rounded-control border cursor-pointer transition-[background-color,border-color,color] duration-(--motion-fast) ease-standard {self.downloadFormat ===
									choice.value
										? 'bg-accent-soft border-accent text-accent-soft-text'
										: 'bg-transparent border-border-strong text-text-soft hover:not-disabled:bg-accent-soft hover:not-disabled:text-accent-soft-text'}"
									type="button"
									role="radio"
									aria-checked={self.downloadFormat === choice.value}
									data-testid="settings-format-{choice.value}"
									onclick={() => void pickDownloadFormat(choice.value)}
								>
									<choice.icon size={16} aria-hidden="true" />
									{$_(choice.labelKey)}
								</button>
							{/each}
						</div>
					</div>
					<div class="settings__field flex flex-col gap-1.5">
						<p class="settings__label font-body font-semibold text-[14px] leading-[1.4] text-text">
							{$_('settings_download_contents')}
						</p>
						<div class="settings__toggles flex flex-col gap-2">
							<label
								class="settings__toggle flex items-center gap-2 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
							>
								<input
									class="settings__toggle-input w-[18px] h-[18px] accent-accent"
									type="checkbox"
									data-testid="settings-download-videos"
									bind:checked={self.downloadIncludeVideos}
									onchange={() => void persistDownloadFlag('downloadIncludeVideos')}
								/>
								{$_('settings_download_videos')}
							</label>
							<label
								class="settings__toggle flex items-center gap-2 font-body font-semibold text-[14px] leading-[1.4] text-text-soft cursor-pointer"
							>
								<input
									class="settings__toggle-input w-[18px] h-[18px] accent-accent"
									type="checkbox"
									data-testid="settings-download-sidecars"
									bind:checked={self.downloadIncludeSidecars}
									onchange={() => void persistDownloadFlag('downloadIncludeSidecars')}
								/>
								{$_('settings_download_sidecars')}
							</label>
						</div>
						<p class="settings__note font-body font-normal text-[12px] leading-[1.5] text-text-faint">
							{$_('settings_download_note')}
						</p>
					</div>
					{#if downloadErrorKey !== ''}
						<p
							class="settings__error font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(downloadErrorKey)}
						</p>
					{/if}
				</div>
			</Card>
		</section>

		<!-- The devices section: the list with the current badge and
		     the revoke buttons; the current row carries none. -->
		<section class="settings__section" data-testid="settings-devices">
			<Card>
				<h2
					class="settings__section-title mb-3.5 flex items-center gap-2 font-display font-bold text-[17px] text-text"
				>
					<Monitor size={18} class="settings__section-icon text-accent" aria-hidden="true" />
					{$_('settings_devices_title')}
				</h2>
				{#if devicesStatus === 'loading' || devicesStatus === 'idle'}
					<p
						class="settings__devices-status py-4 text-center text-text-soft animate-loading-in"
						role="status"
					>
						{$_('settings_loading')}
					</p>
				{:else if devicesStatus === 'error'}
					<div
						class="settings__devices-failure flex flex-col items-center gap-3 py-4 text-center"
						role="alert"
					>
						<p class="text-danger-text">{errorText(devicesErrorKey)}</p>
						<Button variant="ghost" onclick={retryDevices}>{$_('action_retry')}</Button>
					</div>
				{:else}
					<ul class="settings__devices flex flex-col list-none p-0 m-0" data-testid="settings-device-rows">
						{#each devices as device (device.id)}
							<li
								class="settings__device flex items-center gap-3 py-3 border-b border-border last:border-b-0"
								data-testid="settings-device-row"
								data-device-id={device.id}
							>
								<span
									class="settings__device-icon inline-flex items-center justify-center w-9 h-9 rounded-control bg-accent-soft text-accent-soft-text"
									aria-hidden="true"
								>
									{#if device.kind === 'mobile'}
										<Smartphone size={18} />
									{:else}
										<Monitor size={18} />
									{/if}
								</span>
								<span class="settings__device-info flex-1 min-w-0">
									<span
										class="settings__device-name flex items-center gap-2 font-body font-semibold text-[14px] leading-[1.5] text-text"
									>
										{deviceLabel(device)}
										{#if device.isCurrent}
											<span
												class="settings__device-current py-0.5 px-1.5 font-body font-semibold text-[11px] leading-[1.4] text-success bg-[color-mix(in_srgb,var(--color-success)_14%,transparent)] rounded-control"
												data-testid="settings-device-current"
												>{$_('settings_devices_current')}</span
											>
										{/if}
									</span>
									<span
										class="settings__device-meta block font-body font-normal text-[12px] leading-[1.5] text-text-faint"
										>{deviceMeta(device)}</span
									>
								</span>
								{#if !device.isCurrent}
									<Button
										variant="danger"
										disabled={revoking !== null}
										data-testid="settings-device-revoke"
										onclick={() => void revoke(device)}
									>
										<LogOut size={16} aria-hidden="true" />
										{$_('settings_devices_revoke')}
									</Button>
								{/if}
							</li>
						{/each}
					</ul>
					{#if revokeErrorKey !== ''}
						<p
							class="settings__error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
							role="alert"
						>
							{errorText(revokeErrorKey)}
						</p>
					{/if}
					<p class="settings__note mt-3 font-body font-normal text-[12px] leading-[1.5] text-text-faint">
						{$_('settings_devices_note')}
					</p>
				{/if}
			</Card>
		</section>
	{/if}
</div>
