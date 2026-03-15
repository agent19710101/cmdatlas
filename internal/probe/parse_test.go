package probe

import (
	"testing"
)

func TestNormalizeAndParseHelp(t *testing.T) {
	text := `
Usage: demo [command]

Demo tool for testing

Available Commands:
  scan        build the atlas
  export      emit JSON

Flags:
  -h, --help   show help
  --json       emit json
`

	lines := normalizeLines(text, 20)
	if got, want := detectSummary(lines), "Demo tool for testing"; got != want {
		t.Fatalf("detectSummary() = %q, want %q", got, want)
	}

	flags := detectFlags(lines)
	if len(flags) < 3 {
		t.Fatalf("detectFlags() found %d flags, want at least 3", len(flags))
	}

	subs := detectSubcommands(lines)
	if len(subs) != 2 {
		t.Fatalf("detectSubcommands() found %d subcommands, want 2", len(subs))
	}
	if subs[0].Name != "scan" || subs[1].Name != "export" {
		t.Fatalf("unexpected subcommands: %#v", subs)
	}
}

func TestDetectSummaryFallsBackGracefully(t *testing.T) {
	lines := []string{"Usage: demo [flags]", "Flags:", "  -h, --help show help"}
	if got := detectSummary(lines); got != "-h, --help show help" {
		t.Fatalf("detectSummary() = %q, want %q", got, "-h, --help show help")
	}
}
