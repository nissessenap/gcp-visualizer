package collector

import (
	"context"
	"fmt"
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
func (c *Collector) getClient(ctx context.Context, projectID string) (*pubsub.Client, error) {
	// First check with read lock (fast path for existing clients)
	c.mu.RLock()
	client, exists := c.clients[projectID]
	c.mu.RUnlock()
	if exists {
		return client, nil
	}

	// Acquire write lock to create new client
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check: another goroutine might have created the client while we waited
	if client, exists := c.clients[projectID]; exists {
		return client, nil
	}

	// Create new client
	client, err := auth.NewPubSubClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client for project %s: %w", projectID, err)
	}

	c.clients[projectID] = client
	return client, nil
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
