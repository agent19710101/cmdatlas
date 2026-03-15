package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type completionInstallTarget struct {
	Path    string
	Script  string
	Message string
}

func installCompletion(shell string) (completionInstallTarget, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return completionInstallTarget{}, err
	}

	shell = strings.ToLower(strings.TrimSpace(shell))
	switch shell {
	case "bash":
		path := filepath.Join(configDir, "bash_completion.d", "cmdatlas")
		return completionInstallTarget{
			Path:    path,
			Script:  bashCompletionScript(),
			Message: fmt.Sprintf("Installed bash completion to %s\nLoad it with: source %s", path, path),
		}, nil
	case "zsh":
		path := filepath.Join(configDir, "zsh", "completions", "_cmdatlas")
		return completionInstallTarget{
			Path:    path,
			Script:  zshCompletionScript(),
			Message: fmt.Sprintf("Installed zsh completion to %s\nAdd %s to your fpath, then run: compinit", path, filepath.Dir(path)),
		}, nil
	case "fish":
		path := filepath.Join(configDir, "fish", "completions", "cmdatlas.fish")
		return completionInstallTarget{
			Path:    path,
			Script:  fishCompletionScript(),
			Message: fmt.Sprintf("Installed fish completion to %s", path),
		}, nil
	case "powershell":
		base := filepath.Join(configDir, "powershell", "Completions")
		if runtime.GOOS == "windows" {
			base = filepath.Join(filepath.Dir(configDir), "Documents", "PowerShell", "Completions")
		}
		path := filepath.Join(base, "cmdatlas.ps1")
		return completionInstallTarget{
			Path:    path,
			Script:  powershellCompletionScript(),
			Message: fmt.Sprintf("Installed PowerShell completion to %s\nLoad it with: . %s", path, path),
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
