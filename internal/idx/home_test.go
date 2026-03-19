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

func TestCookiePath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envHome, dir)

	got, err := CookiePath()
	if err != nil {
		t.Fatalf("CookiePath() error: %v", err)
	}

	expected := filepath.Join(dir, cookieFile)
	if got != expected {
		t.Errorf("CookiePath() = %q, want %q", got, expected)
	}
}

func TestCompaniesPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envHome, dir)

	got, err := CompaniesPath()
	if err != nil {
		t.Fatalf("CompaniesPath() error: %v", err)
	}

	expected := filepath.Join(dir, companyFile)
	if got != expected {
		t.Errorf("CompaniesPath() = %q, want %q", got, expected)
	}
}
