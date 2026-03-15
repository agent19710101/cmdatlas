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

var scanProfiles = map[string][]string{
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
	commands, _ := CommandsForProfile(DefaultProfileName)
	return commands
}

func CommandsForProfile(name string) ([]string, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		key = DefaultProfileName
	}
	candidates, ok := scanProfiles[key]
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

func ProfileNames() []string {
	names := make([]string, 0, len(scanProfiles))
	for name := range scanProfiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
