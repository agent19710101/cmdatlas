package atlas

import (
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
