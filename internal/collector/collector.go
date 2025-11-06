package collector

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"cloud.google.com/go/pubsub/v2"
	"github.com/NissesSenap/gcp-visualizer/internal/auth"
	"github.com/NissesSenap/gcp-visualizer/internal/storage"
	"golang.org/x/time/rate"
)

// Collector manages GCP resource collection
type Collector struct {
	mu      sync.RWMutex // Protects clients map for concurrent access
	clients map[string]*pubsub.Client
	storage storage.Store
	limiter *rate.Limiter
}

// New creates a new Collector with the provided storage and rate limiter
func New(store storage.Store, requestsPerSecond float64) *Collector {
	return &Collector{
		clients: make(map[string]*pubsub.Client),
		storage: store,
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), int(requestsPerSecond*2)),
	}
}

// getClient returns a cached client for the project, or creates a new one.
// This method is thread-safe and uses double-checked locking for optimal performance.
// The client creation I/O operation happens outside the lock to avoid blocking other goroutines.
func (c *Collector) getClient(ctx context.Context, projectID string) (*pubsub.Client, error) {
	// First check with read lock (fast path for existing clients)
	c.mu.RLock()
	client, exists := c.clients[projectID]
	c.mu.RUnlock()
	if exists {
		return client, nil
	}

	// Create new client WITHOUT holding the lock
	// This allows other goroutines to proceed with their own I/O operations concurrently
	newClient, err := auth.NewPubSubClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client for project %s: %w", projectID, err)
	}

	// Acquire write lock only to store the client in the map
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check: another goroutine might have created and stored a client
	// while we were creating ours (race condition handling)
	if existingClient, exists := c.clients[projectID]; exists {
		// Another goroutine won the race and stored their client first
		// Close our client to avoid resource leak and return the existing one
		_ = newClient.Close()
		return existingClient, nil
	}

	// We won the race (or there was no race) - store our client
	c.clients[projectID] = newClient
	return newClient, nil
}

// CollectProject collects all Pub/Sub resources from a single project
func (c *Collector) CollectProject(ctx context.Context, projectID string) error {
	client, err := c.getClient(ctx, projectID)
	if err != nil {
		return err
	}

	// Collect topics
	if err := c.collectTopics(ctx, client, projectID); err != nil {
		return fmt.Errorf("failed to collect topics: %w", err)
	}

	// Collect subscriptions
	if err := c.collectSubscriptions(ctx, client, projectID); err != nil {
		return fmt.Errorf("failed to collect subscriptions: %w", err)
	}

	// Update project sync time
	if err := c.storage.UpdateProjectSyncTime(ctx, projectID); err != nil {
		return fmt.Errorf("failed to update project sync time: %w", err)
	}

	return nil
}

// Close closes all Pub/Sub clients, collecting all errors.
// Even if some clients fail to close, all others will still be closed.
func (c *Collector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error
	for projectID, client := range c.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close client for project %s: %w", projectID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing clients: %v", errs)
	}
	return nil
}

// extractResourceName extracts the resource name from a full resource path.
// It returns the last non-empty segment of the path, making it safe for paths
// with trailing slashes or empty segments.
//
// Returns empty string when:
//   - Input is empty
//   - Input contains only slashes
//   - All segments are empty
//
// Examples:
//   - "projects/my-project/topics/my-topic" -> "my-topic"
//   - "projects/my-project/subscriptions/my-subscription" -> "my-subscription"
//   - "simple-name" -> "simple-name"
//   - "" -> ""
//   - "/" -> ""
//   - "//" -> ""
//   - "projects/my-project/topics/" -> "topics" (malformed path, returns last valid segment)
func extractResourceName(fullPath string) string {
	// Handle empty string explicitly
	if fullPath == "" {
		return ""
	}

	parts := strings.Split(fullPath, "/")
	// Return the last non-empty part
	// This handles trailing slashes and empty segments correctly
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}

	// If all parts are empty (e.g., "/" or "//"), return empty string
	return ""
}
