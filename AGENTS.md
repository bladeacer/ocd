# Project Knowledge

## ocd — Obsidian CSS Diff

Extract and diff `app.css` across Obsidian versions. Downloads Obsidian's ASAR
bundle directly from GitHub releases - no Docker or Node.js needed.

This project uses British English in its documentation, following the
discipline of [ASD-STE100 Simplified Technical English](https://www.asd-ste100.org/)
via the [SimpleEnglish](https://github.com/AminBlg/SimpleEnglish) skill.

## Repository Structure

```
ocd/
├── cmd/                      # Cobra CLI commands
│   ├── check.go              # ocd check <version> <theme.css>
│   ├── cmd_test.go           # Tests for all commands and marshal helpers
│   ├── check_test.go         # Tests for the check command
│   ├── diff.go               # ocd diff [version-a] [version-b]
│   ├── interact.go           # ocd interact (TUI browser)
│   ├── extract.go            # ocd extract <version|label>
│   ├── stat.go               # ocd stat <version>
│   └── clean.go              # ocd clean [label]
├── docs/
│   ├── CONFIG.md           # Full configuration reference
│   └── changelogs/           # Versioned changelog documentation
│       ├── index.md          # Changelog index with links to all versions
│       ├── 0.1.0.md          # v0.1.0 changelog
│       ├── 0.2.0.md          # v0.2.0 changelog
│       ├── 0.3.0.md          # v0.3.0 changelog
│       └── 0.4.0.md          # v0.4.0 (Unreleased) changelog
├── internal/
│   ├── cache/                # Cache management for extracted CSS
│   ├── config/               # OS-aware configuration file resolution
│   │   ├── config.go         # Config struct and Resolve/ResolveIn functions
│   │   └── config_test.go    # Tests for config resolution
│   ├── core/                 # Shared business logic
│   │   ├── diff.go           # DiffCSS and related helpers
│   │   ├── diff_test.go      # Tests for CSS diffing
│   │   ├── export.go         # Generalised ExportFile helper
│   │   ├── export_test.go    # Tests for ExportFile and helpers
│   │   ├── extract.go        # ExtractCSS, ImportFile, copyFile
│   │   ├── extract_test.go   # Tests for extraction
│   │   ├── tldr.go           # TLDR analysis, marshaling, and rendering
│   │   ├── tldr_test.go      # Tests for TLDR analysis
│   │   ├── variables.go      # CSS variable extraction and comparison
│   │   └── variables_test.go # Tests for CSS variable logic
│   ├── models/               # Shared data types
│   │   └── types.go          # VersionType, FetchResult, DiffResult, etc.
│   ├── sources/              # RSS, Docker, and Electron data sources
│   │   ├── rss.go            # RSS changelog fetching
│   │   ├── rss_test.go       # Tests for RSS source
│   │   ├── docker.go         # Docker tag fetching
│   │   ├── docker_test.go    # Tests for Docker source
│   │   ├── electron.go       # Electron-to-Chromium mapping
│   │   ├── electron_test.go  # Tests for Electron mapping
│   │   └── fetcher.go        # Combined data fetcher
│   │   └── fetcher_test.go   # Tests for fetcher
│   └── tui/                  # Terminal UI components
│       ├── diffmodel.go      # Diff viewer model with keybind handling
│       ├── diffmodel_test.go # Tests for diff viewer
│       ├── pickermodel.go    # Version picker model
│       └── view.go           # Shared UI view helpers
├── scripts/                  # Utility scripts (check_links.py, gen_structure.py)
├── .ocd.toml             # Sample per-project config file (feature parity with CLI)
├── .obsidian_cache/          # Cached extracted CSS files
├── Makefile                  # Build, test, cover, fmt, lint targets
├── main.go                   # Application entry point
├── go.mod                    # Go module definition
├── go.sum                    # Go module checksums
├── README.md                 # Project documentation
├── CHANGELOG.md              # Redirects to docs/changelogs/index.md
├── .goreleaser.yml           # Goreleaser configuration
├── .gitignore                # Git ignore rules
└── AGENTS.md                 # This file
```

## Key Design Decisions

### Configuration Resolution

Config files are resolved in priority order (highest to lowest):
1. Direct CLI flag value
2. Per-project `.ocd.toml` in the working directory
3. Global config at `$XDG_CONFIG_HOME/ocd/config.toml`, falling back to
   `~/.config/ocd/config.toml`

Only explicitly set fields participate in the merge; unset fields keep their
command default.

### File Export

All commands share one export path through `core.ExportFile`. Commands
pass a marshal function, output directory, filename, and format. The helper
handles `~/` and `$VAR` expansion, directory creation, and extension
derivation.

`stat`, `tldr`, and `check` all default file export to the current working
directory when no output directory is configured via `--output` flag or
the corresponding `*_dir` config key. Use `output_dir` in config as a
global fallback.

### Changelogs

Historical changelogs live at `docs/changelogs/`. Each version has its own
markdown file with a link to the corresponding GitHub releases page.
The `v0.4.0` release is pending (unreleased). The index at
`docs/changelogs/index.md` provides an overview table with all versions.

## Building and Testing

```bash
make build       # Build the binary
make test        # Run all unit tests
make cover       # Tests + coverage report
make fmt         # go fmt + go vet
make lint        # golangci-lint
make check-links # Check all markdown files for broken links
```

Tests should achieve at least 80% coverage across all packages.

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/BurntSushi/toml` — TOML encoding/decoding
- `gopkg.in/yaml.v3` — YAML encoding/decoding
- `github.com/hexops/gotextdiff` — Diff computation
- `github.com/atotto/clipboard` — Clipboard access
- `github.com/charmbracelet/lipgloss` — Terminal styling
