package changerisk

import (
	"fmt"
	"maps"
	"slices"
)

func Evaluate(base, head Snapshot, config Config) Report {
	changes, reasons := functionChanges(base, head, config)
	reasons = append(reasons, analysisReasons(base, head)...)
	pdgRisk := reasonsLevel(reasons)
	dependency, changed := dependencyReason(base, head, config)
	dependencyRisk := Low
	if dependency != nil {
		dependencyRisk = dependency.Level
		reasons = append(reasons, *dependency)
	}
	return Report{SchemaVersion: 1, Base: base.Commit, Head: head.Commit, PDGRisk: pdgRisk,
		DependencyRisk: dependencyRisk, DependenciesChanged: changed,
		Risk: strongest(Unknown, strongest(pdgRisk, dependencyRisk)), Reasons: reasons, Changes: changes,
		ChangedFunctions: len(changes), Unmeasured: []string{"compatibility", "checks", "test_coverage"}}
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
	if before.Graph.Shape == after.Graph.Shape {
		level, message = Unknown, "source changed outside the compared PDG projection"
	}
	reasons := []Reason{atFunction(current, Reason{Rule: "pdg_change", Level: level, Message: message})}
	if delta.Ambiguous {
		reasons = append(reasons, atFunction(current, Reason{Rule: "pdg_matching_ambiguous", Level: Unknown, Message: "operation correspondence is ambiguous; graph illustration omitted"}))
	}
	if hasUnresolvedCalls(before.Graph.Nodes) || hasUnresolvedCalls(after.Graph.Nodes) {
		reasons = append(reasons, atFunction(current, Reason{Rule: "unresolved_calls", Level: Unknown, Message: "changed function contains unresolved call targets; call rules are incomplete"}))
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
	var reasons []Reason
	for _, snapshot := range []Snapshot{base, head} {
		if snapshot.AnalysisError != "" {
			reasons = append(reasons, Reason{Rule: "analysis_unavailable", Level: Unknown, Message: snapshot.AnalysisError})
		}
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
	return &Reason{Rule: "dependency_change", Level: configured(config.DependencyChange),
		Message: "module requirements, replacements or package imports changed"}, true
}
