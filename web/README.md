# web

The lumemor web client: a pure SvelteKit single-page app (no SSR, no
prerendering) built with `@sveltejs/adapter-static`, whose output is
embedded into the Go server binary.

- Svelte 5 runes; virtualized photo grid; perceptual-hash placeholders
  and three-tier srcset thumbnails; light/dark themes
- API clients are generated from the root `openapi/` spec via
  openapi-typescript — never hand-written
- i18n consumes the flat JSON files in the root `i18n/` directory

## Build pipeline

```
make web-install   # pnpm install (lockfile committed)
make web-check     # svelte-check + vitest (the CI gate)
make web-build     # adapter-static → web/build, staged into
                   # server/internal/web/dist for the Go embed
```

`web/build/` and the staged `server/internal/web/dist/` copy are never
committed. A Go build without a prior `make web-build` fails at the
embed directive in `server/internal/web/embed.go` ("no matching files
found") — that loud failure pins the build order; CI runs the web
build before the Go build.

## Route groups (directory contract from the web IA design)

```
src/routes/
  +layout.ts        SPA mode: ssr=false, prerender=false
  +layout.svelte    app shell skeleton (top bar, static placeholders)
  +page.svelte      welcome placeholder at /
  (content)/        content shell routes
    timeline/ albums/ a/[id]/ trash/ favorites/
  (auth)/           minimal shell routes
    login/
  (tool)/           tool shell routes
    admin/
```

Current pages are placeholders; later tasks fill them in and add the
per-group layouts, remaining routes and behaviors.

## Serving contract

The Go server serves the built shell for `/` and every unknown
non-API GET path (client-side routing fallback); `_app/immutable/`
assets carry `Cache-Control: public, max-age=31536000, immutable`.
The first-run wizard at `/setup` stays a separate embedded static
page: setup is a pre-SPA process.
