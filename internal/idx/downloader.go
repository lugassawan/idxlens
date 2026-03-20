package idx

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/lugassawan/idxlens/internal/safefile"
)

// DownloadResult holds the outcome of a single file download.
type DownloadResult struct {
	Attachment
	LocalPath string
	Err       error
}

// Download fetches a single attachment and saves it to destDir.
// It writes to a temporary file first, then renames for atomic writes.
func (c *Client) Download(ctx context.Context, att Attachment, destDir string) (*DownloadResult, error) {
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return nil, fmt.Errorf("create destination directory: %w", err)
	}

	url := c.baseURL + att.FilePath
	result := &DownloadResult{Attachment: att}

	resp, err := retryDo(ctx, func() (*http.Response, error) {
		r, reqErr := c.newRequest(ctx, http.MethodGet, url)
		if reqErr != nil {
			return nil, reqErr
		}

		//nolint:gosec // URL built from trusted baseURL set at client construction
		return c.httpClient.Do(r)
	})
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", att.FileName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: unexpected status %d", att.FileName, resp.StatusCode)
	}

	finalPath := filepath.Join(destDir, att.FileName)

	if err := safefile.Write(finalPath, resp.Body); err != nil {
		return nil, fmt.Errorf("download %s: %w", att.FileName, err)
	}

	result.LocalPath = finalPath

	return result, nil
}

// DownloadAll downloads multiple attachments with bounded concurrency.
func (c *Client) DownloadAll(ctx context.Context, atts []Attachment, destDir string, workers int) []DownloadResult {
	if workers < 1 {
		workers = 1
	}

	results := make([]DownloadResult, len(atts))
	sem := make(chan struct{}, workers)

	var wg sync.WaitGroup

	for i, att := range atts {
		wg.Add(1)

		go func(idx int, a Attachment) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := c.Download(ctx, a, destDir)
			if err != nil {
				results[idx] = DownloadResult{Attachment: a, Err: err}
				return
			}

			results[idx] = *result
		}(i, att)
	}

	wg.Wait()

	return results
}
