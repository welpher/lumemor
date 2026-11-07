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
	"strconv"
	"strings"
	"sync"
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

// testOwnerID is the owner of every placement test's uploads: a fixed
// uuid whose string form lands in the per-owner originals root.
const testOwnerID = "0197a1b2-c3d4-4e5f-8a9b-0c1d2e3f4a5b"

// sealReceiver streams payload through a fresh receiver and seals it,
// failing the test on any error.
func sealReceiver(t *testing.T, tmpDir string, payload []byte, ext string) *Receiver {
	t.Helper()
	rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if err := rec.Seal(ext); err != nil {
		t.Fatalf("Seal(%q) error = %v", ext, err)
	}
	return rec
}

// placeSealed seals payload and places it at
// originals/<ownerID>/<rendered>.<ext>, failing the test on any error;
// it answers the receiver (for the discard tests) and the stored
// relative path.
func placeSealed(t *testing.T, tmpDir string, payload []byte, ext, ownerID, rendered string) (*Receiver, string) {
	t.Helper()
	rec := sealReceiver(t, tmpDir, payload, ext)
	relPath, err := rec.Place(ownerID, rendered, ext)
	if err != nil {
		t.Fatalf("Place(%q, %q) error = %v", rendered, ext, err)
	}
	return rec, relPath
}

func TestSealPlaceAcceptsEveryWhitelistedFormat(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
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

	for i, tt := range formats {
		t.Run(tt.name, func(t *testing.T) {
			// Each format places under its own rendered name so the
			// subtests sharing one layout do not collide.
			rendered := "2026/01/photo-" + strconv.Itoa(i)
			rec, relPath := placeSealed(t, tmpDir, tt.data, tt.ext, testOwnerID, rendered)

			checksum := sha256Hex(tt.data)
			if got := rec.Checksum(); got != checksum {
				t.Errorf("Checksum() = %q, want %q", got, checksum)
			}

			// The file lands at the per-owner template path, with the
			// normalized extension, byte-identical to the upload.
			wantExt := NormalizeExt(tt.ext)
			wantRel := "originals/" + testOwnerID + "/2026/01/photo-" + strconv.Itoa(i) + "." + wantExt
			if relPath != wantRel {
				t.Errorf("Place() relPath = %q, want %q", relPath, wantRel)
			}
			committed, err := os.ReadFile(filepath.Join(dataDir, filepath.FromSlash(relPath)))
			if err != nil {
				t.Fatalf("read placed file %s: %v", relPath, err)
			}
			if !bytes.Equal(committed, tt.data) {
				t.Errorf("placed file differs from the upload (%d vs %d bytes)", len(committed), len(tt.data))
			}

			// A finished upload leaves no staged file behind.
			requireEmptyDir(t, tmpDir)
		})
	}
}

func TestChecksumMatchesIndependentDigestOnMultiChunkStream(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	data := largeJPEGFixture(t)

	rec, relPath := placeSealed(t, tmpDir, data, "jpg", testOwnerID, "big/payload")

	if got := rec.Checksum(); got != sha256Hex(data) {
		t.Errorf("Checksum() = %q, want the sha256 of the payload", got)
	}
	committed, err := os.ReadFile(filepath.Join(dataDir, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("read placed file: %v", err)
	}
	if !bytes.Equal(committed, data) {
		t.Errorf("placed file differs from the payload (%d vs %d bytes)", len(committed), len(data))
	}
	requireEmptyDir(t, tmpDir)
}

func TestChecksumEmptyBeforeSeal(t *testing.T) {
	_, tmpDir := setupLayout(t)
	rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(pngFixture(t)))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if got := rec.Checksum(); got != "" {
		t.Errorf("Checksum() before Seal = %q, want empty", got)
	}
	if got := rec.Size(); got != 0 {
		t.Errorf("Size() before Seal = %d, want 0", got)
	}
}

func TestSealRejectsDisguisedPayloads(t *testing.T) {
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

			err = rec.Seal(tt.ext)
			rej := rejectionOf(err)
			if rej == nil {
				t.Fatalf("Seal(%q) error = %v, want *ErrRejected", tt.ext, err)
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
			if err := rec.Seal(tt.ext); err == nil {
				t.Errorf("Seal(%q) after a rejection succeeded, want failure", tt.ext)
			}
		})
	}
}

func TestSealRejectsUnknownExtensionBeforeReading(t *testing.T) {
	_, tmpDir := setupLayout(t)
	for _, ext := range []string{"exe", "txt", "svg", "zip", "cr2", "JPG"} {
		t.Run(ext, func(t *testing.T) {
			cr := &countingReader{r: bytes.NewReader(textPayload)}
			rec, err := NewReceiver(tmpDir, 1<<20, cr)
			if err != nil {
				t.Fatalf("NewReceiver() error = %v", err)
			}

			err = rec.Seal(ext)
			rej := rejectionOf(err)
			if rej == nil {
				t.Fatalf("Seal(%q) error = %v, want *ErrRejected", ext, err)
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

func TestSealAbortsWhenPayloadExceedsLimit(t *testing.T) {
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
	err = rec.Seal("jpg")
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Seal() error = %v, want ErrTooLarge", err)
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
	if err := rec.Seal("jpg"); err == nil {
		t.Error("Seal() after ErrTooLarge succeeded, want failure")
	}
}

func TestSealRemovesStagedFileWhenSourceFails(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	// A stream that delivers a valid JPEG header and then fails, like
	// a client aborting its request body mid-transfer.
	src := io.MultiReader(bytes.NewReader(jpegFixture(t)), failingReader{err: errClientAborted})

	rec, err := NewReceiver(tmpDir, 1<<20, src)
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	err = rec.Seal("jpg")
	if !errors.Is(err, errClientAborted) {
		t.Fatalf("Seal() error = %v, want the wrapped reader error", err)
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

func TestSealAfterRejectionFails(t *testing.T) {
	_, tmpDir := setupLayout(t)
	rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(textPayload))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if err := rec.Seal("jpg"); rejectionOf(err) == nil {
		t.Fatalf("Seal(\"jpg\") error = %v, want *ErrRejected", err)
	}

	err = rec.Seal("jpg")
	if err == nil {
		t.Fatal("Seal() after a rejection succeeded, want failure")
	}
	if rejectionOf(err) != nil {
		t.Errorf("double Seal error = %v, want a state error, not another rejection", err)
	}
	requireEmptyDir(t, tmpDir)
}

func TestDoubleSealFails(t *testing.T) {
	_, tmpDir := setupLayout(t)
	data := pngFixture(t)
	rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if err := rec.Seal("png"); err != nil {
		t.Fatalf("Seal(\"png\") error = %v", err)
	}
	checksum := rec.Checksum()

	err = rec.Seal("png")
	if err == nil {
		t.Fatal("second Seal() succeeded, want failure")
	}

	// The first seal stands: the checksum is unchanged and the staged
	// file survives for Place.
	if rec.Checksum() != checksum {
		t.Errorf("Checksum() after a double Seal = %q, want %q", rec.Checksum(), checksum)
	}
	if _, err := os.Stat(rec.stagedPath); err != nil {
		t.Fatalf("staged file did not survive a double Seal: %v", err)
	}
}

func TestDoublePlaceFails(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	data := pngFixture(t)
	rec := sealReceiver(t, tmpDir, data, "png")

	relPath, err := rec.Place(testOwnerID, "2026/01/twin", "png")
	if err != nil {
		t.Fatalf("Place() error = %v", err)
	}
	if _, err := rec.Place(testOwnerID, "2026/01/twin", "png"); err == nil {
		t.Fatal("second Place() succeeded, want failure")
	}
	// The first placement stands.
	if _, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(relPath))); err != nil {
		t.Fatalf("placed file did not survive a double Place: %v", err)
	}
}

func TestPlaceFailsWhenOwnerDirectoryCannotBeCreated(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	data := pngFixture(t)

	// A regular file squatting on the owner directory path makes the
	// placement fail after the payload already streamed.
	blocked := filepath.Join(dataDir, "originals", testOwnerID)
	if err := os.WriteFile(blocked, []byte("blocked"), 0o644); err != nil {
		t.Fatalf("block owner directory: %v", err)
	}

	rec := sealReceiver(t, tmpDir, data, "png")
	_, err := rec.Place(testOwnerID, "2026/01/photo", "png")
	if err == nil {
		t.Fatal("Place() succeeded although the owner directory cannot exist, want failure")
	}
	requireEmptyDir(t, tmpDir)
	if got := rec.Checksum(); got != "" {
		t.Errorf("Checksum() after a failed Place = %q, want empty", got)
	}
}

func TestNewReceiverFailsWhenStagingDirectoryIsUnusable(t *testing.T) {
	_, err := NewReceiver(filepath.Join(t.TempDir(), "missing"), 1<<20, strings.NewReader("x"))
	if err == nil {
		t.Fatal("NewReceiver() error = nil, want an error for an unusable staging directory")
	}
}

func TestSealPanicsBeforeEnsureLayout(t *testing.T) {
	resetDataRoot(t)
	cr := &countingReader{r: bytes.NewReader(pngFixture(t))}
	rec, err := NewReceiver(t.TempDir(), 1<<20, cr)
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}

	defer func() {
		panicValue := recover()
		if panicValue == nil {
			t.Fatal("Seal() did not panic, want panic before EnsureLayout has succeeded")
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
	_ = rec.Seal("png")
}

// TestPlaceAppendsCollisionSuffixesSequentially pins the collision
// contract: three uploads of DIFFERENT bytes whose template renders to
// one target land at the base name, ~1 and ~2 -- no error, no
// overwrite -- and every stored file holds its own bytes.
func TestPlaceAppendsCollisionSuffixesSequentially(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)

	colors := []color.RGBA{
		{R: 0x10, G: 0x10, B: 0x10, A: 0xff},
		{R: 0x20, G: 0x20, B: 0x20, A: 0xff},
		{R: 0x30, G: 0x30, B: 0x30, A: 0xff},
	}
	want := []string{
		"originals/" + testOwnerID + "/2026/01/IMG_0001.jpg",
		"originals/" + testOwnerID + "/2026/01/IMG_0001~1.jpg",
		"originals/" + testOwnerID + "/2026/01/IMG_0001~2.jpg",
	}
	for i, pixel := range colors {
		payload := jpegColorUpload(t, pixel)
		_, relPath := placeSealed(t, tmpDir, payload, "jpg", testOwnerID, "2026/01/IMG_0001")
		if relPath != want[i] {
			t.Errorf("placement %d = %q, want %q", i, relPath, want[i])
		}
		stored, err := os.ReadFile(filepath.Join(dataDir, filepath.FromSlash(relPath)))
		if err != nil {
			t.Fatalf("read placed file %s: %v", relPath, err)
		}
		if !bytes.Equal(stored, payload) {
			t.Errorf("placed file %s holds the wrong bytes", relPath)
		}
	}
	if got := countFiles(t, filepath.Join(dataDir, "originals")); got != len(want) {
		t.Errorf("originals holds %d files after the collisions, want %d", got, len(want))
	}
}

// jpegColorUpload encodes a 1x1 JPEG of one pixel color, a payload
// distinct per color.
func jpegColorUpload(t *testing.T, pixel color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, pixel)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode JPEG fixture: %v", err)
	}
	return buf.Bytes()
}

// TestPlaceNeverOverwritesAnExistingFile is the safety pin of the
// placement: a file that already sits at the rendered target -- from
// an earlier upload or a crash window -- keeps its bytes untouched,
// and the new upload lands at the ~1 suffix instead. os.Rename would
// have replaced the existing file silently; the link discipline
// cannot.
func TestPlaceNeverOverwritesAnExistingFile(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)

	// The squatter: an earlier upload's file at the exact target.
	target := filepath.Join(dataDir, "originals", testOwnerID, "2026", "01", "IMG_9.jpg")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("create target directory: %v", err)
	}
	squatter := []byte("the original bytes of the earlier upload")
	if err := os.WriteFile(target, squatter, 0o644); err != nil {
		t.Fatalf("write squatter file: %v", err)
	}

	payload := jpegFixture(t)
	_, relPath := placeSealed(t, tmpDir, payload, "jpg", testOwnerID, "2026/01/IMG_9")
	if want := "originals/" + testOwnerID + "/2026/01/IMG_9~1.jpg"; relPath != want {
		t.Errorf("collision placement = %q, want %q", relPath, want)
	}

	// The squatter is untouched: content and count of files.
	stored, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read the squatter file: %v", err)
	}
	if !bytes.Equal(stored, squatter) {
		t.Error("the pre-existing file was overwritten, want its bytes untouched")
	}
	newBytes, err := os.ReadFile(filepath.Join(dataDir, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("read the placed file: %v", err)
	}
	if !bytes.Equal(newBytes, payload) {
		t.Error("the placed file does not hold the upload's bytes")
	}
}

// TestPlaceConcurrentSameTarget pins the collision handling under
// concurrency: several placements racing for one rendered target all
// succeed with DISTINCT paths -- the link's atomicity hands every
// racer either the base name or its own suffix -- and every stored
// file holds its own bytes.
func TestPlaceConcurrentSameTarget(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	const racers = 8

	colors := make([]color.RGBA, racers)
	for i := range colors {
		colors[i] = color.RGBA{R: uint8(0x10 + i*0x10), G: 0x22, B: 0x33, A: 0xff}
	}

	type outcome struct {
		relPath string
		err     error
	}
	outcomes := make([]outcome, racers)
	sealed := make([]*Receiver, racers)
	for i := range sealed {
		sealed[i] = sealReceiver(t, tmpDir, jpegColorUpload(t, colors[i]), "jpg")
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range sealed {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			rel, err := sealed[i].Place(testOwnerID, "2026/01/race", "jpg")
			outcomes[i] = outcome{relPath: rel, err: err}
		}(i)
	}
	close(start)
	wg.Wait()

	seen := make(map[string]bool)
	for i, o := range outcomes {
		if o.err != nil {
			t.Fatalf("concurrent placement %d failed: %v", i, o.err)
		}
		if seen[o.relPath] {
			t.Errorf("placements %d and an earlier racer share the path %q", i, o.relPath)
		}
		seen[o.relPath] = true
		stored, err := os.ReadFile(filepath.Join(dataDir, filepath.FromSlash(o.relPath)))
		if err != nil {
			t.Fatalf("read placed file %s: %v", o.relPath, err)
		}
		if !bytes.Equal(stored, jpegColorUpload(t, colors[i])) {
			t.Errorf("placed file %s holds the wrong racer's bytes", o.relPath)
		}
	}
	if got := countFiles(t, filepath.Join(dataDir, "originals")); got != racers {
		t.Errorf("originals holds %d files after the race, want %d", got, racers)
	}
}

// TestDiscardRemovesTheStagedSealedFile pins the failure-path hygiene
// before placement: an upload whose request failed between Seal and
// Place leaves no staged file behind.
func TestDiscardRemovesTheStagedSealedFile(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	rec := sealReceiver(t, tmpDir, jpegFixture(t), "jpg")

	rec.Discard()

	requireEmptyDir(t, tmpDir)
	if got := countFiles(t, filepath.Join(dataDir, "originals")); got != 0 {
		t.Errorf("originals holds %d files after a discard, want 0", got)
	}
	// Discard is idempotent: a second call is a no-op.
	rec.Discard()
	requireEmptyDir(t, tmpDir)
}

// TestDiscardRemovesThePlacedFile pins the failure-path hygiene after
// placement: an upload whose rows never committed (the unique-index
// conflict path) removes the file it placed, because the link proved
// the path did not exist before it -- nothing else can reference it.
func TestDiscardRemovesThePlacedFile(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	rec, relPath := placeSealed(t, tmpDir, jpegFixture(t), "jpg", testOwnerID, "2026/01/lost")

	rec.Discard()

	if _, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(relPath))); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("placed file %s survived the discard (stat error = %v), want it gone", relPath, err)
	}
	requireEmptyDir(t, tmpDir)
	// A discard never removes anything else: the squatter test above
	// covers the neighbor; here the tree is empty.
	if got := countFiles(t, filepath.Join(dataDir, "originals")); got != 0 {
		t.Errorf("originals holds %d files after the discard, want 0", got)
	}
}

// TestReceiverHeaderReadsTheSealedFile pins the header plumbing: a
// sealed JPEG with EXIF answers its capture context, an unsealed
// receiver answers the zero header.
func TestReceiverHeaderReadsTheSealedFile(t *testing.T) {
	_, tmpDir := setupLayout(t)
	tiff := buildExifTIFF(binary.LittleEndian,
		[]exifEntry{{tag: tagMake, value: "Lumemor"}},
		[]exifEntry{
			{tag: tagDateTimeOriginal, value: "2023:07:14 10:15:30"},
			{tag: tagOffsetTime, value: "+05:30"},
		})
	payload := buildExifJPEG(t, tiff)

	rec, err := NewReceiver(tmpDir, 1<<20, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if got := rec.Header(); got.DateTimeOriginal != "" {
		t.Errorf("Header() before Seal = %+v, want the zero header", got)
	}
	if err := rec.Seal("jpg"); err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	header := rec.Header()
	if header.DateTimeOriginal != "2023:07:14 10:15:30" || header.OffsetTime != "+05:30" || header.Make != "Lumemor" {
		t.Errorf("Header() after Seal = %+v, want the fixture's capture context", header)
	}
	// The header read leaves the staged file in place for Place.
	if _, err := rec.Place(testOwnerID, "2026/01/photo", "jpg"); err != nil {
		t.Fatalf("Place() after Header() error = %v", err)
	}
}
