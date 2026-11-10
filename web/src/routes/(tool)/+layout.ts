import type { LayoutLoad } from './$types';
import { guardSessionRoute } from '$lib/auth/guard';

// Tool-shell routes sit behind the same session guard as the
// content shell (the tool-specific shell itself arrives
// later). The root layout's load -- awaited through
// parent() -- has already resolved the session state.
export const load: LayoutLoad = async ({ url, parent }) => {
	await guardSessionRoute(url, parent);
};
