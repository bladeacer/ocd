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

## Common

| Flag | Config Key | Default | Description |
|------|-----------|---------|-------------|
| `--output` | `output_dir` | `""` | Common output directory fallback. Falls back to cwd when empty. |

## Example `.ocd.toml`

```toml
# diff command defaults
tldr = true
tldr_format = "json"
tldr_dir = "~/reports"
pick = false
refresh = false

# stat command defaults
stat_format = "yaml"
stat_dir = "~/reports"

# check command defaults
check_format = "toml"
check_dir = "~/reports"

# Common output directory
output_dir = "~/reports"
```

## File export behavior

All three commands (`stat`, `tldr`, `check`) default file export to
the current working directory. Use `--output` on the CLI or the
corresponding `*_dir` key in config to override this. See
[`.ocd.toml`](../.ocd.toml) for a sample configuration file.
