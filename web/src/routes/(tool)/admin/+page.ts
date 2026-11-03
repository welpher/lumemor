import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// The admin root lands on its one live section; the other sections
// are placeholders of the sidebar.
export const load: PageLoad = () => {
	redirect(307, '/admin/users');
};
