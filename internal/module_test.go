package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadModulePath(t *testing.T) {
	dir := t.TempDir()
	content := "module github.com/foo/bar\n\ngo 1.26.2\n"
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0644)

	path, err := ReadModulePath(dir)
	if err != nil {
		t.Fatalf("ReadModulePath() error: %v", err)
	}
	if path != "github.com/foo/bar" {
		t.Errorf("got %q, want %q", path, "github.com/foo/bar")
	}
}

func TestReadModulePathNoFile(t *testing.T) {
	_, err := ReadModulePath(t.TempDir())
	if err == nil {
		t.Error("expected error for missing go.mod")
	}
}

func TestReadGoModNoModule(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("go 1.22\n"), 0644)
	_, err := ReadGoMod(dir)
	if err == nil {
		t.Error("expected error for go.mod without module declaration")
	}
}

func TestReadGoModNoGoVersion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/foo/bar\n"), 0644)
	info, err := ReadGoMod(dir)
	if err != nil {
		t.Fatalf("ReadGoMod() error: %v", err)
	}
	if info.GoVersion != "" {
		t.Errorf("GoVersion: got %q, want empty", info.GoVersion)
	}
}
