package config

import "time"

// Config is the root settings object. More fields are added in later phases.
type Config struct {
	Server     ServerConfig
	HTTPClient HTTPClientConfig
	Fetch      FetchConfig
	Link       LinkConfig
}

type ServerConfig struct {
	Addr string
}

// HTTPClientConfig bounds the global HTTP client
type HTTPClientConfig struct {
	MaxRedirects int           `validate:"gte=1"`
	Timeout      time.Duration // 0 means no timeout
	UserAgent    string        // optional; empty falls back in HTTP client if needed
}

// FetchConfig bounds the fetch module.
type FetchConfig struct {
	MaxBodyBytes int64 `validate:"gt=0"` // max bytes to read from the response body
}

// LinkConfig bounds the link module.
type LinkConfig struct {
	Workers           int   `validate:"gte=1"` // max concurrent accessibility probes
	MaxProbeBodyBytes int64 `validate:"gt=0"`  // max bytes read from probe responses
}

// Default returns a new config with built-in defaults (safe to mutate for tests).
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Addr: ":8080",
		},
		HTTPClient: HTTPClientConfig{
			MaxRedirects: 10,
			Timeout:      60 * time.Second,
			UserAgent:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
		},
		Fetch: FetchConfig{
			MaxBodyBytes: 16 << 20, // 16MB
		},
		Link: LinkConfig{
			Workers:           8,
			MaxProbeBodyBytes: 64 << 10,
		},
	}
}
