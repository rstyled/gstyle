package ccgraph

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"unicode/utf8"
)

// StageFunction is an imported, already projected matching input. Known
// interface/reference facts are required even for isolated graphs. Native
// producers need a separate semantic validator before constructing this view.
type StageFunction struct {
	Key, Name       string
	Graph           Graph
	References      *uint64
	Inputs, Outputs *[]Parameter
	UsesNativeWidth bool
	PointerWidth    int
}
type preparedFunction struct {
	StageFunction
	features [9]uint64
	strings  Strings
	diameter int
}
type FunctionTrace struct {
	Key      string
	Features [9]uint64
	Strings  Strings
	Diameter int
}
type Diagnostic struct{ Function, Code, Message string }
type Run struct {
	Profile, ContractCommit, InputScope string
	Config                              Config
	Complete                            bool
	Pairs                               int
	Ineligible                          []Diagnostic
	Functions                           []FunctionTrace
}
type Gate string

const (
	NotEvaluated Gate = "not_evaluated"
	Passed       Gate = "passed"
	Rejected     Gate = "rejected"
)

type Pair struct {
	Left, Right                               string
	Nonempty, Size, Numerical, Strings, Final Gate
	SizeRatio                                 *big.Rat
	NumericalScore                            *Cosine
	StringScore                               *StringScore
	Kernel                                    *Kernel
	Accepted                                  bool
	Diagnostic                                string
}

func prepareFunction(f StageFunction) (preparedFunction, error) {
	if _, err := f.Graph.validate(); err != nil {
		return preparedFunction{}, err
	}
	if f.References == nil {
		return preparedFunction{}, fmt.Errorf("unknown reference count")
	}
	if f.Inputs == nil || f.Outputs == nil {
		return preparedFunction{}, fmt.Errorf("unknown interface")
	}
	if !validText(f.Name) {
		return preparedFunction{}, fmt.Errorf("unknown or invalid name")
	}
	return prepareInterface(f)
}

func validText(text string) bool { return text != "" && utf8.ValidString(text) }

func prepareInterface(f StageFunction) (preparedFunction, error) {
	in, err := Interface(*f.Inputs)
	if err != nil {
		return preparedFunction{}, err
	}
	out, err := Interface(*f.Outputs)
	if err != nil {
		return preparedFunction{}, err
	}
	if nativeInterface(f) && f.PointerWidth <= 0 {
		return preparedFunction{}, fmt.Errorf("unknown native target width")
	}
	return preparedFunction{StageFunction: f, features: graphFeatures(f.Graph, *f.References), strings: Strings{out, in, f.Name}}, nil
}

func nativeInterface(f StageFunction) bool {
	if f.UsesNativeWidth {
		return true
	}
	params := append(slices.Clone(*f.Inputs), (*f.Outputs)...)
	for _, p := range params {
		if p.Type == "isize" || p.Type == "usize" {
			return true
		}
	}
	return false
}

func graphFeatures(g Graph, references uint64) [9]uint64 {
	var counts [9]uint64
	for _, n := range g.Nodes {
		counts[labelCoordinates[n.Label]]++
	}
	for _, e := range g.Edges {
		counts[familyCoordinates[e.Family]]++
	}
	counts[8] = references
	return counts
}

func prepareFunctions(inputs []StageFunction) ([]preparedFunction, []Diagnostic, error) {
	ordered := slices.Clone(inputs)
	slices.SortFunc(ordered, func(a, b StageFunction) int { return slices.Compare([]rune(a.Key), []rune(b.Key)) })
	var out []preparedFunction
	var diagnostics []Diagnostic
	previous := ""
	for _, f := range ordered {
		if f.Key == "" || f.Key == previous || !utf8.ValidString(f.Key) {
			return nil, nil, fmt.Errorf("invalid or duplicate function key %q", f.Key)
		}
		previous = f.Key
		p, err := prepareFunction(f)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{f.Key, "ineligible-stage-input", err.Error()})
			continue
		}
		out = append(out, p)
	}
	return out, diagnostics, nil
}

// Evaluate streams every eligible unordered pair. Sink/cancellation failures
// return an incomplete run; no grouping, retrieval or display limits intervene.
func Evaluate(ctx context.Context, inputs []StageFunction, config Config, emit func(Pair) error) (Run, error) {
	run := Run{Profile: Profile, ContractCommit: ContractCommit, InputScope: "projected_stage_inputs", Config: config}
	if err := config.Validate(); err != nil {
		return run, err
	}
	if err := ctx.Err(); err != nil {
		return run, err
	}
	if emit == nil {
		return run, fmt.Errorf("pair sink is required")
	}
	functions, diagnostics, err := prepareFunctions(inputs)
	if err != nil {
		return run, err
	}
	run.Ineligible = diagnostics
	if err := prepareDiameters(ctx, functions, &run); err != nil {
		return run, err
	}
	return enumerate(ctx, functions, run, emit)
}

func prepareDiameters(ctx context.Context, functions []preparedFunction, run *Run) error {
	for i := range functions {
		d, err := Diameter(ctx, functions[i].Graph)
		if err != nil {
			return err
		}
		functions[i].diameter = d
		run.Functions = append(run.Functions, FunctionTrace{functions[i].Key, functions[i].features, functions[i].strings, d})
	}
	return nil
}

func enumerate(ctx context.Context, functions []preparedFunction, run Run, emit func(Pair) error) (Run, error) {
	for i, a := range functions {
		for _, b := range functions[i+1:] {
			if err := ctx.Err(); err != nil {
				return run, err
			}
			pair, err := comparePair(ctx, a, b, run.Config)
			if err != nil {
				return run, err
			}
			if err := emit(pair); err != nil {
				return run, fmt.Errorf("pair sink: %w", err)
			}
			run.Pairs++
		}
	}
	run.Complete = true
	return run, nil
}

func newPair(a, b preparedFunction) Pair {
	return Pair{Left: a.Key, Right: b.Key, Nonempty: NotEvaluated, Size: NotEvaluated, Numerical: NotEvaluated, Strings: NotEvaluated, Final: NotEvaluated}
}

func comparePair(ctx context.Context, a, b preparedFunction, c Config) (Pair, error) {
	p := newPair(a, b)
	if incompatibleWidths(a, b) {
		p.Diagnostic = "incompatible native target widths"
		return p, nil
	}
	if !sizeGates(&p, len(a.Graph.Nodes), len(b.Graph.Nodes), c.Size) {
		return p, nil
	}
	if !numericalGate(&p, a.features, b.features, c.Numerical) {
		return p, nil
	}
	return (pairComparison{a, b, c}).remainingGates(ctx, p)
}

func incompatibleWidths(a, b preparedFunction) bool {
	return (nativeInterface(a.StageFunction) || nativeInterface(b.StageFunction)) && a.PointerWidth != b.PointerWidth
}

func sizeGates(p *Pair, a, b int, threshold Decimal) bool {
	p.Nonempty = Rejected
	if min(a, b) == 0 {
		return false
	}
	p.Nonempty = Passed
	p.SizeRatio = big.NewRat(int64(min(a, b)), int64(max(a, b)))
	p.Size = gate(atLeast(p.SizeRatio, threshold))
	return p.Size == Passed
}

func numericalGate(p *Pair, a, b [9]uint64, threshold Decimal) bool {
	score := Numerical(a, b)
	p.NumericalScore = &score
	p.Numerical = gate(score.AtLeast(threshold))
	return p.Numerical == Passed
}

type pairComparison struct {
	a, b   preparedFunction
	config Config
}

func (comparison pairComparison) remainingGates(ctx context.Context, p Pair) (Pair, error) {
	a, b, c := comparison.a, comparison.b, comparison.config
	score, err := CompareStrings(a.strings, b.strings)
	if err != nil {
		return p, err
	}
	p.StringScore = &score
	p.Strings = gate(atLeast(score.Mean, c.Strings))
	if p.Strings == Rejected {
		return p, nil
	}
	kernel, err := (kernelPair{a.Graph, b.Graph, a.diameter, b.diameter}).compare(ctx)
	if err != nil {
		return p, err
	}
	p.Kernel = &kernel
	p.Accepted = kernelAccepted(kernel, c)
	p.Final = gate(p.Accepted)
	return p, nil
}

func kernelAccepted(k Kernel, c Config) bool {
	if c.Scoring == CrossCosine {
		return k.Cosine.AtLeast(c.Final)
	}
	return atLeast(k.Overlap, c.Final)
}

func gate(passed bool) Gate {
	if passed {
		return Passed
	}
	return Rejected
}
