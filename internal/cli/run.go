package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/agent19710101/cmdatlas/internal/atlas"
	"github.com/agent19710101/cmdatlas/internal/probe"
)

type scanJSONOutput struct {
	IndexPath      string             `json:"index_path"`
	Summary        scanSummary        `json:"summary"`
	Commands       []atlas.CommandDoc `json:"commands"`
	Warnings       []string           `json:"warnings,omitempty"`
	WarningDetails []scanWarning      `json:"warning_details,omitempty"`
}

type scanWarning struct {
	Command string `json:"command"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type scanSummary struct {
	Added     []string `json:"added"`
	Updated   []string `json:"updated"`
	Unchanged []string `json:"unchanged"`
	Stale     []string `json:"stale"`
}

type profilesJSONExport struct {
	Profiles    map[string][]string              `json:"profiles"`
	ProfileMeta map[string]atlas.ProfileMetadata `json:"profile_meta,omitempty"`
}

type profileImportPlan struct {
	Name       string
	Action     string
	Previous   []string
	Incoming   []string
	ImportedBy string
}

func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "scan":
		return runScan(args[1:], stdout)
	case "search":
		return runSearch(args[1:], stdout)
	case "show":
		return runShow(args[1:], stdout)
	case "annotate":
		return runAnnotate(args[1:], stdout)
	case "profiles":
		return runProfiles(args[1:], stdout)
	case "export":
		return runExport(args[1:], stdout)
	case "completion":
		return runCompletion(args[1:], stdout)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runScan(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "emit JSON")
	profile := fs.String("profile", "", "scan a named command profile")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}

	explicitTargets := len(fs.Args()) > 0
	if explicitTargets && strings.TrimSpace(*profile) != "" {
		return errors.New("scan accepts either explicit COMMAND arguments or --profile, not both")
	}

	indexPath, err := atlas.DefaultIndexPath()
	if err != nil {
		return err
	}
	index, err := atlas.Load(indexPath)
	if err != nil {
		return err
	}

	targets := fs.Args()
	if len(targets) == 0 {
		selectedProfile := firstNonEmpty(strings.TrimSpace(*profile), atlas.DefaultProfileName)
		targets, err = atlas.CommandsForProfile(index, selectedProfile)
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			return fmt.Errorf("no commands from profile %q found on PATH", selectedProfile)
		}
	}
	targets = dedupe(targets)

	previous := index
	previousByName := map[string]atlas.CommandDoc{}
	for _, doc := range previous.Commands {
		previousByName[strings.ToLower(doc.Name)] = doc
	}

	var docs []atlas.CommandDoc
	var failures []string
	var warningDetails []scanWarning
	for _, target := range targets {
		doc, err := probe.ScanCommand(target)
		if err != nil {
			warning := newScanWarning(target, err)
			failures = append(failures, warning.String())
			warningDetails = append(warningDetails, warning)
			continue
		}
		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		return fmt.Errorf("scan failed: %s", strings.Join(failures, ", "))
	}

	index = atlas.Merge(index, docs, targets)
	if err := atlas.Save(indexPath, index); err != nil {
		return err
	}

	var added []string
	var updated []string
	var unchanged []string
	for _, doc := range docs {
		before, ok := previousByName[strings.ToLower(doc.Name)]
		switch {
		case !ok:
			added = append(added, doc.Name)
		case atlas.DocsEquivalent(before, doc):
			unchanged = append(unchanged, doc.Name)
		default:
			updated = append(updated, doc.Name)
		}
	}

	var stale []string
	if !explicitTargets {
		currentTargets := map[string]struct{}{}
		for _, target := range targets {
			currentTargets[strings.ToLower(target)] = struct{}{}
		}
		for _, name := range previous.ScannedSet {
			if _, ok := currentTargets[strings.ToLower(name)]; ok {
				continue
			}
			stale = append(stale, name)
		}
	}

	summary := scanSummary{Added: added, Updated: updated, Unchanged: unchanged, Stale: stale}
	if *jsonOutput {
		return writeJSON(stdout, scanJSONOutput{
			IndexPath:      indexPath,
			Summary:        summary,
			Commands:       docs,
			Warnings:       failures,
			WarningDetails: warningDetails,
		})
	}

	for _, doc := range docs {
		fmt.Fprintf(stdout, "%s\t%s\n", doc.Name, firstNonEmpty(doc.Summary, "indexed"))
	}

	fmt.Fprintf(stdout, "\nScan summary:\n")
	writeScanList(stdout, "Added", added)
	writeScanList(stdout, "Updated", updated)
	writeScanList(stdout, "Unchanged", unchanged)
	writeScanList(stdout, "Stale", stale)

	if len(failures) > 0 {
		fmt.Fprintf(stdout, "\nWarnings:\n")
		for _, failure := range failures {
			fmt.Fprintf(stdout, "- %s\n", failure)
		}
	}
	fmt.Fprintf(stdout, "\nSaved index: %s\n", indexPath)
	return nil
}

func runSearch(args []string, stdout io.Writer) error { /* unchanged body */
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "emit JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) == 0 {
		return errors.New("usage: cmdatlas search [--json] QUERY")
	}
	query := strings.Join(fs.Args(), " ")

	index, err := loadIndex()
	if err != nil {
		return err
	}
	results := atlas.Search(index, query)
	if len(results) == 0 {
		return fmt.Errorf("no results for %q", query)
	}
	if *jsonOutput {
		return writeJSON(stdout, results)
	}

	for _, doc := range results {
		fmt.Fprintf(stdout, "%s\t%s\n", doc.Name, firstNonEmpty(doc.Summary, "(no summary)"))
	}
	return nil
}

func runShow(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "emit JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return errors.New("usage: cmdatlas show [--json] COMMAND")
	}
	index, err := loadIndex()
	if err != nil {
		return err
	}
	doc, ok := atlas.Find(index, fs.Args()[0])
	if !ok {
		return fmt.Errorf("command %q is not indexed", fs.Args()[0])
	}
	if *jsonOutput {
		return writeJSON(stdout, doc)
	}

	fmt.Fprintf(stdout, "Name: %s\n", doc.Name)
	fmt.Fprintf(stdout, "Path: %s\n", doc.Path)
	fmt.Fprintf(stdout, "Summary: %s\n", firstNonEmpty(doc.Summary, "(no summary)"))
	fmt.Fprintf(stdout, "Probe: %s\n", doc.Probe)
	fmt.Fprintf(stdout, "Scanned: %s\n", doc.ScannedAt.Format("2006-01-02 15:04:05 MST"))

	if len(doc.Aliases) > 0 {
		fmt.Fprintf(stdout, "Aliases: %s\n", strings.Join(doc.Aliases, ", "))
	}
	if len(doc.Tags) > 0 {
		fmt.Fprintf(stdout, "Tags: %s\n", strings.Join(doc.Tags, ", "))
	}
	if len(doc.Notes) > 0 {
		fmt.Fprintf(stdout, "Notes:\n")
		for _, note := range doc.Notes {
			fmt.Fprintf(stdout, "  - %s\n", note)
		}
	}

	if len(doc.HelpLines) > 0 {
		fmt.Fprintf(stdout, "\nHelp:\n")
		for _, line := range doc.HelpLines {
			fmt.Fprintf(stdout, "  %s\n", line)
		}
	}
	if len(doc.Flags) > 0 {
		fmt.Fprintf(stdout, "\nFlags:\n")
		for _, flag := range doc.Flags {
			fmt.Fprintf(stdout, "  %s\n", flag.Name)
		}
	}
	if len(doc.Subcommands) > 0 {
		fmt.Fprintf(stdout, "\nSubcommands:\n")
		for _, sub := range doc.Subcommands {
			fmt.Fprintf(stdout, "  %s\t%s\n", sub.Name, sub.Summary)
		}
	}
	return nil
}

type multiValue []string

func (m *multiValue) String() string         { return strings.Join(*m, ",") }
func (m *multiValue) Set(value string) error { *m = append(*m, value); return nil }

func runAnnotate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("annotate", flag.ContinueOnError)
	var aliases multiValue
	var tags multiValue
	var notes multiValue
	fs.Var(&aliases, "alias", "add a local alias (repeatable or comma-separated)")
	fs.Var(&tags, "tag", "add a local tag (repeatable or comma-separated)")
	fs.Var(&notes, "note", "add a local note (repeatable)")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return errors.New("usage: cmdatlas annotate [--alias NAME] [--tag NAME] [--note TEXT] COMMAND")
	}

	commandName := fs.Args()[0]
	aliasList := splitCSV(aliases)
	tagList := splitCSV(tags)
	noteList := dedupe([]string(notes))
	if len(aliasList) == 0 && len(tagList) == 0 && len(noteList) == 0 {
		return errors.New("annotate requires at least one --alias, --tag, or --note")
	}

	indexPath, err := atlas.DefaultIndexPath()
	if err != nil {
		return err
	}
	index, err := atlas.Load(indexPath)
	if err != nil {
		return err
	}
	index, err = atlas.SetAnnotations(index, commandName, aliasList, tagList, noteList)
	if err != nil {
		return err
	}
	if err := atlas.Save(indexPath, index); err != nil {
		return err
	}

	updated, _ := atlas.Find(index, commandName)
	fmt.Fprintf(stdout, "Updated %s\n", updated.Name)
	if len(updated.Aliases) > 0 {
		fmt.Fprintf(stdout, "Aliases: %s\n", strings.Join(updated.Aliases, ", "))
	}
	if len(updated.Tags) > 0 {
		fmt.Fprintf(stdout, "Tags: %s\n", strings.Join(updated.Tags, ", "))
	}
	if len(updated.Notes) > 0 {
		fmt.Fprintf(stdout, "Notes:\n")
		for _, note := range updated.Notes {
			fmt.Fprintf(stdout, "  - %s\n", note)
		}
	}
	fmt.Fprintf(stdout, "Saved index: %s\n", indexPath)
	return nil
}

func runProfiles(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: cmdatlas profiles [list|set|add|remove|delete|export|import] ...")
	}
	indexPath, err := atlas.DefaultIndexPath()
	if err != nil {
		return err
	}
	index, err := atlas.Load(indexPath)
	if err != nil {
		return err
	}

	switch args[0] {
	case "list":
		for _, name := range atlas.ProfileNames(index) {
			commands, _ := atlas.RawCommandsForProfile(index, name)
			source := profileSourceLabel(index, name)
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", name, source, strings.Join(commands, ", "))
		}
		return nil
	case "set":
		if len(args) < 3 {
			return errors.New("usage: cmdatlas profiles set NAME COMMAND [COMMAND ...]")
		}
		name := strings.ToLower(strings.TrimSpace(args[1]))
		index, err = atlas.SetProfile(index, name, args[2:])
		if err != nil {
			return err
		}
		if err := atlas.Save(indexPath, index); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Saved profile %s: %s\n", name, strings.Join(index.Profiles[name], ", "))
		fmt.Fprintf(stdout, "Saved index: %s\n", indexPath)
		return nil
	case "add":
		if len(args) < 3 {
			return errors.New("usage: cmdatlas profiles add NAME COMMAND [COMMAND ...]")
		}
		name := strings.ToLower(strings.TrimSpace(args[1]))
		var added []string
		index, added, err = atlas.AddToProfile(index, name, args[2:])
		if err != nil {
			return err
		}
		if err := atlas.Save(indexPath, index); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Updated profile %s: %s\n", name, strings.Join(index.Profiles[name], ", "))
		fmt.Fprintf(stdout, "Added: %s\n", noneIfEmpty(added))
		fmt.Fprintf(stdout, "Saved index: %s\n", indexPath)
		return nil
	case "remove", "rm":
		if len(args) < 3 {
			return errors.New("usage: cmdatlas profiles remove NAME COMMAND [COMMAND ...]")
		}
		name := strings.ToLower(strings.TrimSpace(args[1]))
		var removed []string
		index, removed, err = atlas.RemoveFromProfile(index, name, args[2:])
		if err != nil {
			return err
		}
		if err := atlas.Save(indexPath, index); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Updated profile %s: %s\n", name, strings.Join(index.Profiles[name], ", "))
		fmt.Fprintf(stdout, "Removed: %s\n", noneIfEmpty(removed))
		fmt.Fprintf(stdout, "Saved index: %s\n", indexPath)
		return nil
	case "delete":
		if len(args) != 2 {
			return errors.New("usage: cmdatlas profiles delete NAME")
		}
		name := strings.ToLower(strings.TrimSpace(args[1]))
		index, err = atlas.DeleteProfile(index, name)
		if err != nil {
			return err
		}
		if err := atlas.Save(indexPath, index); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Deleted profile %s\n", name)
		fmt.Fprintf(stdout, "Saved index: %s\n", indexPath)
		return nil
	case "export":
		return runProfilesExport(index, args[1:], stdout)
	case "import":
		return runProfilesImport(indexPath, index, args[1:], stdout)
	default:
		return fmt.Errorf("unknown profiles command %q", args[0])
	}
}

func runProfilesExport(index atlas.Index, args []string, stdout io.Writer) error {
	var jsonOutput bool
	var names []string
	for _, arg := range args {
		if strings.TrimSpace(arg) == "--json" {
			jsonOutput = true
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(arg), "-") {
			return errors.New("usage: cmdatlas profiles export [NAME] --json")
		}
		names = append(names, arg)
	}
	if !jsonOutput || len(names) > 1 {
		return errors.New("usage: cmdatlas profiles export [NAME] --json")
	}

	exported := map[string][]string{}
	if len(names) == 1 {
		name := strings.ToLower(strings.TrimSpace(names[0]))
		commands, ok := index.Profiles[name]
		if !ok {
			return fmt.Errorf("profile %q does not exist", name)
		}
		exported[name] = append([]string(nil), commands...)
	} else {
		for name, commands := range index.Profiles {
			exported[name] = append([]string(nil), commands...)
		}
	}
	exportedMeta := map[string]atlas.ProfileMetadata{}
	for name := range exported {
		meta := index.ProfileMeta[name]
		if meta.Origin == "" {
			meta.Origin = "custom"
		}
		meta.ExportedAt = nowUTC()
		exportedMeta[name] = meta
	}
	return writeJSON(stdout, profilesJSONExport{Profiles: exported, ProfileMeta: exportedMeta})
}

func runProfilesImport(indexPath string, index atlas.Index, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("profiles import", flag.ContinueOnError)
	filePath := fs.String("file", "", "read profile JSON from file (defaults to stdin)")
	replace := fs.Bool("replace", false, "replace existing custom profiles before import")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return errors.New("usage: cmdatlas profiles import [--replace] [--file PATH]")
	}

	data, err := readProfilesImportData(*filePath)
	if err != nil {
		return err
	}
	var payload profilesJSONExport
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode profile import: %w", err)
	}
	if len(payload.Profiles) == 0 {
		return errors.New("import requires at least one profile")
	}

	plan := buildProfileImportPlan(index, payload, *replace, sourceLabelForImport(*filePath))
	if *replace {
		index.Profiles = nil
		index.ProfileMeta = nil
	}
	importedNames := make([]string, 0, len(payload.Profiles))
	for _, step := range plan {
		var setErr error
		index, setErr = atlas.SetProfile(index, step.Name, step.Incoming)
		if setErr != nil {
			return setErr
		}
		if index.ProfileMeta == nil {
			index.ProfileMeta = map[string]atlas.ProfileMetadata{}
		}
		meta := payload.ProfileMeta[step.Name]
		if meta.Origin == "" {
			meta.Origin = "imported"
		}
		meta.ImportedFrom = step.ImportedBy
		meta.ImportedAt = nowUTC()
		index.ProfileMeta[step.Name] = meta
		importedNames = append(importedNames, step.Name)
	}
	if err := atlas.Save(indexPath, index); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Imported profiles: %s\n", strings.Join(importedNames, ", "))
	if *replace {
		fmt.Fprintf(stdout, "Mode: replace\n")
	} else {
		fmt.Fprintf(stdout, "Mode: merge\n")
	}
	for _, step := range plan {
		fmt.Fprintf(stdout, "%s: %s\n", step.Name, describeProfileImportPlan(step))
	}
	fmt.Fprintf(stdout, "Saved index: %s\n", indexPath)
	return nil
}

func readProfilesImportData(filePath string) ([]byte, error) {
	if strings.TrimSpace(filePath) == "" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(filePath)
}

func buildProfileImportPlan(index atlas.Index, payload profilesJSONExport, replace bool, importedBy string) []profileImportPlan {
	plans := make([]profileImportPlan, 0, len(payload.Profiles))
	for name, commands := range payload.Profiles {
		name = strings.ToLower(strings.TrimSpace(name))
		incoming := append([]string(nil), commands...)
		sort.Strings(incoming)
		step := profileImportPlan{Name: name, Incoming: incoming, ImportedBy: importedBy}
		if !replace {
			step.Previous = append([]string(nil), index.Profiles[name]...)
			sort.Strings(step.Previous)
		}
		switch {
		case replace:
			step.Action = "replace"
		case len(step.Previous) == 0:
			step.Action = "create"
		case strings.Join(step.Previous, ",") == strings.Join(step.Incoming, ","):
			step.Action = "unchanged"
		default:
			step.Action = "merge"
		}
		plans = append(plans, step)
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].Name < plans[j].Name })
	return plans
}

func describeProfileImportPlan(plan profileImportPlan) string {
	switch plan.Action {
	case "create":
		return fmt.Sprintf("new shared profile from %s (%s)", plan.ImportedBy, strings.Join(plan.Incoming, ", "))
	case "unchanged":
		return fmt.Sprintf("unchanged; local and imported definitions match (%s)", strings.Join(plan.Incoming, ", "))
	case "replace":
		return fmt.Sprintf("replaced with shared definition from %s (%s)", plan.ImportedBy, strings.Join(plan.Incoming, ", "))
	default:
		return fmt.Sprintf("merged over local definition; was [%s], now [%s]", strings.Join(plan.Previous, ", "), strings.Join(plan.Incoming, ", "))
	}
}

func sourceLabelForImport(filePath string) string {
	if strings.TrimSpace(filePath) == "" {
		return "stdin"
	}
	return filePath
}

func profileSourceLabel(index atlas.Index, name string) string {
	if atlas.IsBuiltInProfile(name) {
		return "built-in"
	}
	meta := index.ProfileMeta[name]
	if meta.Origin == "imported" {
		if meta.ImportedFrom != "" {
			return fmt.Sprintf("imported (%s)", meta.ImportedFrom)
		}
		return "imported"
	}
	return "custom"
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func runExport(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "emit JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*jsonOutput {
		return errors.New("usage: cmdatlas export --json")
	}

	index, err := loadIndex()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = stdout.Write(data)
	return err
}

func loadIndex() (atlas.Index, error) {
	indexPath, err := atlas.DefaultIndexPath()
	if err != nil {
		return atlas.Index{}, err
	}
	return atlas.Load(indexPath)
}

func newScanWarning(command string, err error) scanWarning {
	warning := scanWarning{Command: command, Kind: "probe_failed", Message: err.Error()}
	if errors.Is(err, exec.ErrNotFound) {
		warning.Kind = "not_found"
	}
	return warning
}

func (w scanWarning) String() string {
	return fmt.Sprintf("%s [%s]: %s", w.Command, w.Kind, w.Message)
}

func writeJSON(stdout io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = stdout.Write(data)
	return err
}

func writeScanList(stdout io.Writer, label string, values []string) {
	if len(values) == 0 {
		fmt.Fprintf(stdout, "%s: none\n", label)
		return
	}
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	fmt.Fprintf(stdout, "%s: %s\n", label, strings.Join(sorted, ", "))
}

func dedupe(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func splitCSV(values []string) []string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			parts = append(parts, part)
		}
	}
	return dedupe(parts)
}

func noneIfEmpty(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	return strings.Join(sorted, ", ")
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "cmdatlas - local command atlas for humans and agents")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  cmdatlas scan [--json] [--profile NAME] [COMMAND ...]")
	fmt.Fprintln(w, "  cmdatlas search [--json] QUERY")
	fmt.Fprintln(w, "  cmdatlas show [--json] COMMAND")
	fmt.Fprintln(w, "  cmdatlas annotate [--alias NAME] [--tag NAME] [--note TEXT] COMMAND")
	fmt.Fprintln(w, "  cmdatlas profiles list")
	fmt.Fprintln(w, "  cmdatlas profiles set NAME COMMAND [COMMAND ...]")
	fmt.Fprintln(w, "  cmdatlas profiles add NAME COMMAND [COMMAND ...]")
	fmt.Fprintln(w, "  cmdatlas profiles remove NAME COMMAND [COMMAND ...]")
	fmt.Fprintln(w, "  cmdatlas profiles delete NAME")
	fmt.Fprintln(w, "  cmdatlas profiles export [NAME] --json")
	fmt.Fprintln(w, "  cmdatlas profiles import [--replace] [--file PATH]")
	fmt.Fprintln(w, "  cmdatlas export --json")
	fmt.Fprintln(w, "  cmdatlas completion [bash|zsh|fish|powershell]")
	fmt.Fprintln(w, "  cmdatlas completion install [bash|zsh|fish|powershell]")
}
