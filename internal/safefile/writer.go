package safefile

import (
	"fmt"
	"io"
	"os"
)

// Write atomically writes data from r to destPath via a temporary file.
// It writes to a .tmp file first, calls fsync, then renames for crash safety.
func Write(destPath string, r io.Reader) error {
	tmpPath := destPath + ".tmp"

	if err := writeFile(tmpPath, r); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

func writeFile(path string, r io.Reader) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}

	return nil
}
