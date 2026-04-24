// Package archive reads and writes llmconfig model bundles (.llmcpkg).
//
// A bundle is an uncompressed POSIX tar containing:
//
//	manifest.yaml          -> list of entries in this bundle
//	configs/<name>.yaml    -> one per entry
//	models/<file>          -> the cached GGUF/etc. (optional per entry)
//
// Compression is intentionally omitted: GGUF files are already quantised,
// so gzip spends minutes of CPU for <1% savings on large models. `tar -xf
// foo.llmcpkg` works on any POSIX system without llmconfig.
package archive

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// ManifestName is the fixed path of the manifest inside a bundle.
	ManifestName = "manifest.yaml"
	// FormatVersion is the currently-supported manifest schema.
	FormatVersion = 1
	// ConfigsDir and ModelsDir are the top-level directories inside a bundle.
	ConfigsDir = "configs"
	ModelsDir  = "models"
)

// Manifest describes the contents of a bundle.
type Manifest struct {
	Version    int       `yaml:"version"`
	ExportedAt time.Time `yaml:"exported_at"`
	ExportedBy string    `yaml:"exported_by"`
	Entries    []Entry   `yaml:"entries"`
}

// Entry is a single model bundled into the archive.
type Entry struct {
	Name      string `yaml:"name"`
	Config    string `yaml:"config"`
	ModelFile string `yaml:"model_file,omitempty"`
	Source    string `yaml:"source,omitempty"`
	Size      int64  `yaml:"size,omitempty"`
}

// CreateEntry is the input to Create — it points at files on disk to pull
// into a new bundle.
type CreateEntry struct {
	Name       string
	ConfigPath string
	ModelPath  string // may be empty if the model isn't cached locally
	Source     string
}

// ProgressFunc, if non-nil, is invoked periodically while a large file is
// being written or extracted. bytes is cumulative for the current file,
// total is the file size (0 if unknown).
type ProgressFunc func(entry string, bytes, total int64)

// Create writes a new bundle at outPath containing the given entries. Each
// model file is streamed (not buffered) so archives scale to tens of GB.
func Create(outPath string, entries []CreateEntry, exportedBy string, onProgress ProgressFunc) error {
	if len(entries) == 0 {
		return fmt.Errorf("archive: no entries")
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("archive: create %s: %w", outPath, err)
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	now := time.Now().UTC()
	manifest := Manifest{
		Version:    FormatVersion,
		ExportedAt: now,
		ExportedBy: exportedBy,
	}

	// First pass: build manifest and stat files so Size fields are populated.
	for _, e := range entries {
		me := Entry{
			Name:   e.Name,
			Config: path.Join(ConfigsDir, e.Name+".yaml"),
			Source: e.Source,
		}
		if e.ModelPath != "" {
			info, err := os.Stat(e.ModelPath)
			if err != nil {
				return fmt.Errorf("archive: stat model %s: %w", e.ModelPath, err)
			}
			me.ModelFile = path.Join(ModelsDir, filepath.Base(e.ModelPath))
			me.Size = info.Size()
		}
		manifest.Entries = append(manifest.Entries, me)
	}

	// Write manifest first so `tar -tf` shows it up top.
	manifestBytes, err := yaml.Marshal(&manifest)
	if err != nil {
		return fmt.Errorf("archive: marshal manifest: %w", err)
	}
	if err := writeTarBytes(tw, ManifestName, manifestBytes, now); err != nil {
		return err
	}

	// Pre-create the two top-level directories so readers that list dirs
	// (some Windows tar tools) show them.
	for _, d := range []string{ConfigsDir + "/", ModelsDir + "/"} {
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     d,
			Mode:     0755,
			ModTime:  now,
		}); err != nil {
			return fmt.Errorf("archive: dir header: %w", err)
		}
	}

	// Entries.
	for i, e := range entries {
		me := manifest.Entries[i]

		// Config (small — read fully into memory).
		cfgBytes, err := os.ReadFile(e.ConfigPath)
		if err != nil {
			return fmt.Errorf("archive: read config %s: %w", e.ConfigPath, err)
		}
		if err := writeTarBytes(tw, me.Config, cfgBytes, now); err != nil {
			return err
		}

		// Model file (can be huge — stream it).
		if me.ModelFile != "" {
			if err := writeTarStream(tw, me.ModelFile, e.ModelPath, me.Size, now, onProgress); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeTarBytes(tw *tar.Writer, name string, body []byte, mtime time.Time) error {
	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     int64(len(body)),
		Mode:     0644,
		ModTime:  mtime,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("archive: header %s: %w", name, err)
	}
	if _, err := tw.Write(body); err != nil {
		return fmt.Errorf("archive: write %s: %w", name, err)
	}
	return nil
}

func writeTarStream(tw *tar.Writer, name, srcPath string, size int64, mtime time.Time, onProgress ProgressFunc) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("archive: open %s: %w", srcPath, err)
	}
	defer f.Close()

	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     size,
		Mode:     0644,
		ModTime:  mtime,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("archive: header %s: %w", name, err)
	}

	buf := make([]byte, 1<<20) // 1 MiB
	var copied int64
	lastReport := time.Now()
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			if _, werr := tw.Write(buf[:n]); werr != nil {
				return fmt.Errorf("archive: write %s: %w", name, werr)
			}
			copied += int64(n)
			if onProgress != nil && time.Since(lastReport) > 200*time.Millisecond {
				onProgress(name, copied, size)
				lastReport = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("archive: read %s: %w", srcPath, readErr)
		}
	}
	if onProgress != nil {
		onProgress(name, copied, size)
	}
	return nil
}

// Open parses the manifest of an archive without extracting any files.
func Open(archivePath string) (*Manifest, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("archive: open %s: %w", archivePath, err)
	}
	defer f.Close()

	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("archive: no %s found", ManifestName)
		}
		if err != nil {
			return nil, fmt.Errorf("archive: read header: %w", err)
		}
		if hdr.Name == ManifestName {
			raw, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("archive: read manifest: %w", err)
			}
			var m Manifest
			if err := yaml.Unmarshal(raw, &m); err != nil {
				return nil, fmt.Errorf("archive: parse manifest: %w", err)
			}
			if m.Version != FormatVersion {
				return nil, fmt.Errorf("archive: unsupported manifest version %d (expected %d)", m.Version, FormatVersion)
			}
			return &m, nil
		}
	}
}

// ExtractResult records what Extract did.
type ExtractResult struct {
	Installed []string // names successfully installed
	Skipped   []string // names skipped due to existing files (no --overwrite)
}

// Extract unpacks an archive into configDir and cacheDir. Each entry in
// the manifest is placed according to Entry.Config and Entry.ModelFile.
// When overwrite is false, entries whose config or model already exists
// are skipped and reported in ExtractResult.Skipped.
func Extract(archivePath, configDir, modelsDir string, overwrite bool, onProgress ProgressFunc) (*ExtractResult, error) {
	manifest, err := Open(archivePath)
	if err != nil {
		return nil, err
	}

	// Decide per-entry whether to extract. Installed is populated only as
	// each entry's files successfully land on disk, not up front — a
	// mid-stream failure otherwise claims models were installed when they
	// weren't.
	extract := map[string]bool{}
	result := &ExtractResult{}
	for _, e := range manifest.Entries {
		cfgDest := filepath.Join(configDir, e.Name+".yaml")
		conflict := false
		if _, err := os.Stat(cfgDest); err == nil && !overwrite {
			conflict = true
		}
		if e.ModelFile != "" {
			modelDest := filepath.Join(modelsDir, path.Base(e.ModelFile))
			if _, err := os.Stat(modelDest); err == nil && !overwrite {
				conflict = true
			}
		}
		if conflict {
			result.Skipped = append(result.Skipped, e.Name)
			continue
		}
		extract[e.Name] = true
	}
	installed := map[string]bool{}

	// Build a name-lookup so we can map archive paths back to entries.
	entryByConfig := map[string]*Entry{}
	entryByModel := map[string]*Entry{}
	for i := range manifest.Entries {
		e := &manifest.Entries[i]
		entryByConfig[e.Config] = e
		if e.ModelFile != "" {
			entryByModel[e.ModelFile] = e
		}
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("archive: mkdir %s: %w", configDir, err)
	}
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return nil, fmt.Errorf("archive: mkdir %s: %w", modelsDir, err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("archive: open %s: %w", archivePath, err)
	}
	defer f.Close()

	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("archive: read header: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if hdr.Name == ManifestName {
			continue
		}

		var dest, destRoot, progressLabel, entryName string
		if e, ok := entryByConfig[hdr.Name]; ok {
			if !extract[e.Name] {
				continue
			}
			dest = filepath.Join(configDir, e.Name+".yaml")
			destRoot = configDir
			progressLabel = hdr.Name
			entryName = e.Name
		} else if e, ok := entryByModel[hdr.Name]; ok {
			if !extract[e.Name] {
				continue
			}
			dest = filepath.Join(modelsDir, path.Base(e.ModelFile))
			destRoot = modelsDir
			progressLabel = hdr.Name
			entryName = e.Name
		} else {
			// Unknown entry in tar — ignore rather than error, forward compat.
			continue
		}

		// Defence-in-depth: verify the computed dest never escapes its
		// intended root. Covers manifest-driven traversal (the entry map
		// keys come from the bundle's own manifest) on both Unix and
		// Windows, where "\" and drive prefixes need active guarding.
		if !isUnder(dest, destRoot) {
			return result, fmt.Errorf("archive: unsafe path %q", hdr.Name)
		}

		if err := extractOne(tr, dest, progressLabel, hdr.Size, onProgress); err != nil {
			return result, err
		}
		installed[entryName] = true
	}

	for _, e := range manifest.Entries {
		if installed[e.Name] {
			result.Installed = append(result.Installed, e.Name)
		}
	}
	return result, nil
}

// isUnder reports whether path is located inside root, using cleaned
// absolute paths. Returns false on any error so callers fail safe.
func isUnder(path, root string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && rel != ".."
}

func extractOne(tr *tar.Reader, dest, label string, size int64, onProgress ProgressFunc) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("archive: mkdir %s: %w", filepath.Dir(dest), err)
	}
	tmp := dest + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("archive: create %s: %w", tmp, err)
	}

	buf := make([]byte, 1<<20)
	var copied int64
	lastReport := time.Now()
	for {
		n, readErr := tr.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				out.Close()
				_ = os.Remove(tmp)
				return fmt.Errorf("archive: write %s: %w", tmp, werr)
			}
			copied += int64(n)
			if onProgress != nil && time.Since(lastReport) > 200*time.Millisecond {
				onProgress(label, copied, size)
				lastReport = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			out.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("archive: read: %w", readErr)
		}
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("archive: close %s: %w", tmp, err)
	}
	if onProgress != nil {
		onProgress(label, copied, size)
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("archive: rename %s: %w", dest, err)
	}
	return nil
}
