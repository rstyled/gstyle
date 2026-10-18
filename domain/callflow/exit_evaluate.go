package callflow

import "github.com/rstyled/gstyle/domain/pdg"

type exitPoint struct {
	values             string
	block              pdg.Ref
	recovery           pdg.Ref
	offset             int
	pending, recovered bool
}

type exitEvaluation struct {
	conditions        map[pdg.Ref]uint8
	returned          string
	captured          captureEffects
	callOutcomes      map[pdg.Ref]Outcomes
	operationOutcomes map[pdg.Ref]Outcomes
	values            map[string]exitValues
	body              *exitBody
	summaries         ExitSummaries
	seen              map[exitPoint]bool
	changed           bool
	active            map[exitPoint]bool
	results           map[exitPoint]completions
}

func (evaluation *exitEvaluation) visit(point exitPoint) completions {
	if evaluation.active[point] {
		result := terminalCompletion(Diverges)
		result.add(evaluation.results[point])
		return result
	}
	if evaluation.seen[point] {
		return evaluation.results[point]
	}
	evaluation.seen[point] = true
	evaluation.active[point] = true
	result := evaluation.step(point)
	delete(evaluation.active, point)
	old := evaluation.results[point]
	if old == nil {
		old = completions{}
		evaluation.results[point] = old
	}
	for value := range result {
		if !old[value] {
			old[value] = true
			evaluation.changed = true
		}
	}
	return old
}

func (evaluation *exitEvaluation) step(point exitPoint) completions {
	block := evaluation.body.flow.Blocks[point.block-1]
	if point.offset < len(block.Operations) {
		ref := block.Operations[point.offset]
		point.offset++
		return evaluation.operation(point, ref)
	}
	if len(block.Successors) == 0 {
		// A missing explicit return is not evidence of normal completion.
		return terminalCompletion(UnknownExit)
	}
	return evaluation.successors(point, block)
}

func (evaluation *exitEvaluation) operation(point exitPoint, ref pdg.Ref) completions {
	body := evaluation.body
	if site, found := body.machine.Site(ref); found {
		context := evaluation.valueContext(point)
		return evaluation.applyDeferred(evaluation.visit(point), site.Invocation, context)
	}
	node := body.graph.Nodes[ref-1]
	if node.Category == pdg.Return {
		return evaluation.returnAt(point, ref)
	}
	if node.Category == pdg.Call {
		return evaluation.call(point, ref)
	}
	point = evaluation.snapshotRead(point, ref)
	outcomes := body.operations.Outcomes(ref)
	if body.operations.effect(ref).NilInput {
		outcomes = body.operations.outcomesWithValues(ref, evaluation.valueContext(point))
	}
	if evaluation.operationOutcomes != nil {
		evaluation.operationOutcomes[ref] |= outcomes
	}
	return evaluation.outcomes(point, ref, outcomes)
}

func (evaluation *exitEvaluation) call(point exitPoint, ref pdg.Ref) completions {
	result := evaluation.priorOutcomes(point, ref)
	result.add(evaluation.invocation(point, ref))
	return result
}

func (evaluation *exitEvaluation) invocation(point exitPoint, ref pdg.Ref) completions {
	targets := evaluation.body.calls.targets[ref]
	if targets.known && len(targets.names) == 1 && targets.names[0] == "recover" {
		return evaluation.recovery(point, ref)
	}
	outcomes := evaluation.body.calls.outcomesWithValues(evaluation.summaries, ref, false, evaluation.valueContext(point))
	if evaluation.callOutcomes != nil {
		evaluation.callOutcomes[ref] |= outcomes
	}
	point = evaluation.callResult(point, ref)
	return evaluation.callCompletions(point, ref, outcomes)
}

func (evaluation *exitEvaluation) outcomes(point exitPoint, ref pdg.Ref, outcomes Outcomes) completions {
	result := evaluation.finish(point, outcomes&^Returns)
	// Unknown execution can return. Observation must visit that continuation
	// so an opaque prefix cannot hide states at a later call-site join. The
	// summary solver keeps its existing terminal UnknownExit abstraction.
	if outcomes&Returns != 0 || (evaluation.callOutcomes != nil && outcomes&UnknownExit != 0) {
		result.add(evaluation.visit(evaluation.transferValues(point, ref)))
	}
	return result
}

func (evaluation *exitEvaluation) priorOutcomes(point exitPoint, ref pdg.Ref) completions {
	candidates, known := evaluation.body.operations.PriorOperations(ref)
	if !known {
		return terminalCompletion(UnknownExit)
	}
	context := evaluation.valueContext(point)
	result := completions{}
	for _, candidate := range candidates {
		if !evaluation.mayFollowRecovery(point, candidate) || !evaluation.body.operations.allowsExecution(candidate, context) {
			continue
		}
		result.add(evaluation.finish(point, evaluation.body.operations.Outcomes(candidate)&^Returns))
	}
	return result
}

type evaluatedBody struct {
	outcomes Outcomes
	returned string
	captured captureEffects
}

func evaluateBodyResult(body *exitBody, summaries ExitSummaries, inherited bool, inputs string) evaluatedBody {
	if body == nil || body.flow == nil {
		return evaluatedBody{outcomes: UnknownExit}
	}
	evaluation := newExitEvaluation(body, summaries)
	outcomes := evaluation.run(inherited, inputs)
	return evaluatedBody{outcomes: outcomes, returned: evaluation.returned, captured: evaluation.captured}
}

func newExitEvaluation(body *exitBody, summaries ExitSummaries) *exitEvaluation {
	return &exitEvaluation{values: map[string]exitValues{}, body: body, summaries: summaries, active: map[exitPoint]bool{}, results: map[exitPoint]completions{}}
}

func (evaluation *exitEvaluation) run(inherited bool, inputs string) Outcomes {
	point := exitPoint{block: evaluation.body.flow.Entry, pending: inherited}
	bindings := evaluation.initialBindings(inputs)
	point.values = evaluation.internValues(exitValues{bindings: bindings, cells: evaluation.initialCellValues(bindings)})
	return evaluation.solveCompletions(point)
}
