<script lang="ts">
	// The tool shell of the web IA: the persistent-navigation shell
	// behind /admin/* (and, later, /settings/*). Its top bar is the
	// simplified one the IA pins -- a back-to-content mark with the
	// wordmark instead of the content shell's full bar -- and the
	// section navigation itself lives in the section layouts (the
	// admin sidebar under (tool)/admin).
	//
	// THE APP FRAME (R2-1's consistency arm): the shell shares the
	// content shell's structure, so it shares the frame treatment
	// too -- the h-dvh flex column pins the top bar by layout, main
	// is the one scroll container (the admin rail's own sticky
	// top-0 pins flush to it), the scroll root takes the focus at
	// mount so keyboard scrolling follows it, and push navigations
	// reset it (popstate keeps the offset).
	//
	// THE FULL-BLEED PIN (2026-11-09, the admin-shell specimen):
	// main carries NO padding -- the content column owns its own
	// gutter (the admin layout's wrapper), so the flush rail and
	// the full-height column split the frame's whole extent.
	import { _ } from 'svelte-i18n';
	import { afterNavigate, goto } from '$app/navigation';
	import OfflineStrip from '$lib/events/OfflineStrip.svelte';
	import ShortcutsDialog from '$lib/components/ShortcutsDialog.svelte';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	// The frame's scroll root (the one scroll container of the shell).
	let main: HTMLElement | undefined = $state();

	// The shortcut sheet's open state (the ShortcutsDialog owns its
	// own `?` toggle; the tool shell mounts it so the app's reference
	// is reachable from the management surfaces too -- the sheet
	// names each binding's context, so its rows stay honest wherever
	// it opens).
	let shortcutsOpen = $state(false);

	// The frame's scroll reset and focus (the content shell's rule,
	// mirrored for consistency -- see that shell's note for the
	// SvelteKit focus-reset ordering).
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

	function backToContent(): void {
		goto('/timeline');
	}
</script>

<div class="tool h-dvh flex flex-col overflow-hidden">
	<!-- The offline strip (web IA §6): the frame's FIRST child, above
	     the topbar, exactly as the content shell mounts it. The tool
	     shell earns the strip honestly: its jobs and models surfaces
	     subscribe to the SAME shared SSE stream (job.* events), so a
	     dropped connection is this shell's truth too -- and on the
	     routes without a subscriber the store reads 'closed' (nothing
	     live, nothing lost), so the strip simply never rises there. -->
	<OfflineStrip />
	<header class="tool-topbar flex items-center gap-3 px-4 py-2 border-b border-border bg-card">
		<button
			class="tool-topbar__back inline-flex items-center justify-center w-9 h-9 rounded-control bg-transparent text-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft"
			type="button"
			aria-label={$_('tool_back_to_content')}
			onclick={backToContent}
		>
			<ChevronLeft size={22} />
		</button>
		<span class="tool-topbar__brand font-display font-bold text-[20px] text-text">{$_('app_name')}</span>
	</header>
	<!-- The `?` shortcut sheet (web IA §6): the same shared Modal the
	     content shell mounts, one instance per shell. -->
	<ShortcutsDialog bind:open={shortcutsOpen} />
	<!-- The frame's one scroll container (the content shell's rule,
	     mirrored for consistency). The 2026-11-09 full-bleed pin: NO
	     padding -- the admin content column below owns its gutter, so
	     the flush rail splits the scroll extent edge to edge. -->
	<main bind:this={main} tabindex="-1" class="tool-main flex-1 min-h-0 overflow-y-auto focus-visible:outline-none">
		{@render children()}
	</main>
</div>
