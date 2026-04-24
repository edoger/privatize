package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseModulesTxt(t *testing.T) {
	dir := t.TempDir()
	content := `# golang.org/x/text v0.32.0
## explicit; go 1.24.0
golang.org/x/text/secure/bidirule
golang.org/x/text/transform
golang.org/x/text/unicode/bidi
golang.org/x/text/unicode/norm
# github.com/user/pkg v1.2.3
## explicit; go 1.26.0
github.com/user/pkg
github.com/user/pkg/sub
`
	srcDir := filepath.Join(dir, "source")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "modules.txt"), []byte(content), 0644)

	w := &Workspace{Dir: dir, Source: srcDir}
	modules, err := w.ParseModules()
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(modules))
	}
	if modules[0].Path != "golang.org/x/text" {
		t.Errorf("module path: got %q", modules[0].Path)
	}
	if modules[0].Version != "v0.32.0" {
		t.Errorf("module version: got %q", modules[0].Version)
	}
	if len(modules[1].Packages) != 2 {
		t.Errorf("packages: got %d", len(modules[1].Packages))
	}
}

func TestCreateWorkspaceFiles(t *testing.T) {
	w, err := CreateWorkspace("1.26.2", []string{"github.com/user/pkg"})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(w.Dir)

	modData, _ := os.ReadFile(filepath.Join(w.Dir, "go.mod"))
	if !strings.Contains(string(modData), "module _privatize_ws") {
		t.Error("go.mod should contain module declaration")
	}

	mainData, _ := os.ReadFile(filepath.Join(w.Dir, "main.go"))
	if !strings.Contains(string(mainData), `_ "github.com/user/pkg"`) {
		t.Error("main.go should contain blank import")
	}
}

func TestCleanup(t *testing.T) {
	w, _ := CreateWorkspace("1.26.2", []string{"github.com/user/pkg"})
	dir := w.Dir
	w.Cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("workspace directory should be removed")
	}
}
