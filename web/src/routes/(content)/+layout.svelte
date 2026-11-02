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
	import { logout } from '$lib/auth/session.svelte';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	let drawerOpen = $state(false);
	let accountMenuOpen = $state(false);
	let signingOut = $state(false);
	let accountWrap: HTMLElement | undefined = $state();

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

<div class="shell">
	<header class="shell-topbar">
		<button
			class="topbar-icon"
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
		<span class="topbar-brand">{$_('app_name')}</span>
		<input
			class="topbar-search"
			type="search"
			placeholder={$_('nav_search')}
			aria-label={$_('nav_search')}
		/>
		<Button class="topbar-upload">{$_('action_upload')}</Button>
		<div class="topbar-account" bind:this={accountWrap}>
			<button
				class="topbar-avatar"
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
				<div class="account-menu">
					<p class="account-menu__heading">{$_('theme_label')}</p>
					<div class="account-menu__options" role="group" aria-label={$_('theme_label')}>
						{#each themeOptions as option (option.value)}
							<button
								class="account-menu__option"
								type="button"
								aria-pressed={themePreference() === option.value}
								onclick={() => setThemePreference(option.value)}
							>
								{$_(option.labelKey)}
							</button>
						{/each}
					</div>
					<p class="account-menu__heading">{$_('language_label')}</p>
					<div class="account-menu__options" role="group" aria-label={$_('language_label')}>
						{#each languageOptions as option (option.value)}
							<button
								class="account-menu__option"
								type="button"
								aria-pressed={($locale ?? 'en') === option.value}
								onclick={() => setLocale(option.value)}
							>
								{$_(option.labelKey)}
							</button>
						{/each}
					</div>
					<div class="account-menu__divider"></div>
					<button
						class="account-menu__option account-menu__signout"
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
	<main class="shell-main">
		{@render children()}
	</main>
</div>

<style>
	.shell {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}

	.shell-topbar {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 16px;
		border-bottom: 1px solid var(--color-border);
		background-color: var(--color-card);
	}

	.topbar-icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border: none;
		border-radius: var(--radius-control);
		background: none;
		color: var(--color-text);
		cursor: pointer;
		transition: background-color var(--motion-fast) var(--ease-standard);
	}

	.topbar-icon:hover {
		background-color: var(--color-accent-soft);
	}

	.topbar-brand {
		/* Wordmark: Nunito 700 in the text color (Navy 900 light,
		 * Cream dark), per the brand spec. */
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 20px;
		letter-spacing: 0.01em;
		color: var(--color-text);
	}

	.topbar-search {
		/* Push the search box away from the brand; the upload and
		 * account marks stay pinned to the right end. */
		flex: 1;
		max-width: 420px;
		margin-left: auto;
		padding: 8px 12px;
		font: 400 15px/1.4 var(--font-body);
		color: var(--color-text);
		background-color: var(--color-page);
		border: 1px solid var(--color-border-strong);
		border-radius: var(--radius-control);
		transition: border-color var(--motion-fast) var(--ease-standard);
	}

	.topbar-search:focus {
		border-color: var(--color-accent);
	}

	.topbar-account {
		position: relative;
	}

	.topbar-avatar {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border: 1px solid var(--color-border-strong);
		border-radius: 50%;
		background: none;
		color: var(--color-text-soft);
		cursor: pointer;
		transition:
			background-color var(--motion-fast) var(--ease-standard),
			color var(--motion-fast) var(--ease-standard);
	}

	.topbar-avatar:hover {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}

	.account-menu {
		position: absolute;
		top: calc(100% + 8px);
		right: 0;
		z-index: 50;
		min-width: 180px;
		padding: 8px;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-overlay);
		box-shadow: var(--shadow-overlay);
		animation: menu-in var(--motion-fast) var(--ease-standard);
	}

	.account-menu__heading {
		margin: 4px 8px 2px;
		font: 600 12px/1.4 var(--font-body);
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--color-text-faint);
	}

	.account-menu__options {
		display: flex;
		flex-direction: column;
	}

	.account-menu__option {
		padding: 7px 8px;
		border: none;
		border-radius: var(--radius-control);
		background: none;
		font: 400 15px/1.4 var(--font-body);
		color: var(--color-text);
		text-align: left;
		cursor: pointer;
		transition: background-color var(--motion-fast) var(--ease-standard);
	}

	.account-menu__option:hover {
		background-color: var(--color-accent-soft);
	}

	.account-menu__option[aria-pressed='true'] {
		font-weight: 700;
		color: var(--color-accent-soft-text);
		background-color: var(--color-accent-soft);
	}

	.account-menu__divider {
		height: 1px;
		margin: 8px 4px;
		background-color: var(--color-border);
	}

	/* The sign-out action closes the account section: the divider
	 * separates it from the appearance toggles, and the label keeps
	 * the regular text color for AA contrast. */
	.account-menu__signout:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}

	.shell-main {
		flex: 1;
		padding: 24px;
	}

	@keyframes menu-in {
		from {
			opacity: 0;
			transform: translateY(-4px);
		}
	}
</style>
