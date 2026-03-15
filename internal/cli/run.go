package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/agent19710101/cmdatlas/internal/atlas"
	"github.com/agent19710101/cmdatlas/internal/probe"
)

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
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}

	targets := fs.Args()
	if len(targets) == 0 {
		targets = atlas.DefaultCommands()
		if len(targets) == 0 {
			return errors.New("no default commands found on PATH")
		}
	}

	indexPath, err := atlas.DefaultIndexPath()
	if err != nil {
		return err
	}
	index, err := atlas.Load(indexPath)
	if err != nil {
		return err
	}

	var docs []atlas.CommandDoc
	var failures []string
	for _, target := range dedupe(targets) {
		doc, err := probe.ScanCommand(target)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s (%v)", target, err))
			continue
		}
		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		return fmt.Errorf("scan failed: %s", strings.Join(failures, ", "))
	}

	index = atlas.Merge(index, docs, dedupe(targets))
	if err := atlas.Save(indexPath, index); err != nil {
		return err
	}

	for _, doc := range docs {
		fmt.Fprintf(stdout, "%s\t%s\n", doc.Name, firstNonEmpty(doc.Summary, "indexed"))
	}
	if len(failures) > 0 {
		fmt.Fprintf(stdout, "\nWarnings:\n")
		for _, failure := range failures {
			fmt.Fprintf(stdout, "- %s\n", failure)
		}
	}
	fmt.Fprintf(stdout, "\nSaved index: %s\n", indexPath)
	return nil
}

func runSearch(args []string, stdout io.Writer) error {
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

func writeJSON(stdout io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = stdout.Write(data)
	return err
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
	fmt.Fprintln(w, "  cmdatlas scan [COMMAND ...]")
	fmt.Fprintln(w, "  cmdatlas search [--json] QUERY")
	fmt.Fprintln(w, "  cmdatlas show [--json] COMMAND")
	fmt.Fprintln(w, "  cmdatlas export --json")
	fmt.Fprintln(w, "  cmdatlas completion [bash|zsh|fish|powershell]")
}
