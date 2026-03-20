package idx

import (
	"context"
	"math"
	"net/http"
	"time"
)

const (
	defaultMaxAttempts = 3
	defaultBaseDelay   = 1 * time.Second
)

// retryableStatusCodes are HTTP status codes that warrant a retry.
var retryableStatusCodes = map[int]bool{
	http.StatusTooManyRequests:     true, // 429
	http.StatusInternalServerError: true, // 500
	http.StatusBadGateway:          true, // 502
	http.StatusServiceUnavailable:  true, // 503
	http.StatusGatewayTimeout:      true, // 504
}

// retrySleep abstracts the sleep-with-context function for testing.
var retrySleep = func(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// retryDo executes fn up to defaultMaxAttempts times with exponential backoff.
// It retries on retryable status codes and network errors. The caller is
// responsible for closing the response body on the final attempt.
func retryDo(ctx context.Context, fn func() (*http.Response, error)) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := range defaultMaxAttempts {
		lastResp, lastErr = fn()

		if shouldReturn, resp, err := handleAttempt(ctx, attempt, lastResp, lastErr); shouldReturn {
			return resp, err
		}
	}

	return lastResp, lastErr
}

// handleAttempt evaluates a single retry attempt and decides whether to return or continue.
// Returns (true, resp, err) if the caller should return, or (false, nil, nil) to continue retrying.
func handleAttempt(
	ctx context.Context, attempt int, resp *http.Response, reqErr error,
) (bool, *http.Response, error) {
	isLastAttempt := attempt >= defaultMaxAttempts-1

	if reqErr != nil {
		if isLastAttempt {
			return true, nil, reqErr
		}

		if err := retrySleep(ctx, backoffDelay(attempt)); err != nil {
			return true, nil, err
		}

		return false, nil, nil
	}

	if !retryableStatusCodes[resp.StatusCode] {
		return true, resp, nil
	}

	if isLastAttempt {
		return true, resp, nil
	}

	_ = resp.Body.Close()

	if err := retrySleep(ctx, backoffDelay(attempt)); err != nil {
		return true, nil, err
	}

	return false, nil, nil
}

// backoffDelay calculates the delay for the given attempt (0-indexed).
// Returns defaultBaseDelay * 2^attempt (1s, 2s, 4s, ...).
func backoffDelay(attempt int) time.Duration {
	return defaultBaseDelay * time.Duration(math.Pow(2, float64(attempt)))
}
