// The storage template engine (the storage-template feature): a
// self-written ~150-line interpreter of the storage path grammar --
// variable substitution plus {{#if}}/{{else}}/{{/if}} conditionals --
// with strict compilation (an unknown variable or a malformed
// conditional fails at CompileTemplate, never at render time) and a
// structural post-pass on every render (separator collapse, traversal
// refusal, depth and length caps). No template library is pulled in:
// the grammar is exactly the two constructs above, and owning the
// parser keeps the validation semantics (compile errors, sample
// renders) inside this package.
//
// The engine renders the RELATIVE SEGMENT of an original's path under
// the per-owner root originals/<ownerUserId>/; the placement step
// appends the normalized extension and resolves collisions. Rendering
// happens exactly once, at ingest: the path is never re-derived, and
// later metadata refinement (a better taken-at, an album membership)
// never moves a file.

package storage

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// DefaultTemplate is the instance's storage template when no setting
// row exists: the year/month organization of the camera roll.
const DefaultTemplate = "{{y}}/{{MM}}/{{filename}}"

// TemplatePresets is the built-in preset list of the admin settings
// surface: sixteen self-authored templates covering the organization
// space -- seven date nestings, three album layouts, three camera
// groupings, the type grouping, the asset-id fallback and the bare
// file name.
var TemplatePresets = []string{
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

// RenderVars is the ingest-time context of one file's path render:
// everything the variable table draws from. The values arrive raw;
// the engine sanitizes the user-influenced families (file name, album,
// camera) itself so no caller can forget.
type RenderVars struct {
	// Filename is the file's own name as the upload carried it. The
	// engine strips the extension and sanitizes the stem; a stem that
	// sanitizes to nothing falls back to the asset id.
	Filename string
	// Ext is the file's extension; the engine normalizes it
	// (jpeg/jpe to jpg, tif to tiff, lower-case).
	Ext string
	// Video reports the video family: filetype renders VID and
	// filetypefull Video instead of IMG and Image.
	Video bool
	// TakenAt is the resolved capture instant (EXIF header time, else
	// the device-reported time, else the upload time). The date tokens
	// render the wall clock of the zone it carries -- the capture
	// zone, not the server's.
	TakenAt time.Time
	// Album is the album the asset belongs to AT INGEST (empty for
	// direct uploads; membership joined later never re-paths).
	Album string
	// AlbumStart and AlbumEnd are the album's date bounds; a zero
	// value renders the album date tokens as empty.
	AlbumStart time.Time
	AlbumEnd   time.Time
	// Make, Model and LensModel come from the file's EXIF header;
	// absent tags render empty.
	Make      string
	Model     string
	LensModel string
	// AssetID is the asset row's uuid string.
	AssetID string
}

// The bounds of a legal template and a legal render. The template
// bound guards the admin input; the render bounds (segment length,
// depth, total) guard the output the file system receives -- literal
// template text is checked there too, so a template whose literals
// alone would build an unsafe path fails the sample render at save
// time and errors the ingest rather than reaching os.Link.
const (
	maxTemplateRunes = 512
	maxSegmentBytes  = 200
	maxRenderDepth   = 10
	maxRenderBytes   = 1000
)

// dateTokens are the wall-clock tokens every time family renders:
// the taken-at capture time and the album date bounds.
var dateTokens = []string{
	"y", "yy", "M", "MM", "MMM", "MMMM",
	"d", "dd", "W", "WW",
	"H", "HH", "h", "hh", "m", "mm", "s", "ss", "SSS",
}

// node is one parsed construct of a compiled template.
type node interface {
	render(b *strings.Builder, values map[string]string)
}

// textNode is literal template text.
type textNode struct{ text string }

func (n textNode) render(b *strings.Builder, _ map[string]string) {
	b.WriteString(n.text)
}

// varNode is one variable substitution.
type varNode struct{ name string }

func (n varNode) render(b *strings.Builder, values map[string]string) {
	b.WriteString(values[n.name])
}

// ifNode is one conditional: the then-branch renders when the
// condition's value is non-empty, the else-branch otherwise.
type ifNode struct {
	cond string
	then []node
	els  []node
}

func (n ifNode) render(b *strings.Builder, values map[string]string) {
	if values[n.cond] != "" {
		for _, child := range n.then {
			child.render(b, values)
		}
		return
	}
	for _, child := range n.els {
		child.render(b, values)
	}
}

// Template is a compiled storage template: a parsed tree that renders
// against RenderVars. Compile once, render per upload part.
type Template struct {
	root []node
}

// CompileTemplate parses text under strict grammar rules and returns
// the compiled template. Every malformed shape is an error here --
// an unknown variable, a broken tag, an unbalanced conditional, a
// control rune, a template over the length bound -- so a template
// that compiles can only fail at render time through its values (the
// structural pass) and never through its syntax.
func CompileTemplate(text string) (*Template, error) {
	if runeLen := len([]rune(text)); runeLen > maxTemplateRunes {
		return nil, fmt.Errorf("storage template: %d runes exceed the %d rune bound", runeLen, maxTemplateRunes)
	}
	for _, r := range text {
		if unicode.IsControl(r) {
			return nil, fmt.Errorf("storage template: control rune %q is not allowed", r)
		}
	}
	// The literal text's own edges refuse early: a trailing separator
	// would silently re-cast the last directory segment as the file
	// stem (the kind re-interpretation the sample preview would show
	// only after the fact), and whitespace-only text renders nothing.
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, fmt.Errorf("storage template: empty")
	}
	if strings.HasSuffix(trimmed, "/") {
		return nil, fmt.Errorf("storage template: a trailing separator would reinterpret the last segment as the file stem")
	}
	nodes, term, _, err := parseNodes(text, false)
	if err != nil {
		return nil, err
	}
	if term != "" {
		// parseNodes only returns a terminator at nesting depth;
		// the top level reaches end-of-text instead. Kept explicit
		// so a future grammar change fails loudly here.
		return nil, fmt.Errorf("storage template: stray %s", term)
	}
	return &Template{root: nodes}, nil
}

// parseNodes parses nodes until end of text or until one of the
// terminators ({{else}} or {{/if}}) of an enclosing conditional;
// depth reports whether such a terminator is legal here. It returns
// the parsed nodes, the terminator matched ("" at end of text) and
// the text following the terminator.
func parseNodes(s string, depth bool) (nodes []node, term, rest string, err error) {
	for {
		open := strings.Index(s, "{{")
		if open < 0 {
			if s != "" {
				nodes = append(nodes, textNode{text: s})
			}
			return nodes, "", "", nil
		}
		if open > 0 {
			nodes = append(nodes, textNode{text: s[:open]})
		}
		close := strings.Index(s[open+2:], "}}")
		if close < 0 {
			return nil, "", "", fmt.Errorf("storage template: unclosed tag %q", s[open:])
		}
		inner := strings.TrimSpace(s[open+2 : open+2+close])
		after := s[open+2+close+2:]
		switch {
		case inner == "":
			return nil, "", "", fmt.Errorf("storage template: empty tag in %q", s)
		case strings.HasPrefix(inner, "#"):
			directive := strings.Fields(inner)
			if len(directive) != 2 || directive[0] != "#if" {
				return nil, "", "", fmt.Errorf("storage template: unknown directive %q", inner)
			}
			cond := directive[1]
			if !knownVariable(cond) {
				return nil, "", "", fmt.Errorf("storage template: unknown variable %q", cond)
			}
			then, matched, remainder, err := parseNodes(after, true)
			if err != nil {
				return nil, "", "", err
			}
			branch := ifNode{cond: cond, then: then}
			switch matched {
			case "{{/if}}":
			case "{{else}}":
				els, matched2, remainder2, err := parseNodes(remainder, true)
				if err != nil {
					return nil, "", "", err
				}
				if matched2 != "{{/if}}" {
					return nil, "", "", fmt.Errorf("storage template: unterminated conditional on %q", cond)
				}
				branch.els = els
				remainder = remainder2
			default:
				return nil, "", "", fmt.Errorf("storage template: unterminated conditional on %q", cond)
			}
			nodes = append(nodes, branch)
			s = remainder
		case inner == "else":
			if !depth {
				return nil, "", "", fmt.Errorf("storage template: else without an enclosing conditional")
			}
			return nodes, "{{else}}", after, nil
		case inner == "/if":
			if !depth {
				return nil, "", "", fmt.Errorf("storage template: /if without an enclosing conditional")
			}
			return nodes, "{{/if}}", after, nil
		default:
			if !knownVariable(inner) {
				return nil, "", "", fmt.Errorf("storage template: unknown variable %q", inner)
			}
			nodes = append(nodes, varNode{name: inner})
			s = after
		}
	}
}

// knownVariable reports whether name is a member of the grammar's
// variable table: the file, date, album, camera and id families.
func knownVariable(name string) bool {
	switch name {
	case "filename", "ext", "filetype", "filetypefull",
		"album", "make", "model", "lensModel", "assetId", "assetIdShort":
		return true
	}
	for _, token := range dateTokens {
		if name == token ||
			name == "album-startDate-"+token ||
			name == "album-endDate-"+token {
			return true
		}
	}
	return false
}

// Render produces the relative path segment of one upload part under
// the per-owner root: the template's output with separators collapsed
// and edge separators trimmed, structurally validated (no traversal,
// no dot segments, bounded depth and length). The extension is NOT
// part of the output -- the placement step appends it, so every
// stored file carries its true normalized extension whatever the
// template says.
func (t *Template) Render(vars RenderVars) (string, error) {
	values := templateValues(vars)
	var b strings.Builder
	for _, child := range t.root {
		child.render(&b, values)
	}
	return cleanRendered(b.String())
}

// templateValues resolves the variable table over one render context:
// the sanitized user-influenced families, the normalized extension,
// the wall-clock date tokens of the capture time and the album bounds.
func templateValues(vars RenderVars) map[string]string {
	values := make(map[string]string, 64)
	values["filename"] = sanitizeSegment(stemOf(vars.Filename))
	if values["filename"] == "" {
		// A name that sanitizes to nothing (dots and separators all
		// the way down) falls back to the asset id: the stored file
		// always has a name.
		values["filename"] = vars.AssetID
	}
	values["ext"] = NormalizeExt(vars.Ext)
	if vars.Video {
		values["filetype"] = "VID"
		values["filetypefull"] = "Video"
	} else {
		values["filetype"] = "IMG"
		values["filetypefull"] = "Image"
	}
	for _, token := range dateTokens {
		values[token] = dateTokenValue(vars.TakenAt, token)
		values["album-startDate-"+token] = dateTokenValue(vars.AlbumStart, token)
		values["album-endDate-"+token] = dateTokenValue(vars.AlbumEnd, token)
	}
	values["album"] = sanitizeSegment(vars.Album)
	values["make"] = sanitizeSegment(vars.Make)
	values["model"] = sanitizeSegment(vars.Model)
	values["lensModel"] = sanitizeSegment(vars.LensModel)
	values["assetId"] = vars.AssetID
	assetIDShort := vars.AssetID
	if len(assetIDShort) > 12 {
		assetIDShort = assetIDShort[len(assetIDShort)-12:]
	}
	values["assetIdShort"] = assetIDShort
	return values
}

// dateTokenValue renders one wall-clock token of t. A zero time
// renders every token as empty: an unknown album bound never leaks
// year one. The clock is read in t's own zone, so a capture instant
// renders with the wall clock it was captured at.
func dateTokenValue(t time.Time, token string) string {
	if t.IsZero() {
		return ""
	}
	switch token {
	case "y":
		return strconv.Itoa(t.Year())
	case "yy":
		return zeroPad(t.Year()%100, 2)
	case "M":
		return strconv.Itoa(int(t.Month()))
	case "MM":
		return zeroPad(int(t.Month()), 2)
	case "MMM":
		return t.Format("Jan")
	case "MMMM":
		return t.Format("January")
	case "d":
		return strconv.Itoa(t.Day())
	case "dd":
		return zeroPad(t.Day(), 2)
	case "W":
		_, week := t.ISOWeek()
		return strconv.Itoa(week)
	case "WW":
		_, week := t.ISOWeek()
		return zeroPad(week, 2)
	case "H":
		return strconv.Itoa(t.Hour())
	case "HH":
		return zeroPad(t.Hour(), 2)
	case "h":
		hour := t.Hour() % 12
		if hour == 0 {
			hour = 12
		}
		return strconv.Itoa(hour)
	case "hh":
		hour := t.Hour() % 12
		if hour == 0 {
			hour = 12
		}
		return zeroPad(hour, 2)
	case "m":
		return strconv.Itoa(t.Minute())
	case "mm":
		return zeroPad(t.Minute(), 2)
	case "s":
		return strconv.Itoa(t.Second())
	case "ss":
		return zeroPad(t.Second(), 2)
	case "SSS":
		return zeroPad(t.Nanosecond()/1e6, 3)
	}
	return ""
}

// zeroPad renders n in at least width digits.
func zeroPad(n, width int) string {
	return fmt.Sprintf("%0*d", width, n)
}

// extAliases maps the extension spellings that share one storage form
// onto that form: the stored file names are normalized so equal
// formats organize identically whatever the client spelled.
var extAliases = map[string]string{
	"jpeg": "jpg",
	"jpe":  "jpg",
	"tif":  "tiff",
}

// NormalizeExt returns the canonical storage form of an extension:
// lower-case, with the aliases folded (jpeg and jpe to jpg, tif to
// tiff).
func NormalizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if alias, ok := extAliases[ext]; ok {
		return alias
	}
	return ext
}

// stemOf splits a raw file name into its extension-less stem.
func stemOf(name string) string {
	return strings.TrimSuffix(name, extOf(name))
}

// extOf returns the extension of a file name including the dot, or
// empty when it carries none.
func extOf(name string) string {
	dot := strings.LastIndexByte(name, '.')
	if dot < 0 {
		return ""
	}
	return name[dot:]
}

// sanitizeSegment renders one user-influenced value as a single safe
// path segment: path separators and control runes dropped, edge dots
// and spaces trimmed, the length capped on a rune boundary. Interior
// runes (spaces, dots, unicode) are kept -- a file name like
// "IMG.1234.jpg" stays legible.
func sanitizeSegment(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	out := strings.Trim(strings.TrimSpace(b.String()), ".")
	if len(out) > maxSegmentBytes {
		out = truncateRunes(out, maxSegmentBytes)
	}
	return out
}

// truncateRunes cuts s to at most max bytes without splitting a rune:
// a cut that lands inside a multi-byte sequence backs off to the
// sequence's start.
func truncateRunes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}

// cleanRendered applies the separator hygiene and the structural
// validation to one rendered path: double separators collapse, edge
// separators trim, the result must be non-empty with no traversal or
// dot segments, bounded depth and bounded length.
func cleanRendered(raw string) (string, error) {
	for strings.Contains(raw, "//") {
		raw = strings.ReplaceAll(raw, "//", "/")
	}
	raw = strings.Trim(raw, "/")
	if raw == "" {
		return "", fmt.Errorf("storage template: rendered an empty path")
	}
	if len(raw) > maxRenderBytes {
		return "", fmt.Errorf("storage template: rendered path of %d bytes exceeds the %d byte bound", len(raw), maxRenderBytes)
	}
	segments := strings.Split(raw, "/")
	if len(segments) > maxRenderDepth {
		return "", fmt.Errorf("storage template: rendered path of %d segments exceeds the depth bound %d", len(segments), maxRenderDepth)
	}
	for _, segment := range segments {
		if segment == "." || segment == ".." {
			return "", fmt.Errorf("storage template: rendered path carries the traversal segment %q", segment)
		}
		if segment == "" {
			return "", fmt.Errorf("storage template: rendered path carries an empty segment")
		}
		if len(segment) > maxSegmentBytes {
			return "", fmt.Errorf("storage template: segment of %d bytes exceeds the %d byte bound", len(segment), maxSegmentBytes)
		}
	}
	return raw, nil
}
