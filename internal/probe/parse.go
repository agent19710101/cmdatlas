package probe

import (
	"regexp"
	"strings"

	"github.com/agent19710101/cmdatlas/internal/atlas"
)

var (
	flagTokenPattern   = regexp.MustCompile(`(--?[a-zA-Z0-9][a-zA-Z0-9-]*)`)
	subHeadingPattern  = regexp.MustCompile(`(?i)^((available )?(commands|subcommands|management commands)|the commands are|these are .* commands.*):\s*$`)
	subcommandPattern  = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9:_-]*)\s{2,}(.+)$`)
	stopSectionPattern = regexp.MustCompile(`(?i)^(flags|options|examples|usage):\s*$`)
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
		if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "<") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") {
			continue
		}
		return trimmed
	}
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
			if stopSectionPattern.MatchString(trimmed) {
				inSection = false
				continue
			}
			matches := subcommandPattern.FindStringSubmatch(trimmed)
			if len(matches) != 3 {
				continue
			}
			name := matches[1]
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			summary := strings.TrimSpace(matches[2])
			subs = append(subs, atlas.Subcommand{Name: name, Summary: summary})
			continue
		}
	}
	return subs
}
