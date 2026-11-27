// THE SELECTED-STATE LANGUAGE (brand §3/§6.5): the primary accent is a
// mass, not a tint. The chosen face of a navigation rail, a tab pair or
// a segmented control is a FILLED accent block carrying the on-accent
// ink; every other face stays transparent with plain ink and a soft
// hover.
//
// THE STATE STRINGS ARE MUTUALLY EXCLUSIVE by construction: a selected
// element never also carries `bg-transparent` or the plain `text-text`
// token, so no utility-order race can repaint it into the idle face --
// exactly the failure the design sample hit, and the reason these class
// strings live here instead of being stacked beside each other.
export const navItemSelected =
	'bg-accent text-accent-on hover:bg-accent-strong hover:text-accent-on focus-visible:bg-accent-strong focus-visible:text-accent-on';

export const navItemIdle =
	'bg-transparent text-text hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text';

// The same two faces for the admin's tab pairs (the face sits on the
// group's own page-coloured track, so the idle ink is the soft tier).
export const tabSelected = 'bg-accent text-accent-on hover:bg-accent-strong';

export const tabIdle = 'bg-transparent text-text-soft hover:bg-accent-soft hover:text-accent-soft-text';

/** Whether `href`'s section owns `pathname`: its own route or any route beneath it. */
export function navItemIsCurrent(pathname: string | undefined, href: string): boolean {
	if (pathname === undefined || pathname === '') {
		return false;
	}
	return pathname === href || pathname.startsWith(href.endsWith('/') ? href : `${href}/`);
}
