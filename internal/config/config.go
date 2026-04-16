package config

import "time"

// Config is the root settings object. More fields are added in later phases.
type Config struct {
	Fetch FetchConfig
}

// FetchConfig bounds the primary page HTTP GET.
type FetchConfig struct {
	MaxBodyBytes int64
	MaxRedirects int
	Timeout      time.Duration
	UserAgent    string
}

// Default returns a new config with built-in defaults (safe to mutate for tests).
func Default() *Config {
	return &Config{
		Fetch: FetchConfig{
			MaxBodyBytes: 16 << 20, // 16MB
			MaxRedirects: 10,
			Timeout:      60 * time.Second,
			UserAgent:    "page-insight-tool/1.0",
		},
	}
}
