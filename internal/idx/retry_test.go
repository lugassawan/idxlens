package idx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func disableSleep(t *testing.T) {
	t.Helper()

	origSleep := retrySleep
	retrySleep = func(_ context.Context, _ time.Duration) error { return nil }
	t.Cleanup(func() { retrySleep = origSleep })
}

func newResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func TestRetryDoSuccess(t *testing.T) {
	disableSleep(t)

	calls := 0
	resp, err := retryDo(context.Background(), func() (*http.Response, error) {
		calls++
		return newResponse(http.StatusOK), nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetryDoRetryableStatusSucceedsOnSecondAttempt(t *testing.T) {
	disableSleep(t)

	calls := 0
	resp, err := retryDo(context.Background(), func() (*http.Response, error) {
		calls++
		if calls == 1 {
			return newResponse(http.StatusServiceUnavailable), nil
		}

		return newResponse(http.StatusOK), nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestRetryDoRetryableStatusExhaustsAttempts(t *testing.T) {
	disableSleep(t)

	calls := 0
	resp, err := retryDo(context.Background(), func() (*http.Response, error) {
		calls++
		return newResponse(http.StatusBadGateway), nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", resp.StatusCode)
	}

	if calls != defaultMaxAttempts {
		t.Fatalf("expected %d calls, got %d", defaultMaxAttempts, calls)
	}
}

func TestRetryDoNetworkErrorSucceedsOnRetry(t *testing.T) {
	disableSleep(t)

	calls := 0
	resp, err := retryDo(context.Background(), func() (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("connection refused")
		}

		return newResponse(http.StatusOK), nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestRetryDoNetworkErrorExhaustsAttempts(t *testing.T) {
	disableSleep(t)

	errNet := errors.New("connection refused")
	calls := 0

	//nolint:bodyclose // retryDo returns nil response when all attempts fail with errors
	_, err := retryDo(context.Background(), func() (*http.Response, error) {
		calls++
		return nil, errNet
	})

	if !errors.Is(err, errNet) {
		t.Fatalf("expected connection refused error, got: %v", err)
	}

	if calls != defaultMaxAttempts {
		t.Fatalf("expected %d calls, got %d", defaultMaxAttempts, calls)
	}
}

func TestRetryDoContextCancellation(t *testing.T) {
	origSleep := retrySleep
	retrySleep = func(ctx context.Context, _ time.Duration) error {
		return ctx.Err()
	}
	t.Cleanup(func() { retrySleep = origSleep })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0

	//nolint:bodyclose // retryDo returns nil response on context cancellation
	_, err := retryDo(ctx, func() (*http.Response, error) {
		calls++
		return nil, errors.New("network error")
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestBackoffDelay(t *testing.T) {
	tests := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{name: "attempt 0", attempt: 0, want: 1 * time.Second},
		{name: "attempt 1", attempt: 1, want: 2 * time.Second},
		{name: "attempt 2", attempt: 2, want: 4 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := backoffDelay(tt.attempt)
			if got != tt.want {
				t.Errorf("backoffDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestRetryableStatusCodes(t *testing.T) {
	retryable := []int{
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}

	for _, code := range retryable {
		if !retryableStatusCodes[code] {
			t.Errorf("expected status %d to be retryable", code)
		}
	}
}

func TestNonRetryableStatusCodes(t *testing.T) {
	disableSleep(t)

	nonRetryable := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
	}

	for _, code := range nonRetryable {
		t.Run(http.StatusText(code), func(t *testing.T) {
			calls := 0
			resp, err := retryDo(context.Background(), func() (*http.Response, error) {
				calls++
				return newResponse(code), nil
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != code {
				t.Fatalf("expected status %d, got %d", code, resp.StatusCode)
			}

			if calls != 1 {
				t.Fatalf("expected 1 call (no retry), got %d", calls)
			}
		})
	}
}
