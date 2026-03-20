package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lugassawan/idxlens/internal/safefile"
)

const (
	releaseEndpoint = "https://api.github.com/repos/lugassawan/idxlens/releases/latest"
	binaryName      = "idxlens"
)

// Release represents a GitHub release.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a downloadable file in a GitHub release.
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// LatestRelease fetches the latest release from GitHub.
func LatestRelease(ctx context.Context) (*Release, error) {
	return latestReleaseFrom(ctx, releaseEndpoint, &http.Client{})
}

// AssetName returns the OS/architecture suffix used to match release assets.
func AssetName() string {
	return fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
}

// FindAsset finds the matching asset for the current platform in a release.
func FindAsset(release *Release) (*Asset, error) {
	name := AssetName()

	for i, a := range release.Assets {
		if strings.Contains(a.Name, name) {
			return &release.Assets[i], nil
		}
	}

	return nil, fmt.Errorf("no asset found for %s", name)
}

// DownloadAsset downloads a release asset to the given path.
func DownloadAsset(ctx context.Context, url, destPath string) error {
	return downloadAssetFrom(ctx, url, destPath, &http.Client{})
}

// CurrentBinaryPath returns the path of the currently running binary.
func CurrentBinaryPath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks: %w", err)
	}

	return resolved, nil
}

func latestReleaseFrom(ctx context.Context, url string, hc *http.Client) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	//nolint:gosec // URL is either the hardcoded GitHub API endpoint or a test server
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch latest release: status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}

	return &release, nil
}

func downloadAssetFrom(ctx context.Context, url, destPath string, hc *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	//nolint:gosec // URL comes from GitHub API response or test server
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("download asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download asset: status %d", resp.StatusCode)
	}

	binary, err := extractBinary(resp.Body)
	if err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}

	if err := safefile.Write(destPath, binary); err != nil {
		return fmt.Errorf("write asset: %w", err)
	}

	if err := os.Chmod(destPath, 0o755); err != nil { //nolint:gosec // binary must be executable
		return fmt.Errorf("chmod asset: %w", err)
	}

	return nil
}

func extractBinary(r io.Reader) (io.Reader, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}

		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == binaryName {
			const maxBinarySize = 256 << 20 // 256 MiB
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, io.LimitReader(tr, maxBinarySize)); err != nil {
				return nil, fmt.Errorf("read binary: %w", err)
			}

			return &buf, nil
		}
	}

	return nil, errors.New(binaryName + " binary not found in archive")
}
