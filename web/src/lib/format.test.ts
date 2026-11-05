import { expect, test } from 'vitest';
import { formatDuration, formatMegabytes } from './format';

// The wall's video duration chip (web polish batch 2): the tile's
// bottom-right badge renders m:ss / h:mm:ss so the numeral columns
// stay steady while the layout reflows -- the exact rhythm the
// tabular-nums utility pins on the chip itself.

test('formatDuration renders minutes and zero-padded seconds', () => {
	expect(formatDuration(0)).toBe('0:00');
	expect(formatDuration(45_000)).toBe('0:45');
	expect(formatDuration(4_000)).toBe('0:04');
});

test('formatDuration rounds to the nearest second', () => {
	// 59.7s rounds up into the next minute.
	expect(formatDuration(59_700)).toBe('1:00');
	// 61.5s rounds to 62s.
	expect(formatDuration(61_500)).toBe('1:02');
});

test('formatDuration keeps two-digit minutes and seconds under an hour', () => {
	expect(formatDuration(3_661_000)).toBe('1:01:01');
});

test('formatDuration clamps malformed wire values instead of rendering junk', () => {
	expect(formatDuration(-5_000)).toBe('0:00');
});

test('formatMegabytes keeps one decimal over the binary megabyte', () => {
	expect(formatMegabytes(1_048_576)).toBe('1.0 MB');
	expect(formatMegabytes(400_000_000)).toBe('381.5 MB');
});
