package callflow

import "github.com/rstyled/gstyle/domain/pdg"

// Memory preservation is a greatest fixed point: a recursive component without
// an external store preserves memory on every finite execution prefix.
func solveMemoryPreservation(bodies map[string]*exitBody, summaries ExitSummaries) {
	for changed := true; changed; {
		changed = false
		for name, body := range bodies {
			summary, _ := readExitSummary(summaries, name)
			if !summary.PreservesCallerMemory || body.preservesMemory(summaries) {
				continue
			}
			summary.PreservesCallerMemory = false
			summaries[name] = summary
			changed = true
		}
	}
}

func memoryBodyKnown(body *exitBody) bool {
	if body == nil || body.paths == nil {
		return false
	}
	return body.flow != nil && body.calls != nil && body.operations != nil && body.graph.Tables != nil
}

func (body *exitBody) preservesMemory(summaries ExitSummaries) bool {
	for _, block := range body.flow.Blocks {
		if !block.Reachable {
			continue
		}
		for _, ref := range block.Operations {
			if !body.operationPreservesMemory(ref, summaries) {
				return false
			}
		}
	}
	return true
}

func (body *exitBody) operationPreservesMemory(ref pdg.Ref, summaries ExitSummaries) bool {
	if site, found := body.machine.Site(ref); found {
		return body.calls.PreservesCallerMemory(summaries, site.Invocation)
	}
	if body.graph.Nodes[ref-1].Category == pdg.Call {
		return body.calls.PreservesCallerMemory(summaries, ref)
	}
	if body.operations.effect(ref).Unknown {
		return false
	}
	return body.privateWrites(ref)
}

func (body *exitBody) privateWrites(ref pdg.Ref) bool {
	for _, write := range body.paths.Writes(ref) {
		if write.Unresolved || write.Symbol == 0 || write.Location != 0 {
			return false
		}
		if !privateMemoryRole(body.graph.Symbols[write.Symbol-1].Role) {
			return false
		}
	}
	return true
}

func privateMemoryRole(role pdg.Role) bool {
	switch role {
	case pdg.LocalRole, pdg.ParameterRole, pdg.ReceiverRole, pdg.OutputRole, pdg.ReturnRole:
		return true
	default:
		return false
	}
}

// PreservesCallerMemory is independent of termination and argument values.
// Unknown targets and missing body summaries never establish this property.
func (calls *CallSites) PreservesCallerMemory(summaries ExitSummaries, ref pdg.Ref) bool {
	return calls.preservesCallerMemory(summaries, ref)
}

func (calls *CallSites) preservesCallerMemory(summaries exitSummaryReader, ref pdg.Ref) bool {
	if calls == nil {
		return false
	}
	targets, admitted := calls.memoryTargets(ref)
	if !admitted {
		return false
	}
	for _, name := range targets.names {
		if !targetPreservesMemory(name, summaries) {
			return false
		}
	}
	return true
}

func (calls *CallSites) memoryTargets(ref pdg.Ref) (callTargets, bool) {
	targets := calls.targets[ref]
	return targets, targets.known && len(targets.names) != 0 && calls.resolvedSiteWrites(ref)
}

func (calls *CallSites) resolvedSiteWrites(ref pdg.Ref) bool {
	// Private target-only solver fragments have no graph storage. Public
	// constructors always retain their validated graph and canonical accesses.
	if calls.graph == nil {
		return true
	}
	attrs := calls.graph.NodeAttributes(calls.graph.Nodes[ref-1])
	for _, write := range attrs.Writes {
		if write.UnresolvedReason != 0 {
			return false
		}
	}
	return true
}

func targetPreservesMemory(name string, summaries exitSummaryReader) bool {
	// These canonical termination boundaries use the caller's
	// explicit completion flow; their own bookkeeping does not store Go cells.
	switch name {
	case "runtime.Goexit", "os.Exit":
		return true
	}
	if summary, found := readExitSummary(summaries, name); found {
		return summary.PreservesCallerMemory
	}
	switch name {
	case "len", "cap", "complex", "real", "imag", "min", "max", "new", "make", "panic", "recover", "print", "println":
		return true
	default:
		return false
	}
}
