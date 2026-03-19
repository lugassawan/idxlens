package safefile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errReader is an io.Reader that always returns an error after reading some bytes.
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("simulated read error")
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		content  string
		wantErr  bool
		errMatch string
	}{
		{
			name: "happy path writes content",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "output.txt")
			},
			content: "hello world",
		},
		{
			name: "rename failure for non-existent parent",
			setup: func(t *testing.T) string {
				t.Helper()
				// Parent directory doesn't exist, so create tmp will fail
				return filepath.Join(t.TempDir(), "no-such-dir", "nested", "output.txt")
			},
			content:  "data",
			wantErr:  true,
			errMatch: "create file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destPath := tt.setup(t)
			err := Write(destPath, strings.NewReader(tt.content))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.errMatch != "" && !strings.Contains(err.Error(), tt.errMatch) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errMatch)
				}
				return
			}

			if err != nil {
				t.Fatalf("Write() error: %v", err)
			}

			got, readErr := os.ReadFile(destPath)
			if readErr != nil {
				t.Fatalf("ReadFile() error: %v", readErr)
			}

			if string(got) != tt.content {
				t.Errorf("content = %q, want %q", string(got), tt.content)
			}

			// Temp file should not remain
			if _, statErr := os.Stat(destPath + ".tmp"); !os.IsNotExist(statErr) {
				t.Error("temporary file was not cleaned up")
			}
		})
	}
}

func TestWriteRenameFailure(t *testing.T) {
	dir := t.TempDir()
	// Create a directory at the destination path — rename will fail
	// because you can't rename a file over a non-empty directory.
	destPath := filepath.Join(dir, "output")
	if err := os.MkdirAll(filepath.Join(destPath, "blocker"), 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := Write(destPath, strings.NewReader("data"))
	if err == nil {
		t.Fatal("expected error for rename over directory")
	}

	if !strings.Contains(err.Error(), "rename") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "rename")
	}

	// Temp file should be cleaned up
	if _, statErr := os.Stat(destPath + ".tmp"); !os.IsNotExist(statErr) {
		t.Error("temporary file was not cleaned up after rename error")
	}
}

func TestWriteReaderError(t *testing.T) {
	dir := t.TempDir()
	destPath := filepath.Join(dir, "output")

	err := Write(destPath, &errReader{})
	if err == nil {
		t.Fatal("expected error from failing reader")
	}

	if !strings.Contains(err.Error(), "write file") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "write file")
	}

	// Temp file should be cleaned up
	if _, statErr := os.Stat(destPath + ".tmp"); !os.IsNotExist(statErr) {
		t.Error("temporary file was not cleaned up after reader error")
	}

	// Dest file should not exist
	if _, statErr := os.Stat(destPath); !os.IsNotExist(statErr) {
		t.Error("destination file should not exist after error")
	}
}
