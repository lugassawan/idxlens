package idx

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCookies(t *testing.T) {
	t.Run("round-trip save and load", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "cookies.json")

		cookies := []*http.Cookie{
			{Name: "cf_clearance", Value: "abc123", Domain: ".idx.co.id", Path: "/"},
			{Name: "__cf_bm", Value: "xyz789", Domain: ".idx.co.id", Path: "/"},
		}

		if err := SaveCookies(path, cookies); err != nil {
			t.Fatalf("SaveCookies() error: %v", err)
		}

		loaded, err := LoadCookies(path)
		if err != nil {
			t.Fatalf("LoadCookies() error: %v", err)
		}

		if len(loaded) != len(cookies) {
			t.Fatalf("loaded %d cookies, want %d", len(loaded), len(cookies))
		}

		for i, c := range loaded {
			if c.Name != cookies[i].Name {
				t.Errorf("cookie[%d].Name = %q, want %q", i, c.Name, cookies[i].Name)
			}

			if c.Value != cookies[i].Value {
				t.Errorf("cookie[%d].Value = %q, want %q", i, c.Value, cookies[i].Value)
			}

			if c.Domain != cookies[i].Domain {
				t.Errorf("cookie[%d].Domain = %q, want %q", i, c.Domain, cookies[i].Domain)
			}

			if c.Path != cookies[i].Path {
				t.Errorf("cookie[%d].Path = %q, want %q", i, c.Path, cookies[i].Path)
			}
		}
	})

	t.Run("empty cookies", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "cookies.json")

		if err := SaveCookies(path, []*http.Cookie{}); err != nil {
			t.Fatalf("SaveCookies() error: %v", err)
		}

		loaded, err := LoadCookies(path)
		if err != nil {
			t.Fatalf("LoadCookies() error: %v", err)
		}

		if len(loaded) != 0 {
			t.Errorf("loaded %d cookies, want 0", len(loaded))
		}
	})
}

func TestLoadCookies(t *testing.T) {
	t.Run("missing file returns error", func(t *testing.T) {
		_, err := LoadCookies("/nonexistent/cookies.json")
		if err == nil {
			t.Fatal("LoadCookies() expected error for missing file")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.json")

		if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
			t.Fatalf("write file: %v", err)
		}

		_, err := LoadCookies(path)
		if err == nil {
			t.Fatal("LoadCookies() expected error for invalid JSON")
		}
	})
}
