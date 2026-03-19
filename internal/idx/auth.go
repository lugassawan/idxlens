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
	authURL     = "https://www.idx.co.id"
	authTimeout = 60 * time.Second
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
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
		)...,
	)
	defer allocCancel()

	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	taskCtx, timeoutCancel := context.WithTimeout(taskCtx, authTimeout)
	defer timeoutCancel()

	if err := chromedp.Run(taskCtx,
		chromedp.Navigate(authURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
	); err != nil {
		return nil, fmt.Errorf("authenticate with IDX: %w", err)
	}

	var browserCookies []*http.Cookie

	if err := chromedp.Run(taskCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromeCookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				return fmt.Errorf("retrieve cookies: %w", err)
			}
			for _, c := range chromeCookies {
				browserCookies = append(browserCookies, &http.Cookie{
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

	return browserCookies, nil
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
