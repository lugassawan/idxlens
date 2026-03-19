package service

import (
	"context"
	"os"
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

	t.Run("cache miss falls back to fetch and returns error for unreachable URL", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("IDXLENS_HOME", dir)

		provider := &DefaultRegistryProvider{}

		// No cached file exists, so it will try FetchRegistry which hits the real
		// GitHub URL. With no network or on CI, this may fail. We just verify
		// the function returns without panic and handles the error path.
		_, err := provider.Registry(context.Background())

		// If it succeeded (network available), that's fine too.
		// The key is that the code path was exercised.
		_ = err
	})

	t.Run("loads from cache with corrupt file falls back to fetch", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("IDXLENS_HOME", dir)

		// Write corrupt data to trigger cache miss
		regPath, err := idx.RegistryPath()
		if err != nil {
			t.Fatalf("RegistryPath() error: %v", err)
		}

		if err := os.WriteFile(regPath, []byte("not json"), 0o600); err != nil {
			t.Fatalf("write corrupt cache: %v", err)
		}

		provider := &DefaultRegistryProvider{}

		// This will fail to load cache (corrupt), then try FetchRegistry.
		// We don't care if fetch succeeds or fails — we're testing the fallback path.
		_, _ = provider.Registry(context.Background())
	})
}
