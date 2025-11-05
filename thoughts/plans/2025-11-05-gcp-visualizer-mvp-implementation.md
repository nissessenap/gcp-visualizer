# GCP Visualizer MVP Implementation Plan

## Overview

Build a CLI tool to visualize GCP resource relationships, focusing on Pub/Sub topics/subscriptions in Part 1, then adding Service Accounts and IAM bindings in Part 2. The tool will handle 10,000+ resources across 40 projects using SQLite for persistence and Graphviz for visualization.

## Current State Analysis

This is a greenfield project with:
- Architecture thoroughly researched and documented
- No existing Go code
- Key technical decisions made (FDP layout engine, SQLite storage, no in-memory cache)
- Clear two-part MVP scope defined

## Testing Approach

**TDD (Test-Driven Development)** will be used throughout implementation where reasonable:

- **Use testify** for all tests: `github.com/stretchr/testify/assert`, `require`, and `mock`
- **Write tests first** for business logic (Phases 1-7, 10-13)
- **In-memory SQLite** (`:memory:`) for fast integration tests - no Docker needed
- **Storage interface** to enable future PostgreSQL migration without significant refactoring
- **Phases 8-9 (Graphviz)**: Focus on output validation rather than strict TDD due to rendering complexity

## Desired End State

A production-ready CLI tool that:
- Scans 40 GCP projects to discover Pub/Sub and Service Account relationships
- Persists data in SQLite for fast regeneration
- Generates high-quality SVG visualizations with project clustering
- Handles cross-project resource relationships
- Respects GCP API rate limits
- Provides both static SVG and optional interactive HTML output

### Verification
- Successfully scan 40 projects with 10,000 resources
- Generate visualization in under 1 minute from cached data
- Handle API rate limits gracefully
- Produce readable graphs with project boundaries

## What We're NOT Doing

- Building a web frontend or server
- Supporting resources beyond Pub/Sub and Service Accounts in MVP
- Real-time monitoring or continuous updates
- Terraform state integration (future phase)
- Cloud Asset Inventory API (starting with direct Pub/Sub API for simplicity)

## Implementation Approach

Build incrementally with 15 phases, delivering MVP Part 1 (Pub/Sub only) by Phase 10, then adding IAM/Service Accounts in Phases 11-13.

---

## Phase 1: Project Setup and Structure

### Overview
Initialize the Go project with proper directory structure and core dependencies.

### Changes Required:

#### 1. Project Structure
**Files to create**:
```
cmd/gcp-visualizer/main.go
internal/cli/cli.go
internal/storage/storage.go
internal/collector/collector.go
internal/graph/builder.go
internal/renderer/renderer.go
internal/config/config.go
Makefile
.gitignore
```

#### 2. Dependencies
**File**: `go.mod`
**Changes**: Add core dependencies

```bash
go get github.com/alecthomas/kong
go get modernc.org/sqlite
go get github.com/goccy/go-graphviz
go get cloud.google.com/go/pubsub
go get golang.org/x/time/rate
go get gopkg.in/yaml.v3
go get github.com/kelseyhightower/envconfig
go get github.com/stretchr/testify
```

#### 3. Main Entry Point
**File**: `cmd/gcp-visualizer/main.go`

```go
package main

import (
    "log"
    "github.com/NissesSenap/gcp-visualizer/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        log.Fatal(err)
    }
}
```

#### 4. Makefile
**File**: `Makefile`

```makefile
.PHONY: build test lint clean

build:
	go build -o gcp-visualizer cmd/gcp-visualizer/main.go

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

clean:
	rm -f gcp-visualizer
	rm -f coverage.out
	rm -rf /tmp/gcp-visualizer-*

install: build
	cp gcp-visualizer $(GOPATH)/bin/
```

### Success Criteria:

#### Automated Verification:
- [x] Project builds successfully: `make build`
- [x] Dependencies resolve: `go mod tidy && go mod download`
- [x] Linting passes: `make lint`
- [x] Directory structure created correctly

#### Manual Verification:
- [x] Binary runs and shows help: `./gcp-visualizer --help`

---

## Phase 2: CLI Framework and Basic Commands

### Overview
Implement the Kong-based CLI with scan, generate, and sync commands.

### Changes Required:

#### 1. CLI Structure
**File**: `internal/cli/cli.go`

```go
package cli

import (
    "github.com/alecthomas/kong"
)

type CLI struct {
    Scan     ScanCmd     `cmd:"scan" help:"Scan GCP projects for resources"`
    Generate GenerateCmd `cmd:"generate" help:"Generate visualization from cached data"`
    Sync     SyncCmd     `cmd:"sync" help:"Smart refresh of stale resources"`
    Config   ConfigCmd   `cmd:"config" help:"Manage configuration"`
    Version  VersionCmd  `cmd:"version" help:"Show version"`
}

type ScanCmd struct {
    Projects []string `help:"Projects to scan" placeholder:"PROJECT_ID"`
    Force    bool     `help:"Force refresh even if cached"`
}

type GenerateCmd struct {
    Output   string   `help:"Output file path" default:"output.svg"`
    Format   string   `help:"Output format" enum:"svg,png,pdf,html" default:"svg"`
    Projects []string `help:"Filter by projects"`
    Layout   string   `help:"Layout engine" enum:"fdp,dot,neato" default:"fdp"`
}

func Execute() error {
    cli := &CLI{}
    ctx := kong.Parse(cli)
    return ctx.Run()
}
```

#### 2. Command Handlers
**File**: `internal/cli/commands.go`

```go
package cli

func (c *ScanCmd) Run() error {
    // TODO: Implement scan logic
    fmt.Printf("Scanning projects: %v\n", c.Projects)
    return nil
}

func (c *GenerateCmd) Run() error {
    // TODO: Implement generate logic
    fmt.Printf("Generating %s output to %s\n", c.Format, c.Output)
    return nil
}
```

### Success Criteria:

#### Automated Verification:
- [x] CLI compiles: `go build ./cmd/gcp-visualizer`
- [x] All commands are registered: `./gcp-visualizer --help` shows all commands
- [x] Command validation works: `./gcp-visualizer generate --format=invalid` fails

#### Manual Verification:
- [ ] Help text is clear and comprehensive
- [ ] Commands accept expected flags
- [ ] Error messages are helpful

---

## Phase 3: Configuration Management

### Overview
Implement YAML configuration with environment variable overrides and Linux-only file locations.

### Changes Required:

#### 1. Config Structure
**File**: `internal/config/config.go`

```go
package config

import (
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
    "github.com/kelseyhightower/envconfig"
)

type Config struct {
    OrganizationID string   `yaml:"organization_id" envconfig:"ORGANIZATION_ID"`
    Projects       []string `yaml:"projects" envconfig:"PROJECTS"`
    Cache          Cache    `yaml:"cache"`
    Visualization  Visual   `yaml:"visualization"`
    RateLimits     Limits   `yaml:"rate_limits"`
}

type Cache struct {
    TTLHours    int `yaml:"ttl_hours" envconfig:"CACHE_TTL_HOURS"`
    MaxAgeHours int `yaml:"max_age_hours" envconfig:"CACHE_MAX_AGE_HOURS"`
}

type Visual struct {
    Layout         string `yaml:"layout" envconfig:"LAYOUT"`
    OutputFormat   string `yaml:"output_format" envconfig:"OUTPUT_FORMAT"`
    IncludeIcons   bool   `yaml:"include_icons" envconfig:"INCLUDE_ICONS"`
    ShowIAMDetails bool   `yaml:"show_iam_details" envconfig:"SHOW_IAM_DETAILS"`
}

type Limits struct {
    RequestsPerSecond float64 `yaml:"requests_per_second" envconfig:"REQUESTS_PER_SECOND"`
    MaxConcurrent     int     `yaml:"max_concurrent" envconfig:"MAX_CONCURRENT"`
}

// ConfigPath returns the configuration file path
// Default: ~/.config/gcp-visualizer/config.yaml
func ConfigPath() string {
    if path := os.Getenv("GCP_VISUALIZER_CONFIG"); path != "" {
        return path
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "gcp-visualizer", "config.yaml")
}

func Load() (*Config, error) {
    cfg := DefaultConfig()

    // Load from YAML file if exists
    configPath := ConfigPath()
    if data, err := os.ReadFile(configPath); err == nil {
        if err := yaml.Unmarshal(data, cfg); err != nil {
            return nil, err
        }
    }

    // Override with environment variables
    if err := envconfig.Process("GCP_VISUALIZER", cfg); err != nil {
        return nil, err
    }

    return cfg, nil
}

func (c *Config) Save() error {
    configPath := ConfigPath()

    // Create directory if not exists
    dir := filepath.Dir(configPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    data, err := yaml.Marshal(c)
    if err != nil {
        return err
    }

    return os.WriteFile(configPath, data, 0644)
}
```

#### 2. Default Configuration
**File**: `internal/config/defaults.go`

```go
func DefaultConfig() *Config {
    return &Config{
        Cache: Cache{
            TTLHours:    1,
            MaxAgeHours: 24,
        },
        Visualization: Visual{
            Layout:       "fdp",
            OutputFormat: "svg",
        },
        RateLimits: Limits{
            RequestsPerSecond: 10,
            MaxConcurrent:     5,
        },
    }
}
```

#### 3. Config Tests
**File**: `internal/config/config_test.go`

```go
package config

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
    cfg := DefaultConfig()
    assert.Equal(t, "fdp", cfg.Visualization.Layout)
    assert.Equal(t, 10.0, cfg.RateLimits.RequestsPerSecond)
}

func TestLoadConfig(t *testing.T) {
    // Test loading from file and env vars
}

func TestConfigPrecedence(t *testing.T) {
    // Test: env vars override YAML, CLI flags override all
}
```

### Success Criteria:

#### Automated Verification:
- [x] Config tests pass: `go test ./internal/config`
- [x] Default config is valid: `go test -run TestDefaultConfig`
- [x] Environment variables override YAML config
- [x] Config precedence: defaults < YAML < env vars < CLI flags

#### Manual Verification:
- [ ] Config file created at `~/.config/gcp-visualizer/config.yaml`
- [ ] Can override config via `GCP_VISUALIZER_PROJECTS` env var
- [ ] CLI flags override all other config sources
- [ ] Invalid config produces clear errors

---

## Phase 4: SQLite Storage Layer

### Overview
Implement SQLite database with schema for Pub/Sub resources. Use Storage interface for future PostgreSQL migration.

### Changes Required:

#### 1. Storage Interface
**File**: `internal/storage/interface.go`

```go
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
```

#### 2. SQLite Implementation
**File**: `internal/storage/sqlite.go`

```go
package storage

import (
    "context"
    "database/sql"
    "os"
    "path/filepath"
    _ "modernc.org/sqlite"
)

type SQLiteStorage struct {
    db *sql.DB
}

// NewSQLite creates a new SQLite storage backend
// For production: uses /tmp/gcp-visualizer/cache.db
// For testing: use ":memory:" as dbPath
func NewSQLite(dbPath string) (*SQLiteStorage, error) {
    // Create directory for file-based databases
    if dbPath != ":memory:" {
        dir := filepath.Dir(dbPath)
        if err := os.MkdirAll(dir, 0755); err != nil {
            return nil, err
        }
    }

    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, err
    }

    // Set pragmas for performance
    if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
        return nil, err
    }
    if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
        return nil, err
    }

    s := &SQLiteStorage{db: db}
    return s, s.migrate()
}

// NewDefaultSQLite creates storage in /tmp/gcp-visualizer/
func NewDefaultSQLite() (*SQLiteStorage, error) {
    dbPath := filepath.Join("/tmp", "gcp-visualizer", "cache.db")
    return NewSQLite(dbPath)
}

func (s *SQLiteStorage) Close() error {
    return s.db.Close()
}
```

#### 3. Database Schema
**File**: `internal/storage/migrations.go`

```go
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
```

#### 4. CRUD Operations (TDD)
**File**: `internal/storage/sqlite_test.go` (write first!)

```go
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
}
```

**File**: `internal/storage/operations.go` (implement after tests)

```go
func (s *SQLiteStorage) SaveTopic(ctx context.Context, topic *Topic) error {
    query := `
        INSERT OR REPLACE INTO topics
        (name, project_id, full_resource_name, metadata, last_synced)
        VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`
    _, err := s.db.ExecContext(ctx, query, topic.Name, topic.ProjectID,
                                topic.FullResourceName, topic.Metadata)
    return err
}

func (s *SQLiteStorage) GetTopics(ctx context.Context, projectID string) ([]*Topic, error) {
    query := `SELECT id, name, project_id, full_resource_name, metadata FROM topics WHERE project_id = ?`
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

// Implement remaining Store interface methods...
```

### Success Criteria:

#### Automated Verification:
- [ ] Storage tests pass: `go test ./internal/storage`
- [ ] In-memory database works for tests
- [ ] CRUD operations work: `go test -run TestSaveAndGetTopic`
- [ ] Indexes are created correctly
- [ ] Store interface fully implemented

#### Manual Verification:
- [ ] Database file created at `/tmp/gcp-visualizer/cache.db`
- [ ] Can query database with sqlite3 CLI tool
- [ ] Schema matches design from research
- [ ] WAL mode enabled for concurrent access

---

## Phase 5: GCP Authentication

### Overview
Implement GCP authentication using Application Default Credentials (ADC) only for MVP simplicity.

### Changes Required:

#### 1. Auth Manager
**File**: `internal/auth/auth.go`

```go
package auth

import (
    "context"
    "cloud.google.com/go/pubsub"
)

// NewPubSubClient creates a Pub/Sub client using Application Default Credentials
// Users must run: gcloud auth application-default login
func NewPubSubClient(ctx context.Context, projectID string) (*pubsub.Client, error) {
    // Uses Application Default Credentials automatically
    return pubsub.NewClient(ctx, projectID)
}
```

#### 2. Auth Tests (TDD)
**File**: `internal/auth/auth_test.go`

```go
package auth

import (
    "context"
    "testing"
    "github.com/stretchr/testify/require"
)

func TestNewPubSubClient(t *testing.T) {
    // Note: This test requires valid GCP credentials
    // Skip in CI unless credentials are configured
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    ctx := context.Background()
    client, err := NewPubSubClient(ctx, "test-project")

    // Should fail gracefully if no credentials
    if err != nil {
        require.Contains(t, err.Error(), "credentials",
            "Error should mention credentials")
    } else {
        require.NotNil(t, client)
        defer client.Close()
    }
}
```

### Success Criteria:

#### Automated Verification:
- [ ] Auth package compiles: `go build ./internal/auth`
- [ ] Unit tests pass: `go test ./internal/auth`
- [ ] Clear error when ADC not configured

#### Manual Verification:
- [ ] Can authenticate with ADC: `gcloud auth application-default login` then run tool
- [ ] Clear error message if ADC not configured
- [ ] Error message guides user to run `gcloud auth application-default login`

---

## Phase 6: Pub/Sub Data Collection

### Overview
Implement collection of Pub/Sub topics and subscriptions from GCP.

### Changes Required:

#### 1. Collector Interface
**File**: `internal/collector/collector.go`

```go
package collector

import (
    "context"
    "cloud.google.com/go/pubsub"
    "google.golang.org/api/iterator"
)

type Collector struct {
    clients map[string]*pubsub.Client
    storage *storage.Storage
    limiter *rate.Limiter
}

func (c *Collector) CollectProject(ctx context.Context, projectID string) error {
    client, err := c.getClient(ctx, projectID)
    if err != nil {
        return err
    }

    // Collect topics
    if err := c.collectTopics(ctx, client, projectID); err != nil {
        return err
    }

    // Collect subscriptions
    return c.collectSubscriptions(ctx, client, projectID)
}
```

#### 2. Topic Collection
**File**: `internal/collector/topics.go`

```go
func (c *Collector) collectTopics(ctx context.Context, client *pubsub.Client, projectID string) error {
    topics := client.Topics(ctx)

    for {
        c.limiter.Wait(ctx) // Rate limiting

        topic, err := topics.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return err
        }

        // Save to storage
        err = c.storage.SaveTopic(&storage.Topic{
            Name:             topic.ID(),
            ProjectID:        projectID,
            FullResourceName: fmt.Sprintf("projects/%s/topics/%s", projectID, topic.ID()),
        })
        if err != nil {
            return err
        }
    }

    return nil
}
```

#### 3. Subscription Collection
**File**: `internal/collector/subscriptions.go`

```go
func (c *Collector) collectSubscriptions(ctx context.Context, client *pubsub.Client, projectID string) error {
    subs := client.Subscriptions(ctx)

    for {
        c.limiter.Wait(ctx)

        sub, err := subs.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return err
        }

        // Get subscription config for topic reference
        config, err := sub.Config(ctx)
        if err != nil {
            return err
        }

        // Extract topic project and name from full resource name
        topicProject, topicName := parseTopicReference(config.Topic.String())

        err = c.storage.SaveSubscription(&storage.Subscription{
            Name:                 sub.ID(),
            ProjectID:            projectID,
            TopicFullResourceName: fmt.Sprintf("projects/%s/topics/%s", topicProject, topicName),
            FullResourceName:     fmt.Sprintf("projects/%s/subscriptions/%s", projectID, sub.ID()),
        })
        if err != nil {
            return err
        }
    }

    return nil
}
```

### Success Criteria:

#### Automated Verification:
- [ ] Collector compiles: `go build ./internal/collector`
- [ ] Unit tests with mocks pass: `go test ./internal/collector`

#### Manual Verification:
- [ ] Can scan a single project successfully
- [ ] Topics and subscriptions saved to database
- [ ] Cross-project subscriptions detected correctly
- [ ] Rate limiting prevents quota exhaustion

---

## Phase 7: Rate Limiting and Error Handling

### Overview
Implement robust rate limiting and partial failure handling.

### Changes Required:

#### 1. Rate Limiter
**File**: `internal/collector/ratelimit.go`

```go
package collector

import (
    "golang.org/x/time/rate"
    "sync"
)

type ProjectPool struct {
    projects    []string
    semaphore   chan struct{}
    rateLimiter *rate.Limiter
    errors      map[string]error
    mu          sync.Mutex
}

func NewProjectPool(projects []string, rps float64, maxConcurrent int) *ProjectPool {
    return &ProjectPool{
        projects:    projects,
        semaphore:   make(chan struct{}, maxConcurrent),
        rateLimiter: rate.NewLimiter(rate.Limit(rps), int(rps*2)),
        errors:      make(map[string]error),
    }
}

func (p *ProjectPool) CollectAll(ctx context.Context, collector *Collector) error {
    var wg sync.WaitGroup

    for _, projectID := range p.projects {
        wg.Add(1)

        go func(pid string) {
            defer wg.Done()

            // Acquire semaphore
            p.semaphore <- struct{}{}
            defer func() { <-p.semaphore }()

            // Rate limit
            p.rateLimiter.Wait(ctx)

            // Collect with error handling
            if err := collector.CollectProject(ctx, pid); err != nil {
                p.mu.Lock()
                p.errors[pid] = err
                p.mu.Unlock()
                // Log error but continue with other projects
                log.Printf("Failed to collect project %s: %v", pid, err)
            }
        }(projectID)
    }

    wg.Wait()

    if len(p.errors) > 0 {
        return fmt.Errorf("failed to collect %d projects", len(p.errors))
    }
    return nil
}
```

#### 2. Retry Logic
**File**: `internal/collector/retry.go`

```go
func retryWithBackoff(ctx context.Context, fn func() error) error {
    backoff := 1 * time.Second
    maxRetries := 3

    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }

        if !isRetryable(err) {
            return err
        }

        if i < maxRetries-1 {
            time.Sleep(backoff)
            backoff *= 2
        }
    }

    return fmt.Errorf("max retries exceeded")
}
```

### Success Criteria:

#### Automated Verification:
- [ ] Rate limiting works: `go test -run TestRateLimit`
- [ ] Retry logic handles transient errors
- [ ] Concurrent collection with semaphore

#### Manual Verification:
- [ ] Can scan 40 projects without hitting quota limits
- [ ] Failed projects are logged clearly
- [ ] Partial failures don't stop entire scan
- [ ] Progress indication during long scans

---

## Phase 8: Graph Building

### Overview
Build internal graph representation from collected data.

### Changes Required:

#### 1. Graph Data Structure
**File**: `internal/graph/graph.go`

```go
package graph

type Graph struct {
    Nodes    map[string]*Node
    Edges    []*Edge
    Clusters map[string]*Cluster // project clusters
}

type Node struct {
    ID       string
    Label    string
    Type     NodeType // Topic, Subscription, ServiceAccount
    Project  string
    Metadata map[string]string
}

type Edge struct {
    From  string
    To    string
    Label string
    Type  EdgeType // Publishes, Subscribes, CrossProject
}

type Cluster struct {
    ID    string
    Label string
    Nodes []string // node IDs
}

type NodeType string
const (
    NodeTypeTopic        NodeType = "topic"
    NodeTypeSubscription NodeType = "subscription"
)
```

#### 2. Graph Builder
**File**: `internal/graph/builder.go`

```go
package graph

import (
    "github.com/NissesSenap/gcp-visualizer/internal/storage"
)

type Builder struct {
    storage *storage.Storage
}

func (b *Builder) Build(projects []string) (*Graph, error) {
    g := &Graph{
        Nodes:    make(map[string]*Node),
        Edges:    make([]*Edge, 0),
        Clusters: make(map[string]*Cluster),
    }

    // Build nodes from topics
    topics, err := b.storage.GetAllTopics(projects)
    if err != nil {
        return nil, err
    }

    for _, topic := range topics {
        nodeID := fmt.Sprintf("topic_%s_%s", topic.ProjectID, topic.Name)
        g.Nodes[nodeID] = &Node{
            ID:      nodeID,
            Label:   topic.Name,
            Type:    NodeTypeTopic,
            Project: topic.ProjectID,
        }

        // Add to project cluster
        b.addToCluster(g, topic.ProjectID, nodeID)
    }

    // Build nodes and edges from subscriptions
    subs, err := b.storage.GetAllSubscriptions(projects)
    if err != nil {
        return nil, err
    }

    for _, sub := range subs {
        subNodeID := fmt.Sprintf("sub_%s_%s", sub.ProjectID, sub.Name)
        g.Nodes[subNodeID] = &Node{
            ID:      subNodeID,
            Label:   sub.Name,
            Type:    NodeTypeSubscription,
            Project: sub.ProjectID,
        }

        b.addToCluster(g, sub.ProjectID, subNodeID)

        // Create edge to topic
        topicProject, topicName := parseTopicReference(sub.TopicFullResourceName)
        topicNodeID := fmt.Sprintf("topic_%s_%s", topicProject, topicName)

        edgeType := EdgeTypeSubscribes
        if topicProject != sub.ProjectID {
            edgeType = EdgeTypeCrossProject
        }

        g.Edges = append(g.Edges, &Edge{
            From:  subNodeID,
            To:    topicNodeID,
            Type:  edgeType,
            Label: "subscribes",
        })
    }

    return g, nil
}

func (b *Builder) addToCluster(g *Graph, projectID, nodeID string) {
    if _, exists := g.Clusters[projectID]; !exists {
        g.Clusters[projectID] = &Cluster{
            ID:    fmt.Sprintf("cluster_%s", projectID),
            Label: projectID,
            Nodes: []string{},
        }
    }
    g.Clusters[projectID].Nodes = append(g.Clusters[projectID].Nodes, nodeID)
}
```

### Success Criteria:

#### Automated Verification:
- [ ] Graph builds from storage: `go test ./internal/graph`
- [ ] Clusters organized by project
- [ ] Cross-project edges identified correctly

#### Manual Verification:
- [ ] Graph structure makes logical sense
- [ ] All resources included in appropriate clusters
- [ ] Edge directions are correct

---

## Phase 9: Graphviz Integration

### Overview
Integrate go-graphviz to render graphs with FDP layout and project clustering.

### Changes Required:

#### 1. Renderer Implementation
**File**: `internal/renderer/graphviz.go`

```go
package renderer

import (
    "github.com/goccy/go-graphviz"
    "github.com/goccy/go-graphviz/cgraph"
    "github.com/NissesSenap/gcp-visualizer/internal/graph"
)

type GraphvizRenderer struct {
    gv    *graphviz.Graphviz
    graph *cgraph.Graph
}

func NewGraphvizRenderer() (*GraphvizRenderer, error) {
    return &GraphvizRenderer{
        gv: graphviz.New(),
    }, nil
}

func (r *GraphvizRenderer) Render(g *graph.Graph, output string, format string) error {
    // Create main graph
    mainGraph, err := r.gv.Graph()
    if err != nil {
        return err
    }
    defer mainGraph.Close()

    // Set layout to FDP for clustering support
    mainGraph.SetLayout(cgraph.FDP)

    // Set graph attributes
    mainGraph.SetOverlap("scale")
    mainGraph.SetSplines("line")
    mainGraph.SetCompound(true) // Enable cross-cluster edges

    // Create project clusters
    clusters := make(map[string]*cgraph.Graph)
    for projectID, cluster := range g.Clusters {
        subgraph := mainGraph.SubGraph(cluster.ID, 1)
        subgraph.SetLabel(cluster.Label)
        subgraph.SetStyle("filled")
        subgraph.SetFillColor("lightgrey")
        clusters[projectID] = subgraph
    }

    // Add nodes
    nodes := make(map[string]*cgraph.Node)
    for _, node := range g.Nodes {
        cluster := clusters[node.Project]

        gvNode, err := cluster.CreateNode(node.ID)
        if err != nil {
            return err
        }

        gvNode.SetLabel(node.Label)

        // Set shape based on type
        switch node.Type {
        case graph.NodeTypeTopic:
            gvNode.SetShape(cgraph.InvHouseShape)
            gvNode.SetFillColor("orange")
        case graph.NodeTypeSubscription:
            gvNode.SetShape(cgraph.BoxShape)
            gvNode.SetFillColor("lightgreen")
        }

        gvNode.SetStyle("filled")
        nodes[node.ID] = gvNode
    }

    // Add edges
    for _, edge := range g.Edges {
        fromNode := nodes[edge.From]
        toNode := nodes[edge.To]

        e, err := mainGraph.CreateEdge("", fromNode, toNode)
        if err != nil {
            return err
        }

        if edge.Type == graph.EdgeTypeCrossProject {
            e.SetStyle("dashed")
            e.SetColor("red")
        }
    }

    // Render to file
    return r.renderToFile(mainGraph, output, format)
}

func (r *GraphvizRenderer) renderToFile(graph *cgraph.Graph, output string, format string) error {
    var gvFormat graphviz.Format
    switch format {
    case "svg":
        gvFormat = graphviz.SVG
    case "png":
        gvFormat = graphviz.PNG
    case "pdf":
        gvFormat = graphviz.PDF
    default:
        gvFormat = graphviz.SVG
    }

    return r.gv.RenderFilename(graph, gvFormat, output)
}
```

### Success Criteria:

#### Automated Verification:
- [ ] Renderer compiles: `go build ./internal/renderer`
- [ ] Can generate valid DOT format
- [ ] Unit tests pass

#### Manual Verification:
- [ ] SVG output renders correctly in browser
- [ ] Project clusters are visible
- [ ] Cross-project edges show as dashed red lines
- [ ] Graph is readable with 100+ nodes

---

## Phase 10: Generate Command Implementation

### Overview
Complete the generate command to produce visualizations from cached data.

### Changes Required:

#### 1. Generate Command Logic
**File**: `internal/cli/generate.go`

```go
package cli

import (
    "github.com/NissesSenap/gcp-visualizer/internal/storage"
    "github.com/NissesSenap/gcp-visualizer/internal/graph"
    "github.com/NissesSenap/gcp-visualizer/internal/renderer"
)

func (c *GenerateCmd) Run() error {
    // Initialize storage
    store, err := storage.New()
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }
    defer store.Close()

    // Determine projects to include
    projects := c.Projects
    if len(projects) == 0 {
        // Load all projects from storage
        projects, err = store.GetAllProjects()
        if err != nil {
            return err
        }
    }

    if len(projects) == 0 {
        return fmt.Errorf("no projects found in cache. Run 'scan' first")
    }

    fmt.Printf("Generating visualization for %d projects...\n", len(projects))

    // Build graph
    builder := graph.NewBuilder(store)
    g, err := builder.Build(projects)
    if err != nil {
        return fmt.Errorf("failed to build graph: %w", err)
    }

    fmt.Printf("Graph contains %d nodes and %d edges\n", len(g.Nodes), len(g.Edges))

    // Render graph
    r, err := renderer.NewGraphvizRenderer()
    if err != nil {
        return err
    }
    defer r.Close()

    layout := c.Layout
    if layout == "" {
        layout = "fdp"
    }

    if err := r.Render(g, c.Output, c.Format); err != nil {
        return fmt.Errorf("failed to render graph: %w", err)
    }

    fmt.Printf("Visualization saved to %s\n", c.Output)
    return nil
}
```

#### 2. Scan Command Implementation
**File**: `internal/cli/scan.go`

```go
func (c *ScanCmd) Run() error {
    store, err := storage.New()
    if err != nil {
        return err
    }
    defer store.Close()

    // Load config
    config, err := config.Load()
    if err != nil {
        config = config.DefaultConfig()
    }

    // Determine projects
    projects := c.Projects
    if len(projects) == 0 {
        projects = config.Projects
    }

    if len(projects) == 0 {
        return fmt.Errorf("no projects specified")
    }

    fmt.Printf("Scanning %d projects...\n", len(projects))

    // Create collector
    coll := collector.New(store, config.RateLimits)

    // Collect with rate limiting
    pool := collector.NewProjectPool(projects,
                                      config.RateLimits.RequestsPerSecond,
                                      config.RateLimits.MaxConcurrent)

    if err := pool.CollectAll(context.Background(), coll); err != nil {
        return fmt.Errorf("collection failed: %w", err)
    }

    fmt.Println("Scan complete!")
    return nil
}
```

### Success Criteria:

#### Automated Verification:
- [ ] Integration test: scan then generate
- [ ] All formats generate successfully
- [ ] Project filtering works

#### Manual Verification:
- [ ] Can scan multiple projects
- [ ] Generate produces valid SVG
- [ ] Visualization opens in browser
- [ ] Performance acceptable for 10K resources

**Implementation Note**: This completes MVP Part 1 (Pub/Sub visualization). Pause here for testing before proceeding to Part 2.

---

## Phase 11: Service Account Collection

### Overview
Add Service Account discovery using Cloud Asset Inventory API.

### Changes Required:

#### 1. Update Schema
**File**: `internal/storage/migrations.go` (update)

```go
// Add to migration
CREATE TABLE IF NOT EXISTS service_accounts (
    id INTEGER PRIMARY KEY,
    email TEXT UNIQUE,
    project_id TEXT NOT NULL,
    display_name TEXT,
    metadata JSON,
    last_synced TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 2. Service Account Collector
**File**: `internal/collector/serviceaccounts.go`

```go
package collector

import (
    "cloud.google.com/go/iam"
)

func (c *Collector) collectServiceAccounts(ctx context.Context, projectID string) error {
    client, err := iam.NewService(ctx)
    if err != nil {
        return err
    }

    // List service accounts
    resp, err := client.Projects.ServiceAccounts.
        List(fmt.Sprintf("projects/%s", projectID)).
        Context(ctx).
        Do()
    if err != nil {
        return err
    }

    for _, sa := range resp.Accounts {
        err = c.storage.SaveServiceAccount(&storage.ServiceAccount{
            Email:       sa.Email,
            ProjectID:   projectID,
            DisplayName: sa.DisplayName,
        })
        if err != nil {
            return err
        }
    }

    return nil
}
```

### Success Criteria:

#### Automated Verification:
- [ ] Service accounts collected: `go test ./internal/collector`
- [ ] Database schema updated correctly

#### Manual Verification:
- [ ] Service accounts appear in database
- [ ] All SAs from test project collected

---

## Phase 12: IAM Binding Discovery

### Overview
Discover IAM bindings to determine which service accounts can access topics/subscriptions.

### Changes Required:

#### 1. IAM Schema
**File**: `internal/storage/migrations.go` (update)

```go
CREATE TABLE IF NOT EXISTS iam_bindings (
    id INTEGER PRIMARY KEY,
    resource_type TEXT NOT NULL,
    resource_name TEXT NOT NULL,
    service_account_email TEXT NOT NULL,
    role TEXT NOT NULL,
    binding_level TEXT NOT NULL, -- 'project' or 'resource'
    last_synced TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 2. IAM Collector
**File**: `internal/collector/iam.go`

```go
func (c *Collector) collectIAMBindings(ctx context.Context, projectID string) error {
    // Collect project-level IAM
    if err := c.collectProjectIAM(ctx, projectID); err != nil {
        return err
    }

    // Collect topic-level IAM
    topics, _ := c.storage.GetTopics(projectID)
    for _, topic := range topics {
        if err := c.collectTopicIAM(ctx, topic); err != nil {
            log.Printf("Failed to get IAM for topic %s: %v", topic.Name, err)
        }
    }

    // Collect subscription-level IAM
    subs, _ := c.storage.GetSubscriptions(projectID)
    for _, sub := range subs {
        if err := c.collectSubscriptionIAM(ctx, sub); err != nil {
            log.Printf("Failed to get IAM for subscription %s: %v", sub.Name, err)
        }
    }

    return nil
}

func (c *Collector) collectTopicIAM(ctx context.Context, topic *storage.Topic) error {
    client := c.getClient(ctx, topic.ProjectID)
    t := client.Topic(topic.Name)

    policy, err := t.IAM().Policy(ctx)
    if err != nil {
        return err
    }

    for _, binding := range policy.Bindings {
        for _, member := range binding.Members {
            if strings.HasPrefix(member, "serviceAccount:") {
                sa := strings.TrimPrefix(member, "serviceAccount:")

                c.storage.SaveIAMBinding(&storage.IAMBinding{
                    ResourceType:        "topic",
                    ResourceName:        topic.FullResourceName,
                    ServiceAccountEmail: sa,
                    Role:               binding.Role,
                    BindingLevel:       "resource",
                })
            }
        }
    }

    return nil
}
```

### Success Criteria:

#### Automated Verification:
- [ ] IAM bindings collected for all resources
- [ ] Both project and resource level bindings captured

#### Manual Verification:
- [ ] Can see which SAs have access to topics
- [ ] Cross-project permissions detected

---

## Phase 13: Enhanced Visualization with IAM

### Overview
Update graph building and rendering to show Service Account relationships.

### Changes Required:

#### 1. Update Graph Structure
**File**: `internal/graph/graph.go` (update)

```go
type NodeType string
const (
    NodeTypeTopic          NodeType = "topic"
    NodeTypeSubscription   NodeType = "subscription"
    NodeTypeServiceAccount NodeType = "service_account"
)

type EdgeType string
const (
    EdgeTypeSubscribes  EdgeType = "subscribes"
    EdgeTypePublishes   EdgeType = "publishes"
    EdgeTypeHasAccess   EdgeType = "has_access"
    EdgeTypeCrossProject EdgeType = "cross_project"
)
```

#### 2. Build SA Nodes and Edges
**File**: `internal/graph/builder.go` (update)

```go
func (b *Builder) buildWithIAM(projects []string) (*Graph, error) {
    // Build basic graph first
    g, err := b.Build(projects)
    if err != nil {
        return nil, err
    }

    // Add service account nodes
    sas, err := b.storage.GetAllServiceAccounts(projects)
    for _, sa := range sas {
        nodeID := fmt.Sprintf("sa_%s", sa.Email)
        g.Nodes[nodeID] = &Node{
            ID:      nodeID,
            Label:   sa.DisplayName,
            Type:    NodeTypeServiceAccount,
            Project: sa.ProjectID,
        }
        b.addToCluster(g, sa.ProjectID, nodeID)
    }

    // Add IAM binding edges
    bindings, err := b.storage.GetAllIAMBindings(projects)
    for _, binding := range bindings {
        saNodeID := fmt.Sprintf("sa_%s", binding.ServiceAccountEmail)

        var targetNodeID string
        if binding.ResourceType == "topic" {
            _, _, topicProj, topicName := parseResourceName(binding.ResourceName)
            targetNodeID = fmt.Sprintf("topic_%s_%s", topicProj, topicName)
        } else {
            _, _, subProj, subName := parseResourceName(binding.ResourceName)
            targetNodeID = fmt.Sprintf("sub_%s_%s", subProj, subName)
        }

        edgeLabel := "viewer"
        if strings.Contains(binding.Role, "publisher") {
            edgeLabel = "publisher"
        } else if strings.Contains(binding.Role, "subscriber") {
            edgeLabel = "subscriber"
        }

        g.Edges = append(g.Edges, &Edge{
            From:  saNodeID,
            To:    targetNodeID,
            Type:  EdgeTypeHasAccess,
            Label: edgeLabel,
        })
    }

    return g, nil
}
```

#### 3. Update Renderer for SAs
**File**: `internal/renderer/graphviz.go` (update)

```go
// In node rendering section
case graph.NodeTypeServiceAccount:
    gvNode.SetShape(cgraph.CircleShape)
    gvNode.SetFillColor("lightblue")
    gvNode.SetStyle("filled")

// In edge rendering section
case graph.EdgeTypeHasAccess:
    e.SetStyle("dotted")
    e.SetLabel(edge.Label)
```

### Success Criteria:

#### Automated Verification:
- [ ] Graph includes service accounts
- [ ] IAM relationships shown correctly

#### Manual Verification:
- [ ] SA → Topic → Subscription flow visible
- [ ] Permissions labeled on edges
- [ ] Visual distinction between resource types

---

## Phase 14: Performance and Caching Optimizations

### Overview
Implement parallel processing and incremental update capabilities.

### Changes Required:

#### 1. Parallel Collection
**File**: `internal/collector/parallel.go`

```go
func (c *Collector) CollectProjectsConcurrently(ctx context.Context, projects []string) error {
    var wg sync.WaitGroup
    errors := make(chan error, len(projects))

    for _, project := range projects {
        wg.Add(1)
        go func(p string) {
            defer wg.Done()
            if err := c.CollectProject(ctx, p); err != nil {
                errors <- fmt.Errorf("project %s: %w", p, err)
            }
        }(project)
    }

    wg.Wait()
    close(errors)

    var errs []error
    for err := range errors {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("collection errors: %v", errs)
    }
    return nil
}
```

#### 2. Incremental Updates
**File**: `internal/storage/sync.go`

```go
func (s *Storage) GetStaleProjects(ttl time.Duration) ([]string, error) {
    query := `
        SELECT project_id FROM projects
        WHERE last_synced < datetime('now', '-' || ? || ' hours')
    `
    rows, err := s.db.Query(query, int(ttl.Hours()))
    // ... scan and return stale projects
}

func (s *Storage) MarkAndSweep(projectID string) error {
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }

    // Mark all as potentially deleted
    tx.Exec(`UPDATE topics SET deleted = 1 WHERE project_id = ?`, projectID)
    tx.Exec(`UPDATE subscriptions SET deleted = 1 WHERE project_id = ?`, projectID)

    return tx.Commit()
}

func (s *Storage) CleanDeleted(projectID string) error {
    _, err := s.db.Exec(`DELETE FROM topics WHERE deleted = 1 AND project_id = ?`, projectID)
    _, err = s.db.Exec(`DELETE FROM subscriptions WHERE deleted = 1 AND project_id = ?`, projectID)
    return err
}
```

#### 3. Progress Reporting
**File**: `internal/cli/progress.go`

```go
func ShowProgress(current, total int, message string) {
    percent := float64(current) / float64(total) * 100
    bar := strings.Repeat("=", int(percent/2))
    fmt.Printf("\r[%-50s] %.0f%% %s", bar, percent, message)
}
```

### Success Criteria:

#### Automated Verification:
- [ ] Parallel collection faster than serial
- [ ] Incremental sync only updates changed resources
- [ ] Progress bar updates correctly

#### Manual Verification:
- [ ] Scan of 40 projects completes in <10 minutes
- [ ] Sync command only refreshes stale data
- [ ] User sees progress during long operations

---

## Phase 15: Interactive HTML Export

### Overview
Add optional interactive HTML output using Vis.js for enhanced exploration.

### Changes Required:

#### 1. HTML Template
**File**: `internal/renderer/html_template.go`

```go
const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>GCP Resource Visualization</title>
    <script src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
    <style>
        body { margin: 0; padding: 0; font-family: Arial, sans-serif; }
        #network { width: 100vw; height: 100vh; }
        #info { position: absolute; top: 10px; left: 10px; background: white;
                padding: 10px; border: 1px solid #ccc; }
    </style>
</head>
<body>
    <div id="info">
        <h3>GCP Resources</h3>
        <p>Projects: {{.ProjectCount}}</p>
        <p>Topics: {{.TopicCount}}</p>
        <p>Subscriptions: {{.SubCount}}</p>
        <p>Service Accounts: {{.SACount}}</p>
    </div>
    <div id="network"></div>
    <script>
        const nodes = new vis.DataSet({{.Nodes}});
        const edges = new vis.DataSet({{.Edges}});

        const container = document.getElementById('network');
        const data = { nodes: nodes, edges: edges };

        const options = {
            physics: {
                stabilization: { iterations: 100 },
                barnesHut: { gravitationalConstant: -8000 }
            },
            nodes: {
                font: { size: 12 }
            },
            edges: {
                smooth: { type: 'continuous' }
            },
            groups: {
                topics: { shape: 'triangle', color: 'orange' },
                subscriptions: { shape: 'box', color: 'lightgreen' },
                service_accounts: { shape: 'dot', color: 'lightblue' }
            }
        };

        const network = new vis.Network(container, data, options);

        network.on("click", function(params) {
            if (params.nodes.length > 0) {
                const nodeId = params.nodes[0];
                const node = nodes.get(nodeId);
                console.log("Selected:", node);
            }
        });
    </script>
</body>
</html>
`
```

#### 2. HTML Renderer
**File**: `internal/renderer/html.go`

```go
package renderer

import (
    "encoding/json"
    "html/template"
)

type HTMLRenderer struct{}

func (r *HTMLRenderer) Render(g *graph.Graph, output string) error {
    // Convert graph to Vis.js format
    visNodes := make([]map[string]interface{}, 0)
    visEdges := make([]map[string]interface{}, 0)

    for _, node := range g.Nodes {
        visNode := map[string]interface{}{
            "id":    node.ID,
            "label": node.Label,
            "group": string(node.Type),
            "title": fmt.Sprintf("Project: %s", node.Project),
        }
        visNodes = append(visNodes, visNode)
    }

    for i, edge := range g.Edges {
        visEdge := map[string]interface{}{
            "id":    i,
            "from":  edge.From,
            "to":    edge.To,
            "label": edge.Label,
        }

        if edge.Type == graph.EdgeTypeCrossProject {
            visEdge["dashes"] = true
            visEdge["color"] = "red"
        }

        visEdges = append(visEdges, visEdge)
    }

    // Prepare template data
    data := map[string]interface{}{
        "Nodes":        json.Marshal(visNodes),
        "Edges":        json.Marshal(visEdges),
        "ProjectCount": len(g.Clusters),
        "TopicCount":   countNodeType(g, graph.NodeTypeTopic),
        "SubCount":     countNodeType(g, graph.NodeTypeSubscription),
        "SACount":      countNodeType(g, graph.NodeTypeServiceAccount),
    }

    // Generate HTML
    tmpl, err := template.New("vis").Parse(htmlTemplate)
    if err != nil {
        return err
    }

    file, err := os.Create(output)
    if err != nil {
        return err
    }
    defer file.Close()

    return tmpl.Execute(file, data)
}
```

#### 3. CLI Integration
**File**: `internal/cli/generate.go` (update)

```go
// In Generate command
if c.Format == "html" {
    htmlRenderer := renderer.NewHTMLRenderer()
    return htmlRenderer.Render(g, c.Output)
}
```

### Success Criteria:

#### Automated Verification:
- [ ] HTML generation works: `gcp-visualizer generate --format html`
- [ ] Valid HTML produced
- [ ] JavaScript executes without errors

#### Manual Verification:
- [ ] HTML opens in browser
- [ ] Interactive zoom/pan works
- [ ] Can click nodes for details
- [ ] Performance acceptable with 10K nodes
- [ ] Cross-project relationships visible

---

## Testing Strategy

### Unit Tests:
- Test each package in isolation with mocks
- Verify database operations
- Test graph building logic
- Validate rate limiting

### Integration Tests:
- End-to-end scan and generate workflow
- Test with mock GCP APIs
- Verify cross-project relationship detection
- Test error recovery

### Manual Testing Steps:
1. Configure authentication with test projects
2. Run scan on 2-3 small projects first
3. Verify database populated correctly
4. Generate visualization and check output
5. Test with larger project set (10+)
6. Verify cross-project subscriptions shown
7. Test all output formats (SVG, PNG, HTML)
8. Test with 40 projects for scale validation

## Performance Considerations

- Parallel project scanning with configurable concurrency
- SQLite WAL mode for better concurrent access
- Rate limiting to respect GCP quotas (400 req/min)
- FDP layout engine for large graphs (handles 10K nodes)
- Incremental updates to avoid full rescans

## Migration Notes

For existing deployments (future consideration):
- Database schema versioning
- Backward compatibility for config files
- Migration scripts for schema updates

## References

- Original research: `thoughts/research/2025-11-05-gcp-visualizer-mvp-architecture.md`
- Graphviz documentation: https://graphviz.org/documentation/
- go-graphviz examples: https://github.com/goccy/go-graphviz/tree/main/examples
- Cloud Pub/Sub Go client: https://pkg.go.dev/cloud.google.com/go/pubsub