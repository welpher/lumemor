import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { expect, test } from 'vitest';

// The palette contract (brand §3, the 2026-11-27 revision): violet is
// the structural accent, gold the emotional fill, and the blue family
// survives only as the logo/chart palette. The tokens are asserted
// verbatim against the brand table, the semantic roles against the
// same table's AA rules -- so a later edit that quietly reintroduces
// a blue control colour or a gold text colour fails here, not in a
// screenshot.

const themeCss = readFileSync(join(import.meta.dirname, 'theme.css'), 'utf8');

/** One selector block's body (`:root { ... }`, `html.dark { ... }`). */
function block(selector: string): string {
	const start = themeCss.indexOf(`${selector} {`);
	expect(start, `theme.css carries the ${selector} block`).toBeGreaterThanOrEqual(0);
	const end = themeCss.indexOf('}', start);
	return themeCss.slice(start, end);
}

/** One block's custom properties, as a name → value map. */
function tokensOf(selector: string): Map<string, string> {
	const tokens = new Map<string, string>();
	for (const match of block(selector).matchAll(/(--[a-z0-9-]+):\s*([^;]+);/g)) {
		tokens.set(match[1]!, match[2]!.trim());
	}
	return tokens;
}

// The two layers the file documents for :root (the raw palette and the
// light roles are declared in the same block) and the dark override
// block.
const light = tokensOf(':root');
const palette = light;
const dark = tokensOf('html.dark');

/** The raw hex of one palette token. */
function hex(name: string): string {
	const value = palette.get(`--color-${name}`);
	expect(value, `--color-${name} is a raw palette token`).toMatch(/^#[0-9a-f]{6}$/);
	return value!;
}

/** WCAG relative luminance of a #rrggbb colour. */
function luminance(value: string): number {
	const channel = (offset: number): number => {
		const c = parseInt(value.slice(offset, offset + 2), 16) / 255;
		return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
	};
	return 0.2126 * channel(1) + 0.7152 * channel(3) + 0.0722 * channel(5);
}

/** The WCAG contrast ratio between two #rrggbb colours. */
function contrast(a: string, b: string): number {
	const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
	return (hi! + 0.05) / (lo! + 0.05);
}

test('the violet and gold families carry the brand table verbatim', () => {
	expect(hex('violet-900')).toBe('#3f2b75');
	expect(hex('violet-700')).toBe('#5b3fa0');
	expect(hex('violet-300')).toBe('#c3b2f0');
	expect(hex('violet-100')).toBe('#ece6f8');
	expect(hex('gold-800')).toBe('#7a5a21');
	expect(hex('gold-200')).toBe('#f7e7c8');
	// The star keeps its own token; the warning/notice tier stays sun.
	expect(hex('gold-600')).toBe('#c9974f');
	expect(hex('sun')).toBe('#f6c15b');
});

test('the light ground is the brightened paper, white cards and warm neutral text', () => {
	expect(hex('paper')).toBe('#fbf7f2');
	expect(hex('surface')).toBe('#ffffff');
	expect(hex('text-secondary')).toBe('#4a4438');
	expect(hex('muted')).toBe('#8a8175');
});

test('the light roles bind accent and link to violet, not blue', () => {
	expect(light.get('--color-accent')).toBe('var(--color-violet-700)');
	expect(light.get('--color-accent-strong')).toBe('var(--color-violet-900)');
	expect(light.get('--color-accent-soft')).toBe('var(--color-violet-100)');
	expect(light.get('--color-accent-soft-text')).toBe('var(--color-violet-900)');
	expect(light.get('--color-accent-on')).toBe('#ffffff');
	expect(light.get('--color-link')).toBe('var(--color-violet-700)');
	expect(light.get('--color-link-hover')).toBe('var(--color-violet-900)');
});

test('the dark roles bind accent to violet-300 and the on-ink to navy', () => {
	expect(dark.get('--color-accent')).toBe('var(--color-violet-300)');
	expect(dark.get('--color-accent-strong')).toBe('color-mix(in srgb, var(--color-violet-300) 82%, white)');
	expect(dark.get('--color-accent-soft')).toBe('color-mix(in srgb, var(--color-violet-300) 18%, transparent)');
	expect(dark.get('--color-accent-soft-text')).toBe('var(--color-violet-300)');
	// The filled block's ink on the dark ground: navy, never a hard white.
	expect(dark.get('--color-accent-on')).toBe('var(--color-navy-900)');
	expect(dark.get('--color-link')).toBe('var(--color-violet-300)');
});

test('the blue family survives as the chart palette and binds no role', () => {
	for (const blue of ['blue-900', 'blue-800', 'blue-700', 'blue-600', 'blue-300', 'blue-100']) {
		expect(palette.has(`--color-${blue}`), `--color-${blue} is kept`).toBe(true);
	}
	const roleBindings = [...light.entries(), ...dark.entries()].filter(
		([name]) =>
			![
				'--color-blue-900',
				'--color-blue-800',
				'--color-blue-700',
				'--color-blue-600',
				'--color-blue-300',
				'--color-blue-100'
			].includes(name)
	);
	for (const [name, value] of roleBindings) {
		expect(value, `${name} does not bind a blue token`).not.toContain('blue');
	}
});

test('the AA pairs hold at their documented ratios (4.5:1 floor)', () => {
	const pairs: Array<[string, string, number]> = [
		// [foreground, background, the brand table's documented ratio]
		['violet-700', 'paper', 7.4],
		['#ffffff', 'violet-700', 7.9],
		['violet-300', 'night', 9.0],
		['navy-900', 'violet-300', 7.2],
		['gold-800', 'gold-200', 5.2]
	];
	for (const [foreground, background, documented] of pairs) {
		const fg = foreground.startsWith('#') ? foreground : hex(foreground);
		const bg = background.startsWith('#') ? background : hex(background);
		const ratio = contrast(fg, bg);
		expect(ratio, `${foreground} on ${background} (${ratio.toFixed(2)}:1)`).toBeGreaterThanOrEqual(4.5);
		expect(ratio, `${foreground} on ${background} keeps its documented ${documented}:1`).toBeCloseTo(documented, 1);
	}
});

test('the empty-state line art carries gold on paper and sun on night', () => {
	expect(themeCss).toMatch(/\.empty-art\s*{[^}]*color:\s*var\(--color-gold-600\);/);
	expect(themeCss).toMatch(/html\.dark\s+\.empty-art\s*{[^}]*color:\s*var\(--color-sun\);/);
});
