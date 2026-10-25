package changerisk

import (
	"context"
	"encoding/json"
	"errors"
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
	check                      bool
}

var ErrReviewRequired = errors.New("merge risk requires human review")

func Flags() *flag.FlagSet { return flagsFor(&options{}) }

func flagsFor(opts *options) *flag.FlagSet {
	flags := flag.NewFlagSet("risk", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&opts.base, "base", "", "base revision (default: merge-base of head with origin/main/master, then local main/master)")
	flags.StringVar(&opts.head, "head", "HEAD", "committed head revision")
	flags.StringVar(&opts.config, "config", "gstyle.toml", "repository-relative policy read from base revision")
	flags.StringVar(&opts.format, "format", "markdown", "markdown or json")
	flags.BoolVar(&opts.check, "check", false, "exit 1 unless merge risk is low")
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
	if err := writeReport(output, opts.format, report); err != nil {
		return err
	}
	if opts.check && report.Risk != changerisk.Low {
		return fmt.Errorf("%w: %s", ErrReviewRequired, report.Risk)
	}
	return nil
}

func writeReport(output io.Writer, format string, report changerisk.Report) error {
	if format == "json" {
		return json.NewEncoder(output).Encode(report)
	}
	_, err := io.WriteString(output, changerisk.Markdown(report))
	return err
}

func parseOptions(args []string) (options, error) {
	opts := options{}
	flags := flagsFor(&opts)
	if err := flags.Parse(args); err != nil {
		return opts, err
	}
	if len(flags.Args()) != 0 {
		return opts, fmt.Errorf("risk accepts --format markdown or json and no positional paths")
	}
	return opts, opts.validate()
}

func (opts options) validate() error {
	if !slices.Contains([]string{"markdown", "json"}, opts.format) {
		return fmt.Errorf("risk --format must be markdown or json")
	}
	if !validConfigPath(opts.config) {
		return fmt.Errorf("risk config must be a clean repository-relative path")
	}
	if opts.check && opts.base == "" {
		return fmt.Errorf("risk --check requires an explicit --base revision")
	}
	return nil
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
	changed, err := changedPaths(ctx, root, opts.base, opts.head)
	if err != nil {
		return changerisk.Report{}, err
	}
	comparison := comparison{root: root, paths: changed.paths}
	base, err := comparison.snapshot(ctx, opts.base, opts.config)
	if err != nil {
		return changerisk.Report{}, err
	}
	head, err := comparison.snapshot(ctx, opts.head, "")
	if err != nil {
		return changerisk.Report{}, err
	}
	api := gorisk.CompareAPI(base.API, head.API)
	base.Source.AnalysisError = strings.TrimSpace(strings.Join([]string{base.Source.AnalysisError, changed.warning}, "\n"))
	report := changerisk.Evaluate(base.Source, head.Source, base.Policy, api)
	report.Policy = opts.config
	report.RepositoryURL = repositoryURL(ctx, root)
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
	Policy changerisk.Config
}

func (c comparison) snapshot(ctx context.Context, commit, configPath string) (repositorySnapshot, error) {
	result := repositorySnapshot{}
	err := withSnapshot(ctx, c.root, commit, func(dir string) error {
		policy, err := snapshotPolicy(dir, configPath)
		if err != nil {
			return err
		}
		loaded, err := populateSnapshot(ctx, dir, c.paths)
		result = repositorySnapshot{Source: loaded.Source, API: loaded.API, Policy: policy}
		return err
	})
	result.Source.Commit = commit
	return result, err
}

func snapshotPolicy(root, name string) (changerisk.Config, error) {
	if name == "" {
		return changerisk.Config{}, nil
	}
	policy, err := baseConfig(root, name)
	return policy.Risk, err
}

func populateSnapshot(ctx context.Context, root string, changed []string) (repositorySnapshot, error) {
	paths, message, err := gorisk.ExistingPaths(root, changed)
	if err != nil {
		return repositorySnapshot{}, err
	}
	source, err := gorisk.ExtractSource(ctx, root, paths)
	if err != nil {
		return repositorySnapshot{}, err
	}
	api := gorisk.ExtractAPI(ctx, root)
	items, dependencyErr := dependencies(ctx, root)
	dependencyMessage := ""
	if dependencyErr != nil {
		dependencyMessage = "Go dependency enumeration failed; no dependency verdict is available"
	}
	source.AnalysisError = strings.TrimSpace(strings.Join([]string{source.AnalysisError, message, embeddedScope(changed, items), embeddedDocumentationScope(ctx, root, changed)}, "\n"))
	source.Dependencies, source.DependencyError = items, dependencyMessage
	return repositorySnapshot{Source: source, API: api}, ctx.Err()
}
