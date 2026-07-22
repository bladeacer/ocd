# ocd — Obsidian CSS Diff

Extract and diff `app.css` across Obsidian versions. Downloads Obsidian's ASAR bundle directly from GitHub releases — no Docker or Node.js needed.

## Install

### Pre-built binaries (via goreleaser)

```bash
curl -LO https://github.com/bladeacer/ocd/releases/latest/download/ocd_linux_amd64.tar.gz
tar xzf ocd_linux_amd64.tar.gz
sudo mv ocd /usr/local/bin/
```

Or download from [releases](https://github.com/bladeacer/ocd/releases) (linux/windows/darwin, amd64/arm64).

### From source

```bash
go install github.com/bladeacer/ocd@latest
```

### Build locally

```bash
git clone https://github.com/bladeacer/ocd
cd ocd
make build
```

## Usage

### Interactive TUI (recommended)

```bash
ocd interact
```

Browse versions, filter/search, and select one to extract. Fetch runs asynchronously with a spinner, rotating messages, and elapsed time.

| Key | Action |
|-----|--------|
| `↑↓←→` | Navigate table |
| `/` | Search/filter |
| `m` | Toggle mobile versions |
| `e` | Toggle early access versions |
| `f` | Show only versions with Docker images |
| `s` | Toggle sort priority (extracted CSS first, then Docker, then missing) |
| `enter` | Select version for extraction |
| `q` | Quit |

### Diff versions

```bash
# Interactive picker with search + filters
ocd diff -p

# Direct CLI
ocd diff 1.12.7 1.12.6
```

The diff viewer shows scrollable, colorized output with green/red for added/removed lines and `n`/`N` to jump between hunks.

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
| `diff [a] [b]` | Interactive picker or direct diff with colored viewer, n/N hunk nav |
| `clean` | Wipe `.obsidian_cache/` metadata and extracted CSS |

## How it works

1. Fetches version list from Obsidian's RSS changelog
2. Cross-references with Docker Hub availability and Electron-to-Chromium mappings
3. Extracts `app.css` directly from `obsidian-{version}.asar.gz` on GitHub releases
4. Parses the Electron ASAR archive in pure Go — no external tools needed

## Development

```bash
make build     # build binary
make fmt       # go fmt + go vet
make test      # run unit tests
make clean     # remove binary and cache
```

Tests cover RSS electron fill, Docker tag parsing, ASAR extraction, CSS diff, and cache operations.

## License

GPL-3.0
