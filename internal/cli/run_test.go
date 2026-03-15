package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/agent19710101/cmdatlas/internal/atlas"
)

func TestRunScanReportsDiffSummary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path and shell fixture assumes unix-like environment")
	}

	configHome := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("PATH", binDir)

	writeFakeCommand(t, filepath.Join(binDir, "git"), "Git fake CLI\nUsage: git [flags]\n")
	writeFakeCommand(t, filepath.Join(binDir, "go"), "Go fake CLI\nUsage: go [flags]\n")

	var stdout bytes.Buffer
	if err := Run([]string{"scan"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("first Run(scan) returned error: %v", err)
	}
	first := stdout.String()
	for _, want := range []string{"Scan summary:", "Added: git, go", "Updated: none", "Unchanged: none", "Stale: none"} {
		if !strings.Contains(first, want) {
			t.Fatalf("expected first scan output to contain %q, got %q", want, first)
		}
	}

	writeFakeCommand(t, filepath.Join(binDir, "go"), "Go fake CLI updated\nUsage: go [flags]\n")
	if err := os.Remove(filepath.Join(binDir, "git")); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	stdout.Reset()
	if err := Run([]string{"scan"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("second Run(scan) returned error: %v", err)
	}
	second := stdout.String()
	for _, want := range []string{"Scan summary:", "Added: none", "Updated: go", "Unchanged: none", "Stale: git"} {
		if !strings.Contains(second, want) {
			t.Fatalf("expected second scan output to contain %q, got %q", want, second)
		}
	}
}

func TestRunScanProfile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path and shell fixture assumes unix-like environment")
	}

	configHome := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("PATH", binDir)

	writeFakeCommand(t, filepath.Join(binDir, "git"), "Git fake CLI\nUsage: git [flags]\n")
	writeFakeCommand(t, filepath.Join(binDir, "go"), "Go fake CLI\nUsage: go [flags]\n")
	writeFakeCommand(t, filepath.Join(binDir, "docker"), "Docker fake CLI\nUsage: docker [flags]\n")
	writeFakeCommand(t, filepath.Join(binDir, "kubectl"), "kubectl fake CLI\nUsage: kubectl [flags]\n")

	var stdout bytes.Buffer
	if err := Run([]string{"scan", "--profile", "ops"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(scan --profile ops) returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"docker\tDocker fake CLI", "git\tGit fake CLI", "kubectl\tkubectl fake CLI", "Added: docker, git, kubectl"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected profile scan output to contain %q, got %q", want, got)
		}
	}
	for _, unwanted := range []string{"go\tGo fake CLI"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("expected profile scan output to omit %q, got %q", unwanted, got)
		}
	}
}

func TestRunScanJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path and shell fixture assumes unix-like environment")
	}

	configHome := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("PATH", binDir)

	writeFakeCommand(t, filepath.Join(binDir, "git"), "Git fake CLI\nUsage: git [flags]\n")
	writeFakeCommand(t, filepath.Join(binDir, "go"), "Go fake CLI\nUsage: go [flags]\n")

	var stdout bytes.Buffer
	if err := Run([]string{"scan", "--json"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(scan --json) returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"\"index_path\":", "\"summary\": {", "\"added\": [", "\"git\"", "\"go\"", "\"commands\": [", "\"help_lines\":"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected JSON output to contain %q, got %q", want, got)
		}
	}
	if strings.Contains(got, "Scan summary:") {
		t.Fatalf("expected JSON output to omit human summary, got %q", got)
	}
}

func TestRunScanJSONIncludesStructuredWarnings(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path and shell fixture assumes unix-like environment")
	}

	configHome := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("PATH", binDir)

	writeFakeCommand(t, filepath.Join(binDir, "git"), "Git fake CLI\nUsage: git [flags]\n")

	var stdout bytes.Buffer
	if err := Run([]string{"scan", "--json", "git", "missing-tool"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(scan --json git missing-tool) returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{
		"\"warnings\": [",
		"missing-tool [not_found]",
		"\"warning_details\": [",
		"\"command\": \"missing-tool\"",
		"\"kind\": \"not_found\"",
		"\"message\":",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected JSON warning output to contain %q, got %q", want, got)
		}
	}
}

func TestRunCompletionScripts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "bash", args: []string{"completion", "bash"}, want: []string{"complete -F _cmdatlas_completion cmdatlas", "bash zsh fish powershell", "--alias --tag --note", "scan|search|export", "profiles", "list set add remove delete export import", "--json --profile", "default", "dev", "ops", "shell"}},
		{name: "zsh", args: []string{"completion", "zsh"}, want: []string{"#compdef cmdatlas", "annotate:add aliases, tags, and notes to an indexed command", "profiles:list, save, edit, import, export, or delete custom scan profiles", "scan:scan commands into the local atlas", "--json[emit JSON]", "--profile[scan a named command profile]", "list set add remove delete export import", "default", "dev", "ops", "shell"}},
		{name: "fish", args: []string{"completion", "fish"}, want: []string{"complete -c cmdatlas", "__cmdatlas_index_commands", "-l alias -d 'add a local alias'", "__fish_seen_subcommand_from scan search export show", "-l json -d 'emit JSON'", "-l profile -d 'scan a named command profile'", "default", "dev", "ops", "shell", "__fish_seen_subcommand_from profiles' -a 'list set add remove delete export import'"}},
		{name: "powershell", args: []string{"completion", "powershell"}, want: []string{"Register-ArgumentCompleter", "Get-CmdAtlasIndexedCommands", "'scan'", "'profiles'", "'list', 'set', 'add', 'remove', 'delete'", "'--json', '--profile'", "$scanProfiles = @('default'", "'dev'", "'ops'", "'shell'"}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			if err := Run(tc.args, &stdout, &bytes.Buffer{}); err != nil {
				t.Fatalf("Run returned error: %v", err)
			}

			got := stdout.String()
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("expected output to contain %q, got %q", want, got)
				}
			}
		})
	}
}

func TestRunScanRejectsProfileWithExplicitTargets(t *testing.T) {
	err := Run([]string{"scan", "--profile", "ops", "git"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error when combining --profile with explicit targets")
	}
	if !strings.Contains(err.Error(), "either explicit COMMAND arguments or --profile") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunProfilesAddAndRemove(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	var stdout bytes.Buffer
	if err := Run([]string{"profiles", "set", "team", "git", "go"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(profiles set) returned error: %v", err)
	}

	stdout.Reset()
	if err := Run([]string{"profiles", "add", "team", "gh", "go"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(profiles add) returned error: %v", err)
	}
	for _, want := range []string{"Updated profile team: gh, git, go", "Added: gh"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected profiles add output to contain %q, got %q", want, stdout.String())
		}
	}

	stdout.Reset()
	if err := Run([]string{"profiles", "remove", "team", "git"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(profiles remove) returned error: %v", err)
	}
	for _, want := range []string{"Updated profile team: gh, go", "Removed: git"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected profiles remove output to contain %q, got %q", want, stdout.String())
		}
	}
}

func TestRunProfilesListShowsBuiltInAndCustomCommands(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := Run([]string{"profiles", "set", "team", "git", "go"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(profiles set) returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run([]string{"profiles", "list"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(profiles list) returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{
		"default\tbuilt-in\taws, cargo, curl, docker, fd, gh, git, go, helm, kubectl, make, node, npm, pip, python3, rg, rustc, terraform",
		"ops\tbuilt-in\taws, curl, docker, gh, git, helm, kubectl, terraform",
		"team\tcustom\tgit, go",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected profiles list output to contain %q, got %q", want, got)
		}
	}
}

func TestRunProfilesExportImport(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := Run([]string{"profiles", "set", "team", "git", "go"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("profiles set returned error: %v", err)
	}
	if err := Run([]string{"profiles", "set", "ops-extra", "gh", "kubectl"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("profiles set returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run([]string{"profiles", "export", "team", "--json"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("profiles export returned error: %v", err)
	}
	for _, want := range []string{"\"profiles\": {", "\"team\": [", "\"git\"", "\"go\""} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected profiles export output to contain %q, got %q", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "ops-extra") {
		t.Fatalf("expected single-profile export to omit other profiles, got %q", stdout.String())
	}

	importPath := filepath.Join(t.TempDir(), "profiles.json")
	payload := `{"profiles":{"shared":["gh","git"],"team":["git","go","make"]}}`
	if err := os.WriteFile(importPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	stdout.Reset()
	if err := Run([]string{"profiles", "import", "--file", importPath}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("profiles import returned error: %v", err)
	}
	for _, want := range []string{"Imported profiles: shared, team", "Mode: merge"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected profiles import output to contain %q, got %q", want, stdout.String())
		}
	}

	indexPath := filepath.Join(configHome, "cmdatlas", "index.json")
	index, err := atlas.Load(indexPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := strings.Join(index.Profiles["team"], ","); got != "git,go,make" {
		t.Fatalf("team profile after import = %q", got)
	}
	if got := strings.Join(index.Profiles["shared"], ","); got != "gh,git" {
		t.Fatalf("shared profile after import = %q", got)
	}

	replacePayload := `{"profiles":{"fresh":["docker"]}}`
	stdout.Reset()
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	readFile, writeFile, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe returned error: %v", err)
	}
	if _, err := writeFile.WriteString(replacePayload); err != nil {
		t.Fatalf("WriteString returned error: %v", err)
	}
	writeFile.Close()
	os.Stdin = readFile
	if err := Run([]string{"profiles", "import", "--replace"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("profiles import --replace returned error: %v", err)
	}
	for _, want := range []string{"Imported profiles: fresh", "Mode: replace"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected replace import output to contain %q, got %q", want, stdout.String())
		}
	}

	index, err = atlas.Load(indexPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(index.Profiles) != 1 || strings.Join(index.Profiles["fresh"], ",") != "docker" {
		t.Fatalf("profiles after replace import = %#v", index.Profiles)
	}
}

func TestRunCompletionRejectsUnsupportedShell(t *testing.T) {
	t.Parallel()

	err := Run([]string{"completion", "elvish"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for unsupported shell")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCompletionInstallWritesScript(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	var stdout bytes.Buffer
	if err := Run([]string{"completion", "install", "bash"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	installedPath := filepath.Join(configHome, "bash_completion.d", "cmdatlas")
	data, err := os.ReadFile(installedPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	got := string(data)
	for _, want := range []string{"complete -F _cmdatlas_completion cmdatlas", "show|export", "--json", "--alias --tag --note"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected installed completion script to contain %q, got %q", want, got)
		}
	}

	message := stdout.String()
	for _, want := range []string{
		installedPath,
		"Load now: source " + installedPath,
		filepath.Join(configHome, "cmdatlas", "completion.bashrc"),
		"[ -f " + installedPath + " ] && source " + installedPath,
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected install output to contain %q, got %q", want, message)
		}
	}
}

func TestPreferredShellProfile(t *testing.T) {
	configHome := "/tmp/cmdatlas-config"

	if got := preferredShellProfile("bash", configHome); got != filepath.Join(configHome, "cmdatlas", "completion.bashrc") {
		t.Fatalf("bash profile = %q", got)
	}
	if got := preferredShellProfile("zsh", configHome); got != filepath.Join(configHome, "cmdatlas", "completion.zshrc") {
		t.Fatalf("zsh profile = %q", got)
	}
	if got := preferredShellProfile("fish", configHome); got != filepath.Join(configHome, "fish", "config.fish") {
		t.Fatalf("fish profile = %q", got)
	}
	wantPowerShell := filepath.Join(configHome, "powershell", "Microsoft.PowerShell_profile.ps1")
	if runtime.GOOS == "windows" {
		wantPowerShell = filepath.Join(filepath.Dir(configHome), "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	}
	if got := preferredShellProfile("powershell", configHome); got != wantPowerShell {
		t.Fatalf("powershell profile = %q, want %q", got, wantPowerShell)
	}
	if got := preferredShellProfile("unknown", configHome); got != "" {
		t.Fatalf("unknown profile = %q, want empty", got)
	}
}

func TestRunShowUsesIndexedCommandNamesForCompletionScript(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	indexPath := filepath.Join(configHome, "cmdatlas", "index.json")
	index := atlas.Index{
		Version: atlas.CurrentIndexVersion,
		Commands: []atlas.CommandDoc{
			{Name: "git"},
			{Name: "go"},
		},
	}
	if err := atlas.Save(indexPath, index); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run([]string{"completion", "bash"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"python3 - \"$index_path\"", "data.get('commands', [])", "scan|search|export", "--json", "annotate", "--profile"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected bash completion script to contain %q, got %q", want, got)
		}
	}

	files, err := os.ReadDir(filepath.Dir(indexPath))
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}
	if len(files) != 1 || files[0].Name() != "index.json" {
		t.Fatalf("expected index.json to exist, got %v", files)
	}
}

func TestRunSearchJSON(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	indexPath := filepath.Join(configHome, "cmdatlas", "index.json")
	index := atlas.Index{
		Version: atlas.CurrentIndexVersion,
		Commands: []atlas.CommandDoc{
			{Name: "git", Summary: "distributed version control system"},
			{Name: "go", Summary: "Go toolchain"},
		},
	}
	if err := atlas.Save(indexPath, index); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run([]string{"search", "--json", "version", "control"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"\n  {", "\"name\": \"git\"", "\"summary\": \"distributed version control system\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected JSON output to contain %q, got %q", want, got)
		}
	}
}

func TestRunShowJSON(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	indexPath := filepath.Join(configHome, "cmdatlas", "index.json")
	index := atlas.Index{
		Version: atlas.CurrentIndexVersion,
		Commands: []atlas.CommandDoc{{
			Name:      "git",
			Path:      "/usr/bin/git",
			Summary:   "distributed version control system",
			Aliases:   []string{"vcs"},
			Tags:      []string{"team"},
			Notes:     []string{"daily driver"},
			Probe:     "--help",
			HelpLines: []string{"usage: git [--help]"},
		}},
	}
	if err := atlas.Save(indexPath, index); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run([]string{"show", "--json", "git"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"\"name\": \"git\"", "\"path\": \"/usr/bin/git\"", "\"probe\": \"--help\"", "\"aliases\": [", "\"notes\": ["} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected JSON output to contain %q, got %q", want, got)
		}
	}
}

func writeFakeCommand(t *testing.T, path string, helpOutput string) {
	t.Helper()
	escaped := strings.ReplaceAll(helpOutput, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "'", "'\\''")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%b' '%s'\n", escaped)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func TestRunAnnotateUpdatesIndexedCommand(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	indexPath := filepath.Join(configHome, "cmdatlas", "index.json")
	index := atlas.Index{
		Version:  atlas.CurrentIndexVersion,
		Commands: []atlas.CommandDoc{{Name: "git", Summary: "distributed version control system"}},
	}
	if err := atlas.Save(indexPath, index); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run([]string{"annotate", "--alias", "vcs,scm", "--tag", "team", "--tag", "cli", "--note", "daily driver", "git"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got, err := atlas.Load(indexPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	doc, ok := atlas.Find(got, "git")
	if !ok {
		t.Fatal("annotated command not found")
	}
	if strings.Join(doc.Aliases, ",") != "scm,vcs" {
		t.Fatalf("aliases = %#v, want normalized aliases", doc.Aliases)
	}
	if strings.Join(doc.Tags, ",") != "cli,team" {
		t.Fatalf("tags = %#v, want normalized tags", doc.Tags)
	}
	if len(doc.Notes) != 1 || doc.Notes[0] != "daily driver" {
		t.Fatalf("notes = %#v, want note saved", doc.Notes)
	}
	for _, want := range []string{"Updated git", "Aliases: scm, vcs", "Tags: cli, team", "daily driver"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected annotate output to contain %q, got %q", want, stdout.String())
		}
	}
}

func TestRunShowPrintsAnnotations(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	indexPath := filepath.Join(configHome, "cmdatlas", "index.json")
	index := atlas.Index{
		Version: atlas.CurrentIndexVersion,
		Commands: []atlas.CommandDoc{{
			Name:      "git",
			Path:      "/usr/bin/git",
			Summary:   "distributed version control system",
			Aliases:   []string{"vcs"},
			Tags:      []string{"team"},
			Notes:     []string{"daily driver"},
			Probe:     "--help",
			HelpLines: []string{"usage: git [--help]"},
		}},
	}
	if err := atlas.Save(indexPath, index); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run([]string{"show", "git"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"Aliases: vcs", "Tags: team", "Notes:", "- daily driver"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected show output to contain %q, got %q", want, got)
		}
	}
}

func TestRunProfilesSetListDelete(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	var stdout bytes.Buffer
	if err := Run([]string{"profiles", "set", "team", "git", "go", "git"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("profiles set returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Saved profile team: git, go") {
		t.Fatalf("unexpected profiles set output: %q", stdout.String())
	}

	stdout.Reset()
	if err := Run([]string{"profiles", "list"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("profiles list returned error: %v", err)
	}
	for _, want := range []string{"default\tbuilt-in\t", "dev\tbuilt-in\t", "ops\tbuilt-in\t", "shell\tbuilt-in\t", "team\tcustom\tgit, go"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected profiles list output to contain %q, got %q", want, stdout.String())
		}
	}

	stdout.Reset()
	if err := Run([]string{"profiles", "delete", "team"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("profiles delete returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Deleted profile team") {
		t.Fatalf("unexpected profiles delete output: %q", stdout.String())
	}
}

func TestRunScanUsesCustomProfile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path and shell fixture assumes unix-like environment")
	}

	configHome := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("PATH", binDir)

	writeFakeCommand(t, filepath.Join(binDir, "git"), "Git fake CLI\nUsage: git [flags]\n")
	writeFakeCommand(t, filepath.Join(binDir, "gh"), "GitHub CLI\nUsage: gh [flags]\n")

	if err := Run([]string{"profiles", "set", "team", "git", "gh"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("profiles set returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run([]string{"scan", "--profile", "team"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("scan with custom profile returned error: %v", err)
	}
	for _, want := range []string{"gh	GitHub CLI", "git	Git fake CLI", "Added: gh, git"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected scan output to contain %q, got %q", want, stdout.String())
		}
	}

	indexPath := filepath.Join(configHome, "cmdatlas", "index.json")
	index, err := atlas.Load(indexPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := strings.Join(index.Profiles["team"], ","); got != "gh,git" {
		t.Fatalf("team profile after scan = %q, want %q", got, "gh,git")
	}
}
