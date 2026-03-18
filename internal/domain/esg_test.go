package domain

import (
	"testing"

	"github.com/lugassawan/idxlens/internal/table"
)

func TestESGExtractorExtract(t *testing.T) {
	extractor := NewESGExtractor()

	tests := []struct {
		name            string
		tables          []table.Table
		wantNil         bool
		wantFramework   string
		wantDisclosures []GRIDisclosure
	}{
		{
			name:    "nil tables returns nil report",
			tables:  nil,
			wantNil: true,
		},
		{
			name:    "empty tables returns nil report",
			tables:  []table.Table{},
			wantNil: true,
		},
		{
			name: "table without GRI content returns nil report",
			tables: []table.Table{
				{
					Headers: []string{"Item", "Amount", "Date"},
					Rows: []table.Row{
						{Cells: []table.Cell{
							{Text: "Revenue"},
							{Text: "1,000,000"},
							{Text: "2024"},
						}},
					},
				},
			},
			wantNil: true,
		},
		{
			name: "single GRI disclosure from header-identified table",
			tables: []table.Table{
				{
					Headers: []string{"GRI Standard", "Disclosure", "Page"},
					Rows: []table.Row{
						{Cells: []table.Cell{
							{Text: "GRI 201-1"},
							{Text: "Direct economic value generated"},
							{Text: "45"},
						}},
					},
				},
			},
			wantFramework: "GRI",
			wantDisclosures: []GRIDisclosure{
				{
					Number:  "201-1",
					Title:   "Direct economic value generated",
					PageRef: "45",
				},
			},
		},
		{
			name: "multiple disclosures from single table",
			tables: []table.Table{
				{
					Headers: []string{"GRI Standard", "Title", "Description", "Page"},
					Rows: []table.Row{
						{Cells: []table.Cell{
							{Text: "GRI 302-1"},
							{Text: "Energy consumption"},
							{Text: "Energy within the organization"},
							{Text: "78"},
						}},
						{Cells: []table.Cell{
							{Text: "GRI 303-1"},
							{Text: "Water withdrawal"},
							{Text: "Interactions with water"},
							{Text: "82"},
						}},
					},
				},
			},
			wantFramework: "GRI",
			wantDisclosures: []GRIDisclosure{
				{
					Number:      "302-1",
					Title:       "Energy consumption",
					Description: "Energy within the organization",
					PageRef:     "78",
				},
				{
					Number:      "303-1",
					Title:       "Water withdrawal",
					Description: "Interactions with water",
					PageRef:     "82",
				},
			},
		},
		{
			name: "disclosure number without GRI prefix",
			tables: []table.Table{
				{
					Headers: []string{"GRI Index", "Topic"},
					Rows: []table.Row{
						{Cells: []table.Cell{
							{Text: "401-1"},
							{Text: "New employee hires"},
						}},
					},
				},
			},
			wantFramework: "GRI",
			wantDisclosures: []GRIDisclosure{
				{
					Number: "401-1",
					Title:  "New employee hires",
				},
			},
		},
		{
			name: "GRI detected in first row cells when headers empty",
			tables: []table.Table{
				{
					Headers: nil,
					Rows: []table.Row{
						{Cells: []table.Cell{
							{Text: "GRI Standard"},
							{Text: "Disclosure Title"},
						}},
						{Cells: []table.Cell{
							{Text: "GRI 205-1"},
							{Text: "Anti-corruption policies"},
						}},
					},
				},
			},
			wantFramework: "GRI",
			wantDisclosures: []GRIDisclosure{
				{
					Number: "205-1",
					Title:  "Anti-corruption policies",
				},
			},
		},
		{
			name: "status detection in disclosure row",
			tables: []table.Table{
				{
					Headers: []string{"GRI", "Topic", "Status"},
					Rows: []table.Row{
						{Cells: []table.Cell{
							{Text: "GRI 306-1"},
							{Text: "Waste generation"},
							{Text: "Reported"},
						}},
						{Cells: []table.Cell{
							{Text: "GRI 306-2"},
							{Text: "Waste management"},
							{Text: "Partially Reported"},
						}},
						{Cells: []table.Cell{
							{Text: "GRI 306-3"},
							{Text: "Waste diverted"},
							{Text: "Not Reported"},
						}},
					},
				},
			},
			wantFramework: "GRI",
			wantDisclosures: []GRIDisclosure{
				{
					Number: "306-1",
					Title:  "Waste generation",
					Status: StatusReported,
				},
				{
					Number: "306-2",
					Title:  "Waste management",
					Status: StatusPartiallyReported,
				},
				{
					Number: "306-3",
					Title:  "Waste diverted",
					Status: StatusNotReported,
				},
			},
		},
		{
			name: "row without disclosure number is skipped",
			tables: []table.Table{
				{
					Headers: []string{"GRI Standard", "Title"},
					Rows: []table.Row{
						{Cells: []table.Cell{
							{Text: "Economic Topics"},
							{Text: ""},
						}},
						{Cells: []table.Cell{
							{Text: "GRI 201-1"},
							{Text: "Economic value"},
						}},
					},
				},
			},
			wantFramework: "GRI",
			wantDisclosures: []GRIDisclosure{
				{
					Number: "201-1",
					Title:  "Economic value",
				},
			},
		},
		{
			name: "empty row is skipped",
			tables: []table.Table{
				{
					Headers: []string{"GRI Standard", "Title"},
					Rows: []table.Row{
						{Cells: []table.Cell{}},
						{Cells: []table.Cell{
							{Text: "GRI 201-1"},
							{Text: "Economic value"},
						}},
					},
				},
			},
			wantFramework: "GRI",
			wantDisclosures: []GRIDisclosure{
				{
					Number: "201-1",
					Title:  "Economic value",
				},
			},
		},
		{
			name: "multiple tables with mixed GRI and non-GRI content",
			tables: []table.Table{
				{
					Headers: []string{"Item", "Amount"},
					Rows: []table.Row{
						{Cells: []table.Cell{
							{Text: "Revenue"},
							{Text: "1,000,000"},
						}},
					},
				},
				{
					Headers: []string{"GRI Standard", "Title", "Page"},
					Rows: []table.Row{
						{Cells: []table.Cell{
							{Text: "GRI 102-1"},
							{Text: "Name of the organization"},
							{Text: "5"},
						}},
					},
				},
			},
			wantFramework: "GRI",
			wantDisclosures: []GRIDisclosure{
				{
					Number:  "102-1",
					Title:   "Name of the organization",
					PageRef: "5",
				},
			},
		},
		{
			name: "page reference with range",
			tables: []table.Table{
				{
					Headers: []string{"GRI", "Title", "Page"},
					Rows: []table.Row{
						{Cells: []table.Cell{
							{Text: "GRI 102-1"},
							{Text: "Organization name"},
							{Text: "5-7"},
						}},
					},
				},
			},
			wantFramework: "GRI",
			wantDisclosures: []GRIDisclosure{
				{
					Number:  "102-1",
					Title:   "Organization name",
					PageRef: "5-7",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.Extract(tt.tables)

			if tt.wantNil {
				if got != nil {
					t.Fatalf("Extract() = %v, want nil", got)
				}

				return
			}

			if got == nil {
				t.Fatal("Extract() returned nil, want non-nil report")
			}

			if got.Framework != tt.wantFramework {
				t.Errorf("Framework = %q, want %q", got.Framework, tt.wantFramework)
			}

			assertDisclosures(t, got.Disclosures, tt.wantDisclosures)
		})
	}
}

func TestIsPageReference(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "single page number", text: "45", want: true},
		{name: "page range", text: "45-47", want: true},
		{name: "multiple pages", text: "45, 47", want: true},
		{name: "text content", text: "Energy consumption", want: false},
		{name: "empty string", text: "", want: false},
		{name: "whitespace only", text: "   ", want: false},
		{name: "mixed content", text: "page 45", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPageReference(tt.text); got != tt.want {
				t.Errorf("isPageReference(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestDetectStatus(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "reported", text: "Reported", want: StatusReported},
		{name: "partially reported", text: "Partially Reported", want: StatusPartiallyReported},
		{name: "not reported", text: "Not Reported", want: StatusNotReported},
		{name: "case insensitive", text: "REPORTED", want: StatusReported},
		{name: "with whitespace", text: "  reported  ", want: StatusReported},
		{name: "no status", text: "Energy consumption", want: ""},
		{name: "empty string", text: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectStatus(tt.text); got != tt.want {
				t.Errorf("detectStatus(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func assertDisclosures(t *testing.T, got, want []GRIDisclosure) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d disclosures, want %d", len(got), len(want))
	}

	for i, w := range want {
		g := got[i]

		if g.Number != w.Number {
			t.Errorf("disclosure[%d].Number = %q, want %q", i, g.Number, w.Number)
		}

		if g.Title != w.Title {
			t.Errorf("disclosure[%d].Title = %q, want %q", i, g.Title, w.Title)
		}

		if g.Description != w.Description {
			t.Errorf("disclosure[%d].Description = %q, want %q", i, g.Description, w.Description)
		}

		if g.PageRef != w.PageRef {
			t.Errorf("disclosure[%d].PageRef = %q, want %q", i, g.PageRef, w.PageRef)
		}

		if g.Status != w.Status {
			t.Errorf("disclosure[%d].Status = %q, want %q", i, g.Status, w.Status)
		}
	}
}
