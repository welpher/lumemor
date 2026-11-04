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
	<div
		class="drawer__scrim fixed inset-0 z-[90] bg-[color-mix(in_srgb,var(--color-night)_48%,transparent)]"
		role="presentation"
		onclick={close}
	></div>
	<nav class="drawer__panel fixed top-0 bottom-0 left-0 z-[100] flex flex-col gap-2 w-[min(320px,85vw)] px-3 py-4 box-content bg-card border-r border-border shadow-overlay" aria-label={$_('nav_label')}>
		<ul class="drawer__list flex flex-col gap-0.5 list-none">
			{#each items as item (item.labelKey)}
				<li>
					{#if item.disabled}
						<!-- The M3 placeholders: present and visible, but inert
						     until their domains arrive; the title carries the
						     coming-later note. -->
						<button
							class="drawer__item drawer__item--disabled block w-full px-3 py-2.5 font-body font-semibold text-[15px] leading-[1.4] text-text text-left rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard disabled:text-text-faint disabled:cursor-not-allowed"
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
						<a
							class="drawer__item block w-full px-3 py-2.5 font-body font-semibold text-[15px] leading-[1.4] text-text text-left no-underline rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text active:bg-[color-mix(in_srgb,var(--color-accent-soft)_60%,transparent)]"
							href={item.href}
							onclick={close}
						>
							{$_(item.labelKey)}
						</a>
					{/if}
				</li>
			{/each}
		</ul>
		<p class="drawer__footnote mt-auto mx-2 pt-3 border-t border-border font-body font-normal text-[13px] leading-[1.4] text-text-faint">{$_('nav_coming_later')}</p>
	</nav>
{/if}

<style>
	/* The entrance motion (verbatim): the keyframes and the panel's
	 * animation declaration are the drawer's residual styles. */
	.drawer__panel {
		animation: drawer-in var(--motion-base) var(--ease-standard);
	}

	@keyframes drawer-in {
		from {
			transform: translateX(-16px);
			opacity: 0;
		}
	}
</style>
