# Privatize

A CLI tool that privatizes Go third-party dependencies by embedding their source code into your project and rewriting import paths.

## Why

Third-party libraries sometimes have bugs or missing features that block your development. Vendor mode doesn't let you modify the code. Privatize solves this by making the dependency part of your own project, so you can freely patch it.

## How It Works

1. Reads your `.privatize.yaml` config
2. Creates a temporary Go module, runs `go mod vendor` to fetch source code
3. Rewrites import paths in vendored source using AST parsing
4. Copies the rewritten source into your project
5. Rewrites imports in your project's `.go` files to use the new paths

## Install

```bash
go install github.com/edoger/privatize/cmd/privatize@latest
```

## Usage

### Initialize config

```bash
cd your-project
privatize init
```

This creates `.privatize.yaml`:

```yaml
imports: []

rules: {}

exclude:
  - golang.org/x
```

### Configure

Edit `.privatize.yaml` to specify which packages to privatize:

```yaml
imports:
  - github.com/user/pkg
  - github.com/other/lib

rules:
  github.com/user/pkg: internal/pkg
  github.com/other/lib: vendor/lib

exclude:
  - golang.org/x
  - my.org/internal
```

**Fields:**
- `imports`: Packages to privatize. Used to generate temporary Go module dependencies.
- `rules`: Maps original import path to a relative path within your project. Sub-packages are automatically derived.
- `exclude`: Prefix-matched patterns. Matching packages are never privatized.

**Versioned packages** (e.g. `/v2`) must be explicitly declared:

```yaml
rules:
  github.com/user/pkg: internal/pkg
  github.com/user/pkg/v2: internal/pkgv2
```

### Preview changes

```bash
privatize run --dry-run
```

Shows what would be changed without modifying any files.

### Run

```bash
privatize run
```

Executes the privatization pipeline.

## Example

Given project module `github.com/foo/bar` and config:

```yaml
imports:
  - github.com/user/pkg

rules:
  github.com/user/pkg: internal/pkg

exclude:
  - golang.org/x
```

Before:

```go
import "github.com/user/pkg"
import "github.com/user/pkg/sub"
```

After:

```go
import "github.com/foo/bar/internal/pkg"
import "github.com/foo/bar/internal/pkg/sub"
```

The source code of `github.com/user/pkg` is copied to `internal/pkg/` with all imports rewritten.

## Path Mapping Rules

| Original Path | Rule | Result |
|---|---|---|
| `github.com/user/pkg` | `-> internal/pkg` | `github.com/foo/bar/internal/pkg` |
| `github.com/user/pkg/sub` | parent rule | `github.com/foo/bar/internal/pkg/sub` |
| `github.com/user/pkg/v2` | no explicit rule | **not rewritten** |
| `golang.org/x/text` | excluded | **not rewritten** |

## Technical Details

- Uses `go/parser` with `parser.ImportsOnly` for AST-level import detection
- Byte-level replacement preserves file structure outside of import declarations
- Handles all import forms: named, blank (`_`), dot (`.`), and aliased imports
- Temporary workspace in system temp directory, cleaned up after execution
- Directory-level copy includes all non-Go files (data files, templates, etc.)

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [bubbletea](https://github.com/charmbracelet/bubbletea) - Terminal UI
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML config parsing
- [x/mod](https://golang.org/x/mod) - go.mod parsing

## License

MIT
