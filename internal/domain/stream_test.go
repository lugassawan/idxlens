package domain

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
	"github.com/lugassawan/idxlens/internal/table"
)

type stubReader struct {
	pages  int
	pageFn func(int) (pdf.Page, error)
}

func (s *stubReader) Open(io.ReadSeeker) error        { return nil }
func (s *stubReader) Metadata() (pdf.Metadata, error) { return pdf.Metadata{}, nil }
func (s *stubReader) PageCount() int                  { return s.pages }
func (s *stubReader) Close() error                    { return nil }

func (s *stubReader) Page(n int) (pdf.Page, error) {
	if s.pageFn != nil {
		return s.pageFn(n)
	}
	return pdf.Page{Number: n}, nil
}

type stubAnalyzer struct {
	analyzeFn func(pdf.Page) (layout.LayoutPage, error)
}

func (s *stubAnalyzer) Analyze(p pdf.Page) (layout.LayoutPage, error) {
	if s.analyzeFn != nil {
		return s.analyzeFn(p)
	}
	return layout.LayoutPage{Number: p.Number}, nil
}

type stubDetector struct {
	detectFn func(layout.LayoutPage) ([]table.Table, error)
}

func (s *stubDetector) Detect(lp layout.LayoutPage) ([]table.Table, error) {
	if s.detectFn != nil {
		return s.detectFn(lp)
	}
	return []table.Table{{PageNum: lp.Number}}, nil
}

func collectResults(ch <-chan PageResult) []PageResult {
	var results []PageResult
	for r := range ch {
		results = append(results, r)
	}
	return results
}

func assertResultCount(t *testing.T, results []PageResult, want int) {
	t.Helper()
	if got := len(results); got != want {
		t.Fatalf("got %d results, want %d", got, want)
	}
}

func assertPageOrder(t *testing.T, results []PageResult, wantOrder []int) {
	t.Helper()
	for i, want := range wantOrder {
		if results[i].PageNum != want {
			t.Errorf("result[%d].PageNum = %d, want %d", i, results[i].PageNum, want)
		}
	}
}

func assertErrors(t *testing.T, results []PageResult, wantErrs []int) {
	t.Helper()
	errPages := make(map[int]bool, len(wantErrs))
	for _, p := range wantErrs {
		errPages[p] = true
	}
	for _, r := range results {
		if errPages[r.PageNum] && r.Err == nil {
			t.Errorf("page %d: expected error, got nil", r.PageNum)
		}
		if !errPages[r.PageNum] && r.Err != nil {
			t.Errorf("page %d: unexpected error: %v", r.PageNum, r.Err)
		}
	}
}

func TestStreamPages(t *testing.T) {
	tests := []struct {
		name      string
		pages     int
		workers   int
		buffer    int
		pageFn    func(int) (pdf.Page, error)
		analyzeFn func(pdf.Page) (layout.LayoutPage, error)
		detectFn  func(layout.LayoutPage) ([]table.Table, error)
		wantCount int
		wantOrder []int
		wantErrs  []int
	}{
		{
			name:      "processes multiple pages concurrently",
			pages:     5,
			workers:   3,
			buffer:    5,
			wantCount: 5,
			wantOrder: []int{1, 2, 3, 4, 5},
		},
		{
			name:      "single worker sequential fallback",
			pages:     3,
			workers:   1,
			buffer:    1,
			wantCount: 3,
			wantOrder: []int{1, 2, 3},
		},
		{
			name:      "zero pages returns empty channel",
			pages:     0,
			workers:   2,
			buffer:    2,
			wantCount: 0,
		},
		{
			name:    "page read error is propagated",
			pages:   2,
			workers: 2,
			buffer:  2,
			pageFn: func(n int) (pdf.Page, error) {
				if n == 2 {
					return pdf.Page{}, errors.New("read failure")
				}
				return pdf.Page{Number: n}, nil
			},
			wantCount: 2,
			wantOrder: []int{1, 2},
			wantErrs:  []int{2},
		},
		{
			name:    "analyze error is propagated",
			pages:   2,
			workers: 2,
			buffer:  2,
			analyzeFn: func(p pdf.Page) (layout.LayoutPage, error) {
				if p.Number == 1 {
					return layout.LayoutPage{}, errors.New("analyze failure")
				}
				return layout.LayoutPage{Number: p.Number}, nil
			},
			wantCount: 2,
			wantOrder: []int{1, 2},
			wantErrs:  []int{1},
		},
		{
			name:    "detect error is propagated",
			pages:   2,
			workers: 2,
			buffer:  2,
			detectFn: func(lp layout.LayoutPage) ([]table.Table, error) {
				if lp.Number == 1 {
					return nil, errors.New("detect failure")
				}
				return []table.Table{{PageNum: lp.Number}}, nil
			},
			wantCount: 2,
			wantOrder: []int{1, 2},
			wantErrs:  []int{1},
		},
		{
			name:      "zero workers defaults to one",
			pages:     2,
			workers:   0,
			buffer:    0,
			wantCount: 2,
			wantOrder: []int{1, 2},
		},
		{
			name:      "negative buffer defaults to zero",
			pages:     2,
			workers:   1,
			buffer:    -1,
			wantCount: 2,
			wantOrder: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &stubReader{pages: tt.pages, pageFn: tt.pageFn}
			analyzer := &stubAnalyzer{analyzeFn: tt.analyzeFn}
			detector := &stubDetector{detectFn: tt.detectFn}

			cfg := StreamConfig{Workers: tt.workers, BufferSize: tt.buffer}
			ch := StreamPages(context.Background(), reader, analyzer, detector, cfg)

			results := collectResults(ch)
			assertResultCount(t, results, tt.wantCount)

			if tt.wantOrder != nil {
				assertPageOrder(t, results, tt.wantOrder)
			}

			assertErrors(t, results, tt.wantErrs)
		})
	}
}

func TestStreamPagesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var processed atomic.Int32

	reader := &stubReader{pages: 100}
	analyzer := &stubAnalyzer{
		analyzeFn: func(p pdf.Page) (layout.LayoutPage, error) {
			processed.Add(1)
			if processed.Load() >= 3 {
				cancel()
			}
			return layout.LayoutPage{Number: p.Number}, nil
		},
	}
	detector := &stubDetector{}

	cfg := StreamConfig{Workers: 1, BufferSize: 1}
	ch := StreamPages(ctx, reader, analyzer, detector, cfg)

	results := collectResults(ch)

	if len(results) >= 100 {
		t.Errorf("expected fewer than 100 results after cancellation, got %d", len(results))
	}
}

func TestStreamPagesResultsOrdered(t *testing.T) {
	reader := &stubReader{pages: 10}
	analyzer := &stubAnalyzer{}
	detector := &stubDetector{}

	cfg := StreamConfig{Workers: 5, BufferSize: 10}
	ch := StreamPages(context.Background(), reader, analyzer, detector, cfg)

	var prev int
	for r := range ch {
		if r.PageNum <= prev {
			t.Errorf("results not ordered: page %d came after page %d", r.PageNum, prev)
		}
		prev = r.PageNum
	}
}
