// Package static embeds the HTML pages served outside the generated
// /api/v1 surface: the first-run wizard and the landing page. Each page
// is a self-contained single file (inline CSS and JS, no external
// assets), so serving them needs no build step; the SvelteKit web app
// replaces them in M2.
package static

import "embed"

// FS holds the embedded pages, keyed by file name ("setup.html",
// "index.html").
//
//go:embed *.html
var FS embed.FS
