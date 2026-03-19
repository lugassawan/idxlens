package idx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
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

	req, err := c.newRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", att.FileName, err)
	}

	//nolint:gosec // URL built from trusted baseURL set at client construction
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", att.FileName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: unexpected status %d", att.FileName, resp.StatusCode)
	}

	finalPath := filepath.Join(destDir, att.FileName)

	if err := atomicWrite(finalPath, resp.Body); err != nil {
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

// atomicWrite writes data from r to destPath via a temporary file for crash safety.
func atomicWrite(destPath string, r io.Reader) error {
	tmpPath := destPath + ".tmp"

	if err := writeFile(tmpPath, r); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

func writeFile(path string, r io.Reader) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}

	return nil
}
