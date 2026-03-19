package service

import (
	"context"
	"testing"

	"github.com/lugassawan/idxlens/internal/idx"
)

func TestDefaultRegistryProvider(t *testing.T) {
	t.Run("loads from cache", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("IDXLENS_HOME", dir)

		reg := map[string]idx.CompanyRegistry{
			"BBCA": {Name: "Bank Central Asia"},
		}

		regPath, err := idx.RegistryPath()
		if err != nil {
			t.Fatalf("RegistryPath() error: %v", err)
		}

		if err := idx.SaveCachedRegistry(regPath, reg); err != nil {
			t.Fatalf("SaveCachedRegistry() error: %v", err)
		}

		provider := &DefaultRegistryProvider{}

		result, err := provider.Registry(context.Background())
		if err != nil {
			t.Fatalf("Registry() error: %v", err)
		}

		if result["BBCA"].Name != "Bank Central Asia" {
			t.Errorf("name = %q, want %q", result["BBCA"].Name, "Bank Central Asia")
		}
	})
}
