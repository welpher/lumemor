// ESLint owns the web package's lint layer (the machine layer's
// other half; golangci-lint is the Go-side twin). Start point: the
// typescript-eslint recommended set plus the svelte recommended
// set. Every rule the existing tree legitimately violates BY
// CONVENTION is trimmed below with its recorded reason -- the goal
// is enforcing the discipline this tree already has, not
// re-litigating it. Formatting rules are off (the flat/prettier
// entry disables the plugin's stylistic rules; prettier owns
// formatting -- see .prettierrc.js).
import eslint from '@eslint/js';
import svelte from 'eslint-plugin-svelte';
import tseslint from 'typescript-eslint';

export default tseslint.config(
	{
		ignores: ['node_modules/', 'build/', '.svelte-kit/', 'test-results/', 'e2e/.auth/', 'playwright-report/']
	},
	eslint.configs.recommended,
	tseslint.configs.recommended,
	...svelte.configs['flat/recommended'],
	...svelte.configs['flat/prettier'],
	{
		// The TS inside svelte files and the runes modules
		// (*.svelte.ts) rides the typescript-eslint parser
		// (svelte-eslint-parser delegates script blocks and module
		// files to it).
		files: ['**/*.svelte', '**/*.svelte.ts'],
		languageOptions: {
			parserOptions: {
				parser: tseslint.parser
			}
		}
	},
	{
		// TRIMS -- each rule below is off because the tree violates
		// it by established convention, not by drift.
		rules: {
			// The tree's navigation predates the resolve() idiom:
			// every goto()/href navigates with plain route strings.
			// Adopting resolve() is a per-surface migration decision
			// (it touches every link), not something the lint layer
			// may impose wholesale.
			'svelte/no-navigation-without-resolve': 'off',
			// The tree's .svelte.ts modules keep plain Map/Set/Date
			// for deliberately non-reactive bookkeeping (the upload
			// queue's XHR index, walk-local accumulators); the
			// Svelte* reactive wrappers are adopted per surface when
			// reactivity is genuinely needed, never by default.
			'svelte/prefer-svelte-reactivity': 'off'
		}
	},
	{
		// Svelte <script lang="ts"> blocks are TypeScript: the type
		// layer (svelte-check, already a gate) owns
		// undefined-identifier checking there, exactly as
		// typescript-eslint's own recommended set turns this rule
		// off for .ts files.
		files: ['**/*.svelte'],
		rules: {
			'no-undef': 'off'
		}
	},
	{
		// vitest's documented way to type the `test` block of a
		// vite.config.ts is the triple-slash reference at the top of
		// that file.
		files: ['vite.config.ts'],
		rules: {
			'@typescript-eslint/triple-slash-reference': 'off'
		}
	},
	{
		// NARROWED, not off: the tree's convention marks intentionally
		// unused parameters and variables with the underscore prefix
		// (see the fetcher stubs in the api tests, the no-op
		// listeners in OfflineStrip.test.ts, login/+page.ts's
		// helper-export rule, and places/+page.svelte's
		// dependency-registration read -- a read whose VALUE is
		// unused but whose occurrence Svelte tracks). Genuinely dead
		// locals stay errors.
		rules: {
			'@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }]
		}
	}
);
