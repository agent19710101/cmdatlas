package atlas

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

func Merge(existing Index, docs []CommandDoc, scannedSet []string) Index {
	byName := make(map[string]CommandDoc, len(existing.Commands)+len(docs))
	for _, doc := range existing.Commands {
		byName[doc.Name] = doc
	}
	for _, doc := range docs {
		if existingDoc, ok := byName[doc.Name]; ok {
			doc.Aliases = append([]string(nil), existingDoc.Aliases...)
			doc.Tags = append([]string(nil), existingDoc.Tags...)
			doc.Notes = append([]string(nil), existingDoc.Notes...)
		}
		byName[doc.Name] = doc
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	commands := make([]CommandDoc, 0, len(names))
	for _, name := range names {
		commands = append(commands, byName[name])
	}

	sort.Strings(scannedSet)

	return Index{
		Version:    CurrentIndexVersion,
		Generated:  time.Now().UTC(),
		Commands:   commands,
		ScannedSet: scannedSet,
	}
}

func Find(index Index, name string) (CommandDoc, bool) {
	for _, doc := range index.Commands {
		if strings.EqualFold(doc.Name, name) {
			return doc, true
		}
	}
	return CommandDoc{}, false
}

func DocsEquivalent(a CommandDoc, b CommandDoc) bool {
	return a.Name == b.Name &&
		a.Path == b.Path &&
		a.Summary == b.Summary &&
		reflect.DeepEqual(a.HelpLines, b.HelpLines) &&
		reflect.DeepEqual(a.Flags, b.Flags) &&
		reflect.DeepEqual(a.Subcommands, b.Subcommands) &&
		a.Probe == b.Probe
}

func SetAnnotations(index Index, name string, aliases []string, tags []string, notes []string) (Index, error) {
	for i, doc := range index.Commands {
		if !strings.EqualFold(doc.Name, name) {
			continue
		}
		doc.Aliases = normalizeAnnotations(aliases)
		doc.Tags = normalizeAnnotations(tags)
		doc.Notes = normalizeNotes(notes)
		index.Commands[i] = doc
		index.Generated = time.Now().UTC()
		if index.Version == 0 {
			index.Version = CurrentIndexVersion
		}
		return index, nil
	}
	return index, fmt.Errorf("command %q is not indexed", name)
}

func normalizeAnnotations(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func normalizeNotes(values []string) []string {
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
