package changerisk

import (
	"fmt"
	"maps"
	"slices"
)

func Evaluate(base, head Snapshot, config Config, api APIReport) Report {
	changes, reasons := functionChanges(base, head, config)
	reasons = append(reasons, analysisReasons(base, head)...)
	pdgRisk := reasonsLevel(reasons)
	dependency := compareDependencies(base, head, config)
	apiFindings := apiReasons(api, config)
	apiRisk := reasonsLevel(apiFindings)
	return Report{SchemaVersion: 1, Base: base.Commit, Head: head.Commit, PolicyConfigured: config.enabled(), PDGRisk: pdgRisk,
		APIRisk: apiRisk, GoAPI: api,
		DependencyRisk: dependency.risk, DependenciesChanged: dependency.changed,
		DependenciesAdded: dependency.added, DependenciesRemoved: dependency.removed,
		Risk: measuredRisk(config, pdgRisk, dependency.risk, apiRisk), Reasons: slices.Concat(reasons, dependency.reasons, apiFindings), Changes: changes,
		ChangedFunctions: len(changes), Unmeasured: unmeasuredFields(api)}
}

func measuredRisk(config Config, pdg, dependencies, api Level) Level {
	if !config.enabled() {
		return Unrated
	}
	return strongest(strongest(pdg, dependencies), api)
}

func functionChanges(base, head Snapshot, config Config) ([]Delta, []Reason) {
	before, after := functionsByName(base.Functions), functionsByName(head.Functions)
	names := maps.Clone(before)
	maps.Copy(names, after)
	changes, reasons := []Delta{}, []Reason{}
	for _, name := range slices.Sorted(maps.Keys(names)) {
		delta, items := compareFunction(before[name], after[name], config)
		if delta != nil {
			changes = append(changes, *delta)
			reasons = append(reasons, items...)
		}
	}
	return changes, reasons
}

func reasonsLevel(reasons []Reason) Level {
	result := Low
	for _, reason := range reasons {
		if reason.Rule == "pdg_matching_ambiguous" {
			continue
		}
		result = strongest(result, reason.Level)
	}
	return result
}

func functionsByName(functions []Function) map[string]Function {
	result := map[string]Function{}
	for _, function := range functions {
		key := function.ID
		if key == "" {
			key = function.Name
		}
		result[key] = function
	}
	return result
}

func compareFunction(before, after Function, config Config) (*Delta, []Reason) {
	if before.Body == after.Body && before.Graph.Shape == after.Graph.Shape {
		return nil, nil
	}
	current := currentFunction(before, after)
	delta := graphDelta(before, after)
	level := configured(config.PDGChange)
	message := "PDG operations or dependencies changed"
	if config.PDGChange == "" {
		message += "; no severity is configured"
	}
	if before.Graph.Shape == after.Graph.Shape {
		level, message = Unknown, "source changed outside the compared PDG projection"
	}
	reasons := []Reason{atFunction(current, Reason{Rule: "pdg_change", Level: level, Message: message})}
	if delta.Ambiguous {
		reasons = append(reasons, atFunction(current, Reason{Rule: "pdg_matching_ambiguous", Level: Unknown, Message: "operation correspondence is ambiguous; graph illustration omitted"}))
	}
	reasons = append(reasons, compareCalls(callComparison{before: before, current: current, nodes: after.Graph.Nodes}, config.Calls)...)
	return &delta, reasons
}

func hasUnresolvedCalls(nodes []Node) bool {
	return slices.ContainsFunc(nodes, func(node Node) bool { return node.UnresolvedCall })
}

func currentFunction(before, after Function) Function {
	if after.Name == "" {
		return before
	}
	return after
}

func atFunction(function Function, reason Reason) Reason {
	reason.Path, reason.Function = function.Path, function.Name
	if reason.Line == 0 {
		reason.Line = function.Line
	}
	return reason
}

func callCounts(nodes []Node) map[string]int {
	result := map[string]int{}
	for _, node := range nodes {
		if node.Callee != "" {
			result[node.Callee]++
		}
	}
	return result
}

type callComparison struct {
	before, current Function
	nodes           []Node
}

func compareCalls(pair callComparison, rules []CallRule) []Reason {
	previous, next := callCounts(pair.before.Graph.Nodes), callCounts(pair.nodes)
	var reasons []Reason
	if hasUnresolvedCalls(pair.before.Graph.Nodes) || hasUnresolvedCalls(pair.nodes) {
		reasons = append(reasons, atFunction(pair.current, Reason{Rule: "unresolved_calls", Level: Unknown, Message: "changed function contains unresolved call targets; call rules are incomplete"}))
	}
	for _, rule := range rules {
		added := next[rule.Callee] - previous[rule.Callee]
		if added <= 0 {
			continue
		}
		reason := Reason{Rule: "added_call", Level: rule.Level,
			Message: addedCallMessage(rule.Callee, added), Callee: rule.Callee}
		if previous[rule.Callee] == 0 {
			reason.Line = callLine(pair.nodes, rule.Callee)
		}
		reasons = append(reasons, atFunction(pair.current, reason))
	}
	return reasons
}

func addedCallMessage(callee string, count int) string {
	if count == 1 {
		return "Added call: " + callee
	}
	return fmt.Sprintf("Added %d calls: %s", count, callee)
}

func callLine(nodes []Node, callee string) int {
	for _, node := range nodes {
		if node.Callee == callee {
			return node.Line
		}
	}
	return 0
}

func analysisReasons(base, head Snapshot) []Reason {
	reasons := declarationReasons(base, head)
	for _, snapshot := range []Snapshot{base, head} {
		if snapshot.AnalysisError != "" {
			reasons = append(reasons, Reason{Rule: "analysis_unavailable", Level: Unknown, Message: snapshot.AnalysisError})
		}
	}
	return reasons
}

func declarationReasons(base, head Snapshot) []Reason {
	paths := map[string]string{}
	maps.Copy(paths, base.Declarations)
	maps.Copy(paths, head.Declarations)
	var reasons []Reason
	for _, path := range slices.Sorted(maps.Keys(paths)) {
		before, existed := base.Declarations[path]
		after, exists := head.Declarations[path]
		if existed == exists && before == after {
			continue
		}
		reasons = append(reasons, Reason{Rule: "analysis_unavailable", Level: Unknown, Path: path,
			Message: "Go declarations or comments changed outside the function-body PDG comparison"})
	}
	return reasons
}

func dependencyReason(base, head Snapshot, config Config) (*Reason, bool) {
	for _, snapshot := range []Snapshot{base, head} {
		if snapshot.DependencyError != "" {
			return &Reason{Rule: "dependencies_unavailable", Level: Unknown, Message: snapshot.DependencyError}, false
		}
	}
	if slices.Equal(base.Dependencies, head.Dependencies) {
		return nil, false
	}
	message := "module requirements, replacements or package imports changed"
	if config.DependencyChange == "" {
		message += "; no severity is configured"
	}
	return &Reason{Rule: "dependency_change", Level: configured(config.DependencyChange),
		Message: message}, true
}
