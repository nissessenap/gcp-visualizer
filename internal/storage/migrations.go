package storage

func (s *SQLiteStorage) migrate() error {
	schema := `
    CREATE TABLE IF NOT EXISTS projects (
        project_id TEXT PRIMARY KEY,
        last_synced TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS topics (
        id INTEGER PRIMARY KEY,
        name TEXT NOT NULL,
        project_id TEXT NOT NULL,
        full_resource_name TEXT UNIQUE,
        metadata JSON,
        last_synced TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS subscriptions (
        id INTEGER PRIMARY KEY,
        name TEXT NOT NULL,
        project_id TEXT NOT NULL,
        topic_full_resource_name TEXT NOT NULL,
        full_resource_name TEXT UNIQUE,
        metadata JSON,
        last_synced TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_subs_topic
        ON subscriptions(topic_full_resource_name);
    CREATE INDEX IF NOT EXISTS idx_topics_project
        ON topics(project_id);
    CREATE INDEX IF NOT EXISTS idx_subs_project
        ON subscriptions(project_id);
    `

	_, err := s.db.Exec(schema)
	return err
}
