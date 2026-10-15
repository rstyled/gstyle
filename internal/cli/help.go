package cli

import (
	"errors"
	"io"
)

const usage = `Usage: gstyle [OPTIONS] [DIRECTORIES OR GO FILES...]
       gstyle COMMAND [OPTIONS] [ARGUMENTS...]

Inspect Go code in the terminal UI, or run a command below.
Without paths, the UI inspects the current directory.

Commands:
  check         Report code quality findings
  patterns      Report pattern candidates
  fix           Apply supported pattern fixes
  next          Show the highest-impact pattern candidate
  score         Show the pattern score
  match-graphs  Match imported graphs or validated Go source
  bug-reducer   Minimize an input with a checker command
  help          Show this help

UI options:
  --config PATH       Quality TOML policy
  --control-port N    Local JSON-RPC port (default: 8197; 0 selects a free port)

Other options:
  --skill       Print agent instructions
  -h, --help    Show this help

Examples:
  gstyle .
  gstyle check --format json .
  gstyle patterns .
`

func writeHelp(args []string, output io.Writer) error {
	if len(args) != 0 {
		return errors.New("usage: gstyle help")
	}
	_, err := io.WriteString(output, usage)
	return err
}
