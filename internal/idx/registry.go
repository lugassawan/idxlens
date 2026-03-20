package idx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	registryURL       = "https://raw.githubusercontent.com/lugassawan/idxlens/main/registry/presentations.json"
	registryFile      = "registry.json"
	etagFile          = "registry.etag"
	headerIfNoneMatch = "If-None-Match"
)

// PresentationEntry represents a single company presentation document.
type PresentationEntry struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Period string `json:"period"`
	Year   int    `json:"year"`
}

// CompanyRegistry holds presentation metadata for a company.
type CompanyRegistry struct {
	Name          string              `json:"name"`
	IRPage        string              `json:"ir_page"`
	Presentations []PresentationEntry `json:"presentations"`
}

// fetchResult holds the result of a registry fetch.
type fetchResult struct {
	Registry    map[string]CompanyRegistry
	ETag        string
	NotModified bool
}

// FetchRegistry downloads the presentation registry from GitHub.
func FetchRegistry(ctx context.Context) (map[string]CompanyRegistry, error) {
	result, err := fetchRegistryFrom(ctx, registryURL, "")
	if err != nil {
		return nil, err
	}

	return result.Registry, nil
}

// FetchRegistryConditional fetches the registry with ETag caching.
// If the registry hasn't changed (304), it returns the cached version.
func FetchRegistryConditional(ctx context.Context) (map[string]CompanyRegistry, error) {
	etagPath, err := ETagPath()
	if err != nil {
		// Fall back to unconditional fetch
		return FetchRegistry(ctx)
	}

	etag, _ := LoadETag(etagPath) //nolint:errcheck // missing etag is not an error

	result, err := fetchRegistryFrom(ctx, registryURL, etag)
	if err != nil {
		return nil, err
	}

	if result.NotModified {
		regPath, err := RegistryPath()
		if err != nil {
			return nil, fmt.Errorf("resolve registry path: %w", err)
		}

		return LoadCachedRegistry(regPath)
	}

	// Save the new registry and ETag
	if result.ETag != "" {
		_ = SaveETag(etagPath, result.ETag) // best-effort
	}

	regPath, err := RegistryPath()
	if err == nil {
		_ = SaveCachedRegistry(regPath, result.Registry) // best-effort
	}

	return result.Registry, nil
}

// RegistryPath returns the path to the cached registry file within IDXLENS_HOME.
func RegistryPath() (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, registryFile), nil
}

// ETagPath returns the path to the cached ETag file.
func ETagPath() (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, etagFile), nil
}

// LoadCachedRegistry reads a presentation registry from a local JSON file.
func LoadCachedRegistry(path string) (map[string]CompanyRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load cached registry: %w", err)
	}

	var registry map[string]CompanyRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("load cached registry: unmarshal: %w", err)
	}

	return registry, nil
}

// SaveCachedRegistry writes a presentation registry to a local JSON file.
func SaveCachedRegistry(path string, reg map[string]CompanyRegistry) error {
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("save cached registry: marshal: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("save cached registry: create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("save cached registry: write file: %w", err)
	}

	return nil
}

// LoadETag reads the stored ETag from the given path.
func LoadETag(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// SaveETag writes the ETag to the given path.
func SaveETag(path, etag string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("save etag: create directory: %w", err)
	}

	return os.WriteFile(path, []byte(etag), 0o600)
}

func fetchRegistryFrom(ctx context.Context, url, etag string) (*fetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch registry: %w", err)
	}

	if etag != "" {
		req.Header.Set(headerIfNoneMatch, etag)
	}

	//nolint:gosec // URL is the hardcoded GitHub raw registry URL, not user input
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return &fetchResult{NotModified: true}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch registry: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch registry: read body: %w", err)
	}

	var registry map[string]CompanyRegistry
	if err := json.Unmarshal(body, &registry); err != nil {
		return nil, fmt.Errorf("fetch registry: unmarshal: %w", err)
	}

	return &fetchResult{
		Registry: registry,
		ETag:     resp.Header.Get("ETag"),
	}, nil
}
