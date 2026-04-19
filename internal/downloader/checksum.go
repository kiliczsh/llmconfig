package downloader

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

func verifyChecksum(path, expected string) error {
	// expected format: "sha256:<hex>"
	parts := strings.SplitN(expected, ":", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return fmt.Errorf("checksum: unsupported format %q (expected sha256:<hex>)", expected)
	}
	expectedHex := parts[1]

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("checksum: open: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("checksum: hash: %w", err)
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != strings.ToLower(expectedHex) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHex, got)
	}
	return nil
}
