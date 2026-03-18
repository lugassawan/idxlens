package cli

import (
	"testing"
)

func TestParsePageRange(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		totalPages int
		want       []int
		wantErr    bool
	}{
		{
			name:       "single page",
			spec:       "1",
			totalPages: 10,
			want:       []int{1},
		},
		{
			name:       "multiple single pages",
			spec:       "1,3,5",
			totalPages: 10,
			want:       []int{1, 3, 5},
		},
		{
			name:       "range",
			spec:       "1-3",
			totalPages: 10,
			want:       []int{1, 2, 3},
		},
		{
			name:       "range and single",
			spec:       "1-3,5",
			totalPages: 10,
			want:       []int{1, 2, 3, 5},
		},
		{
			name:       "multiple ranges",
			spec:       "1-3,7-9",
			totalPages: 10,
			want:       []int{1, 2, 3, 7, 8, 9},
		},
		{
			name:       "deduplicate overlapping",
			spec:       "1-3,2-4",
			totalPages: 10,
			want:       []int{1, 2, 3, 4},
		},
		{
			name:       "single page equals total",
			spec:       "10",
			totalPages: 10,
			want:       []int{10},
		},
		{
			name:       "whitespace around parts",
			spec:       " 1 , 3 ",
			totalPages: 10,
			want:       []int{1, 3},
		},
		{
			name:       "page zero",
			spec:       "0",
			totalPages: 10,
			wantErr:    true,
		},
		{
			name:       "page exceeds total",
			spec:       "11",
			totalPages: 10,
			wantErr:    true,
		},
		{
			name:       "range exceeds total",
			spec:       "8-11",
			totalPages: 10,
			wantErr:    true,
		},
		{
			name:       "reversed range",
			spec:       "5-3",
			totalPages: 10,
			wantErr:    true,
		},
		{
			name:       "invalid number",
			spec:       "abc",
			totalPages: 10,
			wantErr:    true,
		},
		{
			name:       "invalid range format",
			spec:       "1-",
			totalPages: 10,
			wantErr:    true,
		},
		{
			name:       "empty spec",
			spec:       "",
			totalPages: 10,
			wantErr:    true,
		},
		{
			name:       "negative page",
			spec:       "-1",
			totalPages: 10,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePageRange(tt.spec, tt.totalPages)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parsePageRange(%q, %d) = %v, want error", tt.spec, tt.totalPages, got)
				}

				return
			}

			if err != nil {
				t.Errorf("parsePageRange(%q, %d) error = %v", tt.spec, tt.totalPages, err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("parsePageRange(%q, %d) = %v, want %v", tt.spec, tt.totalPages, got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parsePageRange(%q, %d) = %v, want %v", tt.spec, tt.totalPages, got, tt.want)
					return
				}
			}
		})
	}
}

func TestResolvePages(t *testing.T) {
	tests := []struct {
		name       string
		pagesFlag  string
		totalPages int
		want       []int
		wantErr    bool
	}{
		{
			name:       "empty flag returns all pages",
			pagesFlag:  "",
			totalPages: 3,
			want:       []int{1, 2, 3},
		},
		{
			name:       "specific pages",
			pagesFlag:  "1,3",
			totalPages: 5,
			want:       []int{1, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePages(tt.pagesFlag, tt.totalPages)

			if tt.wantErr {
				if err == nil {
					t.Errorf("resolvePages(%q, %d) = %v, want error", tt.pagesFlag, tt.totalPages, got)
				}

				return
			}

			if err != nil {
				t.Errorf("resolvePages(%q, %d) error = %v", tt.pagesFlag, tt.totalPages, err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("resolvePages(%q, %d) = %v, want %v", tt.pagesFlag, tt.totalPages, got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("resolvePages(%q, %d) = %v, want %v", tt.pagesFlag, tt.totalPages, got, tt.want)
					return
				}
			}
		})
	}
}
