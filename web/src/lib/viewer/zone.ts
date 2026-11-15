// The capture-time zone helpers of the viewer: the stored instant
// and its display zone (EXIF offset notation, e.g. `+08:00`) render
// as one wall clock -- `2026-08-12 18:42` beside `· UTC+8`.
//
// Intl cannot name a fixed offset as a time zone, so the display
// shifts the instant by the offset and formats the shifted date in
// UTC parts -- the shift IS the zone, the same arithmetic the edit
// dialog composes the PATCH value with.

/** The stored time shifted into the zone's wall clock. */
export function shiftedToZone(instant: string, zone: string | null): Date {
	const date = new Date(instant);
	const offset = zone ?? '+00:00';
	const match = /^([+-])(\d{2}):(\d{2})$/.exec(offset);
	const sign = match?.[1] === '-' ? -1 : 1;
	const minutes = sign * (Number(match?.[2] ?? 0) * 60 + Number(match?.[3] ?? 0));
	return new Date(date.getTime() + minutes * 60_000);
}

/** The EXIF-notation offset rendered the way the panel reads it (`UTC+8`). */
export function zoneLabel(zone: string): string {
	const match = /^([+-])(\d{2}):(\d{2})$/.exec(zone);
	if (match === null) {
		return zone;
	}
	const [, sign, hours, minutes] = match;
	const half = minutes === '00' ? '' : `:${minutes}`;
	return `UTC${sign}${Number(hours)}${half}`;
}

/** The capture time formatted in its zone's wall clock, locale-aware. */
export function formatTakenAt(instant: string, zone: string | null, locale: string): string {
	const shifted = shiftedToZone(instant, zone);
	return new Intl.DateTimeFormat(locale, {
		timeZone: 'UTC',
		dateStyle: 'medium',
		timeStyle: 'short'
	}).format(shifted);
}

/** The `datetime-local` value of the instant rendered in the zone. */
export function localInputValue(instant: string, zone: string | null): string {
	const shifted = shiftedToZone(instant, zone);
	const pad = (value: number): string => String(value).padStart(2, '0');
	return (
		`${shifted.getUTCFullYear()}-${pad(shifted.getUTCMonth() + 1)}-${pad(shifted.getUTCDate())}` +
		`T${pad(shifted.getUTCHours())}:${pad(shifted.getUTCMinutes())}`
	);
}
