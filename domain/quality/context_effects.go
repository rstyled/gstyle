package quality

// A recursive edge contributes the effects of every reachable bound context.
// Visiting each context once closes cycles without expanding a recursive tree.
type recursiveSummary struct {
	resolver *callResolver
	seen     map[string]bool
	pending  []Call
	result   callSummary
}

func (r *callResolver) recursiveCall(initial Call) callSummary {
	key := initial.storageKey(NormalizeCallee(initial.Callee))
	if summary, found := r.cache[key]; found {
		return initial.atCall(summary)
	}
	walk := recursiveSummary{resolver: r, seen: map[string]bool{}, pending: []Call{initial}}
	for len(walk.pending) > 0 {
		call := walk.pending[len(walk.pending)-1]
		walk.pending = walk.pending[:len(walk.pending)-1]
		if !walk.add(call, walk.context(call)) {
			break
		}
	}
	if r.cache != nil && len(r.cache) < localContextLimit {
		r.cache[key] = walk.result
	}
	return initial.atCall(walk.result)
}

func (walk *recursiveSummary) context(call Call) callSummary {
	kind, configured := call.policyKind(walk.resolver.prefixes)
	if configured {
		return call.classifiedCall(kind)
	}
	if !call.staticLocal() {
		return call.classifiedCall(UnknownEffect)
	}
	name := NormalizeCallee(call.Callee)
	helper, found := walk.resolver.helpers[name]
	if !found {
		return call.classifiedCall(UnknownEffect)
	}
	key := call.storageKey(name)
	if walk.seen[key] {
		return callSummary{}
	}
	if len(walk.seen) >= localContextLimit {
		return contextLimit(call)
	}
	walk.seen[key] = true
	return walk.expand(call, helper)
}

func (walk *recursiveSummary) expand(call Call, helper Helper) callSummary {
	bound := call.bindHelper(helper)
	effects := append(mutationEffects(bound.Mutations), bound.Effects...)
	if len(walk.pending)+len(bound.Calls) > localContextLimit {
		unknown := contextLimit(call)
		return call.atCall(callSummary{effects: append(effects, unknown.effects...), unclassified: unknown.unclassified})
	}
	walk.pending = append(walk.pending, bound.Calls...)
	return call.atCall(callSummary{effects: effects})
}

func (walk *recursiveSummary) add(call Call, summary callSummary) bool {
	walk.result.effects = append(walk.result.effects, summary.effects...)
	walk.result.unclassified += summary.unclassified
	if len(walk.result.effects) <= localEffectLimit {
		return true
	}
	walk.result = limitedSummary(call.Callee, walk.result)
	return false
}

func contextLimit(call Call) callSummary {
	call.Callee += " (local context limit)"
	return call.classifiedCall(UnknownEffect)
}
