package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPipelineMapper(t *testing.T) {
	cfg := &Config{
		Imports: []string{"github.com/user/pkg"},
		Rules:   map[string]string{"github.com/user/pkg": "pkg"},
		Exclude: []string{"golang.org/x"},
	}
	p := &Pipeline{
		ProjectDir: t.TempDir(),
		ModulePath: "github.com/foo/bar",
		Config:     cfg,
		rewriter:   NewPathRewriter("github.com/foo/bar", cfg.Rules, cfg.Exclude),
	}

	got, ok := p.Mapper("github.com/user/pkg")
	if !ok {
		t.Fatal("expected match")
	}
	assertEqual(t, got, "github.com/foo/bar/pkg")

	_, ok = p.Mapper("golang.org/x/text")
	if ok {
		t.Error("excluded should not match")
	}
}

func TestNewPipeline(t *testing.T) {
	dir := t.TempDir()
	// Write go.mod
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/foo/bar\n\ngo 1.22\n"), 0644)
	// Write config
	cfgContent := `
imports:
  - github.com/user/pkg
rules:
  github.com/user/pkg: pkg
exclude:
  - golang.org/x
`
	os.WriteFile(filepath.Join(dir, ".privatize.yaml"), []byte(cfgContent), 0644)

	p, err := NewPipeline(dir)
	if err != nil {
		t.Fatalf("NewPipeline() error: %v", err)
	}
	assertEqual(t, p.ModulePath, "github.com/foo/bar")
	assertEqual(t, p.ProjectDir, dir)
	if p.Config == nil {
		t.Fatal("Config should not be nil")
	}
	if p.rewriter == nil {
		t.Fatal("rewriter should not be nil")
	}
}

func TestNewPipelineMissingGoMod(t *testing.T) {
	dir := t.TempDir()
	_, err := NewPipeline(dir)
	if err == nil {
		t.Fatal("expected error for missing go.mod")
	}
}

func TestNewPipelineMissingConfig(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/foo/bar\n\ngo 1.22\n"), 0644)

	_, err := NewPipeline(dir)
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestNewPipelineInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/foo/bar\n\ngo 1.22\n"), 0644)
	// Config with import but no matching rule
	cfgContent := `
imports:
  - github.com/user/pkg
rules: {}
`
	os.WriteFile(filepath.Join(dir, ".privatize.yaml"), []byte(cfgContent), 0644)

	_, err := NewPipeline(dir)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

