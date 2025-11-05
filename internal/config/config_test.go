package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "fdp", cfg.Visualization.Layout)
	assert.Equal(t, 10.0, cfg.RateLimits.RequestsPerSecond)
	assert.Equal(t, 1, cfg.Cache.TTLHours)
	assert.Equal(t, 24, cfg.Cache.MaxAgeHours)
	assert.Equal(t, "svg", cfg.Visualization.OutputFormat)
	assert.Equal(t, 5, cfg.RateLimits.MaxConcurrent)
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Test loading config when no file exists
	// Set temp config path to avoid conflicts
	t.Setenv("GCP_VISUALIZER_CONFIG", filepath.Join(t.TempDir(), "nonexistent.yaml"))

	cfg, err := Load()
	require.NoError(t, err)

	// Should return default config
	assert.Equal(t, "fdp", cfg.Visualization.Layout)
	assert.Equal(t, 10.0, cfg.RateLimits.RequestsPerSecond)
}

func TestLoadConfig_FromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	yamlContent := `
organization_id: "test-org"
projects:
  - "project-1"
  - "project-2"
cache:
  ttl_hours: 2
  max_age_hours: 48
visualization:
  layout: "dot"
  output_format: "png"
  include_icons: true
  show_iam_details: false
rate_limits:
  requests_per_second: 5.0
  max_concurrent: 3
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	t.Setenv("GCP_VISUALIZER_CONFIG", configPath)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "test-org", cfg.OrganizationID)
	assert.Equal(t, []string{"project-1", "project-2"}, cfg.Projects)
	assert.Equal(t, 2, cfg.Cache.TTLHours)
	assert.Equal(t, 48, cfg.Cache.MaxAgeHours)
	assert.Equal(t, "dot", cfg.Visualization.Layout)
	assert.Equal(t, "png", cfg.Visualization.OutputFormat)
	assert.True(t, cfg.Visualization.IncludeIcons)
	assert.False(t, cfg.Visualization.ShowIAMDetails)
	assert.Equal(t, 5.0, cfg.RateLimits.RequestsPerSecond)
	assert.Equal(t, 3, cfg.RateLimits.MaxConcurrent)
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	// Create a temporary config file with base values
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	yamlContent := `
organization_id: "yaml-org"
projects:
  - "yaml-project"
rate_limits:
  requests_per_second: 10.0
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	t.Setenv("GCP_VISUALIZER_CONFIG", configPath)
	t.Setenv("GCP_VISUALIZER_ORGANIZATION_ID", "env-org")
	t.Setenv("GCP_VISUALIZER_REQUESTS_PER_SECOND", "20.0")

	cfg, err := Load()
	require.NoError(t, err)

	// Environment variable should override YAML
	assert.Equal(t, "env-org", cfg.OrganizationID)
	assert.Equal(t, 20.0, cfg.RateLimits.RequestsPerSecond)

	// Values not overridden by env should come from YAML
	assert.Equal(t, []string{"yaml-project"}, cfg.Projects)
}

func TestConfigPrecedence(t *testing.T) {
	// Test: defaults < YAML < env vars
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	yamlContent := `
cache:
  ttl_hours: 5
visualization:
  layout: "neato"
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	t.Setenv("GCP_VISUALIZER_CONFIG", configPath)
	t.Setenv("GCP_VISUALIZER_LAYOUT", "dot")

	cfg, err := Load()
	require.NoError(t, err)

	// Default value (not in YAML or env)
	assert.Equal(t, 10.0, cfg.RateLimits.RequestsPerSecond)

	// YAML value (not overridden by env)
	assert.Equal(t, 5, cfg.Cache.TTLHours)

	// Env value (overrides YAML)
	assert.Equal(t, "dot", cfg.Visualization.Layout)
}

func TestConfigPath_Default(t *testing.T) {
	// Clear env var to test default
	t.Setenv("GCP_VISUALIZER_CONFIG", "")

	path := ConfigPath()

	// Should be in ~/.config/gcp-visualizer/config.yaml
	assert.Contains(t, path, ".config/gcp-visualizer/config.yaml")
}

func TestConfigPath_CustomEnv(t *testing.T) {
	customPath := "/custom/path/config.yaml"
	t.Setenv("GCP_VISUALIZER_CONFIG", customPath)

	path := ConfigPath()
	assert.Equal(t, customPath, path)
}

func TestConfigSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	t.Setenv("GCP_VISUALIZER_CONFIG", configPath)

	cfg := &Config{
		OrganizationID: "save-test-org",
		Projects:       []string{"project-a", "project-b"},
		Cache: Cache{
			TTLHours:    3,
			MaxAgeHours: 72,
		},
		Visualization: Visual{
			Layout:       "fdp",
			OutputFormat: "svg",
		},
		RateLimits: Limits{
			RequestsPerSecond: 15.0,
			MaxConcurrent:     10,
		},
	}

	err := cfg.Save()
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Load it back and verify
	loadedCfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "save-test-org", loadedCfg.OrganizationID)
	assert.Equal(t, []string{"project-a", "project-b"}, loadedCfg.Projects)
	assert.Equal(t, 3, loadedCfg.Cache.TTLHours)
	assert.Equal(t, 72, loadedCfg.Cache.MaxAgeHours)
}

func TestConfigSave_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "nested", "dir", "config.yaml")

	t.Setenv("GCP_VISUALIZER_CONFIG", configPath)

	cfg := DefaultConfig()
	err := cfg.Save()
	require.NoError(t, err)

	// Verify directory was created
	_, err = os.Stat(filepath.Dir(configPath))
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	invalidYAML := `
this is not: valid: yaml: content
  bad indentation
`

	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	t.Setenv("GCP_VISUALIZER_CONFIG", configPath)

	_, err = Load()
	require.Error(t, err)
}
