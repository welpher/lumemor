import { expect, test } from 'vitest';
import { navItemIdle, navItemIsCurrent, navItemSelected, tabIdle, tabSelected } from './selectedState';

// The selected-state pin of the two navs and the admin's tab pairs
// (brand §6.5: the accent is a mass; the selected face is a filled
// accent block carrying the on-accent ink). The two class strings must
// stay exclusive: stacking a transparent ground and the plain ink
// beside the accent classes is how the design sample's selected face
// lost the cascade and fell back to the idle look.

test('the selected state fills the accent block and carries the on-accent ink', () => {
	expect(navItemSelected).toContain('bg-accent');
	expect(navItemSelected).toContain('text-accent-on');
	expect(navItemSelected).toContain('hover:bg-accent-strong');
	expect(navItemSelected).toContain('hover:text-accent-on');
});

test('the selected state never carries the idle ground or the plain ink', () => {
	// These are the two tokens the sample stacked beside the accent
	// classes and lost to; their absence is the pin.
	expect(navItemSelected.split(/\s+/)).not.toContain('bg-transparent');
	expect(navItemSelected.split(/\s+/)).not.toContain('text-text');
	// And the keyboard's arrival keeps the selected reading: focus
	// never repaints the block into the soft tint.
	expect(navItemSelected).toContain('focus-visible:bg-accent-strong');
	expect(navItemSelected).not.toContain('focus-visible:bg-accent-soft');
});

test('the idle state keeps the plain ground and offers the soft tint only on hover/focus', () => {
	expect(navItemIdle).toContain('bg-transparent');
	expect(navItemIdle).toContain('text-text');
	expect(navItemIdle).toContain('hover:bg-accent-soft');
	expect(navItemIdle).toContain('focus-visible:bg-accent-soft');
	expect(navItemIdle).not.toContain('bg-accent ');
});

test('the tab faces carry the same exclusive pair', () => {
	expect(tabSelected).toContain('bg-accent');
	expect(tabSelected).toContain('text-accent-on');
	expect(tabSelected.split(/\s+/)).not.toContain('bg-transparent');
	expect(tabSelected.split(/\s+/)).not.toContain('text-text-soft');
	expect(tabIdle).toContain('bg-transparent');
	expect(tabIdle).toContain('text-text-soft');
	expect(tabIdle).toContain('hover:bg-accent-soft');
});

test('a section owns its own route and every route beneath it', () => {
	expect(navItemIsCurrent('/timeline', '/timeline')).toBe(true);
	expect(navItemIsCurrent('/albums', '/albums')).toBe(true);
	expect(navItemIsCurrent('/albums/6f9c2d18', '/albums')).toBe(true);
	expect(navItemIsCurrent('/people/unnamed', '/people')).toBe(true);
	expect(navItemIsCurrent('/people/6f9c2d18', '/people')).toBe(true);
	expect(navItemIsCurrent('/places/6f9c2d18', '/places')).toBe(true);
	expect(navItemIsCurrent('/admin/jobs', '/admin')).toBe(true);
});

test('a sibling route and a bare prefix never claim the section', () => {
	expect(navItemIsCurrent('/timeline', '/albums')).toBe(false);
	expect(navItemIsCurrent('/people', '/places')).toBe(false);
	// `/peoplefoo` is not under `/people`: the boundary is the slash.
	expect(navItemIsCurrent('/peoplefoo', '/people')).toBe(false);
	// The drawer's row for the timeline is exact, not the whole app.
	expect(navItemIsCurrent('/settings', '/timeline')).toBe(false);
	// An unset path (the test mount) selects nothing.
	expect(navItemIsCurrent(undefined, '/timeline')).toBe(false);
	expect(navItemIsCurrent('', '/timeline')).toBe(false);
});
