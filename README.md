![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/bladeacer/ocd?style=for-the-badge&logo=go)
![GitHub License](https://img.shields.io/github/license/bladeacer/ocd?style=for-the-badge)

![Coverage](coverage.svg)

# ocd - Obsidian CSS Diff

Extract and diff `app.css` across Obsidian versions. Downloads Obsidian's ASAR
bundle directly from GitHub releases - no Docker or Node.js needed.

Keybinds are Vim-inspired so terminal enjoyers will feel right at home. I try
to stick to a loose interpretation of the UNIX software philosophy here.

Do one thing and do it well, in this case diff-ing `app.css` between
Obsidian versions.

## Install

### go install

```bash
go install github.com/bladeacer/ocd@latest
ocd
```

### From source

```bash
git clone https://github.com/bladeacer/ocd
cd ocd
make build
./ocd
```

### Pre-built binaries

Download from [releases](https://github.com/bladeacer/ocd/releases) (linux/windows/darwin, amd64/arm64).

```bash
curl -LO https://github.com/bladeacer/ocd/releases/latest/download/ocd_linux_amd64.tar.gz
tar xzf ocd_linux_amd64.tar.gz
sudo mv ocd /usr/local/bin/
```

## Usage

### Interactive TUI

```bash
ocd interact
```

Browse versions - fetches RSS changelog, Docker Hub tags, and Electron-Chromium
mappings asynchronously. Loading messages rotate and show elapsed time.

| Key | Action |
|-----|--------|
| `↑` `↓` | Navigate rows |
| `←` `→` | Scroll columns |
| `enter` | Select version for CSS extraction |
| `/` | Enter search mode - type to filter table live |
| `m` | Toggle mobile versions |
| `e` | Toggle early access / insider versions |
| `f` | Show only versions with Docker images |
| `s` | Toggle sort priority: extracted CSS first → Docker found → N/A → missing |
| `?` | Toggle help overlay |
| `Esc` | Close search / help overlay |
| `q` / `ctrl+c` | Quit |

### Diff versions

```bash
ocd diff -p          # interactive picker (two-step selection)
ocd diff 1.12.6 1.12.7  # direct CLI
ocd diff --tldr 1.12.6 1.12.7  # TLDR analysis (CSS heuristics) + TOML export
ocd diff --tldr --tldr-format json 1.12.6 1.12.7  # TLDR as JSON
```

### Version stats

```bash
ocd stat 1.12.7                      # show CSS stats + export to TOML
ocd stat 1.12.7 --format json        # export as JSON
ocd stat 1.12.7 --output ~/reports   # custom output directory
```

Analyze a single version's `app.css` for selector count, CSS variables,
color usage, and specificity - useful for tracking CSS bloat over time.

The diff viewer opens with scrollable, colorized output. Active hunk lines
are highlighted with a blue background.

| Key | Action |
|-----|--------|
| `↑` `↓` `j` `k` | Scroll one line |
| `pgup` `pgdn` | Scroll one page |
| `{}` | Jump prev/next diff hunk |
| `gg` / `G` | Jump to top / bottom of diff |
| `zz` / `zt` / `zb` | Center / top / bottom current hunk |
| `n` / `N` | Next / previous (search match when searching, else hunk) |
| `/` | Search within diff - highlights matching pattern |
| `v` | Toggle vertical split/unified view |
| `y` | Yank current hunk to clipboard |
| `Y` | Yank entire diff to clipboard |
| `yy` | Yank current hunk line content |
| `e` | Export TLDR analysis to TOML |
| `o` | Open diff viewer (`$OCD_DIFF_PAGER` / `$EDITOR` / `delta` / `less -R`) |
| `?` | Toggle help overlay |
| `Esc` | Close search / help overlay |
| `q` / `ctrl+c` | Quit diff viewer |

Picker mode (when called with `-p`):

| Key | Action |
|-----|--------|
| `↑` `↓` | Navigate versions |
| `enter` | Select version |
| `/` | Search/filter versions |
| `m` | Toggle mobile versions |
| `q` / `ctrl+c` | Cancel |

### Extract specific version

```bash
ocd extract 1.12.7
```

### Clean cache

```bash
ocd clean
```

## Commands

| Command | Description |
|---------|-------------|
| `interact` | TUI browser with async loading, search, filters, CSS status column |
| `extract <ver>` | Download + extract `app.css` from GitHub releases |
| `diff [a] [b]` | Interactive picker or direct diff with colored viewer |
| `stat <ver>` | CSS composition stats (selectors, variables, colors) for a single version |
| `clean` | Wipe `.obsidian_cache/` metadata and extracted CSS |

### Example output: Stat

```bash
cwd:~$ ./ocd stat 1.12.7
   ____  __________     1.12.7
  / __ \/ ____/ __ \    20644 LOC, +1439 selectors, +1049 variables
 / / / / /   / / / /    66 !important, hex:252 hsl:38 rgb:249
/ /_/ / /___/ /_/ /     specificity mean 16.4  median 10.0  mode 10.0
\____/\____/_____/

Exported: /home/data/Desktop/projects/ocd/ocd-stat-1.12.7.toml
```

### Example output: Stat TOML output

```toml
additions_loc = 20644
average_specificity = 16.42807505211954
css_variables_added = ["--undo-button-bg-color-hover", "--link-outline", "..."]
deletions_loc = 0
important_count = 66
selectors_added = [".messageBar", ".messageBar .closeButton", "#editorUndoBar", "..."]
total_selectors_analyzed = 1439
version_a = "1.12.7"
version_b = ""

[color_counts]
  hex = 252
  hsl = 38
  rgb = 249
```

### Example output: Diff --tldr

```bash
cwd:~$ ./ocd diff --tldr 1.11.5 1.12.7
   ____  __________     1.11.5 -> 1.12.7  (minor)
  / __ \/ ____/ __ \    507 insertions(+), 258 deletions(-)
 / / / / /   / / / /    +249 LOC, +24 selectors, -15 selectors
/ /_/ / /___/ /_/ /     +26 variables, -6 variables, ~2 changed
\____/\____/_____/      3 !important, hex:13 rgb:7
                        specificity mean 18.7  median 20.0  mode 20.0

Exported: /home/data/Desktop/projects/ocd/ocd-tldr-1.11.5-1.12.7.toml
```

### Example output: Diff --tldr TOML output

```toml
additions_loc = 507
average_specificity = 18.71794871794872
css_variables_added = ["--background-modifier-form-field-hover", "--metadata-property-corner-shape-focus", "..."]
css_variables_removed = ["--nav-item-weight-active", "--corner-smoothing", "..."]
deletions_loc = 258
important_count = 3
selectors_added = [".file-browser-description", ".modal-view-options-toolbar", "..."]
selectors_removed = [".download-attachments", ".download-attachments .download-attachment-item", "..."]
semver_bump = "minor"
total_selectors_analyzed = 39
version_a = "1.11.5"
version_b = "1.12.7"

[color_counts]
  hex = 13
  rgb = 7

[[css_variables_changed]]
  name = "--view-bottom-fade-mask"
  old_value = "linear-gradient(to top, rgba(0, 0, 0, var(--view-bottom-fade-opacity)) 0%, #000000 1px)"
  new_value = "linear-gradient(to top, #000 0%, #000 1px)"

[[css_variables_changed]]
  name = "--hidden-nav-offset"
  old_value = "calc(var(--view-header-height) + var(--header-top-offset))"
  new_value = "calc(var(--view-header-height) + var(--view-header-top-offset))"
```

## How it works

1. Fetches version list from Obsidian's RSS changelog
2. Cross-references with Docker Hub availability and Electron-to-Chromium mappings
3. Extracts `app.css` directly from `obsidian-{version}.asar.gz` on GitHub releases
4. Parses the Electron ASAR archive in pure Go - no external tools needed

## Development

```bash
make build       # build binary
make test        # run unit tests
make cover       # tests + coverage report
make cover-html  # tests + HTML coverage report in browser
make fmt         # go fmt + go vet
make lint        # golangci-lint
make release-test  # goreleaser snapshot (no upload)
make clean       # remove binary and cache
```

Tests cover RSS electron fill, Docker tag parsing, ASAR extraction,
CSS diff, cache operations and other details.

## LLM Usage Disclaimer

AI Assistance is used when working on the codebase.

## License

MIT License.
