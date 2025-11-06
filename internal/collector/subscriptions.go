package collector

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/NissesSenap/gcp-visualizer/internal/storage"
	"google.golang.org/api/iterator"
)

// collectSubscriptions collects all subscriptions from a GCP project with retry logic
// for transient errors. Retries are performed at the collection level to
// ensure a fresh iterator is used on each attempt, preventing data loss.
func (c *Collector) collectSubscriptions(ctx context.Context, client *pubsub.Client, projectID string) error {
	return retryWithBackoff(ctx, func() error {
		return c.collectSubscriptionsOnce(ctx, client, projectID)
	})
}

// collectSubscriptionsOnce performs a single attempt to collect all subscriptions from a GCP project.
// This function creates a fresh iterator and iterates through all subscriptions.
// If any error occurs, it returns immediately to allow the caller to retry
// with a fresh iterator.
func (c *Collector) collectSubscriptionsOnce(ctx context.Context, client *pubsub.Client, projectID string) error {
	// Create list request
	req := &pubsubpb.ListSubscriptionsRequest{
		Project: fmt.Sprintf("projects/%s", projectID),
	}

	// Create fresh iterator for this attempt
	it := client.SubscriptionAdminClient.ListSubscriptions(ctx, req)

	for {
		// Rate limiting
		if err := c.limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter error: %w", err)
		}

		// Fetch next subscription - no retry here, errors propagate to trigger
		// a high-level retry with a fresh iterator
		sub, err := it.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate subscriptions: %w", err)
		}

		// Extract subscription name from the full name
		// sub.Name is in format "projects/{project}/subscriptions/{subscription}"
		fullResourceName := sub.Name
		subName := extractResourceName(fullResourceName)

		// Get topic reference from subscription
		// sub.Topic is in format "projects/{project}/topics/{topic}"
		topicFullResourceName := sub.Topic

		// Save to storage
		err = c.storage.SaveSubscription(ctx, &storage.Subscription{
			Name:                  subName,
			ProjectID:             projectID,
			TopicFullResourceName: topicFullResourceName,
			FullResourceName:      fullResourceName,
			Metadata:              "{}",
		})
		if err != nil {
			return fmt.Errorf("failed to save subscription %s: %w", subName, err)
		}
	}

	return nil
}
