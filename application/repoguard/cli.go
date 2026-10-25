package repoguard

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/rstyled/gstyle/adapters/gitchanged"
	"github.com/rstyled/gstyle/adapters/sarif"
	"github.com/rstyled/gstyle/application/qualitycheck"
	"github.com/rstyled/gstyle/domain/guard"
)

var ErrFindings = errors.New("repository policy violations")

type options struct {
	config, base, format string
	paths                []string
}

type report struct {
	SchemaVersion int             `json:"schema_version"`
	Findings      []guard.Finding `json:"findings"`
}

func Flags() *flag.FlagSet { return flagsFor(&options{}) }

func flagsFor(opts *options) *flag.FlagSet {
	flags := flag.NewFlagSet("guard", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&opts.config, "config", "", "TOML policy (default: repository gstyle.toml)")
	flags.StringVar(&opts.base, "base", "", "base commit for added comments and files (default: merge-base with main/master)")
	flags.StringVar(&opts.format, "format", "text", "text, json, or sarif")
	return flags
}

func Run(ctx context.Context, args []string, output io.Writer) error {
	opts := options{}
	flags := flagsFor(&opts)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !slices.Contains([]string{"text", "json", "sarif"}, opts.format) {
		return fmt.Errorf("format must be text, json, or sarif")
	}
	opts.paths = flags.Args()
	findings, err := check(ctx, opts)
	if err != nil {
		return err
	}
	if err := writeReport(output, findings, opts.format); err != nil {
		return err
	}
	if len(findings) != 0 {
		return ErrFindings
	}
	return nil
}

func check(ctx context.Context, opts options) ([]guard.Finding, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, err := gitchanged.GitRoot(cwd)
	if err != nil {
		return nil, err
	}
	if opts.config == "" {
		opts.config = filepath.Join(root, "gstyle.toml")
	}
	config, err := qualitycheck.LoadConfig(opts.config)
	if err != nil {
		return nil, err
	}
	repository := repository{root: root, paths: gitchanged.RootSpecs(opts.paths, root, cwd), base: opts.base}
	return repository.inspect(ctx, config.Guard)
}

func writeReport(output io.Writer, findings []guard.Finding, format string) error {
	if format == "json" {
		return json.NewEncoder(output).Encode(report{1, findings})
	}
	if format == "sarif" {
		return writeSARIF(output, findings)
	}
	for _, finding := range findings {
		if _, err := fmt.Fprintf(output, "%s:%d:%d: %s: %s\n", finding.Path, finding.Line, finding.Column, finding.RuleID, finding.Message); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(output, "%d repository policy findings\n", len(findings))
	return err
}

func writeSARIF(output io.Writer, findings []guard.Finding) error {
	items := make([]sarif.Finding, 0, len(findings))
	for _, finding := range findings {
		items = append(items, sarif.Finding{RuleID: finding.RuleID, Message: finding.Message,
			Locations: []sarif.Location{{Path: finding.Path, Line: finding.Line}}})
	}
	return sarif.Write(output, items, "gstyle/guard/")
}
