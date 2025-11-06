package cli

import (
	"context"

	"github.com/alecthomas/kong"
)

// CLI is the main CLI structure with embedded context
type CLI struct {
	ctx context.Context // Store context for commands to use

	Scan     ScanCmd     `cmd:"scan" help:"Scan GCP projects for resources"`
	Generate GenerateCmd `cmd:"generate" help:"Generate visualization from cached data"`
	Sync     SyncCmd     `cmd:"sync" help:"Smart refresh of stale resources"`
	Config   ConfigCmd   `cmd:"config" help:"Manage configuration"`
	Version  VersionCmd  `cmd:"version" help:"Show version"`
}

// Context returns the CLI's context for use by commands.
// This allows commands to access the context without directly accessing
// the unexported ctx field.
func (c *CLI) Context() context.Context {
	return c.ctx
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

type SyncCmd struct {
	// Sync command fields will be implemented in Phase 14
}

type ConfigCmd struct {
	// Config command fields will be implemented in Phase 3
}

type VersionCmd struct {
	// Version command fields to be implemented
}

// ExecuteWithContext executes the CLI with a context that can be cancelled
func ExecuteWithContext(ctx context.Context) error {
	cli := &CLI{ctx: ctx}
	kongCtx := kong.Parse(cli)

	// Bind CLI instance so commands can access the context
	return kongCtx.Run(cli)
}

// Execute executes the CLI with a background context (for backwards compatibility)
func Execute() error {
	return ExecuteWithContext(context.Background())
}
