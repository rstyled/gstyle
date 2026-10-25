package changerisk

import (
	"fmt"
	"strings"
)

func analysisSummary(report Report) string {
	var parts []string
	if len(report.GoAPI.Limits) != 0 {
		parts = append(parts, fmt.Sprintf("Go API constraint limits: %d", len(report.GoAPI.Limits)))
	}
	for _, kind := range []struct{ rule, label string }{
		{"unresolved_calls", "functions with unresolved call targets"},
		{"pdg_change", "source changes outside the projection"},
		{"analysis_unavailable", "scope warnings"},
		{"pdg_matching_ambiguous", "diagrams omitted for ambiguous matching"},
	} {
		if count := unknownReasons(report.Reasons, kind.rule); count != 0 {
			parts = append(parts, fmt.Sprintf("%s: %d", kind.label, count))
		}
	}
	if len(parts) == 0 {
		return "None detected in the compared scope."
	}
	return strings.Join(parts, "; ") + "."
}

func unknownReasons(reasons []Reason, rule string) int {
	count := 0
	for _, reason := range reasons {
		if reason.Rule == rule && reason.Level == Unknown {
			count++
		}
	}
	return count
}

func unmeasuredSummary(fields []string) string {
	labels := map[string]string{"go_api": "Go API compatibility", "cli": "CLI contract", "checks": "check results", "test_coverage": "runtime coverage"}
	var names []string
	for _, field := range fields {
		label, ok := labels[field]
		if !ok {
			label = text(field)
		}
		names = append(names, label)
	}
	if len(names) == 0 {
		return "Scope: static changes; this report does not authorize a merge."
	}
	return "Not assessed: " + strings.Join(names, ", ") + "."
}
