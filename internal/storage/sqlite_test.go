package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStorage(t *testing.T) Store {
	store, err := NewSQLite(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

func TestSaveAndGetTopic(t *testing.T) {
	store := setupTestStorage(t)
	ctx := context.Background()

	topic := &Topic{
		Name:             "test-topic",
		ProjectID:        "test-project",
		FullResourceName: "projects/test-project/topics/test-topic",
		Metadata:         `{"labels": {}}`,
	}

	err := store.SaveTopic(ctx, topic)
	require.NoError(t, err)

	topics, err := store.GetTopics(ctx, "test-project")
	require.NoError(t, err)
	assert.Len(t, topics, 1)
	assert.Equal(t, "test-topic", topics[0].Name)
	assert.Equal(t, "test-project", topics[0].ProjectID)
	assert.Equal(t, "projects/test-project/topics/test-topic", topics[0].FullResourceName)
}

func TestSaveSubscription(t *testing.T) {
	store := setupTestStorage(t)
	ctx := context.Background()

	sub := &Subscription{
		Name:                  "test-sub",
		ProjectID:             "test-project",
		TopicFullResourceName: "projects/test-project/topics/test-topic",
		FullResourceName:      "projects/test-project/subscriptions/test-sub",
	}

	err := store.SaveSubscription(ctx, sub)
	require.NoError(t, err)

	subs, err := store.GetSubscriptions(ctx, "test-project")
	require.NoError(t, err)
	assert.Len(t, subs, 1)
	assert.Equal(t, "test-sub", subs[0].Name)
	assert.Equal(t, "projects/test-project/topics/test-topic", subs[0].TopicFullResourceName)
}

func TestGetAllTopics(t *testing.T) {
	store := setupTestStorage(t)
	ctx := context.Background()

	// Save topics in two different projects
	topic1 := &Topic{
		Name:             "topic1",
		ProjectID:        "project-a",
		FullResourceName: "projects/project-a/topics/topic1",
	}
	topic2 := &Topic{
		Name:             "topic2",
		ProjectID:        "project-b",
		FullResourceName: "projects/project-b/topics/topic2",
	}

	require.NoError(t, store.SaveTopic(ctx, topic1))
	require.NoError(t, store.SaveTopic(ctx, topic2))

	// Get all topics for specific projects
	topics, err := store.GetAllTopics(ctx, []string{"project-a", "project-b"})
	require.NoError(t, err)
	assert.Len(t, topics, 2)
}

func TestGetAllSubscriptions(t *testing.T) {
	store := setupTestStorage(t)
	ctx := context.Background()

	sub1 := &Subscription{
		Name:                  "sub1",
		ProjectID:             "project-a",
		TopicFullResourceName: "projects/project-a/topics/topic1",
		FullResourceName:      "projects/project-a/subscriptions/sub1",
	}
	sub2 := &Subscription{
		Name:                  "sub2",
		ProjectID:             "project-b",
		TopicFullResourceName: "projects/project-b/topics/topic2",
		FullResourceName:      "projects/project-b/subscriptions/sub2",
	}

	require.NoError(t, store.SaveSubscription(ctx, sub1))
	require.NoError(t, store.SaveSubscription(ctx, sub2))

	subs, err := store.GetAllSubscriptions(ctx, []string{"project-a", "project-b"})
	require.NoError(t, err)
	assert.Len(t, subs, 2)
}

func TestGetAllProjects(t *testing.T) {
	store := setupTestStorage(t)
	ctx := context.Background()

	// Save topics in different projects
	topic1 := &Topic{
		Name:             "topic1",
		ProjectID:        "project-a",
		FullResourceName: "projects/project-a/topics/topic1",
	}
	topic2 := &Topic{
		Name:             "topic2",
		ProjectID:        "project-b",
		FullResourceName: "projects/project-b/topics/topic2",
	}

	require.NoError(t, store.SaveTopic(ctx, topic1))
	require.NoError(t, store.SaveTopic(ctx, topic2))

	projects, err := store.GetAllProjects(ctx)
	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Contains(t, projects, "project-a")
	assert.Contains(t, projects, "project-b")
}

func TestUpdateProjectSyncTime(t *testing.T) {
	store := setupTestStorage(t)
	ctx := context.Background()

	// First update should insert
	err := store.UpdateProjectSyncTime(ctx, "test-project")
	require.NoError(t, err)

	// Second update should update timestamp
	err = store.UpdateProjectSyncTime(ctx, "test-project")
	require.NoError(t, err)

	// Verify project exists
	projects, err := store.GetAllProjects(ctx)
	require.NoError(t, err)
	assert.Contains(t, projects, "test-project")
}

func TestCrossProjectSubscription(t *testing.T) {
	store := setupTestStorage(t)
	ctx := context.Background()

	// Save a topic in project-a
	topic := &Topic{
		Name:             "shared-topic",
		ProjectID:        "project-a",
		FullResourceName: "projects/project-a/topics/shared-topic",
	}
	require.NoError(t, store.SaveTopic(ctx, topic))

	// Save a subscription in project-b that subscribes to project-a's topic
	sub := &Subscription{
		Name:                  "cross-project-sub",
		ProjectID:             "project-b",
		TopicFullResourceName: "projects/project-a/topics/shared-topic",
		FullResourceName:      "projects/project-b/subscriptions/cross-project-sub",
	}
	require.NoError(t, store.SaveSubscription(ctx, sub))

	// Verify cross-project relationship
	subs, err := store.GetSubscriptions(ctx, "project-b")
	require.NoError(t, err)
	assert.Len(t, subs, 1)
	assert.Equal(t, "project-b", subs[0].ProjectID)
	assert.Equal(t, "projects/project-a/topics/shared-topic", subs[0].TopicFullResourceName)
}

func TestUpsertTopic(t *testing.T) {
	store := setupTestStorage(t)
	ctx := context.Background()

	topic := &Topic{
		Name:             "upsert-topic",
		ProjectID:        "test-project",
		FullResourceName: "projects/test-project/topics/upsert-topic",
		Metadata:         `{"version": 1}`,
	}

	// First save
	require.NoError(t, store.SaveTopic(ctx, topic))

	// Update metadata
	topic.Metadata = `{"version": 2}`
	require.NoError(t, store.SaveTopic(ctx, topic))

	// Should still only have one topic
	topics, err := store.GetTopics(ctx, "test-project")
	require.NoError(t, err)
	assert.Len(t, topics, 1)
	assert.Equal(t, `{"version": 2}`, topics[0].Metadata)
}
