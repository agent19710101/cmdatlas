package cli

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/agent19710101/cmdatlas/internal/atlas"
)

func runCompletion(args []string, stdout io.Writer) error {
	if len(args) == 2 && strings.EqualFold(strings.TrimSpace(args[0]), "install") {
		message, err := writeCompletionInstall(args[1])
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, message+"\n")
		return err
	}
	if len(args) != 1 {
		return errors.New("usage: cmdatlas completion [bash|zsh|fish|powershell]\n       cmdatlas completion install [bash|zsh|fish|powershell]")
	}

	shell := strings.ToLower(strings.TrimSpace(args[0]))
	var script string
	switch shell {
	case "bash":
		script = bashCompletionScript()
	case "zsh":
		script = zshCompletionScript()
	case "fish":
		script = fishCompletionScript()
	case "powershell":
		script = powershellCompletionScript()
	default:
		return fmt.Errorf("unsupported shell %q", args[0])
	}

	_, err := io.WriteString(stdout, script)
	return err
}

func completionCommandNames() []string {
	commands := []string{"annotate", "completion", "export", "help", "history", "profiles", "scan", "search", "show"}
	sort.Strings(commands)
	return commands
}

func completionProfileNames() []string {
	index, _ := loadIndex()
	return atlas.ProfileNames(index)
}

func bashCompletionScript() string {
	commands := strings.Join(completionCommandNames(), " ")
	profiles := strings.Join(completionProfileNames(), " ")
	return fmt.Sprintf(`# bash completion for cmdatlas
__cmdatlas_complete_show() {
    local cur index_path
    cur="${COMP_WORDS[COMP_CWORD]}"
    index_path="${XDG_CONFIG_HOME:-$HOME/.config}/cmdatlas/index.json"
    if [[ -f "$index_path" ]]; then
        COMPREPLY=( $(compgen -W "$(python3 - "$index_path" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, 'r', encoding='utf-8') as fh:
    data = json.load(fh)
for doc in data.get('commands', []):
    name = str(doc.get('name', '')).strip()
    if name:
        print(name)
PY
)" -- "$cur") )
    fi
}

_cmdatlas_completion() {
    local cur prev commands
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="%s"

    case "$prev" in
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish powershell" -- "$cur") )
            return 0
            ;;
        --profile)
            COMPREPLY=( $(compgen -W "%s" -- "$cur") )
            return 0
            ;;
        profiles)
            COMPREPLY=( $(compgen -W "list set add remove delete export import" -- "$cur") )
            return 0
            ;;
        show)
            if [[ "$cur" == --* ]]; then
                COMPREPLY=( $(compgen -W "--json" -- "$cur") )
            else
                __cmdatlas_complete_show
            fi
            return 0
            ;;
        annotate)
            if [[ "$cur" == --* ]]; then
                COMPREPLY=( $(compgen -W "--alias --tag --note" -- "$cur") )
            else
                __cmdatlas_complete_show
            fi
            return 0
            ;;
        scan|search|export)
            COMPREPLY=( $(compgen -W "--json --profile" -- "$cur") )
            return 0
            ;;
    esac

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
        return 0
    fi

    case "${COMP_WORDS[1]}" in
        scan)
            COMPREPLY=( $(compgen -W "--json --profile" -- "$cur") )
            ;;
        search|show|export)
            COMPREPLY=( $(compgen -W "--json" -- "$cur") )
            ;;
        profiles)
            COMPREPLY=( $(compgen -W "list set add remove delete export import" -- "$cur") )
            ;;
        annotate)
            if [[ "$cur" == --* ]]; then
                COMPREPLY=( $(compgen -W "--alias --tag --note" -- "$cur") )
            else
                __cmdatlas_complete_show
            fi
            ;;
        *)
            COMPREPLY=()
            ;;
    esac
}

complete -F _cmdatlas_completion cmdatlas
`, commands, profiles)
}

func zshCompletionScript() string {
	profiles := completionProfileNames()
	return fmt.Sprintf(`#compdef cmdatlas

_cmdatlas_index_commands() {
  local index_path
  index_path=${XDG_CONFIG_HOME:-$HOME/.config}/cmdatlas/index.json
  if [[ -f "$index_path" ]]; then
    python3 - "$index_path" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, 'r', encoding='utf-8') as fh:
    data = json.load(fh)
for doc in data.get('commands', []):
    name = str(doc.get('name', '')).strip()
    if name:
        print(name)
PY
  fi
}

_cmdatlas() {
  local -a commands
  commands=(
    'scan:scan commands into the local atlas'
    'search:search the local atlas'
    'show:show one indexed command'
    'annotate:add aliases, tags, and notes to an indexed command'
    'profiles:list, save, edit, import, export, or delete custom scan profiles'
    'history:show persisted scan history'
    'export:export the atlas as JSON'
    'completion:print shell completion scripts'
    'help:show help'
  )

  case $words[2] in
    completion)
      _values 'shell' bash zsh fish powershell
      return
      ;;
    scan)
      _arguments '--json[emit JSON]' '--profile[scan a named command profile]:profile:(%s)' '*:command:'
      return
      ;;
    profiles)
      _arguments '1:action:(list set add remove delete export import)'
      return
      ;;
    show)
      _arguments '--json[emit JSON]' '1:indexed command:_cmdatlas_index_commands'
      return
      ;;
    annotate)
      _arguments \
        '--alias[add a local alias]:alias:' \
        '--tag[add a local tag]:tag:' \
        '--note[add a local note]:note:' \
        '1:indexed command:_cmdatlas_index_commands'
      return
      ;;
    search|export)
      _arguments '--json[emit JSON]'
      return
      ;;
  esac

  if (( CURRENT == 2 )); then
    _describe 'command' commands
    return
  fi

  case $words[2] in
    scan)
      _arguments '--json[emit JSON]' '--profile[scan a named command profile]:profile:(%s)' '*:command:'
      ;;
    search|export)
      _arguments '--json[emit JSON]'
      ;;
    profiles)
      _arguments '1:action:(list set add remove delete export import)'
      ;;
    show)
      _arguments '--json[emit JSON]' '1:indexed command:_cmdatlas_index_commands'
      ;;
    annotate)
      _arguments \
        '--alias[add a local alias]:alias:' \
        '--tag[add a local tag]:tag:' \
        '--note[add a local note]:note:' \
        '1:indexed command:_cmdatlas_index_commands'
      ;;
  esac
}

_cmdatlas "$@"
`, strings.Join(profiles, " "), strings.Join(profiles, " "))
}

func fishCompletionScript() string {
	profiles := strings.Join(completionProfileNames(), " ")
	return fmt.Sprintf(`function __cmdatlas_index_commands
    set -l index_path
    if set -q XDG_CONFIG_HOME
        set index_path "$XDG_CONFIG_HOME/cmdatlas/index.json"
    else
        set index_path "$HOME/.config/cmdatlas/index.json"
    end

    if test -f "$index_path"
        python3 - "$index_path" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, 'r', encoding='utf-8') as fh:
    data = json.load(fh)
for doc in data.get('commands', []):
    name = str(doc.get('name', '')).strip()
    if name:
        print(name)
PY
    end
end

complete -c cmdatlas -f -n '__fish_use_subcommand' -a 'scan search show annotate profiles history export completion help'
complete -c cmdatlas -f -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish powershell'
complete -c cmdatlas -f -n '__fish_seen_subcommand_from show annotate' -a '(__cmdatlas_index_commands)'
complete -c cmdatlas -f -n '__fish_seen_subcommand_from scan search export show history' -l json -d 'emit JSON'
complete -c cmdatlas -f -n '__fish_seen_subcommand_from scan' -l profile -d 'scan a named command profile' -a '%s'
complete -c cmdatlas -f -n '__fish_seen_subcommand_from annotate' -l alias -d 'add a local alias'
complete -c cmdatlas -f -n '__fish_seen_subcommand_from annotate' -l tag -d 'add a local tag'
complete -c cmdatlas -f -n '__fish_seen_subcommand_from annotate' -l note -d 'add a local note'
complete -c cmdatlas -f -n '__fish_seen_subcommand_from history' -l limit -d 'maximum number of history entries to show'
complete -c cmdatlas -f -n '__fish_seen_subcommand_from history' -l profile -d 'filter to a named scan profile' -a '%s'
complete -c cmdatlas -f -n '__fish_seen_subcommand_from profiles' -a 'list set add remove delete export import'
`, profiles, profiles)
}

func powershellCompletionScript() string {
	profiles := strings.Join(completionProfileNames(), "', '")
	return fmt.Sprintf(`Register-ArgumentCompleter -Native -CommandName cmdatlas -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $words = $commandAst.CommandElements | ForEach-Object { $_.Extent.Text }
    $commands = @('annotate', 'scan', 'search', 'show', 'profiles', 'history', 'export', 'completion', 'help')
    $scanProfiles = @('%s')

    function Get-CmdAtlasIndexedCommands {
        if ($env:XDG_CONFIG_HOME) {
            $indexPath = Join-Path $env:XDG_CONFIG_HOME 'cmdatlas/index.json'
        } else {
            $indexPath = Join-Path $HOME '.config/cmdatlas/index.json'
        }

        if (Test-Path $indexPath) {
            try {
                $index = Get-Content -Raw -Path $indexPath | ConvertFrom-Json
                foreach ($doc in $index.commands) {
                    if ($doc.name) { $doc.name }
                }
            } catch {
            }
        }
    }

    if ($words.Count -le 2) {
        $commands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    switch ($words[1]) {
        'completion' {
            @('bash', 'zsh', 'fish', 'powershell') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
        }
        'history' {
            if ($words.Count -ge 3 -and $words[$words.Count - 2] -eq '--profile') {
                $scanProfiles | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
                }
            } else {
                @('--json', '--limit', '--profile') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
                }
            }
        }
        'show' {
            if ($wordToComplete -like '--*') {
                @('--json') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
                }
            } else {
                Get-CmdAtlasIndexedCommands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
                }
            }
        }
        'annotate' {
            if ($wordToComplete -like '--*') {
                @('--alias', '--tag', '--note') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
                }
            } else {
                Get-CmdAtlasIndexedCommands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
                }
            }
        }
        'scan' {
            if ($words.Count -ge 3 -and $words[$words.Count - 2] -eq '--profile') {
                $scanProfiles | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
                }
            } else {
                @('--json', '--profile') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
                }
            }
        }
        'search' {
            @('--json') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
        }
        'export' {
            @('--json') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
        }
        'profiles' {
            @('list', 'set', 'add', 'remove', 'delete', 'export', 'import') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
        }
    }
}
`, profiles)
}
