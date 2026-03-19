package idx

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultBaseURL   = "https://www.idx.co.id"
	defaultTimeout   = 30 * time.Second
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

// Client is an HTTP client for the IDX API with cookie injection support.
type Client struct {
	baseURL    string
	httpClient *http.Client
	cookies    []*http.Cookie
}

// Option configures a Client.
type Option func(*Client)

// New creates a new IDX API client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithBaseURL sets the base URL for API requests.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithCookies sets the cookies to inject into requests.
func WithCookies(cookies []*http.Cookie) Option {
	return func(c *Client) {
		c.cookies = cookies
	}
}

// WithHTTPClient sets a custom HTTP client for requests.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// newRequest creates an HTTP request with context and cookies injected.
func (c *Client) newRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", defaultBaseURL+"/")

	return req, nil
}
