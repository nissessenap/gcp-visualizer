package config

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
