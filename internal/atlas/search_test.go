package atlas

import "testing"

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
