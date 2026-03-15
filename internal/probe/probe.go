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
	probeTimeout = 2 * time.Second
	maxBytes     = 16 * 1024
	maxLines     = 24
)

var helpVariants = [][]string{
	{"--help"},
	{"help"},
	{"-h"},
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

	return atlas.CommandDoc{
		Name:        filepath.Base(name),
		Path:        path,
		Summary:     summary,
		HelpLines:   lines,
		Flags:       detectFlags(lines),
		Subcommands: detectSubcommands(lines),
		Probe:       probeName,
		ScannedAt:   time.Now().UTC(),
	}, nil
}

func collectHelp(name string) (string, string, error) {
	var errs []string
	for _, variant := range helpVariants {
		out, err := run(name, variant...)
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
