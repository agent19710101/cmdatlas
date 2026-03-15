package atlas

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.json")

	index := Index{
		Version: CurrentIndexVersion,
		Commands: []CommandDoc{
			{
				Name:      "git",
				Path:      "/usr/bin/git",
				Summary:   "distributed version control",
				HelpLines: []string{"Usage: git [command]"},
			},
		},
	}

	if err := Save(path, index); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got.Commands) != 1 || got.Commands[0].Name != "git" {
		t.Fatalf("Load() = %#v, want git command", got.Commands)
	}
}

func TestLoadMissingFileReturnsEmptyIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Version != CurrentIndexVersion {
		t.Fatalf("Load() version = %d, want %d", got.Version, CurrentIndexVersion)
	}
	if len(got.Commands) != 0 {
		t.Fatalf("Load() commands = %d, want 0", len(got.Commands))
	}
}
