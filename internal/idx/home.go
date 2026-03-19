package idx

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	envHome     = "IDXLENS_HOME"
	defaultDir  = ".idxlens"
	dataSubdir  = "data"
	cookieFile  = "cookies.json"
	companyFile = "companies.json"
)

// Home returns the IDXLENS_HOME directory path.
// Resolution order: IDXLENS_HOME env var, then ~/.idxlens fallback.
// The directory is created if it doesn't exist.
func Home() (string, error) {
	dir := os.Getenv(envHome)
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		dir = filepath.Join(home, defaultDir)
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create idxlens home %s: %w", dir, err)
	}

	return dir, nil
}

// DataDir returns the path to the data subdirectory within IDXLENS_HOME.
func DataDir() (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, dataSubdir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create data directory %s: %w", dir, err)
	}

	return dir, nil
}

// CookiePath returns the path to the cookies.json file within IDXLENS_HOME.
func CookiePath() (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, cookieFile), nil
}

// CompaniesPath returns the path to the companies.json file within IDXLENS_HOME.
func CompaniesPath() (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, companyFile), nil
}
