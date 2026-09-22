# Implementation Plan for ocd 0.4.0

## Status

The following work is complete and committed to the working tree (not yet
committed to git):

- CHANGELOG.md with historical changelogs for v0.1.0 to v0.3.0 and the 0.4.0
  scope, written in British English.
- internal/config/config.go: OS-aware config resolution. `Resolve()` merges
  the global config (`$XDG_CONFIG_HOME/ocd/config.toml`, falling back to
  `~/.config/ocd/config.toml`) with the per-project `.ocd.toml` in the
  working directory. Direct CLI flags take priority over the per-project
  config, which takes priority over the global config.
- internal/config/config_test.go: tests for config resolution.
- internal/core/export.go: generalised `ExportFile` helper used by `stat`,
  `tldr`, and `check`.
- internal/core/variables.go: `ExtractCSSVariables`, `CompareVariables`,
  `VariableReport`, and marshal helpers (TOML/JSON/YAML).
- internal/core/variables_test.go: tests for variable extraction and
  comparison.
- internal/tui/diffmodel.go: `h` and `l` keys added as aliases for `{` and `}`
  hunk navigation. Help overlay and footer updated.
- cmd/check.go: new `ocd check <version> <theme.css>` command.
- cmd/diff.go and cmd/stat.go: wired to use `config.Resolve()` for flag
  defaults and switched to the generalised `ExportFile` helper.
- main.go: registers the `check` command.
- README.md: British English rewrite, credits section with the SimpleEnglish
  repo link, configuration section, and `check` command documentation.

## Remaining Work

1. **Run the full test suite and fix any failures.**

   ```
   cd /home/data/Desktop/projects/ocd
   go test ./...
   ```

   The last run showed all packages passing. Re-run after any edits.

2. **Run `go vet` and `gofmt`.**

   The project's `.golangci.yml` uses an old config format that the
   installed golangci-lint cannot load. Use `go vet ./...` and
   `gofmt -w .` instead.

3. **Verify the `check` command end-to-end.**

   A cached `app.css` exists at `.obsidian_cache/css/1.12.7/app.css`. Run:

   ```
   ./ocd check 1.12.7 ./some-theme.css --output /tmp/reports
   ./ocd check 1.12.7 ./some-theme.css --format json --silent --output /tmp/reports
   ```

   Confirm the exported file has a single extension (`.toml`, `.json`, or
   `.yaml`).

4. **Verify config file resolution end-to-end.**

   Create a global config at `$XDG_CONFIG_HOME/ocd/config.toml` and a
   per-project `.ocd.toml` in the working directory, then run `ocd stat` and
   `ocd diff` without flags to confirm the defaults are applied. Confirm that
   a direct flag overrides the config file.

5. **Update the README keybind table (if needed).**

   The diff viewer table already lists `{}` `hl` for hunk navigation. The
   `interact` command table is unchanged.

6. **Commit the changes (only if explicitly requested by the user).**

   Do not commit unless the user asks. When committing, stage only the files
   listed above plus any test fixtures, and write a concise commit message
   matching the repo style (e.g. `feat: config files, check command, hunk
   keys`).

## Notes

- The project already depends on `github.com/BurntSushi/toml` and
  `gopkg.in/yaml.v3`; no new dependencies were added.
- The `ExportFile` helper in `internal/core/export.go` is the single code
  path for file output. Commands should use it rather than calling
  `ExportTLDR` directly.
- The `check` command's `marshalReport` helper in `cmd/check.go` mirrors the
  `marshalTLDR` and `marshalStat` helpers in `cmd/diff.go` and `cmd/stat.go`.
  If a shared helper is preferred, move the format switch into
  `internal/core` and have each command call it.
