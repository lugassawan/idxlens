package cli

import (
	"fmt"
	"strings"

	"github.com/lugassawan/idxlens/internal/upgrade"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade idxlens to the latest version",
	RunE:  runUpgrade,
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

func isDevBuild(v string) bool {
	v = strings.TrimPrefix(v, "v")
	return v == "dev" || strings.Contains(v, "-")
}

func runUpgrade(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	w := cmd.OutOrStdout()

	if isDevBuild(version) {
		fmt.Fprintln(w, "Development build — skipping upgrade")
		return nil
	}

	fmt.Fprintln(w, "Checking for updates...")

	release, err := upgrade.LatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(version, "v")

	if latest == current {
		fmt.Fprintf(w, "Already up to date (v%s)\n", current)
		return nil
	}

	asset, err := upgrade.FindAsset(release)
	if err != nil {
		return fmt.Errorf("find platform asset: %w", err)
	}

	binPath, err := upgrade.CurrentBinaryPath()
	if err != nil {
		return fmt.Errorf("resolve binary path: %w", err)
	}

	fmt.Fprintf(w, "Upgrading from v%s to v%s...\n", current, latest)

	if err := upgrade.DownloadAsset(ctx, asset.DownloadURL, binPath); err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	fmt.Fprintf(w, "Successfully upgraded to v%s\n", latest)

	return nil
}
