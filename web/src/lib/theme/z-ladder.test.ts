import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { expect, test } from 'vitest';

// The stacking-ladder contract (web polish batch 2, G7 + G6 + G5):
// theme.css is the single source of the app's z-index ladder, the
// anti-flicker motion delays, the body tracking and the scrollbar
// language, and no component may grow an ad-hoc z-[...] again --
// every stacking decision resolves through a named rung so the
// ladder's ORDER (what covers what) stays reviewable in one place.

const themePath = join(import.meta.dirname, 'theme.css');
const themeCss = readFileSync(themePath, 'utf8');

/** Every custom property theme.css declares, as a name → value map. */
function themeTokens(): Map<string, string> {
	const tokens = new Map<string, string>();
	for (const match of themeCss.matchAll(/(--[a-z0-9-]+):\s*([^;]+);/g)) {
		tokens.set(match[1]!, match[2]!.trim());
	}
	return tokens;
}

const tokens = themeTokens();

/** One ladder rung's numeric value, as declared. */
function rung(name: string): number {
	const value = tokens.get(`--${name}`);
	expect(value, `theme.css declares --${name}`).toBeDefined();
	const numeric = Number(value);
	expect(Number.isFinite(numeric), `--${name} is a number`).toBe(true);
	return numeric;
}

test('the app ladder climbs in exactly the documented order', () => {
	// The pin's order (content < sticky chrome < drawer < modal),
	// with our app's richer middle rungs frozen from the values the
	// surfaces already carried: page-fixed surfaces, the top bar's
	// menu, the floating toolbars, the status panel, the info
	// drawer, the scrims, the nav drawer, and the modal on top.
	const order = [
		'z-tile-chip',
		'z-tile-check',
		'z-sticky-sub',
		'z-sticky',
		'z-sticky-bar',
		'z-page',
		'z-menu',
		'z-toolbar',
		'z-panel',
		'z-toolbar-active',
		'z-aside',
		'z-scrim',
		'z-drawer',
		'z-modal'
	];
	for (const [index, name] of order.entries()) {
		if (index === 0) {
			continue;
		}
		const previous = order[index - 1]!;
		expect(
			rung(name),
			`--${name} (${rung(name)}) stacks above --${previous} (${rung(previous)})`
		).toBeGreaterThan(rung(previous));
	}
});

test('the viewer stage keeps its own ordered media stack', () => {
	// Scoped inside the viewer's own stacking context: the media
	// layers over the placeholder, the live motion over the still,
	// the status chips and press affordance above the media, and
	// the navigation arrows on top.
	const order = ['z-stage-media', 'z-stage-live', 'z-stage-chrome', 'z-stage-arrow'];
	for (const [index, name] of order.entries()) {
		if (index === 0) {
			continue;
		}
		const previous = order[index - 1]!;
		expect(rung(name)).toBeGreaterThan(rung(previous));
	}
});

test('no ad-hoc z-index survives in the components', () => {
	// Every stacking decision resolves through a ladder rung
	// (z-(--z-...)), so a bare z-[100] or static z-40 in a
	// component is a regression of the centralization pass.
	const sourceRoot = join(import.meta.dirname, '..', '..');
	const offenders: string[] = [];
	const walk = (directory: string): void => {
		for (const entry of readdirSync(directory)) {
			const path = join(directory, entry);
			const stats = statSync(path);
			if (stats.isDirectory()) {
				walk(path);
				continue;
			}
			if (!entry.endsWith('.svelte') && !entry.endsWith('.svelte.ts')) {
				continue;
			}
			const content = readFileSync(path, 'utf8');
			for (const match of content.matchAll(/\bz-(?:\[\d+\]|\d+)\b/g)) {
				offenders.push(`${path}: ${match[0]}`);
			}
		}
	};
	walk(sourceRoot);
	expect(offenders, `ad-hoc z-index utilities: ${offenders.join(', ')}`).toEqual([]);
});

test('the anti-flicker delays are motion tokens: progress 100ms, loading 400ms, placeholder fade 100ms', () => {
	expect(tokens.get('--motion-delay-progress')).toBe('100ms');
	expect(tokens.get('--motion-delay-loading')).toBe('400ms');
	expect(tokens.get('--motion-placeholder-fade')).toBe('100ms');
	// The loading arm waits strictly longer than the progress arm.
	const loading = Number(tokens.get('--motion-delay-loading')!.replace('ms', ''));
	const progress = Number(tokens.get('--motion-delay-progress')!.replace('ms', ''));
	expect(loading).toBeGreaterThan(progress);
	// The delayed-visibility animations reference the delay tokens.
	expect(tokens.get('--animate-progress-in')).toContain('var(--motion-delay-progress)');
	expect(tokens.get('--animate-loading-in')).toContain('var(--motion-delay-loading)');
	// The shared keyframes hold the invisible starting state.
	expect(themeCss).toContain('@keyframes indicator-in');
});

test('the body tracking, scrollbar language and badge text shadow are tokens', () => {
	expect(tokens.get('--tracking-body')).toBe('0.01em');
	expect(tokens.get('--scrollbar-width')).toBe('8px');
	expect(tokens.has('--scrollbar-thumb')).toBe(true);
	expect(tokens.has('--scrollbar-thumb-hover')).toBe(true);
	expect(tokens.has('--text-shadow-badge')).toBe(true);
});
