// Month/day grouping of the timeline wall: the loaded items (already
// ordered takenAt descending, id descending -- the keyset ordering of
// the pagination client) fold into month groups carrying day
// sub-groups. The labels are locale-aware through Intl with the
// active i18n locale, so no month name is hardcoded.
//
// THE PINNED V0 TIMEZONE DECISION: the grouping calendar date is the
// UTC date of the wire value itself, not the browser's local date.
// The server stores takenAt in UTC and normalizes the EXIF offset
// chain, so grouping by the wire date is deterministic across
// devices; a timezone-aware regrouping is an M3 refinement.

import type { TimelineItem } from '$lib/api/timeline';

/** One day of one month group: the items of that UTC calendar day. */
export type DayGroup = {
	/** `YYYY-MM-DD` of the group's UTC calendar day. */
	key: string;
	/** Locale-aware day label (for example "October 13"). */
	label: string;
	items: TimelineItem[];
};

/** One month group: its days in descending date order. */
export type MonthGroup = {
	/** `YYYY-MM` of the group's UTC calendar month. */
	key: string;
	/** Locale-aware month label (for example "October 2026"). */
	label: string;
	days: DayGroup[];
};

function utcDateOf(takenAt: string): Date | null {
	const date = new Date(takenAt);
	return Number.isNaN(date.getTime()) ? null : date;
}

/** The month key (`YYYY-MM`) of a wire takenAt; null when unparseable. */
export function monthKeyOf(takenAt: string): string | null {
	const date = utcDateOf(takenAt);
	return date === null ? null : `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`;
}

/** The day key (`YYYY-MM-DD`) of a wire takenAt; null when unparseable. */
export function dayKeyOf(takenAt: string): string | null {
	const date = utcDateOf(takenAt);
	if (date === null) {
		return null;
	}
	return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}-${String(date.getUTCDate()).padStart(2, '0')}`;
}

/**
 * Fold timeline items into month/day groups. The input order is the
 * rendering order (the keyset ordering, newest first): the fold keeps
 * it, so items never re-sort client-side. An item whose takenAt does
 * not parse is dropped -- a malformed wire value must not take the
 * whole wall down (the pinned defensive choice).
 */
export function groupTimelineItems(items: readonly TimelineItem[], locale: string): MonthGroup[] {
	const monthFormatter = new Intl.DateTimeFormat(locale, {
		month: 'long',
		year: 'numeric',
		timeZone: 'UTC'
	});
	const dayFormatter = new Intl.DateTimeFormat(locale, {
		month: 'long',
		day: 'numeric',
		// The weekday turns a bare date into a memory cue (the rich
		// date labels: "11月1日星期日" / "Sunday, November 1").
		weekday: 'long',
		timeZone: 'UTC'
	});

	const months: MonthGroup[] = [];
	const monthByKey = new Map<string, MonthGroup>();
	const dayByKey = new Map<string, DayGroup>();

	for (const timelineItem of items) {
		const monthKey = monthKeyOf(timelineItem.takenAt);
		const dayKey = dayKeyOf(timelineItem.takenAt);
		if (monthKey === null || dayKey === null) {
			continue;
		}

		let month = monthByKey.get(monthKey);
		if (month === undefined) {
			month = {
				key: monthKey,
				label: monthFormatter.format(new Date(timelineItem.takenAt)),
				days: []
			};
			monthByKey.set(monthKey, month);
			months.push(month);
		}

		let day = dayByKey.get(dayKey);
		if (day === undefined) {
			day = {
				key: dayKey,
				label: dayFormatter.format(new Date(timelineItem.takenAt)),
				items: []
			};
			dayByKey.set(dayKey, day);
			month.days.push(day);
		}
		day.items.push(timelineItem);
	}

	return months;
}
