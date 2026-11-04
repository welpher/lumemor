<script lang="ts">
	// The content shell from the web IA design: a top bar with the
	// drawer toggle (hamburger), the wordmark, the search entry, the
	// upload action and the account avatar. The avatar opens the
	// dropdown that carries the two appearance toggles (theme and
	// language, from the shared wiring task) and the logout action.
	// The drawer carries the IA's seven content sections; search and
	// upload stay placeholders wired by later tasks.
	import { locale, _ } from 'svelte-i18n';
	import { goto } from '$app/navigation';
	import { setLocale, type AppLocale } from '$lib/i18n';
	import {
		setThemePreference,
		themePreference,
		type ThemePreference
	} from '$lib/theme/theme.svelte';
	import Button from '$lib/components/Button.svelte';
	import Drawer from '$lib/components/Drawer.svelte';
	import UploadOverlay from '$lib/upload/UploadOverlay.svelte';
	import { enqueueUploadFiles } from '$lib/upload/queue.svelte';
	import { logout } from '$lib/auth/session.svelte';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	let drawerOpen = $state(false);
	let accountMenuOpen = $state(false);
	let signingOut = $state(false);
	let accountWrap: HTMLElement | undefined = $state();
	// The top bar's own hidden picker (the overlay component carries
	// only the queue panel and the drop zone): the button trigger and
	// its input live side by side, and picked files join the same
	// singleton queue as the drop zone's.
	let picker = $state<HTMLInputElement>();

	// The picker's accept mirrors the hint-level upload whitelist
	// (format set v1; the server stays authoritative).
	const PICKER_ACCEPT = [
		'.jpg',
		'.jpeg',
		'.png',
		'.gif',
		'.webp',
		'.tif',
		'.tiff',
		'.heic',
		'.avif',
		'.mp4',
		'.mov',
		'.webm',
		'.mkv'
	].join(',');

	function pickFiles(): void {
		picker?.click();
	}

	function handlePicked(): void {
		const files = picker?.files;
		if (files !== null && files !== undefined && files.length > 0) {
			enqueueUploadFiles(files);
		}
		if (picker !== undefined) {
			picker.value = '';
		}
	}

	const themeOptions: ReadonlyArray<{ value: ThemePreference; labelKey: string }> = [
		{ value: 'light', labelKey: 'theme_light' },
		{ value: 'dark', labelKey: 'theme_dark' },
		{ value: 'system', labelKey: 'theme_system' }
	];
	const languageOptions: ReadonlyArray<{ value: AppLocale; labelKey: string }> = [
		{ value: 'en', labelKey: 'language_english' },
		{ value: 'zh-Hans', labelKey: 'language_chinese' }
	];

	function closeAccountMenu(): void {
		accountMenuOpen = false;
	}

	function handleWindowPointerdown(event: PointerEvent): void {
		if (accountMenuOpen && accountWrap && !accountWrap.contains(event.target as Node)) {
			closeAccountMenu();
		}
	}

	function handleWindowKeydown(event: KeyboardEvent): void {
		if (accountMenuOpen && event.key === 'Escape') {
			closeAccountMenu();
		}
	}

	// Sign out: revoke the session, clear the store, land on the
	// login page. The SessionGuard watcher reacts to the same store
	// flip and may race this goto; both navigations target /login
	// (the watcher's carries the redirectTo round-trip), so either
	// winner leaves a consistent page.
	async function handleSignOut(): Promise<void> {
		if (signingOut) {
			return;
		}
		signingOut = true;
		closeAccountMenu();
		await logout();
		goto('/login');
		signingOut = false;
	}
</script>

<svelte:window onpointerdown={handleWindowPointerdown} onkeydown={handleWindowKeydown} />

<div class="shell min-h-screen flex flex-col">
	<header class="shell-topbar flex items-center gap-3 px-4 py-2 border-b border-border bg-card">
		<button
			class="topbar-icon inline-flex items-center justify-center w-9 h-9 rounded-control bg-transparent text-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft"
			type="button"
			aria-label={$_('nav_open_menu')}
			aria-expanded={drawerOpen}
			onclick={() => (drawerOpen = !drawerOpen)}
		>
			<svg viewBox="0 0 24 24" width="22" height="22" fill="none" aria-hidden="true">
				<path
					d="M4 7h16M4 12h16M4 17h16"
					stroke="currentColor"
					stroke-width="1.75"
					stroke-linecap="round"
				/>
			</svg>
		</button>
		<!-- Wordmark: Nunito 700 in the text color (Navy 900 light,
		     Cream dark), per the brand spec. -->
		<span class="topbar-brand font-display font-bold text-[20px] tracking-[0.01em] text-text">{$_('app_name')}</span>
		<!-- Push the search box away from the brand; the upload and
		     account marks stay pinned to the right end. -->
		<input
			class="topbar-search flex-1 max-w-[420px] box-content ml-auto px-3 py-2 font-body font-normal text-[15px] leading-[1.4] text-text bg-page border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
			type="search"
			placeholder={$_('nav_search')}
			aria-label={$_('nav_search')}
		/>
		<span class="topbar-upload-wrap">
			<!-- The hidden input of the top bar's Upload button; picked
			     files join the upload queue through enqueueUploadFiles. -->
			<input
				bind:this={picker}
				class="topbar-picker hidden"
				type="file"
				multiple
				accept={PICKER_ACCEPT}
				onchange={handlePicked}
			/>
			<Button class="topbar-upload" onclick={pickFiles}>{$_('action_upload')}</Button>
		</span>
		<div class="topbar-account relative" bind:this={accountWrap}>
			<button
				class="topbar-avatar inline-flex items-center justify-center w-9 h-9 border border-border-strong rounded-full bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
				type="button"
				aria-label={$_('account_menu')}
				aria-expanded={accountMenuOpen}
				onclick={() => (accountMenuOpen = !accountMenuOpen)}
			>
				<svg viewBox="0 0 24 24" width="20" height="20" fill="none" aria-hidden="true">
					<circle cx="12" cy="8.5" r="3.75" stroke="currentColor" stroke-width="1.75" />
					<path
						d="M4.75 19.25c1.1-3.4 4-5 7.25-5s6.15 1.6 7.25 5"
						stroke="currentColor"
						stroke-width="1.75"
						stroke-linecap="round"
					/>
				</svg>
			</button>
			{#if accountMenuOpen}
				<div class="account-menu absolute top-[calc(100%_+_8px)] right-0 z-[50] min-w-[180px] box-content p-2 bg-card border border-border rounded-overlay shadow-overlay">
					<p class="account-menu__heading mt-1 mx-2 mb-0.5 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint">{$_('theme_label')}</p>
					<div class="account-menu__options flex flex-col" role="group" aria-label={$_('theme_label')}>
						{#each themeOptions as option (option.value)}
							<button
								class="account-menu__option px-2 py-[7px] rounded-control bg-transparent font-body font-normal text-[15px] leading-[1.4] text-text text-left cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft aria-pressed:font-bold aria-pressed:text-accent-soft-text aria-pressed:bg-accent-soft"
								type="button"
								aria-pressed={themePreference() === option.value}
								onclick={() => setThemePreference(option.value)}
							>
								{$_(option.labelKey)}
							</button>
						{/each}
					</div>
					<p class="account-menu__heading mt-1 mx-2 mb-0.5 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-text-faint">{$_('language_label')}</p>
					<div class="account-menu__options flex flex-col" role="group" aria-label={$_('language_label')}>
						{#each languageOptions as option (option.value)}
							<button
								class="account-menu__option px-2 py-[7px] rounded-control bg-transparent font-body font-normal text-[15px] leading-[1.4] text-text text-left cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft aria-pressed:font-bold aria-pressed:text-accent-soft-text aria-pressed:bg-accent-soft"
								type="button"
								aria-pressed={($locale ?? 'en') === option.value}
								onclick={() => setLocale(option.value)}
							>
								{$_(option.labelKey)}
							</button>
						{/each}
					</div>
					<div class="account-menu__divider h-px my-2 mx-1 bg-border"></div>
					<!-- The sign-out action closes the account section: the divider
					     separates it from the appearance toggles, and the label keeps
					     the regular text color for AA contrast. -->
					<button
						class="account-menu__option account-menu__signout px-2 py-[7px] rounded-control bg-transparent font-body font-normal text-[15px] leading-[1.4] text-text text-left cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft disabled:opacity-55 disabled:cursor-not-allowed"
						type="button"
						disabled={signingOut}
						onclick={handleSignOut}
					>
						{$_('action_sign_out')}
					</button>
				</div>
			{/if}
		</div>
	</header>
	<Drawer bind:open={drawerOpen} />
	<main class="shell-main flex-1 p-6">
		{@render children()}
	</main>
</div>
<UploadOverlay />

<style>
	/* Residual styles (verbatim): the account menu's pop-in keyframes
	 * and its animation declaration. */
	.account-menu {
		animation: menu-in var(--motion-fast) var(--ease-standard);
	}

	@keyframes menu-in {
		from {
			opacity: 0;
			transform: translateY(-4px);
		}
	}
</style>
