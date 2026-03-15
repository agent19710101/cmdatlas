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
- search by name, summary, help text, flags, or subcommands
- inspect one stored command without reprobeing the tool
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

Search the local atlas:

```bash
cmdatlas search version control
cmdatlas search --json json
```

Inspect one indexed command:

```bash
cmdatlas show git
cmdatlas show --json git
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

The index is persisted at:

```text
$XDG_CONFIG_HOME/cmdatlas/index.json
```

If `XDG_CONFIG_HOME` is not set, Go falls back to the platform user config directory.

## Examples

Example scan:

```bash
$ cmdatlas scan git go
git     distributed version control system
go      Go is a tool for managing Go source code.

Saved index: /home/you/.config/cmdatlas/index.json
```

Example search:

```bash
$ cmdatlas search module
go      Go is a tool for managing Go source code.
```

Example show:

```bash
$ cmdatlas show git
Name: git
Path: /usr/bin/git
Summary: distributed version control system
Probe: --help
Scanned: 2026-03-15 10:42:11 UTC

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
    "probe": "--help",
    "scanned_at": "2026-03-15T10:42:11Z"
  }
]
```

## How v0 Works

`cmdatlas` intentionally uses simple heuristics and tight safety limits:

- it only scans commands you name directly, or a small curated shortlist if you provide none
- it probes help in this order: `--help`, `help`, `-h`
- each probe is run with a timeout and output cap so a bad command cannot hang the scan
- summaries, flags, and subcommands are best-effort extracts from the captured help text

This keeps the binary small and the behavior predictable, but the parser will not perfectly understand every CLI.

## Current Status

- Latest release: `v0.3.0`
- Stable local indexing/search/show/export flow is working.
- JSON output now makes `search` and `show` easier to consume from scripts and agents.
- Completion install helpers now put generated scripts into standard per-user config locations.

v0 ships these commands:

- `cmdatlas scan [COMMAND ...]`
- `cmdatlas search [--json] QUERY`
- `cmdatlas show [--json] COMMAND`
- `cmdatlas export --json`
- `cmdatlas completion [bash|zsh|fish|powershell]`
- `cmdatlas completion install [bash|zsh|fish|powershell]`

Covered by tests:

- help text normalization and extraction heuristics
- search ranking and lookup behavior
- index save/load round trips
- JSON output for `search` and `show`
- completion script generation and unsupported-shell handling

## Roadmap

- richer subcommand graphing with nested command paths
- re-scan diffing and stale-command detection
- optional aliases, tags, and notes per command
- smarter parser strategies for popular CLIs
- shell-specific activation hints and profile wiring for completion install
- next likely UX step: command aliases/tags/notes so teams and agents can layer local semantics on top of raw help output

## License

MIT. See [LICENSE](LICENSE).
