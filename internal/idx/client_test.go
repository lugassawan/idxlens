package idx

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		c := New()

		if c.baseURL != defaultBaseURL {
			t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
		}

		if c.httpClient == nil {
			t.Error("httpClient is nil")
		}

		if len(c.cookies) != 0 {
			t.Errorf("cookies length = %d, want 0", len(c.cookies))
		}
	})

	t.Run("with base URL", func(t *testing.T) {
		url := "https://example.com"
		c := New(WithBaseURL(url))

		if c.baseURL != url {
			t.Errorf("baseURL = %q, want %q", c.baseURL, url)
		}
	})

	t.Run("with cookie file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "cookies.json")

		cookies := []*http.Cookie{
			{Name: "test", Value: "value", Domain: ".example.com", Path: "/"},
		}

		if err := SaveCookies(path, cookies); err != nil {
			t.Fatalf("SaveCookies() error: %v", err)
		}

		c := New(WithCookieFile(path))

		if len(c.cookies) != 1 {
			t.Fatalf("cookies length = %d, want 1", len(c.cookies))
		}

		if c.cookies[0].Name != "test" {
			t.Errorf("cookie name = %q, want %q", c.cookies[0].Name, "test")
		}
	})

	t.Run("with missing cookie file", func(t *testing.T) {
		c := New(WithCookieFile("/nonexistent/cookies.json"))

		if len(c.cookies) != 0 {
			t.Errorf("cookies length = %d, want 0", len(c.cookies))
		}
	})
}

func TestNewRequest(t *testing.T) {
	t.Run("injects cookies", func(t *testing.T) {
		c := New()
		c.cookies = []*http.Cookie{
			{Name: "cf_clearance", Value: "abc123"},
			{Name: "__cf_bm", Value: "xyz789"},
		}

		req, err := c.newRequest(http.MethodGet, "https://example.com/api")
		if err != nil {
			t.Fatalf("newRequest() error: %v", err)
		}

		cookies := req.Cookies()
		if len(cookies) != 2 {
			t.Fatalf("request cookies length = %d, want 2", len(cookies))
		}

		if cookies[0].Name != "cf_clearance" || cookies[0].Value != "abc123" {
			t.Errorf("first cookie = %s=%s, want cf_clearance=abc123", cookies[0].Name, cookies[0].Value)
		}
	})
}

func TestWithCookieFileIgnoresErrors(t *testing.T) {
	// Verify that a bad file doesn't crash but silently skips.
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	c := New(WithCookieFile(path))

	if len(c.cookies) != 0 {
		t.Errorf("cookies length = %d, want 0", len(c.cookies))
	}
}
