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

func TestFindIsCaseInsensitive(t *testing.T) {
	index := Index{
		Commands: []CommandDoc{{Name: "kubectl"}},
	}

	if _, ok := Find(index, "KUBECTL"); !ok {
		t.Fatal("Find() should match command names case-insensitively")
	}
}
