package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	content := `
imports:
  - github.com/user/pkg1
  - github.com/user/pkg2
rules:
  github.com/user/pkg1: pkg1
  github.com/user/pkg2: internal/pkg2
exclude:
  - golang.org/x
`
	path := filepath.Join(dir, ".privatize.yaml")
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Imports) != 2 {
		t.Errorf("Imports: got %d, want 2", len(cfg.Imports))
	}
	if cfg.Rules["github.com/user/pkg1"] != "pkg1" {
		t.Errorf("Rules[pkg1]: got %q, want %q", cfg.Rules["github.com/user/pkg1"], "pkg1")
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "golang.org/x" {
		t.Errorf("Exclude: got %v", cfg.Exclude)
	}
}

func TestLoadMinimalConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".privatize.yaml")
	os.WriteFile(path, []byte("{}"), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Imports != nil || cfg.Rules != nil || cfg.Exclude != nil {
		t.Error("empty config should have nil fields")
	}
}

func TestValidateMissingRuleForImport(t *testing.T) {
	cfg := &Config{
		Imports: []string{"github.com/user/pkg1"},
		Rules:   map[string]string{},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for import without rule")
	}
}

func TestValidateOK(t *testing.T) {
	cfg := &Config{
		Imports: []string{"github.com/user/pkg1"},
		Rules:   map[string]string{"github.com/user/pkg1": "pkg1"},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error: %v", err)
	}
}
