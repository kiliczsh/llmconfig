package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MigrationResult summarises the outcome of one MigrateLegacyConfigs run.
type MigrationResult struct {
	Renamed  []string // basenames of files moved from .yaml → .llmc
	Skipped  []string // basenames left in place because both extensions were present
	Backups  []string // .yaml.bak files written alongside the renames
	BackupOK bool     // false if any backup write failed (rename was aborted for that file)
}

// HasChanges reports whether the run did any work.
func (r MigrationResult) HasChanges() bool {
	return len(r.Renamed) > 0 || len(r.Skipped) > 0
}

// MigrateLegacyConfigs renames every <name>.yaml file in dir to
// <name>.llmc, leaving a <name>.yaml.bak copy alongside as a manual
// undo path.
//
// If <name>.llmc already exists, the .yaml file is left in place and
// reported via Skipped — we never overwrite a file the user (or a
// previous migration) created. The backup is written *before* the
// rename so a crash mid-operation can be recovered from manually.
//
// Idempotent: a second call finds no .yaml files and is a no-op.
func MigrateLegacyConfigs(dir string) (MigrationResult, error) {
	res := MigrationResult{BackupOK: true}

	matches, err := filepath.Glob(filepath.Join(dir, "*"+ExtLegacy))
	if err != nil {
		return res, fmt.Errorf("migrate: glob %s: %w", dir, err)
	}

	for _, oldPath := range matches {
		base := filepath.Base(oldPath)
		nameNoExt := strings.TrimSuffix(base, ExtLegacy)
		newPath := filepath.Join(dir, nameNoExt+ExtPrimary)

		if _, err := os.Stat(newPath); err == nil {
			res.Skipped = append(res.Skipped, base)
			continue
		}

		bakPath := oldPath + ".bak"
		if err := copyFile(oldPath, bakPath); err != nil {
			res.BackupOK = false
			return res, fmt.Errorf("migrate: backup %s: %w", oldPath, err)
		}

		if err := os.Rename(oldPath, newPath); err != nil {
			// Roll back the backup we just wrote so we don't leave a
			// half-finished migration on disk.
			_ = os.Remove(bakPath)
			return res, fmt.Errorf("migrate: rename %s → %s: %w", oldPath, newPath, err)
		}
		res.Renamed = append(res.Renamed, base)
		res.Backups = append(res.Backups, filepath.Base(bakPath))
	}

	return res, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return nil
}
