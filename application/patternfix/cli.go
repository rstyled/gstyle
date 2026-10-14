// Package patternfix is the IO boundary for gstyle fix: static auto-fix
// for patterns miner candidates. Shared PDG source evidence bounds each batch.
// Dry-run by default (prints a diff); --apply writes files.
//
// Unified pipeline: runs the miner, gets candidates with FixSpecs, and
// applies each fixer's transform. No re-detection.
package patternfix

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/patterns"
)

const usage = `Usage: gstyle fix [OPTIONS] [DIRECTORIES OR GO FILES...]

Statically apply fixes for patterns miner candidates. No LLM, no tokens:
Source rewrites with conservative safety checks and whole-result type checks.
Runs fixes in dependency order. Reuses PDG facts until an edit changes the source.
Preview simulates the same phases as apply and shows the final combined diff.
Dry-run by default (shows a diff); --apply writes the files.

  --kind KIND   which fixes to apply: guard_clause, value_object, parameterize,
                trait_method, capability_set, enum_dispatch, generic_fn,
                anemic_model, primitive_obsession, declaration_order,
                offset_fallback, or all (default)
  --apply       write the fixes to disk (default: dry-run diff only)
  --changed     only fix candidates in functions touched by the git diff
  --base REF    git base for --changed (default: merge-base with main/master)

Exit 0: always, on success (even with no fixes). Exit 2: parse or IO failure.
`

type options struct {
	kind      string
	base      string
	apply     bool
	changed   bool
	preview   *previewRevision
	extractor *gopatterns.FixExtractor
	paths     []string
}

func (opts options) validate() error {
	valid := []string{"guard_clause", "offset_fallback", "value_object", "parameterize", "trait_method", "capability_set", "enum_dispatch", "generic_fn", "anemic_model", "primitive_obsession", "type_switch", "entity_identity", "missing_identity", "mutable_identity", "aggregate", "repository", "factory", "specification", "domain_service", "declaration_order", "all"}
	if !slices.Contains(valid, opts.kind) {
		return fmt.Errorf("kind must be one of %v", valid)
	}
	return nil
}

// fixPhases follows the domain dependency order. Each kind gets fresh
// evidence after a write; phases with no writes share one extraction.
var fixPhases = dependencyPhases()

// maxPhasedCycles bounds the fixpoint loop. In practice it terminates
// after 1-2 cycles; 3 is a safety net against oscillation.
const maxPhasedCycles = 3

// phasedState tracks the fixpoint loop state across phases.
type phasedState struct {
	revision   fixRevision
	totalFixed int
}

// phasedPhase bundles the inputs for running one phase.
type phasedPhase struct {
	phase    []patterns.CandidateKind
	phaseIdx int
	revision *fixRevision
}

// Run runs the miner, then applies fixes for candidates with FixSpecs.
func Run(ctx context.Context, args []string, output io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	return runPhased(ctx, opts, output)
}

// reportDryRun prints the dry-run summary.
func reportDryRun(opts options, output io.Writer, fixed int) error {
	if opts.apply {
		return nil
	}
	scope := ""
	if opts.changed {
		scope = " in changed functions"
	}
	_, err := fmt.Fprintf(output, "%d fixable candidate(s)%s (dry-run; use --apply to write)\n", fixed, scope)
	return err
}

// runPhased applies fixes in dependency phases, re-mining changed files
// between revisions. After a write, the next phase refreshes the complete
// input scope, including cross-file callers. Repeats until a full
// cycle produces no fixes or maxPhasedCycles is reached.
func runPhased(ctx context.Context, opts options, output io.Writer) error {
	if !opts.apply && opts.preview == nil {
		return runPhasedDryRun(ctx, opts, output)
	}
	opts.extractor = &gopatterns.FixExtractor{}
	st := &phasedState{revision: fixRevision{mine: mineCandidates}}
	for cycle := 0; cycle < maxPhasedCycles; cycle++ {
		fixed, err := runPhasedCycle(ctx, opts, output, st)
		if err != nil {
			return err
		}
		st.totalFixed += fixed
		if fixed == 0 {
			break
		}
	}
	return reportPhaseRun(opts, output, st.totalFixed)
}

func reportPhaseRun(opts options, output io.Writer, count int) error {
	if !opts.apply {
		return reportDryRun(opts, output, count)
	}
	_, err := fmt.Fprintf(output, "phased fix: %d candidate(s) fixed\n", count)
	return err
}

// runPhasedCycle runs one full pass through all phases. Returns the number
// of fixes applied. Reuse candidates until source changes; never narrow the
// input scope merely because edits touch a subset of files.
func runPhasedCycle(ctx context.Context, opts options, output io.Writer, st *phasedState) (int, error) {
	var cycleFixed int
	for phaseIdx, phase := range fixPhases {
		if err := ctx.Err(); err != nil {
			return cycleFixed, err
		}
		pp := phasedPhase{phase: phase, phaseIdx: phaseIdx, revision: &st.revision}
		fixed, written, err := runRevisionPhase(ctx, opts, output, pp)
		if err != nil {
			return cycleFixed, err
		}
		if len(written) > 0 {
			st.revision.invalidate()
			if opts.extractor != nil {
				opts.extractor.Invalidate(written)
			}
		}
		cycleFixed += fixed
	}
	return cycleFixed, nil
}

// runPhasedPhase mines one phase's kinds and applies the fixes.
// Returns fixes applied and files written.
func runRevisionPhase(ctx context.Context, opts options, output io.Writer, pp phasedPhase) (int, []string, error) {
	candidates, err := pp.revision.load(ctx, opts)
	if err != nil {
		return 0, nil, err
	}
	specs, err := filterKindSpecs(candidates, phaseKindSet(pp.phase), opts)
	if err != nil {
		return 0, nil, err
	}
	if len(specs) == 0 {
		return 0, nil, nil
	}
	fixed, written, err := applyAndTrack(ctx, specs, opts, output)
	if err != nil {
		return fixed, written, err
	}
	_, err = fmt.Fprintf(output, "phase %d (%s): fixed %d candidate(s)\n",
		pp.phaseIdx+1, phaseNames(pp.phase), fixed)
	return fixed, written, err
}

// phaseIndex returns the phase containing kind, or -1.
func phaseIndex(kind patterns.CandidateKind) int {
	for i, phase := range fixPhases {
		if phaseHas(phase, kind) {
			return i
		}
	}
	return -1
}

// phaseHas reports whether a phase includes a kind.
func phaseHas(phase []patterns.CandidateKind, kind patterns.CandidateKind) bool {
	for _, k := range phase {
		if k == kind {
			return true
		}
	}
	return false
}

// phaseNames joins kind names for display.
func phaseNames(phase []patterns.CandidateKind) string {
	names := make([]string, len(phase))
	for i, k := range phase {
		names[i] = string(k)
	}
	return strings.Join(names, ",")
}

// filterKindSpecs keeps FixSpecs for kinds in the set, applying --changed
// filtering when requested.
func filterKindSpecs(candidates []patterns.Candidate, kindSet map[patterns.CandidateKind]bool, opts options) ([]*patterns.FixSpec, error) {
	var selected []patterns.Candidate
	for _, candidate := range candidates {
		if kindSet[candidate.Kind] {
			selected = append(selected, candidate)
		}
	}
	if opts.changed {
		return onlyChangedSpecs(selected, opts)
	}
	return candidateSpecs(selected), nil
}

// applyAndTrack applies specs and returns the files that were written.
func applyAndTrack(ctx context.Context, specs []*patterns.FixSpec, opts options, output io.Writer) (int, []string, error) {
	plan, err := buildFixPlan(ctx, specs)
	if err != nil {
		return 0, nil, err
	}
	if err := reportSkippedPlans(plan, output); err != nil {
		return 0, nil, err
	}
	if opts.preview != nil {
		return opts.preview.advance(ctx, plan)
	}
	return executeFixPlan(ctx, plan, opts, output)
}

// collectFixSpecs runs the miner and returns FixSpecs filtered by kind,
// narrowing to diff-touched functions when --changed is set.
func collectFixSpecs(ctx context.Context, opts options) ([]*patterns.FixSpec, error) {
	candidates, err := mineCandidates(ctx, opts)
	if err != nil {
		return nil, err
	}
	if opts.changed {
		return onlyChangedSpecs(candidates, opts)
	}
	return candidateSpecs(candidates), nil
}

// mineCandidates runs the miner and keeps candidates with a FixSpec of the
// requested kind.
func mineCandidates(ctx context.Context, opts options) ([]patterns.Candidate, error) {
	ext, err := extractFixRevision(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}
	facts, anemicHits, missingHits, serviceHits := toFacts(ext)
	report := patterns.PatternsReport{Candidates: patterns.RunFixCandidates(facts, patterns.Options{DeclarationOrders: ext.DeclarationOrders, OffsetFallbacks: ext.OffsetFallbacks, AnemicModels: anemicHits, MissingIdentities: missingHits, DomainServices: serviceHits}, patterns.CandidateKind(opts.kind))}
	gopatterns.VerifyValueGroups(&report, ext)
	if err := gopatterns.VerifyParameterizations(&report, ext); err != nil {
		return nil, fmt.Errorf("verify parameterization: %w", err)
	}
	candidates := fixableCandidates(report, opts.kind)
	for i := range candidates {
		candidates[i].FixSpec.Context = ext.Sources
		for _, site := range candidates[i].Sites {
			candidates[i].FixSpec.Targets = append(candidates[i].FixSpec.Targets, site.ID)
		}
	}
	return candidates, nil
}

func fixableCandidates(report patterns.PatternsReport, kind string) []patterns.Candidate {
	var candidates []patterns.Candidate
	for _, c := range report.Candidates {
		if c.FixSpec == nil {
			continue
		}
		if kind != "all" && string(c.Kind) != kind {
			continue
		}
		candidates = append(candidates, c)
	}
	return candidates
}

// candidateSpecs extracts FixSpecs from filtered candidates.
func candidateSpecs(candidates []patterns.Candidate) []*patterns.FixSpec {
	specs := make([]*patterns.FixSpec, 0, len(candidates))
	for _, c := range candidates {
		specs = append(specs, c.FixSpec)
	}
	return specs
}

// onlyChangedSpecs keeps FixSpecs whose candidate site the git diff touches.
// The miner runs whole so clustering stays complete; only the fixes that
// land are narrowed. Filtering uses the candidate's site range (the
// containing function) rather than the FixSpec's precise edit location.
func onlyChangedSpecs(candidates []patterns.Candidate, opts options) ([]*patterns.FixSpec, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	diff, err := fixChangedDiff(cwd, opts)
	if err != nil {
		return nil, err
	}
	var kept []*patterns.FixSpec
	for _, c := range candidates {
		if len(c.Sites) == 0 {
			continue
		}
		site := c.Sites[0]
		if diff.Touched(site.Path, site.Line, site.EndLine) {
			kept = append(kept, c.FixSpec)
		}
	}
	return kept, nil
}

// groupByFile groups FixSpecs by their file.
func groupByFile(specs []*patterns.FixSpec) map[string][]*patterns.FixSpec {
	byFile := make(map[string][]*patterns.FixSpec)
	for _, s := range specs {
		byFile[s.File] = append(byFile[s.File], s)
	}
	return byFile
}

// sortSpecsForApply orders specs by pattern dependency rank, then bottom-up
// by line within a kind.
func sortSpecsForApply(specs []*patterns.FixSpec) []*patterns.FixSpec {
	ordered := make([]*patterns.FixSpec, len(specs))
	copy(ordered, specs)
	slices.SortFunc(ordered, func(a, b *patterns.FixSpec) int {
		if ra, rb := patterns.FixKindRank(a.Kind), patterns.FixKindRank(b.Kind); ra != rb {
			return ra - rb
		}
		return b.Line - a.Line
	})
	return ordered
}

// showDiff prints a unified diff for dry-run.
func showDiff(path string, src, out []byte, n int, output io.Writer) (int, error) {
	diff, err := unifiedDiff(path, src, out)
	if err != nil {
		return 0, err
	}
	_, err = io.WriteString(output, diff)
	return n, err
}

// unifiedDiff reports IO failures; diff's exit code 1 means a valid difference.
func unifiedDiff(path string, old, new []byte) (string, error) {
	directory, err := os.MkdirTemp("", "gstyle-fix-diff-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(directory)
	oldPath, newPath := filepath.Join(directory, "old.go"), filepath.Join(directory, "new.go")
	if err := os.WriteFile(oldPath, old, 0600); err != nil {
		return "", err
	}
	if err := os.WriteFile(newPath, new, 0600); err != nil {
		return "", err
	}
	cmd := exec.Command("diff", "-u", "--label", "a/"+path, "--label", "b/"+path, oldPath, newPath)
	return diffCommandOutput(cmd)
}

func diffCommandOutput(cmd *exec.Cmd) (string, error) {
	output, err := cmd.Output()
	if err == nil {
		return string(output), nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return string(output), nil
	}
	return "", fmt.Errorf("create fix diff: %w", err)
}

func parseOptions(args []string) (options, error) {
	flags := flag.NewFlagSet("fix", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	kind := flags.String("kind", "all", "which fixes to apply")
	apply := flags.Bool("apply", false, "write fixes to disk")
	changed := flags.Bool("changed", false, "only fix candidates in diff-touched functions")
	base := flags.String("base", "", "git base for --changed")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	result := options{kind: *kind, apply: *apply, changed: *changed, base: *base, paths: flags.Args()}
	return result, result.validate()
}

// toFacts converts extraction to FuncFacts (copied from patterncheck).
func toFacts(ext *gopatterns.Extraction) ([]*patterns.FuncFacts, []patterns.AnemicModelHit, []patterns.MissingIdentityHit, []patterns.DomainServiceHit) {
	var facts []*patterns.FuncFacts
	for _, fp := range ext.Funcs {
		fp := fp
		facts = append(facts, &patterns.FuncFacts{
			ID:                  fp.Name,
			Name:                fp.Name,
			Path:                fp.Path,
			Line:                fp.Line,
			EndLine:             fp.EndLine,
			Pdg:                 &fp.Pdg,
			SigKey:              gopatterns.SigKeyOf(fp),
			SelfTy:              fp.SelfTy,
			GuardClauses:        guardHits(fp.GuardClauses),
			EnumDispatches:      dispatchHits(fp.EnumDispatches),
			TypeSwitches:        typeSwitchHits(fp.TypeSwitches),
			EntityIdentities:    fp.EntityIdentities,
			MutableIdentities:   fp.MutableIdentities,
			AggregateMods:       fp.AggregateMods,
			DbCalls:             fp.DbCalls,
			FactoryLits:         fp.FactoryLits,
			SpecRules:           fp.SpecRules,
			NilErrHits:          fp.NilErrHits,
			ForceTypeAssertHits: fp.ForceTypeAssertHits,
			TypedNilHits:        fp.TypedNilHits,
			SuppressedKinds:     fp.SuppressedKinds,
			Params:              paramInfos(fp.Params),
		})
	}
	return facts, ext.AnemicModels, ext.MissingIdentities, ext.DomainServices
}

func guardHits(hits []gopatterns.GuardClauseHit) []patterns.GuardClauseHit {
	var out []patterns.GuardClauseHit
	for _, h := range hits {
		out = append(out, patterns.GuardClauseHit{Line: h.Line, BodyStmts: h.BodyStmts})
	}
	return out
}

func dispatchHits(hits []gopatterns.EnumDispatchHit) []patterns.EnumDispatchHit {
	var out []patterns.EnumDispatchHit
	for _, h := range hits {
		out = append(out, patterns.EnumDispatchHit{Line: h.Line, NumCases: h.NumCases})
	}
	return out
}

func typeSwitchHits(hits []gopatterns.TypeSwitchHit) []patterns.TypeSwitchHit {
	var out []patterns.TypeSwitchHit
	for _, h := range hits {
		out = append(out, patterns.TypeSwitchHit{
			Line: h.Line, Bound: h.Bound, Expr: h.Expr,
			Method: h.Method, Types: h.Types, Args: h.Args,
		})
	}
	return out
}

func paramInfos(params []patterns.ParamInfo) []patterns.ParamInfo {
	var out []patterns.ParamInfo
	for _, p := range params {
		out = append(out, patterns.ParamInfo{Name: p.Name, Type: p.Type})
	}
	return out
}

func phaseKindSet(kinds []patterns.CandidateKind) map[patterns.CandidateKind]bool {
	set := make(map[patterns.CandidateKind]bool, len(kinds))
	for _, kind := range kinds {
		set[kind] = true
	}
	return set
}
