package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ErrTooLarge reports an upload whose payload exceeded the configured
// byte limit while streaming. The caller should stop consuming the
// source and discard the request.
var ErrTooLarge = errors.New("storage: upload exceeds the configured byte limit")

// ErrRejected reports an upload that failed content validation: its
// extension is outside the upload whitelist, or the payload's magic
// bytes do not match that extension. Rejected uploads leave no file
// on disk.
type ErrRejected struct {
	// Reason describes the validation failure in a form suitable for
	// server logs and client-facing error messages.
	Reason string
}

func (e *ErrRejected) Error() string {
	return "storage: upload rejected: " + e.Reason
}

// formatRule captures how the payload of an upload carrying one
// whitelisted extension is validated after streaming: the content
// types the standard sniffer may legitimately report for it, plus
// structural fallbacks for the container types the standard sniffer
// cannot classify (it answers application/octet-stream for those).
type formatRule struct {
	// mimes are the content types http.DetectContentType returns for
	// well-formed files of this format.
	mimes []string
	// ftypBrands lists the ISO base media major and compatible brands
	// that identify this format.
	ftypBrands map[string]bool
	// prefixes lists fixed magic byte prefixes that identify this
	// format.
	prefixes []string
}

// uploadFormats is the upload format whitelist (media pipeline spec,
// format set v1): the images JPEG, HEIC, WebP, PNG, GIF, TIFF, and
// AVIF, and the videos MP4, MOV, WebM, and MKV. Keys are lowercase,
// dotless extensions; callers normalize file names before Commit.
//
// The standard content sniffer identifies JPEG, PNG, GIF, WebP, MP4,
// and the EBML family; it cannot identify TIFF, HEIC, AVIF, or MOV
// and answers application/octet-stream for them, so structural
// fallbacks decide for those. The MIME entries that the sniffer never
// produces today keep the table correct should it learn them. The
// first entry of every mimes list is the format's canonical content
// type -- the mime_type LookupUploadFormat hands to the database.
var uploadFormats = map[string]*formatRule{
	// Images.
	"jpg":  {mimes: []string{"image/jpeg"}},
	"jpeg": {mimes: []string{"image/jpeg"}},
	"png":  {mimes: []string{"image/png"}},
	"gif":  {mimes: []string{"image/gif"}},
	"webp": {mimes: []string{"image/webp"}},
	"tif":  {mimes: []string{"image/tiff"}, prefixes: []string{"II*\x00", "MM\x00*"}},
	"tiff": {mimes: []string{"image/tiff"}, prefixes: []string{"II*\x00", "MM\x00*"}},
	"heic": {mimes: []string{"image/heic", "image/heif"}, ftypBrands: heifBrands},
	"avif": {mimes: []string{"image/avif"}, ftypBrands: avifBrands},
	// Videos.
	"mp4": {mimes: []string{"video/mp4"}, ftypBrands: mp4Brands},
	"mov": {mimes: []string{"video/quicktime"}, ftypBrands: quicktimeBrands},
	// Both WebM and Matroska containers start with the EBML magic,
	// which the standard sniffer reports as video/webm; they share
	// the video family and downstream processing. The canonical type
	// (the first entry, which LookupUploadFormat reports) is the
	// registry name of the container itself.
	"webm": {mimes: []string{"video/webm"}},
	"mkv":  {mimes: []string{"video/x-matroska", "video/webm"}},
}

var (
	// heifBrands covers the HEIF brands found in HEIC photo files.
	heifBrands = brandSet("heic", "heix", "heim", "heis", "hevc", "hevx", "mif1", "msf1")
	avifBrands = brandSet("avif", "avis")
	// mp4Brands covers the common MPEG-4 file brands; files whose
	// brands include an mp4x variant are already classified as
	// video/mp4 by the standard sniffer.
	mp4Brands       = brandSet("mp41", "mp42", "isom", "iso2", "iso4", "iso6", "avc1", "avc3", "dash", "mp4 ", "M4V ")
	quicktimeBrands = brandSet("qt  ")
)

// brandSet turns brand strings into a lookup set.
func brandSet(brands ...string) map[string]bool {
	set := make(map[string]bool, len(brands))
	for _, brand := range brands {
		set[brand] = true
	}
	return set
}

// UploadFormat describes one whitelisted upload format: the canonical
// MIME type the database records as the asset's mime_type, and the
// media family it belongs to (Video true for the video family,
// everything else is an image).
type UploadFormat struct {
	// MIMEType is the canonical content type of the format, the first
	// entry of the format's content-type set.
	MIMEType string
	// Video reports whether the format belongs to the video family.
	Video bool
}

// LookupUploadFormat returns the format-table entry for ext, which
// must be canonical (lowercase, dotless) as the callers normalize it
// before Commit. ok is false for extensions outside the upload
// whitelist.
func LookupUploadFormat(ext string) (UploadFormat, bool) {
	rule, ok := uploadFormats[ext]
	if !ok {
		return UploadFormat{}, false
	}
	mime := rule.mimes[0]
	return UploadFormat{
		MIMEType: mime,
		Video:    strings.HasPrefix(mime, "video/"),
	}, true
}

// consistentWith reports whether sniffed, the content type the
// standard sniffer derived from the payload head, is consistent with
// this format.
func (r *formatRule) consistentWith(sniffed string, head []byte) bool {
	for _, mime := range r.mimes {
		if mime == sniffed {
			return true
		}
	}
	// Only an unclassifiable payload falls through to the structural
	// fallbacks: a payload the sniffer identified as something else
	// never matches, whatever its extension claims.
	if sniffed != "application/octet-stream" {
		return false
	}
	for _, prefix := range r.prefixes {
		if bytes.HasPrefix(head, []byte(prefix)) {
			return true
		}
	}
	return hasFTYPBrand(head, r.ftypBrands)
}

// hasFTYPBrand reports whether the payload head carries an ISO base
// media ftyp box whose major or compatible brands include one of
// brands. An ftyp box is size(4) "ftyp"(4) major brand(4) minor
// version(4) compatible brands(4 each); only brands inside the
// captured head window are considered.
func hasFTYPBrand(head []byte, brands map[string]bool) bool {
	if len(head) < 12 || string(head[4:8]) != "ftyp" {
		return false
	}
	limit := int64(len(head))
	if boxSize := int64(binary.BigEndian.Uint32(head[:4])); boxSize < limit {
		limit = boxSize
	}
	for off := int64(8); off+4 <= limit; off += 4 {
		if off == 12 {
			continue // minor version field, not a brand
		}
		if brands[string(head[off:off+4])] {
			return true
		}
	}
	return false
}

// receiverState tracks the upload state machine: a fresh receiver is
// staged, Seal validates the payload and seals the staged file, Place
// links it into the originals tree, and every failure path rejects.
type receiverState int

const (
	stateStaged receiverState = iota
	stateSealed
	statePlaced
	stateRejected
)

func (s receiverState) String() string {
	switch s {
	case stateStaged:
		return "staged"
	case stateSealed:
		return "sealed"
	case statePlaced:
		return "placed"
	case stateRejected:
		return "rejected"
	default:
		return "unknown"
	}
}

// maxCollisionAttempts bounds the collision suffix search: an owner
// directory holding a thousand files of one name is a pathological
// instance, not a placement to keep grinding through.
const maxCollisionAttempts = 1000

// Receiver stages and finalizes one uploaded file. NewReceiver
// creates the staged temp file; Seal streams the payload into it
// while hashing and sniffing; Header reads the EXIF capture context
// of the sealed file; Place links it into the per-owner originals
// tree under a template-rendered path, appending collision suffixes
// when the target exists. Every failed step removes what the receiver
// had put on disk, so only a placed receiver leaves a file behind --
// and Discard removes even that for the upload's failure paths.
type Receiver struct {
	staged     *os.File
	stagedPath string
	source     io.Reader
	limit      int64
	state      receiverState
	checksum   string
	size       int64
	// placedRel is the stored relative path once Place linked the
	// file; the failure-path cleanup removes it.
	placedRel string
}

// NewReceiver prepares to receive one upload: the staged temp file is
// created under tmpDir immediately, so staging failures such as a
// missing or unwritable directory surface before the source is read
// at all. limit is the maximum payload size in bytes; r is the
// payload source, streamed into the staged file without buffering it
// in memory.
func NewReceiver(tmpDir string, limit int64, r io.Reader) (*Receiver, error) {
	staged, err := os.CreateTemp(tmpDir, "upload-*")
	if err != nil {
		return nil, fmt.Errorf("storage: create staged upload in %s: %w", tmpDir, err)
	}
	return &Receiver{
		staged:     staged,
		stagedPath: staged.Name(),
		source:     r,
		limit:      limit,
		state:      stateStaged,
	}, nil
}

// Checksum returns the sha256 hex digest of the uploaded payload. It
// is valid after a successful Seal and empty before.
func (rec *Receiver) Checksum() string {
	if rec.state != stateSealed && rec.state != statePlaced {
		return ""
	}
	return rec.checksum
}

// Size returns the byte count of the uploaded payload, as streamed
// during Seal. It is valid after a successful Seal and zero before;
// the caller records it as the asset's file_size.
func (rec *Receiver) Size() int64 {
	if rec.state != stateSealed && rec.state != statePlaced {
		return 0
	}
	return rec.size
}

// Header reads the media header -- the EXIF capture wall clock and
// camera identity -- of the sealed staged file. It is valid between
// Seal and Place (Place consumes the staged file); outside that
// window it answers the zero header and the caller's fallback chain
// governs.
func (rec *Receiver) Header() MediaHeader {
	if rec.state != stateSealed {
		return MediaHeader{}
	}
	header, _ := ReadMediaHeader(rec.stagedPath)
	return header
}

// Seal streams the payload into the staged file and validates it,
// leaving the receiver ready for Place: the staged file is closed and
// the checksum and size are recorded. No placement happens here -- the
// destination is a template render the caller computes from the
// checksum, the header and the asset identity, none of which exist
// before the payload has streamed.
//
// ext is the lowercase, dotless file extension. Validation runs in a
// fixed order: the extension must be in the upload format whitelist
// before a single byte is read; the payload is then streamed, hashed,
// and counted while the first 512 bytes are captured for sniffing; a
// payload larger than the limit aborts mid-stream with an error
// wrapping ErrTooLarge (the caller should stop consuming the source);
// and after EOF the content type sniffed from the payload's magic
// bytes must be consistent with the extension, else Seal returns
// *ErrRejected. Source read errors are wrapped. Any failure removes
// the staged file and leaves the receiver in a terminal rejected
// state; sealing a terminal receiver fails.
//
// Seal requires a successful EnsureLayout: the layout probe that
// proved staging and the originals tree share a filesystem is what
// makes Place's link step safe.
func (rec *Receiver) Seal(ext string) error {
	if rec.state != stateStaged {
		return fmt.Errorf("storage: Seal on a receiver that is already %s", rec.state)
	}
	// A startup-order bug should surface before a single byte is
	// streamed, not after gigabytes have crossed the wire.
	requireDataRoot("Receiver.Seal")

	// The extension whitelist is the first gate: an unknown extension
	// fails before the source is read at all.
	rule, ok := uploadFormats[ext]
	if !ok {
		return rec.reject(fmt.Sprintf("extension %q is not in the upload format whitelist", ext))
	}

	// Stream into the staged file while hashing, counting, and
	// capturing the head. The limit reader stops one byte past the
	// limit: exceeding the limit aborts mid-stream instead of
	// buffering the whole payload first.
	src := &streamSource{source: rec.source, digest: sha256.New()}
	copied, err := io.Copy(rec.staged, io.LimitReader(src, rec.limit+1))
	if err != nil {
		return rec.fail(fmt.Errorf("storage: stream upload: %w", err))
	}
	if copied > rec.limit {
		return rec.fail(fmt.Errorf("storage: upload of at least %d bytes exceeds the %d byte limit: %w", copied, rec.limit, ErrTooLarge))
	}

	// With the payload complete, the sniffed content type must be
	// consistent with the extension; executables and renamed files of
	// any other type fail here.
	head := src.head[:src.headLen]
	sniffed := http.DetectContentType(head)
	if !rule.consistentWith(sniffed, head) {
		return rec.reject(fmt.Sprintf("sniffed content type %q is not consistent with extension %q", sniffed, ext))
	}

	rec.checksum = hex.EncodeToString(src.digest.Sum(nil))
	rec.size = copied
	// The stream is complete: close the staged file so Header can
	// read it and Place can link it. The path stays until Place (or
	// the failure paths) remove it.
	if err := rec.staged.Close(); err != nil {
		return rec.fail(fmt.Errorf("storage: seal upload: close staged file %s: %w", rec.stagedPath, err))
	}
	rec.staged = nil
	rec.state = stateSealed
	return nil
}

// Place finalizes the sealed upload into the per-owner originals
// tree: originals/<ownerID>/<rendered>.<ext>, where rendered is the
// template engine's output (slash-separated directories plus the file
// stem, structurally validated) and ext is normalized. It returns the
// stored relative path -- the portable slash-separated form recorded
// in file_path.
//
// The placement is OVERWRITE-PROOF: a hard link from the staged file
// to the target, which fails with EEXIST when anything already sits
// at the target, never replaces it. A collision bumps the stem with
// ~1, ~2, ... before the extension and retries; the (owner+checksum)
// dedup of the upload pipeline runs before any placement, so true
// duplicates never even reach this loop. os.Rename is deliberately
// NOT used: on Unix it silently replaces the target, and a template
// path names a file the owner can recognize -- replacing it would
// eat a previously committed original.
//
// The link and the staged-file removal leave a crash window (the file
// exists at both paths): boot's staging sweep cleans the staged side,
// and the placed side is a complete file whose row either commits or
// leaves an honest orphan.
func (rec *Receiver) Place(ownerID, rendered, ext string) (string, error) {
	if rec.state != stateSealed {
		return "", fmt.Errorf("storage: Place on a receiver that is %s", rec.state)
	}
	requireDataRoot("Receiver.Place")
	ext = NormalizeExt(ext)
	if !isExtension(ext) {
		return "", rec.fail(fmt.Errorf("storage: place upload: extension %q must be one to five alphanumeric characters without a dot", ext))
	}
	rendered = strings.Trim(rendered, "/")
	if rendered == "" {
		return "", rec.fail(fmt.Errorf("storage: place upload: rendered path is empty"))
	}
	segments := strings.Split(rendered, "/")
	stem := segments[len(segments)-1]
	dir := filepath.Join(append([]string{dataRoot, "originals", ownerID}, segments[:len(segments)-1]...)...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", rec.fail(fmt.Errorf("storage: place upload: create %s: %w", dir, err))
	}

	candidate := stem + "." + ext
	for attempt := 0; ; attempt++ {
		target := filepath.Join(dir, candidate)
		err := os.Link(rec.stagedPath, target)
		if err == nil {
			// The link landed; the staged copy must go. Failing to
			// remove it means the receiver cannot guarantee a clean
			// state, so the link unwinds too and the upload fails.
			if err := os.Remove(rec.stagedPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				_ = os.Remove(target)
				return "", rec.fail(fmt.Errorf("storage: place upload: remove staged file %s: %w", rec.stagedPath, err))
			}
			rec.placedRel = relativeToDataRoot(target)
			rec.state = statePlaced
			return rec.placedRel, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", rec.fail(fmt.Errorf("storage: place upload: link %s: %w", target, err))
		}
		if attempt >= maxCollisionAttempts {
			return "", rec.fail(fmt.Errorf("storage: place upload: %s exhausted %d collision suffixes", target, maxCollisionAttempts))
		}
		candidate = fmt.Sprintf("%s~%d.%s", stem, attempt+1, ext)
	}
}

// Discard removes the receiver's on-disk artifact on the upload's
// failure paths: the staged file before placement, the placed
// original after. The placed path is safe to remove because the link
// only succeeds on a target that did not exist, so no row other than
// this request's (which never committed, or the caller would not be
// discarding) can reference it. Already-gone is success -- a
// concurrent discard may have removed it first.
func (rec *Receiver) Discard() {
	switch rec.state {
	case stateSealed:
		if err := os.Remove(rec.stagedPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			// A staged file that cannot be removed leaves crash-debris
			// for the boot sweep; nothing else references it.
			_ = err
		}
		rec.state = stateRejected
	case statePlaced:
		if abs, err := ResolveStoredPath(dataRoot, rec.placedRel); err == nil {
			_ = removeFileIfPresent(abs)
		}
		rec.state = stateRejected
	}
}

// streamSource wraps the upload source so that every byte flowing to
// the staged file is hashed, counted, and — for the first 512 of
// them — captured for content sniffing. Nothing is buffered beyond
// the fixed 512-byte head.
type streamSource struct {
	source  io.Reader
	digest  hash.Hash
	head    [512]byte
	headLen int
}

func (s *streamSource) Read(p []byte) (int, error) {
	n, err := s.source.Read(p)
	if n > 0 {
		// sha256 writes never fail.
		_, _ = s.digest.Write(p[:n])
		if s.headLen < len(s.head) {
			s.headLen += copy(s.head[s.headLen:], p[:n])
		}
	}
	return n, err
}

// reject moves the receiver to the rejected state, discards the
// staged file, and returns the rejection carrying reason.
func (rec *Receiver) reject(reason string) error {
	rec.discardStaged()
	rec.state = stateRejected
	return &ErrRejected{Reason: reason}
}

// fail moves the receiver to the rejected state, discards the staged
// file, and returns err unchanged (the caller wrapped it already).
func (rec *Receiver) fail(err error) error {
	rec.discardStaged()
	rec.state = stateRejected
	return err
}

// discardStaged closes and removes the staged file so failed and
// aborted uploads leave no residue in the staging directory. The path
// removal is unconditional: after Seal the handle is closed and nil
// while the file itself still sits in staging.
func (rec *Receiver) discardStaged() {
	if rec.staged != nil {
		_ = rec.staged.Close()
		rec.staged = nil
	}
	if rec.stagedPath != "" {
		_ = os.Remove(rec.stagedPath)
	}
}

// relativeToDataRoot strips the data root from an absolute storage
// path, returning the portable slash-separated form stored in the
// database.
func relativeToDataRoot(absPath string) string {
	root := filepath.Clean(dataRoot) + string(filepath.Separator)
	return filepath.ToSlash(strings.TrimPrefix(absPath, root))
}
