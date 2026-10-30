package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLookupUploadFormat pins the format metadata the upload handler
// derives from an extension: the canonical MIME type recorded in the
// database and the media family it belongs to. Every whitelisted
// extension maps to a MIME type whose video/ prefix decides the video
// family; anything else is an image.
func TestLookupUploadFormat(t *testing.T) {
	cases := []struct {
		ext       string
		wantMIME  string
		wantVideo bool
		wantOK    bool
	}{
		// Images.
		{ext: "jpg", wantMIME: "image/jpeg", wantOK: true},
		{ext: "jpeg", wantMIME: "image/jpeg", wantOK: true},
		{ext: "png", wantMIME: "image/png", wantOK: true},
		{ext: "gif", wantMIME: "image/gif", wantOK: true},
		{ext: "webp", wantMIME: "image/webp", wantOK: true},
		{ext: "tif", wantMIME: "image/tiff", wantOK: true},
		{ext: "tiff", wantMIME: "image/tiff", wantOK: true},
		{ext: "heic", wantMIME: "image/heic", wantOK: true},
		{ext: "avif", wantMIME: "image/avif", wantOK: true},
		// Videos.
		{ext: "mp4", wantMIME: "video/mp4", wantVideo: true, wantOK: true},
		{ext: "mov", wantMIME: "video/quicktime", wantVideo: true, wantOK: true},
		{ext: "webm", wantMIME: "video/webm", wantVideo: true, wantOK: true},
		{ext: "mkv", wantMIME: "video/x-matroska", wantVideo: true, wantOK: true},
		// Outside the whitelist.
		{ext: "exe", wantOK: false},
		{ext: "txt", wantOK: false},
		{ext: "", wantOK: false},
		{ext: "jpg ", wantOK: false},
	}
	for _, testCase := range cases {
		format, ok := LookupUploadFormat(testCase.ext)
		if ok != testCase.wantOK {
			t.Errorf("LookupUploadFormat(%q) ok = %v, want %v", testCase.ext, ok, testCase.wantOK)
			continue
		}
		if !testCase.wantOK {
			continue
		}
		if format.MIMEType != testCase.wantMIME {
			t.Errorf("LookupUploadFormat(%q).MIMEType = %q, want %q", testCase.ext, format.MIMEType, testCase.wantMIME)
		}
		if format.Video != testCase.wantVideo {
			t.Errorf("LookupUploadFormat(%q).Video = %v, want %v", testCase.ext, format.Video, testCase.wantVideo)
		}
	}
}

// TestReceiverSizeCountsStreamedBytes verifies the byte counter the
// upload handler records as file_size: zero before Commit, the payload
// size after, and equal to the committed file's size on disk.
func TestReceiverSizeCountsStreamedBytes(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	payload := jpegFixture(t)

	receiver, err := NewReceiver(tmpDir, int64(len(payload))+16, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if got := receiver.Size(); got != 0 {
		t.Errorf("Size() before Commit = %d, want 0", got)
	}

	relPath, err := receiver.Commit("jpg")
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if got := receiver.Size(); got != int64(len(payload)) {
		t.Errorf("Size() after Commit = %d, want %d", got, len(payload))
	}
	info, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("stat committed file: %v", err)
	}
	if info.Size() != receiver.Size() {
		t.Errorf("committed file holds %d bytes, Size() = %d, want them equal", info.Size(), receiver.Size())
	}
}

// TestReceiverSizeStaysZeroAfterRejection verifies the counter's
// terminal-state discipline: a rejected upload reports zero, like the
// checksum does.
func TestReceiverSizeStaysZeroAfterRejection(t *testing.T) {
	_, tmpDir := setupLayout(t)

	receiver, err := NewReceiver(tmpDir, 1024, strings.NewReader("definitely not a JPEG payload"))
	if err != nil {
		t.Fatalf("NewReceiver() error = %v", err)
	}
	if _, err := receiver.Commit("jpg"); err == nil {
		t.Fatal("Commit() of a disguised payload succeeded, want rejection")
	}
	if got := receiver.Size(); got != 0 {
		t.Errorf("Size() after rejected Commit = %d, want 0", got)
	}
}
