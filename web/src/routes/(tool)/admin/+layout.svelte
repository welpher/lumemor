<script lang="ts">
	// The admin section's sidebar: the IA's five management sections
	// with Users, Models and the Jobs task panel live (this milestone
	// delivers the last two), and the remaining two visible but
	// disabled until their domains arrive -- the same placeholder
	// discipline as the content shell's drawer.
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
		{ labelKey: 'admin_nav_settings', href: '/admin/settings', disabled: true },
		{ labelKey: 'admin_nav_models', href: '/admin/models', disabled: false },
		{ labelKey: 'admin_nav_jobs', href: '/admin/jobs', disabled: false },
		{ labelKey: 'admin_nav_backup', href: '/admin/backup', disabled: true }
	];
</script>

<div class="admin">
	<nav class="admin-nav" aria-label={$_('admin_nav_label')}>
		<ul class="admin-nav__list">
			{#each items as item (item.labelKey)}
				<li>
					{#if item.disabled}
						<button
							class="admin-nav__item admin-nav__item--disabled"
							type="button"
							disabled
							title={$_('nav_coming_later')}
						>
							{$_(item.labelKey)}
						</button>
					{:else}
						<a
							class="admin-nav__item"
							href={item.href}
							class:admin-nav__item--active={page.url.pathname === item.href}
						>
							{$_(item.labelKey)}
						</a>
					{/if}
				</li>
			{/each}
		</ul>
		<p class="admin-nav__footnote">{$_('nav_coming_later')}</p>
	</nav>
	<div class="admin-body">
		{@render children()}
	</div>
</div>

<style>
	.admin {
		display: flex;
		gap: 24px;
		align-items: flex-start;
	}

	.admin-nav {
		position: sticky;
		top: 16px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		width: 200px;
		flex-shrink: 0;
		padding: 12px 8px;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
	}

	.admin-nav__list {
		display: flex;
		flex-direction: column;
		gap: 2px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.admin-nav__item {
		display: block;
		width: 100%;
		padding: 9px 12px;
		font: 600 15px/1.4 var(--font-body);
		color: var(--color-text);
		text-align: left;
		text-decoration: none;
		border: none;
		border-radius: var(--radius-control);
		background: none;
		cursor: pointer;
		transition: background-color var(--motion-fast) var(--ease-standard);
	}

	a.admin-nav__item:hover,
	a.admin-nav__item:focus-visible {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}

	.admin-nav__item--active {
		font-weight: 700;
		color: var(--color-accent-soft-text);
		background-color: var(--color-accent-soft);
	}

	/* The future-domain placeholders: visible but inert. */
	.admin-nav__item--disabled {
		color: var(--color-text-faint);
		cursor: not-allowed;
	}

	.admin-nav__footnote {
		margin: 8px 8px 0;
		padding-top: 10px;
		border-top: 1px solid var(--color-border);
		font: 400 12px/1.4 var(--font-body);
		color: var(--color-text-faint);
	}

	.admin-body {
		flex: 1;
		min-width: 0;
	}
</style>
