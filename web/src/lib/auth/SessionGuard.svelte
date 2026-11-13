<script lang="ts">
	// The runtime half of the session guard: a store watcher mounted
	// exactly once by the root layout. When any API call's 401 flips
	// the session store to unauthenticated (the interceptor in
	// $lib/auth/api) while a guarded route is showing, it navigates
	// to the login page with the current path as the post-login
	// target. The flip is idempotent and this effect stops matching
	// once the login route is current, so a mid-session expiry
	// produces exactly one navigation -- no dead UI, no retry loop.
	import { goto } from '$app/navigation';
	import { isGuardedPath, loginRedirectTarget } from '$lib/auth/guard';
	import { sessionExpired, sessionStatus } from '$lib/auth/session.svelte';

	type Props = {
		/** Current path + query, reactive from the caller's page state. */
		currentPath?: string;
	};

	let { currentPath = '' }: Props = $props();

	$effect(() => {
		if (sessionStatus() === 'unauthenticated' && isGuardedPath(currentPath)) {
			// A mid-session 401 carries the expiry tag, so the login
			// page can say the session ended rather than just ask for
			// credentials again.
			goto(loginRedirectTarget(currentPath, sessionExpired()));
		}
	});
</script>
