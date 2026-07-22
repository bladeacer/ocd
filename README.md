# obsi-css-diff

Extract and diff `app.css` across Obsidian versions. Downloads Obsidian's ASAR bundle directly from GitHub releases — no Docker needed.

## Quick start

```bash
# Build
make build

# Launch interactive TUI (browse versions, select, extract)
./obsi-css-diff interact

# Or direct commands:
./obsi-css-diff extract 1.12.7
./obsi-css-diff diff 1.12.7 1.12.6
./obsi-css-diff clean
```

## Commands

| Command | Description |
|---------|-------------|
| `interact` | TUI browser for Obsidian releases with filtering/search |
| `extract <ver>` | Download and extract `app.css` from a specific version |
| `diff <a> <b>` | Unified diff between two versions' CSS |
| `clean` | Wipe cached data |

## How it works

1. Fetches version list from Obsidian's RSS changelog
2. Cross-references with Docker Hub availability and Electron-to-Chromium mappings
3. Extracts `app.css` directly from `obsidian-{version}.asar.gz` on GitHub releases
4. Parses the Electron ASAR archive in Go (no Node.js required)

## License

GPL-3.0
