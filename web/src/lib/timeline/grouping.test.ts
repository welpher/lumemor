import { describe, expect, test } from 'vitest';
import { groupTimelineItems, monthKeyOf, dayKeyOf } from './grouping';
import type { TimelineItem } from '$lib/api/timeline';

// The month/day grouping of the timeline wall: items ordered takenAt
// descending fold into month groups with day sub-groups, the labels
// come from the active locale through Intl (no hardcoded month
// strings), and the grouping date is the wire value's own UTC date
// (the pinned v0 timezone decision -- no client-side timezone drift).

const baseItem = {
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image',
	originalFilename: 'IMG_0001.jpg',
	fileSize: 4821933,
	mimeType: 'image/jpeg',
	checksum: '9f86d081884c7d65a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
	createdAt: '2026-10-29T09:12:00Z',
	takenAtSource: 'exif',
	isTrashed: false,
	isFavorite: false,
	isReady: true,
	width: 4032,
	height: 3024,
	durationMs: null,
	blurhash: 'UHL6aAof00WBWB00ayWB00',
	livePhotoVideoId: null
} as const;

/** A timeline item factory: defaults plus per-test overrides. */
function item(id: string, takenAt: string, overrides: Partial<TimelineItem> = {}): TimelineItem {
	return { id, takenAt, ...baseItem, ...overrides } as TimelineItem;
}

describe('groupTimelineItems', () => {
	test('groups by month then day, newest first, preserving item order', () => {
		const items = [
			item('a', '2026-11-02T10:00:00Z'),
			item('b', '2026-11-02T09:00:00Z'),
			item('c', '2026-10-13T12:30:00Z'),
			item('d', '2026-09-30T23:59:00Z')
		];

		const groups = groupTimelineItems(items, 'en');

		expect(groups.map((g) => g.key)).toEqual(['2026-11', '2026-10', '2026-09']);
		expect(groups[0]?.days.map((d) => d.key)).toEqual(['2026-11-02']);
		expect(groups[0]?.days[0]?.items.map((i) => i.id)).toEqual(['a', 'b']);
		expect(groups[1]?.days.map((d) => d.key)).toEqual(['2026-10-13']);
	});

	test('same day across different years and months stays separate', () => {
		// Input arrives in keyset order (newest first); the fold
		// preserves it rather than re-sorting.
		const items = [item('b', '2026-10-13T12:00:00Z'), item('a', '2025-10-13T12:00:00Z')];

		const groups = groupTimelineItems(items, 'en');

		expect(groups.map((g) => g.key)).toEqual(['2026-10', '2025-10']);
	});

	test('labels are locale-aware month and day names', () => {
		const items = [item('a', '2026-10-13T12:30:00Z')];

		const english = groupTimelineItems(items, 'en');
		const chinese = groupTimelineItems(items, 'zh-Hans');

		expect(english[0]?.label).toBe('October 2026');
		expect(english[0]?.days[0]?.label).toBe('October 13');
		expect(chinese[0]?.label).toBe('2026年10月');
		expect(chinese[0]?.days[0]?.label).toBe('10月13日');
	});

	test('the grouping date is the UTC date of the wire value', () => {
		// 23:30 UTC on the 13th is already the 14th far east; the v0
		// grouping stays on the wire value's own UTC calendar day.
		const items = [item('a', '2026-10-13T23:30:00Z')];

		const groups = groupTimelineItems(items, 'en');

		expect(groups[0]?.days[0]?.key).toBe('2026-10-13');
	});

	test('items with an unparseable takenAt are dropped, not fatal', () => {
		const items = [item('a', 'not-a-date'), item('b', '2026-10-13T12:30:00Z')];

		const groups = groupTimelineItems(items, 'en');

		expect(groups[0]?.days[0]?.items.map((i) => i.id)).toEqual(['b']);
	});

	test('no items means no groups', () => {
		expect(groupTimelineItems([], 'en')).toEqual([]);
	});
});

describe('key helpers', () => {
	test('monthKeyOf and dayKeyOf format the UTC date of a wire value', () => {
		expect(monthKeyOf('2026-10-13T12:30:00Z')).toBe('2026-10');
		expect(dayKeyOf('2026-10-13T12:30:00Z')).toBe('2026-10-13');
	});

	test('key helpers answer null for unparseable values', () => {
		expect(monthKeyOf('nope')).toBeNull();
		expect(dayKeyOf('nope')).toBeNull();
	});
});
