# web

SvelteKit (adapter-static, pure SPA, no SSR) → output embedded into the Go binary via embed.FS.

- Svelte 5 runes; virtualized photo grid; perceptual-hash placeholders + three-tier srcset thumbnails; light/dark themes
- API clients are generated from the root `openapi/` spec via openapi-typescript — never hand-written
- i18n consumes the flat JSON files in the root `i18n/` directory
