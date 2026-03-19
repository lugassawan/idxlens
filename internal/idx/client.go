package idx

import (
	"fmt"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://www.idx.co.id"
	defaultTimeout = 30 * time.Second
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

// WithCookieFile loads cookies from the given file path and injects them into requests.
func WithCookieFile(path string) Option {
	return func(c *Client) {
		cookies, err := LoadCookies(path)
		if err != nil {
			return
		}
		c.cookies = cookies
	}
}

// newRequest creates an HTTP request with cookies injected.
func (c *Client) newRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}

	return req, nil
}
