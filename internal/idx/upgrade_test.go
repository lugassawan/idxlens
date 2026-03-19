package idx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLatestRelease(t *testing.T) {
	release := Release{
		TagName: "v1.2.3",
		Assets: []Asset{
			{Name: "idxlens_linux_amd64.tar.gz", DownloadURL: "https://example.com/linux"},
			{Name: "idxlens_darwin_arm64.tar.gz", DownloadURL: "https://example.com/darwin"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(release)
	}))
	defer srv.Close()

	got, err := latestReleaseFrom(context.Background(), srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("latestReleaseFrom() error: %v", err)
	}

	if got.TagName != "v1.2.3" {
		t.Errorf("TagName = %q, want %q", got.TagName, "v1.2.3")
	}

	if len(got.Assets) != 2 {
		t.Errorf("Assets count = %d, want 2", len(got.Assets))
	}
}

func TestLatestReleaseServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := latestReleaseFrom(context.Background(), srv.URL, srv.Client())
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestAssetName(t *testing.T) {
	name := AssetName()
	want := "idxlens_" + runtime.GOOS + "_" + runtime.GOARCH

	if name != want {
		t.Errorf("AssetName() = %q, want %q", name, want)
	}
}

func TestFindAsset(t *testing.T) {
	tests := []struct {
		name    string
		release *Release
		wantErr bool
	}{
		{
			name: "matching asset found",
			release: &Release{
				Assets: []Asset{
					{Name: "idxlens_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz", DownloadURL: "https://example.com/download"},
				},
			},
		},
		{
			name: "no matching asset",
			release: &Release{
				Assets: []Asset{
					{Name: "idxlens_windows_386.tar.gz", DownloadURL: "https://example.com/download"},
				},
			},
			wantErr: true,
		},
		{
			name:    "empty assets",
			release: &Release{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := FindAsset(tt.release)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("FindAsset() error: %v", err)
			}

			if !strings.Contains(asset.Name, AssetName()) {
				t.Errorf("asset name %q does not contain %q", asset.Name, AssetName())
			}
		})
	}
}

func TestDownloadAsset(t *testing.T) {
	content := "binary content"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(content))
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "idxlens")

	err := downloadAssetFrom(context.Background(), srv.URL, destPath, srv.Client())
	if err != nil {
		t.Fatalf("downloadAssetFrom() error: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if string(got) != content {
		t.Errorf("file content = %q, want %q", string(got), content)
	}

	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if info.Mode()&0o100 == 0 {
		t.Error("file is not executable")
	}
}

func TestDownloadAssetServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "idxlens")

	err := downloadAssetFrom(context.Background(), srv.URL, destPath, srv.Client())
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestCurrentBinaryPath(t *testing.T) {
	path, err := CurrentBinaryPath()
	if err != nil {
		t.Fatalf("CurrentBinaryPath() error: %v", err)
	}

	if path == "" {
		t.Error("path is empty")
	}
}
