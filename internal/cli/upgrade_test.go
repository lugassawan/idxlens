package cli

import "testing"

func TestIsDevBuild(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"dev", true},
		{"v1.0.2-12-gc0831fb-dirty", true},
		{"1.0.0-rc1", true},
		{"v1.0.2-dirty", true},
		{"1.0.2", false},
		{"v1.0.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := isDevBuild(tt.version); got != tt.want {
				t.Errorf("isDevBuild(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestUpgradeCommandRegistered(t *testing.T) {
	assertCommandRegistered(t, "upgrade")
}
