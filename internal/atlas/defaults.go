package atlas

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

const DefaultProfileName = "default"

var defaultCandidates = []string{
	"git",
	"go",
	"docker",
	"kubectl",
	"gh",
	"npm",
	"node",
	"python3",
	"pip",
	"make",
	"rg",
	"fd",
	"cargo",
	"rustc",
	"terraform",
	"helm",
	"aws",
	"curl",
}

var builtInScanProfiles = map[string][]string{
	DefaultProfileName: defaultCandidates,
	"dev": {
		"git",
		"go",
		"cargo",
		"rustc",
		"node",
		"npm",
		"python3",
		"pip",
		"make",
		"rg",
		"fd",
		"gh",
	},
	"ops": {
		"docker",
		"kubectl",
		"helm",
		"terraform",
		"aws",
		"curl",
		"gh",
		"git",
	},
	"shell": {
		"git",
		"gh",
		"curl",
		"make",
		"rg",
		"fd",
		"python3",
	},
}

func DefaultCommands() []string {
	commands, _ := CommandsForProfile(Index{}, DefaultProfileName)
	return commands
}

func CommandsForProfile(index Index, name string) ([]string, error) {
	key := normalizeProfileName(name)
	if key == "" {
		key = DefaultProfileName
	}
	candidates, ok := profileCommands(index, key)
	if !ok {
		return nil, fmt.Errorf("unknown scan profile %q", name)
	}

	var found []string
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate); err == nil {
			found = append(found, candidate)
		}
	}
	sort.Strings(found)
	return found, nil
}

func ProfileNames(index Index) []string {
	names := make([]string, 0, len(builtInScanProfiles)+len(index.Profiles))
	seen := map[string]struct{}{}
	for name := range builtInScanProfiles {
		names = append(names, name)
		seen[name] = struct{}{}
	}
	for name := range index.Profiles {
		name = normalizeProfileName(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func SetProfile(index Index, name string, commands []string) (Index, error) {
	name = normalizeProfileName(name)
	if name == "" {
		return index, fmt.Errorf("profile name is required")
	}
	if name == DefaultProfileName {
		return index, fmt.Errorf("profile %q is reserved", name)
	}
	commands = dedupeProfileCommands(commands)
	if len(commands) == 0 {
		return index, fmt.Errorf("profile %q requires at least one command", name)
	}
	if index.Profiles == nil {
		index.Profiles = map[string][]string{}
	}
	index.Profiles[name] = commands
	return index, nil
}

func DeleteProfile(index Index, name string) (Index, error) {
	name = normalizeProfileName(name)
	if name == "" {
		return index, fmt.Errorf("profile name is required")
	}
	if _, ok := builtInScanProfiles[name]; ok {
		return index, fmt.Errorf("built-in profile %q cannot be removed", name)
	}
	if _, ok := index.Profiles[name]; !ok {
		return index, fmt.Errorf("profile %q does not exist", name)
	}
	delete(index.Profiles, name)
	if len(index.Profiles) == 0 {
		index.Profiles = nil
	}
	return index, nil
}

func AddToProfile(index Index, name string, commands []string) (Index, []string, error) {
	name = normalizeProfileName(name)
	if name == "" {
		return index, nil, fmt.Errorf("profile name is required")
	}
	if _, ok := builtInScanProfiles[name]; ok {
		return index, nil, fmt.Errorf("built-in profile %q cannot be modified", name)
	}
	if _, ok := index.Profiles[name]; !ok {
		return index, nil, fmt.Errorf("profile %q does not exist", name)
	}

	existing := append([]string(nil), index.Profiles[name]...)
	merged := dedupeProfileCommands(append(existing, commands...))
	if len(merged) == 0 {
		return index, nil, fmt.Errorf("profile %q requires at least one command", name)
	}

	added := diffProfileCommands(merged, existing)
	index.Profiles[name] = merged
	return index, added, nil
}

func RemoveFromProfile(index Index, name string, commands []string) (Index, []string, error) {
	name = normalizeProfileName(name)
	if name == "" {
		return index, nil, fmt.Errorf("profile name is required")
	}
	if _, ok := builtInScanProfiles[name]; ok {
		return index, nil, fmt.Errorf("built-in profile %q cannot be modified", name)
	}
	existing, ok := index.Profiles[name]
	if !ok {
		return index, nil, fmt.Errorf("profile %q does not exist", name)
	}

	toRemove := map[string]struct{}{}
	for _, command := range dedupeProfileCommands(commands) {
		toRemove[command] = struct{}{}
	}

	kept := make([]string, 0, len(existing))
	removed := make([]string, 0, len(existing))
	for _, command := range dedupeProfileCommands(existing) {
		if _, ok := toRemove[command]; ok {
			removed = append(removed, command)
			continue
		}
		kept = append(kept, command)
	}
	if len(kept) == 0 {
		return index, nil, fmt.Errorf("profile %q requires at least one command", name)
	}

	index.Profiles[name] = kept
	return index, removed, nil
}

func diffProfileCommands(next []string, previous []string) []string {
	seen := map[string]struct{}{}
	for _, command := range previous {
		seen[command] = struct{}{}
	}
	added := make([]string, 0, len(next))
	for _, command := range next {
		if _, ok := seen[command]; ok {
			continue
		}
		added = append(added, command)
	}
	return added
}

func profileCommands(index Index, name string) ([]string, bool) {
	if commands, ok := builtInScanProfiles[name]; ok {
		return commands, true
	}
	if index.Profiles == nil {
		return nil, false
	}
	commands, ok := index.Profiles[name]
	return dedupeProfileCommands(commands), ok
}

func normalizeProfileName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func dedupeProfileCommands(commands []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		if _, ok := seen[command]; ok {
			continue
		}
		seen[command] = struct{}{}
		out = append(out, command)
	}
	sort.Strings(out)
	return out
}
