<script lang="ts">
	// The admin section's sidebar: the IA's five management sections
	// with Users, Settings, Models and the Jobs task panel live (the
	// settings section carries the storage template), and the
	// remaining placeholder stays visible but disabled until its
	// domain arrives -- the same placeholder discipline as the
	// content shell's drawer.
	import { _ } from 'svelte-i18n';
	import { page } from '$app/state';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	type AdminNavItem = {
		labelKey: string;
		href: string;
		/** Disabled items are future-domain placeholders, not hidden. */
		disabled: boolean;
	};

	const items: readonly AdminNavItem[] = [
		{ labelKey: 'admin_nav_users', href: '/admin/users', disabled: false },
		{ labelKey: 'admin_nav_settings', href: '/admin/settings', disabled: false },
		{ labelKey: 'admin_nav_models', href: '/admin/models', disabled: false },
		{ labelKey: 'admin_nav_jobs', href: '/admin/jobs', disabled: false },
		{ labelKey: 'admin_nav_backup', href: '/admin/backup', disabled: true }
	];
</script>

<div class="admin flex gap-6 items-start">
	<nav class="admin-nav sticky top-4 flex flex-col gap-2 w-[200px] box-content shrink-0 px-2 py-3 bg-card border border-border rounded-card shadow-card" aria-label={$_('admin_nav_label')}>
		<ul class="admin-nav__list flex flex-col gap-0.5 list-none">
			{#each items as item (item.labelKey)}
				<li>
					{#if item.disabled}
						<!-- The future-domain placeholders: visible but inert. -->
						<button
							class="admin-nav__item admin-nav__item--disabled block w-full px-3 py-[9px] font-body font-semibold text-[15px] leading-[1.4] text-text text-left rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard disabled:text-text-faint disabled:cursor-not-allowed"
							type="button"
							disabled
							title={$_('nav_coming_later')}
						>
							{$_(item.labelKey)}
						</button>
					{:else}
						<a
							class="admin-nav__item block w-full px-3 py-[9px] font-body text-[15px] leading-[1.4] text-left no-underline rounded-control cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
							href={item.href}
							class:admin-nav__item--active={page.url.pathname === item.href}
							class:bg-accent-soft={page.url.pathname === item.href}
							class:bg-transparent={page.url.pathname !== item.href}
							class:text-accent-soft-text={page.url.pathname === item.href}
							class:text-text={page.url.pathname !== item.href}
							class:font-bold={page.url.pathname === item.href}
							class:font-semibold={page.url.pathname !== item.href}
						>
							{$_(item.labelKey)}
						</a>
					{/if}
				</li>
			{/each}
		</ul>
		<p class="admin-nav__footnote mt-2 mx-2 pt-2.5 border-t border-border font-body font-normal text-[12px] leading-[1.4] text-text-faint">{$_('nav_coming_later')}</p>
	</nav>
	<div class="admin-body flex-1 min-w-0">
		{@render children()}
	</div>
</div>
