package probe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/agent19710101/cmdatlas/internal/atlas"
)

const (
	probeTimeout              = 2 * time.Second
	maxBytes                  = 16 * 1024
	maxLines                  = 24
	maxNestedSubcommandDepth  = 2
	maxNestedSubcommandProbes = 16
)

var helpVariants = [][]string{
	{"--help"},
	{"help"},
	{"-h"},
}

var nestedCLIAllowlist = map[string]struct{}{
	"docker":  {},
	"gh":      {},
	"git":     {},
	"kubectl": {},
}

func ScanCommand(name string) (atlas.CommandDoc, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return atlas.CommandDoc{}, err
	}

	output, probeName, err := collectHelp(name)
	if err != nil {
		return atlas.CommandDoc{}, err
	}

	lines := normalizeLines(output, maxLines)
	summary := detectSummary(lines)
	subcommands := detectSubcommands(lines)
	if shouldProbeNestedSubcommands(filepath.Base(name)) {
		subcommands = append(subcommands, detectNestedSubcommands(name, subcommands)...)
		subcommands = dedupeSubcommands(subcommands)
	}

	return atlas.CommandDoc{
		Name:        filepath.Base(name),
		Path:        path,
		Summary:     summary,
		HelpLines:   lines,
		Flags:       detectFlags(lines),
		Subcommands: subcommands,
		Probe:       probeName,
		ScannedAt:   time.Now().UTC(),
	}, nil
}

func collectHelp(command string) (string, string, error) {
	parts := strings.Fields(strings.TrimSpace(command))
	if len(parts) == 0 {
		return "", "", errors.New("empty command")
	}
	name := parts[0]
	baseArgs := parts[1:]

	var errs []string
	for _, variant := range helpVariants {
		args := append(append([]string(nil), baseArgs...), variant...)
		out, err := run(name, args...)
		if strings.TrimSpace(out) != "" {
			return out, strings.Join(variant, " "), nil
		}
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", strings.Join(variant, " "), err))
		}
	}
	if len(errs) == 0 {
		return "", "", errors.New("no help output received")
	}
	return "", "", errors.New(strings.Join(errs, "; "))
}

func detectNestedSubcommands(name string, topLevel []atlas.Subcommand) []atlas.Subcommand {
	queue := make([][]string, 0, len(topLevel))
	for _, sub := range topLevel {
		parts := strings.Fields(strings.TrimSpace(sub.Name))
		if len(parts) != 1 {
			continue
		}
		queue = append(queue, parts)
	}

	var nested []atlas.Subcommand
	probes := 0
	for len(queue) > 0 && probes < maxNestedSubcommandProbes {
		pathParts := queue[0]
		queue = queue[1:]
		if len(pathParts) >= maxNestedSubcommandDepth {
			continue
		}

		output, _, err := collectHelp(strings.Join(append([]string{name}, pathParts...), " "))
		probes++
		if err != nil {
			continue
		}
		children := detectSubcommands(normalizeLines(output, maxLines))
		for _, child := range children {
			childParts := strings.Fields(strings.TrimSpace(child.Name))
			if len(childParts) != 1 {
				continue
			}
			fullPath := append(append([]string(nil), pathParts...), childParts[0])
			nested = append(nested, atlas.Subcommand{
				Name:    strings.Join(fullPath, " "),
				Summary: child.Summary,
			})
			queue = append(queue, fullPath)
		}
	}
	return nested
}

func dedupeSubcommands(subcommands []atlas.Subcommand) []atlas.Subcommand {
	seen := map[string]struct{}{}
	out := make([]atlas.Subcommand, 0, len(subcommands))
	for _, sub := range subcommands {
		name := strings.TrimSpace(sub.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		sub.Name = name
		out = append(out, sub)
	}
	return out
}

func shouldProbeNestedSubcommands(name string) bool {
	_, ok := nestedCLIAllowlist[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func run(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &limitedWriter{buf: &buf, limit: maxBytes}
	cmd.Stderr = &limitedWriter{buf: &buf, limit: maxBytes}
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return buf.String(), fmt.Errorf("timed out after %s", probeTimeout)
	}
	return buf.String(), err
}

type limitedWriter struct {
	buf   *bytes.Buffer
	limit int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.buf.Len()
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	_, err := w.buf.Write(p)
	return len(p), err
}
