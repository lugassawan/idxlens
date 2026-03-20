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

		result, err := fetchRegistryFrom(context.Background(), srv.URL, "")
		if err != nil {
			t.Fatalf("fetchRegistryFrom() error: %v", err)
		}

		reg := result.Registry

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

		_, err := fetchRegistryFrom(context.Background(), srv.URL, "")
		if err == nil {
			t.Fatal("fetchRegistryFrom() expected error for server error")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer srv.Close()

		_, err := fetchRegistryFrom(context.Background(), srv.URL, "")
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
	_, err := fetchRegistryFrom(context.Background(), "://bad-url", "")
	if err == nil {
		t.Fatal("fetchRegistryFrom() expected error for invalid URL")
	}
}

func TestFetchRegistryFromConnectionRefused(t *testing.T) {
	// Use a URL with a port that refuses connections
	_, err := fetchRegistryFrom(context.Background(), "http://127.0.0.1:1/invalid", "")
	if err == nil {
		t.Fatal("fetchRegistryFrom() expected error for connection refused")
	}
}

func TestFetchRegistryConditional(t *testing.T) {
	t.Run("304 returns cached registry", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("IDXLENS_HOME", dir)

		reg := map[string]CompanyRegistry{
			"BBCA": {Name: "Bank Central Asia"},
		}
		regPath, err := RegistryPath()
		if err != nil {
			t.Fatalf("registry path: %v", err)
		}
		if err := SaveCachedRegistry(regPath, reg); err != nil {
			t.Fatalf("save cached: %v", err)
		}

		// Server returns 304 for matching ETag
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("If-None-Match") == `"test-etag"` {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			t.Error("expected If-None-Match header")
		}))
		defer srv.Close()

		// Save etag
		etagPath, err := ETagPath()
		if err != nil {
			t.Fatalf("etag path: %v", err)
		}
		if err := SaveETag(etagPath, `"test-etag"`); err != nil {
			t.Fatalf("save etag: %v", err)
		}

		// Use fetchRegistryFrom directly since FetchRegistryConditional uses hardcoded URL
		result, err := fetchRegistryFrom(context.Background(), srv.URL, `"test-etag"`)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if !result.NotModified {
			t.Error("expected NotModified")
		}
	})

	t.Run("200 returns new registry with ETag", func(t *testing.T) {
		payload := `{"BBCA": {"name": "BCA", "ir_page": "", "presentations": []}}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("ETag", `"new-etag"`)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(payload))
		}))
		defer srv.Close()

		result, err := fetchRegistryFrom(context.Background(), srv.URL, `"old-etag"`)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if result.NotModified {
			t.Error("expected new data, not NotModified")
		}
		if result.ETag != `"new-etag"` {
			t.Errorf("ETag = %q, want %q", result.ETag, `"new-etag"`)
		}
		if _, ok := result.Registry["BBCA"]; !ok {
			t.Error("registry missing BBCA")
		}
	})

	t.Run("no etag sends no If-None-Match", func(t *testing.T) {
		payload := `{"BBCA": {"name": "BCA", "ir_page": "", "presentations": []}}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("If-None-Match") != "" {
				t.Error("should not send If-None-Match with empty etag")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(payload))
		}))
		defer srv.Close()

		result, err := fetchRegistryFrom(context.Background(), srv.URL, "")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if result.Registry == nil {
			t.Error("expected registry data")
		}
	})
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

func TestETagPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IDXLENS_HOME", dir)

	path, err := ETagPath()
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	want := filepath.Join(dir, "registry.etag")
	if path != want {
		t.Errorf("ETagPath() = %q, want %q", path, want)
	}
}

func TestETagLoadSave(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.etag")

		if err := SaveETag(path, `"abc123"`); err != nil {
			t.Fatalf("save: %v", err)
		}

		got, err := LoadETag(path)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if got != `"abc123"` {
			t.Errorf("etag = %q, want %q", got, `"abc123"`)
		}
	})

	t.Run("load missing file", func(t *testing.T) {
		_, err := LoadETag("/nonexistent/path")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("save creates directories", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "sub", "dir", "test.etag")

		if err := SaveETag(path, `"test"`); err != nil {
			t.Fatalf("save: %v", err)
		}
	})

	t.Run("save to invalid path", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		err := SaveETag(filepath.Join(blocker, "sub", "test.etag"), `"test"`)
		if err == nil {
			t.Error("expected error")
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
