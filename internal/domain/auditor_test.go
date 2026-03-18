package domain

import (
	"testing"

	"github.com/lugassawan/idxlens/internal/layout"
)

func TestAuditorParserParse(t *testing.T) {
	tests := []struct {
		name         string
		pages        []layout.LayoutPage
		wantOpinion  OpinionType
		wantFirm     string
		wantDate     string
		wantKAMCount int
		wantLang     string
	}{
		{
			name: "indonesian unqualified opinion",
			pages: makePages(
				"Kantor Akuntan Publik Tanubrata Sutanto Fahmi Bambang & Rekan",
				"Laporan Auditor Independen",
				"Kami telah mengaudit laporan keuangan PT Example Tbk",
				"Menurut pendapat kami, laporan keuangan terlampir menyajikan secara wajar, dalam semua hal yang material",
				"15 maret 2024",
			),
			wantOpinion: OpinionUnqualified,
			wantFirm:    "Tanubrata Sutanto Fahmi Bambang & Rekan",
			wantDate:    "15 Maret 2024",
			wantLang:    "id",
		},
		{
			name: "english qualified opinion",
			pages: makePages(
				"Independent Auditor's Report",
				"We have audited the financial statements",
				"In our qualified opinion, except for the effects of the matter described",
				"12 january 2024",
			),
			wantOpinion: OpinionQualified,
			wantDate:    "12 January 2024",
			wantLang:    "en",
		},
		{
			name: "indonesian adverse opinion",
			pages: makePages(
				"Kantor Akuntan Publik Purwantono Sungkoro & Surja",
				"Laporan Auditor Independen",
				"Menurut pendapat kami, laporan keuangan tidak wajar",
				"20 februari 2024",
			),
			wantOpinion: OpinionAdverse,
			wantFirm:    "Purwantono Sungkoro & Surja",
			wantDate:    "20 Februari 2024",
			wantLang:    "id",
		},
		{
			name: "disclaimer of opinion",
			pages: makePages(
				"Kantor Akuntan Publik Aria Kanaka & Rekan",
				"Laporan Auditor Independen",
				"Kami tidak memberikan pendapat atas laporan keuangan",
				"5 april 2024",
			),
			wantOpinion: OpinionDisclaimer,
			wantFirm:    "Aria Kanaka & Rekan",
			wantDate:    "5 April 2024",
			wantLang:    "id",
		},
		{
			name: "unqualified with emphasis of matter",
			pages: makePages(
				"Laporan Auditor Independen",
				"Menurut pendapat kami, laporan keuangan menyajikan secara wajar tanpa pengecualian",
				"Penekanan Suatu Hal",
				"Tanpa bermaksud memberikan modifikasi atas pendapat kami",
				"10 mei 2024",
			),
			wantOpinion: OpinionUnqualifiedEmphasis,
			wantDate:    "10 Mei 2024",
			wantLang:    "id",
		},
		{
			name: "report with key audit matters",
			pages: makePages(
				"Laporan Auditor Independen",
				"wajar tanpa pengecualian",
				"Key Audit Matters",
				"Valuation of Goodwill",
				"Revenue Recognition",
				"Basis for Opinion",
				"10 juni 2024",
			),
			wantOpinion:  OpinionUnqualified,
			wantKAMCount: 2,
			wantDate:     "10 Juni 2024",
			wantLang:     "id",
		},
		{
			name: "indonesian key audit matters header",
			pages: makePages(
				"Laporan Auditor Independen",
				"wajar tanpa pengecualian",
				"Hal Audit Utama",
				"Penilaian Atas Goodwill",
				"Tanggung Jawab Auditor",
			),
			wantOpinion:  OpinionUnqualified,
			wantKAMCount: 1,
			wantLang:     "id",
		},
		{
			name:        "unknown unrecognizable report",
			pages:       makePages("Some random text without any opinion keywords"),
			wantOpinion: OpinionUnknown,
			wantLang:    "",
		},
		{
			name:        "empty pages",
			pages:       nil,
			wantOpinion: OpinionUnknown,
			wantLang:    "",
		},
	}

	parser := NewAuditorParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.pages)
			if err != nil {
				t.Fatalf("Parse() unexpected error: %v", err)
			}

			if got.Opinion != tt.wantOpinion {
				t.Errorf("Opinion = %q, want %q", got.Opinion, tt.wantOpinion)
			}

			if tt.wantFirm != "" && got.Firm != tt.wantFirm {
				t.Errorf("Firm = %q, want %q", got.Firm, tt.wantFirm)
			}

			if tt.wantDate != "" && got.ReportDate != tt.wantDate {
				t.Errorf("ReportDate = %q, want %q", got.ReportDate, tt.wantDate)
			}

			if tt.wantKAMCount > 0 && len(got.KeyAuditMatters) != tt.wantKAMCount {
				t.Errorf("KeyAuditMatters count = %d, want %d (items: %v)",
					len(got.KeyAuditMatters), tt.wantKAMCount, got.KeyAuditMatters)
			}

			if tt.wantLang != "" && got.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", got.Language, tt.wantLang)
			}
		})
	}
}

func TestDetectOpinionPriority(t *testing.T) {
	tests := []struct {
		name string
		text string
		want OpinionType
	}{
		{
			name: "disclaimer before adverse check",
			text: "auditor tidak memberikan pendapat atas laporan yang tidak wajar",
			want: OpinionDisclaimer,
		},
		{
			name: "adverse before qualified check",
			text: "pendapat tidak wajar atas laporan",
			want: OpinionAdverse,
		},
		{
			name: "qualified before unqualified check",
			text: "wajar dengan pengecualian dan bukan wajar tanpa pengecualian",
			want: OpinionQualified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectOpinion(tt.text)
			if got != tt.want {
				t.Errorf("detectOpinion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "lowercase month", input: "januari", want: "Januari"},
		{name: "already capitalized", input: "March", want: "March"},
		{name: "empty string", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := capitalizeFirst(tt.input)
			if got != tt.want {
				t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func makePages(lines ...string) []layout.LayoutPage {
	textLines := make([]layout.TextLine, 0, len(lines))
	for _, line := range lines {
		textLines = append(textLines, layout.TextLine{Text: line})
	}

	return []layout.LayoutPage{
		{Number: 1, Lines: textLines},
	}
}
