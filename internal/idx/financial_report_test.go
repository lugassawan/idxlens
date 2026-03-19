package idx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestListReports(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		fixture, err := os.ReadFile("testdata/financial_report_response.json")
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
			}

			q := r.URL.Query()
			if got := q.Get("kodeEmiten"); got != "BBCA" {
				t.Errorf("kodeEmiten = %q, want %q", got, "BBCA")
			}

			if got := q.Get("year"); got != "2024" {
				t.Errorf("year = %q, want %q", got, "2024")
			}

			if got := q.Get("periode"); got != "Q3" {
				t.Errorf("periode = %q, want %q", got, "Q3")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fixture)
		}))
		defer srv.Close()

		c := New(WithBaseURL(srv.URL))
		attachments, err := c.ListReports(context.Background(), "BBCA", 2024, "Q3")
		if err != nil {
			t.Fatalf("ListReports() error: %v", err)
		}

		if len(attachments) != 2 {
			t.Fatalf("got %d attachments, want 2", len(attachments))
		}

		tests := []struct {
			name string
			got  string
			want string
		}{
			{"first file name", attachments[0].FileName, "Financial Report Q3 2024.pdf"},
			{"first file type", attachments[0].FileType, ".pdf"},
			{"first emiten code", attachments[0].EmitenCode, "BBCA"},
			{"first report period", attachments[0].ReportPeriod, "Q3"},
			{"first report year", attachments[0].ReportYear, "2025"},
			{"second file name", attachments[1].FileName, "Financial Report Q3 2024.xlsx"},
			{"second file type", attachments[1].FileType, ".xlsx"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.got != tt.want {
					t.Errorf("got %q, want %q", tt.got, tt.want)
				}
			})
		}

		if attachments[0].FileSize != 1048576 {
			t.Errorf("first file size = %d, want %d", attachments[0].FileSize, 1048576)
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		c := New(WithBaseURL(srv.URL))
		_, err := c.ListReports(context.Background(), "BBCA", 2024, "Q3")
		if err == nil {
			t.Fatal("ListReports() expected error for server error")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("not json"))
		}))
		defer srv.Close()

		c := New(WithBaseURL(srv.URL))
		_, err := c.ListReports(context.Background(), "BBCA", 2024, "Q3")
		if err == nil {
			t.Fatal("ListReports() expected error for invalid JSON")
		}
	})

	t.Run("empty results", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Results": []}`))
		}))
		defer srv.Close()

		c := New(WithBaseURL(srv.URL))
		attachments, err := c.ListReports(context.Background(), "BBCA", 2024, "Q3")
		if err != nil {
			t.Fatalf("ListReports() error: %v", err)
		}

		if len(attachments) != 0 {
			t.Errorf("got %d attachments, want 0", len(attachments))
		}
	})
}
