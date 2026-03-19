package idx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchRegistry(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		payload := `{
			"BBCA": {
				"name": "Bank Central Asia",
				"ir_page": "https://example.com/ir",
				"presentations": [
					{"url": "https://example.com/q1.pdf", "title": "Q1 2024", "period": "Q1", "year": 2024}
				]
			}
		}`

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(payload))
		}))
		defer srv.Close()

		reg, err := fetchRegistryFrom(context.Background(), srv.URL)
		if err != nil {
			t.Fatalf("fetchRegistryFrom() error: %v", err)
		}

		entry, ok := reg["BBCA"]
		if !ok {
			t.Fatal("registry missing BBCA entry")
		}

		if entry.Name != "Bank Central Asia" {
			t.Errorf("name = %q, want %q", entry.Name, "Bank Central Asia")
		}

		if len(entry.Presentations) != 1 {
			t.Fatalf("got %d presentations, want 1", len(entry.Presentations))
		}

		p := entry.Presentations[0]
		if p.Year != 2024 {
			t.Errorf("year = %d, want 2024", p.Year)
		}

		if p.Period != "Q1" {
			t.Errorf("period = %q, want %q", p.Period, "Q1")
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		_, err := fetchRegistryFrom(context.Background(), srv.URL)
		if err == nil {
			t.Fatal("fetchRegistryFrom() expected error for server error")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer srv.Close()

		_, err := fetchRegistryFrom(context.Background(), srv.URL)
		if err == nil {
			t.Fatal("fetchRegistryFrom() expected error for invalid JSON")
		}
	})
}

func TestFetchRegistryCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := FetchRegistry(ctx)
	if err == nil {
		t.Fatal("FetchRegistry() expected error for cancelled context")
	}
}

func TestFetchRegistryFromInvalidURL(t *testing.T) {
	_, err := fetchRegistryFrom(context.Background(), "://bad-url")
	if err == nil {
		t.Fatal("fetchRegistryFrom() expected error for invalid URL")
	}
}

func TestFetchRegistryFromConnectionRefused(t *testing.T) {
	// Use a URL with a port that refuses connections
	_, err := fetchRegistryFrom(context.Background(), "http://127.0.0.1:1/invalid")
	if err == nil {
		t.Fatal("fetchRegistryFrom() expected error for connection refused")
	}
}

func TestRegistryPath(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("IDXLENS_HOME", dir)

		path, err := RegistryPath()
		if err != nil {
			t.Fatalf("RegistryPath() error: %v", err)
		}

		want := filepath.Join(dir, "registry.json")
		if path != want {
			t.Errorf("RegistryPath() = %q, want %q", path, want)
		}
	})

	t.Run("home error", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		t.Setenv("IDXLENS_HOME", filepath.Join(blocker, "sub"))

		_, err := RegistryPath()
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCachedRegistry(t *testing.T) {
	t.Run("save and load round-trip", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "registry.json")

		reg := map[string]CompanyRegistry{
			"TLKM": {
				Name:   "Telkom Indonesia",
				IRPage: "https://example.com/ir",
				Presentations: []PresentationEntry{
					{URL: "https://example.com/q1.pdf", Title: "Q1 2024", Period: "Q1", Year: 2024},
				},
			},
		}

		if err := SaveCachedRegistry(path, reg); err != nil {
			t.Fatalf("SaveCachedRegistry() error: %v", err)
		}

		loaded, err := LoadCachedRegistry(path)
		if err != nil {
			t.Fatalf("LoadCachedRegistry() error: %v", err)
		}

		entry, ok := loaded["TLKM"]
		if !ok {
			t.Fatal("loaded registry missing TLKM entry")
		}

		if entry.Name != "Telkom Indonesia" {
			t.Errorf("name = %q, want %q", entry.Name, "Telkom Indonesia")
		}

		if len(entry.Presentations) != 1 {
			t.Fatalf("got %d presentations, want 1", len(entry.Presentations))
		}

		if entry.Presentations[0].Year != 2024 {
			t.Errorf("year = %d, want 2024", entry.Presentations[0].Year)
		}
	})

	t.Run("load missing file returns error", func(t *testing.T) {
		_, err := LoadCachedRegistry("/nonexistent/path/registry.json")
		if err == nil {
			t.Fatal("LoadCachedRegistry() expected error for missing file")
		}
	})

	t.Run("load invalid JSON returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.json")

		if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
			t.Fatalf("write file: %v", err)
		}

		_, err := LoadCachedRegistry(path)
		if err == nil {
			t.Fatal("LoadCachedRegistry() expected error for invalid JSON")
		}
	})

	t.Run("save to invalid path returns error", func(t *testing.T) {
		// Path that can't be created (file exists where directory needed)
		tmpFile := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(tmpFile, []byte("x"), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		reg := map[string]CompanyRegistry{}
		err := SaveCachedRegistry(filepath.Join(tmpFile, "sub", "registry.json"), reg)
		if err == nil {
			t.Fatal("expected error for invalid path")
		}
	})

	t.Run("save creates parent directories", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "sub", "dir", "registry.json")

		reg := map[string]CompanyRegistry{}
		if err := SaveCachedRegistry(path, reg); err != nil {
			t.Fatalf("SaveCachedRegistry() error: %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("registry file should exist after save")
		}
	})
}
