package proxy

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ModelConfig defines the configuration for a single model endpoint.
type ModelConfig struct {
	// Cmd is the command to launch the model server (e.g., llama.cpp server).
	Cmd string `yaml:"cmd" json:"cmd"`

	// Proxy is the upstream address to forward requests to once the model is running.
	Proxy string `yaml:"proxy" json:"proxy"`

	// Aliases are alternative names that map to this model.
	Aliases []string `yaml:"aliases,omitempty" json:"aliases,omitempty"`

	// CheckEndpoint is the URL path used to verify the model server is ready.
	// Defaults to "/health" if not specified.
	CheckEndpoint string `yaml:"checkEndpoint,omitempty" json:"checkEndpoint,omitempty"`

	// TTL is the idle time-to-live before the model process is stopped.
	// Accepts Go duration strings like "10m", "1h".
	TTL string `yaml:"ttl,omitempty" json:"ttl,omitempty"`

	// Env is a list of additional environment variables to pass to the model process.
	Env []string `yaml:"env,omitempty" json:"env,omitempty"`
}

// TTLDuration parses the TTL string into a time.Duration.
// Returns 0 if TTL is empty, or an error if the format is invalid.
func (m *ModelConfig) TTLDuration() (time.Duration, error) {
	if m.TTL == "" {
		return 0, nil
	}
	return time.ParseDuration(m.TTL)
}

// GetCheckEndpoint returns the health check endpoint, defaulting to "/health".
func (m *ModelConfig) GetCheckEndpoint() string {
	if m.CheckEndpoint == "" {
		return "/health"
	}
	return m.CheckEndpoint
}

// Config is the top-level configuration for llama-swap.
type Config struct {
	// ListenAddress is the address the proxy server listens on.
	// Defaults to ":8080".
	ListenAddress string `yaml:"listen" json:"listen"`

	// HealthCheckTimeout is the maximum duration to wait for a model to become ready.
	HealthCheckTimeout string `yaml:"healthCheckTimeout,omitempty" json:"healthCheckTimeout,omitempty"`

	// LogLevel controls verbosity: "debug", "info", "warn", "error".
	LogLevel string `yaml:"logLevel,omitempty" json:"logLevel,omitempty"`

	// Models is a map of model name to its configuration.
	Models map[string]ModelConfig `yaml:"models" json:"models"`
}

// GetListenAddress returns the listen address, defaulting to ":8080".
func (c *Config) GetListenAddress() string {
	if c.ListenAddress == "" {
		return ":8080"
	}
	return c.ListenAddress
}

// GetHealthCheckTimeout parses and returns the health check timeout duration.
// Defaults to 30 seconds if not specified.
func (c *Config) GetHealthCheckTimeout() (time.Duration, error) {
	if c.HealthCheckTimeout == "" {
		return 30 * time.Second, nil
	}
	return time.ParseDuration(c.HealthCheckTimeout)
}

// Validate checks that the configuration is well-formed.
func (c *Config) Validate() error {
	if len(c.Models) == 0 {
		return fmt.Errorf("config must define at least one model")
	}
	for name, model := range c.Models {
		if model.Cmd == "" {
			return fmt.Errorf("model %q: cmd is required", name)
		}
		if model.Proxy == "" {
			return fmt.Errorf("model %q: proxy address is required", name)
		}
		if model.TTL != "" {
			if _, err := time.ParseDuration(model.TTL); err != nil {
				return fmt.Errorf("model %q: invalid ttl %q: %w", name, model.TTL, err)
			}
		}
	}
	return nil
}

// LoadConfig reads and parses a YAML configuration file from the given path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}
