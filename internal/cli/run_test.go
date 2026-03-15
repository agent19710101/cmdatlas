package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agent19710101/cmdatlas/internal/atlas"
)

func TestRunCompletionScripts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "bash", args: []string{"completion", "bash"}, want: []string{"complete -F _cmdatlas_completion cmdatlas", "bash zsh fish powershell", "--alias --tag --note"}},
		{name: "zsh", args: []string{"completion", "zsh"}, want: []string{"#compdef cmdatlas", "annotate:add aliases, tags, and notes to an indexed command"}},
		{name: "fish", args: []string{"completion", "fish"}, want: []string{"complete -c cmdatlas", "__cmdatlas_index_commands", "-l alias -d 'add a local alias'"}},
		{name: "powershell", args: []string{"completion", "powershell"}, want: []string{"Register-ArgumentCompleter", "Get-CmdAtlasIndexedCommands", "--alias', '--tag', '--note"}},
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
	if !strings.Contains(stdout.String(), installedPath) {
		t.Fatalf("expected install output to mention %s, got %q", installedPath, stdout.String())
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
	for _, want := range []string{"python3 - \"$index_path\"", "data.get('commands', [])", "search|show|export", "--json", "annotate"} {
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
