package config

import (
	"os"
	"path/filepath"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

type Config struct {
	OrganizationID string   `yaml:"organization_id" envconfig:"ORGANIZATION_ID"`
	Projects       []string `yaml:"projects" envconfig:"PROJECTS"`
	Cache          Cache    `yaml:"cache"`
	Visualization  Visual   `yaml:"visualization"`
	RateLimits     Limits   `yaml:"rate_limits"`
}

type Cache struct {
	TTLHours    int `yaml:"ttl_hours" envconfig:"TTL_HOURS"`
	MaxAgeHours int `yaml:"max_age_hours" envconfig:"MAX_AGE_HOURS"`
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
	// Process top-level fields
	if err := envconfig.Process("GCP_VISUALIZER", cfg); err != nil {
		return nil, err
	}

	// Process nested structs with the same prefix to support flat env var names
	if err := envconfig.Process("GCP_VISUALIZER", &cfg.Cache); err != nil {
		return nil, err
	}
	if err := envconfig.Process("GCP_VISUALIZER", &cfg.Visualization); err != nil {
		return nil, err
	}
	if err := envconfig.Process("GCP_VISUALIZER", &cfg.RateLimits); err != nil {
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
