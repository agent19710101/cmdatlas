package probe

import (
	"regexp"
	"strings"

	"github.com/agent19710101/cmdatlas/internal/atlas"
)

var (
	flagTokenPattern   = regexp.MustCompile(`(--?[a-zA-Z0-9][a-zA-Z0-9-]*)`)
	subHeadingPattern  = regexp.MustCompile(`(?i)^(available )?(commands|subcommands|management commands):\s*$`)
	usagePrefixPattern = regexp.MustCompile(`(?i)^usage:\s*`)
)

func normalizeLines(text string, limit int) []string {
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, limit)
	for _, line := range raw {
		line = strings.TrimRight(line, "\r\t ")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) >= limit {
			break
		}
	}
	return lines
}

func detectSummary(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if usagePrefixPattern.MatchString(trimmed) {
			continue
		}
		if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") {
			continue
		}
		return trimmed
	}
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}

func detectFlags(lines []string) []atlas.FlagDoc {
	seen := map[string]struct{}{}
	var flags []atlas.FlagDoc
	for _, line := range lines {
		if !strings.Contains(line, "-") {
			continue
		}
		matches := flagTokenPattern.FindAllString(line, -1)
		for _, match := range matches {
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			flags = append(flags, atlas.FlagDoc{
				Name:    match,
				Example: strings.TrimSpace(line),
			})
		}
	}
	return flags
}

func detectSubcommands(lines []string) []atlas.Subcommand {
	var subs []atlas.Subcommand
	inSection := false
	seen := map[string]struct{}{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if subHeadingPattern.MatchString(trimmed) {
			inSection = true
			continue
		}
		if inSection {
			if strings.HasSuffix(trimmed, ":") && !strings.Contains(strings.ToLower(trimmed), "command") {
				inSection = false
				continue
			}
			fields := strings.Fields(trimmed)
			if len(fields) == 0 {
				continue
			}
			name := fields[0]
			if strings.HasPrefix(name, "-") {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			summary := strings.TrimSpace(strings.TrimPrefix(trimmed, name))
			summary = strings.TrimLeft(summary, "- ")
			subs = append(subs, atlas.Subcommand{Name: name, Summary: summary})
			continue
		}
	}
	return subs
}
