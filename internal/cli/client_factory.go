package cli

import (
	"fmt"

	"github.com/lugassawan/idxlens/internal/idx"
)

// newIDXClient creates an authenticated IDX API client using stored cookies.
func newIDXClient() (*idx.Client, error) {
	cookiePath, err := idx.CookiePath()
	if err != nil {
		return nil, fmt.Errorf("resolve cookie path: %w", err)
	}

	cookies, err := idx.LoadCookies(cookiePath)
	if err != nil {
		return nil, fmt.Errorf("load cookies: %w", err)
	}

	return idx.New(idx.WithCookies(cookies)), nil
}
