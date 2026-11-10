# Web code style (SvelteKit + Svelte 5 + Tailwind v4)

> The machine layer is live (the 2026-11-10 web-tooling block):
> **prettier owns formatting** -- run `npx prettier --write .` in
> web/ (or `pnpm run format`); **eslint owns the lint layer** -- run
> `npx eslint .` in web/. `make web-check` fails on both (the
> format check and the lint run before svelte-check and vitest).
> The configs (web/.prettierrc.js, web/eslint.config.js) pin the
> tree's established format and record every rule trim's reason.
> This file keeps only what the machines cannot enforce.

## Naming and files

- Components `PascalCase.svelte` (`Button.svelte`, `Modal.svelte`);
  shared reactive modules `*.svelte.ts` (`session.svelte.ts`) — the suffix is
  the marker that the file uses runes at module scope.
- Variables `camelCase`, types `PascalCase`; API shapes import from the
  generated client — never re-declare a contract shape by hand.
- Tests are colocated (`anchored.test.ts` beside `anchored.ts`); a test's
  name states the pin it proves.
- `web/src/lib/<domain>/` owns domain components and logic
  (`lib/timeline/`, `lib/albums/`); route files compose — logic sinks to lib.

## Svelte 5 idioms

- Runes only: `$state`, `$derived` (prefer plain functions unless
  reactivity is needed), `$props()` destructured with defaults,
  `bindable` only when the parent genuinely writes back.
- Events: the `onselect`-style props; a component never reaches its
  grandparent.
- Comments: the same design-record voice as the Go side — why, not what;
  dense where the decision is non-obvious.

## CSS: the two-class discipline

- **Semantic classes are BEM** (`drawer__item`, `batchbar__error`) and answer
  *what is this* — structural identity, the test/query surface.
- **Tailwind utilities** answer *how is it laid out* — inline, per element.
  The mix is intentional: BEM for identity, utilities for geometry.
- **Tokens are the single source**: semantic values live in
  `lib/theme/theme.css`'s `@theme` block; new visual values enter as tokens
  first, then get consumed. No raw hex in components.

## i18n

- Key grammar: `<surface>_<section>_<item>` snake_case
  (`nav_open_menu`, `timeline_month_count`).
- Every user-facing string goes through `$_()`, in BOTH locales (en +
  zh-Hans), parity enforced by `scripts/check_i18n.py`; non-error UI keys
  join the script's `NON_ERROR_KEYS` allowlist.

## Icons

The icon-first rule: every action affordance
carries a lucide glyph — toolbars/inline/overflow are icon-only with
`aria-label`, primary flows are icon+text, form submits may stay text.
Decorative icons carry `aria-hidden`.
