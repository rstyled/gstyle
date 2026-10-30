package patterns

import "fmt"

// Slicing-based pattern kinds: barrier_slice, thin_slice, and chop.
//
// These are detection-only attention guides, not bug finders. They surface
// the dependence structure of a function so an LLM or human can see what
// affects a return value, how data flows, and which statements lie on the
// input-to-output path. Each fires only when the slice/chop is large enough
// to be non-obvious; trivial functions are skipped.

// Slice thresholds: the minimum slice/chop size (in PDG nodes) for a
// finding. Below this, the dependence structure is obvious from reading
// the function. These are heuristics, not paper thresholds.
const (
	minBarrierSliceNodes = 20
	minThinSliceNodes    = 10
	minChopNodes         = 20
)

// BarrierSliceFinding is a function whose return-value dependence cone
// (excluding error handling) is large enough to be worth surfacing.
type BarrierSliceFinding struct {
	FuncID    string
	SliceSize int
	SeedLine  int
	SeedNode  string
}

// ThinSliceFinding is a function with a long producer chain feeding a
// return value.
type ThinSliceFinding struct {
	FuncID    string
	SliceSize int
	SeedLine  int
	SeedNode  string
}

// ChopFinding is a function where the input-to-output dependence path
// covers many statements.
type ChopFinding struct {
	FuncID   string
	ChopSize int
}

// FindBarrierSlices computes the barrier slice from each return statement,
// using Try nodes (the `if err != nil { return err }` idiom) as barriers.
// This shows the happy-path dependencies of each return, excluding error
// handling. Flags functions where any return's slice is large.
func FindBarrierSlices(facts []*MiningFacts) []BarrierSliceFinding {
	var out []BarrierSliceFinding
	for _, original := range facts {
		f := semanticFact(original)
		if f == nil || f.Pdg == nil {
			continue
		}
		out = append(out, barrierSlicesForFunc(f)...)
	}
	return out
}

// barrierSlicesForFunc computes the barrier slice from each return statement
// in one function, flagging returns whose slice (excluding error handling)
// is large.
func barrierSlicesForFunc(f *MiningFacts) []BarrierSliceFinding {
	var out []BarrierSliceFinding
	barriers := make([]bool, len(f.Pdg.Nodes))
	for i, n := range f.Pdg.Nodes {
		barriers[i] = n.Kind == Try
	}
	traversal := newReturnTraversal(f.Pdg, false)
	for i, n := range f.Pdg.Nodes {
		if n.Kind != Return {
			continue
		}
		size := traversal.size(i, barriers)
		if size >= minBarrierSliceNodes {
			out = append(out, BarrierSliceFinding{
				FuncID:    f.ID,
				SliceSize: size,
				SeedLine:  n.Line,
				SeedNode:  canonicalNodeID(f.Canonical, i),
			})
		}
	}
	return out
}

// FindThinSlices computes the thin (producer-only) slice from each return
// statement. Flags functions where any return's producer chain is long.
func FindThinSlices(facts []*MiningFacts) []ThinSliceFinding {
	var out []ThinSliceFinding
	for _, original := range facts {
		f := semanticFact(original)
		if f == nil || f.Pdg == nil {
			continue
		}
		out = append(out, thinSlicesForFunc(f)...)
	}
	return out
}

// thinSlicesForFunc computes the thin (producer-only) slice from each return
// statement in one function, flagging returns with a long producer chain.
func thinSlicesForFunc(f *MiningFacts) []ThinSliceFinding {
	var out []ThinSliceFinding
	traversal := newReturnTraversal(f.Pdg, true)
	blocked := make([]bool, len(f.Pdg.Nodes))
	for i, n := range f.Pdg.Nodes {
		if n.Kind != Return {
			continue
		}
		size := traversal.size(i, blocked)
		if size >= minThinSliceNodes {
			out = append(out, ThinSliceFinding{
				FuncID:    f.ID,
				SliceSize: size,
				SeedLine:  n.Line,
				SeedNode:  canonicalNodeID(f.Canonical, i),
			})
		}
	}
	return out
}

// FindChops computes the chop from function parameters to return statements
// (the statements on the input-to-output dependence path). Flags functions
// where the chop is large.
func FindChops(facts []*MiningFacts) []ChopFinding {
	var out []ChopFinding
	for _, original := range facts {
		f := semanticFact(original)
		if f == nil || f.Pdg == nil {
			continue
		}
		out = append(out, chopsForFunc(f)...)
	}
	return out
}

// paramToReturnChop unions all param-to-return chops. Distributing the
// union over intersection gives reachability from any parameter intersected
// with reachability to any return, so two traversals replace all pair scans.
func paramToReturnChop(pdg *MiningGraph) map[int]bool {
	params, returns := paramReturnIndices(pdg)
	union := map[int]bool{}
	if len(params) == 0 || len(returns) == 0 {
		return union
	}
	forward, backward := dependenceAdjacency(pdg)
	fwd := reachFrom(forward, params)
	bwd := reachFrom(backward, returns)
	for n, reached := range fwd {
		if reached && bwd[n] {
			union[n] = true
		}
	}
	return union
}

func dependenceAdjacency(pdg *MiningGraph) ([][]int, [][]int) {
	forward := make([][]int, len(pdg.Nodes))
	backward := make([][]int, len(pdg.Nodes))
	for _, e := range pdg.Edges {
		if !e.abstractionEdge() {
			continue
		}
		forward[e.From] = append(forward[e.From], e.To)
		backward[e.To] = append(backward[e.To], e.From)
	}
	return forward, backward
}

func reachFrom(adjacency [][]int, seeds []int) []bool {
	seen := make([]bool, len(adjacency))
	queue := make([]int, 0, len(adjacency))
	for _, seed := range seeds {
		if !seen[seed] {
			seen[seed] = true
			queue = append(queue, seed)
		}
	}
	for head := 0; head < len(queue); head++ {
		for _, next := range adjacency[queue[head]] {
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return seen
}

// paramReturnIndices collects the node indices of parameters and returns.
func paramReturnIndices(pdg *MiningGraph) (params, returns []int) {
	for i, n := range pdg.Nodes {
		switch n.Kind {
		case Param:
			params = append(params, i)
		case Return, NodeKind("formal_output"):
			returns = append(returns, i)
		}
	}
	return params, returns
}

// BarrierSliceCandidates converts barrier slice findings to candidates.
// Detection-only: FixSpec is nil.
func BarrierSliceCandidates(findings []BarrierSliceFinding, facts []*MiningFacts) []Candidate {
	factByID := buildFactMap(facts)
	var out []Candidate
	for _, fl := range findings {
		f := factByID[fl.FuncID]
		if f == nil {
			continue
		}
		out = append(out, Candidate{
			Kind:       BarrierSliceKind,
			Analysis:   analysisEvidence(f, BarrierSliceKind, fl.SeedNode),
			ScoreMilli: sliceScore(fl.SliceSize),
			Observation: fmt.Sprintf(
				"return at line %d has recorded dependencies on %d operations (excluding error handling)",
				fl.SeedLine, fl.SliceSize,
			),
			Inference:        "a large happy-path dependence cone is hard to reason about; consider simplifying",
			PossibleRefactor: "extract helpers for parts of the computation feeding this return",
			Sites: []Site{
				{Path: f.Path, Line: fl.SeedLine, Name: f.Name},
			},
		})
	}
	return out
}

// ThinSliceCandidates converts thin slice findings to candidates.
// Detection-only: FixSpec is nil.
func ThinSliceCandidates(findings []ThinSliceFinding, facts []*MiningFacts) []Candidate {
	factByID := buildFactMap(facts)
	var out []Candidate
	for _, fl := range findings {
		f := factByID[fl.FuncID]
		if f == nil {
			continue
		}
		out = append(out, Candidate{
			Kind:       ThinSliceKind,
			Analysis:   analysisEvidence(f, ThinSliceKind, fl.SeedNode),
			ScoreMilli: sliceScore(fl.SliceSize),
			Observation: fmt.Sprintf(
				"return at line %d has recorded data dependencies on %d graph operations",
				fl.SeedLine, fl.SliceSize,
			),
			Inference:        "these operations contribute to the return through recorded producer dependencies; the set may branch",
			PossibleRefactor: "inspect the contributing values when this return is difficult to understand",
			CounterEvidence:  []string{"graph-operation count alone does not establish a defect or a need to refactor"},
			Sites: []Site{
				{Path: f.Path, Line: fl.SeedLine, Name: f.Name},
			},
		})
	}
	return out
}

// ChopCandidates converts chop findings to candidates.
// Detection-only: FixSpec is nil.
func ChopCandidates(findings []ChopFinding, facts []*MiningFacts) []Candidate {
	factByID := buildFactMap(facts)
	var out []Candidate
	for _, fl := range findings {
		f := factByID[fl.FuncID]
		if f == nil {
			continue
		}
		out = append(out, Candidate{
			Kind:       ChopKind,
			Analysis:   analysisEvidence(f, ChopKind),
			ScoreMilli: sliceScore(fl.ChopSize),
			Observation: fmt.Sprintf(
				"%d operations lie on the input-to-output path",
				fl.ChopSize,
			),
			Inference:        "when most of the function is on the input-output path, the logic is tightly coupled",
			PossibleRefactor: "look for independent computations that can be extracted",
			Sites: []Site{
				{Path: f.Path, Line: f.Line, Name: f.Name},
			},
		})
	}
	return out
}

// sliceScore maps slice size to a milli-score. Larger slices score higher
// (more to pay attention to), capped at 800 — these are guides, not bugs.
func sliceScore(size int) uint32 {
	s := uint32(400 + size*10)
	if s > 800 {
		s = 800
	}
	return s
}

func chopsForFunc(f *MiningFacts) []ChopFinding {
	chop := paramToReturnChop(f.Pdg)
	if len(chop) < minChopNodes {
		return nil
	}
	return []ChopFinding{{FuncID: f.ID, ChopSize: len(chop)}}
}
