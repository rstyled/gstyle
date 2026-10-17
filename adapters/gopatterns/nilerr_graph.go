package gopatterns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

type nilErrGraph struct {
	graph          *pdg.Graph
	source         map[pdg.Ref]ast.Node
	guards         map[pdg.Ref]*ast.IfStmt
	objects        map[pdg.Ref]types.Object
	copies         *bindingflow.Query
	valueFacts     bindingflow.Facts
	valueLocations map[bindingflow.PathKey][]pdg.Ref
}

func newNilErrGraph(fn *ast.FuncDecl, fset *token.FileSet, graph *pdg.Graph) *nilErrGraph {
	if graph == nil || graph.ValueFlow == nil || pdg.ReachableProjection(graph) == nil {
		return nil
	}
	copies, err := bindingflow.New(graph)
	if err != nil {
		return nil
	}
	nodes := nilErrSourceNodes(fn, fset)
	query := &nilErrGraph{graph: graph, source: map[pdg.Ref]ast.Node{}, guards: map[pdg.Ref]*ast.IfStmt{}, objects: map[pdg.Ref]types.Object{}, copies: copies}
	query.indexSource(nodes)
	query.indexGuards(fn)
	return query
}

func (q *nilErrGraph) indexGuards(fn *ast.FuncDecl) {
	byCondition := map[ast.Node]*ast.IfStmt{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, literal := node.(*ast.FuncLit); literal {
			return false
		}
		if branch, ok := node.(*ast.IfStmt); ok {
			byCondition[branch.Cond] = branch
		}
		return true
	})
	for _, block := range q.graph.ValueFlow.Blocks {
		branch := byCondition[q.source[block.Condition]]
		if direct, ok := q.source[block.Condition].(*ast.IfStmt); ok {
			branch = direct
		}
		if branch != nil {
			q.guards[block.Condition] = branch
		}
	}
}

func (q *nilErrGraph) collectOperations(state *nilErrGraphState, block pdg.FlowBlock) bool {
	for _, ref := range block.Operations {
		if q.graph.Nodes[ref-1].Category == pdg.Return {
			return false
		}
		if q.graph.Nodes[ref-1].Category == pdg.Control {
			continue
		}
		if q.copies.HandlingUnknown(ref, state.evidence.checked, state.evidence.evaluation()) && state.hasContext() {
			state.blocked = true
		}
		q.recordOperation(&state.evidence, ref)
	}
	return true
}

func (q *nilErrGraph) nextStates(state nilErrGraphState, block pdg.FlowBlock) []nilErrGraphState {
	if len(block.Successors) == 1 {
		state.block = block.Successors[0]
		return []nilErrGraphState{state}
	}
	if len(block.Successors) != 2 {
		return nil
	}
	if block.Condition == 0 || q.copies.HandlingUnknown(block.Condition, state.evidence.checked, state.evidence.evaluation()) {
		return nil
	}
	return splitNilErrStates(state, block)
}

func (q *nilErrGraph) lostGraphPath(start pdg.Ref, context nilErrEvidence) bool {
	pending := []nilErrGraphState{{block: start, evidence: context}}
	budget := nilErrPathBudget
	for len(pending) > 0 && budget > 0 {
		budget--
		state := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		block := q.graph.ValueFlow.Blocks[state.block-1]
		found, continued := q.lostOperations(&state, block)
		if found {
			return true
		}
		if continued {
			pending = append(pending, q.lossNextStates(state, block)...)
		}
	}
	return false
}

func (q *nilErrGraph) lostOperations(state *nilErrGraphState, block pdg.FlowBlock) (bool, bool) {
	for _, ref := range block.Operations {
		if q.graph.Nodes[ref-1].Category == pdg.Return {
			return q.lostReturn(state.evidence, ref), false
		}
		if q.graph.Nodes[ref-1].Category == pdg.Control {
			continue
		}
		if q.copies.HandlingUnknown(ref, state.evidence.checked, state.evidence.evaluation()) {
			return false, false
		}
		q.recordOperation(&state.evidence, ref)
	}
	return false, true
}

func (q *nilErrGraph) lossNextStates(state nilErrGraphState, block pdg.FlowBlock) []nilErrGraphState {
	if len(block.Successors) != 2 {
		return q.nextStates(state, block)
	}
	if _, known := state.evidence.booleanRef(block.Condition); !known && !state.evidence.canSplitRef(block.Condition) {
		return nil
	}

	return q.nextStates(state, block)
}

func (q *nilErrGraph) indexSource(nodes map[string]ast.Node) {
	for i, node := range q.graph.Nodes {
		if node.Span == 0 || node.OriginalKind.Status != pdg.Known {
			continue
		}
		span := q.graph.Spans[node.Span-1]
		q.source[pdg.Ref(i+1)] = nodes[fmt.Sprintf("%s:%d:%d", q.graph.Tables.Text(node.OriginalKind.Value), span.Start, span.End)]
	}
}

type nilErrGraphState struct {
	block    pdg.Ref
	evidence nilErrEvidence
	blocked  bool
}

func (s nilErrGraphState) hasContext() bool {
	return len(s.evidence.truths) > 0 || len(s.evidence.guards) > 0
}

type nilErrGraphFinding struct {
	position token.Pos
	hit      patterns.NilErrHit
}

func (c *nilErrCollector) walkGraph(q *nilErrGraph, scope nilErrEvidence) {
	pending := []nilErrGraphState{{block: q.graph.ValueFlow.Entry, evidence: scope}}
	budget := nilErrPathBudget
	for len(pending) > 0 && budget > 0 {
		budget--
		state := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		block := q.graph.ValueFlow.Blocks[state.block-1]
		if !q.collectOperations(&state, block) {
			continue
		}
		c.checkGraphGuard(q, state, block)
		pending = append(pending, q.nextStates(state, block)...)
	}
}

func (c *nilErrCollector) checkGraphGuard(q *nilErrGraph, state nilErrGraphState, block pdg.FlowBlock) {
	finding := c.graphGuardFinding(q, state, block)
	if finding == nil {
		return
	}
	c.hits[finding.position] = finding.hit
}

func (c *nilErrCollector) graphGuardFinding(q *nilErrGraph, state nilErrGraphState, block pdg.FlowBlock) *nilErrGraphFinding {
	branch := q.guards[block.Condition]
	if !c.graphGuardAvailable(branch, state, block) {
		return nil
	}
	context, check := c.graphGuardContext(block.Condition, state.evidence)
	if check == nil {
		return nil
	}
	context.checked, context.positions = check.path, c.positions
	context.assumeError(nilErrCheck{path: check.path, isNotNil: true})
	successor := block.Successors[1]
	if check.isNotNil {
		successor = block.Successors[0]
	}
	if !q.lostGraphPath(successor, context) {
		return nil
	}
	return &nilErrGraphFinding{position: branch.Pos(), hit: patterns.NilErrHit{Line: c.fset.Position(branch.Pos()).Line, CondText: condText(c.fset, branch.Cond), Kind: "returns nil when err != nil"}}
}

func (c *nilErrCollector) graphGuardAvailable(branch *ast.IfStmt, state nilErrGraphState, block pdg.FlowBlock) bool {
	if branch == nil {
		return false
	}
	_, seen := c.hits[branch.Pos()]
	return !state.blocked && !seen && len(block.Successors) == 2
}

func (c *nilErrCollector) graphGuardContext(ref pdg.Ref, scope nilErrEvidence) (nilErrEvidence, *nilErrCheck) {
	context := scope.clone()
	if context.graph.copies.HandlingUnknown(ref, context.checked, context.evaluation()) {
		return context, nil
	}
	if _, known := context.booleanRef(ref); known {
		return context, nil
	}
	check := context.graph.nilCheck(ref, c.info)
	if check == nil {
		return context, nil
	}
	if !context.graph.copies.StablePath(check.path, context.graph.valueFacts) {
		return context, nil
	}
	return context, check
}

func findNilErrGraphHits(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info, graph *pdg.Graph) []patterns.NilErrHit {
	positions := errorResultPositions(fn, info)
	if len(positions) == 0 {
		return nil
	}
	query := newNilErrGraph(fn, fset, graph)
	if query == nil {
		return nil
	}
	query.indexTypedBindings(fn, info)
	if err := query.indexValueFacts(info); err != nil {
		return nil
	}
	collector := nilErrCollector{fset: fset, info: info, positions: positions, hits: map[token.Pos]patterns.NilErrHit{}}
	scope := nilErrEvidence{info: info, graph: query, values: map[pdg.Ref]nilErrValue{}, truths: map[pdg.Ref]bool{}}
	scope.initializeResults(info.Defs[fn.Name].Type().(*types.Signature))
	collector.walkGraph(query, scope)
	return collector.findings()
}

func nilErrSourceNodes(fn *ast.FuncDecl, fset *token.FileSet) map[string]ast.Node {
	nodes := map[string]ast.Node{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		key := fmt.Sprintf("%T:%d:%d", node, fset.PositionFor(node.Pos(), false).Offset, fset.PositionFor(node.End(), false).Offset)
		nodes[key] = node
		_, literal := node.(*ast.FuncLit)
		return !literal
	})
	return nodes
}

func splitNilErrStates(state nilErrGraphState, block pdg.FlowBlock) []nilErrGraphState {
	truth, known := state.evidence.booleanRef(block.Condition)
	if !known && !state.evidence.canAssumeRef(block.Condition) {
		return nil
	}
	var next []nilErrGraphState
	for i, successor := range block.Successors {
		selected := i == 0
		if known && selected != truth {
			continue
		}
		branch := nilErrGraphState{block: successor, evidence: state.evidence.clone(), blocked: state.blocked}
		branch.evidence.assumeRef(block.Condition, selected)
		next = append(next, branch)
	}
	return next
}
