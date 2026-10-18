package callflow

import "github.com/rstyled/gstyle/domain/pdg"

// OrdinaryCallOutcomes joins invocation outcomes over the existing exit
// evaluator's reachable ordinary states. Assignments, branch paths, loop states
// and unknown writes use the same value semantics as package exit summaries.
// Deferred registrations are not invocations. The returned map owns its data;
// missing entries establish nothing about continuation. Queries borrow a
// validated graph; summaries must be solved for that graph inventory.
func OrdinaryCallOutcomes(queries Queries, summaries ExitSummaries) map[pdg.Ref]Outcomes {
	return OrdinaryObservations(queries, summaries).Calls
}

// Observations owns joins over all visited ordinary states. A missing condition
// does not establish a constant; an unknown or conflicting value cannot prune.
type Observations struct {
	Calls      map[pdg.Ref]Outcomes
	Operations map[pdg.Ref]Outcomes
	Conditions map[pdg.Ref]pdg.BooleanFact
}

func OrdinaryObservations(queries Queries, summaries ExitSummaries) Observations {
	body := exitBodyWithQueries(queries)
	if body == nil || body.flow == nil {
		return Observations{}
	}
	evaluation := newExitEvaluation(body, summaries)
	evaluation.callOutcomes = map[pdg.Ref]Outcomes{}
	evaluation.operationOutcomes = map[pdg.Ref]Outcomes{}
	evaluation.conditions = map[pdg.Ref]uint8{}
	evaluation.run(false, "")
	return Observations{Calls: evaluation.callOutcomes, Operations: evaluation.operationOutcomes, Conditions: conditionFacts(evaluation.conditions)}
}

func conditionFacts(values map[pdg.Ref]uint8) map[pdg.Ref]pdg.BooleanFact {
	result := map[pdg.Ref]pdg.BooleanFact{}
	for ref, mask := range values {
		if mask == 1 || mask == 2 {
			result[ref] = pdg.BooleanFact{Status: pdg.Known, Value: mask == 1}
		}
	}
	return result
}

func (evaluation *exitEvaluation) observeCondition(ref pdg.Ref, truth, known bool) {
	if evaluation.conditions == nil {
		return
	}
	source := evaluation.body.paths.ConditionSource(ref)
	if source == 0 {
		return
	}
	mask := uint8(3)
	if known {
		mask = 2
		if truth {
			mask = 1
		}
	}
	evaluation.conditions[source] |= mask
}
