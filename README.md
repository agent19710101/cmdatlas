# cmdatlas

Local command atlas for humans and agents.

`cmdatlas` scans a small set of installed CLI tools, probes their help text safely, and builds a local searchable index. The goal for v0 is simple: make the commands already on your machine easier to discover, inspect, and hand off to another human or agent.

## Problem

Shell environments accumulate dozens of useful binaries, but their help surfaces are inconsistent:

- some tools put the summary on the first line
- some expose flags cleanly, others bury them in prose
- some support `--help`, others prefer `help` or `-h`

That makes local tool discovery noisy for both people and agents. `cmdatlas` provides one local index with a consistent interface:

- scan selected commands or a curated safe shortlist from `PATH`
- search by name, summary, help text, flags, subcommands, aliases, tags, or notes
- inspect one stored command without reprobeing the tool
- add local aliases, tags, and notes on top of scanned command docs
- export the atlas as JSON
- generate or install shell completion scripts for bash, zsh, fish, and PowerShell

## Install

Build from source:

```bash
go build ./cmd/cmdatlas
```

Or install into your Go bin directory:

```bash
go install github.com/agent19710101/cmdatlas/cmd/cmdatlas@latest
```

## Quickstart

Scan a few known commands:

```bash
cmdatlas scan git go rg
```

Scan the default curated shortlist that exists on your `PATH`:

```bash
cmdatlas scan
```

Scan a named profile instead of the default shortlist:

```bash
cmdatlas scan --profile dev
cmdatlas scan --profile ops
cmdatlas scan --profile shell
```

Scan with machine-readable output for scripts or agents:

```bash
cmdatlas scan --json
cmdatlas scan --json git go
```

Search the local atlas:

```bash
cmdatlas search version control
cmdatlas search kubernetes
cmdatlas search --json json
```

Inspect one indexed command:

```bash
cmdatlas show git
cmdatlas show --json git
```

Layer local team/agent context onto a scanned command:

```bash
cmdatlas annotate --alias vcs --tag team-tool --note "daily driver for repo work" git
cmdatlas search team-tool
cmdatlas show git
```

Create a reusable custom scan profile for your machine or team:

```bash
cmdatlas profiles set team git gh go make
cmdatlas profiles list
cmdatlas scan --profile team
```

Export the stored index:

```bash
cmdatlas export --json
```

Generate a shell completion script:

```bash
cmdatlas completion bash
cmdatlas completion zsh
cmdatlas completion fish
cmdatlas completion powershell
```

Or install completion into your user config directory:

```bash
cmdatlas completion install bash
cmdatlas completion install zsh
cmdatlas completion install fish
cmdatlas completion install powershell
```

Each install command now prints shell-specific activation hints so you know what to source immediately and which profile/config file to update for future shells.

The index is persisted at:

```text
$XDG_CONFIG_HOME/cmdatlas/index.json
```

If `XDG_CONFIG_HOME` is not set, Go falls back to the platform user config directory.

Index saves are atomic: `cmdatlas` writes a temp file in the same directory and only replaces `index.json` after the new contents are fully written.

## Examples

Example scan:

```bash
$ cmdatlas scan git go
git     distributed version control system
go      Go is a tool for managing Go source code.

Scan summary:
Added: git, go
Updated: none
Unchanged: none
Stale: none

Saved index: /home/you/.config/cmdatlas/index.json
```

Example machine-readable scan output:

```bash
$ cmdatlas scan --json git go missing-tool
{
  "index_path": "/home/you/.config/cmdatlas/index.json",
  "summary": {
    "added": ["git", "go"],
    "updated": [],
    "unchanged": [],
    "stale": []
  },
  "commands": [
    {
      "name": "git",
      "summary": "distributed version control system"
    },
    {
      "name": "go",
      "summary": "Go is a tool for managing Go source code."
    }
  ],
  "warnings": [
    "missing-tool [not_found]: executable file not found in $PATH"
  ],
  "warning_details": [
    {
      "command": "missing-tool",
      "kind": "not_found",
      "message": "executable file not found in $PATH"
    }
  ]
}
```

Example search:

```bash
$ cmdatlas search module
go      Go is a tool for managing Go source code.
```

Example show:

```bash
$ cmdatlas annotate --alias vcs --tag team-tool --note "daily driver for repo work" git
Updated git
Aliases: vcs
Tags: team-tool
Notes:
  - daily driver for repo work
Saved index: /home/you/.config/cmdatlas/index.json

$ cmdatlas show git
Name: git
Path: /usr/bin/git
Summary: distributed version control system
Probe: --help
Scanned: 2026-03-15 10:42:11 UTC
Aliases: vcs
Tags: team-tool
Notes:
  - daily driver for repo work

Help:
  usage: git [-v | --version] [-h | --help] ...
  These are common Git commands used in various situations:

Flags:
  -v
  --version
  -h
  --help
```

Example JSON output for agent/script use:

```bash
$ cmdatlas search --json version control
[
  {
    "name": "git",
    "path": "/usr/bin/git",
    "summary": "distributed version control system",
    "help_lines": ["usage: git [-v | --version] [-h | --help] ..."],
    "flags": [{"name": "--version"}, {"name": "--help"}],
    "aliases": ["vcs"],
    "tags": ["team-tool"],
    "notes": ["daily driver for repo work"],
    "probe": "--help",
    "scanned_at": "2026-03-15T10:42:11Z"
  }
]
```

## How v0 Works

`cmdatlas` intentionally uses simple heuristics and tight safety limits:

- it only scans commands you name directly, a small curated default shortlist, a built-in scan profile (`default`, `dev`, `ops`, `shell`), or a custom profile you save locally
- it probes help in this order: `--help`, `help`, `-h`
- each probe is run with a timeout and output cap so a bad command cannot hang the scan
- summaries, flags, and subcommands are best-effort extracts from the captured help text
- aliases, tags, and notes are local metadata layered onto the stored index and preserved across rescans

This keeps the binary small and the behavior predictable, but the parser will not perfectly understand every CLI.

## Current Status

- Latest release: `v0.13.0`
- Stable local indexing/search/show/export flow is working.
- `cmdatlas scan` now reports added, updated, unchanged, and stale commands so humans and agents can see what changed between rescans.
- `cmdatlas scan` now preserves saved custom profiles instead of dropping them on rescan.
- `cmdatlas profiles set NAME ...`, `profiles add NAME ...`, `profiles remove NAME ...`, `profiles list`, and `profiles delete NAME` support persistent reusable local scan profiles on top of the built-in `default`, `dev`, `ops`, and `shell` sets.
- `cmdatlas scan --profile NAME` works with both built-in profiles and custom local profiles for repeatable machine- or team-specific scans.
- `cmdatlas scan --json` exposes scanned docs plus diff buckets for scripts and agents.
- Machine-readable `warning_details` now classify skipped scan targets like missing binaries versus probe failures, so scripts and agents can react without string parsing.
- JSON output makes `search` and `show` easier to consume from scripts and agents.
- Completion install helpers put generated scripts into standard per-user config locations and print shell-specific activation/profile wiring hints.
- Index writes are atomic, which reduces corruption risk if a save is interrupted.
- GitHub Actions now validate formatting, vetting, tests, build health, built-binary smoke flows, and release-docs drift on pushes, pull requests, and version tags.
- Local aliases/tags/notes can capture team semantics without reprobeing commands.

v0 ships these commands:

- `cmdatlas scan [--json] [--profile NAME] [COMMAND ...]`
- `cmdatlas search [--json] QUERY`
- `cmdatlas show [--json] COMMAND`
- `cmdatlas annotate [--alias NAME] [--tag NAME] [--note TEXT] COMMAND`
- `cmdatlas profiles list`
- `cmdatlas profiles set NAME COMMAND [COMMAND ...]`
- `cmdatlas profiles add NAME COMMAND [COMMAND ...]`
- `cmdatlas profiles remove NAME COMMAND [COMMAND ...]`
- `cmdatlas profiles delete NAME`
- `cmdatlas export --json`
- `cmdatlas completion [bash|zsh|fish|powershell]`
- `cmdatlas completion install [bash|zsh|fish|powershell]`

Covered by tests:

- help text normalization and extraction heuristics
- search ranking and lookup behavior
- annotation normalization/persistence across rescans
- index save/load round trips
- atomic save failure preservation for the index store
- scan diff/stale reporting across rescans
- JSON output for `scan`, `search`, and `show`, including structured scan warning details
- built-in and custom scan-profile selection plus completion suggestions for profile names
- custom profile create/add/remove/list/delete flows and persistence in the local atlas store
- completion script generation and unsupported-shell handling

## Roadmap

- richer subcommand graphing with nested command paths
- smarter parser strategies for popular CLIs
- scan-history snapshots so agents can automate follow-up on atlas changes
- next likely UX step: profile import/share flows and warning handling for more probe-failure categories

## License

MIT. See [LICENSE](LICENSE).
