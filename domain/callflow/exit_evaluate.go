package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type exitPoint struct {
	values             valueStateID
	block              pdg.Ref
	recovery           pdg.Ref
	offset             int
	pending, recovered bool
}

type exitEvaluation struct {
	observing         bool
	observationWork   int
	observationLimit  bool
	stateIndex        valueStateIndex
	contexts          map[valueStateID]bindingflow.Evaluation
	concrete          map[exitValue]bindingflow.Value
	heapResults       heapEffects
	incoming          environmentHeap
	memory            memoryProof
	conditions        map[pdg.Ref]uint8
	returned          string
	captured          captureEffects
	callOutcomes      map[pdg.Ref]Outcomes
	operationOutcomes map[pdg.Ref]Outcomes
	values            map[valueStateID]exitValues
	body              *exitBody
	summaries         exitSummaryReader
	seen              map[exitPoint]bool
	changed           bool
	cyclic            bool
	active            map[exitPoint]bool
	results           map[exitPoint]completions
}

func (evaluation *exitEvaluation) visit(point exitPoint) completions {
	if evaluation.observing {
		evaluation.observe(point)
		return completions{}
	}
	return evaluation.complete(point)
}

func (evaluation *exitEvaluation) complete(point exitPoint) completions {
	if evaluation.body.canceled() {
		return completions{}
	}
	if evaluation.active[point] {
		evaluation.cyclic = true
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
	if evaluation.body.canceled() {
		return completions{}
	}
	old := evaluation.results[point]
	if old.add(result) {
		evaluation.changed = true
	}
	evaluation.results[point] = old
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
	evaluation.observeMemory(ref)
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
	evaluation.observeInvocation(exitInvocationContext{ref: ref, values: evaluation.valueContext(point)})
	if evaluation.body.calls.recoveryCall(ref) {
		return evaluation.recovery(point, ref)
	}
	outcomes := evaluation.body.calls.outcomesWithValues(evaluation.summaries, ref, false, evaluation.valueContext(point))
	if evaluation.callOutcomes != nil {
		evaluation.callOutcomes[ref] |= outcomes
	}
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
	heaps    heapEffects
	memory   memoryProof
	outcomes Outcomes
	returned string
	captured captureEffects
}

func evaluateBodyResult(body *exitBody, summaries exitSummaryReader, inherited bool, inputs string) evaluatedBody {
	if body == nil || body.flow == nil {
		return evaluatedBody{outcomes: UnknownExit}
	}
	evaluation := newExitEvaluation(body, summaries)
	outcomes := evaluation.run(inherited, inputs)
	return evaluatedBody{outcomes: outcomes, returned: evaluation.returned, captured: evaluation.captured, memory: evaluation.memory, heaps: evaluation.heapResults}
}

func newExitEvaluation(body *exitBody, summaries exitSummaryReader) *exitEvaluation {
	return &exitEvaluation{memory: memoryProof{preserves: true, confined: true}, values: map[valueStateID]exitValues{}, body: body, summaries: summaries, active: map[exitPoint]bool{}, results: map[exitPoint]completions{}}
}

func (evaluation *exitEvaluation) run(inherited bool, inputs string) Outcomes {
	point := exitPoint{block: evaluation.body.flow.Entry, pending: inherited}
	bindings := evaluation.initialBindings(inputs)
	input := readEnvironmentInput(inputs)
	evaluation.incoming = heapFromEntries(input.Heap)
	locations := map[pdg.Ref]string{}
	for _, link := range input.Captures {
		locations[link.Symbol] = link.Address
	}
	point.values = evaluation.internValues(exitValues{heap: evaluation.incoming, locations: locations, bindings: bindings, cells: evaluation.initialCellValues(bindings)})
	return evaluation.solveCompletions(point)
}
