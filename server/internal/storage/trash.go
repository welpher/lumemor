// The trash-area moves of the external asset lifecycle (the
// move semantics): a directory album's referenced
// file is not copied at ingest (zero copy, zero move -- it stays
// wherever the family put it), so the trash lifecycle MOVES it --
// into data/trash/<assetID>/ when the owner trashes the asset, back
// to the row's recorded original location when they restore it.
// The move is the delete's permission story: an operating system
// that refuses it (an unreadable source, an unwritable directory,
// an original location that no longer exists) fails the request
// explicitly -- the row is not written, the file stays where it
// was, and the caller answers the honest error code.
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// sniffWindow is the leading byte count the external ingest reads
// for format sniffing -- the same 512-byte window the upload
// receiver captures while streaming (http.DetectContentType's own
// horizon), so a referenced file validates against exactly the
// sniffing discipline an uploaded one does.
const sniffWindow = 512

// TrashHome returns the trash-area directory of one asset under the
// given data root: trash/<assetID>/. The path is keyed by the asset
// id alone -- no other row can hold files inside it -- which is
// what makes the expiry's removal of the moved copy unconditional,
// with no reference counting.
func TrashHome(dataDir, assetID string) string {
	return filepath.Join(dataDir, "trash", assetID)
}

// MoveToTrash moves one external asset's file into its trash home
// and answers the absolute path it now sits at (the value the
// caller records as the row's trash_path and file_path). The target
// directory is created on demand; the move itself is rename-first
// (atomic within one filesystem) with a copy-then-remove fallback
// for the cross-filesystem case -- the mount root and the data root
// legitimately live on different devices. Every failure leaves the
// source exactly where it was: a failed rename simply never moved
// it, and a failed copy or remove cleans the partial target before
// returning, so the caller's error answer describes a state with
// nothing half-moved.
func MoveToTrash(dataDir, source, assetID string) (string, error) {
	if assetID == "" {
		return "", fmt.Errorf("storage: MoveToTrash: asset id is empty")
	}
	target := filepath.Join(TrashHome(dataDir, assetID), filepath.Base(source))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("storage: create the trash home of %s: %w", assetID, err)
	}
	if err := moveFile(source, target); err != nil {
		// The home this call just created is empty (the move's own
		// failure paths clean their partial targets), so removing it
		// leaves the failure state exactly as it was found: no
		// directory, no file, nothing half-moved.
		_ = os.Remove(filepath.Dir(target))
		return "", fmt.Errorf("storage: move %s into the trash: %w", source, err)
	}
	return target, nil
}

// MoveFileBack moves one external asset's file from its trash home
// back to the recorded original location (the restore's move). The
// destination's parent is deliberately NOT fabricated: an original
// location that no longer exists -- the source folder unmounted,
// the directory deleted -- fails the move, which is the honest
// explicit failure the restore answers (the row stays trashed, the
// file stays in the trash area, a retry after the location returns
// completes it). Creating the missing path instead would silently
// write into a fabricated directory at an unmounted volume's mount
// point, a file the family would never find.
func MoveFileBack(trashPath, originalPath string) error {
	if err := moveFile(trashPath, originalPath); err != nil {
		return fmt.Errorf("storage: move %s back to %s: %w", trashPath, originalPath, err)
	}
	return nil
}

// moveFile moves one file from source to target: an os.Rename when
// it works (atomic, and the common same-device case), a
// copy-then-remove otherwise. The fallback is not gated on the
// cross-device error specifically: a rename that failed for a
// reason copy cannot fix (the source unreadable, the target's
// parent unwritable) fails the copy at the same step with the
// honest error, and the remove after it never runs -- one move
// primitive, one failure story, whichever step the operating
// system refused.
func moveFile(source, target string) error {
	if err := os.Rename(source, target); err == nil {
		return nil
	}
	if err := copyFile(source, target); err != nil {
		return err
	}
	if err := os.Remove(source); err != nil {
		// The copy succeeded but the source could not be removed:
		// the file would exist at BOTH paths. Remove the target
		// again -- best-effort, the copy just proved the target's
		// directory writable -- and answer the failure so the
		// caller's state stays "nothing moved".
		_ = os.Remove(target)
		return fmt.Errorf("remove the moved-from file: %w", err)
	}
	return nil
}

// copyFile copies the bytes and mode of source to target.
func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open the file to move: %w", err)
	}
	defer func() { _ = in.Close() }()
	info, err := in.Stat()
	if err != nil {
		return fmt.Errorf("stat the file to move: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("the path to move %q is not a regular file", source)
	}
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create the moved copy: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(target)
		return fmt.Errorf("copy the file to move: %w", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(target)
		return fmt.Errorf("close the moved copy: %w", err)
	}
	return nil
}

// ReadExternalFile reads one referenced file in a single pass and
// answers the values the rescan's ingest needs: the sha256 hex
// digest of the whole content, the byte count, and the leading
// sniff window (the upload receiver's own 512 bytes) for the format
// validation. The digest is computed over the same bytes the sniff
// window came from -- the read is one pass, head first.
func ReadExternalFile(path string) (checksum string, size int64, head []byte, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, nil, fmt.Errorf("open the source file: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return "", 0, nil, fmt.Errorf("stat the source file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", 0, nil, fmt.Errorf("the source file %q is not a regular file", path)
	}

	digest := sha256.New()
	head = make([]byte, sniffWindow)
	read, err := io.ReadFull(file, head)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", 0, nil, fmt.Errorf("read the source head: %w", err)
	}
	head = head[:read]
	if _, err := digest.Write(head); err != nil {
		return "", 0, nil, fmt.Errorf("hash the source head: %w", err)
	}
	copied, err := io.Copy(digest, file)
	if err != nil {
		return "", 0, nil, fmt.Errorf("hash the source body: %w", err)
	}
	return hex.EncodeToString(digest.Sum(nil)), int64(read) + copied, head, nil
}

// FormatMatchesHead reports whether a referenced file's leading
// bytes are consistent with the whitelisted extension it carries
// (canonical, lowercase, dotless): the same magic-byte validation
// the upload receiver's Seal applies over the streamed payload, on
// the same format table -- the rescan's reference-style ingest runs
// the content through the ingest pipeline's own gates rather than
// a parallel set. An extension outside the whitelist answers false
// with the caller's extension-lookup answer already distinguishing
// that case.
func FormatMatchesHead(ext string, head []byte) bool {
	rule, ok := uploadFormats[ext]
	if !ok {
		return false
	}
	return rule.consistentWith(http.DetectContentType(head), head)
}
