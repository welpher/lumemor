package storage

import (
	"strings"
	"testing"
	"time"
)

// sampleRenderVars is the canonical render context of the engine tests:
// an image whose EXIF header carries a capture wall clock with its
// offset, a camera, and a lens, named like a phone photo.
func sampleRenderVars() RenderVars {
	return RenderVars{
		Filename:  "IMG_1234.jpg",
		Ext:       "jpg",
		Video:     false,
		TakenAt:   time.Date(2023, 7, 14, 10, 15, 30, 123000000, time.FixedZone("", 5*3600+30*60)),
		Album:     "",
		Make:      "Acme",
		Model:     "Alpha 5",
		LensModel: "24-70mm f/2.8",
		AssetID:   "0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10",
	}
}

// renderTemplate compiles text and renders it over vars, failing the
// test on any error: the engine's happy paths must not be interleaved
// with error plumbing.
func renderTemplate(t *testing.T, text string, vars RenderVars) string {
	t.Helper()
	compiled, err := CompileTemplate(text)
	if err != nil {
		t.Fatalf("CompileTemplate(%q) error = %v", text, err)
	}
	rendered, err := compiled.Render(vars)
	if err != nil {
		t.Fatalf("Render(%q) error = %v", text, err)
	}
	return rendered
}

// TestDefaultTemplate pins the instance default: the year/month
// organization the storage setting falls back to.
func TestDefaultTemplate(t *testing.T) {
	if got := DefaultTemplate; got != "{{y}}/{{MM}}/{{filename}}" {
		t.Errorf("DefaultTemplate = %q, want {{y}}/{{MM}}/{{filename}}", got)
	}
}

// TestTemplatePresetsPinTheAuthoredList pins the sixteen built-in
// presets verbatim: the list is the feature's curated surface (the
// dropdown's source), not an implementation detail.
func TestTemplatePresetsPinTheAuthoredList(t *testing.T) {
	want := []string{
		"{{y}}/{{MM}}/{{filename}}",
		"{{y}}/{{y}}-{{MM}}/{{filename}}",
		"{{y}}/{{y}}-{{MM}}-{{dd}}/{{filename}}",
		"{{y}}/{{MMMM}}/{{filename}}",
		"{{y}}-{{MM}}-{{dd}}/{{filename}}",
		"{{y}}/{{MM}}/{{dd}}/{{filename}}",
		"{{y}}/{{filename}}",
		"{{#if album}}{{album}}/{{else}}{{y}}/{{MM}}/{{/if}}{{filename}}",
		"{{album}}/{{filename}}",
		"{{album-startDate-y}}/{{album}}/{{filename}}",
		"{{make}}/{{model}}/{{filename}}",
		"{{lensModel}}/{{filename}}",
		"{{filetype}}/{{y}}/{{MM}}/{{filename}}",
		"{{y}}/{{y}}-{{MM}}/{{assetId}}",
		"{{assetId}}",
		"{{filename}}",
	}
	if got := TemplatePresets; len(got) != len(want) {
		t.Fatalf("TemplatePresets has %d entries, want %d", len(got), len(want))
	}
	for i, preset := range want {
		if TemplatePresets[i] != preset {
			t.Errorf("TemplatePresets[%d] = %q, want %q", i, TemplatePresets[i], preset)
		}
	}
}

// TestTemplatePresetsCompileAndRender proves every built-in preset
// compiles strictly and renders over a fully populated context --
// the same guarantee the admin validation owes the dropdown it fills.
func TestTemplatePresetsCompileAndRender(t *testing.T) {
	vars := sampleRenderVars()
	vars.Album = "Trip 2024"
	vars.AlbumStart = time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC)
	vars.AlbumEnd = time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC)

	want := []string{
		"2023/07/IMG_1234",
		"2023/2023-07/IMG_1234",
		"2023/2023-07-14/IMG_1234",
		"2023/July/IMG_1234",
		"2023-07-14/IMG_1234",
		"2023/07/14/IMG_1234",
		"2023/IMG_1234",
		"Trip 2024/IMG_1234",
		"Trip 2024/IMG_1234",
		"2023/Trip 2024/IMG_1234",
		"Acme/Alpha 5/IMG_1234",
		"24-70mm f2.8/IMG_1234",
		"IMG/2023/07/IMG_1234",
		"2023/2023-07/0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10",
		"0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10",
		"IMG_1234",
	}
	for i, preset := range TemplatePresets {
		if got := renderTemplate(t, preset, vars); got != want[i] {
			t.Errorf("preset %q renders %q, want %q", preset, got, want[i])
		}
	}
}

// TestTemplateRendersVariables walks the whole variable table over the
// canonical context: every token of the file family, the date token
// set of the capture wall clock, the camera family and the ids.
func TestTemplateRendersVariables(t *testing.T) {
	vars := sampleRenderVars()
	cases := []struct{ template, want string }{
		{"{{filename}}", "IMG_1234"},
		{"{{ext}}", "jpg"},
		{"{{filetype}}", "IMG"},
		{"{{filetypefull}}", "Image"},
		{"{{y}}", "2023"},
		{"{{yy}}", "23"},
		{"{{M}}", "7"},
		{"{{MM}}", "07"},
		{"{{MMM}}", "Jul"},
		{"{{MMMM}}", "July"},
		{"{{d}}", "14"},
		{"{{dd}}", "14"},
		{"{{W}}", "28"},
		{"{{WW}}", "28"},
		{"{{H}}", "10"},
		{"{{HH}}", "10"},
		{"{{h}}", "10"},
		{"{{hh}}", "10"},
		{"{{m}}", "15"},
		{"{{mm}}", "15"},
		{"{{s}}", "30"},
		{"{{ss}}", "30"},
		{"{{SSS}}", "123"},
		{"{{make}}", "Acme"},
		{"{{model}}", "Alpha 5"},
		{"{{lensModel}}", "24-70mm f2.8"},
		{"{{assetId}}", "0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10"},
		{"{{assetIdShort}}", "6b7c8d9e0f10"},
		{"{{y}}/{{MM}}/{{filename}}", "2023/07/IMG_1234"},
	}
	for _, tt := range cases {
		if got := renderTemplate(t, tt.template, vars); got != tt.want {
			t.Errorf("template %q renders %q, want %q", tt.template, got, tt.want)
		}
	}
}

// TestTemplateDateTokensPadding pins the unpadded/padded pairs on a
// single-digit wall clock: the two-digit forms pad, the single forms
// do not, and the 12-hour clock wraps midnight up to twelve.
func TestTemplateDateTokensPadding(t *testing.T) {
	vars := sampleRenderVars()
	vars.TakenAt = time.Date(2023, 1, 5, 9, 5, 3, 5000000, time.FixedZone("", 0))
	cases := []struct{ template, want string }{
		{"{{M}}/{{MM}}", "1/01"},
		{"{{d}}/{{dd}}", "5/05"},
		{"{{H}}/{{HH}}", "9/09"},
		{"{{m}}/{{mm}}", "5/05"},
		{"{{s}}/{{ss}}", "3/03"},
		{"{{SSS}}", "005"},
	}
	for _, tt := range cases {
		if got := renderTemplate(t, tt.template, vars); got != tt.want {
			t.Errorf("template %q renders %q, want %q", tt.template, got, tt.want)
		}
	}

	// The afternoon wall clock: H stays 24-hour, h wraps to 1-12.
	vars.TakenAt = time.Date(2023, 1, 5, 15, 4, 5, 0, time.FixedZone("", 0))
	for _, tt := range []struct{ template, want string }{
		{"{{H}}:{{HH}}", "15:15"},
		{"{{h}}:{{hh}}", "3:03"},
	} {
		if got := renderTemplate(t, tt.template, vars); got != tt.want {
			t.Errorf("template %q renders %q, want %q", tt.template, got, tt.want)
		}
	}

	// Midnight wraps the 12-hour clock to twelve, not zero.
	vars.TakenAt = time.Date(2023, 1, 5, 0, 0, 0, 0, time.FixedZone("", 0))
	for _, tt := range []struct{ template, want string }{
		{"{{h}}:{{hh}}", "12:12"},
	} {
		if got := renderTemplate(t, tt.template, vars); got != tt.want {
			t.Errorf("template %q renders %q, want %q", tt.template, got, tt.want)
		}
	}
}

// TestTemplateDateTokensUseTheCaptureWallClock pins the zone rule: the
// date tokens render the taken-at instant in the zone it was captured
// in (the EXIF or device offset), not in the server's zone -- the
// folder says the day the photo was taken, not the day the server
// saw it.
func TestTemplateDateTokensUseTheCaptureWallClock(t *testing.T) {
	vars := sampleRenderVars()
	// 2023-07-14 23:30 at +08:00 is 2023-07-14 15:30 UTC: the capture
	// wall clock keeps July 14, a UTC render would say the same day;
	// 2023-07-15 00:30 at +08:00 is 2023-07-14 16:30 UTC, where only
	// the capture wall clock keeps the Friday.
	vars.TakenAt = time.Date(2023, 7, 15, 0, 30, 0, 0, time.FixedZone("", 8*3600))
	if got := renderTemplate(t, "{{y}}/{{MM}}/{{dd}}", vars); got != "2023/07/15" {
		t.Errorf("capture wall clock renders %q, want 2023/07/15", got)
	}
}

// TestTemplateAlbumDateTokens pins the album range tokens: every date
// token of the album's start and end, and the honest empty string when
// the bound is unknown (a zero time).
func TestTemplateAlbumDateTokens(t *testing.T) {
	vars := sampleRenderVars()
	vars.Album = "Trip"
	vars.AlbumStart = time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC)
	vars.AlbumEnd = time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC)

	cases := []struct{ template, want string }{
		{"{{album-startDate-y}}", "2023"},
		{"{{album-startDate-MM}}", "06"},
		{"{{album-startDate-dd}}", "30"},
		{"{{album-endDate-y}}", "2023"},
		{"{{album-endDate-MM}}", "08"},
		{"{{album-endDate-MMMM}}", "August"},
		{"{{album-endDate-d}}", "15"},
	}
	for _, tt := range cases {
		if got := renderTemplate(t, tt.template, vars); got != tt.want {
			t.Errorf("template %q renders %q, want %q", tt.template, got, tt.want)
		}
	}

	// Unknown bounds render empty, never a zero time's year.
	vars.AlbumStart = time.Time{}
	vars.AlbumEnd = time.Time{}
	for _, tt := range []struct{ template, want string }{
		{"{{album-startDate-y}}/{{filename}}", "IMG_1234"},
		{"{{album-endDate-MM}}/{{filename}}", "IMG_1234"},
	} {
		if got := renderTemplate(t, tt.template, vars); got != tt.want {
			t.Errorf("template %q renders %q over unknown bounds, want %q", tt.template, got, tt.want)
		}
	}
}

// TestTemplateConditionals pins the branch semantics: the then-branch
// on a non-empty value, the else-branch on an empty one, nesting, and
// the else clause being optional.
func TestTemplateConditionals(t *testing.T) {
	vars := sampleRenderVars()

	// No album: the else branch runs and leaves the year/month layout.
	if got := renderTemplate(t, "{{#if album}}{{album}}/{{else}}{{y}}/{{MM}}/{{/if}}{{filename}}", vars); got != "2023/07/IMG_1234" {
		t.Errorf("else branch renders %q, want 2023/07/IMG_1234", got)
	}
	// An album: the then branch runs.
	vars.Album = "Trip"
	if got := renderTemplate(t, "{{#if album}}{{album}}/{{else}}{{y}}/{{MM}}/{{/if}}{{filename}}", vars); got != "Trip/IMG_1234" {
		t.Errorf("then branch renders %q, want Trip/IMG_1234", got)
	}

	// Nesting: an album without a camera renders the album alone.
	vars.Make = ""
	if got := renderTemplate(t, "{{#if album}}{{#if make}}{{make}}-{{/if}}{{album}}/{{/if}}{{filename}}", vars); got != "Trip/IMG_1234" {
		t.Errorf("nested conditional renders %q, want Trip/IMG_1234", got)
	}
	vars.Make = "Acme"
	if got := renderTemplate(t, "{{#if album}}{{#if make}}{{make}}-{{/if}}{{album}}/{{/if}}{{filename}}", vars); got != "Acme-Trip/IMG_1234" {
		t.Errorf("nested conditional renders %q, want Acme-Trip/IMG_1234", got)
	}

	// No album at all: the whole conditional collapses.
	vars.Album = ""
	if got := renderTemplate(t, "{{#if album}}{{#if make}}{{make}}-{{/if}}{{album}}/{{/if}}{{filename}}", vars); got != "IMG_1234" {
		t.Errorf("collapsed conditional renders %q, want IMG_1234", got)
	}

	// The else clause is optional.
	if got := renderTemplate(t, "{{#if lensModel}}{{lensModel}}/{{/if}}{{filename}}", vars); got != "24-70mm f2.8/IMG_1234" {
		t.Errorf("else-less conditional renders %q, want 24-70mm f/2.8/IMG_1234", got)
	}
	vars.LensModel = ""
	if got := renderTemplate(t, "{{#if lensModel}}{{lensModel}}/{{/if}}{{filename}}", vars); got != "IMG_1234" {
		t.Errorf("else-less conditional renders %q, want IMG_1234", got)
	}
}

// TestTemplateExtNormalization pins the extension aliases: the
// normalized lowercase form the stored file name carries.
func TestTemplateExtNormalization(t *testing.T) {
	for _, tt := range []struct{ ext, want string }{
		{"jpg", "jpg"},
		{"jpeg", "jpg"},
		{"jpe", "jpg"},
		{"tif", "tiff"},
		{"tiff", "tiff"},
		{"mov", "mov"},
		{"JPG", "jpg"},
	} {
		vars := sampleRenderVars()
		vars.Ext = tt.ext
		if got := renderTemplate(t, "{{ext}}", vars); got != tt.want {
			t.Errorf("ext %q renders %q, want %q", tt.ext, got, tt.want)
		}
	}
}

// TestTemplateVideoFileTypes pins the family tokens of a video part.
func TestTemplateVideoFileTypes(t *testing.T) {
	vars := sampleRenderVars()
	vars.Video = true
	if got := renderTemplate(t, "{{filetype}}/{{filetypefull}}", vars); got != "VID/Video" {
		t.Errorf("video family renders %q, want VID/Video", got)
	}
}

// TestTemplateSanitizesUserInfluencedValues pins the value hygiene:
// separators and control runes dropped, edge dots and spaces trimmed,
// length capped on a rune boundary, and an emptied file name falling
// back to the asset id (the stored file must always have a name).
func TestTemplateSanitizesUserInfluencedValues(t *testing.T) {
	vars := sampleRenderVars()

	cases := []struct {
		name, template, want string
	}{
		{"separators dropped from the file name", "{{filename}}", "etcpasswd"},
		{"control runes dropped from the file name", "{{filename}}", "badname"},
		{"edge spaces and dots trimmed", "{{filename}}", "spaced"},
		{"unicode kept", "{{filename}}", "tüßé"},
		{"separators dropped from the album", "{{album}}", "Trip2024x"},
		{"camera separators dropped", "{{model}}", "Alpha5"},
	}
	vars.Filename = "../../etc/passwd.jpg"
	if got := renderTemplate(t, cases[0].template, vars); got != cases[0].want {
		t.Errorf("%s: renders %q, want %q", cases[0].name, got, cases[0].want)
	}
	vars.Filename = "bad\x00name.jpg"
	if got := renderTemplate(t, cases[1].template, vars); got != cases[1].want {
		t.Errorf("%s: renders %q, want %q", cases[1].name, got, cases[1].want)
	}
	vars.Filename = "  spaced .jpg"
	if got := renderTemplate(t, cases[2].template, vars); got != cases[2].want {
		t.Errorf("%s: renders %q, want %q", cases[2].name, got, cases[2].want)
	}
	vars.Filename = "tüßé.jpg"
	if got := renderTemplate(t, cases[3].template, vars); got != cases[3].want {
		t.Errorf("%s: renders %q, want %q", cases[3].name, got, cases[3].want)
	}
	vars.Album = "Trip/2024\\x"
	if got := renderTemplate(t, cases[4].template, vars); got != cases[4].want {
		t.Errorf("%s: renders %q, want %q", cases[4].name, got, cases[4].want)
	}
	vars.Model = "Alpha\n5"
	if got := renderTemplate(t, cases[5].template, vars); got != cases[5].want {
		t.Errorf("%s: renders %q, want %q", cases[5].name, got, cases[5].want)
	}

	// A file name that sanitizes to nothing falls back to the asset id.
	vars.Filename = "...jpg"
	if got := renderTemplate(t, "{{filename}}", vars); got != vars.AssetID {
		t.Errorf("emptied file name renders %q, want the asset id %q", got, vars.AssetID)
	}

	// The segment cap trims on a rune boundary: a 300-rune name keeps
	// the first 100 runes (the cap is 200 bytes, two bytes per rune).
	vars.Filename = strings.Repeat("ü", 300) + ".jpg"
	rendered := renderTemplate(t, "{{filename}}", vars)
	if got := len(rendered); got != 200 {
		t.Errorf("capped file name holds %d bytes, want 200", got)
	}
	if strings.HasSuffix(rendered, "\uFFFD") {
		t.Error("capped file name splits a rune, want a rune-boundary cut")
	}
}

// TestTemplateCompileRejectsMalformedTexts pins strict compilation:
// every malformed shape fails at CompileTemplate -- unknown variables,
// broken tag syntax, unbalanced conditionals, control runes and the
// length bound -- never at render time.
func TestTemplateCompileRejectsMalformedTexts(t *testing.T) {
	cases := []struct {
		name     string
		template string
		wantIn   string
	}{
		{"unknown variable", "{{nope}}", "nope"},
		{"unknown variable in a conditional", "{{#if nope}}x{{/if}}", "nope"},
		{"unknown album date token", "{{album-startDate-nonsense}}", "nonsense"},
		{"unknown directive", "{{#each album}}x{{/each}}", "#each"},
		{"empty tag", "{{}}", "empty"},
		{"unclosed tag", "{{y}", "unclosed"},
		{"unclosed opener", "{{y}} {{", "unclosed"},
		{"conditional without a condition", "{{#if}}x{{/if}}", "#if"},
		{"else without an if", "{{y}}{{else}}{{MM}}", "else"},
		{"close without an if", "{{y}}{{/if}}", "/if"},
		{"unterminated if", "{{#if album}}x", "unterminated"},
		{"else without a closing if", "{{#if album}}x{{else}}y", "unterminated"},
		{"control rune in the text", "{{y}}/\x01", "control"},
		{"literal trailing separator", "{{y}}/{{MM}}/", "trailing separator"},
		{"whitespace-only template", "   ", "empty"},
		{"template over the length bound", "{{y}}/" + strings.Repeat("x", 520), "exceed"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CompileTemplate(tt.template)
			if err == nil {
				t.Fatalf("CompileTemplate(%q) succeeded, want an error", tt.template)
			}
			if !strings.Contains(err.Error(), tt.wantIn) {
				t.Errorf("error %q does not mention %q", err, tt.wantIn)
			}
		})
	}
}

// TestTemplateRenderRejectsUnsafePaths pins the structural post-pass:
// traversal segments, an over-deep hierarchy, an over-long segment
// and an over-long total are render errors, and an album-only template
// over an album-less context errors rather than rendering empty.
func TestTemplateRenderRejectsUnsafePaths(t *testing.T) {
	vars := sampleRenderVars()
	cases := []struct {
		name     string
		template string
		wantIn   string
	}{
		{"parent traversal", "../{{filename}}", "traversal"},
		{"mid-path traversal", "{{y}}/{{MM}}/../../{{filename}}", "traversal"},
		{"dot segment", "./{{filename}}", "segment"},
		{"depth over the cap", "a/b/c/d/e/f/g/h/i/j/{{filename}}", "depth"},
		{"segment over the cap", strings.Repeat("x", 201) + "/{{filename}}", "exceeds"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			compiled, err := CompileTemplate(tt.template)
			if err != nil {
				t.Fatalf("CompileTemplate(%q) error = %v (the render pass owns this failure)", tt.template, err)
			}
			if _, err := compiled.Render(vars); err == nil {
				t.Fatalf("Render(%q) succeeded, want an error", tt.template)
			} else if !strings.Contains(err.Error(), tt.wantIn) {
				t.Errorf("error %q does not mention %q", err, tt.wantIn)
			}
		})
	}

	// The total-length cap: five sanitized values of 200 bytes each
	// overrun the 1000-byte render bound even though every segment is
	// individually legal.
	long := strings.Repeat("ü", 100) // 200 bytes
	vars.Make, vars.Model, vars.LensModel = long, long, long
	vars.Album, vars.Filename = long, long+".jpg"
	compiled, err := CompileTemplate("{{make}}/{{model}}/{{lensModel}}/{{album}}/{{filename}}")
	if err != nil {
		t.Fatalf("CompileTemplate error = %v", err)
	}
	if _, err := compiled.Render(vars); err == nil {
		t.Fatal("Render over a 1004-byte output succeeded, want the length error")
	}

	// An empty render is an error, not a silent dot-segment.
	vars = sampleRenderVars()
	if compiled, err := CompileTemplate("{{album}}"); err != nil {
		t.Fatalf("CompileTemplate({{album}}) error = %v", err)
	} else if _, err := compiled.Render(vars); err == nil {
		t.Fatal("Render of an empty path succeeded, want an error")
	}
}

// TestTemplateCollapsesAndTrimsSeparators pins the separator hygiene:
// doubled slashes collapse, edge slashes trim away -- the album
// conditional's trailing slash must not become a dot segment. (A
// LITERAL trailing separator is a compile error -- it would
// reinterpret the last segment as the file stem -- but a rendered
// one, like a conditional branch ending in /, still trims.)
func TestTemplateCollapsesAndTrimsSeparators(t *testing.T) {
	vars := sampleRenderVars()
	cases := []struct{ template, want string }{
		{"{{y}}//{{MM}}/{{filename}}", "2023/07/IMG_1234"},
		{"{{y}}///{{MM}}", "2023/07"},
		{"/{{y}}/{{MM}}", "2023/07"},
		{"{{#if album}}{{album}}/{{else}}{{y}}/{{MM}}/{{/if}}{{filename}}", "2023/07/IMG_1234"},
	}
	for _, tt := range cases {
		if got := renderTemplate(t, tt.template, vars); got != tt.want {
			t.Errorf("template %q renders %q, want %q", tt.template, got, tt.want)
		}
	}
}

// TestTemplateCompileAcceptsTheLengthBound proves the bound itself is
// inclusive: a 512-rune template compiles, a 513-rune one does not.
func TestTemplateCompileAcceptsTheLengthBound(t *testing.T) {
	bound := "{{y}}/" + strings.Repeat("x", 512-6)
	if _, err := CompileTemplate(bound); err != nil {
		t.Errorf("CompileTemplate of the 512-rune bound error = %v, want success", err)
	}
	over := "{{y}}/" + strings.Repeat("x", 513-6)
	if _, err := CompileTemplate(over); err == nil {
		t.Error("CompileTemplate of a 513-rune template succeeded, want the length error")
	}
}

// TestTemplateWhitespaceInsideTags tolerates padding inside the braces:
// the grammar reads {{ y }} as {{y}}.
func TestTemplateWhitespaceInsideTags(t *testing.T) {
	if got := renderTemplate(t, "{{ y }}/{{ MM }}/{{ filename }}", sampleRenderVars()); got != "2023/07/IMG_1234" {
		t.Errorf("padded tags render %q, want 2023/07/IMG_1234", got)
	}
}
