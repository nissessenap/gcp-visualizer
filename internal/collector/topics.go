package collector

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/NissesSenap/gcp-visualizer/internal/storage"
	"google.golang.org/api/iterator"
)

// collectTopics collects all topics from a GCP project with retry logic
// for transient errors. Retries are performed at the collection level to
// ensure a fresh iterator is used on each attempt, preventing data loss.
func (c *Collector) collectTopics(ctx context.Context, client *pubsub.Client, projectID string) error {
	return retryWithBackoff(ctx, func() error {
		return c.collectTopicsOnce(ctx, client, projectID)
	})
}

// collectTopicsOnce performs a single attempt to collect all topics from a GCP project.
// This function creates a fresh iterator and iterates through all topics.
// If any error occurs, it returns immediately to allow the caller to retry
// with a fresh iterator.
func (c *Collector) collectTopicsOnce(ctx context.Context, client *pubsub.Client, projectID string) error {
	// Create list request
	req := &pubsubpb.ListTopicsRequest{
		Project: fmt.Sprintf("projects/%s", projectID),
	}

	// Create fresh iterator for this attempt
	it := client.TopicAdminClient.ListTopics(ctx, req)

	for {
		// Rate limiting
		if err := c.limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter error: %w", err)
		}

		// Fetch next topic - no retry here, errors propagate to trigger
		// a high-level retry with a fresh iterator
		topic, err := it.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate topics: %w", err)
		}

		// Extract topic name from the full name
		// topic.Name is in format "projects/{project}/topics/{topic}"
		fullResourceName := topic.Name
		topicName := extractResourceName(fullResourceName)

		// Save to storage
		err = c.storage.SaveTopic(ctx, &storage.Topic{
			Name:             topicName,
			ProjectID:        projectID,
			FullResourceName: fullResourceName,
			Metadata:         "{}",
		})
		if err != nil {
			return fmt.Errorf("failed to save topic %s: %w", topicName, err)
		}
	}

	return nil
}
