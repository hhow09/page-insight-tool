package config

import "time"

// Config is the root settings object. More fields are added in later phases.
type Config struct {
	Server ServerConfig
	Fetch  FetchConfig
	Link   LinkConfig
}

type ServerConfig struct {
	Addr string
}

// FetchConfig bounds the primary page HTTP GET.
type FetchConfig struct {
	MaxBodyBytes int64         `validate:"gt=0"`
	MaxRedirects int           `validate:"gte=1"`
	Timeout      time.Duration `validate:"gt=0"`
	UserAgent    string        // optional; empty falls back in HTTP client if needed
}

// LinkConfig controls URL resolution, classification, and concurrent link probes.
type LinkConfig struct {
	Workers           int           `validate:"gte=1"` // max concurrent accessibility probes
	PerLinkTimeout    time.Duration `validate:"gt=0"`  // per request deadline (HEAD/GET)
	MaxProbeBodyBytes int64         `validate:"gt=0"`  // max bytes read from probe responses
	MaxURLsToCheck    int           `validate:"gte=0"` // max unique http(s) URLs to probe (0 = unlimited)
	UserAgent         string        // optional User-Agent for HEAD/GET probes
}

// Default returns a new config with built-in defaults (safe to mutate for tests).
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Addr: ":8080",
		},
		Fetch: FetchConfig{
			MaxBodyBytes: 16 << 20, // 16MB
			MaxRedirects: 10,
			Timeout:      60 * time.Second,
			UserAgent:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
		},
		Link: LinkConfig{
			Workers:           8,
			PerLinkTimeout:    3 * time.Second,
			MaxProbeBodyBytes: 64 << 10,
			MaxURLsToCheck:    1000,
			UserAgent:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
		},
	}
}
