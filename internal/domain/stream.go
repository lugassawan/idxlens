package domain

import (
	"context"
	"fmt"
	"sync"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
	"github.com/lugassawan/idxlens/internal/table"
)

// StreamConfig controls concurrent page processing behavior.
type StreamConfig struct {
	Workers    int
	BufferSize int
}

// PageResult holds the output of processing a single PDF page.
type PageResult struct {
	PageNum int
	Tables  []table.Table
	Layout  layout.LayoutPage
	Err     error
}

// StreamPages processes PDF pages concurrently using bounded workers and
// streams results ordered by page number. The returned channel is closed
// when all pages have been processed or the context is cancelled.
func StreamPages(
	ctx context.Context,
	reader pdf.Reader,
	analyzer layout.Analyzer,
	detector table.Detector,
	cfg StreamConfig,
) <-chan PageResult {
	cfg = normalizeConfig(cfg)
	out := make(chan PageResult, cfg.BufferSize)

	go func() {
		defer close(out)

		pageCount := reader.PageCount()
		if pageCount == 0 {
			return
		}

		results := make([]PageResult, pageCount)
		var wg sync.WaitGroup
		sem := make(chan struct{}, cfg.Workers)

		for i := 1; i <= pageCount; i++ {
			if err := ctx.Err(); err != nil {
				results[i-1] = PageResult{
					PageNum: i,
					Err:     fmt.Errorf("stream cancelled before page %d: %w", i, err),
				}
				break
			}

			wg.Add(1)
			sem <- struct{}{}

			go func(pageNum int) {
				defer wg.Done()
				defer func() { <-sem }()

				results[pageNum-1] = processPage(ctx, reader, analyzer, detector, pageNum)
			}(i)
		}

		wg.Wait()

		for _, r := range results {
			if r.PageNum == 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case out <- r:
			}
		}
	}()

	return out
}

func normalizeConfig(cfg StreamConfig) StreamConfig {
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.BufferSize < 0 {
		cfg.BufferSize = 0
	}
	return cfg
}

func processPage(
	ctx context.Context,
	reader pdf.Reader,
	analyzer layout.Analyzer,
	detector table.Detector,
	pageNum int,
) PageResult {
	if err := ctx.Err(); err != nil {
		return PageResult{
			PageNum: pageNum,
			Err:     fmt.Errorf("stream cancelled at page %d: %w", pageNum, err),
		}
	}

	page, err := reader.Page(pageNum)
	if err != nil {
		return PageResult{
			PageNum: pageNum,
			Err:     fmt.Errorf("read page %d: %w", pageNum, err),
		}
	}

	lp, err := analyzer.Analyze(page)
	if err != nil {
		return PageResult{
			PageNum: pageNum,
			Err:     fmt.Errorf("analyze page %d: %w", pageNum, err),
		}
	}

	tables, err := detector.Detect(lp)
	if err != nil {
		return PageResult{
			PageNum: pageNum,
			Err:     fmt.Errorf("detect tables on page %d: %w", pageNum, err),
		}
	}

	return PageResult{
		PageNum: pageNum,
		Tables:  tables,
		Layout:  lp,
	}
}
