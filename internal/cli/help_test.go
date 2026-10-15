package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRootHelpDoesNotStartTUIOrCreateState(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	for _, args := range [][]string{{"help"}, {"-h"}, {"--help"}, {"--config", "missing.toml", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var output bytes.Buffer
			if err := run(args, &output); err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"Usage: gstyle", "check", "patterns", "bug-reducer", "match-graphs", "--control-port", "--skill"} {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("help lacks %q: %s", want, output.String())
				}
			}
			if strings.Contains(output.String(), "\x1b") {
				t.Fatalf("help contains terminal escapes: %q", output.String())
			}
		})
	}
	entries, err := os.ReadDir(state)
	if err != nil || len(entries) != 0 {
		t.Fatalf("help creates state: entries=%v err=%v", entries, err)
	}
}

func TestHelpRejectsExtraArguments(t *testing.T) {
	if err := run([]string{"help", "unexpected"}, &bytes.Buffer{}); err == nil {
		t.Fatal("help accepts extra arguments")
	}
}

func TestHelpReturnsOutputError(t *testing.T) {
	want := errors.New("output closed")
	if err := writeHelp(nil, failedHelpWriter{want}); !errors.Is(err, want) {
		t.Fatalf("error=%v, want %v", err, want)
	}
}

type failedHelpWriter struct{ err error }

func (w failedHelpWriter) Write([]byte) (int, error) { return 0, w.err }
