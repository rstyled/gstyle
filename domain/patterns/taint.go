package patterns

import (
	"fmt"
	"strings"
)

// Taint follows typed sources through per-function PDG value dependencies to
// modeled dangerous arguments. Context is part of each sink model: query
// parameters and ordinary argv data are not query text or shell code. A path
// is advisory; sanitization, feasibility and incomplete graph evidence need review.

type taintPropagation struct {
	tainted map[int]bool
}

func (p taintPropagation) spread(from int, tos []int) bool {
	if !p.tainted[from] {
		return false
	}
	spread := false
	for _, to := range tos {
		if p.tainted[to] {
			continue
		}
		p.tainted[to] = true
		spread = true
	}
	return spread
}

// TaintFlow is one tainted-data-to-sink flow.
type TaintFlow struct {
	SourceNode, SinkNode string
	FuncID               string
	SourceFuncID         string
	// SourceLine is where the tainted data enters.
	SourceLine int
	// SourceCall is the source callee, e.g. "(*url.Values).Get".
	SourceCall string
	// SinkLine is where it reaches the dangerous sink.
	SinkLine int
	// SinkCall is the sink callee, e.g. "(*sql.DB).Exec".
	SinkCall string
	// Risk is the human-readable risk, e.g. "SQL injection".
	Risk string
}

// isTaintSource reports whether n is a taint source call.
func (n PdgNode) isTaintSource() bool {
	if n.Kind != Call {
		return false
	}
	for _, s := range taintSources {
		if n.CalleeID == s {
			return true
		}
	}
	return false
}

// matchTaintSink returns the sink rule for n, or nil.
func (n PdgNode) matchTaintSink() *taintSink {
	if n.Kind != Call {
		return nil
	}
	for i := range taintSinks {
		if n.CalleeID == taintSinks[i].callee {
			return &taintSinks[i]
		}
	}
	return nil
}

// No generic sanitizer clears all risks. Parameterized operations avoid
// dangerous argument positions; validation requires its own context model.
func propagateTaint(pdg *MiningGraph) map[int]bool {
	tainted := seedTaintSources(pdg)
	succ := dataSuccessors(pdg)
	for changed := true; changed; {
		changed = propagateTaintStep(succ, tainted)
	}
	return tainted
}

// seedTaintSources marks source call nodes as tainted.
func seedTaintSources(pdg *MiningGraph) map[int]bool {
	tainted := map[int]bool{}
	for i, n := range pdg.Nodes {
		if n.isTaintSource() {
			tainted[i] = true
		}
	}
	return tainted
}

// dataSuccessors builds adjacency lists for data edges only.
func dataSuccessors(pdg *MiningGraph) [][]int {
	succ := make([][]int, len(pdg.Nodes))
	for _, e := range pdg.Edges {
		if e.Kind == Data {
			succ[e.From] = append(succ[e.From], e.To)
		}
	}
	return succ
}

func propagateTaintStep(succ [][]int, tainted map[int]bool) bool {
	changed := false
	propagation := taintPropagation{tainted: tainted}
	for from, tos := range succ {
		if propagation.spread(from, tos) {
			changed = true
		}
	}
	return changed
}

// sinkArgTainted reports whether a dangerous argument of sink node idx
// receives tainted data.
func sinkArgTainted(pdg *MiningGraph, idx int, sink *taintSink, tainted map[int]bool) bool {
	for _, e := range pdg.Edges {
		if taintedSinkEdge(e, idx, sink, tainted) {
			return true
		}
	}
	return false
}

// taintedSinkEdge reports whether e is a tainted data edge into a
// dangerous argument of the sink.
func taintedSinkEdge(e PdgEdge, idx int, sink *taintSink, tainted map[int]bool) bool {
	if e.Kind != Data || e.To != idx || !tainted[e.From] {
		return false
	}
	return sink.dangerousArgs == nil || intInSlice(e.ArgPos, sink.dangerousArgs)
}

// intInSlice reports whether v is in s.
func intInSlice(v int, s []int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// findTaintSource traces back from a tainted node to a source call,
// returning its line and callee. Best-effort: first source found.
func findTaintSource(pdg *MiningGraph, tainted map[int]bool, from int) (line int, callee string) {
	node := findTaintSourceNode(pdg, tainted, from)
	if node < 0 {
		return 0, ""
	}
	return pdg.Nodes[node].Line, pdg.Nodes[node].CalleeID
}

func findTaintSourceNode(pdg *MiningGraph, tainted map[int]bool, from int) int {
	visited := map[int]bool{from: true}
	queue := taintedPredecessors(pdg, tainted, from, visited)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur] {
			continue
		}
		visited[cur] = true
		if pdg.Nodes[cur].isTaintSource() {
			return cur
		}
		queue = append(queue, taintedPredecessors(pdg, tainted, cur, visited)...)
	}
	return -1
}

// taintedPredecessors returns tainted data-predecessors of node, excluding
// already-visited ones.
func taintedPredecessors(pdg *MiningGraph, tainted map[int]bool, to int, visited map[int]bool) []int {
	var out []int
	for _, e := range pdg.Edges {
		if e.Kind == Data && e.To == to && tainted[e.From] && !visited[e.From] {
			out = append(out, e.From)
		}
	}
	return out
}

// FindTaintFlows finds tainted-data-to-sink flows in each function.
func FindTaintFlows(facts []*MiningFacts) []TaintFlow {
	var out []TaintFlow
	for _, original := range facts {
		f := semanticFact(original)
		if f == nil || f.Pdg == nil {
			continue
		}
		out = append(out, taintFlowsForFunction(f)...)
	}
	return out
}

func taintFlowsForFunction(f *MiningFacts) []TaintFlow {
	copy := *f
	copy.Pdg = taintView(f)
	f = &copy
	var out []TaintFlow
	tainted := propagateTaint(f.Pdg)
	for i, n := range f.Pdg.Nodes {
		sink := n.matchTaintSink()
		if sink == nil {
			continue
		}
		if !sinkArgTainted(f.Pdg, i, commandSink(f, i, sink), tainted) {
			continue
		}
		sourceNode := findTaintSourceNode(f.Pdg, tainted, i)
		srcLine, srcCall := taintSourceLocation(f.Pdg, sourceNode)
		out = append(out, TaintFlow{
			FuncID:     f.ID,
			SourceNode: canonicalNodeID(f.Canonical, sourceNode), SinkNode: canonicalNodeID(f.Canonical, i),
			SourceLine: srcLine,
			SourceCall: srcCall,
			SinkLine:   n.Line,
			SinkCall:   n.CalleeID,
			Risk:       sink.desc,
		})
	}
	return out
}

// TaintFlowCandidates converts taint flows to pattern candidates.
// Detection-only: the fix depends on context, so FixSpec is nil.
func TaintFlowCandidates(flows []TaintFlow, facts []*MiningFacts) []Candidate {
	factByID := map[string]*MiningFacts{}
	for _, f := range facts {
		factByID[f.ID] = f
	}
	var out []Candidate
	for _, fl := range flows {
		f := factByID[fl.FuncID]
		if f == nil {
			continue
		}
		out = append(out, Candidate{
			Kind:       TaintFlowKind,
			Security:   securityForRisk(fl.Risk),
			Analysis:   taintEvidence(f, factByID[fl.SourceFuncID], fl),
			ScoreMilli: 900, // Prioritize recorded source-to-sink paths for review.
			Observation: fmt.Sprintf(
				"tainted data from `%s` reaches `%s` (%s)",
				shortCallee(fl.SourceCall), shortCallee(fl.SinkCall), fl.Risk,
			),
			Inference: fmt.Sprintf(
				"recorded dependencies connect an input source to a sink; review sanitization and analysis coverage",
			),
			PossibleRefactor: "sanitize the input or parameterize the operation",
			Sites: []Site{
				taintSourceSite(f, factByID[fl.SourceFuncID], fl.SourceLine),
				{Path: f.Path, Line: fl.SinkLine, Name: f.Name},
			},
		})
	}
	return out
}

// shortCallee trims a CalleeID to its last two dot-separated parts.
func shortCallee(id string) string {
	parts := strings.Split(id, ".")
	if len(parts) <= 2 {
		return id
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

func taintSourceLocation(graph *MiningGraph, index int) (int, string) {
	if index < 0 {
		return 0, ""
	}
	return graph.Nodes[index].Line, graph.Nodes[index].CalleeID
}

func taintSourceSite(sink, source *MiningFacts, line int) Site {
	if source == nil {
		source = sink
	}
	return Site{Path: source.Path, Line: line, Name: source.Name}
}
func taintEvidence(sink, source *MiningFacts, flow TaintFlow) *AnalysisEvidence {
	if source == nil || source.ID == sink.ID {
		return analysisEvidence(sink, TaintFlowKind, flow.SourceNode, flow.SinkNode)
	}
	result := analysisEvidence(sink, TaintFlowKind, flow.SinkNode)
	result.Related = []*AnalysisEvidence{analysisEvidence(source, TaintFlowKind, flow.SourceNode)}
	result.Status = "partial"
	result.Missing = append(result.Missing, "cross-function taint is a may-dependency summary")
	return result
}
