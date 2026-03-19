package service

import (
	"context"
	"fmt"

	"github.com/lugassawan/idxlens/internal/idx"
)

// DefaultRegistryProvider loads the registry from local cache or GitHub.
type DefaultRegistryProvider struct{}

// Registry returns the presentation registry, loading from cache first
// and falling back to fetching from GitHub.
func (d *DefaultRegistryProvider) Registry(ctx context.Context) (map[string]idx.CompanyRegistry, error) {
	regPath, err := idx.RegistryPath()
	if err != nil {
		return nil, fmt.Errorf("resolve registry path: %w", err)
	}

	registry, err := idx.LoadCachedRegistry(regPath)
	if err != nil {
		registry, err = idx.FetchRegistry(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetch registry: %w", err)
		}

		_ = idx.SaveCachedRegistry(regPath, registry)
	}

	return registry, nil
}
