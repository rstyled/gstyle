package changerisk

import "strings"

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
