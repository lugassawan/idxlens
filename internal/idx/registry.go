package idx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const registryURL = "https://raw.githubusercontent.com/lugassawan/idxlens/main/registry/presentations.json"

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

// FetchRegistry downloads the presentation registry from GitHub.
func FetchRegistry(ctx context.Context) (map[string]CompanyRegistry, error) {
	return fetchRegistryFrom(ctx, registryURL)
}

// RegistryPath returns the path to the cached registry file within IDXLENS_HOME.
func RegistryPath() (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "registry.json"), nil
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

func fetchRegistryFrom(ctx context.Context, url string) (map[string]CompanyRegistry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch registry: %w", err)
	}

	//nolint:gosec // URL is the hardcoded GitHub raw registry URL, not user input
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch registry: %w", err)
	}
	defer resp.Body.Close()

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

	return registry, nil
}
