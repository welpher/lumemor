// Small formatting helpers shared by the surfaces: the model
// download's byte counts render as megabytes (the group's pinned
// bytes are hundreds of MB, so the megabyte is the honest unit; one
// decimal keeps the progress line readable while the underlying
// numbers stay exact for the bar's width), and a video's duration
// renders as m:ss for the tile chips (the photo-app convention:
// one column of tabular numerals, no unit letter).

/** One megabyte in bytes, the binary unit the byte counts report in. */
const BYTES_PER_MEGABYTE = 1_048_576;

/** Seconds in one minute, for the duration renderer. */
const SECONDS_PER_MINUTE = 60;

/** Seconds in one hour, for the duration renderer. */
const SECONDS_PER_HOUR = 3600;

/** Format a byte count as megabytes with one decimal, e.g. "381.8 MB". */
export function formatMegabytes(bytes: number): string {
	return `${(bytes / BYTES_PER_MEGABYTE).toFixed(1)} MB`;
}

/**
 * Format a playback duration as m:ss (h:mm:ss past the hour), e.g.
 * "0:45" or "1:02:03" — the wall's duration chips and any other
 * in-place ticking counter read the same rhythm. The seconds round
 * to the nearest whole (a 59.7s clip is "1:00"), and a malformed
 * negative wire value clamps to zero rather than rendering junk.
 */
export function formatDuration(durationMs: number): string {
	const totalSeconds = Math.max(0, Math.round(durationMs / 1000));
	const seconds = totalSeconds % SECONDS_PER_MINUTE;
	const minutes = Math.floor(totalSeconds / SECONDS_PER_MINUTE) % SECONDS_PER_MINUTE;
	const hours = Math.floor(totalSeconds / SECONDS_PER_HOUR);
	const clock = `${hours > 0 ? String(minutes).padStart(2, '0') : minutes}:${String(seconds).padStart(2, '0')}`;
	return hours > 0 ? `${hours}:${clock}` : clock;
}
