package internal

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"unicode"
)

// PathRewriter maps external import paths to internal module paths.
type PathRewriter struct {
	ModulePath string
	Rules      map[string]string
	Exclude    []string
}

// NewPathRewriter creates a rewriter with the given module path, mapping rules,
// and exclusion prefixes.
func NewPathRewriter(modulePath string, rules map[string]string, exclude []string) *PathRewriter {
	return &PathRewriter{
		ModulePath: modulePath,
		Rules:      rules,
		Exclude:    exclude,
	}
}

// Map resolves an import path to an internal path. It returns the rewritten
// path and true on match, or empty string and false otherwise.
func (r *PathRewriter) Map(importPath string) (string, bool) {
	for _, prefix := range r.Exclude {
		if strings.HasPrefix(importPath, prefix) {
			return "", false
		}
	}

	bestKey := ""
	for key := range r.Rules {
		if importPath == key {
			if len(key) > len(bestKey) {
				bestKey = key
			}
			continue
		}
		if !strings.HasPrefix(importPath, key+"/") {
			continue
		}
		suffix := importPath[len(key)+1:]
		if isMajorVersionSegment(suffix) {
			continue
		}
		if len(key) > len(bestKey) {
			bestKey = key
		}
	}

	if bestKey == "" {
		return "", false
	}

	suffix := importPath[len(bestKey):]
	return r.ModulePath + "/" + r.Rules[bestKey] + suffix, true
}

// isMajorVersionSegment reports whether s starts with a Go major version
// suffix like "/v2" or "/v10", optionally followed by more path segments.
func isMajorVersionSegment(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	i := 1
	for i < len(s) && unicode.IsDigit(rune(s[i])) {
		i++
	}
	if i == 1 {
		return false
	}
	// Matches if the rest is empty (e.g. "v2") or starts with "/" (e.g. "v2/sub").
	return i == len(s) || s[i] == '/'
}

// Change records a single import path rewrite.
type Change struct {
	File    string
	OldPath string
	NewPath string
}

// RewriteImports rewrites import paths in Go source data using the mapper function.
// Returns the list of changes, the rewritten source, or an error.
func RewriteImports(data []byte, mapper func(string) (string, bool)) ([]Change, []byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", data, parser.ImportsOnly)
	if err != nil {
		return nil, nil, fmt.Errorf("parse imports: %w", err)
	}

	type replacement struct {
		start   int
		end     int
		newPath string
		oldPath string
	}

	var replacements []replacement
	for _, imp := range f.Imports {
		oldPath := strings.Trim(imp.Path.Value, `"`)
		newPath, ok := mapper(oldPath)
		if !ok {
			continue
		}
		start := fset.Position(imp.Path.ValuePos).Offset
		end := start + len(imp.Path.Value)
		replacements = append(replacements, replacement{
			start:   start,
			end:     end,
			newPath: `"` + newPath + `"`,
			oldPath: oldPath,
		})
	}

	if len(replacements) == 0 {
		return nil, data, nil
	}

	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].start > replacements[j].start
	})

	result := make([]byte, len(data))
	copy(result, data)
	for _, r := range replacements {
		result = append(result[:r.start], append([]byte(r.newPath), result[r.end:]...)...)
	}

	changes := make([]Change, len(replacements))
	for i, r := range replacements {
		changes[i] = Change{OldPath: r.oldPath, NewPath: strings.Trim(r.newPath, `"`)}
	}
	return changes, result, nil
}

// RewriteFile reads a Go file, rewrites its imports using the mapper, and writes
// it back if any changes were made.
func RewriteFile(filePath string, mapper func(string) (string, bool)) ([]Change, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	changes, newData, err := RewriteImports(data, mapper)
	if err != nil {
		return nil, err
	}
	if len(changes) > 0 {
		if err := os.WriteFile(filePath, newData, info.Mode().Perm()); err != nil {
			return nil, fmt.Errorf("write file: %w", err)
		}
	}
	for i := range changes {
		changes[i].File = filePath
	}
	return changes, nil
}
