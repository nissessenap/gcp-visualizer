package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPubSubClient(t *testing.T) {
	// Note: This test requires valid GCP credentials
	// Skip in CI unless credentials are configured
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client, err := NewPubSubClient(ctx, "test-project")

	// Should fail gracefully if no credentials
	if err != nil {
		require.Contains(t, err.Error(), "credentials",
			"Error should mention credentials")
	} else {
		require.NotNil(t, client)
		defer func() {
			if closeErr := client.Close(); closeErr != nil {
				t.Logf("Failed to close client: %v", closeErr)
			}
		}()
	}
}
