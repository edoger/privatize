package internal

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Pipeline orchestrates the full privatize workflow: vendor, rewrite, copy.
type Pipeline struct {
	ProjectDir string
	ModulePath string
	GoVersion  string
	Config     *Config
	rewriter   *PathRewriter
}

// NewPipeline initializes a Pipeline by reading go.mod and config from projectDir.
func NewPipeline(projectDir string) (*Pipeline, error) {
	modInfo, err := ReadGoMod(projectDir)
	if err != nil {
		return nil, fmt.Errorf("read module: %w", err)
	}
	cfgPath := filepath.Join(projectDir, ".privatize.yaml")
	cfg, err := Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return &Pipeline{
		ProjectDir: projectDir,
		ModulePath: modInfo.Path,
		GoVersion:  modInfo.GoVersion,
		Config:     cfg,
		rewriter:   NewPathRewriter(modInfo.Path, cfg.Rules, cfg.Exclude),
	}, nil
}

// Mapper delegates to the underlying PathRewriter.
func (p *Pipeline) Mapper(importPath string) (string, bool) {
	return p.rewriter.Map(importPath)
}

// Result holds the outcome of a pipeline run.
type Result struct {
	Rewrites []Change
	Copied   []string
	Modified []string
}

// Run executes the full pipeline. When dryRun is true, source files are
// rewritten in the workspace but not copied back to the project.
func (p *Pipeline) Run(dryRun bool) (*Result, error) {
	ws, err := CreateWorkspace(p.GoVersion, p.Config.Imports)
	if err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}
	defer ws.Cleanup()

	if err := ws.Vendor(); err != nil {
		return nil, fmt.Errorf("vendor: %w", err)
	}

	result := &Result{}
	if err := p.rewriteSourceImports(ws.Source, result); err != nil {
		return nil, fmt.Errorf("rewrite source: %w", err)
	}

	if !dryRun {
		copied, err := p.copySourceToProject(ws.Source)
		if err != nil {
			return nil, err
		}
		result.Copied = copied
		if err := p.rewriteProjectImports(result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// rewriteSourceImports walks the workspace source tree and rewrites imports.
func (p *Pipeline) rewriteSourceImports(sourceDir string, result *Result) error {
	return filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		changes, err := RewriteFile(path, p.Mapper)
		if err != nil {
			return err
		}
		result.Rewrites = append(result.Rewrites, changes...)
		return nil
	})
}

// copySourceToProject copies vendored packages into the project directory.
func (p *Pipeline) copySourceToProject(sourceDir string) ([]string, error) {
	var copied []string
	for originalPath, relTarget := range p.Config.Rules {
		srcDir := filepath.Join(sourceDir, originalPath)
		dstDir := filepath.Join(p.ProjectDir, relTarget)
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			continue
		}
		if err := os.CopyFS(dstDir, os.DirFS(srcDir)); err != nil {
			return nil, fmt.Errorf("copy %s -> %s: %w", srcDir, dstDir, err)
		}
		copied = append(copied, relTarget)
	}
	return copied, nil
}

// rewriteProjectImports walks the project tree and rewrites imports.
func (p *Pipeline) rewriteProjectImports(result *Result) error {
	return filepath.WalkDir(p.ProjectDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		changes, err := RewriteFile(path, p.Mapper)
		if err != nil {
			return err
		}
		for _, c := range changes {
			result.Modified = append(result.Modified, c.File)
		}
		return nil
	})
}
