package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPrintBanner(t *testing.T) {
	cmd, buf := newBannerTestCmd()
	printBanner(cmd)

	out := buf.String()
	if !strings.Contains(out, "____") {
		t.Errorf("banner missing ASCII art: %q", out)
	}

	if !strings.Contains(out, "v"+version) {
		t.Errorf("banner missing version: %q", out)
	}
}

func TestPrintBannerNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	cmd, buf := newBannerTestCmd()
	printBanner(cmd)

	out := buf.String()
	if strings.Contains(out, "\033[") {
		t.Error("banner contains ANSI codes when NO_COLOR is set")
	}
}

func newBannerTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	return cmd, &buf
}
