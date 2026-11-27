<script lang="ts">
	// The admin section's sidebar (the full-bleed rework:
	// the rounded capsule is DEAD): the rail is a flush full-height
	// track -- no outer radius, no outer border or shadow, no gap to
	// the content, no top offset -- split from the content by its
	// `border-r` alone, 200px wide, the five management sections with
	// Users, Settings, Machine Learning and the Jobs task panel live
	// (the settings page carries the storage template), and the
	// remaining placeholder stays visible but disabled until its
	// domain arrives -- the same placeholder discipline as the
	// content shell's drawer.
	//
	// Each section names itself with its 18px glyph (icon-first): the
	// icons carry aria-hidden and inherit the nav item's text color,
	// so the selected state's color swap covers the glyph with the
	// label.
	import { _ } from 'svelte-i18n';
	import { page } from '$app/state';
	import { navItemIdle, navItemIsCurrent, navItemSelected } from '$lib/components/selectedState';
	import type { LayoutProps } from './$types';
	import Users from '@lucide/svelte/icons/users';
	import Sliders from '@lucide/svelte/icons/sliders';
	import Brain from '@lucide/svelte/icons/brain';
	import ListChecks from '@lucide/svelte/icons/list-checks';
	import DatabaseBackup from '@lucide/svelte/icons/database-backup';

	let { children }: LayoutProps = $props();

	type AdminNavItem = {
		labelKey: string;
		href: string;
		/** Disabled items are future-domain placeholders, not hidden. */
		disabled: boolean;
		icon: typeof Users;
	};

	const items: readonly AdminNavItem[] = [
		{ labelKey: 'admin_nav_users', href: '/admin/users', disabled: false, icon: Users },
		{ labelKey: 'admin_nav_settings', href: '/admin/settings', disabled: false, icon: Sliders },
		{ labelKey: 'admin_nav_models', href: '/admin/models', disabled: false, icon: Brain },
		{ labelKey: 'admin_nav_jobs', href: '/admin/jobs', disabled: false, icon: ListChecks },
		{ labelKey: 'admin_nav_backup', href: '/admin/backup', disabled: true, icon: DatabaseBackup }
	];

	/** Whether the rail's row is the current route (see selectedState.ts). */
	function isCurrent(item: AdminNavItem): boolean {
		return navItemIsCurrent(page.url.pathname, item.href);
	}
</script>

<div class="admin flex min-h-full items-stretch">
	<!-- The rail is its own scroll region in two layers: the OUTER nav
	     stretches to the row's full height (the background and the
	     dividing border span the column edge to edge), and the INNER
	     pin carries the menu -- sticky top-0 with `self-start` (a
	     stretched sticky element has zero travel and scrolls away --
	     the reported bug), capped at the scrollport (the top bar's
	     53px: the 36px control + its 2x8px padding + the border) and
	     scrolling on its own overflow when the menu ever grows past
	     it. The menu never moves with the body's scrolling; the
	     rail's face never leaves. -->
	<nav
		class="admin-nav flex flex-col w-[200px] shrink-0 bg-card border-r border-border"
		aria-label={$_('admin_nav_label')}
	>
		<div
			class="admin-nav__pin sticky top-0 self-start w-full max-h-[calc(100dvh-53px)] overflow-y-auto flex flex-col gap-2 py-3 px-2"
		>
			<ul class="admin-nav__list flex flex-col gap-0.5 list-none">
				{#each items as item (item.labelKey)}
					<li>
						{#if item.disabled}
							<!-- The future-domain placeholders: visible but inert. -->
							<button
								class="admin-nav__item admin-nav__item--disabled flex w-full items-center gap-2 px-3 py-[9px] font-body font-semibold text-[15px] leading-[1.4] text-text text-left rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard disabled:text-text-faint disabled:cursor-not-allowed"
								type="button"
								disabled
								title={$_('nav_coming_later')}
							>
								<item.icon size={18} strokeWidth={2} aria-hidden="true" class="shrink-0" />
								{$_(item.labelKey)}
							</button>
						{:else}
							<a
								class="admin-nav__item flex w-full items-center gap-2 px-3 py-[9px] font-body text-[15px] leading-[1.4] text-left no-underline rounded-control cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard {isCurrent(
									item
								)
									? navItemSelected
									: navItemIdle}"
								href={item.href}
								class:admin-nav__item--active={isCurrent(item)}
								class:font-bold={isCurrent(item)}
								class:font-semibold={!isCurrent(item)}
								aria-current={isCurrent(item) ? 'page' : undefined}
							>
								<item.icon size={18} strokeWidth={2} aria-hidden="true" class="shrink-0" />
								{$_(item.labelKey)}
							</a>
						{/if}
					</li>
				{/each}
			</ul>
			<p
				class="admin-nav__footnote mt-2 mx-2 pt-2.5 border-t border-border font-body font-normal text-[12px] leading-[1.4] text-text-faint"
			>
				{$_('nav_coming_later')}
			</p>
		</div>
	</nav>
	<div class="admin-body flex-1 min-w-0 px-6 pt-4 pb-12">
		{@render children()}
	</div>
</div>
