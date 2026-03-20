package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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
	want := runtime.GOOS + "_" + runtime.GOARCH

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
					{
						Name:        "idxlens_1.2.0_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz",
						DownloadURL: "https://example.com/download",
					},
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
	archive := createTestTarGz(t, binaryName, content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, binaryName)

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

	// Unix only — Windows doesn't have executable permission bits
	if runtime.GOOS != "windows" {
		if info.Mode()&0o100 == 0 {
			t.Error("file is not executable")
		}
	}
}

func TestDownloadAssetServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, binaryName)

	err := downloadAssetFrom(context.Background(), srv.URL, destPath, srv.Client())
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestLatestReleaseMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer srv.Close()

	_, err := latestReleaseFrom(context.Background(), srv.URL, srv.Client())
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}

	if !strings.Contains(err.Error(), "decode release") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "decode release")
	}
}

func TestLatestReleaseInvalidURL(t *testing.T) {
	_, err := latestReleaseFrom(context.Background(), "://bad-url", &http.Client{})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestDownloadAssetWriteError(t *testing.T) {
	archive := createTestTarGz(t, binaryName, "binary content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer srv.Close()

	// Write to a path inside a non-existent directory
	destPath := filepath.Join(t.TempDir(), "no-such-dir", "nested", binaryName)

	err := downloadAssetFrom(context.Background(), srv.URL, destPath, srv.Client())
	if err == nil {
		t.Fatal("expected error for write to non-existent directory")
	}
}

func TestDownloadAssetInvalidURL(t *testing.T) {
	err := downloadAssetFrom(context.Background(), "://bad-url", "/tmp/idxlens", &http.Client{})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestLatestReleaseCancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.0"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := latestReleaseFrom(ctx, srv.URL, srv.Client())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestDownloadAssetCancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := downloadAssetFrom(ctx, srv.URL, filepath.Join(t.TempDir(), "out"), srv.Client())
	if err == nil {
		t.Fatal("expected error for cancelled context")
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

func TestExtractBinary(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    string
		wantErr string
	}{
		{
			name: "valid archive with idxlens",
			data: createTestTarGz(t, binaryName, "hello binary"),
			want: "hello binary",
		},
		{
			name:    "archive without idxlens",
			data:    createTestTarGz(t, "other-file", "content"),
			wantErr: "idxlens binary not found in archive",
		},
		{
			name:    "invalid gzip data",
			data:    []byte("not gzip at all"),
			wantErr: "open gzip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractBinary(bytes.NewReader(tt.data))

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("extractBinary() error: %v", err)
			}

			var buf bytes.Buffer
			if _, err := buf.ReadFrom(got); err != nil {
				t.Fatalf("read result: %v", err)
			}

			if buf.String() != tt.want {
				t.Errorf("content = %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

func createTestTarGz(t *testing.T, filename, content string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: filename,
		Mode: 0o755,
		Size: int64(len(content)),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write tar header: %v", err)
	}

	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatalf("write tar content: %v", err)
	}

	tw.Close()
	gw.Close()

	return buf.Bytes()
}
