package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// GoModInfo holds parsed go.mod information.
type GoModInfo struct {
	Path      string
	GoVersion string
}

// ReadGoMod parses the go.mod file in dir and returns module metadata.
func ReadGoMod(dir string) (*GoModInfo, error) {
	path := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}
	f, err := modfile.Parse(path, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}
	if f.Module == nil {
		return nil, fmt.Errorf("go.mod has no module declaration")
	}
	info := &GoModInfo{Path: f.Module.Mod.Path}
	if f.Go != nil {
		info.GoVersion = f.Go.Version
	}
	return info, nil
}

// ReadModulePath parses the go.mod file in dir and returns the module path.
func ReadModulePath(dir string) (string, error) {
	info, err := ReadGoMod(dir)
	if err != nil {
		return "", err
	}
	return info.Path, nil
}
