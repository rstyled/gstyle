package changerisk

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/adapters/gitchanged"
	"github.com/rstyled/gstyle/adapters/gorisk"
	"github.com/rstyled/gstyle/application/qualitycheck"
	"github.com/rstyled/gstyle/domain/changerisk"
	"github.com/rstyled/gstyle/domain/quality"
)

type options struct {
	base, head, config, format string
}

func Flags() *flag.FlagSet { return flagsFor(&options{}) }

func flagsFor(opts *options) *flag.FlagSet {
	flags := flag.NewFlagSet("risk", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&opts.base, "base", "", "base revision (default: merge-base of head with origin/main/master, then local main/master)")
	flags.StringVar(&opts.head, "head", "HEAD", "committed head revision")
	flags.StringVar(&opts.config, "config", "gstyle.toml", "repository-relative policy read from base revision")
	flags.StringVar(&opts.format, "format", "markdown", "markdown or json")
	return flags
}

func Run(ctx context.Context, args []string, output io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	report, err := inspect(ctx, opts)
	if err != nil {
		return err
	}
	if opts.format == "json" {
		return json.NewEncoder(output).Encode(report)
	}
	_, err = io.WriteString(output, changerisk.Markdown(report))
	return err
}

func parseOptions(args []string) (options, error) {
	opts := options{}
	flags := flagsFor(&opts)
	if err := flags.Parse(args); err != nil {
		return opts, err
	}
	if !slices.Contains([]string{"markdown", "json"}, opts.format) || len(flags.Args()) != 0 {
		return opts, fmt.Errorf("risk accepts --format markdown or json and no positional paths")
	}
	if !validConfigPath(opts.config) {
		return opts, fmt.Errorf("risk config must be a clean repository-relative path")
	}
	return opts, nil
}

func validConfigPath(name string) bool {
	return fs.ValidPath(name) && name != "." && !strings.Contains(name, "\\") && filepath.VolumeName(name) == ""
}

func inspect(ctx context.Context, opts options) (changerisk.Report, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return changerisk.Report{}, err
	}
	root, err := gitchanged.GitRoot(cwd)
	if err != nil {
		return changerisk.Report{}, err
	}
	return inspectRepository(ctx, root, opts)
}

func inspectRepository(ctx context.Context, root string, opts options) (changerisk.Report, error) {
	head, err := revision(ctx, root, opts.head)
	if err != nil {
		return changerisk.Report{}, err
	}
	base, err := baseRevision(ctx, root, opts, head)
	if err != nil {
		return changerisk.Report{}, err
	}
	opts.base, opts.head = base, head
	return compareRevisions(ctx, root, opts)
}

func baseRevision(ctx context.Context, root string, opts options, head string) (string, error) {
	if opts.base != "" {
		return revision(ctx, root, opts.base)
	}
	for _, ref := range []string{"origin/main", "origin/master", "main", "master"} {
		data, err := command(ctx, root, "git", "merge-base", head, ref)
		if err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}
	return head, nil
}

func compareRevisions(ctx context.Context, root string, opts options) (changerisk.Report, error) {
	paths, err := changedPaths(ctx, root, opts.base, opts.head)
	if err != nil {
		return changerisk.Report{}, err
	}
	var config changerisk.Config
	comparison := comparison{root: root, paths: paths}
	base, err := comparison.snapshot(ctx, opts.base, func(dir string) error {
		policy, err := baseConfig(dir, opts.config)
		config = policy.Risk
		return err
	})
	if err != nil {
		return changerisk.Report{}, err
	}
	head, err := comparison.snapshot(ctx, opts.head, nil)
	if err != nil {
		return changerisk.Report{}, err
	}
	api := gorisk.CompareAPI(base.API, head.API)
	report := changerisk.Evaluate(base.Source, head.Source, config, api)
	report.Policy = opts.config
	return report, nil
}

func baseConfig(root, name string) (quality.Config, error) {
	current := root
	for _, component := range strings.Split(name, "/") {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return quality.Config{}, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return quality.Config{}, fmt.Errorf("risk base policy must not traverse symlinks")
		}
	}
	return qualitycheck.LoadConfig(current)
}

type comparison struct {
	root  string
	paths []string
}

type repositorySnapshot struct {
	Source changerisk.Snapshot
	API    gorisk.API
}

func (c comparison) snapshot(ctx context.Context, commit string, configure func(string) error) (repositorySnapshot, error) {
	result := repositorySnapshot{}
	err := withSnapshot(ctx, c.root, commit, func(dir string) error {
		if configure != nil {
			if err := configure(dir); err != nil {
				return err
			}
		}
		loaded, err := populateSnapshot(ctx, dir, c.paths)
		result = loaded
		return err
	})
	result.Source.Commit = commit
	return result, err
}

func populateSnapshot(ctx context.Context, root string, changed []string) (repositorySnapshot, error) {
	paths, message, err := gorisk.ExistingPaths(root, changed)
	if err != nil {
		return repositorySnapshot{}, err
	}
	functions, err := gorisk.Extract(ctx, root, paths)
	if err != nil {
		return repositorySnapshot{}, err
	}
	api := gorisk.ExtractAPI(ctx, root)
	items, dependencyErr := dependencies(ctx, root)
	dependencyMessage := ""
	if dependencyErr != nil {
		dependencyMessage = "Go dependency enumeration failed; no dependency verdict is available"
	}
	return repositorySnapshot{Source: changerisk.Snapshot{Functions: functions, AnalysisError: message, Dependencies: items, DependencyError: dependencyMessage}, API: api}, ctx.Err()
}
