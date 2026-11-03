// Small formatting helpers shared by the admin surfaces: the model
// download's byte counts render as megabytes (the group's pinned
// bytes are hundreds of MB, so the megabyte is the honest unit; one
// decimal keeps the progress line readable while the underlying
// numbers stay exact for the bar's width).

/** One megabyte in bytes, the binary unit the byte counts report in. */
const BYTES_PER_MEGABYTE = 1_048_576;

/** Format a byte count as megabytes with one decimal, e.g. "381.8 MB". */
export function formatMegabytes(bytes: number): string {
	return `${(bytes / BYTES_PER_MEGABYTE).toFixed(1)} MB`;
}
