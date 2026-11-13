import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { expect, test } from 'vitest';

// The field discipline (the form-surface pin): every input, select and
// textarea rests on one edge and focuses with one ring. The platform's
// native inner frame on search fields is closed in the base layer (an
// appearance reset plus the two webkit pseudo-elements), the focus ring
// tucks onto the field's own edge (-1px) so focus never reads as an
// edge inside a ring, and no field may repaint its border on focus or
// delete the ring with `outline-none` -- the ring is the ONLY focus
// feedback a field has.

const themeCss = readFileSync(join(import.meta.dirname, 'theme.css'), 'utf8');
const srcRoot = join(import.meta.dirname, '..', '..');

/** Every .svelte file under web/src, as absolute paths. */
function svelteFiles(directory: string): string[] {
	const found: string[] = [];
	for (const entry of readdirSync(directory)) {
		const path = join(directory, entry);
		if (statSync(path).isDirectory()) {
			found.push(...svelteFiles(path));
		} else if (entry.endsWith('.svelte')) {
			found.push(path);
		}
	}
	return found;
}

/** One file's markup with its <script> blocks dropped (they discuss the rules). */
function markup(path: string): string {
	return readFileSync(path, 'utf8').replaceAll(/<script[\s\S]*?<\/script>/g, '');
}

test('the base layer closes the platform search frame', () => {
	// `appearance: none` alone is not enough: Safari and Chromium paint
	// their own decorations inside the field, so both pseudo-elements
	// are reset with it.
	const rule = /input\[type='search'\]\s*{[^}]*appearance:\s*none;/;
	expect(themeCss).toMatch(rule);
	for (const pseudo of ['::-webkit-search-decoration', '::-webkit-search-cancel-button']) {
		expect(themeCss, `theme.css resets ${pseudo}`).toContain(`input[type='search']${pseudo}`);
	}
});

test('the field ring sits on the field edge, the button and link ring keeps its gap', () => {
	// The global rule keeps the 2px gap; the field selector overrides
	// the offset alone (the ring's width and color stay one source).
	expect(themeCss).toMatch(/:focus-visible\s*{[^}]*outline-offset:\s*2px;/);
	const fieldRule = /input:focus-visible,\s*select:focus-visible,\s*textarea:focus-visible\s*{([^}]*)}/;
	const body = themeCss.match(fieldRule)?.[1] ?? '';
	expect(body).toContain('outline-offset: -1px');
});

test('no field repaints its border on focus or suppresses its ring', () => {
	const offenders: string[] = [];
	for (const path of svelteFiles(srcRoot)) {
		const source = markup(path);
		expect(source, `${path} retires the border-on-focus utility`).not.toContain('focus:border-');
		for (const tag of source.matchAll(/<(input|select|textarea)\b[^>]*>/g)) {
			const text = tag[0];
			if (text.includes('outline-none') || /focus(-visible)?:border-/.test(text)) {
				offenders.push(`${path}: ${text.replaceAll(/\s+/g, ' ').slice(0, 120)}`);
			}
		}
	}
	expect(offenders).toEqual([]);
});
