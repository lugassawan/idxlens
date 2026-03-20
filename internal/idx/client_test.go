package idx

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
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

func TestNewAuthenticatedClient(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, dir string)
		wantErr    bool
		wantClient bool
	}{
		{
			name: "no cookies file",
			setup: func(t *testing.T, dir string) {
				t.Helper()
			},
			wantErr: true,
		},
		{
			name: "valid empty cookies",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				err := os.WriteFile(filepath.Join(dir, "cookies.json"), []byte("[]"), 0o600)
				if err != nil {
					t.Fatalf("write cookies.json: %v", err)
				}
			},
			wantClient: true,
		},
		{
			name: "valid cookies",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				data := `[{"name":"cf_clearance","value":"abc123","domain":".idx.co.id","path":"/"}]`
				err := os.WriteFile(filepath.Join(dir, "cookies.json"), []byte(data), 0o600)
				if err != nil {
					t.Fatalf("write cookies.json: %v", err)
				}
			},
			wantClient: true,
		},
		{
			name: "expired cookies",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				data := `[{"name":"cf_clearance","value":"abc","domain":".idx.co.id","path":"/","expires":"2020-01-01T00:00:00Z"}]`
				err := os.WriteFile(filepath.Join(dir, "cookies.json"), []byte(data), 0o600)
				if err != nil {
					t.Fatalf("write cookies.json: %v", err)
				}
			},
			wantErr: true,
		},
	}

	t.Run("cookie path error", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		t.Setenv("IDXLENS_HOME", filepath.Join(blocker, "sub"))

		_, err := NewAuthenticatedClient()
		if err == nil {
			t.Fatal("expected error when cookie path fails")
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("IDXLENS_HOME", dir)
			tt.setup(t, dir)

			client, err := NewAuthenticatedClient()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantClient && client == nil {
				t.Fatal("expected non-nil client, got nil")
			}
		})
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
