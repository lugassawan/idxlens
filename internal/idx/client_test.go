package idx

import (
	"context"
	"net/http"
	"testing"
	"time"
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

	t.Run("with cookies", func(t *testing.T) {
		cookies := []*http.Cookie{
			{Name: "test", Value: "value", Domain: ".example.com", Path: "/"},
		}

		c := New(WithCookies(cookies))

		if len(c.cookies) != 1 {
			t.Fatalf("cookies length = %d, want 1", len(c.cookies))
		}

		if c.cookies[0].Name != "test" {
			t.Errorf("cookie name = %q, want %q", c.cookies[0].Name, "test")
		}
	})
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 99 * time.Second}
	c := New(WithHTTPClient(custom))

	if c.httpClient != custom {
		t.Error("WithHTTPClient did not set the custom HTTP client")
	}

	if c.httpClient.Timeout != 99*time.Second {
		t.Errorf("timeout = %v, want 99s", c.httpClient.Timeout)
	}
}

func TestNewRequest(t *testing.T) {
	t.Run("injects cookies", func(t *testing.T) {
		c := New()
		c.cookies = []*http.Cookie{
			{Name: "cf_clearance", Value: "abc123"},
			{Name: "__cf_bm", Value: "xyz789"},
		}

		req, err := c.newRequest(context.Background(), http.MethodGet, "https://example.com/api")
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
