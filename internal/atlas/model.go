package atlas

import "time"

const CurrentIndexVersion = 1

type Index struct {
	Version     int                        `json:"version"`
	Generated   time.Time                  `json:"generated_at"`
	Commands    []CommandDoc               `json:"commands"`
	ScannedSet  []string                   `json:"scanned_set,omitempty"`
	Profiles    map[string][]string        `json:"profiles,omitempty"`
	ProfileMeta map[string]ProfileMetadata `json:"profile_meta,omitempty"`
}

type ProfileMetadata struct {
	Origin       string    `json:"origin,omitempty"`
	ImportedFrom string    `json:"imported_from,omitempty"`
	ImportedAt   time.Time `json:"imported_at,omitempty"`
	ExportedAt   time.Time `json:"exported_at,omitempty"`
}

type CommandDoc struct {
	Name        string       `json:"name"`
	Path        string       `json:"path"`
	Summary     string       `json:"summary"`
	HelpLines   []string     `json:"help_lines"`
	Flags       []FlagDoc    `json:"flags,omitempty"`
	Subcommands []Subcommand `json:"subcommands,omitempty"`
	Aliases     []string     `json:"aliases,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	Notes       []string     `json:"notes,omitempty"`
	Probe       string       `json:"probe"`
	ScannedAt   time.Time    `json:"scanned_at"`
}

type FlagDoc struct {
	Name    string `json:"name"`
	Example string `json:"example,omitempty"`
}

type Subcommand struct {
	Name    string `json:"name"`
	Summary string `json:"summary,omitempty"`
}
