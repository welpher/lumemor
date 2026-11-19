// The account's theme is the cross-device truth: the settings page's
// chip renders it, so the page's own class must follow it too. The app
// shell adopts it once per boot (and once per fresh sign-in), which is
// what makes a browser with no stored preference -- a new device, a
// cleared profile -- open in the account's theme instead of the OS
// default. A failed read leaves the local choice standing.

import { getUserSelf } from '$lib/api/userSettings';
import { setThemePreference, themePreferenceOf } from './theme.svelte';

/** Adopt the signed-in account's stored theme; a failed read is a no-op. */
export async function adoptAccountTheme(fetcher: typeof fetch = globalThis.fetch): Promise<void> {
	const self = await getUserSelf(fetcher);
	if (self.ok) {
		setThemePreference(themePreferenceOf(self.self.theme));
	}
}
