# Project Knowledge

## ocd — Obsidian CSS Diff

Extract and diff `app.css` across Obsidian versions. Downloads Obsidian's ASAR
bundle directly from GitHub releases - no Docker or Node.js needed.

This project uses British English in its documentation, following the
discipline of [ASD-STE100 Simplified Technical English](https://www.asd-ste100.org/).
The STE skill is vendored locally at [`skills/simple-english/SKILL.md`](skills/simple-english/SKILL.md).

All agent runs in this project should load the STE skill from `skills/simple-english/SKILL.md`
before producing any documentation, prose, or replies. This is done by invoking the
`simple-english` skill. The skill enforces plain, layman-readable English with
short sentences, active voice, simple tenses, one word one meaning, condition
before command, and every technical term defined at first use. Default mode is
**Plain**; when STE, ASD-STE100, or compliance is named, **Strict** mode adds
full vocabulary compliance from `skills/simple-english/references/strict-vocabulary.md`.

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
│   ├── origin.go             # ocd origin <selector|variable> (full version sweep)
│   ├── origin_test.go        # Tests for the origin command
│   ├── cache_test.go         # Tests for cache flags and helpers
│   └── clean.go              # ocd clean [label]
├── docs/
│   ├── CONFIG.md           # Full configuration reference
│   └── changelogs/           # Versioned changelog documentation
│       ├── index.md          # Changelog index with links to all versions
│       ├── 0.1.0.md          # v0.1.0 changelog
│       ├── 0.2.0.md          # v0.2.0 changelog
│       ├── 0.3.0.md          # v0.3.0 changelog
│       ├── 0.4.0.md          # v0.4.0 changelog
│       ├── 0.5.0.md          # v0.5.0 changelog
│       ├── 0.6.0.md          # v0.6.0 changelog
│       ├── 0.7.0.md          # v0.7.0 changelog
│       ├── 0.8.0.md          # v0.8.0 changelog
│       └── 0.9.0.md          # v0.9.0 changelog
├── internal/
│   ├── cache/                # Cache management for version metadata
│   │   ├── cache.go          # Store with TTL-based expiry
│   │   └── cache_test.go     # Tests for the metadata cache
│   ├── config/               # OS-aware configuration file resolution
│   │   ├── config.go         # Config struct and Resolve/ResolveIn functions
│   │   └── config_test.go    # Tests for config resolution
│   ├── core/                 # Shared business logic
│   │   ├── asar.go           # ASAR archive parsing
│   │   ├── compat.go         # Theme compatibility checking
│   │   ├── cssparse.go       # CSS scanner, ParseCSS and ReadTheme
│   │   ├── cssparse_test.go  # Tests for the CSS scanner and theme loader
│   │   ├── diff.go           # DiffCSS and related helpers
│   │   ├── diff_test.go      # Tests for CSS diffing
│   │   ├── export.go         # Generalised ExportFile helper
│   │   ├── export_test.go    # Tests for ExportFile and helpers
│   │   ├── extract.go        # ExtractCSS, ExtractCSSForce, freshness helpers
│   │   ├── extract_test.go   # Tests for extraction
│   │   ├── origin.go         # FindOrigin and version list helpers
│   │   ├── origin_test.go    # Tests for origin search
│   │   ├── sweep.go          # EnsureAllCSS cache sweep with concurrency
│   │   ├── sweep_test.go     # Tests for the cache sweep
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
├── Makefile                  # Build, test, cover, fmt, lint, watch targets
├── air.toml                  # air hot-reload config used by `make watch`
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

### Version Sweep

`origin` and `check --compat-sweep` need the whole public history to answer
correctly, so they call `core.EnsureAllCSS` before searching.

Prefetching every version is only correct where the answer depends on the
whole history. Elsewhere it would be a large cost for no gain:

| Command | Prefetch | Why |
|---------|----------|-----|
| `origin` | always | The answer depends on every version, not the cached few |
| `check --compat-mode` | opt-in, `--compat-sweep` | Same search space as origin, but `check` is often used in scripts where a 900 MB first run is unwelcome |
| `diff`, `tldr` | never | Two versions only |
| `stat` | never | One version only |
| `extract` | never | One version, and it is the command that fetches |
| `interact` | never | Needs the version list, not the CSS |

The sweep runs in two phases. First it probes every version that needs a
download with a HEAD request, and drops the ones with no GitHub release. Then
it downloads the rest in ascending version order, several at a time. Probing
first matters because about 10 of the 108 public desktop versions have no
release, and a failed download would otherwise waste a full transfer each.

Freshness uses the modification time of the cached `app.css`. There is no
separate metadata file, so the cache has one source of truth. `cache_days`
sets the age and defaults to 14 days. `--refresh` forces a full re-download.
A version with no GitHub release returns `core.ErrNoRelease` and is counted
as unavailable, not as a failure. Those versions are probed again next run,
so a release that appears later is picked up without any action.

### CSS Parsing

`core.ParseCSS` is a byte scanner, not a line or regex reader. Themes are
often minified, with many declarations on one line, so line-based matching
misses almost everything.

A custom property is only recorded where a declaration can start, that is
after a brace or a semicolon. That single rule is what makes the scanner
robust against quoted text: a `--name:` inside a string can never be at a
declaration start, so even a string that runs for thousands of bytes cannot
hide the declarations after it.

Three escapes matter and each one is a regression test:

- `/* */` comments are skipped whole.
- A backslash outside a string escapes the next byte. Themes use `\"` inside
  attribute selectors such as `[data-task=\"]`. Reading that quote as a
  string start swallowed 41 KB of a real theme and hid every declaration
  after it.
- A string ends at an unescaped newline, as CSS requires. Themes use
  backslash-newline continuations for multi-line ASCII art.

Parsed CSS gives O(1) lookups, so `FindOrigins` parses each version once
instead of once per target, and versions are read in parallel
(`core.ParallelFor`). On the flexcyon theme, 742 KB parses in about 6 ms.

### Theme Sources

`core.ReadTheme` accepts a stylesheet or a folder. For a folder every `*.css`
below it is read, with `theme.css` first, and noise directories such as
`node_modules` are skipped.

SCSS sources are deliberately not read. A Sass file defines its palette as
`$variables` that are injected at build time, and Obsidian never sees them, so
comparing them gave results that looked authoritative but were not.

### Selector Matching

Obsidian's `app.css` uses comma-separated selector lists and long descendant
chains, so exact matching alone reports almost no overlap. A comma list is
split into its parts at parse time, and `core.SelectorCovers` decides reach:
a theme rule for `.workspace-leaf` reaches
`.workspace-leaf.mod-active .cm-content`. A descendant combinator in the
theme reaches a child or sibling combinator in the target; a child
combinator does not.

`check` reports both the exact match count and the covered count. Treat the
covered number as indicative, not as a verdict: Obsidian's selectors are
mostly CodeMirror internals that a theme rarely restyles one by one.

### Diff Viewer Keybinds

The `ocd diff` TUI viewer has configurable keybindings exposed via
`diff_keys` in `.ocd.toml`. Defaults are `{`/`h` for prev hunk,
`}`/`l` for next hunk, `j`/`k` for scroll, etc. See `docs/CONFIG.md`
for the full `diff_keys` reference.

### Linting

`make lint` runs `golangci-lint`. The `.golangci.yml` uses `version: 2`
and disables the strict linters (`errcheck`, `staticcheck`, `unused`)
that produce many findings on legacy code patterns.

### Changelogs

Historical changelogs live at `docs/changelogs/`. Each version has its own
markdown file with a link to the corresponding GitHub releases page.
The index at `docs/changelogs/index.md` provides an overview table with all versions.

## Building and Testing

```bash
make build       # Build the binary
make test        # Run all unit tests
make cover       # Tests + coverage report
make fmt         # go fmt + go vet
make lint        # golangci-lint
make check-links # Check all markdown files for broken links
make watch       # Rebuild and rerun the app on save (needs air)
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
