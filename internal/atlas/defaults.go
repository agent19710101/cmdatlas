package atlas

import (
	"os/exec"
	"sort"
)

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

func DefaultCommands() []string {
	var found []string
	for _, name := range defaultCandidates {
		if _, err := exec.LookPath(name); err == nil {
			found = append(found, name)
		}
	}
	sort.Strings(found)
	return found
}
