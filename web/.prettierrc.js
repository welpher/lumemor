// Prettier owns the web package's formatting (the machine layer's
// format half; the Go side's gofmt is the symmetric twin). The
// options below are pinned to what the existing tree already does
// -- the pre-tooling convention, made machine-enforceable -- NOT to
// prettier's defaults: every non-default exists because the default
// would restyle the established format.
/** @type {import('prettier').Config} */
export default {
	// The house indent: one tab per level, everywhere (the Go side's
	// convention, carried into web on day one).
	useTabs: true,
	tabWidth: 4,
	// Statements end with semicolons; strings are single-quoted in
	// TS (and in Svelte template expressions, which follow this
	// file) while markup attributes stay double-quoted -- that split
	// is the prettier-plugin-svelte default and the existing tree's
	// habit.
	semi: true,
	singleQuote: true,
	// Non-default: prettier's "all" would sprinkle trailing commas
	// into every multi-line literal (its v3 default); this tree's
	// convention is bare closers (see any multi-line array or object
	// literal), so "none" keeps the format pass a pure correction.
	trailingComma: 'none',
	// Wide on purpose: the existing templates carry long utility
	// class strings and ~100-col statements as single lines; 120 is
	// the width where those mostly pass and only genuinely over-long
	// lines wrap.
	printWidth: 120,
	plugins: ['prettier-plugin-svelte']
};
