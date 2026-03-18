package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/agent19710101/cmdatlas/internal/atlas"
)

type completionInstallTarget struct {
	Path    string
	Script  string
	Message string
}

func preferredShellProfile(shell string, configDir string) string {
	switch shell {
	case "bash":
		return filepath.Join(configDir, "cmdatlas", "completion.bashrc")
	case "zsh":
		return filepath.Join(configDir, "cmdatlas", "completion.zshrc")
	case "fish":
		return filepath.Join(configDir, "fish", "config.fish")
	case "powershell":
		base := filepath.Join(configDir, "powershell")
		if runtime.GOOS == "windows" {
			base = filepath.Join(filepath.Dir(configDir), "Documents", "PowerShell")
		}
		return filepath.Join(base, "Microsoft.PowerShell_profile.ps1")
	default:
		return ""
	}
}

func installCompletion(shell string) (completionInstallTarget, error) {
	configDir, err := atlas.UserConfigDir()
	if err != nil {
		return completionInstallTarget{}, err
	}

	shell = strings.ToLower(strings.TrimSpace(shell))
	switch shell {
	case "bash":
		path := filepath.Join(configDir, "bash_completion.d", "cmdatlas")
		profile := preferredShellProfile(shell, configDir)
		return completionInstallTarget{
			Path:   path,
			Script: bashCompletionScript(),
			Message: fmt.Sprintf("Installed bash completion to %s\nLoad now: source %s\nPersist for future shells by adding this line to %s:\n  [ -f %s ] && source %s",
				path, path, profile, path, path),
		}, nil
	case "zsh":
		path := filepath.Join(configDir, "zsh", "completions", "_cmdatlas")
		profile := preferredShellProfile(shell, configDir)
		completionDir := filepath.Dir(path)
		return completionInstallTarget{
			Path:   path,
			Script: zshCompletionScript(),
			Message: fmt.Sprintf("Installed zsh completion to %s\nLoad now in the current shell:\n  fpath=(%s $fpath)\n  autoload -Uz compinit && compinit\nPersist for future shells by adding this line to %s:\n  fpath=(%s $fpath)",
				path, completionDir, profile, completionDir),
		}, nil
	case "fish":
		path := filepath.Join(configDir, "fish", "completions", "cmdatlas.fish")
		profile := preferredShellProfile(shell, configDir)
		return completionInstallTarget{
			Path:   path,
			Script: fishCompletionScript(),
			Message: fmt.Sprintf("Installed fish completion to %s\nFish auto-loads completions from this directory. If it is already running, start a new shell or run: exec fish\nMain fish config file: %s",
				path, profile),
		}, nil
	case "powershell":
		base := filepath.Join(configDir, "powershell", "Completions")
		if runtime.GOOS == "windows" {
			base = filepath.Join(filepath.Dir(configDir), "Documents", "PowerShell", "Completions")
		}
		path := filepath.Join(base, "cmdatlas.ps1")
		profile := preferredShellProfile(shell, configDir)
		return completionInstallTarget{
			Path:   path,
			Script: powershellCompletionScript(),
			Message: fmt.Sprintf("Installed PowerShell completion to %s\nLoad now: . %s\nPersist for future shells by adding this line to %s:\n  . %s",
				path, path, profile, path),
		}, nil
	default:
		return completionInstallTarget{}, fmt.Errorf("unsupported shell %q", shell)
	}
}

func writeCompletionInstall(shell string) (string, error) {
	target, err := installCompletion(shell)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target.Path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target.Path, []byte(target.Script), 0o644); err != nil {
		return "", err
	}
	return target.Message, nil
}
