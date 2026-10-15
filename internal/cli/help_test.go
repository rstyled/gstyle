package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
			for _, want := range []string{"Usage:", "--control-port", "--skill"} {
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

func TestCommandHelpUsesRegisteredFlags(t *testing.T) {
	for _, command := range newRoot(io.Discard).Commands() {
		name := command.Name()
		t.Run(name, func(t *testing.T) {
			var direct, topic bytes.Buffer
			if err := run([]string{name, "--help"}, &direct); err != nil {
				t.Fatal(err)
			}
			if err := run([]string{"help", name}, &topic); err != nil {
				t.Fatal(err)
			}
			if direct.String() != topic.String() || !strings.Contains(direct.String(), "Flags:") {
				t.Fatalf("command and topic help differ: %q / %q", direct.String(), topic.String())
			}
			command.Flags().VisitAll(func(value *pflag.Flag) {
				if !strings.Contains(direct.String(), "--"+value.Name) {
					t.Fatalf("registered flag %s missing from help", value.Name)
				}
			})
		})
	}
}

func TestHelpReturnsOutputError(t *testing.T) {
	want := errors.New("output closed")
	if err := run([]string{"--help"}, failedHelpWriter{want}); !errors.Is(err, want) {
		t.Fatalf("error=%v, want %v", err, want)
	}
}

type failedHelpWriter struct{ err error }

func (w failedHelpWriter) Write([]byte) (int, error) { return 0, w.err }

func TestCommandKeepsGoFlagsAndCheckerArguments(t *testing.T) {
	flags := flag.NewFlagSet("example", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Bool("tui", false, "terminal UI")
	flags.String("output", "", "output file")
	arguments := []string{"-tui=false", "-output", "result.go", "input.go", "--", "checker", "--help", "--other"}
	var received []string
	runner := func(_ context.Context, args []string, _ io.Writer) error {
		received = args
		return nil
	}
	command := leaf(&cobra.Command{Use: "example"}, flags, runner, io.Discard)
	command.SetArgs(arguments)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(received, arguments) {
		t.Fatalf("checker arguments changed: %q", received)
	}
}
