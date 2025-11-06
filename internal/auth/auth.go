package auth

import (
	"context"

	"cloud.google.com/go/pubsub/v2"
)

// NewPubSubClient creates a Pub/Sub client using Application Default Credentials
// Users must run: gcloud auth application-default login
func NewPubSubClient(ctx context.Context, projectID string) (*pubsub.Client, error) {
	// Uses Application Default Credentials automatically
	return pubsub.NewClient(ctx, projectID)
}
