package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterYearPeriodFlags(t *testing.T) {
	tests := []struct {
		name         string
		yearRequired bool
	}{
		{"year required", true},
		{"year optional", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			registerYearPeriodFlags(cmd, tt.yearRequired)

			yearFlag := cmd.Flags().Lookup(flagYear)
			if yearFlag == nil {
				t.Fatal("year flag not registered")
			}

			if yearFlag.Shorthand != "y" {
				t.Errorf("year shorthand = %q, want %q", yearFlag.Shorthand, "y")
			}

			periodFlag := cmd.Flags().Lookup(flagPeriod)
			if periodFlag == nil {
				t.Fatal("period flag not registered")
			}

			if periodFlag.Shorthand != "p" {
				t.Errorf("period shorthand = %q, want %q", periodFlag.Shorthand, "p")
			}

			// Verify required annotation
			_, hasRequired := yearFlag.Annotations[cobra.BashCompOneRequiredFlag]
			if tt.yearRequired && !hasRequired {
				t.Error("year flag should be marked required")
			}

			if !tt.yearRequired && hasRequired {
				t.Error("year flag should not be marked required")
			}
		})
	}
}

func TestRegisterOutputFlags(t *testing.T) {
	cmd := &cobra.Command{}
	registerOutputFlags(cmd)

	flags := []struct {
		name      string
		shorthand string
	}{
		{flagFormat, "f"},
		{flagOutput, "o"},
		{flagPretty, ""},
	}

	for _, tt := range flags {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.name)
			if f == nil {
				t.Fatalf("flag %q not registered", tt.name)
			}

			if tt.shorthand != "" && f.Shorthand != tt.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", tt.name, f.Shorthand, tt.shorthand)
			}
		})
	}
}

func TestParseYearPeriodFlags(t *testing.T) {
	tests := []struct {
		name       string
		yearVal    string
		periodVal  string
		wantYear   int
		wantPeriod string
	}{
		{"defaults", "", "", 0, ""},
		{"year only", "2024", "", 2024, ""},
		{"both set", "2025", "Q1", 2025, "Q1"},
		{"FY maps to Audit", "2025", "FY", 2025, "Audit"},
		{"fy lowercase maps to Audit", "2025", "fy", 2025, "Audit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			registerYearPeriodFlags(cmd, false)

			if tt.yearVal != "" {
				if err := cmd.Flags().Set(flagYear, tt.yearVal); err != nil {
					t.Fatalf("set year: %v", err)
				}
			}

			if tt.periodVal != "" {
				if err := cmd.Flags().Set(flagPeriod, tt.periodVal); err != nil {
					t.Fatalf("set period: %v", err)
				}
			}

			year, period := parseYearPeriodFlags(cmd)
			if year != tt.wantYear {
				t.Errorf("year = %d, want %d", year, tt.wantYear)
			}

			if period != tt.wantPeriod {
				t.Errorf("period = %q, want %q", period, tt.wantPeriod)
			}
		})
	}
}

func TestParseOutputFlags(t *testing.T) {
	tests := []struct {
		name       string
		outputVal  string
		prettyVal  string
		wantOutput string
		wantPretty bool
	}{
		{"defaults", "", "", "", false},
		{"output only", "/tmp/out.json", "", "/tmp/out.json", false},
		{"pretty only", "", "true", "", true},
		{"both set", "/tmp/out.json", "true", "/tmp/out.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			registerOutputFlags(cmd)

			if tt.outputVal != "" {
				if err := cmd.Flags().Set(flagOutput, tt.outputVal); err != nil {
					t.Fatalf("set output: %v", err)
				}
			}

			if tt.prettyVal != "" {
				if err := cmd.Flags().Set(flagPretty, tt.prettyVal); err != nil {
					t.Fatalf("set pretty: %v", err)
				}
			}

			output, pretty := parseOutputFlags(cmd)
			if output != tt.wantOutput {
				t.Errorf("output = %q, want %q", output, tt.wantOutput)
			}

			if pretty != tt.wantPretty {
				t.Errorf("pretty = %v, want %v", pretty, tt.wantPretty)
			}
		})
	}
}
