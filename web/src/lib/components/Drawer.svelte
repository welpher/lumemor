<script lang="ts">
	// The navigation drawer of the content shell (the
	// hamburger opens it; Escape and a click outside close it). It
	// carries the content sections in route-table order (the
	// albums wall's entry live since the albums surface; the places
	// grid's entry live since the places surface), plus the archived
	// view's entry, which moved here from the timeline's toolbar --
	// it sits with the other list views (favorites/trash).
	// The list is the eight content sections and NOTHING else:
	// the settings entry that once closed this list is gone (the
	// 2026-11-09 single-home rule -- the account dropdown is
	// settings' ONLY home, landed with the dropdown's own link; a
	// duplicate door in the drawer was navigation debt, not reach).
	// Below the content list, behind a divider, the admin panel's
	// entry renders only for an admin's session (the identity seam's
	// role arm, the deferred item: a member account never sees
	// the management surface's door). The memories entry went live
	// with the memory wall's own route (the manual memories'
	// landing); the section no longer sits disabled.
	//
	// THE LEADING ICONS: every item LEADS with its lucide icon
	// (24-grid, 2px round caps, currentColor -- the brand's icon
	// language); the icons are decorative (aria-hidden), the
	// text stays the accessible name -- an icon never replaces the
	// label it leads.
	import { _ } from 'svelte-i18n';
	import { sessionRole } from '$lib/auth/session.svelte';
	import { navItemIdle, navItemIsCurrent, navItemSelected } from './selectedState';
	import Images from '@lucide/svelte/icons/images';
	import BookImage from '@lucide/svelte/icons/book-image';
	import Users from '@lucide/svelte/icons/users';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Sparkles from '@lucide/svelte/icons/sparkles';
	import Star from '@lucide/svelte/icons/star';
	import Archive from '@lucide/svelte/icons/archive';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Shield from '@lucide/svelte/icons/shield';
	import type { Component } from 'svelte';

	type DrawerItem = {
		labelKey: string;
		href: string;
		/** Disabled items are future-scope placeholders, not hidden: the
		 * content sections stay visible in a disabled state. */
		disabled: boolean;
		/** The leading icon component (decorative, aria-hidden). */
		icon: Component;
		/** The route family the item owns, when it is not simply its href
		 * (the admin door links to its landing but lights up for every
		 * management route). */
		section?: string;
	};

	const items: readonly DrawerItem[] = [
		{ labelKey: 'nav_timeline', href: '/timeline', disabled: false, icon: Images },
		{ labelKey: 'nav_albums', href: '/albums', disabled: false, icon: BookImage },
		{ labelKey: 'nav_people', href: '/people', disabled: false, icon: Users },
		{ labelKey: 'nav_places', href: '/places', disabled: false, icon: MapPin },
		{ labelKey: 'nav_memories', href: '/memories', disabled: false, icon: Sparkles },
		{ labelKey: 'nav_favorites', href: '/favorites', disabled: false, icon: Star },
		{ labelKey: 'nav_archived', href: '/archived', disabled: false, icon: Archive },
		{ labelKey: 'nav_trash', href: '/trash', disabled: false, icon: Trash2 }
	];

	// The admin panel's entry (the tool shell's door): an
	// administrator's session renders it, every other role does not.
	// The landing is the users page -- the first live management
	// section; /admin itself has no index route.
	const adminEntry = $derived(
		sessionRole() === 'admin'
			? { labelKey: 'nav_admin', href: '/admin/users', section: '/admin', icon: Shield }
			: null
	);

	type Props = {
		/** Open state; bind it from the shell, the drawer closes itself. */
		open?: boolean;
		/** The route the shell is on: the drawer marks the owning item as
		 * the selected (filled accent) block with aria-current="page". */
		currentPath?: string;
	};

	let { open = $bindable(false), currentPath = '' }: Props = $props();

	/** The drawer row's classes: the selected block and the idle row are
	 * two exclusive strings (see navItem.ts for why). */
	function itemClass(current: boolean): string {
		const state = current ? navItemSelected : navItemIdle;
		const pressed = current ? '' : 'active:bg-[color-mix(in_srgb,var(--color-accent-soft)_60%,transparent)]';
		return `drawer__item flex items-center gap-3 w-full ps-3 pe-3 py-2.5 font-body font-semibold text-[15px] leading-[1.4] text-left no-underline rounded-control cursor-pointer transition-[background-color,padding] duration-(--motion-fast) ease-standard hover:ps-3.5 focus-visible:ps-3.5 ${state} ${pressed}`;
	}

	function isCurrent(item: { href: string; section?: string }): boolean {
		return navItemIsCurrent(currentPath, item.section ?? item.href);
	}

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
		class="drawer__scrim fixed inset-0 z-(--z-scrim) bg-[color-mix(in_srgb,var(--color-night)_48%,transparent)]"
		role="presentation"
		onclick={close}
	></div>
	<nav
		class="drawer__panel fixed top-0 bottom-0 left-0 z-(--z-drawer) flex flex-col gap-2 w-[min(320px,85vw)] px-3 py-4 box-content bg-card border-r border-border shadow-overlay"
		aria-label={$_('nav_label')}
	>
		<ul class="drawer__list flex flex-col gap-0.5 list-none">
			{#each items as item (item.labelKey)}
				<li>
					{#if item.disabled}
						<!-- The future-scope placeholders: present and visible, but inert
						     until their domains arrive; the title carries the
						     coming-later note. No hover shift: a disabled row
						     owes no affordance. The icon leads the label like
						     every item, decorative. -->
						<button
							class="drawer__item drawer__item--disabled flex items-center gap-3 w-full px-3 py-2.5 font-body font-semibold text-[15px] leading-[1.4] text-text text-left rounded-control bg-transparent cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard disabled:text-text-faint disabled:cursor-not-allowed"
							type="button"
							disabled
							title={$_('nav_coming_later')}
						>
							<item.icon size={18} class="shrink-0" />
							{$_(item.labelKey)}
						</button>
					{:else}
						<!-- Navigating closes the drawer; the SvelteKit router
						     owns the click's default action, so the handler must
						     not prevent it. The hover adds G10's small padding
						     nudge (the 3 -> 3.5 spacing step, ~2px) alongside
						     the background, riding the same fast transition;
						     the icon leads the label, decorative. The current
						     route's row is the filled accent block (the idle
						     row's tint never rides on it: see navItem.ts). -->
						<a
							class={itemClass(isCurrent(item))}
							href={item.href}
							aria-current={isCurrent(item) ? 'page' : undefined}
							onclick={close}
						>
							<item.icon size={18} class="shrink-0" />
							{$_(item.labelKey)}
						</a>
					{/if}
				</li>
			{/each}
		</ul>
		{#if adminEntry !== null}
			<!-- The tool shell's door, separated from the content list:
			     an administrator's session alone renders it (the role
			     arm of the identity seam; a member account's drawer
			     never shows the management surface's entry). -->
			<div class="drawer__divider h-px my-2 bg-border" role="presentation"></div>
			<ul class="drawer__list flex flex-col gap-0.5 list-none">
				<li>
					<a
						class={itemClass(isCurrent(adminEntry))}
						href={adminEntry.href}
						aria-current={isCurrent(adminEntry) ? 'page' : undefined}
						onclick={close}
					>
						<adminEntry.icon size={18} class="shrink-0" />
						{$_(adminEntry.labelKey)}
					</a>
				</li>
			</ul>
		{/if}
		<p
			class="drawer__footnote mt-auto mx-2 pt-3 border-t border-border font-body font-normal text-[13px] leading-[1.4] text-text-faint"
		>
			{$_('nav_coming_later')}
		</p>
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
