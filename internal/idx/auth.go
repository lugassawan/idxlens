package idx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	authURL      = "https://www.idx.co.id"
	authTimeout  = 30 * time.Second
	pollInterval = 2 * time.Second
	cfClearance  = "cf_clearance"
)

// cookieEntry is a serializable representation of an HTTP cookie.
type cookieEntry struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

// Authenticate launches a headless browser to solve the Cloudflare challenge
// on the IDX website and returns the resulting cookies.
func Authenticate(ctx context.Context) ([]*http.Cookie, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	taskCtx, timeoutCancel := context.WithTimeout(taskCtx, authTimeout)
	defer timeoutCancel()

	if err := chromedp.Run(taskCtx,
		chromedp.Navigate(authURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
	); err != nil {
		return nil, fmt.Errorf("authenticate with IDX: %w", err)
	}

	cookies, err := waitForClearance(taskCtx)
	if err != nil {
		return nil, fmt.Errorf("authenticate with IDX: %w", err)
	}

	return cookies, nil
}

// SaveCookies writes cookies to a JSON file at the given path.
func SaveCookies(path string, cookies []*http.Cookie) error {
	entries := make([]cookieEntry, len(cookies))
	for i, c := range cookies {
		entries[i] = cookieEntry{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		}
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cookies: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write cookies to %s: %w", path, err)
	}

	return nil
}

// LoadCookies reads cookies from a JSON file at the given path.
func LoadCookies(path string) ([]*http.Cookie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cookies from %s: %w", path, err)
	}

	var entries []cookieEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal cookies: %w", err)
	}

	cookies := make([]*http.Cookie, len(entries))
	for i, e := range entries {
		cookies[i] = &http.Cookie{
			Name:   e.Name,
			Value:  e.Value,
			Domain: e.Domain,
			Path:   e.Path,
		}
	}

	return cookies, nil
}

// waitForClearance polls browser cookies until cf_clearance appears
// or the context deadline expires.
func waitForClearance(ctx context.Context) ([]*http.Cookie, error) {
	for {
		cookies, err := extractBrowserCookies(ctx)
		if err != nil {
			return nil, err
		}

		for _, c := range cookies {
			if c.Name == cfClearance {
				return cookies, nil
			}
		}

		select {
		case <-ctx.Done():
			// Cloudflare may not issue cf_clearance if no challenge was triggered.
			// Return whatever cookies we have — the API may still work.
			return cookies, nil
		case <-time.After(pollInterval):
		}
	}
}

// extractBrowserCookies retrieves all cookies from the browser via CDP.
func extractBrowserCookies(ctx context.Context) ([]*http.Cookie, error) {
	var cookies []*http.Cookie

	if err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromeCookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				return fmt.Errorf("retrieve cookies: %w", err)
			}

			for _, c := range chromeCookies {
				cookies = append(cookies, &http.Cookie{
					Name:   c.Name,
					Value:  c.Value,
					Domain: c.Domain,
					Path:   c.Path,
				})
			}

			return nil
		}),
	); err != nil {
		return nil, fmt.Errorf("extract cookies: %w", err)
	}

	return cookies, nil
}
