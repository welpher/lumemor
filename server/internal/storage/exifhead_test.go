package storage

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// exifEntry is one ASCII tag the test builder writes into an IFD.
type exifEntry struct {
	tag   uint16
	value string
}

// buildExifTIFF assembles a minimal TIFF structure -- header, IFD0
// with the given entries plus the Exif sub-IFD pointer, the Exif
// sub-IFD with its entries, then the out-of-line string values -- in
// the given byte order. It is the inverse of the header parser: every
// layout rule the parser reads (the 42 magic, TIFF-relative offsets,
// inline versus out-of-line values) is written here by hand.
func buildExifTIFF(bo binary.ByteOrder, ifd0, exif []exifEntry) []byte {
	// Layout: header(8) | IFD0 | Exif IFD | value area.
	ifd0Size := 2 + 12*(len(ifd0)+1) + 4 // +1: the Exif sub-IFD pointer
	exifSize := 2 + 12*len(exif) + 4
	exifOffset := 8 + ifd0Size
	valueBase := exifOffset + exifSize

	// The out-of-line values are appended after the IFDs, in the
	// order the entries appear; the map records each entry's offset.
	valueOffsets := make(map[uint16]uint32)
	valueArea := make([]byte, 0)
	appendValue := func(tag uint16, value string) {
		// Align the value area to two bytes like real writers do.
		if len(valueArea)%2 == 1 {
			valueArea = append(valueArea, 0)
		}
		valueOffsets[tag] = uint32(valueBase + len(valueArea))
		valueArea = append(valueArea, value...)
		valueArea = append(valueArea, 0) // the terminating NUL
	}

	entry := func(tag, typ uint16, count uint32, value uint32) []byte {
		buf := make([]byte, 12)
		bo.PutUint16(buf[0:2], tag)
		bo.PutUint16(buf[2:4], typ)
		bo.PutUint32(buf[4:8], count)
		bo.PutUint32(buf[8:12], value)
		return buf
	}
	appendUint16 := func(buf []byte, v uint16) []byte {
		var raw [2]byte
		bo.PutUint16(raw[:], v)
		return append(buf, raw[:]...)
	}
	asciiEntry := func(e exifEntry) []byte {
		count := uint32(len(e.value) + 1) // including the NUL
		if count <= 4 {
			// Inline: the value bytes sit in the entry's value field
			// itself, NUL padded.
			value := make([]byte, 4)
			copy(value, e.value)
			return entry(e.tag, 2, count, bo.Uint32(value))
		}
		appendValue(e.tag, e.value)
		return entry(e.tag, 2, count, valueOffsets[e.tag])
	}

	buf := make([]byte, 0, valueBase)
	// The header: byte order mark, the 42 magic, IFD0 at offset 8 --
	// all in the structure's own byte order.
	var hdr [8]byte
	if bo == binary.BigEndian {
		copy(hdr[0:2], "MM")
	} else {
		copy(hdr[0:2], "II")
	}
	bo.PutUint16(hdr[2:4], 42)
	bo.PutUint32(hdr[4:8], 8)
	buf = append(buf, hdr[:]...)

	// IFD0: the entry count, the entries, the Exif pointer, no next
	// IFD.
	buf = appendUint16(buf, uint16(len(ifd0)+1))
	for _, e := range ifd0 {
		buf = append(buf, asciiEntry(e)...)
	}
	buf = append(buf, entry(tagExifIFD, 4, 1, uint32(exifOffset))...)
	buf = append(buf, 0, 0, 0, 0)

	// The Exif sub-IFD.
	buf = appendUint16(buf, uint16(len(exif)))
	for _, e := range exif {
		buf = append(buf, asciiEntry(e)...)
	}
	buf = append(buf, 0, 0, 0, 0)

	buf = append(buf, valueArea...)
	return buf
}

// buildExifJPEG wraps an EXIF TIFF structure in a real JPEG: the SOI,
// the APP1 Exif segment, then the marker stream of a genuinely
// encoded image -- a payload the upload receiver's content sniff
// accepts, so the same builder feeds the full ingest tests.
func buildExifJPEG(t *testing.T, tiff []byte) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 0x7f, G: 0x33, B: 0xcc, A: 0xff})
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, nil); err != nil {
		t.Fatalf("encode JPEG body: %v", err)
	}
	segmentLength := len(exifSignature) + len(tiff)
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xD8}) // SOI
	buf.Write([]byte{0xFF, 0xE1}) // APP1
	buf.Write(binary.BigEndian.AppendUint16(nil, uint16(2+segmentLength)))
	buf.Write(exifSignature)
	buf.Write(tiff)
	buf.Write(encoded.Bytes()[2:]) // the encoded body's own markers, past its SOI
	return buf.Bytes()
}

// writeFixtureFile writes data to a fresh file and returns its path.
func writeFixtureFile(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// goldenHeader reads one of the jobs package's committed golden JPEGs
// -- real bytes a real camera-like EXIF writer produced -- so the
// parser is proven against files it did not build itself.
func goldenHeader(t *testing.T, name string) MediaHeader {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "jobs", "testdata", name))
	if err != nil {
		t.Fatalf("read golden fixture %s: %v", name, err)
	}
	header, ok := ReadMediaHeader(writeFixtureFile(t, data))
	if !ok {
		t.Fatalf("ReadMediaHeader(%s) reported no header, want the golden EXIF", name)
	}
	return header
}

// TestReadMediaHeaderJpegExif walks a built little-endian fixture: the
// capture wall clock and offset tags from the Exif sub-IFD, the camera
// identity from IFD0, the inline and the out-of-line value forms alike.
func TestReadMediaHeaderJpegExif(t *testing.T) {
	tiff := buildExifTIFF(binary.LittleEndian,
		[]exifEntry{
			{tag: tagMake, value: "Lumemor"},  // out-of-line (7 bytes)
			{tag: tagModel, value: "Alpha 5"}, // out-of-line
		},
		[]exifEntry{
			{tag: tagDateTimeOriginal, value: "2023:07:14 10:15:30"},
			{tag: tagOffsetTime, value: "+05:30"},
			{tag: tagLensModel, value: "35mm f/2.8"},
		})
	header, ok := ReadMediaHeader(writeFixtureFile(t, buildExifJPEG(t, tiff)))
	if !ok {
		t.Fatal("ReadMediaHeader reported no header, want the EXIF parse")
	}
	if header.DateTimeOriginal != "2023:07:14 10:15:30" {
		t.Errorf("DateTimeOriginal = %q, want 2023:07:14 10:15:30", header.DateTimeOriginal)
	}
	if header.OffsetTime != "+05:30" {
		t.Errorf("OffsetTime = %q, want +05:30", header.OffsetTime)
	}
	if header.OffsetTimeOriginal != "" {
		t.Errorf("OffsetTimeOriginal = %q, want empty", header.OffsetTimeOriginal)
	}
	if header.Make != "Lumemor" {
		t.Errorf("Make = %q, want Lumemor", header.Make)
	}
	if header.Model != "Alpha 5" {
		t.Errorf("Model = %q, want Alpha 5", header.Model)
	}
	if header.LensModel != "35mm f/2.8" {
		t.Errorf("LensModel = %q, want 35mm f/2.8", header.LensModel)
	}
}

// TestReadMediaHeaderTiffByteOrder proves the big-endian TIFF order:
// the same structure in "MM" order parses to the same values.
func TestReadMediaHeaderTiffByteOrder(t *testing.T) {
	tiff := buildExifTIFF(binary.BigEndian,
		[]exifEntry{{tag: tagMake, value: "Northcam"}},
		[]exifEntry{
			{tag: tagDateTimeOriginal, value: "2020:01:02 03:04:05"},
			{tag: tagOffsetTimeOriginal, value: "-08:00"},
		})
	header, ok := ReadMediaHeader(writeFixtureFile(t, tiff))
	if !ok {
		t.Fatal("ReadMediaHeader of the big-endian TIFF reported no header")
	}
	if header.DateTimeOriginal != "2020:01:02 03:04:05" {
		t.Errorf("DateTimeOriginal = %q, want 2020:01:02 03:04:05", header.DateTimeOriginal)
	}
	if header.OffsetTimeOriginal != "-08:00" {
		t.Errorf("OffsetTimeOriginal = %q, want -08:00", header.OffsetTimeOriginal)
	}
	if header.Make != "Northcam" {
		t.Errorf("Make = %q, want Northcam", header.Make)
	}
}

// TestReadMediaHeaderInlineValues proves the short-string form: a
// value of at most four bytes lives inside the entry itself.
func TestReadMediaHeaderInlineValues(t *testing.T) {
	tiff := buildExifTIFF(binary.LittleEndian,
		[]exifEntry{{tag: tagModel, value: "X100"}}, // 5 bytes with NUL: still out-of-line
		[]exifEntry{
			{tag: tagDateTimeOriginal, value: "2023:01:01 00:00:00"},
			{tag: tagOffsetTime, value: "Z"}, // 2 bytes with NUL: inline
		})
	header, ok := ReadMediaHeader(writeFixtureFile(t, buildExifJPEG(t, tiff)))
	if !ok {
		t.Fatal("ReadMediaHeader reported no header")
	}
	if header.OffsetTime != "Z" {
		t.Errorf("inline OffsetTime = %q, want Z", header.OffsetTime)
	}
}

// TestReadMediaHeaderGoldenFixtures proves the parser against the
// committed golden JPEGs whose exiftool extraction the metadata job
// tests already pin: the same tags, the same values.
func TestReadMediaHeaderGoldenFixtures(t *testing.T) {
	golden := goldenHeader(t, "golden.jpg")
	if golden.DateTimeOriginal != "2023:07:14 10:15:30" {
		t.Errorf("golden DateTimeOriginal = %q, want 2023:07:14 10:15:30", golden.DateTimeOriginal)
	}
	if golden.OffsetTime != "+05:30" {
		t.Errorf("golden OffsetTime = %q, want +05:30", golden.OffsetTime)
	}
	if golden.Make != "Lumemor" {
		t.Errorf("golden Make = %q, want Lumemor", golden.Make)
	}
	if golden.Model != "Golden Camera" {
		t.Errorf("golden Model = %q, want Golden Camera", golden.Model)
	}
	if golden.LensModel != "Golden 35mm f/2.8" {
		t.Errorf("golden LensModel = %q, want Golden 35mm f/2.8", golden.LensModel)
	}

	// The variant that carries only the per-tag offset.
	offsetOriginal := goldenHeader(t, "golden-offsetoriginal.jpg")
	if offsetOriginal.DateTimeOriginal != "2023:07:14 10:15:30" {
		t.Errorf("offset-original DateTimeOriginal = %q, want 2023:07:14 10:15:30", offsetOriginal.DateTimeOriginal)
	}
	if offsetOriginal.OffsetTime != "" {
		t.Errorf("offset-original OffsetTime = %q, want empty", offsetOriginal.OffsetTime)
	}
	if offsetOriginal.OffsetTimeOriginal != "+07:00" {
		t.Errorf("offset-original OffsetTimeOriginal = %q, want +07:00", offsetOriginal.OffsetTimeOriginal)
	}

	// The variant with no offset at all: the wall clock stands, the
	// offsets stay empty, and the ingest's zone chain falls through.
	noOffset := goldenHeader(t, "golden-nooffset.jpg")
	if noOffset.DateTimeOriginal == "" {
		t.Error("no-offset DateTimeOriginal is empty, want the wall clock")
	}
	if noOffset.OffsetTime != "" || noOffset.OffsetTimeOriginal != "" {
		t.Errorf("no-offset offsets = %q/%q, want empty/empty", noOffset.OffsetTime, noOffset.OffsetTimeOriginal)
	}
}

// TestReadMediaHeaderWithoutExif proves the honest no-header answer
// for everything the narrow parse does not understand: a plain JPEG
// (no APP1), a PNG, an MP4 and an empty file.
func TestReadMediaHeaderWithoutExif(t *testing.T) {
	for name, data := range map[string][]byte{
		"plain jpeg": jpegFixture(t),
		"png":        pngFixture(t),
		"mp4":        readFixture(t, "minimal.mp4"),
		"heic":       readFixture(t, "minimal.heic"),
		"empty":      nil,
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := ReadMediaHeader(writeFixtureFile(t, data)); ok {
				t.Errorf("ReadMediaHeader of %s reported a header, want none", name)
			}
		})
	}
}

// TestReadMediaHeaderMalformedStructure proves the bounded walk: a
// TIFF whose IFD0 offset points past the head answers a structure with
// empty fields instead of following the pointer.
func TestReadMediaHeaderMalformedStructure(t *testing.T) {
	tiff := buildExifTIFF(binary.LittleEndian,
		[]exifEntry{{tag: tagMake, value: "Lumemor"}},
		[]exifEntry{{tag: tagDateTimeOriginal, value: "2023:07:14 10:15:30"}})
	// Corrupt the IFD0 offset (bytes 4-8 of the TIFF header) to point
	// far past the buffer.
	bo := binary.LittleEndian
	bo.PutUint32(tiff[4:8], 1<<30)
	header, ok := ReadMediaHeader(writeFixtureFile(t, buildExifJPEG(t, tiff)))
	if !ok {
		t.Fatal("ReadMediaHeader of a corrupted IFD offset reported no TIFF header")
	}
	if header.DateTimeOriginal != "" || header.Make != "" {
		t.Errorf("corrupted IFD offset yielded %q/%q, want empty fields (the walk refused the pointer)",
			header.DateTimeOriginal, header.Make)
	}
}
