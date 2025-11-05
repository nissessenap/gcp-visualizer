package storage

import "context"

// Store defines the interface for all storage operations
// This allows swapping SQLite for PostgreSQL in the future
type Store interface {
	// Topics
	SaveTopic(ctx context.Context, topic *Topic) error
	GetTopics(ctx context.Context, projectID string) ([]*Topic, error)
	GetAllTopics(ctx context.Context, projects []string) ([]*Topic, error)

	// Subscriptions
	SaveSubscription(ctx context.Context, sub *Subscription) error
	GetSubscriptions(ctx context.Context, projectID string) ([]*Subscription, error)
	GetAllSubscriptions(ctx context.Context, projects []string) ([]*Subscription, error)

	// Projects
	GetAllProjects(ctx context.Context) ([]string, error)
	UpdateProjectSyncTime(ctx context.Context, projectID string) error

	// Lifecycle
	Close() error
}

// Topic represents a Pub/Sub topic
type Topic struct {
	ID               int64
	Name             string
	ProjectID        string
	FullResourceName string
	Metadata         string // JSON
}

// Subscription represents a Pub/Sub subscription
type Subscription struct {
	ID                    int64
	Name                  string
	ProjectID             string
	TopicFullResourceName string
	FullResourceName      string
	Metadata              string // JSON
}
