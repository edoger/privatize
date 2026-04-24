package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the privatize tool configuration.
type Config struct {
	Imports []string          `yaml:"imports"`
	Rules   map[string]string `yaml:"rules"`
	Exclude []string          `yaml:"exclude"`
}

// Load reads and parses a YAML config file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Validate checks that every import has a corresponding rule and that
// rule values are safe relative paths without traversal sequences.
func (c *Config) Validate() error {
	for _, imp := range c.Imports {
		if _, ok := c.Rules[imp]; !ok {
			return fmt.Errorf("import %q has no matching rule", imp)
		}
	}
	for orig, target := range c.Rules {
		if target == "" {
			return fmt.Errorf("rule %q: target path must not be empty", orig)
		}
		if filepath.IsAbs(target) {
			return fmt.Errorf("rule %q: target path must be relative, got %q", orig, target)
		}
		cleaned := filepath.Clean(target)
		if cleaned == "." || strings.Contains(cleaned, "..") {
			return fmt.Errorf("rule %q: target path must not contain \"..\" segments, got %q", orig, target)
		}
	}
	return nil
}
