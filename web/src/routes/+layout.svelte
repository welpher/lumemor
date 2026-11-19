<script lang="ts">
	// The app shell root: global styles, fonts and theme state live
	// here so every route -- the standalone login page included --
	// shares them, and the SessionGuard watcher covers mid-session
	// expiry for every guarded tree. The visible chrome (top bar,
	// drawer, avatar menu) belongs to the (content) group layout.
	import '@fontsource/nunito/400.css';
	import '@fontsource/nunito/600.css';
	import '@fontsource/nunito/700.css';
	import '../lib/theme/theme.css';
	import { page } from '$app/state';
	import { initTheme } from '$lib/theme/theme.svelte';
	import { adoptAccountTheme } from '$lib/theme/accountTheme';
	import { sessionStatus } from '$lib/auth/session.svelte';
	import SessionGuard from '$lib/auth/SessionGuard.svelte';
	import Toasts from '$lib/components/Toasts.svelte';

	let { children } = $props();

	initTheme();

	// The account's theme, once the session resolves: a restored session
	// is authenticated before the first render (the boot probe), a fresh
	// sign-in flips the status later -- both land here exactly once. The
	// account row is the cross-device truth the settings chip renders,
	// so the page's class must follow it too (a browser with no stored
	// preference otherwise opens in the OS theme while the chip says
	// something else).
	let adoptedAccountTheme = false;
	$effect(() => {
		if (adoptedAccountTheme || sessionStatus() !== 'authenticated') {
			return;
		}
		adoptedAccountTheme = true;
		void adoptAccountTheme();
	});
</script>

<SessionGuard currentPath={page.url.pathname + page.url.search} />
<!-- The transient-confirmation organ: mounted once at the root so
     every surface's success echo (the face-correction flows, the
     batch counts) renders above whatever page is live. -->
<Toasts />
{@render children()}
