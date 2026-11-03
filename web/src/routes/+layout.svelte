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
	import SessionGuard from '$lib/auth/SessionGuard.svelte';

	let { children } = $props();

	initTheme();
</script>

<SessionGuard currentPath={page.url.pathname + page.url.search} />
{@render children()}
