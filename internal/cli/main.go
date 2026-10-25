package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/rstyled/gstyle/adapters/gocyclo"
	"github.com/rstyled/gstyle/adapters/goquality"
	"github.com/rstyled/gstyle/adapters/sqlite"
	"github.com/rstyled/gstyle/application"
	"github.com/rstyled/gstyle/application/qualitycheck"
)

const defaultControlPort = 8197

type runOptions struct {
	controlPort int
	paths       []string
	config      string
	skill       bool
}

// Main runs the CLI and exits with the command status on failure.
func Main() {
	err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(commandExitCode(os.Args[1:], err))
	}
}

// commandExitCode maps a failed subcommand to its process exit code.
// Findings are exit 1; hard failures exit 2. patterns never reports
// findings, so any of its errors is a hard failure.
func commandExitCode(args []string, err error) int {
	if len(args) == 0 {
		return 1
	}
	switch args[0] {
	case "check":
		if errors.Is(err, qualitycheck.ErrFindings) {
			return 1
		}
		return 2
	case "patterns", "match-graphs", "search":
		return 2
	case "fix":
		return 2
	default:
		return 1
	}
}

func runTUI(options runOptions, output io.Writer) (result error) {
	config, err := qualitycheck.LoadConfig(options.config)
	if err != nil {
		return err
	}
	statePath, err := sqlite.StatePath()
	if err != nil {
		return err
	}
	store, err := sqlite.Open(statePath)
	if err != nil {
		return err
	}
	defer func() {
		result = errors.Join(result, store.Close())
	}()

	control, err := application.NewControlServer(options.controlPort)
	if err != nil {
		return err
	}

	analyzer := goquality.ReportAnalyzer{Complexity: gocyclo.NewAnalyzer(), Config: config, Tests: true}
	model := application.NewModel(analyzer, options.paths).
		WithAnnotationStore(store).
		WithControlPort(control.Port())
	program := tea.NewProgram(model, tea.WithInput(os.Stdin), tea.WithOutput(output))
	control.Start(program.Send)
	_, programErr := program.Run()
	closeErr := control.Close()
	return errors.Join(programErr, closeErr)
}

func runFlags(result *runOptions) *flag.FlagSet {
	flags := flag.NewFlagSet("gstyle", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.IntVar(&result.controlPort, "control-port", defaultControlPort, "localhost JSON-RPC port; 0 selects a free port")
	flags.StringVar(&result.config, "config", "", "quality TOML policy")
	flags.BoolVar(&result.skill, "skill", false, "print agent instructions")
	return flags
}

func parseRunOptions(args []string) (runOptions, error) {
	result := runOptions{}
	flags := runFlags(&result)
	if err := flags.Parse(args); err != nil {
		return runOptions{}, err
	}
	if result.controlPort < 0 || result.controlPort > 65535 {
		return runOptions{}, errors.New("control port must be between 0 and 65535")
	}

	result.paths = flags.Args()
	return result, nil
}

func writeSkill(args []string, output io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: gstyle --skill")
	}
	return application.WriteSkill(output)
}
