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
		{name: "bash", args: []string{"completion", "bash"}, want: []string{"complete -F _cmdatlas_completion cmdatlas", "bash zsh fish powershell"}},
		{name: "zsh", args: []string{"completion", "zsh"}, want: []string{"#compdef cmdatlas", "completion:print shell completion scripts"}},
		{name: "fish", args: []string{"completion", "fish"}, want: []string{"complete -c cmdatlas", "__cmdatlas_index_commands"}},
		{name: "powershell", args: []string{"completion", "powershell"}, want: []string{"Register-ArgumentCompleter", "Get-CmdAtlasIndexedCommands"}},
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
	for _, want := range []string{"python3 - \"$index_path\"", "data.get('commands', [])"} {
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
