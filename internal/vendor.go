package internal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ModuleInfo holds parsed module metadata from vendor/modules.txt.
type ModuleInfo struct {
	Path     string
	Version  string
	Packages []string
}

// Workspace manages a temporary Go module used to vendor dependencies.
type Workspace struct {
	Dir    string
	Source string
}

// CreateWorkspace creates a temp directory with go.mod and main.go that
// blank-import the given packages, ready for "go mod vendor".
func CreateWorkspace(goVersion string, imports []string) (*Workspace, error) {
	dir, err := os.MkdirTemp("", "privatize-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	modContent := fmt.Sprintf("module _privatize_ws\n\ngo %s\n", goVersion)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(modContent), 0644); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	var blankImports []string
	for _, imp := range imports {
		blankImports = append(blankImports, fmt.Sprintf("\t_ %q", imp))
	}
	mainContent := fmt.Sprintf("package main\n\nimport (\n%s\n)\n", strings.Join(blankImports, "\n"))
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainContent), 0644); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	return &Workspace{Dir: dir}, nil
}

// Vendor runs "go mod tidy" and "go mod vendor", then renames the vendor
// directory to Source and cleans up auxiliary files.
func (w *Workspace) Vendor() error {
	run := func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Dir = w.Dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %s: %w", name, string(out), err)
		}
		return nil
	}

	if err := run("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}
	if err := run("go", "mod", "vendor"); err != nil {
		return fmt.Errorf("go mod vendor: %w", err)
	}

	for _, f := range []string{"go.mod", "go.sum", "main.go"} {
		os.Remove(filepath.Join(w.Dir, f))
	}

	vendorDir := filepath.Join(w.Dir, "vendor")
	sourceDir := filepath.Join(w.Dir, "source")
	if err := os.Rename(vendorDir, sourceDir); err != nil {
		return fmt.Errorf("rename vendor to source: %w", err)
	}
	w.Source = sourceDir
	return nil
}

// ParseModules reads vendor/modules.txt and returns module metadata.
func (w *Workspace) ParseModules() ([]ModuleInfo, error) {
	path := filepath.Join(w.Source, "modules.txt")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var modules []ModuleInfo
	var current *ModuleInfo

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if rest, ok := strings.CutPrefix(line, "# "); ok {
			parts := strings.SplitN(rest, " ", 2)
			if len(parts) == 2 {
				modules = append(modules, ModuleInfo{Path: parts[0], Version: parts[1]})
				current = &modules[len(modules)-1]
			}
		} else if current != nil && !strings.HasPrefix(line, "## ") && line != "" {
			current.Packages = append(current.Packages, line)
		}
	}
	return modules, scanner.Err()
}

// Cleanup removes the workspace directory.
func (w *Workspace) Cleanup() {
	os.RemoveAll(w.Dir)
}
