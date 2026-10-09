# ocd Configuration Reference

Place config files at `.ocd.toml` (per-project) or
`$XDG_CONFIG_HOME/ocd/config.toml` (global, falling back to
`~/.config/ocd/config.toml`).

Resolution order (highest to lowest priority):
1. Direct command-line flag value
2. Per-project `.ocd.toml` in the working directory
3. Global config at `$XDG_CONFIG_HOME/ocd/config.toml`

Only fields explicitly set in a config file participate in the merge.
Unset fields keep their command default.

## diff

| Flag | Config Key | Default | Description |
|------|-----------|---------|-------------|
| `--tldr` | `tldr` | `false` | Enable TLDR analysis and export |
| `--tldr-format` | `tldr_format` | `"toml"` | Export format: `toml`, `json`, or `yaml` |
| `--tldr-output` | `tldr_dir` | `"~/reports"` | Output directory for TLDR exports |
| `--pick` | `pick` | `false` | Launch interactive version picker |
| `--refresh` | `refresh` | `false` | Force refresh metadata cache |

## stat

| Flag | Config Key | Default | Description |
|------|-----------|---------|-------------|
| `--format` | `stat_format` | `"toml"` | Export format: `toml`, `json`, or `yaml` |
| `--output` | `stat_dir` | `"~/reports"` | Output directory for stat exports |

## check

| Flag | Config Key | Default | Description |
|------|-----------|---------|-------------|
| `--format` | `check_format` | `"toml"` | Export format: `toml`, `json`, or `yaml` |
| `--output` | `check_dir` | `""` | Output directory for check exports. Falls back to cwd when empty. |
| `--silent` | _(CLI only)_ | `false` | Suppress stdout report (file export still occurs) |
| `--compat-mode` | `check_compat_mode` | `""` | Compatibility mode: `strict` or `relaxed`. When set, run `CheckCompatibility` on the theme's CSS variables against the target version, using the cached versions as the origin search space. In strict mode, an incompatible result yields a non-zero exit code. |
| `--compat-sweep` | _(CLI only)_ | `false` | Cache `app.css` for every public desktop version before the compatibility check, so the check covers the whole public history |
| `--cache-days` | `cache_days` | `14` | Days a cached `app.css` stays fresh before it is downloaded again. `0` disables expiry |

## origin

| Flag | Config Key | Default | Description |
|------|-----------|---------|-------------|
| `--format` | `origin_format` | `"toml"` | Export format: `toml`, `json`, or `yaml` |
| `--output` | `origin_dir` | `""` | Output directory for origin exports. Falls back to cwd when empty. |
| `--refresh` | _(CLI only)_ | `false` | Force refresh metadata cache and download every `app.css` again |
| `--cache-days` | `cache_days` | `14` | Days a cached `app.css` stays fresh before it is downloaded again. `0` disables expiry |

## extract

| Flag | Config Key | Default | Description |
|------|-----------|---------|-------------|
| `--from-file` | _(CLI only)_ | `""` | Import a local `.asar` or `.css` file instead of downloading from GitHub |
| `--refresh` | _(CLI only)_ | `false` | Download the version again even when it is already cached |

## Common

| Flag | Config Key | Default | Description |
|------|-----------|---------|-------------|
| `--output` | `output_dir` | `""` | Common output directory fallback. Falls back to cwd when empty. |
| `--cache-days` | `cache_days` | `14` | Days a cached `app.css` and the version metadata cache stay fresh. `0` disables expiry |

## Cache behaviour

`ocd` keeps two caches in `.obsidian_cache/`:

- `app.css` files, one directory per version
- version metadata from the RSS feed, Docker Hub, and the Electron map

`ocd origin` needs the whole public history to answer correctly, so it
downloads every public desktop version that is missing or expired before it
searches. The first run is large: about 98 versions, roughly 880 MB of
downloads. Later runs reuse the cache and do no downloads.

Cached entries are reused for 14 days. After that they are downloaded again,
because Obsidian can republish or correct a release. Use `--cache-days` to
change the age, `--refresh` to download everything again now, and
`ocd clean <version>` to drop a single entry.

Versions with no GitHub release are reported as `unavailable` and skipped.
They are checked again on the next sweep, so a release that appears later is
picked up without any action.

## diff_keys

Configure keybind overrides for the diff viewer (`ocd diff`).
Each field accepts a single key string or a list of key strings.
The default keys are shown below. See the diff viewer help
(`?` inside the diff viewer) for a full list of all keybindings.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `prev_hunk` | `[]string` | `["{", "h"]` | Navigate to previous hunk |
| `next_hunk` | `[]string` | `["}", "l"]` | Navigate to next hunk |
| `scroll_down` | `[]string` | `["j", "down"]` | Scroll down |
| `scroll_up` | `[]string` | `["k", "up"]` | Scroll up |
| `toggle_side_by_side` | `string` | `"v"` | Toggle side-by-side view |
| `quit` | `[]string` | `["q"]` | Quit the diff viewer |
| `help` | `string` | `"?"` | Toggle help overlay |
| `export` | `string` | `"e"` | Export TLDR analysis |
| `next_search` | `string` | `"n"` | Next search match |
| `prev_search` | `string` | `"N"` | Previous search match |
| `scroll_to_hunk` | `string` | `"z"` | Scroll to current hunk |
| `scroll_to_top_of_hunk` | `string` | `"t"` | Scroll to top of current hunk |
| `scroll_to_bottom_of_hunk` | `string` | `"b"` | Scroll to bottom of current hunk |

Example `.ocd.toml` with `diff_keys`:

```toml
# diff command defaults
tldr = true
tldr_format = "json"
tldr_dir = "~/reports"
pick = false
refresh = false

# diff keybind overrides
[diff_keys]
prev_hunk = ["{"]
next_hunk = ["}"]
quit = ["q"]

# stat command defaults
stat_format = "yaml"
stat_dir = "~/reports"

# check command defaults
check_format = "toml"
check_dir = "~/reports"
check_compat_mode = "relaxed"

# origin command defaults
origin_format = "toml"
origin_dir = "~/reports"

# cache defaults
cache_days = 14

# Common output directory
output_dir = "~/reports"
```

## File export behavior

All three commands (`stat`, `tldr`, `check`) default file export to
the current working directory. Use `--output` on the CLI or the
corresponding `*_dir` key in config to override this. See
[`.ocd.toml`](../.ocd.toml) for a sample configuration file.
