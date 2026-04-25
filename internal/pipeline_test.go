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

func TestCopySourceToProject(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	// Create a fake vendored package with one Go file
	pkgDir := filepath.Join(sourceDir, "github.com/user/pkg")
	os.MkdirAll(pkgDir, 0755)
	os.WriteFile(filepath.Join(pkgDir, "lib.go"), []byte("package pkg\n"), 0644)

	cfg := &Config{
		Rules: map[string]string{"github.com/user/pkg": "internal/pkg"},
	}
	p := &Pipeline{
		ProjectDir: projectDir,
		ModulePath: "github.com/foo/bar",
		Config:     cfg,
		rewriter:   NewPathRewriter("github.com/foo/bar", cfg.Rules, nil),
	}

	copied, err := p.copySourceToProject(sourceDir)
	if err != nil {
		t.Fatalf("copySourceToProject() error: %v", err)
	}
	if len(copied) != 1 || copied[0] != "internal/pkg" {
		t.Fatalf("copied: got %v, want [internal/pkg]", copied)
	}

	dstFile := filepath.Join(projectDir, "internal/pkg/lib.go")
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}
	if string(data) != "package pkg\n" {
		t.Errorf("copied content: got %q", data)
	}
}

func TestCopySourceToProjectSkipsMissing(t *testing.T) {
	projectDir := t.TempDir()
	cfg := &Config{
		Rules: map[string]string{"github.com/user/pkg": "internal/pkg"},
	}
	p := &Pipeline{
		ProjectDir: projectDir,
		Config:     cfg,
	}

	copied, err := p.copySourceToProject(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(copied) != 0 {
		t.Errorf("copied: got %v, want empty", copied)
	}
}
