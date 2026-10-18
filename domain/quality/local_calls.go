package quality

import (
	"slices"
	"strings"
)

const localDepthLimit = 32

const localEffectLimit = 4096

type callSummary struct {
	effects      []Effect
	unclassified int
}

type callResolver struct {
	prefixes []Prefix
	helpers  map[string]Helper
	active   map[string]bool
	cache    map[string]callSummary
}

func (r *callResolver) calls(calls []Call) callSummary {
	var result callSummary
	for _, call := range calls {
		summary := r.call(call)
		if len(result.effects)+len(summary.effects) > localEffectLimit {
			unknown := (Call{Callee: call.Callee + " (local summary limit)", Line: call.Line}).classifiedCall(UnknownEffect)
			result.effects = append(result.effects[:min(len(result.effects), localEffectLimit-1)], unknown.effects...)
			result.unclassified += unknown.unclassified
			return result
		}
		result.effects = append(result.effects, summary.effects...)
		result.unclassified += summary.unclassified
	}
	return result
}

func (r *callResolver) call(call Call) callSummary {
	name := NormalizeCallee(call.Callee)
	kind, configured := prefixKind(name, r.prefixes)
	if configured {
		return call.classifiedCall(kind)
	}
	if !call.staticLocal() {
		return call.classifiedCall(UnknownEffect)
	}
	if r.active[name] || len(r.active) >= localDepthLimit {
		return call.classifiedCall(UnknownEffect)
	}
	helper, ok := r.helpers[name]
	if !ok {
		return call.classifiedCall(UnknownEffect)
	}
	return call.atCall(r.helper(name, helper))
}

func (r *callResolver) helper(name string, helper Helper) callSummary {
	if summary, ok := r.cache[name]; ok {
		return summary
	}
	r.active[name] = true
	defer delete(r.active, name)
	calls := r.calls(helper.Calls)
	effects := append(calls.effects, helper.Effects...)
	effects = append(effects, mutationEffects(helper.Mutations)...)
	summary := callSummary{effects: effects, unclassified: calls.unclassified}
	if len(summary.effects) > localEffectLimit {
		summary = limitedSummary(name, summary)
	}
	r.cache[name] = summary
	return summary
}

func (call Call) classifiedCall(kind Kind) callSummary {
	if kind == None || (kind == MutationEffect && call.HasLocalContainerTarget()) {
		return callSummary{}
	}
	unknown := 0
	if kind == UnknownEffect {
		unknown = 1
	}
	return callSummary{effects: []Effect{{Kind: kind, Detail: call.Callee, Line: call.Line}}, unclassified: unknown}
}

func (call Call) atCall(summary callSummary) callSummary {
	if len(summary.effects) == 0 {
		return summary
	}
	via := shortCallee(NormalizeCallee(call.Callee))
	effects := make([]Effect, len(summary.effects))
	for index, effect := range summary.effects {
		effects[index] = Effect{Kind: effect.Kind, Detail: call.Callee + " → " + effect.Detail, Line: call.Line, Via: via}
	}
	return callSummary{effects: effects, unclassified: summary.unclassified}
}

func (call Call) staticLocal() bool { return call.Local && !call.Dynamic }

func localCalls(f Function, prefixes []Prefix) ([]Effect, int) {
	helpers := map[string]Helper{NormalizeCallee(f.Name): {Name: f.Name, Mutations: f.Mutations, Calls: f.Calls, Effects: f.Effects}}
	for _, helper := range f.Helpers {
		helpers[NormalizeCallee(helper.Name)] = helper
	}
	resolver := callResolver{prefixes: prefixes, helpers: helpers, active: map[string]bool{}, cache: map[string]callSummary{}}
	summary := resolver.calls(f.Calls)
	return summary.effects, summary.unclassified
}

// shortCallee strips the module path from a callee for display, keeping the
// package-qualified name: "example.com/mod/pkg.helper" → "pkg.helper".
func shortCallee(callee string) string {
	if i := strings.LastIndex(callee, "/"); i >= 0 {
		return callee[i+1:]
	}
	return callee
}

func prefixKind(normalizedCallee string, prefixes []Prefix) (Kind, bool) {
	best, kind := -1, UnknownEffect
	for _, prefix := range prefixes {
		if len(prefix.Path) >= best && strings.HasPrefix(normalizedCallee, prefix.Path) {
			best, kind = len(prefix.Path), prefix.Kind
		}
	}
	return kind, best >= 0
}

// Preserve established effects when marking a truncated helper summary unknown.
func limitedSummary(name string, summary callSummary) callSummary {
	unknown := (Call{Callee: name + " (local summary limit)", Line: 1}).classifiedCall(UnknownEffect)
	effects := slices.Clone(summary.effects[:localEffectLimit-1])
	effects = append(effects, unknown.effects...)
	return callSummary{effects: effects, unclassified: summary.unclassified + 1}
}
