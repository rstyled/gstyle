package callflow

import "github.com/rstyled/gstyle/domain/pdg"

func solveFrameConfinement(bodies map[string]*exitBody, summaries ExitSummaries) {
	for changed := true; changed; {
		changed = false
		for name, body := range bodies {
			summary := summaries[name]
			if !summary.frameConfined || body.confinedFrame(summaries) {
				continue
			}
			summary.frameConfined = false
			summaries[name] = summary
			changed = true
		}
	}
}

func (body *exitBody) confinedFrame(summaries ExitSummaries) bool {
	for _, block := range body.flow.Blocks {
		if !block.Reachable {
			continue
		}
		for _, ref := range block.Operations {
			if !body.operationWithinFrame(ref, summaries) {
				return false
			}
		}
	}
	return true
}

func (body *exitBody) operationWithinFrame(ref pdg.Ref, summaries ExitSummaries) bool {
	if site, found := body.machine.Site(ref); found {
		return body.calls.confinedTargets(summaries, site.Invocation)
	}
	if body.graph.Nodes[ref-1].Category == pdg.Call {
		return body.calls.confinedTargets(summaries, ref)
	}
	if body.operations.effect(ref).Unknown {
		return false
	}
	return body.frameWrites(ref)
}

func (body *exitBody) frameWrites(ref pdg.Ref) bool {
	for _, write := range body.paths.Writes(ref) {
		if !directFrameWrite(write) {
			return false
		}
		role := body.graph.Symbols[write.Symbol-1].Role
		if role != pdg.CaptureRole && !privateMemoryRole(role) {
			return false
		}
	}
	return true
}

func (calls *CallSites) confinedTargets(summaries ExitSummaries, ref pdg.Ref) bool {
	targets, admitted := calls.memoryTargets(ref)
	if !admitted {
		return false
	}
	for _, name := range targets.names {
		if !calls.targetWithinFrame(name, summaries) {
			return false
		}
	}
	return true
}

func (calls *CallSites) targetWithinFrame(name string, summaries ExitSummaries) bool {
	if targetPreservesMemory(name, summaries) {
		return true
	}
	summary := summaries[name]
	if !summary.frameConfined {
		return false
	}
	if summary.captureArity == 0 {
		return true
	}
	if summary.invocations == nil {
		return false
	}
	return summary.invocations.captureBindings(calls.graph) != nil
}
