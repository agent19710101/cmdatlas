package probe

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/agent19710101/cmdatlas/internal/atlas"
)

func TestScanCommandAddsNestedSubcommandsForAllowlistedCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture assumes unix-like environment")
	}

	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	writeDispatchCommand(t, filepath.Join(binDir, "gh"), map[string]string{
		"":     "GitHub CLI\n\nAvailable Commands:\n  pr        Manage pull requests\n  repo      Manage repositories\n",
		"pr":   "Manage pull requests\n\nAvailable Commands:\n  checks    Show CI checks\n  view      View a pull request\n",
		"repo": "Manage repositories\n\nAvailable Commands:\n  clone     Clone a repository\n",
	})

	doc, err := ScanCommand("gh")
	if err != nil {
		t.Fatalf("ScanCommand returned error: %v", err)
	}

	got := joinSubcommandNames(doc.Subcommands)
	for _, want := range []string{"pr", "repo", "pr checks", "pr view", "repo clone"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected nested subcommands to contain %q, got %q", want, got)
		}
	}
}

func TestScanCommandDoesNotProbeNestedSubcommandsForOtherCLIs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture assumes unix-like environment")
	}

	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	writeDispatchCommand(t, filepath.Join(binDir, "demo"), map[string]string{
		"":      "Demo CLI\n\nAvailable Commands:\n  child     Child command\n",
		"child": "Child command\n\nAvailable Commands:\n  grandchild     Nested command\n",
	})

	doc, err := ScanCommand("demo")
	if err != nil {
		t.Fatalf("ScanCommand returned error: %v", err)
	}

	got := joinSubcommandNames(doc.Subcommands)
	if strings.Contains(got, "child grandchild") {
		t.Fatalf("unexpected nested subcommand probing for non-allowlisted CLI: %q", got)
	}
}

func joinSubcommandNames(subcommands []atlas.Subcommand) string {
	parts := make([]string, 0, len(subcommands))
	for _, sub := range subcommands {
		parts = append(parts, sub.Name)
	}
	return strings.Join(parts, ",")
}

func writeDispatchCommand(t *testing.T, path string, outputs map[string]string) {
	t.Helper()
	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("mode=\"\"\n")
	script.WriteString("for arg in \"$@\"; do\n")
	script.WriteString("  case \"$arg\" in\n")
	script.WriteString("    --help|-h|help) break ;;\n")
	script.WriteString("    *)\n")
	script.WriteString("      if [ -z \"$mode\" ]; then\n")
	script.WriteString("        mode=\"$arg\"\n")
	script.WriteString("      else\n")
	script.WriteString("        mode=\"$mode $arg\"\n")
	script.WriteString("      fi\n")
	script.WriteString("      ;;\n")
	script.WriteString("  esac\n")
	script.WriteString("done\n")
	script.WriteString("case \"$mode\" in\n")
	for key, output := range outputs {
		escapedKey := strings.ReplaceAll(key, "'", "'\\''")
		escapedOutput := strings.ReplaceAll(output, "\\", "\\\\")
		escapedOutput = strings.ReplaceAll(escapedOutput, "'", "'\\''")
		escapedOutput = strings.ReplaceAll(escapedOutput, "\n", "\\n")
		script.WriteString(fmt.Sprintf("  '%s') printf '%%b' '%s' ;;\n", escapedKey, escapedOutput))
	}
	script.WriteString("  *) exit 1 ;;\n")
	script.WriteString("esac\n")
	if err := os.WriteFile(path, []byte(script.String()), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
