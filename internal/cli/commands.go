package cli

import (
	"context"
	"errors"
	"flag"
	"io"
	"os/signal"

	"github.com/rstyled/gstyle/application/bugreducer"
	"github.com/rstyled/gstyle/application/graphmatch"
	"github.com/rstyled/gstyle/application/nextscore"
	"github.com/rstyled/gstyle/application/patterncheck"
	"github.com/rstyled/gstyle/application/patternfix"
	"github.com/rstyled/gstyle/application/qualitycheck"
	"github.com/rstyled/gstyle/application/searchcli"
	"github.com/spf13/cobra"
)

func run(args []string, output io.Writer) error {
	ctx, cancel := signal.NotifyContext(context.Background(), reducerSignals()...)
	defer cancel()
	captured := &helpOutput{Writer: output}
	root := newRoot(output)
	root.SetOut(captured)
	root.SetErr(captured)
	root.SetArgs(args)
	if err := root.ExecuteContext(ctx); err != nil {
		return err
	}
	return captured.err
}

func newRoot(output io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use: "gstyle [DIRECTORIES OR GO FILES...]", Short: "Inspect Go code in the terminal UI or run a headless command",
		Args: cobra.ArbitraryArgs, DisableFlagParsing: true,
		Long:          "Inspect Go code in the terminal UI or run a headless command.\nWithout paths, the UI inspects the current directory.",
		Example:       "  gstyle .\n  gstyle check --format json .\n  gstyle patterns .",
		SilenceErrors: true, SilenceUsage: true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		RunE:              func(cmd *cobra.Command, args []string) error { return runRoot(cmd, args, output) },
	}
	root.Flags().AddGoFlagSet(runFlags(&runOptions{}))
	root.AddCommand(leaf(&cobra.Command{Use: "search [DIRECTORIES OR GO FILES...]", Short: "Search functions and quality findings", Long: "Search functions and typed quality findings without a TUI. Flags precede paths; defaults to .\nResults use the same ranking as getReport. No matches is exit 0; operational failure is exit 2."}, searchcli.Flags(), searchcli.Run, output))
	root.AddCommand(leaf(&cobra.Command{Use: "check [DIRECTORIES OR GO FILES...]", Short: "Report code quality findings"}, qualitycheck.Flags(), qualitycheck.Run, output))
	root.AddCommand(leaf(&cobra.Command{Use: "patterns [DIRECTORIES OR GO FILES...]", Short: "Report pattern candidates"}, patterncheck.Flags(), patterncheck.Run, output))
	root.AddCommand(leaf(&cobra.Command{Use: "fix [DIRECTORIES OR GO FILES...]", Short: "Apply supported pattern fixes"}, patternfix.Flags(), patternfix.Run, output))
	root.AddCommand(leaf(&cobra.Command{Use: "bug-reducer INPUT -- CHECKER [ARGS...]", Short: "Minimize an input with a checker command",
		Long: "Minimize an input while preserving a reproducible bug.\nThe checker receives the absolute candidate file path as its last argument.\nExit 0 accepts a candidate; an ordinary nonzero exit rejects it.\nThe original input is preserved. The output file must not already exist."}, bugreducer.Flags(output), bugreducer.Run, output))
	root.AddCommand(leaf(&cobra.Command{Use: "match-graphs", Short: "Match imported graphs or validated Go source"}, graphmatch.Flags(), graphmatch.Run, output))
	root.AddCommand(leaf(&cobra.Command{Use: "next [DIRECTORIES OR GO FILES...]", Short: "Show the highest-impact pattern candidate"}, patterncheck.Flags(), nextscore.Run, output))
	root.AddCommand(leaf(&cobra.Command{Use: "score [DIRECTORIES OR GO FILES...]", Short: "Show the pattern score"}, patterncheck.Flags(), nextscore.Score, output))
	return root
}

type commandRunner func(context.Context, []string, io.Writer) error

// Keep Go flag syntax and checker separators intact while Cobra owns routing
// and generates help from the same definitions the application parsers use.
func leaf(cmd *cobra.Command, flags *flag.FlagSet, runner commandRunner, output io.Writer) *cobra.Command {
	cmd.Args = cobra.ArbitraryArgs
	cmd.DisableFlagParsing = true
	cmd.Flags().AddGoFlagSet(flags)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := flags.Parse(args)
		if errors.Is(err, flag.ErrHelp) {
			return cmd.Help()
		}
		if err != nil {
			return err
		}
		return runner(cmd.Context(), args, output)
	}
	return cmd
}

func runRoot(cmd *cobra.Command, args []string, output io.Writer) error {
	opts, err := parseRunOptions(args)
	if errors.Is(err, flag.ErrHelp) {
		return cmd.Help()
	}
	if err != nil {
		return err
	}
	if opts.skill {
		return writeSkill(args, output)
	}
	return runTUI(opts, output)
}

// Cobra's help callback cannot return writer errors. Preserve them for Main.
type helpOutput struct {
	io.Writer
	err error
}

func (output *helpOutput) Write(data []byte) (int, error) {
	n, err := output.Writer.Write(data)
	if output.err == nil {
		output.err = err
	}
	return n, err
}
