package atlas

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestSaveAtomicReplaceFailureKeepsPreviousIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.json")

	original := Index{Version: CurrentIndexVersion, Commands: []CommandDoc{{Name: "git"}}}
	if err := Save(path, original); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}

	oldRename := renameFile
	renameFile = func(oldPath, newPath string) error {
		return errors.New("boom")
	}
	defer func() { renameFile = oldRename }()

	err := Save(path, Index{Version: CurrentIndexVersion, Commands: []CommandDoc{{Name: "go"}}})
	if err == nil {
		t.Fatal("Save() error = nil, want replace failure")
	}
	if !strings.Contains(err.Error(), "replace") {
		t.Fatalf("Save() error = %v, want replace context", err)
	}

	got, loadErr := Load(path)
	if loadErr != nil {
		t.Fatalf("Load() after failed Save error = %v", loadErr)
	}
	if len(got.Commands) != 1 || got.Commands[0].Name != "git" {
		t.Fatalf("Load() after failed Save = %#v, want original git command", got.Commands)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "index.json" {
		t.Fatalf("expected only index.json to remain, got %v", entries)
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
