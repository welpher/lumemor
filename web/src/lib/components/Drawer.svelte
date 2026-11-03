<script lang="ts">
	// The navigation drawer of the content shell (web IA: the
	// hamburger opens it; Escape and a click outside close it). It
	// carries the IA's seven content sections in route-table order:
	// Timeline, Favorites and Trash link to their routes now (their
	// real views are later tasks of this milestone), while Albums,
	// People, Places and Memories stay visible-but-disabled
	// placeholders until their M3 domains arrive.
	import { _ } from 'svelte-i18n';

	type DrawerItem = {
		labelKey: string;
		href: string;
		/** Disabled items are M3 placeholders, not hidden: the IA's
		 * seven sections stay visible in a disabled state. */
		disabled: boolean;
	};

	const items: readonly DrawerItem[] = [
		{ labelKey: 'nav_timeline', href: '/timeline', disabled: false },
		{ labelKey: 'nav_albums', href: '/albums', disabled: true },
		{ labelKey: 'nav_people', href: '/people', disabled: true },
		{ labelKey: 'nav_places', href: '/places', disabled: true },
		{ labelKey: 'nav_memories', href: '/memories', disabled: true },
		{ labelKey: 'nav_favorites', href: '/favorites', disabled: false },
		{ labelKey: 'nav_trash', href: '/trash', disabled: false }
	];

	type Props = {
		/** Open state; bind it from the shell, the drawer closes itself. */
		open?: boolean;
	};

	let { open = $bindable(false) }: Props = $props();

	function close(): void {
		open = false;
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (open && event.key === 'Escape') {
			close();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<!-- The scrim is the click-outside dismissal: it covers everything
	     except the panel, so a click on the page content, the top bar
	     or the hamburger closes the drawer. -->
	<div class="drawer__scrim" role="presentation" onclick={close}></div>
	<nav class="drawer__panel" aria-label={$_('nav_label')}>
		<ul class="drawer__list">
			{#each items as item (item.labelKey)}
				<li>
					{#if item.disabled}
						<button
							class="drawer__item drawer__item--disabled"
							type="button"
							disabled
							title={$_('nav_coming_later')}
						>
							{$_(item.labelKey)}
						</button>
					{:else}
						<!-- Navigating closes the drawer; the SvelteKit router
						     owns the click's default action, so the handler must
						     not prevent it. -->
						<a class="drawer__item" href={item.href} onclick={close}>
							{$_(item.labelKey)}
						</a>
					{/if}
				</li>
			{/each}
		</ul>
		<p class="drawer__footnote">{$_('nav_coming_later')}</p>
	</nav>
{/if}

<style>
	.drawer__scrim {
		position: fixed;
		inset: 0;
		z-index: 90;
		background-color: color-mix(in srgb, var(--color-night) 48%, transparent);
	}

	.drawer__panel {
		position: fixed;
		top: 0;
		bottom: 0;
		left: 0;
		z-index: 100;
		display: flex;
		flex-direction: column;
		gap: 8px;
		width: min(320px, 85vw);
		padding: 16px 12px;
		background-color: var(--color-card);
		border-right: 1px solid var(--color-border);
		box-shadow: var(--shadow-overlay);
		animation: drawer-in var(--motion-base) var(--ease-standard);
	}

	.drawer__list {
		display: flex;
		flex-direction: column;
		gap: 2px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.drawer__item {
		display: block;
		width: 100%;
		padding: 10px 12px;
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

	a.drawer__item:hover,
	a.drawer__item:focus-visible {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}

	a.drawer__item:active {
		background-color: color-mix(in srgb, var(--color-accent-soft) 60%, transparent);
	}

	/* The M3 placeholders: present and visible, but inert until their
	 * domains arrive; the title carries the coming-later note. */
	.drawer__item--disabled {
		color: var(--color-text-faint);
		cursor: not-allowed;
	}

	.drawer__footnote {
		margin: auto 8px 0;
		padding-top: 12px;
		border-top: 1px solid var(--color-border);
		font: 400 13px/1.4 var(--font-body);
		color: var(--color-text-faint);
	}

	@keyframes drawer-in {
		from {
			transform: translateX(-16px);
			opacity: 0;
		}
	}
</style>
