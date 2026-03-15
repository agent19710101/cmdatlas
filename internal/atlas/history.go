package atlas

import (
	"sort"
	"strings"
	"time"
)

const MaxScanHistoryEntries = 20

type ScanSnapshot struct {
	ScannedAt      time.Time          `json:"scanned_at"`
	Profile        string             `json:"profile,omitempty"`
	Targets        []string           `json:"targets,omitempty"`
	Summary        ScanSummary        `json:"summary"`
	Warnings       []string           `json:"warnings,omitempty"`
	WarningDetails []ScanWarning      `json:"warning_details,omitempty"`
	Commands       []ScanCommandState `json:"commands,omitempty"`
}

type ScanSummary struct {
	Added     []string `json:"added,omitempty"`
	Updated   []string `json:"updated,omitempty"`
	Unchanged []string `json:"unchanged,omitempty"`
	Stale     []string `json:"stale,omitempty"`
}

type ScanWarning struct {
	Command string `json:"command"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type ScanCommandState struct {
	Name      string    `json:"name"`
	Path      string    `json:"path,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	ScannedAt time.Time `json:"scanned_at,omitempty"`
}

func AppendScanHistory(index Index, snapshot ScanSnapshot) Index {
	snapshot.Profile = strings.TrimSpace(snapshot.Profile)
	snapshot.Targets = cloneSorted(snapshot.Targets)
	snapshot.Summary = normalizeScanSummary(snapshot.Summary)
	snapshot.Warnings = append([]string(nil), snapshot.Warnings...)
	snapshot.WarningDetails = append([]ScanWarning(nil), snapshot.WarningDetails...)
	snapshot.Commands = cloneCommandStates(snapshot.Commands)
	if snapshot.ScannedAt.IsZero() {
		snapshot.ScannedAt = time.Now().UTC()
	}

	index.History = append([]ScanSnapshot{snapshot}, index.History...)
	if len(index.History) > MaxScanHistoryEntries {
		index.History = index.History[:MaxScanHistoryEntries]
	}
	return index
}

func cloneCommandStates(values []ScanCommandState) []ScanCommandState {
	out := append([]ScanCommandState(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func normalizeScanSummary(summary ScanSummary) ScanSummary {
	summary.Added = cloneSorted(summary.Added)
	summary.Updated = cloneSorted(summary.Updated)
	summary.Unchanged = cloneSorted(summary.Unchanged)
	summary.Stale = cloneSorted(summary.Stale)
	return summary
}

func cloneSorted(values []string) []string {
	out := append([]string(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}
