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

type SyncCmd struct {
	// Sync command fields will be implemented in Phase 14
}

type ConfigCmd struct {
	// Config command fields will be implemented in Phase 3
}

type VersionCmd struct {
	// Version command fields to be implemented
}

func Execute() error {
	cli := &CLI{}
	ctx := kong.Parse(cli)
	return ctx.Run()
}
