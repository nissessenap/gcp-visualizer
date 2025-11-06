package collector

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/NissesSenap/gcp-visualizer/internal/storage"
	"google.golang.org/api/iterator"
)

// collectTopics collects all topics from a GCP project
func (c *Collector) collectTopics(ctx context.Context, client *pubsub.Client, projectID string) error {
	// Create list request
	req := &pubsubpb.ListTopicsRequest{
		Project: fmt.Sprintf("projects/%s", projectID),
	}

	it := client.TopicAdminClient.ListTopics(ctx, req)

	for {
		// Rate limiting
		if err := c.limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter error: %w", err)
		}

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

// extractResourceName extracts the resource name from a full resource path
// e.g., "projects/my-project/topics/my-topic" -> "my-topic"
func extractResourceName(fullPath string) string {
	parts := strings.Split(fullPath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullPath
}
