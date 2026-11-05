<script lang="ts">
	// The content shell from the web IA design: a top bar with the
	// drawer toggle (hamburger), the wordmark, the search entry, the
	// upload action and the account avatar. The avatar opens the
	// dropdown that carries the two appearance toggles (theme and
	// language, from the shared wiring task) and the logout action.
	// The drawer carries the IA's seven content sections; search and
	// upload stay placeholders wired by later tasks.
	//
	// THE APP FRAME (R2-1): the shell is an h-dvh flex column with
	// overflow hidden -- the top bar is pinned by the frame itself
	// (no sticky hack, the document never scrolls) and <main> is the
	// one scroll container. Two frame duties follow from that:
	// - Keyboard scrolling follows FOCUS, and a body-focused page
	//   stops scrolling once the document does, so the frame keeps
	//   the focus on its scroll root (tabindex -1: a scroll vehicle
	//   that never joins the tab order; a click on any non-focusable
	//   content keeps the focus there too).
	// - SvelteKit's navigation scroll reset acts on the document, so
	//   the frame resets <main> itself: push navigations land at the
	//   top, popstate keeps whatever offset the wall left (the wall
	//   remounts and refetches on the back-from-viewer return either
	//   way -- nothing restores a stale offset).
	import { locale, _ } from 'svelte-i18n';
	import { afterNavigate, goto } from '$app/navigation';
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
	import Menu from '@lucide/svelte/icons/menu';
	import Search from '@lucide/svelte/icons/search';
	import CircleUser from '@lucide/svelte/icons/circle-user';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	let drawerOpen = $state(false);
	let accountMenuOpen = $state(false);
	let signingOut = $state(false);
	let accountWrap: HTMLElement | undefined = $state();
	// The frame's scroll root (the one scroll container of the app).
	let main: HTMLElement | undefined = $state();
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

	// The frame's scroll reset AND its focus: SvelteKit's navigation
	// focus management resets focus to the BODY before the
	// afterNavigate callbacks run, and the body cannot scroll inside
	// the frame -- so this callback (running after that reset) puts
	// the focus on the scroll root whenever nothing else took it (an
	// autofocus field, a persisted link), keeping the browser's
	// keyboard scrolling (Space, PageDown, the arrows) alive. Push
	// navigations also start at the frame's top: SvelteKit's own
	// scroll reset acts on the document, which no longer scrolls.
	afterNavigate((navigation) => {
		if (main !== undefined) {
			if (navigation.type !== 'popstate') {
				main.scrollTop = 0;
			}
			if (document.activeElement === document.body) {
				main.focus({ preventScroll: true });
			}
		}
	});

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

<div class="shell h-dvh flex flex-col overflow-hidden">
	<header class="shell-topbar flex items-center gap-3 px-4 py-2 border-b border-border bg-card">
		<button
			class="topbar-icon inline-flex items-center justify-center w-9 h-9 rounded-control bg-transparent text-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft"
			type="button"
			aria-label={$_('nav_open_menu')}
			aria-expanded={drawerOpen}
			onclick={() => (drawerOpen = !drawerOpen)}
		>
			<Menu size={22} />
		</button>
		<!-- Wordmark: Nunito 700 in the text color (Navy 900 light,
		     Cream dark), per the brand spec. -->
		<span class="topbar-brand font-display font-bold text-[20px] text-text">{$_('app_name')}</span>
		<!-- Push the search box away from the brand; the upload and
		     account marks stay pinned to the right end. The leading
		     magnifier is decorative (the field's own aria-label names
		     it) and never takes the pointer. -->
		<span class="topbar-search-wrap relative flex-1 max-w-[420px] ml-auto">
			<Search size={16} class="absolute start-3 top-1/2 -translate-y-1/2 text-text-faint pointer-events-none" />
			<input
				class="topbar-search w-full box-content ps-9 pe-3 py-2 font-body font-normal text-[15px] leading-[1.4] text-text bg-page border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
				type="search"
				placeholder={$_('nav_search')}
				aria-label={$_('nav_search')}
			/>
		</span>
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
				<CircleUser size={20} />
			</button>
			{#if accountMenuOpen}
				<div class="account-menu absolute top-[calc(100%_+_8px)] right-0 z-(--z-menu) min-w-[180px] box-content p-2 bg-card border border-border rounded-overlay shadow-overlay">
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
	<!-- The frame's one scroll container: the sticky month/day
	     headers, the scrubber tracking and the batch bar all answer
	     to THIS element now (see the pages' capture-phase scroll
	     listeners). tabindex -1 keeps the scroll root out of the tab
	     order while staying focusable (the navigation focus rule
	     above keeps the browser's keyboard scrolling on it); the
	     outline suppression keeps that focus visually silent. -->
	<main bind:this={main} tabindex="-1" class="shell-main flex-1 min-h-0 overflow-y-auto p-6 focus-visible:outline-none">
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
