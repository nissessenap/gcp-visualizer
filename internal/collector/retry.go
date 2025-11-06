package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// retryWithBackoff executes a function with exponential backoff on retryable errors.
//
// The retry logic:
//   - Attempts the function up to maxRetries times
//   - Uses exponential backoff: 1s, 2s, 4s
//   - Only retries if isRetryable returns true
//   - Respects context cancellation during sleep
//
// Returns:
//   - nil if function succeeds on any attempt
//   - original error if not retryable
//   - "max retries exceeded" error if all retries fail
func retryWithBackoff(ctx context.Context, fn func() error) error {
	backoff := 1 * time.Second
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		//	for i := range maxRetries {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry if error is not retryable (e.g., permission denied, not found)
		if !isRetryable(err) {
			return err
		}

		// Don't sleep after the last attempt
		if i < maxRetries-1 {
			// Use a timer to respect context cancellation during backoff
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				// Continue to next retry
			}
			backoff *= 2
		}
	}

	return fmt.Errorf("max retries (%d) exceeded, last error: %w", maxRetries, lastErr)
}

// isRetryable determines if an error should trigger a retry.
//
// Retryable errors include:
//   - Rate limit errors (HTTP 429)
//   - Temporary network failures (HTTP 502, 503, 504)
//   - gRPC UNAVAILABLE status
//   - gRPC RESOURCE_EXHAUSTED status (rate limiting)
//   - Errors containing "timeout", "deadline", or "temporary"
//
// Non-retryable errors include:
//   - Permission errors (HTTP 403)
//   - Not found errors (HTTP 404)
//   - Invalid request errors (HTTP 400)
//   - Authentication errors (HTTP 401)
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation - never retry these
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Check for gRPC status errors
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable, codes.ResourceExhausted, codes.DeadlineExceeded:
			return true
		case codes.PermissionDenied, codes.Unauthenticated, codes.NotFound, codes.InvalidArgument:
			return false
		}
	}

	// Check for Google API errors (HTTP-based)
	if apiErr, ok := err.(*googleapi.Error); ok {
		switch apiErr.Code {
		case 429, 502, 503, 504: // Rate limit, bad gateway, service unavailable, gateway timeout
			return true
		case 400, 401, 403, 404: // Bad request, unauthorized, forbidden, not found
			return false
		}
	}

	// Check error message for transient error indicators
	errMsg := strings.ToLower(err.Error())
	transientIndicators := []string{
		"timeout",
		"timed out",
		"deadline",
		"temporary",
		"connection reset",
		"connection refused",
		"broken pipe",
	}

	for _, indicator := range transientIndicators {
		if strings.Contains(errMsg, indicator) {
			return true
		}
	}

	// Default to not retrying unknown errors
	return false
}
