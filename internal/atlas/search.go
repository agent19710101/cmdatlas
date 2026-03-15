package atlas

import (
	"sort"
	"strings"
)

func Search(index Index, query string) []CommandDoc {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil
	}

	type scored struct {
		doc   CommandDoc
		score int
	}

	var matches []scored
	for _, doc := range index.Commands {
		score := scoreDoc(doc, query)
		if score > 0 {
			matches = append(matches, scored{doc: doc, score: score})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			return matches[i].doc.Name < matches[j].doc.Name
		}
		return matches[i].score > matches[j].score
	})

	out := make([]CommandDoc, 0, len(matches))
	for _, match := range matches {
		out = append(out, match.doc)
	}
	return out
}

func scoreDoc(doc CommandDoc, query string) int {
	score := 0
	name := strings.ToLower(doc.Name)
	if name == query {
		score += 100
	} else if strings.Contains(name, query) {
		score += 50
	}
	if strings.Contains(strings.ToLower(doc.Summary), query) {
		score += 20
	}
	for _, line := range doc.HelpLines {
		if strings.Contains(strings.ToLower(line), query) {
			score += 5
		}
	}
	for _, flag := range doc.Flags {
		if strings.Contains(strings.ToLower(flag.Name), query) {
			score += 10
		}
	}
	for _, sub := range doc.Subcommands {
		if strings.Contains(strings.ToLower(sub.Name), query) || strings.Contains(strings.ToLower(sub.Summary), query) {
			score += 10
		}
	}
	return score
}
