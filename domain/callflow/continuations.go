package callflow

import "github.com/rstyled/gstyle/domain/pdg"

// OrdinaryCallOutcomes joins invocation outcomes over the existing exit
// evaluator's reachable ordinary states. Assignments, branch paths, loop states
// and unknown writes use the same value semantics as package exit summaries.
// Deferred registrations are not invocations. The returned map owns its data;
// missing entries establish nothing about continuation. Queries borrow a
// validated graph; summaries must be solved for that graph inventory.
func OrdinaryCallOutcomes(queries Queries, summaries ExitSummaries) map[pdg.Ref]Outcomes {
	body := exitBodyWithQueries(queries)
	if body == nil || body.flow == nil {
		return nil
	}
	evaluation := newExitEvaluation(body, summaries)
	evaluation.callOutcomes = map[pdg.Ref]Outcomes{}
	evaluation.run(false, "")
	return evaluation.callOutcomes
}
