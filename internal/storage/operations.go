package storage

import (
	"context"
	"fmt"
	"strings"
)

// SaveTopic inserts or updates a topic
func (s *SQLiteStorage) SaveTopic(ctx context.Context, topic *Topic) error {
	// Start transaction to ensure project is also saved
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ensure project exists in projects table
	projectQuery := `
        INSERT OR REPLACE INTO projects (project_id, last_synced)
        VALUES (?, CURRENT_TIMESTAMP)`
	if _, err := tx.ExecContext(ctx, projectQuery, topic.ProjectID); err != nil {
		return err
	}

	// Insert or update topic
	topicQuery := `
        INSERT OR REPLACE INTO topics
        (name, project_id, full_resource_name, metadata, last_synced)
        VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`
	if _, err := tx.ExecContext(ctx, topicQuery,
		topic.Name,
		topic.ProjectID,
		topic.FullResourceName,
		topic.Metadata); err != nil {
		return err
	}

	return tx.Commit()
}

// GetTopics retrieves all topics for a specific project
func (s *SQLiteStorage) GetTopics(ctx context.Context, projectID string) ([]*Topic, error) {
	query := `SELECT id, name, project_id, full_resource_name, metadata
              FROM topics
              WHERE project_id = ?`

	rows, err := s.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []*Topic
	for rows.Next() {
		t := &Topic{}
		if err := rows.Scan(&t.ID, &t.Name, &t.ProjectID, &t.FullResourceName, &t.Metadata); err != nil {
			return nil, err
		}
		topics = append(topics, t)
	}
	return topics, rows.Err()
}

// GetAllTopics retrieves topics for multiple projects
func (s *SQLiteStorage) GetAllTopics(ctx context.Context, projects []string) ([]*Topic, error) {
	if len(projects) == 0 {
		// Return all topics if no projects specified
		query := `SELECT id, name, project_id, full_resource_name, metadata FROM topics`
		rows, err := s.db.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return scanTopics(rows)
	}

	// Build query with placeholders for projects
	placeholders := make([]string, len(projects))
	args := make([]interface{}, len(projects))
	for i, p := range projects {
		placeholders[i] = "?"
		args[i] = p
	}

	query := fmt.Sprintf(`SELECT id, name, project_id, full_resource_name, metadata
                           FROM topics
                           WHERE project_id IN (%s)`,
		strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTopics(rows)
}

// SaveSubscription inserts or updates a subscription
func (s *SQLiteStorage) SaveSubscription(ctx context.Context, sub *Subscription) error {
	// Start transaction to ensure project is also saved
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ensure project exists in projects table
	projectQuery := `
        INSERT OR REPLACE INTO projects (project_id, last_synced)
        VALUES (?, CURRENT_TIMESTAMP)`
	if _, err := tx.ExecContext(ctx, projectQuery, sub.ProjectID); err != nil {
		return err
	}

	// Insert or update subscription
	subscriptionQuery := `
        INSERT OR REPLACE INTO subscriptions
        (name, project_id, topic_full_resource_name, full_resource_name, metadata, last_synced)
        VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
	if _, err := tx.ExecContext(ctx, subscriptionQuery,
		sub.Name,
		sub.ProjectID,
		sub.TopicFullResourceName,
		sub.FullResourceName,
		sub.Metadata); err != nil {
		return err
	}

	return tx.Commit()
}

// GetSubscriptions retrieves all subscriptions for a specific project
func (s *SQLiteStorage) GetSubscriptions(ctx context.Context, projectID string) ([]*Subscription, error) {
	query := `SELECT id, name, project_id, topic_full_resource_name, full_resource_name, metadata
              FROM subscriptions
              WHERE project_id = ?`

	rows, err := s.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSubscriptions(rows)
}

// GetAllSubscriptions retrieves subscriptions for multiple projects
func (s *SQLiteStorage) GetAllSubscriptions(ctx context.Context, projects []string) ([]*Subscription, error) {
	if len(projects) == 0 {
		// Return all subscriptions if no projects specified
		query := `SELECT id, name, project_id, topic_full_resource_name, full_resource_name, metadata
                  FROM subscriptions`
		rows, err := s.db.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return scanSubscriptions(rows)
	}

	// Build query with placeholders for projects
	placeholders := make([]string, len(projects))
	args := make([]interface{}, len(projects))
	for i, p := range projects {
		placeholders[i] = "?"
		args[i] = p
	}

	query := fmt.Sprintf(`SELECT id, name, project_id, topic_full_resource_name, full_resource_name, metadata
                           FROM subscriptions
                           WHERE project_id IN (%s)`,
		strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSubscriptions(rows)
}

// GetAllProjects returns all unique project IDs from the database
func (s *SQLiteStorage) GetAllProjects(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT project_id FROM projects ORDER BY project_id`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []string
	for rows.Next() {
		var projectID string
		if err := rows.Scan(&projectID); err != nil {
			return nil, err
		}
		projects = append(projects, projectID)
	}
	return projects, rows.Err()
}

// UpdateProjectSyncTime updates or inserts the last sync time for a project
func (s *SQLiteStorage) UpdateProjectSyncTime(ctx context.Context, projectID string) error {
	query := `
        INSERT OR REPLACE INTO projects (project_id, last_synced)
        VALUES (?, CURRENT_TIMESTAMP)`

	_, err := s.db.ExecContext(ctx, query, projectID)
	return err
}

// Helper function to scan topics from rows
func scanTopics(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]*Topic, error) {
	var topics []*Topic
	for rows.Next() {
		t := &Topic{}
		if err := rows.Scan(&t.ID, &t.Name, &t.ProjectID, &t.FullResourceName, &t.Metadata); err != nil {
			return nil, err
		}
		topics = append(topics, t)
	}
	return topics, rows.Err()
}

// Helper function to scan subscriptions from rows
func scanSubscriptions(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]*Subscription, error) {
	var subscriptions []*Subscription
	for rows.Next() {
		s := &Subscription{}
		if err := rows.Scan(&s.ID, &s.Name, &s.ProjectID, &s.TopicFullResourceName, &s.FullResourceName, &s.Metadata); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, s)
	}
	return subscriptions, rows.Err()
}
