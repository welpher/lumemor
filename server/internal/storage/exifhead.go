// The ingest-time media header parse (the storage-template feature):
// a bounded, header-only read of the EXIF capture context -- the
// DateTimeOriginal wall clock with its offset tags, and the camera
// identity -- from a file whose head carries a JPEG APP1 Exif segment
// or a bare TIFF header. No subprocess and no library: the tags the
// path render needs are a handful of fixed IFD entries, and owning the
// reader keeps the ingest request path free of exiftool's latency and
// availability.
//
// The parse is deliberately NARROW: only JPEG (whose EXIF lives in the
// first APP1 segment) and TIFF (whose head IS the TIFF header) are
// understood. HEIC, AVIF, videos and the other formats answer "no
// header" and the ingest's taken-at fallback chain (device-reported
// time, then upload time) takes over -- the metadata job still
// extracts the full sidecar for every format later; only the path,
// which renders exactly once, uses this head start.

package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
)

// exifSignature is the APP1 payload signature that precedes a JPEG's
// embedded TIFF header.
var exifSignature = []byte("Exif\x00\x00")

// headerReadBytes bounds the head read of the header parse: EXIF
// segments are at most 64 KiB by the JPEG specification, and the
// parse refuses any offset outside what it actually read, so a
// crafted header cannot turn the reader into an unbounded walk.
const headerReadBytes = 128 << 10

// The TIFF entry tags the header parse reads: the camera identity in
// IFD0, the capture wall clock and its offset tags in the Exif sub-IFD.
const (
	tagMake               = 0x010F
	tagModel              = 0x0110
	tagExifIFD            = 0x8769
	tagDateTimeOriginal   = 0x9003
	tagOffsetTime         = 0x9010
	tagOffsetTimeOriginal = 0x9011
	tagLensModel          = 0xA434
)

// MediaHeader is the ingest-time capture context of one media file:
// the raw EXIF values the taken-at resolution and the path render
// consume. All fields may be empty -- a header with no useful tags
// reads as all-empty and the ingest's fallback chain answers.
type MediaHeader struct {
	// DateTimeOriginal is the EXIF capture wall clock
	// ("2006:01:02 15:04:05"), empty when the file carries none.
	DateTimeOriginal string
	// OffsetTime and OffsetTimeOriginal are the EXIF zone offsets
	// ("+05:30") next to the capture wall clock; many cameras write
	// only the per-tag one.
	OffsetTime         string
	OffsetTimeOriginal string
	// Make, Model and LensModel are the camera identity tags.
	Make      string
	Model     string
	LensModel string
}

// ReadMediaHeader reads the media header of the file at path and
// parses its EXIF capture context. ok is false when the file's head
// carries no parseable EXIF structure (videos, HEIC, PNG and friends
// -- or an unreadable file); the caller then has no header and its
// fallback chain governs.
func ReadMediaHeader(path string) (MediaHeader, bool) {
	f, err := os.Open(path)
	if err != nil {
		return MediaHeader{}, false
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, headerReadBytes)
	read, err := f.ReadAt(head, 0)
	if read <= 0 || (err != nil && !errors.Is(err, io.EOF)) {
		return MediaHeader{}, false
	}
	return parseMediaHeader(head[:read])
}

// parseMediaHeader locates the EXIF TIFF structure in one file head
// and walks it. JPEG files carry it behind the first APP1 marker; TIFF
// files carry it at offset zero.
func parseMediaHeader(head []byte) (MediaHeader, bool) {
	tiffStart := -1
	switch {
	case len(head) >= 4 && head[0] == 0xFF && head[1] == 0xD8:
		tiffStart = jpegExifOffset(head)
	case len(head) >= 4 && (string(head[0:2]) == "II" || string(head[0:2]) == "MM"):
		tiffStart = 0
	}
	if tiffStart < 0 {
		return MediaHeader{}, false
	}
	return parseExifTIFF(head, tiffStart)
}

// jpegExifOffset walks the JPEG segment structure from just past the
// SOI marker and returns the offset of the first APP1 segment's TIFF
// header, or -1 when no EXIF segment precedes the image data. Every
// step is bounds-checked against the head actually read.
func jpegExifOffset(head []byte) int {
	i := 2 // past SOI
	for i+4 <= len(head) {
		if head[i] != 0xFF {
			return -1
		}
		marker := head[i+1]
		switch {
		case marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01:
			// Standalone markers carry no length field.
			i += 2
			continue
		case marker == 0xD9 || marker == 0xDA:
			// End of image or start of scan: the metadata segments
			// all precede the scan, so there is no EXIF to find.
			return -1
		}
		length := int(binary.BigEndian.Uint16(head[i+2 : i+4]))
		if length < 2 {
			return -1
		}
		if marker == 0xE1 && i+4+len(exifSignature) <= len(head) &&
			bytes.HasPrefix(head[i+4:], exifSignature) {
			return i + 4 + len(exifSignature)
		}
		i += 2 + length
	}
	return -1
}

// parseExifTIFF walks the TIFF header at tiffStart: the byte order and
// IFD0 offset from the header, then the IFD0 entries (camera identity,
// the Exif sub-IFD pointer) and the sub-IFD entries (capture wall
// clock, offset tags). Offsets are TIFF-relative; every access is
// bounds-checked, and a malformed structure answers no header rather
// than a best guess.
func parseExifTIFF(head []byte, tiffStart int) (MediaHeader, bool) {
	if tiffStart+8 > len(head) {
		return MediaHeader{}, false
	}
	var bo binary.ByteOrder
	switch string(head[tiffStart : tiffStart+2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return MediaHeader{}, false
	}
	if bo.Uint16(head[tiffStart+2:tiffStart+4]) != 42 {
		return MediaHeader{}, false
	}
	ifd0 := int(bo.Uint32(head[tiffStart+4 : tiffStart+8]))
	header := MediaHeader{}
	walkExifIFD(head, bo, tiffStart, ifd0, &header, 0)
	return header, true
}

// walkExifIFD reads one image file directory and fills header. The
// Exif sub-IFD pointer recurses one level; deeper nesting is refused
// so a crafted chain cannot loop.
func walkExifIFD(head []byte, bo binary.ByteOrder, tiffStart, ifdOffset int, header *MediaHeader, depth int) {
	if depth > 1 || ifdOffset <= 0 || tiffStart+ifdOffset+2 > len(head) {
		return
	}
	base := tiffStart + ifdOffset
	entries := int(bo.Uint16(head[base : base+2]))
	entry := base + 2
	for i := 0; i < entries; i++ {
		if entry+12 > len(head) {
			return
		}
		tag := bo.Uint16(head[entry : entry+2])
		typ := bo.Uint16(head[entry+2 : entry+4])
		count := bo.Uint32(head[entry+4 : entry+8])
		switch tag {
		case tagExifIFD:
			if typ == 4 {
				offset := int(bo.Uint32(head[entry+8 : entry+12]))
				walkExifIFD(head, bo, tiffStart, offset, header, depth+1)
			}
		case tagMake, tagModel, tagLensModel,
			tagDateTimeOriginal, tagOffsetTime, tagOffsetTimeOriginal:
			if text, ok := asciiTag(head, bo, tiffStart, entry, typ, count); ok {
				switch tag {
				case tagMake:
					header.Make = text
				case tagModel:
					header.Model = text
				case tagLensModel:
					header.LensModel = text
				case tagDateTimeOriginal:
					header.DateTimeOriginal = text
				case tagOffsetTime:
					header.OffsetTime = text
				case tagOffsetTimeOriginal:
					header.OffsetTimeOriginal = text
				}
			}
		}
		entry += 12
	}
}

// asciiTag reads one ASCII-typed (type 2) entry value: the bytes are
// inline in the entry's value field when they fit four bytes, else at
// the TIFF-relative offset the value field carries. The trailing NUL
// and padding are trimmed.
func asciiTag(head []byte, bo binary.ByteOrder, tiffStart, entry int, typ uint16, count uint32) (string, bool) {
	if typ != 2 || count == 0 || count > 1024 {
		return "", false
	}
	start := entry + 8 // the inline value field
	if count > 4 {
		start = tiffStart + int(bo.Uint32(head[entry+8:entry+12]))
	}
	if start < 0 || start+int(count) > len(head) {
		return "", false
	}
	return strings.Trim(strings.Trim(string(head[start:start+int(count)]), "\x00"), " "), true
}
