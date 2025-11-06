package collector

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "context canceled",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "context deadline exceeded",
			err:       context.DeadlineExceeded,
			retryable: false,
		},
		{
			name:      "gRPC unavailable",
			err:       status.Error(codes.Unavailable, "service unavailable"),
			retryable: true,
		},
		{
			name:      "gRPC resource exhausted (rate limit)",
			err:       status.Error(codes.ResourceExhausted, "quota exceeded"),
			retryable: true,
		},
		{
			name:      "gRPC deadline exceeded",
			err:       status.Error(codes.DeadlineExceeded, "deadline exceeded"),
			retryable: true,
		},
		{
			name:      "gRPC permission denied",
			err:       status.Error(codes.PermissionDenied, "permission denied"),
			retryable: false,
		},
		{
			name:      "gRPC unauthenticated",
			err:       status.Error(codes.Unauthenticated, "unauthenticated"),
			retryable: false,
		},
		{
			name:      "gRPC not found",
			err:       status.Error(codes.NotFound, "not found"),
			retryable: false,
		},
		{
			name:      "gRPC invalid argument",
			err:       status.Error(codes.InvalidArgument, "invalid argument"),
			retryable: false,
		},
		{
			name:      "HTTP 429 rate limit",
			err:       &googleapi.Error{Code: 429, Message: "too many requests"},
			retryable: true,
		},
		{
			name:      "HTTP 502 bad gateway",
			err:       &googleapi.Error{Code: 502, Message: "bad gateway"},
			retryable: true,
		},
		{
			name:      "HTTP 503 service unavailable",
			err:       &googleapi.Error{Code: 503, Message: "service unavailable"},
			retryable: true,
		},
		{
			name:      "HTTP 504 gateway timeout",
			err:       &googleapi.Error{Code: 504, Message: "gateway timeout"},
			retryable: true,
		},
		{
			name:      "HTTP 400 bad request",
			err:       &googleapi.Error{Code: 400, Message: "bad request"},
			retryable: false,
		},
		{
			name:      "HTTP 401 unauthorized",
			err:       &googleapi.Error{Code: 401, Message: "unauthorized"},
			retryable: false,
		},
		{
			name:      "HTTP 403 forbidden",
			err:       &googleapi.Error{Code: 403, Message: "forbidden"},
			retryable: false,
		},
		{
			name:      "HTTP 404 not found",
			err:       &googleapi.Error{Code: 404, Message: "not found"},
			retryable: false,
		},
		{
			name:      "timeout in error message",
			err:       errors.New("operation timed out"),
			retryable: true,
		},
		{
			name:      "deadline in error message",
			err:       errors.New("deadline exceeded"),
			retryable: true,
		},
		{
			name:      "temporary error in message",
			err:       errors.New("temporary failure"),
			retryable: true,
		},
		{
			name:      "connection reset",
			err:       errors.New("connection reset by peer"),
			retryable: true,
		},
		{
			name:      "connection refused",
			err:       errors.New("connection refused"),
			retryable: true,
		},
		{
			name:      "broken pipe",
			err:       errors.New("broken pipe"),
			retryable: true,
		},
		{
			name:      "unknown error",
			err:       errors.New("some unknown error"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryable(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

func TestRetryWithBackoff_Success(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := retryWithBackoff(ctx, func() error {
		attempts++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts, "Should succeed on first attempt")
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := retryWithBackoff(ctx, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure") // Retryable error
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts, "Should succeed on third attempt")
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	expectedErr := &googleapi.Error{Code: 403, Message: "permission denied"}

	err := retryWithBackoff(ctx, func() error {
		attempts++
		return expectedErr
	})

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 1, attempts, "Should not retry non-retryable errors")
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	retryableErr := errors.New("timeout")

	err := retryWithBackoff(ctx, func() error {
		attempts++
		return retryableErr
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")
	assert.Equal(t, 3, attempts, "Should attempt exactly 3 times")
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	// Cancel context after first attempt triggers a retry
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retryWithBackoff(ctx, func() error {
		attempts++
		return errors.New("temporary failure") // Retryable error
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	// Should have attempted once, then cancelled during backoff
	assert.Equal(t, 1, attempts, "Should stop retrying when context is cancelled")
}

func TestRetryWithBackoff_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	var attemptTimes []time.Time

	startTime := time.Now()
	err := retryWithBackoff(ctx, func() error {
		attempts++
		attemptTimes = append(attemptTimes, time.Now())
		return errors.New("temporary failure") // Always fail with retryable error
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")
	assert.Equal(t, 3, attempts)
	require.Len(t, attemptTimes, 3)

	// Verify exponential backoff timing:
	// Attempt 1: immediate
	// Attempt 2: after ~1s backoff
	// Attempt 3: after ~2s backoff (total ~3s from start)

	timeSinceStart := attemptTimes[2].Sub(startTime)
	expectedMinTime := 3 * time.Second // 1s + 2s backoff
	expectedMaxTime := 4 * time.Second // Allow some slack for test execution time

	assert.True(t, timeSinceStart >= expectedMinTime,
		"Total time should be at least 3 seconds (1s + 2s backoff), got %v", timeSinceStart)
	assert.True(t, timeSinceStart <= expectedMaxTime,
		"Total time should not exceed 4 seconds, got %v", timeSinceStart)
}

func TestRetryWithBackoff_ImmediateReturnOnSuccess(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	startTime := time.Now()
	err := retryWithBackoff(ctx, func() error {
		attempts++
		return nil
	})
	duration := time.Since(startTime)

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
	assert.Less(t, duration, 100*time.Millisecond, "Should return immediately on success")
}

func TestRetryWithBackoff_RealWorldScenario(t *testing.T) {
	ctx := context.Background()

	// Simulate a real-world scenario: transient failure followed by success
	attempts := 0
	err := retryWithBackoff(ctx, func() error {
		attempts++
		switch attempts {
		case 1:
			// First attempt: rate limited
			return &googleapi.Error{Code: 429, Message: "too many requests"}
		case 2:
			// Second attempt: timeout
			return errors.New("timeout connecting to service")
		case 3:
			// Third attempt: success
			return nil
		default:
			return fmt.Errorf("unexpected attempt %d", attempts)
		}
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts, "Should succeed after 2 retries")
}
