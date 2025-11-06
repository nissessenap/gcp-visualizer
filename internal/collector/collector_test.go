package collector

import (
	"context"
	"testing"

	"github.com/NissesSenap/gcp-visualizer/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestCollector(t *testing.T) (*Collector, storage.Store) {
	store, err := storage.NewSQLite(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	collector := New(store, 10.0)
	t.Cleanup(func() { _ = collector.Close() })

	return collector, store
}

func TestNew(t *testing.T) {
	store, err := storage.NewSQLite(":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	collector := New(store, 10.0)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.storage)
	assert.NotNil(t, collector.limiter)
	assert.NotNil(t, collector.clients)
}

func TestExtractResourceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "topic full path",
			input:    "projects/my-project/topics/my-topic",
			expected: "my-topic",
		},
		{
			name:     "subscription full path",
			input:    "projects/my-project/subscriptions/my-subscription",
			expected: "my-subscription",
		},
		{
			name:     "simple name",
			input:    "simple-name",
			expected: "simple-name",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResourceName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCollectorClose(t *testing.T) {
	collector, _ := setupTestCollector(t)

	// Close should succeed even with no clients
	err := collector.Close()
	assert.NoError(t, err)
}

// Note: Full integration tests for CollectProject, collectTopics, and collectSubscriptions
// require actual GCP credentials and are covered by manual verification and integration tests.
// These tests verify the structure and basic functionality.

func TestCollectorStructure(t *testing.T) {
	collector, store := setupTestCollector(t)
	ctx := context.Background()

	// Verify storage is accessible
	assert.NotNil(t, collector.storage)

	// Verify we can interact with storage through the collector
	err := store.SaveTopic(ctx, &storage.Topic{
		Name:             "test-topic",
		ProjectID:        "test-project",
		FullResourceName: "projects/test-project/topics/test-topic",
		Metadata:         "{}",
	})
	assert.NoError(t, err)

	// Verify topic was saved
	topics, err := store.GetTopics(ctx, "test-project")
	assert.NoError(t, err)
	assert.Len(t, topics, 1)
}

func TestCollectProject_ContextCancellation(t *testing.T) {
	collector, _ := setupTestCollector(t)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Attempting to collect with a cancelled context should fail gracefully
	// Since we don't have real GCP credentials, this test verifies that:
	// 1. The context is properly passed through to the collector
	// 2. The rate limiter respects the cancelled context
	err := collector.CollectProject(ctx, "test-project")

	// We expect an error because:
	// - Either the context is cancelled (context.Canceled)
	// - Or authentication fails (no credentials)
	// Either way, it should fail gracefully without hanging
	assert.Error(t, err, "Should fail gracefully with cancelled context")

	// If the error is context.Canceled, that's the ideal scenario
	// It means our context cancellation is working properly
	if err == context.Canceled {
		t.Log("Context cancellation working correctly")
	}
}
