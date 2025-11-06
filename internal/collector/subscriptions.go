package collector

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/NissesSenap/gcp-visualizer/internal/storage"
	"google.golang.org/api/iterator"
)

// collectSubscriptions collects all subscriptions from a GCP project
func (c *Collector) collectSubscriptions(ctx context.Context, client *pubsub.Client, projectID string) error {
	// Create list request
	req := &pubsubpb.ListSubscriptionsRequest{
		Project: fmt.Sprintf("projects/%s", projectID),
	}

	it := client.SubscriptionAdminClient.ListSubscriptions(ctx, req)

	for {
		// Rate limiting
		if err := c.limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter error: %w", err)
		}

		// Fetch next subscription with retry logic for transient errors
		var sub *pubsubpb.Subscription
		err := retryWithBackoff(ctx, func() error {
			var iterErr error
			sub, iterErr = it.Next()
			return iterErr
		})

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
