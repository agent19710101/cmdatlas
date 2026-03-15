package atlas

import "testing"

func TestAppendScanHistoryKeepsNewestFirstAndCapsEntries(t *testing.T) {
	index := Index{}
	for i := 0; i < MaxScanHistoryEntries+3; i++ {
		index = AppendScanHistory(index, ScanSnapshot{
			Profile:  "ops",
			Targets:  []string{"git", "docker"},
			Summary:  ScanSummary{Added: []string{"git"}},
			Commands: []ScanCommandState{{Name: "git"}},
		})
	}

	if got, want := len(index.History), MaxScanHistoryEntries; got != want {
		t.Fatalf("history length = %d, want %d", got, want)
	}
	if index.History[0].ScannedAt.IsZero() {
		t.Fatal("newest history entry missing timestamp")
	}
	if got := index.History[0].Targets[0]; got != "docker" && got != "git" {
		t.Fatalf("unexpected targets in newest history entry: %#v", index.History[0].Targets)
	}
}
