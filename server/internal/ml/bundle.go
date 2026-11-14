// The offline bundle import: the no-network path into the very same
// registry the online download writes. A deployment that cannot reach
// the pinned sources carries a bundle across -- one tar archive
// holding the manifest and the weights, plus a `checksums.txt` side
// file naming every member's sha256 -- and the import walks it with
// the same gates the online chain applies:
//
//   - every member is digest-checked against `checksums.txt` first
//     (the archive itself is not the authority -- a member that
//     drifted inside it is refused with its name),
//   - every manifest entry's pinned digest must match the unpacked
//     bytes (a bundle built against a different manifest revision is
//     refused naming the file and both digests),
//   - the landed files register through the downloader's own
//     registration path, so an offline install and an online download
//     are indistinguishable afterwards: same layout, same manifest
//     copies, same rows, same empty-slot activations.
//
// The bundle carries no state the online path would not have: nothing
// is activated that an online download would not have activated.
package ml

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// BundleChecksumsName is the side file the bundle carries: one
// `<sha256>  <member>` line per archive member, the two-space shape
// the sha256sum tool writes. The member names are archive-relative
// ("manifest.json", "<model-id>/<file>").
const BundleChecksumsName = "checksums.txt"

// ImportReport summarizes one bundle import.
type ImportReport struct {
	// Members is the number of members the checksums file named.
	Members int
	// ModelsDone is the number of models fully landed and registered.
	ModelsDone int
}

// ImportBundle unpacks the bundle tarball at bundlePath, verifies it
// against its own checksums file and the manifest's pins, and lands
// every model into modelsDir through the same registration the online
// download runs. A member that fails either gate aborts the import
// with the member named; a model registers only once every file of it
// verified, and a refused import leaves the models dir exactly as it
// was (the unpack happens in a temporary directory that is removed
// either way).
func ImportBundle(ctx context.Context, bundlePath, modelsDir string, registry *Registry, queries *dbgen.Queries) (ImportReport, error) {
	if registry == nil || queries == nil {
		return ImportReport{}, errors.New("ml: bundle import requires a registry and queries")
	}
	archive, err := os.Open(bundlePath)
	if err != nil {
		return ImportReport{}, fmt.Errorf("ml: open the bundle %s: %w", bundlePath, err)
	}
	defer func() { _ = archive.Close() }()

	// The unpack target: a scratch directory beside the models dir
	// (same filesystem, so the final moves stay renames), removed
	// whatever the import decides.
	scratch, err := os.MkdirTemp(filepath.Dir(modelsDir), ".bundle-")
	if err != nil {
		return ImportReport{}, fmt.Errorf("ml: create the bundle scratch dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(scratch) }()

	checksums, err := unpackBundle(ctx, archive, scratch)
	if err != nil {
		return ImportReport{}, err
	}
	manifestPath := filepath.Join(scratch, "manifest.json")
	// The manifest itself is a member: its own line must verify too
	// (checksums.txt names it like any other file).
	if err := verifyMember(manifestPath, checksums["manifest.json"]); err != nil {
		return ImportReport{}, err
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return ImportReport{}, err
	}

	report := ImportReport{Members: len(checksums)}
	var failures []error
	for i := range manifest.Models {
		model := &manifest.Models[i]
		if err := importBundleModel(ctx, scratch, modelsDir, model, checksums, registry, queries); err != nil {
			failures = append(failures, err)
			continue
		}
		report.ModelsDone++
	}
	if len(failures) > 0 {
		return report, fmt.Errorf("ml: bundle import failed for %d model(s): %w", len(failures), errors.Join(failures...))
	}
	return report, nil
}

// importBundleModel lands one model of the bundle: every file
// verified against BOTH the bundle's own checksums file and the
// manifest's pin, then renamed into place and registered exactly as
// the online download registers.
func importBundleModel(ctx context.Context, scratch, modelsDir string, model *ManifestModel, checksums map[string]string, registry *Registry, queries *dbgen.Queries) error {
	slot, ok := manifestSlotValues[model.Slot]
	if !ok {
		return fmt.Errorf("model %s fills unknown slot %q", model.ID, model.Slot)
	}
	if len(model.Files) == 0 {
		return fmt.Errorf("model %s lists no files", model.ID)
	}
	modelDir := filepath.Join(modelsDir, model.ID)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return fmt.Errorf("create the model dir of %s: %w", model.ID, err)
	}
	for _, file := range model.Files {
		staged := filepath.Join(scratch, model.ID, file.Name)
		// Gate one: the bundle's own checksums file. A member that
		// drifted inside the archive is refused with its name.
		if err := verifyMember(staged, checksums[model.ID+"/"+file.Name]); err != nil {
			return err
		}
		// Gate two: the manifest's pin. A bundle built against a
		// different manifest revision is refused naming both
		// digests, so the operator knows which file disagrees.
		if err := verifyModelDigest(staged, file.SHA256); err != nil {
			return err
		}
		target := filepath.Join(modelDir, file.Name)
		if err := os.Rename(staged, target); err != nil {
			return fmt.Errorf("move %s into place: %w", file.Name, err)
		}
	}
	if err := writeManifestCopy(modelDir, model); err != nil {
		return err
	}
	params := []byte("{}")
	if model.Params != nil {
		blob, err := json.Marshal(model.Params)
		if err != nil {
			return fmt.Errorf("marshal the params of %s: %w", model.ID, err)
		}
		params = blob
	}
	if err := queries.InsertModel(ctx, dbgen.InsertModelParams{
		ID:       model.ID,
		Slot:     dbgen.ModelSlot(slot),
		Version:  model.Version,
		Source:   dbgen.ModelSourceDownloaded,
		FilePath: model.ID + "/" + model.Files[0].Name,
		InputSize: pgtype.Int4{
			Int32: int32(model.InputSize.Width),
			Valid: model.InputSize.Width > 0,
		},
		Dim: pgtype.Int4{
			Int32: int32(model.Dim),
			Valid: model.Dim > 0,
		},
		Params: params,
	}); err != nil {
		return fmt.Errorf("register %s: %w", model.ID, err)
	}
	if _, err := registry.Active(slot); errors.Is(err, ErrNoActiveModel) {
		if _, err := registry.Activate(slot, model.ID); err != nil {
			return fmt.Errorf("activate %s into slot %s: %w", model.ID, slot, err)
		}
	}
	return nil
}

// unpackBundle streams the (optionally gzip-compressed) tar into
// scratch and answers the parsed checksums file. Member paths are
// zip-slip checked (no absolute paths, no `..`), and every member is
// extracted as the archive streams it -- the checksums file may sit
// anywhere in the archive, so the gates run AFTER the unpack, over
// the extracted tree.
func unpackBundle(ctx context.Context, archive io.Reader, scratch string) (map[string]string, error) {
	// The bundle may be plain tar or gzip tar: sniff the first two
	// bytes instead of trusting an extension.
	head := make([]byte, 2)
	if _, err := io.ReadFull(archive, head); err != nil {
		return nil, fmt.Errorf("ml: read the bundle head: %w", err)
	}
	reader := io.MultiReader(strings.NewReader(string(head)), archive)
	if head[0] == 0x1f && head[1] == 0x8b {
		gz, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("ml: open the gzip bundle: %w", err)
		}
		defer func() { _ = gz.Close() }()
		reader = gz
	}
	tr := tar.NewReader(reader)
	var checksums map[string]string
	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("ml: the bundle import was cancelled: %w", err)
		}
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("ml: read the bundle: %w", err)
		}
		name := strings.TrimPrefix(hdr.Name, "./")
		if filepath.IsAbs(name) || strings.HasPrefix(name, "../") || strings.Contains(name, "/../") {
			return nil, fmt.Errorf("ml: the bundle carries the unsafe member %q", hdr.Name)
		}
		target := filepath.Join(scratch, filepath.FromSlash(name))
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, fmt.Errorf("ml: create the member dir %s: %w", name, err)
			}
			continue
		}
		if name == BundleChecksumsName {
			blob, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("ml: read %s: %w", BundleChecksumsName, err)
			}
			parsed, err := parseChecksums(blob)
			if err != nil {
				return nil, err
			}
			checksums = parsed
			if err := os.WriteFile(target, blob, 0o644); err != nil {
				return nil, fmt.Errorf("ml: write %s: %w", BundleChecksumsName, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, fmt.Errorf("ml: create the member dir of %s: %w", name, err)
		}
		if err := writeMember(tr, target); err != nil {
			return nil, err
		}
	}
	if checksums == nil {
		return nil, errors.New("ml: the bundle carries no " + BundleChecksumsName)
	}
	return checksums, nil
}

// writeMember streams one tar member to disk.
func writeMember(src io.Reader, target string) error {
	out, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("ml: create the member %s: %w", target, err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, src); err != nil {
		return fmt.Errorf("ml: extract the member %s: %w", target, err)
	}
	return nil
}

// parseChecksums reads one `<sha256>  <member>` line per bundle
// member, refusing malformed lines with the line number.
func parseChecksums(blob []byte) (map[string]string, error) {
	result := map[string]string{}
	for i, line := range strings.Split(string(blob), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 || len(parts[0]) != 64 {
			return nil, fmt.Errorf("ml: %s line %d is not <sha256> <member>", BundleChecksumsName, i+1)
		}
		result[parts[1]] = parts[0]
	}
	return result, nil
}

// verifyMember checks one unpacked member against its checksums line.
func verifyMember(path, expected string) error {
	if expected == "" {
		return fmt.Errorf("ml: %s names no digest for %s", BundleChecksumsName, filepath.Base(path))
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ml: the bundle is missing the member %s: %w", filepath.Base(path), err)
	}
	got := sha256.Sum256(blob)
	if hex.EncodeToString(got[:]) != expected {
		return fmt.Errorf("ml: the member %s drifted inside the bundle (sha256 %s, want %s)",
			filepath.Base(path), hex.EncodeToString(got[:]), expected)
	}
	return nil
}
