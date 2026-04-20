package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	HTTPClient HTTPClientConfig `mapstructure:"http_client"`
	Fetch      FetchConfig      `mapstructure:"fetch"`
	Link       LinkConfig       `mapstructure:"link"`
}

type ServerConfig struct {
	Addr string `mapstructure:"addr"` // bind address and port for the HTTP server
}

// HTTPClientConfig bounds the global HTTP client
type HTTPClientConfig struct {
	MaxRedirects int           `mapstructure:"max_redirects" validate:"gte=1"` // maximum number of redirects to follow
	Timeout      time.Duration `mapstructure:"timeout" validate:"gte=0"`       // 0 = no timeout; YAML: Go duration strings (e.g. "60s", "2m")
	UserAgent    string        `mapstructure:"user_agent" validate:"-"`        // user-agent string for HTTP client, optional; empty falls back in HTTP client if needed
}

// FetchConfig bounds the fetch module.
type FetchConfig struct {
	MaxBodyBytes int64 `mapstructure:"max_body_bytes" validate:"gt=0"` // max bytes to read from the response body
}

// LinkConfig bounds the link module.
type LinkConfig struct {
	Workers           int   `mapstructure:"workers" validate:"gte=1"`             // max concurrent accessibility probes
	MaxProbeBodyBytes int64 `mapstructure:"max_probe_body_bytes" validate:"gt=0"` // max bytes read from probe responses
}

// Load merges defaults with YAML from disk.
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		return Default(), nil
	}
	v := viper.New()
	setDefaults(v)
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config: read %q: %w", configPath, err)
	}
	cfg, err := unmarshalViper(v)
	if err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	d := Default()
	v.SetDefault("server.addr", d.Server.Addr)
	v.SetDefault("http_client.max_redirects", d.HTTPClient.MaxRedirects)
	v.SetDefault("http_client.timeout", d.HTTPClient.Timeout.String())
	v.SetDefault("http_client.user_agent", d.HTTPClient.UserAgent)
	v.SetDefault("fetch.max_body_bytes", d.Fetch.MaxBodyBytes)
	v.SetDefault("link.workers", d.Link.Workers)
	v.SetDefault("link.max_probe_body_bytes", d.Link.MaxProbeBodyBytes)
}

func unmarshalViper(v *viper.Viper) (*Config, error) {
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}
	return cfg, nil
}

// Default returns a new config with built-in defaults (safe to mutate for tests).
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Addr: "127.0.0.1:8080",
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
			MaxProbeBodyBytes: 64 << 10, // 64KB
		},
	}
}
