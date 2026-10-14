package patterns

import (
	"fmt"
	"strings"
)

// Taint analysis: track untrusted data from sources → through propagators
// → to sinks. Flag when tainted data reaches a dangerous sink without
// sanitization.
//
// The three abstractions (from SAST engineering):
//   - Sources: entry points of untrusted input (HTTP params, CLI args, ...)
//   - Propagators: operations that carry taint (string concat, assignment, ...)
//   - Sinks: sensitive operations that must not receive tainted input
//     (SQL queries, shell commands, ...)
// Sanitizers break the chain: their output is clean even with tainted input.
//
// For gstyle: v1 is intraprocedural. We walk the PDG's data edges forward
// from sources, marking tainted nodes to a fixpoint. A sink whose dangerous
// argument receives tainted data is a finding. Detection-only: the fix
// depends on context (parameterize the query? escape the output? validate
// the path?), so the LLM judges.

// taintSourceSuffixes are CalleeID suffixes whose result is tainted.
var taintSourceSuffixes = []string{
	// HTTP request input.
	"(*url.Values).Get", // r.URL.Query().Get("x")
	"(*http.Request).FormValue",
	"(*http.Header).Get", // r.Header.Get("x")
	// OS input.
	"os.Getenv",
	// I/O: result tainted (conservative: assume the reader is untrusted).
	"io.ReadAll",
	"ioutil.ReadAll",
}

// taintSink describes a dangerous operation and which argument positions
// must not receive tainted data. A nil dangerousArgs means any argument.
type taintSink struct {
	// callee is a CalleeID suffix, e.g. "(*sql.DB).Exec".
	callee string
	// dangerousArgs are the ArgPos values that must be clean. Nil = any.
	// Note: for method calls the receiver is ArgPos 0, so the first
	// real argument is ArgPos 1.
	dangerousArgs []int
	// desc is the human-readable risk, e.g. "SQL injection".
	desc string
}

// taintSinks is the hardcoded sink list.
var taintSinks = []taintSink{
	// SQL: only the query string is dangerous (ArgPos 1; ArgPos 0 is the
	// receiver). Tainted query parameters (ArgPos 2+) are the safe
	// parameterized pattern — do NOT flag those.
	{callee: "(*sql.DB).Exec", dangerousArgs: []int{1}, desc: "SQL injection"},
	{callee: "(*sql.DB).Query", dangerousArgs: []int{1}, desc: "SQL injection"},
	{callee: "(*sql.DB).QueryRow", dangerousArgs: []int{1}, desc: "SQL injection"},
	{callee: "(*sql.Tx).Exec", dangerousArgs: []int{1}, desc: "SQL injection"},
	{callee: "(*sql.Tx).Query", dangerousArgs: []int{1}, desc: "SQL injection"},
	{callee: "(*sql.Conn).Exec", dangerousArgs: []int{1}, desc: "SQL injection"},
	// Shell: any tainted argument is dangerous.
	{callee: "exec.Command", desc: "command injection"},
	{callee: "exec.CommandContext", desc: "command injection"},
	// Templates: the data argument (ArgPos 2; receiver 0, writer 1).
	{callee: "(*template.Template).Execute", dangerousArgs: []int{2}, desc: "XSS"},
	// File paths: the path argument.
	{callee: "os.Open", dangerousArgs: []int{0}, desc: "path traversal"},
	{callee: "os.Create", dangerousArgs: []int{0}, desc: "path traversal"},
	{callee: "os.WriteFile", dangerousArgs: []int{0}, desc: "path traversal"},
	{callee: "ioutil.WriteFile", dangerousArgs: []int{0}, desc: "path traversal"},
}

type taintPropagation struct {
	graph   *MiningGraph
	tainted map[int]bool
	risk    string
}

// TaintFlow is one tainted-data-to-sink flow.
type TaintFlow struct {
	SourceNode, SinkNode string
	FuncID               string
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
	for _, s := range taintSourceSuffixes {
		if taintCalleeMatches(n.CalleeID, s) {
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
		if taintCalleeMatches(n.CalleeID, taintSinks[i].callee) {
			return &taintSinks[i]
		}
	}
	return nil
}

func (n PdgNode) sanitizesRisk(risk string) bool {
	if n.Kind != Call {
		return false
	}
	if risk == "XSS" {
		return taintCalleeMatches(n.CalleeID, "html.EscapeString") || taintCalleeMatches(n.CalleeID, "template.HTMLEscapeString")
	}
	// URL escaping is not SQL, shell or filesystem validation. Numeric parse
	// results are typed numbers; error/string outputs require result-position proof.
	return false
}

// Propagate separately for each sink risk: escaping in one context must not
// clear taint for an unrelated context.
func propagateTaintForRisk(pdg *MiningGraph, risk string) map[int]bool {
	tainted := seedTaintSources(pdg)
	succ := dataSuccessors(pdg)
	for changed := true; changed; {
		changed = propagateTaintRiskStep(pdg, succ, tainted, risk)
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

func propagateTaintRiskStep(pdg *MiningGraph, succ [][]int, tainted map[int]bool, risk string) bool {
	changed := false
	propagation := taintPropagation{graph: pdg, tainted: tainted, risk: risk}
	for from, tos := range succ {
		if propagation.spread(from, tos) {
			changed = true
		}
	}
	return changed
}

func (p taintPropagation) spread(from int, tos []int) bool {
	if !p.tainted[from] {
		return false
	}
	spread := false
	for _, to := range tos {
		if p.tainted[to] || p.graph.Nodes[to].sanitizesRisk(p.risk) {
			continue
		}
		p.tainted[to] = true
		spread = true
	}
	return spread
}

// sinkArgTainted reports whether a dangerous argument of sink node idx
// receives tainted data.
func sinkArgTainted(pdg *MiningGraph, idx int, sink *taintSink, tainted map[int]bool) bool {
	for _, e := range pdg.Edges {
		if taintedSinkEdge(pdg, e, idx, sink, tainted) {
			return true
		}
	}
	return false
}

// taintedSinkEdge reports whether e is a tainted data edge into a
// dangerous argument of the sink.
func taintedSinkEdge(pdg *MiningGraph, e PdgEdge, idx int, sink *taintSink, tainted map[int]bool) bool {
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
	var out []TaintFlow
	taintByRisk := map[string]map[int]bool{}
	for i, n := range f.Pdg.Nodes {
		sink := n.matchTaintSink()
		if sink == nil {
			continue
		}
		tainted := taintByRisk[sink.desc]
		if tainted == nil {
			tainted = propagateTaintForRisk(f.Pdg, sink.desc)
			taintByRisk[sink.desc] = tainted
		}
		if !sinkArgTainted(f.Pdg, i, sink, tainted) {
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
			Analysis:   analysisEvidence(f, TaintFlowKind, fl.SourceNode, fl.SinkNode),
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
				{Path: f.Path, Line: fl.SourceLine, Name: f.Name},
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

func taintCalleeMatches(callee, pattern string) bool {
	canonical := strings.ReplaceAll(strings.ReplaceAll(pattern, "(*", ""), ").", ".")
	return strings.HasSuffix(callee, pattern) || strings.HasSuffix(callee, canonical)
}

func taintSourceLocation(graph *MiningGraph, index int) (int, string) {
	if index < 0 {
		return 0, ""
	}
	return graph.Nodes[index].Line, graph.Nodes[index].CalleeID
}
