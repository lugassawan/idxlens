package idx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHome(t *testing.T) {
	t.Run("uses IDXLENS_HOME env var", func(t *testing.T) {
		dir := t.TempDir()
		expected := filepath.Join(dir, "custom-home")

		t.Setenv(envHome, expected)

		got, err := Home()
		if err != nil {
			t.Fatalf("Home() error: %v", err)
		}

		if got != expected {
			t.Errorf("Home() = %q, want %q", got, expected)
		}

		info, err := os.Stat(expected)
		if err != nil {
			t.Fatalf("directory not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("Home() path is not a directory")
		}
	})

	t.Run("falls back to ~/.idxlens", func(t *testing.T) {
		tmpHome := t.TempDir()

		t.Setenv(envHome, "")
		t.Setenv("HOME", tmpHome)
		t.Setenv("USERPROFILE", tmpHome) // Windows support

		got, err := Home()
		if err != nil {
			t.Fatalf("Home() error: %v", err)
		}

		expected := filepath.Join(tmpHome, defaultDir)
		if got != expected {
			t.Errorf("Home() = %q, want %q", got, expected)
		}
	})
}

func TestDataDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envHome, dir)

	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() error: %v", err)
	}

	expected := filepath.Join(dir, dataSubdir)
	if got != expected {
		t.Errorf("DataDir() = %q, want %q", got, expected)
	}

	info, err := os.Stat(expected)
	if err != nil {
		t.Fatalf("data directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("DataDir() path is not a directory")
	}
}

func TestHomeMkdirAllError(t *testing.T) {
	// Set IDXLENS_HOME to a path where mkdir will fail (file exists as non-dir)
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Setenv(envHome, filepath.Join(blocker, "sub"))

	_, err := Home()
	if err == nil {
		t.Fatal("expected error when MkdirAll fails")
	}
}

func TestDataDirMkdirAllError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envHome, dir)

	// Create a file where the data directory should be
	blocker := filepath.Join(dir, dataSubdir)
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := DataDir()
	if err == nil {
		t.Fatal("expected error when data MkdirAll fails")
	}
}

func TestHomeDerivedPaths(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() (string, error)
		wantFile string
	}{
		{"CookiePath", CookiePath, cookieFile},
		{"CompaniesPath", CompaniesPath, companyFile},
	}

	for _, tt := range tests {
		t.Run(tt.name+" success", func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv(envHome, dir)

			got, err := tt.fn()
			if err != nil {
				t.Fatalf("%s() error: %v", tt.name, err)
			}

			expected := filepath.Join(dir, tt.wantFile)
			if got != expected {
				t.Errorf("%s() = %q, want %q", tt.name, got, expected)
			}
		})

		t.Run(tt.name+" home error", func(t *testing.T) {
			blocker := filepath.Join(t.TempDir(), "blocker")
			if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
				t.Fatalf("setup: %v", err)
			}

			t.Setenv(envHome, filepath.Join(blocker, "sub"))

			_, err := tt.fn()
			if err == nil {
				t.Fatalf("%s() expected error", tt.name)
			}
		})
	}
}
