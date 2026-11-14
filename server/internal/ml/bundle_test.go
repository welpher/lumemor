// The offline bundle import tests: a bundle built from the same
// fixture models the download tests use, archived in-process and
// imported through the real registry and queries.
//
//   - the equivalence pin: a bundle import and an online download end
//     in the same state -- same layout, manifest copies, rows with
//     source 'downloaded', empty-slot activations;
//   - gate one: a member that drifted inside the archive is refused
//     with its name;
//   - gate two: a bundle whose bytes disagree with the manifest's pins
//     is refused naming the file;
//   - a bundle without its checksums file verifies nothing and is
//     refused.
package ml

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bundleArchive builds the bundle tar for the models: manifest.json
// plus checksums.txt plus every file, plain tar. The manifest's model
// list is written verbatim, so the pins travel with the bytes.
func bundleArchive(t *testing.T, manifest *Manifest, models []downloadModel) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	// The manifest member.
	blob, err := manifest.marshal()
	if err != nil {
		t.Fatalf("marshal the bundle manifest: %v", err)
	}
	writeTarMember(t, tw, "manifest.json", blob)
	// The model files, in their model directories.
	type member struct {
		name string
		data []byte
	}
	members := []member{}
	for _, m := range models {
		for _, f := range m.files {
			members = append(members, member{m.id + "/" + f.name, f.content})
		}
	}
	// The checksums file last, naming everything written before it.
	var lines []string
	digestOf := func(data []byte) string {
		got := sha256.Sum256(data)
		return hex.EncodeToString(got[:])
	}
	lines = append(lines, fmt.Sprintf("%s  manifest.json", digestOf(blob)))
	for _, member := range members {
		lines = append(lines, fmt.Sprintf("%s  %s", digestOf(member.data), member.name))
	}
	for _, member := range members {
		writeTarMember(t, tw, member.name, member.data)
	}
	writeTarMember(t, tw, BundleChecksumsName, []byte(strings.Join(lines, "\n")+"\n"))
	if err := tw.Close(); err != nil {
		t.Fatalf("close the bundle tar: %v", err)
	}
	return buf.Bytes()
}

// writeTarMember writes one regular file member.
func writeTarMember(t *testing.T, tw *tar.Writer, name string, data []byte) {
	t.Helper()
	dir := filepath.Dir(name)
	if dir != "." {
		hdr := &tar.Header{Name: dir + "/", Typeflag: tar.TypeDir, Mode: 0o755}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write the dir header of %s: %v", name, err)
		}
	}
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data))}); err != nil {
		t.Fatalf("write the header of %s: %v", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("write the body of %s: %v", name, err)
	}
}

// TestImportBundleMatchesTheOnlinePath pins the equivalence: the
// bundle's models end exactly where an online download puts them --
// same layout, manifest copies, rows with source 'downloaded', and
// the empty slots activated through the real registry.
func TestImportBundleMatchesTheOnlinePath(t *testing.T) {
	e := newDownloadEnv(t)
	group := downloadGroup()
	manifest := downloadManifest(group...)
	bundle := filepath.Join(t.TempDir(), "models-test.tar")
	if err := os.WriteFile(bundle, bundleArchive(t, manifest, group), 0o644); err != nil {
		t.Fatalf("write the bundle: %v", err)
	}

	report, err := ImportBundle(context.Background(), bundle, e.modelsDir, e.registry, e.queries)
	if err != nil {
		t.Fatalf("ImportBundle error = %v, want nil", err)
	}
	if report.ModelsDone != len(group) {
		t.Errorf("report models done = %d, want %d", report.ModelsDone, len(group))
	}
	// The checksums file names the manifest and every model file; it
	// does not name itself.
	if report.Members != 6 {
		t.Errorf("report members = %d, want 6 (the manifest and the five fixture files)", report.Members)
	}
	for _, m := range group {
		e.assertModelLanded(t, m)
	}
	// No scratch leftovers beside the models dir.
	entries, err := os.ReadDir(filepath.Dir(e.modelsDir))
	if err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".bundle-") {
				t.Errorf("the bundle scratch dir %s survived the import", entry.Name())
			}
		}
	}
}

// TestImportBundleRefusesDriftedMember pins gate one: a member whose
// bytes drifted inside the archive fails the import with the member
// named, and nothing of the failed model registers.
func TestImportBundleRefusesDriftedMember(t *testing.T) {
	e := newDownloadEnv(t)
	group := downloadGroup()
	manifest := downloadManifest(group...)
	blob := bundleArchive(t, manifest, group)
	// Tamper with the first model's first file inside the archive:
	// its checksums line names different bytes, gate one refuses.
	tampered := bytes.Replace(blob, group[0].files[0].content[:8], bytes.Repeat([]byte{0xff}, 8), 1)
	bundle := filepath.Join(t.TempDir(), "models-test.tar")
	if err := os.WriteFile(bundle, tampered, 0o644); err != nil {
		t.Fatalf("write the bundle: %v", err)
	}

	if _, err := ImportBundle(context.Background(), bundle, e.modelsDir, e.registry, e.queries); err == nil {
		t.Fatalf("ImportBundle error = nil, want the drifted member to refuse the import")
	}
	if _, ok := e.modelRowOf(t, group[0].id); ok {
		t.Errorf("the drifted model registered despite the refused member")
	}
}

// TestImportBundleRefusesPinMismatch pins gate two: a bundle whose
// member bytes differ from the manifest's pinned digest is refused
// naming the file -- the same bytes cannot satisfy two manifests.
func TestImportBundleRefusesPinMismatch(t *testing.T) {
	e := newDownloadEnv(t)
	group := downloadGroup()
	manifest := downloadManifest(group...)
	// The manifest pins tampered digests: the bytes inside the bundle
	// are the fixture's, the pins name something else.
	for i := range manifest.Models {
		manifest.Models[i].Files[0].SHA256 = strings.Repeat("ab", 32)
	}
	bundle := filepath.Join(t.TempDir(), "models-test.tar")
	if err := os.WriteFile(bundle, bundleArchive(t, manifest, group), 0o644); err != nil {
		t.Fatalf("write the bundle: %v", err)
	}

	if _, err := ImportBundle(context.Background(), bundle, e.modelsDir, e.registry, e.queries); err == nil {
		t.Fatalf("ImportBundle error = nil, want the pin mismatch to refuse the import")
	}
	if got := e.modelCount(t); got != 0 {
		t.Errorf("models rows = %d, want 0 (nothing registered)", got)
	}
}

// TestImportBundleRefusesMissingChecksums pins the authority rule: a
// bundle without its checksums file verifies nothing and is refused.
func TestImportBundleRefusesMissingChecksums(t *testing.T) {
	e := newDownloadEnv(t)
	group := downloadGroup()[:1]
	manifest := downloadManifest(group...)
	blob, err := manifest.marshal()
	if err != nil {
		t.Fatalf("marshal the manifest: %v", err)
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	writeTarMember(t, tw, "manifest.json", blob)
	writeTarMember(t, tw, group[0].id+"/"+group[0].files[0].name, group[0].files[0].content)
	_ = tw.Close()
	bundle := filepath.Join(t.TempDir(), "models-test.tar")
	if err := os.WriteFile(bundle, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write the bundle: %v", err)
	}

	if _, err := ImportBundle(context.Background(), bundle, e.modelsDir, e.registry, e.queries); err == nil {
		t.Fatalf("ImportBundle error = nil, want the missing checksums file to refuse the import")
	}
	if got := e.modelCount(t); got != 0 {
		t.Errorf("models rows = %d, want 0 (nothing registered)", got)
	}
}
