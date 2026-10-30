package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errClientAborted stands in for the read error a request body
// returns when the client disappears mid-upload.
var errClientAborted = errors.New("client aborted the upload")

// countingReader counts the bytes that flowed through it, proving
// whether the receiver stopped consuming the source early.
type countingReader struct {
	r io.Reader
	n int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.n += int64(n)
	return n, err
}

// failingReader fails every read with a fixed error.
type failingReader struct{ err error }

func (fr failingReader) Read([]byte) (int, error) { return 0, fr.err }

// setupLayout creates a fresh data layout and returns its data root
// and upload staging directory.
func setupLayout(t *testing.T) (dataDir, tmpDir string) {
	t.Helper()
	dataDir = t.TempDir()
	if err := EnsureLayout(dataDir, t.TempDir()); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}
	return dataDir, filepath.Join(dataDir, "tmp")
}

// requireEmptyDir fails the test when dir holds any entry.
func requireEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	if len(entries) != 0 {
		t.Errorf("%s holds %d entries after the upload settled, want 0", dir, len(entries))
	}
}

// countFiles counts the regular files under root, including those in
// subdirectories.
func countFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return count
}

// sha256Hex returns the hex digest of data, computed independently of
// the receiver under test.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// readFixture loads a committed binary fixture from testdata.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

// pngFixture encodes a real 1x1 PNG through the standard encoder.
func pngFixture(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 0x7f, G: 0x33, B: 0xcc, A: 0xff})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode PNG fixture: %v", err)
	}
	return buf.Bytes()
}

// jpegFixture encodes a real 1x1 JPEG through the standard encoder.
func jpegFixture(t *testing.T) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode JPEG fixture: %v", err)
	}
	return buf.Bytes()
}

// largeJPEGFixture encodes a deterministic noise JPEG large enough to
// cross io.Copy's internal buffer boundaries, so a streamed digest is
// accumulated over many reads.
func largeJPEGFixture(t *testing.T) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 384, 384))
	rng := rand.New(rand.NewSource(1))
	for i := range img.Pix {
		img.Pix[i] = uint8(rng.Intn(256))
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode JPEG fixture: %v", err)
	}
	return buf.Bytes()
}

// ftypBox assembles a minimal ISO base media ftyp header: box size,
// "ftyp", the major brand, minor version zero, and the compatible
// brands.
func ftypBox(major string, compatible ...string) []byte {
	size := 4 + 4 + 4 + 4 + 4*len(compatible)
	box := make([]byte, 4, size)
	binary.BigEndian.PutUint32(box, uint32(size))
	box = append(box, "ftyp"...)
	box = append(box, major...)
	box = append(box, 0, 0, 0, 0)
	for _, brand := range compatible {
		box = append(box, brand...)
	}
	return box
}

// webpFixture assembles a small RIFF container with one VP8L chunk.
func webpFixture() []byte {
	payload := []byte{0x2f, 0x08, 0xd7, 0x01, 0x00, 0x00}
	chunk := make([]byte, 0, 8+len(payload))
	chunk = append(chunk, 'V', 'P', '8', 'L')
	var chunkSize [4]byte
	binary.LittleEndian.PutUint32(chunkSize[:], uint32(len(payload)))
	chunk = append(chunk, chunkSize[:]...)
	chunk = append(chunk, payload...)
	body := append([]byte("WEBP"), chunk...)
	out := make([]byte, 0, 8+len(body))
	out = append(out, 'R', 'I', 'F', 'F')
	var riffSize [4]byte
	binary.LittleEndian.PutUint32(riffSize[:], uint32(len(body)))
	out = append(out, riffSize[:]...)
	return append(out, body...)
}

// ebmlFixture assembles an EBML header whose DocType is docType; WebM
// and Matroska containers share the same EBML magic.
func ebmlFixture(docType string) []byte {
	header := []byte{0x1a, 0x45, 0xdf, 0xa3, 0x84, 0x42, 0x82, byte(0x80 | len(docType))}
	return append(header, docType...)
}

// gifFixture is the smallest structurally valid GIF: a GIF89a header,
// a logical screen descriptor, and a trailer.
var gifFixture = []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")

// tiffFixture is a little-endian TIFF header followed by an empty IFD.
var tiffFixture = []byte{
	'I', 'I', 0x2a, 0x00, // magic
	0x08, 0x00, 0x00, 0x00, // offset of the first IFD
	0x00, 0x00, // entry count
	0x00, 0x00, 0x00, 0x00, // offset of the next IFD
}

// textPayload is plain text pretending to be an image.
var textPayload = []byte("This is definitely a text file, not a JPEG.\n")

// peExecutable starts with the MZ magic of a Windows executable.
var peExecutable = []byte{
	0x4d, 0x5a, 0x90, 0x00, 0x03, 0x00, 0x00, 0x00,
	0x04, 0x00, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00,
}

// elfExecutable starts with the ELF magic of a Unix executable.
var elfExecutable = append([]byte{0x7f, 'E', 'L', 'F'}, make([]byte, 20)...)

// rejectionOf returns the *ErrRejected carried by err, or nil.
func rejectionOf(err error) *ErrRejected {
	var rej *ErrRejected
	errors.As(err, &rej)
	return rej
}

func TestCommitAcceptsEveryWhitelistedFormat(t *testing.T) {
	_, tmpDir := setupLayout(t)
	jpegBytes := jpegFixture(t)
	pngBytes := pngFixture(t)
	mp4Bytes := readFixture(t, "minimal.mp4")
	heicBytes := readFixture(t, "minimal.heic")

	formats := []struct {
		name string
		ext  string
		data []byte
	}{
		{"jpg", "jpg", jpegBytes},
		{"jpeg", "jpeg", jpegBytes},
		{"png", "png", pngBytes},
		{"gif", "gif", gifFixture},
		{"webp", "webp", webpFixture()},
		{"tif", "tif", tiffFixture},
		{"tiff", "tiff", tiffFixture},
		{"heic", "heic", heicBytes},
		{"avif", "avif", ftypBox("avif", "avif", "mif1", "miaf")},
		{"mp4 with an unclassified brand set", "mp4", mp4Bytes},
		{"mp4 with an mp4x brand", "mp4", ftypBox("isom", "isom", "iso2", "mp41")},
		{"mov", "mov", ftypBox("qt  ", "qt  ")},
		{"webm", "webm", ebmlFixture("webm")},
		{"mkv", "mkv", ebmlFixture("matroska")},
	}

	for _, tt := range formats {
		t.Run(tt.name, func(t *testing.T) {
			rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(tt.data))
			if err != nil {
				t.Fatalf("NewReceiver() error = %v", err)
			}
			relPath, err := rec.Commit(tt.ext)
			if err != nil {
				t.Fatalf("Commit(%q) error = %v", tt.ext, err)
			}

			checksum := sha256Hex(tt.data)
			if got := rec.Checksum(); got != checksum {
				t.Errorf("Checksum() = %q, want %q", got, checksum)
			}

			// The file lands at the content-addressed original path,
			// byte-identical to the upload.
			absPath := OriginalPath(checksum, tt.ext)
			committed, err := os.ReadFile(absPath)
			if err != nil {
				t.Fatalf("read committed file %s: %v", absPath, err)
			}
			if !bytes.Equal(committed, tt.data) {
				t.Errorf("committed file differs from the upload (%d vs %d bytes)", len(committed), len(tt.data))
			}

			// The returned path is the portable form stored in the
			// database: relative to the data root, slash separated.
			wantRel := "originals/" + checksum[:2] + "/" + checksum[2:4] + "/" + checksum + "." + tt.ext
			if relPath != wantRel {
				t.Errorf("Commit(%q) relPath = %q, want %q", tt.ext, relPath, wantRel)
			}

			// A finished upload leaves no staged file behind.
			requireEmptyDir(t, tmpDir)
		})
	}
}

func TestChecksumMatchesIndependentDigestOnMultiChunkStream(t *testing.T) {
	_, tmpDir := setupLayout(t)
	data := largeJPEGFixture(t)

	rec, err := NewReceiver(tmpDir, 1<<30, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if _, err := rec.Commit("jpg"); err != nil {
		t.Fatalf("Commit(\"jpg\") error = %v", err)
	}

	if got := rec.Checksum(); got != sha256Hex(data) {
		t.Errorf("Checksum() = %q, want the sha256 of the payload", got)
	}
	committed, err := os.ReadFile(OriginalPath(rec.Checksum(), "jpg"))
	if err != nil {
		t.Fatalf("read committed file: %v", err)
	}
	if !bytes.Equal(committed, data) {
		t.Errorf("committed file differs from the payload (%d vs %d bytes)", len(committed), len(data))
	}
	requireEmptyDir(t, tmpDir)
}

func TestChecksumEmptyBeforeCommit(t *testing.T) {
	_, tmpDir := setupLayout(t)
	rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(pngFixture(t)))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if got := rec.Checksum(); got != "" {
		t.Errorf("Checksum() before Commit = %q, want empty", got)
	}
}

func TestCommitRejectsDisguisedPayloads(t *testing.T) {
	cases := []struct {
		name string
		ext  string
		data []byte
	}{
		{"text file renamed to jpg", "jpg", textPayload},
		{"windows executable renamed to jpg", "jpg", peExecutable},
		{"unix executable renamed to mp4", "mp4", elfExecutable},
		{"png bytes renamed to mp4", "mp4", pngFixture(t)},
		{"heic bytes renamed to jpg", "jpg", readFixture(t, "minimal.heic")},
		{"empty payload", "jpg", nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dataDir, tmpDir := setupLayout(t)
			rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(tt.data))
			if err != nil {
				t.Fatalf("NewReceiver() error = %v", err)
			}

			relPath, err := rec.Commit(tt.ext)
			rej := rejectionOf(err)
			if rej == nil {
				t.Fatalf("Commit(%q) error = %v, want *ErrRejected", tt.ext, err)
			}
			if relPath != "" {
				t.Errorf("Commit(%q) relPath = %q, want empty on rejection", tt.ext, relPath)
			}
			if !strings.Contains(rej.Reason, tt.ext) {
				t.Errorf("rejection reason %q does not name the extension %q", rej.Reason, tt.ext)
			}
			if got := rec.Checksum(); got != "" {
				t.Errorf("Checksum() after rejection = %q, want empty", got)
			}

			// A rejected upload leaves no trace on disk, in staging
			// and in the originals tree alike.
			requireEmptyDir(t, tmpDir)
			if n := countFiles(t, filepath.Join(dataDir, "originals")); n != 0 {
				t.Errorf("originals holds %d files after a rejection, want 0", n)
			}

			// The rejection is terminal.
			if _, err := rec.Commit(tt.ext); err == nil {
				t.Errorf("Commit(%q) after a rejection succeeded, want failure", tt.ext)
			}
		})
	}
}

func TestCommitRejectsUnknownExtensionBeforeReading(t *testing.T) {
	_, tmpDir := setupLayout(t)
	for _, ext := range []string{"exe", "txt", "svg", "zip", "cr2", "JPG"} {
		t.Run(ext, func(t *testing.T) {
			cr := &countingReader{r: bytes.NewReader(textPayload)}
			rec, err := NewReceiver(tmpDir, 1<<20, cr)
			if err != nil {
				t.Fatalf("NewReceiver() error = %v", err)
			}

			_, err = rec.Commit(ext)
			rej := rejectionOf(err)
			if rej == nil {
				t.Fatalf("Commit(%q) error = %v, want *ErrRejected", ext, err)
			}
			if !strings.Contains(rej.Reason, "whitelist") {
				t.Errorf("rejection reason %q does not mention the whitelist", rej.Reason)
			}
			if cr.n != 0 {
				t.Errorf("source was read for %d bytes before the whitelist rejection, want 0", cr.n)
			}
			requireEmptyDir(t, tmpDir)
		})
	}
}

func TestCommitAbortsWhenPayloadExceedsLimit(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	const limit = 1024
	// Four times the limit; the payload itself never needs to be valid
	// media because the size check aborts before content validation.
	payload := make([]byte, 4096)
	rng := rand.New(rand.NewSource(2))
	for i := range payload {
		payload[i] = byte(rng.Intn(256))
	}
	cr := &countingReader{r: bytes.NewReader(payload)}

	rec, err := NewReceiver(tmpDir, limit, cr)
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	_, err = rec.Commit("jpg")
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Commit() error = %v, want ErrTooLarge", err)
	}
	if rejectionOf(err) != nil {
		t.Errorf("over-limit error also carries *ErrRejected, want a pure size failure")
	}

	// The abort happens mid-stream: at most one byte past the limit
	// was consumed, far from the whole payload.
	if cr.n > limit+1 {
		t.Errorf("source was read for %d bytes, want at most %d", cr.n, limit+1)
	}
	if cr.n >= int64(len(payload)) {
		t.Errorf("source was fully consumed (%d of %d bytes) before the size rejection", cr.n, len(payload))
	}

	// The partial file is gone, from staging and originals alike.
	requireEmptyDir(t, tmpDir)
	if n := countFiles(t, filepath.Join(dataDir, "originals")); n != 0 {
		t.Errorf("originals holds %d files after an aborted upload, want 0", n)
	}
	if got := rec.Checksum(); got != "" {
		t.Errorf("Checksum() after an aborted upload = %q, want empty", got)
	}

	// The receiver is terminal after the abort.
	if _, err := rec.Commit("jpg"); err == nil {
		t.Error("Commit() after ErrTooLarge succeeded, want failure")
	}
}

func TestCommitRemovesStagedFileWhenSourceFails(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	// A stream that delivers a valid JPEG header and then fails, like
	// a client aborting its request body mid-transfer.
	src := io.MultiReader(bytes.NewReader(jpegFixture(t)), failingReader{err: errClientAborted})

	rec, err := NewReceiver(tmpDir, 1<<20, src)
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	_, err = rec.Commit("jpg")
	if !errors.Is(err, errClientAborted) {
		t.Fatalf("Commit() error = %v, want the wrapped reader error", err)
	}
	if rejectionOf(err) != nil {
		t.Errorf("read failure masquerades as *ErrRejected, want a wrapped transport error")
	}

	// The half-written staged file left no residue.
	requireEmptyDir(t, tmpDir)
	if n := countFiles(t, filepath.Join(dataDir, "originals")); n != 0 {
		t.Errorf("originals holds %d files after a client abort, want 0", n)
	}
	if got := rec.Checksum(); got != "" {
		t.Errorf("Checksum() after a client abort = %q, want empty", got)
	}
}

func TestCommitAfterRejectionFails(t *testing.T) {
	_, tmpDir := setupLayout(t)
	rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(textPayload))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if _, err := rec.Commit("jpg"); rejectionOf(err) == nil {
		t.Fatalf("Commit(\"jpg\") error = %v, want *ErrRejected", err)
	}

	_, err = rec.Commit("jpg")
	if err == nil {
		t.Fatal("Commit() after a rejection succeeded, want failure")
	}
	if rejectionOf(err) != nil {
		t.Errorf("double Commit error = %v, want a state error, not another rejection", err)
	}
	requireEmptyDir(t, tmpDir)
}

func TestDoubleCommitFails(t *testing.T) {
	_, tmpDir := setupLayout(t)
	data := pngFixture(t)
	rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if _, err := rec.Commit("png"); err != nil {
		t.Fatalf("Commit(\"png\") error = %v", err)
	}
	checksum := rec.Checksum()

	_, err = rec.Commit("png")
	if err == nil {
		t.Fatal("second Commit() succeeded, want failure")
	}

	// The first commit stands: checksum and file are unchanged.
	if rec.Checksum() != checksum {
		t.Errorf("Checksum() after a double Commit = %q, want %q", rec.Checksum(), checksum)
	}
	committed, err := os.ReadFile(OriginalPath(checksum, "png"))
	if err != nil {
		t.Fatalf("read committed file: %v", err)
	}
	if !bytes.Equal(committed, data) {
		t.Error("committed file changed after a double Commit")
	}
	requireEmptyDir(t, tmpDir)
}

func TestCommitFailsWhenShardDirectoryCannotBeCreated(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	data := pngFixture(t)
	checksum := sha256Hex(data)

	// A regular file squatting on the shard directory path makes the
	// finalization fail after the payload already streamed.
	shard := filepath.Join(dataDir, "originals", checksum[:2])
	if err := os.WriteFile(shard, []byte("blocked"), 0o644); err != nil {
		t.Fatalf("block shard directory: %v", err)
	}

	rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	_, err = rec.Commit("png")
	if err == nil {
		t.Fatal("Commit() succeeded although the shard directory cannot exist, want failure")
	}
	if !strings.Contains(err.Error(), "originals") {
		t.Errorf("error %q does not name the target directory", err.Error())
	}
	requireEmptyDir(t, tmpDir)
	if got := rec.Checksum(); got != "" {
		t.Errorf("Checksum() after a failed commit = %q, want empty", got)
	}
}

func TestNewReceiverFailsWhenStagingDirectoryIsUnusable(t *testing.T) {
	_, err := NewReceiver(filepath.Join(t.TempDir(), "missing"), 1<<20, strings.NewReader("x"))
	if err == nil {
		t.Fatal("NewReceiver() error = nil, want an error for an unusable staging directory")
	}
}

func TestCommitPanicsBeforeEnsureLayout(t *testing.T) {
	resetDataRoot(t)
	cr := &countingReader{r: bytes.NewReader(pngFixture(t))}
	rec, err := NewReceiver(t.TempDir(), 1<<20, cr)
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}

	defer func() {
		panicValue := recover()
		if panicValue == nil {
			t.Fatal("Commit() did not panic, want panic before EnsureLayout has succeeded")
		}
		msg, ok := panicValue.(string)
		if !ok || !strings.Contains(msg, "EnsureLayout") {
			t.Errorf("panic message = %v, want it to mention EnsureLayout", panicValue)
		}
		if cr.n != 0 {
			t.Errorf("source was read for %d bytes before the panic, want 0", cr.n)
		}
	}()
	// The call must panic before it can return anything worth
	// checking.
	_, _ = rec.Commit("png")
}
