package cli

import (
	"fmt"
)

func (c *ScanCmd) Run() error {
	// TODO: Implement scan logic in Phase 6
	fmt.Printf("Scanning projects: %v\n", c.Projects)
	if c.Force {
		fmt.Println("Force refresh enabled")
	}
	return nil
}

func (c *GenerateCmd) Run() error {
	// TODO: Implement generate logic in Phase 10
	fmt.Printf("Generating %s output to %s\n", c.Format, c.Output)
	if len(c.Projects) > 0 {
		fmt.Printf("Filtering by projects: %v\n", c.Projects)
	}
	fmt.Printf("Using layout engine: %s\n", c.Layout)
	return nil
}

func (c *SyncCmd) Run() error {
	// TODO: Implement sync logic in Phase 14
	fmt.Println("Sync command (to be implemented)")
	return nil
}

func (c *ConfigCmd) Run() error {
	// TODO: Implement config management in Phase 3
	fmt.Println("Config management (to be implemented)")
	return nil
}

func (c *VersionCmd) Run() error {
	// TODO: Add proper version from build
	fmt.Println("gcp-visualizer version: dev")
	return nil
}
