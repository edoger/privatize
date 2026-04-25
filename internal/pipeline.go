package internal

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Pipeline phase indices shared between Pipeline and RunModel.
const (
	PhaseSetup   = 0
	PhaseVendor  = 1
	PhaseRewrite = 2
	PhaseMove    = 3
	PhaseCleanup = 4
)

// PhaseNames holds the display names for each pipeline phase, in order.
var PhaseNames = [...]string{"Setup", "Vendor", "Rewrite", "Move", "Cleanup"}

// ProgressFunc is called when a pipeline phase changes state.
type ProgressFunc func(phaseIndex int, status string)

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
// The progress callback is called at each phase transition with the phase
// index and its new status ("active", "done", or "error").
func (p *Pipeline) Run(dryRun bool, progress ProgressFunc) (result *Result, err error) {
	progress(PhaseSetup, "active")
	ws, err := CreateWorkspace(p.GoVersion, p.Config.Imports)
	if err != nil {
		progress(PhaseSetup, "error")
		return nil, fmt.Errorf("setup: %w", err)
	}
	progress(PhaseSetup, "done")

	defer func() {
		progress(PhaseCleanup, "active")
		if cleanupErr := ws.Cleanup(); cleanupErr != nil {
			progress(PhaseCleanup, "error")
			if err == nil {
				err = fmt.Errorf("cleanup workspace: %w", cleanupErr)
			}
			return
		}
		progress(PhaseCleanup, "done")
	}()

	progress(PhaseVendor, "active")
	if err := ws.Vendor(); err != nil {
		progress(PhaseVendor, "error")
		return nil, fmt.Errorf("vendor: %w", err)
	}
	progress(PhaseVendor, "done")

	progress(PhaseRewrite, "active")
	result = &Result{}
	if err := p.rewriteSourceImports(ws.Source, result); err != nil {
		progress(PhaseRewrite, "error")
		return nil, fmt.Errorf("rewrite source: %w", err)
	}
	progress(PhaseRewrite, "done")

	if !dryRun {
		progress(PhaseMove, "active")
		copied, err := p.copySourceToProject(ws.Source)
		if err != nil {
			progress(PhaseMove, "error")
			return nil, err
		}
		result.Copied = copied
		if err := p.rewriteProjectImports(result); err != nil {
			progress(PhaseMove, "error")
			return nil, err
		}
		progress(PhaseMove, "done")
	}
	return result, nil
}

// rewriteSourceImports walks the workspace source tree and rewrites imports.
func (p *Pipeline) rewriteSourceImports(sourceDir string, result *Result) error {
	return filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
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
		if _, err := os.Stat(srcDir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat source %s: %w", srcDir, err)
		}
		if err := os.RemoveAll(dstDir); err != nil {
			return nil, fmt.Errorf("remove %s: %w", dstDir, err)
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
	seen := map[string]struct{}{}
	return filepath.WalkDir(p.ProjectDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		changes, err := RewriteFile(path, p.Mapper)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			if _, ok := seen[path]; !ok {
				seen[path] = struct{}{}
				result.Modified = append(result.Modified, path)
			}
		}
		return nil
	})
}
