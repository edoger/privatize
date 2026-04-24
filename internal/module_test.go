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
