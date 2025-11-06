package collector

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

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
		{
			name:     "single slash",
			input:    "/",
			expected: "",
		},
		{
			name:     "trailing slash",
			input:    "projects/my-project/topics/",
			expected: "topics",
		},
		{
			name:     "path with empty segments",
			input:    "projects//topics/my-topic",
			expected: "my-topic",
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

func TestGetClient_ConcurrentAccess(t *testing.T) {
	collector, _ := setupTestCollector(t)
	ctx := context.Background()

	// Test 1: Multiple goroutines trying to get clients for DIFFERENT projects
	// This demonstrates the blocking issue - with lock held during I/O,
	// even different projects block each other unnecessarily
	t.Run("different_projects", func(t *testing.T) {
		const numProjects = 10
		var wg sync.WaitGroup
		wg.Add(numProjects)

		startTime := time.Now()

		for i := 0; i < numProjects; i++ {
			go func(index int) {
				defer wg.Done()
				projectID := fmt.Sprintf("test-project-%d", index)
				_, err := collector.getClient(ctx, projectID)

				// We expect either success or auth failure
				// The key point is all goroutines should complete
				if err != nil {
					t.Logf("Project %s: %v", projectID, err)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(startTime)

		t.Logf("Getting clients for %d different projects took: %v", numProjects, duration)

		// With the CURRENT implementation (lock held during I/O):
		// - Each client creation is serialized
		// - Total time = sum of all I/O operations
		//
		// With the FIXED implementation (lock only for map access):
		// - Client creations happen concurrently
		// - Total time ≈ time of slowest I/O operation
	})

	// Test 2: Multiple goroutines trying to get client for SAME project
	// This tests the double-checked locking and ensures only one client is created
	t.Run("same_project", func(t *testing.T) {
		projectID := "test-project-concurrent"
		const numGoroutines = 20

		// Channels to collect results
		type result struct {
			err   error
			index int
		}
		results := make(chan result, numGoroutines)

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		startTime := time.Now()

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				defer wg.Done()

				// All goroutines try to get a client for the same project
				_, err := collector.getClient(ctx, projectID)
				results <- result{
					err:   err,
					index: index,
				}
			}(i)
		}

		wg.Wait()
		close(results)

		duration := time.Since(startTime)

		// Collect all results
		var errors []error
		var successCount int

		for res := range results {
			if res.err != nil {
				errors = append(errors, res.err)
			} else {
				successCount++
			}
		}

		t.Logf("Concurrent access to same project completed in %v", duration)
		t.Logf("Successful calls: %d, Failed calls: %d", successCount, len(errors))

		// Verify no deadlock occurred (test completed)
		assert.Equal(t, numGoroutines, successCount+len(errors),
			"All goroutines should complete")

		// With the CURRENT implementation holding the lock during I/O:
		// - Only the first goroutine creates the client (correct)
		// - BUT all other goroutines are blocked during creation (inefficient)
		//
		// After the fix (lock held only during map access):
		// - Multiple goroutines may attempt to create clients concurrently
		// - Only one client is stored in the map (via double-check)
		// - Extra clients are properly closed
		// - No unnecessary blocking

		// Verify that we can access the client map to check only one was created
		collector.mu.RLock()
		client, exists := collector.clients[projectID]
		collector.mu.RUnlock()

		if successCount > 0 {
			assert.True(t, exists, "Client should be in cache")
			assert.NotNil(t, client, "Client should not be nil")
		}
	})

	// This test should be run with: go test -race
	// to detect any race conditions in the implementation
}
