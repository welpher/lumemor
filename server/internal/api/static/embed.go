// Package static embeds the HTML pages served outside the generated
// /api/v1 surface and outside the single-page app: the first-run
// wizard, a self-contained single file (inline CSS and JS, no external
// assets) that needs no build step and predates the SPA. The landing
// page this package used to carry was replaced by the SvelteKit app
// embedded in server/internal/web; the wizard stays because first-run
// setup is a pre-SPA process.
package static

import "embed"

// FS holds the embedded wizard page, keyed by file name ("setup.html").
//
//go:embed *.html
var FS embed.FS
