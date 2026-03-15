package atlas

import (
	"strings"
	"testing"
)

func TestSearchRanksNameBeforeHelpText(t *testing.T) {
	index := Index{
		Commands: []CommandDoc{
			{Name: "git", Summary: "distributed version control"},
			{Name: "grep", Summary: "search text", HelpLines: []string{"git appears in this help line"}},
		},
	}

	results := Search(index, "git")
	if len(results) != 2 {
		t.Fatalf("Search() returned %d results, want 2", len(results))
	}
	if results[0].Name != "git" {
		t.Fatalf("Search() ranked %q first, want git", results[0].Name)
	}
}

func TestSearchMatchesAliasesTagsAndNotes(t *testing.T) {
	index := Index{
		Commands: []CommandDoc{
			{Name: "kubectl", Aliases: []string{"k"}, Tags: []string{"kubernetes"}, Notes: []string{"cluster admin helper"}},
			{Name: "kind", Summary: "local kubernetes clusters"},
		},
	}

	for _, query := range []string{"k", "kubernetes", "cluster admin"} {
		results := Search(index, query)
		if len(results) == 0 || results[0].Name != "kubectl" {
			t.Fatalf("Search(%q) did not rank annotated command first: %#v", query, results)
		}
	}
}

func TestFindIsCaseInsensitive(t *testing.T) {
	index := Index{
		Commands: []CommandDoc{{Name: "kubectl"}},
	}

	if _, ok := Find(index, "KUBECTL"); !ok {
		t.Fatal("Find() should match command names case-insensitively")
	}
}

func TestMergePreservesAnnotationsAcrossRescan(t *testing.T) {
	existing := Index{
		Commands: []CommandDoc{{
			Name:    "git",
			Aliases: []string{"vcs"},
			Tags:    []string{"source-control"},
			Notes:   []string{"daily driver"},
		}},
	}

	merged := Merge(existing, []CommandDoc{{Name: "git", Summary: "distributed version control"}}, []string{"git"})
	got, ok := Find(merged, "git")
	if !ok {
		t.Fatal("Find() did not return merged command")
	}
	if len(got.Aliases) != 1 || got.Aliases[0] != "vcs" {
		t.Fatalf("aliases not preserved: %#v", got.Aliases)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "source-control" {
		t.Fatalf("tags not preserved: %#v", got.Tags)
	}
	if len(got.Notes) != 1 || got.Notes[0] != "daily driver" {
		t.Fatalf("notes not preserved: %#v", got.Notes)
	}
}

func TestDocsEquivalentIgnoresAnnotationsAndScanTime(t *testing.T) {
	a := CommandDoc{
		Name:        "git",
		Path:        "/usr/bin/git",
		Summary:     "distributed version control",
		HelpLines:   []string{"usage: git [flags]"},
		Flags:       []FlagDoc{{Name: "--help"}},
		Subcommands: []Subcommand{{Name: "clone", Summary: "Clone a repo"}},
		Aliases:     []string{"vcs"},
		Tags:        []string{"team"},
		Notes:       []string{"daily driver"},
		Probe:       "--help",
	}
	b := a
	b.Aliases = []string{"scm"}
	b.Tags = nil
	b.Notes = []string{"other note"}

	if !DocsEquivalent(a, b) {
		t.Fatal("DocsEquivalent() should ignore annotations")
	}

	b.Summary = "another summary"
	if DocsEquivalent(a, b) {
		t.Fatal("DocsEquivalent() should detect scan-content changes")
	}
}

func TestSetAnnotationsNormalizesValues(t *testing.T) {
	index := Index{Commands: []CommandDoc{{Name: "git"}}}

	updated, err := SetAnnotations(index, "git", []string{" vcs ", "VCS", "git"}, []string{" team ", "TEAM"}, []string{" note ", "note", "other"})
	if err != nil {
		t.Fatalf("SetAnnotations() error = %v", err)
	}
	got, _ := Find(updated, "git")
	if len(got.Aliases) != 2 || got.Aliases[0] != "git" || got.Aliases[1] != "vcs" {
		t.Fatalf("aliases = %#v, want normalized values", got.Aliases)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "team" {
		t.Fatalf("tags = %#v, want normalized values", got.Tags)
	}
	if len(got.Notes) != 2 || got.Notes[0] != "note" || got.Notes[1] != "other" {
		t.Fatalf("notes = %#v, want normalized values", got.Notes)
	}
}

func TestAddToProfileMergesWithoutReplacing(t *testing.T) {
	index := Index{Profiles: map[string][]string{"team": {"git", "go"}}}

	updated, added, err := AddToProfile(index, "team", []string{"gh", "go"})
	if err != nil {
		t.Fatalf("AddToProfile() error = %v", err)
	}
	if got := strings.Join(updated.Profiles["team"], ","); got != "gh,git,go" {
		t.Fatalf("profile commands = %q", got)
	}
	if got := strings.Join(added, ","); got != "gh" {
		t.Fatalf("added = %q", got)
	}
}

func TestRemoveFromProfileKeepsRemainingCommands(t *testing.T) {
	index := Index{Profiles: map[string][]string{"team": {"gh", "git", "go"}}}

	updated, removed, err := RemoveFromProfile(index, "team", []string{"gh", "missing"})
	if err != nil {
		t.Fatalf("RemoveFromProfile() error = %v", err)
	}
	if got := strings.Join(updated.Profiles["team"], ","); got != "git,go" {
		t.Fatalf("profile commands = %q", got)
	}
	if got := strings.Join(removed, ","); got != "gh" {
		t.Fatalf("removed = %q", got)
	}
}

func TestRemoveFromProfileRejectsEmptyProfile(t *testing.T) {
	index := Index{Profiles: map[string][]string{"team": {"git"}}}

	_, _, err := RemoveFromProfile(index, "team", []string{"git"})
	if err == nil {
		t.Fatal("expected error when removing last command")
	}
}
