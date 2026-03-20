package idx

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestSaveCookiesError(t *testing.T) {
	t.Run("write to invalid path returns error", func(t *testing.T) {
		cookies := []*http.Cookie{
			{Name: "test", Value: "value"},
		}

		err := SaveCookies("/nonexistent/dir/cookies.json", cookies)
		if err == nil {
			t.Fatal("SaveCookies() expected error for invalid path")
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

func TestSaveLoadCookiesExpires(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")

	expires := time.Date(2030, 6, 15, 12, 0, 0, 0, time.UTC)
	cookies := []*http.Cookie{
		{Name: "cf_clearance", Value: "abc", Domain: ".idx.co.id", Path: "/", Expires: expires},
		{Name: "session", Value: "xyz", Domain: ".idx.co.id", Path: "/"},
	}

	if err := SaveCookies(path, cookies); err != nil {
		t.Fatalf("SaveCookies() error: %v", err)
	}

	loaded, err := LoadCookies(path)
	if err != nil {
		t.Fatalf("LoadCookies() error: %v", err)
	}

	if !loaded[0].Expires.Equal(expires) {
		t.Errorf("cookie[0].Expires = %v, want %v", loaded[0].Expires, expires)
	}

	if !loaded[1].Expires.IsZero() {
		t.Errorf("cookie[1].Expires = %v, want zero", loaded[1].Expires)
	}
}

func TestCookiesValid(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) string
		want  bool
	}{
		{
			name: "valid cookies without expiry",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "cookies.json")
				cookies := []*http.Cookie{{Name: "test", Value: "val"}}
				if err := SaveCookies(path, cookies); err != nil {
					t.Fatalf("save: %v", err)
				}
				return path
			},
			want: true,
		},
		{
			name: "expired cookie",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "cookies.json")
				cookies := []*http.Cookie{{
					Name:    "test",
					Value:   "val",
					Expires: time.Now().Add(-1 * time.Hour),
				}}
				if err := SaveCookies(path, cookies); err != nil {
					t.Fatalf("save: %v", err)
				}
				return path
			},
			want: false,
		},
		{
			name: "future expiry is valid",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "cookies.json")
				cookies := []*http.Cookie{{
					Name:    "test",
					Value:   "val",
					Expires: time.Now().Add(24 * time.Hour),
				}}
				if err := SaveCookies(path, cookies); err != nil {
					t.Fatalf("save: %v", err)
				}
				return path
			},
			want: true,
		},
		{
			name: "missing file",
			setup: func(t *testing.T) string {
				t.Helper()
				return "/nonexistent/cookies.json"
			},
			want: false,
		},
		{
			name: "empty cookies",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "cookies.json")
				if err := os.WriteFile(path, []byte("[]"), 0o600); err != nil {
					t.Fatalf("write: %v", err)
				}
				return path
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			got := CookiesValid(path)
			if got != tt.want {
				t.Errorf("CookiesValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthTimeoutDuration(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got := authTimeoutDuration()
		if got != authTimeout {
			t.Errorf("got %v, want %v", got, authTimeout)
		}
	})

	t.Run("custom from env", func(t *testing.T) {
		t.Setenv("IDXLENS_AUTH_TIMEOUT", "60s")
		got := authTimeoutDuration()
		if got != 60*time.Second {
			t.Errorf("got %v, want 60s", got)
		}
	})

	t.Run("invalid env falls back to default", func(t *testing.T) {
		t.Setenv("IDXLENS_AUTH_TIMEOUT", "invalid")
		got := authTimeoutDuration()
		if got != authTimeout {
			t.Errorf("got %v, want %v", got, authTimeout)
		}
	})

	t.Run("negative duration falls back to default", func(t *testing.T) {
		t.Setenv("IDXLENS_AUTH_TIMEOUT", "-5s")
		got := authTimeoutDuration()
		if got != authTimeout {
			t.Errorf("got %v, want %v", got, authTimeout)
		}
	})
}
