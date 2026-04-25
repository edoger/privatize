package internal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathRewriter_ExactMatch(t *testing.T) {
	r := NewPathRewriter("github.com/foo/bar",
		map[string]string{"github.com/user/pkg": "internal/pkg"},
		nil,
	)
	got, ok := r.Map("github.com/user/pkg")
	if !ok {
		t.Fatal("expected match")
	}
	assertEqual(t, got, "github.com/foo/bar/internal/pkg")
}

func TestPathRewriter_SubPackage(t *testing.T) {
	r := NewPathRewriter("github.com/foo/bar",
		map[string]string{"github.com/user/pkg": "internal/pkg"},
		nil,
	)
	got, ok := r.Map("github.com/user/pkg/sub/deep")
	if !ok {
		t.Fatal("expected match")
	}
	assertEqual(t, got, "github.com/foo/bar/internal/pkg/sub/deep")
}

func TestPathRewriter_Excluded(t *testing.T) {
	r := NewPathRewriter("github.com/foo/bar",
		map[string]string{"github.com/user/pkg": "pkg"},
		[]string{"golang.org/x"},
	)
	_, ok := r.Map("golang.org/x/text")
	if ok {
		t.Error("excluded path should not match")
	}
}

func TestPathRewriter_NoMatch(t *testing.T) {
	r := NewPathRewriter("github.com/foo/bar",
		map[string]string{"github.com/user/pkg": "pkg"},
		nil,
	)
	_, ok := r.Map("github.com/other/lib")
	if ok {
		t.Error("unrelated path should not match")
	}
}

func TestPathRewriter_VersionedPackageNotAutoDerived(t *testing.T) {
	r := NewPathRewriter("github.com/foo/bar",
		map[string]string{"github.com/user/pkg": "pkg"},
		nil,
	)
	_, ok := r.Map("github.com/user/pkg/v2")
	if ok {
		t.Error("versioned package should not auto-match without explicit rule")
	}
}

func TestPathRewriter_VersionedPackageExplicitRule(t *testing.T) {
	r := NewPathRewriter("github.com/foo/bar",
		map[string]string{
			"github.com/user/pkg":    "pkg",
			"github.com/user/pkg/v2": "pkgv2",
		},
		nil,
	)
	got, ok := r.Map("github.com/user/pkg/v2")
	if !ok {
		t.Fatal("expected match")
	}
	assertEqual(t, got, "github.com/foo/bar/pkgv2")
}

func TestPathRewriter_LongestMatchWins(t *testing.T) {
	r := NewPathRewriter("github.com/foo/bar",
		map[string]string{
			"github.com/user/pkg":    "pkg",
			"github.com/user/pkg/v2": "pkgv2",
		},
		nil,
	)
	got, ok := r.Map("github.com/user/pkg/v2/sub")
	if !ok {
		t.Fatal("expected match")
	}
	assertEqual(t, got, "github.com/foo/bar/pkgv2/sub")
}

func TestRewriteImports(t *testing.T) {
	input := []byte(`package main

import (
	"fmt"
	"github.com/user/pkg"
	"github.com/user/pkg/sub"
)

func main() {}
`)
	mapper := func(path string) (string, bool) {
		if path == "github.com/user/pkg" {
			return "github.com/foo/bar/pkg", true
		}
		if strings.HasPrefix(path, "github.com/user/pkg/") {
			return "github.com/foo/bar/pkg/" + path[len("github.com/user/pkg/"):], true
		}
		return "", false
	}

	changes, result, err := RewriteImports(input, mapper)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	if !bytes.Contains(result, []byte(`"github.com/foo/bar/pkg"`)) {
		t.Error("result should contain rewritten import")
	}
	if !bytes.Contains(result, []byte(`"fmt"`)) {
		t.Error("result should preserve unchanged imports")
	}
	if !bytes.Contains(result, []byte("func main() {}")) {
		t.Error("result should preserve non-import code")
	}
}

func TestRewriteImportsNoChanges(t *testing.T) {
	input := []byte(`package main

import "fmt"

func main() {}
`)
	mapper := func(string) (string, bool) { return "", false }

	changes, result, err := RewriteImports(input, mapper)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
	if !bytes.Equal(result, input) {
		t.Error("result should be unchanged when no imports match")
	}
}

func TestRewriteImportsAliasedImport(t *testing.T) {
	input := []byte(`package main

import (
	. "github.com/user/pkg"
	alias "github.com/user/pkg/sub"
)

func main() {}
`)
	mapper := func(path string) (string, bool) {
		if path == "github.com/user/pkg" {
			return "github.com/foo/bar/pkg", true
		}
		if strings.HasPrefix(path, "github.com/user/pkg/") {
			return "github.com/foo/bar/pkg/" + path[len("github.com/user/pkg/"):], true
		}
		return "", false
	}

	changes, _, err := RewriteImports(input, mapper)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
}

func TestRewriteFilePreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := []byte(`package test

import "github.com/user/pkg"
`)
	if err := os.WriteFile(path, content, 0755); err != nil {
		t.Fatal(err)
	}

	mapper := func(p string) (string, bool) {
		if p == "github.com/user/pkg" {
			return "github.com/foo/bar/pkg", true
		}
		return "", false
	}

	if _, err := RewriteFile(path, mapper); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("permissions: got %o, want 0755", info.Mode().Perm())
	}
}
