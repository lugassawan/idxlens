package cli

import (
	"errors"
	"fmt"

	"github.com/lugassawan/idxlens/internal/idx"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with IDX website via headless browser",
	Long: `Launch a headless browser to solve the Cloudflare challenge on the IDX
website and save the resulting cookies for subsequent API requests.`,
	RunE: runAuth,
}

func init() {
	rootCmd.AddCommand(authCmd)
}

func runAuth(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cookies, err := idx.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if len(cookies) == 0 {
		return errors.New("authentication failed: no cookies received")
	}

	cookiePath, err := idx.CookiePath()
	if err != nil {
		return fmt.Errorf("resolve cookie path: %w", err)
	}

	if err := idx.SaveCookies(cookiePath, cookies); err != nil {
		return fmt.Errorf("save cookies: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Authentication successful. %d cookies saved to %s\n", len(cookies), cookiePath)

	return nil
}
