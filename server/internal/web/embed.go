// Package web embeds the built lumemor single-page app and serves it:
// the SvelteKit adapter-static output -- the index.html app shell plus
// the content-hashed _app/ assets -- produced by the pnpm build under
// web/build and staged into dist/ by `make web-build`.
//
// The staged dist/ tree is never committed. A Go build that runs
// without a prior `make web-build` fails here at compile time: the
// embed directive below reports "pattern all:dist: no matching files
// found", which is the intended loud guard pinning the build order
// (CI runs the web build before the Go build; local runs invoke
// `make web-build` first).
package web

import "embed"

// dist holds the staged adapter-static output. The `all:` prefix is
// load-bearing: embed patterns without it exclude files and
// directories whose names begin with "." or "_", which would silently
// drop the entire _app/ asset tree.
//
//go:embed all:dist
var dist embed.FS
