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
// produces today keep the table correct should it learn them.
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
	// the video family and downstream processing.
	"webm": {mimes: []string{"video/webm"}},
	"mkv":  {mimes: []string{"video/webm", "video/x-matroska"}},
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
// staged, and Commit moves it to a terminal state.
type receiverState int

const (
	stateStaged receiverState = iota
	stateCommitted
	stateRejected
)

func (s receiverState) String() string {
	switch s {
	case stateStaged:
		return "staged"
	case stateCommitted:
		return "committed"
	case stateRejected:
		return "rejected"
	default:
		return "unknown"
	}
}

// Receiver stages and finalizes one uploaded file. NewReceiver
// creates the staged temp file; Commit streams the payload into it
// while hashing and sniffing, then moves accepted uploads into the
// content-addressed originals tree. Every failed Commit removes the
// staged file, so only a successful Commit leaves a file on disk.
type Receiver struct {
	staged     *os.File
	stagedPath string
	source     io.Reader
	limit      int64
	state      receiverState
	checksum   string
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
// is valid after a successful Commit and empty before.
func (rec *Receiver) Checksum() string {
	if rec.state != stateCommitted {
		return ""
	}
	return rec.checksum
}

// Commit streams the payload into the staged file, validates it, and
// finalizes accepted uploads: the staged file is renamed into the
// content-addressed original path derived from the payload digest.
// relPath is that path relative to the data root,
// originals/aa/bb/<checksum>.<ext> — the portable form stored in the
// database, which must not embed the configurable data root.
//
// ext is the lowercase, dotless file extension. Validation runs in a
// fixed order: the extension must be in the upload format whitelist
// before a single byte is read; the payload is then streamed, hashed,
// and counted while the first 512 bytes are captured for sniffing; a
// payload larger than the limit aborts mid-stream with an error
// wrapping ErrTooLarge (the caller should stop consuming the source);
// and after EOF the content type sniffed from the payload's magic
// bytes must be consistent with the extension, else Commit returns
// *ErrRejected. Source read errors are wrapped. Any failure removes
// the staged file and leaves the receiver in a terminal rejected
// state; committing a terminal receiver fails.
//
// Commit requires a successful EnsureLayout: the content-addressed
// path derives from the recorded data root.
func (rec *Receiver) Commit(ext string) (string, error) {
	if rec.state != stateStaged {
		return "", fmt.Errorf("storage: Commit on a receiver that is already %s", rec.state)
	}
	// A startup-order bug should surface before a single byte is
	// streamed, not after gigabytes have crossed the wire.
	requireDataRoot("Receiver.Commit")

	// The extension whitelist is the first gate: an unknown extension
	// fails before the source is read at all.
	rule, ok := uploadFormats[ext]
	if !ok {
		return "", rec.reject(fmt.Sprintf("extension %q is not in the upload format whitelist", ext))
	}

	// Stream into the staged file while hashing, counting, and
	// capturing the head. The limit reader stops one byte past the
	// limit: exceeding the limit aborts mid-stream instead of
	// buffering the whole payload first.
	src := &streamSource{source: rec.source, digest: sha256.New()}
	copied, err := io.Copy(rec.staged, io.LimitReader(src, rec.limit+1))
	if err != nil {
		return "", rec.fail(fmt.Errorf("storage: stream upload: %w", err))
	}
	if copied > rec.limit {
		return "", rec.fail(fmt.Errorf("storage: upload of at least %d bytes exceeds the %d byte limit: %w", copied, rec.limit, ErrTooLarge))
	}

	// With the payload complete, the sniffed content type must be
	// consistent with the extension; executables and renamed files of
	// any other type fail here.
	head := src.head[:src.headLen]
	sniffed := http.DetectContentType(head)
	if !rule.consistentWith(sniffed, head) {
		return "", rec.reject(fmt.Sprintf("sniffed content type %q is not consistent with extension %q", sniffed, ext))
	}

	rec.checksum = hex.EncodeToString(src.digest.Sum(nil))
	absPath := OriginalPath(rec.checksum, ext)

	// Finalize with an atomic rename inside the data root. The
	// original shard directories are created on demand; the rename
	// itself stays within one filesystem, which the layout rename
	// probe proved at startup.
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", rec.fail(fmt.Errorf("storage: commit upload: create %s: %w", filepath.Dir(absPath), err))
	}
	if err := rec.staged.Close(); err != nil {
		return "", rec.fail(fmt.Errorf("storage: commit upload: close staged file %s: %w", rec.stagedPath, err))
	}
	if err := os.Rename(rec.stagedPath, absPath); err != nil {
		return "", rec.fail(fmt.Errorf("storage: commit upload: rename %s to %s: %w", rec.stagedPath, absPath, err))
	}
	rec.staged = nil
	rec.state = stateCommitted
	return relativeToDataRoot(absPath), nil
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
// aborted uploads leave no residue in the staging directory.
func (rec *Receiver) discardStaged() {
	if rec.staged == nil {
		return
	}
	_ = rec.staged.Close()
	rec.staged = nil
	_ = os.Remove(rec.stagedPath)
}

// relativeToDataRoot strips the data root from an absolute storage
// path, returning the portable slash-separated form stored in the
// database.
func relativeToDataRoot(absPath string) string {
	root := filepath.Clean(dataRoot) + string(filepath.Separator)
	return filepath.ToSlash(strings.TrimPrefix(absPath, root))
}
